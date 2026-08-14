package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/platform"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	queueObservationFile = "queue_observations.jsonl"
	queueSessionFile     = "queue_sessions.jsonl"
	queueStatsFile       = "queue_stats.json"
	queueHolidayFile     = "holidays.json"
	queueDataStaleAfter  = 6 * time.Hour

	queueSourceEndpointStores    = "stores"
	queueSourceEndpointStoreByID = "getStoreById"
	queueAPIProfilePublicV1      = "public-profile-v1"
	queueAPIProfileStoreDetailV1 = "store-detail-profile-v1"
)

// queueObservationMu 串行化对 queue_observations.jsonl 的追加写。
// sampling、基准采集等多个 goroutine 会并发写入，不串行化会导致 JSON 行交错/截断，
// 下游逐行 Unmarshal 时静默丢观测数据（参考 history.go 的 historyMu）。
var queueObservationMu sync.Mutex

type QueueObservation struct {
	CollectedAt       string           `json:"collected_at"`
	StoreID           string           `json:"store_id"`
	WaitMinutes       int              `json:"wait_minutes"`
	GroupQueuesCount  int              `json:"group_queues_count"`
	StoreStatus       string           `json:"store_status"`
	NetTicketStatus   string           `json:"net_ticket_status"`
	ReservationStatus string           `json:"reservation_status"`
	OnlineOpen        bool             `json:"online_open"`
	WaitTimeCounter   int              `json:"wait_time_counter"`
	WaitTimeCap       int              `json:"wait_time_cap"`
	SourceEndpoint    string           `json:"source_endpoint"`
	APIProfileVersion string           `json:"api_profile_version"`
	DisplayCalledNo   int              `json:"display_called_no,omitempty"`
	GroupQueues       QueueGroupQueues `json:"group_queues,omitempty"`

	// Timestamp is the legacy local JSONL field. New records use collected_at,
	// but old files are still accepted so existing local history stays usable.
	Timestamp string `json:"ts,omitempty"`
}

type QueueSession struct {
	StoreID                string `json:"store_id"`
	TakenAt                string `json:"taken_at"`
	TicketNo               int    `json:"ticket_no,omitempty"`
	DisplayCalledNoAtTake  int    `json:"display_called_no_at_take,omitempty"`
	CheckedInAt            string `json:"checked_in_at,omitempty"`
	CalledForUserAt        string `json:"called_for_user_at,omitempty"`
	CalledNoWhenUserCalled int    `json:"called_no_when_user_called,omitempty"`
	ActualWaitMinutes      int    `json:"actual_wait_minutes,omitempty"`
	PartySize              int    `json:"party_size,omitempty"`
	TableType              string `json:"table_type,omitempty"`
	ExpiredOrMissed        bool   `json:"expired_or_missed,omitempty"`
	Manual                 bool   `json:"manual,omitempty"`
}

type QueueTrendQuery struct {
	StoreIDs      []string `json:"store_ids"`
	DateType      string   `json:"date_type"`
	From          string   `json:"from"`
	To            string   `json:"to"`
	Start         string   `json:"start"`
	End           string   `json:"end"`
	BucketMinutes int      `json:"bucket_minutes"`
}

type QueueTrendResponse struct {
	GeneratedAt     string                     `json:"generated_at"`
	Filters         QueueTrendQuery            `json:"filters"`
	Summary         QueueTrendSummary          `json:"summary"`
	Series          []QueueTrendPoint          `json:"series"`
	Recommendations []QueueTrendRecommendation `json:"recommendations"`
	Stores          []QueueTrendStore          `json:"stores"`
	Latest          []QueueBaselineLatest      `json:"latest,omitempty"`
	Sampling        QueueSamplingStatus        `json:"sampling"`
	Baseline        QueueBaselineRemoteStatus  `json:"baseline"`
	Scope           QueueTrendScope            `json:"scope"`
	Warnings        []string                   `json:"warnings,omitempty"`
}

type QueueTrendSummary struct {
	ObservationRecords int    `json:"observation_records"`
	SessionRecords     int    `json:"session_records"`
	ActualSamples      int    `json:"actual_samples"`
	GlobalSamples      int    `json:"global_samples"`
	ActualPassedTotal  int    `json:"actual_passed_total"`
	GlobalPassedTotal  int    `json:"global_passed_total"`
	BaselineRecords    int    `json:"baseline_records"`
	BaselineSamples    int    `json:"baseline_samples"`
	LastObservationAt  string `json:"last_observation_at,omitempty"`
	LastSessionAt      string `json:"last_session_at,omitempty"`
	BaselineUpdatedAt  string `json:"baseline_updated_at,omitempty"`
}

type QueueTrendPoint struct {
	StoreID             string   `json:"store_id"`
	StoreName           string   `json:"store_name"`
	DateType            string   `json:"date_type"`
	DateTypeName        string   `json:"date_type_name"`
	Bucket              string   `json:"bucket"`
	ActualPassed        int      `json:"actual_passed"`
	GlobalPassed        int      `json:"global_passed"`
	ActualSamples       int      `json:"actual_samples"`
	GlobalSamples       int      `json:"global_samples"`
	ObservationSamples  int      `json:"observation_samples"`
	SessionSamples      int      `json:"session_samples"`
	BaselineSamples     int      `json:"baseline_samples"`
	WaitP50Minutes      *float64 `json:"wait_p50_minutes,omitempty"`
	WaitP80Minutes      *float64 `json:"wait_p80_minutes,omitempty"`
	BaselineWaitMinutes *float64 `json:"baseline_wait_minutes,omitempty"`
	BaselineSafeMinutes *float64 `json:"baseline_safe_minutes,omitempty"`
	BusyRate            *float64 `json:"busy_rate,omitempty"`
	OnlineOpenRate      *float64 `json:"online_open_rate,omitempty"`
	QueueGroupsTypical  *float64 `json:"queue_groups_typical,omitempty"`
	QueueGroupsSafe     *float64 `json:"queue_groups_safe,omitempty"`
	MissedRate          float64  `json:"missed_rate"`
	Confidence          string   `json:"confidence"`
	LastObservationAt   string   `json:"last_observation_at,omitempty"`
}

type QueueTrendRecommendation struct {
	StoreID              string   `json:"store_id"`
	StoreName            string   `json:"store_name"`
	DateType             string   `json:"date_type"`
	DateTypeName         string   `json:"date_type_name"`
	Bucket               string   `json:"bucket"`
	Score                float64  `json:"score"`
	Confidence           string   `json:"confidence"`
	PredictedWaitMinutes *float64 `json:"predicted_wait_minutes,omitempty"`
	ActualPassed         int      `json:"actual_passed"`
	GlobalPassed         int      `json:"global_passed"`
	Samples              int      `json:"samples"`
	ActionLabel          string   `json:"action_label"`
	Reason               string   `json:"reason"`
}

type QueueTrendStore struct {
	StoreID   string `json:"store_id"`
	StoreName string `json:"store_name"`
}

type QueueTrendScope struct {
	Mode       string `json:"mode"`
	StoreCount int    `json:"store_count"`
	Message    string `json:"message"`
}

type QueueSamplingStatus struct {
	Enabled            bool            `json:"enabled"`
	Running            bool            `json:"running"`
	DaemonRunning      bool            `json:"daemon_running"`
	AppAutoStart       bool            `json:"app_auto_start"`
	SystemAutoStart    AutoStartStatus `json:"system_auto_start"`
	AuthOK             bool            `json:"auth_ok"`
	NeedsAuth          bool            `json:"needs_auth"`
	NeedsBackground    bool            `json:"needs_background"`
	NeedsDataRefresh   bool            `json:"needs_data_refresh"`
	PermissionStatus   string          `json:"permission_status"`
	Message            string          `json:"message"`
	LastRunAt          string          `json:"last_run_at,omitempty"`
	LastError          string          `json:"last_error,omitempty"`
	LastDataAt         string          `json:"last_data_at,omitempty"`
	DataStaleHours     int             `json:"data_stale_hours,omitempty"`
	SamplingLogPath    string          `json:"sampling_log_path,omitempty"`
	SamplingConfigPath string          `json:"sampling_config_path,omitempty"`
}

type QueueLocalStat struct {
	StoreID                 string   `json:"store_id"`
	Weekday                 int      `json:"weekday"`
	TimeBucket              string   `json:"time_bucket"`
	TableType               string   `json:"table_type"`
	PartySizeBucket         string   `json:"party_size_bucket"`
	Samples                 int      `json:"samples"`
	WaitP50Minutes          *float64 `json:"wait_p50_minutes,omitempty"`
	WaitP80Minutes          *float64 `json:"wait_p80_minutes,omitempty"`
	CheckinToCallP50Minutes *float64 `json:"checkin_to_call_p50_minutes,omitempty"`
	MissedRate              float64  `json:"missed_rate"`
}

type queueTrendAccumulator struct {
	point          QueueTrendPoint
	waits          []float64
	missed         int
	sessionTotal   int
	lastObservedAt time.Time
	baseline       queueBaselineWeightedAccumulator
}

type queueBaselineWeightedAccumulator struct {
	waitTypicalSum  float64
	waitTypicalN    int
	waitSafeSum     float64
	waitSafeN       int
	busyRateSum     float64
	busyRateN       int
	onlineRateSum   float64
	onlineRateN     int
	queueTypicalSum float64
	queueTypicalN   int
	queueSafeSum    float64
	queueSafeN      int
}

type queueLocalBucket struct {
	storeID         string
	weekday         int
	timeBucket      string
	tableType       string
	partySizeBucket string
	waits           []float64
	checkinWaits    []float64
	missed          int
	total           int
}

type queueHolidayConfig struct {
	Holidays []string `json:"holidays"`
	Workdays []string `json:"workdays"`
}

func queueObservationPath() string {
	return filepath.Join(AppDirPath(), queueObservationFile)
}

func queueSessionPath() string {
	return filepath.Join(AppDirPath(), queueSessionFile)
}

func queueStatsPath() string {
	return filepath.Join(AppDirPath(), queueStatsFile)
}

func queueHolidayPath() string {
	return filepath.Join(AppDirPath(), queueHolidayFile)
}

func appendQueueObservation(observation QueueObservation) error {
	queueObservationMu.Lock()
	defer queueObservationMu.Unlock()

	if strings.TrimSpace(observation.StoreID) == "" {
		return nil
	}
	normalizeQueueObservationForWrite(&observation, time.Now())
	if strings.TrimSpace(observation.CollectedAt) == "" {
		observation.CollectedAt = time.Now().Format(time.RFC3339)
	}
	// 空值过滤：没有叫号、没有等待、没有排队桌、各队列为空，则不落盘。
	// 防止关店/接口异常的全零快照污染历史，拉低分位数和速率。
	if observation.DisplayCalledNo <= 0 && observation.WaitMinutes <= 0 && observation.GroupQueuesCount <= 0 && !queueGroupQueuesHasAny(observation.GroupQueues) {
		return nil
	}
	if err := os.MkdirAll(AppDirPath(), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(queueObservationPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	// Best-effort: keep local sampling traces owner-only when the filesystem supports it.
	_ = f.Chmod(0o600)
	data, err := json.Marshal(observation)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	// 周期性裁剪：每 queueJSONLTrimInterval 次追加检查一次行数上限，
	// 防止长期运行无限膨胀拖慢所有 loader。
	queueObservationWriteCounter++
	if queueObservationWriteCounter%queueJSONLTrimInterval == 0 {
		trimJSONLFileLocked(queueObservationPath(), queueObservationMaxLines, time.Now())
	}
	// 叫号推进了 → 回填可能已被叫到的开放预测，积累「预测 vs 实际」回测样本。
	// 放在写盘成功之后、best-effort：失败不影响观测写入。
	if observation.DisplayCalledNo > 0 {
		backfillEtaOnObservation(observation.StoreID, observation.DisplayCalledNo, time.Now())
	}
	return nil
}

func normalizeQueueObservationForWrite(observation *QueueObservation, now time.Time) {
	if observation == nil {
		return
	}
	if now.IsZero() {
		now = time.Now()
	}
	if strings.TrimSpace(observation.CollectedAt) == "" {
		if legacy := strings.TrimSpace(observation.Timestamp); legacy != "" {
			observation.CollectedAt = legacy
		} else {
			observation.CollectedAt = now.Format(time.RFC3339)
		}
	}
	observation.Timestamp = ""
	observation.StoreID = strings.TrimSpace(observation.StoreID)
	observation.StoreStatus = strings.TrimSpace(observation.StoreStatus)
	observation.NetTicketStatus = strings.TrimSpace(observation.NetTicketStatus)
	observation.ReservationStatus = strings.TrimSpace(observation.ReservationStatus)
	observation.SourceEndpoint = strings.TrimSpace(observation.SourceEndpoint)
	if observation.SourceEndpoint == "" {
		observation.SourceEndpoint = queueSourceEndpointStoreByID
	}
	observation.APIProfileVersion = strings.TrimSpace(observation.APIProfileVersion)
	if observation.APIProfileVersion == "" {
		observation.APIProfileVersion = queueAPIProfileStoreDetailV1
	}
	observation.GroupQueues = normalizeQueueGroupQueues(observation.GroupQueues)
	if observation.DisplayCalledNo <= 0 {
		observation.DisplayCalledNo = queueObservationCalledNo(*observation)
	}
}

func normalizeQueueObservationForRead(observation *QueueObservation) {
	if observation == nil {
		return
	}
	if strings.TrimSpace(observation.CollectedAt) == "" {
		observation.CollectedAt = strings.TrimSpace(observation.Timestamp)
	}
	observation.GroupQueues = normalizeQueueGroupQueues(observation.GroupQueues)
	if observation.DisplayCalledNo <= 0 {
		observation.DisplayCalledNo = queueObservationCalledNo(*observation)
	}
}

func queueObservationCollectedAt(observation QueueObservation) string {
	if value := strings.TrimSpace(observation.CollectedAt); value != "" {
		return value
	}
	return strings.TrimSpace(observation.Timestamp)
}

func queueObservationFromStoreInfo(storeID string, store StoreInfo, now time.Time) (QueueObservation, bool) {
	storeID = strings.TrimSpace(storeID)
	if storeID == "" {
		return QueueObservation{}, false
	}
	if now.IsZero() {
		now = time.Now()
	}
	wait := store.Wait
	if wait < 0 {
		wait = 0
	}
	observation := QueueObservation{
		CollectedAt:       now.Format(time.RFC3339),
		StoreID:           storeID,
		WaitMinutes:       wait,
		GroupQueuesCount:  store.GroupQueuesCount,
		StoreStatus:       strings.TrimSpace(store.StoreStatus),
		NetTicketStatus:   strings.TrimSpace(store.NetTicketStatus),
		ReservationStatus: strings.TrimSpace(store.ReservationStatus),
		OnlineOpen:        isQueueOnlineOpen(store),
		WaitTimeCounter:   store.WaitTimeCounter,
		SourceEndpoint:    queueSourceEndpointStoreByID,
		APIProfileVersion: queueAPIProfileStoreDetailV1,
		GroupQueues:       normalizeQueueGroupQueues(store.GroupQueues),
	}
	observation.DisplayCalledNo = queueObservationCalledNo(observation)
	return observation, wait > 0 || store.GroupQueuesCount > 0 || observation.OnlineOpen || queueGroupQueuesHasAny(observation.GroupQueues)
}

func normalizeQueueGroupQueues(queues QueueGroupQueues) QueueGroupQueues {
	return QueueGroupQueues{
		ReservationQueue: cleanQueueNumbers(queues.ReservationQueue),
		CounterQueue:     cleanQueueNumbers(queues.CounterQueue),
		BoothQueue:       cleanQueueNumbers(queues.BoothQueue),
		MixedQueue:       cleanQueueNumbers(queues.MixedQueue),
	}
}

func cleanQueueNumbers(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func queueGroupQueuesHasAny(queues QueueGroupQueues) bool {
	return len(queues.MixedQueue) > 0 ||
		len(queues.ReservationQueue) > 0 ||
		len(queues.CounterQueue) > 0 ||
		len(queues.BoothQueue) > 0
}

func queueObservationCalledNo(observation QueueObservation) int {
	if observation.DisplayCalledNo > 0 {
		return observation.DisplayCalledNo
	}
	return currentDineInCalledNo(observation.GroupQueues)
}

func currentDineInCalledNo(queues QueueGroupQueues) int {
	best := 0
	for _, values := range [][]string{queues.BoothQueue, queues.MixedQueue, queues.CounterQueue} {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if n, err := strconv.Atoi(value); err == nil && n > best {
				best = n
			}
		}
	}
	return best
}

func isQueueOnlineOpen(store StoreInfo) bool {
	values := []string{store.NetTicketStatus, store.RemoteTicketing, store.StoreStatus}
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if strings.Contains(value, "OPEN") || value == "ON" || value == "ONLINE" {
			return true
		}
	}
	return false
}

func BuildQueueTrends(query QueueTrendQuery, now time.Time) QueueTrendResponse {
	return BuildQueueTrendsWithContext(context.Background(), query, now)
}

func BuildQueueTrendsWithContext(ctx context.Context, query QueueTrendQuery, now time.Time) QueueTrendResponse {
	if now.IsZero() {
		now = time.Now()
	}
	query = normalizeQueueTrendQuery(query, now)
	sessions := loadQueueSessions()
	observations := loadQueueObservations()
	holidays, workdays, holidayConfigured := loadQueueHolidayDates()
	storeNames := queueTrendStoreNames(query.StoreIDs, sessions, observations)
	storeFilter := stringSet(query.StoreIDs)
	series := map[string]*queueTrendAccumulator{}
	summary := QueueTrendSummary{
		ObservationRecords: len(observations),
		SessionRecords:     len(sessions),
	}

	for _, session := range sessions {
		takenAt, ok := parseRFC3339Local(session.TakenAt)
		if !ok {
			continue
		}
		// 归一化到寿司郎门店时区（UTC+8）再切日期/日型/桶：桶键按 CST 营业时间定义，
		// 机器本地时区不同（如 UTC 容器）会把同一时刻劈到不同日期桶。与下方叫号推进
		// 第二遍、queue_advisor.go buildLocalQueuePressureCurvePoints 的做法保持一致。
		takenAt = takenAt.In(SushiroTimezone)
		if !queueTrendMatches(query, takenAt, storeFilter, session.StoreID, holidays, workdays) {
			continue
		}
		updateLatest(&summary.LastSessionAt, takenAt)
		dateType := queueTrendDateType(takenAt, holidays, workdays)
		acc := queueTrendAcc(series, session.StoreID, storeNames[session.StoreID], dateType, queueTrendBucket(takenAt, query.BucketMinutes))
		acc.sessionTotal++
		acc.point.SessionSamples++
		if session.ExpiredOrMissed {
			acc.missed++
		}
		if wait, ok := queueSessionWaitMinutes(session); ok {
			acc.waits = append(acc.waits, wait)
		}
		if passed, ok := queueActualPassed(session); ok {
			acc.point.ActualPassed += passed
			acc.point.ActualSamples++
			summary.ActualPassedTotal += passed
			summary.ActualSamples++
		}
	}

	summary = addQueueObservationsToTrend(series, summary, query, observations, storeNames, storeFilter, holidays, workdays)

	baseline, baselineStatus, baselineErr := loadRemoteQueueBaselineForStores(ctx, query.StoreIDs, now)
	if baselineStatus.Used {
		summary = addQueueBaselineToTrend(series, summary, query, baseline, storeNames, storeFilter)
	}
	latest := filterQueueBaselineLatest(baseline.Latest, storeFilter)
	points := finalizeQueueTrendPoints(series)
	warnings := queueTrendWarnings(query, holidayConfigured, summary)
	if baselineErr != nil {
		warnings = append(warnings, "全国基准数据库连接失败，已只使用本机数据。")
	}
	stores := queueTrendStores(storeNames, query.StoreIDs, points)
	scope := QueueTrendScope{
		Mode:       "local",
		StoreCount: len(stores),
		Message:    "选择你关心的门店持续收集，预测会按门店分别生成。",
	}
	if baselineStatus.Used {
		if summary.ObservationRecords == 0 && summary.SessionRecords == 0 {
			scope.Mode = "baseline"
			scope.Message = "当前使用 Turso 全国基准数据；无需本机凭证也可以先看门店时段基准。"
		} else {
			scope.Mode = "hybrid"
			scope.Message = "当前混合使用 Turso 全国基准和本机真实数据。"
		}
	}
	return QueueTrendResponse{
		GeneratedAt:     now.Format(time.RFC3339),
		Filters:         query,
		Summary:         summary,
		Series:          points,
		Recommendations: BuildQueueTrendRecommendations(points, 5),
		Stores:          stores,
		Latest:          latest,
		Sampling:        buildQueueSamplingStatus(now, summary),
		Baseline:        baselineStatus,
		Scope:           scope,
		Warnings:        warnings,
	}
}

func BuildQueueLocalStats(sessions []QueueSession, minSamples int) ([]QueueLocalStat, int) {
	if minSamples <= 0 {
		minSamples = 1
	}
	buckets := map[string]*queueLocalBucket{}
	usable := 0
	for _, session := range sessions {
		bucket, ok := queueSessionBucket(session)
		if !ok {
			continue
		}
		usable++
		key := strings.Join([]string{bucket.storeID, strconv.Itoa(bucket.weekday), bucket.timeBucket, bucket.tableType, bucket.partySizeBucket}, "|")
		acc := buckets[key]
		if acc == nil {
			acc = &bucket
			buckets[key] = acc
		}
		acc.total++
		if session.ExpiredOrMissed {
			acc.missed++
		}
		if wait, ok := queueSessionWaitMinutes(session); ok {
			acc.waits = append(acc.waits, wait)
		}
		if wait, ok := queueSessionCheckinToCallMinutes(session); ok {
			acc.checkinWaits = append(acc.checkinWaits, wait)
		}
	}

	stats := make([]QueueLocalStat, 0, len(buckets))
	for _, bucket := range buckets {
		if bucket.total < minSamples {
			continue
		}
		stat := QueueLocalStat{
			StoreID:         bucket.storeID,
			Weekday:         bucket.weekday,
			TimeBucket:      bucket.timeBucket,
			TableType:       bucket.tableType,
			PartySizeBucket: bucket.partySizeBucket,
			Samples:         bucket.total,
			MissedRate:      float64(bucket.missed) / float64(bucket.total),
		}
		stat.WaitP50Minutes = floatPtr(queueQuantile(bucket.waits, 0.50))
		stat.WaitP80Minutes = floatPtr(queueQuantile(bucket.waits, 0.80))
		stat.CheckinToCallP50Minutes = floatPtr(queueQuantile(bucket.checkinWaits, 0.50))
		stats = append(stats, stat)
	}
	sort.Slice(stats, func(i, j int) bool {
		a, b := stats[i], stats[j]
		if a.StoreID != b.StoreID {
			return a.StoreID < b.StoreID
		}
		if a.Weekday != b.Weekday {
			return a.Weekday < b.Weekday
		}
		if a.TimeBucket != b.TimeBucket {
			return a.TimeBucket < b.TimeBucket
		}
		if a.TableType != b.TableType {
			return a.TableType < b.TableType
		}
		return a.PartySizeBucket < b.PartySizeBucket
	})
	return stats, usable
}

func addQueueObservationsToTrend(series map[string]*queueTrendAccumulator, summary QueueTrendSummary, query QueueTrendQuery, observations []QueueObservation, storeNames map[string]string, storeFilter map[string]bool, holidays, workdays map[string]bool) QueueTrendSummary {
	observationsByStore := map[string][]QueueObservation{}
	for _, observation := range observations {
		storeID := strings.TrimSpace(observation.StoreID)
		if storeID == "" {
			continue
		}
		observation.GroupQueues = normalizeQueueGroupQueues(observation.GroupQueues)
		observationsByStore[storeID] = append(observationsByStore[storeID], observation)
		if at, ok := parseRFC3339Local(queueObservationCollectedAt(observation)); ok {
			updateLatest(&summary.LastObservationAt, at)
		}
	}
	for storeID, storeObservations := range observationsByStore {
		sort.Slice(storeObservations, func(i, j int) bool {
			leftRaw := queueObservationCollectedAt(storeObservations[i])
			rightRaw := queueObservationCollectedAt(storeObservations[j])
			left, lok := parseRFC3339Local(leftRaw)
			right, rok := parseRFC3339Local(rightRaw)
			if !lok || !rok {
				return leftRaw < rightRaw
			}
			return left.Before(right)
		})
		for _, observation := range storeObservations {
			at, ok := parseRFC3339Local(queueObservationCollectedAt(observation))
			if !ok {
				continue
			}
			// 同 session 循环：先归一到 CST 再匹配/分桶，等待样本和叫号推进才落在同一桶。
			at = at.In(SushiroTimezone)
			if !queueTrendMatches(query, at, storeFilter, storeID, holidays, workdays) {
				continue
			}
			if observation.WaitMinutes > 0 {
				dateType := queueTrendDateType(at, holidays, workdays)
				acc := queueTrendAcc(series, storeID, storeNames[storeID], dateType, queueTrendBucket(at, query.BucketMinutes))
				acc.waits = append(acc.waits, float64(observation.WaitMinutes))
				acc.point.ObservationSamples++
				if at.After(acc.lastObservedAt) {
					acc.lastObservedAt = at
				}
			}
		}
		// 第二遍：从同一天内相邻观测的叫号差推算「叫号推进」(GlobalPassed)，
		// 这是无需用户取号、仅靠公开叫号就能积累的预测信号。
		// lastCalledByDate 按天记上一次叫号，只取 diff>0（叫号前进）且 <=500 的差：
		// 跨天/叫号重置会出现负数或巨大跳变，必须丢弃，否则会把「重置」误算成「几百桌推进」。
		lastCalledByDate := map[string]int{}
		for _, observation := range storeObservations {
			at, ok := parseRFC3339Local(queueObservationCollectedAt(observation))
			if !ok {
				continue
			}
			// 归一化到寿司郎门店时区（UTC+8）再切 dateKey/日型/桶，避免机器本地时区与
			// 远端基准（按 +08 存）错位，导致同一时刻样本劈到不同日期桶。
			at = at.In(SushiroTimezone)
			calledNo := queueObservationCalledNo(observation)
			if calledNo <= 0 {
				continue
			}
			dateKey := at.Format("2006-01-02")
			prevCalledNo := lastCalledByDate[dateKey]
			if prevCalledNo > 0 && queueTrendMatches(query, at, storeFilter, storeID, holidays, workdays) {
				diff := calledNo - prevCalledNo
				if diff > 0 && diff <= 500 {
					dateType := queueTrendDateType(at, holidays, workdays)
					acc := queueTrendAcc(series, storeID, storeNames[storeID], dateType, queueTrendBucket(at, query.BucketMinutes))
					acc.point.GlobalPassed += diff
					acc.point.GlobalSamples++
					if at.After(acc.lastObservedAt) {
						acc.lastObservedAt = at
					}
					summary.GlobalPassedTotal += diff
					summary.GlobalSamples++
				}
			}
			lastCalledByDate[dateKey] = calledNo
		}
	}
	return summary
}

// addQueueBaselineToTrend 把远端全国基准的 rollup 累加进趋势。
// 每条 rollup 已是某店某日型某时段的聚合（typical/safe 等），这里按 SampleCount 做加权累积
// （存 sum 和 N，最后在 finalizeQueueTrendPoints 里除出加权均值），让样本多的时段权重更高。
// 注意 rollup 的 TimeBucket 可能与查询桶宽不同，要用 queueBaselineBucketForQuery 归一到查询桶。
func addQueueBaselineToTrend(series map[string]*queueTrendAccumulator, summary QueueTrendSummary, query QueueTrendQuery, baseline QueueBaselineExport, storeNames map[string]string, storeFilter map[string]bool) QueueTrendSummary {
	for _, store := range baseline.Stores {
		storeID := strconv.Itoa(store.StoreID)
		if storeID == "" {
			continue
		}
		name := strings.TrimSpace(store.Name)
		if name == "" {
			name = storeID
		}
		if _, exists := storeNames[storeID]; !exists {
			storeNames[storeID] = name
		}
	}
	if baseline.Stats.SourceUpdatedAt != "" {
		summary.BaselineUpdatedAt = baseline.Stats.SourceUpdatedAt
	}
	for _, rollup := range baseline.Rollups {
		storeID := strconv.Itoa(rollup.StoreID)
		if storeID == "" || rollup.SampleCount <= 0 {
			continue
		}
		if len(storeFilter) > 0 && !storeFilter[storeID] {
			continue
		}
		dateType := strings.ToLower(strings.TrimSpace(rollup.DateType))
		if query.DateType != "all" && query.DateType != dateType {
			continue
		}
		if !queueTrendBucketInRange(rollup.TimeBucket, query.Start, query.End) {
			continue
		}
		bucket := queueBaselineBucketForQuery(rollup.TimeBucket, query.BucketMinutes)
		acc := queueTrendAcc(series, storeID, storeNames[storeID], dateType, bucket)
		samples := rollup.SampleCount
		acc.point.BaselineSamples += samples
		summary.BaselineRecords++
		summary.BaselineSamples += samples
		if rollup.WaitTypicalMinutes != nil {
			acc.baseline.waitTypicalSum += *rollup.WaitTypicalMinutes * float64(samples)
			acc.baseline.waitTypicalN += samples
		}
		if rollup.WaitSafeMinutes != nil {
			acc.baseline.waitSafeSum += *rollup.WaitSafeMinutes * float64(samples)
			acc.baseline.waitSafeN += samples
		}
		acc.baseline.busyRateSum += rollup.BusyRate * float64(samples)
		acc.baseline.busyRateN += samples
		acc.baseline.onlineRateSum += rollup.OnlineOpenRate * float64(samples)
		acc.baseline.onlineRateN += samples
		if rollup.QueueGroupsTypical != nil {
			acc.baseline.queueTypicalSum += *rollup.QueueGroupsTypical * float64(samples)
			acc.baseline.queueTypicalN += samples
		}
		if rollup.QueueGroupsSafe != nil {
			acc.baseline.queueSafeSum += *rollup.QueueGroupsSafe * float64(samples)
			acc.baseline.queueSafeN += samples
		}
	}
	return summary
}

func filterQueueBaselineLatest(latest []QueueBaselineLatest, storeFilter map[string]bool) []QueueBaselineLatest {
	if len(latest) == 0 {
		return nil
	}
	if len(storeFilter) == 0 {
		return latest
	}
	out := make([]QueueBaselineLatest, 0, len(storeFilter))
	for _, item := range latest {
		if storeFilter[strconv.Itoa(item.StoreID)] {
			out = append(out, item)
		}
	}
	return out
}

func normalizeQueueTrendQuery(query QueueTrendQuery, now time.Time) QueueTrendQuery {
	// 归一化到寿司郎门店时区：from/to 的默认日期窗口（最近14天）必须按门店当地日期切，
	// 否则机器在非 +08 时区时默认窗口会偏移半天。
	now = now.In(SushiroTimezone)
	query.StoreIDs = UniqueNonEmptyStrings(query.StoreIDs)
	query.DateType = strings.ToLower(strings.TrimSpace(query.DateType))
	if query.DateType == "" {
		query.DateType = "all"
	}
	switch query.DateType {
	case "all", "weekday", "workday", "weekend", "holiday":
	default:
		query.DateType = "all"
	}
	if query.BucketMinutes != 60 {
		query.BucketMinutes = 30
	}
	from, ok := parseTrendDateParam(query.From, now.Location())
	if !ok {
		from = BeginningOfDay(now).AddDate(0, 0, -14)
	}
	to, ok := parseTrendDateParam(query.To, now.Location())
	if !ok {
		to = BeginningOfDay(now)
	}
	if to.Before(from) {
		from, to = to, from
	}
	query.From = from.Format("2006-01-02")
	query.To = to.Format("2006-01-02")
	if ParseTimeSeconds(compactTrendTime(query.Start)) < 0 {
		query.Start = "10:00"
	} else {
		query.Start = displayTrendTime(query.Start)
	}
	if ParseTimeSeconds(compactTrendTime(query.End)) < 0 {
		query.End = "22:00"
	} else {
		query.End = displayTrendTime(query.End)
	}
	return query
}

func queueTrendMatches(query QueueTrendQuery, at time.Time, storeFilter map[string]bool, storeID string, holidays, workdays map[string]bool) bool {
	storeID = strings.TrimSpace(storeID)
	if storeID == "" {
		return false
	}
	if len(storeFilter) > 0 && !storeFilter[storeID] {
		return false
	}
	from, _ := parseTrendDateParam(query.From, at.Location())
	to, _ := parseTrendDateParam(query.To, at.Location())
	day := BeginningOfDay(at)
	if day.Before(from) || day.After(to) {
		return false
	}
	dateType := queueTrendDateType(at, holidays, workdays)
	if query.DateType != "all" && query.DateType != dateType {
		return false
	}
	return queueTrendTimeInRange(at, query.Start, query.End)
}

// queueTrendTimeInRange 判断某时刻是否落在 [start,end) 时间区间内。
// start==end（或解析失败）视为不限制；start<end 是普通区间；
// start>end（如 22:00-02:00）按跨午夜处理：>=start 或 <end 都算在内。
func queueTrendTimeInRange(at time.Time, startRaw, endRaw string) bool {
	start := ParseTimeSeconds(compactTrendTime(startRaw))
	end := ParseTimeSeconds(compactTrendTime(endRaw))
	if start < 0 || end < 0 || start == end {
		return true
	}
	current := at.Hour()*3600 + at.Minute()*60 + at.Second()
	if start < end {
		return current >= start && current < end
	}
	return current >= start || current < end
}

func queueTrendBucketInRange(bucket, startRaw, endRaw string) bool {
	seconds := ParseTimeSeconds(compactTrendTime(bucket))
	if seconds < 0 {
		return false
	}
	start := ParseTimeSeconds(compactTrendTime(startRaw))
	end := ParseTimeSeconds(compactTrendTime(endRaw))
	if start < 0 || end < 0 || start == end {
		return true
	}
	if start < end {
		return seconds >= start && seconds < end
	}
	return seconds >= start || seconds < end
}

func queueBaselineBucketForQuery(bucket string, minutes int) string {
	seconds := ParseTimeSeconds(compactTrendTime(bucket))
	if seconds < 0 {
		return strings.TrimSpace(bucket)
	}
	hour := seconds / 3600
	minute := (seconds % 3600) / 60
	if minutes == 60 {
		minute = 0
	}
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

func queueTrendAcc(series map[string]*queueTrendAccumulator, storeID, storeName, dateType, bucket string) *queueTrendAccumulator {
	key := strings.Join([]string{storeID, dateType, bucket}, "|")
	acc := series[key]
	if acc == nil {
		if strings.TrimSpace(storeName) == "" {
			storeName = storeID
		}
		acc = &queueTrendAccumulator{
			point: QueueTrendPoint{
				StoreID:      storeID,
				StoreName:    storeName,
				DateType:     dateType,
				DateTypeName: queueTrendDateTypeName(dateType),
				Bucket:       bucket,
			},
		}
		series[key] = acc
	}
	return acc
}

// finalizeQueueTrendPoints 把各累加器收口成对外点：本机真实样本算 P50/P80 分位数，
// 远端基准做兜底——当某桶本机无样本时，用基准的 typical 充当 P50、safe 充当 P80，
// 让无本机数据的时段也有曲线，而非空缺。最后按 bucket/dateType/storeID 稳定排序。
func finalizeQueueTrendPoints(series map[string]*queueTrendAccumulator) []QueueTrendPoint {
	points := make([]QueueTrendPoint, 0, len(series))
	for _, acc := range series {
		point := acc.point
		point.WaitP50Minutes = floatPtr(queueQuantile(acc.waits, 0.50))
		point.WaitP80Minutes = floatPtr(queueQuantile(acc.waits, 0.80))
		if acc.baseline.waitTypicalN > 0 {
			// 基准加权均值；若本机没有 P50（waits 为空），用基准 typical 顶上，保证有值。
			point.BaselineWaitMinutes = floatPtr(acc.baseline.waitTypicalSum / float64(acc.baseline.waitTypicalN))
			if point.WaitP50Minutes == nil {
				point.WaitP50Minutes = cloneFloatPtr(point.BaselineWaitMinutes)
			}
		}
		if acc.baseline.waitSafeN > 0 {
			// 基准 safe（偏保守）充当 P80 的兜底。
			point.BaselineSafeMinutes = floatPtr(acc.baseline.waitSafeSum / float64(acc.baseline.waitSafeN))
			if point.WaitP80Minutes == nil {
				point.WaitP80Minutes = cloneFloatPtr(point.BaselineSafeMinutes)
			}
		}
		if acc.baseline.busyRateN > 0 {
			point.BusyRate = floatPtr(acc.baseline.busyRateSum / float64(acc.baseline.busyRateN))
		}
		if acc.baseline.onlineRateN > 0 {
			point.OnlineOpenRate = floatPtr(acc.baseline.onlineRateSum / float64(acc.baseline.onlineRateN))
		}
		if acc.baseline.queueTypicalN > 0 {
			point.QueueGroupsTypical = floatPtr(acc.baseline.queueTypicalSum / float64(acc.baseline.queueTypicalN))
		}
		if acc.baseline.queueSafeN > 0 {
			point.QueueGroupsSafe = floatPtr(acc.baseline.queueSafeSum / float64(acc.baseline.queueSafeN))
		}
		if acc.sessionTotal > 0 {
			point.MissedRate = float64(acc.missed) / float64(acc.sessionTotal)
		}
		point.Confidence = queueTrendConfidence(point)
		if !acc.lastObservedAt.IsZero() {
			point.LastObservationAt = acc.lastObservedAt.Format(time.RFC3339)
		}
		points = append(points, point)
	}
	sort.Slice(points, func(i, j int) bool {
		a, b := points[i], points[j]
		if a.Bucket != b.Bucket {
			return a.Bucket < b.Bucket
		}
		if a.DateType != b.DateType {
			return queueTrendDateTypeRank(a.DateType) < queueTrendDateTypeRank(b.DateType)
		}
		return a.StoreID < b.StoreID
	})
	return points
}

func queueTrendConfidence(point QueueTrendPoint) string {
	samples := point.ActualSamples + point.GlobalSamples + point.ObservationSamples + point.BaselineSamples
	switch {
	case samples >= 12:
		return "high"
	case samples >= 4:
		return "medium"
	case samples > 0:
		return "low"
	default:
		return "none"
	}
}

func BuildQueueTrendRecommendations(points []QueueTrendPoint, limit int) []QueueTrendRecommendation {
	if limit <= 0 {
		limit = 5
	}
	recommendations := make([]QueueTrendRecommendation, 0, len(points))
	for _, point := range points {
		samples := point.ActualSamples + point.GlobalSamples + point.ObservationSamples + point.BaselineSamples
		if samples == 0 {
			continue
		}
		score := queueRecommendationConfidenceScore(point.Confidence)
		score += math.Min(float64(samples)*2, 20)
		if point.ActualSamples > 0 {
			score += 8
		}
		if point.GlobalSamples > 0 {
			score += 4
		}
		if point.BaselineSamples > 0 && point.ActualSamples == 0 && point.GlobalSamples == 0 {
			score += 3
		}
		if point.WaitP50Minutes != nil {
			score += math.Max(0, 45-math.Min(*point.WaitP50Minutes, 90)) * 0.8
		} else if point.GlobalPassed > 0 {
			score += math.Max(0, 25-math.Abs(float64(point.GlobalPassed)-30)/2)
		}
		score -= point.MissedRate * 30
		recommendations = append(recommendations, QueueTrendRecommendation{
			StoreID:              point.StoreID,
			StoreName:            point.StoreName,
			DateType:             point.DateType,
			DateTypeName:         point.DateTypeName,
			Bucket:               point.Bucket,
			Score:                math.Round(score*10) / 10,
			Confidence:           point.Confidence,
			PredictedWaitMinutes: cloneFloatPtr(point.WaitP50Minutes),
			ActualPassed:         point.ActualPassed,
			GlobalPassed:         point.GlobalPassed,
			Samples:              samples,
			ActionLabel:          queueRecommendationAction(point),
			Reason:               queueRecommendationReason(point),
		})
	}
	sort.Slice(recommendations, func(i, j int) bool {
		a, b := recommendations[i], recommendations[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.StoreName != b.StoreName {
			return a.StoreName < b.StoreName
		}
		if a.DateType != b.DateType {
			return queueTrendDateTypeRank(a.DateType) < queueTrendDateTypeRank(b.DateType)
		}
		return a.Bucket < b.Bucket
	})
	if len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}
	return recommendations
}

func queueRecommendationConfidenceScore(confidence string) float64 {
	switch confidence {
	case "high":
		return 40
	case "medium":
		return 25
	case "low":
		return 10
	default:
		return 0
	}
}

func queueRecommendationAction(point QueueTrendPoint) string {
	if point.Confidence == "high" || point.Confidence == "medium" {
		if point.WaitP50Minutes != nil && *point.WaitP50Minutes <= 45 && point.MissedRate < 0.25 {
			return "优先考虑"
		}
		return "候选时段"
	}
	return "继续观察"
}

func queueRecommendationReason(point QueueTrendPoint) string {
	parts := []string{}
	if point.WaitP50Minutes != nil {
		parts = append(parts, fmt.Sprintf("P50 等待约 %d 分钟", int(math.Round(*point.WaitP50Minutes))))
	} else if point.GlobalSamples > 0 {
		parts = append(parts, fmt.Sprintf("已有 %d 段公开叫号推进记录", point.GlobalSamples))
	}
	if point.ActualSamples > 0 {
		parts = append(parts, fmt.Sprintf("%d 次真实取号样本", point.ActualSamples))
	}
	if point.GlobalPassed > 0 {
		parts = append(parts, fmt.Sprintf("公开叫号推进 %d 号", point.GlobalPassed))
	}
	if point.BaselineSamples > 0 {
		parts = append(parts, fmt.Sprintf("全国基准 %d 个样本", point.BaselineSamples))
	}
	if point.OnlineOpenRate != nil {
		parts = append(parts, fmt.Sprintf("远程取号开放率约 %d%%", int(math.Round(*point.OnlineOpenRate*100))))
	}
	if point.MissedRate >= 0.25 {
		parts = append(parts, "过号风险偏高，注意签到时间")
	}
	if len(parts) == 0 {
		return "样本刚开始积累，先作为观察时段。"
	}
	return strings.Join(parts, "；") + "。"
}

func cloneFloatPtr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	copy := *v
	return &copy
}

func queueTrendWarnings(query QueueTrendQuery, holidayConfigured bool, summary QueueTrendSummary) []string {
	warnings := []string{}
	if !holidayConfigured {
		warnings = append(warnings, "节假日表未配置，节假日会先按工作日/周末处理。")
	}
	if query.DateType == "holiday" && !holidayConfigured {
		warnings = append(warnings, "要单独看节假日，可在 ~/.sushiro/holidays.json 写入 holidays/workdays 日期列表。")
	}
	if summary.ObservationRecords == 0 && summary.SessionRecords == 0 && summary.BaselineSamples == 0 {
		warnings = append(warnings, "还没有足够数据，先开启信息收集并选择关心的门店。")
	}
	if summary.GlobalSamples == 0 && summary.BaselineSamples == 0 {
		warnings = append(warnings, "公开叫号推进还不够，保持信息收集会补齐折线。")
	}
	if summary.ActualSamples == 0 && summary.BaselineSamples == 0 {
		warnings = append(warnings, "真实取号样本还不够，到店取号后记录叫到号会让预测更准。")
	} else if summary.ActualSamples == 0 && summary.BaselineSamples > 0 {
		warnings = append(warnings, "当前推荐主要来自全国基准；真实取号样本会进一步修正过号风险。")
	}
	return warnings
}

func buildQueueSamplingStatus(now time.Time, summary QueueTrendSummary) QueueSamplingStatus {
	cfg := LoadSamplingConfig()
	state := sampler.GetState()
	systemAuto := SamplingAutoStartStatus()
	status := QueueSamplingStatus{
		Enabled:            cfg.Enabled,
		Running:            state.Running,
		DaemonRunning:      isSamplingDaemonRunning(),
		AppAutoStart:       cfg.AutoStart,
		SystemAutoStart:    systemAuto,
		LastRunAt:          state.LastRunAt,
		LastError:          state.LastError,
		SamplingLogPath:    SamplingLogPath(),
		SamplingConfigPath: samplingConfigPath(),
	}
	if holder, ok := processLockHolder(samplingLockFileName); ok && holder > 0 {
		status.DaemonRunning = true
	}
	tokens, err := LoadLocalConfig()
	if err == nil {
		err = tokens.ValidateForQuery()
	}
	status.AuthOK = err == nil
	status.NeedsAuth = err != nil

	lastDataAt := latestRFC3339(summary.LastObservationAt, summary.LastSessionAt)
	if !lastDataAt.IsZero() {
		status.LastDataAt = lastDataAt.Format(time.RFC3339)
		age := now.Sub(lastDataAt)
		if age > queueDataStaleAfter {
			status.NeedsDataRefresh = true
			status.DataStaleHours = int(math.Round(age.Hours()))
		}
	} else {
		status.NeedsDataRefresh = true
	}
	status.NeedsBackground = !status.Running && !status.DaemonRunning && !systemAuto.Enabled

	switch {
	case status.NeedsAuth:
		status.PermissionStatus = "needs_auth"
		status.Message = "通行证需要更新，重新获取后才能继续信息收集。"
	case status.NeedsBackground:
		status.PermissionStatus = "needs_background"
		status.Message = "还没有持续信息收集。开启后会按门店积累预测数据。"
	case status.NeedsDataRefresh:
		status.PermissionStatus = "needs_update"
		if status.LastDataAt == "" {
			status.Message = "还没有本地排队数据，请先启动信息收集或记录一次真实取号。"
		} else {
			status.Message = "本地数据较旧，建议立即收集一次。"
		}
	default:
		status.PermissionStatus = "ok"
		status.Message = "信息收集状态正常。"
	}
	if state.LastError != "" && strings.Contains(strings.ToLower(state.LastError), "凭证") {
		status.PermissionStatus = "needs_auth"
		status.NeedsAuth = true
		status.Message = "最近信息收集提示通行证异常，请重新获取通行证。"
	}
	return status
}

// queueActualPassed 从一次真实取号会话算「叫到用户时共推进了多少号」：
// passed = 叫到用户时的叫号(CalledNoWhenUserCalled) - 取号时的叫号(DisplayCalledNoAtTake)。
// 这是反映真实排队长度的最强信号（用户亲自等过）。负数或 >1000 视为记录异常/叫号重置，丢弃。
func queueActualPassed(session QueueSession) (int, bool) {
	if session.CalledNoWhenUserCalled <= 0 || session.DisplayCalledNoAtTake <= 0 {
		return 0, false
	}
	passed := session.CalledNoWhenUserCalled - session.DisplayCalledNoAtTake
	if passed < 0 || passed > 1000 {
		return 0, false
	}
	return passed, true
}

func queueSessionBucket(session QueueSession) (queueLocalBucket, bool) {
	storeID := strings.TrimSpace(session.StoreID)
	if storeID == "" {
		return queueLocalBucket{}, false
	}
	takenAt, ok := parseRFC3339Local(session.TakenAt)
	if !ok {
		return queueLocalBucket{}, false
	}
	// weekday/半小时桶按 CST 语义定义，先归一（同 BuildQueueTrendsWithContext）。
	takenAt = takenAt.In(SushiroTimezone)
	if _, ok := queueSessionWaitMinutes(session); !ok && !session.ExpiredOrMissed {
		return queueLocalBucket{}, false
	}
	return queueLocalBucket{
		storeID:         storeID,
		weekday:         isoWeekday(takenAt.Weekday()),
		timeBucket:      halfHourBucket(takenAt),
		tableType:       normalizeQueueTableType(session.TableType),
		partySizeBucket: partySizeBucket(session.PartySize),
	}, true
}

// queueSessionWaitMinutes 取会话等待分钟，优先用显式记录的 ActualWaitMinutes；
// 没有则用 calledForUserAt - takenAt 倒推。calledAt 早于 takenAt 视为无效（时间倒错）。
func queueSessionWaitMinutes(session QueueSession) (float64, bool) {
	if session.ActualWaitMinutes > 0 {
		return float64(session.ActualWaitMinutes), true
	}
	takenAt, ok := parseRFC3339Local(session.TakenAt)
	if !ok {
		return 0, false
	}
	calledAt, ok := parseRFC3339Local(session.CalledForUserAt)
	if !ok || calledAt.Before(takenAt) {
		return 0, false
	}
	return calledAt.Sub(takenAt).Minutes(), true
}

func queueSessionCheckinToCallMinutes(session QueueSession) (float64, bool) {
	checkedInAt, ok := parseRFC3339Local(session.CheckedInAt)
	if !ok {
		return 0, false
	}
	calledAt, ok := parseRFC3339Local(session.CalledForUserAt)
	if !ok || calledAt.Before(checkedInAt) {
		return 0, false
	}
	return calledAt.Sub(checkedInAt).Minutes(), true
}

func loadQueueSessions() []QueueSession {
	f, err := os.Open(queueSessionPath())
	if err != nil {
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := []QueueSession{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var session QueueSession
		if json.Unmarshal([]byte(line), &session) == nil {
			out = append(out, session)
		}
	}
	return out
}

func loadQueueObservations() []QueueObservation {
	return queueObservationsReadCache.load(queueObservationPath(), normalizeQueueObservationForRead)
}

func loadQueueHolidayDates() (map[string]bool, map[string]bool, bool) {
	data, err := os.ReadFile(queueHolidayPath())
	if err != nil {
		return map[string]bool{}, map[string]bool{}, false
	}
	var cfg queueHolidayConfig
	if json.Unmarshal(data, &cfg) != nil {
		return map[string]bool{}, map[string]bool{}, false
	}
	holidays := normalizedDateSet(cfg.Holidays)
	workdays := normalizedDateSet(cfg.Workdays)
	return holidays, workdays, len(holidays) > 0 || len(workdays) > 0
}

func normalizedDateSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		if day, ok := parseTrendDateParam(value, SushiroTimezone); ok {
			out[day.Format("2006-01-02")] = true
		}
	}
	return out
}

func queueTrendStoreNames(queryStores []string, sessions []QueueSession, observations []QueueObservation) map[string]string {
	names := map[string]string{}
	reg := GetStoreRegistry()
	add := func(storeID string) {
		storeID = strings.TrimSpace(storeID)
		if storeID == "" {
			return
		}
		if _, ok := names[storeID]; !ok {
			names[storeID] = reg.DisplayName(storeID, storeID)
		}
	}
	for _, storeID := range queryStores {
		add(storeID)
	}
	for _, session := range sessions {
		add(session.StoreID)
	}
	for _, observation := range observations {
		add(observation.StoreID)
	}
	return names
}

func queueTrendStores(names map[string]string, queryStores []string, points []QueueTrendPoint) []QueueTrendStore {
	seen := map[string]bool{}
	ids := []string{}
	for _, storeID := range queryStores {
		if storeID != "" && !seen[storeID] {
			seen[storeID] = true
			ids = append(ids, storeID)
		}
	}
	for _, point := range points {
		if point.StoreID != "" && !seen[point.StoreID] {
			seen[point.StoreID] = true
			ids = append(ids, point.StoreID)
		}
	}
	sort.Strings(ids)
	out := make([]QueueTrendStore, 0, len(ids))
	for _, storeID := range ids {
		name := names[storeID]
		if name == "" {
			name = storeID
		}
		out = append(out, QueueTrendStore{StoreID: storeID, StoreName: name})
	}
	return out
}

// queueTrendDateType 判定某时刻的日型，优先级：调休工作日(workday) > 节假日(holiday) > 周末窗口(weekend) > 工作日(weekday)。
// 调休/节假日查 holidays.json 显式配置；周末用 queueTrendWeekendWindow 的时间窗口判断（周五晚也算周末）。
// 日型决定了历史样本归到哪一类聚合，不同日型的客流差异很大，混在一起会让预测失真。
func queueTrendDateType(at time.Time, holidays, workdays map[string]bool) string {
	key := at.Format("2006-01-02")
	if workdays[key] {
		return "workday"
	}
	if holidays[key] {
		return "holiday"
	}
	if queueTrendWeekendWindow(at) {
		return "weekend"
	}
	return "weekday"
}

// queueTrendWeekendWindow 判断某时刻是否落在「周末客流」窗口内（而非单纯按星期几）。
// 用时间窗口而非整天，是因为周末客流是连续的：周五 16:30 起进入晚高峰（接近周末节奏），
// 周日 22:00 前仍是周末客流、之后夜宵档回落接近工作日。这样把周五晚和周日晚分别划入/划出周末，
// 让同日型聚合更纯净。周六全天算周末。
func queueTrendWeekendWindow(at time.Time) bool {
	seconds := at.Hour()*3600 + at.Minute()*60 + at.Second()
	switch at.Weekday() {
	case time.Friday:
		return seconds >= 16*3600+30*60
	case time.Saturday:
		return true
	case time.Sunday:
		return seconds < 22*3600
	default:
		return false
	}
}

func queueTrendDateTypeName(dateType string) string {
	switch dateType {
	case "weekday":
		return "工作日"
	case "workday":
		return "调休工作日"
	case "weekend":
		return "周末"
	case "holiday":
		return "节假日"
	default:
		return "全部"
	}
}

// queueTrendDateTypeRank 给日型定一个稳定排序序号（weekday<workday<weekend<holiday<all）。
// 用于多处 sort 的次序键（趋势点、推荐排序），让相同时间桶的点按日型从「常态」到「特殊」排列，
// 保持输出顺序稳定可预期。
func queueTrendDateTypeRank(dateType string) int {
	switch dateType {
	case "weekday":
		return 1
	case "workday":
		return 2
	case "weekend":
		return 3
	case "holiday":
		return 4
	default:
		return 9
	}
}

func queueTrendBucket(at time.Time, minutes int) string {
	if minutes == 60 {
		return at.Format("15:00")
	}
	return halfHourBucket(at)
}

// halfHourBucket 把任意时刻向下取整到半小时桶（如 14:07、14:29 都归 14:00，14:30-14:59 归 14:30）。
// 向下取整保证桶内样本可比较；下游所有「半小时聚合」都基于此。
func halfHourBucket(t time.Time) string {
	minute := 0
	if t.Minute() >= 30 {
		minute = 30
	}
	return t.Format("15:") + twoDigits(minute)
}

func partySizeBucket(size int) string {
	switch {
	case size <= 0:
		return "unknown"
	case size <= 2:
		return "1-2"
	case size <= 4:
		return "3-4"
	default:
		return "5+"
	}
}

func normalizeQueueTableType(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	switch v {
	case "T", "TABLE":
		return "T"
	case "C", "COUNTER":
		return "C"
	default:
		return "unknown"
	}
}

// queueQuantile 计算线性插值分位数（类似 numpy 默认 linear 方法）。
// 空集返回 NaN（配合 floatPtr 会被转成 nil，表示「无数据」）。
// 单元素直接返回该值；多元素时按 q*(n-1) 定位，落在两个样本之间则线性插值，
// 避免分位数只能取到样本值导致 P50/P80 阶跃。
func queueQuantile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	out := append([]float64(nil), values...)
	sort.Float64s(out)
	if len(out) == 1 {
		return out[0]
	}
	pos := q * float64(len(out)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return out[lo]
	}
	weight := pos - float64(lo)
	return out[lo]*(1-weight) + out[hi]*weight
}

func floatPtr(v float64) *float64 {
	if math.IsNaN(v) {
		return nil
	}
	return &v
}

func parseRFC3339Local(raw string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func parseTrendDateParam(raw string, loc *time.Location) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if loc == nil {
		loc = SushiroTimezone
	}
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{"2006-01-02", "20060102"} {
		if day, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return BeginningOfDay(day), true
		}
	}
	return time.Time{}, false
}

func compactTrendTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) == 5 && raw[2] == ':' {
		return raw[:2] + raw[3:] + "00"
	}
	if len(raw) == 4 {
		return raw + "00"
	}
	return strings.ReplaceAll(raw, ":", "")
}

func displayTrendTime(raw string) string {
	raw = compactTrendTime(raw)
	if len(raw) >= 4 {
		return raw[:2] + ":" + raw[2:4]
	}
	return raw
}

func sameLocalDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func updateLatest(target *string, t time.Time) {
	if t.IsZero() {
		return
	}
	current, ok := parseRFC3339Local(*target)
	if !ok || t.After(current) {
		*target = t.Format(time.RFC3339)
	}
}

func latestRFC3339(values ...string) time.Time {
	var latest time.Time
	for _, value := range values {
		if t, ok := parseRFC3339Local(value); ok && t.After(latest) {
			latest = t
		}
	}
	return latest
}

func stringSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = true
		}
	}
	return out
}

func twoDigits(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
