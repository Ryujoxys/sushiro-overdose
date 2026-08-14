# Turso 排队数据库只读查询指南

本文说明 `sushiro-overdose` 的 Turso 排队数据库结构、字段口径，以及常见需求应该查询哪张表。

适用数据库：

```text
libsql://su-shiro-ryujoxys.aws-us-west-2.turso.io
```

本文只使用只读查询。数据库 token 属于敏感凭证，不要写入本文、Git、脚本参数或终端历史。

## 连接方式

推荐通过环境变量临时注入凭证：

```bash
export SUSHIRO_MCP_TURSO_URL='libsql://su-shiro-ryujoxys.aws-us-west-2.turso.io'
printf 'Turso read-only token: '
IFS= read -r -s SUSHIRO_MCP_TURSO_TOKEN
printf '\n'
export SUSHIRO_MCP_TURSO_TOKEN
```

项目自带标准库实现的 Turso HTTP 客户端，不需要额外安装 SDK：

```bash
cd /path/to/sushiro
PYTHONPATH=collector python3 - <<'PY'
import os
from collector.turso import TursoClient

db = TursoClient(
    os.environ["SUSHIRO_MCP_TURSO_URL"],
    os.environ["SUSHIRO_MCP_TURSO_TOKEN"],
)

rows = db.execute("""
SELECT store_id, name, area
FROM store_dimension
WHERE is_active = 1
ORDER BY store_id
LIMIT 10
""")

for row in rows:
    print(row)
PY
```

桌面端 MCP 助手使用相同的两个环境变量：

```text
SUSHIRO_MCP_TURSO_URL
SUSHIRO_MCP_TURSO_TOKEN
```

## 选表原则

| 需求 | 首选表 | 原因 |
|---|---|---|
| 搜门店、查门店 ID | `store_dimension` | 一店一行，名称和地址最完整 |
| 查每家店最新状态 | `store_latest` | 一店一行，查询成本最低 |
| 查某天某个时刻的真实快照 | `queue_snapshots` | 约 15 分钟粒度的原始数据 |
| 查长期工作日/周末规律 | `store_bucket_rollups` | 已按日期类型、星期和半小时聚合 |
| 查每天的差异或重新计算分位数 | `daily_store_bucket_rollups` | 一天一店一时段一行 |
| 查叫号速度 | `called_intervals_rollups` | 已聚合相邻叫号的推进速度 |
| 判断节假日、调休工作日 | `holiday_calendar` | 日期类型覆盖表 |
| 检查采集是否正常 | `collector_runs` | 采集轮次与每日维护任务的结果 |
| 检查归档状态 | `archive_runs` | 原始快照归档和清理记录 |

默认先查聚合表。只有需要精确到某天、某个 15 分钟点或排查数据问题时，才查询 `queue_snapshots`。

## 数据库结构

### `store_dimension`

门店维度表，一家门店一行。

主键：

```text
store_id
```

常用字段：

| 字段 | 含义 |
|---|---|
| `store_id` | 寿司郎门店 ID |
| `name` | 门店名称 |
| `city`、`area`、`address` | 城市、区域、地址 |
| `latitude`、`longitude` | 经纬度 |
| `tables_capacity` | 桌席容量参考值 |
| `counters_capacity` | 吧台容量参考值 |
| `last_seen_at` | 最近一次在官方门店列表中出现的时间 |
| `is_active` | 是否仍为活跃门店 |

### `store_latest`

每家店的最新公开状态，一家门店一行。适合实时总览，不适合历史分析。

常用字段：

| 字段 | 含义 |
|---|---|
| `collected_at` | 最新快照采集时间，使用 `+08:00` |
| `wait_minutes` | 官方等待分钟 |
| `group_queues_count` | 当前在等桌数 |
| `store_status` | 门店营业状态 |
| `net_ticket_status` | 在线取号状态 |
| `online_open` | 是否可在线取号 |
| `display_called_no` | 当前堂食叫号 |
| `group_queues_json` | 官方接口返回的各桌型叫号数组 |

### `queue_snapshots`

原始排队快照表。当前采集器每店每轮只写一帧：单店详情成功时写
`store_detail`，详情失败时写 `stores_list` 兜底。优化前的历史存量会在同一门店、相近时间
同时存在两种来源，查询历史时仍需去重。

唯一性约束：

```text
(collected_at, store_id, dq_source)
```

关键字段：

| 字段 | 含义 |
|---|---|
| `collected_at` | ISO 8601 采集时间，带 `+08:00` |
| `store_id` | 门店 ID |
| `wait_minutes` | 官方等待分钟 |
| `group_queues_count` | 当前在等桌数 |
| `display_called_no` | 当前堂食叫号 |
| `dq_source` | `stores_list` 或 `store_detail` |
| `dq_anomaly` | 上游标记的数据异常 |
| `wait_time_cap` | 官方等待时间参考上限 |
| `group_queues_json` | 单店详情中的原始叫号队列 |

`dq_source` 的选择规则：

- 需要叫号时使用 `store_detail`。
- `stores_list` 是详情请求失败时的压力兜底，没有叫号。
- 查询优化前历史时不要把两种来源直接相加，否则会重复计数。

`display_called_no` 是三态字段：

- `NULL`：本轮没有请求单店详情，不能判断叫号。
- `0`：请求到了单店详情，但当前没有可用堂食叫号。
- `> 0`：当前堂食叫号。

当前堂食叫号取 `boothQueue`、`mixedQueue`、`counterQueue` 中的最大整数，不包含 `reservationQueue`。

### `store_bucket_rollups`

长期时段规律表，粒度为：

```text
(store_id, date_type, weekday, time_bucket)
```

字段口径：

| 字段 | 含义 |
|---|---|
| `date_type` | `weekday`、`weekend`、`holiday` 或 `workday` |
| `weekday` | 周一为 `0`，周日为 `6` |
| `time_bucket` | 30 分钟桶，例如 `19:30` |
| `sample_count` | 参与该桶聚合的日期数 |
| `busy_rate` | 有有效等待分钟或在等桌数的日期比例 |
| `wait_typical_minutes` | 有排队时等待分钟 P50 |
| `wait_safe_minutes` | 有排队时等待分钟 P80 |
| `queue_groups_typical` | 有排队时在等桌数 P50 |
| `queue_groups_safe` | 有排队时在等桌数 P80 |
| `called_no_slow` | 当前叫号 P20 |
| `called_no_typical` | 当前叫号 P50 |
| `called_no_fast` | 当前叫号 P80 |
| `called_sample_count` | 有有效叫号的日期数 |
| `confidence` | `low`、`medium` 或 `high` |

置信度规则：

```text
high:   sample_count >= 20
medium: sample_count >= 8
low:    sample_count < 8
```

### `daily_store_bucket_rollups`

逐日时段表，粒度为：

```text
(snapshot_date, store_id, time_bucket)
```

它保留每天每个半小时桶的代表值，适合：

- 比较周一和周五。
- 计算跨日期的 P50、P80。
- 检查某个结论是否由少数异常日期驱动。
- 倒推某个目标用餐时间的历史取号窗口。

每个半小时桶取该桶内最后一帧，所以 `19:30` 桶通常更接近 `19:45` 的状态，而不是严格代表 `19:30:00`。

### `called_intervals_rollups`

叫号推进速度表，粒度为：

```text
(store_id, date_type, time_bucket)
```

常用字段：

| 字段 | 含义 |
|---|---|
| `pair_count` | 参与计算的相邻叫号对数量 |
| `interval_typical_seconds` | 相邻有效叫号观测的典型间隔 |
| `delta_typical` | 每对观测的典型叫号增量 |
| `throughput_per_hour` | 估算每小时推进号数 |

`pair_count` 很小时不要单独依赖速度结果。叫号可能因过号、桌型和批量叫号发生跳变。

### 运维表

| 表 | 用途 |
|---|---|
| `holiday_calendar` | 标记法定节假日和调休工作日 |
| `collector_runs` | 记录采集轮次及聚合/归档任务是否成功、写入条数和错误 |
| `archive_runs` | 记录原始快照归档、删除和保留周期 |

## 日期类型口径

日期类型不是简单的周一至周五：

1. 调休工作日优先标为 `workday`。
2. 法定节假日标为 `holiday`。
3. 周五 `16:30` 以后、整个周六、周日 `22:00` 前标为 `weekend`。
4. 其余普通工作时段标为 `weekday`。

因此查询“周五晚上”时必须使用：

```sql
date_type = 'weekend' AND weekday = 4
```

只查 `date_type = 'weekday'` 会漏掉周五晚间。

## 常见需求查询

以下示例使用北京国贸商城北区店。先确认门店 ID，不要只凭名称猜测：

```sql
SELECT store_id, name, area, address
FROM store_dimension
WHERE is_active = 1
  AND (
    name LIKE '%国贸%'
    OR area LIKE '%国贸%'
    OR address LIKE '%国贸%'
  )
ORDER BY store_id;
```

当前库中北京国贸商城北区店的门店 ID 为 `3050`。

### 查当前排队

```sql
SELECT
  store_id,
  name,
  collected_at,
  wait_minutes,
  group_queues_count,
  display_called_no,
  store_status,
  net_ticket_status,
  online_open
FROM store_latest
WHERE store_id = 3050;
```

### 查普通工作日全天规律

下面的查询会跨周一至周五的有效 `weekday` 桶做近似汇总：

```sql
SELECT
  time_bucket,
  ROUND(AVG(busy_rate), 2) AS busy_rate,
  ROUND(AVG(wait_typical_minutes)) AS wait_minutes,
  ROUND(AVG(queue_groups_typical)) AS queue_groups,
  ROUND(AVG(called_no_typical)) AS called_no,
  SUM(sample_count) AS samples,
  SUM(called_sample_count) AS called_samples
FROM store_bucket_rollups
WHERE store_id = 3050
  AND date_type = 'weekday'
  AND time_bucket BETWEEN '10:00' AND '22:00'
GROUP BY time_bucket
ORDER BY time_bucket;
```

这是对每个星期聚合结果的再平均，适合快速查看形状。需要严格的跨日期 P50、P80 时，应读取 `daily_store_bucket_rollups` 后在客户端重新计算分位数。

### 查周五晚间规律

```sql
SELECT
  time_bucket,
  sample_count,
  busy_rate,
  wait_typical_minutes,
  wait_safe_minutes,
  queue_groups_typical,
  queue_groups_safe,
  called_no_slow,
  called_no_typical,
  called_no_fast,
  called_sample_count,
  confidence
FROM store_bucket_rollups
WHERE store_id = 3050
  AND date_type = 'weekend'
  AND weekday = 4
  AND time_bucket BETWEEN '16:30' AND '21:30'
ORDER BY time_bucket;
```

### 查最近若干个周五 19:45 叫到多少号

原始采样大约每 15 分钟一次，可以直接取 `19:45` 快照：

```sql
SELECT
  substr(collected_at, 1, 10) AS snapshot_date,
  MAX(display_called_no) AS called_no,
  MAX(group_queues_count) AS waiting_groups,
  MAX(wait_minutes) AS official_wait_minutes
FROM queue_snapshots
WHERE store_id = 3050
  AND dq_source = 'store_detail'
  AND strftime('%w', substr(collected_at, 1, 10)) = '5'
  AND substr(collected_at, 12, 5) = '19:45'
  AND display_called_no > 0
GROUP BY substr(collected_at, 1, 10)
ORDER BY snapshot_date DESC
LIMIT 12;
```

SQLite 的 `strftime('%w')` 使用周日 `0`、周五 `5`。这与聚合表的 `weekday` 周一 `0` 口径不同，写查询时不要混用。

快速查看典型范围：

```sql
WITH friday_calls AS (
  SELECT
    substr(collected_at, 1, 10) AS snapshot_date,
    MAX(display_called_no) AS called_no
  FROM queue_snapshots
  WHERE store_id = 3050
    AND dq_source = 'store_detail'
    AND strftime('%w', substr(collected_at, 1, 10)) = '5'
    AND substr(collected_at, 12, 5) = '19:45'
    AND display_called_no > 0
  GROUP BY substr(collected_at, 1, 10)
)
SELECT
  COUNT(*) AS sample_days,
  MIN(called_no) AS slow_end,
  ROUND(AVG(called_no)) AS average_called_no,
  MAX(called_no) AS fast_end
FROM friday_calls;
```

`MIN/AVG/MAX` 只是快速范围，不等同于严格的 P20/P50/P80。

### 倒推目标用餐时间的取号窗口

下面示例寻找每个周五最接近“19:45 叫到”的历史取号时刻。它用官方等待分钟做探索式倒推：

```sql
WITH candidates AS (
  SELECT
    substr(collected_at, 1, 10) AS snapshot_date,
    substr(collected_at, 12, 5) AS take_time,
    wait_minutes,
    group_queues_count,
    display_called_no,
    (
      CAST(substr(collected_at, 12, 2) AS INTEGER) * 60
      + CAST(substr(collected_at, 15, 2) AS INTEGER)
      + wait_minutes
    ) AS projected_called_minute
  FROM queue_snapshots
  WHERE store_id = 3050
    AND dq_source = 'store_detail'
    AND strftime('%w', substr(collected_at, 1, 10)) = '5'
    AND substr(collected_at, 12, 5) BETWEEN '17:00' AND '18:30'
),
ranked AS (
  SELECT
    *,
    ABS(projected_called_minute - (19 * 60 + 45)) AS error_minutes,
    ROW_NUMBER() OVER (
      PARTITION BY snapshot_date
      ORDER BY ABS(projected_called_minute - (19 * 60 + 45))
    ) AS rn
  FROM candidates
)
SELECT
  snapshot_date,
  take_time,
  wait_minutes,
  group_queues_count,
  display_called_no,
  error_minutes
FROM ranked
WHERE rn = 1
ORDER BY snapshot_date DESC
LIMIT 12;
```

这个查询只能回答“按当时官方等待估计，几点取最接近目标时间”，不能证明真实入座时间。更稳妥的实际规则是：

1. 提前约两小时开始观察。
2. 等官方等待进入目标剩余时间附近再取号。
3. 同时看在等桌数和叫号推进，避免只信突然跳变的等待分钟。
4. 拿到具体号码后，用当天实时叫号速度重新估算。

### 查叫号速度

```sql
SELECT
  date_type,
  time_bucket,
  pair_count,
  interval_typical_seconds,
  delta_typical,
  throughput_per_hour
FROM called_intervals_rollups
WHERE store_id = 3050
  AND date_type = 'weekend'
  AND time_bucket BETWEEN '16:30' AND '21:30'
ORDER BY time_bucket;
```

速度结果至少结合 `pair_count`、当前在等桌数和官方等待时间一起解释。

### 检查采集新鲜度

```sql
SELECT
  MAX(collected_at) AS latest_snapshot,
  COUNT(*) AS snapshot_rows
FROM queue_snapshots;
```

```sql
SELECT
  MAX(finished_at) AS latest_success
FROM collector_runs
WHERE endpoint = 'stores+detail'
  AND ok = 1;
```

### 检查某门店的数据覆盖

```sql
SELECT
  dq_source,
  MIN(collected_at) AS first_seen,
  MAX(collected_at) AS last_seen,
  COUNT(*) AS rows,
  COUNT(DISTINCT substr(collected_at, 1, 10)) AS days,
  SUM(
    CASE
      WHEN display_called_no IS NOT NULL AND display_called_no > 0 THEN 1
      ELSE 0
    END
  ) AS called_rows
FROM queue_snapshots
WHERE store_id = 3050
GROUP BY dq_source
ORDER BY dq_source;
```

## 数据质量注意事项

1. 历史原始表有 `stores_list` 和 `store_detail` 双帧，叫号分析只用 `store_detail`；新数据每店每轮仅一帧。
2. `wait_minutes = 0` 不一定代表完全没人排队，要同时看 `group_queues_count`。
3. 官方等待分钟可能突然跳变，特别是晚餐开排的 15 至 30 分钟内。
4. 上游长期聚合会过滤明显异常值：等待超过参考上限、在等桌数超过 `200` 的帧不进入典型分位数。
5. `called_no` 会因桌型、过号和批量叫号跳变，不能简单假设每个号码对应一桌。
6. `called_sample_count` 或 `pair_count` 太小时，不要给出精确到 5 分钟的结论。
7. 原始快照有保留周期，长期规律应使用 rollup 表。
8. SQL 中必须限定门店、日期和时间范围，避免无意读取整张原始快照表。

## 读写量与维护策略

以 125 家店、营业时段每天 48 轮为例，新版每天新增原始快照约 `6000` 行。优化前每店每轮
同时写列表和详情两帧，原始快照约为新版的两倍。

当前写入规则：

- `queue_snapshots` 使用幂等插入，网络响应丢失后重试不会触发唯一键错误。
- `store_latest` 每轮更新，保证实时状态的新鲜度。
- `store_dimension` 仅静态字段变化或日期跨天时实际更新。
- rollup 仅业务字段变化时更新，单纯 `updated_at` 变化不会产生实际写入。
- 定时 daily rollup 只刷新最新日期和行数不完整的日期；首次升级会做一次 30 天基线修复。
- 每日聚合和归档用 `collector_runs` 的确定性任务 ID 记完成状态，重启后不会重复执行。
- 维护任务只在 02:00 到营业开始前补跑，白天重启不会阻塞实时采集。
- 归档不再为日志全表统计剩余行数；`archive_runs.raw_remaining = -1` 表示主动跳过该计数。

30 天聚合仍需读取窗口内原始帧，但采用 `id` keyset 分页，每页 `5000` 行。这样单页失败只重试
该页，不会再让约 50 万行、上百 MB 的单个 HTTP 响应反复重读。

## 推荐的需求拆解顺序

收到“某店某天几点去吃、几点取号、一般叫到多少号”这类问题时，按以下顺序查询：

1. 用 `store_dimension` 确认门店 ID。
2. 用 `holiday_calendar` 和日期规则确认 `date_type`。
3. 用 `store_bucket_rollups` 看长期时段形状和样本量。
4. 用 `daily_store_bucket_rollups` 检查跨日期稳定性。
5. 用 `queue_snapshots` 查看目标时刻的真实叫号和等待快照。
6. 给出区间，不给单点承诺。
7. 如果是当天决策，再用 `store_latest` 和最近 15 至 30 分钟快照修正。

## 凭证安全

- 只使用权限声明为只读的 token。
- 不在 Markdown、JSON、Shell 脚本或 Git 配置中保存 token。
- 不把 token 放在命令行参数中，避免进入 shell 历史和进程列表。
- 本地长期保存时使用桌面端 MCP 配置或操作系统安全存储，并确保文件权限为 `0600`。
- 如果 token 曾进入公开日志、提交记录或公开聊天，应撤销并重新生成。
