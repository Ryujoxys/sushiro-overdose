package notify

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// notifierClient is shared by all notifiers with a 10s timeout.
var notifierClient = &http.Client{Timeout: 10 * time.Second}

// Notifier sends a notification with a title and markdown content.
type Notifier interface {
	Send(ctx context.Context, title, content string) error
	Name() string
}

// SessionNotifier 是可选能力：同一 sessionKey 的连续通知「原地更新」同一条消息，
// 而不是堆叠多条——用于叫号进度卡（「还差 12 桌 → 5 桌 → 1 桌」更新同一张卡）。
// 不实现该接口的渠道由 MultiNotifier 回退到普通 Send。
type SessionNotifier interface {
	SendSession(ctx context.Context, sessionKey, title, content string) error
}

// MultiNotifier fans out notifications to multiple channels.
type MultiNotifier struct {
	mu        sync.Mutex
	notifiers []Notifier
}

func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

// Send 把通知扇出到所有已注册渠道：先快照 notifiers（持锁时间最短），再对每个渠道起一个协程
// 并发发送，wg.Wait() 等全部发完才返回。单个渠道失败只记日志、不影响其他渠道——所以 Send 没有
// 返回值，调用方不需要知道哪个渠道挂了。
func (m *MultiNotifier) Send(ctx context.Context, title, content string) {
	m.mu.Lock()
	snapshot := make([]Notifier, len(m.notifiers))
	copy(snapshot, m.notifiers)
	m.mu.Unlock()

	var wg sync.WaitGroup
	for _, n := range snapshot {
		wg.Add(1)
		go func(n Notifier) {
			defer wg.Done()
			if err := n.Send(ctx, title, content); err != nil {
				LogMessage(time.Now(), fmt.Sprintf("[%s] 通知发送失败: %s", n.Name(), err))
			}
		}(n)
	}
	wg.Wait()
}

// SendSession 扇出「会话内可原地更新」的通知：实现了 SessionNotifier 的渠道（如 Telegram）
// 用 sessionKey 更新同一条消息；其余渠道回退到普通 Send（仍会逐条推送）。
func (m *MultiNotifier) SendSession(ctx context.Context, sessionKey, title, content string) {
	m.mu.Lock()
	snapshot := make([]Notifier, len(m.notifiers))
	copy(snapshot, m.notifiers)
	m.mu.Unlock()

	var wg sync.WaitGroup
	for _, n := range snapshot {
		wg.Add(1)
		go func(n Notifier) {
			defer wg.Done()
			var err error
			if sn, ok := n.(SessionNotifier); ok && sessionKey != "" {
				err = sn.SendSession(ctx, sessionKey, title, content)
			} else {
				err = n.Send(ctx, title, content)
			}
			if err != nil {
				LogMessage(time.Now(), fmt.Sprintf("[%s] 通知发送失败: %s", n.Name(), err))
			}
		}(n)
	}
	wg.Wait()
}

func (m *MultiNotifier) Add(n Notifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifiers = append(m.notifiers, n)
}

func (m *MultiNotifier) List() []Notifier {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Notifier, len(m.notifiers))
	copy(out, m.notifiers)
	return out
}

// channelAllowed 判断名为 name 的渠道是否在白名单 channels 内。
// channels 为空 = 不限（全发），与老配置（无 channels 字段）语义一致。
func channelAllowed(name string, channels []string) bool {
	if len(channels) == 0 {
		return true
	}
	for _, c := range channels {
		if c == name {
			return true
		}
	}
	return false
}

// SendToChannels 同 Send，但只扇出到 channels 白名单内的渠道（按 Notifier.Name() 匹配）。
// channels 为空时退化为全发（兼容老配置）。
func (m *MultiNotifier) SendToChannels(ctx context.Context, title, content string, channels []string) {
	m.mu.Lock()
	snapshot := make([]Notifier, len(m.notifiers))
	copy(snapshot, m.notifiers)
	m.mu.Unlock()

	var wg sync.WaitGroup
	for _, n := range snapshot {
		if !channelAllowed(n.Name(), channels) {
			continue
		}
		wg.Add(1)
		go func(n Notifier) {
			defer wg.Done()
			if err := n.Send(ctx, title, content); err != nil {
				LogMessage(time.Now(), fmt.Sprintf("[%s] 通知发送失败: %s", n.Name(), err))
			}
		}(n)
	}
	wg.Wait()
}

// SendSessionToChannels 同 SendSession，但只扇出到 channels 白名单内的渠道。
func (m *MultiNotifier) SendSessionToChannels(ctx context.Context, sessionKey, title, content string, channels []string) {
	m.mu.Lock()
	snapshot := make([]Notifier, len(m.notifiers))
	copy(snapshot, m.notifiers)
	m.mu.Unlock()

	var wg sync.WaitGroup
	for _, n := range snapshot {
		if !channelAllowed(n.Name(), channels) {
			continue
		}
		wg.Add(1)
		go func(n Notifier) {
			defer wg.Done()
			var err error
			if sn, ok := n.(SessionNotifier); ok && sessionKey != "" {
				err = sn.SendSession(ctx, sessionKey, title, content)
			} else {
				err = n.Send(ctx, title, content)
			}
			if err != nil {
				LogMessage(time.Now(), fmt.Sprintf("[%s] 通知发送失败: %s", n.Name(), err))
			}
		}(n)
	}
	wg.Wait()
}

// ---- Notification config ----

type NotifyConfig struct {
	Feishu struct {
		Webhook string `json:"webhook"`
	} `json:"feishu"`
	Telegram struct {
		Token  string `json:"token"`
		ChatID string `json:"chat_id"`
	} `json:"telegram"`
	Bark struct {
		URL string `json:"url"`
		Key string `json:"key"`
	} `json:"bark"`
	ServerChan struct {
		Key string `json:"key"`
	} `json:"server_chan"`
}

func NotifyConfigPath() string {
	return fmt.Sprintf("%s/notify.json", AppDirPath())
}

func LoadNotifyConfig() (*NotifyConfig, error) {
	data, err := os.ReadFile(NotifyConfigPath())
	if err != nil {
		return nil, err
	}
	var cfg NotifyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveNotifyConfig(cfg *NotifyConfig) error {
	os.MkdirAll(AppDirPath(), 0o755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(NotifyConfigPath(), data, 0o600)
}

// BuildNotifierFromConfig creates a MultiNotifier from saved config.
// BuildNotifierFromConfig 按 notify.json 构造多渠道通知器：飞书/Telegram/Bark/ServerChan 各自
// 配置齐全才加入。容错：读不到新配置文件时回退尝试旧的飞书单渠道配置（历史迁移），
// 保证老用户升级后通知不突然哑掉。
func BuildNotifierFromConfig() *MultiNotifier {
	mn := &MultiNotifier{}

	cfg, err := LoadNotifyConfig()
	if err != nil {
		// Try loading legacy feishu config
		if webhook := LoadFeishuConfig(); webhook != "" {
			mn.Add(&feishuNotifier{webhook: webhook})
		}
		return mn
	}

	if cfg.Feishu.Webhook != "" {
		mn.Add(&feishuNotifier{webhook: cfg.Feishu.Webhook})
	}
	if cfg.Telegram.Token != "" && cfg.Telegram.ChatID != "" {
		mn.Add(&telegramNotifier{token: cfg.Telegram.Token, chatID: cfg.Telegram.ChatID})
	}
	if cfg.Bark.URL != "" && cfg.Bark.Key != "" {
		mn.Add(&barkNotifier{url: cfg.Bark.URL, key: cfg.Bark.Key})
	}
	if cfg.ServerChan.Key != "" {
		mn.Add(&serverChanNotifier{key: cfg.ServerChan.Key})
	}

	return mn
}
