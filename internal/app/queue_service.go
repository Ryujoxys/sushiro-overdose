package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
	"github.com/Ryujoxys/sushiro-overdose/internal/platform"
)

var queueServiceControlMu sync.Mutex

func publicQueueServiceLockPath() string { return filepath.Join(AppDirPath(), "queue_service.lock") }

func checkLegacySamplingProcess() error {
	if pid := atoi(readSamplingPID()); pid > 0 && platform.IsProcessAlive(pid) {
		return fmt.Errorf("采样 PID %d 仍存活，但未确认是新版公开服务；请先在旧版停止采样再重试，不会自动覆盖标记或结束未知进程", pid)
	}
	return nil
}

// This entrypoint is deliberately independent of the booking engine/scheduler.
func runPublicQueueService(ctx context.Context) error {
	lock, err := platform.TryFileLock(publicQueueServiceLockPath())
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := checkLegacySamplingProcess(); err != nil {
		return err
	}
	if err := AtomicWriteFile(samplingPidFilePath(), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		return err
	}
	defer removeSamplingPID(os.Getpid())
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	control, err := newQueueServiceControl()
	if err != nil {
		return err
	}
	defer control.close()
	controlDone := make(chan struct{})
	go func() { defer close(controlDone); control.watch(ctx, cancel) }()
	defer func() { cancel(); <-controlDone }()
	queueBaselineCollector.Start(ctx)
	<-ctx.Done()
	queueBaselineCollector.wait()
	return nil
}

func cmdPublicQueueDaemon() {
	if err := os.MkdirAll(AppDirPath(), 0o700); err != nil {
		return
	}
	log, err := os.OpenFile(SamplingLogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer log.Close()
	os.Stdout, os.Stderr = log, log
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runPublicQueueService(ctx); err != nil {
		fmt.Fprintln(log, err)
	}
}

func publicQueueServiceRunning() bool {
	pid := atoi(readSamplingPID())
	if pid <= 0 || !platform.IsProcessAlive(pid) {
		return false
	}
	lock, err := platform.TryFileLock(publicQueueServiceLockPath())
	if err == nil {
		lock.Close()
		return false
	}
	return err == platform.ErrFileLocked
}

func startPublicQueueDaemon() error {
	if publicQueueServiceRunning() {
		return nil
	}
	if err := checkLegacySamplingProcess(); err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "--queue-collector-child")
	cmd.SysProcAttr = platform.DaemonProcessAttrs()
	if err := cmd.Start(); err != nil {
		return err
	}
	// Reap the child without holding up the request. No terminal or window is opened.
	go func() { _ = cmd.Wait() }()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if publicQueueServiceRunning() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("后台采集尚未就绪，请查看 %s；未自动再次启动", SamplingLogPath())
}

type publicQueueServiceActions struct {
	install func() error
	remove  func() error
	start   func() error
}

func configurePublicQueueAutoStart(enabled bool, actions publicQueueServiceActions) error {
	queueServiceControlMu.Lock()
	defer queueServiceControlMu.Unlock()
	if !enabled {
		return actions.remove()
	}
	cfg := LoadQueueBaselineConfig()
	if len(queueBaselineStoreIDs(cfg)) == 0 {
		return fmt.Errorf("先选择要持续记录的门店")
	}
	cfg.Enabled = true
	if err := SaveQueueBaselineConfig(cfg); err != nil {
		return err
	}
	if err := actions.install(); err != nil {
		return fmt.Errorf("注册登录自启动失败: %w", err)
	}
	if err := actions.start(); err != nil {
		return fmt.Errorf("自启动已配置，但当前启动未确认: %w", err)
	}
	return nil
}

func setPublicQueueAutoStart(enabled bool) error {
	return configurePublicQueueAutoStart(enabled, publicQueueServiceActions{
		install: platform.InstallSamplingAutoStart, remove: platform.RemoveSamplingAutoStart, start: startPublicQueueDaemon,
	})
}

func publicQueueServiceResponse() map[string]any {
	return map[string]any{
		"config": LoadQueueBaselineConfig(), "state": queueBaselineCollector.status(),
		"background_running": publicQueueServiceRunning(), "autostart": platform.SamplingAutoStartStatus(),
		"model":     localQueueModelStatus(time.Now()),
		"data_path": AppDirPath(),
	}
}

func handlePublicQueueService(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, publicQueueServiceResponse())
	case http.MethodPost:
		var body struct {
			Action string `json:"action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, 400, "无效请求")
			return
		}
		var err error
		switch body.Action {
		case "enable_autostart":
			err = setPublicQueueAutoStart(true)
		case "disable_autostart":
			err = setPublicQueueAutoStart(false)
		case "repair_autostart":
			queueServiceControlMu.Lock()
			err = platform.RepairSamplingAutoStart()
			queueServiceControlMu.Unlock()
		case "start":
			queueServiceControlMu.Lock()
			cfg := LoadQueueBaselineConfig()
			if len(queueBaselineStoreIDs(cfg)) == 0 {
				err = fmt.Errorf("先选择要记录的门店")
			} else {
				cfg.Enabled = true
				err = SaveQueueBaselineConfig(cfg)
				if err == nil {
					err = startPublicQueueDaemon()
				}
			}
			queueServiceControlMu.Unlock()
		case "pause":
			queueServiceControlMu.Lock()
			cfg := LoadQueueBaselineConfig()
			cfg.Enabled = false
			err = SaveQueueBaselineConfig(cfg)
			queueServiceControlMu.Unlock()
		default:
			writeError(w, 400, "未知服务操作")
			return
		}
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, publicQueueServiceResponse())
	default:
		writeError(w, 405, "GET or POST only")
	}
}

func cmdCollect(args []string) {
	action := "status"
	if len(args) > 0 {
		action = strings.ToLower(args[0])
	}
	switch action {
	case "run":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := runPublicQueueService(ctx); err != nil {
			fmt.Println(err)
		}
	case "start":
		cfg := LoadQueueBaselineConfig()
		if len(queueBaselineStoreIDs(cfg)) == 0 {
			fmt.Println("先在界面选择要记录的门店")
			return
		}
		cfg.Enabled = true
		if err := SaveQueueBaselineConfig(cfg); err != nil {
			fmt.Println(err)
			return
		}
		if err := startPublicQueueDaemon(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("公开排队采集已在后台运行，不需要通行证")
		}
	case "stop":
		cfg := LoadQueueBaselineConfig()
		cfg.Enabled = false
		if err := SaveQueueBaselineConfig(cfg); err != nil {
			fmt.Println(err)
			return
		}
		if publicQueueServiceRunning() {
			_, _ = stopSamplingDaemon()
		}
		fmt.Println("已暂停公开采集；不会删除历史，登录自启动需另行关闭")
	case "autostart":
		cmdSampleAutoStart(args[1:])
	case "status", "":
		data, _ := json.MarshalIndent(publicQueueServiceResponse(), "", "  ")
		fmt.Println(string(data))
	default:
		fmt.Println("Usage: sushiro collect [status|run|start|stop|autostart on|autostart off]")
	}
}
