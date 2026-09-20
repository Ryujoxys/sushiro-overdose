package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAppExitRejectsPendingWritesAndNativeExports(t *testing.T) {
	bridge := &DesktopBridge{}
	stops := 0
	quit := make(chan struct{}, 1)
	s := &appExitState{prepare: bridge.prepareClose, abort: bridge.abortClose, stop: func(context.Context) error { stops++; return nil }, quit: func() { quit <- struct{}{} }}
	request := func() int {
		w := httptest.NewRecorder()
		s.handleQuit(w, httptest.NewRequest(http.MethodPost, "/api/app/quit", nil))
		return w.Code
	}
	s.writes = 1
	if request() != 409 || stops != 0 {
		t.Fatal("quit interrupted an HTTP mutation")
	}
	s.writes = 0
	if err := bridge.beginWrite(); err != nil {
		t.Fatal(err)
	}
	if request() != 409 || stops != 0 {
		t.Fatal("quit interrupted native export")
	}
	bridge.endWrite()
	if request() != 200 || stops != 1 {
		t.Fatal("quit failed after writes finished")
	}
	select {
	case <-quit:
	case <-time.After(time.Second):
		t.Fatal("not closed")
	}
	if err := bridge.beginWrite(); err == nil {
		t.Fatal("accepted native write after close")
	}
	if request() != 409 || stops != 1 {
		t.Fatal("duplicate close accepted")
	}
	w := httptest.NewRecorder()
	s.middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("write handler executed while closing") })).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/queue/ticket", nil))
	if w.Code != 409 {
		t.Fatal(w.Code)
	}
}

func TestAppExitFailureLeavesWindowAndWritesAvailable(t *testing.T) {
	bridge := &DesktopBridge{}
	s := &appExitState{prepare: bridge.prepareClose, abort: bridge.abortClose, stop: func(context.Context) error { return errors.New("后台尚未退出") }, quit: func() { t.Error("closed on failure") }}
	w := httptest.NewRecorder()
	s.handleQuit(w, httptest.NewRequest(http.MethodPost, "/api/app/quit", nil))
	if w.Code != 409 || s.closing {
		t.Fatal(w.Code, s.closing)
	}
	if err := bridge.beginWrite(); err != nil {
		t.Fatal(err)
	}
	bridge.endWrite()
	w = httptest.NewRecorder()
	s.handleQuit(w, httptest.NewRequest(http.MethodGet, "/api/app/quit", nil))
	if w.Code != 405 {
		t.Fatal("GET allowed exit")
	}
}

func TestAppExitPreservesCSRFBoundary(t *testing.T) {
	stops := 0
	s := &appExitState{stop: func(context.Context) error { stops++; return nil }, quit: func() {}}
	old := getWebCSRFToken()
	setWebCSRFToken("exit-csrf-fixture")
	defer setWebCSRFToken(old)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/app/quit", s.handleQuit)
	handler := webSecurityMiddleware(s.middleware(mux))
	r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:39871/api/app/quit", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 403 || stops != 0 {
		t.Fatal("exit bypassed CSRF", w.Code, stops)
	}
}
