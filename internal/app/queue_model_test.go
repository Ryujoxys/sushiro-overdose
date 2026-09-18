package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

func TestLocalQueueModelCountsUniqueSamplesAndLocalDays(t *testing.T) {
	reliabilityHome(t)
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, SushiroTimezone)
	first := QueueBaselineRecord{StoreID: 1012, CollectedAt: "2026-09-13T18:05:00+08:00", WaitMinutes: 20, StoreStatus: "OPEN"}
	second := first
	second.CollectedAt, second.WaitMinutes = "2026-09-13T18:10:00+08:00", 40
	third := first
	third.CollectedAt = "2026-09-14T17:00:00Z" // September 15 in the restaurant timezone.
	future := first
	future.CollectedAt = now.Add(time.Hour).Format(time.RFC3339)
	model := buildLocalQueueModel([]QueueBaselineRecord{first, first, second, third, future, {StoreID: 1012, CollectedAt: "invalid"}}, now)
	if model.SampleCount != 3 || model.DayCount != 2 || model.Baseline.Source != "local" || model.Version != 1 {
		t.Fatalf("model counts/source: %+v", model)
	}
	foundSunday := false
	for _, rollup := range model.Baseline.Rollups {
		if rollup.Weekday == 7 {
			foundSunday = true
			if rollup.TimeBucket != "18:00" || rollup.SampleCount != 2 || rollup.WaitTypicalMinutes == nil || *rollup.WaitTypicalMinutes < 20 || *rollup.WaitSafeMinutes < *rollup.WaitTypicalMinutes {
				t.Fatalf("bad Sunday quantiles: %+v", rollup)
			}
		}
	}
	if !foundSunday {
		t.Fatal("model did not use ISO weekdays")
	}
	data, _ := json.Marshal(model)
	for _, private := range []string{"phone_number", "wechat_id", "authorization", "ticket_id", "session_token"} {
		if strings.Contains(string(data), private) {
			t.Fatalf("private field in model: %s", private)
		}
	}
}

func TestLocalQueueModelPersistsRefreshesAndInvalidates(t *testing.T) {
	reliabilityHome(t)
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, SushiroTimezone)
	if status := localQueueModelStatus(now); status.Status != "empty" || status.SampleCount != 0 {
		t.Fatal(status)
	}
	record := QueueBaselineRecord{StoreID: 1012, CollectedAt: now.Add(-time.Minute).Format(time.RFC3339), WaitMinutes: 30}
	if err := appendQueueBaselineRecords([]QueueBaselineRecord{record}); err != nil {
		t.Fatal(err)
	}
	if err := rebuildLocalQueueModel(now); err != nil {
		t.Fatal(err)
	}
	model, err := loadLocalQueueModel()
	if err != nil || !localQueueModelCurrent(model, now) {
		t.Fatalf("model not current: %v", err)
	}
	if status := localQueueModelStatus(now); status.Stale || status.SampleCount != 1 || status.DayCount != 1 || status.Status != "collecting" {
		t.Fatal(status)
	}
	stamp := queueFileStamp(queueModelPath())
	if err := rebuildLocalQueueModel(now.Add(time.Minute)); err != nil || queueFileStamp(queueModelPath()) != stamp {
		t.Fatal("unchanged history rewrote the model", err)
	}
	if got := localQueueBaselineForRecords(nil, now); !reflect.DeepEqual(got, model.Baseline) {
		t.Fatal("dashboard did not use the current local model")
	}
	if localQueueModelCurrent(model, now.Add(25*time.Hour)) || localQueueModelCurrent(model, now.Add(-time.Hour)) {
		t.Fatal("expired/future model accepted")
	}
	if err := os.WriteFile(filepath.Join(AppDirPath(), "holidays.json"), []byte(`{"holidays":["2026-09-16"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if localQueueModelCurrent(model, now) {
		t.Fatal("holiday changes must invalidate the buckets")
	}
	record.CollectedAt = now.Format(time.RFC3339)
	if err := appendQueueBaselineRecords([]QueueBaselineRecord{record}); err != nil {
		t.Fatal(err)
	}
	if !localQueueModelStatus(now).Stale {
		t.Fatal("new history did not invalidate model")
	}
	if err := rebuildLocalQueueModel(now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if status := localQueueModelStatus(now.Add(time.Minute)); status.Stale || status.SampleCount != 2 {
		t.Fatal(status)
	}
	if err := os.WriteFile(queueModelPath(), []byte(`{"version":99}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if localQueueModelStatus(now).Status != "error" {
		t.Fatal("corrupt model not exposed")
	}
	if got := localQueueBaselineForRecords([]QueueBaselineRecord{record}, now); len(got.Rollups) != 1 {
		t.Fatal("invalid model did not fall back to raw local history")
	}
}

func TestJSONLCacheSeparatesFilesWithIdenticalFingerprint(t *testing.T) {
	dir := t.TempDir()
	paths := []string{filepath.Join(dir, "a.jsonl"), filepath.Join(dir, "b.jsonl")}
	at := time.Now().Add(-time.Hour)
	for i, path := range paths {
		data, _ := json.Marshal(map[string]int{"value": i + 1})
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}
	cache := &jsonlReadCache[map[string]int]{}
	if cache.load(paths[0], nil)[0]["value"] != 1 || cache.load(paths[1], nil)[0]["value"] != 2 {
		t.Fatal("cache reused a different file with the same size and timestamp")
	}
}
