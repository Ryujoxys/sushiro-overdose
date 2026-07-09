package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

const (
	cloudAuthConfigFile      = "cloud_auth.json"
	cloudAuthURLEnv          = "SUSHIRO_CLOUD_URL"
	cloudAuthSessionTokenEnv = "SUSHIRO_CLOUD_SESSION_TOKEN"
	// 默认用 Cloudflare workers.dev，避免自定义域名到期后线上基准/GitHub 登录全挂。
	// Turso / GitHub OAuth 密钥只在 Worker secrets，不进本机、不进源码。
	defaultCloudAuthBaseURL       = "https://sushiro-cloud.sushiro-ryujoxys.workers.dev"
	cloudAuthBaselineProbeStoreID = "3006" // 健康探测固定用的门店号，只验证线上基准接口能否返回数据
	cloudAuthTimeout              = 12 * time.Second
	cloudOAuthStateTTL            = 10 * time.Minute // OAuth state 有效期：留出用户在浏览器走完 GitHub 登录的窗口，又不至于无限堆积
	cloudAuthSessionTokenSize     = 32               // 生成 state 时的随机字节数（base64 后约 43 字符），足够防爆破
)

// CloudAuthConfig 是 cloud_auth.json 的持久化结构：GitHub 登录后存本机，用于读取线上排队基准。
// 安全语义：SessionToken 是登录云端（Cloudflare Worker）的会话令牌，仅用于读取公开排队数据，
// 数据库密钥不会落到本机；omitempty 让未登录时不落盘敏感字段。所有时间字段均为 RFC3339 字符串。
type CloudAuthConfig struct {
	BaseURL        string `json:"base_url"`                   // 云端 Worker 地址
	SessionToken   string `json:"session_token,omitempty"`    // Bearer 令牌，请求 /api/me 等接口时带
	UserLogin      string `json:"user_login,omitempty"`       // GitHub login（展示用）
	UserName       string `json:"user_name,omitempty"`        // GitHub 昵称（展示用）
	AvatarURL      string `json:"avatar_url,omitempty"`       // 头像地址（展示用）
	ConnectedAt    string `json:"connected_at,omitempty"`     // 首次连上的时间
	ExpiresAt      string `json:"expires_at,omitempty"`       // 云端会话过期时间（由云端返回）
	LastVerifiedAt string `json:"last_verified_at,omitempty"` // 上次本地校验通过的时间
}

type CloudAuthStatus struct {
	Configured          bool   `json:"configured"`
	Connected           bool   `json:"connected"`
	BaseURL             string `json:"base_url,omitempty"`
	UserLogin           string `json:"user_login,omitempty"`
	UserName            string `json:"user_name,omitempty"`
	AvatarURL           string `json:"avatar_url,omitempty"`
	ConnectedAt         string `json:"connected_at,omitempty"`
	ExpiresAt           string `json:"expires_at,omitempty"`
	LastVerifiedAt      string `json:"last_verified_at,omitempty"`
	SessionFromEnv      bool   `json:"session_from_env,omitempty"`
	BaselineConnected   bool   `json:"baseline_connected,omitempty"`
	BaselineStoreCount  int    `json:"baseline_store_count,omitempty"`
	BaselineLatestCount int    `json:"baseline_latest_count,omitempty"`
	BaselineRollupCount int    `json:"baseline_rollup_count,omitempty"`
	BaselineUpdatedAt   string `json:"baseline_updated_at,omitempty"`
	LastError           string `json:"last_error,omitempty"`
	ProviderMessage     string `json:"provider_message,omitempty"`
}

type cloudMeResponse struct {
	OK      bool            `json:"ok"`
	User    cloudUser       `json:"user"`
	Session cloudSession    `json:"session"`
	Error   string          `json:"error,omitempty"`
	Meta    json.RawMessage `json:"meta,omitempty"`
}

type cloudUser struct {
	ID        int64  `json:"id,omitempty"`
	Login     string `json:"login,omitempty"`
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type cloudSession struct {
	ExpiresAt string `json:"expires_at,omitempty"`
}

// cloudOAuthStates 保存 OAuth 回调的 state 一次性票据：值=过期时间，TTL=cloudOAuthStateTTL。
// 防御 CSRF：state 由本机生成随机串，随授权 URL 带给 GitHub，回调时必须原样带回并被本机
// 单次消费（consumeCloudOAuthState 命中即删），防止重放。同时 newCloudOAuthState 顺带清扫
// 过期项，避免 map 长期堆积（无独立 GC 线程）。
var cloudOAuthStates struct {
	sync.Mutex
	values map[string]time.Time
}

func cloudAuthConfigPath() string {
	return filepath.Join(AppDirPath(), cloudAuthConfigFile)
}

// LoadCloudAuthConfig 从 cloud_auth.json 读配置，再用环境变量覆盖。优先级坑：环境变量
// SUSHIRO_CLOUD_URL / SUSHIRO_CLOUD_SESSION_TOKEN 一旦设置就盖过磁盘值（用于 CI/容器注入），
// 因此写盘的值在 env 存在时会被忽略——排查"改了配置不生效"先看环境变量。最后对每个字段做 TrimSpace 兜底。
func LoadCloudAuthConfig() CloudAuthConfig {
	var cfg CloudAuthConfig
	if data, err := os.ReadFile(cloudAuthConfigPath()); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	if value := strings.TrimSpace(os.Getenv(cloudAuthURLEnv)); value != "" {
		cfg.BaseURL = value
	}
	if value := strings.TrimSpace(os.Getenv(cloudAuthSessionTokenEnv)); value != "" {
		cfg.SessionToken = value
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = defaultCloudAuthBaseURL
	}
	cfg.BaseURL = normalizeCloudBaseURL(cfg.BaseURL)
	cfg.SessionToken = strings.TrimSpace(cfg.SessionToken)
	cfg.UserLogin = strings.TrimSpace(cfg.UserLogin)
	cfg.UserName = strings.TrimSpace(cfg.UserName)
	cfg.AvatarURL = strings.TrimSpace(cfg.AvatarURL)
	cfg.ConnectedAt = strings.TrimSpace(cfg.ConnectedAt)
	cfg.ExpiresAt = strings.TrimSpace(cfg.ExpiresAt)
	cfg.LastVerifiedAt = strings.TrimSpace(cfg.LastVerifiedAt)
	return cfg
}

// SaveCloudAuthConfig 写回 cloud_auth.json。写盘前重新 normalize + 校验 URL，
// 并用 0o600 权限（仅属主可读写）落盘，避免会话令牌被同机其他用户读到。
func SaveCloudAuthConfig(cfg CloudAuthConfig) error {
	cfg.BaseURL = normalizeCloudBaseURL(cfg.BaseURL)
	cfg.SessionToken = strings.TrimSpace(cfg.SessionToken)
	cfg.UserLogin = strings.TrimSpace(cfg.UserLogin)
	cfg.UserName = strings.TrimSpace(cfg.UserName)
	cfg.AvatarURL = strings.TrimSpace(cfg.AvatarURL)
	cfg.ConnectedAt = strings.TrimSpace(cfg.ConnectedAt)
	cfg.ExpiresAt = strings.TrimSpace(cfg.ExpiresAt)
	cfg.LastVerifiedAt = strings.TrimSpace(cfg.LastVerifiedAt)
	if err := validateCloudBaseURL(cfg.BaseURL, true); err != nil {
		return err
	}
	if err := os.MkdirAll(AppDirPath(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cloudAuthConfigPath(), data, 0o600)
}

// normalizeCloudBaseURL 把 BaseURL 规整成无尾斜杠、无 query/fragment 的形式，
// 保证后续拼 endpoint 时 base + path 不会出现 // 或带脏参数。
func normalizeCloudBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := neturl.Parse(raw)
	if err != nil {
		return strings.TrimRight(raw, "/")
	}
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/")
}

// validateCloudBaseURL 校验云端地址合法：必须有 scheme+host 且只允许 http/https。
// allowEmpty=true 时允许空串（用于"未连接"的中间态），否则空串报错。
func validateCloudBaseURL(raw string, allowEmpty bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if allowEmpty {
			return nil
		}
		return fmt.Errorf("缺少 Cloudflare Worker URL")
	}
	u, err := neturl.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("Cloudflare Worker URL 无效")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("Cloudflare Worker URL 只支持 http/https")
	}
	return nil
}

func (cfg CloudAuthConfig) configured() bool {
	return strings.TrimSpace(cfg.BaseURL) != ""
}

func (cfg CloudAuthConfig) connected() bool {
	return cfg.configured() && strings.TrimSpace(cfg.SessionToken) != ""
}

// cacheKey 用 "cloud\x00"+BaseURL+"\x00"+SessionToken 拼出缓存键。\x00 不可出现在
// 正常 URL/令牌里，避免两个不同组合撞键。
func (cfg CloudAuthConfig) cacheKey() string {
	return "cloud\x00" + strings.TrimSpace(cfg.BaseURL) + "\x00" + strings.TrimSpace(cfg.SessionToken)
}

// endpoint 把 BaseURL 与 path、query 安全拼成最终请求地址：先保证 base 以 / 结尾，
// 再用 base.Parse 解析相对路径（自动处理路径合并），避免手拼字符串造成的 // 或错位。
func (cfg CloudAuthConfig) endpoint(path string, query neturl.Values) (string, error) {
	if err := validateCloudBaseURL(cfg.BaseURL, false); err != nil {
		return "", err
	}
	base, err := neturl.Parse(strings.TrimRight(cfg.BaseURL, "/") + "/")
	if err != nil {
		return "", err
	}
	path = strings.TrimLeft(path, "/")
	u, err := base.Parse(path)
	if err != nil {
		return "", err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u.String(), nil
}

// BuildCloudAuthStatus 构造给前端的云端连接状态。verify=true 时会真正打 /api/me 校验会话，
// 并把拉回的用户信息回写配置（含首次写入 ConnectedAt），再做一次基准接口探测。
// verify=false 或未连接时只回静态状态，不发网络请求。
func BuildCloudAuthStatus(ctx context.Context, verify bool) CloudAuthStatus {
	cfg := LoadCloudAuthConfig()
	status := cloudAuthStatusFromConfig(cfg)
	if !verify || !cfg.connected() {
		return status
	}
	me, err := fetchCloudMe(ctx, cfg)
	if err != nil {
		status.LastError = err.Error()
		return status
	}
	cfg.UserLogin = me.User.Login
	cfg.UserName = me.User.Name
	cfg.AvatarURL = me.User.AvatarURL
	cfg.ExpiresAt = me.Session.ExpiresAt
	cfg.LastVerifiedAt = time.Now().Format(time.RFC3339)
	if cfg.ConnectedAt == "" {
		cfg.ConnectedAt = cfg.LastVerifiedAt
	}
	_ = SaveCloudAuthConfig(cfg)
	status = cloudAuthStatusFromConfig(cfg)
	if err := applyCloudBaselineProbe(ctx, cfg, &status); err != nil {
		status.LastError = err.Error()
		return status
	}
	return status
}

func cloudAuthStatusFromConfig(cfg CloudAuthConfig) CloudAuthStatus {
	return CloudAuthStatus{
		Configured:      cfg.configured(),
		Connected:       cfg.connected(),
		BaseURL:         cfg.BaseURL,
		UserLogin:       cfg.UserLogin,
		UserName:        cfg.UserName,
		AvatarURL:       cfg.AvatarURL,
		ConnectedAt:     cfg.ConnectedAt,
		ExpiresAt:       cfg.ExpiresAt,
		LastVerifiedAt:  cfg.LastVerifiedAt,
		SessionFromEnv:  strings.TrimSpace(os.Getenv(cloudAuthSessionTokenEnv)) != "",
		ProviderMessage: "GitHub 登录只用于读取线上排队基准；数据库密钥不会写入本机。",
	}
}

// applyCloudBaselineProbe 用一个固定门店号探测线上基准接口，确认"登录可用 + 基准接口已响应"。
// 各 count 为 0 时再用本地数据兜底重算一次（兼容旧版响应缺字段）。全部为 0 说明接口通了但
// 暂无该门店数据，给出温和提示而非报错。
func applyCloudBaselineProbe(ctx context.Context, cfg CloudAuthConfig, status *CloudAuthStatus) error {
	export, err := fetchQueueBaselineFromCloud(ctx, cfg, cloudAuthBaselineProbeStoreID, time.Now())
	if err != nil {
		return err
	}
	status.BaselineConnected = true
	status.BaselineStoreCount = export.Stats.StoreCount
	status.BaselineLatestCount = export.Stats.LatestCount
	status.BaselineRollupCount = export.Stats.RollupCount
	status.BaselineUpdatedAt = export.Stats.SourceUpdatedAt
	if status.BaselineStoreCount == 0 {
		status.BaselineStoreCount = countQueueBaselineStores(export)
	}
	if status.BaselineLatestCount == 0 {
		status.BaselineLatestCount = len(export.Latest)
	}
	if status.BaselineRollupCount == 0 {
		status.BaselineRollupCount = len(export.Rollups)
	}
	if status.BaselineUpdatedAt == "" {
		status.BaselineUpdatedAt = export.GeneratedAt
	}
	if status.BaselineLatestCount+status.BaselineRollupCount == 0 {
		status.ProviderMessage = "GitHub 会话可用，线上基准接口已响应；探测门店暂未返回基准数据。"
		return nil
	}
	status.ProviderMessage = "GitHub 登录和线上排队基准均已验证；图表会按门店自动读取 Turso 基准。"
	return nil
}

func countQueueBaselineStores(export QueueBaselineExport) int {
	seen := map[int]bool{}
	for _, store := range export.Stores {
		if store.StoreID > 0 {
			seen[store.StoreID] = true
		}
	}
	for _, latest := range export.Latest {
		if latest.StoreID > 0 {
			seen[latest.StoreID] = true
		}
	}
	for _, rollup := range export.Rollups {
		if rollup.StoreID > 0 {
			seen[rollup.StoreID] = true
		}
	}
	return len(seen)
}

func fetchCloudMe(ctx context.Context, cfg CloudAuthConfig) (cloudMeResponse, error) {
	var out cloudMeResponse
	if err := cloudGETJSON(ctx, cfg, "/api/me", nil, &out); err != nil {
		return out, err
	}
	if !out.OK {
		if out.Error != "" {
			return out, errors.New(out.Error)
		}
		return out, fmt.Errorf("云端会话验证失败")
	}
	return out, nil
}

func cloudGETJSON(ctx context.Context, cfg CloudAuthConfig, path string, query neturl.Values, out any) error {
	return cloudJSON(ctx, http.MethodGet, cfg, path, query, nil, out)
}

func cloudPOSTJSON(ctx context.Context, cfg CloudAuthConfig, path string, query neturl.Values, in any, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return cloudJSON(ctx, http.MethodPost, cfg, path, query, body, out)
}

// cloudJSON 是所有云端请求的统一入口：带 Bearer、设 UA、统一 12s 超时、限制响应体 64MB。
// 限制响应体是因为全国基准导出能到 6-8MB 且持续增长，给余量并在超限时报明确错误，
// 而不是让下游 JSON 解析报"unexpected end of JSON input"。
func cloudJSON(ctx context.Context, method string, cfg CloudAuthConfig, path string, query neturl.Values, body []byte, out any) error {
	if !cfg.connected() {
		return fmt.Errorf("尚未登录云端数据服务")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, cloudAuthTimeout)
	defer cancel()
	endpoint, err := cfg.endpoint(path, query)
	if err != nil {
		return err
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(timeoutCtx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.SessionToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sushiro-overdose-cloud-client")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 全国基准导出当前约 6-8MB 且随门店数增长；上限给足余量，并在截断时给出明确错误
	// 而不是让 JSON 解析报“unexpected end of JSON input”。
	const cloudRespLimit = 64 * 1024 * 1024
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, cloudRespLimit+1))
	if err != nil {
		return err
	}
	if len(respBody) > cloudRespLimit {
		return fmt.Errorf("云端响应超过 %dMB 上限，已拒绝解析", cloudRespLimit/(1024*1024))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("云端服务 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("解析云端响应失败: %w", err)
	}
	return nil
}

// newCloudOAuthState 生成一次性的 OAuth state：crypto/rand 取随机字节，base64 编码，
// 写入 cloudOAuthStates 并设置 TTL。写锁内顺带 GC 掉已过期项，避免 map 无限增长。
// 该 state 会随授权 URL 一起发给 GitHub，回调必须原样带回并被 consumeCloudOAuthState 消费。
func newCloudOAuthState() (string, error) {
	buf := make([]byte, cloudAuthSessionTokenSize)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(buf)
	now := time.Now()
	cloudOAuthStates.Lock()
	if cloudOAuthStates.values == nil {
		cloudOAuthStates.values = map[string]time.Time{}
	}
	for key, expires := range cloudOAuthStates.values {
		if now.After(expires) {
			delete(cloudOAuthStates.values, key)
		}
	}
	cloudOAuthStates.values[state] = now.Add(cloudOAuthStateTTL)
	cloudOAuthStates.Unlock()
	return state, nil
}

// consumeCloudOAuthState 校验并单次消费 OAuth state：命中即从 map 删除（无论是否过期），
// 仅当"存在且未过期"返回 true。单次消费是核心——防止同一个 state 被重放成多次登录。
// 空 state 直接拒绝。
func consumeCloudOAuthState(state string) bool {
	state = strings.TrimSpace(state)
	if state == "" {
		return false
	}
	now := time.Now()
	cloudOAuthStates.Lock()
	defer cloudOAuthStates.Unlock()
	expires, ok := cloudOAuthStates.values[state]
	if ok {
		delete(cloudOAuthStates.values, state)
	}
	return ok && now.Before(expires)
}

// localCloudCallbackURL 拼出 OAuth 回调地址给云端跳转。优先用请求自带的 r.Host
// （反映用户实际访问的本机地址/端口，反代场景也能用），缺失时兜底用本机实际监听端口，
// 而不是硬编码端口——避免用户换了端口后回调地址失效。
func localCloudCallbackURL(r *http.Request) string {
	host := r.Host
	if host == "" {
		// r.Host 一般都有；兜底用实际监听端口，而不是硬编码。
		host = "127.0.0.1:" + strconv.Itoa(GetActiveWebPort())
	}
	return "http://" + host + "/api/cloud/auth/callback"
}
