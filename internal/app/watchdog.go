package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/platform"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const proxyStateFile = "proxy_active.json"

func proxyStatePath() string {
	return fmt.Sprintf("%s/%s", AppDirPath(), proxyStateFile)
}

type proxyState struct {
	RecoveryVersion int       `json:"recovery_version,omitempty"`
	Active          bool      `json:"active"`
	Port            int       `json:"port"`
	SetAt           time.Time `json:"set_at"`
	PID             int       `json:"pid"`
}

// markProxyActive records that the system proxy is currently set.
func markProxyActive(port, pid int) error {
	state := proxyState{
		RecoveryVersion: 2,
		Active:          true,
		Port:            port,
		SetAt:           time.Now(),
		PID:             pid,
	}
	data, _ := json.MarshalIndent(state, "", "  ")
	if err := os.MkdirAll(AppDirPath(), 0o700); err != nil {
		return err
	}
	return AtomicWriteFile(proxyStatePath(), data, 0o600)
}

// markProxyInactive clears the proxy state marker.
func markProxyInactive() {
	os.Remove(proxyStatePath())
}

// checkStaleProxy checks for leftover proxy state and cleans it up.
// Returns true if a stale proxy was found and cleaned.
func checkStaleProxy() bool {
	data, err := os.ReadFile(proxyStatePath())
	if err != nil {
		return false
	}

	var state proxyState
	if err := json.Unmarshal(data, &state); err != nil {
		return false
	}

	if !state.Active {
		return false
	}

	// Check if the process that set the proxy is still alive
	if state.PID > 0 && IsProcessAlive(state.PID) {
		// The process is still running, don't interfere
		return false
	}

	// Stale proxy detected — clean up
	LogMessage(time.Now(), fmt.Sprintf("检测到残留代理设置 (PID %d 已退出)，正在清除...", state.PID))
	if err := ClearSystemProxy(); err != nil {
		LogMessage(time.Now(), "恢复残留代理失败，保留标记: "+err.Error())
		return false
	}
	markProxyInactive()
	return true
}
