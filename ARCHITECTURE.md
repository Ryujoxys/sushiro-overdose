# sushiro-overdose 架构分层

项目零外部依赖（纯 Go 标准库），单二进制发布。代码按职责拆分到 `internal/` 下的若干包，根目录只保留 `main.go` 作为入口。

## 包结构

```
main.go                入口：注入版本号 (ldflags) 后调用 internal/app.Run()
internal/
  app/        应用编排 + CLI 命令 + Web 层（最上层，依赖其余所有包）
  core/       共享内核：配置、令牌、时段、偏好、门店、状态、抓包数据结构
  api/        寿司郎 API 客户端（时段查询、预约、排队取号、门店信息）
  proxy/      MITM 抓包代理 + CA 证书生成/信任
  platform/   系统/OS 适配（macOS / Linux / Windows 的代理设置、进程、证书信任）
  notify/     通知渠道（飞书 / Telegram / Bark / Server酱）
```

依赖方向自上而下：`app` → (`api`, `proxy`, `platform`, `notify`, `core`)；`core` 不依赖其他内部包，作为公共底座。`app` 通过点导入（dot import）聚合下层包的导出符号。

## 各包职责

- **app/**
  - 入口与 CLI：`main.go`（`Run()` 命令分发 + `printUsage`）、`calendar.go`、`booking.go`、`daemon.go`、`sampling_cli.go`
  - 后台编排：`engine.go`、`health.go`、`watchdog.go`、`activity.go`
  - 领域与策略：`insights.go`、`recommend.go`、`queue_trends.go`、`queue_alerts.go`、`booking_errors.go`
  - 排队取号：`netticket.go`（旧取号结果兼容，无调度器）、`queue_live.go`、`queue_live_panel.go`
  - 采样与历史：`sampling.go`、`history.go`、`local_records.go`（公开记录查询与导出）
  - Web 层：`web.go`（路由注册）+ `web_*.go`（按功能拆分的 handler：calendar/engine/preferences/sampling/queue_live/queue_trends/events/pac/static）、`web_handlers.go`（通用响应工具 + 首页）
  - 诊断与维护：`diagnostics.go`、`diag_bundle.go`、`auth_probe.go`、`maintenance.go`、`update_check.go`

- **core/**：`config.go`、`tokens.go`、`slot.go`、`preferences.go`、`store.go`、`state.go`、`runtime.go`、`capture.go`、`discovery.go`（接口发现调试）、`paths.go`、`ports.go`、`redact.go`（脱敏）、`util.go`

- **api/**：`api.go` —— 封装官方接口调用与凭证头注入。

- **proxy/**：`proxy.go`（只对寿司郎 API 域名做 TLS 解密，其余 CONNECT 透传）、`cert.go`（CA 证书）。

- **platform/**：`platform.go`（统一接口）+ `platform_{darwin,linux,windows}.go`（平台实现）+ `processes.go`。

- **notify/**：`notifier.go`（`MultiNotifier` 聚合）+ 各渠道 `notifier_*.go`。

## 生命周期边界

- `auth_lifecycle.go` 串行化凭证提交与重置，代次隔离旧捕获和旧探测；保存字段不等于认证有效，验证只调用只读接口。
- 认证引擎启动统一经过 `beginRun`，取消句柄和完成信号同时归属一次运行；停止超时保持 `stopping`，只有拥有该运行的协程才能释放资源。
- 公开采集驱动叫号提醒，不依赖个人凭证或预约采样；GET 查号不修改计划或发送成功通知。
- Windows 代理修改通过 `proxy_transaction.go` 持久化原配置并逐步回滚，不修改机器级 WinHTTP；恢复失败保留备份及 marker。
- `mcp/assets.go` 嵌入 MCP 源码，显式启用时由 `mcp_assets.go` 按内容版本释放。Python 依赖仍需用户机器安装。
- 应用通知统一经过 `notification_delivery.go`，测试内存替身禁止真实桌面通知和外部推送。

## 记录优先与手动取号

默认页面是“我的记录”，选门店、启停服务、查看样本与历史等待分布。“手动取号”单独展示实时数据、用餐偏好和确认；“想几点吃”仅提供建议。

`webui/app.js` 管理记录和只读查询，`auth_ticket.js` 管理认证、确认、单次提交及取消。认证完成必须返回确认框；请求结果不明时先查号，不能重试写请求。所有写请求带 CSRF token。

`webui/record_chart.js` 单独管理叫号/等待曲线、范围带、时段标注、键盘/触屏交互和样本表。两种指标使用各自的单位及有效样本数；叫号字段来自公开堂食叫号，不能用等待时间、排队桌数或预约号推算。

自动预约循环、狙击及每日/定时取号调度器已删除，应用层没有 `CreateReservation` 调用。旧自动 API 在 `automation_retired.go` 返回 410，旧取号文件读取时强制禁用计划；保留已有号码与历史。旧 CLI 自动命令不再启动后台。

`local_records.go` 只读 `queue_baseline.jsonl` 并导出个人快照。`history_bundle.go` 通过 `go:embed` 加载固定 gzip 历史包，校验并缓存，只读且没有网络客户端。`record_view.go` 在查询层按店/日/半小时取最后一条，本机代表值覆盖历史；仅在用户 POST 设置时写入 `record_view.json`。营业状态未知、非营业或等待缺失不进入曲线，缺失时段不补零。原始记录数和个人模型不混入内置历史。

`queue_service.go` 是独立的公开采集生命周期、CLI 和 `/api/queue/service` 控制入口。`--queue-collector-child` 在凭证迁移之前分流，绝不启动认证采样、认证引擎或取号。用户登录自启动沿用旧系统注册项名称以兼容升级，但执行新入口；旧 child 参数也指向纯公开服务。

`queue_collection_state.go` 负责跨进程心跳、热配置、主流程避让和共享采集间隔；`queue_service.lock` 持有整个服务生命周期，`queue_collection.lock` 只锁一次采集。`platform/file_lock*.go` 使用操作系统文件锁，崩溃时自动释放，但不删除锁文件，避免 inode 分裂。JSONL 写入/裁剪、叫号提醒去重也分别持有文件锁。

`queue_model.go` 将本机公开快照转换为持久化统计模型，复用 `QueueBaselineExport` 协议 1。原始文件或节假日表变化、模型超过一天时失效；看板可回退现场聚合，采集成功后更新模型。不读取用户票号/凭证、不触发网络写操作。

## 约定

前端已拆为 `internal/app/webui/index.html`、`app.css`、`auth_ticket.js`、`app.js`，由 `web_static.go` 嵌入；不再修改 Go 字符串中的整页脚本。默认端口为 `39871`，以 `internal/core/ports.go` 为准。前端用 Node 行为测试和可选 Playwright 浏览器检查覆盖认证、确认、防连点、状态切换及响应式。

个人模型的提供者只有本机 JSONL，`queue_baseline_local.go` 生成统一基准信封；历史曲线可选固定离线包，不影响实时/预测 API。`cloud_retired.go` 对旧云端路径返回 410。MCP 只调用本机接口。`collector/` 是独立的历史服务端采集工具，不由桌面端启动。`scripts/export-history-bundle.mjs` 是维护者显式执行的只读导出工具，不加入构建或发布。未来开放边界见 [数据协议](docs/local-data-contract.md)，包质量与更新方法见 [离线历史](docs/bundled-history.md)，线上停服步骤见 [停服清单](docs/cloudflare-github-turso.md)。

- 新 Web API 优先按职责放到 `internal/app/web_*.go`；`web_handlers.go` 只保留通用响应工具和首页。
- 记录服务生命周期放在 `queue_service.go`；`engine.go` 仅负责认证，不得恢复自动预约或取号。
- 可复用算法先在 `core/` 做成纯函数并配测试，再从 Web/CLI 调用。
- 平台相关能力必须通过 `platform/` 暴露，业务代码不直接调用平台命令。
- 用户数据路径统一通过 `core` 的 path helper（`paths.go`），不在业务逻辑中拼硬编码路径。
- `core` 保持无内部依赖；不要让它反向依赖 `app`/`api`/`proxy`。

## Spec-Driven Development

大 feature 必须先写规格再实现，避免 LLM 在代码基准上自由发挥。

- 项目宪法：`.specify/memory/constitution.md`
- Feature 模板：`specs/_template/`
- 当前规格：`specs/NNN-short-name/`

任何触碰预约、排队、取消、公开采集、本地状态、后台进程的改动，都必须在 `spec.md` / `plan.md` / `tasks.md` 中说明：

- 用户可见行为和异常状态
- 官方 API 调用是只读还是 mutation
- 本地状态文件的 writer/reader/清理规则
- 预约和排队是否严格隔离
- 测试和手动验证方式

本地及手动检查工作流中的架构守卫测试会检查高风险边界，例如官方取消 API 只能出现在显式取消 handler 中。新增例外必须先更新项目宪法和对应 spec。自动 push/PR 检查和发布全量测试已停用，发布保留静态检查及 Windows 产物清单校验。

> 更详细的架构说明与打包流程见 [AGENTS.md](AGENTS.md)。
