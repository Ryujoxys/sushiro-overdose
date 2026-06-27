package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/platform"

import . "github.com/Ryujoxys/sushiro-overdose/internal/proxy"

import . "github.com/Ryujoxys/sushiro-overdose/internal/notify"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	diagnosticLogLines = 50
	diagnosticLogBytes = 64 * 1024
)

type Diagnostics struct {
	GeneratedAt   string                `json:"generated_at"`
	Version       string                `json:"version"`
	Platform      DiagnosticPlatform    `json:"platform"`
	Running       DiagnosticRunning     `json:"running"`
	Data          DiagnosticDataPaths   `json:"data"`
	Config        DiagnosticConfig      `json:"config"`
	Certificate   DiagnosticCertificate `json:"certificate"`
	Ports         []DiagnosticPort      `json:"ports"`
	ProxyMarker   DiagnosticProxyMarker `json:"proxy_marker"`
	SystemProxy   DiagnosticSystemProxy `json:"system_proxy"`
	ProxyChain    DiagnosticProxyChain  `json:"proxy_chain"`
	Network       DiagnosticNetwork     `json:"network"`
	Engine        EngineState           `json:"engine"`
	LogTail       []string              `json:"log_tail"`
	LogError      string                `json:"log_error,omitempty"`
	EngineLogTail []DiagnosticLogEntry  `json:"engine_log_tail"`
}

type DiagnosticPlatform struct {
	GOOS            string `json:"goos"`
	GOARCH          string `json:"goarch"`
	Quarantined     bool   `json:"quarantined,omitempty"`      // 仅 macOS：当前可执行文件是否被 Gatekeeper 隔离
	QuarantineError string `json:"quarantine_error,omitempty"` // 检测隔离属性时的错误（如 xattr 缺失）
}

type DiagnosticRunning struct {
	Running bool   `json:"running"`
	PID     string `json:"pid,omitempty"`
}

type DiagnosticDataPaths struct {
	AppDir          string `json:"app_dir"`
	ConfigPath      string `json:"config_path"`
	PreferencesPath string `json:"preferences_path"`
	NotifyPath      string `json:"notify_path"`
	LogPath         string `json:"log_path"`
	StatePath       string `json:"state_path"`
	CertDir         string `json:"cert_dir"`
}

type DiagnosticConfig struct {
	Path                 string   `json:"path"`
	Exists               bool     `json:"exists"`
	Complete             bool     `json:"complete"`
	QueryComplete        bool     `json:"query_complete"`
	ReservationComplete  bool     `json:"reservation_complete"`
	Missing              []string `json:"missing"`
	PhoneMasked          string   `json:"phone_masked,omitempty"`
	StoreCount           int      `json:"store_count"`
	SelectedStoreCount   int      `json:"selected_store_count"`
	NotificationChannels []string `json:"notification_channels"`
	Error                string   `json:"error,omitempty"`
}

type DiagnosticCertificate struct {
	Dir                 string `json:"dir"`
	CertPath            string `json:"cert_path"`
	KeyPath             string `json:"key_path"`
	CertExists          bool   `json:"cert_exists"`
	KeyExists           bool   `json:"key_exists"`
	Trusted             bool   `json:"trusted"`
	CurrentUserTrusted  bool   `json:"current_user_trusted,omitempty"`
	LocalMachineTrusted bool   `json:"local_machine_trusted,omitempty"`
	Disallowed          bool   `json:"disallowed,omitempty"`
	TrustError          string `json:"trust_error,omitempty"`
	Subject             string `json:"subject,omitempty"`
	NotBefore           string `json:"not_before,omitempty"`
	NotAfter            string `json:"not_after,omitempty"`
	ParseError          string `json:"parse_error,omitempty"`
}

type DiagnosticPort struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Available    bool   `json:"available"`
	InUse        bool   `json:"in_use"`
	Current      bool   `json:"current,omitempty"`
	FallbackPort int    `json:"fallback_port,omitempty"`
	Note         string `json:"note,omitempty"`
	Error        string `json:"error,omitempty"`
}

type DiagnosticProxyMarker struct {
	Path     string `json:"path"`
	Exists   bool   `json:"exists"`
	Active   bool   `json:"active"`
	Port     int    `json:"port,omitempty"`
	PID      int    `json:"pid,omitempty"`
	PIDAlive bool   `json:"pid_alive"`
	Stale    bool   `json:"stale"`
	SetAt    string `json:"set_at,omitempty"`
	Error    string `json:"error,omitempty"`
}

type DiagnosticNetwork struct {
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	Addresses []string `json:"addresses,omitempty"`
	Reachable bool     `json:"reachable"`
	LatencyMS int64    `json:"latency_ms,omitempty"`
	Error     string   `json:"error,omitempty"`
}

type DiagnosticSystemProxy struct {
	Available bool     `json:"available"`
	Summary   []string `json:"summary,omitempty"`
	Error     string   `json:"error,omitempty"`
}

type DiagnosticProxyChain struct {
	Checked            bool              `json:"checked"`
	OK                 bool              `json:"ok"`
	Port               int               `json:"port,omitempty"`
	Active             bool              `json:"active"`
	SystemProxyMatches bool              `json:"system_proxy_matches"`
	Summary            string            `json:"summary,omitempty"`
	Probes             []DiagnosticProbe `json:"probes,omitempty"`
}

type DiagnosticProbe struct {
	Name      string `json:"name"`
	OK        bool   `json:"ok"`
	Skipped   bool   `json:"skipped,omitempty"`
	Detail    string `json:"detail,omitempty"`
	LatencyMS int64  `json:"latency_ms,omitempty"`
}

type DiagnosticLogEntry struct {
	Time    string `json:"time"`
	Message string `json:"message"`
	Level   string `json:"level,omitempty"`
}

type NotificationTestResult struct {
	Channel string `json:"channel"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

func CollectDiagnostics() Diagnostics {
	logTail, logErr := readSanitizedLogTail(LogPath(), diagnosticLogLines)
	engineLogs := sanitizedEngineLogTail(engine.GetLogs(), diagnosticLogLines)
	certificate := collectCertificateDiagnostics()
	proxyMarker := collectProxyMarkerDiagnostics()
	systemProxy := collectSystemProxyDiagnostics()

	quarantined, qerr := IsQuarantined()
	platformField := DiagnosticPlatform{
		GOOS:   runtime.GOOS,
		GOARCH: runtime.GOARCH,
	}
	if quarantined {
		platformField.Quarantined = true
	}
	if qerr != nil {
		platformField.QuarantineError = qerr.Error()
	}

	d := Diagnostics{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Version:     Version,
		Platform:    platformField,
		Running: DiagnosticRunning{
			Running: isRunning(),
			PID:     readPID(),
		},
		Data: DiagnosticDataPaths{
			AppDir:          AppDirPath(),
			ConfigPath:      LocalConfigPath(),
			PreferencesPath: filepath.Join(AppDirPath(), "preferences.json"),
			NotifyPath:      NotifyConfigPath(),
			LogPath:         LogPath(),
			StatePath:       StateFilePath(),
			CertDir:         CertDirPath(),
		},
		Config:        collectConfigDiagnostics(),
		Certificate:   certificate,
		Ports:         collectPortDiagnostics(),
		ProxyMarker:   proxyMarker,
		SystemProxy:   systemProxy,
		ProxyChain:    collectProxyChainDiagnostics(proxyMarker, systemProxy, certificate),
		Network:       collectNetworkDiagnostics(),
		Engine:        engine.GetState(),
		LogTail:       logTail,
		EngineLogTail: engineLogs,
	}
	if logErr != nil {
		d.LogError = logErr.Error()
	}
	return d
}

func collectConfigDiagnostics() DiagnosticConfig {
	out := DiagnosticConfig{
		Path:    LocalConfigPath(),
		Missing: NewCapturedTokens().MissingFields(true),
	}
	if _, err := os.Stat(out.Path); err == nil {
		out.Exists = true
	} else if err != nil && !os.IsNotExist(err) {
		out.Error = err.Error()
	}

	tokens, err := LoadLocalConfig()
	if err != nil {
		if out.Error == "" && !os.IsNotExist(err) {
			out.Error = err.Error()
		}
		out.NotificationChannels = configuredNotificationChannels()
		return out
	}

	queryMissing := tokens.MissingFields(false)
	reservationMissing := tokens.MissingFields(true)
	out.QueryComplete = len(queryMissing) == 0
	out.ReservationComplete = len(reservationMissing) == 0
	out.Complete = out.ReservationComplete
	out.Missing = reservationMissing

	tokens.Lock()
	if strings.TrimSpace(tokens.PhoneNumber) != "" {
		out.PhoneMasked = MaskPhone(tokens.PhoneNumber)
	}
	out.StoreCount = len(tokens.StoreIDs)
	tokens.Unlock()

	prefs := LoadPreferences()
	out.SelectedStoreCount = len(prefs.SelectedStores)
	out.NotificationChannels = configuredNotificationChannels()
	return out
}

func configuredNotificationChannels() []string {
	channels := make([]string, 0, 4)
	for _, notifier := range BuildNotifierFromConfig().List() {
		channels = append(channels, notifier.Name())
	}
	sort.Strings(channels)
	return channels
}

func collectCertificateDiagnostics() DiagnosticCertificate {
	dir := CertDirPath()
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")
	out := DiagnosticCertificate{
		Dir:      dir,
		CertPath: certPath,
		KeyPath:  keyPath,
	}

	out.CertExists = fileExists(certPath)
	out.KeyExists = fileExists(keyPath)

	trusted, err := IsCertTrusted()
	out.Trusted = trusted
	if err != nil {
		out.TrustError = err.Error()
	}
	if runtime.GOOS == "windows" {
		if thumbprint, thumbErr := LocalCACertSHA1Thumbprint(); thumbErr == nil {
			applyWindowsCertificateTrustDetails(&out, thumbprint)
		}
	}

	data, err := os.ReadFile(certPath)
	if err != nil {
		if !os.IsNotExist(err) {
			out.ParseError = err.Error()
		}
		return out
	}
	block, _ := pem.Decode(data)
	if block == nil {
		out.ParseError = "PEM 解析失败"
		return out
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		out.ParseError = err.Error()
		return out
	}
	out.Subject = cert.Subject.String()
	out.NotBefore = cert.NotBefore.Format(time.RFC3339)
	out.NotAfter = cert.NotAfter.Format(time.RFC3339)
	return out
}

func collectPortDiagnostics() []DiagnosticPort {
	ports := []DiagnosticPort{
		{Name: "MITM 代理", Host: "127.0.0.1", Port: ProxyPort},
		{Name: "Web UI 默认端口", Host: "127.0.0.1", Port: defaultWebPort},
	}
	activeWebPort := GetActiveWebPort()
	for i := range ports {
		addr := fmt.Sprintf("%s:%d", ports[i].Host, ports[i].Port)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ports[i].Available = true
			_ = ln.Close()
			continue
		}
		ports[i].InUse = true
		ports[i].Error = err.Error()
		if ports[i].Port == activeWebPort {
			ports[i].Current = true
		}
		if ports[i].Port == ProxyPort {
			if fallback, ok := FirstAvailableLocalPort(ProxyPort, ProxyPortSearchLimit); ok && fallback != ProxyPort {
				ports[i].FallbackPort = fallback
				ports[i].Note = fmt.Sprintf("捕获代理会自动改用 127.0.0.1:%d", fallback)
			}
		}
	}
	return ports
}

func collectProxyMarkerDiagnostics() DiagnosticProxyMarker {
	path := proxyStatePath()
	out := DiagnosticProxyMarker{Path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			out.Error = err.Error()
		}
		return out
	}
	out.Exists = true

	var state proxyState
	if err := json.Unmarshal(data, &state); err != nil {
		out.Error = err.Error()
		return out
	}
	out.Active = state.Active
	out.Port = state.Port
	out.PID = state.PID
	if !state.SetAt.IsZero() {
		out.SetAt = state.SetAt.Format(time.RFC3339)
	}
	if state.PID > 0 {
		out.PIDAlive = IsProcessAlive(state.PID)
	}
	out.Stale = out.Active && state.PID > 0 && !out.PIDAlive
	return out
}

func collectNetworkDiagnostics() DiagnosticNetwork {
	out := DiagnosticNetwork{Host: SushiroHost, Port: 443}
	addrs, err := net.LookupHost(SushiroHost)
	if err != nil {
		out.Error = "DNS: " + err.Error()
		return out
	}
	out.Addresses = addrs

	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(SushiroHost, "443"), 3*time.Second)
	if err != nil {
		out.Error = "TCP: " + err.Error()
		return out
	}
	_ = conn.Close()
	out.Reachable = true
	out.LatencyMS = time.Since(start).Milliseconds()
	return out
}

func collectSystemProxyDiagnostics() DiagnosticSystemProxy {
	switch runtime.GOOS {
	case "darwin":
		return collectDarwinProxySummary()
	case "linux":
		return collectLinuxProxySummary()
	case "windows":
		return collectWindowsProxySummary()
	default:
		return DiagnosticSystemProxy{Available: false, Error: "当前平台未实现系统代理摘要"}
	}
}

func collectProxyChainDiagnostics(marker DiagnosticProxyMarker, systemProxy DiagnosticSystemProxy, cert DiagnosticCertificate) DiagnosticProxyChain {
	out := DiagnosticProxyChain{
		Port:   marker.Port,
		Active: marker.Active && marker.PIDAlive,
	}
	if !marker.Active || marker.Port == 0 {
		out.Summary = "捕获代理未运行；点击获取凭证后再打开诊断可检查代理链路"
		return out
	}
	out.Checked = true
	out.SystemProxyMatches = systemProxyMentionsPort(systemProxy, marker.Port)
	out.Probes = append(out.Probes, DiagnosticProbe{
		Name: "系统代理指向本应用",
		OK:   out.SystemProxyMatches,
		Detail: func() string {
			if out.SystemProxyMatches {
				return fmt.Sprintf("系统代理包含 127.0.0.1:%d", marker.Port)
			}
			return fmt.Sprintf("系统代理未指向当前捕获端口 127.0.0.1:%d", marker.Port)
		}(),
	})
	out.Probes = append(out.Probes, probeProxyTCP(marker.Port))
	out.Probes = append(out.Probes, probePlainHTTPProxy(marker.Port))
	out.Probes = append(out.Probes, probeSushiroMITMProxy(marker.Port, cert.CertPath))

	out.OK = true
	for _, probe := range out.Probes {
		if !probe.OK && !probe.Skipped {
			out.OK = false
			break
		}
	}
	if out.OK {
		out.Summary = "代理链路正常"
	} else {
		out.Summary = "代理链路存在异常，查看下方失败项"
	}
	return out
}

func systemProxyMentionsPort(systemProxy DiagnosticSystemProxy, port int) bool {
	if port <= 0 {
		return false
	}
	needles := []string{
		fmt.Sprintf("127.0.0.1:%d", port),
		fmt.Sprintf("localhost:%d", port),
		fmt.Sprintf("127.0.0.1 %d", port),
		fmt.Sprintf("localhost %d", port),
	}
	for _, line := range systemProxy.Summary {
		normalized := strings.ToLower(strings.Join(strings.Fields(line), " "))
		compact := strings.ReplaceAll(normalized, " ", "")
		hasLocalHost := strings.Contains(normalized, "127.0.0.1") || strings.Contains(normalized, "localhost")
		hasPort := strings.Contains(normalized, fmt.Sprintf("%d", port))
		if hasLocalHost && hasPort {
			return true
		}
		for _, needle := range needles {
			needle = strings.ToLower(needle)
			if strings.Contains(normalized, needle) || strings.Contains(compact, strings.ReplaceAll(needle, " ", "")) {
				return true
			}
		}
	}
	return false
}

func probeProxyTCP(port int) DiagnosticProbe {
	probe := DiagnosticProbe{Name: "本地代理端口"}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	probe.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		probe.Detail = err.Error()
		return probe
	}
	_ = conn.Close()
	probe.OK = true
	probe.Detail = "TCP 可连接"
	return probe
}

func probePlainHTTPProxy(port int) DiagnosticProbe {
	probe := DiagnosticProbe{Name: "普通 HTTP 代理透传"}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		probe.Skipped = true
		probe.Detail = "本地探针服务启动失败: " + err.Error()
		return probe
	}
	defer ln.Close()

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/__sushiro_proxy_probe__" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Sushiro-Proxy-Probe", "ok")
		_, _ = w.Write([]byte("ok"))
	})}
	go func() {
		_ = server.Serve(ln)
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	targetHost := ln.Addr().String()
	request := fmt.Sprintf("GET http://%s/__sushiro_proxy_probe__ HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", targetHost, targetHost)
	start := time.Now()
	resp, body, err := roundTripRawProxyRequest(port, request, 3*time.Second)
	probe.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		probe.Detail = err.Error()
		return probe
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("X-Sushiro-Proxy-Probe") != "ok" || strings.TrimSpace(string(body)) != "ok" {
		probe.Detail = fmt.Sprintf("响应异常: HTTP %d", resp.StatusCode)
		return probe
	}
	probe.OK = true
	probe.Detail = "本地 HTTP 请求已经通过代理透传"
	return probe
}

func probeSushiroMITMProxy(port int, certPath string) DiagnosticProbe {
	probe := DiagnosticProbe{Name: "寿司郎 HTTPS MITM 链路"}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		probe.Skipped = true
		probe.Detail = "读取本地 CA 失败: " + err.Error()
		return probe
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certPEM) {
		probe.Skipped = true
		probe.Detail = "本地 CA PEM 解析失败"
		return probe
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 3*time.Second)
	if err != nil {
		probe.LatencyMS = time.Since(start).Milliseconds()
		probe.Detail = "连接本地代理失败: " + err.Error()
		return probe
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(6 * time.Second))

	if _, err := fmt.Fprintf(conn, "CONNECT %s:443 HTTP/1.1\r\nHost: %s:443\r\n\r\n", SushiroHost, SushiroHost); err != nil {
		probe.LatencyMS = time.Since(start).Milliseconds()
		probe.Detail = "发送 CONNECT 失败: " + err.Error()
		return probe
	}
	connectReader := bufio.NewReader(conn)
	connectResp, err := http.ReadResponse(connectReader, nil)
	if err != nil {
		probe.LatencyMS = time.Since(start).Milliseconds()
		probe.Detail = "读取 CONNECT 响应失败: " + err.Error()
		return probe
	}
	if connectResp.Body != nil {
		connectResp.Body.Close()
	}
	if connectResp.StatusCode != http.StatusOK {
		probe.LatencyMS = time.Since(start).Milliseconds()
		probe.Detail = fmt.Sprintf("CONNECT 返回 HTTP %d", connectResp.StatusCode)
		return probe
	}

	tlsConn := tls.Client(NewBufferedConn(conn, connectReader), &tls.Config{
		ServerName: SushiroHost,
		RootCAs:    roots,
		MinVersion: tls.VersionTLS12,
	})
	if err := tlsConn.Handshake(); err != nil {
		probe.LatencyMS = time.Since(start).Milliseconds()
		probe.Detail = "TLS 握手失败: " + err.Error()
		return probe
	}
	defer tlsConn.Close()
	_ = tlsConn.SetDeadline(time.Now().Add(6 * time.Second))

	if _, err := fmt.Fprintf(tlsConn, "GET / HTTP/1.1\r\nHost: %s\r\nUser-Agent: sushiro-overdose-diagnostic/%s\r\nConnection: close\r\n\r\n", SushiroHost, Version); err != nil {
		probe.LatencyMS = time.Since(start).Milliseconds()
		probe.Detail = "发送 HTTPS 探针失败: " + err.Error()
		return probe
	}
	resp, err := http.ReadResponse(bufio.NewReader(tlsConn), nil)
	probe.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		probe.Detail = "读取 HTTPS 响应失败: " + err.Error()
		return probe
	}
	defer resp.Body.Close()
	probe.OK = true
	probe.Detail = fmt.Sprintf("CONNECT/TLS/上游响应正常: HTTP %d %s", resp.StatusCode, resp.Proto)
	return probe
}

func roundTripRawProxyRequest(port int, request string, timeout time.Duration) (*http.Response, []byte, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), timeout)
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if _, err := io.WriteString(conn, request); err != nil {
		return nil, nil, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		return nil, nil, err
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		resp.Body.Close()
		return nil, nil, readErr
	}
	return resp, body, nil
}

func collectDarwinProxySummary() DiagnosticSystemProxy {
	services, err := diagnosticDarwinNetworkServices()
	if err != nil {
		return DiagnosticSystemProxy{Available: false, Error: err.Error()}
	}
	if len(services) > 3 {
		services = services[:3]
	}
	out := DiagnosticSystemProxy{Available: true}
	for _, service := range services {
		webOut, webErr := exec.Command("networksetup", "-getwebproxy", service).CombinedOutput()
		secureOut, secureErr := exec.Command("networksetup", "-getsecurewebproxy", service).CombinedOutput()
		if webErr != nil || secureErr != nil {
			parts := []string{service + ": 读取失败"}
			if webErr != nil {
				parts = append(parts, strings.TrimSpace(string(webOut)))
			}
			if secureErr != nil {
				parts = append(parts, strings.TrimSpace(string(secureOut)))
			}
			out.Summary = append(out.Summary, strings.Join(nonEmptyStrings(parts), " "))
			continue
		}
		out.Summary = append(out.Summary, service+" HTTP["+compactProxyOutput(string(webOut))+"] HTTPS["+compactProxyOutput(string(secureOut))+"]")
	}
	return out
}

func diagnosticDarwinNetworkServices() ([]string, error) {
	out, err := exec.Command("networksetup", "-listallnetworkservices").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	services := make([]string, 0, len(lines))
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "*") {
			services = append(services, line)
		}
	}
	return services, nil
}

func collectLinuxProxySummary() DiagnosticSystemProxy {
	out := DiagnosticSystemProxy{Available: true}
	envKeys := []string{"http_proxy", "https_proxy", "HTTP_PROXY", "HTTPS_PROXY"}
	for _, key := range envKeys {
		if value := os.Getenv(key); value != "" {
			out.Summary = append(out.Summary, key+"="+SanitizeDiagnosticLine(value))
		}
	}
	if gsettings, err := exec.LookPath("gsettings"); err == nil {
		mode, modeErr := exec.Command(gsettings, "get", "org.gnome.system.proxy", "mode").CombinedOutput()
		httpHost, _ := exec.Command(gsettings, "get", "org.gnome.system.proxy.http", "host").CombinedOutput()
		httpPort, _ := exec.Command(gsettings, "get", "org.gnome.system.proxy.http", "port").CombinedOutput()
		httpsHost, _ := exec.Command(gsettings, "get", "org.gnome.system.proxy.https", "host").CombinedOutput()
		httpsPort, _ := exec.Command(gsettings, "get", "org.gnome.system.proxy.https", "port").CombinedOutput()
		if modeErr == nil {
			out.Summary = append(out.Summary, fmt.Sprintf("gsettings mode=%s http=%s:%s https=%s:%s",
				strings.TrimSpace(string(mode)),
				trimQuotes(strings.TrimSpace(string(httpHost))),
				strings.TrimSpace(string(httpPort)),
				trimQuotes(strings.TrimSpace(string(httpsHost))),
				strings.TrimSpace(string(httpsPort))),
			)
		}
	}
	if len(out.Summary) == 0 {
		out.Summary = []string{"未发现环境变量或 gsettings 代理配置"}
	}
	return out
}

func collectWindowsProxySummary() DiagnosticSystemProxy {
	script := `
$p = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings'
$enable = (Get-ItemProperty -Path $p -Name ProxyEnable -ErrorAction SilentlyContinue).ProxyEnable
$server = (Get-ItemProperty -Path $p -Name ProxyServer -ErrorAction SilentlyContinue).ProxyServer
$override = (Get-ItemProperty -Path $p -Name ProxyOverride -ErrorAction SilentlyContinue).ProxyOverride
$autoConfig = (Get-ItemProperty -Path $p -Name AutoConfigURL -ErrorAction SilentlyContinue).AutoConfigURL
$autoDetect = (Get-ItemProperty -Path $p -Name AutoDetect -ErrorAction SilentlyContinue).AutoDetect
Write-Output ("ProxyEnable={0}" -f $enable)
Write-Output ("ProxyServer={0}" -f $server)
Write-Output ("ProxyOverride={0}" -f $override)
Write-Output ("AutoConfigURL={0}" -f $autoConfig)
Write-Output ("AutoDetect={0}" -f $autoDetect)
`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return DiagnosticSystemProxy{Available: false, Error: strings.TrimSpace(string(out))}
	}
	summary := splitNonEmptyLines(string(out))
	if netshOut, netshErr := exec.Command("netsh", "winhttp", "show", "proxy").CombinedOutput(); netshErr == nil {
		for _, line := range splitNonEmptyLines(string(netshOut)) {
			summary = append(summary, "WinHTTP "+line)
		}
	}
	if netshOut, netshErr := exec.Command("netsh", "winhttp", "show", "advproxy").CombinedOutput(); netshErr == nil {
		for _, line := range splitNonEmptyLines(string(netshOut)) {
			summary = append(summary, "WinHTTP advproxy "+line)
		}
	}
	return DiagnosticSystemProxy{Available: true, Summary: summary}
}

func applyWindowsCertificateTrustDetails(out *DiagnosticCertificate, thumbprint string) {
	script := `
$thumb = $args[0].ToUpperInvariant()
$cu = Get-ChildItem -Path Cert:\CurrentUser\Root -ErrorAction SilentlyContinue | Where-Object { $_.Thumbprint -eq $thumb } | Select-Object -First 1
$lm = Get-ChildItem -Path Cert:\LocalMachine\Root -ErrorAction SilentlyContinue | Where-Object { $_.Thumbprint -eq $thumb } | Select-Object -First 1
$bad = @('Cert:\CurrentUser\Disallowed','Cert:\LocalMachine\Disallowed') |
  ForEach-Object { Get-ChildItem -Path $_ -ErrorAction SilentlyContinue } |
  Where-Object { $_.Thumbprint -eq $thumb } |
  Select-Object -First 1
[pscustomobject]@{
  current_user = ($null -ne $cu)
  local_machine = ($null -ne $lm)
  disallowed = ($null -ne $bad)
} | ConvertTo-Json -Compress
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "& {\n"+script+"\n}", thumbprint)
	data, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	var parsed struct {
		CurrentUser  bool `json:"current_user"`
		LocalMachine bool `json:"local_machine"`
		Disallowed   bool `json:"disallowed"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &parsed); err != nil {
		return
	}
	out.CurrentUserTrusted = parsed.CurrentUser
	out.LocalMachineTrusted = parsed.LocalMachine
	out.Disallowed = parsed.Disallowed
}

func readSanitizedLogTail(path string, maxLines int) ([]string, error) {
	lines, err := readTailLines(path, maxLines, diagnosticLogBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	for i := range lines {
		lines[i] = SanitizeDiagnosticLine(lines[i])
	}
	return lines, nil
}

func readTailLines(path string, maxLines int, maxBytes int64) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	start := int64(0)
	if size > maxBytes {
		start = size - maxBytes
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	text := string(data)
	if start > 0 {
		if idx := strings.IndexByte(text, '\n'); idx >= 0 {
			text = text[idx+1:]
		}
	}
	text = strings.TrimRight(text, "\r\n")
	if text == "" {
		return nil, nil
	}
	lines := strings.Split(text, "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return lines, nil
}

func sanitizedEngineLogTail(entries []LogEntry, maxLines int) []DiagnosticLogEntry {
	if len(entries) > maxLines {
		entries = entries[len(entries)-maxLines:]
	}
	out := make([]DiagnosticLogEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, DiagnosticLogEntry{
			Time:    entry.Time,
			Message: SanitizeDiagnosticLine(entry.Message),
			Level:   entry.Level,
		})
	}
	return out
}

func runNotificationTest(ctx context.Context, onlyChannels ...string) ([]NotificationTestResult, bool) {
	notifiers := BuildNotifierFromConfig().List()
	if len(onlyChannels) > 0 {
		wanted := map[string]bool{}
		for _, channel := range onlyChannels {
			channel = strings.ToLower(strings.TrimSpace(channel))
			if channel != "" && channel != "all" {
				wanted[channel] = true
			}
		}
		if len(wanted) > 0 {
			filtered := make([]Notifier, 0, len(notifiers))
			for _, notifier := range notifiers {
				if wanted[notifier.Name()] {
					filtered = append(filtered, notifier)
				}
			}
			notifiers = filtered
		}
	}
	if len(notifiers) == 0 {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	results := make([]NotificationTestResult, 0, len(notifiers))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, notifier := range notifiers {
		wg.Add(1)
		go func(n Notifier) {
			defer wg.Done()
			result := NotificationTestResult{Channel: n.Name()}
			err := n.Send(ctx, "寿司郎通知测试", "这是一条来自 sushiro-overdose 的测试通知。")
			if err != nil {
				result.Error = err.Error()
			} else {
				result.OK = true
			}
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(notifier)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool { return results[i].Channel < results[j].Channel })
	ok := true
	for _, result := range results {
		if !result.OK {
			ok = false
			break
		}
	}
	return results, ok
}

func handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	writeJSON(w, CollectDiagnostics())
}

// handleCertCheck 是证书一键自检：只查证书文件存在性 + 系统信任库状态（毫秒级、不依赖代理运行），
// 让用户在通行证状态区直接看到「证书已信任」绿灯 / 「未信任/未生成」红灯，不必进全量诊断翻找。
func handleCertCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	writeJSON(w, collectCertificateDiagnostics())
}

func handleNotificationTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var req struct {
		Channel  string   `json:"channel"`
		Channels []string `json:"channels"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	channels := req.Channels
	if req.Channel != "" {
		channels = append(channels, req.Channel)
	}
	results, ok := runNotificationTest(r.Context(), channels...)
	if len(results) == 0 {
		writeError(w, http.StatusBadRequest, "未配置通知渠道")
		return
	}
	status := http.StatusOK
	if !ok {
		status = http.StatusBadGateway
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"ok":      ok,
		"results": results,
	})
}

func cmdDoctor() {
	printDiagnostics(os.Stdout, CollectDiagnostics())
}

func printDiagnostics(w io.Writer, d Diagnostics) {
	fmt.Fprintf(w, "sushiro doctor (%s)\n", d.GeneratedAt)
	fmt.Fprintf(w, "版本: %s\n", d.Version)
	fmt.Fprintf(w, "平台: %s/%s\n", d.Platform.GOOS, d.Platform.GOARCH)
	fmt.Fprintf(w, "运行: %t", d.Running.Running)
	if d.Running.PID != "" {
		fmt.Fprintf(w, " (PID %s)", d.Running.PID)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "数据目录: %s\n", d.Data.AppDir)
	fmt.Fprintf(w, "配置: complete=%t query=%t reservation=%t path=%s\n", d.Config.Complete, d.Config.QueryComplete, d.Config.ReservationComplete, d.Config.Path)
	if len(d.Config.Missing) > 0 {
		fmt.Fprintf(w, "缺失项: %s\n", strings.Join(d.Config.Missing, ", "))
	}
	if d.Config.Error != "" {
		fmt.Fprintf(w, "配置错误: %s\n", d.Config.Error)
	}
	if d.Config.PhoneMasked != "" {
		fmt.Fprintf(w, "手机号: %s\n", d.Config.PhoneMasked)
	}
	fmt.Fprintf(w, "门店数量: %d", d.Config.StoreCount)
	if d.Config.SelectedStoreCount > 0 {
		fmt.Fprintf(w, " (已选择 %d)", d.Config.SelectedStoreCount)
	}
	fmt.Fprintln(w)
	if len(d.Config.NotificationChannels) > 0 {
		fmt.Fprintf(w, "通知渠道: %s\n", strings.Join(d.Config.NotificationChannels, ", "))
	} else {
		fmt.Fprintln(w, "通知渠道: 未配置")
	}
	fmt.Fprintf(w, "证书: cert=%t key=%t trusted=%t path=%s\n", d.Certificate.CertExists, d.Certificate.KeyExists, d.Certificate.Trusted, d.Certificate.CertPath)
	if d.Platform.GOOS == "windows" {
		fmt.Fprintf(w, "Windows 证书信任: CurrentUser=%t LocalMachine=%t Disallowed=%t\n", d.Certificate.CurrentUserTrusted, d.Certificate.LocalMachineTrusted, d.Certificate.Disallowed)
	}
	if d.Certificate.TrustError != "" {
		fmt.Fprintf(w, "证书信任检查错误: %s\n", d.Certificate.TrustError)
	}
	if d.Certificate.ParseError != "" {
		fmt.Fprintf(w, "证书解析错误: %s\n", d.Certificate.ParseError)
	}
	if d.Certificate.NotAfter != "" {
		fmt.Fprintf(w, "证书有效期至: %s\n", d.Certificate.NotAfter)
	}
	fmt.Fprintln(w, "端口检查:")
	for _, p := range d.Ports {
		state := "可用"
		if p.Current {
			state = "当前使用中"
		} else if p.InUse {
			state = "占用"
		}
		fmt.Fprintf(w, "  %s %s:%d %s", p.Name, p.Host, p.Port, state)
		if p.Error != "" {
			fmt.Fprintf(w, " (%s)", p.Error)
		}
		if p.Note != "" {
			fmt.Fprintf(w, " - %s", p.Note)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "代理 marker: exists=%t active=%t stale=%t path=%s\n", d.ProxyMarker.Exists, d.ProxyMarker.Active, d.ProxyMarker.Stale, d.ProxyMarker.Path)
	if d.ProxyMarker.Exists {
		fmt.Fprintf(w, "代理 marker 详情: port=%d pid=%d pid_alive=%t set_at=%s\n", d.ProxyMarker.Port, d.ProxyMarker.PID, d.ProxyMarker.PIDAlive, d.ProxyMarker.SetAt)
	}
	if d.ProxyMarker.Error != "" {
		fmt.Fprintf(w, "代理 marker 错误: %s\n", d.ProxyMarker.Error)
	}
	if d.SystemProxy.Available {
		fmt.Fprintln(w, "系统代理摘要:")
		for _, line := range d.SystemProxy.Summary {
			fmt.Fprintf(w, "  %s\n", line)
		}
	} else if d.SystemProxy.Error != "" {
		fmt.Fprintf(w, "系统代理摘要: %s\n", d.SystemProxy.Error)
	}
	fmt.Fprintf(w, "代理链路: %s", d.ProxyChain.Summary)
	if d.ProxyChain.Checked {
		fmt.Fprintf(w, " port=%d ok=%t system_proxy_matches=%t", d.ProxyChain.Port, d.ProxyChain.OK, d.ProxyChain.SystemProxyMatches)
	}
	fmt.Fprintln(w)
	for _, probe := range d.ProxyChain.Probes {
		state := "异常"
		if probe.Skipped {
			state = "跳过"
		} else if probe.OK {
			state = "正常"
		}
		fmt.Fprintf(w, "  %s %s", probe.Name, state)
		if probe.LatencyMS > 0 {
			fmt.Fprintf(w, " %dms", probe.LatencyMS)
		}
		if probe.Detail != "" {
			fmt.Fprintf(w, " - %s", probe.Detail)
		}
		fmt.Fprintln(w)
	}
	if d.Network.Reachable {
		fmt.Fprintf(w, "网络连通性: %s:%d 可达 (%dms)\n", d.Network.Host, d.Network.Port, d.Network.LatencyMS)
	} else {
		fmt.Fprintf(w, "网络连通性: %s:%d 不可达", d.Network.Host, d.Network.Port)
		if d.Network.Error != "" {
			fmt.Fprintf(w, " (%s)", d.Network.Error)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "引擎: %s %s\n", d.Engine.Status, d.Engine.Message)
	if d.LogError != "" {
		fmt.Fprintf(w, "日志读取错误: %s\n", d.LogError)
	}
	if len(d.LogTail) > 0 {
		fmt.Fprintln(w, "最近日志:")
		for _, line := range d.LogTail {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}
	if len(d.EngineLogTail) > 0 {
		fmt.Fprintln(w, "最近 Web 引擎日志:")
		for _, entry := range d.EngineLogTail {
			if entry.Level != "" {
				fmt.Fprintf(w, "  [%s] [%s] %s\n", entry.Time, entry.Level, entry.Message)
			} else {
				fmt.Fprintf(w, "  [%s] %s\n", entry.Time, entry.Message)
			}
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func splitNonEmptyLines(s string) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, SanitizeDiagnosticLine(line))
		}
	}
	return out
}

func compactProxyOutput(s string) string {
	lines := splitNonEmptyLines(s)
	for i := range lines {
		lines[i] = strings.Join(strings.Fields(lines[i]), " ")
	}
	return strings.Join(lines, "; ")
}

func nonEmptyStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func trimQuotes(s string) string {
	return strings.Trim(s, `'"`)
}
