package core

import (
	"net/http/httptest"
	"testing"
)

func TestCaptureIdentityFromReadOnlyRequests(t *testing.T) {
	tokens := NewCapturedTokens()
	req := httptest.NewRequest("GET", "https://example.invalid/wechat/api/2.0/getStoreById?storeId=1012", nil)
	req.Header.Set("X-App-Code", "app")
	req.Header.Set("Authorization", "query")
	req.Header.Set("User-Agent", "ua")
	req.Header.Set("Referer", "ref")
	tokens.CaptureFromRequest(req, nil)
	req = httptest.NewRequest("GET", "https://example.invalid/wechat/api_auth/2.0/ticket/status?wechatId=wechat&phoneNumber=13800138000", nil)
	req.Header.Set("Authorization", "reservation")
	tokens.CaptureFromRequest(req, nil)
	if !tokens.IsComplete() {
		t.Fatalf("missing fields: %v", tokens.MissingFields(true))
	}
	req = httptest.NewRequest("POST", "https://example.invalid/wechat/api_auth/2.0/ticketing/getReservations", nil)
	tokens.CaptureFromRequest(req, []byte(`{"storeId":3006}`))
	tokens.CaptureFromRequest(req, []byte(`{"storeId":"3006"}`))
	if len(tokens.StoreIDs) != 2 || tokens.StoreIDs[1] != "3006" {
		t.Fatal(tokens.StoreIDs)
	}
}

func TestCaptureKeepsTheNewestAuthorization(t *testing.T) {
	tokens := NewCapturedTokens()
	for _, step := range []struct{ path, auth string }{
		{"/wechat/api_auth/2.0/ticket/status", "Bearer old-reservation"},
		{"/wechat/api/2.0/getStoreById?storeId=1012", "Bearer old-query"},
		{"/wechat/api_auth/2.0/ticket/status", "Bearer new-reservation"},
		{"/wechat/api/2.0/getStoreById?storeId=1012", "Bearer new-query"},
		{"/wechat/api/2.0/getStoreById?storeId=1012", ""},
	} {
		req := httptest.NewRequest("GET", "https://example.invalid"+step.path, nil)
		if step.auth != "" {
			req.Header.Set("Authorization", step.auth)
		}
		req.Header.Set("X-App-Code", "app-"+step.auth)
		tokens.CaptureFromRequest(req, nil)
	}
	if tokens.ReservationAuth != "Bearer new-reservation" || tokens.QueryAuth != "Bearer new-query" {
		t.Fatalf("reservation=%q query=%q, want the newest tokens", tokens.ReservationAuth, tokens.QueryAuth)
	}
	if tokens.XAppCode != "app-Bearer old-reservation" {
		t.Fatalf("fixed app context should keep the first value: %q", tokens.XAppCode)
	}
}
