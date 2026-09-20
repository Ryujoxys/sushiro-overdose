package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	queueLiveBaseURL      = "https://crm-cn-prd.sushiro.com.cn/wechat/api/2.0"
	queueLiveDefaultToken = "4OI44O844Kv44Oz5qSc6Ki855So77yad2VjaGF05YWx6YCa4"
	queueLiveReferer      = "https://servicewechat.com/wx7ac31ef6c073a7ed/159/page-frame.html"
	queueLiveUserAgent    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
)

type QueueLiveClient struct {
	baseURL    string
	token      string
	referer    string
	userAgent  string
	httpClient *http.Client
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
}

// QueueLiveGroupQueues 是 getStoreById 返回的当前叫号队列。booth/mixed/counter
// 是堂食桌位号（同一套号段），reservationQueue 是预约号（独立号段，数值很大）。
type QueueLiveGroupQueues struct {
	ReservationQueue []string `json:"reservationQueue,omitempty"`
	CounterQueue     []string `json:"counterQueue,omitempty"`
	BoothQueue       []string `json:"boothQueue,omitempty"`
	MixedQueue       []string `json:"mixedQueue,omitempty"`
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

func NewQueueLiveClient() *QueueLiveClient {
	token := strings.TrimSpace(os.Getenv("SUSHIRO_TOKEN"))
	if token == "" {
		token = queueLiveDefaultToken
	}
	return &QueueLiveClient{
		baseURL:   queueLiveBaseURL,
		token:     token,
		referer:   queueLiveReferer,
		userAgent: queueLiveUserAgent,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *QueueLiveClient) ListStores(ctx context.Context, query QueueLiveStoreQuery) ([]QueueLiveStore, error) {
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
	return decodeQueueLiveStores(body)
}

const allStoresCacheTTL = 45 * time.Second

var (
	allStoresCacheMu sync.Mutex
	allStoresCache   []QueueLiveStore
	allStoresCacheAt time.Time
)

// CachedAllStores 返回全国门店全量（不带定位距离），带进程内短 TTL 缓存。
// 选择器、基准采集、开放轮询三处共用，避免反复打公开接口；取数失败时回退旧缓存。
//
// 注意：网络拉取在锁外执行。原来全程持锁会让一次 15s 的门店列表请求把所有调用者
// （含后台采集 tick、门店选择器）串行卡住，这里改成「锁内查缓存 → 锁外拉取 → 锁内双检写回」。
func (c *QueueLiveClient) CachedAllStores(ctx context.Context) ([]QueueLiveStore, error) {
	allStoresCacheMu.Lock()
	if len(allStoresCache) > 0 && time.Since(allStoresCacheAt) < allStoresCacheTTL {
		out := allStoresCache
		allStoresCacheMu.Unlock()
		return out, nil
	}
	allStoresCacheMu.Unlock()

	stores, err := c.fetchStores(ctx, "")
	if err != nil {
		// 失败回退旧缓存（锁内快照）。
		allStoresCacheMu.Lock()
		defer allStoresCacheMu.Unlock()
		if len(allStoresCache) > 0 {
			return allStoresCache, nil
		}
		return nil, err
	}

	allStoresCacheMu.Lock()
	defer allStoresCacheMu.Unlock()
	// 双检：并发拉取期间可能已有别的调用者写入了更新鲜的缓存，直接用它。
	if len(allStoresCache) > 0 && time.Since(allStoresCacheAt) < allStoresCacheTTL {
		return allStoresCache, nil
	}
	allStoresCache = stores
	allStoresCacheAt = time.Now()
	return stores, nil
}

func (c *QueueLiveClient) GetStore(ctx context.Context, storeID string) (QueueLiveStore, error) {
	storeID = strings.TrimSpace(storeID)
	if storeID == "" {
		return QueueLiveStore{}, fmt.Errorf("store id is required")
	}
	v := url.Values{}
	v.Set("storeId", storeID)
	body, err := c.get(ctx, "getStoreById?"+v.Encode())
	if err != nil {
		return QueueLiveStore{}, err
	}
	var store QueueLiveStore
	if err := json.Unmarshal(body, &store); err != nil {
		return QueueLiveStore{}, fmt.Errorf("queue store response is not a JSON object: %w", err)
	}
	return store, nil
}

func (c *QueueLiveClient) ListAreas(ctx context.Context) ([]string, error) {
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
	target := strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("create queue request: %w", err)
	}
	req.Header.Set("Authorization", EnsureBearer(c.token))
	req.Header.Set("Referer", c.referer)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("queue request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read queue response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: NormalizeErrorBody(body)}
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("queue response is not JSON: %s", string(body))
	}
	return body, nil
}

func decodeQueueLiveStores(body []byte) ([]QueueLiveStore, error) {
	var stores []QueueLiveStore
	if err := json.Unmarshal(body, &stores); err == nil {
		return stores, nil
	}

	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("queue stores response is not a JSON array/object: %w", err)
	}
	for _, raw := range wrapped {
		var part []QueueLiveStore
		if err := json.Unmarshal(raw, &part); err == nil {
			stores = append(stores, part...)
		}
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
