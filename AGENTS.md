# AGENTS.md — LLM 专用项目文档

> 本文件专为 AI 编码助手 (Cursor / Claude / Copilot 等) 编写。  
> 人类开发者请看 `README.md`。

---

## 项目概述

**sushiro-overdose** 是寿司郎 (SUSHIRO) 的本地排队记录与手动取号工具。

v4.0 的版本名称为「lite正式版」，在应用设置、CLI 和发布标题中展示；数字版本仍独立用于更新比较。GoReleaser 保留 tag 去掉 v 后的版本文本，因此 v4.0 的产物名使用 4.0，不要擅自补成 4.0.0。

默认主流程：选门店 → 开始记录 → 查看自己的排队规律。手动取号独立保留：选店、必要时认证、显式确认、提交一次。所有狙击、定时取号、循环预约与每日自动计划均已移除；旧接口返回 410，旧计划不执行，历史数据保留。“想几点吃”仅提供只读建议。

**技术栈**：Go 1.23，Go 端零外部依赖（纯标准库），已按职责拆到 `internal/`，根目录只保留入口。可选 MCP 为独立 Python 模块。

**目标平台**：macOS (amd64/arm64 Universal)、Windows (amd64/arm64)、Linux (amd64/arm64)。

---

## 架构概览

```
用户双击运行
    │
    ▼
main.go (默认启动 Web UI)
    │
    ├── internal/app/web.go  HTTP 服务器 127.0.0.1:39871
    │   ├── web_*.go         REST API + SSE
    │   └── web_static.go    嵌入 webui/index.html、app.css、auth_ticket.js、record_chart.js、app.js
    │
    ├── queue_service.go    无凭证的公开记录服务
    ├── local_records.go    本机历史聚合与导出
    ├── engine.go           仅管理显式认证和代理清理
    │
    └── CLI 子命令 (collect/status/doctor/...)
```

### 两种使用模式

1. **Web UI 模式（默认）**：无参数运行 → 启动 HTTP 服务 → 优先打开独立应用窗口，失败时回退默认浏览器
2. **记录服务模式**：`sushiro collect run` → 前台公开记录；`collect start` → 独立后台记录

---

## 文件清单与职责

下表为职责索引，未带目录的文件名不代表根目录文件。当前包位置以 `ARCHITECTURE.md` 和 `internal/` 实际文件为准。历史图表使用本机记录和可关闭的固定内置历史包，标注截止时间，包不注入个人模型。GitHub 登录和运行时数据库客户端已移除，旧云端路径统一返回 410。不要根据旧发布文档重新接入共享数据库，协议见 `docs/local-data-contract.md`，离线包见 `docs/bundled-history.md`。

### 核心入口

| 文件 | 职责 |
|------|------|
| `main.go` | 程序入口、CLI 命令分发，旧自动命令只给出移除说明 |
| `daemon.go` | 旧进程停止/status 与 PID 兼容工具，不再启动自动预约 |
| `engine.go` | **认证引擎**：仅管理凭证捕获生命周期、代理恢复及状态广播 |
| `queue_service.go` | 独立公开采集服务、`collect` CLI、显式登录自启动与 `/api/queue/service`；不触碰凭证或订单调度 |
| `queue_collection_state.go` | 桌面与服务共享采集心跳/间隔，OS 文件锁互斥，主流程活动后自动恢复 |

### Web UI

| 文件 | 职责 |
|------|------|
| `web.go` | HTTP 服务器启动，端口冲突自动换端口，Settings 注入 |
| `web_handlers.go` | Web 通用 handler/helper 与首页 |
| `web_calendar.go` | 日历/门店 API |
| `web_engine.go` | 状态、预约、引擎控制、洞察 API |
| `web_preferences.go` | 偏好、通知、repair/uninstall API |
| `auth_import.go` | 手动导入凭证 API：解析手机抓包导出的 JSON/curl/raw headers 并保存凭证参数 |
| `web_sampling.go` | Web 信息收集 API |
| `mobile_auth_capture.go` | 手机凭证捕获 API：局域网引导页 + 手机代理捕获真实微信凭证参数 |
| `web_queue_trends.go` | 本地到店预测 API |
| `web_queue_live.go` | 实时排队 API（公开门店等位/区域/单店详情） |
| `local_records.go` | 本机公开快照查询、等待分布、样本覆盖与 JSONL 导出 |
| `history_bundle.go`、`record_view.go` | 固定历史包嵌入和校验、图表半小时代表值合并、历史开关持久化；不参与取号预测或个人模型 |
| `webui/record_chart.js` | 叫号与等待曲线切换、时段数值标注、范围和样本解释；鼠标、键盘、触屏均可查看 |
| `automation_retired.go` | 旧自动抢号 API 返回 410，不执行或写计划 |
| `netticket.go` | 手动取号结果与旧文件兼容，强制禁用旧计划 |
| `cloud_retired.go` | 已下线的云端 API 统一返回 410，不跳转或保存会话 |
| `web_events.go` | SSE 事件总线 |
| `web_static.go` + `webui/` | 嵌入的页面、样式和脚本，保留 Sushiro 视觉规范 |

### API 与数据

| 文件 | 职责 |
|------|------|
| `api.go` | `Client` — 寿司郎官方 API 封装（门店/时段/创建预约/取消预约） |
| `queue_live.go` | 公开排队接口客户端：门店列表、单店排队、区域列表（标准库实现，支持 `SUSHIRO_TOKEN` 覆盖）；解析 `getStoreById` 的 `groupQueues` 得到当前叫号 |
| `queue_live_panel.go` | 单店实时面板聚合：实时叫号/在等桌数/预估等待 + 由本机采样历史算近15分钟叫号与历史均速 |
| `queue_alerts.go` | 叫号提醒规则与去重状态：`wait_below`（预估等待降到阈值）/`called_reach`（叫号接近手中号），采样循环命中即经通知渠道推送 |
| `queue_baseline_local.go` | 本机公开快照聚合为统一版本化数据协议，无远程数据库请求 |
| `queue_model.go` | 本机分位数统计模型持久化、文件指纹失效与样本覆盖状态，不包含个人凭证/票号 |
| `config.go` | `Settings` 结构体定义，`LoadSettings` 从 JSON 文件加载（备用，当前未被调用） |
| `tokens.go` | 捕获到的凭证参数模型、本地配置读写、旧配置迁移、凭证参数 → `Settings` 转换 |
| `preferences.go` | **用户偏好持久化**：人数/桌型/自定义时段范围/日期与时段优先级，存到 `~/.sushiro/preferences.json` |
| `slot.go` | `Slot`/`StoreInfo`/`ReservationRecord` 数据结构，时间格式化工具 |

### 代理与捕获

| 文件 | 职责 |
|------|------|
| `proxy.go` | MITM 代理服务器、请求解析捕获、门店选择、旧版时段配置 |
| `cert.go` | CA/叶子证书生成，存储路径 `~/.sushiro-proxy/` |
| `watchdog.go` | `proxy_active.json` — 异常退出后清理残留系统代理 |

### 平台适配

| 文件 | 职责 |
|------|------|
| `platform.go` | 跨平台函数转发（大写导出 → 小写平台实现） |
| `platform_darwin.go` | macOS：`networksetup` 代理、`security` 证书、`osascript` 通知 |
| `platform_windows.go` | Windows：注册表代理 + `InternetSetOption` 刷新、`certutil` 证书、PowerShell 通知 |
| `platform_linux.go` | Linux：环境变量 + `gsettings` 代理、系统证书目录、`notify-send` |

### 通知系统

| 文件 | 职责 |
|------|------|
| `notifier.go` | `MultiNotifier` 多通道扇出，`notifyConfig` 读写 `~/.sushiro/notify.json` |
| `notifier_feishu.go` | 飞书 Webhook 卡片通知 |
| `notifier_telegram.go` | Telegram Bot API |
| `notifier_bark.go` | Bark iOS 推送 |
| `notifier_serverchan.go` | Server酱 |
| `notify.go` | `defaultString` 等小工具 |

### 功能模块

| 文件 | 职责 |
|------|------|
| `booking.go` | `cmdList`/`cmdCancel` CLI 命令，`onBookingSuccess` 成功后逻辑（状态/通知/日志） |
| `calendar.go` | `cmdCalendar` 终端日历网格 |
| `history.go` | `history.jsonl` 追加（节流 30s），`cmdTrends` 趋势分析 |
| `recommend.go` | `cmdRecommend` 基于历史数据的时段推荐 |
| `insights.go` | Web/CLI 可复用的历史洞察：按门店/星期/时段统计开放概率、售罄速度与推荐 |
| `activity.go` | 主流程活动标记与信息收集跨进程锁，确保信息收集避让主动认证和手动取号 |
| `queue_trends.go` | 本地排队数据结构、到店预测推荐、过号趋势聚合、节假日分类、信息收集状态提示 |
| `sampling.go` | 后台信息收集配置、运行状态、定时 runner，仅记录历史不抢号 |
| `sampling_cli.go` | `sample once/run` 保留认证时段采集；`start/stop/status/autostart` 兼容转到公开采集服务 |
| `update_check.go` | GitHub Latest Release 检查与版本比较 |
| `health.go` | 每 5 分钟验证 Token 有效性 |
| `state.go` | `State` JSON 读写，`logMessage`，`readInput` |
| `store.go` | `StoreRegistry` 门店昵称管理 `~/.sushiro/stores.json` |
| `diagnostics.go` | doctor 只读诊断、通知测试、本机网络/证书/端口/代理链路检查 |
| `maintenance.go` | repair-proxy / uninstall 的代理恢复和本地敏感数据清理 |

### 资源与脚本

| 文件 | 职责 |
|------|------|
| `assets/sushiro.png` | 寿司郎官方 Logo PNG（base64 嵌入到 `web_static.go` 的 `logoBase64` 常量中） |
| `scripts/bundle-macos.sh` | Mac .app + DMG 桌面应用打包脚本 |
| `cloudflare/sushiro-cloud/` | Worker 停用版本：全部路径返回 410；需要另行发布才能关闭线上入口 |
| `install/install.sh` | macOS/Linux 一键安装脚本 |
| `install/install.ps1` | Windows PowerShell 一键安装脚本 |

### CI/CD

| 文件 | 职责 |
|------|------|
| `.github/workflows/ci.yml` | 手动检查：仅 workflow_dispatch，运行隔离测试、静态检查及跨平台编译，不由 push/PR 自动触发 |
| `.goreleaser.yml` | GoReleaser v2 配置：多平台编译 + Mac Universal Binary |
| `.github/workflows/release.yml` | GitHub Actions：tag 触发 → GoReleaser → Mac .app 打包 → 上传 Release |
| `resource_windows_{amd64,arm64}.syso` | Windows PE 资源（图标 + 干净 application manifest）；`go build` 按 GOARCH 自动链接 |
| `winres/winres.json` + `scripts/gen-windows-resources.sh` | 重新生成上述 syso 的源配置与脚本（发版前若改图标/清单必跑） |
| `assets/windows/app.manifest` | Windows 清单策略说明（SxS：禁止 VC CRT / Common-Controls 依赖） |

---

## 数据文件路径

所有用户数据统一存放在 `~/.sushiro/` 目录：

开发预览可设置绝对路径 `SUSHIRO_DATA_HOME`，应用数据和 CA 分别写入该根目录下的 `.sushiro/`、`.sushiro-proxy/`。不要覆盖 `HOME` / `USERPROFILE`，否则 macOS 钥匙串等系统能力会失效。独立数据目录不迁移旧凭证，也不允许修改系统自启动；它不是系统沙箱，用户显式认证仍需授权证书和代理。

```
~/.sushiro/
├── config.json          凭证参数（X-App-Code, Authorization 等）
├── mobile_ua.json       手机微信 User-Agent（手机凭证捕获/扫码采集后写入）
├── preferences.json     用户偏好（人数/桌型/目标时段/优先级）
├── notify.json          通知渠道配置
├── stores.json          门店昵称
├── sampling.json        信息收集配置
├── cloud_auth.json      遗留文件，已不再读取或写入
├── holidays.json        可选节假日/调休工作日本地表
├── history.jsonl        历史时段数据（JSONL 格式）
├── queue_observations.jsonl 实时排队/公开叫号快照（本地私有）
├── queue_sessions.jsonl 真实取号等待 session（本地私有）
├── queue_stats.json     本地聚合排队统计缓存
├── queue_baseline.json  公开采集配置
├── queue_baseline.jsonl 公开门店快照，统一协议字段
├── queue_model.json     本机分位数模型与源文件指纹
├── record_view.json     是否将内置历史计入图表，默认开启
├── queue_collection_state.json 共享采集进度、心跳与错误
├── queue_service.lock   公开采集服务生命周期 OS 锁
├── queue_collection.lock 单轮公开采集 OS 锁
├── sushiro.log          后台模式日志
├── sampling.log         后台信息收集日志
├── sushiro.pid          后台进程 PID
├── sampling.pid         后台信息收集进程 PID
├── sampling.lock        后台信息收集跨进程互斥锁
├── main_active.json     主流程活动标记（信息收集避让用）
├── .sushiro_state.json  预约状态
└── proxy_active.json    代理活跃标记（watchdog 用）

~/.sushiro-proxy/
├── ca.crt               CA 证书
└── ca.key               CA 私钥
```

**旧版兼容**：常规启动调用 `MigrateOldConfig()`，自动将旧版当前目录的 `.sushiro_local.json` 迁移到 `~/.sushiro/config.json`；公开服务在迁移之前独立分流，不读取个人凭证。

---

## Web API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | 内嵌 HTML 单页应用 |
| GET | `/api/status` | 版本、运行状态、是否有配置、引擎状态、平台信息 |
| GET | `/api/stores` | 已配置门店列表（含名称/昵称/地址） |
| GET | `/api/calendar?store=ID` 或 `/api/calendar?stores=ID1,ID2&available=1&period=lunch` | 门店时段数据，支持多选、只看可预约、午餐/晚餐过滤 |
| GET | `/api/reservations` | 当前预约列表 |
| GET | `/api/records`、`/api/records/export` | 图表分析与个人 JSONL 导出，支持 days=all 或 1..365、store、date_type、date；导出不含内置历史 |
| GET/POST | `/api/records/settings` | 包含历史数据开关，仅 POST 写入本机，GET 不写文件 |
| POST | `/api/queue/ticket` | 用户显式确认后手动取号一次，不自动重试 |
| GET | `/api/queue/ticket/status` | 只读查询当前排队号 |
| POST | `/api/queue/ticket/cancel` | 用户显式确认后取消当前排队号 |
| GET | `/api/insights` | 历史洞察与推荐 |
| GET | `/api/queue/trends` | 本地到店预测：推荐时段、实际过号、全局过号、信息收集权限与数据新鲜度 |
| GET/POST | `/api/queue/service` | 公开采集服务与本机模型状态；显式启用后台、自启动、暂停和关闭自启动 |
| GET | `/api/queue/stores?city=深圳&waiting=1&limit=10` | 实时排队门店列表，支持 city/area/q/store/stores/open/waiting/near/limit |
| GET | `/api/queue/store?id=1012` | 实时单店排队详情 |
| GET | `/api/queue/live?store=1012` | 单店实时面板：当前叫号/在等桌数/预估等待/近15分钟叫号/历史均速 |
| GET/POST | `/api/queue/alerts` | 读取/保存叫号提醒规则 |
| GET | `/api/queue/areas` | 官方区域列表 |
| 任意 | `/api/cloud`、`/api/cloud/*` | 已下线，统一返回 410；不再接受配置、登录或回调 |
| GET/POST | `/api/preferences` | 读取/保存用户偏好 |
| GET/POST | `/api/config` | 读取/保存通知配置 |
| POST | `/api/auth/import` | 手动导入凭证参数，支持 JSON、curl、raw headers |
| POST | `/api/auth/reset` | 停止认证、删除本机凭证并清内存 client；不删除公开记录、不取消号码 |
| GET/POST | `/api/mobile-ua` | 读取/手动保存移动端 UA |
| POST | `/api/mobile-ua/capture/start` | 启动手机扫码 UA 采集页 |
| POST | `/api/mobile-ua/capture/stop` | 停止手机扫码 UA 采集 |
| GET | `/api/mobile-auth` | 手机凭证捕获状态、二维码、局域网引导链接与字段完成度 |
| POST | `/api/mobile-auth/start` | 启动手机凭证捕获：监听局域网代理，只捕获寿司郎凭证参数，不修改 Windows 系统代理 |
| POST | `/api/mobile-auth/stop` | 停止手机凭证捕获 |
| GET | `/api/diagnostics` | 只读、脱敏的本机诊断信息 |
| GET | `/api/update` | 检查 GitHub 最新 Release |
| POST | `/api/notifications/test` | 发送通知渠道测试 |
| POST | `/api/repair-proxy` | 恢复系统代理并清理代理 marker |
| POST | `/api/uninstall` | 清理本地敏感数据和证书 |
| POST | `/api/processes/stop` | 恢复代理并停止本应用相关进程，支持响应后退出当前进程 |
| GET | `/api/engine/state` | 认证状态（idle/capturing/stopping/error） |
| POST | `/api/engine/capture` | 启动参数捕获（MITM 代理） |
| 任意 | `/api/engine/booking`、`/api/sniper/*`、`/api/queue/ticket/plan`、`/api/queue/ticket/routine` | 自动功能已移除，返回 410 |
| POST | `/api/engine/stop` | 停止当前操作 |
| GET | `/api/engine/logs` | 获取引擎日志 |
| GET/POST | `/api/sampling` | 读取/保存后台信息收集配置 |
| POST | `/api/sampling/start` | 启动后台信息收集 |
| POST | `/api/sampling/stop` | 停止后台信息收集 |
| POST | `/api/sampling/once` | 立即信息收集一次 |
| GET/POST | `/api/sampling/autostart` | 查看/配置系统开机自启动信息收集 |
| GET | `/api/events` | SSE 事件流（engine/log/calendar/sampling 事件） |

---

## Web UI 设计规范

### 配色（来自 Sushiro 官网）

| Token | 值 | 用途 |
|-------|-----|------|
| `--red` | `#B81C22` | 主色、按钮、Logo、导航高亮 |
| `--red-dark` | `#A9151A` | 按钮 hover |
| `--bg` | `#F2F2F2` | 页面背景 |
| `--white` | `#FFFFFF` | 卡片背景 |
| `--text` | `#1a1a1a` | 主文字 |
| `--text2` | `#666666` | 辅助文字 |
| `--border` | `#e5e5e5` | 卡片/分隔线 |
| `--green` | `#2d9c4a` | 成功/可用状态 |
| `--yellow` | `#F5BA24` | 警告/进行中 |

### 设计原则

- **Claude 式简约**：大量留白，typography 驱动层级，避免过度装饰
- **Sushiro 品牌一致**：胶囊按钮 `border-radius: 9999px`，卡片圆角 `10px`
- **字体**：PingFang SC → system-ui 回退链
- **响应式**：768px 断点，移动端侧栏变顶栏

---

## 构建指令

### 本地开发

```bash
# 编译
go build -o sushiro .

# 运行（默认打开 Web UI）
./sushiro

# CLI 模式
./sushiro collect run

# 指定版本号编译
go build -ldflags "-X main.Version=1.2.3" -o sushiro .

# 交叉编译
GOOS=windows GOARCH=amd64 go build -o sushiro.exe .
GOOS=linux GOARCH=amd64 go build -o sushiro .

# 代码检查
go vet ./...
```

### 本地测试 GoReleaser（不发布）

```bash
goreleaser release --snapshot --clean
# 产物在 dist/ 目录
```

---

## 发布流程（完整步骤）

### 前置条件

- 代码已合并到 `master`/`main` 分支
- `go build ./...` 和 `go vet ./...` 通过
- 已确认版本号（遵循 semver）
- 写好 `docs/release-notes-<版本>.md`；tag 支持 `v4.0` 或 `v4.0.1`，当前流程只发布稳定版
- 修改版本时同步 `winres/winres.json` 的数值版本并重新生成 syso

发布工作流先创建草稿，再补齐 DMG、Windows 双击版及 8 个文件的 SHA-256 校验值，全部存在后才公开并设为 Latest。GoReleaser 固定为已验证版本，避免工具升级导致产物名或格式变化。

### 步骤

```bash
# 1. 确认代码状态
git status                    # 确保工作区干净
go build ./... && go vet ./... # 编译和静态检查通过

# 2. 打 tag（触发 CI）
git tag v1.2.0
git push origin v1.2.0

# 3. GitHub Actions 自动执行以下流程：
#    a. checkout 代码
#    b. setup Go 1.23
#    c. GoReleaser 编译所有平台（含 Mac Universal Binary）
#    d. 创建 GitHub Release 并上传所有 archive
#    e. 运行 scripts/bundle-macos.sh 创建 Mac .app 并封装 DMG
#    f. 上传 DMG 和 Windows 裸 .exe 到同一个 Release

# 4. 验证 Release
# 打开 https://github.com/Ryujoxys/sushiro-overdose/releases
# 确认以下产物存在：
#   - sushiro-overdose_1.2.0_darwin_all.tar.gz      (Mac Universal Binary)
#   - sushiro-overdose_1.2.0_windows_amd64.zip       (Windows 64位)
#   - sushiro-overdose_1.2.0_windows_arm64.zip        (Windows ARM)
#   - sushiro-overdose_1.2.0_linux_amd64.tar.gz      (Linux 64位)
#   - sushiro-overdose_1.2.0_linux_arm64.tar.gz      (Linux ARM)
#   - Sushiro-Overdose-1.2.0-macOS.dmg              (Mac 双击安装镜像)
#   - Sushiro-Overdose-1.2.0-windows-amd64.exe      (Windows 双击运行)
#   - Sushiro-Overdose-1.2.0-windows-arm64.exe      (Windows ARM 双击运行)
#   - checksums.txt
```

### Release 产物说明

| 文件 | 目标用户 | 使用方式 |
|------|---------|---------|
| `*_darwin_all.tar.gz` | Mac 高级用户 | 解压后命令行运行 |
| `Sushiro-Overdose-*-macOS.dmg` | Mac 普通用户 | 双击打开，拖到 Applications 后运行，独立窗口优先 |
| `Sushiro-Overdose-*-windows-amd64.exe` | Windows 用户 | 下载后双击运行，GUI 子系统无终端黑框 |
| `Sushiro-Overdose-*-windows-arm64.exe` | Windows ARM 用户 | 下载后双击运行，GUI 子系统无终端黑框 |
| `*_windows_amd64.zip` | Windows 高级用户 | 解压后命令行运行 |
| `*_windows_arm64.zip` | Windows ARM 高级用户 | 同上 |
| `*_linux_amd64.tar.gz` | Linux 用户 | 解压后命令行运行 |
| `*_linux_arm64.tar.gz` | Linux ARM 用户 | 同上 |

### Windows 打包注意（并行配置 / SxS）

用户侧报错「**应用程序无法启动，因为应用程序的并行配置不正确**」属于 Windows loader / SxS，发生在进程入口之前，与寿司郎业务逻辑无关。

打包侧硬规则：

1. **双击版**（`Sushiro-Overdose-*-windows-*.exe`）用  
   `CGO_ENABLED=0` + `-H windowsgui` + 根目录 `resource_windows_{amd64,arm64}.syso`。
2. **syso 必须**含：图标 + `asInvoker` + PerMonitorV2；**禁止**声明  
   `Microsoft.VC*.CRT` 或 `Microsoft.Windows.Common-Controls`（纯 Go 不需要，声明了反而在坏 WinSxS 机器上直接起不来）。
3. 改图标 / 清单后必须：

```bash
./scripts/gen-windows-resources.sh
go test ./internal/app/ -run TestWindowsResourceSysoHasCleanManifest
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
  -ldflags "-s -w -H windowsgui -X main.Version=dev" \
  -o /tmp/Sushiro-Overdose-dev-windows-amd64.exe .
```

4. Release CI 在上传 Windows 双击版前会跑 `TestWindowsResourceSysoHasCleanManifest`，并拒绝过小的 exe。
5. 控制台 zip 版（GoReleaser 的 `sushiro-overdose_*_windows_*.zip`）**不**加 `-H windowsgui`，可作「双击版起不来」时的对照包。

### 热修复发布

```bash
# 在 master 上修复 bug
git commit -m "fix: 修复某某问题"
git push

# 打 patch 版本
git tag v1.2.1
git push origin v1.2.1
# CI 自动发布
```

### 删除错误的 Release

```bash
# 删除远程 tag
git push origin :refs/tags/v1.2.0
# 在 GitHub Release 页面手动删除对应 Release
# 修复后重新打 tag
git tag v1.2.0
git push origin v1.2.0
```

---

## Mac .app 打包细节

`scripts/bundle-macos.sh` 的工作流程：

```
输入: 编译好的二进制 + 版本号
输出: "Sushiro-Overdose-1.2.0-macOS.dmg"

目录结构:
Sushiro Overdose.app/
└── Contents/
    ├── Info.plist          (应用元数据: 名称/版本/Bundle ID)
    ├── MacOS/
    │   └── sushiro          (可执行二进制)
    └── Resources/           (预留给图标 .icns)
```

用户双击 .app → macOS 执行 `Contents/MacOS/sushiro` → 启动 Web UI → 优先打开独立应用窗口，失败时回退默认浏览器。

如需添加应用图标，将 `.icns` 文件放入 `Resources/` 并在 `Info.plist` 中添加 `CFBundleIconFile`。

签名/公证是可选流程：Release workflow 会把 `MACOS_CODESIGN_IDENTITY`、`MACOS_NOTARY_APPLE_ID`、`MACOS_NOTARY_PASSWORD`、`MACOS_NOTARY_TEAM_ID` 传给 `scripts/bundle-macos.sh`。缺少这些 secrets 时会生成未签名 DMG，并在日志中明确跳过签名/公证。

---

## 关键设计决策

### 为什么 Web UI 是默认模式？

大部分用户不熟悉终端操作。Web UI 提供可视化引导，并优先以独立应用窗口承载，降低使用门槛。CLI 保留给高级用户和自动化场景。

### 为什么用内嵌 HTML 而不是前后端分离？

单二进制分发是核心优势。用户下载一个文件就能运行主程序，无需安装 Node.js、npm 等。HTML/CSS/JS 在 `internal/app/webui/` 中维护，通过 `web_static.go` 嵌入二进制。

### 为什么零外部 Go 依赖？

- 编译速度快
- 二进制体积小（约 8-10MB）
- 无供应链攻击风险
- Go 标准库的 `crypto/tls`、`net/http` 已足够实现 MITM 代理
- 代理只对寿司郎 API 域名做 TLS 解密；其他 HTTPS 域名保持 CONNECT 透传，不读取或解密内容

### 配置文件为什么在 ~/.sushiro/？

之前放在当前工作目录（CWD），用户换个目录就找不到配置。统一到 `~/.sushiro/` 后：
- Windows 用户双击 .exe 不用担心 CWD
- Mac .app 运行时 CWD 不确定
- 多终端窗口共享同一份配置

### 端口冲突处理

Web 与 MITM 的默认端口以 `internal/core/ports.go` 为准，Web 从 39871 开始尝试可用端口。代理实际端口会写入系统代理和 `proxy_active.json`，不能在页面或客户端硬编码实际监听端口。

---

## 编码约定

自动测试不得弹出真实系统通知或发送外部推送。应用层通知统一经过 `notification_delivery.go`，测试由 `TestMain` 替换为内存通知器并隔离用户目录；不得绕过该边界。保留通知状态、通道路由和去重测试。按维护者要求，检查工作流仅手动触发，发布不跑全量测试，但保留 Windows 产物清单校验；本地或手动执行测试同样必须隔离副作用。

1. **代码按职责分包到 `internal/`**：`app`（编排+CLI+Web）依赖 `core`/`api`/`proxy`/`platform`/`notify`；`core` 为无内部依赖的公共底座。详见 [ARCHITECTURE.md](ARCHITECTURE.md)
2. **跨平台函数**：`internal/platform/platform.go` 导出大写函数 → `platform_*.go` 小写实现
3. **错误处理**：用户可见的错误用中文，内部日志用英文
4. **时间格式**：API 使用紧凑格式（date: `20260413`, time: `193000`），展示时转换为 `2026-04-13`、`19:30`
5. **配置文件**：JSON 格式，`MarshalIndent` 便于人类阅读
6. **Git commit**：中文 commit message，遵循 conventional commits 风格
7. **命名**：可执行/命令名为 `sushiro`（短命令）；仓库名、Go module、release 压缩包前缀（`sushiro-overdose_*.tar.gz`）及 DMG/exe 产品名（`Sushiro-Overdose-*`）保留 `sushiro-overdose` 不变

---

## 常见修改场景

### 添加新的通知渠道

1. 创建 `notifier_xxx.go`，实现 `Notifier` 接口（`Send` + `Name`）
2. 在 `notifier.go` 的 `notifyConfig` 中添加字段
3. 在 `BuildNotifierFromConfig()` 中添加初始化逻辑
4. 在 `web_handlers.go` 的 `handleNotifyConfig` 中无需改动（自动序列化）
5. 在 `web_static.go` 的设置页面添加对应输入框
6. 在 `main.go` 的 `cmdConfig` 中添加 CLI 配置命令

### 添加新的 Web API

1. 在 `web_handlers.go` 中添加 handler 函数
2. 在 `web.go` 的 `cmdWeb()` 中注册路由 `mux.HandleFunc("/api/xxx", handleXxx)`
3. 前端在 `web_static.go` 中调用

### 修改 Web UI 样式

前端代码在 `internal/app/webui/index.html`、`app.css`、`app.js` 中；`web_static.go` 负责嵌入和注入。CSS 变量定义在 `app.css` 的 `:root` 块。修改后 `go build` 即生效。

如需更换 Logo，将新 PNG 放到 `assets/`，然后 `base64 -i assets/new-logo.png` 替换 `logoBase64` 的值。

### 添加新的 CLI 命令

1. 在对应文件中添加 `cmdXxx()` 函数
2. 在 `main.go` 的 `main()` 函数中添加分支
3. 在 `printUsage()` 中添加帮助文本

### 修改打包配置

- 添加/移除平台：编辑 `.goreleaser.yml` 的 `goos`/`goarch`/`ignore`
- 修改 Mac .app 元数据：编辑 `scripts/bundle-macos.sh` 中的 `Info.plist`
- 修改 CI 流程：编辑 `.github/workflows/release.yml`
