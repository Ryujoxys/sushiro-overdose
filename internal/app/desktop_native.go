//go:build desktop && (darwin || windows)

package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/Ryujoxys/sushiro-overdose/internal/core"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func cmdDesktop() {
	if err := runDesktop(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		showDesktopError(err.Error())
	}
}

func runDesktop() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	claimCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	instance, err := claimDesktop(claimCtx, core.AppDirPath())
	cancel()
	if err != nil || instance == nil {
		return err
	}
	defer instance.close()
	backend, err := startWebBackend(ctx, instance.activate)
	if err != nil {
		return err
	}
	defer backend.Close()
	if err := instance.publish(backend.URL); err != nil {
		return fmt.Errorf("无法保存窗口状态：%w", err)
	}

	var nativeContext context.Context
	ready := make(chan struct{})
	closed := make(chan struct{})
	bridge := &DesktopBridge{backend: backend, client: localHTTPClient()}
	bridge.saveDialog = func(name string) (string, error) {
		select {
		case <-ready:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		return wailsruntime.SaveFileDialog(nativeContext, wailsruntime.SaveDialogOptions{
			Title: "保存到本机", DefaultFilename: name,
			Filters: []wailsruntime.FileFilter{{DisplayName: "导出文件", Pattern: "*" + filepath.Ext(name)}},
		})
	}
	appMenu := menu.NewMenu()
	if runtime.GOOS == "darwin" {
		appMenu.Append(menu.AppMenu())
		appMenu.Append(menu.EditMenu())
	}
	appMenu.AddSubmenu("窗口").AddText("关闭窗口", nil, func(*menu.CallbackData) {
		select {
		case <-ready:
			wailsruntime.Quit(nativeContext)
		default:
		}
	})
	messages := windows.DefaultMessages()
	messages.MissingRequirements = "需要安装 WebView2"
	messages.InstallationRequired = "应用需要 Microsoft WebView2。点击确定下载安装，完成后重新打开应用。"
	messages.UpdateRequired = "请更新 Microsoft WebView2 后重新打开应用。点击确定下载安装。"
	messages.Webview2NotInstalled = "尚未安装 WebView2，请安装后重新打开应用。"
	messages.Error = "应用启动失败"
	messages.FailedToInstall = "WebView2 安装未完成，请重试或从微软官网下载运行时。"
	messages.DownloadPage = "请安装 Microsoft WebView2 后重新打开应用。点击确定前往微软官网。最低版本："
	messages.WebView2ProcessCrash = "窗口进程意外退出，请重新打开应用。本机记录仍在。"
	return wails.Run(&options.App{
		Title: "寿司郎 · 本地记录", Width: 1200, Height: 850, MinWidth: 760, MinHeight: 560,
		BackgroundColour:  options.NewRGB(242, 242, 242),
		HideWindowOnClose: false,
		AssetServer:       &assetserver.Options{Handler: http.HandlerFunc(handleIndex)},
		Bind:              []interface{}{bridge},
		Menu:              appMenu,
		Windows: &windows.Options{
			WebviewUserDataPath: filepath.Join(core.AppDirPath(), "webview"),
			Theme:               windows.Light, Messages: messages, WindowClassName: "SushiroDesktop",
		},
		Mac: &mac.Options{Appearance: mac.NSAppearanceNameAqua, About: &mac.AboutInfo{
			Title: "寿司郎 Overdose v" + Version, Message: Edition + "\n本地记录排队规律，需要时手动取号。",
		}},
		OnStartup: func(wctx context.Context) {
			nativeContext = wctx
			close(ready)
			go func() {
				for {
					select {
					case <-closed:
						return
					case <-ctx.Done():
						select {
						case <-closed:
							return
						default:
						}
						wailsruntime.Quit(wctx)
						return
					case <-instance.focus:
						wailsruntime.WindowUnminimise(wctx)
						wailsruntime.WindowShow(wctx)
					}
				}
			}()
		},
		OnShutdown: func(context.Context) { close(closed); backend.Close() },
		OnBeforeClose: func(wctx context.Context) bool {
			if ctx.Err() == nil && !bridge.prepareClose() {
				wailsruntime.EventsEmit(wctx, "desktop:busy")
				return true
			}
			return false
		},
	})
}
