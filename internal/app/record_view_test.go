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

func historyTestNow() time.Time {
	t, _ := time.Parse(time.RFC3339, "2026-09-18T12:00:00+08:00")
	return t
}

func TestRecordViewHistoryKeepsZeroAndMissingDistinct(t *testing.T) {
	reliabilityHome(t)
	got := buildRecordView(nil, fixtureHistory(t), recordViewSettings{true}, localRecordsQuery{}, historyTestNow())
	if got.SelectedStore != 1 || got.Source != "bundled_history" || got.Samples != 2 || got.HistorySamples != 2 || got.LocalSamples != 0 || len(got.Stores) != 0 {
		t.Fatalf("%+v", got)
	}
	if len(got.Points) != 1 || got.Points[0].Median != 30 || got.Points[0].Days != 2 {
		t.Fatalf("unknown or closed became zero: %+v", got.Points)
	}
	if len(got.Dates) != 2 || got.Dates[0] != "2026-09-02" {
		t.Fatal(got.Dates)
	}
	if !got.History.Included || got.History.CutoffAt != "2026-09-02T12:30:00+08:00" {
		t.Fatal(got.History)
	}
}

func TestRecordViewLocalRepresentativeWinsWithoutFrequencyBias(t *testing.T) {
	reliabilityHome(t)
	rows := []QueueBaselineRecord{
		{StoreID: 1, Name: "本机店名", CollectedAt: "2026-09-01T04:05:00Z", StoreStatus: "OPEN", WaitMinutes: 5},
		{StoreID: 1, Name: "本机店名", CollectedAt: "2026-09-01T12:25:00+08:00", StoreStatus: "OPEN", WaitMinutes: 20},
		{StoreID: 1, Name: "本机店名", CollectedAt: "2026-09-01T12:10:00+08:00", StoreStatus: "OPEN", WaitMinutes: 10},
	}
	got := buildRecordView(rows, fixtureHistory(t), recordViewSettings{true}, localRecordsQuery{store: 1}, historyTestNow())
	if got.Samples != 2 || got.HistorySamples != 1 || got.LocalSamples != 1 || got.Points[0].Median != 40 || got.Stores[0].Samples != 3 || got.Source != "mixed" {
		t.Fatalf("%+v", got)
	}
	rows = append(rows, QueueBaselineRecord{StoreID: 1, CollectedAt: "2026-09-01T12:29:00+08:00", StoreStatus: "CLOSED", WaitMinutes: 0})
	got = buildRecordView(rows, fixtureHistory(t), recordViewSettings{true}, localRecordsQuery{store: 1}, historyTestNow())
	if got.Samples != 1 || got.Points[0].Median != 60 {
		t.Fatal("closed local row fell back to history", got.Points)
	}
}

func TestRecordViewKeepsCitiesForPersonalStoreSearch(t *testing.T) {
	reliabilityHome(t)
	pack := fixtureHistory(t)
	pack.Stores = []historyStore{{ID: 1, Name: "历史店名", City: "深圳"}}
	for _, tc := range []struct {
		name, city, want string
	}{
		{"local city", "广州", "广州"},
		{"legacy fallback", "", "深圳"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := []QueueBaselineRecord{
				{StoreID: 1, Name: "海岸城", City: tc.city, CollectedAt: "2026-09-18T11:00:00+08:00", StoreStatus: "OPEN", WaitMinutes: 12},
				{StoreID: 1, CollectedAt: "2026-09-18T11:05:00+08:00", StoreStatus: "OPEN", WaitMinutes: 15},
			}
			for _, includeHistory := range []bool{true, false} {
				got := buildRecordView(rows, pack, recordViewSettings{includeHistory}, localRecordsQuery{store: 1}, historyTestNow())
				if len(got.Stores) != 1 || got.Stores[0].Name != "海岸城" || got.Stores[0].City != tc.want {
					t.Fatalf("personal metadata lost (history=%t): %+v", includeHistory, got.Stores)
				}
				if len(got.AvailableStores) != 1 || got.AvailableStores[0].Name != "海岸城" || got.AvailableStores[0].City != tc.want {
					t.Fatalf("search metadata lost (history=%t): %+v", includeHistory, got.AvailableStores)
				}
				if !includeHistory && got.HistorySamples != 0 {
					t.Fatal("metadata fallback reenabled historical samples")
				}
			}
		})
	}
	got := buildRecordView([]QueueBaselineRecord{{StoreID: 2, Name: "新店", City: "广州", CollectedAt: "2026-09-18T11:00:00+08:00", StoreStatus: "OPEN", WaitMinutes: 12}}, nil, recordViewSettings{}, localRecordsQuery{store: 2}, historyTestNow())
	if len(got.AvailableStores) != 1 || got.AvailableStores[0].City != "广州" {
		t.Fatalf("new local store city missing without bundle: %+v", got.AvailableStores)
	}
}

func TestRecordViewFiltersAndOptOutNeverFallback(t *testing.T) {
	reliabilityHome(t)
	pack := fixtureHistory(t)
	for _, tc := range []struct {
		name     string
		settings recordViewSettings
		q        localRecordsQuery
		samples  int
	}{
		{"all", recordViewSettings{true}, localRecordsQuery{}, 2},
		{"off", recordViewSettings{false}, localRecordsQuery{}, 0},
		{"off selected", recordViewSettings{false}, localRecordsQuery{store: 1}, 0},
		{"recent", recordViewSettings{true}, localRecordsQuery{days: 7}, 0},
		{"day", recordViewSettings{true}, localRecordsQuery{date: "2026-09-02"}, 1},
		{"weekend", recordViewSettings{true}, localRecordsQuery{dateType: "weekend"}, 0},
		{"different store", recordViewSettings{true}, localRecordsQuery{store: 99}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := buildRecordView(nil, pack, tc.settings, tc.q, historyTestNow())
			if got.Samples != tc.samples {
				t.Fatalf("%+v", got)
			}
		})
	}
	rows := []QueueBaselineRecord{{StoreID: 1, CollectedAt: "2026-09-18T11:00:00+08:00", StoreStatus: "OPEN", WaitMinutes: 12}}
	got := buildRecordView(rows, pack, recordViewSettings{false}, localRecordsQuery{}, historyTestNow())
	if got.HistorySamples != 0 || got.LocalSamples != 1 || got.Points[0].Median != 12 {
		t.Fatal(got)
	}
}

func TestRecordViewSettingsPersistExplicitChoiceAndRejectInvalid(t *testing.T) {
	reliabilityHome(t)
	initial, err := loadRecordViewSettings()
	if err != nil || !initial.IncludeHistory {
		t.Fatal(initial, err)
	}
	if _, err := os.Stat(recordViewSettingsPath()); !os.IsNotExist(err) {
		t.Fatal("GET wrote settings")
	}
	for _, body := range []string{`{}`, `null`, `{"include_history":null}`, `{"include_history":"false"}`, `{"include_history":false,"token":"no"}`, `{"include_history":false}{}`} {
		w := httptest.NewRecorder()
		handleRecordViewSettings(w, httptest.NewRequest("POST", "/api/records/settings", strings.NewReader(body)))
		if w.Code != 400 {
			t.Fatalf("%s: %d", body, w.Code)
		}
	}
	w := httptest.NewRecorder()
	handleRecordViewSettings(w, httptest.NewRequest("POST", "/api/records/settings", strings.NewReader(`{"include_history":false}`)))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	saved, err := loadRecordViewSettings()
	if err != nil || saved.IncludeHistory {
		t.Fatal("choice did not persist", saved, err)
	}
	w = httptest.NewRecorder()
	handleLocalRecords(w, httptest.NewRequest("GET", "/api/records", nil))
	var view localRecordsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil || view.History.Included || view.HistorySamples != 0 || len(view.Points) != 0 {
		t.Fatalf("opt-out fallback: %s", w.Body.String())
	}
	if err := os.WriteFile(recordViewSettingsPath(), []byte(`null`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRecordViewSettings(); err == nil {
		t.Fatal("invalid settings silently reenabled history")
	}
}

func TestHistoryReadDoesNotWriteIntoPersonalRecordsOrExport(t *testing.T) {
	reliabilityHome(t)
	w := httptest.NewRecorder()
	handleLocalRecords(w, httptest.NewRequest("GET", "/api/records", nil))
	var view localRecordsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil || len(view.Points) == 0 || view.ExportRecordCount != 0 || view.TotalRecordCount != 0 {
		t.Fatalf("first curve missing: %s", w.Body.String())
	}
	for _, path := range []string{queueBaselineRecordsPath(), queueModelPath(), recordViewSettingsPath()} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("view wrote personal file", path)
		}
	}
	w = httptest.NewRecorder()
	handleLocalRecordsExport(w, httptest.NewRequest("GET", "/api/records/export?days=all", nil))
	if w.Code != http.StatusConflict || w.Header().Get("Content-Disposition") != "" {
		t.Fatal("bundled history must not enable a personal export", w.Code, w.Body.String())
	}
}

func TestRecordViewSettingsUsesServerCSRFContract(t *testing.T) {
	reliabilityHome(t)
	previous := getWebCSRFToken()
	setWebCSRFToken("record-view-csrf-test")
	t.Cleanup(func() { setWebCSRFToken(previous) })
	handler := webSecurityMiddleware(http.HandlerFunc(handleRecordViewSettings))
	for _, header := range []string{"", "X-CSRF-Token", "X-Sushiro-CSRF"} {
		r := httptest.NewRequest("POST", "http://127.0.0.1:39871/api/records/settings", strings.NewReader(`{"include_history":false}`))
		r.Header.Set("Origin", "http://127.0.0.1:39871")
		if header != "" {
			r.Header.Set(header, getWebCSRFToken())
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		want := http.StatusForbidden
		if header == "X-Sushiro-CSRF" {
			want = http.StatusOK
		}
		if w.Code != want {
			t.Fatalf("header %q: %d, want %d", header, w.Code, want)
		}
	}
	source, err := os.ReadFile("webui/app.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "headers['X-Sushiro-CSRF']") || strings.Contains(string(source), "X-CSRF-Token") {
		t.Fatal("frontend diverged from server CSRF header")
	}
}

func TestCalledCurveUsesDailyRepresentativesAndSeparateMissingValues(t *testing.T) {
	reliabilityHome(t)
	pack, err := decodeHistoryBundle(fixtureHistoryArchive(t, func(p map[string]any) {
		rows := p["records"].([]map[string]any)
		rows[0]["display_called_no"] = 100
		rows[1]["display_called_no"] = 150 // Wait is missing but the called number is valid.
		rows[2]["display_called_no"] = 300
		rows[3]["display_called_no"] = 900 // Closed stores must not enter either curve.
	}))
	if err != nil {
		t.Fatal(err)
	}
	got := buildRecordView(nil, pack, recordViewSettings{true}, localRecordsQuery{}, historyTestNow())
	if len(got.CalledPoints) != 2 || got.CalledHistorySamples != 3 || got.CalledLocalSamples != 0 || got.CalledDays != 2 {
		t.Fatal(got)
	}
	point := got.CalledPoints[0]
	if point.Median != 200 || point.Lower != 140 || point.Upper != 260 || point.Samples != 2 || point.Days != 2 {
		t.Fatal(point)
	}
	if got.CalledPoints[1].Median != 150 || got.CalledPoints[1].Samples != 1 || len(got.Points) != 1 {
		t.Fatal("wait missingness or closed store contaminated called curve", got)
	}
	local := []QueueBaselineRecord{
		{StoreID: 1, CollectedAt: "2026-09-01T12:05:00+08:00", StoreStatus: "OPEN", WaitMinutes: 0, DisplayCalledNo: 180},
		{StoreID: 1, CollectedAt: "2026-09-01T04:25:00Z", StoreStatus: "OPEN", WaitMinutes: -1, DisplayCalledNo: 200},
	}
	got = buildRecordView(local, pack, recordViewSettings{true}, localRecordsQuery{date: "2026-09-01"}, historyTestNow())
	if got.CalledPoints[0].Median != 200 || got.CalledLocalSamples != 1 || got.CalledHistorySamples != 1 || len(got.Points) != 0 {
		t.Fatal("local bucket did not replace historical bucket", got)
	}
	got = buildRecordView(local, pack, recordViewSettings{false}, localRecordsQuery{}, historyTestNow())
	if got.CalledHistorySamples != 0 || len(got.CalledPoints) != 1 || got.CalledPoints[0].Median != 200 {
		t.Fatal("opt-out leaked historical called numbers", got)
	}
	local = append(local, QueueBaselineRecord{StoreID: 1, CollectedAt: "2026-09-01T12:29:00+08:00", StoreStatus: "OPEN", DisplayCalledNo: 0})
	got = buildRecordView(local, pack, recordViewSettings{true}, localRecordsQuery{date: "2026-09-01"}, historyTestNow())
	if len(got.CalledPoints) != 1 || got.CalledPoints[0].Time != "12:30" {
		t.Fatal("zero local number fell back to history or an older local number", got)
	}
}

func TestCalledCurveRetainsCalledOnlyDatesAndAppliesFilters(t *testing.T) {
	reliabilityHome(t)
	rows := []QueueBaselineRecord{
		{StoreID: 1, CollectedAt: "2026-09-18T11:00:00+08:00", StoreStatus: "OPEN", WaitMinutes: -1, DisplayCalledNo: 200},
		{StoreID: 2, CollectedAt: "2026-09-18T11:00:00+08:00", StoreStatus: "OPEN", WaitMinutes: -1, DisplayCalledNo: 9000},
		{StoreID: 1, CollectedAt: "2026-09-18T11:30:00+08:00", StoreStatus: "OPEN", WaitMinutes: -1, DisplayCalledNo: -5},
	}
	got := buildRecordView(rows, nil, recordViewSettings{false}, localRecordsQuery{store: 1, dateType: "weekday"}, historyTestNow())
	if len(got.Points) != 0 || len(got.CalledPoints) != 1 || got.CalledPoints[0].Median != 200 || len(got.Dates) != 1 || got.Dates[0] != "2026-09-18" {
		t.Fatal(got)
	}
	got = buildRecordView(rows, nil, recordViewSettings{false}, localRecordsQuery{store: 1, dateType: "weekend"}, historyTestNow())
	if len(got.CalledPoints) != 0 {
		t.Fatal("date type filter not applied to calls", got)
	}
}
