# sushiro 采集器（Turso）

从寿司郎公开接口采集**排队压力 + 叫号**，写入 Turso 数据库。独立 Python 包（纯标准库），可单独部署，不由桌面端启动。

> **当前状态（2026-10-20）：** 寿司郎换了接口地址和认证方式，固定公共 token 已失效，采集器每轮 403，数据停在 2026-09-06。需要改用新接口和会员授权 token 后才能恢复，见 [线上现状](../docs/cloudflare-github-turso.md#数据库与采集2026-10-20)。

每轮既调 `stores?`（全国门店压力，无叫号）又对每店调 `getStoreById?`（`groupQueues` 里的当前叫号）。数据库每 24 小时由 `.github/workflows/public-data.yml` 只读导出到 `data` 分支，见 [公开数据](../docs/public-data.md)。

## 数据流

```
每 15 min（营业时段 10-22 点）：
  stores? + 每店 getStoreById?（详情失败用列表帧兜底）
  → 每店一帧 UPSERT 到 queue_records 的当前半小时（后一帧覆盖前一帧）+ store_latest
每天 03:00：清理 30 天前的 collector_runs；配置了 retention.records_days 时裁剪更早的记录
```

库里只存一张事实表，粒度和语义与桌面端内置历史包、`data` 分支一致；P50/P80 等规律由读取方计算，不再有夜间聚合、原始快照和归档。

## 数据库结构（v2）

| 表 | 内容 |
|----|------|
| `queue_records` | 每店每半小时一条：`slot`、`collected_at`、`store_status`、`online_open`、`wait_minutes`、`group_queues_count`、`display_called_no`；主键 `(slot, store_id)` |
| `store_dimension` | 门店静态信息与 `is_active` |
| `store_latest` | 每店最近一帧 |
| `holiday_calendar` | 节假日 / 调休工作日 |
| `collector_runs` | 最近 30 天采集与维护日志 |

缺失一律 `NULL`，不写 0 冒充；`wait_minutes` 存接口原值，读取方把超过 180（官方显示上限）的值当作未知；`display_called_no` 只存正数。字段说明和查询示例见 [查询指南](../docs/turso-readonly-query-guide.md)。体量约每年 60MB。

## 快速开始

**前置**：Python ≥ 3.9，零外部依赖。

```bash
cd collector
python3 -m venv venv
cp config.example.json config.json   # 填 turso.url / turso.auth_token（读写）/ sushiro.token

venv/bin/python -m collector.main init-schema     # 建表（幂等）
venv/bin/python -m collector.main seed-holidays   # 节假日日历
venv/bin/python -m collector.main bootstrap       # 门店维度
venv/bin/python -m collector.main run-once        # 跑一轮验证
venv/bin/python -m collector.main run             # 常驻
```

服务器部署（systemd）、验证和排障见 [AGENTS.md](AGENTS.md)。

## 从 v1 升级

v1 库有 `queue_snapshots`（原始逐帧）和三张聚合表。线上库已于 2026-10-20 迁移完成（395MB → 16MB 数据），其他 v1 库按以下顺序：

```bash
# 1. 停旧采集器，完整备份全部表（迁移的删除步骤不可逆）
sudo systemctl stop collector
# 2. 建 queue_records 并从逐日代表值 + 原始快照填充，可重复执行
venv/bin/python -m collector.main migrate-v2
# 3. 核对后删除旧表、重建 store_latest（不可逆）
venv/bin/python -m collector.main migrate-v2 --drop
# 4. 部署新代码后启动
sudo systemctl start collector
```

Turso 的 HTTP 接口不允许 `VACUUM`，删表后文件里的空闲页会被之后的写入复用。要立刻缩小文件，用 Turso CLI 从紧凑副本重建数据库（见 AGENTS.md「缩小数据库文件」）。

## 命令

| 命令 | 作用 |
|------|------|
| `init-schema` | 建全部表（幂等） |
| `seed-holidays [--year N]` | 初始化节假日日历 |
| `bootstrap` | 拉一次 `stores?` 写 `store_dimension` |
| `run-once [--stores id...] [--no-detail]` | 跑一轮采集 |
| `migrate-v2 [--drop]` | v1 → v2 迁移 |
| `run` | 常驻：每 15min 采集 + 每日清理 |

## 配置

- `config.json`（不进 git）：Turso URL / 读写 token、寿司郎 token、采集频率、门店列表、`retention`。
- 环境变量优先：`SUSHIRO_COLLECTOR_SUSHIRO_TOKEN`、`SUSHIRO_COLLECTOR_TURSO_URL`、`SUSHIRO_COLLECTOR_TURSO_AUTH_TOKEN`（systemd 用 `EnvironmentFile=.env`）。
- `retention.records_days` 默认 0（永久保留）；v1 的 `archive.retention_days` 不再生效，避免把历史按 60 天删掉。

token 绝不写进代码或日志（日志已 redact）。

## 关键算法

- **叫号**：`groupQueues` 里 booth/mixed/counter 三队列取最大整数，预约队列不参与；没有叫号存 `NULL`。
- **半小时槽**：`collected_at` 为 `+08:00` 文本，`slot` = 日期 + `HH:00`/`HH:30`；乱序到达的旧帧不覆盖新帧。
- **日期类型**（读取时计算）：调休工作日 > 节假日 > 周末窗口（周五 ≥16:30 / 周六全天 / 周日 <22:00）> 工作日。

## 测试

```bash
python3 -m unittest discover -s tests -v
```
