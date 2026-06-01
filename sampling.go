package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultSamplingIntervalSeconds = 300
	defaultSamplingStart           = "100000"
	defaultSamplingEnd             = "220000"
	samplingStatusIdle             = "idle"
	samplingStatusRunning          = "running"
)

type SamplingConfig struct {
	Enabled             bool     `json:"enabled"`
	AutoStart           bool     `json:"auto_start"`
	IntervalSeconds     int      `json:"interval_seconds"`
	ActiveStart         string   `json:"active_start"`
	ActiveEnd           string   `json:"active_end"`
	StoreIDs            []string `json:"store_ids"`
	UsePreferenceStores bool     `json:"use_preference_stores"`
}

type SamplingState struct {
	Status         string   `json:"status"`
	Message        string   `json:"message"`
	Running        bool     `json:"running"`
	Enabled        bool     `json:"enabled"`
	AutoStart      bool     `json:"auto_start"`
	StoreIDs       []string `json:"store_ids"`
	Interval       int      `json:"interval_seconds"`
	ActiveStart    string   `json:"active_start"`
	ActiveEnd      string   `json:"active_end"`
	LastRunAt      string   `json:"last_run_at,omitempty"`
	NextRunAt      string   `json:"next_run_at,omitempty"`
	LastError      string   `json:"last_error,omitempty"`
	SampleRuns     int      `json:"sample_runs"`
	Snapshots      int      `json:"snapshots"`
	QueueSnapshots int      `json:"queue_snapshots"`
	StoreErrors    int      `json:"store_errors"`
	LastStoreIDs   []string `json:"last_store_ids,omitempty"`
}

type SamplingRunResult struct {
	StartedAt      string                   `json:"started_at"`
	FinishedAt     string                   `json:"finished_at"`
	Stores         []SamplingStoreRunResult `json:"stores"`
	Snapshots      int                      `json:"snapshots"`
	QueueSnapshots int                      `json:"queue_snapshots"`
	StoreErrors    int                      `json:"store_errors"`
	Skipped        bool                     `json:"skipped"`
	SkipReason     string                   `json:"skip_reason,omitempty"`
	Config         SamplingConfig           `json:"config"`
	Diagnostics    map[string]any           `json:"diagnostics,omitempty"`
}

type SamplingRunOptions struct {
	IgnoreActiveWindow bool
	UseProcessLock     bool
}

type SamplingStoreRunResult struct {
	StoreID         string `json:"store_id"`
	StoreName       string `json:"store_name,omitempty"`
	Slots           int    `json:"slots"`
	QueueObserved   bool   `json:"queue_observed,omitempty"`
	QueueWaitGroups int    `json:"queue_wait_groups,omitempty"`
	QueueStatus     string `json:"queue_status,omitempty"`
	QueueError      string `json:"queue_error,omitempty"`
	Error           string `json:"error,omitempty"`
}

type SlotSampler struct {
	mu         sync.Mutex
	runMu      sync.Mutex
	cancel     context.CancelFunc
	done       chan struct{}
	generation int
	state      SamplingState
}

var sampler = &SlotSampler{state: SamplingState{Status: samplingStatusIdle, Message: "未启动"}}

func samplingConfigPath() string {
	return filepath.Join(appDirPath(), "sampling.json")
}

func defaultSamplingConfig() SamplingConfig {
	return SamplingConfig{
		Enabled:             false,
		AutoStart:           false,
		IntervalSeconds:     defaultSamplingIntervalSeconds,
		ActiveStart:         defaultSamplingStart,
		ActiveEnd:           defaultSamplingEnd,
		UsePreferenceStores: true,
	}
}

func LoadSamplingConfig() SamplingConfig {
	data, err := os.ReadFile(samplingConfigPath())
	if err != nil {
		return defaultSamplingConfig()
	}
	var cfg SamplingConfig
	if json.Unmarshal(data, &cfg) != nil {
		return defaultSamplingConfig()
	}
	return NormalizeSamplingConfig(cfg)
}

func SaveSamplingConfig(cfg SamplingConfig) error {
	cfg = NormalizeSamplingConfig(cfg)
	if err := os.MkdirAll(appDirPath(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(samplingConfigPath(), data, 0o600)
}

func NormalizeSamplingConfig(cfg SamplingConfig) SamplingConfig {
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = defaultSamplingIntervalSeconds
	}
	if cfg.IntervalSeconds < 60 {
		cfg.IntervalSeconds = 60
	}
	if cfg.IntervalSeconds > 24*3600 {
		cfg.IntervalSeconds = 24 * 3600
	}
	if parseTimeSeconds(cfg.ActiveStart) < 0 {
		cfg.ActiveStart = defaultSamplingStart
	}
	if parseTimeSeconds(cfg.ActiveEnd) < 0 {
		cfg.ActiveEnd = defaultSamplingEnd
	}
	cfg.ActiveStart = normalizeTimeStr(cfg.ActiveStart)
	cfg.ActiveEnd = normalizeTimeStr(cfg.ActiveEnd)
	cfg.StoreIDs = uniqueNonEmptyStrings(cfg.StoreIDs)
	return cfg
}

func (s *SlotSampler) GetState() SamplingState {
	cfg := LoadSamplingConfig()
	s.mu.Lock()
	state := s.state
	state.Running = s.cancel != nil
	s.mu.Unlock()
	state.Enabled = cfg.Enabled
	state.AutoStart = cfg.AutoStart
	state.Interval = cfg.IntervalSeconds
	state.ActiveStart = cfg.ActiveStart
	state.ActiveEnd = cfg.ActiveEnd
	state.StoreIDs = cfg.StoreIDs
	if state.Status == "" {
		state.Status = samplingStatusIdle
	}
	return state
}

func (s *SlotSampler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancel != nil
}

func (s *SlotSampler) Start(parent context.Context) error {
	cfg := LoadSamplingConfig()
	cfg.Enabled = true
	if err := SaveSamplingConfig(cfg); err != nil {
		return err
	}
	return s.startWithConfig(parent, cfg)
}

func (s *SlotSampler) StartIfAuto(parent context.Context) {
	cfg := LoadSamplingConfig()
	if cfg.Enabled && cfg.AutoStart {
		if isSamplingDaemonRunning() {
			s.setState(samplingStatusIdle, "采样守护进程已运行")
			return
		}
		if err := s.startWithConfig(parent, cfg); err != nil {
			s.setState(samplingStatusIdle, "采样自启动失败: "+err.Error())
		}
	}
}

func (s *SlotSampler) startWithConfig(parent context.Context, cfg SamplingConfig) error {
	cfg = NormalizeSamplingConfig(cfg)
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	lock, err := acquireProcessLock(samplingLockFileName)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		cancel()
		lock.Release()
		return nil
	}
	s.generation++
	generation := s.generation
	s.cancel = cancel
	s.done = done
	s.state.Status = samplingStatusRunning
	s.state.Message = "后台采样已启动"
	s.state.LastError = ""
	s.state.Interval = cfg.IntervalSeconds
	s.state.ActiveStart = cfg.ActiveStart
	s.state.ActiveEnd = cfg.ActiveEnd
	s.mu.Unlock()
	s.publish()

	go s.loop(ctx, cfg, generation, lock, done)
	return nil
}

func (s *SlotSampler) Stop() {
	s.stopAndWait(2 * time.Second)
}

func (s *SlotSampler) Restart(parent context.Context, cfg SamplingConfig) error {
	if !s.stopAndWait(2 * time.Second) {
		return fmt.Errorf("采样正在停止，请稍后重试")
	}
	return s.startWithConfig(parent, cfg)
}

func (s *SlotSampler) stopAndWait(timeout time.Duration) bool {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	if cancel == nil {
		s.state.Status = samplingStatusIdle
		s.state.Message = "后台采样已停止"
		s.state.NextRunAt = ""
		s.mu.Unlock()
		s.publish()
		return true
	}
	s.state.Message = "正在停止后台采样"
	s.state.NextRunAt = ""
	s.mu.Unlock()
	cancel()

	stopped := true
	if timeout > 0 && done != nil {
		select {
		case <-done:
		case <-time.After(timeout):
			stopped = false
		}
	}
	s.publish()
	return stopped
}

func (s *SlotSampler) loop(ctx context.Context, cfg SamplingConfig, generation int, lock *processLock, done chan struct{}) {
	defer func() {
		lock.Release()
		s.mu.Lock()
		if s.generation == generation {
			s.cancel = nil
			s.done = nil
		}
		if s.generation == generation && s.state.Status == samplingStatusRunning {
			s.state.Status = samplingStatusIdle
			s.state.Message = "后台采样已结束"
		}
		s.mu.Unlock()
		close(done)
		s.publish()
	}()

	first := true
	for {
		fileCfg := LoadSamplingConfig()
		if !fileCfg.Enabled {
			return
		}
		cfg = fileCfg
		wait := samplingWaitDuration(cfg, time.Now())
		if first && samplingInActiveWindow(cfg, time.Now()) {
			wait = 0
		}
		first = false
		s.setNextRun(time.Now().Add(wait))
		if !sleepContext(ctx, wait) {
			return
		}
		fileCfg = LoadSamplingConfig()
		if !fileCfg.Enabled {
			return
		}
		cfg = fileCfg
		result := s.RunOnce(ctx, cfg)
		s.applyRunResult(result)
	}
}

func (s *SlotSampler) RunOnce(ctx context.Context, cfg SamplingConfig) SamplingRunResult {
	return s.runOnce(ctx, cfg, SamplingRunOptions{})
}

func (s *SlotSampler) RunOnceNow(ctx context.Context, cfg SamplingConfig) SamplingRunResult {
	return s.runOnce(ctx, cfg, SamplingRunOptions{IgnoreActiveWindow: true, UseProcessLock: true})
}

func (s *SlotSampler) runOnce(ctx context.Context, cfg SamplingConfig, opts SamplingRunOptions) SamplingRunResult {
	cfg = NormalizeSamplingConfig(cfg)
	started := time.Now()
	result := SamplingRunResult{
		StartedAt: started.Format(time.RFC3339),
		Config:    cfg,
	}

	if !s.runMu.TryLock() {
		result.Skipped = true
		result.SkipReason = "已有采样任务正在执行"
		result.FinishedAt = time.Now().Format(time.RFC3339)
		return result
	}
	defer s.runMu.Unlock()

	if opts.UseProcessLock {
		lock, err := acquireProcessLock(samplingLockFileName)
		if err != nil {
			result.Skipped = true
			result.SkipReason = err.Error()
			result.FinishedAt = time.Now().Format(time.RFC3339)
			return result
		}
		defer lock.Release()
	}

	if !opts.IgnoreActiveWindow && !samplingInActiveWindow(cfg, started) {
		result.Skipped = true
		result.SkipReason = "不在采样时间窗内"
		result.FinishedAt = time.Now().Format(time.RFC3339)
		return result
	}

	if reason := samplingBlockedReason(); reason != "" {
		result.Skipped = true
		result.SkipReason = reason
		result.FinishedAt = time.Now().Format(time.RFC3339)
		return result
	}

	tokens, err := loadLocalConfig()
	if err != nil {
		result.Skipped = true
		result.SkipReason = "暂无认证参数，请先完成参数捕获"
		result.FinishedAt = time.Now().Format(time.RFC3339)
		return result
	}
	if err := tokens.validateForQuery(); err != nil {
		result.Skipped = true
		result.SkipReason = err.Error()
		result.FinishedAt = time.Now().Format(time.RFC3339)
		return result
	}

	prefs := LoadPreferences()
	storeIDs := samplingStoreIDs(cfg, tokens, prefs)
	if len(storeIDs) == 0 {
		result.Skipped = true
		result.SkipReason = "暂无采样门店"
		result.FinishedAt = time.Now().Format(time.RFC3339)
		return result
	}

	settings := tokens.toSettingsWithPrefs(prefs)
	settings.StoreIDs = storeIDs
	client := NewClient(settings)
	reg := GetStoreRegistry()

	for _, storeID := range storeIDs {
		storeResult := SamplingStoreRunResult{StoreID: storeID}
		storeResult.StoreName = reg.DisplayName(storeID, "")
		if storeResult.StoreName == "" {
			storeResult.StoreName = storeID
		}
		if reason := samplingBlockedReason(); reason != "" {
			if result.Snapshots == 0 && len(result.Stores) == 0 {
				result.Skipped = true
				result.SkipReason = reason
			} else {
				storeResult.Error = reason
				result.StoreErrors++
				result.Stores = append(result.Stores, storeResult)
			}
			result.FinishedAt = time.Now().Format(time.RFC3339)
			return result
		}
		// 排队快照：用认证态 getStoreById（含 groupQueues=当前叫号），写入观测并评估叫号提醒。
		if storeInfo, err := client.GetStoreInfo(ctx, storeID); err != nil {
			storeResult.QueueError = err.Error()
			logMessage(time.Now(), fmt.Sprintf("采样排队快照获取失败，门店 %s: %v", storeID, err))
		} else if observation, ok := queueObservationFromStoreInfo(storeID, storeInfo, time.Now()); ok {
			if err := appendQueueObservation(observation); err != nil {
				storeResult.QueueError = err.Error()
				logMessage(time.Now(), fmt.Sprintf("采样排队快照保存失败，门店 %s: %v", storeID, err))
			} else {
				storeResult.QueueObserved = true
				storeResult.QueueWaitGroups = storeInfo.GroupQueuesCount
				storeResult.QueueStatus = storeInfo.StoreStatus
				result.QueueSnapshots++
				if storeResult.StoreName == storeID && strings.TrimSpace(storeInfo.Name) != "" {
					storeResult.StoreName = storeInfo.Name
				}
				evaluateQueueAlerts(ctx, observation, storeInfo.Name)
			}
		}
		slots, err := client.GetTimeslots(ctx, storeID)
		if err != nil {
			storeResult.Error = err.Error()
			result.StoreErrors++
			result.Stores = append(result.Stores, storeResult)
			continue
		}
		appendHistory(slots, storeID)
		storeResult.Slots = len(slots)
		result.Snapshots += len(slots)
		result.Stores = append(result.Stores, storeResult)
	}

	result.FinishedAt = time.Now().Format(time.RFC3339)
	return result
}

func (s *SlotSampler) applyRunResult(result SamplingRunResult) {
	s.mu.Lock()
	if result.Skipped {
		s.state.Message = result.SkipReason
		s.state.LastError = result.SkipReason
	} else {
		s.state.Message = fmt.Sprintf("已采样 %d 家门店，记录 %d 条时段、%d 条排队快照", len(result.Stores), result.Snapshots, result.QueueSnapshots)
		s.state.LastError = ""
		if result.StoreErrors > 0 {
			s.state.LastError = fmt.Sprintf("%d 家门店采样失败", result.StoreErrors)
		}
	}
	s.state.LastRunAt = result.FinishedAt
	s.state.SampleRuns++
	s.state.Snapshots += result.Snapshots
	s.state.QueueSnapshots += result.QueueSnapshots
	s.state.StoreErrors += result.StoreErrors
	s.state.LastStoreIDs = make([]string, 0, len(result.Stores))
	for _, store := range result.Stores {
		s.state.LastStoreIDs = append(s.state.LastStoreIDs, store.StoreID)
	}
	s.mu.Unlock()
	s.publish()
}

func (s *SlotSampler) setState(status, message string) {
	s.mu.Lock()
	s.state.Status = status
	s.state.Message = message
	s.mu.Unlock()
	s.publish()
}

func (s *SlotSampler) setNextRun(next time.Time) {
	s.mu.Lock()
	s.state.NextRunAt = next.Format(time.RFC3339)
	s.mu.Unlock()
	s.publish()
}

func (s *SlotSampler) publish() {
	bus.publish("sampling", mustJSON(s.GetState()))
}

func samplingStoreIDs(cfg SamplingConfig, tokens *CapturedTokens, prefs UserPreferences) []string {
	if len(cfg.StoreIDs) > 0 {
		return cfg.StoreIDs
	}
	if cfg.UsePreferenceStores && len(prefs.SelectedStores) > 0 {
		return prefs.SelectedStores
	}
	tokens.mu.Lock()
	defer tokens.mu.Unlock()
	return append([]string(nil), tokens.StoreIDs...)
}

func samplingWaitDuration(cfg SamplingConfig, now time.Time) time.Duration {
	if !samplingInActiveWindow(cfg, now) {
		return time.Until(nextSamplingWindowStart(cfg, now))
	}
	return time.Duration(cfg.IntervalSeconds) * time.Second
}

func samplingInActiveWindow(cfg SamplingConfig, now time.Time) bool {
	start := parseTimeSeconds(cfg.ActiveStart)
	end := parseTimeSeconds(cfg.ActiveEnd)
	current := now.Hour()*3600 + now.Minute()*60 + now.Second()
	if start < 0 || end < 0 || start == end {
		return true
	}
	if start < end {
		return current >= start && current < end
	}
	return current >= start || current < end
}

func nextSamplingWindowStart(cfg SamplingConfig, now time.Time) time.Time {
	start := parseTimeSeconds(cfg.ActiveStart)
	if start < 0 {
		return now
	}
	hour := start / 3600
	minute := (start % 3600) / 60
	second := start % 60
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func isMainFlowRunning() bool {
	switch engine.GetState().Status {
	case EngineCapturing, EngineBooking, EngineSniping:
		return true
	default:
		return false
	}
}

func samplingBlockedReason() string {
	if active, status := externalMainFlowActive(); active {
		if status == "" {
			status = "running"
		}
		return "另一个主流程正在运行 (" + status + ")，已跳过本轮采样"
	}
	if isMainFlowRunning() {
		return "主流程正在运行，已跳过本轮采样"
	}
	if isRunning() {
		return "后台抢号进程正在运行，已跳过本轮采样"
	}
	return ""
}

func pauseSamplingForMainFlow() {
	sampler.Stop()
	if stopped, _ := stopSamplingDaemon(); stopped {
		return
	}
	if holder, ok := processLockHolder(samplingLockFileName); ok && holder != os.Getpid() {
		_ = KillProcess(holder)
	}
}

func samplingSummary(cfg SamplingConfig) string {
	stores := "偏好门店"
	if len(cfg.StoreIDs) > 0 {
		stores = strings.Join(cfg.StoreIDs, ",")
	}
	return fmt.Sprintf("每 %d 秒，%s-%s，门店: %s", cfg.IntervalSeconds, formatCompactTime(cfg.ActiveStart), formatCompactTime(cfg.ActiveEnd), stores)
}
