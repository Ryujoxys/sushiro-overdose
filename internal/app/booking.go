package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/platform"

import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func cmdList() {
	printBanner()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tokens, ok := tryLoadConfig()
	if !ok {
		fmt.Println("暂无配置，请先运行 sushiro 完成参数捕获")
		return
	}
	if err := tokens.ValidateForReservation(); err != nil {
		fmt.Println(err)
		fmt.Println("请重新运行 sushiro 完成参数捕获")
		return
	}
	settings := tokens.ToSettings()
	client := NewClient(settings)

	reservations, err := client.GetReservations(ctx)
	if err != nil {
		if errors.Is(err, ErrReservationsEndpointUnavailable) {
			state, stateErr := LoadState(StateFilePath())
			if stateErr == nil && (state.ActiveReservation != nil || state.ActiveNetTicket != nil) {
				fmt.Println("官方当前预约接口不可用，显示本地保存的最近预约/排队号记录。")
				if state.ActiveNetTicket != nil {
					reservations = append(reservations, *state.ActiveNetTicket)
				}
				if state.ActiveReservation != nil {
					reservations = append(reservations, *state.ActiveReservation)
				}
			} else {
				fmt.Println("官方当前预约接口不可用，且本地没有保存的预约记录。")
				return
			}
		} else {
			fmt.Println("获取预约列表失败:", err)
			return
		}
	}

	if len(reservations) == 0 {
		fmt.Println("当前无预约记录")
		return
	}

	fmt.Println("\n=== 当前预约 ===")
	for i, r := range reservations {
		status := "未知"
		if r.Status != "" {
			status = r.Status
		}
		queueDate := r.QueueDate
		if queueDate == "" {
			queueDate = "未知"
		}
		fmt.Printf("\n  %d. Ticket #%d\n", i+1, r.TicketID)
		fmt.Printf("     号码: %s\n", r.Number)
		fmt.Printf("     日期: %s\n", queueDate)
		if r.Start != "" {
			fmt.Printf("     时段: %s-%s\n", FormatCompactTime(r.Start), FormatCompactTime(DefaultString(r.End, r.Start)))
		}
		fmt.Printf("     状态: %s\n", status)
		if r.StoreID != "" {
			info, _ := client.GetStoreInfo(ctx, r.StoreID)
			if info.Name != "" {
				fmt.Printf("     门店: %s\n", info.Name)
			}
		}
		if r.Wait > 0 {
			fmt.Printf("     等待: %d 桌\n", r.Wait)
		}
		fmt.Printf("     成人: %d / 儿童: %d / 桌型: %s\n", r.NumAdult, r.NumChild, r.TableType)
	}
	fmt.Println()
}

func cmdCancel(args []string) {
	printBanner()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if len(args) == 0 {
		fmt.Println("Usage: sushiro cancel <ticket_id>")
		fmt.Println("用 sushiro list 查看预约列表")
		return
	}

	ticketID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Println("无效的 ticket_id:", args[0])
		return
	}

	tokens, ok := tryLoadConfig()
	if !ok {
		fmt.Println("暂无配置，请先运行 sushiro 完成参数捕获")
		return
	}
	if err := tokens.ValidateForReservation(); err != nil {
		fmt.Println(err)
		fmt.Println("请重新运行 sushiro 完成参数捕获")
		return
	}
	settings := tokens.ToSettings()
	client := NewClient(settings)

	fmt.Printf("正在取消预约 #%d...\n", ticketID)
	if err := client.CancelReservation(ctx, ticketID); err != nil {
		fmt.Println("取消失败:", err)
		return
	}

	fmt.Printf("预约 #%d 已取消\n", ticketID)
	LogMessage(time.Now(), fmt.Sprintf("预约 #%d 已取消", ticketID))
	// 清掉本地预约槽，避免 fallback 仍显示已取消的预约。
	if err := clearActiveReservationSlot(); err != nil {
		LogMessage(time.Now(), "清除本地预约状态失败: "+err.Error())
	}
}

// onBookingSuccess handles the shared logic for a successful booking:
// save state, print banner, log, send notifications.
func onBookingSuccess(reservation ReservationRecord, storeName, storeAddress, slotLabel, mode string) {
	now := time.Now()

	reservation.StoreName = storeName
	reservation.StoreAddress = storeAddress
	reservation.SlotLabel = slotLabel

	// 按 mode 分流写入对应槽：取号写 ActiveNetTicket，预约/狙击写 ActiveReservation。
	// load-merge-save 保证另一槽不被覆盖（历史 bug：单槽整盘写会冲掉先有的另一类记录）。
	var saveErr error
	if mode == "取号" {
		if strings.TrimSpace(reservation.Kind) == "" {
			reservation.Kind = "net_ticket"
		}
		saveErr = saveActiveNetTicketSlot(reservation)
	} else {
		if strings.TrimSpace(reservation.Kind) == "" {
			reservation.Kind = "reservation"
		}
		saveErr = saveActiveReservationSlot(reservation)
	}
	if saveErr != nil {
		LogMessage(now, "保存预约状态失败: "+saveErr.Error())
	}

	fmt.Println()
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Printf("  ║         🎉 %s！              ║\n", mode)
	fmt.Println("  ╠══════════════════════════════════════╣")
	fmt.Printf("  ║  门店：%s\n", storeName)
	fmt.Printf("  ║  时段：%s\n", slotLabel)
	fmt.Printf("  ║  号码：%s\n", reservation.Number)
	if storeAddress != "" {
		fmt.Printf("  ║  地址：%s\n", storeAddress)
	}
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Println()

	LogMessage(now, fmt.Sprintf("=== %s成功 ===", mode))
	LogMessage(now, fmt.Sprintf("  门店：%s", storeName))
	LogMessage(now, fmt.Sprintf("  时段：%s", slotLabel))
	LogMessage(now, fmt.Sprintf("  号码：%s", reservation.Number))
	if storeAddress != "" {
		LogMessage(now, fmt.Sprintf("  地址：%s", storeAddress))
	}

	title := fmt.Sprintf("寿司郎%s成功 - %s", mode, storeName)
	message := fmt.Sprintf("号码: %s | 时段: %s", reservation.Number, slotLabel)
	DesktopNotification(title, message)

	content := fmt.Sprintf("### %s成功 - %s\n**号码**：`%s`\n**时段**：%s\n**地址**：%s",
		mode, storeName, reservation.Number, slotLabel, storeAddress)
	sendNotification(title, content)
}
