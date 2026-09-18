// Package platform 是平台适配层（facade）：把 OS 相关能力（系统代理、证书信任、通知、
// 进程管理、开机自启、浏览器拉起）封装成统一接口，内部按 build tag 分发到各平台实现
// （如 platform_desktop_darwin.go / platform_windows.go）。
// 跨平台业务代码只依赖这里导出的函数，不直接接触 syscall 或平台 API。
// 注意：本文件用 dot import 引入 core 包，是为了直接复用 core.MaintenanceResult 等类型
// 而不必每次写 core. 前缀。
package platform

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"errors"
	"sync"
	"syscall"
)

var ErrUserKeychainUnavailable = errors.New("macOS 用户钥匙串不可用，请在正常用户环境中重新启动应用，不要覆盖 HOME")

var errIsolatedAutoStart = errors.New("独立数据目录下不修改系统自启动；请使用正常安装的应用设置自启动")

// DesktopNotification sends an OS-level notification to the user.
func DesktopNotification(title, message string) {
	desktopNotification(title, message)
}

// SetSystemProxy configures the system HTTP/HTTPS proxy to localhost:port.
func SetSystemProxy(port int) error {
	return setSystemProxy(port)
}

// proxyWarnings 收集设代理过程中的非致命提示（如 Windows QUIC 屏蔽失败可能抓不到包）。
// engine 在 SetSystemProxy 后读它，把提示推给前端（不中断采集）。
var (
	proxyWarningsMu sync.Mutex
	proxyWarnings   []string
)

// RecordProxyWarning 记录一条非致命代理提示（平台层调用）。
func RecordProxyWarning(msg string) {
	proxyWarningsMu.Lock()
	proxyWarnings = append(proxyWarnings, msg)
	proxyWarningsMu.Unlock()
}

// DrainProxyWarnings 取出并清空已记录的代理提示（engine 在 SetSystemProxy 后调用）。
func DrainProxyWarnings() []string {
	proxyWarningsMu.Lock()
	defer proxyWarningsMu.Unlock()
	out := proxyWarnings
	proxyWarnings = nil
	return out
}

// ClearSystemProxy removes the system proxy configuration.
func ClearSystemProxy() error {
	return clearSystemProxy()
}

// IsCertTrusted checks if the CA certificate at certPath is trusted by the OS.
func IsCertTrusted() (bool, error) {
	return isCertTrusted()
}

// InstallCert installs the CA certificate into the OS trust store.
func InstallCert() error {
	return installCert()
}

// UninstallCert removes the CA certificate from the OS trust store when supported.
func UninstallCert() error {
	return uninstallCert()
}

// DaemonProcessAttrs returns syscall.SysProcAttr appropriate for spawning a daemon.
func DaemonProcessAttrs() *syscall.SysProcAttr {
	return daemonProcessAttrs()
}

// KillProcess sends a termination signal to the process with the given PID.
func KillProcess(pid int) error {
	return killProcess(pid)
}

// IsProcessAlive checks if a process with the given PID exists.
func IsProcessAlive(pid int) bool {
	return isProcessAlive(pid)
}

// KillRelatedAppProcesses terminates other running sushiro-overdose processes
// discovered by platform process listing. excludePID is never terminated.
func KillRelatedAppProcesses(excludePID int) []MaintenanceResult {
	return killRelatedAppProcesses(excludePID)
}

// OpenBrowser opens the local Web UI. Desktop platforms prefer a standalone
// app-style window when a Chromium-based browser is available, then fall back
// to the default browser.
func OpenBrowser(url string) error {
	return openBrowser(url)
}

// AutoStartStatus 描述「开机/登录自启取号」的能力与当前状态。
// Supported=false 表示该平台不支持自启（如某些 Linux 桌面）；Enabled 表示当前是否已注册自启；
// Path 是自启条目指向的可执行路径；Message/Error 供 UI 展示诊断信息。
type AutoStartStatus struct {
	Supported bool   `json:"supported"`
	Enabled   bool   `json:"enabled"`
	Path      string `json:"path,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SamplingAutoStartStatus 查询当前平台自启状态。不同平台语义不同
// （macOS 用 LaunchAgent，Windows 用注册表/启动文件夹），由各平台实现填充。
func SamplingAutoStartStatus() AutoStartStatus {
	if HasCustomDataHome() {
		return AutoStartStatus{Message: errIsolatedAutoStart.Error()}
	}
	return samplingAutoStartStatus()
}

// InstallSamplingAutoStart registers the public collector at user login, not a booking scheduler.
func InstallSamplingAutoStart() error {
	if HasCustomDataHome() {
		return errIsolatedAutoStart
	}
	return installSamplingAutoStart()
}

// RemoveSamplingAutoStart removes the public collector's login registration.
func RemoveSamplingAutoStart() error {
	if HasCustomDataHome() {
		return errIsolatedAutoStart
	}
	return removeSamplingAutoStart()
}

// MCPAutoStartStatus 查询 MCP 助手开机自启状态。
func MCPAutoStartStatus() AutoStartStatus {
	if HasCustomDataHome() {
		return AutoStartStatus{Message: errIsolatedAutoStart.Error()}
	}
	return mcpAutoStartStatus()
}

// InstallMCPAutoStart 注册 MCP 助手开机自启（启动 sushiro --mcp-daemon-child 确保 venv 就绪）。
func InstallMCPAutoStart() error {
	if HasCustomDataHome() {
		return errIsolatedAutoStart
	}
	return installMCPAutoStart()
}

// RemoveMCPAutoStart 取消 MCP 助手开机自启。幂等。
func RemoveMCPAutoStart() error {
	if HasCustomDataHome() {
		return errIsolatedAutoStart
	}
	return removeMCPAutoStart()
}

// IsQuarantined 报告当前可执行文件是否被 macOS Gatekeeper 隔离（带 com.apple.quarantine
// 扩展属性）。隔离状态下系统可能限制网络/代理设置或弹 Gatekeeper 拦截。
// Windows/Linux 恒返回 (false, nil)。
func IsQuarantined() (bool, error) {
	return isQuarantined()
}

// WeChatProcessInfo 描述一个被识别为微信系的进程（Windows: WeChat/WeChatAppEx/Weixin/
// WeChatPlayer；macOS: 微信.app）。StartTime 为 RFC3339 字符串，便于跨平台一致地做基线比对。
type WeChatProcessInfo struct {
	PID       int    `json:"pid"`
	Name      string `json:"name"`
	StartTime string `json:"start_time,omitempty"`
	Path      string `json:"path,omitempty"`
}

// ListWeChatProcesses 枚举当前运行的微信系进程。Windows 用 PowerShell（结构化输出），
// macOS 用 pgrep+ps，Linux 恒返回空切片（Linux 无微信小程序客户端）。失败返回空切片，不 panic。
func ListWeChatProcesses() []WeChatProcessInfo {
	return listWeChatProcesses()
}

// KillWeChatProcesses 终止所有微信系进程并返回逐个结果（复用 MaintenanceResult）。
// 用于 PC 微信抓包场景：用户忘关 WeChatAppEx 导致抓不到包时，一键结束。
func KillWeChatProcesses() []MaintenanceResult {
	return killWeChatProcesses()
}
