package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Ryujoxys/sushiro-overdose/internal/api"
)

// countingCollector counts rounds; a round reporting records really writes them, as the
// shared writer would.
func countingCollector(calls *int, result func() (int, error)) *QueueBaselineCollector {
	return &QueueBaselineCollector{collect: func(context.Context, QueueBaselineConfig) (int, error) {
		*calls++
		n, err := result()
		for i := 0; i < n; i++ {
			record := QueueBaselineRecord{StoreID: 1012, CollectedAt: time.Now().Format(time.RFC3339), WaitMinutes: 20, StoreStatus: "OPEN"}
			if writeErr := appendQueueBaselineRecords([]QueueBaselineRecord{record}); writeErr != nil {
				return 0, writeErr
			}
		}
		return n, err
	}}
}

func TestQueueCollectionBackoffSequence(t *testing.T) {
	want := []time.Duration{30 * time.Second, time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 15 * time.Minute, 15 * time.Minute}
	for i, delay := range want {
		if got := queueCollectionBackoff(i + 1); got != delay {
			t.Errorf("failure %d: %v, want %v", i+1, got, delay)
		}
	}
	if queueCollectionBackoff(100) != 15*time.Minute || queueCollectionBackoff(0) != 30*time.Second {
		t.Fatal("backoff bounds")
	}
}

func TestCollectorWithoutConnectionStopsWithoutRequests(t *testing.T) {
	publicCollectorFixture(t)
	useLiveSession(t, nil)
	t.Setenv("SUSHIRO_TOKEN", "")
	var requests atomic.Int32
	gatewayTestServer(t, func(w http.ResponseWriter, r *http.Request) { requests.Add(1) })
	attempts := 0
	c := &QueueBaselineCollector{collect: func(ctx context.Context, cfg QueueBaselineConfig) (int, error) {
		attempts++
		return collectQueueBaselineWithConfig(ctx, cfg)
	}}
	c.tick(context.Background())
	status := c.status()
	if attempts != 1 || !status.WaitingForConnection || status.LastErrorCode != liveErrAuthRequired || !strings.Contains(status.LastError, "连接") || status.NextAttemptAt == "" {
		t.Fatalf("missing connection not reported: attempts=%d %+v", attempts, status)
	}
	if strings.Contains(status.LastError, "VPN") || strings.Contains(status.LastError, "HTTP") || strings.Contains(status.LastError, "会自动重试") {
		t.Fatalf("status text: %q", status.LastError)
	}
	for i := 0; i < 3; i++ {
		c.tick(context.Background())
	}
	if attempts != 1 {
		t.Fatalf("stopped collector tried again %d times", attempts-1)
	}
	if requests.Load() != 0 {
		t.Fatalf("a process without a connection sent %d requests", requests.Load())
	}
	if _, err := os.Stat(queueBaselineRecordsPath()); !os.IsNotExist(err) {
		t.Fatal("wrote records without data")
	}
}

func TestCollectorAuthFailureWaitsForANewConnection(t *testing.T) {
	publicCollectorFixture(t)
	now := time.Now()
	first := testLiveSession(now.Add(-time.Minute), time.Hour)
	useLiveSession(t, &first)
	calls := 0
	fail := true
	c := countingCollector(&calls, func() (int, error) {
		if fail {
			return 0, &api.APIError{StatusCode: http.StatusUnauthorized, Body: `{"code":"E401"}`}
		}
		return 1, nil
	})
	c.tick(context.Background())
	state := loadQueueCollectionState()
	if calls != 1 || !state.WaitingForConnection || state.SessionMark != first.mark() || state.LastErrorCode != liveErrAuthRequired {
		t.Fatalf("401 did not stop collection: %+v", state)
	}
	if strings.Contains(state.LastError, "HTTP") || strings.Contains(state.LastError, "VPN") {
		t.Fatalf("status text: %q", state.LastError)
	}
	// Ticks with the same connection send nothing, even when the interval is due.
	state.LastAt = ""
	persistCollectorState(t, state)
	c.tick(context.Background())
	if calls != 1 {
		t.Fatal("retried with the connection that was just rejected")
	}
	second := testLiveSession(now, time.Hour)
	useLiveSession(t, &second)
	if status := sharedQueueCollectionStatus(time.Now()); status.WaitingForConnection || status.LastError != "" {
		t.Fatalf("status still asks for a connection after reconnecting: %+v", status)
	}
	fail = false
	c.tick(context.Background())
	state = loadQueueCollectionState()
	if calls != 2 || state.WaitingForConnection || state.LastErrorCode != "" || state.NextAttemptAt != "" || state.ConsecutiveFailures != 0 || state.SessionMark != "" {
		t.Fatalf("new connection did not resume collection: calls=%d %+v", calls, state)
	}
}

func TestCollectorNoSessionStopDoesNotBlockSessionHolder(t *testing.T) {
	publicCollectorFixture(t)
	now := time.Now()
	persistCollectorState(t, queueCollectionState{PublicQueueCollectionStatus: PublicQueueCollectionStatus{
		WaitingForConnection: true, LastErrorCode: liveErrAuthRequired, LastError: liveAuthRequiredText,
		NextAttemptAt: now.Add(6 * time.Hour).Format(time.RFC3339),
	}, ConsecutiveFailures: 3})
	// Without a connection this process stays stopped.
	useLiveSession(t, nil)
	calls := 0
	c := countingCollector(&calls, func() (int, error) { return 1, nil })
	c.tick(context.Background())
	if calls != 0 {
		t.Fatal("a process without a connection ignored the stop marker")
	}
	session := testLiveSession(now, time.Hour)
	useLiveSession(t, &session)
	c.tick(context.Background())
	if calls != 1 || loadQueueCollectionState().WaitingForConnection {
		t.Fatalf("a connected process was blocked by a no-connection stop marker: calls=%d", calls)
	}
	if state := loadQueueCollectionState(); state.LiveHolderPID != os.Getpid() || state.LiveHolderAt == "" {
		t.Fatalf("connected collector not recorded as holder: %+v", state)
	}
}

func TestCollectorAuthStopRetriesAfterSixHours(t *testing.T) {
	publicCollectorFixture(t)
	useLiveSession(t, nil)
	persistCollectorState(t, queueCollectionState{PublicQueueCollectionStatus: PublicQueueCollectionStatus{
		WaitingForConnection: true, LastErrorCode: liveErrAuthRequired,
		NextAttemptAt: time.Now().Add(-time.Minute).Format(time.RFC3339),
	}})
	calls := 0
	c := countingCollector(&calls, func() (int, error) { return 0, ErrLiveAuthRequired })
	c.tick(context.Background())
	state := loadQueueCollectionState()
	next, _ := time.Parse(time.RFC3339, state.NextAttemptAt)
	if calls != 1 || !state.WaitingForConnection || next.Sub(time.Now()) < 5*time.Hour {
		t.Fatalf("six-hour retry: calls=%d %+v", calls, state)
	}
	// A clock that jumped backwards does not keep the stop marker forever.
	state.NextAttemptAt = time.Now().Add(30 * time.Hour).Format(time.RFC3339)
	persistCollectorState(t, state)
	c.tick(context.Background())
	if calls != 2 {
		t.Fatal("stop marker from the future blocked collection")
	}
}

func TestNoConnectionProcessYieldsToConnectedCollector(t *testing.T) {
	publicCollectorFixture(t)
	useLiveSession(t, nil)
	holder := os.Getppid()
	persistCollectorState(t, queueCollectionState{LiveHolderPID: holder, LiveHolderAt: time.Now().Format(time.RFC3339)})
	calls := 0
	c := countingCollector(&calls, func() (int, error) { return 0, ErrLiveAuthRequired })
	c.tick(context.Background())
	if state := loadQueueCollectionState(); calls != 0 || state.WaitingForConnection || state.LastError != "" {
		t.Fatalf("background process overwrote a connected collector's status: calls=%d %+v", calls, state)
	}
	state := loadQueueCollectionState()
	state.LiveHolderAt = time.Now().Add(-3 * time.Minute).Format(time.RFC3339)
	persistCollectorState(t, state)
	c.tick(context.Background())
	if calls != 1 || !loadQueueCollectionState().WaitingForConnection {
		t.Fatal("background process kept yielding to a holder that stopped")
	}
}

func TestCollectorPartialFailureDoesNotBackOff(t *testing.T) {
	publicCollectorFixture(t)
	calls := 0
	c := countingCollector(&calls, func() (int, error) {
		return 1, &api.APIError{StatusCode: http.StatusBadGateway, Body: "x"}
	})
	c.tick(context.Background())
	state := loadQueueCollectionState()
	if calls != 1 || state.LastError == "" || state.NextAttemptAt != "" || state.ConsecutiveFailures != 0 || state.WaitingForConnection || state.LastAt == "" {
		t.Fatalf("partial round: %+v", state)
	}
}

func TestPublicServiceStatusReportsConnectionState(t *testing.T) {
	publicCollectorFixture(t)
	useLiveSession(t, nil)
	c := &QueueBaselineCollector{collect: func(context.Context, QueueBaselineConfig) (int, error) { return 0, ErrLiveAuthRequired }}
	c.tick(context.Background())
	rr := httptest.NewRecorder()
	handlePublicQueueService(rr, httptest.NewRequest(http.MethodGet, "/api/queue/service", nil))
	var body struct {
		State map[string]any `json:"state"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.State["waiting_for_connection"] != true || body.State["last_error_code"] != liveErrAuthRequired || body.State["next_attempt_at"] == "" {
		t.Fatalf("service state: %v", body.State)
	}
}

func TestCollectionRoundStopsAtTheFirstRejectedRequest(t *testing.T) {
	cfg := publicCollectorFixture(t)
	cfg.StoreIDs = []string{"1012", "3006", "3014"}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	client := &QueueLiveClient{baseURL: server.URL, httpClient: server.Client()}
	n, err := collectQueueBaselineWithClient(context.Background(), cfg, client)
	if n != 0 || liveErrorCode(err) != liveErrAuthRequired || calls.Load() != 1 {
		t.Fatalf("round continued after a rejection: n=%d calls=%d %v", n, calls.Load(), err)
	}
	if _, err := os.Stat(queueBaselineRecordsPath()); !os.IsNotExist(err) {
		t.Fatal("a rejected round wrote records")
	}
}

func TestCollectionRoundWritesNothingForMalformedStores(t *testing.T) {
	cfg := publicCollectorFixture(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","data":{"id":1012,"storeStatus":"OPEN","wait":30}}`))
	}))
	defer server.Close()
	client := &QueueLiveClient{baseURL: server.URL, httpClient: server.Client()}
	n, err := collectQueueBaselineWithClient(context.Background(), cfg, client)
	if n != 0 || !errors.Is(err, errLiveStoreFormat) {
		t.Fatalf("malformed store: n=%d %v", n, err)
	}
	for _, path := range []string{queueBaselineRecordsPath(), queueObservationPath()} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("malformed reply wrote %s", filepath.Base(path))
		}
	}
}

func TestBackendNoLongerPromisesAutomaticRetry(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "会自动重试") {
			t.Fatalf("%s still promises an automatic retry", file)
		}
	}
}

func TestCollectionIntervalNeverBelowOneMinute(t *testing.T) {
	publicCollectorFixture(t)
	for _, minutes := range []int{0, 1, -5} {
		if got := queueBaselineIntervalSeconds(QueueBaselineConfig{IntervalMinutes: minutes}, nil); got < 60 {
			t.Fatalf("interval %d minutes => %d seconds", minutes, got)
		}
	}
}

func TestCollectorStatusDropsConnectionErrorOnceReconnected(t *testing.T) {
	publicCollectorFixture(t)
	useLiveSession(t, nil)
	c := &QueueBaselineCollector{collect: func(context.Context, QueueBaselineConfig) (int, error) { return 0, ErrLiveAuthRequired }}
	c.tick(context.Background())
	if status := c.status(); !status.WaitingForConnection || status.LastError == "" {
		t.Fatalf("missing connection not reported: %+v", status)
	}
	session := testLiveSession(time.Now(), time.Hour)
	useLiveSession(t, &session)
	if status := c.status(); status.WaitingForConnection || status.LastError != "" || status.LastErrorCode != "" {
		t.Fatalf("status still asks for a connection right after connecting: %+v", status)
	}
}

func TestBackgroundCopyNoLongerPromisesCredentialFreeRecording(t *testing.T) {
	for _, file := range []string{"queue_service.go", "queue_model.go", "queue_baseline.go", "queue_collection_state.go"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, claim := range []string{"不需要通行证", "无需通行证", "不需要认证"} {
			if strings.Contains(string(data), claim) {
				t.Fatalf("%s still says %q although live data now needs a connection", file, claim)
			}
		}
	}
}
