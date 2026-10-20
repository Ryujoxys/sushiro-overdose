package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	testLiveAppClient = "app-client-secret-value"
	testLiveAppCode   = "app-code-secret-value"
)

func testLiveJWT(iat, exp time.Time) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"sub":"member-secret","iat":%d,"exp":%d}`, iat.Unix(), exp.Unix())))
	return header + "." + payload + ".c2lnbmF0dXJlLXNlY3JldA"
}

// testLiveSession returns a connection captured at now and valid for ttl.
func testLiveSession(now time.Time, ttl time.Duration) LiveSession {
	return LiveSession{
		Authorization: "Bearer " + testLiveJWT(now, now.Add(ttl)),
		AppClient:     testLiveAppClient,
		AppCode:       testLiveAppCode,
		UserAgent:     "test-wechat-agent",
		Referer:       "https://servicewechat.com/test/page-frame.html",
		ContentType:   "application/json",
		IssuedAt:      time.Unix(now.Unix(), 0),
		ExpiresAt:     time.Unix(now.Add(ttl).Unix(), 0),
		CapturedAt:    now,
		Source:        liveSessionSourceImport,
	}
}

// useLiveSession makes this test process behave like the UI process holding s (nil = none).
func useLiveSession(t *testing.T, s *LiveSession) {
	t.Helper()
	previous := queueLiveSessionProvider
	queueLiveSessionProvider = func() (*LiveSession, error) {
		if s == nil {
			return nil, nil
		}
		copy := *s
		return &copy, nil
	}
	t.Cleanup(func() {
		queueLiveSessionProvider = previous
		liveSessionRejectedMu.Lock()
		liveSessionRejectedMark = ""
		liveSessionRejectedMu.Unlock()
	})
}

func gatewayCurl(token string) string {
	return `curl 'https://sapi.sushiro.com.cn/gateway/wechat/api/2.0/getStoreById?storeId=1012' \
  -H 'authorization: Bearer ` + token + `' \
  -H 'x-app-client: ` + testLiveAppClient + `' \
  -H 'x-app-code: ` + testLiveAppCode + `' \
  -H 'content-type: application/json' \
  -H 'referer: https://servicewechat.com/wx7ac31ef6c073a7ed/170/page-frame.html' \
  -H 'user-agent: Mozilla/5.0 MicroMessenger/8.0.50 MiniProgramEnv/Mac' \
  --compressed`
}

func assertNoLiveSecrets(t *testing.T, where, text, token string) {
	t.Helper()
	for _, secret := range []string{token, testLiveAppClient, testLiveAppCode, "c2lnbmF0dXJl"} {
		if secret != "" && strings.Contains(text, secret) {
			t.Fatalf("%s exposes a connection value: %s", where, text)
		}
	}
}

func TestLiveSessionFromHeadersRequiresBearerAndAppContext(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	token := testLiveJWT(now.Add(-time.Minute), now.Add(59*time.Minute))
	full := http.Header{}
	full.Set("authorization", "bearer "+token)
	full.Set("x-app-client", testLiveAppClient)
	full.Set("x-app-code", testLiveAppCode)
	s, err := LiveSessionFromHeaders(full, now, liveSessionSourceCapture)
	if err != nil {
		t.Fatal(err)
	}
	if s.Authorization != "Bearer "+token || s.AppClient != testLiveAppClient || s.AppCode != testLiveAppCode || s.Source != liveSessionSourceCapture {
		t.Fatalf("session fields: %+v", s)
	}
	if s.UserAgent == "" || !strings.Contains(s.UserAgent, "MicroMessenger") || s.Referer != queueLiveReferer || s.ContentType != "application/json" {
		t.Fatalf("missing defaults: ua=%q referer=%q ct=%q", s.UserAgent, s.Referer, s.ContentType)
	}
	if !s.IssuedAt.Equal(time.Unix(now.Add(-time.Minute).Unix(), 0)) || !s.ExpiresAt.Equal(time.Unix(now.Add(59*time.Minute).Unix(), 0)) || !s.CapturedAt.Equal(now) {
		t.Fatalf("jwt times: iat=%v exp=%v captured=%v", s.IssuedAt, s.ExpiresAt, s.CapturedAt)
	}

	bare := full.Clone()
	bare.Set("Authorization", token)
	if s, err := LiveSessionFromHeaders(bare, now, liveSessionSourceImport); err != nil || s.Authorization != "Bearer "+token {
		t.Fatalf("bare JWT: %+v %v", s, err)
	}
	for name, mutate := range map[string]func(http.Header){
		"no authorization": func(h http.Header) { h.Del("Authorization") },
		"basic scheme":     func(h http.Header) { h.Set("Authorization", "Basic dXNlcjpwYXNz") },
		"opaque bare":      func(h http.Header) { h.Set("Authorization", "not-a-jwt") },
		"no app client":    func(h http.Header) { h.Del("X-App-Client") },
		"no app code":      func(h http.Header) { h.Del("X-App-Code") },
		"control chars":    func(h http.Header) { h["X-App-Code"] = []string{"a\nb"} },
	} {
		h := full.Clone()
		mutate(h)
		if _, err := LiveSessionFromHeaders(h, now, liveSessionSourceImport); err == nil {
			t.Errorf("%s: accepted", name)
		} else {
			assertNoLiveSecrets(t, name, err.Error(), token)
		}
	}
}

func TestLiveSessionUsableAndRemaining(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	s := testLiveSession(now, time.Hour)
	if !s.Usable(now) || s.Remaining(now) != time.Hour-30*time.Second {
		t.Fatalf("fresh session: usable=%v remaining=%v", s.Usable(now), s.Remaining(now))
	}
	if s.Usable(now.Add(time.Hour-30*time.Second)) || s.Remaining(now.Add(2*time.Hour)) != 0 {
		t.Fatal("session usable inside the 30s expiry margin")
	}
	noExp := s
	noExp.ExpiresAt = time.Time{}
	if !noExp.Usable(now.Add(54*time.Minute)) || noExp.Usable(now.Add(55*time.Minute)) {
		t.Fatal("session without exp must last 55 minutes from capture")
	}
	missing := s
	missing.AppCode = ""
	if missing.Usable(now) {
		t.Fatal("incomplete session reported usable")
	}
}

func TestParseLiveSessionTextFormats(t *testing.T) {
	now := time.Now()
	token := testLiveJWT(now.Add(-time.Minute), now.Add(time.Hour))
	raw := "GET /gateway/wechat/api/2.0/getStoreById?storeId=1012 HTTP/1.1\r\n" +
		"Host: sapi.sushiro.com.cn\r\n" +
		"authorization: Bearer " + token + "\r\n" +
		"x-app-client: " + testLiveAppClient + "\r\n" +
		"x-app-code: " + testLiveAppCode + "\r\n" +
		"\r\n{}"
	h2 := ":method: GET\n:authority: sapi.sushiro.com.cn\n:path: /gateway/wechat/api/2.0/stores?latitude=1\n" +
		"authorization: Bearer " + token + "\nx-app-client: " + testLiveAppClient + "\nx-app-code: " + testLiveAppCode + "\n"
	cmd := `curl ^"https://sapi.sushiro.com.cn/gateway/wechat/api/2.0/getStoreById?storeId=1012^" ^
  -H ^"authorization: Bearer ` + token + `^" ^
  -H ^"x-app-client: ` + testLiveAppClient + `^" ^
  -H ^"x-app-code: ` + testLiveAppCode + `^"`
	for name, text := range map[string]string{
		"curl": gatewayCurl(token), "raw headers": raw, "http2 headers": h2, "windows cmd curl": cmd,
		"bom": string(rune(0xFEFF)) + gatewayCurl(token),
	} {
		s, err := ParseLiveSessionText(text, now)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if s.Authorization != "Bearer "+token || s.AppClient != testLiveAppClient || s.AppCode != testLiveAppCode || s.Source != liveSessionSourceImport || !s.Usable(now) {
			t.Errorf("%s: %+v", name, s)
		}
	}
	if s, _ := ParseLiveSessionText(gatewayCurl(token), now); !strings.Contains(s.UserAgent, "MiniProgramEnv/Mac") || !strings.Contains(s.Referer, "/170/") {
		t.Errorf("curl lost UA/Referer: %q %q", s.UserAgent, s.Referer)
	}
}

func TestParseLiveSessionHARPicksNewestGatewayRequest(t *testing.T) {
	now := time.Now()
	older := testLiveJWT(now.Add(-20*time.Minute), now.Add(40*time.Minute))
	newer := testLiveJWT(now.Add(-time.Minute), now.Add(59*time.Minute))
	entry := func(at time.Time, rawURL, token string) map[string]any {
		return map[string]any{
			"startedDateTime": at.UTC().Format(time.RFC3339Nano),
			"request": map[string]any{"method": "GET", "url": rawURL, "headers": []map[string]string{
				{"name": ":authority", "value": "sapi.sushiro.com.cn"},
				{"name": "authorization", "value": "Bearer " + token},
				{"name": "x-app-client", "value": testLiveAppClient},
				{"name": "x-app-code", "value": testLiveAppCode},
			}},
		}
	}
	gateway := "https://sapi.sushiro.com.cn/gateway/wechat/api/2.0/getStoreById?storeId=1012"
	har := map[string]any{"log": map[string]any{"entries": []any{
		entry(now.Add(-time.Minute), gateway, newer),
		entry(now.Add(-20*time.Minute), gateway, older),
		entry(now, "https://crm-cn-prd.sushiro.com.cn/wechat/api/2.0/getStoreById?storeId=1012", "legacy-token"),
		entry(now, "https://sapi.sushiro.com.cn/static/app.js", "static-token"),
		entry(now, "https://example.com/gateway/x", "other-token"),
	}}}
	data, _ := json.Marshal(har)
	s, err := ParseLiveSessionText(string(data), now)
	if err != nil || s.Authorization != "Bearer "+newer {
		t.Fatalf("HAR picked %q, %v", s.Authorization, err)
	}
}

func TestParseLiveSessionTextRejectsOtherRequests(t *testing.T) {
	now := time.Now()
	token := testLiveJWT(now.Add(-time.Hour), now.Add(time.Hour))
	expired := testLiveJWT(now.Add(-2*time.Hour), now.Add(-time.Hour))
	legacy := strings.Replace(gatewayCurl(token), "sapi.sushiro.com.cn/gateway/wechat/api/2.0", "crm-cn-prd.sushiro.com.cn/wechat/api/2.0", 1)
	for name, tc := range map[string]struct {
		text string
		want error
	}{
		"empty":           {"  ", errLiveImportEmpty},
		"legacy host":     {legacy, errLiveImportLegacy},
		"other host":      {strings.Replace(gatewayCurl(token), "sapi.sushiro.com.cn", "sapi.sushiro.com.cn.example.com", 1), errLiveImportNotGateway},
		"no gateway path": {strings.Replace(gatewayCurl(token), "/gateway/", "/static/", 1), errLiveImportNotGateway},
		"expired":         {gatewayCurl(expired), errLiveImportExpired},
		"no app code":     {strings.Replace(gatewayCurl(token), "-H 'x-app-code: "+testLiveAppCode+"'", "", 1), errLiveNoAppContext},
		"not HAR JSON":    {`{"hello":"world"}`, errLiveImportUnreadable},
		"plain words":     {"authorization please", errLiveImportNotGateway},
	} {
		_, err := ParseLiveSessionText(tc.text, now)
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: got %v, want %v", name, err, tc.want)
		}
		if err != nil {
			assertNoLiveSecrets(t, name, err.Error(), token)
		}
	}
}

func TestSaveLiveSessionKeepsOnlyNewerConnections(t *testing.T) {
	reliabilityHome(t)
	if s, err := LoadLiveSession(); s != nil || err != nil {
		t.Fatalf("empty load: %+v %v", s, err)
	}
	now := time.Now()
	first := testLiveSession(now.Add(-10*time.Minute), time.Hour)
	if saved, err := SaveLiveSession(first); !saved || err != nil {
		t.Fatalf("first save: %v %v", saved, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(liveSessionPath())
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("connection file permissions: %v %v", info.Mode(), err)
		}
	}
	again := first
	again.CapturedAt = now
	if saved, _ := SaveLiveSession(again); saved {
		t.Fatal("rewrote the same token")
	}
	shorter := testLiveSession(now, 20*time.Minute)
	if saved, _ := SaveLiveSession(shorter); saved {
		t.Fatal("replaced a connection that lasts longer")
	}
	longer := testLiveSession(now, 2*time.Hour)
	if saved, err := SaveLiveSession(longer); !saved || err != nil {
		t.Fatalf("newer connection not saved: %v", err)
	}
	if expired := testLiveSession(now.Add(-3*time.Hour), time.Hour); func() bool { saved, _ := SaveLiveSession(expired); return saved }() {
		t.Fatal("saved an expired connection")
	}
	loaded, err := LoadLiveSession()
	if err != nil || loaded == nil || loaded.Authorization != longer.Authorization || !loaded.ExpiresAt.Equal(longer.ExpiresAt) {
		t.Fatalf("loaded %+v %v", loaded, err)
	}
	if _, err := SaveLiveSession(LiveSession{Authorization: "Bearer x.y.z"}); err == nil {
		t.Fatal("saved a connection without app context")
	}
	if err := ClearLiveSession(); err != nil {
		t.Fatal(err)
	}
	if err := ClearLiveSession(); err != nil {
		t.Fatal("clearing twice failed:", err)
	}
	if s, err := LoadLiveSession(); s != nil || err != nil {
		t.Fatalf("after clear: %+v %v", s, err)
	}
	// A stored connection that has expired is replaced by any usable one.
	stale := testLiveSession(now.Add(-2*time.Hour), time.Hour)
	data, _ := json.Marshal(stale)
	if err := os.WriteFile(liveSessionPath(), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if saved, _ := SaveLiveSession(shorter); !saved {
		t.Fatal("expired stored connection blocked a usable one")
	}
}

func TestLiveSessionEndpointsNeverReturnHeaderValues(t *testing.T) {
	reliabilityHome(t)
	mux := newWebMux()
	call := func(method, path, body string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rr
	}
	decode := func(rr *httptest.ResponseRecorder) map[string]any {
		var out map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
			t.Fatalf("%d %s", rr.Code, rr.Body.String())
		}
		return out
	}
	if got := decode(call("GET", "/api/live-session", "")); got["state"] != "none" || got["connected"] != false {
		t.Fatalf("initial state: %v", got)
	}
	now := time.Now()
	token := testLiveJWT(now.Add(-time.Minute), now.Add(time.Hour))
	body, _ := json.Marshal(map[string]string{"text": gatewayCurl(token)})
	rr := call("POST", "/api/live-session/import", string(body))
	got := decode(rr)
	if rr.Code != 200 || got["state"] != "active" || got["connected"] != true || got["saved"] != true || got["source"] != liveSessionSourceImport {
		t.Fatalf("import: %d %v", rr.Code, got)
	}
	if remaining, _ := got["remaining_seconds"].(float64); remaining < 3000 || remaining > 3600 {
		t.Fatalf("remaining: %v", got["remaining_seconds"])
	}
	assertNoLiveSecrets(t, "import response", rr.Body.String(), token)
	status := call("GET", "/api/live-session", "")
	assertNoLiveSecrets(t, "status response", status.Body.String(), token)
	if decode(status)["state"] != "active" {
		t.Fatalf("status after import: %s", status.Body.String())
	}
	again := decode(call("POST", "/api/live-session/import", string(body)))
	if again["saved"] != false || again["message"] != "这条连接已经在用了" {
		t.Fatalf("re-import: %v", again)
	}

	soon := testLiveJWT(now.Add(-55*time.Minute), now.Add(5*time.Minute))
	if err := ClearLiveSession(); err != nil {
		t.Fatal(err)
	}
	body, _ = json.Marshal(map[string]string{"text": gatewayCurl(soon)})
	if got := decode(call("POST", "/api/live-session/import", string(body))); got["state"] != "expiring" {
		t.Fatalf("expiring: %v", got)
	}

	for _, bad := range []string{`{"text":"curl https://example.com"}`, `{"text":""}`, `not json`} {
		rr := call("POST", "/api/live-session/import", bad)
		if rr.Code != http.StatusBadRequest || !strings.Contains(decode(rr)["error"].(string), "请") {
			t.Fatalf("bad import %q: %d %s", bad, rr.Code, rr.Body.String())
		}
	}
	if got := decode(call("POST", "/api/live-session/clear", "{}")); got["state"] != "none" {
		t.Fatalf("clear: %v", got)
	}
	if s, _ := LoadLiveSession(); s != nil {
		t.Fatal("clear left the connection on disk")
	}
	for _, tc := range []struct{ method, path string }{
		{"POST", "/api/live-session"}, {"GET", "/api/live-session/import"}, {"GET", "/api/live-session/clear"},
	} {
		if rr := call(tc.method, tc.path, ""); rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestLiveSessionWritesRequireCSRF(t *testing.T) {
	reliabilityHome(t)
	old := getWebCSRFToken()
	setWebCSRFToken("live-session-test-token")
	t.Cleanup(func() { setWebCSRFToken(old) })
	handler := webSecurityMiddleware(newWebMux())
	for _, path := range []string{"/api/live-session/import", "/api/live-session/clear"} {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:39871"+path, strings.NewReader(`{"text":"x"}`))
		req.Header.Set("Origin", "http://127.0.0.1:39871")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("%s without CSRF: %d", path, rr.Code)
		}
	}
}

func TestLiveSessionStatusReportsRejectedConnectionAsExpired(t *testing.T) {
	now := time.Now()
	s := testLiveSession(now, time.Hour)
	useLiveSession(t, &s)
	if status := liveSessionStatusAt(&s, now); status.State != "active" || !status.Connected {
		t.Fatalf("active: %+v", status)
	}
	noteLiveSessionRejected(s.mark())
	if status := liveSessionStatusAt(&s, now); status.State != "expired" || status.Connected || status.RemainingSeconds != 0 {
		t.Fatalf("rejected: %+v", status)
	}
	if liveSessionMark(now) != "" {
		t.Fatal("a rejected connection still counts as usable")
	}
}

func TestOnlyTheInterfaceProcessInjectsTheLiveSession(t *testing.T) {
	web, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(web), "queueLiveSessionProvider = LoadLiveSession") {
		t.Fatal("the interface process no longer injects the live session")
	}
	for _, path := range []string{"queue_service.go", "queue_collection_state.go", "queue_model.go", "queue_baseline.go", "daemon.go", "sampling_cli.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"LoadLiveSession", "queueLiveSessionProvider =", "LoadLocalConfig("} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("%s reaches the member connection: %s", path, forbidden)
			}
		}
	}
	if queueLiveSessionProvider != nil {
		t.Fatal("tests start with a live session provider set")
	}
	publicCollectorFixture(t)
	old := queueBaselineCollector
	queueBaselineCollector = &QueueBaselineCollector{collect: func(context.Context, QueueBaselineConfig) (int, error) { return 0, nil }}
	t.Cleanup(func() { queueBaselineCollector = old })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runPublicQueueService(ctx); err != nil {
		t.Fatal(err)
	}
	if queueLiveSessionProvider != nil {
		t.Fatal("the background service set a live session provider")
	}
}

func TestImportReplacesAConnectionTheGatewayRejected(t *testing.T) {
	reliabilityHome(t)
	useLiveSession(t, nil)
	now := time.Now()
	// The stored connection was issued later (longer exp) but the gateway has since rejected it.
	stored := testLiveSession(now.Add(-time.Minute), 2*time.Hour)
	if saved, err := SaveLiveSession(stored); !saved || err != nil {
		t.Fatalf("seed: %v %v", saved, err)
	}
	noteLiveSessionRejected(stored.mark())

	again := stored
	again.CapturedAt = now
	if saved, _ := SaveLiveSession(again); saved {
		t.Fatal("rewrote the rejected token itself")
	}
	token := testLiveJWT(now.Add(-time.Minute), now.Add(time.Hour))
	body, _ := json.Marshal(map[string]string{"text": gatewayCurl(token)})
	rr := httptest.NewRecorder()
	newWebMux().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/live-session/import", strings.NewReader(string(body))))
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil || rr.Code != http.StatusOK || got["saved"] != true || got["connected"] != true {
		t.Fatalf("fresh connection refused after a rejection: %d %s", rr.Code, rr.Body.String())
	}
	assertNoLiveSecrets(t, "import response", rr.Body.String(), token)
	if loaded, err := LoadLiveSession(); err != nil || loaded == nil || loaded.Authorization != "Bearer "+token {
		t.Fatalf("stored connection after import: %v", err)
	}
}

func TestParseLiveSessionTextKeepsSeveralCurlCommandsApart(t *testing.T) {
	now := time.Now()
	token := testLiveJWT(now.Add(-time.Minute), now.Add(time.Hour))
	other := `curl 'https://crm-cn-prd.sushiro.com.cn/wechat/api/2.0/stores?latitude=1' \
  -H 'authorization: Bearer legacy-anonymous-token' \
  -H 'x-app-client: other-client' \
  -H 'x-app-code: other-code'`
	cmdOther := `curl ^"https://example.com/api^" ^
  -H ^"authorization: Bearer other.service.token^"`
	for name, text := range map[string]string{
		"gateway first":    gatewayCurl(token) + " ;\n" + other,
		"gateway last":     other + " ;\n" + gatewayCurl(token),
		"windows cmd list": cmdOther + " &\r\n" + strings.ReplaceAll(gatewayCurl(token), "'", `^"`),
	} {
		s, err := ParseLiveSessionText(text, now)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if s.Authorization != "Bearer "+token || s.AppClient != testLiveAppClient || s.AppCode != testLiveAppCode {
			t.Errorf("%s mixed headers from another request: client=%q", name, s.AppClient)
		}
	}
	if _, err := ParseLiveSessionText(other+" ;\n"+other, now); !errors.Is(err, errLiveImportLegacy) {
		t.Fatalf("only old-host commands: %v", err)
	}
}

func TestSavingANewConnectionForgetsTheLastLiveFailure(t *testing.T) {
	noteQueueLiveResult(ErrLiveAuthRequired)
	if !queueLiveRecentlyFailed() {
		t.Fatal("failure was not remembered")
	}
	if saved, err := SaveLiveSession(testLiveSession(time.Now(), time.Hour)); !saved || err != nil {
		t.Fatalf("save = %v, %v", saved, err)
	}
	if queueLiveRecentlyFailed() {
		t.Fatal("a fresh connection must let the next plan use live data right away")
	}
}
