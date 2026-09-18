package app

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func fixtureHistoryArchive(t *testing.T, change func(map[string]any)) []byte {
	t.Helper()
	pack := map[string]any{
		"version": 1, "id": "test-history", "timezone": "Asia/Shanghai", "bucket_minutes": 30, "source": "maintainer_snapshot",
		"exported_at": "2026-09-18T12:00:00+08:00", "first_at": "2026-09-01T12:00:00+08:00", "cutoff_at": "2026-09-02T12:30:00+08:00",
		"stores": []map[string]any{{"id": 1, "name": "测试店", "city": "", "area": ""}},
		"records": []map[string]any{
			{"store_id": 1, "collected_at": "2026-09-01T12:00:00+08:00", "date_type": "weekday", "store_status": "OPEN", "wait_minutes": 0},
			{"store_id": 1, "collected_at": "2026-09-01T12:30:00+08:00", "date_type": "weekday", "store_status": "OPEN", "wait_minutes": nil},
			{"store_id": 1, "collected_at": "2026-09-02T12:00:00+08:00", "date_type": "weekday", "store_status": "OPEN", "wait_minutes": 60},
			{"store_id": 1, "collected_at": "2026-09-02T12:30:00+08:00", "date_type": "weekday", "store_status": "CLOSED", "wait_minutes": 0},
		},
	}
	if change != nil {
		change(pack)
	}
	var buf bytes.Buffer
	z := gzip.NewWriter(&buf)
	if err := json.NewEncoder(z).Encode(pack); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func fixtureHistory(t *testing.T) *historyBundle {
	t.Helper()
	pack, err := decodeHistoryBundle(fixtureHistoryArchive(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	return pack
}

func TestHistoryBundleRejectsInvalidData(t *testing.T) {
	for name, change := range map[string]func(map[string]any){
		"version":         func(p map[string]any) { p["version"] = 9 },
		"timezone":        func(p map[string]any) { p["timezone"] = "UTC" },
		"private field":   func(p map[string]any) { p["auth_token"] = "not allowed" },
		"cutoff":          func(p map[string]any) { p["cutoff_at"] = "2026-09-01T12:00:00+08:00" },
		"orphan":          func(p map[string]any) { p["records"].([]map[string]any)[0]["store_id"] = 2 },
		"negative":        func(p map[string]any) { p["records"].([]map[string]any)[0]["wait_minutes"] = -1 },
		"called zero":     func(p map[string]any) { p["records"].([]map[string]any)[0]["display_called_no"] = 0 },
		"called negative": func(p map[string]any) { p["records"].([]map[string]any)[0]["display_called_no"] = -1 },
		"duplicate":       func(p map[string]any) { rows := p["records"].([]map[string]any); p["records"] = append(rows, rows[0]) },
		"date type":       func(p map[string]any) { p["records"].([]map[string]any)[0]["date_type"] = "oops" },
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeHistoryBundle(fixtureHistoryArchive(t, change)); err == nil {
				t.Fatal("invalid bundle accepted")
			}
		})
	}
	valid := fixtureHistoryArchive(t, nil)
	if _, err := decodeHistoryBundle(valid[:len(valid)-8]); err == nil {
		t.Fatal("truncated gzip accepted")
	}
}

func TestEmbeddedHistoryMatchesManifest(t *testing.T) {
	raw, err := os.ReadFile("data/queue-history-v1.manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		historyMetadata
		StoreCount  int    `json:"store_count"`
		RecordCount int    `json:"record_count"`
		DayCount    int    `json:"day_count"`
		Bytes       int    `json:"bytes"`
		SHA256      string `json:"sha256"`
		Quality     struct {
			Usable        int `json:"usable_waits"`
			Zeros         int `json:"valid_zero_waits"`
			Unknown       int `json:"unknown_waits"`
			Closed        int `json:"non_open"`
			Called        int `json:"usable_called_numbers"`
			CalledUnknown int `json:"unknown_called_numbers"`
		} `json:"quality"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	pack, err := loadBundledHistory()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(historyBundleGzip)
	if manifest.SHA256 != hex.EncodeToString(digest[:]) || manifest.Bytes != len(historyBundleGzip) || manifest.historyMetadata != pack.historyMetadata {
		t.Fatal("manifest does not describe shipped archive")
	}
	var total, usable, zeros, unknown, closed, called, calledUnknown int
	days := map[string]bool{}
	for _, rows := range pack.byStore {
		for _, row := range rows {
			total++
			days[row.at.Format("2006-01-02")] = true
			if row.wait == nil {
				unknown++
			}
			if !row.open {
				closed++
			}
			if row.called == nil {
				calledUnknown++
			} else if row.open {
				called++
			}
			if row.open && row.wait != nil {
				usable++
				if *row.wait == 0 {
					zeros++
				}
			}
		}
	}
	if total != manifest.RecordCount || len(pack.Stores) != manifest.StoreCount || len(days) != manifest.DayCount || usable != manifest.Quality.Usable || zeros != manifest.Quality.Zeros || unknown != manifest.Quality.Unknown || closed != manifest.Quality.Closed {
		t.Fatal("quality counts differ from source receipt")
	}
	if total < 1 || zeros < 1 || unknown < 1 {
		t.Fatal("source coverage lost")
	}
	if called == 0 || called != manifest.Quality.Called || calledUnknown != manifest.Quality.CalledUnknown {
		t.Fatal("called-number coverage differs from manifest")
	}
	for _, s := range pack.Stores {
		for _, field := range []string{s.Name, s.City, s.Area} {
			if strings.Contains(field, "://") || strings.Contains(field, "eyJ") {
				t.Fatal("unexpected private metadata")
			}
		}
	}
}
