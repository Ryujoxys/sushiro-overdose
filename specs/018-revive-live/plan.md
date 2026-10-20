# Implementation Plan: 实时数据恢复与离线可用

依据：`spec.md`；现状调研见 `docs/cloudflare-github-turso.md`。新网关事实来自第三方实测记录（getStoreById 直接返回旧字段的门店对象；stores 固定参数返回数组；六个上下文头；JWT 约 3600 秒）。实现时对响应做防御性解码，任何不符都按错误处理，不写零值记录。

## 宪章例外（必须先写在这里）

| 规则 | 例外 | 约束 |
|---|---|---|
| 公开本地服务不得导入凭证（Public local service 行） | 新增 **Live session** 领域：`internal/app/live_session.go` 持有用户自己的网关会员 token，仅用于只读 `getStoreById` / `stores` | 只有界面进程在启动时把读取函数注入 `queueLiveSessionProvider`；独立后台进程（`collect run/start`、`--queue-collector-child`）从不设置它，`queue_service.go` / `queue_collection_state.go` / `queue_model.go` 不出现 `LoadLiveSession` / `LoadLocalConfig`。token 不写日志、不出现在任何 HTTP 响应、不离开本机，只发往 `sapi.sushiro.com.cn` |
| 内置历史不参与取号预测（AGENTS.md、docs/bundled-history.md） | 允许内置历史为“想几点吃”和“叫到时间”提供历史规律与叫号速度 | 每个结果标明来源和截止日期；本机记录较新时优先；不写回个人模型 |

宪章 Architecture Ownership 增加一行：`Live session | live_session.go, web_live_session.go | ~/.sushiro/live_session.json | 被独立后台读取、发往网关以外的主机、调用取号/取消`。

## 模块与归属（并行实施时互不修改对方文件）

| 单元 | 文件 | 内容 |
|---|---|---|
| A 连接与采集 | `live_session.go`(新)、`web_live_session.go`(新)、`queue_live.go`、`queue_baseline.go`、`queue_collection_state.go`、`queue_service.go`、`web.go`、`desktop_bridge.go`、`diagnostics.go` 及对应测试 | 会员连接、网关客户端、解码加固、采集熔断、共享写入器 |
| B 离线建议 | `queue_offline.go`(新)、`queue_advisor.go`、`queue_trends.go`、`web_queue_live.go`、`queue_live_panel.go`、`history_bundle.go`、`record_view.go`(只读复用) 及测试；路由与桥白名单的两行由合并时补 | 叫号速度推算、历史等待规律兜底、离线门店目录、手动记一笔 |
| C 抓包与取号安全 | `internal/proxy/*`、`engine.go`、`mobile_auth_capture.go`、`auth_import.go`、`auth_probe.go`、`web_engine.go`、`web_pac.go`、`platform_windows.go` 中 QUIC 部分 | 代理解密 sapi、捕获会员连接、被动记录、取号失败即停 |
| D 界面 | `webui/*`、`scripts/test-webui-records.mjs`、`web_static_test.go` | 连接卡片、离线输入、叫到时间、文案 |
| E 文档与技能 | AGENTS.md、README、docs、宪章、`.claude/skills/sushiro-queue` | 规则变更、`eta --called` |

## 已落地的共享接口（基线提交）

- `ErrLiveAuthRequired`（queue_live.go）：无可用连接，本地返回，不发请求。
- `QueueLiveStore.ObservedAt`、`QueueLiveStore.Source`（`json:"-"`）：写入方用 ObservedAt（零值=现在）；Source ∈ gateway / legacy / passive / manual。
- `liveErrorCode(err)`、`writeLiveError(w, err)`、`writeErrorCode(w, status, code, msg)`；code ∈ `live_auth_required` / `live_rate_limited` / `live_unavailable`，HTTP 仍为 502。
- `friendlyPublicQueueError`：401/403/451 与 `ErrLiveAuthRequired` 一律提示“打开小程序刷新一下，再在应用里连接”，不再提 VPN 或状态码。

## A：会员连接与网关客户端

```go
// live_session.go —— ~/.sushiro/live_session.json，0600，原子写入
type LiveSession struct {
    Authorization string    `json:"authorization"`   // "Bearer <jwt>"
    AppClient     string    `json:"app_client"`
    AppCode       string    `json:"app_code"`
    UserAgent     string    `json:"user_agent"`
    Referer       string    `json:"referer"`
    ContentType   string    `json:"content_type"`
    IssuedAt      time.Time `json:"issued_at,omitempty"`  // JWT iat，未验签
    ExpiresAt     time.Time `json:"expires_at,omitempty"` // JWT exp，未验签
    CapturedAt    time.Time `json:"captured_at"`
    Source        string    `json:"source"`               // import | capture
}
func LoadLiveSession() (*LiveSession, error)           // 不存在返回 nil, nil
func SaveLiveSession(s LiveSession) (bool, error)      // 只在比已存的更新（ExpiresAt 或 CapturedAt 更晚）时写入；返回是否写入
func ClearLiveSession() error
func (s LiveSession) Usable(now time.Time) bool        // 有 exp：now < exp-30s；无 exp：now < CapturedAt+55m
func (s LiveSession) Remaining(now time.Time) time.Duration
func LiveSessionFromHeaders(h http.Header, now time.Time, source string) (LiveSession, error) // 必须有 Bearer authorization、x-app-client、x-app-code；UA/Referer/Content-Type 缺省时用已知小程序值
func ParseLiveSessionText(text string, now time.Time) (LiveSession, error) // curl / 原始请求头 / HAR；只接受 sapi.sushiro.com.cn 且路径含 /gateway/ 的请求；HAR 取最新的一条
```

- 状态接口 `GET /api/live-session` → `{state: none|active|expiring|expired, connected, expires_at, remaining_seconds, captured_at, source}`，从不返回头部的值；`expiring` = 剩余 < 10 分钟。
- `POST /api/live-session/import {text}` → 同上状态，或 400 中文错误（“没找到寿司郎新接口的请求”“这条请求已经过期”等）。
- `POST /api/live-session/clear`。三者加入桌面桥白名单（GET / POST / POST），走 CSRF。
- `queueLiveSessionProvider func() (*LiveSession, error)`：包级变量，默认 nil；`web.go` 的界面进程启动时设为 `LoadLiveSession`。
- `NewQueueLiveClient()` 选择顺序：provider 返回可用连接 → 网关客户端（base `https://sapi.sushiro.com.cn/gateway/wechat/api/2.0`，六个头，`Accept: */*`，Source=gateway）；否则若设置了 `SUSHIRO_TOKEN` → 旧地址（Source=legacy，供调试）；否则返回“需要连接”的客户端，所有方法立即返回 `ErrLiveAuthRequired`，零请求。8 个调用点不改。
- 网关 `ListStores`：`stores?latitude=1&longitude=1&numresults=10000`，缓存沿用；`ListAreas`：网关无已知路径，用门店列表的 `area` 去重得到。
- 解码加固：`id` 接受数字或数字字符串；`groupQueues` 各队列接受字符串或字符串数组；`GetStore` 要求返回的 id 等于请求的 id 且 `storeStatus` 非空，否则报“门店数据格式不对”；列表对象里找不到数组时报错；请求失败或解码失败都不写记录。
- 采集熔断（状态写在 `queue_collection_state.json`，界面与后台共享）：`last_error_code`、`consecutive_failures`、`next_attempt_at`、`waiting_for_connection`、`session_mark`（连接的 captured_at，用于判断换了新连接）。
  - `live_auth_required`：停止，直到“本进程拿到的连接 session_mark 与停止时不同”或 6 小时后；没有连接的进程从不发请求。持有可用连接的进程不受“无连接进程”写入的停止标记影响。
  - 其他失败：退避 30s→1m→2m→4m→8m→15m 封顶；成功清零。
  - 状态接口新增 `last_error_code`、`next_attempt_at`、`waiting_for_connection`；不再写“会自动重试”。
- 共享写入器（queue_baseline.go）：`recordQueueLiveSnapshots(ctx, stores []QueueLiveStore) (int, error)`：逐条写基线与观测（时间用 ObservedAt）、评估提醒，最后重建一次模型；`Source == "manual"` 只写观测、不写基线。
- 网关请求频率：采集间隔不低于 60 秒；实时面板复用 30 秒缓存，避免一次页面刷新打多次。
- `diagnostics.go`：网络检查同时列出两个主机，说明旧主机可达不代表能用。

## B：离线建议

- `queue_offline.go`：
  - `offlineSpeedProfile(storeID string, now time.Time)`：来自内置历史与本机观测的叫号速度（号/分钟），按门店 × 日期类型 × 半小时；同一天相邻两个半小时代表值都营业且叫号已知、增量 ≥ 0 才计入；10% 截尾均值，外加 P20/P80；不足 3 天依次回退到同店全部日期类型、全部门店中位数。本机观测比内置历史新时优先。
  - `estimateCalledAt(storeID string, myNo, calledNo int, calledAt time.Time, now time.Time) QueueAdvisorEta`：从 calledAt 起按分钟累加速度直到 myNo；用 P80/P20 速度给早/晚区间；返回现有 `QueueAdvisorEta` 形状，`Source = "history_speed"`，附 `estimated_called_at_hhmm`、`range_hhmm`（HH:MM）和 `basis`（“按这家店以往的叫号速度推算（历史截至 2026-09-06）”）。
  - 回测测试：以 2026-08-15 前的内置历史为训练、之后为测试，1 小时提前量平均绝对误差 ≤ 20 分钟。
  - `offlineStoreDirectory()`：内置历史门店 + 本机记录里的门店 + 门店昵称；`area` 作为区域，城市取 `area` 中“区/县/市”前的城市名或直接用于包含匹配；给 `/api/queue/stores`、`/api/queue/areas`、`/api/stores` 名称回退使用，标 `offline: true`。
- `/api/queue/advisor` 新增参数 `called_no`、`called_at`（可选，默认现在）：有 called_no 时不请求实时；实时失败且有 30 分钟内的手动记录时用它；都没有时返回 200 `{needs_called_no: true, message}`，不再 502。所有时间范围同时给 HH:MM。
- “想几点吃”：本机等待记录不足时用内置历史等待 P50/P80（按目标时刻的日期类型，含零等待）；候选取号时刻不早于现在；`travel_minutes` 从请求透传；实时失败时立即用历史，不等 8 秒；`liveWaitEstimate` 失败文案不再说“可能门店未开放线上取号”。
- `POST /api/queue/reading {store_id, called_no, wait_groups?}`：写一条 `source_endpoint = manual` 的观测（等待记为未知：所有读观测等待的地方跳过负值），并评估叫号提醒；返回 `{ok, observed_at}`。加入桥白名单。
- 观测的等待未知：`QueueObservation` 中等待 < 0 表示未知；压力、等待分布、`wait_below` 提醒跳过它。

## C：抓包与取号安全

- 代理解密主机集合 `{crm-cn-prd.sushiro.com.cn, sapi.sushiro.com.cn}`（精确匹配）；PAC、Windows QUIC 拦截、证书签发同步；`shouldBufferSushiroAPIResponse` 加 `sapi` 的 `/gateway/wechat/api/2.0/getStoreById` 与 `/gateway/wechat/api/2.0/stores`。
- `ProxyOptions.OnGatewayExchange func(req *http.Request, status int, body []byte)`：界面进程注入处理函数：(1) `LiveSessionFromHeaders` → `SaveLiveSession`（新的才写）；(2) `getStoreById` 且 200 → 解码为 `QueueLiveStore{Source: "passive", ObservedAt: now}`，id 与请求一致才 `recordQueueLiveSnapshots`。响应体不落盘。
- 手机 / 电脑捕获流程：拿到网关连接即视为“实时数据已连接”，状态接口给出；原有取号凭证的完成条件不变。引导文案改为过滤 `sapi.sushiro.com.cn`。
- 取号安全：只读验证不因公开路径 401/403 把账号标成失效；当前号码查询返回无法识别的内容时不标记“健康”、不当成“没有号”；取号前查询现有号码失败时拒绝提交，提示“查不到你现在有没有号，为避免重复取号，请在小程序里取号”。

## D：界面

- 设置 → 账号卡片下新增“实时数据”：状态（已连接，剩 N 分钟 / 未连接 / 已到期）、“用手机连接”（复用现有手机捕获）、“粘贴抓包”（文本框 → import）、“断开”。说明：连接用的是你自己的小程序登录，约 1 小时有效，只在这台电脑上用。
- 排队页：`code == live_auth_required` 时显示连接提示；同时显示“小程序里现在叫到 [ ] 号”“我的号码 [ ]”→ 调 advisor 得到 HH:MM 区间；“记一笔”调 reading。没有实时数据时“想几点吃”照样可用。
- 门店选择：`/api/queue/stores` 失败或 `offline` 时照样可选，修正 `nameKana` 键名。
- 记录状态：`waiting_for_connection` 时写“后台记录需要连接实时数据（应用开着时记录）”，不写“会自动重试”。
- 取号按钮：没有实时数据时不可用，旁边说明“先连接实时数据，或在小程序里取号后填号码看叫到时间”。
- 叫到时间显示 HH:MM，测试模拟真实 RFC3339 返回。

## 测试

| 测试 | 证明 |
|---|---|
| live_session 解析 curl / 头 / HAR、拒绝旧主机与过期、只升级更新的连接、0600 | 连接只来自用户自己的网关请求 |
| 网关客户端用 httptest：六个头、id 字符串、信封 / 空对象报错 | 不写零值记录 |
| 无连接零请求；401 后熔断不重试；换新连接后恢复；无连接进程不阻塞有连接进程 | 不撞接口 |
| 回测 1 小时 MAE ≤ 20 分钟；advisor called_no 路径；needs_called_no | 离线叫到时间可用 |
| 代理只解密两个精确主机；被动记录 id 不符不写；OnGatewayExchange 保存连接 | 抓包范围不扩大 |
| 取号前状态查询失败拒绝提交 | 不重复取号 |
| 独立后台文件里没有 LoadLiveSession / LoadLocalConfig | 宪章边界 |
