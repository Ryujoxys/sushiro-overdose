package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

// probeClientFor answers the three read-only probe endpoints in-process; nothing leaves the machine.
func probeClientFor(t *testing.T, publicStatus, reservationsStatus int, reservationsBody string) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		code, body := http.StatusOK, "{}"
		switch r.URL.Path {
		case "/wechat/api/2.0/getStoreById":
			code, body = publicStatus, `{"id":1012,"name":"南山店"}`
		case "/wechat/api/2.0/store/timeslots":
			code, body = publicStatus, `[]`
		case "/wechat/api_auth/2.0/ticketing/getReservations":
			code, body = reservationsStatus, reservationsBody
		default:
			t.Fatalf("probe reached an unexpected endpoint: %s %s", r.Method, r.URL.Path)
		}
		if code >= 400 && body != reservationsBody {
			body = `{"message":"unauthorized"}`
		}
		return &http.Response{StatusCode: code, Status: http.StatusText(code), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
}

func TestPublicStoreRejectionDoesNotMarkCredentialsStale(t *testing.T) {
	for _, publicStatus := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		reliabilityHome(t)
		resetAuthHealth()

		// Account endpoint works: verified, healthy, and the public rejections are only "skipped".
		report := runAuthProbeWithClient(context.Background(), "", reliabilityTokens(), UserPreferences{}, probeClientFor(t, publicStatus, http.StatusOK, `[]`))
		if !report.OK || !report.Authenticated || authProbeRejected(report) {
			t.Fatalf("public %d with working account: %+v", publicStatus, report)
		}
		for _, result := range report.Results[:2] {
			if !result.Skipped || result.OK || !strings.Contains(result.Detail, "不用来判断账号是否可用") || strings.Contains(result.Detail, "HTTP") {
				t.Fatalf("public result not skipped plainly: %+v", result)
			}
		}
		applyAuthProbeHealth(report)
		if getAuthHealth().Status != authHealthOK {
			t.Fatalf("health = %s, want ok", getAuthHealth().Status)
		}

		// Nothing could be verified (account endpoint gone): not passed, but not stale either.
		resetAuthHealth()
		report = runAuthProbeWithClient(context.Background(), "", reliabilityTokens(), UserPreferences{}, probeClientFor(t, publicStatus, http.StatusNotFound, `{"error":"Not Found"}`))
		if report.OK || report.Authenticated || authProbeRejected(report) {
			t.Fatalf("nothing verified: %+v", report)
		}
		if len(report.Advice) == 0 || !strings.Contains(strings.Join(report.Advice, ""), "没有被标成失效") {
			t.Fatalf("advice = %v", report.Advice)
		}
		applyAuthProbeHealth(report)
		if getAuthHealth().Status == authHealthStale {
			t.Fatal("public rejection marked credentials stale")
		}

		// The account endpoint itself rejecting the credentials still marks them stale.
		report = runAuthProbeWithClient(context.Background(), "", reliabilityTokens(), UserPreferences{}, probeClientFor(t, publicStatus, http.StatusUnauthorized, `{"error":"expired"}`))
		if report.OK || report.Authenticated || !authProbeRejected(report) {
			t.Fatalf("account rejection: %+v", report)
		}
		applyAuthProbeHealth(report)
		if getAuthHealth().Status != authHealthStale {
			t.Fatal("account rejection not marked stale")
		}
	}
}

func TestAuthProbeRejectionIgnoresOnlyPublicPaths(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/wechat/api/2.0/getStoreById?storeId=1012", false},
		{"/wechat/api/2.0/store/timeslots", false},
		{"/wechat/api_auth/2.0/ticketing/getReservations", true},
		{"", true}, // unknown origin stays conservative
	}
	for _, c := range cases {
		report := AuthProbeReport{Results: []AuthProbeResult{{Path: c.path, Status: http.StatusForbidden}}}
		if got := authProbeRejected(report); got != c.want {
			t.Errorf("authProbeRejected(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}
