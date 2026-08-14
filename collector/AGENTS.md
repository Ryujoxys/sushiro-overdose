# AGENTS.md — 采集器部署指南（给 LLM / 自动化）

> 本文件专为 AI agent（Claude/Cursor 等）写，目标是**零猜测地把 sushiro 采集器部署到一台新服务器并跑起来**。
> 人类可看 README.md；LLM 看本文件。按顺序执行，每步有验证检查点。

## 这是什么

`collector/` 是一个 Python 采集器：每 15 分钟调寿司郎公开接口，采集**全国 116+ 家店的排队压力 + 叫号**，写入 Turso 数据库。桌面端图表（叫号曲线、忙率热力图）依赖这些数据。

技术栈：Python 3.9+，纯标准库（urllib），零外部依赖。Turso 走 HTTP，不需要 SDK。

## 前置条件（部署前确认）

1. **服务器有 Python ≥ 3.9**：`python3 --version`。没有就装（`apt install python3` / `brew install python`）。
2. **Turso 库 + 读写 token**：
   - 库：`libsql://su-shiro-ryujoxys.aws-us-west-2.turso.io`（已存在）
   - token：**读写**（采集器要写入，不是只读；只读 token 会 INSERT 失败）
   - 若要导入旧库历史，还需旧库 `libsql://sushiro-public-ryujoxys.aws-us-west-2.turso.io` 的**只读** token
3. **能访问 Turso（出站 HTTPS）** 和 `crm-cn-prd.sushiro.com.cn`（寿司郎接口）。

## 部署步骤（按顺序，每步验证）

### 1. 拿到代码

```bash
git clone https://github.com/Ryujoxys/sushiro-overdose.git
cd sushiro-overdose/collector
```

或拷贝 `collector/` 目录到服务器。

### 2. 建虚拟环境 + 安装（零外部依赖）

```bash
python3 -m venv venv
venv/bin/pip install -r requirements.txt   # requirements.txt 实际无依赖，仅建 venv 环境
```

**验证**：`venv/bin/python -c "import urllib, json; print('ok')"` → 输出 `ok`。

### 3. 写配置（真实凭证，不进 git）

```bash
cp config.example.json config.json
```

编辑 `config.json`，填：
- `turso.url` = `libsql://su-shiro-ryujoxys.aws-us-west-2.turso.io`
- `turso.auth_token` = **读写** token
- `sushiro.token` = 寿司郎公开 token（已在 example 里，一般不用改）
- `old_turso.*` = 旧库只读 token（仅 `migrate-old` 用，不导入历史可留空）

**验证**：`venv/bin/python -c "from collector.config import load_config; c=load_config(); print('turso:', bool(c['turso']['auth_token']))"` → `turso: True`。

### 4. 建表（Turso schema）

```bash
venv/bin/python -m collector.main init-schema
```

**验证**：输出含 9 张表（store_dimension / store_latest / queue_snapshots / store_bucket_rollups / daily_store_bucket_rollups / called_intervals_rollups / holiday_calendar / collector_runs / archive_runs）+ `✅ 建表完成`。

### 5. 初始化节假日日历

```bash
venv/bin/python -m collector.main seed-holidays
```

**验证**：`✅ 写入 N 条节假日`（N ≈ 68）。

### 6. 冷启动门店维度

```bash
venv/bin/python -m collector.main bootstrap
```

**验证**：`✅ store_dimension 现有 118 行`（或接近，寿司郎门店数会变）。

### 7.（可选）导入旧库压力历史

```bash
venv/bin/python -m collector.main migrate-old
```

约 25000 行，跨太平洋写入慢（几分钟）。**验证**：`✅ 导入完成：快照 N 行`。

### 8. 跑一轮验证采集

```bash
venv/bin/python -m collector.main run-once
```

**验证**：输出 `{'stores_seen': 1xx, 'snapshots_written': 1xx, 'detail_ok': 1xx, ...}`。每店每轮只落一帧；营业时段（10-22 点）`detail_ok` 应接近门店数，详情失败时由列表帧兜底。

### 9.（可选）立即聚合一次

```bash
venv/bin/python -m collector.main aggregate-now
```

**验证**：`✅ 聚合完成：rollups=N daily=N ...`。

### 10. 装成 systemd 服务（常驻 + 开机自启 + 崩溃自愈）

改 `collector.service` 里的路径：
- `WorkingDirectory` = collector 目录绝对路径
- `ExecStart` = `<collector目录>/venv/bin/python -m collector.main run`

```bash
sudo cp collector.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now collector
```

**验证**：
- `systemctl status collector` → `active (running)`
- `journalctl -u collector -f` → 看到每 15 分钟一轮采集日志（营业时段）
- 杀进程 `sudo kill $(pgrep -f collector.main)` → 30s 内自动重启（Restart=always）

## 命令一览

| 命令 | 作用 | 频率 |
|------|------|------|
| `init-schema` | 建 9 张表 | 首次一次 |
| `seed-holidays [--year N]` | 写节假日日历 | 首次一次 |
| `bootstrap` | 拉门店写 store_dimension | 首次一次 |
| `migrate-old [--limit N] [--dry-run]` | 导旧库压力历史 | 首次一次（可选） |
| `run-once [--stores id...] [--no-detail]` | 跑一轮采集 | 验证用 |
| `aggregate-now [--days N]` | 立即聚合 | 验证用 |
| `run` | 常驻：每 15min 采集 + 每日 02:00 聚合 + 03:00 归档 | systemd 跑这个 |

## 配置项（config.json）

```json
{
  "sushiro": { "base_url", "token", "referer", "user_agent" },
  "turso": { "url", "auth_token" },              // 读写 token
  "old_turso": { "url", "auth_token" },          // 只读，仅 migrate-old
  "collect": {
    "interval_seconds": 900,                       // 15 分钟
    "store_ids": [],                              // 空=全国全采
    "active_hours": [10, 22],                      // 营业时段才采
    "concurrency": 8,
    "list_latitude": 23.13, "list_longitude": 113.26
  },
  "archive": { "retention_days": 60 }
}
```

也可用环境变量覆盖（systemd 用 EnvironmentFile）：`SUSHIRO_COLLECTOR_TURSO_URL` / `SUSHIRO_COLLECTOR_TURSO_AUTH_TOKEN` / `SUSHIRO_COLLECTOR_SUSHIRO_TOKEN`。

## 故障排查

| 现象 | 原因 / 处理 |
|------|------|
| `init-schema` 报 401 Unauthorized | Turso token 错或过期；去 Turso 控制台重生成**读写** token |
| `init-schema` 报 invalid type string expected f64 | （已修）args 编码问题，用最新代码 |
| `bootstrap` 报 `stores_seen: 0` | 寿司郎接口不通；检查服务器能否访问 `crm-cn-prd.sushiro.com.cn` |
| `run-once` detail_ok=0 全失败 | 同上，接口网络问题 |
| 营业时段 `display_called_no` 全 0 | 门店都关了（非营业时段正常）；或 getStoreById 接口变了，查日志 |
| systemd 启动后秒退 | WorkingDirectory/ExecStart 路径错；`journalctl -u collector` 看报错 |
| 配置加载报 `缺少配置` | config.json 没填真值（还是 PUT_ 占位），或 venv 没激活 |
| `migrate-old` 中途断 | 网络抖动；重跑幂等（INSERT OR IGNORE 跳过已导入） |

## 安全

- `config.json` / `.env` / `venv/` 都在 `.gitignore`，**不进 git**
- Turso token 是读写权限，**不要泄露**；systemd 用 EnvironmentFile 而非写进 service 文件
- 日志里 token 已 redact

## 关键设计（LLM 理解上下文用）

- **每店每轮一帧**：详情成功写 `store_detail`，失败才写 `stores_list` 兜底；历史存量仍可能同时存在两种来源。
- **为什么调两个接口**：`stores?`（批量，全国压力，无叫号）+ `getStoreById?`（单店，叫号 groupQueues）。旧库只调前者所以零叫号。
- **聚合只读 30 天**：按 `id` 分页读取，daily 定时任务只写最新/缺行日期，避免超大响应和历史行重写。
- **脏数据过滤**：wait>180（cap）/groups>200 丢弃（接口异常值）。
- **schema 兼容桌面端**：表名/列名（含 `called_no_slow/typical/fast`）与桌面端 Cloudflare Worker 一致，桌面端零改动读新库。

## 验证采集真的在跑（部署后确认）

```bash
# 1. 进程在
systemctl is-active collector   # → active

# 2. 数据在涨（隔 15 分钟跑两次对比）
venv/bin/python -c "
from collector.config import load_config
from collector.turso import TursoClient
c=load_config(); t=TursoClient(c['turso']['url'], c['turso']['auth_token'])
print('snapshots:', t.execute('SELECT COUNT(*) AS c FROM queue_snapshots')[0]['c'])
print('latest:', t.execute('SELECT MAX(collected_at) AS m FROM queue_snapshots')[0]['m'])
"
# collected_at 应是几分钟内的时间戳；隔 15min 再跑一次 snapshots 数应增加
```
