package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/api"
	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

func reliabilityHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Cleanup(func() { clearWebSettings(); resetAuthHealth() })
}

func reliabilityTokens() *CapturedTokens {
	return &CapturedTokens{XAppCode: "app", QueryAuth: "query", ReservationAuth: "reservation", UserAgent: "ua", Referer: "ref", WechatID: "wechat", PhoneNumber: "13800138000", StoreIDs: []string{"1012"}}
}

func TestTicketStatusReadDoesNotMutatePlanOrNotify(t *testing.T) {
	reliabilityHome(t)
	plan := NetTicketPlan{Enabled: true, StoreID: "3006", Status: "armed", TargetTime: "1800"}
	if err := SaveNetTicketPlan(plan); err != nil {
		t.Fatal(err)
	}
	before := LoadNetTicketPlan()
	for _, payload := range []string{
		`{"netTicket":{"ticketId":123,"number":"1050","storeId":"1012","status":"WAITING"}}`,
		`{"reservationTicket":{"ticketId":456,"number":"99","storeId":"3006","start":"193000","queueDate":"20260916","status":"WAITING"}}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/ticket/status") {
				t.Errorf("unexpected call: %s %s", r.Method, r.URL.Path)
			}
			w.Write([]byte(payload))
		}))
		client := NewClient(Settings{BaseURL: server.URL, WechatID: "wechat"})
		client.SetHTTPClient(server.Client())
		notifications := notificationCountForTest("desktop")
		for i := 0; i < 3; i++ {
			rr := httptest.NewRecorder()
			writeQueueTicketStatus(rr, httptest.NewRequest(http.MethodGet, "/api/queue/ticket/status", nil), client)
			if rr.Code != http.StatusOK {
				t.Fatalf("%d %s", rr.Code, rr.Body.String())
			}
			if strings.Contains(payload, "reservationTicket") && !strings.Contains(rr.Body.String(), `"ticket":null`) {
				t.Fatal("reservation reported as queue ticket")
			}
		}
		server.Close()
		if !reflect.DeepEqual(before, LoadNetTicketPlan()) {
			t.Fatal("read changed unrelated plan")
		}
		if notificationCountForTest("desktop") != notifications {
			t.Fatal("read sent notification")
		}
		if _, err := os.Stat(filepath.Join(AppDirPath(), ".sushiro_state.json")); !os.IsNotExist(err) {
			t.Fatal("read wrote booking state")
		}
	}
}

func TestStoppingKeepsOwnershipUntilWorkerExits(t *testing.T) {
	reliabilityHome(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	e := &BookingEngine{state: EngineState{Status: EngineCapturing}, cancel: cancel, done: done}
	e.stopAndWait(time.Millisecond)
	if ctx.Err() == nil || e.GetState().Status != EngineStopping || e.done != done {
		t.Fatal("lost stopping ownership")
	}
	e.setState(EngineCapturing, "late installer result")
	if e.GetState().Status != EngineStopping {
		t.Fatal("late worker reset stopping status")
	}
	if err := e.StartCapture(); err == nil {
		t.Fatal("restarted before exit")
	}
	e.finishRun(make(chan struct{}))
	if e.done != done {
		t.Fatal("old finalizer cleared current run")
	}
	e.finishRun(done)
	select {
	case <-done:
	default:
		t.Fatal("worker completion not signaled")
	}
	if e.GetState().Status != EngineIdle || e.done != nil {
		t.Fatal("run did not finish")
	}
}

func TestOldMobileSessionCannotSaveOrStopNewSession(t *testing.T) {
	reliabilityHome(t)
	oldTokens, currentTokens := reliabilityTokens(), reliabilityTokens()
	m := &mobileAuthCaptureManager{token: "new", tokens: currentTokens, generation: authGeneration()}
	m.finish("old", oldTokens)
	m.stopSession("old", "old timeout")
	if m.token != "new" || m.tokens != currentTokens {
		t.Fatal("old session changed new session")
	}
	if _, err := LoadLocalConfig(); err == nil {
		t.Fatal("old session saved credentials")
	}
}

func TestAuthResetInvalidatesPendingMobileCommit(t *testing.T) {
	reliabilityHome(t)
	tokens := reliabilityTokens()
	old := mobileAuthCapture
	m := &mobileAuthCaptureManager{token: "before-reset", tokens: tokens, generation: authGeneration()}
	mobileAuthCapture = m
	t.Cleanup(func() { mobileAuthCapture = old })
	rr := httptest.NewRecorder()
	handleAuthReset(rr, httptest.NewRequest(http.MethodPost, "/api/auth/reset", nil))
	m.finish("before-reset", tokens)
	if rr.Code != 200 || m.token != "" || m.tokens != nil {
		t.Fatal("reset left capture active")
	}
	if _, err := LoadLocalConfig(); err == nil {
		t.Fatal("late completion restored deleted credentials")
	}
	if getWebClient() != nil || getAuthHealth().Status != authHealthUnknown {
		t.Fatal("late completion restored client/health")
	}
}

func TestMobileSaveIsUnverifiedAndStaleProbeCannotOverwrite(t *testing.T) {
	reliabilityHome(t)
	generation := authGeneration()
	tokens := reliabilityTokens()
	m := &mobileAuthCaptureManager{token: "current", tokens: tokens, generation: generation}
	m.finish("current", tokens)
	if !m.saved || !HasValidConfig() {
		t.Fatal("credentials not saved")
	}
	if getAuthHealth().Status != authHealthUnknown {
		t.Fatal("saving marked credentials valid")
	}
	if withAuthGeneration(generation, func() { markAuthHealthy() }) {
		t.Fatal("obsolete probe accepted")
	}
}

func TestAuthProbeHealthDoesNotPromoteSkippedOrRejectedAuth(t *testing.T) {
	reliabilityHome(t)
	resetAuthHealth()
	applyAuthProbeHealth(AuthProbeReport{OK: true})
	if getAuthHealth().Status != authHealthUnknown {
		t.Fatal("query-only success marked healthy")
	}
	applyAuthProbeHealth(AuthProbeReport{Results: []AuthProbeResult{{Status: 401}}})
	if getAuthHealth().Status != authHealthStale {
		t.Fatal("401 not stale")
	}
	applyAuthProbeHealth(AuthProbeReport{OK: true})
	if getAuthHealth().Status != authHealthStale {
		t.Fatal("skipped check cleared rejection")
	}
	applyAuthProbeHealth(AuthProbeReport{Authenticated: true})
	if getAuthHealth().Status != authHealthOK {
		t.Fatal("authenticated success not recognized")
	}
}

func TestReservationsProbeRejectsUnknownSuccessBody(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `{"error":"unauthorized"}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		result := probeReservations(context.Background(), server.Client(), Settings{BaseURL: server.URL})
		server.Close()
		if result.OK {
			t.Fatalf("accepted unknown body: %s", body)
		}
	}
}

func TestPublicCollectionEvaluatesAlertsAfterTicketIssuedWithoutAuth(t *testing.T) {
	reliabilityHome(t)
	if err := SaveNetTicketPlan(NetTicketPlan{Status: "success", FiredDate: time.Now().Format("2006-01-02")}); err != nil {
		t.Fatal(err)
	}
	if !netTicketIssuedToday(time.Now()) {
		t.Fatal("fixture did not issue ticket")
	}
	rule := QueueAlertRule{StoreID: "1012", Type: queueAlertCalledReach, TargetNo: 1078, NotifyAtNo: 1050, Enabled: true}
	if err := SaveQueueAlertConfig(QueueAlertConfig{Rules: []QueueAlertRule{rule}}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "getStoreById") {
			t.Errorf("unexpected public path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"id":1012,"name":"test","storeStatus":"OPEN","wait":30,"groupQueues":{"boothQueue":["1051"]}}`))
	}))
	defer server.Close()
	client := &QueueLiveClient{baseURL: server.URL, httpClient: server.Client()}
	before := notificationCountForTest("desktop")
	for i := 0; i < 2; i++ {
		count, err := collectQueueBaselineWithClient(context.Background(), QueueBaselineConfig{Enabled: true}, client)
		if err != nil || count != 1 {
			t.Fatalf("collection = %d, %v", count, err)
		}
	}
	if notificationCountForTest("desktop") != before+1 {
		t.Fatal("public collector did not notify exactly once")
	}
	if HasValidConfig() {
		t.Fatal("public collection unexpectedly required credentials")
	}
}

func TestWaitAlertPersistsArmedStateBeforeThreshold(t *testing.T) {
	reliabilityHome(t)
	rule := QueueAlertRule{StoreID: "1012", Type: queueAlertWaitBelow, WaitMinutes: 15, Enabled: true}
	if err := SaveQueueAlertConfig(QueueAlertConfig{Rules: []QueueAlertRule{rule}}); err != nil {
		t.Fatal(err)
	}
	before := notificationCountForTest("desktop")
	evaluateQueueAlerts(context.Background(), QueueObservation{StoreID: "1012", WaitMinutes: 60}, "test")
	evaluateQueueAlerts(context.Background(), QueueObservation{StoreID: "1012", WaitMinutes: 10}, "test")
	evaluateQueueAlerts(context.Background(), QueueObservation{StoreID: "1012", WaitMinutes: 10}, "test")
	if notificationCountForTest("desktop") != before+1 {
		t.Fatal("armed state lost or notification duplicated")
	}
}

func TestBundledMCPWorksWithoutRepository(t *testing.T) {
	reliabilityHome(t)
	dir := findMCPDir()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("discovery wrote files")
	}
	got, err := materializeMCPAssets()
	if err != nil || got != dir {
		t.Fatalf("extract: %s %v", got, err)
	}
	for _, file := range []string{"pyproject.toml", "mcp_server/__main__.py", "mcp_server/server.py", "docs/faq.md"} {
		if info, err := os.Stat(filepath.Join(dir, file)); err != nil || info.Size() == 0 {
			t.Fatalf("missing %s", file)
		}
	}
	if _, err := materializeMCPAssets(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mcp_server/turso.py")); !os.IsNotExist(err) {
		t.Fatal("retired provider included")
	}
	if _, err := os.Stat(filepath.Join(dir, "venv")); !os.IsNotExist(err) {
		t.Fatal("test installed Python dependencies")
	}
}

func TestVerifyResultIsReadOnlyWithoutCredentials(t *testing.T) {
	reliabilityHome(t)
	rr := httptest.NewRecorder()
	handleAuthVerify(rr, httptest.NewRequest(http.MethodPost, "/api/auth/verify", nil))
	var result AuthVerifyResult
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Valid || result.OK || result.Method != "read_only" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestVerificationOnlyUsesReadEndpoints(t *testing.T) {
	reliabilityHome(t)
	for _, status := range []int{200, 401, 403, 404, 500} {
		calls := 0
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			code, body := 200, "{}"
			switch r.URL.Path {
			case "/wechat/api/2.0/getStoreById":
				body = `{"id":1012,"name":"test"}`
			case "/wechat/api/2.0/store/timeslots":
				body = "[]"
			case "/wechat/api_auth/2.0/ticketing/getReservations":
				if r.Method != "POST" {
					t.Error(r.Method)
				}
				code, body = status, "[]"
			default:
				t.Fatalf("verification called a mutation or unknown endpoint: %s", r.URL.Path)
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		})}
		report := runAuthProbeWithClient(context.Background(), "", reliabilityTokens(), UserPreferences{}, client)
		if calls != 3 || report.Authenticated != (status == 200) {
			t.Fatalf("status=%d calls=%d report=%+v", status, calls, report)
		}
	}
}

func TestCaptureDeadlineHasMissingFieldsAndRecovery(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	e := &BookingEngine{state: EngineState{Status: EngineCapturing}}
	e.reportCaptureTimeout(ctx, NewCapturedTokens())
	state := e.GetState()
	if state.Status != EngineError || state.ErrorKind != ErrKindCaptureTimeout || !strings.Contains(state.Message, "缺少") || !strings.Contains(state.Message, "导入") {
		t.Fatalf("%+v", state)
	}
}

func TestBeginRunKeepsOneCaptureOwner(t *testing.T) {
	reliabilityHome(t)
	e := &BookingEngine{state: EngineState{Status: EngineIdle}}
	ready := make(chan struct{})
	results := make(chan bool, 2)
	for range 2 {
		go func() {
			<-ready
			_, _, _, _, err := e.beginRun("starting")
			results <- err == nil
		}()
	}
	close(ready)
	a, b := <-results, <-results
	if a == b {
		t.Fatalf("exactly one start should win: %v %v", a, b)
	}
	e.mu.RLock()
	done, status := e.done, e.state.Status
	e.mu.RUnlock()
	if done == nil || status != EngineCapturing {
		t.Fatal("capture not bound to winning run")
	}
	e.finishRun(done)
}
