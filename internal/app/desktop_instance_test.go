package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDesktopSecondLaunchOnlyFocuses(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	first, err := claimDesktop(ctx, dir)
	if err != nil || first == nil {
		t.Fatalf("claim failed: %v", err)
	}
	defer first.close()
	server := httptest.NewServer(webSecurityMiddleware(http.HandlerFunc(first.activate)))
	defer server.Close()
	if err := first.publish(server.URL); err != nil {
		t.Fatal(err)
	}
	second, err := claimDesktop(ctx, dir)
	if err != nil || second != nil {
		t.Fatalf("second launch initialized another owner: %v %v", second, err)
	}
	select {
	case <-first.focus:
	default:
		t.Fatal("did not focus existing window")
	}
	first.close()
	if _, err := os.Stat(filepath.Join(dir, "desktop_session.json")); !os.IsNotExist(err) {
		t.Fatal("left activation token on disk")
	}
	next, err := claimDesktop(ctx, dir)
	if err != nil || next == nil {
		t.Fatalf("did not release instance lock: %v", err)
	}
	next.close()
}

func TestDesktopActivationRejectsBrowserAndWrongToken(t *testing.T) {
	d := &desktopInstance{token: newWebCSRFToken(), focus: make(chan struct{}, 1)}
	for _, tc := range []struct{ method, origin, token string }{
		{"POST", "", "wrong"}, {"GET", "", d.token}, {"POST", "http://external.example", d.token},
	} {
		req := httptest.NewRequest(tc.method, "http://127.0.0.1/desktop/activate", nil)
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("X-Sushiro-Desktop", tc.token)
		res := httptest.NewRecorder()
		d.activate(res, req)
		if res.Code != 403 {
			t.Fatalf("allowed activation: %+v", tc)
		}
	}
	if len(d.focus) != 0 {
		t.Fatal("untrusted request focused app")
	}
}

func TestDesktopInterruptedStartupCanRecover(t *testing.T) {
	dir := t.TempDir()
	first, err := claimDesktop(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer first.close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if second, err := claimDesktop(ctx, dir); err == nil || second != nil {
		t.Fatal("unready owner allowed another instance")
	}
	first.close()
	next, err := claimDesktop(context.Background(), dir)
	if err != nil || next == nil {
		t.Fatalf("stale instance blocked restart: %v", err)
	}
	next.close()
}
