package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// 旧地址只在设置了 SUSHIRO_TOKEN 时用于调试；匿名请求自 2026-10 起被拒绝。
	queueLiveBaseURL   = "https://crm-cn-prd.sushiro.com.cn/wechat/api/2.0"
	queueLiveReferer   = "https://servicewechat.com/wx7ac31ef6c073a7ed/159/page-frame.html"
	queueLiveUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"

	queueLiveRequestTimeout = 15 * time.Second
	// 实时面板、建议和门店名同一次页面刷新会各查一次同一家店，30 秒内复用同一份。
	queueLiveStoreCacheTTL = 30 * time.Second
)

// queueLiveGatewayBaseURL 是小程序现在使用的网关；测试里替换成本机 httptest 地址。
var queueLiveGatewayBaseURL = "https://" + liveGatewayHost + "/gateway/wechat/api/2.0"

// Where a live snapshot came from (QueueLiveStore.Source).
const (
	queueLiveSourceGateway = "gateway"
	queueLiveSourceLegacy  = "legacy"
	queueLiveSourcePassive = "passive"
	queueLiveSourceManual  = "manual"
)

// ErrLiveAuthRequired 表示没有可用的实时数据连接：新网关要求用户自己的会员 token，
// 本地直接返回这个错误，不向任何寿司郎主机发请求。
var ErrLiveAuthRequired = errors.New("live_auth_required")

// The reply did not look like the store (or store list) that was asked for. Nothing is recorded.
var (
	errLiveStoreFormat     = errors.New("寿司郎返回的门店数据格式不对，这次没有记录")
	errLiveStoreListFormat = errors.New("寿司郎返回的门店列表格式不对，请稍后再试")
)

type QueueLiveClient struct {
	baseURL    string
	token      string
	referer    string
	userAgent  string
	httpClient *http.Client

	// source 为 gateway / legacy；测试直接构造的客户端为空，按旧地址处理。
	source string
	// headers 是网关请求的六个上下文头（来自用户自己的连接）；为空时按旧地址发请求。
	headers http.Header
	// sessionMark 标识所用连接；网关拒绝这条连接后，本进程不再用它发请求。
	sessionMark string
	// authErr 非空：没有可用连接，所有方法直接返回它，不发任何请求。
	authErr error
	// storeTTL > 0 时 GetStore 复用 30 秒内同一家店的结果，并合并同时发出的相同请求。
	storeTTL time.Duration
}

type QueueLiveStore struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	NameKana          string  `json:"nameKana"` // City label in the public China API, not a phonetic store name.
	Address           string  `json:"address"`
	Area              string  `json:"area"`
	Latitude          float64 `json:"latitude,omitempty"`
	Longitude         float64 `json:"longitude,omitempty"`
	Distance          string  `json:"distance,omitempty"`
	StoreStatus       string  `json:"storeStatus"`
	NetTicketStatus   string  `json:"netTicketStatus"`
	Wait              int     `json:"wait"`
	WaitTimeCounter   int     `json:"waitTimeCounter,omitempty"`
	WaitTimeCap       int     `json:"waitTimeCap,omitempty"`
	TablesCapacity    int     `json:"tablesCapacity,omitempty"`
	CountersCapacity  int     `json:"countersCapacity,omitempty"`
	GroupQueuesCount  int     `json:"groupQueuesCount,omitempty"`
	ReservationStatus string  `json:"reservationStatus,omitempty"`
	OpenDate          string  `json:"openDate,omitempty"`
	// GroupQueues 仅 getStoreById 返回，列表接口没有。各队列里是当前正在叫的号。
	GroupQueues QueueLiveGroupQueues `json:"groupQueues,omitempty"`

	// ObservedAt 是这帧数据的观测时间；零值表示刚请求到，写入方使用当前时间。
	ObservedAt time.Time `json:"-"`
	// Source 标记来源：gateway（新网关）、legacy（旧地址）、passive（经本机代理看到的小程序响应）、manual（用户填写）。
	Source string `json:"-"`
}

// UnmarshalJSON accepts the store id as a number or a numeric string; any other id is an error.
func (s *QueueLiveStore) UnmarshalJSON(data []byte) error {
	type plain QueueLiveStore
	var aux struct {
		plain
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	id, err := decodeQueueLiveID(aux.ID)
	if err != nil {
		return err
	}
	*s = QueueLiveStore(aux.plain)
	s.ID = id
	return nil
}

func decodeQueueLiveID(raw json.RawMessage) (int, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}
	text := string(raw)
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &text); err != nil {
			return 0, err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return 0, nil
		}
	}
	if n, err := strconv.Atoi(text); err == nil {
		return n, nil
	}
	if f, err := strconv.ParseFloat(text, 64); err == nil && f == math.Trunc(f) && math.Abs(f) < 1e9 {
		return int(f), nil
	}
	return 0, fmt.Errorf("store id %q is not a number", text)
}

// QueueLiveGroupQueues 是 getStoreById 返回的当前叫号队列。booth/mixed/counter
// 是堂食桌位号（同一套号段），reservationQueue 是预约号（独立号段，数值很大）。
type QueueLiveGroupQueues struct {
	ReservationQueue []string `json:"reservationQueue,omitempty"`
	CounterQueue     []string `json:"counterQueue,omitempty"`
	BoothQueue       []string `json:"boothQueue,omitempty"`
	MixedQueue       []string `json:"mixedQueue,omitempty"`
}

// UnmarshalJSON accepts each queue as a string, a number, or an array of them.
func (g *QueueLiveGroupQueues) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*g = QueueLiveGroupQueues{}
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("groupQueues is not an object: %w", err)
	}
	var out QueueLiveGroupQueues
	for key, target := range map[string]*[]string{
		"reservationQueue": &out.ReservationQueue,
		"counterQueue":     &out.CounterQueue,
		"boothQueue":       &out.BoothQueue,
		"mixedQueue":       &out.MixedQueue,
	} {
		value, ok := raw[key]
		if !ok {
			continue
		}
		numbers, err := decodeQueueLiveNumbers(value)
		if err != nil {
			return fmt.Errorf("groupQueues.%s: %w", key, err)
		}
		*target = numbers
	}
	*g = out
	return nil
}

func decodeQueueLiveNumbers(raw json.RawMessage) ([]string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var items []json.RawMessage
	if raw[0] == '[' {
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
	} else {
		items = []json.RawMessage{raw}
	}
	out := []string{}
	for _, item := range items {
		item = bytes.TrimSpace(item)
		switch {
		case len(item) == 0 || string(item) == "null":
		case item[0] == '"':
			var text string
			if err := json.Unmarshal(item, &text); err != nil {
				return nil, err
			}
			for _, part := range strings.Split(text, ",") {
				if part = strings.TrimSpace(part); part != "" {
					out = append(out, part)
				}
			}
		case item[0] == '-' || (item[0] >= '0' && item[0] <= '9'):
			var n json.Number
			if err := json.Unmarshal(item, &n); err != nil {
				return nil, err
			}
			out = append(out, n.String())
		default:
			return nil, fmt.Errorf("unexpected queue entry %s", item)
		}
	}
	return out, nil
}

// CurrentCalledNo 返回堂食当前叫到的号（取 booth/mixed/counter 队列里的最大值）。
// reservationQueue 是另一套号段，不参与堂食叫号统计。返回 0 表示当前没有可用叫号。
func (g QueueLiveGroupQueues) CurrentCalledNo() int {
	best := 0
	for _, queue := range [][]string{g.BoothQueue, g.MixedQueue, g.CounterQueue} {
		for _, raw := range queue {
			if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && n > best {
				best = n
			}
		}
	}
	return best
}

type QueueLiveStoreQuery struct {
	StoreIDs    []string
	City        string
	Area        string
	Keyword     string
	Near        string
	OpenOnly    bool
	WaitingOnly bool
	Limit       int
}

// NewQueueLiveClient 按顺序选择：本进程持有可用的会员连接 → 新网关；否则设置了
// SUSHIRO_TOKEN → 旧地址（调试用）；否则返回“需要连接”的客户端，不发任何请求。
func NewQueueLiveClient() *QueueLiveClient {
	if session, mark := usableLiveSession(time.Now()); session != nil {
		return newGatewayQueueLiveClient(*session, mark)
	}
	if token := strings.TrimSpace(os.Getenv("SUSHIRO_TOKEN")); token != "" {
		return &QueueLiveClient{
			baseURL:    queueLiveBaseURL,
			token:      token,
			referer:    queueLiveReferer,
			userAgent:  queueLiveUserAgent,
			httpClient: &http.Client{Timeout: queueLiveRequestTimeout},
			source:     queueLiveSourceLegacy,
			storeTTL:   queueLiveStoreCacheTTL,
		}
	}
	return &QueueLiveClient{authErr: ErrLiveAuthRequired}
}

func newGatewayQueueLiveClient(s LiveSession, mark string) *QueueLiveClient {
	headers := http.Header{}
	headers.Set("Authorization", s.Authorization)
	headers.Set("X-App-Client", s.AppClient)
	headers.Set("X-App-Code", s.AppCode)
	headers.Set("User-Agent", firstNonEmptyString(s.UserAgent, DefaultMobileWeixinUA()))
	headers.Set("Referer", firstNonEmptyString(s.Referer, queueLiveReferer))
	headers.Set("Content-Type", firstNonEmptyString(s.ContentType, liveSessionDefaultContentType))
	headers.Set("Accept", "*/*")
	return &QueueLiveClient{
		baseURL: queueLiveGatewayBaseURL,
		httpClient: &http.Client{
			Timeout: queueLiveRequestTimeout,
			// 连接只发往网关：不跟随任何跳转，免得请求头被带去别的主机。
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		source:      queueLiveSourceGateway,
		headers:     headers,
		sessionMark: mark,
		storeTTL:    queueLiveStoreCacheTTL,
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func (c *QueueLiveClient) sourceName() string {
	if c.source == "" {
		return queueLiveSourceLegacy
	}
	return c.source
}

func (c *QueueLiveClient) ListStores(ctx context.Context, query QueueLiveStoreQuery) ([]QueueLiveStore, error) {
	if c.authErr != nil {
		return nil, c.authErr
	}
	var (
		stores []QueueLiveStore
		err    error
	)
	if strings.TrimSpace(query.Near) != "" {
		// 带定位的请求要按真实坐标算距离，不能走全量缓存。
		stores, err = c.fetchStores(ctx, query.Near)
	} else {
		stores, err = c.CachedAllStores(ctx)
	}
	if err != nil {
		return nil, err
	}
	return filterQueueLiveStores(stores, query), nil
}

// fetchStores 请求门店列表；不带定位时就是网关已知的 stores?latitude=1&longitude=1&numresults=10000。
func (c *QueueLiveClient) fetchStores(ctx context.Context, near string) ([]QueueLiveStore, error) {
	lat, lng := queueLiveCoordinates(near)
	v := url.Values{}
	v.Set("latitude", lat)
	v.Set("longitude", lng)
	v.Set("numresults", "10000")
	body, err := c.get(ctx, "stores?"+v.Encode())
	if err != nil {
		return nil, err
	}
	stores, err := decodeQueueLiveStores(body)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range stores {
		stores[i].ObservedAt, stores[i].Source = now, c.sourceName()
	}
	return stores, nil
}

const allStoresCacheTTL = 45 * time.Second

type queueLiveStoresCacheEntry struct {
	stores []QueueLiveStore
	at     time.Time
}

var (
	allStoresCacheMu sync.Mutex
	// 按接口地址分开缓存：新网关、旧地址和测试服务器的数据互不混用。
	allStoresCache = map[string]queueLiveStoresCacheEntry{}
)

// CachedAllStores 返回全国门店全量（不带定位距离），带进程内短 TTL 缓存。
// 选择器、基准采集、开放轮询三处共用，避免反复打公开接口；取数失败时回退旧缓存。
//
// 注意：网络拉取在锁外执行。原来全程持锁会让一次 15s 的门店列表请求把所有调用者
// （含后台采集 tick、门店选择器）串行卡住，这里改成「锁内查缓存 → 锁外拉取 → 锁内双检写回」。
func (c *QueueLiveClient) CachedAllStores(ctx context.Context) ([]QueueLiveStore, error) {
	if c.authErr != nil {
		return nil, c.authErr
	}
	key := c.baseURL
	allStoresCacheMu.Lock()
	if entry := allStoresCache[key]; len(entry.stores) > 0 && time.Since(entry.at) < allStoresCacheTTL {
		allStoresCacheMu.Unlock()
		return entry.stores, nil
	}
	allStoresCacheMu.Unlock()

	stores, err := c.fetchStores(ctx, "")
	if err != nil {
		// 失败回退旧缓存（锁内快照）。
		allStoresCacheMu.Lock()
		defer allStoresCacheMu.Unlock()
		if entry := allStoresCache[key]; len(entry.stores) > 0 {
			return entry.stores, nil
		}
		return nil, err
	}

	allStoresCacheMu.Lock()
	defer allStoresCacheMu.Unlock()
	// 双检：并发拉取期间可能已有别的调用者写入了更新鲜的缓存，直接用它。
	if entry := allStoresCache[key]; len(entry.stores) > 0 && time.Since(entry.at) < allStoresCacheTTL {
		return entry.stores, nil
	}
	allStoresCache[key] = queueLiveStoresCacheEntry{stores: stores, at: time.Now()}
	return stores, nil
}

func (c *QueueLiveClient) GetStore(ctx context.Context, storeID string) (QueueLiveStore, error) {
	if c.authErr != nil {
		return QueueLiveStore{}, c.authErr
	}
	if c.storeTTL > 0 {
		return c.cachedStore(ctx, strings.TrimSpace(storeID))
	}
	return c.fetchStore(ctx, storeID)
}

// fetchStore always asks the server; the collector uses it so every record is a fresh reading.
func (c *QueueLiveClient) fetchStore(ctx context.Context, storeID string) (QueueLiveStore, error) {
	if c.authErr != nil {
		return QueueLiveStore{}, c.authErr
	}
	storeID = strings.TrimSpace(storeID)
	if storeID == "" {
		return QueueLiveStore{}, fmt.Errorf("store id is required")
	}
	want, err := strconv.Atoi(storeID)
	if err != nil || want <= 0 {
		return QueueLiveStore{}, fmt.Errorf("门店编号不对：%s", storeID)
	}
	v := url.Values{}
	v.Set("storeId", storeID)
	body, err := c.get(ctx, "getStoreById?"+v.Encode())
	if err != nil {
		return QueueLiveStore{}, err
	}
	store, err := decodeQueueLiveStore(body, want)
	if err != nil {
		return QueueLiveStore{}, err
	}
	store.ObservedAt, store.Source = time.Now(), c.sourceName()
	if c.storeTTL > 0 {
		queueLiveStoreCacheMu.Lock()
		queueLiveStoreCache[c.baseURL+"|"+storeID] = store
		queueLiveStoreCacheMu.Unlock()
	}
	return store, nil
}

// decodeQueueLiveStore accepts only a bare store object whose id is the one requested and
// whose status is present; envelopes, empty objects and other stores are format errors.
func decodeQueueLiveStore(body []byte, wantID int) (QueueLiveStore, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 || body[0] != '{' {
		return QueueLiveStore{}, errLiveStoreFormat
	}
	var store QueueLiveStore
	if err := json.Unmarshal(body, &store); err != nil {
		return QueueLiveStore{}, errLiveStoreFormat
	}
	if store.ID != wantID || strings.TrimSpace(store.StoreStatus) == "" {
		return QueueLiveStore{}, errLiveStoreFormat
	}
	return store, nil
}

type queueLiveStoreCall struct {
	done  chan struct{}
	store QueueLiveStore
	err   error
}

var (
	queueLiveStoreCacheMu sync.Mutex
	queueLiveStoreCache   = map[string]QueueLiveStore{}
	queueLiveStoreCalls   = map[string]*queueLiveStoreCall{}
)

// cachedStore 复用 storeTTL 内同一家店的结果；同时发出的相同请求只发一次。
func (c *QueueLiveClient) cachedStore(ctx context.Context, storeID string) (QueueLiveStore, error) {
	key := c.baseURL + "|" + storeID
	queueLiveStoreCacheMu.Lock()
	if store, ok := queueLiveStoreCache[key]; ok {
		if age := time.Since(store.ObservedAt); age >= 0 && age < c.storeTTL {
			queueLiveStoreCacheMu.Unlock()
			return store, nil
		}
	}
	if call, ok := queueLiveStoreCalls[key]; ok {
		queueLiveStoreCacheMu.Unlock()
		select {
		case <-call.done:
			return call.store, call.err
		case <-ctx.Done():
			return QueueLiveStore{}, ctx.Err()
		}
	}
	call := &queueLiveStoreCall{done: make(chan struct{})}
	queueLiveStoreCalls[key] = call
	queueLiveStoreCacheMu.Unlock()

	// 共享的请求不随第一个调用方取消：切换门店或关掉页面时，其他等着同一家店的调用方
	// 仍拿到结果。请求本身受 15 秒超时约束。
	go func() {
		call.store, call.err = c.fetchStore(context.WithoutCancel(ctx), storeID)
		queueLiveStoreCacheMu.Lock()
		delete(queueLiveStoreCalls, key)
		queueLiveStoreCacheMu.Unlock()
		close(call.done)
	}()
	select {
	case <-call.done:
		return call.store, call.err
	case <-ctx.Done():
		return QueueLiveStore{}, ctx.Err()
	}
}

func (c *QueueLiveClient) ListAreas(ctx context.Context) ([]string, error) {
	if c.authErr != nil {
		return nil, c.authErr
	}
	if c.source == queueLiveSourceGateway {
		// 网关没有已知的区域接口：用门店列表里的 area 去重得到。
		stores, err := c.CachedAllStores(ctx)
		if err != nil {
			return nil, err
		}
		areas := make([]string, 0, len(stores))
		for _, store := range stores {
			areas = append(areas, store.Area)
		}
		areas = UniqueNonEmptyStrings(areas)
		sort.Strings(areas)
		return areas, nil
	}
	body, err := c.get(ctx, "areas")
	if err != nil {
		return nil, err
	}
	var areas []string
	if err := json.Unmarshal(body, &areas); err != nil {
		return nil, fmt.Errorf("queue areas response is not a JSON array: %w", err)
	}
	return UniqueNonEmptyStrings(areas), nil
}

func (c *QueueLiveClient) get(ctx context.Context, path string) ([]byte, error) {
	if c.authErr != nil {
		return nil, c.authErr
	}
	target := strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("create queue request: %w", err)
	}
	if c.headers != nil {
		for name, values := range c.headers {
			req.Header[name] = append([]string(nil), values...)
		}
	} else {
		req.Header.Set("Authorization", EnsureBearer(c.token))
		req.Header.Set("Referer", c.referer)
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "*/*")
	}
	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: queueLiveRequestTimeout}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("queue request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read queue response: %w", err)
	}
	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Body: NormalizeErrorBody(body)}
		if c.source == queueLiveSourceGateway && liveErrorCode(apiErr) == liveErrAuthRequired {
			noteLiveSessionRejected(c.sessionMark)
		}
		return nil, apiErr
	}
	if resp.StatusCode >= 300 || !json.Valid(body) {
		return nil, fmt.Errorf("queue response is not JSON (status %d)", resp.StatusCode)
	}
	return body, nil
}

// decodeQueueLiveStores 接受门店数组，或外层对象里的门店数组。找不到数组、或数组里
// 没有一条像门店的数据时报错；没有 id 的条目丢弃，不写零值记录。
func decodeQueueLiveStores(body []byte) ([]QueueLiveStore, error) {
	body = bytes.TrimSpace(body)
	var arrays []json.RawMessage
	switch {
	case len(body) > 0 && body[0] == '[':
		arrays = append(arrays, body)
	case len(body) > 0 && body[0] == '{':
		var wrapped map[string]json.RawMessage
		if err := json.Unmarshal(body, &wrapped); err != nil {
			return nil, errLiveStoreListFormat
		}
		keys := make([]string, 0, len(wrapped))
		for key := range wrapped {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if raw := bytes.TrimSpace(wrapped[key]); len(raw) > 0 && raw[0] == '[' {
				arrays = append(arrays, raw)
			}
		}
		if len(arrays) == 0 {
			return nil, errLiveStoreListFormat
		}
	default:
		return nil, errLiveStoreListFormat
	}
	stores := []QueueLiveStore{}
	entries := 0
	for _, raw := range arrays {
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, errLiveStoreListFormat
		}
		entries += len(items)
		for _, item := range items {
			var store QueueLiveStore
			if item = bytes.TrimSpace(item); len(item) == 0 || item[0] != '{' || json.Unmarshal(item, &store) != nil || store.ID <= 0 {
				continue
			}
			stores = append(stores, store)
		}
	}
	if entries > 0 && len(stores) == 0 {
		return nil, errLiveStoreListFormat
	}
	return stores, nil
}

func filterQueueLiveStores(stores []QueueLiveStore, query QueueLiveStoreQuery) []QueueLiveStore {
	allowed := stringSet(query.StoreIDs)
	keyword := strings.TrimSpace(query.Keyword)
	keywordCity := ""
	// Only infer a city from an exact existing label. Explicit city filters keep
	// their keyword available for street/store searches within that city.
	if query.City == "" {
		city := queueLiveCityKey(keyword)
		for _, store := range stores {
			if city != "" && queueLiveCityKey(store.NameKana) == city {
				keywordCity = city
				break
			}
		}
	}
	out := make([]QueueLiveStore, 0, len(stores))
	for _, store := range stores {
		id := strconv.Itoa(store.ID)
		if len(allowed) > 0 && !allowed[id] {
			continue
		}
		if query.City != "" && store.NameKana != query.City {
			continue
		}
		if query.Area != "" && !strings.Contains(store.Area, query.Area) {
			continue
		}
		if keywordCity != "" {
			if queueLiveCityKey(store.NameKana) != keywordCity {
				continue
			}
		} else if keyword != "" && !queueLiveStoreContains(store, keyword) {
			continue
		}
		if query.OpenOnly && !strings.EqualFold(store.StoreStatus, "OPEN") {
			continue
		}
		if query.WaitingOnly && store.Wait <= 0 {
			continue
		}
		out = append(out, store)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Wait != out[j].Wait {
			return out[i].Wait > out[j].Wait
		}
		if out[i].NameKana != out[j].NameKana {
			return out[i].NameKana < out[j].NameKana
		}
		return out[i].Name < out[j].Name
	})
	if query.Limit > 0 && len(out) > query.Limit {
		out = out[:query.Limit]
	}
	return out
}

func queueLiveCityKey(city string) string {
	return strings.TrimSuffix(strings.TrimSpace(city), "市")
}

func queueLiveStoreContains(store QueueLiveStore, keyword string) bool {
	return strings.Contains(store.Name, keyword) ||
		strings.Contains(store.Area, keyword) ||
		strings.Contains(store.NameKana, keyword) ||
		strings.Contains(store.Address, keyword)
}

func queueLiveCoordinates(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "1", "1"
	}
	lat, lng, ok := strings.Cut(raw, ",")
	if !ok {
		return "1", "1"
	}
	lat = strings.TrimSpace(lat)
	lng = strings.TrimSpace(lng)
	if lat == "" || lng == "" {
		return "1", "1"
	}
	return lat, lng
}

// queueLiveStoreOnlineOpen 判断门店线上取号是否已开放（netTicketStatus 翻为开放态）。
// 已知取值：ONLINE=开放、OFFLINE_CLOSED=未开放。
func queueLiveStoreOnlineOpen(s QueueLiveStore) bool {
	v := strings.ToUpper(strings.TrimSpace(s.NetTicketStatus))
	if v == "" || strings.Contains(v, "CLOSED") || strings.Contains(v, "OFFLINE") {
		return false
	}
	return v == "ONLINE" || v == "ON" || strings.Contains(v, "OPEN")
}

func queueObservationFromLiveStore(store QueueLiveStore, at time.Time) QueueObservation {
	if at.IsZero() {
		at = time.Now()
	}
	return QueueObservation{
		CollectedAt:       at.Format(time.RFC3339),
		StoreID:           strconv.Itoa(store.ID),
		WaitMinutes:       store.Wait,
		GroupQueuesCount:  store.GroupQueuesCount,
		StoreStatus:       store.StoreStatus,
		NetTicketStatus:   store.NetTicketStatus,
		ReservationStatus: store.ReservationStatus,
		OnlineOpen:        queueLiveStoreOnlineOpen(store),
		WaitTimeCounter:   store.WaitTimeCounter,
		WaitTimeCap:       store.WaitTimeCap,
		SourceEndpoint:    queueSourceEndpointStoreByID,
		APIProfileVersion: queueAPIProfilePublicV1,
		DisplayCalledNo:   store.GroupQueues.CurrentCalledNo(),
		GroupQueues:       queueLiveGroupQueuesToCore(store.GroupQueues),
	}
}

func queueLiveGroupQueuesToCore(queues QueueLiveGroupQueues) QueueGroupQueues {
	return QueueGroupQueues{
		ReservationQueue: queues.ReservationQueue,
		CounterQueue:     queues.CounterQueue,
		BoothQueue:       queues.BoothQueue,
		MixedQueue:       queues.MixedQueue,
	}
}
