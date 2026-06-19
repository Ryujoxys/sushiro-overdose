package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Ryujoxys/sushiro-overdose/internal/platform"
)

func handleStatus(w http.ResponseWriter, r *http.Request) {
	pid := readPID()
	hasConfig := HasValidConfig()
	// quarantine 检测：<10ms（一次 xattr 子进程），darwin 外恒 false；executable_path 给前端拼修复命令用。
	quarantined, _ := platform.IsQuarantined()
	exePath, _ := os.Executable()
	status := map[string]any{
		"version":           Version,
		"running":           isRunning(),
		"pid":               pid,
		"has_config":        hasConfig,
		"platform":          runtime.GOOS,
		"engine":            engine.GetState(),
		"sampling":          sampler.GetState(),
		"auth_health":       getAuthHealth(),
		"notify_configured": len(configuredNotificationChannels()) > 0,
		"quarantined":       quarantined,
		"executable_path":   exePath,
	}
	writeJSON(w, status)
}

func handleReservations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	ws := getWebSettings()
	// 没绑定门店就不去打官方接口：寿司郎预约列表是「按已绑定门店」查询的，
	// 无门店时返回空数组，避免空配置触发无意义的远端请求与潜在错误。
	if len(ws.StoreIDs) == 0 {
		writeJSON(w, []ReservationRecord{})
		return
	}
	client := getWebClient()
	reservations, err := client.GetReservations(r.Context())
	if err != nil {
		if errors.Is(err, ErrReservationsEndpointUnavailable) {
			// 端点变更（404）不代表凭证失效，凭证健康保持不变。
			items := loadReservationsFallback()
			items = refreshReservationItemsWithCurrentNetTicket(r.Context(), client, items)
			writeJSON(w, map[string]any{
				"items":       items,
				"unavailable": true,
				"message":     "官方当前预约列表接口已变更或不可用；这里显示的是本地保存/补录记录，不代表寿司郎小程序里没有预约。",
			})
			return
		}
		noteAuthResult(err) // 凭证失败则标记 stale，触发提醒
		writeError(w, http.StatusInternalServerError, friendlyOfficialAPIError(err))
		return
	}
	markAuthHealthy()
	syncLocalReservationState(reservations)
	writeJSON(w, refreshReservationItemsWithCurrentNetTicket(r.Context(), client, reservations))
}

func syncLocalReservationState(reservations []ReservationRecord) {
	if len(reservations) == 0 {
		clearLocalReservationOnly()
		return
	}
	latest := reservations[0]
	if strings.TrimSpace(latest.Kind) == "" {
		latest.Kind = "reservation"
	}
	if err := SaveState(StateFilePath(), State{ActiveReservation: &latest, SavedAt: time.Now().Format(time.RFC3339)}); err != nil {
		LogMessage(time.Now(), "保存预约状态失败: "+err.Error())
	}
}

func clearLocalReservationOnly() {
	state, err := LoadState(StateFilePath())
	if err != nil || state.ActiveReservation == nil {
		return
	}
	active := *state.ActiveReservation
	if isLocalNetTicketRecord(active) {
		return
	}
	if err := ClearState(StateFilePath()); err != nil {
		LogMessage(time.Now(), "清除预约状态失败: "+err.Error())
	}
}

func refreshReservationItemsWithCurrentNetTicket(ctx context.Context, client *Client, items []ReservationRecord) []ReservationRecord {
	if client == nil {
		return filterStaleLocalNetTickets(items, time.Now())
	}
	ticket, err := client.GetNetTicketStatus(ctx)
	if err == nil && reservationRecordLooksSuccessful(ticket) {
		if reservationRecordIsReservation(ticket) {
			ticket.Kind = "reservation"
			return upsertReservationItem(items, ticket)
		}
		ticket = normalizeNetTicketRecord(ticket)
		syncLocalNetTicketState(ticket)
		return upsertReservationItem(items, ticket)
	}
	if isNoCurrentNetTicketError(err) {
		clearLocalNetTicketState()
		return removeNetTicketRecords(items)
	}
	clearStaleLocalNetTicketState(time.Now())
	return filterStaleLocalNetTickets(items, time.Now())
}

func normalizeNetTicketRecord(ticket ReservationRecord) ReservationRecord {
	if reservationRecordIsReservation(ticket) {
		ticket.Kind = "reservation"
		return ticket
	}
	ticket.Kind = "net_ticket"
	if strings.TrimSpace(ticket.Status) == "" {
		ticket.Status = "WAITING"
	}
	if strings.TrimSpace(ticket.QueueDate) == "" {
		ticket.QueueDate = time.Now().Format("20060102")
	}
	return ticket
}

func syncLocalNetTicketState(ticket ReservationRecord) {
	netTicketMu.Lock()
	plan := LoadNetTicketPlan()
	plan.Status = "success"
	plan.Number = ticket.Number
	plan.TicketID = ticket.TicketID
	plan.LastError = ""
	if strings.TrimSpace(plan.StoreID) == "" {
		plan.StoreID = DefaultString(ticket.MonitoredStoreID, ticket.StoreID)
	}
	if err := SaveNetTicketPlan(plan); err != nil {
		LogMessage(time.Now(), "保存排队号计划失败: "+err.Error())
	}
	netTicketMu.Unlock()

	state, err := LoadState(StateFilePath())
	if err == nil && (state.ActiveReservation == nil || isLocalNetTicketRecord(*state.ActiveReservation)) {
		if err := SaveState(StateFilePath(), State{ActiveReservation: &ticket, SavedAt: time.Now().Format(time.RFC3339)}); err != nil {
			LogMessage(time.Now(), "保存排队号状态失败: "+err.Error())
		}
	}
}

func clearLocalNetTicketState() {
	state, err := LoadState(StateFilePath())
	if err == nil && state.ActiveReservation != nil && isLocalNetTicketRecord(*state.ActiveReservation) {
		if err := ClearState(StateFilePath()); err != nil {
			LogMessage(time.Now(), "清除排队号状态失败: "+err.Error())
		}
	}
	netTicketMu.Lock()
	plan := LoadNetTicketPlan()
	if strings.TrimSpace(plan.Status) == "success" || strings.TrimSpace(plan.Status) == "issued_unknown" || plan.Number != "" || plan.TicketID != 0 {
		plan.Status = "idle"
		plan.Number = ""
		plan.TicketID = 0
		plan.LastError = ""
		if err := SaveNetTicketPlan(plan); err != nil {
			LogMessage(time.Now(), "保存排队号计划失败: "+err.Error())
		}
	}
	netTicketMu.Unlock()
}

func clearStaleLocalNetTicketState(now time.Time) {
	state, err := LoadState(StateFilePath())
	if err != nil || state.ActiveReservation == nil {
		return
	}
	if localNetTicketIsStale(*state.ActiveReservation, state.SavedAt, now) {
		clearLocalNetTicketState()
	}
}

func filterStaleLocalNetTickets(items []ReservationRecord, now time.Time) []ReservationRecord {
	out := items[:0]
	for _, item := range items {
		if localNetTicketIsStale(item, "", now) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func removeNetTicketRecords(items []ReservationRecord) []ReservationRecord {
	out := items[:0]
	for _, item := range items {
		if !isLocalNetTicketRecord(item) {
			out = append(out, item)
		}
	}
	return out
}

func upsertReservationItem(items []ReservationRecord, item ReservationRecord) []ReservationRecord {
	for i := range items {
		if reservationItemSameIdentity(items[i], item) {
			items[i] = item
			return items
		}
	}
	return append(items, item)
}

func reservationItemSameIdentity(a, b ReservationRecord) bool {
	if a.TicketID != 0 && b.TicketID != 0 {
		return a.TicketID == b.TicketID
	}
	return strings.TrimSpace(a.Kind) == strings.TrimSpace(b.Kind) &&
		strings.TrimSpace(a.Number) != "" &&
		strings.TrimSpace(a.Number) == strings.TrimSpace(b.Number)
}

func isLocalNetTicketRecord(record ReservationRecord) bool {
	if reservationRecordIsReservation(record) {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(record.Kind))
	return kind == "net_ticket" || record.Wait > 0 || strings.ToUpper(strings.TrimSpace(record.Status)) == "WAITING"
}

func reservationRecordLooksSuccessful(record ReservationRecord) bool {
	return strings.TrimSpace(record.Number) != "" || record.TicketID != 0
}

func reservationRecordIsReservation(record ReservationRecord) bool {
	kind := strings.ToLower(strings.TrimSpace(record.Kind))
	if kind == "reservation" || kind == "reservation_ticket" {
		return true
	}
	return reservationRecordHasSchedule(record)
}

func reservationRecordHasSchedule(record ReservationRecord) bool {
	return strings.TrimSpace(record.SlotLabel) != "" ||
		strings.TrimSpace(record.Start) != "" ||
		strings.TrimSpace(record.End) != ""
}

// localNetTicketIsStale 判断本地保存的排队号记录是否「跨天失效」。
// 排队号只对当天有效，过夜即为废号。判定维度二选一满足即判 stale：
//   - 记录自带的 QueueDate（YYYYMMDD）与今天不等；
//   - 或保存时刻 savedAt（RFC3339）落到今天以外的日期（用 now 的时区比较，避免 UTC 偏移误判）。
//
// now 为零值时回退 time.Now()，方便调用方不传。非本地排队号记录直接返回 false（不归本函数管）。
func localNetTicketIsStale(record ReservationRecord, savedAt string, now time.Time) bool {
	if !isLocalNetTicketRecord(record) {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	today := now.Format("20060102")
	if date := strings.TrimSpace(record.QueueDate); len(date) >= 8 && date[:8] != today {
		return true
	}
	if savedAt != "" {
		if at, err := time.Parse(time.RFC3339, savedAt); err == nil && at.In(now.Location()).Format("20060102") != today {
			return true
		}
	}
	return false
}

// isNoCurrentNetTicketError 判定错误是否语义上等价于「当前没有排队号」（而非网络/鉴权故障）。
// 用途：区分「该清掉本地过号记录」与「只是请求失败要保留」。命中 404 或中英文「无号」提示即视为前者。
// 注意最后一条 missing ticket id/number 还要叠加 null/{}/[] 才算，避免把含号错误误判。
func isNoCurrentNetTicketError(err error) bool {
	if err == nil {
		return false
	}
	if IsHTTPStatus(err, http.StatusNotFound) {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "no ticket") ||
		strings.Contains(text, "not issued") ||
		strings.Contains(text, "not found") ||
		strings.Contains(text, "no data") ||
		strings.Contains(text, "没有排队") ||
		strings.Contains(text, "未取号") ||
		strings.Contains(text, "不存在") ||
		strings.Contains(text, "missing ticket id/number") && (strings.Contains(text, "null") || strings.Contains(text, "{}") || strings.Contains(text, "[]"))
}

func loadReservationsFallback() []ReservationRecord {
	state, err := LoadState(StateFilePath())
	if err != nil || state.ActiveReservation == nil {
		return []ReservationRecord{}
	}
	reservation := *state.ActiveReservation
	if localNetTicketIsStale(reservation, state.SavedAt, time.Now()) {
		clearLocalNetTicketState()
		return []ReservationRecord{}
	}
	if strings.TrimSpace(reservation.Status) == "" {
		reservation.Status = "本地记录"
	}
	if strings.TrimSpace(reservation.Kind) == "" {
		reservation.Kind = "reservation"
	}
	return []ReservationRecord{reservation}
}

func handleLocalReservation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var body struct {
		Number    string `json:"number"`
		Date      string `json:"date"`
		Start     string `json:"start"`
		End       string `json:"end"`
		StoreID   string `json:"store_id"`
		StoreName string `json:"store_name"`
		TicketID  int64  `json:"ticket_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "无效的请求格式: "+err.Error())
		return
	}
	number := strings.TrimSpace(body.Number)
	date := normalizeLocalReservationDate(body.Date)
	if number == "" && date == "" {
		writeError(w, http.StatusBadRequest, "至少填写预约号或日期")
		return
	}
	start := normalizeLocalReservationTime(body.Start)
	end := normalizeLocalReservationTime(body.End)
	storeID := strings.TrimSpace(body.StoreID)
	storeName := strings.TrimSpace(body.StoreName)
	if storeID == "" {
		ws := getWebSettings()
		if len(ws.StoreIDs) > 0 {
			storeID = ws.StoreIDs[0]
		}
	}
	if storeName == "" && storeID != "" {
		if client := getWebClient(); client != nil {
			if info, err := client.GetStoreInfo(r.Context(), storeID); err == nil {
				storeName = info.Name
			}
		}
	}
	slotLabel := strings.TrimSpace(body.Date)
	if date != "" {
		slotLabel = date
	}
	if start != "" {
		slotLabel = strings.TrimSpace(slotLabel + " " + FormatCompactTime(start))
		if end != "" {
			slotLabel += "-" + FormatCompactTime(end)
		}
	}
	record := ReservationRecord{
		Kind:             "reservation",
		Status:           "本地补录",
		Number:           number,
		QueueDate:        date,
		Start:            start,
		End:              end,
		StoreID:          storeID,
		MonitoredStoreID: storeID,
		StoreName:        storeName,
		SlotLabel:        slotLabel,
		TicketID:         body.TicketID,
	}
	if err := SaveState(StateFilePath(), State{ActiveReservation: &record, SavedAt: time.Now().Format(time.RFC3339)}); err != nil {
		writeError(w, http.StatusInternalServerError, "保存本地预约失败: "+err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "reservation": record})
}

func normalizeLocalReservationDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 10 && value[4] == '-' && value[7] == '-' {
		return strings.ReplaceAll(value, "-", "")
	}
	if len(value) == 8 {
		return value
	}
	return ""
}

func normalizeLocalReservationTime(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ":", "")
	if len(value) == 4 {
		return value + "00"
	}
	if len(value) == 6 {
		return value
	}
	return ""
}

// handleQueueTicket 远程取号（实验性）。需要已捕获的凭证态。
func handleQueueTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var body struct {
		Store string `json:"store"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	storeID := strings.TrimSpace(body.Store)
	if storeID == "" {
		writeError(w, http.StatusBadRequest, "缺少门店 ID")
		return
	}
	// 写操作前强制重新读盘刷新凭证：取号是会产生真实副作用的操作（占用排队号），
	// 必须用最新凭证，避免用过期/被顶掉的旧凭证请求。
	refreshWebClient()
	client := getWebClient()
	if client == nil {
		writeError(w, http.StatusBadRequest, "尚未捕获凭证参数，请先完成凭证再取号")
		return
	}
	ticket, err := client.CreateNetTicket(r.Context(), storeID)
	if err != nil {
		noteAuthResult(err) // 凭证失败则标记 stale
		if isTicketAlreadyIssuedError(err) {
			plan := LoadNetTicketPlan()
			plan.Enabled = false
			plan.StoreID = storeID
			plan.Status = "issued_unknown"
			plan.FiredDate = time.Now().Format("2006-01-02")
			plan.FiredAt = time.Now().Format(time.RFC3339)
			if recovered, ok := recoverExistingNetTicket(r.Context(), client, &plan); ok {
				if err := SaveNetTicketPlan(recovered); err != nil {
					LogMessage(time.Now(), "保存排队号计划失败: "+err.Error())
				}
				writeJSON(w, map[string]any{"ok": true, "ticket": recovered, "recovered": true})
				return
			}
			markNetTicketIssuedUnknown(&plan, friendlyNetTicketError(err))
			writeError(w, http.StatusConflict, plan.LastError)
			return
		}
		writeError(w, http.StatusBadGateway, friendlyNetTicketError(err))
		return
	}
	markAuthHealthy() // 取号成功 → 凭证有效
	plan := LoadNetTicketPlan()
	plan.Enabled = false
	plan.StoreID = storeID
	plan.FiredDate = time.Now().Format("2006-01-02")
	plan.FiredAt = time.Now().Format(time.RFC3339)
	applyNetTicketSuccess(r.Context(), client, &plan, ticket)
	if err := SaveNetTicketPlan(plan); err != nil {
		LogMessage(time.Now(), "保存排队号计划失败: "+err.Error())
	}
	writeJSON(w, map[string]any{"ok": true, "ticket": ticket})
}

func handleQueueTicketStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	refreshWebClient()
	client := getWebClient()
	if client == nil {
		writeError(w, http.StatusBadRequest, "尚未捕获凭证参数，请先完成凭证")
		return
	}
	ticket, err := client.GetNetTicketStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, friendlyNetTicketError(err))
		return
	}
	plan := LoadNetTicketPlan()
	if strings.TrimSpace(plan.StoreID) == "" {
		plan.StoreID = ticket.StoreID
	}
	applyNetTicketSuccess(r.Context(), client, &plan, ticket)
	if err := SaveNetTicketPlan(plan); err != nil {
		LogMessage(time.Now(), "保存排队号计划失败: "+err.Error())
	}
	writeJSON(w, map[string]any{"ok": true, "ticket": ticket, "plan": plan})
}

// handleNetTicketPlan 读取/设置「定时取号」计划。
func handleNetTicketPlan(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, LoadNetTicketPlan())
	case http.MethodPost:
		var body struct {
			Enabled     bool   `json:"enabled"`
			Store       string `json:"store"`
			StoreName   string `json:"store_name"`
			TriggerMode string `json:"trigger_mode"`
			TargetTime  string `json:"target_time"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "无效的请求格式: "+err.Error())
			return
		}
		mode := strings.TrimSpace(body.TriggerMode)
		if mode != "on_open" {
			mode = "time"
		}
		netTicketMu.Lock()
		plan := LoadNetTicketPlan()
		plan.Enabled = body.Enabled
		plan.StoreID = strings.TrimSpace(body.Store)
		plan.StoreName = strings.TrimSpace(body.StoreName)
		plan.TriggerMode = mode
		plan.TargetTime = strings.TrimSpace(body.TargetTime)
		plan.Source = ""
		plan.TargetMealTime = ""
		plan.RoutinePlannedDate = ""
		// 重新设定即重置当天执行状态，允许（重新）到点取号。
		plan.FiredDate = ""
		plan.FiredAt = ""
		plan.Number = ""
		plan.TicketID = 0
		plan.LastError = ""
		plan.ServerRetryCount = 0
		plan.RetryDate = ""
		if body.Enabled {
			plan.Status = "armed"
		} else {
			plan.Status = "idle"
		}
		clearNetTicketFire(time.Now().Format("2006-01-02"))
		if err := SaveNetTicketPlan(plan); err != nil {
			netTicketMu.Unlock()
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		netTicketMu.Unlock()
		writeJSON(w, plan)
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST only")
	}
}

// handleNetTicketRoutine 读取/设置「每天想几点吃」Routine。
func handleNetTicketRoutine(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		netTicketMu.Lock()
		resp := NetTicketRoutineResponse{Routine: LoadNetTicketRoutine(), Plan: LoadNetTicketPlan()}
		netTicketMu.Unlock()
		writeJSON(w, resp)
	case http.MethodPost:
		var body struct {
			Enabled        bool   `json:"enabled"`
			Store          string `json:"store"`
			StoreID        string `json:"store_id"`
			StoreName      string `json:"store_name"`
			TargetMeal     string `json:"target_meal"`
			TargetMealTime string `json:"target_meal_time"`
			TravelMinutes  int    `json:"travel_minutes"`
			SafetyMinutes  *int   `json:"safety_minutes"`
			NotifyBefore   *int   `json:"notify_before_minutes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "无效的请求格式: "+err.Error())
			return
		}
		storeID := strings.TrimSpace(DefaultString(body.StoreID, body.Store))
		targetMeal := strings.TrimSpace(DefaultString(body.TargetMealTime, body.TargetMeal))
		if body.Enabled {
			if storeID == "" {
				writeError(w, http.StatusBadRequest, "请先选择门店")
				return
			}
			if _, ok := parseHHMM(targetMeal, time.Now()); !ok {
				writeError(w, http.StatusBadRequest, "请提供有效的目标就餐时间，例如 1300 或 13:00")
				return
			}
			if !routineNotifyConfigured() {
				writeError(w, http.StatusBadRequest, "启用 Routine 前必须先配置通知渠道，否则无法提醒你取号")
				return
			}
		}
		notifyBefore := netTicketRoutineDefaultNotifyBeforeMins
		if body.NotifyBefore != nil {
			notifyBefore = *body.NotifyBefore
		} else if body.SafetyMinutes != nil {
			notifyBefore = *body.SafetyMinutes
		}
		if notifyBefore < 0 {
			notifyBefore = 0
		}
		routine := NetTicketRoutine{
			Enabled:             body.Enabled,
			StoreID:             storeID,
			StoreName:           strings.TrimSpace(body.StoreName),
			TargetMealTime:      targetMeal,
			TravelMinutes:       max(0, body.TravelMinutes),
			NotifyBeforeMinutes: notifyBefore,
		}
		netTicketMu.Lock()
		resp := saveNetTicketRoutineConfigLocked(routine, time.Now())
		netTicketMu.Unlock()
		writeJSON(w, resp)
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST only")
	}
}

// handleCancelNetTicket 取消当前排队号（cancelNetTicket，只需 wechatId）。
func handleCancelNetTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	refreshWebClient()
	client := getWebClient()
	if client == nil {
		writeError(w, http.StatusBadRequest, "尚未捕获凭证参数，无法取消")
		return
	}
	if err := client.CancelNetTicket(r.Context()); err != nil {
		noteAuthResult(err)
		writeError(w, http.StatusBadGateway, friendlyNetTicketError(err))
		return
	}
	markAuthHealthy()
	// 取消成功后清掉本地取号计划状态，避免继续显示已取消的号。
	netTicketMu.Lock()
	plan := LoadNetTicketPlan()
	plan.Status = "idle"
	plan.Number = ""
	plan.TicketID = 0
	plan.LastError = ""
	if err := SaveNetTicketPlan(plan); err != nil {
		LogMessage(time.Now(), "保存排队号计划失败: "+err.Error())
	}
	netTicketMu.Unlock()
	writeJSON(w, map[string]any{"ok": true})
}

// handleCancelReservation 只取消预约单。排队号必须走 cancelNetTicket，避免误删未来预约。
func handleCancelReservation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var body struct {
		TicketID int64  `json:"ticket_id"`
		Kind     string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TicketID == 0 {
		writeError(w, http.StatusBadRequest, "缺少有效的 ticket_id")
		return
	}
	if strings.TrimSpace(body.Kind) != "reservation" {
		// 安全保护：取消是真实副作用，必须显式以 kind=reservation 调用本接口，
		// 防止前端误传或调用方把「排队号」当「预约」取消（排队号应走 cancelNetTicket），
		// 也避免 ticket_id 张冠李戴误删未来的预约。
		writeError(w, http.StatusBadRequest, "安全保护：此接口只允许取消明确标记为预约的记录；排队号请使用“取消排队号”。")
		return
	}
	refreshWebClient()
	client := getWebClient()
	if client == nil {
		writeError(w, http.StatusBadRequest, "尚未捕获凭证参数，无法取消")
		return
	}
	if err := client.CancelReservation(r.Context(), body.TicketID); err != nil {
		noteAuthResult(err)
		writeError(w, http.StatusBadGateway, friendlyOfficialAPIError(err))
		return
	}
	markAuthHealthy()
	writeJSON(w, map[string]any{"ok": true})
}

func handleEngineState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, engine.GetState())
}

func handleAuthReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	engine.Stop()
	DeleteLocalConfig()
	clearWebSettings()
	resetAuthHealth()
	resetNetTicketPlanAfterAuthReset()
	writeJSON(w, map[string]any{
		"ok":          true,
		"has_config":  false,
		"auth_health": getAuthHealth(),
		"message":     "已重置本地寿司郎凭证；凭证会过期或被手机端登录顶掉，请重新获取凭证。",
	})
}

func handleEngineCapture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	if err := engine.StartCapture(); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "message": "捕获已开始"})
}

func handleEngineBooking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	refreshWebClient()

	// 可选时段参数：齐全则直接预约这个确切时段，否则维持原"按偏好自动抢"。
	var body struct {
		Store string `json:"store"`
		Date  string `json:"date"`
		Start string `json:"start"`
		End   string `json:"end"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	if strings.TrimSpace(body.Store) != "" && strings.TrimSpace(body.Date) != "" && strings.TrimSpace(body.Start) != "" {
		if err := engine.StartBookingSlot(body.Store, body.Date, body.Start, body.End); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true, "message": "正在预约这个时段"})
		return
	}

	if err := engine.StartBooking(); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "message": "抢号已开始"})
}

func handleEngineStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	engine.Stop()
	writeJSON(w, map[string]any{"ok": true, "message": "已停止"})
}

// handleEngineReset 重置抓包状态：断开代理、清残留、回到 idle，便于手动重新连接。
func handleEngineReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	engine.ResetCapture()
	writeJSON(w, map[string]any{"ok": true, "engine": engine.GetState()})
}

func handleEngineLogs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, engine.GetLogs())
}

func handleInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	topN := defaultInsightTopN
	if raw := strings.TrimSpace(r.URL.Query().Get("top")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			topN = parsed
		}
	}
	analysis, err := LoadSlotHistoryInsights(topN, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, analysis)
}

func handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	writeJSON(w, CheckLatestRelease(r.Context()))
}
