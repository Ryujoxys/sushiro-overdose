package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Ryujoxys/sushiro-overdose/internal/proxy"
)

func capturedGatewayRequest(t *testing.T, method, rawURL, token string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, rawURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-App-Client", testLiveAppClient)
	req.Header.Set("X-App-Code", testLiveAppCode)
	req.Header.Set("User-Agent", "Mozilla/5.0 MicroMessenger/8.0.50 MiniProgramEnv/iOS")
	req.Header.Set("Referer", "https://servicewechat.com/wx7ac31ef6c073a7ed/170/page-frame.html")
	return req
}

func resetPassiveStoreRecords(t *testing.T) {
	t.Helper()
	reset := func() {
		passiveStoreRecords.Lock()
		passiveStoreRecords.last = map[int]time.Time{}
		passiveStoreRecords.Unlock()
	}
	reset()
	t.Cleanup(reset)
}

const capturedStorePath = "https://sapi.sushiro.com.cn" + proxy.SushiroGatewayStorePath

func TestCapturedGatewaySessionComesOnlyFromSuccessfulGatewayRequests(t *testing.T) {
	reliabilityHome(t)
	resetPassiveStoreRecords(t)
	now := time.Now()
	token := testLiveJWT(now.Add(-time.Minute), now.Add(time.Hour))
	listURL := "https://sapi.sushiro.com.cn" + proxy.SushiroGatewayStoresPath + "?latitude=1&longitude=1&numresults=10000"

	var out string
	out = captureStdout(t, func() {
		for _, rejected := range []struct {
			name   string
			req    *http.Request
			status int
		}{
			{"rejected by the gateway", capturedGatewayRequest(t, http.MethodGet, listURL, token), http.StatusUnauthorized},
			{"server error", capturedGatewayRequest(t, http.MethodGet, listURL, token), http.StatusBadGateway},
			{"legacy host", capturedGatewayRequest(t, http.MethodGet, "https://crm-cn-prd.sushiro.com.cn/wechat/api/2.0/stores", token), http.StatusOK},
			{"gateway host outside /gateway/", capturedGatewayRequest(t, http.MethodGet, "https://sapi.sushiro.com.cn/static/app.js", token), http.StatusOK},
			{"other sushiro host", capturedGatewayRequest(t, http.MethodGet, "https://static.sushiro.com.cn/gateway/wechat/api/2.0/stores", token), http.StatusOK},
		} {
			if got := handleCapturedGatewayExchange(rejected.req, rejected.status, nil, now, nil); got.SessionSaved {
				t.Fatalf("%s: session saved", rejected.name)
			}
			if s, err := LoadLiveSession(); err != nil || s != nil {
				t.Fatalf("%s: live session file written: %+v %v", rejected.name, s, err)
			}
		}

		connected := []LiveSession{}
		onConnected := func(s LiveSession) { connected = append(connected, s) }
		if got := handleCapturedGatewayExchange(capturedGatewayRequest(t, http.MethodGet, listURL, token), http.StatusOK, []byte(`[]`), now, onConnected); !got.SessionSaved || got.Recorded != 0 {
			t.Fatalf("successful gateway request: %+v", got)
		}
		saved, err := LoadLiveSession()
		if err != nil || saved == nil {
			t.Fatalf("session not saved: %v", err)
		}
		if saved.Authorization != "Bearer "+token || saved.AppCode != testLiveAppCode || saved.Source != liveSessionSourceCapture || !saved.Usable(time.Now()) {
			t.Fatalf("saved session = source %q usable %v", saved.Source, saved.Usable(time.Now()))
		}
		if len(connected) != 1 || connected[0].Authorization != saved.Authorization {
			t.Fatalf("onConnected calls = %d", len(connected))
		}
		if info, err := os.Stat(liveSessionPath()); err != nil || (info.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/') {
			t.Fatalf("live session file permissions: %v %v", info, err)
		}

		// The same token again changes nothing; a newer one replaces it; an older one never does.
		if got := handleCapturedGatewayExchange(capturedGatewayRequest(t, http.MethodGet, listURL, token), http.StatusOK, nil, now.Add(time.Minute), onConnected); got.SessionSaved {
			t.Fatal("same session written twice")
		}
		newer := testLiveJWT(now, now.Add(2*time.Hour))
		if got := handleCapturedGatewayExchange(capturedGatewayRequest(t, http.MethodGet, listURL, newer), http.StatusOK, nil, now.Add(2*time.Minute), onConnected); !got.SessionSaved {
			t.Fatal("newer session not kept")
		}
		older := testLiveJWT(now.Add(-10*time.Minute), now.Add(50*time.Minute))
		if got := handleCapturedGatewayExchange(capturedGatewayRequest(t, http.MethodGet, listURL, older), http.StatusOK, nil, now.Add(3*time.Minute), onConnected); got.SessionSaved {
			t.Fatal("older session replaced a newer one")
		}
		if saved, _ := LoadLiveSession(); saved == nil || saved.Authorization != "Bearer "+newer {
			t.Fatal("kept session is not the newest")
		}
		if len(connected) != 2 {
			t.Fatalf("onConnected calls = %d, want 2", len(connected))
		}
		if msg := liveCaptureConnectedMessage(connected[1], now); !strings.Contains(msg, "实时数据已连接") || strings.Contains(msg, newer) {
			t.Fatalf("connected message = %q", msg)
		}
	})
	for _, secret := range []string{token, testLiveJWT(now, now.Add(2*time.Hour))} {
		assertNoLiveSecrets(t, "capture log", out, secret)
	}
	if !strings.Contains(out, "Live session captured from the mini program") {
		t.Fatalf("capture was not logged:\n%s", out)
	}
}

func TestPassiveStoreRecordNeedsTheRequestedStore(t *testing.T) {
	reliabilityHome(t)
	resetPassiveStoreRecords(t)
	now := time.Now()
	token := testLiveJWT(now.Add(-time.Minute), now.Add(time.Hour))
	good := `{"id":"1012","name":"南山店","storeStatus":"OPEN","wait":30,"groupQueuesCount":9,"groupQueues":{"boothQueue":["820"]}}`

	out := captureStdout(t, func() {
		for _, bad := range []struct {
			name   string
			query  string
			body   string
			status int
		}{
			{"other store", "?storeId=1013", good, http.StatusOK},
			{"no store id", "", good, http.StatusOK},
			{"non-numeric store id", "?storeId=abc", good, http.StatusOK},
			{"envelope", "?storeId=1012", `{"code":0,"data":` + good + `}`, http.StatusOK},
			{"empty object", "?storeId=1012", `{}`, http.StatusOK},
			{"array", "?storeId=1012", `[` + good + `]`, http.StatusOK},
			{"truncated", "?storeId=1012", good[:20], http.StatusOK},
			{"missing status", "?storeId=1012", `{"id":1012,"name":"南山店","wait":30}`, http.StatusOK},
			{"undecodable body", "?storeId=1012", "", http.StatusOK},
			{"gateway error", "?storeId=1012", good, http.StatusInternalServerError},
		} {
			got := handleCapturedGatewayExchange(capturedGatewayRequest(t, http.MethodGet, capturedStorePath+bad.query, token), bad.status, []byte(bad.body), now, nil)
			if got.Recorded != 0 {
				t.Fatalf("%s: recorded %d", bad.name, got.Recorded)
			}
			if bad.status == http.StatusOK && !errors.Is(got.RecordErr, errLiveStoreFormat) {
				t.Fatalf("%s: error = %v", bad.name, got.RecordErr)
			}
			if observations := loadQueueObservations(); len(observations) != 0 {
				t.Fatalf("%s: observations written: %+v", bad.name, observations)
			}
			if _, err := os.Stat(queueBaselineRecordsPath()); !os.IsNotExist(err) {
				t.Fatalf("%s: records file written", bad.name)
			}
		}

		got := handleCapturedGatewayExchange(capturedGatewayRequest(t, http.MethodGet, capturedStorePath+"?storeId=1012", token), http.StatusOK, []byte(good), now, nil)
		if got.Recorded != 1 || got.RecordErr != nil {
			t.Fatalf("matching store: %+v", got)
		}
		// The mini program refreshes every few seconds: one passive record per store per minute.
		again := handleCapturedGatewayExchange(capturedGatewayRequest(t, http.MethodGet, capturedStorePath+"?storeId=1012", token), http.StatusOK, []byte(good), now.Add(20*time.Second), nil)
		if again.Recorded != 0 || again.RecordErr != nil {
			t.Fatalf("refresh within a minute: %+v", again)
		}
		later := handleCapturedGatewayExchange(capturedGatewayRequest(t, http.MethodGet, capturedStorePath+"?storeId=1012", token), http.StatusOK, []byte(good), now.Add(61*time.Second), nil)
		if later.Recorded != 1 {
			t.Fatalf("a minute later: %+v", later)
		}
		// A POST to the same path is not a store read.
		post := handleCapturedGatewayExchange(capturedGatewayRequest(t, http.MethodPost, capturedStorePath+"?storeId=1012", token), http.StatusOK, []byte(good), now.Add(5*time.Minute), nil)
		if post.Recorded != 0 {
			t.Fatalf("POST recorded: %+v", post)
		}
	})
	assertNoLiveSecrets(t, "passive record log", out, token)
	if strings.Contains(out, "南山店") || strings.Contains(out, "820") {
		t.Fatalf("response body reached the log:\n%s", out)
	}

	observations := loadQueueObservations()
	if len(observations) != 2 {
		t.Fatalf("observations = %+v", observations)
	}
	for _, o := range observations {
		if o.StoreID != "1012" || o.DisplayCalledNo != 820 || o.SourceEndpoint == queueLiveSourceManual {
			t.Fatalf("passive observation = %+v", o)
		}
	}
	if at, ok := parseRFC3339Local(observations[0].CollectedAt); !ok || at.Unix() != now.Unix() {
		t.Fatalf("observation time = %s, want the capture time", observations[0].CollectedAt)
	}
	records, err := parseJSONLFile[QueueBaselineRecord](queueBaselineRecordsPath(), normalizeQueueBaselineRecordForRead)
	if err != nil || len(records) != 2 || records[0].StoreID != 1012 {
		t.Fatalf("records = %+v %v", records, err)
	}
}

// useInterfaceLiveSessionProvider makes this test process read the saved connection the way the
// interface process does (web.go injects LoadLiveSession).
func useInterfaceLiveSessionProvider(t *testing.T) {
	t.Helper()
	previous := queueLiveSessionProvider
	queueLiveSessionProvider = LoadLiveSession
	t.Cleanup(func() {
		queueLiveSessionProvider = previous
		liveSessionRejectedMu.Lock()
		liveSessionRejectedMark = ""
		liveSessionRejectedMu.Unlock()
	})
}

func TestLiveConnectedIsReportedByBothCaptureFlows(t *testing.T) {
	reliabilityHome(t)
	now := time.Now()
	e := &BookingEngine{state: EngineState{Status: EngineCapturing}}
	m := &mobileAuthCaptureManager{}
	s := testLiveSession(now, time.Hour)
	if saved, err := SaveLiveSession(s); !saved || err != nil {
		t.Fatalf("save: %v %v", saved, err)
	}
	// A process without the injected provider (background service, CLI) never reads the connection.
	if e.GetState().LiveConnected || m.status()["live_connected"] != false {
		t.Fatal("a process without the provider reported the saved connection")
	}
	useInterfaceLiveSessionProvider(t)
	state := e.GetState()
	if !state.LiveConnected || m.status()["live_connected"] != true {
		t.Fatal("saved connection not reported")
	}
	// Ticket-credential completeness is separate: no captured fields, still not complete.
	if state.Capture != nil {
		t.Fatalf("live connection changed ticket capture status: %+v", state.Capture)
	}
	for name, value := range map[string]any{"engine": state, "mobile": m.status()} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), `"live_connected":true`) {
			t.Fatalf("%s JSON lacks live_connected: %s", name, data)
		}
		assertNoLiveSecrets(t, name+" status", string(data), s.Authorization)
	}
	// A connection the gateway rejected is not "connected" any more.
	noteLiveSessionRejected(s.mark())
	if e.GetState().LiveConnected {
		t.Fatal("rejected connection still reported")
	}
	if err := ClearLiveSession(); err != nil {
		t.Fatal(err)
	}
	if e.GetState().LiveConnected || m.status()["live_connected"] != false {
		t.Fatal("cleared connection still reported")
	}
}

func TestBothCaptureProxiesInjectTheGatewayHandler(t *testing.T) {
	for _, file := range []string{"engine.go", "mobile_auth_capture.go"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		source := string(data)
		if !strings.Contains(source, "OnGatewayExchange = newCaptureGatewayHandler(") || !strings.Contains(source, "StartProxyWithOptions(caCert, caKey, tokens, proxyOptions,") {
			t.Fatalf("%s does not wire the gateway handler into its capture proxy", file)
		}
	}
	// The handler only saves the connection and writes local records: it never reaches ticket mutations.
	data, err := os.ReadFile("capture_gateway.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"CreateNetTicket(", "CancelNetTicket(", "LoadLocalConfig(", "NewQueueLiveClient(", "http.Client"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("capture_gateway.go reaches %s", forbidden)
		}
	}
}

func TestCaptureGatewayHandlerDoesNotHoldTheProxyLoop(t *testing.T) {
	reliabilityHome(t)
	resetPassiveStoreRecords(t)
	now := time.Now()
	token := testLiveJWT(now.Add(-time.Minute), now.Add(time.Hour))
	good := []byte(`{"id":1012,"name":"南山店","storeStatus":"OPEN","wait":30,"groupQueues":{"boothQueue":["820"]}}`)

	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan int, 1)
	previous := recordPassiveSnapshots
	recordPassiveSnapshots = func(ctx context.Context, stores []QueueLiveStore) (int, error) {
		close(entered)
		<-release // a slow model rebuild or alert delivery
		finished <- len(stores)
		return len(stores), nil
	}
	t.Cleanup(func() { recordPassiveSnapshots = previous })

	connected := make(chan LiveSession, 1)
	handler := newCaptureGatewayHandler(func(s LiveSession) { connected <- s })
	req := capturedGatewayRequest(t, http.MethodGet, capturedStorePath+"?storeId=1012", token)

	returned := make(chan struct{})
	go func() {
		handler(req, http.StatusOK, good)
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("the gateway hook blocked the proxy while the passive record was being written")
	}
	// The proxy may reuse or change its request afterwards; the handler works on its own copy.
	req.Header.Set("Authorization", "Bearer changed-after-return")
	req.URL.RawQuery = "storeId=9999"

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("passive record never started")
	}
	select {
	case s := <-connected:
		if s.Authorization != "Bearer "+token {
			t.Fatal("handler did not use the request it was given")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("session was not saved")
	}
	close(release)
	select {
	case n := <-finished:
		if n != 1 {
			t.Fatalf("recorded %d stores", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("passive record did not finish")
	}
	if saved, err := LoadLiveSession(); err != nil || saved == nil || saved.Authorization != "Bearer "+token {
		t.Fatalf("saved session = %v %v", saved != nil, err)
	}
}

func TestCaptureGatewayHandlerSurvivesAPanickingWriter(t *testing.T) {
	reliabilityHome(t)
	resetPassiveStoreRecords(t)
	now := time.Now()
	token := testLiveJWT(now.Add(-time.Minute), now.Add(time.Hour))
	entered := make(chan struct{})
	previous := recordPassiveSnapshots
	recordPassiveSnapshots = func(context.Context, []QueueLiveStore) (int, error) {
		close(entered)
		panic("writer bug")
	}
	t.Cleanup(func() { recordPassiveSnapshots = previous })
	handler := newCaptureGatewayHandler(nil)
	handler(nil, http.StatusOK, nil) // nothing to do, no goroutine
	handler(capturedGatewayRequest(t, http.MethodGet, capturedStorePath+"?storeId=1012", token), http.StatusOK,
		[]byte(`{"id":1012,"name":"南山店","storeStatus":"OPEN","wait":30}`))
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("passive record never started")
	}
	// A panic in the background writer must not crash the interface process; give the recover a moment.
	time.Sleep(50 * time.Millisecond)
}
