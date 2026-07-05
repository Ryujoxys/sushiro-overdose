package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/notify"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
)

// preferencesMu 序列化所有对 preferences 的「读-改-写」与整盘写。
// SavePreferences 本身是原子写（不写半截），但不防 lost-update：
// engine 抓到凭证回填 SelectedStores 与用户在 UI 点保存（整盘覆写）并发时，
// 后写者会冲掉先写者的字段。这把锁把两端串起来。
var preferencesMu sync.Mutex

// UpdatePreferences 在锁内执行 Load→mutate→Save，供所有读改写场景使用。
func UpdatePreferences(mutate func(*UserPreferences)) error {
	preferencesMu.Lock()
	defer preferencesMu.Unlock()
	prefs := LoadPreferences()
	mutate(&prefs)
	return SavePreferences(prefs)
}

// SavePreferencesLocked 在 preferencesMu 持有时整盘保存（用户显式覆盖全部字段）。
func SavePreferencesLocked(prefs UserPreferences) error {
	preferencesMu.Lock()
	defer preferencesMu.Unlock()
	return SavePreferences(prefs)
}

func handlePreferences(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, LoadPreferences())
	case http.MethodPost, http.MethodPut:
		var prefs UserPreferences
		if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
			writeError(w, http.StatusBadRequest, "无效的请求格式: "+err.Error())
			return
		}
		prefs = NormalizePreferences(prefs)
		// 整盘写也走锁，防止与 engine 的 UpdatePreferences 读改写并发冲掉字段。
		if err := SavePreferencesLocked(prefs); err != nil {
			writeError(w, http.StatusInternalServerError, "保存失败: "+err.Error())
			return
		}
		refreshWebClient()
		writeJSON(w, map[string]any{"ok": true, "preferences": prefs})
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST")
	}
}

func handleNotifyConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, _ := LoadNotifyConfig()
		if cfg == nil {
			cfg = &NotifyConfig{}
		}
		writeJSON(w, cfg)
	case http.MethodPost, http.MethodPut:
		var cfg NotifyConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "无效的请求格式: "+err.Error())
			return
		}
		if err := SaveNotifyConfig(&cfg); err != nil {
			writeError(w, http.StatusInternalServerError, "保存失败: "+err.Error())
			return
		}
		setNotifier(BuildNotifierFromConfig())
		writeJSON(w, map[string]any{"ok": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST")
	}
}

func handleDiscoveryConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := LoadAPIDiscoveryConfig()
		writeJSON(w, map[string]any{
			"config":        cfg,
			"records_count": APIDiscoveryRecordCount(),
			"records_path":  APIDiscoveryRecordsPath(),
		})
	case http.MethodPost, http.MethodPut:
		var cfg APIDiscoveryConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "无效的请求格式: "+err.Error())
			return
		}
		if err := SaveAPIDiscoveryConfig(cfg); err != nil {
			writeError(w, http.StatusInternalServerError, "保存失败: "+err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true, "config": LoadAPIDiscoveryConfig()})
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST")
	}
}

func handleDiscoveryRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	records, err := LoadAPIDiscoveryRecords(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取调试记录失败: "+err.Error())
		return
	}
	writeJSON(w, map[string]any{
		"records": records,
		"path":    APIDiscoveryRecordsPath(),
	})
}

func handleDiscoveryClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	if err := ClearAPIDiscoveryRecords(); err != nil {
		writeError(w, http.StatusInternalServerError, "清空调试记录失败: "+err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func handleRepairProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	report := RepairProxy()
	status := http.StatusOK
	if !report.OK {
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(report)
}

func handleUninstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var options UninstallOptions
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&options)
	}
	if !uninstallOptionsSelected(options) {
		options.All = true
		options.Certificates = true
		options.SystemCert = true
	}
	repair := dryRunRepairProxyReport()
	if !options.DryRun {
		repair = RepairProxy()
	}
	uninstall := UninstallLocalData(options)
	status := http.StatusOK
	if !repair.OK || !uninstall.OK {
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"ok":        repair.OK && uninstall.OK,
		"repair":    repair,
		"uninstall": uninstall,
	})
}

func handleStopProcesses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var options StopProcessOptions
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&options)
	}
	report := StopAppProcesses(options)
	status := http.StatusOK
	if !report.OK {
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(report)
	if options.IncludeSelf && !options.DryRun {
		scheduleSelfExit()
	}
}

func handleKillWeChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	report := KillWeChat()
	// 反馈到 engine 日志，前端 SSE 'log' 事件会渲染，让用户知道杀了几/结果。
	killed := 0
	for _, res := range report.Results {
		if res.Status == MaintenanceStatusOK {
			killed++
		}
	}
	engine.addLog("已结束 " + strconv.Itoa(killed) + " 个微信进程（共 " + strconv.Itoa(len(report.Results)) + " 项）")
	status := http.StatusOK
	if !report.OK {
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(report)
}

func uninstallOptionsSelected(options UninstallOptions) bool {
	return options.All || options.Config || options.Notify || options.Feishu ||
		options.Preferences || options.Stores || options.State || options.History ||
		options.PID || options.ProxyMarker || options.Certificates || options.SystemCert
}
