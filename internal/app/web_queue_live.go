package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

// handleQueueLiveStores 返回门店列表。实时读不到（包括没有连接）时改用离线目录：
// 内置历史 + 本机记录 + 门店昵称，响应里 offline=true 并说明来源，原有 stores/count 不变。
func handleQueueLiveStores(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	query := queueLiveStoreQueryFromRequest(r)
	stores, err := queueLiveListStores(r.Context(), query)
	if err != nil {
		directory := offlineStoreDirectory()
		if len(directory) == 0 {
			writeLiveError(w, err)
			return
		}
		offline := filterOfflineQueueStores(directory, query)
		writeJSON(w, map[string]any{
			"stores":  offline,
			"count":   len(offline),
			"offline": true,
			"source":  "history",
			"history": queueHistoryLabel(),
			"code":    liveErrorCode(err),
			"message": queueOfflineStoresMessage(),
		})
		return
	}
	writeJSON(w, map[string]any{
		"stores": stores,
		"count":  len(stores),
	})
}

func handleQueueLiveStore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	storeID := strings.TrimSpace(r.URL.Query().Get("id"))
	if storeID == "" {
		storeID = strings.TrimSpace(r.URL.Query().Get("store"))
	}
	if storeID == "" {
		writeError(w, http.StatusBadRequest, "缺少门店 ID")
		return
	}
	store, err := fetchQueueLiveStore(r.Context(), storeID)
	if err != nil {
		writeLiveError(w, err)
		return
	}
	writeJSON(w, store)
}

func handleQueueLivePanel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	storeID := strings.TrimSpace(r.URL.Query().Get("store"))
	if storeID == "" {
		storeID = strings.TrimSpace(r.URL.Query().Get("id"))
	}
	if storeID == "" {
		writeError(w, http.StatusBadRequest, "缺少门店 ID")
		return
	}
	panel, err := buildQueueLivePanel(r.Context(), storeID, time.Now())
	if err != nil {
		writeLiveError(w, err)
		return
	}
	writeJSON(w, panel)
}

func handleQueueAdvisor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	q := r.URL.Query()
	storeID := strings.TrimSpace(q.Get("store"))
	if storeID == "" {
		storeID = strings.TrimSpace(q.Get("id"))
	}
	if storeID == "" {
		writeError(w, http.StatusBadRequest, "缺少门店 ID")
		return
	}
	now := time.Now()
	reading, err := queueReadingFromQuery(q.Get("called_no"), q.Get("called_at"), now)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	advisor, err := buildQueueAdvisorWithReading(r.Context(), storeID, atoiDefault(q.Get("target_no"), 0), queueTravelMinutes(q.Get("travel_minutes")), reading, now)
	if err != nil {
		writeLiveError(w, err)
		return
	}
	writeJSON(w, advisor)
}

// handleQueueAccuracy 返回基于实测回测的「预测 vs 实际」精度报告（按店）。
func handleQueueAccuracy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	writeJSON(w, getEtaAccuracyReport())
}

func handleQueuePressureCurve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	q := r.URL.Query()
	storeID := strings.TrimSpace(q.Get("store"))
	if storeID == "" {
		storeID = strings.TrimSpace(q.Get("id"))
	}
	if storeID == "" {
		writeError(w, http.StatusBadRequest, "缺少门店 ID")
		return
	}
	writeJSON(w, buildQueuePressureCurve(r.Context(), storeID, strings.TrimSpace(q.Get("date")), time.Now()))
}

// handleQueuePlan 时间互推：?pickup=HHMM 算几点能吃；?target_meal=HHMM 算几点取号。
func handleQueuePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	q := r.URL.Query()
	storeID := strings.TrimSpace(q.Get("store"))
	if storeID == "" {
		storeID = strings.TrimSpace(q.Get("id"))
	}
	if storeID == "" {
		writeError(w, http.StatusBadRequest, "缺少门店 ID")
		return
	}
	now := time.Now()
	if pickup := strings.TrimSpace(q.Get("pickup")); pickup != "" {
		writeJSON(w, buildQueuePickupPlan(r.Context(), storeID, pickup, now))
		return
	}
	if meal := strings.TrimSpace(q.Get("target_meal")); meal != "" {
		writeJSON(w, buildQueueMealPlan(r.Context(), storeID, meal, queueTravelMinutes(q.Get("travel_minutes")), now, false))
		return
	}
	writeError(w, http.StatusBadRequest, "缺少 pickup 或 target_meal 参数")
}

func handleQueueAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, LoadQueueAlertConfig())
	case http.MethodPost:
		var cfg QueueAlertConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		cfg = normalizeQueueAlertConfig(cfg)
		if err := SaveQueueAlertConfig(cfg); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, cfg)
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST only")
	}
}

func handleQueueAlertStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	writeJSON(w, BuildQueueAlertStatus(time.Now()))
}

func handleQueueLiveAreas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	areas, err := queueLiveListAreas(r.Context())
	if err != nil {
		directory := offlineStoreDirectory()
		if len(directory) == 0 {
			writeLiveError(w, err)
			return
		}
		offline := offlineStoreAreas(directory)
		writeJSON(w, map[string]any{
			"areas":   offline,
			"count":   len(offline),
			"offline": true,
			"source":  "history",
			"history": queueHistoryLabel(),
			"code":    liveErrorCode(err),
			"message": queueOfflineStoresMessage(),
		})
		return
	}
	writeJSON(w, map[string]any{
		"areas": areas,
		"count": len(areas),
	})
}

func queueLiveStoreQueryFromRequest(r *http.Request) QueueLiveStoreQuery {
	q := r.URL.Query()
	return QueueLiveStoreQuery{
		StoreIDs:    queueTrendRequestStores(q["store"], q.Get("stores")),
		City:        strings.TrimSpace(q.Get("city")),
		Area:        strings.TrimSpace(q.Get("area")),
		Keyword:     strings.TrimSpace(q.Get("q")),
		Near:        strings.TrimSpace(q.Get("near")),
		OpenOnly:    truthyQuery(q.Get("open")),
		WaitingOnly: truthyQuery(q.Get("waiting")),
		Limit:       atoiDefault(q.Get("limit"), 0),
	}
}

func truthyQuery(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

// queueTravelMinutes 解析路程分钟，限定在 0–240。
func queueTravelMinutes(raw string) int {
	return min(240, max(0, atoiDefault(raw, 0)))
}

const queueReadingMaxCalledNo = 99999

// queueReadingFromQuery 解析“小程序里现在叫到几号”和读数时间（HH:MM / HHMM / RFC3339，默认现在）。
// 没填叫号返回 nil。
func queueReadingFromQuery(rawNo, rawAt string, now time.Time) (*queueCalledReading, error) {
	rawNo = strings.TrimSpace(rawNo)
	if rawNo == "" {
		return nil, nil
	}
	calledNo, err := strconv.Atoi(rawNo)
	if err != nil || calledNo <= 0 || calledNo > queueReadingMaxCalledNo {
		return nil, fmt.Errorf("叫到的号码请填 1 到 %d 之间的整数", queueReadingMaxCalledNo)
	}
	now = now.In(SushiroTimezone)
	at := now
	if rawAt = strings.TrimSpace(rawAt); rawAt != "" {
		if parsed, ok := parseHHMM(rawAt, now); ok {
			at = parsed
		} else if parsed, err := time.Parse(time.RFC3339, rawAt); err == nil {
			at = parsed.In(SushiroTimezone)
		} else {
			return nil, fmt.Errorf("叫号时间请填 HH:MM，例如 18:30")
		}
	}
	if at.After(now.Add(time.Minute)) {
		return nil, fmt.Errorf("叫号时间不能晚于现在")
	}
	if now.Sub(at) > 12*time.Hour {
		return nil, fmt.Errorf("叫号时间太早了，请填今天看到的时间")
	}
	return &queueCalledReading{calledNo: calledNo, at: at, source: "input"}, nil
}

// handleQueueReading 手动记一笔：POST {store_id, called_no, wait_groups?}。
// 只写一条本机观测（来源 manual，等待未知），并评估叫号提醒；不请求任何寿司郎接口。
func handleQueueReading(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var input struct {
		StoreID    json.RawMessage `json:"store_id"`
		CalledNo   *int            `json:"called_no"`
		WaitGroups *int            `json:"wait_groups"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	d.DisallowUnknownFields()
	if err := d.Decode(&input); err != nil || d.Decode(new(any)) != io.EOF {
		writeError(w, http.StatusBadRequest, "请填写门店和叫到的号码")
		return
	}
	storeID := queueReadingStoreID(input.StoreID)
	if storeID == "" {
		writeError(w, http.StatusBadRequest, "请先选择门店")
		return
	}
	if input.CalledNo == nil || *input.CalledNo <= 0 || *input.CalledNo > queueReadingMaxCalledNo {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("叫到的号码请填 1 到 %d 之间的整数", queueReadingMaxCalledNo))
		return
	}
	if input.WaitGroups != nil && (*input.WaitGroups < 0 || *input.WaitGroups > 999) {
		writeError(w, http.StatusBadRequest, "在等桌数请填 0 到 999 之间的整数，不知道可以不填")
		return
	}
	obs, err := recordQueueReading(r.Context(), storeID, *input.CalledNo, input.WaitGroups, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "这次叫号没能记下，请重试")
		return
	}
	at, _ := parseRFC3339Local(obs.CollectedAt)
	writeJSON(w, map[string]any{
		"ok":               true,
		"store_id":         storeID,
		"called_no":        obs.DisplayCalledNo,
		"observed_at":      obs.CollectedAt,
		"observed_at_hhmm": queueClockHHMM(at),
	})
}

// queueReadingStoreID 接受 "1012" 或 1012，只允许 1–9 位数字。
func queueReadingStoreID(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		var n json.Number
		if err := json.Unmarshal(raw, &n); err != nil {
			return ""
		}
		text = n.String()
	}
	text = strings.TrimSpace(text)
	if len(text) == 0 || len(text) > 9 {
		return ""
	}
	for _, r := range text {
		if r < '0' || r > '9' {
			return ""
		}
	}
	n, err := strconv.Atoi(text)
	if err != nil || n <= 0 {
		return ""
	}
	return strconv.Itoa(n)
}
