package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/platform"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

// Keep the legacy PID filename for installed startup entries. The public service
// also holds queue_service.lock for its lifetime; a PID alone is not ownership.
const samplingPidFile = "sampling.pid"

func samplingPidFilePath() string {
	return filepath.Join(AppDirPath(), samplingPidFile)
}

func cmdSample(args []string) {
	action := "status"
	if len(args) > 0 {
		action = strings.ToLower(strings.TrimSpace(args[0]))
	}
	switch action {
	case "status", "":
		cmdSampleStatus()
	case "once":
		cmdSampleOnce()
	case "run":
		cmdSampleRun()
	case "start":
		cmdSampleStart()
	case "stop", "exit":
		cmdSampleStop()
	case "autostart", "login":
		cmdSampleAutoStart(args[1:])
	default:
		fmt.Println("Usage: sushiro sample [status|once|run|start|stop|autostart]")
	}
}

func cmdSampleStatus() {
	cmdCollect([]string{"status"})
}

func cmdSampleOnce() {
	cfg := LoadSamplingConfig()
	cfg.Enabled = true
	result := sampler.RunOnceNow(context.Background(), cfg)
	printSamplingResult(result)
}

// The advanced foreground sampler still reads authenticated reservation slots.
// Use collect run for credential-free public queue data.
func cmdSampleRun() {
	printBanner()
	cfg := LoadSamplingConfig()
	cfg.Enabled = true
	if err := SaveSamplingConfig(cfg); err != nil {
		fmt.Println("保存信息收集配置失败:", err)
		return
	}
	fmt.Println("信息收集前台运行:", samplingSummary(cfg))
	fmt.Println("按 Ctrl+C 退出")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := sampler.startWithConfig(ctx, cfg); err != nil {
		fmt.Println("启动信息收集失败:", err)
		return
	}
	<-ctx.Done()
	sampler.Stop()
}

// Legacy background command now starts only the credential-free public service.
func cmdSampleStart() { cmdCollect([]string{"start"}) }

func cmdSampleStop() {
	cmdCollect([]string{"stop"})
}

func cmdSampleAutoStart(args []string) {
	action := "status"
	if len(args) > 0 {
		action = strings.ToLower(strings.TrimSpace(args[0]))
	}
	switch action {
	case "status", "":
		status := SamplingAutoStartStatus()
		fmt.Println("登录自启动:", autoStartSummary(status))
		if status.Path != "" {
			fmt.Println("位置:", status.Path)
		}
	case "on", "enable", "start":
		if err := setPublicQueueAutoStart(true); err != nil {
			fmt.Println("启用失败:", err)
			return
		}
		fmt.Println("公开排队采集已配置登录自启动，不会自动取号")
	case "off", "disable", "stop":
		if err := setPublicQueueAutoStart(false); err != nil {
			fmt.Println("取消失败:", err)
			return
		}
		fmt.Println("登录自启动已取消")
	default:
		fmt.Println("Usage: sushiro sample autostart [status|on|off]")
	}
}

func autoStartSummary(status AutoStartStatus) string {
	if !status.Supported {
		if status.Message != "" {
			return "不支持 (" + status.Message + ")"
		}
		return "不支持"
	}
	if status.Enabled {
		return "已启用"
	}
	if status.Error != "" {
		return "状态异常: " + status.Error
	}
	return "未启用"
}

// Old installed startup entries remain compatible, but no longer start a ticket scheduler.
func cmdSamplerDaemon() { cmdPublicQueueDaemon() }

func readSamplingPID() string {
	data, err := os.ReadFile(samplingPidFilePath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// isSamplingDaemonRunning 判断独立守护进程是否在跑，并顺带做 PID 自愈：
// PID 文件里的进程已死时，顺手清掉过期 PID 文件，避免下次误判。
func isSamplingDaemonRunning() bool {
	pid := readSamplingPID()
	if pid == "" {
		return false
	}
	n := atoi(pid)
	if n > 0 && IsProcessAlive(n) {
		return true
	}
	removeSamplingPID(n)
	return false
}

// writeSamplingPID 覆盖写当前进程 PID 到 PID 文件。由守护进程自己在采样启动成功后调用。
func writeSamplingPID(pid int) {
	_ = os.WriteFile(samplingPidFilePath(), []byte(fmt.Sprintf("%d", pid)), 0o644)
}

// removeSamplingPID 只在 PID 文件里记录的仍是自己（或 pid<=0）时才删除。
// 这个「先比对再删」是为了防止守护进程 A 崩溃后被清理、期间守护进程 B 已写入新 PID，
// 此时 A 的 defer 若无脑 os.Remove 会误删 B 的 PID 文件。
func removeSamplingPID(pid int) {
	current := atoi(readSamplingPID())
	if pid <= 0 || current == pid {
		_ = os.Remove(samplingPidFilePath())
	}
}

// stopSamplingDaemon 按 PID 文件停止守护进程，带两层自我保护：
//  1. PID 指向的进程已死 -> 自愈清掉过期 PID 文件，返回未运行。
//  2. PID == 当前进程 -> 拒绝自杀（调用方自己就是守护进程时不能 kill 自己）。
//
// 成功 kill 后清掉 PID 文件。返回 (是否真停了一个进程, 错误)。
func stopSamplingDaemon() (bool, error) {
	pidStr := readSamplingPID()
	if pidStr == "" {
		return false, nil
	}
	pid := atoi(pidStr)
	if pid <= 0 || !IsProcessAlive(pid) {
		removeSamplingPID(pid)
		return false, nil
	}
	// 自杀保护：PID 文件指向自己时不 kill，否则守护进程自己收到 stop 命令会把自己干掉。
	if pid == os.Getpid() {
		return false, nil
	}
	if err := KillProcess(pid); err != nil {
		return true, err
	}
	removeSamplingPID(pid)
	return true, nil
}

func printSamplingResult(result SamplingRunResult) {
	if result.Skipped {
		fmt.Println("本轮跳过:", result.SkipReason)
		return
	}
	fmt.Printf("信息收集完成: %d 家门店, %d 条时段, %d 个错误\n", len(result.Stores), result.Snapshots, result.StoreErrors)
	for _, store := range result.Stores {
		if store.Error != "" {
			fmt.Printf("  %s %s: %s\n", store.StoreID, store.StoreName, store.Error)
			continue
		}
		fmt.Printf("  %s %s: %d 条\n", store.StoreID, store.StoreName, store.Slots)
	}
}
