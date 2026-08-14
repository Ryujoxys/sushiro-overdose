package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	netTicketRoutineFile                    = "netticket_routine.json"
	netTicketRoutineDefaultNotifyBeforeMins = 10
	// netTicketRoutineRefreshThrottle 限制「样本不足（waiting_data）」状态下重新拉历史数据推算计划的频率。
	// 不限流的话每个 20s tick 都会重算一次，而门店历史数据不会 20s 变一次，纯属浪费请求。
	netTicketRoutineRefreshThrottle = 5 * time.Minute
	// 下面是 Routine 当天计划的状态机取值，由 refreshNetTicketRoutineLocked /
	// planNetTicketRoutineReminderForTodayLocked 推进。典型流转：
	//   idle(刚改配置/未启用) -> armed(已就绪，等提醒时刻)
	//   armed -> notified(到点已发提醒，当天不再重发)
	//   armed -> missed(今天推算的提醒窗口已过，未到点就过点了，等明天)
	//   任意 -> done(今天已成功取号)
	//   任意 -> waiting_data(缺门店/就餐时间/历史样本不足，节流后重试)
	//   任意 -> needs_notify(没配通知渠道，提醒发不出去)
	// 跨天时 PlannedDate!=today 会重新走一遍推算（见 routineSkipRefresh）。
	netTicketRoutineStatusIdle        = "idle"
	netTicketRoutineStatusArmed       = "armed"
	netTicketRoutineStatusWaitingData = "waiting_data"
	netTicketRoutineStatusNeedsNotify = "needs_notify"
	netTicketRoutineStatusMissed      = "missed"
	netTicketRoutineStatusNotified    = "notified"
	netTicketRoutineStatusDone        = "done"
)

// NetTicketRoutine 是「每天想几点吃」配置：每天按历史等待倒推取号窗口，并提醒用户手动取号。
type NetTicketRoutine struct {
	Enabled             bool   `json:"enabled"`
	StoreID             string `json:"store_id"`
	StoreName           string `json:"store_name,omitempty"`
	TargetMealTime      string `json:"target_meal_time"`
	TravelMinutes       int    `json:"travel_minutes"`
	NotifyBeforeMinutes int    `json:"notify_before_minutes"`
	// SafetyMinutes 是 v2.22 的遗留字段，当时用作「自动取号安全提前量」。
	// 现 Routine 只提醒、不自动取号，normalizeNetTicketRoutine 会把它迁移成 NotifyBeforeMinutes 后清零。
	SafetyMinutes int    `json:"safety_minutes,omitempty"`
	Status        string `json:"status"`
	// PlannedDate 是当前计划所属的日期 YYYY-MM-DD，跨天后与今天不一致即视为「需重新推算」。
	PlannedDate          string `json:"planned_date,omitempty"`
	PlannedPickupTime    string `json:"planned_pickup_time,omitempty"`
	PlannedPickupEndTime string `json:"planned_pickup_end_time,omitempty"`
	// ReminderTime 是「提前提醒」时刻，等于取号窗口起点减 NotifyBeforeMinutes。
	ReminderTime         string          `json:"reminder_time,omitempty"`
	RecommendPickupRange *QueueTimeRange `json:"recommend_pickup_range,omitempty"`
	WaitMinutesRange     *QueueWaitRange `json:"wait_minutes_range,omitempty"`
	Risk                 string          `json:"risk,omitempty"`
	Basis                string          `json:"basis,omitempty"`
	// LastPlannedAt 记录上次推算计划的 RFC3339 时刻，仅用于 waiting_data 的节流判断（见 routineSkipRefresh）。
	LastPlannedAt string `json:"last_planned_at,omitempty"`
	// LastReminderDate 记录当天是否已发过提醒，防止同一日重复推送。
	LastReminderDate string `json:"last_reminder_date,omitempty"`
	LastError        string `json:"last_error,omitempty"`
}

type NetTicketRoutineResponse struct {
	Routine NetTicketRoutine `json:"routine"`
	Plan    NetTicketPlan    `json:"plan"`
}

func netTicketRoutinePath() string { return filepath.Join(AppDirPath(), netTicketRoutineFile) }

func LoadNetTicketRoutine() NetTicketRoutine {
	data, err := os.ReadFile(netTicketRoutinePath())
	if err != nil {
		return NetTicketRoutine{Status: netTicketRoutineStatusIdle, NotifyBeforeMinutes: netTicketRoutineDefaultNotifyBeforeMins}
	}
	var r NetTicketRoutine
	if json.Unmarshal(data, &r) != nil {
		return NetTicketRoutine{Status: netTicketRoutineStatusIdle, NotifyBeforeMinutes: netTicketRoutineDefaultNotifyBeforeMins}
	}
	return normalizeNetTicketRoutine(r)
}

func SaveNetTicketRoutine(r NetTicketRoutine) error {
	os.MkdirAll(AppDirPath(), 0o755)
	r = normalizeNetTicketRoutine(r)
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(netTicketRoutinePath(), data, 0o600)
}

// normalizeNetTicketRoutine 在落盘/读取时规整 Routine 配置：把 v2.22 遗留的
// SafetyMinutes 迁移到 NotifyBeforeMinutes 并清零，钳制负值，并给「全新空配置」
// 补一个默认提醒提前量（只有当前配置完全空白时才补，避免覆盖用户显式设的 0）。
func normalizeNetTicketRoutine(r NetTicketRoutine) NetTicketRoutine {
	r.StoreID = strings.TrimSpace(r.StoreID)
	r.StoreName = strings.TrimSpace(r.StoreName)
	r.TargetMealTime = compactRoutineHHMM(r.TargetMealTime)
	if r.TravelMinutes < 0 {
		r.TravelMinutes = 0
	}
	// 遗留字段迁移：v2.22 的 SafetyMinutes 语义改为 NotifyBeforeMinutes。
	if r.NotifyBeforeMinutes == 0 && r.SafetyMinutes > 0 {
		r.NotifyBeforeMinutes = r.SafetyMinutes
	}
	if r.NotifyBeforeMinutes < 0 {
		r.NotifyBeforeMinutes = 0
	}
	// 仅在配置完全空白（未启用、无门店、无就餐时间）时补默认提前量，
	// 否则用户主动设 0 提前量会被这里误覆盖。
	if r.NotifyBeforeMinutes == 0 && !r.Enabled && r.TargetMealTime == "" && r.StoreID == "" {
		r.NotifyBeforeMinutes = netTicketRoutineDefaultNotifyBeforeMins
	}
	r.SafetyMinutes = 0
	if strings.TrimSpace(r.Status) == "" {
		r.Status = netTicketRoutineStatusIdle
	}
	return r
}

// refreshNetTicketRoutineLocked 是 Routine 状态机的入口，由 netTicketTick 每个 tick（约 20s）持 netTicketMu 调一次。
// 它负责：迁移旧的「自动取号计划」、校验前置数据（门店/就餐时间）、判断今天是否已取到号，
// 再决定是跳过本次刷新（当天已终态/节流）还是进入 planNetTicketRoutineReminderForTodayLocked 推算提醒。
func refreshNetTicketRoutineLocked(now time.Time) NetTicketRoutineResponse {
	if now.IsZero() {
		now = time.Now()
	}
	routine := LoadNetTicketRoutine()
	// Routine 现在只做提醒，不再自动取号；把历史遗留的「routine 自动取号计划」退役掉。
	plan := retireRoutineGeneratedNetTicketPlan(LoadNetTicketPlan())
	if !routine.Enabled {
		return NetTicketRoutineResponse{Routine: routine, Plan: plan}
	}
	if strings.TrimSpace(routine.StoreID) == "" {
		routine.Status = netTicketRoutineStatusWaitingData
		routine.LastError = "Routine 还没有选择门店。"
		routine.LastPlannedAt = now.Format(time.RFC3339)
		_ = SaveNetTicketRoutine(routine)
		return NetTicketRoutineResponse{Routine: routine, Plan: plan}
	}
	if _, ok := parseHHMM(routine.TargetMealTime, now); !ok {
		routine.Status = netTicketRoutineStatusWaitingData
		routine.LastError = "Routine 还没有有效的目标就餐时间。"
		routine.LastPlannedAt = now.Format(time.RFC3339)
		_ = SaveNetTicketRoutine(routine)
		return NetTicketRoutineResponse{Routine: routine, Plan: plan}
	}
	today := now.Format("2006-01-02")
	// 今天已经在排队号计划里取到号：Routine 当天任务完成，直接置 done。
	if netTicketPlanSuccessfulOn(plan, now) {
		routine.Status = netTicketRoutineStatusDone
		routine.PlannedDate = today
		routine.LastError = "今天已经取到排队号。"
		routine.LastPlannedAt = now.Format(time.RFC3339)
		_ = SaveNetTicketRoutine(routine)
		return NetTicketRoutineResponse{Routine: routine, Plan: plan}
	}
	// 当天已到终态（missed/notified/done）或 waiting_data 仍在节流窗口内：本 tick 跳过重算。
	if routineSkipRefresh(routine, now, today) {
		return NetTicketRoutineResponse{Routine: routine, Plan: plan}
	}
	// 快路径：今天的取号窗口已经推算过且仍未终结（armed/needs_notify）——窗口结论
	// 不随 tick 变化，不需要每 20s 重跑 buildQueueMealPlan（它要读全量趋势文件、
	// 可能打远端基准，秒级耗时），既浪费请求又拉长 netTicketMu 持锁时间，
	// 把用户手动取号/改计划卡在后面。armed 只需判断是否到提醒时刻。
	if routine.PlannedDate == today {
		if advanced, ok := advanceRoutineArmedToday(routine, now, today); ok {
			return NetTicketRoutineResponse{Routine: advanced, Plan: plan}
		}
		// needs_notify 且通知渠道仍未配置：结论不变，跳过重算。
		// 配置渠道是用户操作，配好后下一个 tick 会重新推算。
		if routine.Status == netTicketRoutineStatusNeedsNotify && !routineNotifyConfigured() {
			return NetTicketRoutineResponse{Routine: routine, Plan: plan}
		}
	}
	routine, plan = planNetTicketRoutineReminderForTodayLocked(routine, plan, now)
	return NetTicketRoutineResponse{Routine: routine, Plan: plan}
}

// advanceRoutineArmedToday 是 armed 状态的 tick 快路径：今天的取号窗口已经落盘在
// routine 字段里（PlannedPickupTime / PlannedPickupEndTime / NotifyBeforeMinutes），
// 只用这些已存的结论推进状态机，不重跑 buildQueueMealPlan。
// 返回 ok=false 表示快路径不适用（状态不是 armed，或存的窗口解析不了），调用方走慢路径重算。
func advanceRoutineArmedToday(routine NetTicketRoutine, now time.Time, today string) (NetTicketRoutine, bool) {
	if routine.Status != netTicketRoutineStatusArmed || strings.TrimSpace(routine.PlannedPickupTime) == "" {
		return routine, false
	}
	start, ok := parseHHMM(routine.PlannedPickupTime, now)
	if !ok {
		return routine, false
	}
	end, ok := parseHHMM(DefaultString(routine.PlannedPickupEndTime, routine.PlannedPickupTime), now)
	if !ok || end.Before(start) {
		end = start
	}
	// 判定顺序与 planNetTicketRoutineReminderForTodayLocked 保持一致：
	// 过窗口末端 → missed；到提醒时刻 → 发提醒并置 notified；否则保持 armed。
	if now.After(end) {
		routine.Status = netTicketRoutineStatusMissed
		routine.LastError = "今天推算出的取号提醒窗口已过，Routine 明天再提醒。"
		routine.LastPlannedAt = now.Format(time.RFC3339)
		_ = SaveNetTicketRoutine(routine)
		return routine, true
	}
	reminderAt := start.Add(-time.Duration(routine.NotifyBeforeMinutes) * time.Minute)
	if !now.Before(reminderAt) {
		sendRoutineTakeTicketReminder(context.Background(), routine)
		routine.LastReminderDate = today
		routine.Status = netTicketRoutineStatusNotified
		routine.LastError = ""
		routine.LastPlannedAt = now.Format(time.RFC3339)
		_ = SaveNetTicketRoutine(routine)
		return routine, true
	}
	// 未到提醒时刻：状态无变化，不落盘（armed 每 tick 都进来，重复写盘没有意义）。
	return routine, true
}

// planNetTicketRoutineReminderForTodayLocked 推算今天的取号提醒计划，并按当前时刻决定状态机的下一个状态。
// 顺序很重要，每个分支都是当天计划的终态或推进点：
//
//	waiting_data  —— 样本不足 / 取号窗口算不出来，等待节流后重试。
//	needs_notify  —— 没配通知渠道，提醒发不出去，要求用户先配。
//	notified      —— 今天已经发过提醒（LastReminderDate==today）或刚刚发完。
//	missed        —— 当前时间已晚于取号窗口终点，今天来不及了，等明天。
//	armed         —— 还没到提醒时刻，挂起等待。
//
// 注意：reminderAt 用「窗口起点 - NotifyBeforeMinutes」算，所以已过 reminderAt 即触发提醒；
// 但只要还没过窗口 end，即使错过了 reminderAt 也会补发一次（落到 notified 分支）。
func planNetTicketRoutineReminderForTodayLocked(routine NetTicketRoutine, existing NetTicketPlan, now time.Time) (NetTicketRoutine, NetTicketPlan) {
	today := now.Format("2006-01-02")
	routine.PlannedDate = today
	routine.LastPlannedAt = now.Format(time.RFC3339)
	mealPlan := buildQueueMealPlan(context.Background(), routine.StoreID, routine.TargetMealTime, routine.TravelMinutes, now, false)
	routine.RecommendPickupRange = mealPlan.RecommendPickupRange
	routine.WaitMinutesRange = mealPlan.WaitMinutesRange
	routine.Risk = mealPlan.Risk
	routine.Basis = mealPlan.Basis

	startRaw, endRaw := routinePickupWindowFromMealPlan(mealPlan)
	if startRaw == "" {
		routine.Status = netTicketRoutineStatusWaitingData
		routine.PlannedPickupTime = ""
		routine.PlannedPickupEndTime = ""
		routine.ReminderTime = ""
		routine.LastError = DefaultString(mealPlan.Message, "这家店历史样本不足，Routine 暂不提醒取号。")
		_ = SaveNetTicketRoutine(routine)
		return routine, existing
	}
	start, ok := parseHHMM(startRaw, now)
	if !ok {
		routine.Status = netTicketRoutineStatusWaitingData
		routine.PlannedPickupTime = ""
		routine.PlannedPickupEndTime = ""
		routine.ReminderTime = ""
		routine.LastError = "推算出的取号时间不可用，Routine 暂不提醒取号。"
		_ = SaveNetTicketRoutine(routine)
		return routine, existing
	}
	end, ok := parseHHMM(DefaultString(endRaw, startRaw), now)
	if !ok || end.Before(start) {
		end = start
	}
	routine.PlannedPickupTime = start.Format("15:04")
	routine.PlannedPickupEndTime = end.Format("15:04")
	// 提醒时刻 = 取号窗口起点往前推 NotifyBeforeMinutes。提醒只发一次，由 LastReminderDate 去重。
	reminderAt := start.Add(-time.Duration(routine.NotifyBeforeMinutes) * time.Minute)
	routine.ReminderTime = reminderAt.Format("15:04")

	if !routineNotifyConfigured() {
		routine.Status = netTicketRoutineStatusNeedsNotify
		routine.LastError = "启用 Routine 前必须先配置通知渠道；否则无法提醒你取号。"
		_ = SaveNetTicketRoutine(routine)
		return routine, existing
	}
	// 今天已经发过提醒：幂等，不再重发。
	if routine.LastReminderDate == today {
		routine.Status = netTicketRoutineStatusNotified
		routine.LastError = ""
		_ = SaveNetTicketRoutine(routine)
		return routine, existing
	}
	// 已过窗口终点：今天来不及了（连补发都没意义），置 missed 等明天。
	if now.After(end) {
		routine.Status = netTicketRoutineStatusMissed
		routine.LastError = "今天推算出的取号提醒窗口已过，Routine 明天再提醒。"
		_ = SaveNetTicketRoutine(routine)
		return routine, existing
	}
	// 已到或过提醒时刻（但还没过窗口 end）：现在就发提醒，并记下今天已发。
	if !now.Before(reminderAt) {
		sendRoutineTakeTicketReminder(context.Background(), routine)
		routine.LastReminderDate = today
		routine.Status = netTicketRoutineStatusNotified
		routine.LastError = ""
		_ = SaveNetTicketRoutine(routine)
		return routine, existing
	}

	// 还没到提醒时刻：挂起等待，下一个 tick 会重新进来判断。
	routine.Status = netTicketRoutineStatusArmed
	routine.LastError = ""
	_ = SaveNetTicketRoutine(routine)
	return routine, existing
}

func routinePickupWindowFromMealPlan(plan QueueMealPlan) (string, string) {
	if plan.RecommendPickupRange != nil && strings.TrimSpace(plan.RecommendPickupRange.Early) != "" {
		return strings.TrimSpace(plan.RecommendPickupRange.Early), strings.TrimSpace(plan.RecommendPickupRange.Late)
	}
	if strings.TrimSpace(plan.StablePickup) != "" {
		return strings.TrimSpace(plan.StablePickup), strings.TrimSpace(plan.LatestPickup)
	}
	return strings.TrimSpace(plan.LatestPickup), strings.TrimSpace(plan.LatestPickup)
}

func sendRoutineTakeTicketReminder(ctx context.Context, routine NetTicketRoutine) {
	store := DefaultString(routine.StoreName, routine.StoreID)
	window := routine.PlannedPickupTime
	if routine.PlannedPickupEndTime != "" && routine.PlannedPickupEndTime != routine.PlannedPickupTime {
		window += "-" + routine.PlannedPickupEndTime
	}
	wait := ""
	if routine.WaitMinutesRange != nil {
		wait = "，预计等待 " + strconv.Itoa(routine.WaitMinutesRange.Low) + "-" + strconv.Itoa(routine.WaitMinutesRange.High) + " 分钟"
	}
	sendQueueAlert(ctx,
		"🍣 该取号了",
		store+"：建议在 "+window+" 之间取号，目标 "+displayTrendTime(routine.TargetMealTime)+" 吃"+wait+"。这只是提醒，不会自动向寿司郎取号。")
}

func routineNotifyConfigured() bool {
	return len(configuredNotificationChannels()) > 0
}

// netTicketPlanSuccessfulOn 判断排队号计划在今天是否已成功取到号。
// 成功的判定：今天触发过（FiredDate/ FiredAt 命中今天），且状态为 success/issued_unknown，
// 或即便状态字段缺失，只要有号码/TicketID 也算（兼容历史数据）。
func netTicketPlanSuccessfulOn(plan NetTicketPlan, now time.Time) bool {
	if !netTicketPlanFiredOn(plan, now) {
		return false
	}
	switch strings.TrimSpace(plan.Status) {
	case "success", "issued_unknown":
		return true
	default:
		return strings.TrimSpace(plan.Number) != "" || plan.TicketID != 0
	}
}

// retireRoutineGeneratedNetTicketPlan 把历史遗留的「Routine 自动取号计划」退役。
// 历史 Routine 会写一个 source==routine 的 NetTicketPlan 自动取号；现在 Routine 只提醒不取号，
// 所以遇到非终态的这种计划就置 idle、关掉自动取号，避免旧计划还在后台自动取号与提醒行为冲突。
func retireRoutineGeneratedNetTicketPlan(plan NetTicketPlan) NetTicketPlan {
	if plan.Source != netTicketPlanSourceRoutine || netTicketPlanTerminal(plan.Status) {
		return plan
	}
	plan.Enabled = false
	plan.Source = ""
	plan.TargetMealTime = ""
	plan.RoutinePlannedDate = ""
	plan.Status = "idle"
	plan.LastError = "Routine 已改为提醒取号，不再自动提交取号。"
	if err := SaveNetTicketPlan(plan); err != nil {
		LogMessage(time.Now(), "保存排队号计划失败: "+err.Error())
	}
	return plan
}

// routineSkipRefresh 判断本 tick 是否可以跳过重新推算。
// 跳过的两种情况：
//  1. 今天已经推算过、且已进入终态（missed/notified/done）——当天结论不会变，重算无意义。
//  2. 处于 waiting_data（样本不足等）且距离上次推算不足 netTicketRoutineRefreshThrottle——
//     限流避免高频拉历史数据。
//
// 注意：armed 状态永远不跳过（需要持续检查是否到了提醒时刻）。
func routineSkipRefresh(r NetTicketRoutine, now time.Time, today string) bool {
	if r.PlannedDate == today {
		switch r.Status {
		case netTicketRoutineStatusMissed, netTicketRoutineStatusNotified, netTicketRoutineStatusDone:
			return true
		}
	}
	if r.Status != netTicketRoutineStatusWaitingData {
		return false
	}
	last, ok := parseRFC3339Local(r.LastPlannedAt)
	return ok && now.Sub(last) < netTicketRoutineRefreshThrottle
}

func saveNetTicketRoutineConfigLocked(routine NetTicketRoutine, now time.Time) NetTicketRoutineResponse {
	routine = normalizeNetTicketRoutine(routine)
	routine.Status = netTicketRoutineStatusIdle
	routine.PlannedDate = ""
	routine.PlannedPickupTime = ""
	routine.PlannedPickupEndTime = ""
	routine.ReminderTime = ""
	routine.RecommendPickupRange = nil
	routine.WaitMinutesRange = nil
	routine.Risk = ""
	routine.Basis = ""
	routine.LastError = ""
	routine.LastPlannedAt = ""
	plan := retireRoutineGeneratedNetTicketPlan(LoadNetTicketPlan())
	if !routine.Enabled {
		routine.Status = netTicketRoutineStatusIdle
		_ = SaveNetTicketRoutine(routine)
		return NetTicketRoutineResponse{Routine: routine, Plan: plan}
	}
	_ = SaveNetTicketRoutine(routine)
	return refreshNetTicketRoutineLocked(now)
}

// compactRoutineHHMM 把就餐时间归一成「HHMM」字符串：去空格、去冒号；
// 若长度为 6（HMMSS 形式）且秒位为 00，则截掉秒位变回 HHMM。
func compactRoutineHHMM(raw string) string {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, ":", ""))
	if len(raw) == 6 && strings.HasSuffix(raw, "00") {
		raw = raw[:4]
	}
	return raw
}
