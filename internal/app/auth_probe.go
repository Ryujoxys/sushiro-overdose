package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type AuthProbeReport struct {
	OK            bool              `json:"ok"`
	Authenticated bool              `json:"authenticated"`
	StoreID       string            `json:"store_id,omitempty"`
	Store         string            `json:"store,omitempty"`
	Missing       []string          `json:"missing,omitempty"`
	Results       []AuthProbeResult `json:"results"`
	Advice        []string          `json:"advice,omitempty"`
}

type AuthProbeResult struct {
	Name      string `json:"name"`
	Method    string `json:"method,omitempty"`
	Path      string `json:"path,omitempty"`
	OK        bool   `json:"ok"`
	Skipped   bool   `json:"skipped,omitempty"`
	Status    int    `json:"status,omitempty"`
	LatencyMS int64  `json:"latency_ms,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

func handleAuthProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET or POST")
		return
	}
	report := RunAuthProbe(r.Context(), strings.TrimSpace(r.URL.Query().Get("store")))
	status := http.StatusOK
	if !report.OK {
		status = http.StatusBadGateway
	}
	writeJSONStatus(w, status, report)
}

func RunAuthProbe(ctx context.Context, requestedStore string) AuthProbeReport {
	generation := authGeneration()
	tokens, err := LoadLocalConfig()
	if err != nil {
		report := AuthProbeReport{}
		report.Missing = []string{"config.json"}
		report.Advice = []string{"本地还没有凭证参数；先从可用设备获取凭证，或导入已有 config.json。"}
		report.Results = append(report.Results, AuthProbeResult{Name: "读取本地凭证", OK: false, Detail: err.Error()})
		return report
	}

	prefs := LoadPreferences()
	report := runAuthProbeWithTokens(ctx, requestedStore, tokens, prefs)
	withAuthGeneration(generation, func() { applyAuthProbeHealth(report) })
	return report
}

func runAuthProbeWithTokens(ctx context.Context, requestedStore string, tokens *CapturedTokens, prefs UserPreferences) AuthProbeReport {
	return runAuthProbeWithClient(ctx, requestedStore, tokens, prefs, directProbeHTTPClient())
}

func runAuthProbeWithClient(ctx context.Context, requestedStore string, tokens *CapturedTokens, prefs UserPreferences, httpClient *http.Client) AuthProbeReport {
	report := AuthProbeReport{}
	settings := tokens.ToSettingsWithPrefs(prefs)
	tokens.Lock()
	capturedStores := append([]string(nil), tokens.StoreIDs...)
	tokens.Unlock()
	storeID := chooseProbeStoreID(requestedStore, settings.StoreIDs, capturedStores)
	report.StoreID = storeID

	if missing := tokens.MissingFields(false); len(missing) > 0 {
		report.Missing = append(report.Missing, missing...)
		report.Results = append(report.Results, AuthProbeResult{Name: "查询凭证参数", OK: false, Detail: "缺少: " + strings.Join(missing, ", ")})
		report.Advice = append(report.Advice, "查询凭证不完整，基础门店/时段接口无法验证。")
		return report
	}
	if storeID == "" {
		report.Missing = append(report.Missing, "门店")
		report.Results = append(report.Results, AuthProbeResult{Name: "选择探测门店", OK: false, Detail: "没有可用门店 ID"})
		report.Advice = append(report.Advice, "凭证配置里没有门店 ID，先在小程序里打开目标门店，或手动补充 store_ids。")
		return report
	}

	storeResult, storeName := probeGetStoreInfo(ctx, httpClient, settings, storeID)
	report.Results = append(report.Results, storeResult)
	report.Store = storeName
	report.Results = append(report.Results, probeTimeslots(ctx, httpClient, settings, storeID))

	if missing := tokens.MissingFields(true); len(missing) > 0 {
		report.Results = append(report.Results, AuthProbeResult{
			Name:    "预约凭证接口",
			Skipped: true,
			Detail:  "缺少: " + strings.Join(missing, ", "),
		})
	} else {
		report.Results = append(report.Results, probeReservations(ctx, httpClient, settings))
	}

	report.OK = true
	for _, result := range report.Results {
		if !result.OK && !result.Skipped {
			report.OK = false
			break
		}
	}
	report.Advice = authProbeAdvice(report)
	for _, result := range report.Results {
		if result.OK && !result.Skipped && strings.Contains(result.Path, "/api_auth/") {
			report.Authenticated = true
		}
	}
	if authProbeRejected(report) {
		report.Authenticated = false
	}
	return report
}

func chooseProbeStoreID(requested string, preferred []string, captured []string) string {
	if requested != "" {
		return requested
	}
	for _, values := range [][]string{preferred, captured} {
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func directProbeHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 12 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				NextProtos: []string{"h2", "http/1.1"},
			},
			ForceAttemptHTTP2: true,
		},
	}
}

func probeGetStoreInfo(ctx context.Context, client *http.Client, settings Settings, storeID string) (AuthProbeResult, string) {
	query := url.Values{}
	query.Set("storeId", storeID)
	path := "/wechat/api/2.0/getStoreById?" + query.Encode()
	result, body := probeOfficialAPI(ctx, client, http.MethodGet, settings.BaseURL+path, path, NewClient(settings).BaseHeaders(settings.QueryAuthorization, ""), nil)
	result.Name = "基础接口：门店信息"
	if result.OK {
		var store StoreInfo
		if err := json.Unmarshal(body, &store); err == nil {
			name := strings.TrimSpace(store.Name)
			result.Detail = fmt.Sprintf("门店 %s %s", storeID, DefaultString(name, "返回正常"))
			return result, name
		}
	}
	return result, ""
}

func probeTimeslots(ctx context.Context, client *http.Client, settings Settings, storeID string) AuthProbeResult {
	query := url.Values{}
	query.Set("tableType", settings.TableType)
	query.Set("storeId", storeID)
	query.Set("numpersons", fmt.Sprintf("%d", settings.NumPersons()))
	path := "/wechat/api/2.0/store/timeslots?" + query.Encode()
	result, body := probeOfficialAPI(ctx, client, http.MethodGet, settings.BaseURL+path, path, NewClient(settings).BaseHeaders(settings.QueryAuthorization, ""), nil)
	result.Name = "基础接口：时段列表"
	if result.OK {
		var slots []Slot
		if err := json.Unmarshal(body, &slots); err == nil {
			result.Detail = fmt.Sprintf("返回 %d 个时段", len(slots))
		}
	}
	return result
}

func probeReservations(ctx context.Context, client *http.Client, settings Settings) AuthProbeResult {
	path := "/wechat/api_auth/2.0/ticketing/getReservations"
	payload := map[string]any{
		"wechatId":    settings.WechatID,
		"phoneNumber": settings.PhoneNumber,
	}
	result, body := probeOfficialAPI(ctx, client, http.MethodPost, settings.BaseURL+path, path, NewClient(settings).BaseHeaders(settings.ReservationAuth, "application/json"), payload)
	result.Name = "凭证接口：当前预约"
	result = normalizeReservationsProbeResult(result)
	if result.Skipped {
		return result
	}
	if result.OK {
		var reservations []ReservationRecord
		data := bytes.TrimSpace(body)
		if len(data) > 0 && data[0] == '{' {
			var envelope struct {
				Data json.RawMessage `json:"data"`
			}
			_ = json.Unmarshal(data, &envelope)
			data = bytes.TrimSpace(envelope.Data)
		}
		if len(data) > 0 && data[0] == '[' && json.Unmarshal(data, &reservations) == nil {
			result.Detail = fmt.Sprintf("返回 %d 条预约", len(reservations))
		} else {
			result.OK = false
			result.Detail = "响应未包含预约列表，无法确认认证查询成功"
		}
	}
	return result
}

func normalizeReservationsProbeResult(result AuthProbeResult) AuthProbeResult {
	if result.Status == http.StatusNotFound && strings.Contains(result.Path, "/ticketing/getReservations") {
		result.Skipped = true
		result.OK = false
		result.Detail = "当前预约查询接口不可用或已变更；门店/时段基础接口已单独验证，抢号提交不依赖此自检项。"
	}
	return result
}

func probeOfficialAPI(ctx context.Context, client *http.Client, method, target, path string, headers map[string]string, payload any) (AuthProbeResult, []byte) {
	result := AuthProbeResult{Method: method, Path: path}
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			result.Detail = "构造请求失败: " + err.Error()
			return result, nil
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		result.Detail = "构造请求失败: " + err.Error()
		return result, nil
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	start := time.Now()
	resp, err := client.Do(req)
	result.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		result.Detail = SanitizeDiagnosticLine(err.Error())
		return result, nil
	}
	defer resp.Body.Close()
	result.Status = resp.StatusCode
	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		result.Detail = "读取响应失败: " + readErr.Error()
		return result, nil
	}
	if resp.StatusCode >= 400 {
		result.Detail = NormalizeErrorBody(respBody)
		return result, respBody
	}
	if len(respBody) > 0 && !json.Valid(respBody) {
		result.Detail = "响应不是 JSON: " + SanitizeDiagnosticLine(string(respBody))
		return result, respBody
	}
	result.OK = true
	result.Detail = "HTTP " + resp.Status
	return result, respBody
}

func authProbeAdvice(report AuthProbeReport) []string {
	if report.OK {
		return []string{"只读检查已完成；跳过的接口未验证，预约和取号能力仍以实际提交结果为准。"}
	}
	out := []string{}
	for _, result := range report.Results {
		if result.OK || result.Skipped {
			continue
		}
		if result.Status == http.StatusUnauthorized || result.Status == http.StatusForbidden {
			out = append(out, "官方接口返回凭证失败，优先重新获取凭证参数。")
			continue
		}
		if result.Status >= 500 {
			out = append(out, "官方接口返回服务器错误，保留本结果继续排查请求头/接口兼容性。")
			continue
		}
		if result.Status == 0 && strings.TrimSpace(result.Detail) != "" {
			out = append(out, "本机直连官方接口失败，先排查网络、DNS 或系统代理残留。")
			continue
		}
	}
	if len(out) == 0 {
		out = append(out, "基础接口未全部通过，复制本结果继续排查。")
	}
	return UniqueNonEmptyStrings(out)
}

func authProbeRejected(report AuthProbeReport) bool {
	for _, result := range report.Results {
		if result.Status == http.StatusUnauthorized || result.Status == http.StatusForbidden {
			return true
		}
	}
	return false
}

func applyAuthProbeHealth(report AuthProbeReport) {
	switch {
	case authProbeRejected(report):
		markAuthStale("只读检查返回认证失败（401/403）")
	case report.Authenticated:
		markAuthHealthy()
	default:
		// Temporary failures are not evidence that previously rejected tokens recovered.
		if getAuthHealth().Status != authHealthStale {
			markAuthUnverified()
		}
	}
}

func authProbeFailureSummary(report AuthProbeReport) string {
	parts := []string{}
	for _, result := range report.Results {
		if result.OK || result.Skipped {
			continue
		}
		label := strings.TrimSpace(result.Name)
		if label == "" {
			label = "接口"
		}
		detail := strings.TrimSpace(result.Detail)
		if result.Status != 0 {
			detail = fmt.Sprintf("HTTP %d %s", result.Status, detail)
		}
		if detail != "" {
			parts = append(parts, label+" "+SanitizeDiagnosticLine(detail))
		} else {
			parts = append(parts, label+" 失败")
		}
	}
	if len(parts) == 0 && len(report.Advice) > 0 {
		parts = append(parts, report.Advice...)
	}
	if len(parts) == 0 {
		return "未知原因"
	}
	return strings.Join(parts, "；")
}

func cmdAuthProbe() {
	report := RunAuthProbe(context.Background(), "")
	printAuthProbeReport(report)
	if !report.OK {
		os.Exit(1)
	}
}

func printAuthProbeReport(report AuthProbeReport) {
	status := "FAILED"
	if report.OK {
		status = "OK"
	}
	fmt.Printf("凭证基础接口自检: %s\n", status)
	if report.StoreID != "" {
		fmt.Printf("门店: %s %s\n", report.StoreID, report.Store)
	}
	for _, result := range report.Results {
		state := "FAIL"
		if result.OK {
			state = "OK"
		} else if result.Skipped {
			state = "SKIP"
		}
		fmt.Printf("  - %s [%s]", result.Name, state)
		if result.Status != 0 {
			fmt.Printf(" HTTP %d", result.Status)
		}
		if result.Detail != "" {
			fmt.Printf(" %s", result.Detail)
		}
		fmt.Println()
	}
	for _, advice := range report.Advice {
		fmt.Println("建议:", advice)
	}
}
