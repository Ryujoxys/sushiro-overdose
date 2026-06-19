//go:build linux

package platform

import . "github.com/Ryujoxys/sushiro-overdose/internal/proxy"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func desktopNotification(title, message string) {
	_ = exec.Command("notify-send", title, message).Run()
}

func setSystemProxy(port int) error {
	p := fmt.Sprintf("127.0.0.1:%d", port)

	// Set environment variables as baseline
	_ = os.Setenv("http_proxy", "http://"+p)
	_ = os.Setenv("https_proxy", "http://"+p)
	_ = os.Setenv("HTTP_PROXY", "http://"+p)
	_ = os.Setenv("HTTPS_PROXY", "http://"+p)

	// Try GNOME/KDE system proxy via gsettings
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "host", "127.0.0.1").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "port", fmt.Sprintf("%d", port)).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "host", "127.0.0.1").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "port", fmt.Sprintf("%d", port)).Run()

	return nil
}

func clearSystemProxy() error {
	_ = os.Unsetenv("http_proxy")
	_ = os.Unsetenv("https_proxy")
	_ = os.Unsetenv("HTTP_PROXY")
	_ = os.Unsetenv("HTTPS_PROXY")

	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()

	return nil
}

func isCertTrusted() (bool, error) {
	dir := CertDirPath()
	certPath := filepath.Join(dir, "ca.crt")
	if _, err := os.Stat(certPath); err != nil {
		return false, nil
	}
	_, err := os.Stat("/usr/local/share/ca-certificates/sushiro-proxy.crt")
	return err == nil, nil
}

func installCert() error {
	dir := CertDirPath()
	certPath := filepath.Join(dir, "ca.crt")
	target := "/usr/local/share/ca-certificates/sushiro-proxy.crt"

	src, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("read cert: %w", err)
	}
	if err := os.WriteFile(target, src, 0o644); err != nil {
		return fmt.Errorf("write cert (may need sudo): %w", err)
	}
	return exec.Command("update-ca-certificates").Run()
}

func uninstallCert() error {
	target := "/usr/local/share/ca-certificates/sushiro-proxy.crt"
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove cert (may need sudo): %w", err)
	}
	return exec.Command("update-ca-certificates").Run()
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

// isQuarantined 在 Linux 上恒返回未隔离（Gatekeeper 是 macOS 专属机制）。
func isQuarantined() (bool, error) {
	return false, nil
}

// Linux 无微信小程序客户端，枚举恒空、杀进程恒 missing。
func listWeChatProcesses() []WeChatProcessInfo { return nil }
func killWeChatProcesses() []MaintenanceResult {
	return []MaintenanceResult{{
		Name:   "wechat_processes",
		Action: "kill_wechat",
		Status: MaintenanceStatusMissing,
	}}
}

func openBrowser(url string) error {
	for _, name := range []string{"microsoft-edge", "google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "brave-browser"} {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		if err := exec.Command(path, "--app="+url, "--new-window").Start(); err == nil {
			return nil
		}
	}
	return exec.Command("xdg-open", url).Start()
}

func samplingAutoStartStatus() AutoStartStatus {
	path := linuxSamplingServicePath()
	status := AutoStartStatus{Supported: true, Path: path}
	if _, err := os.Stat(path); err == nil {
		status.Enabled = true
		status.Message = "已配置 systemd user 开机启动采样"
	} else if os.IsNotExist(err) {
		status.Message = "未配置系统开机自启动"
	} else {
		status.Error = err.Error()
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		status.Supported = false
		status.Message = "未找到 systemctl，当前环境不支持自动配置开机启动"
	}
	return status
}

func installSamplingAutoStart() error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	path := linuxSamplingServicePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	unit := `[Unit]
Description=Sushiro Overdose sampler

[Service]
Type=simple
ExecStart=` + exe + ` --sampler-daemon-child
Restart=on-failure

[Install]
WantedBy=default.target
`
	if err := os.WriteFile(path, []byte(unit), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return exec.Command("systemctl", "--user", "enable", "--now", "sushiro-overdose-sampler.service").Run()
}

func removeSamplingAutoStart() error {
	if _, err := exec.LookPath("systemctl"); err == nil {
		_ = exec.Command("systemctl", "--user", "disable", "--now", "sushiro-overdose-sampler.service").Run()
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	}
	if err := os.Remove(linuxSamplingServicePath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func linuxSamplingServicePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(AppDirPath(), "sushiro-overdose-sampler.service")
	}
	return filepath.Join(home, ".config", "systemd", "user", "sushiro-overdose-sampler.service")
}
