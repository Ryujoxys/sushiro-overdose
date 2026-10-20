package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
	"github.com/Ryujoxys/sushiro-overdose/internal/platform"
)

// 采集熔断（界面与后台共享 queue_collection_state.json）：
//   - 没有可用连接或连接被拒（live_auth_required）：停下，直到本进程拿到与停下时不同的连接，
//     或 6 小时后再试一次。没有连接的进程拿到的客户端本来就不发请求。
//   - 其他失败：30 秒起翻倍退避，最长 15 分钟；成功一次清零。
//   - 持有可用连接的进程不受“无连接进程”写下的停止标记影响；它在记录时，无连接的进程让开，
//     不把“需要连接”写进共享状态。
const (
	queueCollectionBackoffBase   = 30 * time.Second
	queueCollectionBackoffMax    = 15 * time.Minute
	queueCollectionAuthRetry     = 6 * time.Hour
	queueCollectionHolderTimeout = 2 * time.Minute
)

type queueCollectionState struct {
	PublicQueueCollectionStatus
	PID         int    `json:"pid"`
	HeartbeatAt string `json:"heartbeat_at"`
	ConfigKey   string `json:"config_key"`

	ConsecutiveFailures int `json:"consecutive_failures,omitempty"`
	// SessionMark 是停下时所用连接的标记（连接的 captured_at）；空表示停下的进程没有连接。
	SessionMark string `json:"session_mark,omitempty"`
	// LiveHolderPID/At：最近一个持有可用连接的采集进程及其心跳。
	LiveHolderPID int    `json:"live_holder_pid,omitempty"`
	LiveHolderAt  string `json:"live_holder_at,omitempty"`
}

func queueCollectionStatePath() string {
	return filepath.Join(AppDirPath(), "queue_collection_state.json")
}

func loadQueueCollectionState() queueCollectionState {
	var state queueCollectionState
	data, _ := os.ReadFile(queueCollectionStatePath())
	_ = json.Unmarshal(data, &state)
	return state
}

// queueCollectionBackoff is 30s, 1m, 2m, 4m, 8m, then 15m for every later failure.
func queueCollectionBackoff(failures int) time.Duration {
	if failures < 1 {
		failures = 1
	}
	delay := queueCollectionBackoffBase
	for i := 1; i < failures && delay < queueCollectionBackoffMax; i++ {
		delay *= 2
	}
	if delay > queueCollectionBackoffMax {
		delay = queueCollectionBackoffMax
	}
	return delay
}

// otherLiveHolderActive: another process with a usable connection ticked recently.
func otherLiveHolderActive(state queueCollectionState, now time.Time) bool {
	if state.LiveHolderPID <= 0 || state.LiveHolderPID == os.Getpid() {
		return false
	}
	at, err := time.Parse(time.RFC3339, state.LiveHolderAt)
	return err == nil && !at.After(now) && now.Sub(at) < queueCollectionHolderTimeout && platform.IsProcessAlive(state.LiveHolderPID)
}

// queueCollectionMayAttempt decides whether a due round may send requests. mark is this
// process's connection ("" when it holds none).
func queueCollectionMayAttempt(state queueCollectionState, mark string, now time.Time) bool {
	if mark == "" && otherLiveHolderActive(state, now) {
		return false
	}
	next, err := time.Parse(time.RFC3339, state.NextAttemptAt)
	waited := err != nil || !now.Before(next)
	if state.WaitingForConnection {
		if mark != "" && mark != state.SessionMark {
			return true // 连上了新的连接，或停下的是一个没有连接的进程
		}
		// 时钟回拨时不让停止标记一直有效。
		return waited || (err == nil && next.Sub(now) > queueCollectionAuthRetry)
	}
	return waited || (err == nil && next.Sub(now) > queueCollectionBackoffMax)
}

// recordQueueCollectionResult updates the breaker after a round that sent requests.
func recordQueueCollectionResult(state *queueCollectionState, n int, runErr error, mark string, now time.Time) {
	if runErr == nil || n > 0 {
		state.ConsecutiveFailures = 0
		state.NextAttemptAt, state.WaitingForConnection, state.SessionMark = "", false, ""
		state.LastError, state.LastErrorCode = "", ""
		if runErr != nil { // 部分门店失败：照常显示，但不退避
			state.LastError, state.LastErrorCode = friendlyPublicQueueError(runErr), liveErrorCode(runErr)
		}
		return
	}
	state.ConsecutiveFailures++
	state.LastError, state.LastErrorCode = friendlyPublicQueueError(runErr), liveErrorCode(runErr)
	if state.LastErrorCode == liveErrAuthRequired {
		state.WaitingForConnection, state.SessionMark = true, mark
		state.NextAttemptAt = now.Add(queueCollectionAuthRetry).Format(time.RFC3339)
		return
	}
	state.WaitingForConnection, state.SessionMark = false, ""
	state.NextAttemptAt = now.Add(queueCollectionBackoff(state.ConsecutiveFailures)).Format(time.RFC3339)
}

func (c *QueueBaselineCollector) collectTick(ctx context.Context) {
	lock, err := platform.TryFileLock(filepath.Join(AppDirPath(), "queue_collection.lock"))
	if err != nil {
		if !errors.Is(err, platform.ErrFileLocked) {
			c.mu.Lock()
			c.lastError = err.Error()
			c.mu.Unlock()
		}
		return
	}
	defer lock.Close()
	cfg := LoadQueueBaselineConfig()
	state := loadQueueCollectionState()
	now := time.Now()
	mark := liveSessionMark(now)
	if mark != "" {
		state.LiveHolderPID, state.LiveHolderAt = os.Getpid(), now.Format(time.RFC3339)
	}
	key := strings.Join(queueBaselineStoreIDs(cfg), ",")
	state.Enabled, state.Running = cfg.Enabled, cfg.Enabled && ctx.Err() == nil
	state.PID, state.StoreIDs = os.Getpid(), queueBaselineStoreIDs(cfg)
	state.IntervalSeconds = queueBaselineIntervalSeconds(cfg, state.StoreIDs)
	state.PausedReason = publicQueueCollectionBlockedReason()
	if len(state.StoreIDs) == 0 {
		state.PausedReason = "先选择要记录的门店"
	}
	last, _ := time.Parse(time.RFC3339, state.LastAt)
	due := last.IsZero() || last.After(now) || now.Sub(last) >= time.Duration(state.IntervalSeconds)*time.Second || key != state.ConfigKey
	if state.Running && state.PausedReason == "" && due && queueCollectionMayAttempt(state, mark, now) {
		collect := c.collect
		if collect == nil {
			collect = collectQueueBaselineWithConfig
		}
		runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		n, runErr := collect(runCtx, cfg)
		cancel()
		if n > 0 {
			state.LastAt, state.ConfigKey = time.Now().Format(time.RFC3339), key
			runErr = errors.Join(runErr, rebuildLocalQueueModel(time.Now()))
		}
		if !(ctx.Err() != nil && errors.Is(runErr, context.Canceled)) {
			recordQueueCollectionResult(&state, n, runErr, mark, time.Now())
		}
	}
	state.HeartbeatAt = time.Now().Format(time.RFC3339)
	state.Running = state.Running && ctx.Err() == nil
	data, _ := json.Marshal(state)
	// The round's own error lives in the shared file; keep locally only what could not be
	// written there, so status() never re-adds an error the shared state has since cleared.
	localError := ""
	if err := AtomicWriteFile(queueCollectionStatePath(), data, 0o600); err != nil {
		state.LastError = err.Error()
		localError = state.LastError
	}
	c.mu.Lock()
	c.lastAt, _ = time.Parse(time.RFC3339, state.LastAt)
	c.lastError, c.pausedReason = localError, state.PausedReason
	c.mu.Unlock()
}

func sharedQueueCollectionStatus(now time.Time) PublicQueueCollectionStatus {
	state := loadQueueCollectionState()
	status := state.PublicQueueCollectionStatus
	beat, err := time.Parse(time.RFC3339, state.HeartbeatAt)
	status.Running = status.Running && err == nil && !beat.After(now) && now.Sub(beat) < 2*time.Minute && platform.IsProcessAlive(state.PID)
	cfg := LoadQueueBaselineConfig()
	status.Enabled, status.StoreIDs = cfg.Enabled, queueBaselineStoreIDs(cfg)
	status.Running = status.Running && cfg.Enabled
	status.IntervalSeconds = queueBaselineIntervalSeconds(cfg, status.StoreIDs)
	if status.WaitingForConnection {
		// 这个进程已经连上了新的连接：下一轮就会恢复，不再显示“需要连接”。
		if mark := liveSessionMark(now); mark != "" && mark != state.SessionMark {
			status.WaitingForConnection = false
			status.LastError, status.LastErrorCode, status.NextAttemptAt = "", "", ""
		}
	}
	if !cfg.Enabled {
		status.PausedReason = "已暂停采集"
	}
	return status
}
