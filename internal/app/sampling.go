package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/platform"

import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

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

// SamplingRunOptions 控制单次采样的行为。
// IgnoreActiveWindow：忽略时间窗限制（手动「立刻采一次」时用）。
// UseProcessLock：抢进程级独占锁，防止手动触发与后台守护同时采样。
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

// SlotSampler 是采样后台循环的状态机，单例（sampler）。
// 并发模型：mu 保护 cancel/done/generation/state；runMu 保证同一时刻只有一轮采样在跑（TryLock 失败即跳过）。
// generation 是循环的代次，Stop/Restart 时旧循环的代次对不上，defer 清理就不会误清新循环的状态。
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
	return filepath.Join(AppDirPath(), "sampling.json")
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
	if err := os.MkdirAll(AppDirPath(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(samplingConfigPath(), data, 0o600)
}

// NormalizeSamplingConfig 钳制并补全采样配置：
//
//	间隔区间 [60, 24h]（过密会刷接口、过疏等于没采）；时间窗解析失败回退默认 10:00-22:00。
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
	if ParseTimeSeconds(cfg.ActiveStart) < 0 {
		cfg.ActiveStart = defaultSamplingStart
	}
	if ParseTimeSeconds(cfg.ActiveEnd) < 0 {
		cfg.ActiveEnd = defaultSamplingEnd
	}
	cfg.ActiveStart = NormalizeTimeStr(cfg.ActiveStart)
	cfg.ActiveEnd = NormalizeTimeStr(cfg.ActiveEnd)
	cfg.StoreIDs = UniqueNonEmptyStrings(cfg.StoreIDs)
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

// StartIfAuto 在进程启动时被调用：仅当配置同时 Enabled+AutoStart 才自启动。
// 关键去重：若已有独立守护进程（PID 文件存在且活着）在跑，就不在本进程再起一份，避免双开争抢采样。
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

// startWithConfig 启动后台采样循环。幂等：已运行直接返回 nil。
// 两段式上锁是为了避免竞态——先抢进程级独占锁（防止 web 和守护进程同时采样），
// 抢到后再在 mu 下二次检查 cancel 是否已被并发路径置上，是则放弃本次启动（释放刚拿的资源）。
// generation 在这里自增并传给 loop，loop 退出时只在代次匹配时才清 cancel/done/状态。
func (s *SlotSampler) startWithConfig(parent context.Context, cfg SamplingConfig) error {
	cfg = NormalizeSamplingConfig(cfg)
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	// 进程级独占锁：保证同一台机器同一时刻只有一个采样进程在跑。
	lock, err := acquireProcessLock(samplingLockFileName)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	s.mu.Lock()
	// 二次检查：在拿进程锁的间隙，可能已有别的路径启动了循环。
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

// Restart 先同步等旧循环退出再启动新循环；若旧循环在 timeout 内没退出则报错让调用方重试。
// 这里必须用 stopAndWait 而非 Stop（fire-and-forget），否则新旧循环可能短暂并存争抢 runMu/进程锁。
func (s *SlotSampler) Restart(parent context.Context, cfg SamplingConfig) error {
	if !s.stopAndWait(2 * time.Second) {
		return fmt.Errorf("采样正在停止，请稍后重试")
	}
	return s.startWithConfig(parent, cfg)
}

// stopAndWait 发出 cancel 后最多等 timeout 让 loop 退出。
// 返回值表示「是否在超时前真正停掉」。注意：超时只放弃等待，loop 仍可能在后台继续跑（依赖 loop defer 自行清理）。
// 没有运行中的循环（cancel==nil）时直接置 idle 并返回 true。
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

// loop 是后台采样主循环。每个周期：重新读配置（支持运行中改配置热生效）-> 算下次等待时长 ->
// 设定 NextRunAt -> 可中断睡眠 -> 再次校验配置 -> 跑一轮采样。
// defer 里按 generation 清理：只有当代次仍匹配（没被 Restart 抢占）才清 cancel/done 并把状态从 running 收回 idle。
// done 在最后 close，通知 stopAndWait 等待者循环已退出。
func (s *SlotSampler) loop(ctx context.Context, cfg SamplingConfig, generation int, lock *processLock, done chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			// 后台采样循环崩溃不能拖垮整个进程；记日志后仍走下面的收尾（释放锁、回 idle、广播），避免泄漏。
			LogMessage(time.Now(), "后台采样循环发生 panic 已恢复："+fmt.Sprint(r))
		}
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
		// 每轮重新读配置：用户在 UI/CLI 改配置后无需重启即可生效（关掉 Enabled 即可让循环自然退出）。
		fileCfg := LoadSamplingConfig()
		if !fileCfg.Enabled {
			return
		}
		cfg = fileCfg
		wait := samplingWaitDuration(cfg, time.Now())
		// 首轮若已在采样窗口内，跳过等待立刻采一次，保证启动后能尽快出数据。
		if first && samplingInActiveWindow(cfg, time.Now()) {
			wait = 0
		}
		first = false
		s.setNextRun(time.Now().Add(wait))
		if !sleepContext(ctx, wait) {
			return // 被 cancel（Stop/Restart）唤醒，退出循环。
		}
		// 睡醒后再读一次配置：睡眠期间配置可能被关掉。
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

// runOnce 执行一轮采样。返回值带 Skipped/SkipReason：任何一个前置条件不满足都会直接跳过本轮（不算失败）。
// 跳过的层次（短路顺序）：单实例重入(runMu) -> 进程锁(仅手动触发) -> 时间窗 -> 主流程冲突 -> 凭证 -> 门店列表。
// 逐门店循环里也会在每家门店前复查 samplingBlockedReason：因为采样耗时，期间用户可能开始抢号，
// 一旦检测到主流程/抢号进程启动就立刻中止剩余门店，避免采样请求和抢号请求撞车。
func (s *SlotSampler) runOnce(ctx context.Context, cfg SamplingConfig, opts SamplingRunOptions) SamplingRunResult {
	cfg = NormalizeSamplingConfig(cfg)
	started := time.Now()
	result := SamplingRunResult{
		StartedAt: started.Format(time.RFC3339),
		Config:    cfg,
	}

	// 单实例重入保护：后台循环和手动 RunOnceNow 可能并发，TryLock 失败即跳过，不阻塞。
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

	tokens, err := LoadLocalConfig()
	if err != nil {
		result.Skipped = true
		result.SkipReason = "暂无凭证参数，请先完成参数捕获"
		result.FinishedAt = time.Now().Format(time.RFC3339)
		return result
	}
	if err := tokens.ValidateForQuery(); err != nil {
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

	settings := tokens.ToSettingsWithPrefs(prefs)
	settings.StoreIDs = storeIDs
	client := NewClient(settings)
	reg := GetStoreRegistry()

	for _, storeID := range storeIDs {
		storeResult := SamplingStoreRunResult{StoreID: storeID}
		storeResult.StoreName = reg.DisplayName(storeID, "")
		if storeResult.StoreName == "" {
			storeResult.StoreName = storeID
		}
		// 每家门店前复查阻塞条件：采样是批量请求，期间主流程/抢号可能被用户启动，
		// 此时必须立刻停手，避免采样与抢号的凭证请求互相干扰。
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
		// 排队快照：用凭证态 getStoreById（含 groupQueues=当前叫号），写入观测并评估叫号提醒。
		if storeInfo, err := client.GetStoreInfo(ctx, storeID); err != nil {
			storeResult.QueueError = err.Error()
			LogMessage(time.Now(), fmt.Sprintf("采样排队快照获取失败，门店 %s: %v", storeID, err))
		} else if observation, ok := queueObservationFromStoreInfo(storeID, storeInfo, time.Now()); ok {
			if err := appendQueueObservation(observation); err != nil {
				storeResult.QueueError = err.Error()
				LogMessage(time.Now(), fmt.Sprintf("采样排队快照保存失败，门店 %s: %v", storeID, err))
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

// samplingStoreIDs 按优先级解析本轮要采样的门店列表：
// 显式配置 StoreIDs > 偏好门店(UsePreferenceStores) > 凭证里捕获到的 StoreIDs（兜底）。
func samplingStoreIDs(cfg SamplingConfig, tokens *CapturedTokens, prefs UserPreferences) []string {
	if len(cfg.StoreIDs) > 0 {
		return cfg.StoreIDs
	}
	if cfg.UsePreferenceStores && len(prefs.SelectedStores) > 0 {
		return prefs.SelectedStores
	}
	tokens.Lock()
	defer tokens.Unlock()
	return append([]string(nil), tokens.StoreIDs...)
}

// samplingWaitDuration 决定本周期 loop 应该睡多久：
// 在窗口内就按 IntervalSeconds 间隔采样；不在窗口内则睡到下一个窗口起点（窗口外不采样）。
func samplingWaitDuration(cfg SamplingConfig, now time.Time) time.Duration {
	if !samplingInActiveWindow(cfg, now) {
		return time.Until(nextSamplingWindowStart(cfg, now))
	}
	return time.Duration(cfg.IntervalSeconds) * time.Second
}

// samplingInActiveWindow 判断当前时刻是否在配置的活跃窗口 [ActiveStart, ActiveEnd) 内。
// 边界：start==end 或任一解析失败视为「全天活跃」（不限制）；start<end 是普通同日窗口；
// start>end 是跨午夜窗口（如 22:00-04:00），此时「current>=start || current<end」即可命中。
// 区间为左闭右开：刚好等于 end 不算在内，避免窗口边界与下一轮等待计算打架。
func samplingInActiveWindow(cfg SamplingConfig, now time.Time) bool {
	start := ParseTimeSeconds(cfg.ActiveStart)
	end := ParseTimeSeconds(cfg.ActiveEnd)
	// 窗口配置（如 100000-220000）是 CST 门店营业时间，必须按 CST 解释当前时刻，
	// 否则 TZ=UTC/海外机会整体偏移 8 小时（采到打烊数据或一直空转）。
	now = now.In(SushiroTimezone)
	current := now.Hour()*3600 + now.Minute()*60 + now.Second()
	if start < 0 || end < 0 || start == end {
		return true
	}
	if start < end {
		return current >= start && current < end
	}
	return current >= start || current < end
}

// nextSamplingWindowStart 算出下一个窗口起点的绝对时刻。
// 把今天的 start 时刻构造出来，若已过（<=now）则加一天，得到下一次开窗时间。
func nextSamplingWindowStart(cfg SamplingConfig, now time.Time) time.Time {
	start := ParseTimeSeconds(cfg.ActiveStart)
	if start < 0 {
		return now
	}
	// 窗口时刻按 CST 构造，与 samplingInActiveWindow 的时区解释一致。
	now = now.In(SushiroTimezone)
	hour := start / 3600
	minute := (start % 3600) / 60
	second := start % 60
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, 0, SushiroTimezone)
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

// isMainFlowRunning 判断本进程的抢号引擎是否正处于占用凭证的状态（捕获/订座/狙击）。
// 这些状态会跟手机端争抢凭证，所以采样必须让路。
func isMainFlowRunning() bool {
	switch engine.GetState().Status {
	case EngineCapturing, EngineBooking, EngineSniping:
		return true
	default:
		return false
	}
}

// samplingBlockedReason 返回当前不能采样的原因（空串表示可以采）。
// 检查顺序即优先级，任一命中即返回。覆盖四类冲突：
//  1. externalMainFlowActive：另一个 sushiro 主进程正在抢号（跨进程）。
//  2. netTicketIssuedToday：今天已取到排队号——继续采样会顶掉手机端的排队号视图。
//  3. 本进程引擎在抢号。
//  4. 后台抢号守护进程在跑。
func samplingBlockedReason() string {
	if active, status := externalMainFlowActive(); active {
		if status == "" {
			status = "running"
		}
		return "另一个主流程正在运行 (" + status + ")，已跳过本轮采样"
	}
	if netTicketIssuedToday(time.Now()) {
		return "今天已经取到排队号，已停止本轮采样，避免影响手机端查看排队信息"
	}
	if isMainFlowRunning() {
		return "主流程正在运行，已跳过本轮采样"
	}
	if isRunning() {
		return "后台抢号进程正在运行，已跳过本轮采样"
	}
	return ""
}

// netTicketIssuedToday 判断今天是否已经成功取到排队号（success/issued_unknown 且今天触发过）。
// 用于采样让路：取号后官方会把这条排队记录挂在账号上，继续采样会改写手机端看到的排队视图。
func netTicketIssuedToday(now time.Time) bool {
	if now.IsZero() {
		now = time.Now()
	}
	plan := LoadNetTicketPlan()
	if plan.Status != "success" && plan.Status != "issued_unknown" {
		return false
	}
	return netTicketPlanFiredOn(plan, now)
}

// pauseSamplingForMainFlow 在主流程（抢号）即将启动前紧急叫停采样，三段式尽力停：
//  1. 先停本进程的 sampler 循环；
//  2. 再尝试停独立的采样守护进程（PID 文件指向的那个）；
//  3. 若守护进程不在 PID 文件里、但进程锁还被别的 PID 持着，直接 kill 那个持锁进程。
//
// 注意最后一段的 holder != os.Getpid() 自我保护：锁若恰好被自己持有就不自杀。
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
	return fmt.Sprintf("每 %d 秒，%s-%s，门店: %s", cfg.IntervalSeconds, FormatCompactTime(cfg.ActiveStart), FormatCompactTime(cfg.ActiveEnd), stores)
}
