package api

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 业务语义错误哨兵：与 HTTP 层错误（APIError）区分，调用方用 errors.Is 判断后再决定重试/提示。
var ErrNoReservationAvailable = errors.New("no reservation available")                   // 名额已满（E044 等）
var ErrReservationsEndpointUnavailable = errors.New("reservations endpoint unavailable") // 预约列表接口 404，老门店/老版本可能没有
var ErrActiveReservationExists = errors.New("active reservation exists")                 // 已有未完成预约（E052 等，官方一次只允许一个）

// APIError 包装 HTTP >=400 的响应，Body 已经过 NormalizeErrorBody 规整成可读字符串。
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// Client 是官方接口的薄封装。所有写操作都走 api_auth 路径（需 ReservationAuth），
// 查询走 api 路径（需 QueryAuthorization）。storeCache 缓存门店详情，避免重复拉取。
// 并发约定：mu 只保护 storeCache；httpClient 本身并发安全，doJSON 不加锁。
type Client struct {
	settings   Settings
	httpClient *http.Client
	mu         sync.Mutex
	storeCache map[string]StoreInfo
}

// NewClient 构造 Client。HTTP 超时固定 15s——官方接口偶有抖动，但取号场景不能无限等。
func NewClient(settings Settings) *Client {
	return &Client{
		settings: settings,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		storeCache: map[string]StoreInfo{},
	}
}

// GetStoreInfo 拉门店详情（GET /wechat/api/2.0/getStoreById?storeId=...）。
// 用 QueryAuthorization；结果按 storeID 缓存（门店静态信息短期内不变）。
// 缓存读写都在临界区内，避免并发重复请求互相覆盖。
func (c *Client) GetStoreInfo(ctx context.Context, storeID string) (StoreInfo, error) {
	c.mu.Lock()
	if store, ok := c.storeCache[storeID]; ok {
		c.mu.Unlock()
		return store, nil
	}
	c.mu.Unlock()

	query := url.Values{}
	query.Set("storeId", storeID)
	target := c.settings.BaseURL + "/wechat/api/2.0/getStoreById?" + query.Encode()
	body, err := c.doJSON(ctx, http.MethodGet, target, c.BaseHeaders(c.settings.QueryAuthorization, ""), nil)
	if err != nil {
		return StoreInfo{}, err
	}

	var store StoreInfo
	if err := json.Unmarshal(body, &store); err != nil {
		return StoreInfo{}, fmt.Errorf("store info response is not a JSON object: %w", err)
	}
	c.mu.Lock()
	c.storeCache[storeID] = store
	c.mu.Unlock()
	return store, nil
}

// GetTimeslots 拉某门店可约时段（GET /wechat/api/2.0/store/timeslots）。
// 用 QueryAuthorization；query 带桌型/门店/人数，人数影响返回哪些桌型有档期。
func (c *Client) GetTimeslots(ctx context.Context, storeID string) ([]Slot, error) {
	query := url.Values{}
	query.Set("tableType", c.settings.TableType)
	query.Set("storeId", storeID)
	query.Set("numpersons", strconv.Itoa(c.settings.NumPersons()))
	target := c.settings.BaseURL + "/wechat/api/2.0/store/timeslots?" + query.Encode()
	body, err := c.doJSON(ctx, http.MethodGet, target, c.BaseHeaders(c.settings.QueryAuthorization, ""), nil)
	if err != nil {
		return nil, err
	}

	var slots []Slot
	if err := json.Unmarshal(body, &slots); err != nil {
		return nil, fmt.Errorf("timeslots response is not a JSON array: %w", err)
	}
	return slots, nil
}

// CreateReservation 提交一个预约（POST /wechat/api_auth/2.0/ticketing/createReservation）。
// 走 api_auth（需 ReservationAuth）；slotDate/slotTime 为紧凑日期(YYYYMMDD)/时间(HHMMSS)。
// 失败时把错误原文映射成语义哨兵：名额已满→ErrNoReservationAvailable，已有预约→ErrActiveReservationExists。
// 官方一次只允许一个活跃预约，重复提交会被它拒绝。
func (c *Client) CreateReservation(ctx context.Context, storeID, slotDate, slotTime string) (ReservationRecord, error) {
	target := c.settings.BaseURL + "/wechat/api_auth/2.0/ticketing/createReservation"
	payload := map[string]any{
		"storeId":     storeID,
		"adult":       c.settings.Adult,
		"child":       c.settings.Child,
		"tableType":   c.settings.TableType,
		"wechatId":    c.settings.WechatID,
		"phoneNumber": c.settings.PhoneNumber,
		"date":        slotDate,
		"time":        slotTime,
	}
	body, err := c.doJSON(ctx, http.MethodPost, target, c.BaseHeaders(c.settings.ReservationAuth, "application/json"), payload)
	if err != nil {
		if IsNoReservationText(err.Error()) {
			return ReservationRecord{}, ErrNoReservationAvailable
		}
		if IsActiveReservationText(err.Error()) {
			return ReservationRecord{}, ErrActiveReservationExists
		}
		return ReservationRecord{}, err
	}

	return parseReservationRecord(body, "reservation")
}

// CreateNetTicket 远程取号（日常排队），对应小程序「排队取号」。端点与 body 字段
// (storeId/adult/child/tableType/wechatId/phoneNumber) 已用接口抓包(api_discovery)
// 比对过官方成功请求，确认一致。失败多为官方在该门店/时段不放行（E010 等），错误原文
// 原样返回便于定位。注意：复用同一套令牌，取号会顶掉手机端会话，反之亦然（见 spec 006）。
func (c *Client) CreateNetTicket(ctx context.Context, storeID string) (ReservationRecord, error) {
	target := c.settings.BaseURL + "/wechat/api_auth/2.0/ticketing/createNetTicket"
	// 取号人数/桌型兜底：若预约偏好未配置（adult/child 都为 0、tableType 为空），
	// 用合理默认，避免拿"0 个人"提交导致官方 500/E010。
	adult, child := c.settings.Adult, c.settings.Child
	if adult <= 0 && child <= 0 {
		adult = 2
	}
	tableType := c.settings.TableType
	if strings.TrimSpace(tableType) == "" {
		tableType = "T"
	}
	payload := map[string]any{
		"storeId":     storeID,
		"adult":       adult,
		"child":       child,
		"tableType":   tableType,
		"wechatId":    c.settings.WechatID,
		"phoneNumber": c.settings.PhoneNumber,
	}
	body, err := c.doJSON(ctx, http.MethodPost, target, c.BaseHeaders(c.settings.ReservationAuth, "application/json"), payload)
	if err != nil {
		return ReservationRecord{}, err
	}
	return parseReservationRecord(body, "net ticket")
}

// GetNetTicketStatus 查当前排队号状态（GET /wechat/api_auth/2.0/ticket/status?wechatId=...）。
// 用 ReservationAuth；按微信ID查当前活跃号，用于轮询「前面还排多少人」。
func (c *Client) GetNetTicketStatus(ctx context.Context) (ReservationRecord, error) {
	query := url.Values{}
	query.Set("wechatId", c.settings.WechatID)
	target := c.settings.BaseURL + "/wechat/api_auth/2.0/ticket/status?" + query.Encode()
	body, err := c.doJSON(ctx, http.MethodGet, target, c.BaseHeaders(c.settings.ReservationAuth, ""), nil)
	if err != nil {
		return ReservationRecord{}, err
	}
	return parseReservationRecord(body, "net ticket status")
}

// GetReservations 查历史/当前预约列表（POST /wechat/api_auth/2.0/ticketing/getReservations）。
// 响应可能是顶层数组，也可能是 {data:[...]} 包裹，故两种形态都试。404 特判为接口不可用哨兵
// （部分门店/版本没开放此接口）。
func (c *Client) GetReservations(ctx context.Context) ([]ReservationRecord, error) {
	target := c.settings.BaseURL + "/wechat/api_auth/2.0/ticketing/getReservations"
	body, err := c.doJSON(ctx, http.MethodPost, target, c.BaseHeaders(c.settings.ReservationAuth, "application/json"), map[string]any{
		"wechatId":    c.settings.WechatID,
		"phoneNumber": c.settings.PhoneNumber,
	})
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, ErrReservationsEndpointUnavailable
		}
		return nil, err
	}
	data := bytes.TrimSpace(body)
	if len(data) > 0 && data[0] == '{' {
		var wrapper struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return nil, fmt.Errorf("reservations response parse error: %w", err)
		}
		data = bytes.TrimSpace(wrapper.Data)
	}
	// An unknown envelope or null must not erase a previously saved reservation.
	if len(data) == 0 || data[0] != '[' {
		return nil, fmt.Errorf("reservations response does not contain a list")
	}
	var reservations []ReservationRecord
	if err := json.Unmarshal(data, &reservations); err != nil {
		return nil, fmt.Errorf("reservations response parse error: %w", err)
	}
	markReservationRecordsKind(reservations, "reservation")
	return reservations, nil
}

// CancelReservation 按票据ID取消预约（POST .../cancelReservation）。
// 走 api_auth；需带 ticketId + 身份字段（wechatId/phoneNumber）。
func (c *Client) CancelReservation(ctx context.Context, ticketID int64) error {
	target := c.settings.BaseURL + "/wechat/api_auth/2.0/ticketing/cancelReservation"
	_, err := c.doJSON(ctx, http.MethodPost, target, c.BaseHeaders(c.settings.ReservationAuth, "application/json"), map[string]any{
		"ticketId":    ticketID,
		"wechatId":    c.settings.WechatID,
		"phoneNumber": c.settings.PhoneNumber,
	})
	return err
}

// CancelNetTicket 取消当前排队号（日常排队），对应小程序「取消取号」。端点与
// 参数来自抓包：POST cancelNetTicket，body 只带 wechatId（取消当前活跃号，不按 id）。
func (c *Client) CancelNetTicket(ctx context.Context) error {
	target := c.settings.BaseURL + "/wechat/api_auth/2.0/ticketing/cancelNetTicket"
	_, err := c.doJSON(ctx, http.MethodPost, target, c.BaseHeaders(c.settings.ReservationAuth, "application/json"), map[string]any{
		"wechatId": c.settings.WechatID,
	})
	return err
}

// BaseHeaders 组装官方接口需要的公共请求头。authorization 由调用方决定用查询还是预约令牌；
// EnsureBearer 保证带 Bearer 前缀。Xweb_xhr 模拟微信小程序 WebView，提高和官方请求的相似度
// （降低被风控拦的概率）。contentType 非空时才加（GET 请求不带）。
func (c *Client) BaseHeaders(authorization, contentType string) map[string]string {
	headers := map[string]string{
		"Authorization": EnsureBearer(authorization),
		"X-App-Code":    c.settings.XAppCode,
		"X-App-Client":  c.settings.XAppClient,
		"User-Agent":    c.settings.UserAgent,
		"Referer":       c.settings.Referer,
		// 微信小程序 WebView 在每个 api_auth 请求都会带这个头，补齐以更贴近官方请求。
		"Xweb_xhr": "1",
	}
	if strings.TrimSpace(contentType) != "" {
		headers["Content-Type"] = contentType
	}
	return headers
}

// doJSON 是所有接口请求的统一通道：序列化 payload、发请求、读响应。
// 错误处理约定：
//   - HTTP >=400 → 返回 *APIError（Body 已规整），调用方可用 IsHTTPStatus/errors.As 判断状态码；
//   - 响应体非合法 JSON → 返回普通 error（官方偶尔返回 HTML 错误页，必须兜住）；
//   - 空响应体 → 返回 (nil, nil)，调用方自行处理。
//
// 重试：只对 GET（查询）的瞬时错误（429/408/5xx）做指数退避重试（尊重 Retry-After，带 jitter，
// 上限 3 次）。写操作（创建预约/取号）不自动重试——即使服务端 5xx 也可能已落库，重试有重复提交
// 风险，交给上层业务兜底（ErrActiveReservationExists / isTicketAlreadyIssuedError）。这避免
// 高频轮询时对接口的无退避轰炸触发风控。所有请求都带 ctx，超时由 httpClient.Timeout（15s）
// 和 ctx 双重约束；重试等待也响应 ctx 取消。
func (c *Client) doJSON(ctx context.Context, method, target string, headers map[string]string, payload any) ([]byte, error) {
	// payload 仅 marshal 一次，保存 data 供重试时重建可读 Reader。
	var data []byte
	if payload != nil {
		var err error
		data, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request payload: %w", err)
		}
	}

	const maxRetries = 3
	var lastErr error
	for attempt := 0; ; attempt++ {
		// 每次迭代重建 Reader：bytes.NewReader 从 data 起点读，重试也能发送完整请求体。
		var body io.Reader
		if data != nil {
			body = bytes.NewReader(data)
		}

		req, err := http.NewRequestWithContext(ctx, method, target, body)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// 网络层错误（连接重置/超时）对 GET 也按瞬时错误退避重试。
			lastErr = fmt.Errorf("request failed: %w", err)
			if method == http.MethodGet && attempt < maxRetries && ctx.Err() == nil {
				if waitErr := sleepForRetry(ctx, attempt, 0); waitErr == nil {
					continue
				}
			}
			return nil, lastErr
		}

		responseBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read response: %w", readErr)
			if method == http.MethodGet && attempt < maxRetries && ctx.Err() == nil {
				if waitErr := sleepForRetry(ctx, attempt, 0); waitErr == nil {
					continue
				}
			}
			return nil, lastErr
		}

		if resp.StatusCode >= 400 {
			apiErr := &APIError{StatusCode: resp.StatusCode, Body: NormalizeErrorBody(responseBody)}
			// 只对 GET 的瞬时错误（429/408/5xx）重试；尊重 Retry-After（秒）。
			if method == http.MethodGet && IsTransientHTTPStatus(resp.StatusCode) && attempt < maxRetries && ctx.Err() == nil {
				retryAfter := parseRetryAfterSeconds(resp.Header.Get("Retry-After"))
				if waitErr := sleepForRetry(ctx, attempt, retryAfter); waitErr == nil {
					lastErr = apiErr
					continue
				}
			}
			return nil, apiErr
		}

		if len(responseBody) == 0 {
			return nil, nil
		}
		if !json.Valid(responseBody) {
			return nil, fmt.Errorf("response is not JSON: %s", string(responseBody))
		}
		return responseBody, nil
	}
}

// IsTransientHTTPStatus 判断 HTTP 状态码是否属于「瞬时错误」值得重试：429（限流）、
// 408（请求超时）、5xx（服务端临时故障）。4xx 业务错误（404/409 等）不重试。
func IsTransientHTTPStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusRequestTimeout || status >= 500
}

// IsTransient 判断一个 doJSON 返回的错误是否是值得上层退避/降速的瞬时错误。
func IsTransient(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return IsTransientHTTPStatus(apiErr.StatusCode)
	}
	// 网络层错误（request failed / read response）也视为瞬时。
	return err != nil
}

// parseRetryAfterSeconds 解析 Retry-After 头（仅支持 delta-seconds 形式，HTTP-date 不解析返回 0）。
func parseRetryAfterSeconds(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0
	}
	if n > 30 {
		n = 30 // 封顶 30s，避免单次重试等太久拖垮取号窗口。
	}
	return n
}

// sleepForRetry 在重试前等待：基础指数退避（500ms / 1s / 2s）叠加 ±25% jitter，
// 或在 Retry-After 指定时取较大者。响应 ctx 取消（被取消时返回 ctx.Err()）。
func sleepForRetry(ctx context.Context, attempt, retryAfter int) error {
	base := 500 * time.Millisecond
	for i := 0; i < attempt && i < 4; i++ {
		base *= 2
	}
	jitter := time.Duration(rand.Int63n(int64(base) / 2)) // 0..base/2
	wait := base + jitter
	if ra := time.Duration(retryAfter) * time.Second; ra > wait {
		wait = ra
	}
	if wait > 30*time.Second {
		wait = 30 * time.Second
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// dataReaderReset 已废弃：doJSON 现在在循环外 marshal 一次、循环内用 bytes.NewReader(data) 重建。

// reservationLooksSuccessful 判断一条记录是否代表「真的拿到了号」：
// 要么有官方票据 ID，要么有排队号字符串。两者都空视为没成功。
func reservationLooksSuccessful(r ReservationRecord) bool {
	return r.TicketID != 0 || strings.TrimSpace(r.Number) != ""
}

// parseReservationRecord 解析预约/取号接口的响应。官方响应结构多变：
// 1) 直接是 ReservationRecord；2) 套在 netTicket/reservationTicket/data/ticket/currentTicket 等任意键里；
// 3) 更深的 TICKET_DETAIL + STORE_INFO 嵌套。这里按这个优先级逐层尝试。
// 都拿不到有效票据时再尝试映射业务错误（满座/已有预约），最后才报「missing ticket id/number」。
func parseReservationRecord(body []byte, label string) (ReservationRecord, error) {
	var record ReservationRecord
	if err := json.Unmarshal(body, &record); err != nil {
		return ReservationRecord{}, fmt.Errorf("%s response is not a JSON object: %w", label, err)
	}
	if reservationLooksSuccessful(record) {
		record = markReservationRecordKind(record, recordKindForLabel(label))
		return record, nil
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err == nil {
		for _, item := range []struct {
			key  string
			kind string
		}{
			{"netTicket", "net_ticket"},
			{"reservationTicket", "reservation"},
			{"reservation", "reservation"},
			{"data", recordKindForLabel(label)},
			{"ticket", recordKindForLabel(label)},
			{"currentTicket", recordKindForLabel(label)},
			{"current", recordKindForLabel(label)},
		} {
			key := item.key
			raw, ok := payload[key]
			if !ok || len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
				continue
			}
			if nested, ok := parseReservationCandidate(raw); ok {
				nested = markReservationRecordKind(nested, item.kind)
				return nested, nil
			}
		}
	}

	if err := ReservationBusinessError(body); err != nil {
		return ReservationRecord{}, err
	}
	return ReservationRecord{}, fmt.Errorf("%s response missing ticket id/number: %s", label, NormalizeErrorBody(body))
}

// parseReservationCandidate 处理「票务详情套在某个键里」的嵌套结构。
// 先试直接反序列化；不行就找 TICKET_DETAIL（大写/驼峰/下划线三种命名），
// 找到后再尝试从同级的 STORE_INFO 补门店名/地址。返回 bool=false 表示这层也没拿到有效票。
func parseReservationCandidate(raw json.RawMessage) (ReservationRecord, bool) {
	var record ReservationRecord
	if err := json.Unmarshal(raw, &record); err == nil && reservationLooksSuccessful(record) {
		return record, true
	}

	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return ReservationRecord{}, false
	}
	for _, detailKey := range []string{"TICKET_DETAIL", "ticketDetail", "ticket_detail"} {
		detailRaw, ok := wrapper[detailKey]
		if !ok || len(detailRaw) == 0 || bytes.Equal(detailRaw, []byte("null")) {
			continue
		}
		var detail ReservationRecord
		if err := json.Unmarshal(detailRaw, &detail); err != nil || !reservationLooksSuccessful(detail) {
			continue
		}
		for _, storeKey := range []string{"STORE_INFO", "storeInfo", "store_info"} {
			storeRaw, ok := wrapper[storeKey]
			if !ok || len(storeRaw) == 0 || bytes.Equal(storeRaw, []byte("null")) {
				continue
			}
			var store StoreInfo
			if err := json.Unmarshal(storeRaw, &store); err == nil {
				detail.StoreName = store.Name
				detail.StoreAddress = store.Address
				if detail.StoreID == "" && store.ID != 0 {
					detail.StoreID = strconv.Itoa(store.ID)
				}
			}
			break
		}
		return detail, true
	}
	return ReservationRecord{}, false
}

func recordKindForLabel(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	if strings.Contains(label, "net ticket") {
		return "net_ticket"
	}
	if strings.Contains(label, "reservation") {
		return "reservation"
	}
	return ""
}

func markReservationRecordKind(record ReservationRecord, kind string) ReservationRecord {
	if strings.TrimSpace(record.Kind) == "" {
		record.Kind = kind
	}
	return record
}

func markReservationRecordsKind(records []ReservationRecord, kind string) {
	for i := range records {
		records[i] = markReservationRecordKind(records[i], kind)
	}
}

// ReservationBusinessError 从响应体里找业务错误码/文案，映射成语义哨兵。
// 先看规整后的文本，再看 JSON 里的 code/errorCode/message 等键值，匹配 IsNoReservationText/IsActiveReservationText。
// 命中返回对应哨兵，否则返回 nil（表示不是已知的业务错误）。
func ReservationBusinessError(body []byte) error {
	text := NormalizeErrorBody(body)
	if IsNoReservationText(text) {
		return ErrNoReservationAvailable
	}
	if IsActiveReservationText(text) {
		return ErrActiveReservationExists
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	for _, key := range []string{"code", "errorCode", "error_code", "errCode", "message", "msg", "error"} {
		if v, ok := payload[key]; ok {
			text := fmt.Sprintf("%v", v)
			if IsNoReservationText(text) {
				return ErrNoReservationAvailable
			}
			if IsActiveReservationText(text) {
				return ErrActiveReservationExists
			}
		}
	}
	return nil
}

// IsNoReservationText 判断文本是否表示「名额已满/无号可约」。
// 覆盖官方多语言/多形态：错误码 E044、英文 no_more_reservations、中文「名额已满/已满」。
func IsNoReservationText(text string) bool {
	text = strings.ToLower(text)
	return strings.Contains(text, "e044") ||
		strings.Contains(text, "no_more_reservations") ||
		strings.Contains(text, "no reservation available") ||
		strings.Contains(text, "名额已满") ||
		strings.Contains(text, "已满")
}

// IsActiveReservationText 判断文本是否表示「已存在活跃预约」（官方一次只允许一个）。
// 覆盖错误码 E052、英文 already...reservation、日文「すでにご予約」、中文「已有/已经有预约/一次只能预约」。
func IsActiveReservationText(text string) bool {
	text = strings.ToLower(text)
	return strings.Contains(text, "e052") ||
		strings.Contains(text, "already") && strings.Contains(text, "reservation") ||
		strings.Contains(text, "one reservation at a time") ||
		strings.Contains(text, "すでにご予約") ||
		strings.Contains(text, "予約をいただいております") ||
		strings.Contains(text, "已有预约") ||
		strings.Contains(text, "已经有预约") ||
		strings.Contains(text, "一次只能预约")
}

func IsHTTPStatus(err error, status int) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == status
}

// SetHTTPClient 替换内部 HTTP 客户端（主要用于测试注入）。
func (c *Client) SetHTTPClient(h *http.Client) { c.httpClient = h }
