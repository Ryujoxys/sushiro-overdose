package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	if err := AtomicWriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestJSONLReadCacheInvalidatesOnAppend(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	writeLines(t, queueObservationPath(),
		`{"store_id":"3006","collected_at":"2026-06-09T12:00:00+08:00","wait_minutes":30}`)
	first := loadQueueObservations()
	if len(first) != 1 || first[0].WaitMinutes != 30 {
		t.Fatalf("first load = %+v", first)
	}
	// 追加一条（mtime/size 变化）后必须重读，不能吐旧缓存。
	writeLines(t, queueObservationPath(),
		`{"store_id":"3006","collected_at":"2026-06-09T12:05:00+08:00","wait_minutes":35}`,
		`{"store_id":"3006","collected_at":"2026-06-09T12:10:00+08:00","wait_minutes":40}`)
	second := loadQueueObservations()
	if len(second) != 2 {
		t.Fatalf("after append, load = %d rows, want 2", len(second))
	}
}

func TestJSONLReadCacheHitsUnchangedFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	writeLines(t, queueObservationPath(),
		`{"store_id":"3006","collected_at":"2026-06-09T12:00:00+08:00","wait_minutes":30}`)
	a := loadQueueObservations()
	b := loadQueueObservations()
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("loads = %d/%d rows, want 1/1", len(a), len(b))
	}
	// 命中缓存时返回同一份切片（零拷贝）；调用方约定只读。
	if &a[0] != &b[0] {
		t.Fatal("unchanged file should return the cached slice, not a re-parse")
	}
}

func TestJSONLReadCacheMissingFileReturnsNil(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if got := loadQueueObservations(); got != nil {
		t.Fatalf("missing file should load nil, got %d rows", len(got))
	}
}

func TestQueueSnapshotsRetainAllRecordsBeyondFormerLimit(t *testing.T) {
	const formerLimit, formerTrimInterval = 100000, 200
	start := time.Now().Add(-time.Duration(formerLimit+formerTrimInterval+60) * time.Second).Truncate(time.Second)
	for _, kind := range []string{"baseline", "observation"} {
		t.Run(kind, func(t *testing.T) {
			t.Setenv("SUSHIRO_DATA_HOME", t.TempDir())
			path, storeID := queueBaselineRecordsPath(), "3006"
			if kind == "observation" {
				path, storeID = queueObservationPath(), `"3006"`
			}
			var source strings.Builder
			for i := 0; i < formerLimit; i++ {
				fmt.Fprintf(&source, "{\"store_id\":%s,\"collected_at\":%q,\"wait_minutes\":30,\"store_status\":\"OPEN\"}\n", storeID, start.Add(time.Duration(i)*time.Second).Format(time.RFC3339))
			}
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := AtomicWriteFile(path, []byte(source.String()), 0600); err != nil {
				t.Fatal(err)
			}
			// Cross the former trim cadence as well as the former row limit.
			for i := 0; i < formerTrimInterval; i++ {
				at := start.Add(time.Duration(formerLimit+i) * time.Second).Format(time.RFC3339)
				var err error
				if kind == "baseline" {
					err = appendQueueBaselineRecords([]QueueBaselineRecord{{StoreID: 3006, CollectedAt: at, WaitMinutes: 30, StoreStatus: "OPEN"}})
				} else {
					err = appendQueueObservation(QueueObservation{StoreID: "3006", CollectedAt: at, WaitMinutes: 30, StoreStatus: "OPEN"})
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			want := formerLimit + formerTrimInterval
			if count := bytes.Count(raw, []byte{'\n'}); count != want || !bytes.HasPrefix(raw, []byte(source.String())) {
				t.Fatalf("old snapshots were changed or lost: count=%d, want=%d", count, want)
			}
			if kind == "baseline" {
				rows, err := readLocalRecords(localRecordsQuery{}, time.Now())
				if err != nil || len(rows) != want || rows[0].CollectedAt != start.Format(time.RFC3339) {
					t.Fatalf("all-history query: count=%d, err=%v", len(rows), err)
				}
				w := httptest.NewRecorder()
				handleLocalRecordsExport(w, httptest.NewRequest("GET", "/api/records/export?days=all", nil))
				if w.Code != http.StatusOK || bytes.Count(w.Body.Bytes(), []byte{'\n'}) != want {
					t.Fatalf("all-history backup: status=%d, count=%d", w.Code, bytes.Count(w.Body.Bytes(), []byte{'\n'}))
				}
			} else if rows := loadQueueObservations(); len(rows) != want || rows[0].CollectedAt != start.Format(time.RFC3339) {
				t.Fatalf("observation query lost old snapshots: count=%d", len(rows))
			}
		})
	}
}
