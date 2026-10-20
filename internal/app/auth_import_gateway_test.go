package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

const legacyTicketCurl = `curl 'https://crm-cn-prd.sushiro.com.cn/wechat/api_auth/2.0/ticketing/take?storeId=3006' \
  -H 'X-App-Code: app-code' \
  -H 'Authorization: Bearer reservation' \
  -H 'User-Agent: Mozilla/5.0 (iPhone) MicroMessenger/8.0 MiniProgramEnv/ios' \
  -H 'Referer: https://servicewechat.com/wx123/1/page-frame.html' \
  --data-raw '{"wechatId":"wx-user","phoneNumber":"13800138000"}'`

func TestGatewayRequestsNeverBecomeTicketCredentials(t *testing.T) {
	now := time.Now()
	token := testLiveJWT(now, now.Add(time.Hour))

	tokens, _, err := parseAuthImportText(gatewayCurl(token))
	if err != nil {
		t.Fatal(err)
	}
	finalizeImportedTokens(tokens)
	if tokens.QueryAuth != "" || tokens.ReservationAuth != "" || tokens.XAppCode != "" || tokens.XAppClient != "" {
		t.Fatalf("gateway headers became ticket credentials: query=%q reservation=%q app=%q", tokens.QueryAuth, tokens.ReservationAuth, tokens.XAppCode)
	}
	if strings.Join(tokens.StoreIDs, ",") != "1012" {
		t.Fatalf("store ids = %v", tokens.StoreIDs)
	}

	// "Copy all as cURL": the legacy request still supplies ticket credentials, the gateway one does not.
	for _, text := range []string{legacyTicketCurl + "\n" + gatewayCurl(token), gatewayCurl(token) + "\n" + legacyTicketCurl} {
		tokens, _, err = parseAuthImportText(text)
		if err != nil {
			t.Fatal(err)
		}
		finalizeImportedTokens(tokens)
		if err := tokens.ValidateForReservation(); err != nil {
			t.Fatalf("legacy credentials lost in a mixed paste: %v", err)
		}
		if strings.Contains(tokens.QueryAuth+tokens.ReservationAuth, token) || tokens.XAppCode != "app-code" {
			t.Fatalf("mixed paste took gateway values: query=%q reservation=%q app=%q", tokens.QueryAuth, tokens.ReservationAuth, tokens.XAppCode)
		}
	}

	har := `{"log":{"entries":[{"startedDateTime":"2026-10-20T12:00:00Z","request":{"method":"GET","url":"https://sapi.sushiro.com.cn/gateway/wechat/api/2.0/getStoreById?storeId=1012","headers":[{"name":"authorization","value":"Bearer ` + token + `"},{"name":"x-app-code","value":"` + testLiveAppCode + `"},{"name":"user-agent","value":"MicroMessenger"}]}}]}}`
	tokens, _, err = parseAuthImportText(har)
	if err != nil {
		t.Fatal(err)
	}
	if tokens.QueryAuth != "" || tokens.ReservationAuth != "" || tokens.XAppCode != "" || tokens.UserAgent != "" {
		t.Fatalf("HAR gateway entry became ticket credentials: %+v", captureStatusForTokens(tokens))
	}

	for current, next := range map[[2]string]string{
		{"", "query"}:                     "query",
		{"query", "reservation"}:          "reservation",
		{"reservation", "query"}:          "reservation",
		{"reservation", "gateway"}:        "gateway",
		{"gateway", "reservation"}:        "gateway",
		{"query", ""}:                     "query",
		{importContextGateway, ""}:        importContextGateway,
		{"", importContextGateway}:        importContextGateway,
		{"reservation", ""}:               "reservation",
		{"query", importContextGateway}:   importContextGateway,
		{importContextGateway, "query"}:   importContextGateway,
		{"reservation", "reservation"}:    "reservation",
		{"", ""}:                          "",
		{"query", "query"}:                "query",
		{"", "reservation"}:               "reservation",
		{importContextGateway, "gateway"}: importContextGateway,
	} {
		if got := mergeImportContext(current[0], current[1]); got != next {
			t.Errorf("mergeImportContext(%q, %q) = %q, want %q", current[0], current[1], got, next)
		}
	}
}

func TestCredentialImportConnectsLiveDataFromAGatewayPaste(t *testing.T) {
	reliabilityHome(t)
	useInterfaceLiveSessionProvider(t)
	now := time.Now()
	token := testLiveJWT(now, now.Add(time.Hour))
	body, _ := json.Marshal(map[string]string{"text": gatewayCurl(token)})
	rr := httptest.NewRecorder()
	handleAuthImport(rr, httptest.NewRequest(http.MethodPost, "/api/auth/import", strings.NewReader(string(body))))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Saved         bool   `json:"saved"`
		LiveConnected bool   `json:"live_connected"`
		Message       string `json:"message"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Saved || !resp.LiveConnected || !strings.Contains(resp.Message, "实时数据已连接") {
		t.Fatalf("response = %+v", resp)
	}
	assertNoLiveSecrets(t, "import response", rr.Body.String(), token)
	if s, err := LoadLiveSession(); err != nil || s == nil || s.Authorization != "Bearer "+token {
		t.Fatalf("live session not saved: %v", err)
	}
	if _, err := os.Stat(LocalConfigPath()); !os.IsNotExist(err) {
		t.Fatal("a gateway paste wrote ticket credentials")
	}

	// A legacy-only paste does not touch the live connection.
	if err := ClearLiveSession(); err != nil {
		t.Fatal(err)
	}
	body, _ = json.Marshal(map[string]string{"text": legacyTicketCurl})
	rr = httptest.NewRecorder()
	handleAuthImport(rr, httptest.NewRequest(http.MethodPost, "/api/auth/import", strings.NewReader(string(body))))
	if rr.Code != http.StatusOK || strings.Contains(rr.Body.String(), `"live_connected":true`) {
		t.Fatalf("legacy import: %d %s", rr.Code, rr.Body.String())
	}
	if s, _ := LoadLiveSession(); s != nil {
		t.Fatal("legacy paste created a live connection")
	}
	t.Cleanup(func() { _ = os.Remove(LocalConfigPath()) })
}

func TestCredentialImportExplainsAnUnusableGatewayPaste(t *testing.T) {
	reliabilityHome(t)
	useInterfaceLiveSessionProvider(t)
	now := time.Now()
	expired := testLiveJWT(now.Add(-2*time.Hour), now.Add(-time.Hour))
	importText := func(text string) (int, map[string]any, string) {
		body, _ := json.Marshal(map[string]string{"text": text})
		rr := httptest.NewRecorder()
		handleAuthImport(rr, httptest.NewRequest(http.MethodPost, "/api/auth/import", strings.NewReader(string(body))))
		var resp map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &resp)
		return rr.Code, resp, rr.Body.String()
	}

	code, resp, raw := importText(gatewayCurl(expired))
	if code != http.StatusOK || resp["saved"] != false || resp["live_connected"] != false {
		t.Fatalf("expired gateway paste: %d %s", code, raw)
	}
	message, _ := resp["message"].(string)
	liveError, _ := resp["live_error"].(string)
	if !strings.HasPrefix(message, "实时数据没有连上") || !strings.Contains(message, "过期") || liveError != message {
		t.Fatalf("message = %q live_error = %q", message, liveError)
	}
	assertNoLiveSecrets(t, "import response", raw, expired)
	if s, _ := LoadLiveSession(); s != nil {
		t.Fatal("an expired paste was saved as the live connection")
	}
	if _, err := os.Stat(LocalConfigPath()); !os.IsNotExist(err) {
		t.Fatal("a gateway paste wrote ticket credentials")
	}

	// A paste without any gateway request says nothing about live data.
	code, resp, raw = importText(legacyTicketCurl)
	if code != http.StatusOK || resp["live_error"] != nil || strings.Contains(raw, "实时数据") {
		t.Fatalf("legacy paste mentions live data: %d %s", code, raw)
	}
	t.Cleanup(func() { _ = os.Remove(LocalConfigPath()) })
}

func TestCredentialImportIgnoresIncidentalGatewayMentions(t *testing.T) {
	reliabilityHome(t)
	useInterfaceLiveSessionProvider(t)
	// A legacy request whose Referer happens to name the gateway host is not a failed live connection.
	text := strings.Replace(legacyTicketCurl, "https://servicewechat.com/wx123/1/page-frame.html", "https://sapi.sushiro.com.cn/page", 1)
	connected, problem := connectLiveSessionFromImport(text, time.Now())
	if connected || problem != "" {
		t.Fatalf("connected=%v problem=%q", connected, problem)
	}
	// A gateway request that lacks its app context does explain itself.
	now := time.Now()
	broken := strings.Replace(gatewayCurl(testLiveJWT(now, now.Add(time.Hour))), "-H 'x-app-code: "+testLiveAppCode+"'", "", 1)
	if connected, problem = connectLiveSessionFromImport(broken, now); connected || !strings.HasPrefix(problem, "实时数据没有连上：") {
		t.Fatalf("broken gateway paste: connected=%v problem=%q", connected, problem)
	}
}
