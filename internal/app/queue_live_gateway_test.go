package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// gatewayTestServer points the gateway client at a local server for one test.
func gatewayTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	previous := queueLiveGatewayBaseURL
	queueLiveGatewayBaseURL = server.URL + "/gateway/wechat/api/2.0"
	t.Cleanup(func() { queueLiveGatewayBaseURL = previous })
	return server
}

func TestNewQueueLiveClientWithoutConnectionSendsNothing(t *testing.T) {
	t.Setenv("SUSHIRO_TOKEN", "")
	var calls atomic.Int32
	gatewayTestServer(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })
	now := time.Now()
	expired := testLiveSession(now.Add(-2*time.Hour), time.Hour)
	previous := queueLiveSessionProvider
	t.Cleanup(func() { queueLiveSessionProvider = previous })
	for name, provider := range map[string]func() (*LiveSession, error){
		"no provider":   nil,
		"no session":    func() (*LiveSession, error) { return nil, nil },
		"read error":    func() (*LiveSession, error) { return nil, errors.New("damaged") },
		"expired":       func() (*LiveSession, error) { return &expired, nil },
		"missing parts": func() (*LiveSession, error) { return &LiveSession{Authorization: "Bearer a.b.c", CapturedAt: now}, nil },
	} {
		queueLiveSessionProvider = provider
		client := NewQueueLiveClient()
		ctx := context.Background()
		_, errStore := client.GetStore(ctx, "1012")
		_, errList := client.ListStores(ctx, QueueLiveStoreQuery{})
		_, errAreas := client.ListAreas(ctx)
		_, errAll := client.CachedAllStores(ctx)
		_, errFetch := client.fetchStore(ctx, "1012")
		for _, err := range []error{errStore, errList, errAreas, errAll, errFetch} {
			if !errors.Is(err, ErrLiveAuthRequired) {
				t.Fatalf("%s: %v", name, err)
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("a client without a connection sent %d requests", calls.Load())
	}
}

func TestNewQueueLiveClientLegacyOnlyWithExplicitToken(t *testing.T) {
	useLiveSession(t, nil)
	t.Setenv("SUSHIRO_TOKEN", "debug-token")
	client := NewQueueLiveClient()
	if client.authErr != nil || client.source != queueLiveSourceLegacy || client.baseURL != queueLiveBaseURL || client.token != "debug-token" || client.headers != nil {
		t.Fatalf("legacy client: %+v", client)
	}
}

func TestGatewayClientSendsSessionHeadersAndHardensDecoding(t *testing.T) {
	t.Setenv("SUSHIRO_TOKEN", "should-not-be-used")
	now := time.Now()
	session := testLiveSession(now, time.Hour)
	useLiveSession(t, &session)
	var calls atomic.Int32
	gatewayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/gateway/wechat/api/2.0/getStoreById" || r.URL.RawQuery != "storeId=1012" {
			t.Errorf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		for name, want := range map[string]string{
			"Authorization": session.Authorization, "X-App-Client": testLiveAppClient, "X-App-Code": testLiveAppCode,
			"User-Agent": session.UserAgent, "Referer": session.Referer, "Content-Type": "application/json", "Accept": "*/*",
		} {
			if got := r.Header.Get(name); got != want {
				t.Errorf("%s = %q, want %q", name, got, want)
			}
		}
		w.Write([]byte(`{"id":"1012","name":"南山店","storeStatus":"OPEN","wait":25,"groupQueuesCount":8,
			"groupQueues":{"boothQueue":"819","mixedQueue":[820,"826"],"counterQueue":null,"reservationQueue":["7261"]}}`))
	})
	client := NewQueueLiveClient()
	if client.source != queueLiveSourceGateway {
		t.Fatalf("client source = %q", client.source)
	}
	store, err := client.GetStore(context.Background(), "1012")
	if err != nil {
		t.Fatal(err)
	}
	if store.ID != 1012 || store.Source != queueLiveSourceGateway || store.ObservedAt.IsZero() || store.GroupQueues.CurrentCalledNo() != 826 {
		t.Fatalf("store = %+v", store)
	}
	if !reflect.DeepEqual(store.GroupQueues.BoothQueue, []string{"819"}) || !reflect.DeepEqual(store.GroupQueues.MixedQueue, []string{"820", "826"}) {
		t.Fatalf("queues = %+v", store.GroupQueues)
	}
	// The panel, advisor and store name lookups of one page refresh share one request.
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := NewQueueLiveClient().GetStore(context.Background(), "1012"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("cached reads sent %d requests", calls.Load())
	}
	// The collector always reads fresh data.
	if _, err := client.fetchStore(context.Background(), "1012"); err != nil || calls.Load() != 2 {
		t.Fatalf("collector read: %v calls=%d", err, calls.Load())
	}
}

func TestGatewayConcurrentReadsShareOneRequest(t *testing.T) {
	now := time.Now()
	session := testLiveSession(now, time.Hour)
	useLiveSession(t, &session)
	var calls atomic.Int32
	release := make(chan struct{})
	gatewayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		<-release
		w.Write([]byte(`{"id":3006,"name":"太阳宫店","storeStatus":"OPEN"}`))
	})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if store, err := NewQueueLiveClient().GetStore(context.Background(), "3006"); err != nil || store.ID != 3006 {
				t.Errorf("shared read: %+v %v", store, err)
			}
		}()
	}
	deadline := time.Now().Add(2 * time.Second)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond) // let the other readers join the in-flight request
	close(release)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("concurrent reads sent %d requests", calls.Load())
	}
}

func TestGatewayGetStoreRejectsUnexpectedBodies(t *testing.T) {
	now := time.Now()
	session := testLiveSession(now, time.Hour)
	useLiveSession(t, &session)
	var body atomic.Value
	gatewayTestServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body.Load().(string))) })
	for _, reply := range []string{
		`{}`,
		`{"code":"0","message":"ok","data":{"id":1012,"name":"南山店","storeStatus":"OPEN"}}`,
		`{"id":3006,"name":"别的店","storeStatus":"OPEN"}`,
		`{"id":1012,"name":"南山店"}`,
		`{"id":"10x12","name":"南山店","storeStatus":"OPEN"}`,
		`{"id":1012,"name":"南山店","storeStatus":"OPEN","groupQueues":{"boothQueue":{"no":1}}}`,
		`[{"id":1012,"name":"南山店","storeStatus":"OPEN"}]`,
		`null`,
	} {
		body.Store(reply)
		_, err := NewQueueLiveClient().fetchStore(context.Background(), "1012")
		if !errors.Is(err, errLiveStoreFormat) {
			t.Errorf("%s: %v", reply, err)
		}
	}
	if _, err := NewQueueLiveClient().fetchStore(context.Background(), "abc"); err == nil {
		t.Fatal("accepted a non-numeric store id")
	}
}

func TestDecodeQueueLiveStoresHardening(t *testing.T) {
	for _, tc := range []struct {
		body string
		want []int
		ok   bool
	}{
		{`[]`, []int{}, true},
		{`{"data":[]}`, []int{}, true},
		{`[{"id":"1012","name":"a"},{"name":"no id"},"x",{"id":"bad"}]`, []int{1012}, true},
		{`{"stores":[{"id":3006}],"meta":{"page":1}}`, []int{3006}, true},
		{`{}`, nil, false},
		{`{"code":"E012","message":"unauthorized"}`, nil, false},
		{`{"data":null}`, nil, false},
		{`{"tags":["a","b"]}`, nil, false},
		{`[{"name":"no id"}]`, nil, false},
		{`"stores"`, nil, false},
		{``, nil, false},
	} {
		stores, err := decodeQueueLiveStores([]byte(tc.body))
		if (err == nil) != tc.ok {
			t.Errorf("%s: err=%v", tc.body, err)
			continue
		}
		if err != nil {
			if !errors.Is(err, errLiveStoreListFormat) {
				t.Errorf("%s: error not user readable: %v", tc.body, err)
			}
			continue
		}
		got := []int{}
		for _, store := range stores {
			got = append(got, store.ID)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: ids %v, want %v", tc.body, got, tc.want)
		}
	}
}

func TestGatewayListStoresAndAreas(t *testing.T) {
	now := time.Now()
	session := testLiveSession(now, time.Hour)
	useLiveSession(t, &session)
	var (
		mu    sync.Mutex
		paths []string
	)
	gatewayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
		mu.Unlock()
		w.Write([]byte(`[
			{"id":1012,"name":"南山店","nameKana":"深圳","area":"深圳南山区","wait":0,"storeStatus":"OPEN"},
			{"id":"3014","name":"中关村店","nameKana":"北京","area":"北京海淀区","wait":50,"storeStatus":"OPEN"},
			{"id":3015,"name":"海淀二店","nameKana":"北京","area":"北京海淀区","wait":5,"storeStatus":"OPEN"}
		]`))
	})
	client := NewQueueLiveClient()
	stores, err := client.ListStores(context.Background(), QueueLiveStoreQuery{WaitingOnly: true})
	if err != nil || len(stores) != 2 || stores[0].ID != 3014 || stores[0].Source != queueLiveSourceGateway {
		t.Fatalf("stores = %+v %v", stores, err)
	}
	areas, err := client.ListAreas(context.Background())
	if err != nil || !reflect.DeepEqual(areas, []string{"北京海淀区", "深圳南山区"}) {
		t.Fatalf("areas = %v %v", areas, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(paths) != 1 || paths[0] != "/gateway/wechat/api/2.0/stores?latitude=1&longitude=1&numresults=10000" {
		t.Fatalf("requests = %v", paths)
	}
}

func TestGatewayRejectionStopsRequestsUntilANewConnection(t *testing.T) {
	now := time.Now()
	first := testLiveSession(now.Add(-time.Minute), time.Hour)
	second := testLiveSession(now, time.Hour)
	var current atomic.Pointer[LiveSession]
	current.Store(&first)
	useLiveSession(t, nil)
	queueLiveSessionProvider = func() (*LiveSession, error) { copy := *current.Load(); return &copy, nil }
	var calls atomic.Int32
	gatewayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") == first.Authorization {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"code":"E401"}`))
			return
		}
		w.Write([]byte(`{"id":1012,"name":"南山店","storeStatus":"OPEN"}`))
	})
	_, err := NewQueueLiveClient().GetStore(context.Background(), "1012")
	if liveErrorCode(err) != liveErrAuthRequired {
		t.Fatalf("401 code = %q (%v)", liveErrorCode(err), err)
	}
	for i := 0; i < 3; i++ {
		if _, err := NewQueueLiveClient().GetStore(context.Background(), "1012"); !errors.Is(err, ErrLiveAuthRequired) {
			t.Fatalf("rejected connection still used: %v", err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("rejected connection sent %d requests", calls.Load())
	}
	current.Store(&second)
	if _, err := NewQueueLiveClient().GetStore(context.Background(), "1012"); err != nil || calls.Load() != 2 {
		t.Fatalf("new connection not used: %v calls=%d", err, calls.Load())
	}
}

func TestGatewayClientDoesNotFollowRedirects(t *testing.T) {
	now := time.Now()
	session := testLiveSession(now, time.Hour)
	useLiveSession(t, &session)
	var leaked atomic.Int32
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked.Add(1) }))
	defer elsewhere.Close()
	gatewayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/steal", http.StatusFound)
	})
	if _, err := NewQueueLiveClient().fetchStore(context.Background(), "1012"); err == nil {
		t.Fatal("redirect accepted as data")
	}
	if leaked.Load() != 0 {
		t.Fatal("connection headers followed a redirect to another host")
	}
}
