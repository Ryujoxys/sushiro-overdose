package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

type recordViewSettings struct {
	IncludeHistory bool `json:"include_history"`
}

type recordHistoryStatus struct {
	historyMetadata
	Included bool   `json:"included"`
	Error    string `json:"error,omitempty"`
}

var recordViewSettingsMu sync.Mutex

func recordViewSettingsPath() string { return filepath.Join(AppDirPath(), "record_view.json") }

func loadRecordViewSettings() (recordViewSettings, error) {
	settings := recordViewSettings{IncludeHistory: true}
	raw, err := os.ReadFile(recordViewSettingsPath())
	if os.IsNotExist(err) {
		return settings, nil
	}
	if err != nil {
		return recordViewSettings{}, err
	}
	var saved struct {
		IncludeHistory *bool `json:"include_history"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		return recordViewSettings{}, err
	}
	if saved.IncludeHistory == nil {
		return recordViewSettings{}, fmt.Errorf("missing include_history setting")
	}
	settings.IncludeHistory = *saved.IncludeHistory
	return settings, nil
}

func saveRecordViewSettings(settings recordViewSettings) error {
	recordViewSettingsMu.Lock()
	defer recordViewSettingsMu.Unlock()
	if err := os.MkdirAll(AppDirPath(), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(AppDirPath(), ".record-view-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := json.NewEncoder(f).Encode(settings); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), recordViewSettingsPath())
}

func handleRecordViewSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := loadRecordViewSettings()
		if err != nil {
			writeError(w, 500, "图表设置读取失败")
			return
		}
		writeJSON(w, settings)
	case http.MethodPost:
		var input struct {
			IncludeHistory *bool `json:"include_history"`
		}
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
		d.DisallowUnknownFields()
		if err := d.Decode(&input); err != nil || input.IncludeHistory == nil || d.Decode(new(any)) != io.EOF {
			writeError(w, 400, "请指定是否包含历史数据")
			return
		}
		settings := recordViewSettings{IncludeHistory: *input.IncludeHistory}
		if err := saveRecordViewSettings(settings); err != nil {
			writeError(w, 500, "图表设置保存失败")
			return
		}
		writeJSON(w, settings)
	default:
		writeError(w, 405, "GET or POST only")
	}
}

func buildRecordView(rows []QueueBaselineRecord, pack *historyBundle, settings recordViewSettings, q localRecordsQuery, now time.Time) localRecordsResponse {
	// Personal counts and latest snapshots never include the bundled data.
	out := summarizeLocalRecords(rows, q.store)
	out.Version = 2
	out.Samples, out.Days, out.LatestAt = 0, 0, ""
	out.Points = []localRecordPoint{}
	out.CalledPoints = []localCalledPoint{}
	out.AvailableStores = []historyStore{}
	out.Dates = []string{}
	out.History.Included = settings.IncludeHistory
	stores := map[int]historyStore{}
	if pack != nil {
		out.History.historyMetadata = pack.historyMetadata
		if settings.IncludeHistory {
			for _, s := range pack.Stores {
				if len(pack.byStore[s.ID]) > 0 {
					stores[s.ID] = s
				}
			}
		}
	}
	for _, s := range out.Stores {
		stores[s.ID] = historyStore{ID: s.ID, Name: s.Name}
	}
	for _, s := range stores {
		out.AvailableStores = append(out.AvailableStores, s)
	}
	sort.Slice(out.AvailableStores, func(i, j int) bool { return out.AvailableStores[i].ID < out.AvailableStores[j].ID })
	if q.store == 0 {
		if len(out.Stores) > 0 {
			q.store = out.Stores[0].ID
		} else if len(out.AvailableStores) > 0 {
			q.store = out.AvailableStores[0].ID
		}
	}
	out.SelectedStore = q.store
	merged := map[string]recordViewSample{}
	if pack != nil && settings.IncludeHistory {
		for _, sample := range pack.byStore[q.store] {
			merged[sample.key()] = sample
		}
	}
	holidays, workdays, _ := loadQueueHolidayDates()
	for _, row := range rows {
		if row.StoreID != q.store {
			continue
		}
		at, err := time.Parse(time.RFC3339, row.CollectedAt)
		if err != nil || at.After(now) {
			continue
		}
		at = at.In(SushiroTimezone)
		wait := row.WaitMinutes
		sample := recordViewSample{storeID: row.StoreID, at: at, dateType: queueTrendDateType(at, holidays, workdays), wait: &wait, open: row.StoreStatus == "OPEN"}
		if wait < 0 || (row.WaitTimeCap > 0 && wait > row.WaitTimeCap) {
			sample.wait = nil
		}
		if row.DisplayCalledNo > 0 && row.DisplayCalledNo <= 2147483647 {
			called := row.DisplayCalledNo
			sample.called = &called
		}
		key := sample.key()
		old, exists := merged[key]
		// A closed/unknown local representative also supersedes the historical one.
		if !exists || old.history || sample.at.After(old.at) {
			merged[key] = sample
		}
	}
	type bucket struct {
		waits      []float64
		days       map[string]bool
		called     []float64
		calledDays map[string]bool
	}
	buckets := map[string]*bucket{}
	days, dates := map[string]bool{}, map[string]bool{}
	calledDays := map[string]bool{}
	var latest time.Time
	for _, s := range merged {
		if !recordDateMatches(s.at, s.dateType, q, now, false) {
			continue
		}
		if s.open && (s.wait != nil || s.called != nil) {
			dates[s.at.Format("2006-01-02")] = true
		}
		if !recordDateMatches(s.at, s.dateType, q, now, true) || !s.open || (s.wait == nil && s.called == nil) {
			continue
		}
		day, key := s.at.Format("2006-01-02"), halfHourBucket(s.at)
		if buckets[key] == nil {
			buckets[key] = &bucket{days: map[string]bool{}, calledDays: map[string]bool{}}
		}
		b := buckets[key]
		if s.wait != nil {
			b.waits = append(b.waits, float64(*s.wait))
			b.days[day], days[day] = true, true
			out.Samples++
			if s.history {
				out.HistorySamples++
			} else {
				out.LocalSamples++
			}
		}
		if s.called != nil {
			b.called = append(b.called, float64(*s.called))
			b.calledDays[day], calledDays[day] = true, true
			if s.history {
				out.CalledHistorySamples++
			} else {
				out.CalledLocalSamples++
			}
		}
		if s.at.After(latest) {
			latest = s.at
			out.LatestAt = s.at.Format(time.RFC3339)
		}
	}
	for key, b := range buckets {
		if len(b.waits) > 0 {
			out.Points = append(out.Points, localRecordPoint{Time: key, Samples: len(b.waits), Days: len(b.days), Median: queueQuantile(b.waits, .5), Upper: queueQuantile(b.waits, .8)})
		}
		if len(b.called) > 0 {
			out.CalledPoints = append(out.CalledPoints, localCalledPoint{Time: key, Samples: len(b.called), Days: len(b.calledDays), Lower: queueQuantile(b.called, .2), Median: queueQuantile(b.called, .5), Upper: queueQuantile(b.called, .8)})
		}
	}
	sort.Slice(out.Points, func(i, j int) bool { return out.Points[i].Time < out.Points[j].Time })
	sort.Slice(out.CalledPoints, func(i, j int) bool { return out.CalledPoints[i].Time < out.CalledPoints[j].Time })
	for date := range dates {
		out.Dates = append(out.Dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out.Dates)))
	out.Days = len(days)
	out.CalledDays = len(calledDays)
	if out.HistorySamples+out.CalledHistorySamples > 0 {
		out.Source = "bundled_history"
		if out.LocalSamples+out.CalledLocalSamples > 0 {
			out.Source = "mixed"
		}
	}
	return out
}

func recordDateMatches(at time.Time, dateType string, q localRecordsQuery, now time.Time, includeDate bool) bool {
	at, now = at.In(SushiroTimezone), now.In(SushiroTimezone)
	if at.After(now) {
		return false
	}
	if q.days > 0 {
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, SushiroTimezone).AddDate(0, 0, 1-q.days)
		if at.Before(start) {
			return false
		}
	}
	return (q.dateType == "" || q.dateType == "all" || q.dateType == dateType) && (!includeDate || q.date == "" || at.Format("2006-01-02") == q.date)
}

func validateRecordDate(date string) error {
	if date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return fmt.Errorf("无效的日期，请使用 YYYY-MM-DD")
		}
	}
	return nil
}
