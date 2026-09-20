package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

func TestQueueLiveStoreOnlineOpen(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{"ONLINE", true},
		{"online", true},
		{"ON", true},
		{"OFFLINE_CLOSED", false},
		{"OFFLINE", false},
		{"CLOSED", false},
		{"", false},
		{"  ONLINE  ", true},
	}
	for _, c := range cases {
		if got := queueLiveStoreOnlineOpen(QueueLiveStore{NetTicketStatus: c.status}); got != c.want {
			t.Errorf("queueLiveStoreOnlineOpen(%q) = %v, want %v", c.status, got, c.want)
		}
	}
}

func TestNormalizeQueueBaselineConfig(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, queueBaselineDefaultMinutes},
		{-5, queueBaselineDefaultMinutes},
		{3, 3},
		{5000, 1440},
	}
	for _, c := range cases {
		got := NormalizeQueueBaselineConfig(QueueBaselineConfig{IntervalMinutes: c.in})
		if got.IntervalMinutes != c.want {
			t.Errorf("NormalizeQueueBaselineConfig(%d) = %d, want %d", c.in, got.IntervalMinutes, c.want)
		}
		if got.UsePreferenceStores || got.Enabled {
			t.Errorf("NormalizeQueueBaselineConfig(%d) must preserve an explicit empty selection", c.in)
		}
	}
}

func TestQueueBaselineExplicitEmptySelectionStaysPaused(t *testing.T) {
	t.Setenv("SUSHIRO_DATA_HOME", t.TempDir())
	if err := SavePreferences(UserPreferences{SelectedStores: []string{"3006"}}); err != nil {
		t.Fatal(err)
	}
	if err := SaveQueueAlertConfig(QueueAlertConfig{Rules: []QueueAlertRule{{Enabled: true, StoreID: "3050", Type: queueAlertCalledReach, TargetNo: 100, NotifyAtNo: 90}}}); err != nil {
		t.Fatal(err)
	}
	record := QueueBaselineRecord{StoreID: 1012, CollectedAt: time.Now().Format(time.RFC3339), WaitMinutes: 20}
	if err := appendQueueBaselineRecords([]QueueBaselineRecord{record}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(queueBaselineRecordsPath())
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handleQueueBaseline(rr, httptest.NewRequest(http.MethodPost, "/api/queue/baseline", strings.NewReader(`{"enabled":true,"store_ids":[],"use_preference_stores":false,"interval_minutes":5}`)))
	if rr.Code != http.StatusOK {
		t.Fatalf("clear selection: %d %s", rr.Code, rr.Body.String())
	}
	for i := 0; i < 2; i++ {
		cfg := LoadQueueBaselineConfig()
		if cfg.Enabled || cfg.UsePreferenceStores || len(queueBaselineStoreIDs(cfg)) != 0 {
			t.Fatalf("empty selection revived old stores: %+v, effective %v", cfg, queueBaselineStoreIDs(cfg))
		}
		if err := SaveQueueBaselineConfig(cfg); err != nil {
			t.Fatal(err)
		}
		collector := &QueueBaselineCollector{collect: func(context.Context, QueueBaselineConfig) (int, error) {
			t.Error("empty selection collected old preference or alert stores")
			return 0, nil
		}}
		collector.tick(context.Background())
		if status := collector.status(); status.Enabled || status.Running || len(status.StoreIDs) != 0 {
			t.Fatalf("empty selection status: %+v", status)
		}
	}
	after, err := os.ReadFile(queueBaselineRecordsPath())
	if err != nil || string(before) != string(after) {
		t.Fatalf("clearing stores changed history: %v", err)
	}
	if got := LoadPreferences().SelectedStores; len(got) != 1 || got[0] != "3006" {
		t.Fatalf("clearing records changed ticket preferences: %v", got)
	}
	if got := queueAlertStoreIDs(); len(got) != 1 || got[0] != "3050" {
		t.Fatalf("clearing records changed reminder settings: %v", got)
	}
	var actions []string
	injected := publicQueueServiceActions{
		install: func() error { actions = append(actions, "install"); return nil },
		start:   func() error { actions = append(actions, "start"); return nil },
		remove:  func() error { actions = append(actions, "remove"); return nil },
	}
	if err := configurePublicQueueAutoStart(true, injected); err == nil || len(actions) != 0 {
		t.Fatalf("empty selection enabled autostart: %v %v", actions, err)
	}
	if err := configurePublicQueueAutoStart(false, injected); err != nil || strings.Join(actions, ",") != "remove" {
		t.Fatalf("empty selection prevented disabling existing autostart: %v %v", actions, err)
	}
	if LoadQueueBaselineConfig().Enabled {
		t.Fatal("disabling autostart resumed empty collection")
	}
}

func TestQueueBaselineLegacyPreferenceSelectionRemainsReadable(t *testing.T) {
	t.Setenv("SUSHIRO_DATA_HOME", t.TempDir())
	if err := SavePreferences(UserPreferences{StorePriority: []string{"3006"}}); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		`{"enabled":true,"interval_minutes":5}`,
		`{"enabled":true,"store_ids":[],"use_preference_stores":true}`,
	} {
		if err := os.WriteFile(queueBaselinePath(), []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg := LoadQueueBaselineConfig()
		if !cfg.Enabled || !cfg.UsePreferenceStores || strings.Join(queueBaselineStoreIDs(cfg), ",") != "3006" {
			t.Fatalf("legacy preference selection changed: %+v", cfg)
		}
	}
}

func TestQueueBaselineExplicitSelectionExcludesOldReminderStores(t *testing.T) {
	t.Setenv("SUSHIRO_DATA_HOME", t.TempDir())
	if err := SavePreferences(UserPreferences{SelectedStores: []string{"3006"}}); err != nil {
		t.Fatal(err)
	}
	if err := SaveQueueAlertConfig(QueueAlertConfig{Rules: []QueueAlertRule{{Enabled: true, StoreID: "3050", Type: queueAlertCalledReach, TargetNo: 100, NotifyAtNo: 90}}}); err != nil {
		t.Fatal(err)
	}
	cfg := QueueBaselineConfig{Enabled: true, IntervalMinutes: 5, StoreIDs: []string{"1012"}, UsePreferenceStores: false}
	if err := SaveQueueBaselineConfig(cfg); err != nil {
		t.Fatal(err)
	}
	cfg = LoadQueueBaselineConfig()
	if got := strings.Join(queueBaselineStoreIDs(cfg), ","); got != "1012" {
		t.Fatalf("explicit selection added old stores: %s", got)
	}
	if status := sharedQueueCollectionStatus(time.Now()); strings.Join(status.StoreIDs, ",") != "1012" || status.IntervalSeconds != 300 {
		t.Fatalf("old reminder affected selected stores or cadence: %+v", status)
	}
	cfg.UsePreferenceStores = true
	if got := strings.Join(queueBaselineStoreIDs(cfg), ","); got != "1012,3050" {
		t.Fatalf("legacy reminder selection changed: %s", got)
	}
	if seconds := queueBaselineIntervalSeconds(cfg, queueBaselineStoreIDs(cfg)); seconds != 60 {
		t.Fatalf("legacy reminder cadence changed: %d", seconds)
	}
	if got := queueAlertStoreIDs(); len(got) != 1 || got[0] != "3050" {
		t.Fatalf("editing records deleted reminder settings: %v", got)
	}
}

func TestQueueBaselineCollectionRequiresExplicitOptIn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	if cfg := LoadQueueBaselineConfig(); cfg.Enabled {
		t.Fatal("missing collection config must not opt in")
	}
	if err := SaveQueueBaselineConfig(QueueBaselineConfig{Enabled: true, StoreIDs: []string{"1012"}}); err != nil {
		t.Fatal(err)
	}
	if cfg := LoadQueueBaselineConfig(); !cfg.Enabled {
		t.Fatal("explicit saved opt-in must be retained")
	}
	if err := os.WriteFile(queueBaselinePath(), []byte(`{"enabled":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if cfg := LoadQueueBaselineConfig(); cfg.Enabled {
		t.Fatal("damaged collection config must not silently opt in")
	}
}

func TestQueueBaselineStoreIDsUseExplicitOrPreferenceStores(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.MkdirAll(AppDirPath(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SavePreferences(UserPreferences{
		SelectedStores: []string{"3006", "3006", "3050"},
		StorePriority:  []string{"3050", "3006"},
	}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(queueBaselineStoreIDs(QueueBaselineConfig{StoreIDs: []string{" 3006 ", "3006"}}), ","); got != "3006" {
		t.Fatalf("explicit store ids = %q, want 3006", got)
	}
	if got := strings.Join(queueBaselineStoreIDs(QueueBaselineConfig{UsePreferenceStores: true}), ","); got != "3006,3050" {
		t.Fatalf("preference store ids = %q, want 3006,3050", got)
	}
}

func TestQueueBaselineExportJSONShape(t *testing.T) {
	data := []byte(`{
		"version": 1,
		"generated_at": "2026-06-03T22:30:00+08:00",
		"source": "sushiro-public-collector",
		"bucket_minutes": 30,
		"date_types": ["weekday", "workday", "weekend", "holiday"],
		"stores": [{"store_id": 3015, "name": "深圳店", "city": "深圳", "area": "南山区"}],
		"latest": [{
			"store_id": 3015,
			"collected_at": "2026-06-03T22:20:00+08:00",
			"name": "深圳店",
			"city": "深圳",
			"area": "南山区",
			"wait_minutes": 28,
			"group_queues_count": 24,
			"store_status": "OPEN",
			"net_ticket_status": "ONLINE",
			"reservation_status": "ON",
			"online_open": true,
			"wait_time_cap": 180
		}],
		"rollups": [{
			"store_id": 3015,
			"date_type": "workday",
			"weekday": 6,
			"time_bucket": "18:30",
			"sample_count": 12,
			"open_rate": 1,
			"online_open_rate": 0.8,
			"busy_rate": 0.7,
			"wait_typical_minutes": 35,
			"wait_safe_minutes": 50,
			"wait_max_minutes": 80,
			"queue_groups_typical": 22,
			"queue_groups_safe": 38,
			"confidence": "high",
			"updated_at": "2026-06-03T22:30:00+08:00"
		}],
		"stats": {"store_count": 1, "rollup_count": 1, "source_updated_at": "2026-06-03T22:30:00+08:00"}
	}`)
	var got QueueBaselineExport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Version != 1 || got.BucketMinutes != 30 || len(got.Stores) != 1 || len(got.Latest) != 1 || len(got.Rollups) != 1 {
		t.Fatalf("unexpected baseline export: %+v", got)
	}
	if got.Latest[0].WaitMinutes != 28 || !got.Latest[0].OnlineOpen {
		t.Fatalf("unexpected baseline latest: %+v", got.Latest[0])
	}
	rollup := got.Rollups[0]
	if rollup.DateType != "workday" || rollup.WaitTypicalMinutes == nil || *rollup.WaitTypicalMinutes != 35 {
		t.Fatalf("unexpected baseline rollup: %+v", rollup)
	}
}

func TestQueueBaselineRecordWritesDatabaseShape(t *testing.T) {
	record := QueueBaselineRecord{
		Timestamp:        "2026-06-03T22:20:00+08:00",
		StoreID:          3015,
		Name:             "深圳店",
		City:             "深圳",
		Area:             "南山区",
		Wait:             28,
		GroupQueuesCount: 24,
		StoreStatus:      "OPEN",
		NetTicketStatus:  "ONLINE",
		OnlineOpen:       true,
	}
	normalizeQueueBaselineRecordForWrite(&record)
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(data)
	for _, want := range []string{
		`"collected_at":"2026-06-03T22:20:00+08:00"`,
		`"wait_minutes":28`,
		`"source_endpoint":"stores"`,
		`"api_profile_version":"public-profile-v1"`,
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("missing %s in %s", want, raw)
		}
	}
	if strings.Contains(raw, `"ts"`) || strings.Contains(raw, `"wait":`) {
		t.Fatalf("legacy fields should not be written: %s", raw)
	}
}

func TestQueueBaselineRecordFromStoreWritesDetailShape(t *testing.T) {
	record := queueBaselineRecordFromStore(QueueLiveStore{
		ID:               3006,
		Name:             "太阳宫凯德店",
		NameKana:         "北京",
		Area:             "朝阳区",
		Wait:             35,
		GroupQueuesCount: 12,
		NetTicketStatus:  "ONLINE",
		GroupQueues: QueueLiveGroupQueues{
			BoothQueue:   []string{"540"},
			CounterQueue: []string{"535"},
		},
	}, "2026-06-04T14:30:00+08:00")
	if record.StoreID != 3006 || record.DisplayCalledNo != 540 {
		t.Fatalf("record called no = %+v", record)
	}
	if record.SourceEndpoint != queueSourceEndpointStoreByID || record.APIProfileVersion != queueAPIProfileStoreDetailV1 {
		t.Fatalf("record source = %+v", record)
	}
	if !strings.Contains(record.GroupQueuesJSON, "540") {
		t.Fatalf("group queues json = %q", record.GroupQueuesJSON)
	}
}
