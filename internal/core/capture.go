package core

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// CaptureFromRequest 从 MITM 捕获到的请求里提取凭证参数填入 CapturedTokens。
// 调用方需保证 req 是寿司郎域名的请求（MITM 只对该域名解密）。
func (t *CapturedTokens) CaptureFromRequest(req *http.Request, bodyBytes []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if v := req.Header.Get("X-App-Code"); v != "" && t.XAppCode == "" {
		t.XAppCode = v
	}
	if v := req.Header.Get("User-Agent"); v != "" && t.UserAgent == "" {
		t.UserAgent = v
	}
	if v := req.Header.Get("Referer"); v != "" && t.Referer == "" {
		t.Referer = v
	}
	if v := req.Header.Get("X-App-Client"); v != "" && t.XAppClient == "" {
		t.XAppClient = v
	}

	authHeader := req.Header.Get("Authorization")
	if authHeader != "" {
		path := req.URL.Path
		if strings.Contains(path, "/api_auth/") || strings.Contains(path, "createReservation") {
			if t.ReservationAuth == "" {
				t.ReservationAuth = authHeader
			}
		} else if t.QueryAuth == "" {
			t.QueryAuth = authHeader
		}
	}

	query := req.URL.Query()
	if t.WechatID == "" {
		t.WechatID = query.Get("wechatId")
	}
	if t.PhoneNumber == "" {
		t.PhoneNumber = query.Get("phoneNumber")
	}
	addStore := func(sid string) {
		if sid == "" {
			return
		}
		found := false
		for _, existing := range t.StoreIDs {
			if existing == sid {
				found = true
				break
			}
		}
		if !found {
			t.StoreIDs = append(t.StoreIDs, sid)
		}
	}
	addStore(query.Get("storeId"))

	if req.Method == http.MethodPost && len(bodyBytes) > 0 {
		var body map[string]any
		if json.Unmarshal(bodyBytes, &body) == nil {
			if sid, ok := body["storeId"].(string); ok {
				addStore(sid)
			}
			if sid, ok := body["storeId"].(float64); ok && sid > 0 && sid == float64(int64(sid)) {
				addStore(strconv.FormatInt(int64(sid), 10))
			}
			if wid, ok := body["wechatId"].(string); ok && t.WechatID == "" {
				t.WechatID = wid
			}
			if pn, ok := body["phoneNumber"].(string); ok && t.PhoneNumber == "" {
				t.PhoneNumber = pn
			}
		}
	}
}

// Lock/Unlock 暴露 CapturedTokens 的内部锁，便于调用方在批量读取字段时加锁。
func (t *CapturedTokens) Lock()   { t.mu.Lock() }
func (t *CapturedTokens) Unlock() { t.mu.Unlock() }
