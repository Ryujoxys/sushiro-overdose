//go:build windows

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// 抓包期间屏蔽到寿司郎域名的出站 QUIC 的防火墙规则名。
const quicBlockRuleName = "SushiroOverdoseBlockQUIC"

func powershellCommand(script string, args ...string) *exec.Cmd {
	return powershellCommandWithOptions(false, script, args...)
}

func hiddenPowerShellCommand(script string, args ...string) *exec.Cmd {
	return powershellCommandWithOptions(true, script, args...)
}

func powershellCommandWithOptions(windowHidden bool, script string, args ...string) *exec.Cmd {
	psArgs := []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass"}
	if windowHidden {
		psArgs = append(psArgs, "-WindowStyle", "Hidden")
	}
	psArgs = append(psArgs, "-Command", "& {\n"+script+"\n}")
	psArgs = append(psArgs, args...)
	cmd := exec.Command("powershell", psArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

func desktopNotification(title, message string) {
	script := `
Add-Type -AssemblyName System.Windows.Forms
$n = New-Object System.Windows.Forms.NotifyIcon
$n.Icon = [System.Drawing.SystemIcons]::Information
$n.Visible = $true
$n.ShowBalloonTip(5000, $args[0], $args[1], 'Info')
Start-Sleep -Seconds 6
$n.Dispose()
`
	cmd := hiddenPowerShellCommand(script, title, message)
	_ = cmd.Start()
}

func setSystemProxy(port int) error {
	if webPort := getActiveWebPort(); webPort > 0 {
		return setWindowsPACProxy(port, webPort)
	}
	return setWindowsManualProxy(port)
}

func setWindowsPACProxy(proxyPort, webPort int) error {
	pacURL := fmt.Sprintf("http://127.0.0.1:%d/proxy.pac?proxy=%d", webPort, proxyPort)
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	_ = runHiddenWindowsCommand("reg", "delete", key, "/v", "ProxyServer", "/f")
	if err := runHiddenWindowsCommand("reg", "add", key, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f"); err != nil {
		return fmt.Errorf("写入 ProxyEnable 失败: %w", err)
	}
	if err := runHiddenWindowsCommand("reg", "add", key, "/v", "AutoConfigURL", "/t", "REG_SZ", "/d", pacURL, "/f"); err != nil {
		return fmt.Errorf("写入 AutoConfigURL 失败: %w", err)
	}
	if err := runHiddenWindowsCommand("reg", "add", key, "/v", "AutoDetect", "/t", "REG_DWORD", "/d", "0", "/f"); err != nil {
		return fmt.Errorf("写入 AutoDetect 失败: %w", err)
	}
	if err := setWinHTTPAutoProxy(pacURL); err != nil {
		logMessage(time.Now(), "WinHTTP PAC 代理设置跳过: "+err.Error())
	}

	refreshProxySettings()
	blockSushiroQUIC()
	logMessage(time.Now(), fmt.Sprintf("Windows PAC 代理已设置: 仅 %s 走 127.0.0.1:%d，其它域名直连", sushiroHost, proxyPort))
	return nil
}

func setWindowsManualProxy(port int) error {
	proxyServer := fmt.Sprintf("http=127.0.0.1:%d;https=127.0.0.1:%d", port, port)
	proxyOverride := "<local>;localhost;127.*;::1;10.*;192.168.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*"
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	_ = runHiddenWindowsCommand("reg", "delete", key, "/v", "AutoConfigURL", "/f")
	if err := runHiddenWindowsCommand("reg", "add", key, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f"); err != nil {
		return fmt.Errorf("写入 ProxyEnable 失败: %w", err)
	}
	if err := runHiddenWindowsCommand("reg", "add", key, "/v", "ProxyServer", "/t", "REG_SZ", "/d", proxyServer, "/f"); err != nil {
		return fmt.Errorf("写入 ProxyServer 失败: %w", err)
	}
	if err := runHiddenWindowsCommand("reg", "add", key, "/v", "ProxyOverride", "/t", "REG_SZ", "/d", proxyOverride, "/f"); err != nil {
		return fmt.Errorf("写入 ProxyOverride 失败: %w", err)
	}
	if err := setWinHTTPProxy(proxyServer, proxyOverride); err != nil {
		logMessage(time.Now(), "WinHTTP 代理设置跳过: "+err.Error())
	}

	refreshProxySettings()
	blockSushiroQUIC()
	return nil
}

func clearSystemProxy() error {
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	err := runHiddenWindowsCommand("reg", "add", key, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f")
	_ = runHiddenWindowsCommand("reg", "delete", key, "/v", "AutoConfigURL", "/f")
	if resetErr := clearWinHTTPProxy(); resetErr != nil {
		logMessage(time.Now(), "WinHTTP 代理清理跳过: "+resetErr.Error())
	}
	unblockSushiroQUIC()
	refreshProxySettings()
	return err
}

// blockSushiroQUIC 屏蔽到寿司郎域名的出站 QUIC(UDP 443)，逼微信的 Chromium/XWeb
// 内核回退 TCP，从而能被只处理 TCP 的 MITM 代理解密。需要管理员权限；失败仅记录不阻断。
func blockSushiroQUIC() {
	_ = removeQUICBlockRule()
	ips := resolveSushiroIPs()
	if len(ips) == 0 {
		logMessage(time.Now(), "QUIC 屏蔽跳过: 无法解析 "+sushiroHost+" 的 IP")
		return
	}
	if err := runHiddenWindowsCommand("netsh", "advfirewall", "firewall", "add", "rule",
		"name="+quicBlockRuleName, "dir=out", "action=block",
		"protocol=UDP", "remoteport=443", "remoteip="+strings.Join(ips, ",")); err != nil {
		logMessage(time.Now(), "QUIC 屏蔽设置失败(可能需管理员权限): "+err.Error())
		return
	}
	logMessage(time.Now(), fmt.Sprintf("已屏蔽到 %s 的出站 QUIC(UDP 443)，强制微信走 TCP 以便抓包", sushiroHost))
}

func unblockSushiroQUIC() {
	if err := removeQUICBlockRule(); err != nil {
		logMessage(time.Now(), "QUIC 屏蔽清理跳过: "+err.Error())
	}
}

func removeQUICBlockRule() error {
	return runHiddenWindowsCommand("netsh", "advfirewall", "firewall", "delete", "rule", "name="+quicBlockRuleName)
}

func resolveSushiroIPs() []string {
	addrs, err := net.LookupIP(sushiroHost)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		s := a.String()
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func setWinHTTPProxy(proxyServer, proxyOverride string) error {
	return runHiddenWindowsCommand("netsh", "winhttp", "set", "proxy", "proxy-server="+proxyServer, "bypass-list="+proxyOverride)
}

func setWinHTTPAutoProxy(pacURL string) error {
	settings := fmt.Sprintf(`{"Proxy":"","ProxyBypass":"","AutoconfigUrl":%q,"AutoDetect":false}`, pacURL)
	if err := runHiddenWindowsCommand("netsh", "winhttp", "set", "advproxy", "setting-scope=user", "settings="+settings); err == nil {
		return nil
	} else if importErr := runHiddenWindowsCommand("netsh", "winhttp", "import", "proxy", "source=ie"); importErr != nil {
		return fmt.Errorf("advproxy=%w; import=%w", err, importErr)
	}
	return nil
}

func clearWinHTTPProxy() error {
	return runHiddenWindowsCommand("netsh", "winhttp", "reset", "proxy")
}

func runHiddenWindowsCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}

// refreshProxySettings notifies the system that proxy settings have changed
// so that applications using WinINet pick up the new settings immediately.
func refreshProxySettings() {
	script := `
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public class WinINet {
    [DllImport("wininet.dll", SetLastError=true)]
    public static extern bool InternetSetOption(IntPtr hInternet, int dwOption, IntPtr lpBuffer, int lpdwBufferLength);
    public const int INTERNET_OPTION_SETTINGS_CHANGED = 39;
    public const int INTERNET_OPTION_REFRESH = 37;
    public const int INTERNET_OPTION_PROXY_SETTINGS_CHANGED = 95;
    public static void Refresh() {
        InternetSetOption(IntPtr.Zero, INTERNET_OPTION_PROXY_SETTINGS_CHANGED, IntPtr.Zero, 0);
        InternetSetOption(IntPtr.Zero, INTERNET_OPTION_SETTINGS_CHANGED, IntPtr.Zero, 0);
        InternetSetOption(IntPtr.Zero, INTERNET_OPTION_REFRESH, IntPtr.Zero, 0);
    }
}
"@
[WinINet]::Refresh()
`
	cmd := powershellCommand(script)
	_ = cmd.Run()
}

func isCertTrusted() (bool, error) {
	thumbprint, err := localCACertSHA1Thumbprint()
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	script := `
$thumb = $args[0].ToUpperInvariant()
$blocked = @('Cert:\CurrentUser\Disallowed','Cert:\LocalMachine\Disallowed') |
  ForEach-Object { Get-ChildItem -Path $_ -ErrorAction SilentlyContinue } |
  Where-Object { $_.Thumbprint -eq $thumb } |
  Select-Object -First 1
if ($null -ne $blocked) { throw "certificate is present in Windows Disallowed store" }
$currentUser = Get-ChildItem -Path Cert:\CurrentUser\Root -ErrorAction SilentlyContinue | Where-Object { $_.Thumbprint -eq $thumb } | Select-Object -First 1
$localMachine = Get-ChildItem -Path Cert:\LocalMachine\Root -ErrorAction SilentlyContinue | Where-Object { $_.Thumbprint -eq $thumb } | Select-Object -First 1
if ($null -ne $currentUser -and $null -ne $localMachine) { Write-Output "trusted" }
`
	cmd := powershellCommand(script, thumbprint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("检查证书信任失败: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.Contains(string(out), "trusted"), nil
}

func installCert() error {
	dir := certDirPath()
	certPath := filepath.Join(dir, "ca.crt")
	thumbprint, err := localCACertSHA1Thumbprint()
	if err != nil {
		return fmt.Errorf("读取本地 CA 证书失败: %w", err)
	}
	userTrusted, _ := isCertTrustedInStore(`Cert:\CurrentUser\Root`, thumbprint)
	machineTrusted, _ := isCertTrustedInStore(`Cert:\LocalMachine\Root`, thumbprint)
	if userTrusted && machineTrusted {
		return nil
	}

	if !userTrusted {
		userScript := `
$path = $args[0]
$thumb = $args[1].ToUpperInvariant()
Get-ChildItem -Path Cert:\CurrentUser\Root |
  Where-Object { $_.Subject -like '*CN=Sushiro Proxy CA*' -and $_.Thumbprint -ne $thumb } |
  Remove-Item -ErrorAction SilentlyContinue
$existing = Get-ChildItem -Path Cert:\CurrentUser\Root | Where-Object { $_.Thumbprint -eq $thumb } | Select-Object -First 1
if ($null -eq $existing) {
  $imported = Import-Certificate -FilePath $path -CertStoreLocation Cert:\CurrentUser\Root
  if ($null -eq $imported) { throw "Import-Certificate returned no certificate" }
}
$cert = Get-ChildItem -Path Cert:\CurrentUser\Root | Where-Object { $_.Thumbprint -eq $thumb } | Select-Object -First 1
if ($null -eq $cert) { throw "certificate was not found in CurrentUser Root after import" }
`
		cmd := powershellCommand(userScript, certPath, thumbprint)
		out, err := cmd.CombinedOutput()
		if err != nil {
			fallback := exec.Command("certutil", "-f", "-user", "-addstore", "Root", certPath)
			fallback.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			fallbackOut, fallbackErr := fallback.CombinedOutput()
			if fallbackErr != nil {
				return fmt.Errorf("导入证书失败: powershell=%w: %s; certutil=%w: %s", err, strings.TrimSpace(string(out)), fallbackErr, strings.TrimSpace(string(fallbackOut)))
			}
		}
	}
	if machineTrusted, _ = isCertTrustedInStore(`Cert:\LocalMachine\Root`, thumbprint); !machineTrusted {
		if err := installCertToLocalMachine(certPath, thumbprint); err != nil {
			return fmt.Errorf("当前用户证书已安装，但机器级 Root 证书安装失败；Windows PC 微信可能仍不信任本地 CA，请允许管理员权限后重试: %w", err)
		}
	}
	trusted, trustErr := isCertTrusted()
	if trustErr != nil {
		return trustErr
	}
	if !trusted {
		return fmt.Errorf("证书已导入，但 Windows CurrentUser/LocalMachine Root 中没有同时找到匹配指纹 %s", thumbprint)
	}
	return nil
}

func isCertTrustedInStore(storePath, thumbprint string) (bool, error) {
	script := `
$path = $args[0]
$thumb = $args[1].ToUpperInvariant()
$cert = Get-ChildItem -Path $path -ErrorAction SilentlyContinue | Where-Object { $_.Thumbprint -eq $thumb } | Select-Object -First 1
if ($null -ne $cert) { Write-Output "trusted" }
`
	cmd := powershellCommand(script, storePath, thumbprint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.Contains(string(out), "trusted"), nil
}

func installCertToLocalMachine(certPath, thumbprint string) error {
	script := `
$path = $args[0]
$thumb = $args[1].ToUpperInvariant()
$existing = Get-ChildItem -Path Cert:\LocalMachine\Root -ErrorAction SilentlyContinue | Where-Object { $_.Thumbprint -eq $thumb } | Select-Object -First 1
if ($null -eq $existing) {
  $quotedPath = '"' + ($path -replace '"','\"') + '"'
  $p = Start-Process -FilePath certutil.exe -ArgumentList @('-f','-addstore','Root',$quotedPath) -Verb RunAs -Wait -PassThru
  if ($null -eq $p) { throw 'elevated certutil did not start' }
  if ($p.ExitCode -ne 0) { throw "elevated certutil failed with exit code $($p.ExitCode)" }
}
$cert = Get-ChildItem -Path Cert:\LocalMachine\Root -ErrorAction SilentlyContinue | Where-Object { $_.Thumbprint -eq $thumb } | Select-Object -First 1
if ($null -eq $cert) { throw 'certificate was not found in LocalMachine Root after import' }
`
	cmd := powershellCommand(script, certPath, thumbprint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func uninstallCert() error {
	thumbprint, thumbErr := localCACertSHA1Thumbprint()
	if thumbErr == nil {
		script := `
$thumb = $args[0].ToUpperInvariant()
Get-ChildItem -Path Cert:\CurrentUser\Root |
  Where-Object { $_.Thumbprint -eq $thumb -or $_.Subject -like '*CN=Sushiro Proxy CA*' } |
  Remove-Item -ErrorAction SilentlyContinue
$machineCert = Get-ChildItem -Path Cert:\LocalMachine\Root -ErrorAction SilentlyContinue |
  Where-Object { $_.Thumbprint -eq $thumb } |
  Select-Object -First 1
if ($null -ne $machineCert) {
  $p = Start-Process -FilePath certutil.exe -ArgumentList @('-delstore','Root',$thumb) -Verb RunAs -Wait -PassThru
  if ($null -eq $p) { throw 'elevated certutil did not start' }
  if ($p.ExitCode -ne 0) { throw "elevated certutil failed with exit code $($p.ExitCode)" }
}
$left = @('Cert:\CurrentUser\Root','Cert:\LocalMachine\Root') |
  ForEach-Object { Get-ChildItem -Path $_ -ErrorAction SilentlyContinue } |
  Where-Object { $_.Thumbprint -eq $thumb } |
  Select-Object -First 1
if ($null -ne $left) { throw 'certificate still exists after removal' }
`
		cmd := powershellCommand(script, thumbprint)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("删除证书失败: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	cmd := exec.Command("certutil", "-delstore", "-user", "Root", "Sushiro Proxy CA")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(strings.ToLower(msg), "cannot find") || strings.Contains(strings.ToLower(msg), "not found") {
			return nil
		}
		return fmt.Errorf("删除证书失败: %w: %s", err, msg)
	}
	return nil
}

func daemonProcessAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func killProcess(pid int) error {
	cmd := exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", pid))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func killRelatedAppProcesses(excludePID int) []MaintenanceResult {
	script := `
$exclude = [int]$args[0]
$matches = Get-CimInstance Win32_Process | Where-Object {
  $_.ProcessId -ne $exclude -and $_.ProcessId -ne $PID -and (
    $_.Name -like 'sushiro-overdose*.exe' -or
    $_.Name -like 'Sushiro-Overdose*.exe' -or
    $_.ExecutablePath -like '*\sushiro-overdose*.exe' -or
    $_.ExecutablePath -like '*\Sushiro-Overdose*.exe' -or
    $_.CommandLine -like '*sushiro-overdose*' -or
    $_.CommandLine -like '*Sushiro-Overdose*'
  )
}
foreach ($p in $matches) {
  $status = 'ok'
  $errorText = ''
  try {
    Stop-Process -Id $p.ProcessId -Force -ErrorAction Stop
  } catch {
    $status = 'error'
    $errorText = $_.Exception.Message
  }
  [pscustomobject]@{
    pid = $p.ProcessId
    name = $p.Name
    path = $p.ExecutablePath
    status = $status
    error = $errorText
  } | ConvertTo-Json -Compress
}
`
	cmd := powershellCommand(script, fmt.Sprintf("%d", excludePID))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return []MaintenanceResult{{
			Name:   "related_processes",
			Action: "kill_by_name",
			Status: maintenanceStatusError,
			Error:  fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(out))),
		}}
	}
	return parseRelatedProcessKillOutput(string(out))
}

func isProcessAlive(pid int) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	line := strings.TrimSpace(string(out))
	if strings.Contains(line, "No tasks") {
		return false
	}
	for _, field := range strings.Fields(line) {
		if field == fmt.Sprintf("%d", pid) {
			return true
		}
	}
	return false
}

func openBrowser(url string) error {
	for _, exe := range windowsChromiumExecutables() {
		if _, err := os.Stat(exe); err != nil {
			continue
		}
		cmd := exec.Command(exe, "--app="+url, "--new-window")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Start(); err == nil {
			return nil
		}
	}
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func windowsChromiumExecutables() []string {
	names := []string{"msedge.exe", "chrome.exe", "brave.exe"}
	out := []string{}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			out = append(out, path)
		}
	}
	localAppData := os.Getenv("LOCALAPPDATA")
	programFiles := os.Getenv("PROGRAMFILES")
	programFilesX86 := os.Getenv("PROGRAMFILES(X86)")
	appendCandidate := func(base string, parts ...string) {
		if base == "" {
			return
		}
		out = append(out, filepath.Join(append([]string{base}, parts...)...))
	}
	appendCandidate(localAppData, "Microsoft", "Edge", "Application", "msedge.exe")
	appendCandidate(programFilesX86, "Microsoft", "Edge", "Application", "msedge.exe")
	appendCandidate(programFiles, "Microsoft", "Edge", "Application", "msedge.exe")
	appendCandidate(localAppData, "Google", "Chrome", "Application", "chrome.exe")
	appendCandidate(programFiles, "Google", "Chrome", "Application", "chrome.exe")
	appendCandidate(programFilesX86, "Google", "Chrome", "Application", "chrome.exe")
	appendCandidate(localAppData, "BraveSoftware", "Brave-Browser", "Application", "brave.exe")
	appendCandidate(programFiles, "BraveSoftware", "Brave-Browser", "Application", "brave.exe")
	appendCandidate(programFilesX86, "BraveSoftware", "Brave-Browser", "Application", "brave.exe")
	return out
}

func samplingAutoStartStatus() AutoStartStatus {
	status := AutoStartStatus{
		Supported: true,
		Path:      `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\SushiroOverdoseSampler`,
	}
	cmd := exec.Command("reg", "query", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "SushiroOverdoseSampler")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err == nil {
		status.Enabled = true
		status.Message = "已配置当前用户开机静默启动采样"
	} else {
		status.Message = "未配置系统开机自启动"
	}
	return status
}

func installSamplingAutoStart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	value := `"` + exe + `" --sampler-daemon-child`
	cmd := exec.Command("reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "SushiroOverdoseSampler", "/t", "REG_SZ", "/d", value, "/f")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func removeSamplingAutoStart() error {
	cmd := exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "SushiroOverdoseSampler", "/f")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return nil
	}
	return nil
}
