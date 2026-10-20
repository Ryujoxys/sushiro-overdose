package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Live session：用户自己的小程序会员连接（spec 018）。
//
// 新网关要求绑定会员账号的 Bearer token（JWT，约 1 小时）和小程序上下文头。这里只保存
// 只读查询需要的六个头，只发往 sapi.sushiro.com.cn 的 getStoreById / stores。
// 宪章边界：只有界面进程在启动时把 LoadLiveSession 注入 queueLiveSessionProvider；
// 独立后台记录进程从不设置它，也不读这个文件。头部的值不写日志、不出现在任何响应里。

const (
	liveSessionFile = "live_session.json"

	liveGatewayHost   = "sapi.sushiro.com.cn"
	liveLegacyHost    = "crm-cn-prd.sushiro.com.cn"
	liveGatewayMarker = "/gateway/"

	liveSessionExpiryMargin   = 30 * time.Second // 到期前 30 秒就不再使用，避免请求途中过期
	liveSessionNoExpiryWindow = 55 * time.Minute // token 里没有 exp 时按抓到后 55 分钟估算
	liveSessionExpiringSoon   = 10 * time.Minute

	liveSessionDefaultContentType = "application/json"
	liveSessionMaxHeaderBytes     = 8 << 10
)

// Sources of a saved connection.
const (
	liveSessionSourceImport  = "import"
	liveSessionSourceCapture = "capture"
)

// LiveSession 是网关只读查询需要的会员连接。IssuedAt/ExpiresAt 取自 JWT，未验签，只用于判断何时停用。
type LiveSession struct {
	Authorization string    `json:"authorization"` // "Bearer <jwt>"
	AppClient     string    `json:"app_client"`
	AppCode       string    `json:"app_code"`
	UserAgent     string    `json:"user_agent"`
	Referer       string    `json:"referer"`
	ContentType   string    `json:"content_type"`
	IssuedAt      time.Time `json:"issued_at,omitempty"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	CapturedAt    time.Time `json:"captured_at"`
	Source        string    `json:"source"` // import | capture
}

// queueLiveSessionProvider 由界面进程启动时设为 LoadLiveSession。默认 nil：独立后台记录进程
// 和命令行从不持有会员连接，NewQueueLiveClient 在它们里面只会得到“需要连接”的客户端。
var queueLiveSessionProvider func() (*LiveSession, error)

var (
	liveSessionMu sync.Mutex

	// A gateway 401/403 means this connection is dead even before its exp. Remember it in
	// memory so the page stops sending requests until a different connection arrives.
	liveSessionRejectedMu   sync.Mutex
	liveSessionRejectedMark string
)

var (
	errLiveImportEmpty      = errors.New("请粘贴一条抓包里的寿司郎请求（curl、请求头或 HAR 文件内容）")
	errLiveImportUnreadable = errors.New("看不懂这段内容：请粘贴 curl、原始请求头或 HAR 文件内容")
	errLiveImportNotGateway = errors.New("没找到寿司郎新接口的请求：请抓小程序门店或排队页发往 sapi.sushiro.com.cn 的请求")
	errLiveImportLegacy     = errors.New("这是寿司郎旧接口的请求，已经不能用了：请在小程序里刷新门店页，抓发往 sapi.sushiro.com.cn 的请求")
	errLiveImportExpired    = errors.New("这条请求已经过期：打开小程序刷新一下，再抓一条新的")
	errLiveNoAuthorization  = errors.New("这条请求里没有登录信息：打开小程序刷新一下，再抓一条新的")
	errLiveBadAuthorization = errors.New("这条请求里的登录信息看不懂：请抓小程序门店或排队页的请求")
	errLiveNoAppContext     = errors.New("这条请求缺少小程序的身份信息：请换一条门店或排队页的请求")
	errLiveBadHeaderValue   = errors.New("这条请求里有无法使用的内容：请重新复制一次")
)

func liveSessionPath() string { return filepath.Join(AppDirPath(), liveSessionFile) }

// LoadLiveSession 读取已保存的连接；文件不存在返回 nil, nil。
func LoadLiveSession() (*LiveSession, error) {
	data, err := os.ReadFile(liveSessionPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s LiveSession
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, errors.New("live session file is damaged")
	}
	if strings.TrimSpace(s.Authorization) == "" {
		return nil, nil
	}
	return &s, nil
}

// SaveLiveSession 只在新连接比已保存的更新时写入（原子写入，0600），返回是否写入。
// 有 exp 时按 exp 比较，否则按抓到的时间；同一个 token 不重复写，已失效的连接不写；
// 已保存的连接过期或被寿司郎拒绝时，任何一条别的可用连接都会替换它。
func SaveLiveSession(s LiveSession) (bool, error) {
	if err := s.validate(); err != nil {
		return false, err
	}
	now := time.Now()
	if !s.Usable(now) {
		return false, nil
	}
	liveSessionMu.Lock()
	defer liveSessionMu.Unlock()
	if old, err := LoadLiveSession(); err == nil && old != nil && !s.newerThan(*old, now) {
		return false, nil
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(AppDirPath(), 0o700); err != nil {
		return false, err
	}
	if err := AtomicWriteFile(liveSessionPath(), data, 0o600); err != nil {
		return false, err
	}
	// 新连接到手，忘掉之前“实时取不到”的记忆，下一次规划立刻用实时数据。
	noteQueueLiveResult(nil)
	return true, nil
}

// ClearLiveSession 删除已保存的连接；本来就没有时不报错。
func ClearLiveSession() error {
	liveSessionMu.Lock()
	defer liveSessionMu.Unlock()
	if err := os.Remove(liveSessionPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s LiveSession) newerThan(old LiveSession, now time.Time) bool {
	if !old.Usable(now) {
		return true
	}
	if s.Authorization == old.Authorization {
		return false
	}
	// 寿司郎已经拒绝了存着的那条连接：换成任何一条别的可用连接，不论两者谁的 exp 更晚。
	if liveSessionRejected(old.mark()) {
		return true
	}
	if !s.ExpiresAt.IsZero() && !old.ExpiresAt.IsZero() && !s.ExpiresAt.Equal(old.ExpiresAt) {
		return s.ExpiresAt.After(old.ExpiresAt)
	}
	return s.CapturedAt.After(old.CapturedAt)
}

func (s LiveSession) validate() error {
	if _, ok := liveBearerToken(s.Authorization); !ok {
		return errLiveNoAuthorization
	}
	if strings.TrimSpace(s.AppClient) == "" || strings.TrimSpace(s.AppCode) == "" {
		return errLiveNoAppContext
	}
	for _, value := range []string{s.Authorization, s.AppClient, s.AppCode, s.UserAgent, s.Referer, s.ContentType} {
		if !liveHeaderValueOK(value) {
			return errLiveBadHeaderValue
		}
	}
	return nil
}

// usableUntil 是停用时间：有 exp 时提前 30 秒，没有 exp 时按抓到后 55 分钟。
func (s LiveSession) usableUntil() time.Time {
	if !s.ExpiresAt.IsZero() {
		return s.ExpiresAt.Add(-liveSessionExpiryMargin)
	}
	if s.CapturedAt.IsZero() {
		return time.Time{}
	}
	return s.CapturedAt.Add(liveSessionNoExpiryWindow)
}

// Usable 判断现在还能不能用这条连接发请求。
func (s LiveSession) Usable(now time.Time) bool {
	if strings.TrimSpace(s.Authorization) == "" || strings.TrimSpace(s.AppClient) == "" || strings.TrimSpace(s.AppCode) == "" {
		return false
	}
	until := s.usableUntil()
	return !until.IsZero() && now.Before(until)
}

// Remaining 是还能用多久；不可用时为 0。
func (s LiveSession) Remaining(now time.Time) time.Duration {
	if !s.Usable(now) {
		return 0
	}
	return s.usableUntil().Sub(now)
}

// mark identifies one connection without exposing it; the collector stores it to tell
// whether the user has connected again since the last authentication failure.
func (s LiveSession) mark() string {
	return s.CapturedAt.UTC().Format(time.RFC3339Nano)
}

// usableLiveSession returns this process's usable connection, or nil when the process
// holds none (always the case in the background service, which never sets the provider).
func usableLiveSession(now time.Time) (*LiveSession, string) {
	provider := queueLiveSessionProvider
	if provider == nil {
		return nil, ""
	}
	s, err := provider()
	if err != nil || s == nil || !s.Usable(now) {
		return nil, ""
	}
	mark := s.mark()
	if liveSessionRejected(mark) {
		return nil, ""
	}
	return s, mark
}

// liveSessionMark is "" when this process has no usable connection.
func liveSessionMark(now time.Time) string {
	_, mark := usableLiveSession(now)
	return mark
}

func noteLiveSessionRejected(mark string) {
	if mark == "" {
		return
	}
	liveSessionRejectedMu.Lock()
	liveSessionRejectedMark = mark
	liveSessionRejectedMu.Unlock()
}

func liveSessionRejected(mark string) bool {
	liveSessionRejectedMu.Lock()
	defer liveSessionRejectedMu.Unlock()
	return mark != "" && mark == liveSessionRejectedMark
}

// LiveSessionFromHeaders 从一条网关请求的头部建立连接。必须有 Bearer authorization、
// x-app-client、x-app-code；UA / Referer / Content-Type 缺省时用已知的小程序值。
func LiveSessionFromHeaders(h http.Header, now time.Time, source string) (LiveSession, error) {
	authorization := strings.TrimSpace(h.Get("Authorization"))
	if authorization == "" {
		return LiveSession{}, errLiveNoAuthorization
	}
	token, ok := liveBearerToken(authorization)
	if !ok {
		return LiveSession{}, errLiveBadAuthorization
	}
	s := LiveSession{
		Authorization: "Bearer " + token,
		AppClient:     strings.TrimSpace(h.Get("X-App-Client")),
		AppCode:       strings.TrimSpace(h.Get("X-App-Code")),
		UserAgent:     strings.TrimSpace(h.Get("User-Agent")),
		Referer:       strings.TrimSpace(h.Get("Referer")),
		ContentType:   strings.TrimSpace(h.Get("Content-Type")),
		CapturedAt:    now,
		Source:        source,
	}
	if s.AppClient == "" || s.AppCode == "" {
		return LiveSession{}, errLiveNoAppContext
	}
	if s.UserAgent == "" {
		s.UserAgent = DefaultMobileWeixinUA()
	}
	if s.Referer == "" {
		s.Referer = queueLiveReferer
	}
	if s.ContentType == "" {
		s.ContentType = liveSessionDefaultContentType
	}
	s.IssuedAt, s.ExpiresAt = liveJWTTimes(token)
	if err := s.validate(); err != nil {
		return LiveSession{}, err
	}
	return s, nil
}

// liveBearerToken accepts "Bearer <token>" (any case) or a bare JWT; other schemes are rejected.
func liveBearerToken(authorization string) (string, bool) {
	authorization = strings.TrimSpace(authorization)
	if len(authorization) > liveSessionMaxHeaderBytes {
		return "", false
	}
	scheme, rest, hasSpace := strings.Cut(authorization, " ")
	if hasSpace {
		if !strings.EqualFold(scheme, "Bearer") {
			return "", false
		}
		token := strings.TrimSpace(rest)
		return token, token != "" && !strings.ContainsAny(token, " \t")
	}
	if strings.Count(authorization, ".") == 2 {
		return authorization, true
	}
	return "", false
}

func liveHeaderValueOK(value string) bool {
	if len(value) > liveSessionMaxHeaderBytes {
		return false
	}
	for i := 0; i < len(value); i++ {
		if c := value[i]; (c < 0x20 && c != '\t') || c == 0x7f {
			return false
		}
	}
	return true
}

// liveJWTTimes reads iat/exp from a JWT payload without verifying the signature.
func liveJWTTimes(token string) (issuedAt, expiresAt time.Time) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return
	}
	var claims struct {
		IssuedAt  json.Number `json:"iat"`
		ExpiresAt json.Number `json:"exp"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if decoder.Decode(&claims) != nil {
		return
	}
	return liveJWTTime(claims.IssuedAt), liveJWTTime(claims.ExpiresAt)
}

func liveJWTTime(n json.Number) time.Time {
	value, err := n.Float64()
	if err != nil || value <= 0 || math.IsInf(value, 0) || math.IsNaN(value) {
		return time.Time{}
	}
	if value > 1e12 { // milliseconds
		value /= 1000
	}
	at := time.Unix(int64(value), 0)
	if at.Year() < 2020 || at.Year() > 2100 {
		return time.Time{}
	}
	return at
}

// ---------- 粘贴抓包 ----------

type liveImportRequest struct {
	target *url.URL
	header http.Header
	at     time.Time
}

// ParseLiveSessionText 从用户粘贴的 curl / 原始请求头 / HAR 里找出发往新网关的请求并建立连接。
// 只接受 sapi.sushiro.com.cn 且路径含 /gateway/ 的请求；HAR 取时间最新的一条可用请求。
func ParseLiveSessionText(text string, now time.Time) (LiveSession, error) {
	text = strings.TrimSpace(strings.TrimPrefix(text, string(rune(0xFEFF)))) // 去掉记事本等加的 BOM
	if text == "" {
		return LiveSession{}, errLiveImportEmpty
	}
	var requests []liveImportRequest
	switch {
	case strings.HasPrefix(text, "{") || strings.HasPrefix(text, "["):
		var err error
		if requests, err = liveRequestsFromHAR(text); err != nil {
			return LiveSession{}, err
		}
	case len(text) >= 4 && strings.EqualFold(text[:4], "curl"):
		// “复制全部为 cURL”会一次给出多条命令：逐条解析，免得把别的请求的头混进网关连接。
		for _, command := range liveSplitCurlCommands(text) {
			requests = append(requests, liveRequestFromCurl(command))
		}
	default:
		requests = []liveImportRequest{liveRequestFromRaw(text)}
	}

	var (
		best      LiveSession
		bestAt    time.Time
		found     bool
		firstErr  error
		sawLegacy bool
		sawTarget bool
	)
	for _, request := range requests {
		kind := liveRequestKind(request.target)
		sawLegacy = sawLegacy || kind == "legacy"
		if kind != "gateway" {
			continue
		}
		sawTarget = true
		s, err := LiveSessionFromHeaders(request.header, now, liveSessionSourceImport)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if !found || !request.at.Before(bestAt) {
			best, bestAt, found = s, request.at, true
		}
	}
	switch {
	case found && !best.Usable(now):
		return LiveSession{}, errLiveImportExpired
	case found:
		return best, nil
	case sawTarget:
		return LiveSession{}, firstErr
	case sawLegacy:
		return LiveSession{}, errLiveImportLegacy
	}
	return LiveSession{}, errLiveImportNotGateway
}

func liveRequestKind(target *url.URL) string {
	if target == nil {
		return ""
	}
	host := strings.ToLower(strings.TrimSuffix(target.Hostname(), "."))
	switch {
	case host == liveGatewayHost && strings.Contains(target.Path, liveGatewayMarker):
		return "gateway"
	case host == liveLegacyHost:
		return "legacy"
	}
	return ""
}

func liveRequestsFromHAR(text string) ([]liveImportRequest, error) {
	var har struct {
		Log struct {
			Entries []struct {
				StartedDateTime string `json:"startedDateTime"`
				Request         struct {
					URL     string `json:"url"`
					Headers []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					} `json:"headers"`
				} `json:"request"`
			} `json:"entries"`
		} `json:"log"`
	}
	if err := json.Unmarshal([]byte(text), &har); err != nil || len(har.Log.Entries) == 0 {
		return nil, errLiveImportUnreadable
	}
	requests := make([]liveImportRequest, 0, len(har.Log.Entries))
	for _, entry := range har.Log.Entries {
		target, err := url.Parse(strings.TrimSpace(entry.Request.URL))
		if err != nil {
			continue
		}
		header := http.Header{}
		for _, h := range entry.Request.Headers {
			liveAddHeader(header, h.Name, h.Value)
		}
		at, _ := time.Parse(time.RFC3339Nano, strings.TrimSpace(entry.StartedDateTime))
		requests = append(requests, liveImportRequest{target: target, header: header, at: at})
	}
	return requests, nil
}

func liveRequestFromCurl(text string) liveImportRequest {
	if strings.Contains(text, `^"`) { // Windows cmd 格式：^ 是转义符
		text = liveUnescapeCmd(text)
	}
	request := liveImportRequest{header: http.Header{}}
	fields := shellLikeFields(text)
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		next := func() (string, bool) {
			if i+1 >= len(fields) {
				return "", false
			}
			i++
			return fields[i], true
		}
		switch {
		case field == "-H" || field == "--header":
			if value, ok := next(); ok {
				liveAddHeaderLine(request.header, value)
			}
		case strings.HasPrefix(field, "--header="):
			liveAddHeaderLine(request.header, strings.TrimPrefix(field, "--header="))
		case strings.HasPrefix(field, "-H") && len(field) > 2:
			liveAddHeaderLine(request.header, field[2:])
		case field == "-A" || field == "--user-agent":
			if value, ok := next(); ok {
				liveAddHeader(request.header, "User-Agent", value)
			}
		case field == "-e" || field == "--referer":
			if value, ok := next(); ok {
				liveAddHeader(request.header, "Referer", value)
			}
		case field == "--url":
			if value, ok := next(); ok {
				request.target = liveParseURL(value, request.target)
			}
		case strings.HasPrefix(field, "http://") || strings.HasPrefix(field, "https://"):
			request.target = liveParseURL(field, request.target)
		}
	}
	return request
}

var liveCurlCommandPattern = regexp.MustCompile(`(?im)^[ \t]*curl[ \t]`)

// liveSplitCurlCommands splits pasted text at every line that starts a new curl command.
func liveSplitCurlCommands(text string) []string {
	starts := liveCurlCommandPattern.FindAllStringIndex(text, -1)
	if len(starts) <= 1 {
		return []string{text}
	}
	commands := make([]string, 0, len(starts))
	for i, start := range starts {
		end := len(text)
		if i+1 < len(starts) {
			end = starts[i+1][0]
		}
		commands = append(commands, text[start[0]:end])
	}
	return commands
}

func liveParseURL(raw string, current *url.URL) *url.URL {
	if current != nil {
		return current
	}
	target, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil
	}
	return target
}

func liveUnescapeCmd(text string) string {
	text = strings.NewReplacer("^\r\n", " ", "^\n", " ").Replace(text)
	var b strings.Builder
	escaped := false
	for _, r := range text {
		if !escaped && r == '^' {
			escaped = true
			continue
		}
		escaped = false
		b.WriteRune(r)
	}
	return b.String()
}

var (
	liveRequestLinePattern = regexp.MustCompile(`^(?i:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+(\S+)`)
	liveURLPattern         = regexp.MustCompile(`https?://[^\s'"<>]+`)
)

// liveRequestFromRaw reads a pasted request: an optional request line, then "Name: value"
// lines (HTTP/2 pseudo-headers such as ":authority" are understood too).
func liveRequestFromRaw(text string) liveImportRequest {
	request := liveImportRequest{header: http.Header{}}
	requestTarget, authority, path := "", "", ""
	inHeaders := false
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if inHeaders {
				break // 空行后是请求体
			}
			continue
		}
		if m := liveRequestLinePattern.FindStringSubmatch(line); m != nil && requestTarget == "" {
			requestTarget = m[1]
			inHeaders = true
			continue
		}
		if strings.HasPrefix(line, ":") {
			name, value, ok := strings.Cut(line[1:], ":")
			if !ok {
				continue
			}
			value = strings.TrimSpace(value)
			switch strings.ToLower(strings.TrimSpace(name)) {
			case "authority":
				authority = value
			case "path":
				path = value
			}
			inHeaders = true
			continue
		}
		if liveAddHeaderLine(request.header, line) {
			inHeaders = true
		}
	}
	if authority == "" {
		authority = request.header.Get("Host")
	}
	switch {
	case strings.HasPrefix(requestTarget, "http://") || strings.HasPrefix(requestTarget, "https://"):
		request.target = liveParseURL(requestTarget, nil)
	case authority != "" && (requestTarget != "" || path != ""):
		if path == "" {
			path = requestTarget
		}
		request.target = liveParseURL("https://"+authority+path, nil)
	default:
		if match := liveURLPattern.FindString(text); match != "" {
			request.target = liveParseURL(match, nil)
		}
	}
	return request
}

func liveAddHeaderLine(header http.Header, line string) bool {
	line = strings.TrimSpace(line)
	name, value, ok := strings.Cut(line, ":")
	name = strings.TrimSpace(name)
	if !ok || name == "" || strings.ContainsAny(name, " \t") {
		return false
	}
	liveAddHeader(header, name, value)
	return true
}

func liveAddHeader(header http.Header, name, value string) {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, ":") {
		return
	}
	header.Set(textproto.CanonicalMIMEHeaderKey(name), strings.TrimSpace(value))
}

// ---------- 状态 ----------

// liveSessionStatus is what the page may know about the connection; header values never appear.
type liveSessionStatus struct {
	State            string `json:"state"` // none | active | expiring | expired
	Connected        bool   `json:"connected"`
	ExpiresAt        string `json:"expires_at,omitempty"`
	RemainingSeconds int    `json:"remaining_seconds"`
	CapturedAt       string `json:"captured_at,omitempty"`
	Source           string `json:"source,omitempty"`
}

func liveSessionStatusAt(s *LiveSession, now time.Time) liveSessionStatus {
	if s == nil {
		return liveSessionStatus{State: "none"}
	}
	status := liveSessionStatus{State: "expired", Source: s.Source}
	if !s.CapturedAt.IsZero() {
		status.CapturedAt = s.CapturedAt.Format(time.RFC3339)
	}
	if expires := s.ExpiresAt; !expires.IsZero() {
		status.ExpiresAt = expires.Format(time.RFC3339)
	} else if until := s.usableUntil(); !until.IsZero() {
		status.ExpiresAt = until.Format(time.RFC3339)
	}
	if !s.Usable(now) || liveSessionRejected(s.mark()) {
		return status
	}
	remaining := s.Remaining(now)
	status.Connected = true
	status.RemainingSeconds = int(remaining / time.Second)
	status.State = "active"
	if remaining < liveSessionExpiringSoon {
		status.State = "expiring"
	}
	return status
}

func currentLiveSessionStatus(now time.Time) liveSessionStatus {
	s, err := LoadLiveSession()
	if err != nil {
		return liveSessionStatus{State: "none"}
	}
	return liveSessionStatusAt(s, now)
}
