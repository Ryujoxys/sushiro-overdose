package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

// Store IDs outside the bundled history keep assertions about personal samples exact.
const flowStoreA, flowStoreB = 990101, 990102

func flowRequest(t *testing.T, handler http.HandlerFunc, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload *strings.Reader
	if body == nil {
		payload = strings.NewReader("")
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = strings.NewReader(string(data))
	}
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(method, target, payload))
	if w.Code != http.StatusOK {
		t.Fatalf("%s %s: HTTP %d %s", method, target, w.Code, w.Body.String())
	}
	return w
}

func flowJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	return out
}

type flowService struct {
	Config QueueBaselineConfig `json:"config"`
	State  struct {
		LastAt          string `json:"last_at"`
		LastError       string `json:"last_error"`
		PausedReason    string `json:"paused_reason"`
		IntervalSeconds int    `json:"interval_seconds"`
	} `json:"state"`
	Model struct {
		SampleCount int `json:"sample_count"`
		DayCount    int `json:"day_count"`
		StoreCount  int `json:"store_count"`
	} `json:"model"`
}

// flowSnapshot mirrors what the public detail endpoint returns for one store at one time.
func flowSnapshot(id, half int) QueueLiveStore {
	called := strconv.Itoa(100 + half*40 + id%10)
	return QueueLiveStore{
		ID: id, Name: fmt.Sprintf("测试门店 %d", id), NameKana: "深圳", Area: "南山区",
		StoreStatus: "OPEN", NetTicketStatus: "ON", Wait: 10 + half*3, GroupQueuesCount: 5 + half,
		GroupQueues: QueueLiveGroupQueues{BoothQueue: []string{called}},
	}
}

func countJSONL(t *testing.T, w *httptest.ResponseRecorder) int {
	t.Helper()
	n := 0
	scanner := bufio.NewScanner(strings.NewReader(w.Body.String()))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			n++
		}
	}
	return n
}

// TestRecordingFlowEndToEnd drives the recording page's real handlers in order:
// choose stores, start, collect, read curves, export, change interval, pause and
// clear stores. Only the network fetch is replaced; files, model and views are real.
func TestRecordingFlowEndToEnd(t *testing.T) {
	reliabilityHome(t)
	previousEngine, previousStart := engine, startQueueDaemon
	engine = &BookingEngine{state: EngineState{Status: EngineIdle}}
	daemonStarts := 0
	startQueueDaemon = func() error { daemonStarts++; return nil }
	t.Cleanup(func() { engine, startQueueDaemon = previousEngine, previousStart })

	// 1. New user: nothing selected, nothing recorded, nothing written by reading.
	service := flowJSON[flowService](t, flowRequest(t, handlePublicQueueService, "GET", "/api/queue/service", nil))
	if service.Config.Enabled || len(service.Config.StoreIDs) != 0 || service.Model.SampleCount != 0 {
		t.Fatalf("fresh service: %+v", service)
	}
	if _, err := os.Stat(queueBaselineRecordsPath()); !os.IsNotExist(err) {
		t.Fatal("reading status created a record file")
	}

	// 2. Choose two stores, as the store picker posts them. Saving alone must not start recording.
	ids := []string{strconv.Itoa(flowStoreA), strconv.Itoa(flowStoreB)}
	flowRequest(t, handleQueueBaseline, "POST", "/api/queue/baseline", map[string]any{
		"enabled": false, "store_ids": ids, "use_preference_stores": false, "interval_minutes": 5,
	})
	service = flowJSON[flowService](t, flowRequest(t, handlePublicQueueService, "GET", "/api/queue/service", nil))
	if service.Config.Enabled || strings.Join(service.Config.StoreIDs, ",") != strings.Join(ids, ",") || daemonStarts != 0 {
		t.Fatalf("saving stores changed recording state: %+v starts=%d", service.Config, daemonStarts)
	}

	// 3. Start recording: enabled, background requested once.
	service = flowJSON[flowService](t, flowRequest(t, handlePublicQueueService, "POST", "/api/queue/service", map[string]string{"action": "start"}))
	if !service.Config.Enabled || daemonStarts != 1 {
		t.Fatalf("start: %+v starts=%d", service.Config, daemonStarts)
	}

	// 4. One collection round writes two days of half-hourly snapshots for both stores,
	// through the same record/observation writers as the real collector.
	china := time.FixedZone("CST", 8*3600)
	today := time.Now().In(china)
	days := []time.Time{today.AddDate(0, 0, -2), today.AddDate(0, 0, -1)}
	collections := 0
	collector := &QueueBaselineCollector{collect: func(ctx context.Context, cfg QueueBaselineConfig) (int, error) {
		collections++
		var records []QueueBaselineRecord
		for _, raw := range queueBaselineStoreIDs(cfg) {
			id := atoi(raw)
			for _, day := range days {
				for half := 0; half < 6; half++ {
					at := time.Date(day.Year(), day.Month(), day.Day(), 17, 30*half, 0, 0, china)
					store := flowSnapshot(id, half)
					records = append(records, queueBaselineRecordFromStore(store, at.Format(time.RFC3339)))
					if err := appendQueueObservation(queueObservationFromLiveStore(store, at)); err != nil {
						return 0, err
					}
				}
			}
		}
		return len(records), appendQueueBaselineRecords(records)
	}}
	collector.tick(context.Background())
	if collections != 1 {
		t.Fatalf("collector did not run once: %d", collections)
	}
	wantRows := len(ids) * len(days) * 6

	service = flowJSON[flowService](t, flowRequest(t, handlePublicQueueService, "GET", "/api/queue/service", nil))
	if service.Model.SampleCount != wantRows || service.Model.DayCount != len(days) || service.Model.StoreCount != len(ids) {
		t.Fatalf("model after collection: %+v want %d rows", service.Model, wantRows)
	}

	// 5. Personal view only (history off) so every count below is ours.
	flowRequest(t, handleRecordViewSettings, "POST", "/api/records/settings", map[string]bool{"include_history": false})
	type recordsView struct {
		Stores []struct {
			ID      int                 `json:"id"`
			Samples int                 `json:"samples"`
			Latest  QueueBaselineRecord `json:"latest"`
		} `json:"stores"`
		Points       []localRecordPoint `json:"points"`
		CalledPoints []localCalledPoint `json:"called_points"`
		Dates        []string           `json:"dates"`
		LocalSamples int                `json:"local_samples"`
		Export       int                `json:"export_record_count"`
		Total        int                `json:"total_record_count"`
	}
	query := fmt.Sprintf("/api/records?days=all&date_type=all&store=%d", flowStoreA)
	view := flowJSON[recordsView](t, flowRequest(t, handleLocalRecords, "GET", query, nil))
	if len(view.Stores) != 2 || view.Total != wantRows || view.Export != wantRows/2 {
		t.Fatalf("records summary: stores=%d total=%d export=%d", len(view.Stores), view.Total, view.Export)
	}
	if len(view.Points) != 6 || len(view.CalledPoints) != 6 || len(view.Dates) != len(days) {
		t.Fatalf("curves: wait=%d called=%d dates=%v", len(view.Points), len(view.CalledPoints), view.Dates)
	}
	first := view.CalledPoints[0]
	if first.Time != "17:00" || first.Days != len(days) || int(first.Median) != 100+flowStoreA%10 {
		t.Fatalf("called curve start: %+v", first)
	}
	if last := view.Points[len(view.Points)-1]; last.Time != "19:30" || int(last.Median) != 25 {
		t.Fatalf("wait curve end: %+v", last)
	}
	for _, store := range view.Stores {
		if store.Samples != wantRows/2 || store.Latest.StoreStatus != "OPEN" || store.Latest.CollectedAt == "" {
			t.Fatalf("store row: %+v", store)
		}
	}

	// 6. A single date narrows the curve to that day; recent ranges still include it.
	day := days[1].Format("2006-01-02")
	single := flowJSON[recordsView](t, flowRequest(t, handleLocalRecords, "GET", query+"&date="+day, nil))
	if len(single.CalledPoints) != 6 || single.CalledPoints[0].Days != 1 || single.Export != 6 {
		t.Fatalf("single date: %+v", single.CalledPoints)
	}
	recent := flowJSON[recordsView](t, flowRequest(t, handleLocalRecords, "GET", strings.Replace(query, "days=all", "days=7", 1), nil))
	if recent.Export != wantRows/2 {
		t.Fatalf("7-day range dropped records: %d", recent.Export)
	}

	// 7. Exports: filtered view and full backup.
	if n := countJSONL(t, flowRequest(t, handleLocalRecordsExport, "GET", query, nil)); n != wantRows/2 {
		t.Fatalf("filtered export rows: %d", n)
	}
	if n := countJSONL(t, flowRequest(t, handleLocalRecordsExport, "GET", "/api/records/export?days=all", nil)); n != wantRows {
		t.Fatalf("backup rows: %d", n)
	}

	// 8. Interval change keeps recording on and is reflected in the shared state.
	flowRequest(t, handleQueueBaseline, "POST", "/api/queue/baseline", map[string]any{
		"enabled": true, "store_ids": ids, "use_preference_stores": false, "interval_minutes": 15,
	})
	collector.tick(context.Background()) // Not due yet: same stores, last round just finished.
	service = flowJSON[flowService](t, flowRequest(t, handlePublicQueueService, "GET", "/api/queue/service", nil))
	if !service.Config.Enabled || service.Config.IntervalMinutes != 15 || collections != 1 {
		t.Fatalf("interval change: %+v collections=%d", service.Config, collections)
	}

	// 9. Pause: no more collection, records and model stay.
	service = flowJSON[flowService](t, flowRequest(t, handlePublicQueueService, "POST", "/api/queue/service", map[string]string{"action": "pause"}))
	collector.tick(context.Background())
	if service.Config.Enabled || collections != 1 || service.Model.SampleCount != wantRows {
		t.Fatalf("pause: %+v collections=%d model=%+v", service.Config, collections, service.Model)
	}

	// 10. Clearing every store keeps recording off and never deletes history.
	flowRequest(t, handleQueueBaseline, "POST", "/api/queue/baseline", map[string]any{
		"enabled": false, "store_ids": []string{}, "use_preference_stores": false, "interval_minutes": 15,
	})
	service = flowJSON[flowService](t, flowRequest(t, handlePublicQueueService, "GET", "/api/queue/service", nil))
	if service.Config.Enabled || len(service.Config.StoreIDs) != 0 {
		t.Fatalf("cleared stores: %+v", service.Config)
	}
	if n := countJSONL(t, flowRequest(t, handleLocalRecordsExport, "GET", "/api/records/export?days=all", nil)); n != wantRows {
		t.Fatalf("clearing stores lost records: %d", n)
	}
	w := httptest.NewRecorder()
	handlePublicQueueService(w, httptest.NewRequest("POST", "/api/queue/service", strings.NewReader(`{"action":"start"}`)))
	if w.Code == http.StatusOK || daemonStarts != 1 {
		t.Fatalf("started without stores: %d %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(LocalConfigPath()); !os.IsNotExist(err) {
		t.Fatal("recording flow created account credentials")
	}
}

// TestRecordsRestoreIsAdditiveAndValidated restores a backup into a second data
// directory: new rows are appended once, duplicates and broken lines are skipped,
// and a file with nothing usable leaves the records untouched.
func TestRecordsRestoreIsAdditiveAndValidated(t *testing.T) {
	reliabilityHome(t)
	china := time.FixedZone("CST", 8*3600)
	day := time.Now().In(china).AddDate(0, 0, -1)
	var backup strings.Builder
	enc := json.NewEncoder(&backup)
	for half := 0; half < 4; half++ {
		at := time.Date(day.Year(), day.Month(), day.Day(), 18, 30*half, 0, 0, china)
		if err := enc.Encode(queueBaselineRecordFromStore(flowSnapshot(flowStoreA, half), at.Format(time.RFC3339))); err != nil {
			t.Fatal(err)
		}
	}
	future := queueBaselineRecordFromStore(flowSnapshot(flowStoreA, 0), time.Now().Add(48*time.Hour).Format(time.RFC3339))
	line, _ := json.Marshal(future)
	backup.WriteString("not json\n\n" + string(line) + "\n{\"store_id\":0}\n")

	restore := func(body string) (int, localRecordsImportResult, string) {
		w := httptest.NewRecorder()
		handleLocalRecordsImport(w, httptest.NewRequest("POST", "/api/records/import", strings.NewReader(body)))
		var result localRecordsImportResult
		_ = json.Unmarshal(w.Body.Bytes(), &result)
		return w.Code, result, w.Body.String()
	}
	code, result, raw := restore(backup.String())
	if code != 200 || result.Imported != 4 || result.Invalid != 3 || result.Duplicates != 0 || result.Stores != 1 {
		t.Fatalf("first restore: %d %s", code, raw)
	}
	if model, err := loadLocalQueueModel(); err != nil || model.SampleCount != 4 {
		t.Fatalf("model not rebuilt after restore: %+v %v", model, err)
	}
	code, result, raw = restore(backup.String())
	if code != 200 || result.Imported != 0 || result.Duplicates != 4 {
		t.Fatalf("second restore must only skip: %d %s", code, raw)
	}
	if n := countJSONL(t, flowRequest(t, handleLocalRecordsExport, "GET", "/api/records/export?days=all", nil)); n != 4 {
		t.Fatalf("records after restoring twice: %d", n)
	}
	before, err := os.ReadFile(queueBaselineRecordsPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"", "hello\nworld\n", strings.Repeat("x", 5<<20)} {
		if code, _, raw := restore(body); code != http.StatusBadRequest {
			t.Fatalf("unusable backup accepted: %d %s", code, raw)
		}
	}
	after, _ := os.ReadFile(queueBaselineRecordsPath())
	if string(before) != string(after) {
		t.Fatal("a rejected backup changed the record file")
	}
	w := httptest.NewRecorder()
	handleLocalRecordsImport(w, httptest.NewRequest("GET", "/api/records/import", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET import: %d", w.Code)
	}
}
