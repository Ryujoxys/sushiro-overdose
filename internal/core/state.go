package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// State 是持久化到 ~/.sushiro/.sushiro_state.json 的运行态。
//
// 预约和排队号物理隔离成两个槽，避免历史上「单槽混装 + 事后用 isLocalNetTicketRecord
// 区分」导致后写入的一方静默覆盖先写入的另一方（预约被取号冲掉、或反之）。
//   - ActiveReservation：当前活跃的预约（含本地手填的预约记录）
//   - ActiveNetTicket：当前活跃的排队号（当天有效）
//
// SavedAt 是写入时间戳（RFC3339），便于诊断「这个状态是多久前写的」。
type State struct {
	ActiveReservation *ReservationRecord `json:"active_reservation,omitempty"`
	ActiveNetTicket   *ReservationRecord `json:"active_net_ticket,omitempty"`
	SavedAt           string             `json:"saved_at,omitempty"`
}

// LoadState 从 path 读 State。文件不存在视为「无活跃预约」，返回零值且不报错——这是正常首次启动的情况。
// 其它读取错误（权限/IO）才向上抛。
func LoadState(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, fmt.Errorf("read state: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("invalid JSON in state file %s: %w", path, err)
	}
	return state, nil
}

// SaveState 用「写临时文件 + 原子 rename」的方式落盘 State，避免写到一半进程挂掉导致状态文件损坏。
// 约定：tempPath = path + ".tmp"，先全量写临时文件，再 rename 到目标路径——rename 在同文件系统内是原子的。
// 因此 .tmp 必须和目标在同一目录（这里就是同目录拼接），跨文件系统 rename 会退化为非原子拷贝。
func SaveState(path string, state State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}

// ClearState 删除状态文件（如取消预约/取号后）。文件本就不存在不算错误，幂等。
func ClearState(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove state: %w", err)
	}
	return nil
}

// LogMessage 以 "[RFC3339 时间] 消息" 格式打印一行到 stdout。用于运行过程的可见日志。
func LogMessage(now time.Time, message string) {
	fmt.Printf("[%s] %s\n", now.Format(time.RFC3339), message)
}

// StdinReader is a shared buffered reader for stdin to avoid losing data.
var StdinReader = bufio.NewReader(os.Stdin)

// ReadInput reads a trimmed line from stdin.
func ReadInput() string {
	line, _ := StdinReader.ReadString('\n')
	return strings.TrimSpace(line)
}
