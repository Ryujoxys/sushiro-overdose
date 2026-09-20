package app

import (
	"encoding/json"
	"fmt"
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
			if w.Code != http.StatusConflict || w.Header().Get("Content-Disposition") != "" {
				t.Fatalf("empty export must fail without a download: %d %s", w.Code, w.Body.String())
			}
		} else {
			handleLocalRecords(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("%s: %d", path, w.Code)
			}
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

func TestLocalRecordsExportCountsMatchPersonalRawSnapshots(t *testing.T) {
	reliabilityHome(t)
	base := time.Now().In(time.FixedZone("CST", 8*3600)).AddDate(0, 0, -1)
	base = time.Date(base.Year(), base.Month(), base.Day(), 12, 0, 0, 0, base.Location())
	rows := []QueueBaselineRecord{
		{StoreID: 1, CollectedAt: base.Format(time.RFC3339), Name: "店一", StoreStatus: "OPEN", WaitMinutes: 30},
		{StoreID: 1, CollectedAt: base.UTC().Format(time.RFC3339), Name: "店一", StoreStatus: "OPEN", WaitMinutes: 30},
		{StoreID: 1, CollectedAt: base.Add(5 * time.Minute).Format(time.RFC3339), Name: "店一", StoreStatus: "CLOSED", WaitMinutes: 0},
		{StoreID: 2, CollectedAt: base.Format(time.RFC3339), Name: "店二", StoreStatus: "OPEN", WaitMinutes: 20},
		{StoreID: 1, CollectedAt: base.AddDate(0, 0, -400).Format(time.RFC3339), Name: "店一", StoreStatus: "OPEN", WaitMinutes: 10},
		{StoreID: 1, CollectedAt: base.AddDate(0, 0, 3).Format(time.RFC3339), Name: "店一", StoreStatus: "OPEN", WaitMinutes: 90},
	}
	if err := appendQueueBaselineRecords(rows); err != nil {
		t.Fatal(err)
	}
	dateType := queueTrendDateType(base, nil, nil)
	for _, tc := range []struct {
		name, query string
		want        int
	}{
		{"default-store", "days=all", 3},
		{"selected-store", "days=all&store=2", 1},
		{"recent", "days=7&store=1", 2},
		{"specific-date", "days=all&store=1&date=" + base.Format("2006-01-02"), 2},
		{"date-type", "days=7&store=1&date_type=" + dateType, 2},
		{"different-date", "days=all&store=1&date=2000-01-01", 0},
		{"unknown-store", "days=all&store=999999", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, includeHistory := range []bool{true, false} {
				if err := saveRecordViewSettings(recordViewSettings{IncludeHistory: includeHistory}); err != nil {
					t.Fatal(err)
				}
				w := httptest.NewRecorder()
				handleLocalRecords(w, httptest.NewRequest("GET", "/api/records?"+tc.query, nil))
				var view localRecordsResponse
				if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil || w.Code != http.StatusOK {
					t.Fatalf("view: %d %s", w.Code, w.Body.String())
				}
				if view.ExportRecordCount != tc.want || view.TotalRecordCount != 4 {
					t.Fatalf("history=%t: export count=%d total=%d, want %d/4", includeHistory, view.ExportRecordCount, view.TotalRecordCount, tc.want)
				}
				request := httptest.NewRequest("GET", "/api/records/export?"+tc.query, nil)
				query := request.URL.Query()
				query.Set("store", fmt.Sprint(view.SelectedStore))
				request.URL.RawQuery = query.Encode()
				w = httptest.NewRecorder()
				handleLocalRecordsExport(w, request)
				if tc.want == 0 {
					var result map[string]string
					if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != http.StatusConflict || result["error"] != "当前筛选下没有可导出的个人记录。" || w.Header().Get("Content-Disposition") != "" {
						t.Fatalf("empty export: %d %s", w.Code, w.Body.String())
					}
				} else if w.Code != http.StatusOK || strings.Count(w.Body.String(), "\n") != tc.want {
					t.Fatalf("export count mismatch: %d %s", w.Code, w.Body.String())
				}
			}
		})
	}
	w := httptest.NewRecorder()
	handleLocalRecordsExport(w, httptest.NewRequest("GET", "/api/records/export?days=all", nil))
	if w.Code != http.StatusOK || strings.Count(w.Body.String(), "\n") != 4 || !strings.Contains(w.Body.String(), base.AddDate(0, 0, -400).Format(time.RFC3339)) {
		t.Fatalf("full backup must include every store and records older than one year: %d %s", w.Code, w.Body.String())
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
