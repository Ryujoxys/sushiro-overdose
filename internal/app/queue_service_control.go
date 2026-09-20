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
)

// A per-run token makes a stop request belong to one collector in one data
// directory. No process is killed by PID or by executable name.
type queueServiceControl struct {
	PID   int    `json:"pid"`
	Token string `json:"token"`
}

func queueServiceControlPath() string {
	return filepath.Join(AppDirPath(), "queue_service_session.json")
}
func queueServiceStopPath() string { return filepath.Join(AppDirPath(), "queue_service_stop") }

func readQueueServiceControl() queueServiceControl {
	var control queueServiceControl
	data, _ := os.ReadFile(queueServiceControlPath())
	_ = json.Unmarshal(data, &control)
	return control
}

func newQueueServiceControl() (queueServiceControl, error) {
	if err := os.MkdirAll(AppDirPath(), 0o700); err != nil {
		return queueServiceControl{}, err
	}
	control := queueServiceControl{PID: os.Getpid(), Token: newWebCSRFToken()}
	data, err := json.Marshal(control)
	if err == nil {
		err = AtomicWriteFile(queueServiceControlPath(), data, 0o600)
	}
	return control, err
}

func (c queueServiceControl) watch(ctx context.Context, stop context.CancelFunc) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data, _ := os.ReadFile(queueServiceStopPath())
			if strings.TrimSpace(string(data)) == c.Token {
				stop()
				return
			}
		}
	}
}

func (c queueServiceControl) close() {
	if readQueueServiceControl() == c {
		_ = os.Remove(queueServiceControlPath())
	}
	data, _ := os.ReadFile(queueServiceStopPath())
	if strings.TrimSpace(string(data)) == c.Token {
		_ = os.Remove(queueServiceStopPath())
	}
}

func stopPublicQueueService(ctx context.Context) error {
	queueServiceControlMu.Lock()
	defer queueServiceControlMu.Unlock()
	if !publicQueueServiceRunning() {
		return checkLegacySamplingProcess()
	}
	control := readQueueServiceControl()
	if control.PID != atoi(readSamplingPID()) || len(control.Token) < 32 {
		return errors.New("后台由旧版程序运行，请先用旧版的 collect stop 退出；不会强制结束未知进程")
	}
	if err := AtomicWriteFile(queueServiceStopPath(), []byte(control.Token), 0o600); err != nil {
		return errors.New("无法通知后台退出，请检查本机数据目录权限")
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if !publicQueueServiceRunning() {
			return nil
		}
		if current := readQueueServiceControl(); current != control && current.Token != "" {
			return errors.New("后台状态已变化，请重新确认退出")
		}
		select {
		case <-ctx.Done():
			return errors.New("后台还在完成本次记录，请稍后再退出")
		case <-ticker.C:
		}
	}
}
