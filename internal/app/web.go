package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/platform"

import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import . "github.com/Ryujoxys/sushiro-overdose/internal/notify"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

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

// defaultWebPort 是 Web UI 的起始端口。用高位端口（39871）避开 8080/8081/3000/5000
// 等常见开发工具端口；若被占会自动递增到下一个可用端口（见 findAvailablePort）。
const defaultWebPort = 39871

// cmdWeb 启动本地 Web UI：注册全部 HTTP 路由、选定可用端口、装上安全中间件、
// 拉起后台采集/调度协程，最后阻塞在 server.ListenAndServe 上。
// 设计上只监听 127.0.0.1（loopback），不对外暴露，配合 webSecurityMiddleware 做请求准入。
func cmdWeb() {
	printBanner()

	// checkStaleProxy：上次进程异常退出（崩溃/被杀）来不及清理系统代理时，这里兜底清掉，
	// 否则系统仍指向已关闭的代理端口，会导致整机联网异常。
	if checkStaleProxy() {
		fmt.Println("已清除上次异常退出的系统代理设置")
	}

	setNotifier(BuildNotifierFromConfig())
	// 进程级 CSRF token：每次启动重新生成，旧页面里的 token 会失效（需刷新页面）。
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
	mux.HandleFunc("/api/auth/import", handleAuthImport)
	mux.HandleFunc("/api/auth/reset", handleAuthReset)
	mux.HandleFunc("/api/insights", handleInsights)
	mux.HandleFunc("/api/queue/dashboard", handleQueueDashboard)
	mux.HandleFunc("/api/queue/trends", handleQueueTrends)
	mux.HandleFunc("/api/queue/stores", handleQueueLiveStores)
	mux.HandleFunc("/api/queue/store", handleQueueLiveStore)
	mux.HandleFunc("/api/queue/live", handleQueueLivePanel)
	mux.HandleFunc("/api/queue/advisor", handleQueueAdvisor)
	mux.HandleFunc("/api/queue/pressure/curve", handleQueuePressureCurve)
	mux.HandleFunc("/api/queue/plan", handleQueuePlan)
	mux.HandleFunc("/api/queue/alerts", handleQueueAlerts)
	mux.HandleFunc("/api/queue/alerts/status", handleQueueAlertStatus)
	mux.HandleFunc("/api/queue/areas", handleQueueLiveAreas)
	mux.HandleFunc("/api/queue/baseline", handleQueueBaseline)
	mux.HandleFunc("/api/cloud/auth", handleCloudAuth)
	mux.HandleFunc("/api/cloud/auth/start", handleCloudAuthStart)
	mux.HandleFunc("/api/cloud/auth/callback", handleCloudAuthCallback)
	mux.HandleFunc("/api/cloud/auth/logout", handleCloudAuthLogout)
	mux.HandleFunc("/api/cloud/auth/test", handleCloudAuthTest)
	mux.HandleFunc("/api/update", handleUpdateCheck)
	mux.HandleFunc("/api/stores", handleStores)

	// Calendar & reservations
	mux.HandleFunc("/api/calendar", handleCalendar)
	mux.HandleFunc("/api/reservations", handleReservations)
	mux.HandleFunc("/api/reservations/local", handleLocalReservation)
	mux.HandleFunc("/api/reservations/cancel", handleCancelReservation)
	mux.HandleFunc("/api/queue/ticket", handleQueueTicket)
	mux.HandleFunc("/api/queue/ticket/status", handleQueueTicketStatus)
	mux.HandleFunc("/api/queue/ticket/cancel", handleCancelNetTicket)
	mux.HandleFunc("/api/queue/ticket/plan", handleNetTicketPlan)
	mux.HandleFunc("/api/queue/ticket/routine", handleNetTicketRoutine)

	// Preferences
	mux.HandleFunc("/api/preferences", handlePreferences)

	// Notifications config
	mux.HandleFunc("/api/config", handleNotifyConfig)
	mux.HandleFunc("/api/mobile-ua", handleMobileUA)
	mux.HandleFunc("/api/mobile-ua/capture/start", handleMobileUACaptureStart)
	mux.HandleFunc("/api/mobile-ua/capture/stop", handleMobileUACaptureStop)
	mux.HandleFunc("/api/mobile-auth", handleMobileAuth)
	mux.HandleFunc("/api/mobile-auth/start", handleMobileAuthStart)
	mux.HandleFunc("/api/mobile-auth/stop", handleMobileAuthStop)
	mux.HandleFunc("/api/discovery", handleDiscoveryConfig)
	mux.HandleFunc("/api/discovery/records", handleDiscoveryRecords)
	mux.HandleFunc("/api/discovery/clear", handleDiscoveryClear)
	mux.HandleFunc("/api/notifications/test", handleNotificationTest)
	mux.HandleFunc("/api/repair-proxy", handleRepairProxy)
	mux.HandleFunc("/api/uninstall", handleUninstall)
	mux.HandleFunc("/api/processes/stop", handleStopProcesses)
	mux.HandleFunc("/api/wechat/kill", handleKillWeChat)

	// Engine control
	mux.HandleFunc("/api/engine/state", handleEngineState)
	mux.HandleFunc("/api/engine/capture", handleEngineCapture)
	mux.HandleFunc("/api/engine/booking", handleEngineBooking)
	mux.HandleFunc("/api/engine/stop", handleEngineStop)
	mux.HandleFunc("/api/engine/reset", handleEngineReset)
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
	SetActiveWebPort(port)
	// 只绑定 loopback：本 UI 只服务本机浏览器/应用窗口，绝不对外网卡监听，
	// 从网络层就杜绝局域网内其他设备直接访问（手机抓包另有 0.0.0.0 代理，与此无关）。
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: webSecurityMiddleware(mux),
		// 超时设置：抵御慢速攻击/异常连接挂死导致的 goroutine 泄漏。
		// 注意不设 WriteTimeout —— SSE (/api/events) 是长连接，WriteTimeout 会掐断它；
		// ReadHeaderTimeout 是安全相关的底线（慢速 header 让连接挂死），必须设。
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Prepare settings for calendar/stores APIs if config exists
	tokens, ok := tryLoadConfig()
	if ok {
		prefs := LoadPreferences()
		settings := tokens.ToSettingsWithPrefs(prefs)
		setWebSettings(settings)
	}
	sampler.StartIfAuto(ctx)
	netTicketSched.Start(ctx)
	queueBaselineCollector.Start(ctx)

	go func() {
		<-ctx.Done()
		engine.Stop()
		sampler.Stop()
		mobileUACapture.stop()
		mobileAuthCapture.stop("")
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

// findAvailablePort 从 preferred（defaultWebPort）开始向上最多探 100 个端口，
// 返回第一个能 bind 的。都占满时退回 preferred（随后 ListenAndServe 会以真实错误失败），
// 避免「端口被占直接 fatal」让用户毫无排查线索。
func findAvailablePort(preferred int) int {
	if port, ok := FirstAvailableLocalPort(preferred, 100); ok {
		return port
	}
	return preferred
}

var (
	// webSettingsMu 保护 webSettings 与 webClient 两个相关字段：写时持写锁同时刷新两者，
	// 读时持读锁整体返回，避免调用方拿到 client 却用了被并发清空的 settings。
	webSettingsMu sync.RWMutex
	webSettings   Settings
	webClient     *Client

	// webCSRFMu 保护 webCSRFToken。token 在进程启动时写一次，校验时高频读，故用 RWMutex。
	webCSRFMu    sync.RWMutex
	webCSRFToken string
)

func setWebSettings(s Settings) {
	webSettingsMu.Lock()
	webSettings = s
	webClient = NewClient(s)
	webSettingsMu.Unlock()
}

func clearWebSettings() {
	webSettingsMu.Lock()
	webSettings = Settings{}
	webClient = nil
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

// refreshWebClient 在写操作前重新从磁盘读取凭证配置并重建 client。
// 凭证可能在 Web 之外被更新（如代理抓包刚捕获到），所以每次敏感写操作都先 reload：
// 读不到 / 校验不过就 clearWebSettings，让上层返回「请先获取凭证」。
func refreshWebClient() {
	tokens, err := LoadLocalConfig()
	if err != nil {
		clearWebSettings()
		return
	}
	if err := tokens.ValidateForQuery(); err != nil {
		clearWebSettings()
		return
	}
	prefs := LoadPreferences()
	settings := tokens.ToSettingsWithPrefs(prefs)
	setWebSettings(settings)
}

// newWebCSRFToken 用 crypto/rand 生成 32 字节随机数后 base64 编码，作为本进程的 CSRF token。
// 失败（系统熵源不可用）视为致命错误直接退出——拿不到强随机就不应继续提供 Web 服务。
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

// webSecurityMiddleware 是所有请求的准入闸门，按顺序做两件事：
//  1. Host 头校验（isAllowedWebHost）：只接受 127.0.0.1/localhost/::1，防止 DNS 重绑定（DNS rebinding）
//     把外部域名解析到本机 loopback 再借 Host 绕过同源策略。
//  2. 对 /api/ 下的写方法（POST/PUT）叠加双重校验：Origin 同源（validWebOrigin）+ 自定义 CSRF 头匹配
//     （validWebCSRF）。GET 等读请求不要求 CSRF 头，因为它们不应有副作用。
//
// 这套组合挡住了「外部网站用浏览器里用户的会话对本 UI 发起写请求」这类 CSRF 攻击。
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
				// 常见诱因：应用重启后 token 轮换，浏览器里还开着旧页面。
				writeError(w, http.StatusForbidden, "CSRF 校验失败：页面与当前应用会话不匹配（通常是应用重启后仍在用旧页面），请刷新页面后重试")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// validWebCSRF 用恒定时间比较校验请求自带的 CSRF 头，避免基于耗时的侧信道猜测 token。
// 先判长度不等直接返回 false，既省一次常量比较，也保证 subtle 比较的两个切片等长。
func validWebCSRF(got string) bool {
	want := getWebCSRFToken()
	if got == "" || want == "" || len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// validWebOrigin 校验写请求的 Origin 头是否与本机 UI 同源。
// 三道关：Origin 不能为空或字面量 "null"（沙箱 iframe / 被篡改时常见）；scheme 只许 http/https；
// 解析出的 host 必须既等于请求 Host（同源），又是允许的 loopback host（isAllowedWebHost）。
// 拒绝非同源 Origin 即可阻断跨站表单/脚本对本 UI 的写操作。
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

// isAllowedWebHost 判断 host 是否为本地回环地址之一（127.0.0.1 / localhost / ::1）。
// 这是 Host 头白名单的核心：本 UI 只预期被本机访问，任何其它 host 名都按可疑拒绝。
// 先 SplitHostPort 去端口，再 Trim "[]" 兼容 IPv6 字面量（如 [::1]）。
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

// normalizeWebHost 统一小写 + 去首尾空白，让 host 比较大小写/空格无关。
func normalizeWebHost(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}
