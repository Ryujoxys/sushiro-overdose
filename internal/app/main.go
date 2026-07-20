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
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Version is injected from the root main package (which receives it via ldflags).
var Version = "dev"

// SetVersion lets the root package pass through the ldflags-provided version.
func SetVersion(v string) {
	if v != "" {
		Version = v
	}
}

func printBanner() {
	fmt.Println(" ▄██████▄  ████████▄     ▄████████ ███    █▄     ▄████████    ▄█    █▄     ▄█")
	fmt.Println("███    ███ ███   ▀███   ███    ███ ███    ███   ███    ███   ███    ███   ███")
	fmt.Println("███    ███ ███    ███   ███    █▀  ███    ███   ███    █▀    ███    ███   ███▌")
	fmt.Println("███    ███ ███    ███   ███        ███    ███   ███         ▄███▄▄▄▄███▄▄ ███▌")
	fmt.Println("███    ███ ███    ███ ▀███████████ ███    ███ ▀███████████ ▀▀███▀▀▀▀███▀  ███▌")
	fmt.Println("███    ███ ███    ███          ███ ███    ███          ███   ███    ███   ███")
	fmt.Println("███    ███ ███   ▄███    ▄█    ███ ███    ███    ▄█    ███   ███    ███   ███")
	fmt.Println(" ▀██████▀  ████████▀   ▄████████▀  ████████▀   ▄████████▀    ███    █▀    █▀")
	fmt.Println()
	fmt.Printf("寿司郎 Overdose v%s — https://github.com/Ryujoxys/sushiro-overdose\n", Version)
	fmt.Println()
}

func printUsage() {
	fmt.Println("用法: sushiro [命令]")
	fmt.Println()
	fmt.Println("默认 (无命令)   启动 Web UI（推荐，独立窗口优先）")
	fmt.Println()
	fmt.Println("运行:")
	fmt.Println("  web            启动 Web UI")
	fmt.Println("  cli            前台终端模式（高级）")
	fmt.Println("  start          后台静默抢号")
	fmt.Println("  status         查看运行状态")
	fmt.Println("  exit           停止后台进程")
	fmt.Println("  version        打印版本号并退出（-v / --version 同义）")
	fmt.Println()
	fmt.Println("预约:")
	fmt.Println("  calendar       查看近 7 天可预约时段")
	fmt.Println("  sniper         狙击模式（提前锁定未开放时段）")
	fmt.Println("  list           查看当前预约")
	fmt.Println("  cancel <id>    取消预约")
	fmt.Println()
	fmt.Println("数据与推荐:")
	fmt.Println("  trends         分析时段可用率趋势")
	fmt.Println("  recommend      智能推荐最佳时段")
	fmt.Println("  sample         信息收集（once|run|start|stop|autostart）")
	fmt.Println()
	fmt.Println("配置与维护:")
	fmt.Println("  config         通知/门店配置（feishu|telegram|bark|serverchan|store）")
	fmt.Println("  doctor         打印只读诊断")
	fmt.Println("  diag-bundle    导出脱敏证据包（zip）")
	fmt.Println("  auth-probe     测试已存凭证连通性")
	fmt.Println("  repair-proxy   恢复系统代理")
	fmt.Println("  stop-processes 停止相关进程")
	fmt.Println("  uninstall      移除本地敏感数据与证书")
	fmt.Println("  help           显示本帮助")
}

func Run() {
	args := os.Args[1:]
	if len(args) == 1 && (args[0] == "doctor" || args[0] == "diagnostics") {
		cmdDoctor()
		return
	}
	if len(args) == 1 && (args[0] == "diag-bundle" || args[0] == "bundle") {
		cmdDiagBundle()
		return
	}
	if len(args) == 1 && (args[0] == "version" || args[0] == "-v" || args[0] == "--version") {
		cmdVersion()
		return
	}

	os.MkdirAll(AppDirPath(), 0o755)
	MigrateOldConfig()

	if len(args) == 0 || (len(args) == 1 && args[0] == "web") {
		cmdWeb()
	} else if len(args) == 1 && (args[0] == "cli" || args[0] == "run" || args[0] == "-f" || args[0] == "--foreground") {
		cmdForeground()
	} else if len(args) == 1 && (args[0] == "start" || args[0] == "-d" || args[0] == "--daemon") {
		cmdStart()
	} else if len(args) == 1 && (args[0] == "exit" || args[0] == "stop") {
		cmdStop()
	} else if len(args) == 1 && args[0] == "status" {
		cmdStatus()
	} else if len(args) == 1 && args[0] == "calendar" {
		cmdCalendar()
	} else if len(args) >= 1 && args[0] == "sniper" {
		cmdSniper(args[1:])
	} else if len(args) == 1 && (args[0] == "list" || args[0] == "reservations") {
		cmdList()
	} else if len(args) >= 1 && args[0] == "cancel" {
		cmdCancel(args[1:])
	} else if len(args) == 1 && (args[0] == "trends" || args[0] == "history") {
		cmdTrends()
	} else if len(args) == 1 && (args[0] == "recommend" || args[0] == "rec") {
		cmdRecommend()
	} else if len(args) == 1 && (args[0] == "auth-probe" || args[0] == "probe-auth") {
		cmdAuthProbe()
	} else if len(args) == 1 && args[0] == "--daemon-child" {
		cmdDaemon()
	} else if len(args) == 1 && args[0] == "--sampler-daemon-child" {
		cmdSamplerDaemon()
	} else if len(args) == 1 && args[0] == "--mcp-daemon-child" {
		cmdMCPDaemon()
	} else if len(args) >= 1 && (args[0] == "sample" || args[0] == "sampling") {
		cmdSample(args[1:])
	} else if len(args) == 1 && (args[0] == "repair-proxy" || args[0] == "repair") {
		cmdRepairProxy()
	} else if len(args) >= 1 && (args[0] == "uninstall" || args[0] == "purge") {
		cmdUninstall(args[1:])
	} else if len(args) == 1 && (args[0] == "stop-processes" || args[0] == "kill-processes") {
		cmdStopProcesses()
	} else if len(args) >= 1 && (args[0] == "config" || args[0] == "setting" || args[0] == "settings") {
		cmdConfig(args[1:])
	} else if len(args) >= 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
		printUsage()
	} else {
		printUsage()
	}
}

// ---- CLI Commands ----

// cmdVersion 只打印版本号就退出，不做任何 IO/目录副作用（排在 MkdirAll 之前）。
// 用途：排障（"你装的是哪个版本？"）、打包/安装脚本校验产物、CI 里确认 ldflags 注入成功。
// Version 未通过 ldflags 注入时为 "dev"（源码构建），这本身也是有用的诊断信息。
func cmdVersion() {
	fmt.Printf("寿司郎 Overdose v%s\n", Version)
}

func cmdForeground() {
	printBanner()

	// Check for stale proxy from a previous crashed run
	if checkStaleProxy() {
		fmt.Println("已清除上次异常退出的系统代理设置")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func cmdConfig(args []string) {
	if len(args) == 0 {
		fmt.Println("当前通知设置:")
		cfg, _ := LoadNotifyConfig()
		if cfg == nil {
			cfg = &NotifyConfig{}
		}
		fmt.Printf("  飞书: %s\n", notifyStatus(cfg.Feishu.Webhook))
		fmt.Printf("  Telegram: %s\n", notifyStatus(cfg.Telegram.Token))
		fmt.Printf("  Bark: %s\n", notifyStatus(cfg.Bark.Key))
		fmt.Printf("  Server酱: %s\n", notifyStatus(cfg.ServerChan.Key))
		fmt.Println()
		fmt.Print("配置飞书通知？输入 Webhook 地址（留空跳过，输入 clear 清除）: ")
		input := ReadInput()
		if strings.ToLower(input) == "clear" {
			updateLocalConfigFeishu("")
			fmt.Println("飞书通知已清除")
		} else if input != "" {
			updateLocalConfigFeishu(input)
			fmt.Println("飞书通知已配置!")
		}
		return
	}

	switch args[0] {
	case "feishu":
		if len(args) < 2 {
			fmt.Println("Usage: sushiro config feishu <webhook_url>")
			return
		}
		if args[1] == "--clear" {
			updateLocalConfigFeishu("")
			fmt.Println("飞书通知已清除")
			return
		}
		updateLocalConfigFeishu(args[1])
		fmt.Println("飞书通知已配置!")
	case "telegram":
		if len(args) < 3 {
			fmt.Println("Usage: sushiro config telegram <bot_token> <chat_id>")
			return
		}
		cfg, _ := LoadNotifyConfig()
		if cfg == nil {
			cfg = &NotifyConfig{}
		}
		cfg.Telegram.Token = args[1]
		cfg.Telegram.ChatID = args[2]
		SaveNotifyConfig(cfg)
		fmt.Println("Telegram 通知已配置!")
	case "bark":
		if len(args) < 3 {
			fmt.Println("Usage: sushiro config bark <server_url> <device_key>")
			return
		}
		cfg, _ := LoadNotifyConfig()
		if cfg == nil {
			cfg = &NotifyConfig{}
		}
		cfg.Bark.URL = args[1]
		cfg.Bark.Key = args[2]
		SaveNotifyConfig(cfg)
		fmt.Println("Bark 通知已配置!")
	case "serverchan", "server-chan", "sct":
		if len(args) < 2 {
			fmt.Println("Usage: sushiro config serverchan <send_key>")
			return
		}
		cfg, _ := LoadNotifyConfig()
		if cfg == nil {
			cfg = &NotifyConfig{}
		}
		cfg.ServerChan.Key = args[1]
		SaveNotifyConfig(cfg)
		fmt.Println("Server酱 通知已配置!")
	case "store":
		if len(args) < 2 {
			reg := GetStoreRegistry()
			stores := reg.List()
			if len(stores) == 0 {
				fmt.Println("无已配置的门店昵称")
			} else {
				for _, s := range stores {
					fmt.Printf("  %s -> %s\n", s.ID, s.Nickname)
				}
			}
			return
		}
		reg := GetStoreRegistry()
		switch args[1] {
		case "add":
			if len(args) < 4 {
				fmt.Println("Usage: sushiro config store add <store_id> <nickname>")
				return
			}
			reg.Add(args[2], args[3])
			fmt.Printf("门店 %s 昵称已设置为 %s\n", args[2], args[3])
		case "remove", "rm", "delete":
			if len(args) < 3 {
				fmt.Println("Usage: sushiro config store remove <store_id>")
				return
			}
			reg.Remove(args[2])
			fmt.Printf("门店 %s 昵称已移除\n", args[2])
		default:
			fmt.Println("Usage: sushiro config store [add|remove]")
		}
	default:
		fmt.Println("未知配置项:", args[0])
		fmt.Println("可用: feishu, telegram, bark, serverchan, store")
	}
}

func notifyStatus(v string) string {
	if v != "" {
		return "已配置 (" + v[:min(30, len(v))] + "...)"
	}
	return "未配置"
}

// ---- Update feishu in local config ----

func updateLocalConfigFeishu(webhook string) {
	SaveFeishuConfig(webhook)
	// Also update notify config
	cfg, _ := LoadNotifyConfig()
	if cfg == nil {
		cfg = &NotifyConfig{}
	}
	cfg.Feishu.Webhook = webhook
	SaveNotifyConfig(cfg)
	// Also update in-memory tokens if loaded
	tokens, err := LoadLocalConfig()
	if err == nil {
		tokens.FeishuWebhook = webhook
	}
}

// ---- Notification helper ----

var globalNotifierMu sync.RWMutex
var globalNotifier *MultiNotifier

func setNotifier(n *MultiNotifier) {
	globalNotifierMu.Lock()
	globalNotifier = n
	globalNotifierMu.Unlock()
}

func sendNotification(title, content string) {
	globalNotifierMu.RLock()
	n := globalNotifier
	globalNotifierMu.RUnlock()
	if n != nil {
		n.Send(context.Background(), title, content)
	}
}

// ---- Main run logic (shared by foreground and daemon) ----

func run(ctx context.Context) error {
	tokens, configExists := tryLoadConfig()

	if !configExists {
		var err error
		tokens, err = runCapturePhase(ctx)
		if err != nil {
			return err
		}
		if err := SaveLocalConfig(tokens); err != nil {
			LogMessage(time.Now(), "保存配置失败: "+err.Error())
		} else {
			LogMessage(time.Now(), "配置已保存到 "+LocalConfigPath())
		}
	}

	settings := tokens.ToSettings()

	// Initialize notifier
	setNotifier(BuildNotifierFromConfig())

	// Interactive Feishu config (legacy support)
	tokens.Lock()
	feishu := tokens.FeishuWebhook
	tokens.Unlock()
	if feishu == "" {
		fmt.Println()
		fmt.Print("是否配置飞书通知机器人？(y/N): ")
		if answer := ReadInput(); strings.ToLower(answer) == "y" {
			fmt.Println("飞书群 → 群设置 → 群机器人 → 添加自定义机器人 → 复制 Webhook 地址")
			fmt.Print("请输入 Webhook 地址: ")
			if webhook := ReadInput(); webhook != "" {
				tokens.Lock()
				tokens.FeishuWebhook = webhook
				tokens.Unlock()
				SaveFeishuConfig(webhook)
				settings.FeishuWebhook = webhook
				// Add feishu to notifier
				globalNotifier.Add(NewFeishuNotifier(webhook))
				fmt.Println("飞书通知已配置!")
			}
		}
	}

	client := NewClient(settings)

	// Verify config still works
	LogMessage(time.Now(), "验证凭证参数...")
	if _, err := client.GetTimeslots(ctx, settings.StoreIDs[0]); err != nil {
		LogMessage(time.Now(), "验证失败: "+err.Error())
		LogMessage(time.Now(), "凭证参数可能已过期，需要重新获取...")
		sendNotification("寿司郎 - 凭证过期", "需要重新运行捕获凭证参数")
		DeleteLocalConfig()
		tokens, err = runCapturePhase(ctx)
		if err != nil {
			return err
		}
		SaveLocalConfig(tokens)
		settings = tokens.ToSettings()
		client = NewClient(settings)
	}

	if err := tokens.ValidateForReservation(); err != nil {
		LogMessage(time.Now(), "预约参数不完整，需要重新捕获: "+err.Error())
		DeleteLocalConfig()
		tokens, err = runCapturePhase(ctx)
		if err != nil {
			return err
		}
		if err := tokens.ValidateForReservation(); err != nil {
			return err
		}
		if err := SaveLocalConfig(tokens); err != nil {
			LogMessage(time.Now(), "保存配置失败: "+err.Error())
		}
		settings = tokens.ToSettings()
		client = NewClient(settings)
	}

	selectedStores, err := SelectStores(ctx, client, tokens)
	if err != nil {
		return fmt.Errorf("选择门店失败: %w", err)
	}
	settings.StoreIDs = selectedStores

	tokens.Lock()
	tokens.StoreIDs = selectedStores
	tokens.Unlock()
	SaveLocalConfig(tokens)

	prefs := LoadPreferences()
	prefs.SelectedStores = selectedStores

	// CLI mode: offer to configure slots interactively or use saved
	if len(prefs.WeekdaySlots) == 0 && len(prefs.SaturdaySlots) == 0 && len(prefs.SundaySlots) == 0 {
		slotConfig := ConfigureSlots()
		prefs.WeekdaySlots = SlotPrefToRanges(slotConfig.Weekday)
		prefs.SaturdaySlots = SlotPrefToRanges(slotConfig.Saturday)
		prefs.SundaySlots = SlotPrefToRanges(slotConfig.Sunday)
	} else {
		fmt.Println("\n使用已保存的时段偏好（可通过 Web UI 修改）")
	}
	SavePreferences(prefs)

	healthStop := startHealthCheck(ctx, client, selectedStores)
	defer close(healthStop)

	LogMessage(time.Now(), "开始抢号...")
	runBookingLoop(ctx, client, settings, selectedStores, prefs)
	return nil
}

func runCapturePhase(ctx context.Context) (*CapturedTokens, error) {
	doneActivity := markMainFlowActive("capturing")
	defer doneActivity()

	// Load or generate CA certificate
	caCert, caKey, err := LoadOrGenerateCA()
	if err != nil {
		return nil, fmt.Errorf("CA证书加载失败: %w", err)
	}

	// Check if cert is trusted, offer to install
	trusted, _ := IsCertTrusted()
	if !trusted {
		fmt.Println("\n首次运行需要安装CA证书（用于拦截HTTPS流量）")
		fmt.Println("安装后需要输入登录密码确认")
		if err := InstallCert(); err != nil {
			return nil, fmt.Errorf("证书安装失败: %w", err)
		}
		fmt.Println("证书安装成功!")
	}

	tokens := NewCapturedTokens()

	// Start MITM proxy
	proxy, err := StartProxy(caCert, caKey, tokens)
	if err != nil {
		return nil, fmt.Errorf("启动代理失败: %w", err)
	}
	defer proxy.Close()
	actualPort := proxy.Port()

	// Set system proxy
	if err := SetSystemProxy(actualPort); err != nil {
		return nil, fmt.Errorf("设置系统代理失败: %w", err)
	}
	fmt.Printf("系统代理已设置 (127.0.0.1:%d)\n", actualPort)
	fmt.Println("请彻底关闭 PC 微信后重新打开，在寿司郎小程序里选任意门店点一次「排队」或「预约」（不必真的提交）")
	markProxyActive(actualPort, os.Getpid())

	// Ensure proxy is cleared on exit
	defer func() {
		ClearSystemProxy()
		markProxyInactive()
		fmt.Println("系统代理已清除")
	}()

	// Wait for capture with skip channel
	skipCapture := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			close(skipCapture)
		default:
			var buf [1]byte
			os.Stdin.Read(buf[:])
			close(skipCapture)
		}
	}()

	if err := WaitForCapture(ctx, tokens, skipCapture); err != nil {
		return nil, err
	}

	// Prompt phone number if not captured
	tokens.Lock()
	phone := tokens.PhoneNumber
	tokens.Unlock()
	if phone == "" {
		fmt.Print("\n请输入手机号: ")
		input := ReadInput()
		tokens.Lock()
		tokens.PhoneNumber = input
		tokens.Unlock()
	}

	return tokens, nil
}

func tryLoadConfig() (*CapturedTokens, bool) {
	tokens, err := LoadLocalConfig()
	if err != nil {
		return nil, false
	}
	if err := tokens.ValidateForQuery(); err != nil {
		LogMessage(time.Now(), "已保存配置不可用: "+err.Error())
		return nil, false
	}
	LogMessage(time.Now(), "使用已保存的配置")
	LogMessage(time.Now(), fmt.Sprintf("  手机号: %s", MaskPhone(tokens.PhoneNumber)))
	LogMessage(time.Now(), fmt.Sprintf("  门店: %v", tokens.StoreIDs))
	return tokens, true
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	if IsHTTPStatus(err, http.StatusUnauthorized) || IsHTTPStatus(err, http.StatusForbidden) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "HTTP 401") ||
		strings.Contains(msg, "HTTP 403")
}

func runBookingLoop(ctx context.Context, client *Client, settings Settings, storeIDs []string, prefs UserPreferences) {
	doneActivity := markMainFlowActive("booking")
	defer doneActivity()

	var booked map[string]bool
	temporarySkips := map[string]time.Time{}
	errStreak := 0
	authErrors := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		now := time.Now().In(settings.Location)

		var best *TargetSlot
		for _, storeID := range storeIDs {
			slots, err := client.GetTimeslots(ctx, storeID)
			if err != nil {
				if isAuthError(err) {
					authErrors++
					if authErrors >= 3 {
						LogMessage(now, "凭证失败，请重新运行获取新参数")
						sendNotification("寿司郎 - 凭证失败", "凭证参数已失效，请重新打开 sushiro 重新捕获")
						DeleteLocalConfig()
						return
					}
				}
				errStreak++
				if errStreak >= 5 {
					LogMessage(now, "连续失败过多，等待5秒...")
					time.Sleep(5 * time.Second)
					errStreak = 0
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}
			errStreak = 0
			authErrors = 0

			// Record slot history
			appendHistory(slots, storeID)

			for i := range slots {
				if !prefs.ShouldTarget(slots[i], settings.Location) {
					continue
				}
				// Skip non-available slots
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
				if best == nil || prefs.PreferTargetSlot(*t, *best, settings.Location, storeIDs) {
					best = t
				}
			}
		}

		if best == nil {
			fmt.Printf("\r[%s] 查询中...无目标时段", now.Format("15:04:05"))
			time.Sleep(100 * time.Millisecond)
			continue
		}

		slotLabel := FormatSlotWindow(best.Date, best.Start, best.End, settings.Location)
		fmt.Printf("\r[%s] %s - 尝试预约...", now.Format("15:04:05"), slotLabel)

		reservation, err := client.CreateReservation(ctx, best.StoreID, best.Date, best.Start)
		if err != nil {
			if isAuthError(err) {
				authErrors++
				if authErrors >= 3 {
					LogMessage(now, "凭证失败，请重新运行")
					sendNotification("寿司郎 - 凭证失败", "请重新打开 sushiro 重新捕获")
					DeleteLocalConfig()
					return
				}
			}

			if isOfficialServerHTTPError(err) {
				key := bookingSlotKey(best.StoreID, best.Date, best.Start)
				markTemporaryBookingSkip(temporarySkips, key, now)
				LogMessage(now, bookingServerErrorLog(slotLabel, err))
			} else if errors.Is(err, ErrNoReservationAvailable) {
				key := bookingSlotKey(best.StoreID, best.Date, best.Start)
				if booked == nil {
					booked = make(map[string]bool)
				}
				booked[key] = true
				fmt.Printf("\r[%s] %s - 名额已满", now.Format("15:04:05"), slotLabel)
			} else {
				fmt.Printf("\r[%s] %s - 失败: %s", now.Format("15:04:05"), slotLabel, err)
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Success!
		now = time.Now().In(settings.Location)
		storeInfo, _ := client.GetStoreInfo(ctx, best.StoreID)
		storeName := storeInfo.Name
		if storeName == "" {
			storeName = best.StoreID
		}
		reservation.MonitoredStoreID = best.StoreID
		onBookingSuccess(reservation, storeName, storeInfo.Address, slotLabel, "预约")

		return
	}
}
