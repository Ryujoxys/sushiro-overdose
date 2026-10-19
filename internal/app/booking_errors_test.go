package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	api "github.com/Ryujoxys/sushiro-overdose/internal/api"
)

func TestKnownOfficialServerError(t *testing.T) {
	err := &api.APIError{
		StatusCode: 500,
		Body:       `{"code":"E010","message":"error.server:ご迷惑おかけします。サーバー側でエラーが発生しました。"}`,
	}
	if !isOfficialServerHTTPError(err) {
		t.Fatalf("isOfficialServerHTTPError() = false")
	}
	if !isKnownOfficialServerError(err) {
		t.Fatalf("isKnownOfficialServerError() = false")
	}
	if !isCredentialRefreshLikelyError(err) {
		t.Fatalf("isCredentialRefreshLikelyError() = false")
	}
	msg := friendlyOfficialAPIError(err)
	if !strings.Contains(msg, "E010") || !strings.Contains(msg, "重置通行证") || !strings.Contains(msg, "通行证已经过期") {
		t.Fatalf("friendly message = %q", msg)
	}
}

func TestNonServerErrorIsNotOfficialServerError(t *testing.T) {
	err := &api.APIError{StatusCode: 403, Body: `{"message":"forbidden"}`}
	if isOfficialServerHTTPError(err) {
		t.Fatalf("403 should not be treated as official server error")
	}
	if isKnownOfficialServerError(err) {
		t.Fatalf("403 should not be known server error")
	}
	if isCredentialRefreshLikelyError(err) {
		t.Fatalf("403 should not be treated as credential refresh likely")
	}
}

func TestGenericHTTP500IsNotCredentialRefreshLikely(t *testing.T) {
	err := &api.APIError{StatusCode: 500, Body: `{"message":"temporary outage"}`}
	if !isOfficialServerHTTPError(err) {
		t.Fatalf("500 should be treated as official server error")
	}
	if isKnownOfficialServerError(err) {
		t.Fatalf("generic 500 should not be known E010 server error")
	}
	if isCredentialRefreshLikelyError(err) {
		t.Fatalf("generic 500 should not be credential refresh likely")
	}
}

func TestTicketAlreadyIssuedError(t *testing.T) {
	err := &api.APIError{
		StatusCode: 409,
		Body:       `{"code":"E034","message":"error.newticket.too_many_tickets: The ticket has been already issued at your terminal."}`,
	}
	if !isTicketAlreadyIssuedError(err) {
		t.Fatalf("isTicketAlreadyIssuedError() = false")
	}
	msg := friendlyNetTicketError(err)
	if !strings.Contains(msg, "已经发过排队号") || strings.Contains(msg, "没取到") {
		t.Fatalf("friendly net ticket message = %q", msg)
	}
}

func TestFriendlyPublicQueueErrorHidesBlockPages(t *testing.T) {
	block := &api.APIError{StatusCode: 403, Body: "<html>\r\n<head><title>403 Forbidden</title></head></html>"}
	for err, want := range map[error]string{
		block:                                                     "海外 VPN",
		fmt.Errorf("store 1012: %w", block):                       "海外 VPN",
		errors.Join(block, errors.New("model")):                   "海外 VPN",
		&api.APIError{StatusCode: 429}:                            "太频繁",
		&api.APIError{StatusCode: 502, Body: "x"}:                 "暂时不可用",
		&api.APIError{StatusCode: 404, Body: "<HTML>gone</HTML>"}: "异常页面",
		context.DeadlineExceeded:                                  "请求超时",
		errors.New(`queue request failed: Get "https://example/stores": read tcp 127.0.0.1:1->127.0.0.1:2: read: connection reset by peer`): "网络不可达",
		errors.New(`Get "https://example": proxyconnect tcp: dial tcp: refused`):                                                            "网络不可达",
		errors.New(`Get "https://example": EOF`):                                                                                            "网络不可达",
	} {
		got := friendlyPublicQueueError(err)
		if !strings.Contains(got, want) || strings.Contains(strings.ToLower(got), "<html") {
			t.Errorf("%v => %q, want %q", err, got, want)
		}
	}
	// A JSON 403 is not a block page and keeps its original detail.
	if got := friendlyPublicQueueError(&api.APIError{StatusCode: 403, Body: `{"error":"denied"}`}); !strings.Contains(got, "denied") {
		t.Errorf("JSON 403 lost detail: %q", got)
	}
}
