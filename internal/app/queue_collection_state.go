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

type queueCollectionState struct {
	PublicQueueCollectionStatus
	PID         int    `json:"pid"`
	HeartbeatAt string `json:"heartbeat_at"`
	ConfigKey   string `json:"config_key"`
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
	if state.Running && state.PausedReason == "" && due {
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
		state.LastError = ""
		if runErr != nil && !(ctx.Err() != nil && errors.Is(runErr, context.Canceled)) {
			state.LastError = runErr.Error()
		}
	}
	state.HeartbeatAt = time.Now().Format(time.RFC3339)
	state.Running = state.Running && ctx.Err() == nil
	data, _ := json.Marshal(state)
	if err := AtomicWriteFile(queueCollectionStatePath(), data, 0o600); err != nil {
		state.LastError = err.Error()
	}
	c.mu.Lock()
	c.lastAt, _ = time.Parse(time.RFC3339, state.LastAt)
	c.lastError, c.pausedReason = state.LastError, state.PausedReason
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
	if !cfg.Enabled {
		status.PausedReason = "已暂停采集"
	}
	return status
}
