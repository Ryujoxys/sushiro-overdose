package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLocalRecordsFiltersDeduplicatesAndExcludesClosedFromCurve(t *testing.T) {
	now := time.Date(2026, 9, 18, 13, 0, 0, 0, time.FixedZone("CST", 8*3600))
	rows := []QueueBaselineRecord{
		{StoreID: 1, CollectedAt: "2026-09-18T12:00:00+08:00", Name: "店一", StoreStatus: "OPEN", WaitMinutes: 30},
		{StoreID: 1, CollectedAt: "2026-09-18T04:00:00Z", Name: "店一", StoreStatus: "OPEN", WaitMinutes: 30},
		{StoreID: 1, CollectedAt: "2026-09-18T12:05:00+08:00", Name: "店一", StoreStatus: "CLOSED", WaitMinutes: 0},
		{StoreID: 2, CollectedAt: "2026-09-18T12:00:00+08:00", Name: "店二", StoreStatus: "OPEN", WaitMinutes: 90},
		{StoreID: 1, CollectedAt: "2026-09-19T12:00:00+08:00", WaitMinutes: 99},
		{StoreID: 1, CollectedAt: "broken", WaitMinutes: 99},
		{StoreID: 1, CollectedAt: "2026-08-18T12:00:00+08:00", WaitMinutes: 99},
	}
	got := summarizeLocalRecords(filterLocalRecords(rows, localRecordsQuery{days: 7}, now), 1)
	if got.Samples != 2 || got.Days != 1 || len(got.Stores) != 2 {
		t.Fatalf("%+v", got)
	}
	if len(got.Points) != 1 || got.Points[0].Median != 30 || got.Points[0].Samples != 1 {
		t.Fatalf("closed or other-store samples leaked: %+v", got.Points)
	}
}

func TestLocalRecordsEmptyAndExportAreReadOnly(t *testing.T) {
	reliabilityHome(t)
	before := notificationCountForTest("desktop")
	for _, path := range []string{"/api/records", "/api/records/export"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path, nil)
		if strings.Contains(path, "export") {
			handleLocalRecordsExport(w, r)
		} else {
			handleLocalRecords(w, r)
		}
		if w.Code != 200 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	if _, err := os.Stat(queueBaselineRecordsPath()); !os.IsNotExist(err) {
		t.Fatal("read created records")
	}
	if notificationCountForTest("desktop") != before {
		t.Fatal("read sent a notification")
	}
	if err := appendQueueBaselineRecords([]QueueBaselineRecord{{StoreID: 1, CollectedAt: time.Now().Add(-time.Minute).Format(time.RFC3339), Name: "记录", StoreStatus: "OPEN", WaitMinutes: 20}}); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handleLocalRecordsExport(w, httptest.NewRequest("GET", "/api/records/export?store=1", nil))
	var row QueueBaselineRecord
	if err := json.Unmarshal(w.Body.Bytes(), &row); err != nil || row.StoreID != 1 {
		t.Fatalf("invalid export: %s", w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatal("export not an attachment")
	}
}

func TestLocalRecordsRejectsInvalidQueryAndWrites(t *testing.T) {
	for _, path := range []string{"/api/records?days=-1", "/api/records?store=abc", "/api/records?date_type=nope", "/api/records?date=2026-02-30", "/api/records?date=20260901"} {
		w := httptest.NewRecorder()
		handleLocalRecords(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 400 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	w := httptest.NewRecorder()
	handleLocalRecords(w, httptest.NewRequest("POST", "/api/records", nil))
	if w.Code != 405 {
		t.Fatal(w.Code)
	}
}

func TestRetiredAutomationEndpointsCannotExecuteOrWrite(t *testing.T) {
	reliabilityHome(t)
	mux := http.NewServeMux()
	registerRetiredAutomationRoutes(mux)
	for _, path := range []string{"/api/sniper/start", "/api/sniper/plan", "/api/engine/booking", "/api/queue/ticket/plan", "/api/queue/ticket/routine"} {
		for _, method := range []string{"GET", "POST"} {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(`{"enabled":true}`)))
			if w.Code != http.StatusGone {
				t.Fatalf("%s %s: %d", method, path, w.Code)
			}
		}
	}
	if _, err := os.Stat(netTicketPlanPath()); !os.IsNotExist(err) {
		t.Fatal("retired API wrote a plan")
	}
}
