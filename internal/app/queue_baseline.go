package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	queueBaselineFile           = "queue_baseline.json"
	queueBaselineRecordsFile    = "queue_baseline.jsonl"
	queueBaselineDefaultMinutes = 5
	queueBaselineTickSeconds    = 30
)

// QueueBaselineRecord 是单家门店在某时刻的基准快照（全量记录，不做活跃过滤，
// 闲时/关店的 0 等位也是基准数据）。写入独立文件，不污染排队趋势观测。
type QueueBaselineRecord struct {
	CollectedAt       string `json:"collected_at"`
	StoreID           int    `json:"store_id"`
	Name              string `json:"name"`
	City              string `json:"city"`
	Area              string `json:"area"`
	WaitMinutes       int    `json:"wait_minutes"`
	GroupQueuesCount  int    `json:"group_queues_count"`
	StoreStatus       string `json:"store_status"`
	NetTicketStatus   string `json:"net_ticket_status"`
	ReservationStatus string `json:"reservation_status"`
	OnlineOpen        bool   `json:"online_open"`
	WaitTimeCounter   int    `json:"wait_time_counter"`
	WaitTimeCap       int    `json:"wait_time_cap"`
	DisplayCalledNo   int    `json:"display_called_no,omitempty"`
	GroupQueuesJSON   string `json:"group_queues_json,omitempty"`
	UpdatedAt         string `json:"updated_at"`
	SourceEndpoint    string `json:"source_endpoint"`
	APIProfileVersion string `json:"api_profile_version"`

	// Legacy local fields. New records use collected_at/wait_minutes so the
	// local JSONL has the same shape as Turso snapshots/latest rows.
	Timestamp string `json:"ts,omitempty"`
	Wait      int    `json:"wait,omitempty"`
}

// QueueBaselineExport 是远端公开「全国基准」JSON 的协议结构。它只包含门店维度
// 和聚合后的半小时基准，不包含用户票号、手机号、凭证参数或个人取号记录。
type QueueBaselineExport struct {
	Version       int                   `json:"version"`
	GeneratedAt   string                `json:"generated_at"`
	Source        string                `json:"source"`
	BucketMinutes int                   `json:"bucket_minutes"`
	DateTypes     []string              `json:"date_types"`
	Stores        []QueueBaselineStore  `json:"stores"`
	Latest        []QueueBaselineLatest `json:"latest,omitempty"`
	Rollups       []QueueBaselineRollup `json:"rollups"`
	Stats         QueueBaselineStats    `json:"stats"`
}

type QueueBaselineStats struct {
	StoreCount      int    `json:"store_count"`
	LatestCount     int    `json:"latest_count,omitempty"`
	RollupCount     int    `json:"rollup_count"`
	SourceUpdatedAt string `json:"source_updated_at,omitempty"`
}

type QueueBaselineRemoteStatus struct {
	Configured      bool   `json:"configured"`
	Used            bool   `json:"used"`
	Provider        string `json:"provider,omitempty"`
	DatabaseURL     string `json:"database_url,omitempty"`
	CloudURL        string `json:"cloud_url,omitempty"`
	Authenticated   bool   `json:"authenticated,omitempty"`
	UserLogin       string `json:"user_login,omitempty"`
	GeneratedAt     string `json:"generated_at,omitempty"`
	SourceUpdatedAt string `json:"source_updated_at,omitempty"`
	StoreCount      int    `json:"store_count,omitempty"`
	LatestCount     int    `json:"latest_count,omitempty"`
	RollupCount     int    `json:"rollup_count,omitempty"`
	LastError       string `json:"last_error,omitempty"`
	Message         string `json:"message,omitempty"`
}

type QueueBaselineStore struct {
	StoreID          int      `json:"store_id"`
	Name             string   `json:"name"`
	City             string   `json:"city,omitempty"`
	Area             string   `json:"area,omitempty"`
	Address          string   `json:"address,omitempty"`
	Latitude         *float64 `json:"latitude,omitempty"`
	Longitude        *float64 `json:"longitude,omitempty"`
	OpenDate         string   `json:"open_date,omitempty"`
	TablesCapacity   int      `json:"tables_capacity,omitempty"`
	CountersCapacity int      `json:"counters_capacity,omitempty"`
	LastSeenAt       string   `json:"last_seen_at,omitempty"`
}

type QueueBaselineLatest struct {
	StoreID           int    `json:"store_id"`
	CollectedAt       string `json:"collected_at"`
	Name              string `json:"name"`
	City              string `json:"city,omitempty"`
	Area              string `json:"area,omitempty"`
	WaitMinutes       int    `json:"wait_minutes"`
	GroupQueuesCount  int    `json:"group_queues_count"`
	StoreStatus       string `json:"store_status"`
	NetTicketStatus   string `json:"net_ticket_status"`
	ReservationStatus string `json:"reservation_status,omitempty"`
	OnlineOpen        bool   `json:"online_open"`
	WaitTimeCounter   int    `json:"wait_time_counter,omitempty"`
	WaitTimeCap       int    `json:"wait_time_cap,omitempty"`
	DisplayCalledNo   int    `json:"display_called_no,omitempty"`
	GroupQueuesJSON   string `json:"group_queues_json,omitempty"`
}

// QueueBaselineRollup 是某店、某日型、某时段(bucket)的聚合基准。
// 字段语义：
//   - SampleCount：该聚合包含的快照数，越大越可信；下游做加权平均时用它当权重。
//   - OpenRate/OnlineOpenRate/BusyRate：开门率/在线取号开放率/繁忙率(有排队占比)。
//   - WaitTypicalMinutes/WaitSafeMinutes：等待分钟的典型值(约 P50)和安全值(偏保守，约 P80)。
//   - QueueGroupsTypical/Safe：同上，但单位是排队桌数。
//   - CalledNoSlow/Typical/Fast：叫号速度的三档分位(慢/典型/快)，反映叫号推进节奏。
//   - Confidence：聚合可信度(high/medium/low)，由样本量决定。
//
// 指针字段为 nil 表示该店该时段缺这类样本（如叫号字段在旧 schema 里不存在）。
type QueueBaselineRollup struct {
	StoreID            int      `json:"store_id"`
	DateType           string   `json:"date_type"`
	Weekday            int      `json:"weekday"`
	TimeBucket         string   `json:"time_bucket"`
	SampleCount        int      `json:"sample_count"`
	OpenRate           float64  `json:"open_rate"`
	OnlineOpenRate     float64  `json:"online_open_rate"`
	BusyRate           float64  `json:"busy_rate"`
	WaitTypicalMinutes *float64 `json:"wait_typical_minutes,omitempty"`
	WaitSafeMinutes    *float64 `json:"wait_safe_minutes,omitempty"`
	WaitMaxMinutes     int      `json:"wait_max_minutes,omitempty"`
	QueueGroupsTypical *float64 `json:"queue_groups_typical,omitempty"`
	QueueGroupsSafe    *float64 `json:"queue_groups_safe,omitempty"`
	CalledSampleCount  int      `json:"called_sample_count,omitempty"`
	CalledNoSlow       *float64 `json:"called_no_slow,omitempty"`
	CalledNoTypical    *float64 `json:"called_no_typical,omitempty"`
	CalledNoFast       *float64 `json:"called_no_fast,omitempty"`
	Confidence         string   `json:"confidence"`
	UpdatedAt          string   `json:"updated_at"`
}

// QueueBaselineConfig 控制本地公开排队基准采集：周期性快照用户选定门店的
// 实时等位/状态并落盘。全国数据只从线上 Turso 读取，本地不再全量落全国。
type QueueBaselineConfig struct {
	Enabled             bool     `json:"enabled"`
	IntervalMinutes     int      `json:"interval_minutes"`
	StoreIDs            []string `json:"store_ids,omitempty"`
	UsePreferenceStores bool     `json:"use_preference_stores"`
}

func queueBaselinePath() string { return filepath.Join(AppDirPath(), queueBaselineFile) }

func queueBaselineRecordsPath() string {
	return filepath.Join(AppDirPath(), queueBaselineRecordsFile)
}

func NormalizeQueueBaselineConfig(cfg QueueBaselineConfig) QueueBaselineConfig {
	if cfg.IntervalMinutes <= 0 {
		cfg.IntervalMinutes = queueBaselineDefaultMinutes
	}
	if cfg.IntervalMinutes < 1 {
		cfg.IntervalMinutes = 1
	}
	if cfg.IntervalMinutes > 1440 {
		cfg.IntervalMinutes = 1440
	}
	cfg.StoreIDs = UniqueNonEmptyStrings(cfg.StoreIDs)
	if len(cfg.StoreIDs) == 0 && !cfg.UsePreferenceStores {
		cfg.UsePreferenceStores = true
	}
	return cfg
}

func LoadQueueBaselineConfig() QueueBaselineConfig {
	// 默认开启：走公开接口、不需要通行证、只采集常用门店并写本机。
	// 这是“只读用户第一次打开就有曲线可看”的前提（用户可在「现在去吃」高级区关闭）。
	def := QueueBaselineConfig{Enabled: true, IntervalMinutes: queueBaselineDefaultMinutes, UsePreferenceStores: true}
	data, err := os.ReadFile(queueBaselinePath())
	if err != nil {
		return def
	}
	var cfg QueueBaselineConfig
	if json.Unmarshal(data, &cfg) != nil {
		return def
	}
	return NormalizeQueueBaselineConfig(cfg)
}

func SaveQueueBaselineConfig(cfg QueueBaselineConfig) error {
	cfg = NormalizeQueueBaselineConfig(cfg)
	os.MkdirAll(AppDirPath(), 0o755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(queueBaselinePath(), data, 0o600)
}

type QueueBaselineCollector struct {
	mu      sync.Mutex
	running bool
	lastAt  time.Time
}

var queueBaselineCollector = &QueueBaselineCollector{}

// Start 后台周期采集；按 base cadence 轮询，按配置间隔实际落盘，支持热改开关/间隔。
func (c *QueueBaselineCollector) Start(ctx context.Context) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()

	go func() {
		t := time.NewTicker(queueBaselineTickSeconds * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				c.tick(ctx)
			}
		}
	}()
}

// tick 以固定 base cadence(30s) 轮询，但只在达到配置间隔时才真正采集，支持运行时热改开关/间隔。
// 取数失败时不更新 lastAt，这样下一个 30s 周期会立即重试，而不是等满一个间隔——
// 对「间隔设很大但首次失败」的场景更友好。
func (c *QueueBaselineCollector) tick(ctx context.Context) {
	cfg := LoadQueueBaselineConfig()
	if !cfg.Enabled {
		return
	}
	c.mu.Lock()
	// due：从未采过(lastAt 零值)或距上次采集已达配置间隔。
	due := c.lastAt.IsZero() || time.Since(c.lastAt) >= time.Duration(cfg.IntervalMinutes)*time.Minute
	c.mu.Unlock()
	if !due {
		return
	}
	if _, err := collectQueueBaselineWithConfig(ctx, cfg); err != nil {
		return // 取数失败下个周期再试，不更新 lastAt
	}
	c.mu.Lock()
	c.lastAt = time.Now()
	c.mu.Unlock()
}

// collectQueueBaseline 拉取本地选定门店快照并落盘，返回写入条数。
func collectQueueBaseline(ctx context.Context) (int, error) {
	return collectQueueBaselineWithConfig(ctx, LoadQueueBaselineConfig())
}

// collectQueueBaselineWithConfig 拉取本地选定门店的公开快照并落盘，返回写入条数。
// 单店失败不中断整体（记 lastErr、continue），只有全部失败才返回错误。
// 注意双重写入：每店快照既写基准记录(queue_baseline.jsonl)，又顺手写一份排队观测
// (queue_observations.jsonl)——因为公开快照含当前叫号，能让叫号/到店预测在无通行证时也积累曲线。
func collectQueueBaselineWithConfig(ctx context.Context, cfg QueueBaselineConfig) (int, error) {
	storeIDs := queueBaselineStoreIDs(cfg)
	if len(storeIDs) == 0 {
		return 0, fmt.Errorf("暂无本地基准门店")
	}
	client := NewQueueLiveClient()
	now := time.Now().Format(time.RFC3339)
	records := make([]QueueBaselineRecord, 0, len(storeIDs))
	var lastErr error
	for _, storeID := range storeIDs {
		s, err := client.GetStore(ctx, storeID)
		if err != nil {
			lastErr = err
			continue
		}
		records = append(records, queueBaselineRecordFromStore(s, now))
		// 公开快照同样包含当前叫号：顺手写入排队观测，让叫号预测和到店预测
		// 无需通行证也能积累本机曲线（与凭证态采样写入同一份观测文件）。
		_ = appendQueueObservation(queueObservationFromLiveStore(s, time.Now()))
	}
	if len(records) == 0 && lastErr != nil {
		return 0, lastErr
	}
	if err := appendQueueBaselineRecords(records); err != nil {
		return 0, err
	}
	return len(records), nil
}

func queueBaselineStoreIDs(cfg QueueBaselineConfig) []string {
	cfg = NormalizeQueueBaselineConfig(cfg)
	if len(cfg.StoreIDs) > 0 {
		return cfg.StoreIDs
	}
	if !cfg.UsePreferenceStores {
		return nil
	}
	prefs := LoadPreferences()
	if len(prefs.SelectedStores) > 0 {
		return UniqueNonEmptyStrings(prefs.SelectedStores)
	}
	return UniqueNonEmptyStrings(prefs.StorePriority)
}

func queueBaselineRecordFromStore(s QueueLiveStore, collectedAt string) QueueBaselineRecord {
	groupQueuesJSON := ""
	if data, err := json.Marshal(s.GroupQueues); err == nil && string(data) != "{}" {
		groupQueuesJSON = string(data)
	}
	return QueueBaselineRecord{
		CollectedAt:       collectedAt,
		StoreID:           s.ID,
		Name:              s.Name,
		City:              s.NameKana,
		Area:              s.Area,
		WaitMinutes:       s.Wait,
		GroupQueuesCount:  s.GroupQueuesCount,
		StoreStatus:       s.StoreStatus,
		NetTicketStatus:   s.NetTicketStatus,
		ReservationStatus: s.ReservationStatus,
		OnlineOpen:        queueLiveStoreOnlineOpen(s),
		WaitTimeCounter:   s.WaitTimeCounter,
		WaitTimeCap:       s.WaitTimeCap,
		DisplayCalledNo:   s.GroupQueues.CurrentCalledNo(),
		GroupQueuesJSON:   groupQueuesJSON,
		UpdatedAt:         collectedAt,
		SourceEndpoint:    queueSourceEndpointStoreByID,
		APIProfileVersion: queueAPIProfileStoreDetailV1,
	}
}

// queueBaselineRecordsMu 串行化对 queue_baseline.jsonl 的追加写，避免并发写交错。
var queueBaselineRecordsMu sync.Mutex

// appendQueueBaselineRecords 一次性追加一批基准记录到 queue_baseline.jsonl。
func appendQueueBaselineRecords(records []QueueBaselineRecord) error {
	queueBaselineRecordsMu.Lock()
	defer queueBaselineRecordsMu.Unlock()

	if len(records) == 0 {
		return nil
	}
	if err := os.MkdirAll(AppDirPath(), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(queueBaselineRecordsPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_ = f.Chmod(0o600)
	enc := json.NewEncoder(f)
	for i := range records {
		normalizeQueueBaselineRecordForWrite(&records[i])
		if err := enc.Encode(records[i]); err != nil {
			return err
		}
	}
	// 周期性裁剪：基准采集每店每 5 分钟追加一批，需要与 observations 同样的行数上限。
	queueBaselineWriteCounter++
	if queueBaselineWriteCounter%queueJSONLTrimInterval == 0 {
		trimJSONLFileLocked(queueBaselineRecordsPath(), queueBaselineMaxLines, time.Now())
	}
	return nil
}

func normalizeQueueBaselineRecordForWrite(record *QueueBaselineRecord) {
	if record == nil {
		return
	}
	if record.CollectedAt == "" {
		if record.Timestamp != "" {
			record.CollectedAt = record.Timestamp
		} else {
			record.CollectedAt = time.Now().Format(time.RFC3339)
		}
	}
	if record.UpdatedAt == "" {
		record.UpdatedAt = record.CollectedAt
	}
	if record.WaitMinutes == 0 && record.Wait > 0 {
		record.WaitMinutes = record.Wait
	}
	if record.SourceEndpoint == "" {
		record.SourceEndpoint = queueSourceEndpointStores
	}
	if record.APIProfileVersion == "" {
		record.APIProfileVersion = queueAPIProfilePublicV1
	}
	record.Timestamp = ""
	record.Wait = 0
}

func handleQueueBaseline(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := LoadQueueBaselineConfig()
		writeJSON(w, map[string]any{
			"enabled":               cfg.Enabled,
			"interval_minutes":      cfg.IntervalMinutes,
			"store_ids":             cfg.StoreIDs,
			"use_preference_stores": cfg.UsePreferenceStores,
			"effective_store_ids":   queueBaselineStoreIDs(cfg),
			"remote":                queueBaselineRemoteStatus(),
		})
	case http.MethodPost:
		var body QueueBaselineConfig
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := SaveQueueBaselineConfig(body); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, LoadQueueBaselineConfig())
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST only")
	}
}
