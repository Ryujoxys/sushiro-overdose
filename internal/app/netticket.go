package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	netTicketPlanFile          = "netticket_plan.json"
	netTicketPlanSourceRoutine = "routine"
	netTicketWindowMinutes     = 12 // 到点后多久内仍尝试取号；过窗口则放弃
	netTicketTickSeconds       = 20
	// netTicketMaxServerRetries 限制官方服务器 5xx 等临时错误下的重试次数，避免
	// 服务器持续抖动时每 20 秒发一次取号请求（刷号风险）。超过次数即转 error 放弃。
	netTicketMaxServerRetries = 6
)

// NetTicketPlan 是「定时取号」计划：到指定时间自动远程取号。
type NetTicketPlan struct {
	Enabled            bool   `json:"enabled"`
	StoreID            string `json:"store_id"`
	StoreName          string `json:"store_name,omitempty"`
	TriggerMode        string `json:"trigger_mode,omitempty"` // "time"(默认到点) / "on_open"(一开放就取号)
	TargetTime         string `json:"target_time"`            // "HHMM"，仅 time 模式使用
	Source             string `json:"source,omitempty"`       // 空/手动，或 routine
	TargetMealTime     string `json:"target_meal_time,omitempty"`
	RoutinePlannedDate string `json:"routine_planned_date,omitempty"`
	Status             string `json:"status"` // idle/armed/success/error/expired
	Number             string `json:"number,omitempty"`
	TicketID           int64  `json:"ticket_id,omitempty"`
	FiredDate          string `json:"fired_date,omitempty"` // 当天已执行(成功或放弃)的日期 YYYY-MM-DD
	FiredAt            string `json:"fired_at,omitempty"`
	LastError          string `json:"last_error,omitempty"`
	// ServerRetryCount 是官方服务器临时错误（5xx 等）下的连续重试计数，
	// 达到 netTicketMaxServerRetries 后放弃，避免一直重发取号请求。
	ServerRetryCount int `json:"server_retry_count,omitempty"`
	// RetryDate 记录 ServerRetryCount 所属的日期 YYYY-MM-DD，仅跨天才清零计数。
	// 之前的实现只判断 ServerRetryCount!=0 就清零，导致每个 tick 都清零、
	// 重试上限永远失效、5xx 时每 20s 无限重发取号请求（刷号风险）。
	RetryDate string `json:"retry_date,omitempty"`
}

func netTicketPlanPath() string { return filepath.Join(AppDirPath(), netTicketPlanFile) }

// LoadNetTicketPlan 从磁盘读取排队号计划。读取/解析失败都回退成一个 idle 空计划（不报错）。
// 兼容旧数据：若状态是 error 但错误文案其实是「已发过号」，把它改判为 issued_unknown
// （这是历史版本没区分这两种语义留下的脏数据修正）。
func LoadNetTicketPlan() NetTicketPlan {
	data, err := os.ReadFile(netTicketPlanPath())
	if err != nil {
		return NetTicketPlan{Status: "idle"}
	}
	var p NetTicketPlan
	if json.Unmarshal(data, &p) != nil {
		return NetTicketPlan{Status: "idle"}
	}
	if p.Status == "error" && isTicketAlreadyIssuedText(p.LastError) {
		p.Status = "issued_unknown"
	}
	return normalizeNetTicketPlan(p, time.Now())
}

func SaveNetTicketPlan(p NetTicketPlan) error {
	os.MkdirAll(AppDirPath(), 0o755)
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(netTicketPlanPath(), data, 0o600)
}

// normalizeNetTicketPlan 规整排队号计划：补默认状态、以及关键的「终态自动失能」——
// 计划已 success/error/expired 后还 Enabled 会被强行关掉，避免已结束的计划被下个 tick 当成有效计划继续触发。
func normalizeNetTicketPlan(p NetTicketPlan, now time.Time) NetTicketPlan {
	if now.IsZero() {
		now = time.Now()
	}
	if p.Status == "" {
		p.Status = "idle"
	}
	if p.Enabled && netTicketPlanTerminal(p.Status) {
		p.Enabled = false
	}
	return p
}

// netTicketPlanTerminal 判断状态是否为终态（不可再推进）：
// success/issued_unknown（已取到号）、expired（过窗口放弃）、error（失败放弃）。
// armed/idle/retrying 都不是终态，tick 还会继续推进它们。
func netTicketPlanTerminal(status string) bool {
	switch strings.TrimSpace(status) {
	case "success", "issued_unknown", "expired", "error":
		return true
	default:
		return false
	}
}

// netTicketPlanFiredOn 判断计划在指定日期（默认今天）是否已「触发过」（成功或放弃都算）。
// 触发的判定优先看 FiredDate 字段；老数据可能只有 FiredAt（RFC3339 时刻），则解析出日期比对，做向后兼容。
// 用途：跨天判断「今天是否还要取号」，以及 Routine/采样的「今天是否已取到号」判断。
func netTicketPlanFiredOn(p NetTicketPlan, day time.Time) bool {
	if day.IsZero() {
		day = time.Now()
	}
	// 「今天」按 CST 算，避免 TZ=UTC 跨 CST 午夜时重复取号或漏取。
	day = day.In(SushiroTimezone)
	today := day.Format("2006-01-02")
	if strings.TrimSpace(p.FiredDate) == today {
		return true
	}
	if strings.TrimSpace(p.FiredAt) == "" {
		return false
	}
	firedAt, ok := parseRFC3339Local(p.FiredAt)
	return ok && firedAt.Format("2006-01-02") == today
}

var netTicketMu sync.Mutex

// NetTicketScheduler 在后台轮询，到点自动取号。可同时跑在 web 进程和后台守护里，
// 用按日期的独占锁文件保证一天只取一次、不会重复。
type NetTicketScheduler struct {
	mu      sync.Mutex
	running bool
}

var netTicketSched = &NetTicketScheduler{}

func (s *NetTicketScheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go func() {
		t := time.NewTicker(netTicketTickSeconds * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				func() {
					// 每个 tick 单独 recover：某次取号逻辑 panic 不能让调度器整个循环退出，
					// 否则后续到点的取号计划再也不会触发。
					defer func() {
						if r := recover(); r != nil {
							LogMessage(time.Now(), "定时取号 tick 发生 panic 已恢复："+fmt.Sprint(r))
						}
					}()
					netTicketTick(ctx)
				}()
			}
		}
	}()
}

// netTicketTick 是取号调度器每个 tick（约 20s）的入口，全程持 netTicketMu 串行化。
// 它先推进 Routine 状态机（refreshNetTicketRoutineLocked），再处理「定时取号计划」：
// 跨天清重试计数 -> 按 trigger_mode 分发到 time 或 on_open 触发路径。
func netTicketTick(ctx context.Context) {
	netTicketMu.Lock()
	defer netTicketMu.Unlock()

	now := time.Now().In(SushiroTimezone)
	refreshNetTicketRoutineLocked(now)
	plan := LoadNetTicketPlan()
	if !plan.Enabled || strings.TrimSpace(plan.StoreID) == "" {
		return
	}
	today := now.Format("2006-01-02")
	if plan.FiredDate == today {
		return // 今天已处理过
	}
	// 仅在真正跨天时清掉历史重试计数，今天重新开始计数。
	// （不能只看 ServerRetryCount!=0：重试分支会置空 FiredDate，下个 tick 又走到这里，
	// 那样每 tick 都清零，重试上限永远失效。）
	if plan.ServerRetryCount != 0 && plan.RetryDate != today {
		plan.ServerRetryCount = 0
		plan.RetryDate = ""
		if err := SaveNetTicketPlan(plan); err != nil {
			LogMessage(now, "保存排队号计划失败: "+err.Error())
		}
	}
	if plan.TriggerMode == "on_open" {
		netTicketTickOnOpen(ctx, plan, now, today)
		return
	}
	netTicketTickByTime(ctx, plan, now, today)
}

// netTicketTickByTime 是「到点取号」：到设定时间的窗口内自动取号。
func netTicketTickByTime(ctx context.Context, plan NetTicketPlan, now time.Time, today string) {
	if len(strings.TrimSpace(plan.TargetTime)) < 3 {
		return
	}
	target, ok := netTicketTargetToday(plan.TargetTime, now)
	if !ok {
		return
	}
	if now.Before(target) {
		if plan.Status != "armed" {
			plan.Status = "armed"
			plan.LastError = ""
			if err := SaveNetTicketPlan(plan); err != nil {
				LogMessage(now, "保存排队号计划失败: "+err.Error())
			}
		}
		return
	}
	if now.After(target.Add(netTicketWindowMinutes * time.Minute)) {
		plan.Enabled = false
		plan.Status = "expired"
		plan.FiredDate = today
		plan.FiredAt = now.Format(time.RFC3339)
		if err := SaveNetTicketPlan(plan); err != nil {
			LogMessage(now, "保存排队号计划失败: "+err.Error())
		}
		sendQueueAlert(ctx, "⏰ 定时取号未执行", DefaultString(plan.StoreName, plan.StoreID)+"：超过 "+netTicketDisplayTime(plan.TargetTime)+" 窗口仍未取到号")
		return
	}
	fireNetTicket(ctx, plan, now, today)
}

// netTicketTickOnOpen 是「一开放就取号」：轮询门店线上取号状态，翻为开放即取号。
func netTicketTickOnOpen(ctx context.Context, plan NetTicketPlan, now time.Time, today string) {
	stores, err := NewQueueLiveClient().CachedAllStores(ctx)
	if err != nil {
		return // 拿不到门店状态，下个 tick 再试
	}
	open := false
	for _, s := range stores {
		if strconv.Itoa(s.ID) == strings.TrimSpace(plan.StoreID) {
			open = queueLiveStoreOnlineOpen(s)
			break
		}
	}
	if !open {
		if plan.Status != "armed" {
			plan.Status = "armed"
			plan.LastError = ""
			if err := SaveNetTicketPlan(plan); err != nil {
				LogMessage(now, "保存排队号计划失败: "+err.Error())
			}
		}
		return
	}
	fireNetTicket(ctx, plan, now, today)
}

// fireNetTicket 是 time/on_open 两种触发共用的取号执行路径。
func fireNetTicket(ctx context.Context, plan NetTicketPlan, now time.Time, today string) {
	// 命中条件：按日期占位，确保只取一次（web/守护两个进程都安全）。
	if !reserveNetTicketFire(today) {
		return
	}
	plan.FiredDate = today
	plan.FiredAt = now.Format(time.RFC3339)
	if err := SaveNetTicketPlan(plan); err != nil {
		LogMessage(now, "保存排队号计划失败: "+err.Error())
	}

	client := currentAuthedClient()
	if client == nil {
		plan.Enabled = false
		plan.Status = "error"
		plan.LastError = "尚未获取通行证（或已过期），无法自动取号"
		if err := SaveNetTicketPlan(plan); err != nil {
			LogMessage(now, "保存排队号计划失败: "+err.Error())
		}
		sendQueueAlert(ctx, "⚠️ 定时取号失败", plan.LastError)
		return
	}
	ticket, err := client.CreateNetTicket(ctx, plan.StoreID)
	if err != nil {
		noteAuthResult(err)
		if isTicketAlreadyIssuedError(err) {
			if recovered, ok := recoverExistingNetTicket(ctx, client, &plan); ok {
				plan = recovered
				if err := SaveNetTicketPlan(plan); err != nil {
					LogMessage(now, "保存排队号计划失败: "+err.Error())
				}
				sendQueueAlert(ctx, "🎫 已恢复排队号", DefaultString(plan.StoreName, plan.StoreID)+"：号码 "+DefaultString(plan.Number, "(详见我的预约)"))
				return
			}
			markNetTicketIssuedUnknown(&plan, friendlyNetTicketError(err))
			sendQueueAlert(ctx, "⚠️ 已有排队号", DefaultString(plan.StoreName, plan.StoreID)+"："+plan.LastError)
			return
		}
		if isCredentialRefreshLikelyError(err) {
			plan.Enabled = false
			plan.Status = "error"
			plan.LastError = friendlyNetTicketError(err)
			if err := SaveNetTicketPlan(plan); err != nil {
				LogMessage(now, "保存排队号计划失败: "+err.Error())
			}
			sendQueueAlert(ctx, "⚠️ 定时取号需要更新通行证", DefaultString(plan.StoreName, plan.StoreID)+"："+plan.LastError+"。寿司郎通行证会过期或被手机端登录顶掉，请重置通行证后重新启用自动取号。")
			return
		}
		if isOfficialServerHTTPError(err) {
			plan.ServerRetryCount++
			plan.RetryDate = today
			plan.LastError = friendlyNetTicketError(err)
			// 达到重试上限：放弃当天取号，保留 FiredDate 占位避免再次触发，转 error 并通知。
			if plan.ServerRetryCount >= netTicketMaxServerRetries {
				plan.Enabled = false
				plan.Status = "error"
				plan.FiredDate = today
				if err := SaveNetTicketPlan(plan); err != nil {
					LogMessage(now, "保存排队号计划失败: "+err.Error())
				}
				sendQueueAlert(ctx, "⚠️ 定时取号失败", DefaultString(plan.StoreName, plan.StoreID)+"：连续 "+strconv.Itoa(plan.ServerRetryCount)+" 次服务端临时错误（"+plan.LastError+"），已停止重试。可稍后手动取号。")
				return
			}
			// 仍在重试上限内：清掉当天占位，下个 tick 重新抢锁重试。
			plan.Status = "retrying"
			plan.FiredDate = ""
			plan.FiredAt = ""
			if err := SaveNetTicketPlan(plan); err != nil {
				LogMessage(now, "保存排队号计划失败: "+err.Error())
			}
			clearNetTicketFire(today)
			return
		}
		plan.Enabled = false
		plan.Status = "error"
		plan.LastError = friendlyNetTicketError(err)
		if err := SaveNetTicketPlan(plan); err != nil {
			LogMessage(now, "保存排队号计划失败: "+err.Error())
		}
		sendQueueAlert(ctx, "⚠️ 定时取号失败", DefaultString(plan.StoreName, plan.StoreID)+"："+plan.LastError)
		return
	}
	markAuthHealthy()
	applyNetTicketSuccess(ctx, client, &plan, ticket)
	if err := SaveNetTicketPlan(plan); err != nil {
		LogMessage(now, "保存排队号计划失败: "+err.Error())
	}
	sendQueueAlert(ctx, "🎫 已自动取号", DefaultString(plan.StoreName, plan.StoreID)+"：号码 "+DefaultString(ticket.Number, "(详见我的预约)"))
}

// resetNetTicketPlanAfterAuthReset 在用户重置本地凭证后调用：凭证变了，旧的自动取号计划已不可信，
// 需要把它停掉并提示重新启用。已经成功的(success/issued_unknown)不动；armed/retrying 这类挂起中的
// 状态转为 error，避免拿新凭证去续跑一个针对旧凭证设计的计划。
func resetNetTicketPlanAfterAuthReset() {
	plan := LoadNetTicketPlan()
	if !plan.Enabled && plan.Status == "idle" && plan.LastError == "" {
		return
	}
	switch strings.TrimSpace(plan.Status) {
	case "success", "issued_unknown":
		return
	}
	plan.Enabled = false
	if strings.TrimSpace(plan.Status) == "" || strings.TrimSpace(plan.Status) == "armed" || strings.TrimSpace(plan.Status) == "retrying" {
		plan.Status = "error"
	}
	plan.LastError = "已重置本地通行证；寿司郎通行证会过期或被手机端登录顶掉，请重新获取通行证后再启用自动取号"
	if err := SaveNetTicketPlan(plan); err != nil {
		LogMessage(time.Now(), "保存排队号计划失败: "+err.Error())
	}
}

func markNetTicketIssuedUnknown(plan *NetTicketPlan, message string) {
	plan.Enabled = false
	plan.Status = "issued_unknown"
	plan.LastError = message
	if err := SaveNetTicketPlan(*plan); err != nil {
		LogMessage(time.Now(), "保存排队号计划失败: "+err.Error())
	}
}

// recoverExistingNetTicket 处理「取号时被告知已有号」的情况：再查一次当前排队号状态，
// 若确实有一张有效的排队号就当作成功恢复（applyNetTicketSuccess）；查不到或状态不像取号成功，
// 则降级为 issued_unknown（号可能存在但无法确认），避免重复取号。
func recoverExistingNetTicket(ctx context.Context, client *Client, plan *NetTicketPlan) (NetTicketPlan, bool) {
	ticket, err := client.GetNetTicketStatus(ctx)
	if err != nil || !netTicketLooksSuccessful(ticket) {
		markNetTicketIssuedUnknown(plan, friendlyNetTicketError(err))
		return *plan, false
	}
	applyNetTicketSuccess(ctx, client, plan, ticket)
	return *plan, true
}

func applyNetTicketSuccess(ctx context.Context, client *Client, plan *NetTicketPlan, ticket ReservationRecord) {
	plan.Enabled = false
	plan.Status = "success"
	plan.Number = ticket.Number
	plan.TicketID = ticket.TicketID
	plan.LastError = ""
	plan.ServerRetryCount = 0
	plan.RetryDate = ""
	storeName := DefaultString(plan.StoreName, plan.StoreID)
	storeAddress := ""
	if info, err := client.GetStoreInfo(ctx, plan.StoreID); err == nil {
		storeName = DefaultString(info.Name, storeName)
		storeAddress = info.Address
	}
	ticket.MonitoredStoreID = plan.StoreID
	onBookingSuccess(ticket, storeName, storeAddress, "排队取号", "取号")
}

// netTicketLooksSuccessful 判断一张预约记录是否「像一张成功的排队号」：
// 既要看起来成功，又不能是预约订座（订座记录不算排队号）。用于 recoverExistingNetTicket 的降级判断。
func netTicketLooksSuccessful(ticket ReservationRecord) bool {
	return reservationRecordLooksSuccessful(ticket) && !reservationRecordIsReservation(ticket)
}

// reserveNetTicketFire 用独占创建的锁文件占位，返回 true 表示本进程抢到了今天的取号执行权。
func reserveNetTicketFire(date string) bool {
	p := filepath.Join(AppDirPath(), "netticket_fire_"+date+".lock")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return false
	}
	_, _ = f.WriteString(strconv.Itoa(os.Getpid()))
	_ = f.Close()
	return true
}

// clearNetTicketFire 在用户重新设定计划时清掉当天占位，允许再次取号。
func clearNetTicketFire(date string) {
	_ = os.Remove(filepath.Join(AppDirPath(), "netticket_fire_"+date+".lock"))
}

// netTicketTargetToday 把「HHMM」字符串解析成今天的目标时刻。
// 左侧补零容忍简写（如 "900" -> "0900"）；小时>23 或分钟>59 视为非法。
// 目标时刻按 CST 构造，保证与 tick 的 now（CST）比较时区一致。
func netTicketTargetToday(hhmm string, now time.Time) (time.Time, bool) {
	hhmm = strings.TrimSpace(hhmm)
	for len(hhmm) < 4 {
		hhmm = "0" + hhmm
	}
	now = now.In(SushiroTimezone)
	if len(hhmm) < 4 {
		return time.Time{}, false
	}
	h, err1 := strconv.Atoi(hhmm[:2])
	m, err2 := strconv.Atoi(hhmm[2:4])
	if err1 != nil || err2 != nil || h > 23 || m > 59 {
		return time.Time{}, false
	}
	return time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, SushiroTimezone), true
}

func netTicketDisplayTime(hhmm string) string {
	for len(hhmm) < 4 {
		hhmm = "0" + hhmm
	}
	return hhmm[:2] + ":" + hhmm[2:4]
}

// currentAuthedClient 从本地配置构造一个带凭证的 API 客户端（headless 守护也能用）。
func currentAuthedClient() *Client {
	tokens, err := LoadLocalConfig()
	if err != nil {
		return nil
	}
	if tokens.ValidateForReservation() != nil {
		return nil
	}
	prefs := LoadPreferences()
	return NewClient(tokens.ToSettingsWithPrefs(prefs))
}
