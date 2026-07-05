package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/platform"

import . "github.com/Ryujoxys/sushiro-overdose/internal/proxy"

import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import . "github.com/Ryujoxys/sushiro-overdose/internal/notify"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// EngineStatus 是引擎对外暴露的运行状态，UI 据此切换按钮/文案。
// 状态机概览：idle ←→ capturing（抓凭证） / booking（抢号） / sniping（狙击未来时段），
// 运行结束（成功或失败）后落到 success / error，再由 finishRun/Stop 回到 idle。
// 三个 running 态（capturing/booking/sniping）互斥，同一时刻只能有一个在跑，见 isRunningLocked。
type EngineStatus string

const (
	// EngineIdle 引擎空闲，可启动新任务；任何运行结束（含 Stop）后都归位到这里。
	EngineIdle EngineStatus = "idle"
	// EngineCapturing 正在跑 MITM 代理抓微信小程序凭证；代理起好、系统代理已设。
	EngineCapturing EngineStatus = "capturing"
	// EngineBooking 正在按偏好扫描并尝试预约当前可约时段（抢号主循环）。
	EngineBooking EngineStatus = "booking"
	// EngineSniping 正在等待/抢未开放的未来时段（放号时刻狙击），比 booking 更高频。
	EngineSniping EngineStatus = "sniping"
	// EngineSuccess 一次运行成功结束（已抢到预约），等待回到 idle。仅终态。
	EngineSuccess EngineStatus = "success"
	// EngineError 一次运行失败结束（凭证失效/连续失败等），等待回到 idle。仅终态。
	EngineError EngineStatus = "error"
)

// CaptureStage 是凭证采集（runCapture）的结构化阶段，前端据此渲染进度条。
// 替代以前只靠一行 message 描述进度的做法——现在每个阶段有明确枚举，前端能高亮"当前在哪一步"。
type CaptureStage string

const (
	StageIdle                  CaptureStage = "idle"
	StagePreparingCert         CaptureStage = "preparing_cert"                   // 正在生成/检查本地 CA 证书
	StageInstallingCertUser    CaptureStage = "installing_cert_currentuser"      // 装用户级证书（Windows 不需 UAC）
	StageInstallingCertMachine CaptureStage = "installing_cert_localmachine_uac" // 装机器级证书（Windows 必弹 UAC，前端要预告）
	StageStartingProxy         CaptureStage = "starting_proxy"                   // 正在起 MITM 代理监听
	StageSettingSystemProxy    CaptureStage = "setting_system_proxy"             // 正在设系统代理（PAC/手动）
	StageWaitingCapture        CaptureStage = "waiting_capture"                  // 代理就绪，等用户在微信里点门店/预约产生凭证请求
	StageProbing               CaptureStage = "probing"                          // 已抓到凭证，自检基础接口
	StageDone                  CaptureStage = "done"
)

// ErrorKind 是结构化错误分类，替代以前前端用正则猜 message 字符串（explainMsg）的脆弱做法。
// 每类对应前端一条人话文案 + 一个出路按钮。
type ErrorKind string

const (
	ErrKindNone            ErrorKind = ""
	ErrKindCertUACDeclined ErrorKind = "cert_uac_declined"   // Windows 装 LocalMachine 证书时用户在 UAC 弹窗点了"否"
	ErrKindCertInstall     ErrorKind = "cert_install_failed" // 证书安装失败（权限/锁定/其他）
	ErrKindCertLocked      ErrorKind = "cert_locked"         // macOS keychain 锁定，需要解锁
	ErrKindProxy           ErrorKind = "proxy_failed"        // 系统代理设置失败
	ErrKindQUICBlock       ErrorKind = "quic_block_failed"   // Windows 屏蔽微信 QUIC 失败（可能抓不到包）
	ErrKindCaptureTimeout  ErrorKind = "capture_timeout"     // 等待凭证超时
	ErrKindAuthStale       ErrorKind = "auth_stale"          // 凭证过期/被手机端顶掉
	ErrKindNetwork         ErrorKind = "network"             // 网络错误
	ErrKindUnknown         ErrorKind = "unknown"
)

// weChatProbeInterval 是 runCapture 轮询里探测 PC 微信进程的节流间隔。
// PowerShell/ps 冷启动约 300ms，4s 间隔下平均开销可忽略；capture 等待期本就空转，
// 4s 粒度对「微信是否被关再重开」的信号灯足够（用户自己主导重启动作，不需秒级反馈）。
const weChatProbeInterval = 4 * time.Second

type EngineState struct {
	Status      EngineStatus       `json:"status"`
	Message     string             `json:"message"`
	LastCheck   string             `json:"last_check,omitempty"`
	Attempts    int                `json:"attempts"`
	Capture     *CaptureStatusJSON `json:"capture,omitempty"`
	Reservation *ReservationRecord `json:"reservation,omitempty"`
	// Stage 是采集（capturing）阶段的结构化进度，前端渲染进度条。仅 capturing/error 期有意义。
	Stage CaptureStage `json:"stage,omitempty"`
	// StageStep/StageTotal 是子进度（如装证书 1/2 库、字段 5/8）。0 表示无子进度。
	StageStep  int `json:"stage_step,omitempty"`
	StageTotal int `json:"stage_total,omitempty"`
	// ErrorKind 是结构化错误分类，前端据此显示人话文案 + 出路按钮（替代正则猜 message）。
	ErrorKind ErrorKind `json:"error_kind,omitempty"`
	// Warning 是非致命提示（如 Windows QUIC 屏蔽失败可能抓不到包），不改变 status/不中断采集，
	// 前端显示一条黄色提示。空=无。
	Warning string `json:"warning,omitempty"`
}

type CaptureStatusJSON struct {
	XAppCode        bool `json:"x_app_code"`
	QueryAuth       bool `json:"query_auth"`
	ReservationAuth bool `json:"reservation_auth"`
	UserAgent       bool `json:"user_agent"`
	Referer         bool `json:"referer"`
	WechatID        bool `json:"wechat_id"`
	PhoneNumber     bool `json:"phone_number"`
	StoreIDs        bool `json:"store_ids"`
	Complete        bool `json:"complete"`
	// WeChat 是 PC 微信进程状态（仅 Windows/darwin capturing 期有意义）。nil 表示无信息/非该平台。
	// 指针+omitempty 保证旧前端忽略未知字段（向前兼容）。
	WeChat *WeChatStatusJSON `json:"wechat,omitempty"`
}

// WeChatStatusJSON 给前端做信号灯：Running=是否有微信在跑；
// Restarted=本次 capture 期间是否检测到微信被关掉再重开。
type WeChatStatusJSON struct {
	Running   bool     `json:"running"`
	Restarted bool     `json:"restarted"`
	PIDs      []int    `json:"pids,omitempty"`
	Names     []string `json:"names,omitempty"`
}

type LogEntry struct {
	Time    string `json:"time"`
	Message string `json:"message"`
	Level   string `json:"level,omitempty"`
}

// BookingEngine 管理抓凭证 / 抢号 / 狙击的完整生命周期，对外由 Web UI 驱动。
// 并发模型：单进程内只有一个全局 engine 实例；UI 的 HTTP handler 在各自 goroutine 里
// 调 Start*/Stop，运行循环在独立 goroutine 里跑 runCapture/runBooking/runSniper。
// 所有对 state/cancel/done/tokens/proxy/pinnedSlot 的读写都须经 mu 保护（state 只读
// 场景可用 RLock）。cancel+done 组合用于让 Stop 等运行循环真正退出后再回 idle。
type BookingEngine struct {
	mu     sync.RWMutex
	state  EngineState
	cancel context.CancelFunc
	done   chan struct{}
	tokens *CapturedTokens
	proxy  *ProxyServer
	logs   []LogEntry
	// pinnedSlot，非空时 booking 只预约这一个确切时段（直接预约），不按偏好扫描。
	// 仅单次运行有效，finishRun 清空。
	pinnedSlot *TargetSlot
	// captureWeChat 缓存 runCapture 最近一次微信进程探测结果，供 GetState→captureStatus 附到
	// CaptureStatusJSON.WeChat 推给前端做信号灯。capturing 期由 runCapture 持 mu 写，退出时清空。
	captureWeChat *WeChatStatusJSON
}

// engine 是全局单例：Web handler 和 CLI 都操作这一个实例，状态集中在它身上。
var engine = &BookingEngine{
	state: EngineState{Status: EngineIdle},
}

func (e *BookingEngine) GetState() EngineState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	s := e.state
	if e.tokens != nil && e.state.Status == EngineCapturing {
		s.Capture = e.captureStatus()
	}
	return s
}

func (e *BookingEngine) captureStatus() *CaptureStatusJSON {
	cs := captureStatusForTokens(e.tokens)
	// 附上缓存的本轮微信进程探测结果（runCapture 持 mu 写入）。
	if cs != nil && e.captureWeChat != nil {
		cs.WeChat = e.captureWeChat
	}
	return cs
}

func captureStatusForTokens(tokens *CapturedTokens) *CaptureStatusJSON {
	if tokens == nil {
		return nil
	}
	tokens.Lock()
	defer tokens.Unlock()
	return &CaptureStatusJSON{
		XAppCode:        tokens.XAppCode != "",
		QueryAuth:       tokens.QueryAuth != "",
		ReservationAuth: tokens.ReservationAuth != "",
		UserAgent:       tokens.UserAgent != "",
		Referer:         tokens.Referer != "",
		WechatID:        tokens.WechatID != "",
		PhoneNumber:     tokens.PhoneNumber != "",
		StoreIDs:        len(tokens.StoreIDs) > 0,
		Complete:        tokens.IsCompleteUnlocked(),
	}
}

// countCapturedFields 数已抓到的凭证字段数（0~8），供 waiting 阶段子进度"X/8"用。
func countCapturedFields(tokens *CapturedTokens) int {
	if tokens == nil {
		return 0
	}
	cs := captureStatusForTokens(tokens)
	if cs == nil {
		return 0
	}
	n := 0
	for _, ok := range []bool{cs.XAppCode, cs.QueryAuth, cs.ReservationAuth, cs.UserAgent, cs.Referer, cs.WechatID, cs.PhoneNumber, cs.StoreIDs} {
		if ok {
			n++
		}
	}
	return n
}

// setState 更新状态文案并广播给 UI（经 SSE 总线）。运行循环中频繁调用，
// 故写成「写 state + 发布快照」两步；读快照走 GetState 自带锁，避免调用方各处加锁。
func (e *BookingEngine) setState(status EngineStatus, message string) {
	e.mu.Lock()
	e.state.Status = status
	e.state.Message = message
	e.mu.Unlock()
	bus.publish("engine", mustJSON(e.GetState()))
}

// setStage 更新采集阶段进度（stage/子进度），并 publish 到前端。status/message 保持不变。
// 用于 runCapture 在不改变状态机的前提下推进"当前在第几步"的细粒度进度。
func (e *BookingEngine) setStage(stage CaptureStage, step, total int) {
	e.mu.Lock()
	e.state.Stage = stage
	e.state.StageStep = step
	e.state.StageTotal = total
	snap := e.snapshotStateLocked()
	e.mu.Unlock()
	bus.publish("engine", mustJSON(snap))
}

// setError 标记结构化错误分类（前端据此显示人话 + 出路按钮），并 publish。
// 通常紧跟 setState(EngineError, ...) 调用。
func (e *BookingEngine) setError(kind ErrorKind) {
	e.mu.Lock()
	e.state.ErrorKind = kind
	snap := e.snapshotStateLocked()
	e.mu.Unlock()
	bus.publish("engine", mustJSON(snap))
}

// resetProgress 清空 stage/error_kind，运行开始/结束时调用，避免上次状态残留。
func (e *BookingEngine) resetProgress() {
	e.mu.Lock()
	e.state.Stage = StageIdle
	e.state.StageStep = 0
	e.state.StageTotal = 0
	e.state.ErrorKind = ErrKindNone
	e.state.Warning = ""
	e.mu.Unlock()
}

// setWarning 设非致命提示（不改变 status，前端显示黄色提示条）。
func (e *BookingEngine) setWarning(msg string) {
	e.mu.Lock()
	e.state.Warning = msg
	snap := e.snapshotStateLocked()
	e.mu.Unlock()
	bus.publish("engine", mustJSON(snap))
}

// classifyCertError 按 err 文案兜底分类证书错误（仅当平台层未回填 ErrorKind 时）。
// 平台层（platform_windows.go 等）应优先通过 setError 在 InstallCert 内部精确回填；
// 这里是 message 原文兜底，覆盖平台层没设的情况。
func (e *BookingEngine) classifyCertError(err error) {
	if err == nil {
		return
	}
	e.mu.Lock()
	if e.state.ErrorKind != ErrKindNone {
		e.mu.Unlock()
		return // 平台层已设，不覆盖
	}
	e.mu.Unlock()
	msg := strings.ToLower(err.Error())
	var kind ErrorKind
	switch {
	case strings.Contains(msg, "exit code") || strings.Contains(msg, "runas") ||
		strings.Contains(msg, "uac") || strings.Contains(msg, "管理员") ||
		strings.Contains(msg, "elevated") || strings.Contains(msg, "localmachine") ||
		strings.Contains(msg, "机器级"):
		kind = ErrKindCertUACDeclined
	case strings.Contains(msg, "locked") || strings.Contains(msg, "锁定") ||
		strings.Contains(msg, "unlock") || strings.Contains(msg, "authfailed") ||
		strings.Contains(msg, "user interaction is not allowed") ||
		strings.Contains(msg, "interaction not allowed"):
		// macOS keychain 锁定：security add-trusted-cert 报 "User interaction is not allowed"
		kind = ErrKindCertLocked
	default:
		kind = ErrKindCertInstall
	}
	e.setError(kind)
}

// snapshotStateLocked 返回当前状态的深拷贝快照（调用方持锁）。
func (e *BookingEngine) snapshotStateLocked() EngineState {
	snap := e.state
	if e.state.Capture != nil {
		c := *e.state.Capture
		snap.Capture = &c
	}
	if e.state.Reservation != nil {
		r := *e.state.Reservation
		snap.Reservation = &r
	}
	return snap
}

// finishRun 在一次运行（capture/booking/sniper）退出时收尾：清掉 cancel/done 句柄，
// 释放 pinnedSlot。关键点：只有当传入的 done 仍是当前 e.done 时才清空——避免一次新
// 运行已经把 e.done 换成新 channel 后，被旧运行的延迟 defer 误清掉。pinnedSlot 每次
// 都清，保证「直接预约」是一次性的。
func (e *BookingEngine) finishRun(done chan struct{}) {
	e.mu.Lock()
	if e.done == done {
		e.cancel = nil
		e.done = nil
	}
	e.pinnedSlot = nil
	e.mu.Unlock()
	bus.publish("engine", mustJSON(e.GetState()))
}

// recoverEngineRun 在运行循环 goroutine 的 defer 里捕获 panic，避免单个循环崩溃
// 把整个进程拖垮（抓号/取号中断、错过放号窗口）。捕获后记日志并把引擎置 error 态，
// 随后由调用方的 finishRun 正常收尾、回 idle 并广播。r 通过 recover() 获取。
func (e *BookingEngine) recoverEngineRun(runName string) {
	if r := recover(); r != nil {
		LogMessage(time.Now(), runName+" 运行发生 panic 已恢复："+fmt.Sprint(r))
		e.setState(EngineError, runName+" 内部异常，已停止，请重试或查看日志")
	}
}

// abortStart 用于 Start* 入口在校验阶段就失败时的统一回滚：取消刚建的 ctx、走 finishRun
// 回收句柄、关闭 done 通知等待者。注意 done 在这里 close，因此调用它的入口不会再 defer close。
func (e *BookingEngine) abortStart(done chan struct{}, cancel context.CancelFunc) {
	if cancel != nil {
		cancel()
	}
	e.finishRun(done)
	close(done)
}

func (e *BookingEngine) addLog(msg string) {
	e.addLogLevel(msg, "info")
}

// addLogLevel 追加一条日志并广播。内存里只留最近 500 条（环形截断，防止长时间运行无限增长），
// 同时推一份给 SSE 订阅的 UI 和持久化日志（LogMessage）。level: info/warn/error。
func (e *BookingEngine) addLogLevel(msg, level string) {
	e.mu.Lock()
	entry := LogEntry{
		Time:    time.Now().Format("15:04:05"),
		Message: msg,
		Level:   level,
	}
	e.logs = append(e.logs, entry)
	if len(e.logs) > 500 {
		e.logs = e.logs[len(e.logs)-500:]
	}
	e.mu.Unlock()
	bus.publish("log", mustJSON(entry))
	LogMessage(time.Now(), msg)
}

func (e *BookingEngine) GetLogs() []LogEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]LogEntry, len(e.logs))
	copy(out, e.logs)
	return out
}

// StartCapture 启动 MITM 代理抓微信小程序凭证（X-App-Code、各 auth、UA 等）。
// 流程：占住运行态 → 建独立 goroutine 跑 runCapture；返回 nil 表示已启动，
// 引擎状态由 runCapture 在内部推进（装证书 → 起代理 → 设系统代理 → 轮询凭证）。
// 若已有任务在跑（isRunningLocked）则拒绝并取消刚建的 ctx。
func (e *BookingEngine) StartCapture() error {
	ctx, cancel := context.WithCancel(context.Background())
	// 跨进程互斥：另一个 sushiro 主进程（如 daemon）正在抢号时拒绝启动，避免双进程同时
	// 打官方接口、争抢同一账号预约（请求翻倍、风控升级）。本进程自己的 marker 不算。
	if active, status := externalMainFlowActive(); active {
		cancel()
		return fmt.Errorf("另一个实例正在运行中（%s），请先停止它", status)
	}
	e.mu.Lock()
	if e.isRunningLocked() {
		e.mu.Unlock()
		cancel()
		return fmt.Errorf("引擎正在运行中")
	}
	e.cancel = cancel
	done := make(chan struct{})
	e.done = done
	e.state.Status = EngineCapturing
	e.state.Message = "正在启动参数捕获..."
	e.mu.Unlock()
	bus.publish("engine", mustJSON(e.GetState()))
	pauseSamplingForMainFlow()

	go func() {
		defer func() {
			e.recoverEngineRun("抓取凭证")
			e.finishRun(done)
			close(done)
		}()
		e.runCapture(ctx)
	}()
	return nil
}

// runCapture 是抓凭证主流程：装 CA 证书 → 起 MITM 代理 → 设系统代理 → 每秒轮询凭证是否抓全。
// 收到 ctx.Done（Stop/ResetCapture）即拆代理回 idle；抓全后落盘 SaveLocalConfig，自检仅作
// 诊断不影响保存。接口发现模式（APIDiscoveryEnabled）下抓全后保持代理运行，方便用户继续点接口。
func (e *BookingEngine) runCapture(ctx context.Context) {
	doneActivity := markMainFlowActive("capturing")
	defer doneActivity()

	e.resetProgress()
	e.setState(EngineCapturing, "正在准备证书...")
	e.setStage(StagePreparingCert, 0, 0)

	caCert, caKey, err := LoadOrGenerateCA()
	if err != nil {
		e.setState(EngineError, "CA证书加载失败: "+err.Error())
		e.setError(ErrKindCertInstall)
		e.addLogLevel("CA证书加载失败: "+err.Error(), "error")
		return
	}

	trusted, _ := IsCertTrusted()
	if !trusted {
		// 装证书分两阶段：Windows 先装 CurrentUser（不需 UAC），再装 LocalMachine（弹 UAC）。
		// 在装 LocalMachine 前把 stage 切到 uac 阶段，前端据此预告"马上会弹系统窗口，点是"。
		if runtime.GOOS == "windows" {
			e.addLog("首次运行，正在安装CA证书（用户级）...")
			e.setStage(StageInstallingCertUser, 1, 2)
		} else {
			e.addLog("首次运行，正在安装CA证书...")
			e.setStage(StageInstallingCertUser, 1, 1)
		}
		// Windows 装 LocalMachine 前，InstallCertToMachine 阶段由平台层内部触发 UAC；
		// 这里先切到 machine 阶段，让前端提前预告（即便 InstallCert 是一步调用）。
		if runtime.GOOS == "windows" {
			e.setStage(StageInstallingCertMachine, 2, 2)
		}
		if err := InstallCert(); err != nil {
			// ErrorKind 由 InstallCert 内部回填到 e.state.ErrorKind（见 platform 层 setErrorForCertFailure）。
			// 这里兜底：若平台层没设，按 message 关键词粗分（UAC 拒绝/锁定/通用）。
			e.setState(EngineError, "证书安装失败："+err.Error())
			e.classifyCertError(err)
			e.addLogLevel("证书安装失败: "+err.Error(), "error")
			return
		}
		e.addLog("证书安装成功")
	} else {
		e.addLog("CA证书已信任，跳过安装")
	}

	tokens := NewCapturedTokens()
	e.mu.Lock()
	e.tokens = tokens
	e.mu.Unlock()

	e.setStage(StageStartingProxy, 0, 0)
	proxy, err := StartProxy(caCert, caKey, tokens, e.addLog)
	if err != nil {
		e.setState(EngineError, "启动代理失败: "+err.Error())
		e.setError(ErrKindProxy)
		e.addLogLevel("启动代理失败: "+err.Error(), "error")
		return
	}
	e.mu.Lock()
	e.proxy = proxy
	e.mu.Unlock()
	actualPort := proxy.Port()

	e.setStage(StageSettingSystemProxy, 0, 0)
	if err := SetSystemProxy(actualPort); err != nil {
		proxy.Close()
		e.setState(EngineError, "设置系统代理失败: "+err.Error())
		e.setError(ErrKindProxy)
		e.addLogLevel("设置系统代理失败: "+err.Error(), "error")
		return
	}
	// 读平台层记录的非致命提示（如 Windows QUIC 屏蔽失败），推给前端但不中断。
	if warns := DrainProxyWarnings(); len(warns) > 0 {
		e.setWarning(strings.Join(warns, "；"))
	}
	markProxyActive(actualPort, os.Getpid())

	e.setStage(StageWaitingCapture, 0, 8)
	e.setState(EngineCapturing, "等待捕获凭证参数，请彻底关闭 PC 微信后重新打开，并在寿司郎小程序里点一次排队或预约...")
	proxyHint := fmt.Sprintf("捕获代理已设置 (127.0.0.1:%d)", actualPort)
	if GetActiveWebPort() > 0 && (runtime.GOOS == "windows" || runtime.GOOS == "darwin") {
		proxyHint += "；已使用 PAC 仅代理寿司郎域名"
	}
	e.addLog(proxyHint + "。请彻底关闭 PC 微信后重新打开，进入寿司郎小程序，选任意门店点一次「排队」或「预约」（不用真的提交）")
	e.addLog("提示：如果 PC 微信小程序弹出“服务器出错/网络错误”，但本工具抓到凭证并通过基础接口自检，可以直接忽略小程序弹窗。")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	savedForDiscovery := false

	// PC 微信进程探测状态（局部变量，不进 e 字段，避免与 Stop goroutine 的数据竞争）。
	// 仅 Windows/darwin 有意义；Linux 无微信客户端，跳过探测省一次子进程。
	probeWeChat := runtime.GOOS == "windows" || runtime.GOOS == "darwin"
	var wechatBaseline []WeChatProcessInfo // 进入「等待捕获」态时的微信进程基线
	var wechatClosedOnce bool              // 是否已观察到微信被关停（锁存）
	var wechatRestarted bool               // 是否判定为已重启（锁存，置 true 后不再翻回）
	var wechatLastProbe time.Time
	if probeWeChat {
		wechatBaseline = ListWeChatProcesses()
		e.mu.Lock()
		e.captureWeChat = buildWeChatStatus(wechatBaseline, false)
		e.mu.Unlock()
	}

	for {
		select {
		case <-ctx.Done():
			e.cleanupProxy()
			// 清掉微信探测缓存，避免残留到 idle 态（GetState 在非 capturing 不读 Capture，但保险）。
			e.mu.Lock()
			e.captureWeChat = nil
			e.mu.Unlock()
			e.setState(EngineIdle, "已停止捕获")
			return
		case <-ticker.C:
			// 微信进程探测：每 weChatProbeInterval 秒一次（PowerShell 冷启动~300ms，
			// 不必每秒探；4s 粒度对「是否重启」信号灯足够，用户自己主导重启动作）。
			if probeWeChat && !wechatRestarted && (wechatLastProbe.IsZero() || time.Since(wechatLastProbe) >= weChatProbeInterval) {
				current := ListWeChatProcesses()
				restarted, nowClosed := weChatAppearsRestarted(wechatBaseline, current, wechatClosedOnce)
				if nowClosed {
					wechatClosedOnce = true
				}
				if restarted {
					wechatRestarted = true
					e.addLog("检测到 PC 微信已重新打开，请在寿司郎小程序里点一次排队或预约。")
				}
				e.mu.Lock()
				e.captureWeChat = buildWeChatStatus(current, wechatRestarted)
				e.mu.Unlock()
				wechatLastProbe = time.Now()
			}
			// waiting_capture 阶段每秒更新"已抓字段数"作为子进度，前端显示"X/8"。
			// 不额外 publish，复用下面那次。仅 waiting 阶段有意义。
			e.mu.Lock()
			if e.state.Stage == StageWaitingCapture {
				e.state.StageStep = countCapturedFields(e.tokens)
				e.state.StageTotal = 8
			}
			e.mu.Unlock()
			bus.publish("engine", mustJSON(e.GetState()))
			if tokens.IsComplete() {
				prefs := LoadPreferences()
				// 接口发现调试开启时：抓到凭证先存好，但保持代理运行，让用户继续在
				// 小程序里点想记录的接口（如「排队取号」），手动「停止」前不拆代理。
				// 否则代理会在凭证一抓全就关闭，后续接口根本来不及记录。
				if APIDiscoveryEnabled() {
					if !savedForDiscovery {
						if err := SaveLocalConfig(tokens); err == nil {
							setWebSettings(tokens.ToSettingsWithPrefs(prefs))
						}
						savedForDiscovery = true
						e.addLog("凭证已抓到并保存；接口发现调试开启中——请在小程序里点你想记录的接口（如「排队取号」），完成后点「停止」。")
						e.setState(EngineCapturing, "接口发现调试中：凭证已抓到，代理保持运行。请在小程序里操作要记录的接口，记录完点「停止」。")
					}
					continue
				}
				// 自检只作诊断，不拦保存：抓到完整凭证就落盘并完成捕获，
				// 自检结果仅决定提示语。避免基础接口偶发失败/被拒时把用户卡死。
				e.setStage(StageProbing, 0, 0)
				e.setState(EngineCapturing, "已抓到凭证参数，正在自检基础接口...")
				report := runAuthProbeWithTokens(ctx, "", tokens, prefs)
				if err := SaveLocalConfig(tokens); err != nil {
					e.addLogLevel("保存配置失败: "+err.Error(), "error")
					e.cleanupProxy()
					e.setState(EngineError, "凭证参数保存失败: "+err.Error())
					e.setError(ErrKindUnknown)
					return
				}
				markAuthHealthy()                         // 重新捕获凭证 → 清除"凭证过期"提醒
				recordAuthCaptured(captureMethodPCWechat) // 记录捕获时间/方式，重置寿命周期
				setWebSettings(tokens.ToSettingsWithPrefs(prefs))
				e.cleanupProxy()
				if report.OK {
					e.addLog("凭证参数已捕获、基础接口自检通过并保存！")
					e.setStage(StageDone, 8, 8)
					e.setState(EngineIdle, "凭证参数捕获完成！")
				} else {
					e.addLogLevel("凭证参数已捕获并保存；基础接口自检未通过（"+authProbeFailureSummary(report)+"），可直接尝试使用，如不可用再重新捕获。", "warn")
					e.setState(EngineIdle, "凭证参数已保存；基础接口自检未通过，但可直接尝试使用。即使小程序显示服务器出错也不影响。")
				}

				tokens.Lock()
				storeIDs := append([]string(nil), tokens.StoreIDs...)
				tokens.Unlock()
				if len(storeIDs) > 0 && len(prefs.SelectedStores) == 0 {
					// 读改写走 preferencesMu，避免与 UI 保存并发冲掉字段。
					_ = UpdatePreferences(func(p *UserPreferences) {
						if len(p.SelectedStores) == 0 {
							p.SelectedStores = storeIDs
						}
					})
				}
				return
			}
		}
	}
}

func (e *BookingEngine) cleanupProxy() {
	e.mu.Lock()
	if e.proxy != nil {
		e.proxy.Close()
		e.proxy = nil
	}
	e.mu.Unlock()
	ClearSystemProxy()
	markProxyInactive()
}

// ResetCapture 强制停止当前抓包/运行、关闭代理并清理系统代理残留，把引擎重置回
// idle。抓到凭证后会自动断连，状态可能卡在 capturing（尤其接口发现保持连接时），
// 导致「获取凭证」被 isRunningLocked 挡住而连不回来；重置后即可手动重新连接抓包。
func (e *BookingEngine) ResetCapture() {
	e.mu.Lock()
	cancel := e.cancel
	e.cancel = nil
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	e.cleanupProxy()
	checkStaleProxy()
	e.setState(EngineIdle, "已重置抓包状态，可点「获取凭证」手动重新连接")
	e.addLog("已重置抓包状态：代理已断开并清理，点「获取凭证」可重新连接抓包。")
}

// StartBooking begins the automated booking loop.
// normalizeBookingTime 把 HHMM / HH:MM:SS 等规范成官方接口要求的 HHMMSS。
func normalizeBookingTime(t string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, t)
	if len(digits) == 4 {
		return digits + "00"
	}
	return digits
}

// StartBookingSlot 直接预约一个确切时段（来自「预约未来」里已放出、可约的卡片）。
// 与 StartBooking 共用校验/运行链路，只是钉死单个目标、不按偏好扫描，且为一次性。
func (e *BookingEngine) StartBookingSlot(storeID, date, start, end string) error {
	storeID = strings.TrimSpace(storeID)
	date = strings.TrimSpace(date)
	start = normalizeBookingTime(start)
	end = normalizeBookingTime(end)
	if storeID == "" || date == "" || start == "" {
		return fmt.Errorf("时段信息不完整，无法直接预约")
	}
	e.mu.Lock()
	if e.isRunningLocked() {
		e.mu.Unlock()
		return fmt.Errorf("引擎正在运行中")
	}
	e.pinnedSlot = &TargetSlot{StoreID: storeID, Date: date, Start: start, End: end}
	e.mu.Unlock()
	if err := e.StartBooking(); err != nil {
		e.mu.Lock()
		e.pinnedSlot = nil
		e.mu.Unlock()
		return err
	}
	return nil
}

// StartBooking 启动自动抢号循环（偏好扫描模式或 pinnedSlot 直接预约模式）。
// 启动前的同步校验链：isRunningLocked 互斥 → 加载并校验凭证 → 拉偏好的目标门店
// （pinnedSlot 非空时钉死单店）→ ValidateForReservation → GetTimeslots 快速验活。
// 任一失败都走 abortStart 回滚并回 idle；全过则起 goroutine 跑 runBooking。
func (e *BookingEngine) StartBooking() error {
	ctx, cancel := context.WithCancel(context.Background())
	if active, status := externalMainFlowActive(); active {
		cancel()
		return fmt.Errorf("另一个实例正在运行中（%s），请先停止它", status)
	}
	e.mu.Lock()
	if e.isRunningLocked() {
		e.mu.Unlock()
		cancel()
		return fmt.Errorf("引擎正在运行中")
	}
	e.cancel = cancel
	done := make(chan struct{})
	e.done = done
	e.state.Status = EngineBooking
	e.state.Message = "正在验证凭证参数..."
	e.state.Attempts = 0
	e.state.Reservation = nil
	e.mu.Unlock()
	bus.publish("engine", mustJSON(e.GetState()))
	pauseSamplingForMainFlow()

	tokens, err := LoadLocalConfig()
	if err != nil {
		e.abortStart(done, cancel)
		e.setState(EngineIdle, "暂无凭证参数")
		return fmt.Errorf("暂无凭证参数，请先完成参数捕获")
	}

	prefs := LoadPreferences()
	e.mu.RLock()
	pinned := e.pinnedSlot
	e.mu.RUnlock()
	if pinned != nil {
		// 直接预约：钉死该时段所属门店，即使它不在偏好门店里。
		tokens.Lock()
		tokens.StoreIDs = []string{pinned.StoreID}
		tokens.Unlock()
	} else if len(prefs.SelectedStores) > 0 {
		tokens.Lock()
		tokens.StoreIDs = prefs.SelectedStores
		tokens.Unlock()
	}
	if err := tokens.ValidateForReservation(); err != nil {
		e.abortStart(done, cancel)
		e.setState(EngineIdle, "预约参数不完整")
		return err
	}

	settings := tokens.ToSettingsWithPrefs(prefs)
	client := NewClient(settings)

	// Quick verify
	if _, err := client.GetTimeslots(context.Background(), settings.StoreIDs[0]); err != nil {
		e.abortStart(done, cancel)
		e.setState(EngineIdle, "验证失败")
		noteAuthResult(err) // 凭证失败则标记 stale
		if isAuthError(err) {
			DeleteLocalConfig()
			return fmt.Errorf("凭证参数已过期，请重新捕获")
		}
		return fmt.Errorf("验证失败: %w", err)
	}
	markAuthHealthy() // 验证 GetTimeslots 成功 → 凭证有效

	setNotifier(BuildNotifierFromConfig())

	go func() {
		defer func() {
			e.recoverEngineRun("抢预约")
			e.finishRun(done)
			close(done)
		}()
		e.runBooking(ctx, client, settings, prefs)
	}()
	return nil
}

// isRunningLocked 判断是否正有任务在跑（三个 running 态之一）。调用方必须已持 mu。
// 这是 StartCapture/StartBooking/StartSniper 互斥的依据：同一时刻只允许一个运行循环。
func (e *BookingEngine) isRunningLocked() bool {
	return e.state.Status == EngineCapturing ||
		e.state.Status == EngineBooking ||
		e.state.Status == EngineSniping
}

// runBooking 是抢号主循环。分两条路径：
//  1. pinnedSlot 非空：直接预约该单个时段，最多重试 3 次，结果即终态（成功/失败均停）。
//  2. 否则进入偏好扫描循环：每轮遍历目标门店 GetTimeslots，挑出满足偏好且可约的最佳时段尝试
//     CreateReservation，成功即停；空轮 100ms 节流。
//
// 三类失败计数器协同：
//   - authErrors：凭证类失败累计，达 3 次判定凭证失效、删配置并终止（见下方）。
//   - errStreak：任意失败连续累计，达 5 次后额外 sleep 5s 给服务端喘口气再重试，避免被风控。
//   - temporarySkips：5xx（isOfficialServerHTTPError）命中的时段冷却 30s 再试；booked 则永久跳过。
func (e *BookingEngine) runBooking(ctx context.Context, client *Client, settings Settings, prefs UserPreferences) {
	doneActivity := markMainFlowActive("booking")
	defer doneActivity()

	e.setState(EngineBooking, "正在抢号...")
	e.addLog(fmt.Sprintf("开始抢号 — 门店: %v, 成人: %d, 儿童: %d", settings.StoreIDs, settings.Adult, settings.Child))

	healthStop := startHealthCheck(ctx, client, settings.StoreIDs)
	defer close(healthStop)

	// 直接预约：钉死单个时段，一次性尝试（成功即停，不可约/业务错误即停），不进入偏好扫描循环。
	e.mu.RLock()
	pinned := e.pinnedSlot
	e.mu.RUnlock()
	if pinned != nil {
		best := pinned
		slotLabel := FormatSlotWindow(best.Date, best.Start, best.End, settings.Location)
		e.addLog(fmt.Sprintf("直接预约: %s — 尝试预约...", slotLabel))
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			select {
			case <-ctx.Done():
				e.setState(EngineIdle, "已停止")
				return
			default:
			}
			reservation, err := client.CreateReservation(ctx, best.StoreID, best.Date, best.Start)
			if err == nil {
				storeInfo, _ := client.GetStoreInfo(ctx, best.StoreID)
				storeName := storeInfo.Name
				if storeName == "" {
					storeName = best.StoreID
				}
				reservation.MonitoredStoreID = best.StoreID
				onBookingSuccess(reservation, storeName, storeInfo.Address, slotLabel, "预约")
				markAuthHealthy() // 创建预约成功 → 凭证有效
				e.mu.Lock()
				e.state.Reservation = &reservation
				e.mu.Unlock()
				e.setState(EngineSuccess, fmt.Sprintf("预约成功！%s @ %s", slotLabel, storeName))
				e.addLog(fmt.Sprintf("🎉 预约成功！号码: %s, 时段: %s, 门店: %s", reservation.Number, slotLabel, storeName))
				return
			}
			lastErr = err
			if isAuthError(err) {
				noteAuthResult(err) // 凭证失败则标记 stale
				e.addLogLevel("凭证失败", "error")
				sendNotification("寿司郎 - 凭证失败", "请重新捕获参数")
				DeleteLocalConfig()
				e.setState(EngineError, "凭证参数已失效")
				return
			}
			if errors.Is(err, ErrActiveReservationExists) {
				msg := friendlyOfficialAPIError(err)
				e.addLogLevel(slotLabel+" — "+msg, "error")
				e.setState(EngineError, msg)
				return
			}
			if errors.Is(err, ErrNoReservationAvailable) {
				e.addLogLevel(slotLabel+" — 这个时段已经约满或被抢走了", "error")
				e.setState(EngineError, "这个时段已经约满或被抢走了；可以挑别的时段，或对未放出的时段加入狙击。")
				return
			}
			if isOfficialServerHTTPError(err) {
				e.addLog(bookingServerErrorLog(slotLabel, err))
				if !sleepContext(ctx, 400*time.Millisecond) {
					return
				}
				continue
			}
			e.addLogLevel(fmt.Sprintf("%s — 预约失败: %s", slotLabel, err), "error")
			e.setState(EngineError, "预约失败："+err.Error())
			return
		}
		e.addLogLevel(fmt.Sprintf("%s — 预约失败: %v", slotLabel, lastErr), "error")
		e.setState(EngineError, "这个时段暂时预约不上，请稍后重试或挑别的时段。")
		return
	}

	// 三个循环内状态（仅本 goroutine 访问，无需加锁）：
	//   booked         : 已经判定「名额已满」(ErrNoReservationAvailable) 的时段，本轮永久跳过，
	//                    避免对同一时段反复 CreateReservation 浪费配额。
	//   temporarySkips : 5xx 命中的时段 → 冷却 30s（reservationServerErrorCooldown），到点自动恢复重试；
	//                    区别于 booked：5xx 多是临时抖动，不应永久放弃。
	//   errStreak      : 连续失败计数，达 5 触发一次 5s 退避再清零，防止被服务端限流/风控。
	//   authErrors     : 凭证类失败（isAuthError）累计，达 3 判定凭证真失效，删配置并终止整个抢号。
	var booked map[string]bool
	temporarySkips := map[string]time.Time{}
	errStreak := 0
	authErrors := 0

	for {
		select {
		case <-ctx.Done():
			e.setState(EngineIdle, "已停止抢号")
			e.addLog("抢号已停止")
			return
		default:
		}

		now := time.Now().In(settings.Location)

		var best *TargetSlot
		for _, storeID := range settings.StoreIDs {
			slots, err := client.GetTimeslots(ctx, storeID)
			if err != nil {
				if isAuthError(err) {
					noteAuthResult(err) // 凭证失败则标记 stale
					authErrors++
					// 凭证失败累计达 3 次才终止：单次 401 可能是偶发/代理抖动，连续 3 次
					// 才较可靠地判定凭证真过期；避免一次抖动就清掉配置逼用户重抓。
					if authErrors >= 3 {
						e.addLogLevel("凭证失败，请重新捕获参数", "error")
						sendNotification("寿司郎 - 凭证失败", "凭证参数已失效，请重新捕获")
						DeleteLocalConfig()
						e.setState(EngineError, "凭证参数已失效，请重新捕获")
						return
					}
				} else if isCredentialRefreshLikelyError(err) {
					// 与 CreateReservation 失败分支保持对称：软过期也要喂给凭证健康监测，否则不触发 stale 提醒。
					noteAuthResult(err)
				}
				errStreak++
				if errStreak >= 5 {
					// 连续 5 次失败（含非凭证类）：服务端可能在抽风或已限流，整体退避 5s 再继续，
					// 比无脑每 100ms 重试更不容易被风控盯上。
					e.addLog("连续失败过多，等待5秒...")
					if !sleepContext(ctx, 5*time.Second) {
						return
					}
					errStreak = 0
				}
				// 空转节流：100ms 一轮 ≈ 10 次/秒，在「还没放号、等时段刷新」时既不漏掉刚冒出来的
				// 可约时段，又不至于把接口打爆；与狙击阶段（50ms）相比更温和。
				if !sleepContext(ctx, 100*time.Millisecond) {
					return
				}
				continue
			}
			errStreak = 0
			authErrors = 0
			markAuthHealthy() // GetTimeslots 成功 → 凭证有效
			appendHistory(slots, storeID)

			for i := range slots {
				if !prefs.ShouldTarget(slots[i], settings.Location) {
					continue
				}
				if strings.ToUpper(slots[i].Availability) != "AVAILABLE" {
					continue
				}
				key := bookingSlotKey(storeID, slots[i].Date, slots[i].Start)
				if booked != nil && booked[key] {
					continue
				}
				if isTemporaryBookingSkipped(temporarySkips, key, now) {
					continue
				}
				t := &TargetSlot{
					StoreID: storeID,
					Date:    slots[i].Date,
					Start:   slots[i].Start,
					End:     slots[i].End,
				}
				if best == nil || prefs.PreferTargetSlot(*t, *best, settings.Location, settings.StoreIDs) {
					best = t
				}
			}
		}

		if best == nil {
			e.mu.Lock()
			e.state.Attempts++
			e.state.LastCheck = now.Format("15:04:05")
			e.state.Message = fmt.Sprintf("查询中...第 %d 次，暂无目标时段", e.state.Attempts)
			e.mu.Unlock()
			bus.publish("engine", mustJSON(e.GetState()))
			// 无目标空轮也节流 100ms，与上方失败分支一致，防止 CPU 空转 + 接口过载。
			if !sleepContext(ctx, 100*time.Millisecond) {
				return
			}
			continue
		}

		slotLabel := FormatSlotWindow(best.Date, best.Start, best.End, settings.Location)
		e.addLog(fmt.Sprintf("发现目标: %s — 尝试预约...", slotLabel))

		reservation, err := client.CreateReservation(ctx, best.StoreID, best.Date, best.Start)
		if err != nil {
			if isAuthError(err) {
				noteAuthResult(err) // 凭证失败则标记 stale
				authErrors++
				// CreateReservation 侧同样用 3 次门槛判定凭证失效，与 GetTimeslots 分支对称。
				if authErrors >= 3 {
					e.addLogLevel("凭证失败", "error")
					sendNotification("寿司郎 - 凭证失败", "请重新捕获参数")
					DeleteLocalConfig()
					e.setState(EngineError, "凭证参数已失效")
					return
				}
			}

			if errors.Is(err, ErrActiveReservationExists) {
				msg := friendlyOfficialAPIError(err)
				e.addLogLevel(slotLabel+" — "+msg, "error")
				e.setState(EngineError, msg)
				return
			}

			if isOfficialServerHTTPError(err) {
				key := bookingSlotKey(best.StoreID, best.Date, best.Start)
				markTemporaryBookingSkip(temporarySkips, key, now)
				e.addLog(bookingServerErrorLog(slotLabel, err))
			} else if errors.Is(err, ErrNoReservationAvailable) {
				key := bookingSlotKey(best.StoreID, best.Date, best.Start)
				if booked == nil {
					booked = make(map[string]bool)
				}
				booked[key] = true
				e.addLog(fmt.Sprintf("%s — 名额已满，继续尝试", slotLabel))
			} else {
				e.addLog(fmt.Sprintf("%s — 预约失败: %s", slotLabel, err))
			}
			if !sleepContext(ctx, 100*time.Millisecond) {
				return
			}
			continue
		}

		// Success!
		storeInfo, _ := client.GetStoreInfo(ctx, best.StoreID)
		storeName := storeInfo.Name
		if storeName == "" {
			storeName = best.StoreID
		}
		reservation.MonitoredStoreID = best.StoreID
		onBookingSuccess(reservation, storeName, storeInfo.Address, slotLabel, "预约")
		markAuthHealthy() // 创建预约成功 → 凭证有效

		e.mu.Lock()
		e.state.Reservation = &reservation
		e.mu.Unlock()
		e.setState(EngineSuccess, fmt.Sprintf("预约成功！%s @ %s", slotLabel, storeName))
		e.addLog(fmt.Sprintf("🎉 预约成功！号码: %s, 时段: %s, 门店: %s", reservation.Number, slotLabel, storeName))
		return
	}
}

// Stop 取消当前运行的任意任务（capture/booking/sniper）：cancel ctx、关代理、清系统代理，
// 然后最多等运行 goroutine 3s 退出（done 关闭）再强制回 idle，避免状态卡在 running。
func (e *BookingEngine) Stop() {
	e.mu.Lock()
	cancel := e.cancel
	done := e.done
	if e.cancel != nil {
		e.cancel()
	}
	if e.proxy != nil {
		e.proxy.Close()
		e.proxy = nil
		ClearSystemProxy()
		markProxyInactive()
	}
	e.mu.Unlock()

	if cancel != nil && done != nil {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}

	e.mu.Lock()
	if e.done == done {
		e.cancel = nil
		e.done = nil
	}
	e.state.Status = EngineIdle
	e.state.Message = "已停止"
	e.mu.Unlock()
	bus.publish("engine", mustJSON(e.GetState()))
}

// HasValidConfig checks if we have saved authentication tokens.
func HasValidConfig() bool {
	tokens, err := LoadLocalConfig()
	return err == nil && tokens.ValidateForReservation() == nil
}
