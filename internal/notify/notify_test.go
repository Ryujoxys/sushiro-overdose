package notify

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// captureTransport 拦截 notifierClient 的请求，避免真打外部服务，并可指定返回状态码。
type captureTransport struct {
	mu     sync.Mutex
	status int
	last   *http.Request
	body   string
}

func (c *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.last = req
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		c.body = string(b)
	}
	st := c.status
	if st == 0 {
		st = http.StatusOK
	}
	return &http.Response{StatusCode: st, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
}

func withCapture(t *testing.T, status int) *captureTransport {
	t.Helper()
	ct := &captureTransport{status: status}
	old := notifierClient.Transport
	notifierClient.Transport = ct
	t.Cleanup(func() { notifierClient.Transport = old })
	return ct
}

func allRealNotifiers() []Notifier {
	return []Notifier{
		&feishuNotifier{webhook: "https://example.com/feishu"},
		&telegramNotifier{token: "123:abc", chatID: "-100"},
		&barkNotifier{url: "https://api.day.app", key: "k"},
		&serverChanNotifier{key: "SCT123"},
	}
}

func TestNotifiersSucceedOn2xx(t *testing.T) {
	ct := withCapture(t, http.StatusOK)
	for _, n := range allRealNotifiers() {
		if err := n.Send(context.Background(), "标题", "内容"); err != nil {
			t.Errorf("%s Send on 200 returned err: %v", n.Name(), err)
		}
		if ct.last == nil || ct.last.URL.String() == "" {
			t.Errorf("%s did not issue a request", n.Name())
		}
	}
}

func TestNotifiersErrorOn5xx(t *testing.T) {
	withCapture(t, http.StatusInternalServerError)
	for _, n := range allRealNotifiers() {
		if err := n.Send(context.Background(), "t", "c"); err == nil {
			t.Errorf("%s Send on 500 should return error", n.Name())
		}
	}
}

func TestFeishuPayloadShape(t *testing.T) {
	ct := withCapture(t, http.StatusOK)
	_ = (&feishuNotifier{webhook: "https://example.com/feishu"}).Send(context.Background(), "MYTITLE", "MYCONTENT")
	for _, want := range []string{"MYTITLE", "MYCONTENT", "interactive"} {
		if !strings.Contains(ct.body, want) {
			t.Errorf("feishu payload missing %q: %s", want, ct.body)
		}
	}
}

type fakeNotifier struct {
	name string
	mu   sync.Mutex
	got  int
	fail bool
}

func (f *fakeNotifier) Name() string { return f.name }
func (f *fakeNotifier) Send(ctx context.Context, title, content string) error {
	f.mu.Lock()
	f.got++
	f.mu.Unlock()
	if f.fail {
		return errors.New("boom")
	}
	return nil
}

func TestMultiNotifierFanOut(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	a := &fakeNotifier{name: "a"}
	b := &fakeNotifier{name: "b", fail: true}
	c := &fakeNotifier{name: "c"}
	mn := NewMultiNotifier(a, b)
	mn.Add(c)
	if len(mn.List()) != 3 {
		t.Fatalf("List len = %d, want 3", len(mn.List()))
	}
	// b 失败不应中断 a/c，也不应 panic
	mn.Send(context.Background(), "t", "c")
	if a.got != 1 || b.got != 1 || c.got != 1 {
		t.Errorf("fan-out counts a=%d b=%d c=%d, want 1/1/1", a.got, b.got, c.got)
	}
}

// TestMultiNotifierSendToChannels 验证 per-level 通道路由：channels 白名单非空时只发命中的渠道，
// 为空时全发（兼容老配置）。这是块 A 防止透传链静默失效的核心测试。
func TestMultiNotifierSendToChannels(t *testing.T) {
	feishu := &fakeNotifier{name: "feishu"}
	telegram := &fakeNotifier{name: "telegram"}
	bark := &fakeNotifier{name: "bark"}
	mn := NewMultiNotifier(feishu, telegram, bark)

	// 只发 bark
	mn.SendToChannels(context.Background(), "t", "c", []string{"bark"})
	if bark.got != 1 || feishu.got != 0 || telegram.got != 0 {
		t.Errorf("channels=[bark]: feishu=%d telegram=%d bark=%d, want 0/0/1", feishu.got, telegram.got, bark.got)
	}

	// 多选 feishu+bark
	mn.SendToChannels(context.Background(), "t", "c", []string{"feishu", "bark"})
	if feishu.got != 1 || bark.got != 2 || telegram.got != 0 {
		t.Errorf("channels=[feishu,bark]: feishu=%d telegram=%d bark=%d, want 1/0/2", feishu.got, telegram.got, bark.got)
	}

	// 空 channels = 全发（兼容老配置）
	mn.SendToChannels(context.Background(), "t", "c", nil)
	if feishu.got != 2 || telegram.got != 1 || bark.got != 3 {
		t.Errorf("channels=nil: feishu=%d telegram=%d bark=%d, want 2/1/3", feishu.got, telegram.got, bark.got)
	}

	// 白名单里没有的渠道名 = 谁都不发
	before := feishu.got + telegram.got + bark.got
	mn.SendToChannels(context.Background(), "t", "c", []string{"nonexistent"})
	if feishu.got+telegram.got+bark.got != before {
		t.Errorf("channels=[nonexistent] should send to no one, got extra sends")
	}
}
