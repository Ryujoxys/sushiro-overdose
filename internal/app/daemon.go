package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/platform"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"fmt"
	"os"
	"strings"
)

func cmdStop() {
	pid := readPID()
	if pid == "" {
		fmt.Println("sushiro is not running")
		return
	}
	if err := KillProcess(atoi(pid)); err != nil {
		fmt.Println("停止失败:", err)
		os.Remove(PidFilePath())
		return
	}
	os.Remove(PidFilePath())
	fmt.Println("sushiro stopped")
}

func cmdStatus() {
	pid := readPID()
	if pid == "" || !isRunning() {
		fmt.Println("sushiro is not running")
		return
	}
	fmt.Printf("sushiro is running (PID %s)\n", pid)

	log, err := os.ReadFile(LogPath())
	if err == nil && len(log) > 0 {
		lines := strings.Split(strings.TrimSpace(string(log)), "\n")
		start := len(lines) - 10
		if start < 0 {
			start = 0
		}
		fmt.Println("\n最近日志:")
		for _, line := range lines[start:] {
			fmt.Println("  " + line)
		}
	}
}

func readPID() string {
	data, err := os.ReadFile(PidFilePath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func isRunning() bool {
	pid := readPID()
	if pid == "" {
		return false
	}
	return IsProcessAlive(atoi(pid))
}

func atoi(s string) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n <= 0 {
		return -1
	}
	return n
}
