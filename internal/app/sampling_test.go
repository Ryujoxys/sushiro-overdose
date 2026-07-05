package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	samplingHTTPTransportMu sync.Mutex
	samplingStdoutMu        sync.Mutex
)

func TestNormalizeSamplingConfigBoundsIntervalAndTimes(t *testing.T) {
	cfg := NormalizeSamplingConfig(SamplingConfig{
		IntervalSeconds: 10,
		ActiveStart:     "bad",
		ActiveEnd:       "22:30",
		StoreIDs:        []string{"001", "001", " 002 "},
	})
	if cfg.IntervalSeconds != 60 {
		t.Fatalf("IntervalSeconds = %d, want 60", cfg.IntervalSeconds)
	}
	if cfg.ActiveStart != "100000" {
		t.Fatalf("ActiveStart = %q, want 100000", cfg.ActiveStart)
	}
	if cfg.ActiveEnd != "223000" {
		t.Fatalf("ActiveEnd = %q, want 223000", cfg.ActiveEnd)
	}
	if got := len(cfg.StoreIDs); got != 2 {
		t.Fatalf("StoreIDs len = %d, want 2", got)
	}
}

func TestSamplingActiveWindowSupportsOvernight(t *testing.T) {
	cfg := NormalizeSamplingConfig(SamplingConfig{IntervalSeconds: 300, ActiveStart: "2200", ActiveEnd: "0200"})
	loc := testLocation(t)
	if !samplingInActiveWindow(cfg, time.Date(2026, 5, 15, 23, 0, 0, 0, loc)) {
		t.Fatal("23:00 should be inside overnight window")
	}
	if !samplingInActiveWindow(cfg, time.Date(2026, 5, 16, 1, 0, 0, 0, loc)) {
		t.Fatal("01:00 should be inside overnight window")
	}
	if samplingInActiveWindow(cfg, time.Date(2026, 5, 16, 12, 0, 0, 0, loc)) {
		t.Fatal("12:00 should be outside overnight window")
	}
}

func TestSamplingRunLogsStoreInfoFailureWithoutStoreError(t *testing.T) {
	storeID := "9301"
	cfg := setupSamplingRunTest(t, storeID)
	restoreHTTP := stubSamplingHTTP(t, func(r *http.Request) (int, string) {
		switch r.URL.Path {
		case "/wechat/api/2.0/store/timeslots":
			return http.StatusOK, `[{"date":"20260515","start":"193000","end":"200000","availability":"AVAILABLE"}]`
		case "/wechat/api/2.0/getStoreById":
			return http.StatusInternalServerError, "store info unavailable"
		default:
			return http.StatusNotFound, "unexpected path: " + r.URL.Path
		}
	})

	var result SamplingRunResult
	output := captureStdout(t, func() {
		result = (&SlotSampler{}).runOnce(context.Background(), cfg, SamplingRunOptions{IgnoreActiveWindow: true})
	})
	restoreHTTP()

	if result.Skipped {
		t.Fatalf("run skipped: %s", result.SkipReason)
	}
	if result.StoreErrors != 0 {
		t.Fatalf("StoreErrors = %d, want 0", result.StoreErrors)
	}
	if result.Snapshots != 1 {
		t.Fatalf("Snapshots = %d, want 1", result.Snapshots)
	}
	if !strings.Contains(output, storeID) || !strings.Contains(output, "store info unavailable") {
		t.Fatalf("log output = %q, want store ID and store info error", output)
	}
	if strings.Contains(output, "query-auth") || strings.Contains(output, "reservation-auth") {
		t.Fatalf("log output leaks auth token: %q", output)
	}
}

func TestSamplingRunLogsQueueObservationAppendFailureWithoutStoreError(t *testing.T) {
	storeID := "9302"
	cfg := setupSamplingRunTest(t, storeID)
	if err := os.MkdirAll(queueObservationPath(), 0o755); err != nil {
		t.Fatalf("create queue observation path as directory: %v", err)
	}
	restoreHTTP := stubSamplingHTTP(t, func(r *http.Request) (int, string) {
		switch r.URL.Path {
		case "/wechat/api/2.0/store/timeslots":
			return http.StatusOK, `[{"date":"20260515","start":"193000","end":"200000","availability":"AVAILABLE"}]`
		case "/wechat/api/2.0/getStoreById":
			return http.StatusOK, `{"wait":20,"storeStatus":"OPEN","netTicketStatus":"ONLINE_OPEN"}`
		default:
			return http.StatusNotFound, "unexpected path: " + r.URL.Path
		}
	})

	var result SamplingRunResult
	output := captureStdout(t, func() {
		result = (&SlotSampler{}).runOnce(context.Background(), cfg, SamplingRunOptions{IgnoreActiveWindow: true})
	})
	restoreHTTP()

	if result.Skipped {
		t.Fatalf("run skipped: %s", result.SkipReason)
	}
	if result.StoreErrors != 0 {
		t.Fatalf("StoreErrors = %d, want 0", result.StoreErrors)
	}
	if result.Snapshots != 1 {
		t.Fatalf("Snapshots = %d, want 1", result.Snapshots)
	}
	if !strings.Contains(output, storeID) || !strings.Contains(output, queueObservationFile) {
		t.Fatalf("log output = %q, want store ID and append error", output)
	}
	if strings.Contains(output, "query-auth") || strings.Contains(output, "reservation-auth") {
		t.Fatalf("log output leaks auth token: %q", output)
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("v2.4.0", "2.3.9") <= 0 {
		t.Fatal("v2.4.0 should be newer than 2.3.9")
	}
	if compareVersions("v2.3.0", "2.3.0") != 0 {
		t.Fatal("same versions should compare equal")
	}
}

func setupSamplingRunTest(t *testing.T, storeID string) SamplingConfig {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	tokens := &CapturedTokens{
		XAppCode:        "app-code",
		QueryAuth:       "query-auth",
		ReservationAuth: "reservation-auth",
		UserAgent:       "test-agent",
		Referer:         "https://example.test",
		WechatID:        "wechat-id",
		PhoneNumber:     "13800138000",
		StoreIDs:        []string{storeID},
	}
	if err := SaveLocalConfig(tokens); err != nil {
		t.Fatalf("save local config: %v", err)
	}
	return NormalizeSamplingConfig(SamplingConfig{
		IntervalSeconds: 60,
		ActiveStart:     "000000",
		ActiveEnd:       "235959",
		StoreIDs:        []string{storeID},
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func stubSamplingHTTP(t *testing.T, handler func(*http.Request) (int, string)) func() {
	t.Helper()
	samplingHTTPTransportMu.Lock()
	oldTransport := http.DefaultTransport
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			http.DefaultTransport = oldTransport
			samplingHTTPTransportMu.Unlock()
		})
	}
	t.Cleanup(cleanup)
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		status, body := handler(r)
		if status == http.StatusNotFound {
			return nil, fmt.Errorf("%s", body)
		}
		return &http.Response{
			StatusCode: status,
			Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})
	return cleanup
}

func captureStdout(t *testing.T, fn func()) (result string) {
	t.Helper()
	samplingStdoutMu.Lock()
	defer samplingStdoutMu.Unlock()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = writer
	output := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, reader)
		output <- buf.String()
	}()
	defer func() {
		os.Stdout = oldStdout
		_ = writer.Close()
		result = <-output
		_ = reader.Close()
	}()
	fn()
	return result
}

// TestSamplingWindowUsesCSTNotLocalTime 验证采样活跃窗口按 CST 解释，不被机器本地时区带偏。
// 关键回归：TZ=UTC 时，CST 10:00 (=UTC 02:00) 必须落在 [100000,220000] 窗口内；
//
//	若用 time.Local 解释，UTC 02:00 会被判为「不在窗口」而空转。
func TestSamplingWindowUsesCSTNotLocalTime(t *testing.T) {
	cfg := SamplingConfig{ActiveStart: "100000", ActiveEnd: "220000"}
	// CST 10:30 = UTC 02:30。用 UTC 时刻喂进去，窗口判定必须仍认它在 CST 10:00-22:00 内。
	cst1030 := time.Date(2026, 6, 3, 2, 30, 0, 0, time.UTC)
	if !samplingInActiveWindow(cfg, cst1030) {
		t.Fatal("CST 10:30 (UTC 02:30) should be inside the 10:00-22:00 window; tz normalization regressed")
	}
	// CST 22:30 = UTC 14:30，应落在窗口外（22:00 关门）。
	cst2230 := time.Date(2026, 6, 3, 14, 30, 0, 0, time.UTC)
	if samplingInActiveWindow(cfg, cst2230) {
		t.Fatal("CST 22:30 (UTC 14:30) should be outside the 10:00-22:00 window")
	}
	// nextSamplingWindowStart：CST 22:30 的下一次开窗应是次日 CST 10:00。
	next := nextSamplingWindowStart(cfg, cst2230)
	want := time.Date(2026, 6, 4, 2, 0, 0, 0, time.UTC) // 次日 CST 10:00 = UTC 02:00
	if !next.Equal(want) {
		t.Fatalf("next window = %v (UTC), want %v", next.UTC(), want)
	}
}
