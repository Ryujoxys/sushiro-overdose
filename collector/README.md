# sushiro 采集器（Turso）

从寿司郎公开接口采集**排队压力 + 叫号**，写入 Turso 数据库。独立 Python 包，可单独部署。

## 为什么有这个

桌面端 sushiro-overdose 的图表（叫号曲线、忙率热力图、到店建议）依赖线上历史数据。
旧采集项目只调批量列表接口 `stores?`，那个接口**不返回叫号**——所以旧库有 2.5 万条压力快照
但零条叫号数据，"按历史几点叫到几号"的曲线永远是空的。

本采集器**每轮既调 `stores?`（全国压力）又对每店调 `getStoreById?`（拿叫号 groupQueues）**，
补上叫号，并新增预测辅助字段（叫号间隔/吞吐率、节假日、数据质量标记）。

## 数据流

```
每 15 min（营业时段 10-22 点）：
  stores?（全国压力）+ 每店 getStoreById?（叫号）
  → 每店择优一帧写 queue_snapshots + store_latest（详情失败才用列表兜底）
每天 02:00：分页聚合 → store_bucket_rollups（含叫号三档 P20/P50/P80）+ called_intervals_rollups
每天 03:00：归档，删 60 天前的原始快照
```

## 快速开始（本机）

**前置**：Python ≥ 3.9（开发/测试于 3.9.6）。采集器零外部依赖（纯标准库 urllib）。

```bash
cd collector

# 1. 建虚拟环境
python3 -m venv venv
venv/bin/pip install -r requirements.txt

# 2. 填配置（复制模板，填入真实 Turso URL/token + 寿司郎 token）
cp config.example.json config.json
#   编辑 config.json，填 turso.url / turso.auth_token / sushiro.token

# 3. 建表
venv/bin/python -m collector.main init-schema

# 4. 初始化节假日日历（影响 date_type 分桶）
venv/bin/python -m collector.main seed-holidays

# 5. 冷启动门店维度（从 stores? 拉全国门店写 store_dimension）
venv/bin/python -m collector.main bootstrap

# 6.（可选）从旧库导入压力历史（叫号列留 NULL，叫号从今天攒）
venv/bin/python -m collector.main migrate-old

# 7. 跑一轮验证
venv/bin/python -m collector.main run-once

# 8. 立即聚合一次（产出 rollups，验证叫号三档）
venv/bin/python -m collector.main aggregate-now

# 9. 常驻运行
venv/bin/python -m collector.main run
```

## 部署到服务器（systemd）

**前置**：服务器需 Python ≥ 3.9。采集器零外部依赖。

```bash
# 拷贝 collector/ 到服务器（不要带本机的 venv——它是 macOS 二进制，Linux 上不可用）
rsync -av --exclude venv --exclude config.json --exclude '__pycache__' collector/ user@server:/opt/sushiro-collector/

# 服务器上：
cd /opt/sushiro-collector
python3 -m venv venv          # 服务器上重建 venv（Linux 二进制）
venv/bin/pip install -r requirements.txt   # 零依赖，仅建立 venv 环境
cp config.example.json config.json && vi config.json   # 填凭证

venv/bin/python -m collector.main init-schema
venv/bin/python -m collector.main seed-holidays
venv/bin/python -m collector.main bootstrap
venv/bin/python -m collector.main run-once     # 验证有数据

# 安装 systemd 服务（改 collector.service 里的 WorkingDirectory/ExecStart 路径）
sudo cp collector.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now collector

# 查看
systemctl status collector
journalctl -u collector -f
```

崩溃自动重启（Restart=always，30s 间隔）；SIGTERM 优雅退出（完成当前轮）。

## 迁移到新服务器

1. 在新机器重复上面「部署到服务器」步骤。
2. Turso 库不用迁——数据在云端，采集器换机器只要 `config.json` 填同样的 Turso URL/token，新机器接着旧数据继续写。
3. 想换新 Turso 库：改 `config.json` 的 `turso.url`，重跑 init-schema + bootstrap + migrate-old。

## 命令一览

| 命令 | 作用 |
|------|------|
| `init-schema` | 建全部表 + 索引 |
| `seed-holidays [--year N]` | 初始化节假日日历（内置近两年，可改 holidays.json 扩展） |
| `bootstrap` | 冷启动：拉一次 stores? 写 store_dimension |
| `migrate-old [--limit N] [--dry-run]` | 从旧库导入压力历史（叫号 NULL） |
| `run-once [--stores id...] [--no-detail]` | 跑一轮采集 |
| `aggregate-now [--days N]` | 立即聚合（产出 rollups） |
| `run` | 常驻：每 15min 采集 + 每日聚合/归档 |

## 配置

- `config.json`（**不进 git**）：真实 Turso URL/token、寿司郎 token、采集频率、店列表。
- 环境变量（优先级高于 config.json，systemd 用 `EnvironmentFile=.env`）：
  - `SUSHIRO_COLLECTOR_SUSHIRO_TOKEN`
  - `SUSHIRO_COLLECTOR_TURSO_URL` / `SUSHIRO_COLLECTOR_TURSO_AUTH_TOKEN`
  - `SUSHIRO_COLLECTOR_OLD_TURSO_URL` / `SUSHIRO_COLLECTOR_OLD_TURSO_AUTH_TOKEN`（仅 migrate-old）

**安全**：token 绝不写进代码或日志（日志已 redact）。`config.json` / `.env` / `venv/` 均在 `.gitignore`。

## 数据库结构（Turso）

| 表 | 内容 |
|----|------|
| `store_dimension` | 门店静态信息（116+ 家） |
| `store_latest` | 每店最新一帧（含叫号） |
| `queue_snapshots` | 原始逐帧快照（每店每轮一行，`dq_source` 标记详情或列表兜底） |
| `store_bucket_rollups` | 时段聚合（压力 P50/P80 + **叫号三档** P20/P50/P80）← worker 查它画图 |
| `daily_store_bucket_rollups` | 按天细分的同结构 |
| `called_intervals_rollups` | 叫号间隔/吞吐率（预测辅助） |
| `holiday_calendar` | 节假日日历 |
| `collector_runs` / `archive_runs` | 运维日志 |

`store_bucket_rollups` 的表名和列名（含 `called_no_slow/typical/fast`）与桌面端
Cloudflare Worker 期望完全兼容——**桌面端零改动就能读到新叫号数据**。

## 读写量控制

- 125 家店、每天 48 轮时，新增原始快照约 `125 × 48 = 6000` 行，不再同时写列表帧和详情帧。
- `store_latest` 每轮更新，用于实时查询；`store_dimension` 仅字段变化或跨日时实际更新。
- 夜间 30 天聚合按 `id` 分页读取，避免一次返回数十万行导致连接中断后整批重读。
- rollup UPSERT 只有业务字段变化才更新；逐日表定时任务只刷新最新日期和缺行日期。
- 聚合/归档成功状态写入 `collector_runs`，进程重启不会重复执行当天维护任务。
- 每日维护只在 02:00 到营业开始前补跑，白天重启不会用聚合任务阻塞实时采集。

## 关键算法（移植自主项目 Go 代码，语义 1:1）

- **叫号**：`getStoreById` 返回的 `groupQueues` 里 booth/mixed/counter 三队列取最大整数（reservation 预约号不参与）。0 = 无堂食叫号。
- **分位数**：线性插值 `pos=q*(n-1)`（移植 `queue_trends.go queueQuantile`）。
- **时段桶**：30 分钟（`HH:00` / `HH:30`），与旧库一致。
- **date_type**：workday_override > holiday > 周末窗口（周五≥16:30 / 周六全天 / 周日<22:00）> weekday。
