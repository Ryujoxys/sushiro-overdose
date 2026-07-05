package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleAuthResetClearsLocalAuthAndPendingNetTicketPlan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Cleanup(func() {
		clearWebSettings()
		resetAuthHealth()
	})

	if err := SaveLocalConfig(&CapturedTokens{
		XAppCode:        "app-code",
		QueryAuth:       "Bearer query",
		ReservationAuth: "Bearer reservation",
		UserAgent:       "ua",
		Referer:         "https://servicewechat.com/",
		WechatID:        "wechat-id",
		PhoneNumber:     "13800138000",
		StoreIDs:        []string{"3006"},
	}); err != nil {
		t.Fatal(err)
	}
	setWebSettings(Settings{BaseURL: "https://example.invalid", StoreIDs: []string{"3006"}})
	markAuthStale("test stale")
	if err := SaveNetTicketPlan(NetTicketPlan{Enabled: true, StoreID: "3006", Status: "armed", TargetTime: "1800"}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/reset", nil)
	rr := httptest.NewRecorder()

	handleAuthReset(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if _, err := LoadLocalConfig(); err == nil {
		t.Fatal("local auth config should be deleted")
	}
	if getWebClient() != nil {
		t.Fatal("web client should be cleared")
	}
	if got := getAuthHealth().Status; got != authHealthUnknown {
		t.Fatalf("auth health = %q, want %q", got, authHealthUnknown)
	}
	plan := LoadNetTicketPlan()
	if plan.Enabled || plan.Status != "error" || !strings.Contains(plan.LastError, "重新获取通行证") {
		t.Fatalf("net ticket plan not reset for auth refresh: %+v", plan)
	}
}

func TestRefreshWebClientClearsStaleClientWhenConfigMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Cleanup(clearWebSettings)

	setWebSettings(Settings{BaseURL: "https://example.invalid", StoreIDs: []string{"3006"}})
	if getWebClient() == nil {
		t.Fatal("expected test web client before refresh")
	}

	refreshWebClient()

	if getWebClient() != nil {
		t.Fatal("refreshWebClient should clear stale client when config is missing")
	}
}
