package app

import (
	. "github.com/Ryujoxys/sushiro-overdose/internal/core"

	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	queueDashboardDefaultWindowHours = 6
	queueDashboardDefaultBucketMins  = 10
	queueDashboardMaxWindowHours     = 24
	queueDashboardDayStart           = "10:00"
	queueDashboardDayEnd             = "22:00"
	queueDashboardArrivalLeadMins    = 20
)

type QueueDashboardQuery struct {
	StoreIDs      []string `json:"store_ids"`
	Scope         string   `json:"scope"`
	DateType      string   `json:"date_type"`
	WindowHours   int      `json:"window_hours"`
	BucketMinutes int      `json:"bucket_minutes"`
	TargetNo      int      `json:"target_no,omitempty"`
}

type QueueDashboardResponse struct {
	GeneratedAt       string                       `json:"generated_at"`
	Filters           QueueDashboardQuery          `json:"filters"`
	Scope             QueueDashboardScope          `json:"scope"`
	Summary           QueueDashboardSummary        `json:"summary"`
	CalledSummary     QueueDashboardCalledSummary  `json:"called_summary"`
	CalledCurve       []QueueDashboardCalledPoint  `json:"called_curve"`
	Advisor           QueueDashboardAdvisor        `json:"advisor"`
	Trend             []QueueDashboardTrendPoint   `json:"trend"`
	WeekdayProfiles   []QueueDashboardWeekday      `json:"weekday_profiles"`
	Heatmap           []QueueDashboardHeatmapPoint `json:"heatmap"`
	DateTypeSummaries []QueueDashboardDateType     `json:"date_type_summaries"`
	Sampling          QueueSamplingStatus          `json:"sampling"`
	Baseline          QueueBaselineRemoteStatus    `json:"baseline"`
	Warnings          []string                     `json:"warnings,omitempty"`
}

type QueueDashboardScope struct {
	Mode       string `json:"mode"`
	Source     string `json:"source"`
	StoreCount int    `json:"store_count"`
	Message    string `json:"message"`
}

type QueueDashboardSummary struct {
	StoreCount       int    `json:"store_count"`
	OpenStores       int    `json:"open_stores"`
	OnlineOpenStores int    `json:"online_open_stores"`
	TotalQueueGroups int    `json:"total_queue_groups"`
	TotalWaitMinutes int    `json:"total_wait_minutes"`
	TotalCalledNo    int    `json:"total_called_no,omitempty"`
	LatestAt         string `json:"latest_at,omitempty"`
	TrendDelta       int    `json:"trend_delta"`
	WindowHours      int    `json:"window_hours"`
	LocalRecords     int    `json:"local_records"`
	RemoteStores     int    `json:"remote_stores"`
	RemoteRollups    int    `json:"remote_rollups"`
}

type QueueDashboardCalledSummary struct {
	StoreID           string `json:"store_id,omitempty"`
	StoreName         string `json:"store_name,omitempty"`
	DateType          string `json:"date_type"`
	DateTypeName      string `json:"date_type_name"`
	BucketMinutes     int    `json:"bucket_minutes"`
	Start             string `json:"start"`
	End               string `json:"end"`
	SampleCount       int    `json:"sample_count"`
	DayCount          int    `json:"day_count"`
	PointCount        int    `json:"point_count"`
	LatestAt          string `json:"latest_at,omitempty"`
	LatestBucket      string `json:"latest_bucket,omitempty"`
	LatestCalledNo    int    `json:"latest_called_no,omitempty"`
	LatestQueueGroups int    `json:"latest_queue_groups,omitempty"`
	LatestWaitMinutes int    `json:"latest_wait_minutes,omitempty"`
	Confidence        string `json:"confidence"`
	Source            string `json:"source"`
	Message           string `json:"message,omitempty"`
}

type QueueDashboardCalledPoint struct {
	StoreID           string `json:"store_id"`
	StoreName         string `json:"store_name"`
	Bucket            string `json:"bucket"`
	CalledNoSlow      int    `json:"called_no_slow,omitempty"`
	CalledNoTypical   int    `json:"called_no_typical,omitempty"`
	CalledNoFast      int    `json:"called_no_fast,omitempty"`
	QueueGroups       int    `json:"queue_groups,omitempty"`
	WaitMinutes       int    `json:"wait_minutes,omitempty"`
	SampleCount       int    `json:"sample_count"`
	DayCount          int    `json:"day_count"`
	LatestAt          string `json:"latest_at,omitempty"`
	LatestCalledNo    int    `json:"latest_called_no,omitempty"`
	LatestQueueGroups int    `json:"latest_queue_groups,omitempty"`
	LatestWaitMinutes int    `json:"latest_wait_minutes,omitempty"`
	// TodayProjectedNo 是「推测未来叫号」：按今天当前叫号速度，从今天最近一个观测点外推到
	// 该 bucket 时刻大概叫到几号。只对「当前时刻之后」的 bucket 填，今天就能看到接下来叫到哪，
	// 不依赖多天历史。前端画橙色虚线。0/省略=无法外推。
	TodayProjectedNo int    `json:"today_projected_no,omitempty"`
	Confidence       string `json:"confidence"`
	Source           string `json:"source"`
}

type QueueDashboardTrendPoint struct {
	Bucket           string `json:"bucket"`
	Label            string `json:"label"`
	TotalQueueGroups int    `json:"total_queue_groups"`
	TotalWaitMinutes int    `json:"total_wait_minutes"`
	OpenStores       int    `json:"open_stores"`
	SampleCount      int    `json:"sample_count"`
	Source           string `json:"source"`
}

type QueueDashboardStoreRow struct {
	StoreID         string   `json:"store_id"`
	StoreName       string   `json:"store_name"`
	City            string   `json:"city,omitempty"`
	Area            string   `json:"area,omitempty"`
	WaitMinutes     int      `json:"wait_minutes"`
	QueueGroups     int      `json:"queue_groups"`
	StoreStatus     string   `json:"store_status"`
	NetTicketStatus string   `json:"net_ticket_status"`
	OnlineOpen      bool     `json:"online_open"`
	CalledNo        int      `json:"called_no,omitempty"`
	LatestAt        string   `json:"latest_at,omitempty"`
	Source          string   `json:"source"`
	BusyScore       float64  `json:"busy_score"`
	TypicalGroups   *float64 `json:"typical_groups,omitempty"`
	SafeGroups      *float64 `json:"safe_groups,omitempty"`
	BaselineSamples int      `json:"baseline_samples,omitempty"`
	Confidence      string   `json:"confidence,omitempty"`
}

type QueueDashboardWeekday struct {
	Weekday         int      `json:"weekday"`
	WeekdayName     string   `json:"weekday_name"`
	DateType        string   `json:"date_type"`
	SampleCount     int      `json:"sample_count"`
	QueueGroupsAvg  *float64 `json:"queue_groups_avg,omitempty"`
	WaitMinutesAvg  *float64 `json:"wait_minutes_avg,omitempty"`
	OnlineOpenRate  *float64 `json:"online_open_rate,omitempty"`
	BusyRate        *float64 `json:"busy_rate,omitempty"`
	PeakBucket      string   `json:"peak_bucket,omitempty"`
	PeakQueueGroups *float64 `json:"peak_queue_groups,omitempty"`
	Confidence      string   `json:"confidence"`
}

type QueueDashboardHeatmapPoint struct {
	Weekday        int      `json:"weekday"`
	WeekdayName    string   `json:"weekday_name"`
	DateType       string   `json:"date_type"`
	Bucket         string   `json:"bucket"`
	SampleCount    int      `json:"sample_count"`
	QueueGroupsAvg *float64 `json:"queue_groups_avg,omitempty"`
	WaitMinutesAvg *float64 `json:"wait_minutes_avg,omitempty"`
	OnlineOpenRate *float64 `json:"online_open_rate,omitempty"`
	BusyRate       *float64 `json:"busy_rate,omitempty"`
	Confidence     string   `json:"confidence"`
}

type QueueDashboardDateType struct {
	DateType       string   `json:"date_type"`
	DateTypeName   string   `json:"date_type_name"`
	SampleCount    int      `json:"sample_count"`
	QueueGroupsAvg *float64 `json:"queue_groups_avg,omitempty"`
	WaitMinutesAvg *float64 `json:"wait_minutes_avg,omitempty"`
	OnlineOpenRate *float64 `json:"online_open_rate,omitempty"`
	BusyRate       *float64 `json:"busy_rate,omitempty"`
	PeakBucket     string   `json:"peak_bucket,omitempty"`
}

type QueueDashboardAdvisor struct {
	State         string                           `json:"state"`
	StoreID       string                           `json:"store_id,omitempty"`
	StoreName     string                           `json:"store_name,omitempty"`
	TargetNo      int                              `json:"target_no,omitempty"`
	Headline      string                           `json:"headline"`
	Copy          string                           `json:"copy"`
	TargetBucket  string                           `json:"target_bucket,omitempty"`
	TargetLabel   string                           `json:"target_label,omitempty"`
	ArrivalBucket string                           `json:"arrival_bucket,omitempty"`
	ArrivalLabel  string                           `json:"arrival_label,omitempty"`
	Confidence    string                           `json:"confidence,omitempty"`
	Source        string                           `json:"source,omitempty"`
	BucketMinutes int                              `json:"bucket_minutes,omitempty"`
	Milestones    []QueueDashboardAdvisorMilestone `json:"milestones,omitempty"`
}

type QueueDashboardAdvisorMilestone struct {
	Label           string `json:"label"`
	Bucket          string `json:"bucket"`
	CalledNoTypical int    `json:"called_no_typical,omitempty"`
	CalledNoSlow    int    `json:"called_no_slow,omitempty"`
	CalledNoFast    int    `json:"called_no_fast,omitempty"`
	QueueGroups     int    `json:"queue_groups,omitempty"`
	WaitMinutes     int    `json:"wait_minutes,omitempty"`
	Confidence      string `json:"confidence,omitempty"`
	Source          string `json:"source,omitempty"`
}

type queueDashboardSnapshot struct {
	storeID         string
	storeName       string
	city            string
	area            string
	collectedAt     time.Time
	waitMinutes     int
	queueGroups     int
	storeStatus     string
	netTicketStatus string
	reservation     string
	onlineOpen      bool
	waitTimeCounter int
	waitTimeCap     int
	calledNo        int
	source          string
}

type queueDashboardCalledSample struct {
	storeID     string
	storeName   string
	collectedAt time.Time
	bucket      string
	day         string
	calledNo    int
	queueGroups int
	waitMinutes int
}

type queueDashboardRollupAcc struct {
	sampleCount       int
	queueGroupsSum    float64
	queueGroupsN      int
	waitMinutesSum    float64
	waitMinutesN      int
	onlineOpenRateSum float64
	busyRateSum       float64
	peakBucket        string
	peakQueueGroups   float64
	peakSet           bool
}

func BuildQueueDashboard(query QueueDashboardQuery, now time.Time) QueueDashboardResponse {
	return BuildQueueDashboardWithContext(context.Background(), query, now)
}

func BuildQueueDashboardWithContext(ctx context.Context, query QueueDashboardQuery, now time.Time) QueueDashboardResponse {
	if now.IsZero() {
		now = time.Now()
	}
	query = normalizeQueueDashboardQuery(query)
	storeFilter := stringSet(query.StoreIDs)
	localBaseline := loadQueueBaselineRecords()
	localObservations := loadQueueObservations()
	baseline, baselineStatus, baselineErr := loadRemoteQueueDashboardBaseline(ctx, query, now)
	storeNames := queueDashboardStoreNames(baseline, localBaseline, localObservations)

	latest := buildQueueDashboardLatestRows(query, baseline, localBaseline, localObservations, storeNames, storeFilter)
	trend := buildQueueDashboardLocalTrend(query, localBaseline, localObservations, storeNames, storeFilter, now)
	trendSource := "local"
	if len(trend) == 0 {
		trend = buildQueueDashboardBaselineTrend(query, baseline.Rollups, storeFilter, now)
		trendSource = "remote_baseline"
	}
	calledSummary, calledCurve := buildQueueDashboardCalledCurve(query, localBaseline, localObservations, storeNames, storeFilter, now)
	remoteCalledSummary, remoteCalledCurve := buildQueueDashboardRemoteCalledCurve(query, baseline, storeNames, storeFilter, latest)
	// 历史叫号曲线优先用线上基准(remote)：它是跨多天、覆盖全天各时段的「几点叫到几号」规律，
	// 远比本机今天刚采的几个点（常只有当前一两小时、sample_count=1）有价值。
	// 旧逻辑只在「本地为空 或 全国汇总」时才用 remote——单店查询时本地哪怕只有 2 个点也会盖掉
	// remote 全天曲线，导致用户看不到历史叫号规律。改为：remote 可用且点数不少于本地时优先 remote。
	if len(remoteCalledCurve) > 0 && len(remoteCalledCurve) >= len(calledCurve) {
		calledSummary, calledCurve = remoteCalledSummary, remoteCalledCurve
	}
	advisor := buildQueueDashboardAdvisor(query, calledSummary, calledCurve, latest, now)
	weekdayProfiles, heatmap, dateTypes := buildQueueDashboardRollupViews(query, baseline.Rollups, storeFilter)
	summary := buildQueueDashboardSummary(query, latest, trend, len(localBaseline)+len(localObservations), baseline)
	scope := queueDashboardScope(query, latest, baselineStatus, trendSource)
	warnings := queueDashboardWarnings(query, baselineStatus, baselineErr, len(localBaseline), len(localObservations), len(trend), len(heatmap), len(calledCurve))
	samplingSummary := QueueTrendSummary{
		ObservationRecords: len(localObservations),
		BaselineSamples:    baseline.Stats.RollupCount,
	}
	return QueueDashboardResponse{
		GeneratedAt:       now.Format(time.RFC3339),
		Filters:           query,
		Scope:             scope,
		Summary:           summary,
		CalledSummary:     calledSummary,
		CalledCurve:       calledCurve,
		Advisor:           advisor,
		Trend:             trend,
		WeekdayProfiles:   weekdayProfiles,
		Heatmap:           heatmap,
		DateTypeSummaries: dateTypes,
		Sampling:          buildQueueSamplingStatus(now, samplingSummary),
		Baseline:          baselineStatus,
		Warnings:          warnings,
	}
}

func loadRemoteQueueDashboardBaseline(ctx context.Context, query QueueDashboardQuery, now time.Time) (QueueBaselineExport, QueueBaselineRemoteStatus, error) {
	return loadRemoteQueueBaselineForStores(ctx, query.StoreIDs, now)
}

func normalizeQueueDashboardQuery(query QueueDashboardQuery) QueueDashboardQuery {
	query.StoreIDs = UniqueNonEmptyStrings(query.StoreIDs)
	query.Scope = strings.ToLower(strings.TrimSpace(query.Scope))
	if query.Scope == "" {
		if len(query.StoreIDs) > 0 {
			query.Scope = "local"
		} else {
			query.Scope = "all"
		}
	}
	switch query.Scope {
	case "all", "local":
	default:
		query.Scope = "all"
	}
	query.DateType = strings.ToLower(strings.TrimSpace(query.DateType))
	switch query.DateType {
	case "weekday", "workday", "weekend", "holiday":
	default:
		query.DateType = "all"
	}
	if query.WindowHours <= 0 {
		query.WindowHours = queueDashboardDefaultWindowHours
	}
	if query.WindowHours > queueDashboardMaxWindowHours {
		query.WindowHours = queueDashboardMaxWindowHours
	}
	if query.TargetNo < 0 {
		query.TargetNo = 0
	}
	switch query.BucketMinutes {
	case 5, 10, 15, 30, 60:
	default:
		query.BucketMinutes = queueDashboardDefaultBucketMins
	}
	return query
}

func loadQueueBaselineRecords() []QueueBaselineRecord {
	return queueBaselineReadCache.load(queueBaselineRecordsPath(), normalizeQueueBaselineRecordForRead)
}

func normalizeQueueBaselineRecordForRead(record *QueueBaselineRecord) {
	if record == nil {
		return
	}
	if record.CollectedAt == "" {
		record.CollectedAt = strings.TrimSpace(record.Timestamp)
	}
	if record.WaitMinutes == 0 && record.Wait > 0 {
		record.WaitMinutes = record.Wait
	}
	if record.UpdatedAt == "" {
		record.UpdatedAt = record.CollectedAt
	}
	if record.SourceEndpoint == "" {
		record.SourceEndpoint = queueSourceEndpointStores
	}
	if record.APIProfileVersion == "" {
		record.APIProfileVersion = queueAPIProfilePublicV1
	}
}

func queueDashboardStoreNames(baseline QueueBaselineExport, localBaseline []QueueBaselineRecord, observations []QueueObservation) map[string]string {
	names := map[string]string{}
	for _, store := range baseline.Stores {
		if store.StoreID > 0 && strings.TrimSpace(store.Name) != "" {
			names[strconv.Itoa(store.StoreID)] = strings.TrimSpace(store.Name)
		}
	}
	for _, latest := range baseline.Latest {
		if latest.StoreID > 0 && strings.TrimSpace(latest.Name) != "" {
			names[strconv.Itoa(latest.StoreID)] = strings.TrimSpace(latest.Name)
		}
	}
	for _, record := range localBaseline {
		storeID := strconv.Itoa(record.StoreID)
		if _, exists := names[storeID]; !exists && strings.TrimSpace(record.Name) != "" {
			names[storeID] = strings.TrimSpace(record.Name)
		}
	}
	for _, observation := range observations {
		storeID := strings.TrimSpace(observation.StoreID)
		if storeID != "" {
			if _, exists := names[storeID]; !exists {
				names[storeID] = storeID
			}
		}
	}
	return names
}

func buildQueueDashboardLatestRows(query QueueDashboardQuery, baseline QueueBaselineExport, localBaseline []QueueBaselineRecord, observations []QueueObservation, names map[string]string, storeFilter map[string]bool) []QueueDashboardStoreRow {
	snapshots := map[string]queueDashboardSnapshot{}
	for _, item := range baseline.Latest {
		storeID := strconv.Itoa(item.StoreID)
		if !queueDashboardStoreAllowed(storeID, storeFilter) {
			continue
		}
		at, _ := parseRFC3339Local(item.CollectedAt)
		snapshots[storeID] = queueDashboardSnapshot{
			storeID:         storeID,
			storeName:       dashboardStoreName(names, storeID, item.Name),
			city:            item.City,
			area:            item.Area,
			collectedAt:     at,
			waitMinutes:     item.WaitMinutes,
			queueGroups:     item.GroupQueuesCount,
			storeStatus:     item.StoreStatus,
			netTicketStatus: item.NetTicketStatus,
			reservation:     item.ReservationStatus,
			onlineOpen:      item.OnlineOpen,
			waitTimeCounter: item.WaitTimeCounter,
			waitTimeCap:     item.WaitTimeCap,
			calledNo:        item.DisplayCalledNo,
			source:          "remote",
		}
	}
	for _, record := range localBaseline {
		storeID := strconv.Itoa(record.StoreID)
		if !queueDashboardStoreAllowed(storeID, storeFilter) {
			continue
		}
		at, ok := parseRFC3339Local(record.CollectedAt)
		if !ok {
			continue
		}
		snapshot := queueDashboardSnapshot{
			storeID:         storeID,
			storeName:       dashboardStoreName(names, storeID, record.Name),
			city:            record.City,
			area:            record.Area,
			collectedAt:     at,
			waitMinutes:     record.WaitMinutes,
			queueGroups:     record.GroupQueuesCount,
			storeStatus:     record.StoreStatus,
			netTicketStatus: record.NetTicketStatus,
			reservation:     record.ReservationStatus,
			onlineOpen:      record.OnlineOpen,
			waitTimeCounter: record.WaitTimeCounter,
			waitTimeCap:     record.WaitTimeCap,
			calledNo:        record.DisplayCalledNo,
			source:          "local_baseline",
		}
		if current, exists := snapshots[storeID]; !exists || snapshot.collectedAt.After(current.collectedAt) {
			snapshots[storeID] = snapshot
		}
	}
	for _, observation := range observations {
		storeID := strings.TrimSpace(observation.StoreID)
		if !queueDashboardStoreAllowed(storeID, storeFilter) {
			continue
		}
		if query.Scope == "all" && len(storeFilter) == 0 {
			continue
		}
		at, ok := parseRFC3339Local(queueObservationCollectedAt(observation))
		if !ok {
			continue
		}
		snapshot := queueDashboardSnapshot{
			storeID:         storeID,
			storeName:       dashboardStoreName(names, storeID, ""),
			collectedAt:     at,
			waitMinutes:     observation.WaitMinutes,
			queueGroups:     observation.GroupQueuesCount,
			storeStatus:     observation.StoreStatus,
			netTicketStatus: observation.NetTicketStatus,
			reservation:     observation.ReservationStatus,
			onlineOpen:      observation.OnlineOpen,
			waitTimeCounter: observation.WaitTimeCounter,
			waitTimeCap:     observation.WaitTimeCap,
			calledNo:        queueObservationCalledNo(observation),
			source:          "local_detail",
		}
		current, exists := snapshots[storeID]
		if !exists || snapshot.collectedAt.After(current.collectedAt) {
			snapshots[storeID] = snapshot
		} else if current.calledNo == 0 && snapshot.calledNo > 0 {
			current.calledNo = snapshot.calledNo
			snapshots[storeID] = current
		}
	}
	rows := make([]QueueDashboardStoreRow, 0, len(snapshots))
	for _, snapshot := range snapshots {
		rows = append(rows, queueDashboardStoreRow(snapshot, nil, nil, 0, ""))
	}
	sortQueueDashboardStoreRows(rows)
	return rows
}

func buildQueueDashboardSummary(query QueueDashboardQuery, rows []QueueDashboardStoreRow, trend []QueueDashboardTrendPoint, localRecords int, baseline QueueBaselineExport) QueueDashboardSummary {
	summary := QueueDashboardSummary{
		StoreCount:       len(rows),
		WindowHours:      query.WindowHours,
		LocalRecords:     localRecords,
		RemoteStores:     baseline.Stats.LatestCount,
		RemoteRollups:    baseline.Stats.RollupCount,
		TotalCalledNo:    0,
		TotalWaitMinutes: 0,
	}
	for _, row := range rows {
		if queueDashboardIsOpen(row.StoreStatus) {
			summary.OpenStores++
		}
		if row.OnlineOpen {
			summary.OnlineOpenStores++
		}
		summary.TotalQueueGroups += row.QueueGroups
		summary.TotalWaitMinutes += row.WaitMinutes
		summary.TotalCalledNo += row.CalledNo
		if row.LatestAt > summary.LatestAt {
			summary.LatestAt = row.LatestAt
		}
	}
	if len(trend) >= 2 {
		summary.TrendDelta = trend[len(trend)-1].TotalQueueGroups - trend[0].TotalQueueGroups
	}
	return summary
}

func buildQueueDashboardCalledCurve(query QueueDashboardQuery, baselineRecords []QueueBaselineRecord, observations []QueueObservation, names map[string]string, storeFilter map[string]bool, now time.Time) (QueueDashboardCalledSummary, []QueueDashboardCalledPoint) {
	summary := QueueDashboardCalledSummary{
		DateType:      query.DateType,
		DateTypeName:  queueDashboardCalledDateTypeName(query.DateType),
		BucketMinutes: query.BucketMinutes,
		Start:         queueDashboardDayStart,
		End:           queueDashboardDayEnd,
		Confidence:    "none",
		Source:        "local",
		Message:       "这条曲线优先使用本机选定门店的叫号明细；本机没有样本时会回退线上叫号基准。",
	}
	holidays, workdays, _ := loadQueueHolidayDates()
	samples := make([]queueDashboardCalledSample, 0, len(baselineRecords)+len(observations))
	for _, record := range baselineRecords {
		storeID := strconv.Itoa(record.StoreID)
		if !queueDashboardStoreAllowed(storeID, storeFilter) || record.DisplayCalledNo <= 0 {
			continue
		}
		at, ok := parseRFC3339Local(record.CollectedAt)
		if !ok || (!now.IsZero() && at.After(now.Add(time.Minute))) {
			continue
		}
		if !queueDashboardCalledTimeInRange(at) {
			continue
		}
		dateType := queueTrendDateType(at, holidays, workdays)
		if !queueDashboardRollupDateTypeAllowed(dateType, query.DateType, false) {
			continue
		}
		bucket := queueDashboardTimeBucket(at, query.BucketMinutes).Format("15:04")
		samples = append(samples, queueDashboardCalledSample{
			storeID:     storeID,
			storeName:   dashboardStoreName(names, storeID, record.Name),
			collectedAt: at,
			bucket:      bucket,
			day:         at.Format("2006-01-02"),
			calledNo:    record.DisplayCalledNo,
			queueGroups: record.GroupQueuesCount,
			waitMinutes: record.WaitMinutes,
		})
	}
	for _, observation := range observations {
		storeID := strings.TrimSpace(observation.StoreID)
		if !queueDashboardStoreAllowed(storeID, storeFilter) {
			continue
		}
		calledNo := queueObservationCalledNo(observation)
		if calledNo <= 0 {
			continue
		}
		at, ok := parseRFC3339Local(queueObservationCollectedAt(observation))
		if !ok || (!now.IsZero() && at.After(now.Add(time.Minute))) {
			continue
		}
		if !queueDashboardCalledTimeInRange(at) {
			continue
		}
		dateType := queueTrendDateType(at, holidays, workdays)
		if !queueDashboardRollupDateTypeAllowed(dateType, query.DateType, false) {
			continue
		}
		bucket := queueDashboardTimeBucket(at, query.BucketMinutes).Format("15:04")
		samples = append(samples, queueDashboardCalledSample{
			storeID:     storeID,
			storeName:   dashboardStoreName(names, storeID, ""),
			collectedAt: at,
			bucket:      bucket,
			day:         at.Format("2006-01-02"),
			calledNo:    calledNo,
			queueGroups: observation.GroupQueuesCount,
			waitMinutes: observation.WaitMinutes,
		})
	}
	if len(samples) == 0 {
		return summary, nil
	}
	storeID := queueDashboardCalledStoreID(query, samples)
	if storeID == "" {
		return summary, nil
	}
	byDayBucket := map[string]queueDashboardCalledSample{}
	for _, sample := range samples {
		if sample.storeID != storeID {
			continue
		}
		key := sample.day + "|" + sample.bucket
		current, exists := byDayBucket[key]
		if !exists || sample.collectedAt.After(current.collectedAt) {
			byDayBucket[key] = sample
		}
	}
	if len(byDayBucket) == 0 {
		return summary, nil
	}
	byBucket := map[string][]queueDashboardCalledSample{}
	totalDays := map[string]bool{}
	var latest queueDashboardCalledSample
	for _, sample := range byDayBucket {
		byBucket[sample.bucket] = append(byBucket[sample.bucket], sample)
		totalDays[sample.day] = true
		if latest.collectedAt.IsZero() || sample.collectedAt.After(latest.collectedAt) {
			latest = sample
		}
	}
	buckets := make([]string, 0, len(byBucket))
	for bucket := range byBucket {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)
	points := make([]QueueDashboardCalledPoint, 0, len(buckets))
	for _, bucket := range buckets {
		bucketSamples := byBucket[bucket]
		calledValues := make([]float64, 0, len(bucketSamples))
		queueValues := make([]float64, 0, len(bucketSamples))
		waitValues := make([]float64, 0, len(bucketSamples))
		days := map[string]bool{}
		var latestInBucket queueDashboardCalledSample
		for _, sample := range bucketSamples {
			calledValues = append(calledValues, float64(sample.calledNo))
			if sample.queueGroups > 0 {
				queueValues = append(queueValues, float64(sample.queueGroups))
			}
			if sample.waitMinutes > 0 {
				waitValues = append(waitValues, float64(sample.waitMinutes))
			}
			days[sample.day] = true
			if latestInBucket.collectedAt.IsZero() || sample.collectedAt.After(latestInBucket.collectedAt) {
				latestInBucket = sample
			}
		}
		points = append(points, QueueDashboardCalledPoint{
			StoreID:           storeID,
			StoreName:         dashboardStoreName(names, storeID, latest.storeName),
			Bucket:            bucket,
			CalledNoSlow:      queueDashboardRoundedQuantile(calledValues, 0.20),
			CalledNoTypical:   queueDashboardRoundedQuantile(calledValues, 0.50),
			CalledNoFast:      queueDashboardRoundedQuantile(calledValues, 0.80),
			QueueGroups:       queueDashboardRoundedQuantile(queueValues, 0.50),
			WaitMinutes:       queueDashboardRoundedQuantile(waitValues, 0.50),
			SampleCount:       len(bucketSamples),
			DayCount:          len(days),
			LatestAt:          latestInBucket.collectedAt.Format(time.RFC3339),
			LatestCalledNo:    latestInBucket.calledNo,
			LatestQueueGroups: latestInBucket.queueGroups,
			LatestWaitMinutes: latestInBucket.waitMinutes,
			Confidence:        queueDashboardConfidence(len(bucketSamples)),
			Source:            "local",
		})
	}
	// 【推测未来叫号】对「当前时刻之后」的 bucket，按今天叫号速度从今天最近一个观测点外推。
	// 不依赖多天历史——只要有今天的叫号速度就能算，让用户今天就能看到「接下来几点叫到几号」。
	points = projectTodayCalledNo(points, observations, storeID, latest, query.BucketMinutes, now)
	summary.StoreID = storeID
	summary.StoreName = dashboardStoreName(names, storeID, latest.storeName)
	summary.SampleCount = len(byDayBucket)
	summary.DayCount = len(totalDays)
	summary.PointCount = len(points)
	summary.LatestAt = latest.collectedAt.Format(time.RFC3339)
	summary.LatestBucket = latest.bucket
	summary.LatestCalledNo = latest.calledNo
	summary.LatestQueueGroups = latest.queueGroups
	summary.LatestWaitMinutes = latest.waitMinutes
	summary.Confidence = queueDashboardConfidence(summary.SampleCount)
	summary.Message = "按本机选定门店采集记录聚合：同一天同一时间桶只取最后一条，避免高频采样把某个时间点放大。"
	return summary, points
}

// projectTodayCalledNo 给 called_curve 的「当前时刻之后」bucket 填 TodayProjectedNo（推测未来叫号），
// 并在已有 points 之后追加未来 bucket 点（now→关店），让外推线能延伸到未来。
// 算法：取今天最新叫号观测点作锚点（latest），用本机近 30 分钟叫号速度 rate 外推——
//
//	projected = 锚点号 + rate × (bucket时刻 - 锚点时刻的分钟数)
//
// 守卫：无锚点/锚点不是今天/rate<=0 不填不追加；外推上限防速度异常吹飞。
// 不依赖多天历史：只要有今天的速度，今天就能看到「接下来几点叫到几号」。
func projectTodayCalledNo(points []QueueDashboardCalledPoint, observations []QueueObservation, storeID string, latest queueDashboardCalledSample, bucketMinutes int, now time.Time) []QueueDashboardCalledPoint {
	if now.IsZero() || latest.calledNo <= 0 {
		return points
	}
	nowS := now.In(SushiroTimezone)
	today := nowS.Format("2006-01-02")
	anchorAt := latest.collectedAt.In(SushiroTimezone)
	if anchorAt.Format("2006-01-02") != today {
		return points // 锚点不是今天，没法基于今天进度外推
	}
	rate, _, _, _, _, ok := calledRatePerMinuteWeighted(recentStoreObservations(observations, storeID, now, queueAdvisorWindow30), now, queueAdvisorWindow30)
	if !ok || rate <= 0 {
		return points
	}
	if bucketMinutes <= 0 {
		bucketMinutes = 10
	}
	storeName := ""
	if len(points) > 0 {
		storeName = points[0].StoreName
	}
	anchorMinute := anchorAt.Hour()*60 + anchorAt.Minute()
	nowMinute := nowS.Hour()*60 + nowS.Minute()
	endMinute := queueDashboardBucketMinute(queueDashboardDayEnd) // 关店 22:00
	capExtra := rate * 180                                        // 软上限：最多外推 3 小时号数，防速度异常吹飞
	// 已有 points 里落在未来的，填外推值。
	for i := range points {
		bucketMinute := queueDashboardBucketMinute(points[i].Bucket)
		if bucketMinute < 0 || bucketMinute <= nowMinute {
			continue
		}
		extra := rate * float64(bucketMinute-anchorMinute)
		if extra < 0 || extra > capExtra {
			if extra > capExtra {
				extra = capExtra
			}
			if extra < 0 {
				continue
			}
		}
		if projected := latest.calledNo + int(extra); projected > latest.calledNo {
			points[i].TodayProjectedNo = projected
		}
	}
	// 追加未来 bucket 点（now 之后到关店，每个 bucketMinutes 一个），只带 TodayProjectedNo。
	for bm := ((nowMinute / bucketMinutes) + 1) * bucketMinutes; bm <= endMinute; bm += bucketMinutes {
		extra := rate * float64(bm-anchorMinute)
		if extra > capExtra {
			extra = capExtra
		}
		projected := latest.calledNo + int(extra)
		if projected <= latest.calledNo {
			continue
		}
		points = append(points, QueueDashboardCalledPoint{
			StoreID:          storeID,
			StoreName:        storeName,
			Bucket:           queueDashboardMinuteBucket(bm),
			TodayProjectedNo: projected,
			Source:           "projected",
		})
	}
	return points
}

func buildQueueDashboardRemoteCalledCurve(query QueueDashboardQuery, baseline QueueBaselineExport, names map[string]string, storeFilter map[string]bool, latestRows []QueueDashboardStoreRow) (QueueDashboardCalledSummary, []QueueDashboardCalledPoint) {
	summary := QueueDashboardCalledSummary{
		DateType:      query.DateType,
		DateTypeName:  queueDashboardCalledDateTypeName(query.DateType),
		BucketMinutes: baseline.BucketMinutes,
		Start:         queueDashboardDayStart,
		End:           queueDashboardDayEnd,
		Confidence:    "none",
		Source:        "remote_baseline",
		Message:       "按线上 Turso 叫号聚合基准展示；跨日期类型时会按样本数做近似加权。",
	}
	if summary.BucketMinutes <= 0 {
		summary.BucketMinutes = query.BucketMinutes
	}
	storeID := queueDashboardRemoteCalledStoreID(query, baseline.Rollups, storeFilter)
	if storeID == "" {
		return summary, nil
	}

	type bucketAcc struct {
		samples        int
		rollups        int
		slowSum        float64
		typicalSum     float64
		fastSum        float64
		queueGroupsSum float64
		queueGroupsN   int
		waitMinutesSum float64
		waitMinutesN   int
		updatedAt      string
	}
	byBucket := map[string]*bucketAcc{}
	for _, rollup := range baseline.Rollups {
		if strconv.Itoa(rollup.StoreID) != storeID {
			continue
		}
		if !queueDashboardRollupDateTypeAllowed(rollup.DateType, query.DateType, false) || !queueDashboardCalledBucketInRange(rollup.TimeBucket) {
			continue
		}
		if rollup.CalledSampleCount <= 0 || rollup.CalledNoTypical == nil {
			continue
		}
		acc := byBucket[rollup.TimeBucket]
		if acc == nil {
			acc = &bucketAcc{}
			byBucket[rollup.TimeBucket] = acc
		}
		samples := maxInt(rollup.CalledSampleCount, 1)
		typical := *rollup.CalledNoTypical
		slow := typical
		if rollup.CalledNoSlow != nil {
			slow = *rollup.CalledNoSlow
		}
		fast := typical
		if rollup.CalledNoFast != nil {
			fast = *rollup.CalledNoFast
		}
		acc.samples += samples
		acc.rollups++
		acc.slowSum += slow * float64(samples)
		acc.typicalSum += typical * float64(samples)
		acc.fastSum += fast * float64(samples)
		if rollup.QueueGroupsTypical != nil {
			acc.queueGroupsSum += *rollup.QueueGroupsTypical * float64(samples)
			acc.queueGroupsN += samples
		}
		if rollup.WaitTypicalMinutes != nil {
			acc.waitMinutesSum += *rollup.WaitTypicalMinutes * float64(samples)
			acc.waitMinutesN += samples
		}
		if rollup.UpdatedAt > acc.updatedAt {
			acc.updatedAt = rollup.UpdatedAt
		}
	}
	if len(byBucket) == 0 {
		return summary, nil
	}

	latestRow, hasLatest := queueDashboardLatestRowForStore(latestRows, storeID)
	buckets := make([]string, 0, len(byBucket))
	for bucket := range byBucket {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)
	points := make([]QueueDashboardCalledPoint, 0, len(buckets))
	for _, bucket := range buckets {
		acc := byBucket[bucket]
		point := QueueDashboardCalledPoint{
			StoreID:         storeID,
			StoreName:       dashboardStoreName(names, storeID, ""),
			Bucket:          bucket,
			CalledNoSlow:    int(math.Round(acc.slowSum / float64(acc.samples))),
			CalledNoTypical: int(math.Round(acc.typicalSum / float64(acc.samples))),
			CalledNoFast:    int(math.Round(acc.fastSum / float64(acc.samples))),
			SampleCount:     acc.samples,
			DayCount:        acc.rollups,
			LatestAt:        acc.updatedAt,
			Confidence:      queueDashboardConfidence(acc.samples),
			Source:          "remote_baseline",
		}
		if acc.queueGroupsN > 0 {
			point.QueueGroups = int(math.Round(acc.queueGroupsSum / float64(acc.queueGroupsN)))
		}
		if acc.waitMinutesN > 0 {
			point.WaitMinutes = int(math.Round(acc.waitMinutesSum / float64(acc.waitMinutesN)))
		}
		if hasLatest {
			point.LatestAt = latestRow.LatestAt
			point.LatestCalledNo = latestRow.CalledNo
			point.LatestQueueGroups = latestRow.QueueGroups
			point.LatestWaitMinutes = latestRow.WaitMinutes
		}
		points = append(points, point)
		summary.SampleCount += acc.samples
		summary.DayCount += acc.rollups
		if point.LatestAt > summary.LatestAt {
			summary.LatestAt = point.LatestAt
		}
	}
	if hasLatest {
		summary.LatestAt = latestRow.LatestAt
		summary.LatestCalledNo = latestRow.CalledNo
		summary.LatestQueueGroups = latestRow.QueueGroups
		summary.LatestWaitMinutes = latestRow.WaitMinutes
		if at, ok := parseRFC3339Local(latestRow.LatestAt); ok && queueDashboardCalledTimeInRange(at) {
			summary.LatestBucket = queueDashboardTimeBucket(at, summary.BucketMinutes).Format("15:04")
		}
	} else {
		summary.LatestAt = baseline.Stats.SourceUpdatedAt
	}
	summary.StoreID = storeID
	summary.StoreName = dashboardStoreName(names, storeID, "")
	summary.PointCount = len(points)
	summary.Confidence = queueDashboardConfidence(summary.SampleCount)
	return summary, points
}

func buildQueueDashboardAdvisor(query QueueDashboardQuery, summary QueueDashboardCalledSummary, curve []QueueDashboardCalledPoint, latestRows []QueueDashboardStoreRow, now time.Time) QueueDashboardAdvisor {
	advisor := QueueDashboardAdvisor{
		State:         "empty",
		StoreID:       summary.StoreID,
		StoreName:     summary.StoreName,
		TargetNo:      query.TargetNo,
		Headline:      "暂无可用叫号判断",
		Copy:          "先开启信息收集，或选择一个有线上基准的门店。",
		Confidence:    summary.Confidence,
		Source:        summary.Source,
		BucketMinutes: query.BucketMinutes,
	}
	points := queueDashboardAdvisorPoints(curve)
	if len(points) == 0 {
		return advisor
	}
	if advisor.StoreID == "" {
		advisor.StoreID = points[0].StoreID
	}
	if advisor.StoreName == "" {
		advisor.StoreName = points[0].StoreName
	}
	advisor.Confidence = queueDashboardMergeConfidence(summary.Confidence, points)
	advisor.Source = points[0].Source

	if query.TargetNo > 0 {
		return buildQueueDashboardTargetAdvisor(advisor, query.TargetNo, summary, points)
	}
	return buildQueueDashboardMilestoneAdvisor(advisor, points, now)
}

func buildQueueDashboardTargetAdvisor(base QueueDashboardAdvisor, targetNo int, summary QueueDashboardCalledSummary, points []QueueDashboardCalledPoint) QueueDashboardAdvisor {
	base.TargetNo = targetNo
	if summary.LatestCalledNo >= targetNo && summary.LatestCalledNo > 0 {
		base.State = "passed"
		base.TargetBucket = summary.LatestBucket
		base.TargetLabel = queueDashboardBucketText(summary.LatestBucket)
		base.Headline = "当前已经叫到 " + strconv.Itoa(summary.LatestCalledNo) + " 号"
		base.Copy = strconv.Itoa(targetNo) + " 号可能已经过号；请用手机小程序确认现场状态。"
		base.ArrivalLabel = "尽快到店或重新取号"
		return base
	}
	for _, point := range points {
		if point.CalledNoTypical <= 0 || point.CalledNoTypical < targetNo {
			continue
		}
		arrivalBucket := queueDashboardArrivalBucket(point.Bucket)
		base.State = "target"
		base.TargetBucket = point.Bucket
		base.TargetLabel = queueDashboardBucketText(point.Bucket)
		base.ArrivalBucket = arrivalBucket
		base.ArrivalLabel = queueDashboardBucketText(arrivalBucket) + " 前到店"
		base.Confidence = point.Confidence
		base.Source = point.Source
		base.Headline = "预计 " + queueDashboardBucketText(point.Bucket) + " 左右叫到 " + strconv.Itoa(targetNo) + " 号"
		base.Copy = "建议 " + queueDashboardBucketText(arrivalBucket) + " 前到店；这个点一般叫到 " + strconv.Itoa(point.CalledNoTypical) + " 号。"
		base.Milestones = []QueueDashboardAdvisorMilestone{queueDashboardAdvisorMilestone("命中时间点", point)}
		return base
	}
	last := points[len(points)-1]
	base.State = "uncovered"
	base.TargetBucket = last.Bucket
	base.TargetLabel = queueDashboardBucketText(last.Bucket)
	base.Confidence = last.Confidence
	base.Source = last.Source
	base.Headline = "样本还没覆盖到 " + strconv.Itoa(targetNo) + " 号"
	base.Copy = "目前曲线到 " + queueDashboardBucketText(last.Bucket) + " 一般叫到 " + strconv.Itoa(last.CalledNoTypical) + " 号；继续收集后会更准。"
	base.Milestones = []QueueDashboardAdvisorMilestone{queueDashboardAdvisorMilestone("样本末尾", last)}
	return base
}

func buildQueueDashboardMilestoneAdvisor(base QueueDashboardAdvisor, points []QueueDashboardCalledPoint, now time.Time) QueueDashboardAdvisor {
	base.State = "milestones"
	base.Headline = "这家店几点大概叫到多少号"
	base.Copy = "下面是按同类日期聚合出来的关键时间点；输入你手里的号可以直接估算叫到时间。"
	base.Milestones = queueDashboardAdvisorMilestones(points, now)
	if len(base.Milestones) > 0 {
		first := base.Milestones[0]
		base.TargetBucket = first.Bucket
		base.TargetLabel = queueDashboardBucketText(first.Bucket)
	}
	return base
}

func queueDashboardAdvisorPoints(curve []QueueDashboardCalledPoint) []QueueDashboardCalledPoint {
	points := make([]QueueDashboardCalledPoint, 0, len(curve))
	for _, point := range curve {
		if point.CalledNoTypical <= 0 || !queueDashboardCalledBucketInRange(point.Bucket) {
			continue
		}
		points = append(points, point)
	}
	sort.SliceStable(points, func(i, j int) bool {
		return queueDashboardBucketMinute(points[i].Bucket) < queueDashboardBucketMinute(points[j].Bucket)
	})
	return points
}

func queueDashboardAdvisorMilestones(points []QueueDashboardCalledPoint, now time.Time) []QueueDashboardAdvisorMilestone {
	type target struct {
		label  string
		minute int
	}
	targets := []target{}
	nowMinute := now.Hour()*60 + now.Minute()
	if now.IsZero() || nowMinute < queueDashboardBucketMinute(queueDashboardDayStart) || nowMinute > queueDashboardBucketMinute(queueDashboardDayEnd) {
		targets = append(targets, target{label: "开店后", minute: queueDashboardBucketMinute(queueDashboardDayStart)})
	} else {
		targets = append(targets, target{label: "现在附近", minute: nowMinute})
		targets = append(targets, target{label: "1小时后", minute: nowMinute + 60})
		targets = append(targets, target{label: "2小时后", minute: nowMinute + 120})
	}
	targets = append(targets,
		target{label: "晚高峰", minute: 18*60 + 30},
		target{label: "收尾前", minute: 21*60 + 30},
	)
	out := []QueueDashboardAdvisorMilestone{}
	seen := map[string]bool{}
	for _, target := range targets {
		point, ok := queueDashboardPointAtOrAfter(points, target.minute)
		if !ok || seen[target.label+"|"+point.Bucket] {
			continue
		}
		seen[target.label+"|"+point.Bucket] = true
		out = append(out, queueDashboardAdvisorMilestone(target.label, point))
	}
	if len(out) == 0 && len(points) > 0 {
		out = append(out, queueDashboardAdvisorMilestone("样本末尾", points[len(points)-1]))
	}
	return out
}

func queueDashboardAdvisorMilestone(label string, point QueueDashboardCalledPoint) QueueDashboardAdvisorMilestone {
	return QueueDashboardAdvisorMilestone{
		Label:           label,
		Bucket:          point.Bucket,
		CalledNoTypical: point.CalledNoTypical,
		CalledNoSlow:    point.CalledNoSlow,
		CalledNoFast:    point.CalledNoFast,
		QueueGroups:     point.QueueGroups,
		WaitMinutes:     point.WaitMinutes,
		Confidence:      point.Confidence,
		Source:          point.Source,
	}
}

func queueDashboardPointAtOrAfter(points []QueueDashboardCalledPoint, minute int) (QueueDashboardCalledPoint, bool) {
	if len(points) == 0 {
		return QueueDashboardCalledPoint{}, false
	}
	dayStart := queueDashboardBucketMinute(queueDashboardDayStart)
	dayEnd := queueDashboardBucketMinute(queueDashboardDayEnd)
	if minute < dayStart {
		minute = dayStart
	}
	if minute > dayEnd {
		minute = dayEnd
	}
	for _, point := range points {
		if queueDashboardBucketMinute(point.Bucket) >= minute {
			return point, true
		}
	}
	return points[len(points)-1], true
}

func queueDashboardArrivalBucket(bucket string) string {
	minute := queueDashboardBucketMinute(bucket)
	if minute < 0 {
		return ""
	}
	minute -= queueDashboardArrivalLeadMins
	if minute < queueDashboardBucketMinute(queueDashboardDayStart) {
		minute = queueDashboardBucketMinute(queueDashboardDayStart)
	}
	return queueDashboardMinuteBucket(minute)
}

func queueDashboardBucketMinute(bucket string) int {
	seconds := ParseTimeSeconds(compactTrendTime(bucket))
	if seconds < 0 {
		return -1
	}
	return seconds / 60
}

func queueDashboardMinuteBucket(minute int) string {
	if minute < 0 {
		return ""
	}
	hour := minute / 60
	min := minute % 60
	return strconv.Itoa(hour/10) + strconv.Itoa(hour%10) + ":" + strconv.Itoa(min/10) + strconv.Itoa(min%10)
}

func queueDashboardBucketText(bucket string) string {
	if strings.TrimSpace(bucket) == "" {
		return "现在"
	}
	return bucket
}

func queueDashboardMergeConfidence(summaryConfidence string, points []QueueDashboardCalledPoint) string {
	for _, confidence := range []string{"high", "medium", "low"} {
		if summaryConfidence == confidence {
			return confidence
		}
		for _, point := range points {
			if point.Confidence == confidence {
				return confidence
			}
		}
	}
	return summaryConfidence
}

func queueDashboardRemoteCalledStoreID(query QueueDashboardQuery, rollups []QueueBaselineRollup, storeFilter map[string]bool) string {
	type stat struct {
		samples   int
		updatedAt string
	}
	stats := map[string]stat{}
	for _, rollup := range rollups {
		storeID := strconv.Itoa(rollup.StoreID)
		if !queueDashboardStoreAllowed(storeID, storeFilter) || !queueDashboardRollupDateTypeAllowed(rollup.DateType, query.DateType, false) {
			continue
		}
		if !queueDashboardCalledBucketInRange(rollup.TimeBucket) || rollup.CalledSampleCount <= 0 || rollup.CalledNoTypical == nil {
			continue
		}
		current := stats[storeID]
		current.samples += rollup.CalledSampleCount
		if rollup.UpdatedAt > current.updatedAt {
			current.updatedAt = rollup.UpdatedAt
		}
		stats[storeID] = current
	}
	for _, storeID := range query.StoreIDs {
		storeID = strings.TrimSpace(storeID)
		if storeID != "" && stats[storeID].samples > 0 {
			return storeID
		}
	}
	best := ""
	for storeID, stat := range stats {
		if best == "" {
			best = storeID
			continue
		}
		current := stats[best]
		if stat.samples > current.samples ||
			stat.samples == current.samples && (stat.updatedAt > current.updatedAt || stat.updatedAt == current.updatedAt && storeID < best) {
			best = storeID
		}
	}
	return best
}

func queueDashboardLatestRowForStore(rows []QueueDashboardStoreRow, storeID string) (QueueDashboardStoreRow, bool) {
	for _, row := range rows {
		if row.StoreID == storeID {
			return row, true
		}
	}
	return QueueDashboardStoreRow{}, false
}

func queueDashboardCalledStoreID(query QueueDashboardQuery, samples []queueDashboardCalledSample) string {
	stats := map[string]struct {
		count  int
		latest time.Time
	}{}
	for _, sample := range samples {
		stat := stats[sample.storeID]
		stat.count++
		if sample.collectedAt.After(stat.latest) {
			stat.latest = sample.collectedAt
		}
		stats[sample.storeID] = stat
	}
	for _, storeID := range query.StoreIDs {
		storeID = strings.TrimSpace(storeID)
		if storeID != "" && stats[storeID].count > 0 {
			return storeID
		}
	}
	best := ""
	for storeID, stat := range stats {
		if best == "" {
			best = storeID
			continue
		}
		current := stats[best]
		if stat.count > current.count ||
			stat.count == current.count && (stat.latest.After(current.latest) || stat.latest.Equal(current.latest) && storeID < best) {
			best = storeID
		}
	}
	return best
}

func queueDashboardCalledTimeInRange(at time.Time) bool {
	seconds := at.Hour()*3600 + at.Minute()*60 + at.Second()
	start := ParseTimeSeconds(compactTrendTime(queueDashboardDayStart))
	end := ParseTimeSeconds(compactTrendTime(queueDashboardDayEnd))
	return start >= 0 && end >= 0 && seconds >= start && seconds <= end
}

func queueDashboardCalledBucketInRange(bucket string) bool {
	seconds := ParseTimeSeconds(compactTrendTime(bucket))
	start := ParseTimeSeconds(compactTrendTime(queueDashboardDayStart))
	end := ParseTimeSeconds(compactTrendTime(queueDashboardDayEnd))
	return seconds >= 0 && start >= 0 && end >= 0 && seconds >= start && seconds <= end
}

func queueDashboardRoundedQuantile(values []float64, q float64) int {
	value := queueQuantile(values, q)
	if math.IsNaN(value) {
		return 0
	}
	return int(math.Round(value))
}

func queueDashboardCalledDateTypeName(dateType string) string {
	if dateType == "" || dateType == "all" {
		return "剔除节假日"
	}
	return queueTrendDateTypeName(dateType)
}

func buildQueueDashboardLocalTrend(query QueueDashboardQuery, baselineRecords []QueueBaselineRecord, observations []QueueObservation, names map[string]string, storeFilter map[string]bool, now time.Time) []QueueDashboardTrendPoint {
	cutoff := now.Add(-time.Duration(query.WindowHours) * time.Hour)
	byStoreBucket := map[string]queueDashboardSnapshot{}
	addSnapshot := func(snapshot queueDashboardSnapshot) {
		if snapshot.storeID == "" || snapshot.collectedAt.IsZero() || snapshot.collectedAt.Before(cutoff) || snapshot.collectedAt.After(now) {
			return
		}
		if !queueDashboardStoreAllowed(snapshot.storeID, storeFilter) {
			return
		}
		bucket := queueDashboardTimeBucket(snapshot.collectedAt, query.BucketMinutes)
		key := snapshot.storeID + "|" + bucket.Format(time.RFC3339)
		current, exists := byStoreBucket[key]
		if !exists || snapshot.collectedAt.After(current.collectedAt) {
			snapshot.collectedAt = bucket
			byStoreBucket[key] = snapshot
		}
	}
	for _, record := range baselineRecords {
		at, ok := parseRFC3339Local(record.CollectedAt)
		if !ok {
			continue
		}
		addSnapshot(queueDashboardSnapshot{
			storeID:         strconv.Itoa(record.StoreID),
			storeName:       dashboardStoreName(names, strconv.Itoa(record.StoreID), record.Name),
			collectedAt:     at,
			waitMinutes:     record.WaitMinutes,
			queueGroups:     record.GroupQueuesCount,
			storeStatus:     record.StoreStatus,
			netTicketStatus: record.NetTicketStatus,
			onlineOpen:      record.OnlineOpen,
			source:          "local_baseline",
		})
	}
	for _, observation := range observations {
		if query.Scope == "all" && len(storeFilter) == 0 {
			continue
		}
		at, ok := parseRFC3339Local(queueObservationCollectedAt(observation))
		if !ok {
			continue
		}
		addSnapshot(queueDashboardSnapshot{
			storeID:         strings.TrimSpace(observation.StoreID),
			storeName:       dashboardStoreName(names, strings.TrimSpace(observation.StoreID), ""),
			collectedAt:     at,
			waitMinutes:     observation.WaitMinutes,
			queueGroups:     observation.GroupQueuesCount,
			storeStatus:     observation.StoreStatus,
			netTicketStatus: observation.NetTicketStatus,
			onlineOpen:      observation.OnlineOpen,
			calledNo:        queueObservationCalledNo(observation),
			source:          "local_detail",
		})
	}
	return queueDashboardTrendFromSnapshots(byStoreBucket, "local")
}

func queueDashboardTrendFromSnapshots(byStoreBucket map[string]queueDashboardSnapshot, source string) []QueueDashboardTrendPoint {
	type acc struct {
		queueGroups int
		waitMinutes int
		openStores  int
		samples     int
	}
	byBucket := map[string]*acc{}
	bucketTimes := map[string]time.Time{}
	for _, snapshot := range byStoreBucket {
		key := snapshot.collectedAt.Format(time.RFC3339)
		a := byBucket[key]
		if a == nil {
			a = &acc{}
			byBucket[key] = a
			bucketTimes[key] = snapshot.collectedAt
		}
		a.queueGroups += snapshot.queueGroups
		a.waitMinutes += snapshot.waitMinutes
		if queueDashboardIsOpen(snapshot.storeStatus) {
			a.openStores++
		}
		a.samples++
	}
	keys := make([]string, 0, len(byBucket))
	for key := range byBucket {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return bucketTimes[keys[i]].Before(bucketTimes[keys[j]]) })
	out := make([]QueueDashboardTrendPoint, 0, len(keys))
	for _, key := range keys {
		at := bucketTimes[key]
		a := byBucket[key]
		out = append(out, QueueDashboardTrendPoint{
			Bucket:           at.Format(time.RFC3339),
			Label:            at.Format("15:04"),
			TotalQueueGroups: a.queueGroups,
			TotalWaitMinutes: a.waitMinutes,
			OpenStores:       a.openStores,
			SampleCount:      a.samples,
			Source:           source,
		})
	}
	return out
}

func buildQueueDashboardBaselineTrend(query QueueDashboardQuery, rollups []QueueBaselineRollup, storeFilter map[string]bool, now time.Time) []QueueDashboardTrendPoint {
	type acc struct {
		queueGroups float64
		waitMinutes float64
		openStores  float64
		samples     int
	}
	byBucket := map[string]*acc{}
	for _, rollup := range rollups {
		storeID := strconv.Itoa(rollup.StoreID)
		if !queueDashboardStoreAllowed(storeID, storeFilter) || !queueDashboardRollupDateTypeAllowed(rollup.DateType, query.DateType, false) {
			continue
		}
		if !queueDashboardBucketInWindow(rollup.TimeBucket, now, query.WindowHours) {
			continue
		}
		a := byBucket[rollup.TimeBucket]
		if a == nil {
			a = &acc{}
			byBucket[rollup.TimeBucket] = a
		}
		samples := maxInt(rollup.SampleCount, 1)
		if rollup.QueueGroupsTypical != nil {
			a.queueGroups += *rollup.QueueGroupsTypical
		}
		if rollup.WaitTypicalMinutes != nil {
			a.waitMinutes += *rollup.WaitTypicalMinutes
		}
		a.openStores += rollup.OpenRate
		a.samples += samples
	}
	buckets := make([]string, 0, len(byBucket))
	for bucket := range byBucket {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)
	out := make([]QueueDashboardTrendPoint, 0, len(buckets))
	for _, bucket := range buckets {
		a := byBucket[bucket]
		out = append(out, QueueDashboardTrendPoint{
			Bucket:           bucket,
			Label:            bucket,
			TotalQueueGroups: int(math.Round(a.queueGroups)),
			TotalWaitMinutes: int(math.Round(a.waitMinutes)),
			OpenStores:       int(math.Round(a.openStores)),
			SampleCount:      a.samples,
			Source:           "remote_baseline",
		})
	}
	return out
}

func buildQueueDashboardRollupViews(query QueueDashboardQuery, rollups []QueueBaselineRollup, storeFilter map[string]bool) ([]QueueDashboardWeekday, []QueueDashboardHeatmapPoint, []QueueDashboardDateType) {
	weekdayAcc := map[int]*queueDashboardRollupAcc{}
	heatmapAcc := map[string]*queueDashboardRollupAcc{}
	dateTypeAcc := map[string]*queueDashboardRollupAcc{}
	for _, rollup := range rollups {
		storeID := strconv.Itoa(rollup.StoreID)
		if !queueDashboardStoreAllowed(storeID, storeFilter) {
			continue
		}
		addRollupToAcc(dateTypeAcc, strings.ToLower(strings.TrimSpace(rollup.DateType)), rollup)
		if !queueDashboardRollupDateTypeAllowed(rollup.DateType, query.DateType, false) {
			continue
		}
		addRollupToWeekdayAcc(weekdayAcc, rollup.Weekday, rollup)
		heatKey := strconv.Itoa(rollup.Weekday) + "|" + rollup.TimeBucket
		addRollupToAcc(heatmapAcc, heatKey, rollup)
	}
	weekdays := make([]QueueDashboardWeekday, 0, len(weekdayAcc))
	for weekday, acc := range weekdayAcc {
		point := QueueDashboardWeekday{
			Weekday:         weekday,
			WeekdayName:     queueDashboardWeekdayName(weekday),
			DateType:        query.DateType,
			SampleCount:     acc.sampleCount,
			QueueGroupsAvg:  acc.queueGroupsAvg(),
			WaitMinutesAvg:  acc.waitMinutesAvg(),
			OnlineOpenRate:  acc.rateAvg(acc.onlineOpenRateSum),
			BusyRate:        acc.rateAvg(acc.busyRateSum),
			PeakBucket:      acc.peakBucket,
			PeakQueueGroups: acc.peakQueueGroupsPtr(),
			Confidence:      queueDashboardConfidence(acc.sampleCount),
		}
		weekdays = append(weekdays, point)
	}
	sort.Slice(weekdays, func(i, j int) bool { return weekdays[i].Weekday < weekdays[j].Weekday })

	heatmap := make([]QueueDashboardHeatmapPoint, 0, len(heatmapAcc))
	for key, acc := range heatmapAcc {
		parts := strings.Split(key, "|")
		if len(parts) != 2 {
			continue
		}
		weekday, _ := strconv.Atoi(parts[0])
		heatmap = append(heatmap, QueueDashboardHeatmapPoint{
			Weekday:        weekday,
			WeekdayName:    queueDashboardWeekdayName(weekday),
			DateType:       query.DateType,
			Bucket:         parts[1],
			SampleCount:    acc.sampleCount,
			QueueGroupsAvg: acc.queueGroupsAvg(),
			WaitMinutesAvg: acc.waitMinutesAvg(),
			OnlineOpenRate: acc.rateAvg(acc.onlineOpenRateSum),
			BusyRate:       acc.rateAvg(acc.busyRateSum),
			Confidence:     queueDashboardConfidence(acc.sampleCount),
		})
	}
	sort.Slice(heatmap, func(i, j int) bool {
		if heatmap[i].Weekday != heatmap[j].Weekday {
			return heatmap[i].Weekday < heatmap[j].Weekday
		}
		return heatmap[i].Bucket < heatmap[j].Bucket
	})

	dateTypes := make([]QueueDashboardDateType, 0, len(dateTypeAcc))
	for dateType, acc := range dateTypeAcc {
		if dateType == "" {
			continue
		}
		dateTypes = append(dateTypes, QueueDashboardDateType{
			DateType:       dateType,
			DateTypeName:   queueTrendDateTypeName(dateType),
			SampleCount:    acc.sampleCount,
			QueueGroupsAvg: acc.queueGroupsAvg(),
			WaitMinutesAvg: acc.waitMinutesAvg(),
			OnlineOpenRate: acc.rateAvg(acc.onlineOpenRateSum),
			BusyRate:       acc.rateAvg(acc.busyRateSum),
			PeakBucket:     acc.peakBucket,
		})
	}
	sort.Slice(dateTypes, func(i, j int) bool {
		return queueTrendDateTypeRank(dateTypes[i].DateType) < queueTrendDateTypeRank(dateTypes[j].DateType)
	})
	return weekdays, heatmap, dateTypes
}

func addRollupToWeekdayAcc(target map[int]*queueDashboardRollupAcc, weekday int, rollup QueueBaselineRollup) {
	if weekday <= 0 {
		return
	}
	key := weekday
	acc := target[key]
	if acc == nil {
		acc = &queueDashboardRollupAcc{}
		target[key] = acc
	}
	acc.add(rollup)
}

func addRollupToAcc(target map[string]*queueDashboardRollupAcc, key string, rollup QueueBaselineRollup) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	acc := target[key]
	if acc == nil {
		acc = &queueDashboardRollupAcc{}
		target[key] = acc
	}
	acc.add(rollup)
}

func (a *queueDashboardRollupAcc) add(rollup QueueBaselineRollup) {
	samples := maxInt(rollup.SampleCount, 1)
	a.sampleCount += samples
	if rollup.QueueGroupsTypical != nil {
		value := *rollup.QueueGroupsTypical
		a.queueGroupsSum += value * float64(samples)
		a.queueGroupsN += samples
		if !a.peakSet || value > a.peakQueueGroups {
			a.peakSet = true
			a.peakQueueGroups = value
			a.peakBucket = rollup.TimeBucket
		}
	}
	if rollup.WaitTypicalMinutes != nil {
		a.waitMinutesSum += *rollup.WaitTypicalMinutes * float64(samples)
		a.waitMinutesN += samples
	}
	a.onlineOpenRateSum += rollup.OnlineOpenRate * float64(samples)
	a.busyRateSum += rollup.BusyRate * float64(samples)
}

func (a queueDashboardRollupAcc) queueGroupsAvg() *float64 {
	if a.queueGroupsN <= 0 {
		return nil
	}
	return floatPtr(a.queueGroupsSum / float64(a.queueGroupsN))
}

func (a queueDashboardRollupAcc) waitMinutesAvg() *float64 {
	if a.waitMinutesN <= 0 {
		return nil
	}
	return floatPtr(a.waitMinutesSum / float64(a.waitMinutesN))
}

func (a queueDashboardRollupAcc) rateAvg(sum float64) *float64 {
	if a.sampleCount <= 0 {
		return nil
	}
	return floatPtr(sum / float64(a.sampleCount))
}

func (a queueDashboardRollupAcc) peakQueueGroupsPtr() *float64 {
	if !a.peakSet {
		return nil
	}
	return floatPtr(a.peakQueueGroups)
}

func queueDashboardStoreRow(snapshot queueDashboardSnapshot, typical, safe *float64, samples int, confidence string) QueueDashboardStoreRow {
	latestAt := ""
	if !snapshot.collectedAt.IsZero() {
		latestAt = snapshot.collectedAt.Format(time.RFC3339)
	}
	queueGroups := snapshot.queueGroups
	busyScore := float64(queueGroups)
	if queueGroups == 0 && snapshot.waitMinutes > 0 {
		busyScore = float64(snapshot.waitMinutes) * 0.35
	}
	if snapshot.onlineOpen {
		busyScore += 6
	}
	if queueDashboardIsOpen(snapshot.storeStatus) {
		busyScore += 4
	}
	return QueueDashboardStoreRow{
		StoreID:         snapshot.storeID,
		StoreName:       dashboardStoreName(nil, snapshot.storeID, snapshot.storeName),
		City:            snapshot.city,
		Area:            snapshot.area,
		WaitMinutes:     snapshot.waitMinutes,
		QueueGroups:     queueGroups,
		StoreStatus:     snapshot.storeStatus,
		NetTicketStatus: snapshot.netTicketStatus,
		OnlineOpen:      snapshot.onlineOpen,
		CalledNo:        snapshot.calledNo,
		LatestAt:        latestAt,
		Source:          snapshot.source,
		BusyScore:       busyScore,
		TypicalGroups:   typical,
		SafeGroups:      safe,
		BaselineSamples: samples,
		Confidence:      confidence,
	}
}

func sortQueueDashboardStoreRows(rows []QueueDashboardStoreRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].BusyScore != rows[j].BusyScore {
			return rows[i].BusyScore > rows[j].BusyScore
		}
		if rows[i].QueueGroups != rows[j].QueueGroups {
			return rows[i].QueueGroups > rows[j].QueueGroups
		}
		if rows[i].WaitMinutes != rows[j].WaitMinutes {
			return rows[i].WaitMinutes > rows[j].WaitMinutes
		}
		return rows[i].StoreID < rows[j].StoreID
	})
}

func queueDashboardScope(query QueueDashboardQuery, rows []QueueDashboardStoreRow, baselineStatus QueueBaselineRemoteStatus, trendSource string) QueueDashboardScope {
	source := trendSource
	if baselineStatus.Used && trendSource == "remote_baseline" {
		source = "remote"
	}
	scope := QueueDashboardScope{
		Mode:       query.Scope,
		Source:     source,
		StoreCount: len(rows),
		Message:    "默认看全国公开基准；选择门店后会叠加本机常用门店的叫号细节。",
	}
	if query.Scope == "local" {
		scope.Message = "当前优先看本机常用门店；线上全国数据只按需补门店维度和基准。"
	}
	return scope
}

func queueDashboardWarnings(query QueueDashboardQuery, baselineStatus QueueBaselineRemoteStatus, baselineErr error, localBaselineRecords, localObservationRecords, trendPoints, heatmapPoints, calledCurvePoints int) []string {
	warnings := []string{}
	if baselineErr != nil {
		warnings = append(warnings, "线上排队基准暂时不可用，已退回本机数据："+baselineErr.Error())
	}
	if !baselineStatus.Used {
		warnings = append(warnings, "未连接线上排队基准时，只能看到本机已采集门店。")
	}
	if query.DateType == "all" {
		warnings = append(warnings, "默认已把节假日从周一到周日规律里剔出；要看节假日请切换到“节假日”。")
	}
	if trendPoints == 0 && localBaselineRecords+localObservationRecords == 0 {
		warnings = append(warnings, "本机还没有实时趋势样本；连接线上排队基准或开启本机采集后会出现近 1/3/6/12 小时曲线。")
	}
	if calledCurvePoints == 0 && localObservationRecords == 0 {
		warnings = append(warnings, "本机和线上排队基准都没有叫号明细；先换门店，或开启本机采集积累叫号曲线。")
	} else if calledCurvePoints == 0 {
		warnings = append(warnings, "当前筛选下没有 10:00-22:00 的叫号点；换门店或日期类型再看。")
	}
	if heatmapPoints == 0 && baselineStatus.Used {
		warnings = append(warnings, "线上排队基准还在积累，日期类型补充暂时没有足够样本。")
	}
	return warnings
}

func queueDashboardStoreAllowed(storeID string, storeFilter map[string]bool) bool {
	storeID = strings.TrimSpace(storeID)
	if storeID == "" {
		return false
	}
	return len(storeFilter) == 0 || storeFilter[storeID]
}

func dashboardStoreName(names map[string]string, storeID, fallback string) string {
	if names != nil {
		if name := strings.TrimSpace(names[storeID]); name != "" {
			return name
		}
	}
	if fallback = strings.TrimSpace(fallback); fallback != "" {
		return fallback
	}
	return strings.TrimSpace(storeID)
}

func queueDashboardIsOpen(status string) bool {
	status = strings.ToUpper(strings.TrimSpace(status))
	return status == "OPEN" || status == "PRE_OPEN" || strings.Contains(status, "OPEN")
}

func queueDashboardTimeBucket(at time.Time, minutes int) time.Time {
	if minutes <= 0 {
		minutes = queueDashboardDefaultBucketMins
	}
	minute := (at.Minute() / minutes) * minutes
	return time.Date(at.Year(), at.Month(), at.Day(), at.Hour(), minute, 0, 0, at.Location())
}

func queueDashboardBucketInWindow(bucket string, now time.Time, windowHours int) bool {
	seconds := ParseTimeSeconds(compactTrendTime(bucket))
	if seconds < 0 {
		return false
	}
	if windowHours <= 0 {
		windowHours = queueDashboardDefaultWindowHours
	}
	current := now.Hour()*3600 + now.Minute()*60 + now.Second()
	start := current - windowHours*3600
	if start >= 0 {
		return seconds >= start && seconds <= current
	}
	return seconds >= 0 && seconds <= current || seconds >= 24*3600+start
}

func queueDashboardRollupDateTypeAllowed(dateType, filter string, includeHolidayInAll bool) bool {
	dateType = strings.ToLower(strings.TrimSpace(dateType))
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" || filter == "all" {
		return includeHolidayInAll || dateType != "holiday"
	}
	return dateType == filter
}

func queueDashboardWeekdayName(weekday int) string {
	switch weekday {
	case 1:
		return "周一"
	case 2:
		return "周二"
	case 3:
		return "周三"
	case 4:
		return "周四"
	case 5:
		return "周五"
	case 6:
		return "周六"
	case 7:
		return "周日"
	default:
		return "未知"
	}
}

func queueDashboardConfidence(samples int) string {
	switch {
	case samples >= 24:
		return "high"
	case samples >= 8:
		return "medium"
	case samples > 0:
		return "low"
	default:
		return "none"
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
