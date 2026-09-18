package app

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

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
		if !got.UsePreferenceStores {
			t.Errorf("NormalizeQueueBaselineConfig(%d) should default to preference stores", c.in)
		}
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
