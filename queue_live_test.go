package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDecodeQueueLiveStores(t *testing.T) {
	arrayBody := []byte(`[{"id":1012,"name":"南山天利名城店","nameKana":"深圳","wait":3}]`)
	stores, err := decodeQueueLiveStores(arrayBody)
	if err != nil {
		t.Fatalf("decode array: %v", err)
	}
	if len(stores) != 1 || stores[0].ID != 1012 || stores[0].Wait != 3 {
		t.Fatalf("stores = %#v", stores)
	}

	wrappedBody := []byte(`{"data":[{"id":3014,"name":"中关村大融城店","nameKana":"北京","wait":12}],"meta":{"ignored":true}}`)
	stores, err = decodeQueueLiveStores(wrappedBody)
	if err != nil {
		t.Fatalf("decode wrapped: %v", err)
	}
	if len(stores) != 1 || stores[0].ID != 3014 || stores[0].Wait != 12 {
		t.Fatalf("stores = %#v", stores)
	}
}

func TestQueueLiveClientListStoresHeadersAndFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stores" {
			t.Fatalf("path = %s, want /stores", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		if got := r.Header.Get("Referer"); got != "test-referer" {
			t.Fatalf("referer = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"id":1012,"name":"南山天利名城店","nameKana":"深圳","area":"深圳南山区","wait":0,"storeStatus":"OPEN"},
			{"id":3014,"name":"中关村大融城店","nameKana":"北京","area":"北京海淀区","wait":50,"storeStatus":"OPEN"},
			{"id":3037,"name":"济南万象城店","nameKana":"济南","area":"济南历下区","wait":20,"storeStatus":"CLOSED"}
		]`))
	}))
	defer server.Close()

	client := &QueueLiveClient{
		baseURL:    server.URL,
		token:      "test-token",
		referer:    "test-referer",
		userAgent:  "test-agent",
		httpClient: server.Client(),
	}
	stores, err := client.ListStores(context.Background(), QueueLiveStoreQuery{OpenOnly: true, WaitingOnly: true, Limit: 1})
	if err != nil {
		t.Fatalf("ListStores: %v", err)
	}
	if len(stores) != 1 || stores[0].ID != 3014 {
		t.Fatalf("stores = %#v", stores)
	}
}

func TestQueueObservationFromLiveStore(t *testing.T) {
	observation := queueObservationFromLiveStore(QueueLiveStore{
		ID:              1012,
		Name:            "南山天利名城店",
		Wait:            0,
		WaitTimeCap:     180,
		StoreStatus:     "OPEN",
		NetTicketStatus: "ONLINE",
	}, mustParseLocalTime(t, "2026-05-26T18:30:00+08:00"))

	if observation.StoreID != "1012" {
		t.Fatalf("observation = %#v", observation)
	}
	if !observation.OnlineOpen {
		t.Fatalf("observation = %#v", observation)
	}
}

func TestQueueLiveGroupQueuesCurrentCalledNo(t *testing.T) {
	g := QueueLiveGroupQueues{
		ReservationQueue: []string{"7261"}, // 独立号段，不参与堂食叫号
		CounterQueue:     []string{},
		BoothQueue:       []string{"819", "824", "826"},
		MixedQueue:       []string{"819", "824", "826"},
	}
	if got := g.CurrentCalledNo(); got != 826 {
		t.Fatalf("CurrentCalledNo() = %d, want 826", got)
	}
	if got := (QueueLiveGroupQueues{}).CurrentCalledNo(); got != 0 {
		t.Fatalf("empty CurrentCalledNo() = %d, want 0", got)
	}
}

func TestQueueObservationCapturesCalledNo(t *testing.T) {
	store := QueueLiveStore{
		ID:               3006,
		Name:             "太阳宫凯德店",
		Wait:             345,
		GroupQueuesCount: 54,
		WaitTimeCap:      180,
		StoreStatus:      "OPEN",
		NetTicketStatus:  "ONLINE",
		GroupQueues:      QueueLiveGroupQueues{BoothQueue: []string{"819", "826"}, MixedQueue: []string{"819", "826"}},
	}
	o := queueObservationFromLiveStore(store, mustParseLocalTime(t, "2026-06-01T16:30:00+08:00"))
	if o.DisplayCalledNo != 826 {
		t.Fatalf("DisplayCalledNo = %d, want 826", o.DisplayCalledNo)
	}
	if o.GroupQueuesCount != 54 {
		t.Fatalf("GroupQueuesCount = %d, want 54", o.GroupQueuesCount)
	}
	if o.WaitMinutes != 345 {
		t.Fatalf("WaitMinutes = %d, want 345 (wait)", o.WaitMinutes)
	}
}

func mustParseLocalTime(t *testing.T, raw string) time.Time {
	t.Helper()
	at, ok := parseRFC3339Local(raw)
	if !ok {
		t.Fatalf("parse %s", raw)
	}
	return at
}
