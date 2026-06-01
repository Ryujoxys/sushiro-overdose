# SUSHIRO 寿司郎 Overdose

全自动抢号的寿司郎预约工具。通过本地 MITM 代理捕获微信小程序认证参数，然后自动轮询预约目标时段。

零外部依赖，纯 Go 标准库实现。支持 macOS / Windows / Linux。

**双击运行，独立应用窗口操作，不需要任何编程知识。**

---

## 下载安装

### Windows 用户（推荐：一键安装）

任选一种方式：

**方式 A：双击 install.bat（最简单）**

1. 下载 [install.bat](https://raw.githubusercontent.com/Ryujoxys/sushiro-overdose/master/install.bat)（右键「另存为」）
2. 双击运行，自动完成下载、解压、加入 PATH、创建桌面快捷方式
3. 安装完成后，双击桌面「Sushiro Overdose」图标即可使用

**方式 B：PowerShell 一行命令**

打开 PowerShell（按 `Win+X` → 选「终端」或「PowerShell」），粘贴执行：

```powershell
irm https://raw.githubusercontent.com/Ryujoxys/sushiro-overdose/master/install/install.ps1 | iex
```

**方式 C：手动下载 .exe**

从 [GitHub Releases](https://github.com/Ryujoxys/sushiro-overdose/releases) 下载 `Sushiro-Overdose-*-windows-amd64.exe`，双击即可。ARM 设备下载 `Sushiro-Overdose-*-windows-arm64.exe`。

> Windows 注意事项：
> - 首次运行 SmartScreen 可能弹窗提示「Windows 已保护你的电脑」，点击「更多信息」→「仍要运行」即可。
> - 推荐下载 `Sushiro-Overdose-*.exe` 直下版，双击会打开独立应用窗口且不显示终端黑框。压缩包内的 `sushiro-overdose.exe` 保留给高级用户命令行使用。
> - 程序会自动安装一张本地 MITM 证书并临时设置系统代理，**退出时自动恢复**。
> - 如杀毒软件误报，请将 `%LOCALAPPDATA%\sushiro\sushiro-overdose.exe` 加入白名单。
> - 抓包阶段只解密寿司郎 API 域名；其他站点流量保持 CONNECT 透传，不读取或解密内容。

### macOS / Linux 用户

| 平台 | 下载文件 | 使用方式 |
|------|---------|---------|
| **macOS** | `Sushiro-Overdose-*-macOS.dmg` | 双击打开 DMG，将 App 拖到 Applications 后运行，独立窗口优先 |
| **Linux** | `sushiro-overdose_*_linux_amd64.tar.gz` | 解压后终端运行 |

> macOS 注意事项：当前 Release 默认未签名/未公证。首次打开如果提示无法验证开发者，请在「系统设置 → 隐私与安全性」中允许打开，或右键 App 选择「打开」。

也可使用一键脚本：

```bash
curl -fsSL https://raw.githubusercontent.com/Ryujoxys/sushiro-overdose/master/install/install.sh | bash
```

### 从源码构建

```bash
git clone https://github.com/Ryujoxys/sushiro-overdose.git
cd sushiro-overdose
go build -o sushiro-overdose .       # macOS / Linux
# Windows (PowerShell):
# go build -o sushiro-overdose.exe .
```

---

## 使用教程

### 第一次使用

1. **运行程序** → 自动打开独立应用窗口（无法打开时回退浏览器）
2. **设置向导** → 安装证书（按提示确认即可）
3. **捕获参数** → 在 PC 微信中打开寿司郎小程序，进行一次排队/预约操作
4. **信息收集** → 先选择关心的门店和时间窗，积累到店预测数据
5. **设置预约优先级** → 选择人数、桌型、目标时段、门店顺序
6. **开始抢号** → 点击「开始抢号」按钮

> **注意：** 必须使用 **PC 版微信** 中的小程序，手机端无效。

### 日常使用

认证参数捕获后自动保存，下次运行无需重复捕获（过期时会自动提示）。

打开程序 → 先看「到店预测」或预约日历 → 需要预约时点击「开始抢号」→ 成功后自动通知。

### 预约优先级

Web UI 的「设置」页面支持明确控制选号策略：

- **日期优先级**：按日期优先、周末优先、工作日优先
- **时段策略**：最早可约、最晚可约、接近目标时间
- **目标时间**：当选择「接近目标时间」时使用，例如 `1930`
- **每日时段范围**：工作日、周六、周日可分别配置多个时间段

同一天同一时间有多个门店时，会按已选择门店的优先顺序尝试。

---

## 全部命令

程序默认启动 Web UI。高级用户可使用命令行：

```
sushiro-overdose                 启动 Web UI（默认，推荐，独立窗口优先）
sushiro-overdose cli             终端交互模式（高级）
sushiro-overdose start           后台静默运行
sushiro-overdose status          查看运行状态
sushiro-overdose exit            停止后台进程

sushiro-overdose calendar        查看近 7 天可预约时段
sushiro-overdose sniper          狙击模式 - 提前锁定未开放时段
sushiro-overdose list            查看当前预约
sushiro-overdose cancel <id>     取消预约

sushiro-overdose trends          分析时段可用率趋势
sushiro-overdose recommend       智能推荐最佳时段
sushiro-overdose sample once     信息收集一次，写入本地历史数据
sushiro-overdose sample run      前台持续信息收集，不抢号
sushiro-overdose sample start    后台静默信息收集
sushiro-overdose sample autostart on|off   配置系统开机自启动信息收集
sushiro-overdose sample stop     停止后台信息收集
sushiro-overdose doctor          打印只读诊断信息
sushiro-overdose repair-proxy    一键恢复系统代理
sushiro-overdose uninstall       恢复代理、移除证书并清理本地敏感数据
sushiro-overdose stop-processes  停止本应用相关进程，便于删除 exe

sushiro-overdose config                          查看通知配置
sushiro-overdose config feishu <webhook>         配置飞书通知
sushiro-overdose config telegram <token> <id>    配置 Telegram
sushiro-overdose config bark <url> <key>         配置 Bark
sushiro-overdose config serverchan <key>         配置 Server酱
sushiro-overdose config store add <id> <name>    添加门店昵称
```

---

## 通知配置

支持多个通知渠道，可在 Web UI 的「设置」页面配置，也可用命令行：

- **飞书** — 群机器人 Webhook
- **Telegram** — Bot Token + Chat ID
- **Bark** — iOS 推送
- **Server酱** — 微信推送

Web UI 的「设置」页面可点击「测试全部」，也可单独测试飞书、Telegram、Bark、Server酱。

## Web 增强功能

- **日历增强**：支持门店多选、只看可预约、午餐/晚餐过滤、自动刷新。
- **门店优先级**：设置页可选择抢号门店并调整优先顺序，日历筛选可一键保存为抢号门店。
- **历史洞察**：按门店、星期、时段统计开放概率和售罄速度，并反向推荐更值得抢的目标时段。
- **实时排队**：参考微信小程序公开接口展示中国大陆门店当前等位组数、营业状态和取号状态。
- **信息收集**：按门店和时间窗静默记录本地预约/排队变化，不抢号、不上传；可配置系统开机自启动。
- **到店预测**：按每家店单独展示工作日、周末、节假日趋势，给出候选到店时段，并展示实际过号数与全局过号数。
- **Web 狙击计划器**：支持多个目标、开放倒计时、开放窗口状态、尝试次数和最后错误。
- **本机诊断**：检查证书信任、端口占用、代理状态、代理链路、配置完整性和寿司郎网络连通性；捕获代理端口被占用时会自动换端口，并可一键复制脱敏诊断信息。
- **一键修复/卸载**：设置页可恢复代理、停止本应用相关进程；`uninstall` 可移除本地认证、通知配置、历史、证书文件并尝试移除系统信任证书。
- **更新提醒**：Web UI 可检查 GitHub 最新 Release，提示可下载的新版本。

排障时可运行 `sushiro-overdose doctor`，或在 Web UI 服务启动后访问 `GET /api/diagnostics` 获取只读、脱敏的诊断信息。

---

## 数据文件

所有数据统一存放在 `~/.sushiro/` 目录：

| 文件 | 说明 |
|------|------|
| `config.json` | 认证参数 |
| `preferences.json` | 用户偏好（人数/桌型/时段/优先级） |
| `notify.json` | 通知渠道配置 |
| `stores.json` | 门店昵称 |
| `sampling.json` | 信息收集配置 |
| `holidays.json` | 可选节假日/调休工作日本地表，用于到店预测分类 |
| `history.jsonl` | 历史时段数据 |
| `queue_observations.jsonl` | 实时排队/公开叫号快照，本地私有 |
| `queue_sessions.jsonl` | 用户真实取号等待 session，本地私有 |
| `queue_stats.json` | 本地聚合后的排队统计缓存 |
| `sushiro.log` | 后台模式日志 |
| `sampling.log` | 后台信息收集日志 |
| `sampling.pid` / `sampling.lock` | 后台信息收集进程状态与互斥锁 |
| `main_active.json` | 抢号/捕获/狙击运行标记，信息收集会自动避让 |

---

## 信息收集与到店预测

到店预测是纯本地能力，不上传数据。建议先选择自己常去或准备去的门店持续信息收集；每家店会单独计算趋势，预测只作为选择到店时间和预约优先级的参考。

- **实际过号数**：只在记录到 `called_no_when_user_called` 和 `display_called_no_at_take` 时计算，不用自己的号和当前叫号做简单推断。
- **全局过号数**：来自同一门店连续公开叫号快照的正向增量。
- **推荐时段**：优先参考等待更短、样本更多、过号风险更低的门店和时段；低样本会显示为继续观察。
- **节假日**：可在 `~/.sushiro/holidays.json` 写入 `{"holidays":["2026-05-01"],"workdays":["2026-05-09"]}`，未配置时只按自然工作日/周末分类。

## 工作原理

```
┌──────────┐     HTTPS (MITM)     ┌──────────────┐
│  PC 微信  │ ──────────────────→ │ 寿司郎服务器    │
│  小程序   │   ←── 本地代理 ──→   │              │
└──────────┘     捕获认证参数      └──────────────┘
       │                                    │
       └── 捕获完成后，清理代理，直连抢号 ──┘
```

1. 启动本地 HTTPS 代理 (MITM)，只对寿司郎 API 域名做 TLS 解密，其他域名保持 CONNECT 透传
2. 设置系统代理（退出时自动恢复）
3. 捕获认证参数后清理代理，直连 API 抢号
4. 后台每 5 分钟验证 Token 有效性

---

## 开发

代码分层说明见 [ARCHITECTURE.md](ARCHITECTURE.md)。

```bash
go build -o sushiro-overdose .       # 构建
go vet ./...                         # 静态检查
./sushiro-overdose                   # 运行（Web UI，独立窗口优先）
./sushiro-overdose cli               # 运行（终端模式）
```

### 发布新版本

```bash
git tag v1.2.0
git push origin v1.2.0
# GitHub Actions 自动构建所有平台并发布 Release
```

> 详细的架构文档和打包流程见 [AGENTS.md](AGENTS.md)。

## License

MIT
