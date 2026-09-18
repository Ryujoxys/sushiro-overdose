package app

import (
	"sort"
	"strconv"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

const queueBaselineSchemaVersion = 1

// The legacy status fields remain false so old clients cannot infer a cloud session.
func localQueueBaselineStatus() QueueBaselineRemoteStatus {
	return QueueBaselineRemoteStatus{Provider: "local", Message: "仅使用本机历史数据，不连接线上数据库。"}
}

// buildLocalQueueBaselineExport shares the wire contract with future data providers.
// It only accepts public store snapshots, never credentials or personal tickets.
func buildLocalQueueBaselineExport(records []QueueBaselineRecord, now time.Time) QueueBaselineExport {
	if now.IsZero() {
		now = time.Now()
	}
	out := QueueBaselineExport{
		Version: queueBaselineSchemaVersion, GeneratedAt: now.Format(time.RFC3339),
		Source: "local", BucketMinutes: 30,
		DateTypes: []string{"weekday", "workday", "weekend", "holiday"},
		Stores:    []QueueBaselineStore{}, Latest: []QueueBaselineLatest{}, Rollups: []QueueBaselineRollup{},
	}
	type bucketKey struct {
		store, weekday int
		dateType, time string
	}
	type bucket struct {
		waits, groups, called []float64
		open, online, busy    int
		latest                time.Time
	}
	groups := map[bucketKey]*bucket{}
	latest := map[int]QueueBaselineRecord{}
	latestAt := map[int]time.Time{}
	seen := map[string]bool{}
	holidays, workdays, _ := loadQueueHolidayDates()
	var sourceUpdated time.Time
	for _, record := range records {
		normalizeQueueBaselineRecordForRead(&record)
		at, err := time.Parse(time.RFC3339, record.CollectedAt)
		if err != nil || record.StoreID <= 0 || at.After(now) {
			continue
		}
		at = at.In(SushiroTimezone)
		id := strconv.Itoa(record.StoreID) + "/" + at.Format(time.RFC3339Nano)
		if seen[id] {
			continue
		}
		seen[id] = true
		if at.After(latestAt[record.StoreID]) {
			latestAt[record.StoreID], latest[record.StoreID] = at, record
		}
		if at.After(sourceUpdated) {
			sourceUpdated = at
		}
		// Wire weekday is ISO 1..7, independent of Go/Python weekday enums.
		key := bucketKey{record.StoreID, (int(at.Weekday())+6)%7 + 1, queueTrendDateType(at, holidays, workdays), halfHourBucket(at)}
		acc := groups[key]
		if acc == nil {
			acc = &bucket{}
			groups[key] = acc
		}
		acc.waits = append(acc.waits, float64(record.WaitMinutes))
		acc.groups = append(acc.groups, float64(record.GroupQueuesCount))
		if record.DisplayCalledNo > 0 {
			acc.called = append(acc.called, float64(record.DisplayCalledNo))
		}
		if queueDashboardIsOpen(record.StoreStatus) {
			acc.open++
		}
		if record.OnlineOpen {
			acc.online++
		}
		if record.GroupQueuesCount > 0 {
			acc.busy++
		}
		if at.After(acc.latest) {
			acc.latest = at
		}
	}
	for id, record := range latest {
		out.Stores = append(out.Stores, QueueBaselineStore{StoreID: id, Name: record.Name, City: record.City, Area: record.Area, LastSeenAt: record.CollectedAt})
		out.Latest = append(out.Latest, QueueBaselineLatest{
			StoreID: id, CollectedAt: record.CollectedAt, Name: record.Name, City: record.City, Area: record.Area,
			WaitMinutes: record.WaitMinutes, GroupQueuesCount: record.GroupQueuesCount,
			StoreStatus: record.StoreStatus, NetTicketStatus: record.NetTicketStatus, ReservationStatus: record.ReservationStatus,
			OnlineOpen: record.OnlineOpen, WaitTimeCounter: record.WaitTimeCounter, WaitTimeCap: record.WaitTimeCap,
			DisplayCalledNo: record.DisplayCalledNo, GroupQueuesJSON: record.GroupQueuesJSON,
		})
	}
	for key, acc := range groups {
		n := len(acc.waits)
		rollup := QueueBaselineRollup{
			StoreID: key.store, Weekday: key.weekday, DateType: key.dateType, TimeBucket: key.time,
			SampleCount: n, OpenRate: float64(acc.open) / float64(n), OnlineOpenRate: float64(acc.online) / float64(n), BusyRate: float64(acc.busy) / float64(n),
			WaitTypicalMinutes: floatPtr(queueQuantile(acc.waits, .5)), WaitSafeMinutes: floatPtr(queueQuantile(acc.waits, .8)), WaitMaxMinutes: int(queueQuantile(acc.waits, 1)),
			QueueGroupsTypical: floatPtr(queueQuantile(acc.groups, .5)), QueueGroupsSafe: floatPtr(queueQuantile(acc.groups, .8)),
			CalledSampleCount: len(acc.called), Confidence: queueDashboardConfidence(n), UpdatedAt: acc.latest.Format(time.RFC3339),
		}
		if len(acc.called) > 0 {
			rollup.CalledNoSlow = floatPtr(queueQuantile(acc.called, .2))
			rollup.CalledNoTypical = floatPtr(queueQuantile(acc.called, .5))
			rollup.CalledNoFast = floatPtr(queueQuantile(acc.called, .8))
		}
		out.Rollups = append(out.Rollups, rollup)
	}
	sort.Slice(out.Stores, func(i, j int) bool { return out.Stores[i].StoreID < out.Stores[j].StoreID })
	sort.Slice(out.Latest, func(i, j int) bool { return out.Latest[i].StoreID < out.Latest[j].StoreID })
	sort.Slice(out.Rollups, func(i, j int) bool {
		a, b := out.Rollups[i], out.Rollups[j]
		if a.StoreID != b.StoreID {
			return a.StoreID < b.StoreID
		}
		if a.DateType != b.DateType {
			return a.DateType < b.DateType
		}
		if a.Weekday != b.Weekday {
			return a.Weekday < b.Weekday
		}
		return a.TimeBucket < b.TimeBucket
	})
	out.Stats = QueueBaselineStats{StoreCount: len(out.Stores), LatestCount: len(out.Latest), RollupCount: len(out.Rollups)}
	if !sourceUpdated.IsZero() {
		out.Stats.SourceUpdatedAt = sourceUpdated.Format(time.RFC3339)
	}
	return out
}
