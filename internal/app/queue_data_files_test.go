package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
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

func TestTrimJSONLFileKeepsLatestLines(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	path := filepath.Join(home, "trim.jsonl")
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, `{"n":`+string(rune('0'+i%10))+`}`)
	}
	writeLines(t, path, lines...)
	trimJSONLFileLocked(path, 10, time.Now())
	kept, err := readAllLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 10 {
		t.Fatalf("kept %d lines, want 10", len(kept))
	}
	if kept[0] != lines[20] || kept[9] != lines[29] {
		t.Fatalf("trim kept wrong slice: first=%s want=%s", kept[0], lines[20])
	}
}

func TestTrimJSONLFileNoopUnderLimit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	path := filepath.Join(home, "noop.jsonl")
	writeLines(t, path, `{"a":1}`, `{"a":2}`)
	trimJSONLFileLocked(path, 10, time.Now())
	kept, err := readAllLines(path)
	if err != nil || len(kept) != 2 {
		t.Fatalf("under-limit trim should be noop: %d lines, err=%v", len(kept), err)
	}
}
