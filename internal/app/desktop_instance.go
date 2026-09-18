package app

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/Ryujoxys/sushiro-overdose/internal/core"
	"github.com/Ryujoxys/sushiro-overdose/internal/platform"
)

type desktopSession struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type desktopInstance struct {
	lock  *platform.FileLock
	path  string
	token string
	focus chan struct{}
	once  sync.Once
}

// Claim before config migration, proxy cleanup or collector startup. A second
// launch may only ask the existing window to focus, never initialize the app.
func claimDesktop(ctx context.Context, dir string) (*desktopInstance, error) {
	sessionPath := filepath.Join(dir, "desktop_session.json")
	for {
		lock, err := platform.TryFileLock(filepath.Join(dir, "desktop.lock"))
		if err == nil {
			return &desktopInstance{lock: lock, path: sessionPath, token: newWebCSRFToken(), focus: make(chan struct{}, 1)}, nil
		}
		if !errors.Is(err, platform.ErrFileLocked) {
			return nil, fmt.Errorf("无法打开应用数据目录：%w", err)
		}
		if activateDesktop(ctx, sessionPath) {
			return nil, nil
		}
		select {
		case <-ctx.Done():
			return nil, errors.New("应用正在启动或退出，请稍后重新打开")
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (d *desktopInstance) publish(baseURL string) error {
	data, err := json.Marshal(desktopSession{URL: baseURL, Token: d.token})
	if err != nil {
		return err
	}
	return core.AtomicWriteFile(d.path, data, 0o600)
}

func (d *desktopInstance) close() {
	d.once.Do(func() {
		_ = os.Remove(d.path)
		d.lock.Close()
	})
}

func (d *desktopInstance) activate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.Header.Get("Origin") != "" ||
		subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Sushiro-Desktop")), []byte(d.token)) != 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	select {
	case d.focus <- struct{}{}:
	default:
	}
	w.WriteHeader(http.StatusNoContent)
}

func activateDesktop(parent context.Context, sessionPath string) bool {
	data, err := os.ReadFile(sessionPath)
	if err != nil || len(data) > 4096 {
		return false
	}
	var session desktopSession
	if json.Unmarshal(data, &session) != nil || len(session.Token) != 43 {
		return false
	}
	u, err := url.Parse(session.URL)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return false
	}
	ctx, cancel := context.WithTimeout(parent, 500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, session.URL+"/desktop/activate", nil)
	if err != nil {
		return false
	}
	req.Header.Set("X-Sushiro-Desktop", session.Token)
	resp, err := localHTTPClient().Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusNoContent
}
