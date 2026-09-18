package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TargetSlot represents a timeslot the user wants to book.
type TargetSlot struct {
	StoreID string
	Date    string // compact YYYYMMDD
	Start   string // compact HHMMSS
	End     string // compact HHMMSS
}

// CapturedTokens holds auth parameters intercepted from WeChat mini-program traffic.
// 这是抓包链路（capture）与运行链路（Settings）之间的中间结构：抓包器边收边填，
// 凑齐后再通过 ToSettings 转成可跑预约的 Settings。
//
// 并发约定：mu 保护所有字段。抓包器（写）和 UI/状态查询（读）可能并发访问，
// 因此所有公开方法要么自己加锁，要么调 IsCompleteUnlocked/MissingFields 等已加锁版本。
// 不持有锁的代码禁止直接读写字段。
//
// 字段语义：
//   - QueryAuth / ReservationAuth：与 Settings 的 QueryAuthorization/ReservationAuth 同义，
//     查询类 vs 写操作的两组令牌，可能相同也可能不同。
//   - StoreIDs：抓包过程中观察到的门店，会被回填到本地配置；空表示还没观察到任何门店。
type CapturedTokens struct {
	mu              sync.Mutex
	XAppCode        string
	QueryAuth       string
	ReservationAuth string
	UserAgent       string
	Referer         string
	XAppClient      string
	WechatID        string
	PhoneNumber     string
	StoreIDs        []string
	FeishuWebhook   string
}

func NewCapturedTokens() *CapturedTokens {
	return &CapturedTokens{}
}

func (t *CapturedTokens) IsComplete() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.IsCompleteUnlocked()
}

// IsCompleteUnlocked checks completeness without locking (caller must hold lock).
// 「完整」= 查询 + 预约两套接口所需的字段都已就位，可进入运行流程。
// 注意 FeishuWebhook 不在这里：它是可选的通知渠道，不强制。
func (t *CapturedTokens) IsCompleteUnlocked() bool {
	return t.XAppCode != "" &&
		t.QueryAuth != "" &&
		t.ReservationAuth != "" &&
		t.UserAgent != "" &&
		t.Referer != "" &&
		t.WechatID != "" &&
		t.PhoneNumber != "" &&
		len(t.StoreIDs) > 0
}

// Status 返回各字段的就绪状态（带掩码预览），供 UI 渲染抓包进度。读取加锁。
func (t *CapturedTokens) Status() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	check := func(v string) string {
		if v != "" {
			return "✅"
		}
		return "⏳"
	}
	return []string{
		fmt.Sprintf("  X-App-Code:       %s %s", check(t.XAppCode), maskToken(t.XAppCode)),
		fmt.Sprintf("  Query Auth:       %s %s", check(t.QueryAuth), maskToken(t.QueryAuth)),
		fmt.Sprintf("  Reservation Auth: %s %s", check(t.ReservationAuth), maskToken(t.ReservationAuth)),
		fmt.Sprintf("  User-Agent:       %s %s", check(t.UserAgent), maskToken(t.UserAgent)),
		fmt.Sprintf("  Referer:          %s %s", check(t.Referer), maskToken(t.Referer)),
		fmt.Sprintf("  Wechat ID:        %s %s", check(t.WechatID), maskToken(t.WechatID)),
		fmt.Sprintf("  Phone Number:     %s %s", check(t.PhoneNumber), MaskPhone(t.PhoneNumber)),
		fmt.Sprintf("  Store IDs:        %s %v", check(strings.Join(t.StoreIDs, ",")), t.StoreIDs),
	}
}

// maskToken 给 token 做掩码展示：短于等于 12 字符全掩成 ***，长的保留前 8 位。
// 用于 Status 里展示进度而不泄露完整凭证。
func maskToken(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 12 {
		return "***"
	}
	return v[:8] + "..."
}

// LocalConfigPath 返回抓包凭证落盘路径（~/.sushiro/config.json）。
// 这是抓包流程保存凭证的位置，与 config.go 里 LoadSettings 读的用户 config.json 是不同文件，
// 不要混淆——后者是用户手填的运行配置，前者是自动捕获的。
func LocalConfigPath() string {
	return filepath.Join(AppDirPath(), "config.json")
}

// MigrateOldConfig moves the old CWD-based config to ~/.sushiro/ if it exists.
// 历史迁移：早期版本把抓包配置写在当前目录的 .sushiro_local.json，现在统一迁到用户主目录。
// 仅当目标不存在时迁移，避免覆盖新位置已有的更新配置。
func MigrateOldConfig() {
	if HasCustomDataHome() {
		return
	}
	oldPath := ".sushiro_local.json"
	if _, err := os.Stat(oldPath); err != nil {
		return
	}
	newPath := LocalConfigPath()
	if _, err := os.Stat(newPath); err == nil {
		return
	}
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return
	}
	os.MkdirAll(AppDirPath(), 0o755)
	if AtomicWriteFile(newPath, data, 0o600) == nil {
		os.Remove(oldPath)
		LogMessage(time.Now(), "配置已迁移到 "+newPath)
	}
}

// localConfigJSON 是 ~/.sushiro/config.json（抓包凭证）的磁盘格式。
// 注意它不存 FeishuWebhook——飞书 webhook 单独存 feishu.json，使其在凭证刷新时不丢。
type localConfigJSON struct {
	XAppCode        string   `json:"x_app_code"`
	QueryAuth       string   `json:"query_authorization"`
	ReservationAuth string   `json:"reservation_authorization"`
	UserAgent       string   `json:"user_agent"`
	Referer         string   `json:"referer"`
	XAppClient      string   `json:"x_app_client"`
	WechatID        string   `json:"wechat_id"`
	PhoneNumber     string   `json:"phone_number"`
	StoreIDs        []string `json:"store_ids"`
}

// SaveLocalConfig 把抓包凭证原子地写回 ~/.sushiro/config.json。全程持锁，保证读到的是一致快照。
// 文件权限 0600：里面是真实凭证，禁止其它用户读。
func SaveLocalConfig(tokens *CapturedTokens) error {
	tokens.mu.Lock()
	defer tokens.mu.Unlock()
	data := localConfigJSON{
		XAppCode:        tokens.XAppCode,
		QueryAuth:       tokens.QueryAuth,
		ReservationAuth: tokens.ReservationAuth,
		UserAgent:       EffectiveUserAgent(tokens.UserAgent),
		Referer:         tokens.Referer,
		XAppClient:      tokens.XAppClient,
		WechatID:        tokens.WechatID,
		PhoneNumber:     tokens.PhoneNumber,
		StoreIDs:        tokens.StoreIDs,
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll(AppDirPath(), 0o755)
	// 凭证是核心配置，原子写避免并发读读到半截 JSON 被判为「无配置」。
	return AtomicWriteFile(LocalConfigPath(), raw, 0o600)
}

// LoadLocalConfig 从 ~/.sushiro/config.json 还原 CapturedTokens。
// FeishuWebhook 单独从 feishu.json 加载后并入，避免凭证刷新覆盖飞书配置。
func LoadLocalConfig() (*CapturedTokens, error) {
	raw, err := os.ReadFile(LocalConfigPath())
	if err != nil {
		return nil, err
	}
	var data localConfigJSON
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return &CapturedTokens{
		XAppCode:        data.XAppCode,
		QueryAuth:       data.QueryAuth,
		ReservationAuth: data.ReservationAuth,
		UserAgent:       EffectiveUserAgent(data.UserAgent),
		Referer:         data.Referer,
		XAppClient:      data.XAppClient,
		WechatID:        data.WechatID,
		PhoneNumber:     data.PhoneNumber,
		StoreIDs:        data.StoreIDs,
		FeishuWebhook:   LoadFeishuConfig(),
	}, nil
}

// ValidateForQuery 检查「仅查询」所需字段是否齐全（不需要预约那套额外字段）。
func (t *CapturedTokens) ValidateForQuery() error {
	missing := t.MissingFields(false)
	if len(missing) > 0 {
		return fmt.Errorf("凭证参数不完整，缺少: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateForReservation 检查「下预约」所需字段是否齐全，比查询多了预约凭证/微信ID/手机号。
func (t *CapturedTokens) ValidateForReservation() error {
	missing := t.MissingFields(true)
	if len(missing) > 0 {
		return fmt.Errorf("预约参数不完整，缺少: %s", strings.Join(missing, ", "))
	}
	return nil
}

// MissingFields 返回当前缺失的字段名（中文标签）。reservation=true 时把预约所需的额外字段
// （预约凭证/微信ID/手机号）也算进去；false 时只看查询所需。
func (t *CapturedTokens) MissingFields(reservation bool) []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	fields := []struct {
		name  string
		value string
	}{
		{"X-App-Code", t.XAppCode},
		{"查询凭证", t.QueryAuth},
		{"User-Agent", t.UserAgent},
		{"Referer", t.Referer},
	}
	if reservation {
		fields = append(fields,
			struct {
				name  string
				value string
			}{"预约凭证", t.ReservationAuth},
			struct {
				name  string
				value string
			}{"微信ID", t.WechatID},
			struct {
				name  string
				value string
			}{"手机号", t.PhoneNumber},
		)
	}

	var missing []string
	for _, f := range fields {
		if strings.TrimSpace(f.value) == "" {
			missing = append(missing, f.name)
		}
	}
	if len(t.StoreIDs) == 0 {
		missing = append(missing, "门店")
	}
	return missing
}

// DeleteLocalConfig 删除抓包凭证文件（注销场景）。缺失不报错。
func DeleteLocalConfig() {
	os.Remove(LocalConfigPath())
}

// FeishuConfigPath 返回飞书 webhook 的独立存储路径（~/.sushiro/feishu.json）。
// 单独存是为了让它独立于凭证刷新——凭证文件被覆盖/删除时飞书配置不受影响。
func FeishuConfigPath() string {
	return filepath.Join(AppDirPath(), "feishu.json")
}

func LoadFeishuConfig() string {
	data, err := os.ReadFile(FeishuConfigPath())
	if err != nil {
		return ""
	}
	var cfg struct {
		Webhook string `json:"webhook"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return ""
	}
	return strings.TrimSpace(cfg.Webhook)
}

func SaveFeishuConfig(webhook string) {
	os.MkdirAll(AppDirPath(), 0o755)
	cfg := struct {
		Webhook string `json:"webhook"`
	}{Webhook: strings.TrimSpace(webhook)}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	_ = AtomicWriteFile(FeishuConfigPath(), data, 0o600)
}

// ToSettings 把抓包凭证转成可跑预约的 Settings，偏好从默认存储加载。
func (t *CapturedTokens) ToSettings() Settings {
	return t.ToSettingsWithPrefs(LoadPreferences())
}

// ToSettingsWithPrefs 把抓包凭证 + 用户偏好合并成 Settings。读字段时全程持锁。
// 优先级约定（关键，避免误以为是 bug）：
//   - 门店列表：偏好里显式选的 SelectedStores 优先于抓包观察到的 StoreIDs；
//     抓包只是「看到过」，用户在偏好里的选择才是「我想约的」。
//   - 手机号/微信ID：偏好里预填的优先于抓包值——方便取号前提前配置好。
//   - 成人/桌型：偏好未配则用默认值（成人 2、桌型 T）。
//
// 时区固定 Asia/Shanghai（寿司大陆门店），与 config.go 默认一致。
func (t *CapturedTokens) ToSettingsWithPrefs(prefs UserPreferences) Settings {
	t.mu.Lock()
	defer t.mu.Unlock()

	timezone := "Asia/Shanghai"
	location := SushiroTimezone

	storeIDs := t.StoreIDs
	if len(prefs.SelectedStores) > 0 {
		storeIDs = prefs.SelectedStores
	}

	adult := prefs.Adult
	if adult <= 0 {
		adult = 2
	}

	tableType := prefs.TableType
	if tableType == "" {
		tableType = "T"
	}

	// 偏好里预填的手机号/微信ID 优先于捕获值，方便取号前提前配置。
	phoneNumber := t.PhoneNumber
	if v := NormalizePreferencePhoneNumber(prefs.PhoneNumber); v != "" {
		phoneNumber = v
	}
	wechatID := t.WechatID
	if v := strings.TrimSpace(prefs.WechatID); v != "" {
		wechatID = v
	}

	return Settings{
		StoreIDs:           storeIDs,
		Adult:              adult,
		Child:              prefs.Child,
		TableType:          tableType,
		Debug:              true,
		PhoneNumber:        phoneNumber,
		WechatID:           wechatID,
		XAppCode:           t.XAppCode,
		QueryAuthorization: t.QueryAuth,
		ReservationAuth:    t.ReservationAuth,
		XAppClient:         DefaultString(t.XAppClient, "miniapp"),
		UserAgent:          EffectiveUserAgent(t.UserAgent),
		Referer:            t.Referer,
		StateFile:          StateFilePath(),
		Timezone:           timezone,
		Location:           location,
		PollInterval:       60 * time.Second,
		AvailableStatuses:  map[string]struct{}{"AVAILABLE": {}},
		BaseURL:            "https://crm-cn-prd.sushiro.com.cn",
		FeishuWebhook:      t.FeishuWebhook,
	}
}
