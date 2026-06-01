# AGENTS.md — LLM 专用项目文档

> 本文件专为 AI 编码助手 (Cursor / Claude / Copilot 等) 编写。  
> 人类开发者请看 `README.md`。

---

## 项目概述

**sushiro-overdose** 是寿司郎 (SUSHIRO) 餐厅的自动预约抢号工具。

核心流程：本地 MITM 代理拦截 PC 微信小程序流量 → 捕获认证参数 → 直连官方 API 轮询/抢号。

**技术栈**：Go 1.23，零外部依赖（纯标准库），单 `package main`。

**目标平台**：macOS (amd64/arm64 Universal)、Windows (amd64/arm64)、Linux (amd64/arm64)。

---

## 架构概览

```
用户双击运行
    │
    ▼
main.go (默认启动 Web UI)
    │
    ├── web.go          HTTP 服务器 127.0.0.1:8081
    │   ├── web_handlers.go   REST API + SSE
    │   └── web_static.go     内嵌 HTML/CSS/JS 单页应用
    │
    ├── engine.go       后台引擎（可从 Web 控制）
    │   ├── 捕获模式: proxy.go + cert.go + platform_*.go
    │   └── 抢号模式: api.go + preferences.go
    │
    └── CLI 子命令 (cli/calendar/sniper/list/cancel/...)
```

### 两种使用模式

1. **Web UI 模式（默认）**：无参数运行 → 启动 HTTP 服务 → 优先打开独立应用窗口，失败时回退默认浏览器
2. **CLI 模式（高级）**：`sushiro-overdose cli` → 传统终端交互

---

## 文件清单与职责

### 核心入口

| 文件 | 职责 |
|------|------|
| `main.go` | 程序入口、CLI 命令分发、前台 CLI 流程、`runBookingLoop` 抢号循环 |
| `daemon.go` | 后台启动/停止/status、守护进程子进程与 PID 读写 |
| `engine.go` | **Web 控制的后台引擎**：管理捕获/抢号生命周期，状态广播到 SSE，可启动/停止 |
| `engine_sniper.go` | Web 狙击计划执行引擎 |

### Web UI

| 文件 | 职责 |
|------|------|
| `web.go` | HTTP 服务器启动，端口冲突自动换端口，Settings 注入 |
| `web_handlers.go` | Web 通用 handler/helper 与首页 |
| `web_calendar.go` | 日历/门店 API |
| `web_engine.go` | 状态、预约、引擎控制、洞察 API |
| `web_preferences.go` | 偏好、通知、repair/uninstall API |
| `web_sniper.go` | Web 狙击计划 API |
| `web_sampling.go` | Web 信息收集 API |
| `web_queue_trends.go` | 本地到店预测 API |
| `web_queue_live.go` | 实时排队 API（公开门店等位/区域/单店详情） |
| `web_events.go` | SSE 事件总线 |
| `web_static.go` | `sushiroLogoSVG` Logo SVG 常量 + `indexHTML` 完整单页（Sushiro 品牌配色 + 官网同款布局） |

### API 与数据

| 文件 | 职责 |
|------|------|
| `api.go` | `Client` — 寿司郎官方 API 封装（门店/时段/创建预约/取消预约） |
| `queue_live.go` | 公开排队接口客户端：门店列表、单店排队、区域列表（标准库实现，支持 `SUSHIRO_TOKEN` 覆盖）；解析 `getStoreById` 的 `groupQueues` 得到当前叫号 |
| `queue_live_panel.go` | 单店实时面板聚合：实时叫号/在等桌数/预估等待 + 由本机采样历史算近15分钟叫号与历史均速 |
| `queue_alerts.go` | 叫号提醒规则与去重状态：`wait_below`（预估等待降到阈值）/`called_reach`（叫号接近手中号），采样循环命中即经通知渠道推送 |
| `config.go` | `Settings` 结构体定义，`LoadSettings` 从 JSON 文件加载（备用，当前未被调用） |
| `tokens.go` | 捕获到的认证参数模型、本地配置读写、旧配置迁移、认证参数 → `Settings` 转换 |
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
| `sniper.go` | 狙击模式：开放前 30 天精准抢号，50ms 高速轮询 |
| `history.go` | `history.jsonl` 追加（节流 30s），`cmdTrends` 趋势分析 |
| `recommend.go` | `cmdRecommend` 基于历史数据的时段推荐 |
| `insights.go` | Web/CLI 可复用的历史洞察：按门店/星期/时段统计开放概率、售罄速度与推荐 |
| `activity.go` | 主流程活动标记与信息收集跨进程锁，确保信息收集避让抢号/捕获/狙击 |
| `queue_trends.go` | 本地排队数据结构、到店预测推荐、过号趋势聚合、节假日分类、信息收集状态提示 |
| `sampling.go` | 后台信息收集配置、运行状态、定时 runner，仅记录历史不抢号 |
| `sampling_cli.go` | `sample` CLI：单次信息收集、前台信息收集、后台静默信息收集 start/stop/status |
| `update_check.go` | GitHub Latest Release 检查与版本比较 |
| `health.go` | 每 5 分钟验证 Token 有效性 |
| `state.go` | `State` JSON 读写，`logMessage`，`readInput` |
| `store.go` | `StoreRegistry` 门店昵称管理 `~/.sushiro/stores.json` |
| `diagnostics.go` | doctor 只读诊断、通知测试、本机网络/证书/端口/代理链路检查 |
| `maintenance.go` | repair-proxy / uninstall 的代理恢复和本地敏感数据清理 |
| `sniper_plan.go` | Web 狙击计划持久化、倒计时、尝试次数与状态摘要 |

### 资源与脚本

| 文件 | 职责 |
|------|------|
| `assets/sushiro.png` | 寿司郎官方 Logo PNG（base64 嵌入到 `web_static.go` 的 `logoBase64` 常量中） |
| `scripts/bundle-macos.sh` | Mac .app + DMG 桌面应用打包脚本 |
| `install/install.sh` | macOS/Linux 一键安装脚本 |
| `install/install.ps1` | Windows PowerShell 一键安装脚本 |

### CI/CD

| 文件 | 职责 |
|------|------|
| `.github/workflows/ci.yml` | 常规 CI：push/PR 运行测试、vet、gofmt、go mod tidy diff、安装脚本语法检查 |
| `.goreleaser.yml` | GoReleaser v2 配置：多平台编译 + Mac Universal Binary |
| `.github/workflows/release.yml` | GitHub Actions：tag 触发 → GoReleaser → Mac .app 打包 → 上传 Release |

---

## 数据文件路径

所有用户数据统一存放在 `~/.sushiro/` 目录：

```
~/.sushiro/
├── config.json          认证参数（X-App-Code, Authorization 等）
├── preferences.json     用户偏好（人数/桌型/目标时段/优先级）
├── notify.json          通知渠道配置
├── stores.json          门店昵称
├── sampling.json        信息收集配置
├── holidays.json        可选节假日/调休工作日本地表
├── history.jsonl        历史时段数据（JSONL 格式）
├── queue_observations.jsonl 实时排队/公开叫号快照（本地私有）
├── queue_sessions.jsonl 真实取号等待 session（本地私有）
├── queue_stats.json     本地聚合排队统计缓存
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

**旧版兼容**：`main.go` 启动时调用 `migrateOldConfig()`，自动将旧版放在当前目录的 `.sushiro_local.json` 迁移到 `~/.sushiro/config.json`。

---

## Web API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | 内嵌 HTML 单页应用 |
| GET | `/api/status` | 版本、运行状态、是否有配置、引擎状态、平台信息 |
| GET | `/api/stores` | 已配置门店列表（含名称/昵称/地址） |
| GET | `/api/calendar?store=ID` 或 `/api/calendar?stores=ID1,ID2&available=1&period=lunch` | 门店时段数据，支持多选、只看可预约、午餐/晚餐过滤 |
| GET | `/api/reservations` | 当前预约列表 |
| GET | `/api/insights` | 历史洞察与推荐 |
| GET | `/api/queue/trends` | 本地到店预测：推荐时段、实际过号、全局过号、信息收集权限与数据新鲜度 |
| GET | `/api/queue/stores?city=深圳&waiting=1&limit=10` | 实时排队门店列表，支持 city/area/q/store/stores/open/waiting/near/limit |
| GET | `/api/queue/store?id=1012` | 实时单店排队详情 |
| GET | `/api/queue/live?store=1012` | 单店实时面板：当前叫号/在等桌数/预估等待/近15分钟叫号/历史均速 |
| GET/POST | `/api/queue/alerts` | 读取/保存叫号提醒规则 |
| GET | `/api/queue/areas` | 官方区域列表 |
| GET/POST | `/api/preferences` | 读取/保存用户偏好 |
| GET/POST | `/api/config` | 读取/保存通知配置 |
| GET | `/api/diagnostics` | 只读、脱敏的本机诊断信息 |
| GET | `/api/update` | 检查 GitHub 最新 Release |
| POST | `/api/notifications/test` | 发送通知渠道测试 |
| POST | `/api/repair-proxy` | 恢复系统代理并清理代理 marker |
| POST | `/api/uninstall` | 清理本地敏感数据和证书 |
| POST | `/api/processes/stop` | 恢复代理并停止本应用相关进程，支持响应后退出当前进程 |
| GET | `/api/engine/state` | 引擎当前状态（idle/capturing/booking/success/error） |
| POST | `/api/engine/capture` | 启动参数捕获（MITM 代理） |
| POST | `/api/engine/booking` | 启动自动抢号 |
| POST | `/api/engine/stop` | 停止当前操作 |
| GET | `/api/engine/logs` | 获取引擎日志 |
| GET/POST | `/api/sniper/plan` | 读取/保存 Web 狙击计划 |
| POST | `/api/sniper/start` | 启动 Web 狙击计划 |
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
go build -o sushiro-overdose .

# 运行（默认打开 Web UI）
./sushiro-overdose

# CLI 模式
./sushiro-overdose cli

# 指定版本号编译
go build -ldflags "-X main.Version=1.2.3" -o sushiro-overdose .

# 交叉编译
GOOS=windows GOARCH=amd64 go build -o sushiro-overdose.exe .
GOOS=linux GOARCH=amd64 go build -o sushiro-overdose .

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
    │   └── sushiro-overdose  (可执行二进制)
    └── Resources/           (预留给图标 .icns)
```

用户双击 .app → macOS 执行 `Contents/MacOS/sushiro-overdose` → 启动 Web UI → 优先打开独立应用窗口，失败时回退默认浏览器。

如需添加应用图标，将 `.icns` 文件放入 `Resources/` 并在 `Info.plist` 中添加 `CFBundleIconFile`。

签名/公证是可选流程：Release workflow 会把 `MACOS_CODESIGN_IDENTITY`、`MACOS_NOTARY_APPLE_ID`、`MACOS_NOTARY_PASSWORD`、`MACOS_NOTARY_TEAM_ID` 传给 `scripts/bundle-macos.sh`。缺少这些 secrets 时会生成未签名 DMG，并在日志中明确跳过签名/公证。

---

## 关键设计决策

### 为什么 Web UI 是默认模式？

大部分用户不熟悉终端操作。Web UI 提供可视化引导，并优先以独立应用窗口承载，降低使用门槛。CLI 保留给高级用户和自动化场景。

### 为什么用内嵌 HTML 而不是前后端分离？

单二进制分发是核心优势。用户下载一个文件就能运行，无需安装 Node.js、npm 等。HTML/CSS/JS 全部内嵌在 `web_static.go` 的 Go 字符串常量中。

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

`web.go` 中的 `findAvailablePort()` 从 8081 开始尝试，冲突则 +1，最多尝试 100 个端口。MITM 捕获代理也从 8080 开始尝试可用端口，并把实际端口写入系统代理和 `proxy_active.json`。避免用户因端口被占用而无法启动或捕获失败。

---

## 编码约定

1. **所有文件在 `package main`**，无子包
2. **跨平台函数**：`platform.go` 导出大写函数 → `platform_*.go` 小写实现
3. **错误处理**：用户可见的错误用中文，内部日志用英文
4. **时间格式**：API 使用紧凑格式（date: `20260413`, time: `193000`），展示时转换为 `2026-04-13`、`19:30`
5. **配置文件**：JSON 格式，`MarshalIndent` 便于人类阅读
6. **Git commit**：中文 commit message，遵循 conventional commits 风格
7. **命名**：发布二进制名 `sushiro-overdose`，所有用户可见文案统一使用此名称

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

前端代码在 `web_static.go` 中：
- `logoBase64` — Logo PNG 的 base64 编码，直接内嵌到 HTML `<img>` 标签
- `indexHTML` — 完整单页 HTML/CSS/JS

CSS 变量定义在 `:root` 块。修改后 `go build` 即生效。

如需更换 Logo，将新 PNG 放到 `assets/`，然后 `base64 -i assets/new-logo.png` 替换 `logoBase64` 的值。

### 添加新的 CLI 命令

1. 在对应文件中添加 `cmdXxx()` 函数
2. 在 `main.go` 的 `main()` 函数中添加分支
3. 在 `printUsage()` 中添加帮助文本

### 修改打包配置

- 添加/移除平台：编辑 `.goreleaser.yml` 的 `goos`/`goarch`/`ignore`
- 修改 Mac .app 元数据：编辑 `scripts/bundle-macos.sh` 中的 `Info.plist`
- 修改 CI 流程：编辑 `.github/workflows/release.yml`
