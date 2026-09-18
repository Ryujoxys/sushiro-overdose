package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/Ryujoxys/sushiro-overdose/internal/core"
)

// DesktopBridge exposes only the actions used by the embedded, local UI.
// It does not change the loopback server's Host, Origin or CSRF checks.
type DesktopBridge struct {
	backend    *webBackend
	client     *http.Client
	saveDialog func(name string) (string, error)
	exportMu   sync.Mutex
	lifecycle  sync.Mutex
	writes     int
	closing    bool
}

type DesktopResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

var desktopMethods = map[string]string{
	"/api/status": "GET", "/api/records": "GET", "/api/records/settings": "GET POST",
	"/api/queue/service": "GET POST", "/api/queue/baseline": "GET POST",
	"/api/queue/stores": "GET", "/api/queue/live": "GET", "/api/queue/plan": "GET",
	"/api/queue/advisor": "GET", "/api/queue/ticket": "POST",
	"/api/queue/ticket/status": "GET", "/api/queue/ticket/cancel": "POST",
	"/api/mobile-auth": "GET", "/api/mobile-auth/start": "POST", "/api/mobile-auth/stop": "POST",
	"/api/engine/capture": "POST", "/api/engine/stop": "POST",
	"/api/auth/import": "POST", "/api/auth/reset": "POST", "/api/repair-proxy": "POST",
}

func desktopPath(raw string) (*url.URL, error) {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.IsAbs() || u.Host != "" || u.User != nil || u.Fragment != "" ||
		!strings.HasPrefix(raw, "/api/") || u.RawPath != "" || strings.ContainsAny(u.Path, "\\\r\n") ||
		path.Clean(u.Path) != u.Path {
		return nil, errors.New("不支持的本地请求")
	}
	return u, nil
}

func localHTTPClient() *http.Client {
	return &http.Client{
		Transport:     &http.Transport{Proxy: nil, DisableKeepAlives: true},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func (b *DesktopBridge) Request(method, resource, body string, timeoutMS int) (DesktopResponse, error) {
	u, err := desktopPath(resource)
	if err != nil {
		return DesktopResponse{}, err
	}
	if (method != "GET" && method != "POST") || !strings.Contains(" "+desktopMethods[u.Path]+" ", " "+method+" ") {
		return DesktopResponse{}, errors.New("不支持的本地操作")
	}
	if len(body) > 1<<20 || (method == "GET" && body != "") {
		return DesktopResponse{}, errors.New("请求内容过大或格式不正确")
	}
	if method == http.MethodPost {
		if err := b.beginWrite(); err != nil {
			return DesktopResponse{}, err
		}
		defer b.endWrite()
	}
	if timeoutMS < 1 || timeoutMS > 60000 {
		timeoutMS = 15000
	}
	status, data, err := b.request(method, u.String(), body, time.Duration(timeoutMS)*time.Millisecond, 16<<20)
	return DesktopResponse{Status: status, Body: string(data)}, err
}

func (b *DesktopBridge) beginWrite() error {
	b.lifecycle.Lock()
	defer b.lifecycle.Unlock()
	if b.closing {
		return errors.New("应用正在关闭，请重新打开后操作")
	}
	b.writes++
	return nil
}

func (b *DesktopBridge) endWrite() {
	b.lifecycle.Lock()
	b.writes--
	b.lifecycle.Unlock()
}

// Closing a native window must not cut off an explicitly submitted ticket or
// file export. Once closing is accepted, new writes are rejected atomically.
func (b *DesktopBridge) prepareClose() bool {
	b.lifecycle.Lock()
	defer b.lifecycle.Unlock()
	if b.writes != 0 {
		return false
	}
	b.closing = true
	return true
}

func (b *DesktopBridge) request(method, resource, body string, timeout time.Duration, limit int64) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(b.backend.ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, b.backend.URL+resource, strings.NewReader(body))
	if err != nil {
		return 0, nil, errors.New("无法创建本地请求")
	}
	req.Header.Set("Origin", b.backend.URL)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sushiro-CSRF", getWebCSRFToken())
	resp, err := b.client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, nil, errors.New("请求超时，请稍后重试")
		}
		return 0, nil, errors.New("本地服务未连接，请重新打开应用")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return 0, nil, errors.New("本地响应读取失败")
	}
	if int64(len(data)) > limit {
		return 0, nil, errors.New("数据过多，请缩小日期范围")
	}
	return resp.StatusCode, data, nil
}

// Export never accepts a destination path from JavaScript. Only the native
// save dialog can choose one, and cancellation must leave the filesystem alone.
func (b *DesktopBridge) Export(resource string) (bool, error) {
	u, err := desktopPath(resource)
	if err != nil {
		return false, err
	}
	name := ""
	switch u.Path {
	case "/api/records/export":
		name = "sushiro-records.jsonl"
	case "/api/diagnostics/bundle":
		if u.RawQuery != "" {
			return false, errors.New("不支持的导出参数")
		}
		name = "sushiro-diagnostics.zip"
	default:
		return false, errors.New("不支持的导出类型")
	}
	if err := b.beginWrite(); err != nil {
		return false, err
	}
	defer b.endWrite()
	if !b.exportMu.TryLock() {
		return false, errors.New("请先完成当前导出")
	}
	defer b.exportMu.Unlock()
	if b.saveDialog == nil {
		return false, errors.New("保存窗口尚未就绪")
	}
	target, err := b.saveDialog(name)
	if err != nil || target == "" {
		return false, err
	}
	status, data, err := b.request("GET", u.String(), "", 60*time.Second, 128<<20)
	if err != nil {
		return false, err
	}
	if status != http.StatusOK {
		return false, fmt.Errorf("导出失败（%d），请重试", status)
	}
	if err := b.backend.ctx.Err(); err != nil {
		return false, errors.New("应用已关闭，未保存导出文件")
	}
	if err := core.AtomicWriteFile(target, data, 0o600); err != nil {
		return false, errors.New("无法保存文件，请选择可写入的位置")
	}
	return true, nil
}
