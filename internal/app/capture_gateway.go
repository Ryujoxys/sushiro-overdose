package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Ryujoxys/sushiro-overdose/internal/proxy"
)

// 抓包代理看到的新网关请求（spec 018 单元 C）。
//
// 电脑抓包和手机抓包的代理都把 sapi.sushiro.com.cn 的 /gateway/ 请求交给这里：
//  1. 请求成功（2xx）时，用它的头部建立会员连接，比已保存的新才写入（SaveLiveSession 自己判断）；
//  2. getStoreById 成功时，按实时客户端同一套严格解码读出门店，编号与请求一致才记一笔
//     （Source = passive），同一家店一分钟最多记一次。
//
// 头部的值和响应体都不写日志；响应体只在内存里解码，写入的只有门店快照本身。

// passiveStoreRecordInterval：小程序停在门店页会每隔几秒刷新一次，和后台记录一样每分钟最多记一次。
const passiveStoreRecordInterval = queueBaselineMinIntervalSeconds * time.Second

// passiveStoreRecordTimeout bounds the local write (records, observations, alerts) for one passive snapshot.
const passiveStoreRecordTimeout = 10 * time.Second

var passiveStoreRecords = struct {
	sync.Mutex
	last map[int]time.Time
}{last: map[int]time.Time{}}

// captureGatewayOutcome is what one gateway exchange changed; tests read it, the proxy ignores it.
type captureGatewayOutcome struct {
	SessionSaved bool
	Recorded     int
	RecordErr    error
}

// recordPassiveSnapshots is the shared writer used for passive records; tests replace it.
var recordPassiveSnapshots = recordQueueLiveSnapshots

// newCaptureGatewayHandler returns the proxy.ProxyOptions.OnGatewayExchange handler for a capture run.
// onConnected (optional) runs after a newer member connection was saved, for a user-facing capture log line.
//
// The proxy calls the hook on the connection's relay loop and reads the mini program's next request only
// after it returns, while a passive record rebuilds the local model and may send queue alerts over the
// network. So the work runs on its own goroutine, on a copy of the request.
func newCaptureGatewayHandler(onConnected func(LiveSession)) proxy.GatewayExchangeFunc {
	return func(req *http.Request, status int, body []byte) {
		if req == nil || req.URL == nil {
			return
		}
		req = req.Clone(context.Background())
		now := time.Now()
		go func() {
			defer func() {
				if r := recover(); r != nil {
					LogMessage(time.Now(), "Gateway capture handler failed; the exchange was skipped")
				}
			}()
			handleCapturedGatewayExchange(req, status, body, now, onConnected)
		}()
	}
}

func handleCapturedGatewayExchange(req *http.Request, status int, body []byte, now time.Time, onConnected func(LiveSession)) captureGatewayOutcome {
	var out captureGatewayOutcome
	if req == nil || req.URL == nil || !isCapturedGatewayRequest(req.URL.Hostname(), req.URL.Path) {
		return out
	}
	// A rejected or failed request says nothing good about its token: never keep it, never record.
	if status < 200 || status > 299 {
		return out
	}
	session, saved := saveCapturedLiveSession(req.Header, now)
	out.SessionSaved = saved
	if saved && onConnected != nil {
		onConnected(session)
	}
	if req.Method == http.MethodGet && strings.TrimSuffix(req.URL.Path, "/") == proxy.SushiroGatewayStorePath {
		out.Recorded, out.RecordErr = recordPassiveGatewayStore(req.URL.Query().Get("storeId"), body, now)
		if out.RecordErr != nil {
			LogMessage(now, "Passive store snapshot skipped: "+passiveRecordErrorLabel(out.RecordErr))
		}
	}
	return out
}

func isCapturedGatewayRequest(host, path string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	return host == liveGatewayHost && strings.HasPrefix(path, liveGatewayMarker)
}

// saveCapturedLiveSession keeps the request's member connection when it is newer than the saved one.
// Logs never carry header values: only whether it was kept and for how long it stays usable.
func saveCapturedLiveSession(header http.Header, now time.Time) (LiveSession, bool) {
	s, err := LiveSessionFromHeaders(header, now, liveSessionSourceCapture)
	if err != nil {
		return LiveSession{}, false
	}
	saved, err := SaveLiveSession(s)
	if err != nil {
		LogMessage(now, "Saving the captured live session failed: "+err.Error())
		return LiveSession{}, false
	}
	if saved {
		noteLiveSessionSeen(s, now)
	} else if stored, _ := usableLiveSession(now); stored != nil && stored.Authorization == s.Authorization {
		// The mini program is still using the connection we already hold: nothing to save, but the capture worked.
		noteLiveSessionSeen(*stored, now)
	}
	if saved {
		LogMessage(now, fmt.Sprintf("Live session captured from the mini program (usable for about %d min)", liveSessionMinutesLeft(s, now)))
	}
	return s, saved
}

// liveSessionMinutesLeft rounds the usable time down to whole minutes for display.
func liveSessionMinutesLeft(s LiveSession, now time.Time) int {
	return int(s.Remaining(now) / time.Minute)
}

// liveCaptureConnectedMessage 是抓包中连上实时数据时给用户看的一行说明。
func liveCaptureConnectedMessage(s LiveSession, now time.Time) string {
	if minutes := liveSessionMinutesLeft(s, now); minutes > 0 {
		return fmt.Sprintf("实时数据已连接，约 %d 分钟内有效，只在这台电脑上用来看叫号。", minutes)
	}
	return "实时数据已连接，只在这台电脑上用来看叫号。"
}

// recordPassiveGatewayStore decodes one getStoreById reply with the live client's strict decoder.
// Nothing is written unless the reply is the store that was asked for.
func recordPassiveGatewayStore(rawStoreID string, body []byte, now time.Time) (int, error) {
	want, err := strconv.Atoi(strings.TrimSpace(rawStoreID))
	if err != nil || want <= 0 {
		return 0, errLiveStoreFormat
	}
	if len(body) == 0 {
		return 0, errLiveStoreFormat
	}
	store, err := decodeQueueLiveStore(body, want)
	if err != nil {
		return 0, err
	}
	if !passiveStoreRecordDue(want, now) {
		return 0, nil
	}
	store.ObservedAt, store.Source = now, queueLiveSourcePassive
	ctx, cancel := context.WithTimeout(context.Background(), passiveStoreRecordTimeout)
	defer cancel()
	return recordPassiveSnapshots(ctx, []QueueLiveStore{store})
}

// passiveStoreRecordDue claims the store's slot when its last passive record is a minute old or more.
func passiveStoreRecordDue(storeID int, now time.Time) bool {
	passiveStoreRecords.Lock()
	defer passiveStoreRecords.Unlock()
	if last, ok := passiveStoreRecords.last[storeID]; ok && now.Sub(last) < passiveStoreRecordInterval && !now.Before(last) {
		return false
	}
	passiveStoreRecords.last[storeID] = now
	return true
}

func passiveRecordErrorLabel(err error) string {
	if errors.Is(err, errLiveStoreFormat) {
		return "reply did not match the requested store"
	}
	return err.Error()
}

// liveSessionConnectedNow reports whether this process holds a usable member connection; never exposes
// its values. It goes through queueLiveSessionProvider, so only the interface process (which injects it)
// ever reads the connection; background and CLI processes always see false without touching the file.
func liveSessionConnectedNow() bool {
	s, _ := usableLiveSession(time.Now())
	return s != nil
}

// captureForLive marks a capture started only to connect live data (spec 018): it ends once the
// mini program's gateway connection reaches this computer and does not save ticket credentials.
const captureForLive = "live"

// capturePurposeFromRequest reads an optional {"purpose":"live"} body; anything else means a ticket-credential capture.
func capturePurposeFromRequest(r *http.Request) string {
	if r == nil || r.Body == nil {
		return ""
	}
	var body struct {
		Purpose string `json:"purpose"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		return ""
	}
	if strings.TrimSpace(body.Purpose) == captureForLive {
		return captureForLive
	}
	return ""
}
