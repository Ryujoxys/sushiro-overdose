# 公开排队数据（`data` 分支）

维护者服务器上的采集器（`collector/`）每 15 分钟读取寿司郎公开门店接口写入数据库。`.github/workflows/public-data.yml` 每天北京时间约 04:00 用**只读** token 导出一份到本仓库的 `data` 分支，供任何人和 agent 无需 key 读取：

```
https://raw.githubusercontent.com/Ryujoxys/sushiro-overdose/data/<文件>
```

桌面端不读这个分支，仍只用本机记录和内置历史包（见 [local-data-contract.md](local-data-contract.md)）。配套技能在 `.claude/skills/sushiro-queue/`。

## 数据边界

- 只有门店公开状态：营业状态、预估等待分钟、当前叫号、门店名称和地址。没有账号、手机号、票号、凭证或个人记录。
- 数据库 `queue_records` 本身就是每店每半小时一条（该半小时内最后一帧），导出不再重采样。缺失和未知保持 `null`，不补值、不当 0。等待到达官方显示上限时记为 `180`，意思是“180 分钟及以上”（`meta.wait_cap_minutes`）；高峰常到上限，丢掉会让规律偏短。应用内置历史包仍把超过上限的值记为未知，两者在高峰时段会不同。
- 默认只发布最近 180 天，分支每次整体替换为一个提交，不保留旧版本历史。
- `latest.json` 是导出时刻的最近一次采集，最长约 24 小时前，**不是实时**。实时叫号用本机应用的 `/api/queue/live`。
- 采集器错误详情不公开，`meta.json` 只给最近运行次数与成功时间。

## 文件

| 文件 | 内容 |
|------|------|
| `meta.json` | `schema`=`sushiro-public-data`、`version`、`source`（`turso` 或 `bundle`）、`generated_at`、`first_at`、`cutoff_at`、覆盖统计、`quality`/`quality_note`、`collector`、`calendar`、`method`、`files`（每个文件的 `bytes` 与 `sha256`） |
| `stores.json` | 门店数组：`id,name,city,area`（数据库来源另有 `address,latitude,longitude,active`），`first_at,last_at`，`days`（有任一可用值的天数），`wait_days`，`called_days` |
| `latest.json` | `{source, stores:[{store_id, collected_at, store_status, online_open, net_ticket_status, wait_minutes, group_queues_count, display_called_no}]}`；`bundle` 来源为空数组 |
| `patterns/<id>.json` | `{store, first_at, last_at, date_types:{all,weekday,weekend,holiday,workday:[bucket]}}` |
| `daily/<id>.json` | `{store_id, fields, rows}`，`fields` 为 `collected_at,date_type,store_status,wait_minutes,display_called_no,group_queues_count,online_open`；后两列在 `bundle` 来源为 `null` |

`bucket`：`time`（北京时间半小时起点，如 `18:00`）、`wait_days`、`wait_p50`、`wait_p80`、`called_days`、`called_p20`、`called_p50`、`called_p80`。只统计营业中的值，分位数线性插值，与应用图表一致（等待“通常要等”=P50、“八成日子不超过”=P80，叫号“常见范围”=P20–P80）。`*_days` 是参与统计的天数，少于 3 天时只能当参考。

`date_type`：节假日表（`meta.calendar`，`holiday`/`workday`）优先；否则周五 16:30 起、周六全天、周日 22:00 前为 `weekend`，其余 `weekday`。

`collector`：`last_run_at`、`last_ok_at`、`recent_runs`、`recent_ok`（最近 96 次采集）。`last_ok_at` 落后很多说明公开接口或采集服务出了问题，数据停在 `cutoff_at`。

## 读取要求

- 先读 `meta.json`，确认 `schema` 和 `version`；其他文件按 `files` 中的 `sha256` 校验和缓存。
- 回答时说明数据截止时间和天数；`*_days` 少、`quality_note` 不为空或 `cutoff_at` 很旧时要明说。
- 叫号是号码，不是人数；不同日子开门时号码起点不同，只能比较同一时段。

## 维护

```bash
# 本地用内置历史包生成（无需 key）
node scripts/export-public-data.mjs --out /tmp/sushiro-data --source bundle

# 用只读 token 生成；token 只放环境变量或 0600 配置文件，不写命令行
SUSHIRO_HISTORY_DATABASE_URL=libsql://... SUSHIRO_HISTORY_DATABASE_TOKEN=... \
  node scripts/export-public-data.mjs --out /tmp/sushiro-data --days 180

node --test scripts/test-public-data.mjs
```

工作流需要仓库 Actions secrets `SUSHIRO_HISTORY_DATABASE_URL` 和 `SUSHIRO_HISTORY_DATABASE_TOKEN`（只读 token）。缺少时定时任务跳过并留下提示；手动运行可选 `bundle` 来源先发布内置历史。导出脚本只发送 `SELECT`/`WITH` 查询。
