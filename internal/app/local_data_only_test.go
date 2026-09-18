package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

func localDataTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.MkdirAll(AppDirPath(), 0o700); err != nil {
		t.Fatal(err)
	}
	return AppDirPath()
}

func TestRetiredCloudEndpointsDoNotRedirectOrSaveSession(t *testing.T) {
	dir := localDataTestHome(t)
	mux := http.NewServeMux()
	registerRetiredCloudRoutes(mux)
	for _, path := range []string{"/api/cloud", "/api/cloud/auth", "/api/cloud/auth/start", "/api/cloud/auth/callback?session=secret", "/api/cloud/auth/test", "/api/cloud/auth/logout"} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(`{"worker_url":"https://example.invalid"}`)))
			if w.Code != http.StatusGone || w.Header().Get("Location") != "" || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("%s %s: %d %v", method, path, w.Code, w.Header())
			}
			if strings.Contains(w.Body.String(), "secret") {
				t.Fatal("callback token echoed")
			}
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "cloud_auth.json")); !os.IsNotExist(err) {
		t.Fatalf("session file created: %v", err)
	}
}

func TestLocalHistoryIgnoresLegacyCloudConfiguration(t *testing.T) {
	dir := localDataTestHome(t)
	var calls atomic.Int32
	trap := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer trap.Close()
	for _, key := range []string{"SUSHIRO_BASELINE_TURSO_URL", "TURSO_DATABASE_URL", "SUSHIRO_CLOUD_URL"} {
		t.Setenv(key, trap.URL)
	}
	for _, key := range []string{"SUSHIRO_BASELINE_TURSO_TOKEN", "TURSO_AUTH_TOKEN", "SUSHIRO_CLOUD_SESSION_TOKEN"} {
		t.Setenv(key, "legacy-secret")
	}
	for name, value := range map[string]any{
		"cloud_auth.json":            map[string]string{"base_url": trap.URL, "session_token": "legacy-secret"},
		"queue_baseline_remote.json": map[string]string{"database_url": trap.URL, "auth_token": "legacy-secret"},
	} {
		data, _ := json.Marshal(value)
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 9, 13, 18, 0, 0, 0, SushiroTimezone)
	record := QueueBaselineRecord{CollectedAt: now.Add(-time.Minute).Format(time.RFC3339), StoreID: 1012, Name: "local store", WaitMinutes: 30, GroupQueuesCount: 10, DisplayCalledNo: 100, StoreStatus: "OPEN", OnlineOpen: true}
	data, _ := json.Marshal(record)
	if err := os.WriteFile(queueBaselineRecordsPath(), append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	dashboard := BuildQueueDashboard(QueueDashboardQuery{StoreIDs: []string{"1012"}}, now)
	trends := BuildQueueTrends(QueueTrendQuery{StoreIDs: []string{"1012"}}, now)
	pressure := buildQueuePressureCurve(context.Background(), "1012", "2026-09-13", now)
	for _, status := range []QueueBaselineRemoteStatus{dashboard.Baseline, trends.Baseline, pressure.Baseline} {
		if status.Provider != "local" || status.Configured || status.Used || status.Authenticated {
			t.Fatalf("cloud status active: %+v", status)
		}
	}
	if dashboard.Summary.RemoteStores != 0 || dashboard.Summary.RemoteRollups != 0 {
		t.Fatal("local data reported as remote")
	}
	if len(dashboard.Heatmap) != 1 || dashboard.Heatmap[0].Weekday != 7 {
		t.Fatalf("local Sunday rollup missing: %+v", dashboard.Heatmap)
	}
	if len(trends.Latest) != 1 || trends.Latest[0].StoreID != 1012 {
		t.Fatalf("local latest missing: %+v", trends.Latest)
	}
	w := httptest.NewRecorder()
	handleQueueBaseline(w, httptest.NewRequest(http.MethodGet, "/api/queue/baseline", nil))
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "legacy-secret") {
		t.Fatalf("baseline response: %s", w.Body.String())
	}
	if calls.Load() != 0 {
		t.Fatalf("made %d legacy cloud requests", calls.Load())
	}
	w = httptest.NewRecorder()
	handleLocalRecords(w, httptest.NewRequest(http.MethodGet, "/api/records?days=all", nil))
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "legacy-secret") || calls.Load() != 0 {
		t.Fatalf("bundled history read legacy cloud configuration: status=%d calls=%d", w.Code, calls.Load())
	}
}

func TestLocalBaselineContractDeduplicatesAndUsesISOWeekdays(t *testing.T) {
	localDataTestHome(t)
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, SushiroTimezone)
	base := QueueBaselineRecord{StoreID: 1012, StoreStatus: "OPEN", OnlineOpen: true, WaitMinutes: 20, GroupQueuesCount: 10, DisplayCalledNo: 100}
	var records []QueueBaselineRecord
	for _, stamp := range []string{"2026-09-12T16:05:00Z", "2026-09-13T00:05:00+08:00", "2026-09-13T00:15:00+08:00", "invalid", "2027-01-01T00:00:00Z"} {
		r := base
		r.CollectedAt = stamp
		records = append(records, r)
	}
	export := buildLocalQueueBaselineExport(records, now)
	if export.Version != 1 || export.Source != "local" || export.BucketMinutes != 30 {
		t.Fatalf("wrong contract: %+v", export)
	}
	if len(export.Rollups) != 1 || len(export.Latest) != 1 || len(export.Stores) != 1 {
		t.Fatalf("invalid records were included: %+v", export)
	}
	r := export.Rollups[0]
	if r.Weekday != 7 || r.TimeBucket != "00:00" || r.SampleCount != 2 || r.CalledSampleCount != 2 || r.BusyRate != 1 || r.WaitTypicalMinutes == nil || *r.WaitTypicalMinutes != 20 {
		t.Fatalf("wrong aggregation: %+v", r)
	}
	if export.Latest[0].CollectedAt != "2026-09-13T00:15:00+08:00" {
		t.Fatal("latest snapshot was not retained")
	}
	empty := buildLocalQueueBaselineExport(nil, now)
	if empty.Stores == nil || empty.Rollups == nil {
		t.Fatal("empty contract must contain arrays")
	}
}

func TestMCPDropsAndRejectsLegacyDatabaseFields(t *testing.T) {
	localDataTestHome(t)
	if err := os.WriteFile(MCPConfigPath(), []byte(`{"enabled":false,"auto_start":true,"turso_url":"https://example.invalid","turso_token":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := LoadMCPConfig()
	if cfg.Enabled || !cfg.AutoStart {
		t.Fatalf("local settings lost: %+v", cfg)
	}
	if err := SaveMCPConfig(cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(MCPConfigPath())
	if err != nil || strings.Contains(string(data), "turso") || strings.Contains(string(data), "secret") {
		t.Fatalf("database fields retained: %s %v", data, err)
	}
	w := httptest.NewRecorder()
	handleMCP(w, httptest.NewRequest(http.MethodPost, "/api/mcp", strings.NewReader(`{"turso_url":"https://example.invalid"}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("legacy configuration accepted: %d", w.Code)
	}
}

func TestPublicBaselineSelectionAndPausePersistWithoutCredentials(t *testing.T) {
	localDataTestHome(t)
	for _, enabled := range []bool{true, false} {
		body, _ := json.Marshal(QueueBaselineConfig{Enabled: enabled, IntervalMinutes: 5, StoreIDs: []string{"1012"}})
		w := httptest.NewRecorder()
		handleQueueBaseline(w, httptest.NewRequest(http.MethodPost, "/api/queue/baseline", strings.NewReader(string(body))))
		if w.Code != http.StatusOK {
			t.Fatalf("save public collection: %d %s", w.Code, w.Body.String())
		}
		cfg := LoadQueueBaselineConfig()
		ids := queueBaselineStoreIDs(cfg)
		if cfg.Enabled != enabled || len(ids) != 1 || ids[0] != "1012" {
			t.Fatalf("incorrect config: %+v", cfg)
		}
	}
	if HasValidConfig() {
		t.Fatal("public collection must not require or create auth configuration")
	}
}

func TestWebUIDoesNotOfferCloudAuthentication(t *testing.T) {
	for _, path := range []string{"webui/index.html", "webui/app.js"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"/api/cloud/auth", "startCloudLogin", "mcpDBURL", "mcpDBToken", "登录 GitHub", "loadCloudAuth"} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("%s still includes %s", path, forbidden)
			}
		}
	}
}
