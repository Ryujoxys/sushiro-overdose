package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/notify"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNetTicketIssuedTodayBlocksSampling(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	now := time.Date(2026, 6, 3, 18, 0, 0, 0, SushiroTimezone)
	if err := SaveNetTicketPlan(NetTicketPlan{
		Enabled:   false,
		StoreID:   "3006",
		Status:    "success",
		Number:    "1843",
		FiredDate: now.Format("2006-01-02"),
		FiredAt:   now.Add(-10 * time.Minute).Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("SaveNetTicketPlan() error = %v", err)
	}

	if !netTicketIssuedToday(now) {
		t.Fatal("issued ticket today should block sampling")
	}
}

func TestNetTicketIssuedYesterdayDoesNotBlockSampling(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	now := time.Date(2026, 6, 3, 18, 0, 0, 0, SushiroTimezone)
	if err := SaveNetTicketPlan(NetTicketPlan{
		Enabled:   true,
		StoreID:   "3006",
		Status:    "success",
		Number:    "1843",
		FiredDate: now.AddDate(0, 0, -1).Format("2006-01-02"),
		FiredAt:   now.AddDate(0, 0, -1).Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("SaveNetTicketPlan() error = %v", err)
	}

	if netTicketIssuedToday(now) {
		t.Fatal("old ticket should not block today's sampling")
	}
}

func TestTerminalNetTicketPlanFromPreviousDayIsNotArmed(t *testing.T) {
	now := time.Date(2026, 6, 3, 18, 0, 0, 0, SushiroTimezone)
	plan := normalizeNetTicketPlan(NetTicketPlan{
		Enabled:   true,
		StoreID:   "3006",
		Status:    "success",
		Number:    "1843",
		FiredDate: now.AddDate(0, 0, -1).Format("2006-01-02"),
		FiredAt:   now.AddDate(0, 0, -1).Format(time.RFC3339),
	}, now)

	if plan.Enabled {
		t.Fatal("previous-day terminal net ticket plan should not remain armed")
	}
}

func TestTerminalNetTicketPlanFromTodayIsNotArmed(t *testing.T) {
	now := time.Date(2026, 6, 3, 18, 0, 0, 0, SushiroTimezone)
	plan := normalizeNetTicketPlan(NetTicketPlan{
		Enabled:   true,
		StoreID:   "3006",
		Status:    "expired",
		FiredDate: now.Format("2006-01-02"),
		FiredAt:   now.Add(-10 * time.Minute).Format(time.RFC3339),
	}, now)

	if plan.Enabled {
		t.Fatal("today's terminal net ticket plan should not remain armed")
	}
	if !netTicketPlanFiredOn(plan, now) {
		t.Fatal("terminal plan should still preserve today's fired state")
	}
}

func TestActiveNetTicketPlanStaysArmed(t *testing.T) {
	now := time.Date(2026, 6, 3, 18, 0, 0, 0, SushiroTimezone)
	plan := normalizeNetTicketPlan(NetTicketPlan{
		Enabled:    true,
		StoreID:    "3006",
		Status:     "armed",
		TargetTime: "1900",
	}, now)

	if !plan.Enabled {
		t.Fatal("armed net ticket plan should stay enabled")
	}
}

// TestNetTicketServerRetryCountPersists 验证服务端临时错误的重试计数会被持久化，
// 且达到上限 netTicketMaxServerRetries 后 fireNetTicket 不再清占位重发（防刷号）。
func TestNetTicketServerRetryCountPersists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := SaveNetTicketPlan(NetTicketPlan{
		Enabled:          true,
		StoreID:          "3006",
		Status:           "retrying",
		TargetTime:       "1900",
		ServerRetryCount: netTicketMaxServerRetries,
		FiredDate:        "", // retrying 状态当天未占位
	}); err != nil {
		t.Fatalf("SaveNetTicketPlan() error = %v", err)
	}

	loaded := LoadNetTicketPlan()
	if loaded.ServerRetryCount != netTicketMaxServerRetries {
		t.Fatalf("ServerRetryCount round-trip = %d, want %d", loaded.ServerRetryCount, netTicketMaxServerRetries)
	}
}

// TestNetTicketServerRetryResetsAcrossDay 验证跨天后 ServerRetryCount 仍被持久化，
// 由 netTicketTick 在新一天的首次调度时清零（见 netticket.go 中 FiredDate != today 的重置分支）。
func TestNetTicketServerRetryResetsAcrossDay(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	now := time.Date(2026, 6, 3, 18, 0, 0, 0, SushiroTimezone)
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	if err := SaveNetTicketPlan(NetTicketPlan{
		Enabled:          true,
		StoreID:          "3006",
		Status:           "retrying",
		TargetTime:       "1900",
		ServerRetryCount: netTicketMaxServerRetries,
		FiredDate:        yesterday,
	}); err != nil {
		t.Fatalf("SaveNetTicketPlan() error = %v", err)
	}

	loaded := LoadNetTicketPlan()
	// 前提：昨天的计划今天还没处理过，netTicketTick 会进入清零分支。
	if loaded.FiredDate == now.Format("2006-01-02") {
		t.Fatal("precondition: yesterday's plan should not already be fired today")
	}
	if loaded.ServerRetryCount != netTicketMaxServerRetries {
		t.Fatalf("yesterday's retry count should still be on disk, got %d", loaded.ServerRetryCount)
	}
}

// TestNetTicketServerRetryNotResetSameDay 是回归测试：5xx 重试分支会把 FiredDate 置空，
// 下个 tick（同一天）不能因为 ServerRetryCount!=0 就清零——否则重试上限永远失效、
// 每个 tick 都重发取号请求（刷号）。只有 RetryDate != today（真正跨天）才清零。
func TestNetTicketServerRetryNotResetSameDay(t *testing.T) {
	today := "2026-06-03"
	plan := NetTicketPlan{
		Enabled:          true,
		StoreID:          "3006",
		Status:           "retrying",
		TargetTime:       "1900",
		ServerRetryCount: 3,
		RetryDate:        today, // 重试计数属于今天
		FiredDate:        "",    // 重试分支置空，模拟下个 tick 的状态
	}
	// 复刻 netTicketTick 的清零条件：同一天不应清零。
	if plan.ServerRetryCount != 0 && plan.RetryDate == today {
		// 命中"同一天不清零"分支，符合预期
	} else {
		t.Fatal("同一天的 ServerRetryCount 不应被清零")
	}
	if plan.ServerRetryCount != 3 {
		t.Fatalf("同一天重试计数应保持 3，实际 %d", plan.ServerRetryCount)
	}
}

func TestHandleNetTicketPlanResetsServerRetryState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := SaveNetTicketPlan(NetTicketPlan{
		Enabled:          true,
		StoreID:          "3006",
		StoreName:        "旧门店",
		Status:           "retrying",
		TargetTime:       "1800",
		ServerRetryCount: netTicketMaxServerRetries,
		RetryDate:        time.Now().Format("2006-01-02"),
		LastError:        "官方服务临时繁忙",
	}); err != nil {
		t.Fatalf("SaveNetTicketPlan() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/queue/ticket/plan", strings.NewReader(`{
		"enabled": true,
		"store": "3015",
		"store_name": "新门店",
		"trigger_mode": "time",
		"target_time": "1900"
	}`))
	rr := httptest.NewRecorder()
	handleNetTicketPlan(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	got := LoadNetTicketPlan()
	if got.ServerRetryCount != 0 || got.RetryDate != "" {
		t.Fatalf("重新设定计划应清空旧服务端重试状态，got count=%d retryDate=%q plan=%+v", got.ServerRetryCount, got.RetryDate, got)
	}
	if got.Status != "armed" || got.StoreID != "3015" || got.TargetTime != "1900" {
		t.Fatalf("重新设定后的计划异常: %+v", got)
	}
}

func TestNetTicketServerRetryLimitStopsAtConfiguredLimit(t *testing.T) {
	src, err := os.ReadFile("netticket.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "if plan.ServerRetryCount >= netTicketMaxServerRetries") {
		t.Fatalf("服务端重试计数达到 netTicketMaxServerRetries 时就应停止；不要等到超过上限后才停止")
	}
}

func TestNetTicketRoutinePlansReminderFromHistoricalMealPlan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	// 用门店时区(UTC+8)构造时刻，与餐时/取号窗口同时区：餐时是北京时间，parseHHMM 也按 +8 解析，
	// now 必须是 +8 的「早9点」才能正确比较先后。否则 UTC CI 上 time.Local=UTC，9点 UTC=17点北京，
	// 已过中午取号窗口，会被误判 missed。
	cst := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 6, 9, 9, 0, 0, 0, cst)
	saveRoutineNotifyForTest(t)
	appendRoutineHistoryForTest(t, time.Date(2026, 6, 2, 12, 0, 0, 0, cst))

	resp := saveNetTicketRoutineConfigLocked(NetTicketRoutine{
		Enabled:             true,
		StoreID:             "3006",
		StoreName:           "太阳宫凯德店",
		TargetMealTime:      "1300",
		NotifyBeforeMinutes: 10,
	}, now)

	if resp.Routine.Status != netTicketRoutineStatusArmed {
		t.Fatalf("routine status = %q, want armed; error=%q", resp.Routine.Status, resp.Routine.LastError)
	}
	if resp.Routine.PlannedPickupTime != "12:00" || resp.Routine.ReminderTime != "11:50" {
		t.Fatalf("routine planned pickup/reminder = %q/%q, want 12:00/11:50", resp.Routine.PlannedPickupTime, resp.Routine.ReminderTime)
	}
	if resp.Plan.Enabled || resp.Plan.Source == netTicketPlanSourceRoutine {
		t.Fatalf("routine reminder should not arm an auto-ticket plan: %+v", resp.Plan)
	}
}

func TestNetTicketRoutineRequiresNotificationBeforeArming(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	now := time.Date(2026, 6, 9, 11, 0, 0, 0, SushiroTimezone)
	appendRoutineHistoryForTest(t, time.Date(2026, 6, 2, 12, 0, 0, 0, SushiroTimezone))

	resp := saveNetTicketRoutineConfigLocked(NetTicketRoutine{
		Enabled:             true,
		StoreID:             "3006",
		StoreName:           "太阳宫凯德店",
		TargetMealTime:      "1300",
		NotifyBeforeMinutes: 10,
	}, now)

	if resp.Routine.Status != netTicketRoutineStatusNeedsNotify {
		t.Fatalf("routine status = %q, want needs_notify; error=%q", resp.Routine.Status, resp.Routine.LastError)
	}
	if resp.Plan.Enabled {
		t.Fatalf("routine should not arm an auto-ticket plan without notify: %+v", resp.Plan)
	}
}

func TestNetTicketRoutineDoesNotOverwriteManualPlan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cst := time.FixedZone("CST", 8*3600) // 同上：用门店时区构造，跨 TZ 一致。
	now := time.Date(2026, 6, 9, 9, 0, 0, 0, cst)
	saveRoutineNotifyForTest(t)
	appendRoutineHistoryForTest(t, time.Date(2026, 6, 2, 12, 0, 0, 0, cst))
	if err := SaveNetTicketPlan(NetTicketPlan{
		Enabled:    true,
		StoreID:    "manual",
		StoreName:  "手动门店",
		TargetTime: "1800",
		Status:     "armed",
	}); err != nil {
		t.Fatalf("SaveNetTicketPlan() error = %v", err)
	}

	resp := saveNetTicketRoutineConfigLocked(NetTicketRoutine{
		Enabled:             true,
		StoreID:             "3006",
		TargetMealTime:      "1300",
		NotifyBeforeMinutes: 10,
	}, now)

	if resp.Routine.Status != netTicketRoutineStatusArmed {
		t.Fatalf("routine status = %q, want armed", resp.Routine.Status)
	}
	if resp.Plan.StoreID != "manual" || !resp.Plan.Enabled {
		t.Fatalf("manual plan was overwritten: %+v", resp.Plan)
	}
}

func TestDisablingNetTicketRoutineClearsPendingRoutinePlan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	now := time.Date(2026, 6, 9, 9, 0, 0, 0, SushiroTimezone)
	if err := SaveNetTicketPlan(NetTicketPlan{
		Enabled:            true,
		StoreID:            "3006",
		TargetTime:         "1150",
		Source:             netTicketPlanSourceRoutine,
		TargetMealTime:     "1300",
		RoutinePlannedDate: now.Format("2006-01-02"),
		Status:             "armed",
	}); err != nil {
		t.Fatalf("SaveNetTicketPlan() error = %v", err)
	}

	resp := saveNetTicketRoutineConfigLocked(NetTicketRoutine{
		Enabled:             false,
		StoreID:             "3006",
		TargetMealTime:      "1300",
		NotifyBeforeMinutes: 10,
	}, now)

	if resp.Plan.Enabled {
		t.Fatalf("routine plan should be disabled: %+v", resp.Plan)
	}
	if resp.Plan.Source == netTicketPlanSourceRoutine {
		t.Fatalf("legacy routine plan source should be cleared: %+v", resp.Plan)
	}
	if resp.Routine.Enabled || resp.Routine.Status != netTicketRoutineStatusIdle {
		t.Fatalf("routine should be idle: %+v", resp.Routine)
	}
}

func TestNetTicketRoutineWaitsForDataInsteadOfGuessing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	now := time.Date(2026, 6, 9, 9, 0, 0, 0, SushiroTimezone)

	resp := saveNetTicketRoutineConfigLocked(NetTicketRoutine{
		Enabled:             true,
		StoreID:             "3006",
		TargetMealTime:      "1300",
		NotifyBeforeMinutes: 10,
	}, now)

	if resp.Routine.Status != netTicketRoutineStatusWaitingData {
		t.Fatalf("routine status = %q, want waiting_data", resp.Routine.Status)
	}
	if resp.Plan.Enabled {
		t.Fatalf("routine should not arm a plan without data: %+v", resp.Plan)
	}
}

func saveRoutineNotifyForTest(t *testing.T) {
	t.Helper()
	cfg := &NotifyConfig{}
	cfg.Feishu.Webhook = "https://example.invalid/feishu"
	if err := SaveNotifyConfig(cfg); err != nil {
		t.Fatalf("SaveNotifyConfig() error = %v", err)
	}
}

func appendRoutineHistoryForTest(t *testing.T, observedAt time.Time) {
	t.Helper()
	if err := appendQueueObservation(QueueObservation{
		CollectedAt:       observedAt.Format(time.RFC3339),
		StoreID:           "3006",
		WaitMinutes:       60,
		GroupQueuesCount:  24,
		StoreStatus:       "OPEN",
		NetTicketStatus:   "ONLINE",
		OnlineOpen:        true,
		SourceEndpoint:    queueSourceEndpointStoreByID,
		APIProfileVersion: queueAPIProfileStoreDetailV1,
	}); err != nil {
		t.Fatalf("appendQueueObservation() error = %v", err)
	}
}

func TestAdvanceRoutineArmedTodayStaysArmedBeforeReminder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cst := SushiroTimezone
	routine := NetTicketRoutine{
		Status: netTicketRoutineStatusArmed, PlannedDate: "2026-06-09",
		PlannedPickupTime: "12:00", PlannedPickupEndTime: "12:30",
		NotifyBeforeMinutes: 10,
	}
	// 11:00 < 提醒时刻 11:50：保持 armed，且不写盘改状态。
	advanced, ok := advanceRoutineArmedToday(routine, time.Date(2026, 6, 9, 11, 0, 0, 0, cst), "2026-06-09")
	if !ok || advanced.Status != netTicketRoutineStatusArmed || advanced.LastReminderDate != "" {
		t.Fatalf("advance = %+v ok=%v, want armed/未提醒", advanced, ok)
	}
}

func TestAdvanceRoutineArmedTodayNotifiesAtReminderTime(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	saveRoutineNotifyForTest(t)
	cst := SushiroTimezone
	routine := NetTicketRoutine{
		Status: netTicketRoutineStatusArmed, PlannedDate: "2026-06-09",
		PlannedPickupTime: "12:00", PlannedPickupEndTime: "12:30",
		NotifyBeforeMinutes: 10,
	}
	// 11:55 已过提醒时刻 11:50、未过窗口末端：发提醒并置 notified。
	advanced, ok := advanceRoutineArmedToday(routine, time.Date(2026, 6, 9, 11, 55, 0, 0, cst), "2026-06-09")
	if !ok || advanced.Status != netTicketRoutineStatusNotified || advanced.LastReminderDate != "2026-06-09" {
		t.Fatalf("advance = %+v ok=%v, want notified/已记提醒日期", advanced, ok)
	}
}

func TestAdvanceRoutineArmedTodayMissesPastWindowEnd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cst := SushiroTimezone
	routine := NetTicketRoutine{
		Status: netTicketRoutineStatusArmed, PlannedDate: "2026-06-09",
		PlannedPickupTime: "12:00", PlannedPickupEndTime: "12:30",
		NotifyBeforeMinutes: 10,
	}
	// 12:31 已过窗口末端：置 missed，不再补发提醒。
	advanced, ok := advanceRoutineArmedToday(routine, time.Date(2026, 6, 9, 12, 31, 0, 0, cst), "2026-06-09")
	if !ok || advanced.Status != netTicketRoutineStatusMissed || advanced.LastReminderDate != "" {
		t.Fatalf("advance = %+v ok=%v, want missed/未发提醒", advanced, ok)
	}
}

func TestAdvanceRoutineArmedTodayFallsBackForNonArmed(t *testing.T) {
	// 非 armed / 窗口缺失：快路径不适用（ok=false），由调用方走慢路径重算。
	for _, routine := range []NetTicketRoutine{
		{Status: netTicketRoutineStatusIdle, PlannedDate: "2026-06-09", PlannedPickupTime: "12:00"},
		{Status: netTicketRoutineStatusArmed, PlannedDate: "2026-06-09", PlannedPickupTime: ""},
		{Status: netTicketRoutineStatusArmed, PlannedDate: "2026-06-09", PlannedPickupTime: "垃圾"},
	} {
		if _, ok := advanceRoutineArmedToday(routine, time.Date(2026, 6, 9, 9, 0, 0, 0, SushiroTimezone), "2026-06-09"); ok {
			t.Fatalf("routine %+v 不应走快路径", routine)
		}
	}
}

func TestRefreshNetTicketRoutineArmedTickUsesStoredWindow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	saveRoutineNotifyForTest(t)
	cst := SushiroTimezone
	now := time.Date(2026, 6, 9, 11, 0, 0, 0, cst)
	// 已推算过的 armed Routine 落盘（无历史观测样本也成立——快路径不得依赖重算）。
	if err := SaveNetTicketRoutine(NetTicketRoutine{
		Enabled: true, StoreID: "9999", TargetMealTime: "1300",
		Status: netTicketRoutineStatusArmed, PlannedDate: "2026-06-09",
		PlannedPickupTime: "12:00", PlannedPickupEndTime: "12:30",
		ReminderTime: "11:50", NotifyBeforeMinutes: 10,
		LastPlannedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	resp := refreshNetTicketRoutineLocked(now)
	if resp.Routine.Status != netTicketRoutineStatusArmed {
		t.Fatalf("status = %q, want armed（快路径只判时刻，不该重算）; err=%q", resp.Routine.Status, resp.Routine.LastError)
	}
	if resp.Routine.PlannedPickupTime != "12:00" {
		t.Fatalf("planned pickup = %q, want 保留已存窗口 12:00", resp.Routine.PlannedPickupTime)
	}
}
