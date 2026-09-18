package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

type localRecordStore struct {
	ID      int                 `json:"id"`
	Name    string              `json:"name"`
	Samples int                 `json:"samples"`
	Days    int                 `json:"days"`
	Latest  QueueBaselineRecord `json:"latest"`
}

type localRecordPoint struct {
	Time    string  `json:"time"`
	Samples int     `json:"samples"`
	Days    int     `json:"days"`
	Median  float64 `json:"median"`
	Upper   float64 `json:"upper"`
}

type localCalledPoint struct {
	Time    string  `json:"time"`
	Samples int     `json:"samples"`
	Days    int     `json:"days"`
	Lower   float64 `json:"lower"`
	Median  float64 `json:"median"`
	Upper   float64 `json:"upper"`
}

type localRecordsResponse struct {
	Version              int                 `json:"version"`
	Source               string              `json:"source"`
	Samples              int                 `json:"samples"`
	Days                 int                 `json:"days"`
	Stores               []localRecordStore  `json:"stores"`
	Points               []localRecordPoint  `json:"points"`
	CalledPoints         []localCalledPoint  `json:"called_points"`
	CalledLocalSamples   int                 `json:"called_local_samples"`
	CalledHistorySamples int                 `json:"called_history_samples"`
	CalledDays           int                 `json:"called_days"`
	SelectedStore        int                 `json:"selected_store"`
	LatestAt             string              `json:"latest_at"`
	AvailableStores      []historyStore      `json:"available_stores"`
	Dates                []string            `json:"dates"`
	LocalSamples         int                 `json:"local_samples"`
	HistorySamples       int                 `json:"history_samples"`
	History              recordHistoryStatus `json:"history"`
}

type localRecordsQuery struct {
	store, days int
	dateType    string
	date        string
}

func localRecordsQueryFromRequest(r *http.Request) (localRecordsQuery, error) {
	q := localRecordsQuery{dateType: r.URL.Query().Get("date_type"), date: r.URL.Query().Get("date")}
	if err := validateRecordDate(q.date); err != nil {
		return q, err
	}
	if raw := r.URL.Query().Get("days"); raw != "" && raw != "all" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 365 {
			return q, fmt.Errorf("记录范围需为 1 到 365 天")
		}
		q.days = n
	}
	if raw := r.URL.Query().Get("store"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return q, fmt.Errorf("无效的门店 ID")
		}
		q.store = n
	}
	switch q.dateType {
	case "", "all", "weekday", "workday", "weekend", "holiday":
	default:
		return q, fmt.Errorf("无效的日期类型")
	}
	return q, nil
}

func readLocalRecords(q localRecordsQuery, now time.Time) ([]QueueBaselineRecord, error) {
	rows, err := parseJSONLFile[QueueBaselineRecord](queueBaselineRecordsPath(), normalizeQueueBaselineRecordForRead)
	if os.IsNotExist(err) {
		return []QueueBaselineRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	return filterLocalRecords(rows, q, now), nil
}

func filterLocalRecords(rows []QueueBaselineRecord, q localRecordsQuery, now time.Time) []QueueBaselineRecord {
	now = now.In(SushiroTimezone)
	holidays, workdays, _ := loadQueueHolidayDates()
	seen := map[string]bool{}
	out := []QueueBaselineRecord{}
	for _, row := range rows {
		normalizeQueueBaselineRecordForRead(&row)
		at, err := time.Parse(time.RFC3339, row.CollectedAt)
		if err != nil || row.StoreID <= 0 {
			continue
		}
		if !recordDateMatches(at, queueTrendDateType(at.In(SushiroTimezone), holidays, workdays), q, now, true) {
			continue
		}
		key := strconv.Itoa(row.StoreID) + "/" + at.UTC().Format(time.RFC3339Nano)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339, out[i].CollectedAt)
		b, _ := time.Parse(time.RFC3339, out[j].CollectedAt)
		return a.Before(b)
	})
	return out
}

func summarizeLocalRecords(rows []QueueBaselineRecord, selected int) localRecordsResponse {
	out := localRecordsResponse{Version: 1, Source: "local", SelectedStore: selected, Stores: []localRecordStore{}, Points: []localRecordPoint{}}
	stores := map[int]*localRecordStore{}
	storeDays := map[int]map[string]bool{}
	days := map[string]bool{}
	type bucket struct {
		waits []float64
		days  map[string]bool
	}
	buckets := map[string]*bucket{}
	for _, row := range rows {
		at, err := time.Parse(time.RFC3339, row.CollectedAt)
		if err != nil {
			continue
		}
		at = at.In(SushiroTimezone)
		day := at.Format("2006-01-02")
		if stores[row.StoreID] == nil {
			stores[row.StoreID] = &localRecordStore{ID: row.StoreID}
			storeDays[row.StoreID] = map[string]bool{}
		}
		s := stores[row.StoreID]
		s.Samples++
		storeDays[row.StoreID][day] = true
		s.Name, s.Latest = row.Name, row
		if selected != 0 && row.StoreID != selected {
			continue
		}
		out.Samples++
		days[day] = true
		out.LatestAt = row.CollectedAt
		// Closed/unknown stores are retained in the raw history, not treated as zero waiting.
		if selected == 0 || row.StoreStatus != "OPEN" || row.WaitMinutes < 0 {
			continue
		}
		key := halfHourBucket(at)
		b := buckets[key]
		if b == nil {
			b = &bucket{days: map[string]bool{}}
			buckets[key] = b
		}
		b.waits = append(b.waits, float64(row.WaitMinutes))
		b.days[day] = true
	}
	for id, s := range stores {
		s.Days = len(storeDays[id])
		out.Stores = append(out.Stores, *s)
	}
	sort.Slice(out.Stores, func(i, j int) bool { return out.Stores[i].ID < out.Stores[j].ID })
	for key, b := range buckets {
		out.Points = append(out.Points, localRecordPoint{Time: key, Samples: len(b.waits), Days: len(b.days), Median: queueQuantile(b.waits, .5), Upper: queueQuantile(b.waits, .8)})
	}
	sort.Slice(out.Points, func(i, j int) bool { return out.Points[i].Time < out.Points[j].Time })
	out.Days = len(days)
	return out
}

func handleLocalRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "GET only")
		return
	}
	q, err := localRecordsQueryFromRequest(r)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	now := time.Now()
	rows, err := readLocalRecords(localRecordsQuery{}, now)
	if err != nil {
		writeError(w, 500, "本机记录读取失败")
		return
	}
	settings, err := loadRecordViewSettings()
	if err != nil {
		writeError(w, 500, "图表设置读取失败")
		return
	}
	pack, packErr := loadBundledHistory()
	out := buildRecordView(rows, pack, settings, q, now)
	if packErr != nil {
		out.History.Error = "内置历史读取失败，当前只显示本机记录"
	}
	writeJSON(w, out)
}

func handleLocalRecordsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "GET only")
		return
	}
	q, err := localRecordsQueryFromRequest(r)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	rows, err := readLocalRecords(q, time.Now())
	if err != nil {
		writeError(w, 500, "本机记录读取失败")
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="sushiro-records.jsonl"`)
	w.Header().Set("Cache-Control", "no-store")
	enc := json.NewEncoder(w)
	for _, row := range rows {
		if q.store != 0 && q.store != row.StoreID {
			continue
		}
		if err := enc.Encode(row); err != nil {
			return
		}
	}
}
