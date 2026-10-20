# Turso 排队数据库只读查询指南

数据库 `libsql://su-shiro-ryujoxys.aws-us-west-2.turso.io`，由 `collector/` 写入，结构为 v2（2026-10-20 起）。不需要 key 的读者优先用 `data` 分支的公开数据，见 [public-data.md](public-data.md)。

token 是敏感凭证：只用只读 token，不写进 Markdown、Git、脚本参数或终端历史；进过聊天、日志的 token 用完即撤销。

## 连接

```bash
export SUSHIRO_HISTORY_DATABASE_URL='libsql://su-shiro-ryujoxys.aws-us-west-2.turso.io'
printf 'Turso read-only token: '; IFS= read -r -s SUSHIRO_HISTORY_DATABASE_TOKEN; printf '\n'
export SUSHIRO_HISTORY_DATABASE_TOKEN

PYTHONPATH=collector python3 - <<'PY'
import os
from collector.turso import TursoClient
db = TursoClient(os.environ["SUSHIRO_HISTORY_DATABASE_URL"], os.environ["SUSHIRO_HISTORY_DATABASE_TOKEN"])
print(db.execute("SELECT store_id, name, area FROM store_dimension WHERE is_active = 1 LIMIT 5"))
PY
```

## 表

| 表 | 内容 |
|---|---|
| `queue_records` | 唯一事实表：每店每半小时一条公开状态，主键 `(slot, store_id)`，`WITHOUT ROWID` |
| `store_dimension` | 门店名称、城市、区域、地址、经纬度、`is_active` |
| `store_latest` | 每店最近一帧（列同 `queue_records`，另有 `net_ticket_status`） |
| `holiday_calendar` | `date_key` → `holiday` / `workday`（调休） |
| `collector_runs` | 最近 30 天的采集轮次和每日清理（`run_id` 以 `maintenance-` 开头） |

`queue_records` 字段：

| 字段 | 含义 |
|---|---|
| `slot` | 北京时间半小时起点，`2026-09-05T18:30`。同一半小时内后采到的一帧覆盖前一帧 |
| `collected_at` | 这一帧的实际时间，`+08:00` |
| `store_status` | `OPEN` / `CLOSED` / `UNKNOWN` |
| `online_open` | 1 = 开放网上取号 |
| `wait_minutes` | 接口给的预估等待原值；`NULL` 未知，0 是有效值。**超过 180（官方显示上限）按未知处理** |
| `group_queues_count` | 在等桌数；`NULL` 未知 |
| `display_called_no` | 堂食当前叫号（booth/mixed/counter 最大值，不含预约队列）；只存正数，没有就是 `NULL` |

库里不存 P50/P80 等聚合，也不存日期类型；按需计算。2026-07-08 前的数据来自旧聚合归档，缺少部分零等待（`NULL`），早期等待曲线偏高。

## 日期类型

调休 `workday` 优先，其次 `holiday`；否则周五 16:30 起、周六全天、周日 22:00 前为 `weekend`，其余 `weekday`。“周五晚上”属于 `weekend`。SQL 里判断星期用 `strftime('%w', substr(slot,1,10))`（周日 0）。

## 常用查询

所有查询都限定 `slot` 范围，走主键，避免全表扫描。

```sql
-- 每店最新状态（不是实时，只到最近一次采集）
SELECT d.name, l.* FROM store_latest l JOIN store_dimension d USING (store_id) WHERE l.store_id = 1;

-- 某店某天的半小时记录
SELECT slot, store_status, wait_minutes, group_queues_count, display_called_no
FROM queue_records WHERE slot >= '2026-09-05' AND slot < '2026-09-06' AND store_id = 1 ORDER BY slot;

-- 最近若干个周五 19:30 叫到多少号
SELECT substr(slot,1,10) AS day, display_called_no, wait_minutes
FROM queue_records
WHERE slot >= '2026-07-01' AND store_id = 1 AND substr(slot,12) = '19:30'
  AND strftime('%w', substr(slot,1,10)) = '5' AND store_status = 'OPEN'
ORDER BY slot DESC LIMIT 12;

-- 数据新鲜度与采集健康
SELECT MAX(slot) FROM queue_records;
SELECT started_at, ok, substr(error_message,1,60) AS err FROM collector_runs ORDER BY started_at DESC LIMIT 10;
```

分位数（应用口径：等待 P50/P80，叫号 P20/P50/P80，线性插值）在客户端算：取出某店某日期类型、某半小时的各天值，排序后插值。`scripts/export-public-data.mjs` 的 `buildPatterns` 是参考实现，`data` 分支的 `patterns/<id>.json` 已经算好。

## 解读注意

1. 等待 0 不一定没人排，结合 `group_queues_count` 看；晚餐开排的 15–30 分钟内等待分钟常跳变。
2. 叫号是号码不是人数，不同日子起点不同，只比较同一时段；会因桌型、过号、批量叫号跳变。
3. 参与统计的天数少于 3 天时不给精确结论；回答时说明数据截止时间。

## 写入量

每轮每店一条 UPSERT 到当前半小时（每半小时两轮只留后一条），一年约 120 万行、60MB 左右。默认永久保留；`retention.records_days` 可按 `slot` 前缀裁剪。夜间不再聚合，只清理 30 天前的 `collector_runs`。
