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
	"github.com/Ryujoxys/sushiro-overdose/internal/proxy"
)

func ticketConfirmationTestHome(t *testing.T) {
	t.Helper()
	t.Setenv("SUSHIRO_DATA_HOME", t.TempDir())
	resetAuthHealth()
	t.Cleanup(func() { clearWebSettings(); resetAuthHealth() })
	if err := SaveLocalConfig(reliabilityTokens()); err != nil {
		t.Fatal(err)
	}
}

func TestCancelTicketRequiresFreshMatchingConfirmation(t *testing.T) {
	for _, scenario := range []string{"missing", "different_account", "new_credentials", "generation", "different_ticket", "different_store", "no_ticket", "query_failed", "expired", "matching", "cancel_uncertain"} {
		t.Run(scenario, func(t *testing.T) {
			ticketConfirmationTestHome(t)
			payload := `{"netTicket":{"ticketId":123,"number":"1050","storeId":"1012","status":"WAITING"}}`
			queries, cancellations := 0, 0
			queryCode := http.StatusOK
			stubSamplingHTTP(t, func(r *http.Request) (int, string) {
				switch {
				case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/ticket/status"):
					queries++
					return queryCode, payload
				case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/cancelNetTicket"):
					cancellations++
					if scenario == "cancel_uncertain" {
						return http.StatusBadGateway, `{"error":"unavailable"}`
					}
					return http.StatusOK, `{}`
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					return http.StatusBadRequest, `{}`
				}
			})
			rr := httptest.NewRecorder()
			handleQueueTicketStatus(rr, httptest.NewRequest(http.MethodGet, "/api/queue/ticket/status", nil))
			var displayed struct {
				CancelToken string `json:"cancel_token"`
			}
			if rr.Code != http.StatusOK || json.Unmarshal(rr.Body.Bytes(), &displayed) != nil || displayed.CancelToken == "" {
				t.Fatalf("missing display confirmation: %d %s", rr.Code, rr.Body.String())
			}
			if strings.Contains(displayed.CancelToken, "wechat") || strings.Contains(displayed.CancelToken, "reservation") {
				t.Fatal("confirmation exposed credential fields")
			}
			switch scenario {
			case "missing":
				displayed.CancelToken = ""
			case "different_account", "new_credentials":
				tokens := reliabilityTokens()
				if scenario == "different_account" {
					tokens.WechatID = "different-wechat"
				} else {
					tokens.ReservationAuth = "new-session"
				}
				if err := SaveLocalConfig(tokens); err != nil {
					t.Fatal(err)
				}
			case "generation":
				authLifecycle.Lock()
				authLifecycle.generation++
				authLifecycle.Unlock()
			case "different_ticket":
				payload = strings.ReplaceAll(payload, `123`, `456`)
			case "different_store":
				payload = strings.ReplaceAll(payload, `1012`, `3006`)
			case "no_ticket":
				payload = `{"netTicket":null}`
			case "query_failed":
				queryCode, payload = http.StatusUnauthorized, `{"error":"expired"}`
			case "expired":
				ticketConfirmations.Lock()
				item := ticketConfirmations.items[displayed.CancelToken]
				item.Expires = time.Now().Add(-time.Second)
				ticketConfirmations.items[displayed.CancelToken] = item
				ticketConfirmations.Unlock()
			}
			body, _ := json.Marshal(map[string]string{"cancel_token": displayed.CancelToken})
			cancel := func() *httptest.ResponseRecorder {
				rr := httptest.NewRecorder()
				handleCancelNetTicket(rr, httptest.NewRequest(http.MethodPost, "/api/queue/ticket/cancel", strings.NewReader(string(body))))
				return rr
			}
			result := cancel()
			wantCancel := 0
			if scenario == "matching" || scenario == "cancel_uncertain" {
				wantCancel = 1
			}
			if cancellations != wantCancel {
				t.Fatalf("cancellations=%d, want=%d (%d %s)", cancellations, wantCancel, result.Code, result.Body.String())
			}
			if scenario == "matching" && result.Code != http.StatusOK {
				t.Fatalf("matching cancellation failed: %d %s", result.Code, result.Body.String())
			}
			if wantCancel == 0 && result.Code < 400 {
				t.Fatal("unsafe cancellation was accepted")
			}
			if wantCancel == 1 {
				if queries != 2 {
					t.Fatalf("no fresh check before cancel: queries=%d", queries)
				}
				if repeated := cancel(); repeated.Code != http.StatusConflict || cancellations != 1 {
					t.Fatal("confirmation was reused for another cancellation")
				}
			}
			if scenario == "query_failed" && getAuthHealth().Status != authHealthStale {
				t.Fatal("fresh-query auth rejection not reflected in account state")
			}
		})
	}
}

func TestTicketStatusUpdatesAuthHealthAndReturnsNoSecret(t *testing.T) {
	ticketConfirmationTestHome(t)
	code := http.StatusUnauthorized
	stubSamplingHTTP(t, func(r *http.Request) (int, string) {
		if r.Method != http.MethodGet {
			t.Fatal("status query attempted a write")
		}
		return code, `{"netTicket":{"ticketId":123,"number":"1050","storeId":"1012","status":"WAITING"}}`
	})
	rr := httptest.NewRecorder()
	handleQueueTicketStatus(rr, httptest.NewRequest(http.MethodGet, "/api/queue/ticket/status", nil))
	if getAuthHealth().Status != authHealthStale {
		t.Fatal("query did not mark rejected credentials stale")
	}
	code = http.StatusOK
	rr = httptest.NewRecorder()
	handleQueueTicketStatus(rr, httptest.NewRequest(http.MethodGet, "/api/queue/ticket/status", nil))
	if getAuthHealth().Status != authHealthOK {
		t.Fatal("successful query did not verify credentials")
	}
	for _, secret := range []string{"13800138000", `"wechat"`, `"reservation"`, `"query"`} {
		if strings.Contains(rr.Body.String(), secret) {
			t.Fatalf("response contains private field %s", secret)
		}
	}
}

func TestIncompleteTicketCannotReceiveCancelConfirmation(t *testing.T) {
	for _, ticket := range []ReservationRecord{{}, {Number: "1"}, {TicketID: 1}, {TicketID: 1, Number: "1", Kind: "reservation"}} {
		if token := issueTicketConfirmation(Settings{}, 0, ticket); token != "" {
			t.Fatal("ambiguous ticket received a cancellation confirmation")
		}
	}
}

func TestTicketActivityChecksAreInsideCredentialLock(t *testing.T) {
	source, err := os.ReadFile("web_engine.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, handler := range []string{"handleQueueTicket", "handleCancelNetTicket"} {
		body, ok := functionBody(string(source), handler)
		lock := strings.Index(body, "authLifecycle.Lock()")
		check := strings.Index(body, "isMainFlowRunning() || mobileAuthCapture.isActive()")
		if !ok || lock < 0 || check < lock {
			t.Fatalf("%s must check capture activity after claiming the credential lifecycle", handler)
		}
	}
}

func TestTicketMutationsRejectActiveCaptureWithoutNetwork(t *testing.T) {
	for _, mode := range []string{"desktop", "stopping", "mobile"} {
		t.Run(mode, func(t *testing.T) {
			ticketConfirmationTestHome(t)
			oldEngine, oldMobile := engine, mobileAuthCapture
			engine = &BookingEngine{state: EngineState{Status: EngineIdle}}
			mobileAuthCapture = &mobileAuthCaptureManager{}
			t.Cleanup(func() { engine, mobileAuthCapture = oldEngine, oldMobile })
			authLifecycle.Lock()
			switch mode {
			case "desktop":
				engine.state.Status = EngineCapturing
			case "stopping":
				engine.state.Status = EngineStopping
			case "mobile":
				// Only the activity marker is needed; no listener or proxy is started.
				mobileAuthCapture.proxy = &proxy.ProxyServer{}
			}
			authLifecycle.Unlock()
			calls := 0
			stubSamplingHTTP(t, func(r *http.Request) (int, string) {
				calls++
				return http.StatusInternalServerError, `{}`
			})
			for _, action := range []struct {
				path string
				body string
				run  http.HandlerFunc
			}{
				{"/api/queue/ticket", `{"store":"1012"}`, handleQueueTicket},
				{"/api/queue/ticket/cancel", `{"cancel_token":"confirmation"}`, handleCancelNetTicket},
			} {
				rr := httptest.NewRecorder()
				action.run(rr, httptest.NewRequest(http.MethodPost, action.path, strings.NewReader(action.body)))
				if rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), "认证") {
					t.Fatalf("%s did not reject active capture: %d %s", action.path, rr.Code, rr.Body.String())
				}
			}
			if calls != 0 {
				t.Fatalf("capture in progress caused %d official requests", calls)
			}
		})
	}
}
