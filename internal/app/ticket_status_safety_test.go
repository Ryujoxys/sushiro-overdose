package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/api"
	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

// ticketSafetyServer answers the ticket status query with (statusCode, statusBody) and counts every
// ticket creation; the store lookup after a successful take is answered so nothing else is needed.
type ticketSafetyServer struct {
	mu         sync.Mutex
	statusCode int
	statusBody string
	creates    int
	server     *httptest.Server
}

func newTicketSafetyServer(t *testing.T, statusCode int, statusBody string) *ticketSafetyServer {
	t.Helper()
	s := &ticketSafetyServer{statusCode: statusCode, statusBody: statusBody}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/wechat/api_auth/2.0/ticket/status":
			w.WriteHeader(s.statusCode)
			_, _ = io.WriteString(w, s.statusBody)
		case r.Method == http.MethodPost && r.URL.Path == "/wechat/api_auth/2.0/ticketing/createNetTicket":
			s.creates++
			_, _ = io.WriteString(w, `{"ticketId":4321,"number":"1088","storeId":"1012","status":"WAITING"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/wechat/api/2.0/getStoreById":
			_, _ = io.WriteString(w, `{"id":1012,"name":"南山店","address":"深圳"}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(s.server.Close)
	return s
}

func (s *ticketSafetyServer) client() *Client {
	settings := Settings{
		BaseURL:         s.server.URL,
		WechatID:        "wechat",
		PhoneNumber:     "13800138000",
		ReservationAuth: "reservation",
		XAppCode:        "app",
		UserAgent:       "ua",
		Referer:         "ref",
		Adult:           2,
		TableType:       "T",
	}
	client := NewClient(settings)
	client.SetHTTPClient(s.server.Client())
	return client
}

func (s *ticketSafetyServer) createCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.creates
}

func TestTakeTicketRefusesWhenCurrentTicketIsUnknown(t *testing.T) {
	for _, tc := range []struct {
		name      string
		code      int
		body      string
		wantStale bool
	}{
		{"credentials rejected", http.StatusUnauthorized, `{"message":"token not found"}`, true},
		{"forbidden", http.StatusForbidden, `{"error":"no ticket permission"}`, true},
		{"server error", http.StatusInternalServerError, `{"message":"busy"}`, false},
		{"endpoint gone", http.StatusNotFound, `{"error":"Not Found"}`, false},
		{"rate limited", http.StatusTooManyRequests, `{"message":"no ticket"}`, false},
		{"unknown business error", http.StatusBadRequest, `{"code":"E999","message":"参数错误"}`, false},
		{"empty reply", http.StatusOK, ``, false},
		{"not json", http.StatusOK, `<html>not found</html>`, false},
		{"unrecognized object", http.StatusOK, `{"code":"E500","message":"cache not found","data":null}`, false},
		{"unrecognized list", http.StatusOK, `[{"queue":"A"}]`, false},
		{"object with message", http.StatusOK, `{"netTicket":null,"message":"系统繁忙"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reliabilityHome(t)
			server := newTicketSafetyServer(t, tc.code, tc.body)
			// Transient statuses are retried with backoff by the client; a short deadline keeps the test quick.
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			before := LoadNetTicketPlan()
			outcome := takeNetTicketLocked(ctx, server.client(), "1012")
			if server.createCount() != 0 {
				t.Fatalf("a ticket was submitted although the current ticket was unknown")
			}
			if outcome.errMsg != "查不到你现在有没有号，为避免重复取号，请在小程序里取号" || outcome.status < 400 || outcome.body != nil {
				t.Fatalf("outcome = %+v", outcome)
			}
			if after := LoadNetTicketPlan(); after.Status != before.Status || after.Number != "" {
				t.Fatalf("plan changed: %+v", after)
			}
			health := getAuthHealth().Status
			if tc.wantStale && health != authHealthStale {
				t.Fatalf("auth health = %s, want stale", health)
			}
			if !tc.wantStale && health == authHealthOK {
				t.Fatal("an unknown reply marked the credentials healthy")
			}
		})
	}
}

func TestTakeTicketProceedsOnlyAfterADefiniteAnswer(t *testing.T) {
	for _, tc := range []struct {
		name        string
		code        int
		body        string
		wantCreates int
		recovered   bool
	}{
		{"no ticket field", http.StatusOK, `{"netTicket":null}`, 1, false},
		{"both ticket fields empty", http.StatusOK, `{"netTicket":null,"reservationTicket":{}}`, 1, false},
		{"empty object", http.StatusOK, `{}`, 1, false},
		{"null", http.StatusOK, `null`, 1, false},
		{"official no-queue message", http.StatusBadRequest, `{"code":"E040","message":"当前没有排队号"}`, 1, false},
		{"reservation only", http.StatusOK, `{"reservationTicket":{"ticketId":7160,"number":"7160","start":"193000","queueDate":"20260611"}}`, 1, false},
		{"existing queue ticket", http.StatusOK, `{"netTicket":{"ticketId":123,"number":"1050","storeId":"1012","status":"WAITING"}}`, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reliabilityHome(t)
			server := newTicketSafetyServer(t, tc.code, tc.body)
			outcome := takeNetTicketLocked(context.Background(), server.client(), "1012")
			if outcome.errMsg != "" || outcome.body == nil {
				t.Fatalf("outcome = %+v", outcome)
			}
			if got := server.createCount(); got != tc.wantCreates {
				t.Fatalf("creates = %d, want %d", got, tc.wantCreates)
			}
			if recovered, _ := outcome.body["recovered"].(bool); recovered != tc.recovered {
				t.Fatalf("recovered = %v, want %v", recovered, tc.recovered)
			}
		})
	}
}

func TestTicketStatusNeverReportsNoTicketForUnknownReplies(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		body string
	}{
		{"auth failure that mentions not found", http.StatusUnauthorized, `{"message":"token not found"}`},
		{"reply mentioning not found", http.StatusOK, `{"code":"E500","message":"Resource not found in cache"}`},
		{"reply mentioning no data", http.StatusOK, `{"code":"E501","msg":"no data","detail":null}`},
		{"empty reply", http.StatusOK, ``},
		{"html", http.StatusOK, `<html>不存在</html>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reliabilityHome(t)
			resetAuthHealth()
			server := newTicketSafetyServer(t, tc.code, tc.body)
			rr := httptest.NewRecorder()
			writeQueueTicketStatusForAuth(rr, httptest.NewRequest(http.MethodGet, "/api/queue/ticket/status", nil), server.client(), Settings{}, 0)
			if rr.Code == http.StatusOK || strings.Contains(rr.Body.String(), `"ticket":null`) {
				t.Fatalf("unknown reply reported as no ticket: %d %s", rr.Code, rr.Body.String())
			}
			if getAuthHealth().Status == authHealthOK {
				t.Fatal("unknown reply marked credentials healthy")
			}
			var payload struct {
				Error string `json:"error"`
			}
			_ = json.Unmarshal(rr.Body.Bytes(), &payload)
			if !strings.Contains(payload.Error, "这不代表你没有号") || strings.Contains(payload.Error, "HTTP") || strings.Contains(payload.Error, "missing ticket") {
				t.Fatalf("message = %q", payload.Error)
			}
		})
	}

	reliabilityHome(t)
	resetAuthHealth()
	server := newTicketSafetyServer(t, http.StatusOK, `{"netTicket":null}`)
	rr := httptest.NewRecorder()
	writeQueueTicketStatusForAuth(rr, httptest.NewRequest(http.MethodGet, "/api/queue/ticket/status", nil), server.client(), Settings{}, 0)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"ticket":null`) || getAuthHealth().Status != authHealthOK {
		t.Fatalf("definite no-ticket reply: %d %s health=%s", rr.Code, rr.Body.String(), getAuthHealth().Status)
	}
}

func TestDefiniteNoCurrentNetTicketClassification(t *testing.T) {
	missing := func(body string) error {
		return fmt.Errorf("net ticket status response missing ticket id/number: %s", body)
	}
	for _, tc := range []struct {
		err  error
		want bool
	}{
		{nil, false},
		{missing(`{"netTicket":null}`), true},
		{missing(`{"netTicket":null,"reservationTicket":null,"current":""}`), true},
		{missing(`null`), true},
		{missing(`{}`), true},
		{missing(`[]`), true},
		{missing(`{"code":"E999","message":"当前没有排队号"}`), true},
		{missing(`{"message":null}`), false},
		{missing(`{"netTicket":null,"code":"E500"}`), false},
		{missing(`{"error":"Resource not found"}`), false},
		{missing(`{"msg":"no data"}`), false},
		{missing(`[{"queue":1}]`), false},
		{missing(`{"netTicket":null} trailing`), false},
		{errors.New("net ticket status response is not a JSON object: unexpected end of JSON input"), false},
		{errors.New("request failed: dial tcp: connection refused"), false},
		{errors.New("no ticket"), false},
		{&APIError{StatusCode: http.StatusBadRequest, Body: `{"message":"没有排队记录"}`}, true},
		{&APIError{StatusCode: http.StatusConflict, Body: `{"message":"no ticket issued"}`}, true},
		{&APIError{StatusCode: http.StatusBadRequest, Body: `{"message":"参数错误"}`}, false},
		{&APIError{StatusCode: http.StatusNotFound, Body: `{"message":"no ticket"}`}, false},
		{&APIError{StatusCode: http.StatusUnauthorized, Body: `{"message":"no ticket"}`}, false},
		{&APIError{StatusCode: http.StatusForbidden, Body: `not issued`}, false},
		{&APIError{StatusCode: http.StatusTooManyRequests, Body: `no ticket`}, false},
		{&APIError{StatusCode: http.StatusInternalServerError, Body: `no ticket`}, false},
	} {
		if got := isDefiniteNoCurrentNetTicket(tc.err); got != tc.want {
			t.Errorf("isDefiniteNoCurrentNetTicket(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
	// Clearing the local copy still follows the status endpoint's 404, but no loose text match.
	if !isNoCurrentNetTicketError(&APIError{StatusCode: http.StatusNotFound, Body: "{}"}) {
		t.Fatal("404 no longer clears the local copy")
	}
	if isNoCurrentNetTicketError(&APIError{StatusCode: http.StatusUnauthorized, Body: `{"message":"token not found"}`}) {
		t.Fatal("an auth failure cleared the local ticket")
	}
}

func TestRefreshKeepsLocalTicketWhenStatusIsUnknown(t *testing.T) {
	reliabilityHome(t)
	today := time.Now().Format("20060102")
	local := ReservationRecord{Kind: "net_ticket", Number: "893", QueueDate: today, Wait: 25}
	if err := SaveState(StateFilePath(), State{ActiveNetTicket: &local, SavedAt: time.Now().Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}
	server := newTicketSafetyServer(t, http.StatusUnauthorized, `{"message":"token not found"}`)
	items := refreshReservationItemsWithCurrentNetTicket(context.Background(), server.client(), []ReservationRecord{local})
	if len(items) != 1 || items[0].Number != "893" {
		t.Fatalf("local ticket dropped on an auth failure: %+v", items)
	}
	state, err := LoadState(StateFilePath())
	if err != nil || state.ActiveNetTicket == nil {
		t.Fatalf("local ticket state cleared: %+v %v", state, err)
	}
}
