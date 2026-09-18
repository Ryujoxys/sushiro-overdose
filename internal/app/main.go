package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/platform"

import . "github.com/Ryujoxys/sushiro-overdose/internal/proxy"

import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import . "github.com/Ryujoxys/sushiro-overdose/internal/notify"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Version is injected from the root main package (which receives it via ldflags).
var Version = "dev"

// Edition is separate from the numeric version used for update checks.
const Edition = "lite正式版"

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
	fmt.Printf("寿司郎 Overdose v%s · %s\n", Version, Edition)
	fmt.Println("本地记录排队规律，需要时手动取号。")
	fmt.Println("https://github.com/Ryujoxys/sushiro-overdose")
	fmt.Println()
}

func printUsage() {
	fmt.Println("用法: sushiro [命令]")
	fmt.Println("  web            打开本地界面（默认）")
	fmt.Println("  collect        排队记录服务（status|start|stop|run|autostart）")
	fmt.Println("  status         查看记录状态")
	fmt.Println("  doctor         只读诊断")
	fmt.Println("  diag-bundle    导出脱敏诊断包")
	fmt.Println("  repair-proxy   恢复旧版残留代理")
	fmt.Println("  config         通知与门店配置")
	fmt.Println("  version        查看版本")
	fmt.Println("记录无需认证；手动取号请在界面确认。")
}

func Run() {
	if err := ValidateDataHome(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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

	// Public collection must not import old credentials or start account work.
	if len(args) == 1 && (args[0] == "--queue-collector-child" || args[0] == "--sampler-daemon-child") {
		cmdPublicQueueDaemon()
		return
	}
	if len(args) >= 1 && args[0] == "collect" {
		cmdCollect(args[1:])
		return
	}
	os.MkdirAll(AppDirPath(), 0o755)
	MigrateOldConfig()

	if len(args) == 0 || (len(args) == 1 && args[0] == "web") {
		cmdWeb()
	} else if len(args) == 1 && (args[0] == "cli" || args[0] == "run" || args[0] == "-f" || args[0] == "--foreground") {
		cmdForeground()
	} else if len(args) == 1 && (args[0] == "start" || args[0] == "-d" || args[0] == "--daemon") {
		fmt.Println(retiredAutomationMessage)
	} else if len(args) == 1 && (args[0] == "exit" || args[0] == "stop") {
		cmdStop()
	} else if len(args) == 1 && args[0] == "status" {
		cmdCollect([]string{"status"})
	} else if len(args) == 1 && args[0] == "calendar" {
		cmdCalendar()
	} else if len(args) >= 1 && args[0] == "sniper" {
		fmt.Println(retiredAutomationMessage)
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
		fmt.Println(retiredAutomationMessage)
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
	fmt.Printf("寿司郎 Overdose v%s · %s\n", Version, Edition)
}

func cmdForeground() { cmdCollect([]string{"run"}) }

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
	if err := markProxyActive(actualPort, os.Getpid()); err != nil {
		return nil, err
	}
	if err := SetSystemProxy(actualPort); err != nil {
		if restoreErr := ClearSystemProxy(); restoreErr == nil {
			markProxyInactive()
		}
		return nil, fmt.Errorf("设置系统代理失败: %w", err)
	}
	fmt.Printf("系统代理已设置 (127.0.0.1:%d)\n", actualPort)
	fmt.Println("请彻底关闭 PC 微信后重新打开，在寿司郎小程序里选任意门店点一次「排队」或「预约」（不必真的提交）")

	// Ensure proxy is cleared on exit
	defer func() {
		if err := ClearSystemProxy(); err != nil {
			fmt.Println("代理恢复失败，已保留恢复标记:", err)
			return
		}
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
