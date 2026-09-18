# Sushiro Spec Constitution

本文件是 `sushiro-overdose` 的规格驱动开发宪法。所有较大 feature 必须先写 spec，再写 plan/tasks，再改代码。LLM 可以实现任务，但不能绕过这些边界。

## Non-Negotiable Principles

1. 只读入口不得产生远端副作用。
   - `GET` handler、自检、诊断、看板、采集、趋势接口不得调用预约、取号、取消预约、取消排队接口。
   - 允许只读入口修正本地过期缓存，但必须只影响 `~/.sushiro/` 本地文件，不能触发官方 mutation API。

2. 预约和排队必须分层隔离。
   - 不再创建预约；旧预约查询与显式取消保留兼容。
   - 排队取号使用 `CreateNetTicket` / `CancelNetTicket`。
   - UI 和后端都必须带 `kind` 或独立入口，禁止用同一个取消接口模糊处理两类记录。
   - 允许的预约取消入口只有显式 Web `handleCancelReservation` 和 CLI `cmdCancel`。
   - 允许的排队取消入口只有显式 Web `handleCancelNetTicket`。
   - 允许的排队取号入口只有 `handleQueueTicket`（内部 `takeNetTicketLocked`）。凭证验证只能只读，不能以验证为由创建或取消排队号。

3. 公开排队数据和认证预约数据必须分层隔离。
   - 公开接口可以用于看板、趋势、排队估算、门店状态。
   - 认证接口才可用于预约提交、排队取号、当前票状态。
   - 公开接口数据不能被当作官方预约状态。

4. 本地缓存只能兜底展示，不能覆盖官方真实状态。
   - `~/.sushiro/.sushiro_state.json` 只能保存本地已知预约/排队记录。
   - 官方当前预约接口不可用时，UI 必须明确提示“不代表小程序没有预约”。
   - 过期排队号必须按日期或官方 no-current-ticket 信号清理。

5. 采集和看板不能影响主动认证与手动取号。
   - 采集只能写历史、基准、趋势数据。
   - 主流程活跃时，采集必须避让。
   - 看板计算不能写预约状态、排队状态、取消状态。

6. 官方业务错误必须分类，不允许只刷“失败”。
   - 名额已满：提示用户，不自动尝试其他目标。
   - 已有号码：查询并展示，不再取第二个号。
   - 官方临时错误或结果不明：先查询已有号码，不自动重试写请求。
   - 认证过期：停止并要求重新捕获。

7. 任何跨模块新增能力必须有测试。
   - API payload/response 解析要有单元测试。
   - 状态机分支要有状态测试。
   - 取消/取号/预约边界要有安全测试。
   - 前端关键 POST 要经过 CSRF 包装或显式 header。

## Architecture Ownership

| Domain | Owner Files | May Write | Must Not Do |
|---|---|---|---|
| Legacy reservations | `internal/app/booking.go`, `internal/app/web_engine.go` | explicit query/cancel results | create reservations or cancel queue ticket |
| Manual queue ticket | `internal/app/netticket.go`, `internal/app/web_engine.go`, `internal/api/api.go` | net ticket plan/state | cancel reservation |
| Public queue live data | `internal/app/queue_live*.go` | observations/baseline | call auth mutation API |
| Sampling | `internal/app/sampling.go`, `internal/app/history.go` | history/baseline/observations | modify reservation or net ticket state |
| Public local service | `internal/app/queue_service.go`, `queue_collection_state.go`, `queue_model.go` | public history/model/heartbeat, explicit autostart | import credentials, start account schedulers, submit tickets |
| Dashboard/trends | `internal/app/queue_dashboard.go`, `internal/app/queue_trends.go` | computed responses/cache | mutate official or local booking state |
| Web UI | `internal/app/web_static.go`, `internal/app/web_*.go` | display and explicit user actions | hidden destructive actions |

## Spec Lifecycle

1. `spec.md` defines user outcome, scope, non-goals, states, and acceptance criteria.
2. `plan.md` defines affected modules, data contracts, state transitions, risks, and tests.
3. `tasks.md` breaks work into verifiable implementation steps.
4. Implementation must cite the task being changed in commit notes or PR summary.
5. Any architecture exception must be written in the feature `plan.md` before code changes.

## Review Checklist

- Does this change introduce a new official API call?
- Is the call read-only or mutating?
- Which local files can this feature write?
- Can the feature accidentally cancel or create something?
- What happens if the official API is stale, 404, 409, or 500?
- What happens if the user changed state from the phone app?
- What test proves the above?
