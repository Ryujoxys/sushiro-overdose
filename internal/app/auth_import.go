package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type authImportRequest struct {
	Text string `json:"text"`
}

// handleAuthImport 接收用户粘贴的抓包文本（JSON / curl / 原始请求头），解析出凭证字段，
// 字段不全则只回预览不落盘；齐全则存 UA、存凭证、补默认门店、刷新设置并标记健康。
// 是凭证捕获代理之外的"手动导入"入口。
func handleAuthImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var req authImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "无效的请求格式: "+err.Error())
		return
	}
	tokens, sources, err := parseAuthImportText(req.Text)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	finalizeImportedTokens(tokens)
	missing := tokens.MissingFields(true)
	resp := map[string]any{
		"saved":   false,
		"message": "已解析，字段尚未完整",
		"missing": missing,
		"capture": captureStatusForTokens(tokens),
		"sources": sources,
	}
	if len(missing) > 0 {
		writeJSON(w, resp)
		return
	}
	tokens.Lock()
	rawUA := tokens.UserAgent
	tokens.Unlock()
	if strings.TrimSpace(rawUA) != "" {
		_, _ = SaveMobileUA(rawUA, "manual-auth-import", r.RemoteAddr)
	}
	if err := SaveLocalConfig(tokens); err != nil {
		writeError(w, http.StatusInternalServerError, "保存凭证参数失败: "+err.Error())
		return
	}
	markAuthHealthy()                       // 重新导入凭证 → 清除"凭证过期"提醒
	recordAuthCaptured(captureMethodImport) // 记录捕获时间/方式，重置寿命周期
	prefs := LoadPreferences()
	tokens.Lock()
	storeIDs := append([]string(nil), tokens.StoreIDs...)
	tokens.Unlock()
	if len(storeIDs) > 0 && len(prefs.SelectedStores) == 0 {
		// 读改写走 preferencesMu，避免与 UI 保存并发冲掉字段。
		_ = UpdatePreferences(func(p *UserPreferences) {
			if len(p.SelectedStores) == 0 {
				p.SelectedStores = append([]string(nil), storeIDs...)
				p.StorePriority = append([]string(nil), storeIDs...)
				prefs = *p // 同步给下方 setWebSettings
			}
		})
	}
	setWebSettings(tokens.ToSettingsWithPrefs(prefs))
	resp["saved"] = true
	resp["message"] = "凭证参数已导入并保存，可到本机诊断里测试基础接口。"
	resp["missing"] = []string{}
	resp["config_complete"] = true
	writeJSON(w, resp)
}

// parseAuthImportText 对同一段文本依次跑 JSON、curl、原始头三种解析器，每种命中就往 tokens 里补字段。
// 三者可以叠加（比如一段 curl 里既有 -H 头又有 -d body），所以不是"命中一种就停"。
// 只要最终没识别到任何凭证字段才算失败。返回的 sources 标注命中的来源，供前端展示。
func parseAuthImportText(text string) (*CapturedTokens, []string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil, fmt.Errorf("请先粘贴 JSON、curl 或请求头")
	}
	tokens := NewCapturedTokens()
	sources := map[string]struct{}{}
	if parseAuthImportJSON(tokens, text) {
		sources["json"] = struct{}{}
	}
	if parseAuthImportCurl(tokens, text) {
		sources["curl"] = struct{}{}
	}
	if parseAuthImportRaw(tokens, text) {
		sources["headers"] = struct{}{}
	}
	if captureStatusForTokens(tokens) == nil {
		return nil, nil, fmt.Errorf("解析失败")
	}
	sourceList := make([]string, 0, len(sources))
	for source := range sources {
		sourceList = append(sourceList, source)
	}
	if len(sourceList) == 0 {
		return nil, nil, fmt.Errorf("没有识别到凭证字段，请粘贴手机抓包导出的 curl、raw headers 或 JSON")
	}
	return tokens, sourceList, nil
}

// finalizeImportedTokens 交叉补齐两套 Authorization：寿司郎的查询接口和预约接口各自带 auth，
// 用户经常只抓到其中一个，这里用已有的那一个填充缺失的另一个，尽量让导入结果可直接用。
func finalizeImportedTokens(tokens *CapturedTokens) {
	tokens.Lock()
	defer tokens.Unlock()
	if tokens.QueryAuth == "" && tokens.ReservationAuth != "" {
		tokens.QueryAuth = tokens.ReservationAuth
	}
	if tokens.ReservationAuth == "" && tokens.QueryAuth != "" {
		tokens.ReservationAuth = tokens.QueryAuth
	}
}

func parseAuthImportJSON(tokens *CapturedTokens, text string) bool {
	var v any
	if json.Unmarshal([]byte(text), &v) != nil {
		return false
	}
	parseJSONNode(tokens, v, "")
	return true
}

func parseJSONNode(tokens *CapturedTokens, v any, context string) {
	switch x := v.(type) {
	case map[string]any:
		localContext := context
		for key, val := range x {
			if strings.EqualFold(key, "url") || strings.EqualFold(key, "requestURL") {
				if s, ok := scalarString(val); ok {
					localContext = mergeImportContext(localContext, authContextFromText(s))
					parseAuthImportURL(tokens, s)
				}
			}
		}
		for key, val := range x {
			keyContext := mergeImportContext(localContext, authContextFromText(key))
			if strings.EqualFold(key, "headers") {
				parseHeaderContainer(tokens, val, keyContext)
				continue
			}
			if shouldParseAsPayload(key) {
				if s, ok := scalarString(val); ok {
					parseBodyPayload(tokens, s, keyContext)
					continue
				}
			}
			if strings.EqualFold(key, "postData") || strings.EqualFold(key, "request") {
				parseJSONNode(tokens, val, keyContext)
				continue
			}
			if s, ok := scalarString(val); ok {
				applyAuthField(tokens, key, s, keyContext)
				continue
			}
			if isStoreField(key) {
				for _, storeID := range scalarStrings(val) {
					addImportedStore(tokens, storeID)
				}
				continue
			}
			parseJSONNode(tokens, val, keyContext)
		}
	case []any:
		for _, item := range x {
			parseJSONNode(tokens, item, context)
		}
	}
}

func parseHeaderContainer(tokens *CapturedTokens, v any, context string) {
	switch h := v.(type) {
	case map[string]any:
		for key, val := range h {
			if s, ok := scalarString(val); ok {
				applyAuthField(tokens, key, s, context)
			}
		}
	case []any:
		for _, item := range h {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name, _ := scalarString(m["name"])
			value, _ := scalarString(m["value"])
			if name != "" && value != "" {
				applyAuthField(tokens, name, value, context)
			}
		}
	}
}

// parseAuthImportCurl 解析 curl 命令：先按 shell 风格分词，再逐个识别 -H/--header、-A/--user-agent、
// -d/--data 等 flag，并把命令里的 URL 拿来抽 storeId 与上下文。同时兼容 "-Hxxx"、"--header=xxx"
// 这种连写形式。命中任意凭证字段即返回 true。
func parseAuthImportCurl(tokens *CapturedTokens, text string) bool {
	fields := shellLikeFields(text)
	if len(fields) == 0 {
		return false
	}
	found := false
	context := ""
	for _, field := range fields {
		if strings.HasPrefix(field, "http://") || strings.HasPrefix(field, "https://") {
			context = mergeImportContext(context, authContextFromText(field))
			parseAuthImportURL(tokens, field)
			found = true
		}
	}
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		switch field {
		case "-H", "--header":
			if i+1 < len(fields) && parseHeaderLine(tokens, fields[i+1], context) {
				found = true
				i++
			}
		case "-A", "--user-agent":
			if i+1 < len(fields) {
				applyAuthField(tokens, "User-Agent", fields[i+1], context)
				found = true
				i++
			}
		case "-d", "--data", "--data-raw", "--data-binary", "--data-ascii":
			if i+1 < len(fields) {
				parseBodyPayload(tokens, fields[i+1], context)
				found = true
				i++
			}
		default:
			if strings.HasPrefix(field, "-H") && len(field) > 2 && parseHeaderLine(tokens, strings.TrimSpace(field[2:]), context) {
				found = true
				continue
			}
			if strings.HasPrefix(field, "--header=") && parseHeaderLine(tokens, strings.TrimPrefix(field, "--header="), context) {
				found = true
				continue
			}
			if strings.HasPrefix(field, "--user-agent=") {
				applyAuthField(tokens, "User-Agent", strings.TrimPrefix(field, "--user-agent="), context)
				found = true
				continue
			}
			if strings.HasPrefix(field, "--data") {
				if idx := strings.Index(field, "="); idx >= 0 {
					parseBodyPayload(tokens, field[idx+1:], context)
					found = true
				}
			}
		}
	}
	return found
}

func parseAuthImportRaw(tokens *CapturedTokens, text string) bool {
	found := false
	context := authContextFromText(text)
	for _, match := range urlPattern.FindAllString(text, -1) {
		parseAuthImportURL(tokens, match)
		context = mergeImportContext(context, authContextFromText(match))
		found = true
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\\"))
		if parseHeaderLine(tokens, line, context) {
			found = true
		}
		if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") || strings.Contains(line, "wechatId") || strings.Contains(line, "phoneNumber") {
			before := captureStatusForTokens(tokens)
			parseBodyPayload(tokens, line, context)
			after := captureStatusForTokens(tokens)
			if before == nil || after != before {
				found = true
			}
		}
	}
	return found
}

func parseHeaderLine(tokens *CapturedTokens, line string, context string) bool {
	line = trimShellToken(line)
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return false
	}
	name := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if !isKnownAuthField(name) {
		return false
	}
	applyAuthField(tokens, name, value, context)
	return true
}

func parseBodyPayload(tokens *CapturedTokens, payload string, context string) bool {
	payload = strings.TrimSpace(trimShellToken(payload))
	if payload == "" {
		return false
	}
	var v any
	if json.Unmarshal([]byte(payload), &v) == nil {
		parseJSONNode(tokens, v, context)
		return true
	}
	if values, err := url.ParseQuery(payload); err == nil && len(values) > 0 {
		for key, vals := range values {
			for _, val := range vals {
				applyAuthField(tokens, key, val, context)
			}
		}
		return true
	}
	return false
}

func parseAuthImportURL(tokens *CapturedTokens, rawURL string) {
	u, err := url.Parse(strings.Trim(rawURL, `"'`))
	if err != nil {
		return
	}
	for _, key := range []string{"storeId", "store_id", "store", "storeID"} {
		if value := strings.TrimSpace(u.Query().Get(key)); value != "" {
			addImportedStore(tokens, value)
		}
	}
}

// applyAuthField 把单个"字段名=值"写入 tokens。两条核心规则：
//  1. 先到先得：每个字段只在为空时写入，避免后面的低质量来源覆盖前面已抓到的准确值。
//  2. authorization 按 context 区分：context=="reservation" 写 ReservationAuth，否则写 QueryAuth——
//     因为同一个 Authorization 头在两个接口里语义不同，靠 URL/正文推断出的 context 来分流。
func applyAuthField(tokens *CapturedTokens, key, value, context string) {
	value = strings.TrimSpace(trimShellToken(value))
	if value == "" {
		return
	}
	normalized := normalizeAuthFieldName(key)
	tokens.Lock()
	defer tokens.Unlock()
	switch normalized {
	case "xappcode":
		if tokens.XAppCode == "" {
			tokens.XAppCode = value
		}
	case "queryauthorization", "queryauth":
		if tokens.QueryAuth == "" {
			tokens.QueryAuth = value
		}
	case "reservationauthorization", "reservationauth":
		if tokens.ReservationAuth == "" {
			tokens.ReservationAuth = value
		}
	case "authorization":
		if context == "reservation" {
			if tokens.ReservationAuth == "" {
				tokens.ReservationAuth = value
			}
		} else if tokens.QueryAuth == "" {
			tokens.QueryAuth = value
		}
	case "useragent":
		if tokens.UserAgent == "" {
			tokens.UserAgent = value
		}
	case "referer", "referrer":
		if tokens.Referer == "" {
			tokens.Referer = value
		}
	case "xappclient":
		if tokens.XAppClient == "" {
			tokens.XAppClient = value
		}
	case "wechatid", "openid", "unionid":
		if tokens.WechatID == "" {
			tokens.WechatID = value
		}
	case "phonenumber", "phone", "mobile":
		if tokens.PhoneNumber == "" {
			tokens.PhoneNumber = value
		}
	case "storeid", "storeids", "store":
		for _, storeID := range splitStoreIDs(value) {
			addImportedStoreUnlocked(tokens, storeID)
		}
	}
}

func addImportedStore(tokens *CapturedTokens, storeID string) {
	tokens.Lock()
	defer tokens.Unlock()
	addImportedStoreUnlocked(tokens, storeID)
}

func addImportedStoreUnlocked(tokens *CapturedTokens, storeID string) {
	storeID = strings.TrimSpace(strings.Trim(storeID, `"'[]`))
	if storeID == "" {
		return
	}
	for _, existing := range tokens.StoreIDs {
		if existing == storeID {
			return
		}
	}
	tokens.StoreIDs = append(tokens.StoreIDs, storeID)
}

// shellLikeFields 做一个轻量的 shell 风格分词：处理行尾反斜杠续行、单/双引号包裹、反斜杠转义。
// 不追求完整 shell 语法（无变量展开、无命令替换），只覆盖 curl/抓包导出常见的引号与续行场景。
func shellLikeFields(s string) []string {
	s = strings.ReplaceAll(s, "\\\r\n", " ")
	s = strings.ReplaceAll(s, "\\\n", " ")
	var fields []string
	var b strings.Builder
	var quote rune
	escaped := false
	for _, r := range s {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if b.Len() > 0 {
				fields = append(fields, b.String())
				b.Reset()
			}
			continue
		}
		b.WriteRune(r)
	}
	if b.Len() > 0 {
		fields = append(fields, b.String())
	}
	return fields
}

func trimShellToken(s string) string {
	return strings.Trim(strings.TrimSpace(s), `"'`)
}

func normalizeAuthFieldName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer("-", "", "_", "", " ", "", ".", "")
	return replacer.Replace(s)
}

func isKnownAuthField(name string) bool {
	switch normalizeAuthFieldName(name) {
	case "xappcode", "queryauthorization", "queryauth", "reservationauthorization", "reservationauth", "authorization", "useragent", "referer", "referrer", "xappclient", "wechatid", "openid", "unionid", "phonenumber", "phone", "mobile", "storeid", "storeids", "store":
		return true
	default:
		return false
	}
}

func isStoreField(name string) bool {
	switch normalizeAuthFieldName(name) {
	case "storeid", "storeids", "store":
		return true
	default:
		return false
	}
}

func shouldParseAsPayload(name string) bool {
	switch normalizeAuthFieldName(name) {
	case "body", "data", "payload", "text", "postdatatext", "requestbody":
		return true
	default:
		return false
	}
}

func scalarString(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case float64:
		return fmt.Sprintf("%.0f", x), true
	case bool:
		if x {
			return "true", true
		}
		return "false", true
	default:
		return "", false
	}
}

func scalarStrings(v any) []string {
	switch x := v.(type) {
	case []any:
		var out []string
		for _, item := range x {
			if s, ok := scalarString(item); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		if s, ok := scalarString(v); ok {
			return splitStoreIDs(s)
		}
	}
	return nil
}

func splitStoreIDs(value string) []string {
	value = strings.Trim(value, `[]"'`)
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '，' || r == '、' || r == ' '
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.Trim(part, `"'`); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// authContextFromText 从 URL 或正文片段推断这套凭证属于哪个接口上下文：
// 含 api_auth/reservation/ticketing → "reservation"（预约接口）；含 /wechat/api/ 或 query → "query"（查询接口）；
// 否则空串（未知，applyAuthField 会按 query 兜底）。这个推断决定 Authorization 写到哪个字段。
func authContextFromText(s string) string {
	lower := strings.ToLower(s)
	if strings.Contains(lower, "api_auth") || strings.Contains(lower, "reservation") || strings.Contains(lower, "ticketing") {
		return "reservation"
	}
	if strings.Contains(lower, "/wechat/api/") || strings.Contains(lower, "query") {
		return "query"
	}
	return ""
}

// mergeImportContext 合并上下文推断：reservation 优先级最高（一旦认定预约就不再降级），
// next 为空保持 current，否则用 next。用于把 URL、字段名等多处线索汇成一个最终 context。
func mergeImportContext(current, next string) string {
	if current == "reservation" || next == "" {
		return current
	}
	return next
}

var urlPattern = regexp.MustCompile(`https?://[^\s'"<>]+`)
