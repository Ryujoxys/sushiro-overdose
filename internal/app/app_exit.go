package app

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type appExitState struct {
	mu       sync.Mutex
	writes   int
	closing  bool
	finished bool
	prepare  func() bool
	abort    func()
	stop     func(context.Context) error
	quit     func()
}

func (s *appExitState) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutation := r.Method != http.MethodGet && r.Method != http.MethodHead
		export := r.URL.Path == "/api/records/export" || r.URL.Path == "/api/diagnostics/bundle"
		if r.URL.Path == "/api/app/quit" || (!mutation && !export) {
			next.ServeHTTP(w, r)
			return
		}
		s.mu.Lock()
		if s.closing {
			s.mu.Unlock()
			writeError(w, http.StatusConflict, "应用正在退出，请稍后重新打开")
			return
		}
		s.writes++
		s.mu.Unlock()
		defer func() { s.mu.Lock(); s.writes--; s.mu.Unlock() }()
		next.ServeHTTP(w, r)
	})
}

func (s *appExitState) handleQuit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	s.mu.Lock()
	if s.closing || s.writes != 0 || (s.prepare != nil && !s.prepare()) {
		s.mu.Unlock()
		writeError(w, http.StatusConflict, "正在完成操作，请稍后再退出")
		return
	}
	s.closing = true
	stop, quit := s.stop, s.quit
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	err := stop(ctx)
	cancel()
	if err != nil {
		s.mu.Lock()
		s.closing = false
		if s.abort != nil {
			s.abort()
		}
		s.mu.Unlock()
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	s.mu.Lock()
	s.finished = true
	s.mu.Unlock()
	writeJSON(w, map[string]bool{"exiting": true})
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	go quit()
}
