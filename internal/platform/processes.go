package platform

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

func killRelatedAppProcessesByPGrep(excludePID int) []MaintenanceResult {
	out, err := exec.Command("pgrep", "-f", "sushiro|Sushiro").Output()
	if err != nil {
		return []MaintenanceResult{{
			Name:   "related_processes",
			Action: "kill_by_name",
			Status: MaintenanceStatusMissing,
		}}
	}
	results := []MaintenanceResult{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		pid, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || pid <= 0 || pid == excludePID {
			continue
		}
		result := MaintenanceResult{
			Name:   "related_process",
			Action: "kill_by_name",
			Status: MaintenanceStatusOK,
			Error:  fmt.Sprintf("pid %d", pid),
		}
		if err := KillProcess(pid); err != nil {
			result.Status = MaintenanceStatusError
			result.Error = fmt.Sprintf("pid %d: %s", pid, err.Error())
		}
		results = append(results, result)
	}
	if len(results) == 0 {
		results = append(results, MaintenanceResult{
			Name:   "related_processes",
			Action: "kill_by_name",
			Status: MaintenanceStatusMissing,
		})
	}
	return results
}

// ---------- 微信进程枚举/杀进程（PC 微信抓包引导用） ----------

// weChatProcessLine 是 PowerShell/ps 输出的结构化行，用于解析成 WeChatProcessInfo。
type weChatProcessLine struct {
	PID       int    `json:"pid"`
	Name      string `json:"name"`
	StartTime string `json:"start_time"`
	Path      string `json:"path"`
}

// weChatKillLine 是 killWeChatProcesses 的 PowerShell 输出行。
type weChatKillLine struct {
	PID    int    `json:"pid"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

// parseWeChatProcessLines 把每行一个 JSON 的进程枚举输出解析成 WeChatProcessInfo 列表。
// 坏行跳过（不 panic、不整体失败），保证幂等：拿不到就当没有。空输入返回 nil。
func parseWeChatProcessLines(out string) []WeChatProcessInfo {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var infos []WeChatProcessInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var item weChatProcessLine
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}
		if item.PID <= 0 {
			continue
		}
		name := item.Name
		if name == "" {
			name = "wechat"
		}
		infos = append(infos, WeChatProcessInfo{
			PID:       item.PID,
			Name:      name,
			StartTime: item.StartTime,
			Path:      item.Path,
		})
	}
	return infos
}

// parseWeChatKillOutput 把每行一个 JSON 的杀进程输出解析成 MaintenanceResult 列表。
// 空输出回退为单个 missing 结果（与 parseRelatedProcessKillOutput 一致）。
func parseWeChatKillOutput(out string) []MaintenanceResult {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	results := []MaintenanceResult{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var item weChatKillLine
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			results = append(results, MaintenanceResult{
				Name:   "wechat_process",
				Action: "kill_wechat",
				Status: MaintenanceStatusError,
				Error:  err.Error() + ": " + line,
			})
			continue
		}
		status := MaintenanceStatusOK
		if item.Status == MaintenanceStatusError || item.Status == "error" {
			status = MaintenanceStatusError
		}
		name := item.Name
		if name == "" {
			name = "wechat_process"
		}
		errMsg := fmt.Sprintf("pid %d", item.PID)
		if item.Error != "" {
			errMsg += ": " + item.Error
		}
		results = append(results, MaintenanceResult{
			Name:   name,
			Action: "kill_wechat",
			Status: status,
			Error:  errMsg,
		})
	}
	if len(results) == 0 {
		results = append(results, MaintenanceResult{
			Name:   "wechat_processes",
			Action: "kill_wechat",
			Status: MaintenanceStatusMissing,
		})
	}
	return results
}

type relatedProcessKillResult struct {
	PID    int    `json:"pid"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

func parseRelatedProcessKillOutput(out string) []MaintenanceResult {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	results := []MaintenanceResult{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var item relatedProcessKillResult
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			results = append(results, MaintenanceResult{
				Name:   "related_process",
				Action: "kill_by_name",
				Status: MaintenanceStatusError,
				Error:  err.Error() + ": " + line,
			})
			continue
		}
		status := MaintenanceStatusOK
		if item.Status == MaintenanceStatusError || item.Status == "error" {
			status = MaintenanceStatusError
		}
		name := item.Name
		if name == "" {
			name = "related_process"
		}
		result := MaintenanceResult{
			Name:   name,
			Action: "kill_by_name",
			Path:   item.Path,
			Status: status,
			Error:  fmt.Sprintf("pid %d", item.PID),
		}
		if item.Error != "" {
			result.Error += ": " + item.Error
		}
		results = append(results, result)
	}
	if len(results) == 0 {
		results = append(results, MaintenanceResult{
			Name:   "related_processes",
			Action: "kill_by_name",
			Status: MaintenanceStatusMissing,
		})
	}
	return results
}
