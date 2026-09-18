package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
	"github.com/Ryujoxys/sushiro-overdose/internal/platform"
)

func publicCollectorFixture(t *testing.T) QueueBaselineConfig {
	t.Helper()
	reliabilityHome(t)
	previousEngine := engine
	engine = &BookingEngine{state: EngineState{Status: EngineIdle}}
	t.Cleanup(func() { engine = previousEngine })
	cfg := QueueBaselineConfig{Enabled: true, IntervalMinutes: 5, StoreIDs: []string{"1012"}}
	if err := SaveQueueBaselineConfig(cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func persistCollectorState(t *testing.T, state queueCollectionState) {
	t.Helper()
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := AtomicWriteFile(queueCollectionStatePath(), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPublicCollectorsShareCadenceWithoutCredentials(t *testing.T) {
	cfg := publicCollectorFixture(t)
	calls := 0
	collect := func(ctx context.Context, cfg QueueBaselineConfig) (int, error) {
		calls++
		record := QueueBaselineRecord{StoreID: atoi(cfg.StoreIDs[0]), CollectedAt: time.Now().Format(time.RFC3339), WaitMinutes: 20}
		return 1, appendQueueBaselineRecords([]QueueBaselineRecord{record})
	}
	front, background := &QueueBaselineCollector{collect: collect}, &QueueBaselineCollector{collect: collect}
	front.tick(context.Background())
	background.tick(context.Background())
	if calls != 1 {
		t.Fatalf("duplicate collection: %d", calls)
	}
	status := background.status()
	if !status.Running || status.LastAt == "" || status.LastError != "" || status.IntervalSeconds != 300 {
		t.Fatalf("shared status: %+v", status)
	}
	if _, err := os.Stat(LocalConfigPath()); !os.IsNotExist(err) {
		t.Fatal("public collection created credentials")
	}
	model, err := loadLocalQueueModel()
	if err != nil || model.SampleCount != 1 {
		t.Fatalf("model not generated: %+v %v", model, err)
	}
	cfg.StoreIDs = []string{"3006"}
	if err := SaveQueueBaselineConfig(cfg); err != nil {
		t.Fatal(err)
	}
	background.tick(context.Background())
	if calls != 2 {
		t.Fatal("changed stores must be sampled without waiting for the old interval")
	}
	state := loadQueueCollectionState()
	state.LastAt = time.Now().Add(time.Hour).Format(time.RFC3339)
	persistCollectorState(t, state)
	front.tick(context.Background())
	if calls != 3 {
		t.Fatal("clock rollback stalled collection")
	}
}

func TestPublicCollectorYieldsResumesAndRespectsCrossProcessLock(t *testing.T) {
	cfg := publicCollectorFixture(t)
	calls := 0
	c := &QueueBaselineCollector{collect: func(context.Context, QueueBaselineConfig) (int, error) {
		calls++
		return 0, errors.New("fixture network failure")
	}}
	lock, err := platform.TryFileLock(filepath.Join(AppDirPath(), "queue_collection.lock"))
	if err != nil {
		t.Fatal(err)
	}
	c.tick(context.Background())
	if calls != 0 {
		t.Fatal("ignored another collector's lock")
	}
	lock.Close()
	netTicketMu.Lock()
	c.tick(context.Background())
	netTicketMu.Unlock()
	if calls != 0 || c.status().PausedReason == "" {
		t.Fatal("ticket submission must pause public collection")
	}
	engine.state.Status = EngineCapturing
	c.tick(context.Background())
	if calls != 0 || c.status().PausedReason == "" {
		t.Fatal("capture must pause public collection")
	}
	engine.state.Status = EngineIdle
	c.tick(context.Background())
	if calls != 1 || c.status().LastError == "" {
		t.Fatal("collector did not resume or hid a failed request")
	}
	c.tick(context.Background())
	if calls != 2 {
		t.Fatal("a wholly failed first round should retry next tick")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.tick(ctx)
	if calls != 2 || c.status().Running {
		t.Fatal("canceled collector made a request or reports running")
	}
	cfg.Enabled = false
	if err := SaveQueueBaselineConfig(cfg); err != nil {
		t.Fatal(err)
	}
	c.tick(context.Background())
	if calls != 2 || c.status().Running {
		t.Fatal("disabled collector made a request or reports running")
	}
}

func TestPublicCollectorHeartbeatAndReminderCadence(t *testing.T) {
	publicCollectorFixture(t)
	if err := SaveQueueAlertConfig(QueueAlertConfig{Rules: []QueueAlertRule{{Enabled: true, StoreID: "3006", Type: queueAlertCalledReach, TargetNo: 100, NotifyAtNo: 90}}}); err != nil {
		t.Fatal(err)
	}
	state := queueCollectionState{PID: os.Getpid(), HeartbeatAt: time.Now().Format(time.RFC3339), PublicQueueCollectionStatus: PublicQueueCollectionStatus{Running: true}}
	persistCollectorState(t, state)
	status := sharedQueueCollectionStatus(time.Now())
	if !status.Running || status.IntervalSeconds != 60 || !reflect.DeepEqual(status.StoreIDs, []string{"1012", "3006"}) {
		t.Fatalf("reminder service status: %+v", status)
	}
	state.HeartbeatAt = time.Now().Add(-3 * time.Minute).Format(time.RFC3339)
	persistCollectorState(t, state)
	if sharedQueueCollectionStatus(time.Now()).Running {
		t.Fatal("stale heartbeat reported as active")
	}
}

func TestPublicAutoStartUsesInjectedActionsAndReportsPartialFailure(t *testing.T) {
	publicCollectorFixture(t)
	var calls []string
	actions := publicQueueServiceActions{
		install: func() error { calls = append(calls, "install"); return nil },
		start:   func() error { calls = append(calls, "start"); return nil },
		remove:  func() error { calls = append(calls, "remove"); return nil },
	}
	if err := configurePublicQueueAutoStart(true, actions); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"install", "start"}) || !LoadQueueBaselineConfig().Enabled {
		t.Fatalf("unexpected enable actions: %v", calls)
	}
	calls = nil
	if err := configurePublicQueueAutoStart(false, actions); err != nil || !reflect.DeepEqual(calls, []string{"remove"}) {
		t.Fatalf("disable must not start a service: %v %v", calls, err)
	}
	calls = nil
	actions.install = func() error { return errors.New("fixture registration failure") }
	if err := configurePublicQueueAutoStart(true, actions); err == nil || len(calls) != 0 {
		t.Fatal("installation failure must not launch a fallback child")
	}
	actions.install = func() error { return nil }
	actions.start = func() error { return errors.New("fixture start failure") }
	if err := configurePublicQueueAutoStart(true, actions); err == nil || !strings.Contains(err.Error(), "自启动已配置") {
		t.Fatal("partial configuration must be reported")
	}
	if err := SaveQueueBaselineConfig(QueueBaselineConfig{Enabled: false, UsePreferenceStores: true}); err != nil {
		t.Fatal(err)
	}
	calls = nil
	actions.install = func() error { calls = append(calls, "install"); return nil }
	if err := configurePublicQueueAutoStart(true, actions); err == nil || len(calls) != 0 {
		t.Fatal("no-store setup must not register a native service")
	}
}

func TestPublicServiceCanceledRunReleasesOwnershipWithoutRequests(t *testing.T) {
	publicCollectorFixture(t)
	old := queueBaselineCollector
	queueBaselineCollector = &QueueBaselineCollector{collect: func(context.Context, QueueBaselineConfig) (int, error) {
		t.Error("canceled service attempted network collection")
		return 0, nil
	}}
	t.Cleanup(func() { queueBaselineCollector = old })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runPublicQueueService(ctx); err != nil {
		t.Fatal(err)
	}
	if readSamplingPID() != "" || publicQueueServiceRunning() {
		t.Fatal("service left a PID or live ownership behind")
	}
	lock, err := platform.TryFileLock(publicQueueServiceLockPath())
	if err != nil {
		t.Fatal("service leaked lifetime lock", err)
	}
	lock.Close()
}

func TestPublicServiceEntryCannotStartAuthenticatedWork(t *testing.T) {
	for _, path := range []string{"queue_service.go", "queue_collection_state.go", "queue_baseline.go", "queue_model.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"netTicketSched.Start(", "netTicketTick(", "sampler.Start(", "CreateNetTicket(", "LoadLocalConfig(", "mobileAuthCapture.status("} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("public service reaches private work: %s in %s", forbidden, path)
			}
		}
	}
}

func TestPublicServiceDoesNotReplaceALiveLegacyPID(t *testing.T) {
	publicCollectorFixture(t)
	writeSamplingPID(os.Getpid())
	before := readSamplingPID()
	if err := startPublicQueueDaemon(); err == nil {
		t.Fatal("started a real process instead of rejecting the unknown live PID")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runPublicQueueService(ctx); err == nil {
		t.Fatal("new service took over a live legacy PID")
	}
	if readSamplingPID() != before {
		t.Fatal("overwrote the legacy PID marker")
	}
}

func TestPublicCollectionPreservesPartialRoundWhenMainFlowStarts(t *testing.T) {
	cfg := publicCollectorFixture(t)
	cfg.StoreIDs = []string{"1012", "3006"}
	calls := 0
	client := &QueueLiveClient{baseURL: "https://example.invalid", httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodGet || !strings.Contains(r.URL.Path, "getStoreById") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		engine.mu.Lock()
		engine.state.Status = EngineCapturing
		engine.mu.Unlock()
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":1012,"name":"test","storeStatus":"OPEN","wait":30}`)), Header: make(http.Header)}, nil
	})}}
	n, err := collectQueueBaselineWithClient(context.Background(), cfg, client)
	if n != 1 || calls != 1 || err == nil {
		t.Fatalf("round did not yield: %d requests, %d records, %v", calls, n, err)
	}
	records, err := parseJSONLFile[QueueBaselineRecord](queueBaselineRecordsPath(), normalizeQueueBaselineRecordForRead)
	if err != nil || len(records) != 1 || records[0].StoreID != 1012 {
		t.Fatalf("partial round discarded: %+v %v", records, err)
	}
}

func TestQuickTicketRejectsBusyCaptureBeforeReadingCredentials(t *testing.T) {
	publicCollectorFixture(t)
	engine.state.Status = EngineCapturing
	rr := httptest.NewRecorder()
	handleQueueTicket(rr, httptest.NewRequest(http.MethodPost, "/api/queue/ticket", strings.NewReader(`{"store":"1012"}`)))
	if rr.Code != http.StatusConflict {
		t.Fatalf("capture allowed ticket submission: %d %s", rr.Code, rr.Body.String())
	}
}

func TestPublicServiceReadAndPauseNeverRequireCredentials(t *testing.T) {
	publicCollectorFixture(t)
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/queue/service", nil),
		httptest.NewRequest(http.MethodPost, "/api/queue/service", strings.NewReader(`{"action":"pause"}`)),
	} {
		rr := httptest.NewRecorder()
		handlePublicQueueService(rr, request)
		if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"model"`) {
			t.Fatalf("%d %s", rr.Code, rr.Body.String())
		}
	}
	if LoadQueueBaselineConfig().Enabled {
		t.Fatal("pause not persisted")
	}
}

func TestQuickTicketRequestValidatesWithoutChangingPreferences(t *testing.T) {
	reliabilityHome(t)
	base := Settings{Adult: 2, Child: 0, TableType: "T"}
	for _, tc := range []struct {
		body string
		ok   bool
	}{
		{`{"store":"1012"}`, true},
		{`{"adult":1,"child":2,"table_type":"C"}`, true},
		{`{"adult":-1}`, false}, {`{"child":11}`, false},
		{`{"adult":0,"child":0}`, false}, {`{"table_type":"bad"}`, false},
		{`{"adult":1.5}`, false}, {`{"child":"2"}`, false},
	} {
		var request queueTicketRequest
		err := json.Unmarshal([]byte(tc.body), &request)
		if err == nil {
			_, err = request.apply(base)
		}
		if (err == nil) != tc.ok {
			t.Errorf("%s: %v", tc.body, err)
		}
	}
	if base.Adult != 2 || base.Child != 0 || base.TableType != "T" {
		t.Fatal("request changed shared settings")
	}
}
