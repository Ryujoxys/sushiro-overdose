package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const defaultWebPort = 8081

func cmdWeb() {
	printBanner()

	if checkStaleProxy() {
		fmt.Println("已清除上次异常退出的系统代理设置")
	}

	setNotifier(BuildNotifierFromConfig())
	setWebCSRFToken(newWebCSRFToken())

	mux := http.NewServeMux()

	// Static
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/proxy.pac", handleProxyPAC)

	// Status & info
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/diagnostics", handleDiagnostics)
	mux.HandleFunc("/api/diagnostics/bundle", handleDiagBundle)
	mux.HandleFunc("/api/auth/probe", handleAuthProbe)
	mux.HandleFunc("/api/insights", handleInsights)
	mux.HandleFunc("/api/queue/trends", handleQueueTrends)
	mux.HandleFunc("/api/queue/stores", handleQueueLiveStores)
	mux.HandleFunc("/api/queue/store", handleQueueLiveStore)
	mux.HandleFunc("/api/queue/live", handleQueueLivePanel)
	mux.HandleFunc("/api/queue/alerts", handleQueueAlerts)
	mux.HandleFunc("/api/queue/areas", handleQueueLiveAreas)
	mux.HandleFunc("/api/update", handleUpdateCheck)
	mux.HandleFunc("/api/stores", handleStores)

	// Calendar & reservations
	mux.HandleFunc("/api/calendar", handleCalendar)
	mux.HandleFunc("/api/reservations", handleReservations)

	// Preferences
	mux.HandleFunc("/api/preferences", handlePreferences)

	// Notifications config
	mux.HandleFunc("/api/config", handleNotifyConfig)
	mux.HandleFunc("/api/notifications/test", handleNotificationTest)
	mux.HandleFunc("/api/repair-proxy", handleRepairProxy)
	mux.HandleFunc("/api/uninstall", handleUninstall)
	mux.HandleFunc("/api/processes/stop", handleStopProcesses)

	// Engine control
	mux.HandleFunc("/api/engine/state", handleEngineState)
	mux.HandleFunc("/api/engine/capture", handleEngineCapture)
	mux.HandleFunc("/api/engine/booking", handleEngineBooking)
	mux.HandleFunc("/api/engine/stop", handleEngineStop)
	mux.HandleFunc("/api/engine/logs", handleEngineLogs)

	// Sniper
	mux.HandleFunc("/api/sniper/start", handleSniperStart)
	mux.HandleFunc("/api/sniper/plan", handleSniperPlan)

	// Background sampling
	mux.HandleFunc("/api/sampling", handleSampling)
	mux.HandleFunc("/api/sampling/start", handleSamplingStart)
	mux.HandleFunc("/api/sampling/stop", handleSamplingStop)
	mux.HandleFunc("/api/sampling/once", handleSamplingOnce)
	mux.HandleFunc("/api/sampling/autostart", handleSamplingAutoStart)

	// SSE
	mux.HandleFunc("/api/events", handleEvents)

	port := findAvailablePort(defaultWebPort)
	setActiveWebPort(port)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: webSecurityMiddleware(mux),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Prepare settings for calendar/stores APIs if config exists
	tokens, ok := tryLoadConfig()
	if ok {
		prefs := LoadPreferences()
		settings := tokens.toSettingsWithPrefs(prefs)
		setWebSettings(settings)
	}
	sampler.StartIfAuto(ctx)

	go func() {
		<-ctx.Done()
		engine.Stop()
		sampler.Stop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	url := fmt.Sprintf("http://%s", addr)
	fmt.Printf("应用窗口启动于 %s\n", url)
	fmt.Println("会优先打开独立应用窗口；无法打开时回退到默认浏览器。按 Ctrl+C 退出")

	_ = OpenBrowser(url)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func findAvailablePort(preferred int) int {
	if port, ok := firstAvailableLocalPort(preferred, 100); ok {
		return port
	}
	return preferred
}

var (
	webSettingsMu sync.RWMutex
	webSettings   Settings
	webClient     *Client
	webPort       int

	webCSRFMu    sync.RWMutex
	webCSRFToken string
)

func setActiveWebPort(port int) {
	webSettingsMu.Lock()
	webPort = port
	webSettingsMu.Unlock()
}

func getActiveWebPort() int {
	webSettingsMu.RLock()
	defer webSettingsMu.RUnlock()
	return webPort
}

func setWebSettings(s Settings) {
	webSettingsMu.Lock()
	webSettings = s
	webClient = NewClient(s)
	webSettingsMu.Unlock()
}

func getWebSettings() Settings {
	webSettingsMu.RLock()
	defer webSettingsMu.RUnlock()
	return webSettings
}

func getWebClient() *Client {
	webSettingsMu.RLock()
	defer webSettingsMu.RUnlock()
	return webClient
}

func refreshWebClient() {
	tokens, err := loadLocalConfig()
	if err != nil {
		return
	}
	if err := tokens.validateForQuery(); err != nil {
		return
	}
	prefs := LoadPreferences()
	settings := tokens.toSettingsWithPrefs(prefs)
	setWebSettings(settings)
}

func newWebCSRFToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		fmt.Fprintln(os.Stderr, "生成 CSRF token 失败:", err)
		os.Exit(1)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func setWebCSRFToken(token string) {
	webCSRFMu.Lock()
	webCSRFToken = token
	webCSRFMu.Unlock()
}

func getWebCSRFToken() string {
	webCSRFMu.RLock()
	defer webCSRFMu.RUnlock()
	return webCSRFToken
}

func webSecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAllowedWebHost(r.Host) {
			writeError(w, http.StatusForbidden, "非法 Host")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && (r.Method == http.MethodPost || r.Method == http.MethodPut) {
			if !validWebOrigin(r) {
				writeError(w, http.StatusForbidden, "非法 Origin")
				return
			}
			if !validWebCSRF(r.Header.Get("X-Sushiro-CSRF")) {
				writeError(w, http.StatusForbidden, "CSRF 校验失败")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func validWebCSRF(got string) bool {
	want := getWebCSRFToken()
	if got == "" || want == "" || len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func validWebOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || origin == "null" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return normalizeWebHost(u.Host) == normalizeWebHost(r.Host) && isAllowedWebHost(u.Host)
}

func isAllowedWebHost(host string) bool {
	h := normalizeWebHost(host)
	if h == "" {
		return false
	}
	name, _, err := net.SplitHostPort(h)
	if err == nil {
		h = name
	}
	h = strings.Trim(h, "[]")
	return h == "127.0.0.1" || h == "localhost" || h == "::1"
}

func normalizeWebHost(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}
