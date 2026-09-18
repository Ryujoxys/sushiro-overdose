package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const netTicketPlanFile = "netticket_plan.json"

// NetTicketPlan 仅兼容旧版取号结果文件，不再调度或执行任何计划。
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
	// 保留旧文件字段供升级读取，不再计数或重试。
	ServerRetryCount int    `json:"server_retry_count,omitempty"`
	RetryDate        string `json:"retry_date,omitempty"`
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
	p = normalizeNetTicketPlan(p, time.Now())
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(netTicketPlanPath(), data, 0o600)
}

// normalizeNetTicketPlan 禁用所有旧自动计划，保留已经取得的号码。
func normalizeNetTicketPlan(p NetTicketPlan, now time.Time) NetTicketPlan {
	if now.IsZero() {
		now = time.Now()
	}
	if p.Status == "" {
		p.Status = "idle"
	}
	p.Enabled = false
	if p.Status == "armed" || p.Status == "retrying" {
		p.Status = "retired"
	}
	return p
}

// netTicketPlanFiredOn 判断计划在指定日期（默认今天）是否已「触发过」（成功或放弃都算）。
// 触发的判定优先看 FiredDate 字段；老数据可能只有 FiredAt（RFC3339 时刻），则解析出日期比对，做向后兼容。
// 仅用于旧认证采样的避让判断，不执行取号。
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
	plan.LastError = "已重置本地通行证；寿司郎通行证会过期或被手机端登录顶掉，请重新认证后手动取号"
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
