//go:build darwin

package platform

import . "github.com/Ryujoxys/sushiro-overdose/internal/proxy"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func desktopNotification(title, message string) {
	// Escape double quotes to prevent AppleScript injection
	t := strings.ReplaceAll(title, `"`, `\"`)
	m := strings.ReplaceAll(message, `"`, `\"`)
	_ = exec.Command("osascript", "-e",
		fmt.Sprintf(`display notification "%s" with title "%s"`, m, t),
	).Run()
}

func setSystemProxy(port int) error {
	services, err := getNetworkServices()
	if err != nil {
		return err
	}
	return darwinRunSystemProxyCommands(darwinSetSystemProxyCommands(services, port, GetActiveWebPort()), runCmd)
}

func clearSystemProxy() error {
	services, err := getNetworkServices()
	if err != nil {
		return err
	}
	return darwinRunSystemProxyCommands(darwinClearSystemProxyCommands(services), runCmd)
}

type darwinCommandRunner func(name string, args ...string) (string, error)

func darwinRunSystemProxyCommands(commands [][]string, runner darwinCommandRunner) error {
	var errs []error
	for _, command := range commands {
		if len(command) == 0 {
			errs = append(errs, errors.New("empty networksetup command"))
			continue
		}
		out, err := runner(command[0], command[1:]...)
		if err == nil {
			continue
		}
		errs = append(errs, fmt.Errorf("%s failed: %w (output: %q)", strings.Join(command, " "), err, strings.TrimSpace(out)))
	}
	return errors.Join(errs...)
}

func darwinSetSystemProxyCommands(services []string, ProxyPort, webPort int) [][]string {
	commands := make([][]string, 0, len(services)*5)
	proxyPortString := fmt.Sprintf("%d", ProxyPort)
	for _, svc := range services {
		if webPort > 0 {
			pacURL := fmt.Sprintf("http://127.0.0.1:%d/proxy.pac?proxy=%d", webPort, ProxyPort)
			commands = append(commands,
				[]string{"networksetup", "-setautoproxyurl", svc, pacURL},
				[]string{"networksetup", "-setautoproxystate", svc, "on"},
				[]string{"networksetup", "-setwebproxystate", svc, "off"},
				[]string{"networksetup", "-setsecurewebproxystate", svc, "off"},
			)
			continue
		}
		commands = append(commands,
			[]string{"networksetup", "-setautoproxystate", svc, "off"},
			[]string{"networksetup", "-setwebproxy", svc, "127.0.0.1", proxyPortString},
			[]string{"networksetup", "-setsecurewebproxy", svc, "127.0.0.1", proxyPortString},
			[]string{"networksetup", "-setwebproxystate", svc, "on"},
			[]string{"networksetup", "-setsecurewebproxystate", svc, "on"},
		)
	}
	return commands
}

func darwinClearSystemProxyCommands(services []string) [][]string {
	commands := make([][]string, 0, len(services)*3)
	for _, svc := range services {
		commands = append(commands,
			[]string{"networksetup", "-setautoproxystate", svc, "off"},
			[]string{"networksetup", "-setwebproxystate", svc, "off"},
			[]string{"networksetup", "-setsecurewebproxystate", svc, "off"},
		)
	}
	return commands
}

func getNetworkServices() ([]string, error) {
	out, err := runCmd("networksetup", "-listallnetworkservices")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var services []string
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "*") {
			services = append(services, line)
		}
	}
	return services, nil
}

func isCertTrusted() (bool, error) {
	dir := CertDirPath()
	certPath := filepath.Join(dir, "ca.crt")

	if _, err := os.Stat(certPath); err != nil {
		return false, nil
	}

	cmd := exec.Command("security", "verify-cert", "-c", certPath, "-p", "basic")
	if err := cmd.Run(); err != nil {
		return false, nil
	}
	return true, nil
}

func installCert() error {
	return darwinInstallCert(filepath.Join(CertDirPath(), "ca.crt"), defaultUserKeychain, runCmd)
}

func darwinInstallCert(certPath string, resolveKeychain func() (string, error), runner darwinCommandRunner) error {
	keychain, err := resolveKeychain()
	if err != nil {
		return err
	}

	out, err := runner("security", "add-certificates", "-k", keychain, certPath)
	if err != nil && !isAlreadyExistsOutput([]byte(out)) {
		return fmt.Errorf("add-certificates: %w: %s", err, strings.TrimSpace(out))
	}

	out, err = runner("security", "add-trusted-cert", "-r", "trustRoot", "-k", keychain, certPath)
	if err != nil {
		low := strings.ToLower(out)
		// keychain 锁定时 security 报 "User interaction is not allowed" 或 "authfailed"。
		// 给出明确关键词，engine.classifyCertError 据此归为 cert_locked 并提示解锁。
		if strings.Contains(low, "user interaction is not allowed") || strings.Contains(low, "authfailed") {
			return fmt.Errorf("add-trusted-cert: keychain locked (User interaction is not allowed): %w: %s；请在终端运行 security unlock-keychain 解锁钥匙串后重试", err, strings.TrimSpace(out))
		}
		return fmt.Errorf("add-trusted-cert: %w: %s", err, strings.TrimSpace(out))
	}
	return nil
}

func uninstallCert() error {
	keychain, err := defaultUserKeychain()
	if err != nil {
		return err
	}
	out, err := exec.Command("security", "delete-certificate", "-c", "Sushiro Proxy CA", keychain).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(strings.ToLower(msg), "could not be found") || strings.Contains(msg, "未找到") {
			return nil
		}
		return fmt.Errorf("delete-certificate: %w: %s", err, msg)
	}
	return nil
}

func defaultUserKeychain() (string, error) {
	return resolveDarwinUserKeychain(runCmd, os.Stat)
}

func resolveDarwinUserKeychain(runner darwinCommandRunner, stat func(string) (os.FileInfo, error)) (string, error) {
	out, err := runner("security", "default-keychain", "-d", "user")
	if err != nil {
		return "", fmt.Errorf("%w: default-keychain: %v: %s", ErrUserKeychainUnavailable, err, strings.TrimSpace(out))
	}
	keychain := strings.Trim(strings.TrimSpace(out), `"`)
	if !filepath.IsAbs(keychain) || strings.ContainsAny(keychain, "\r\n\x00") {
		return "", fmt.Errorf("%w: default-keychain returned an invalid path", ErrUserKeychainUnavailable)
	}
	info, err := stat(keychain)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %v", ErrUserKeychainUnavailable, keychain, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%w: %s is a directory", ErrUserKeychainUnavailable, keychain)
	}
	return keychain, nil
}

func isAlreadyExistsOutput(out []byte) bool {
	msg := strings.ToLower(string(out))
	return strings.Contains(msg, "already exists") || strings.Contains(msg, "已存在")
}

func daemonProcessAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func killProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

func killRelatedAppProcesses(excludePID int) []MaintenanceResult {
	return killRelatedAppProcessesByPGrep(excludePID)
}

func isProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}

// isQuarantined 判断当前可执行文件是否带 com.apple.quarantine 扩展属性（Gatekeeper 隔离）。
// 用 xattr -p 读该属性：退出码 0 = 有隔离属性（已隔离）；退出码 1 = 无此属性（未隔离）；
// 其它错误（如 xattr 命令缺失）返回 err。os.Executable() 取自身路径。
func isQuarantined() (bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return false, err
	}
	cmd := exec.Command("xattr", "-p", "com.apple.quarantine", exe)
	if err := cmd.Run(); err != nil {
		// 退出码 1 = 属性不存在 = 未隔离；这是正常路径，不是错误。
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// listWeChatProcesses 用 pgrep + ps 枚举 macOS 微信进程。StartTime 取 ps 的 lstart（如
// "Mon Jan 2 15:04:05 2026"），转 RFC3339；解析失败留空。失败/无进程返回 nil。
func listWeChatProcesses() []WeChatProcessInfo {
	out, err := exec.Command("pgrep", "-f", `WeChat\.app|微信\.app`).Output()
	if err != nil {
		return nil
	}
	var infos []WeChatProcessInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		pidStr := strings.TrimSpace(line)
		if pidStr == "" {
			continue
		}
		pid, perr := strconv.Atoi(pidStr)
		if perr != nil || pid <= 0 {
			continue
		}
		info := WeChatProcessInfo{PID: pid}
		if comm, err := exec.Command("ps", "-p", pidStr, "-o", "comm=").Output(); err == nil {
			info.Path = strings.TrimSpace(string(comm))
			info.Name = filepath.Base(info.Path)
		}
		if ls, err := exec.Command("ps", "-p", pidStr, "-o", "lstart=").Output(); err == nil {
			if t, perr := time.Parse("Mon Jan 2 15:04:05 2006", strings.TrimSpace(string(ls))); perr == nil {
				info.StartTime = t.Format(time.RFC3339)
			}
		}
		infos = append(infos, info)
	}
	return infos
}

func killWeChatProcesses() []MaintenanceResult {
	procs := listWeChatProcesses()
	if len(procs) == 0 {
		return []MaintenanceResult{{
			Name:   "wechat_processes",
			Action: "kill_wechat",
			Status: MaintenanceStatusMissing,
		}}
	}
	results := make([]MaintenanceResult, 0, len(procs))
	for _, p := range procs {
		name := p.Name
		if name == "" {
			name = "wechat_process"
		}
		r := MaintenanceResult{
			Name:   name,
			Action: "kill_wechat",
			Path:   p.Path,
			Status: MaintenanceStatusOK,
			Error:  fmt.Sprintf("pid %d", p.PID),
		}
		if err := killProcess(p.PID); err != nil {
			r.Status = MaintenanceStatusError
			r.Error = fmt.Sprintf("pid %d: %s", p.PID, err.Error())
		}
		results = append(results, r)
	}
	return results
}

func openBrowser(url string) error {
	for _, exe := range darwinChromiumExecutables() {
		if _, err := os.Stat(exe); err != nil {
			continue
		}
		if err := exec.Command(exe, "--app="+url, "--new-window").Start(); err == nil {
			return nil
		}
	}
	return exec.Command("open", url).Start()
}

func darwinChromiumExecutables() []string {
	exes := []string{
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	}
	if home, err := os.UserHomeDir(); err == nil {
		exes = append(exes,
			filepath.Join(home, "Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"),
			filepath.Join(home, "Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
			filepath.Join(home, "Applications/Brave Browser.app/Contents/MacOS/Brave Browser"),
			filepath.Join(home, "Applications/Chromium.app/Contents/MacOS/Chromium"),
		)
	}
	return exes
}

func samplingAutoStartStatus() AutoStartStatus {
	path := darwinSamplingLaunchAgentPath()
	if path == "" {
		return AutoStartStatus{Supported: false, Message: "无法定位 LaunchAgents 目录"}
	}
	status := AutoStartStatus{Supported: true, Path: path}
	if _, err := os.Stat(path); err == nil {
		status.Enabled = true
		status.Message = "已配置 LaunchAgent，登录后会静默启动采样"
	} else if os.IsNotExist(err) {
		status.Message = "未配置系统开机自启动"
	} else {
		status.Error = err.Error()
	}
	return status
}

func installSamplingAutoStart() error {
	path := darwinSamplingLaunchAgentPath()
	if path == "" {
		return fmt.Errorf("无法定位 LaunchAgents 目录")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(AppDirPath(), 0o755); err != nil {
		return err
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.sushiro-overdose.sampler</string>
  <key>ProgramArguments</key>
  <array>
    <string>` + xmlEscape(exe) + `</string>
    <string>--queue-collector-child</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>StandardOutPath</key><string>` + xmlEscape(SamplingLogPath()) + `</string>
  <key>StandardErrorPath</key><string>` + xmlEscape(SamplingLogPath()) + `</string>
</dict>
</plist>
`
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return err
	}
	_ = exec.Command("launchctl", "unload", path).Run()
	return exec.Command("launchctl", "load", path).Run()
}

func removeSamplingAutoStart() error {
	path := darwinSamplingLaunchAgentPath()
	if path == "" {
		return nil
	}
	_ = exec.Command("launchctl", "unload", path).Run()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func darwinSamplingLaunchAgentPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "LaunchAgents", "com.sushiro-overdose.sampler.plist")
}

func xmlEscape(value string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	).Replace(value)
}

// --- MCP 助手自启（LaunchAgent，拉起 sushiro --mcp-daemon-child 确保 venv 就绪）---

func mcpAutoStartStatus() AutoStartStatus {
	path := darwinMCPLaunchAgentPath()
	if path == "" {
		return AutoStartStatus{Supported: false, Message: "无法定位 LaunchAgents 目录"}
	}
	status := AutoStartStatus{Supported: true, Path: path}
	if _, err := os.Stat(path); err == nil {
		status.Enabled = true
		status.Message = "已配置 LaunchAgent，登录后会准备 MCP 环境"
	} else if os.IsNotExist(err) {
		status.Message = "未配置 MCP 开机自启动"
	} else {
		status.Error = err.Error()
	}
	return status
}

func installMCPAutoStart() error {
	path := darwinMCPLaunchAgentPath()
	if path == "" {
		return fmt.Errorf("无法定位 LaunchAgents 目录")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// --mcp-daemon-child：确保 venv 就绪 + 写 claude_desktop_config 后退出（不常驻）。
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.sushiro-overdose.mcp</string>
  <key>ProgramArguments</key>
  <array>
    <string>` + xmlEscape(exe) + `</string>
    <string>--mcp-daemon-child</string>
  </array>
  <key>RunAtLoad</key><true/>
</dict>
</plist>
`
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return err
	}
	_ = exec.Command("launchctl", "unload", path).Run()
	return exec.Command("launchctl", "load", path).Run()
}

func removeMCPAutoStart() error {
	path := darwinMCPLaunchAgentPath()
	if path == "" {
		return nil
	}
	_ = exec.Command("launchctl", "unload", path).Run()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func darwinMCPLaunchAgentPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "LaunchAgents", "com.sushiro-overdose.mcp.plist")
}
