# 本地与未来共享数据协议

## 边界

运行时没有在线数据库读取或上传入口。个人模型只用本机历史；“排队规律”图表另可使用随二进制固定分发的离线历史包，带真实截止时间和持久化关闭开关，见 [离线历史数据包](bundled-history.md)。未来服务端应适配同一份版本化协议，不应让页面或 MCP 直接接 SQL 数据库。

唯一的基准类型定义在 `internal/app/queue_baseline.go`，本机实现为 `buildLocalQueueBaselineExport`。这些类型尚未移到独立数据包，避免此次下线同时引入大规模领域重构。

## 公开快照

| 字段 | 类型与含义 |
|---|---|
| `collected_at` | RFC3339 时间，允许偏移量；聚合前统一为门店时区 UTC+8 |
| `store_id` | 正整数，门店标识 |
| `name`、`city`、`area` | 门店公开维度 |
| `wait_minutes` | 非负整数分钟，0 是有效观测，不等同于缺失 |
| `group_queues_count` | 非负整数桌数，0 是有效观测 |
| `store_status`、`net_ticket_status`、`reservation_status` | 原接口状态字符串，不猜测未知状态 |
| `online_open` | 是否开放网上取号，不是应用云端连接状态 |
| `display_called_no` | 当前公开叫号；缺省或 0 不用于叫号统计，不是个人手中号 |
| `group_queues_json` | 公开分组叫号信息序列化字符串，可缺省 |
| `wait_time_counter`、`wait_time_cap` | 原接口等待统计字段，不能直接当实际等待时间 |
| `updated_at`、`source_endpoint`、`api_profile_version` | 采集来源元数据 |

当前本地快照的非指针数值无法表达“未知”和 0 的区别。未来提供者必须只提交有依据的数值；如需缺失值语义，应升级协议并迁移，不能把未知强行写成 0。

内置历史包是上述公开字段的最小、独立版本化投影：`wait_minutes` 明确允许 `null`。只在图表查询层转换为可空代表值，与本机记录按统一的半小时粒度合并，不能写回旧 JSONL 或混入协议 1 的个人模型缓存。

## 聚合信封

固定 `version: 1`、`bucket_minutes: 30`，本机 `source: "local"`，包括 `generated_at`、`date_types`、`stores`、`latest`、`rollups`、`stats`。

| 聚合字段 | 约定 |
|---|---|
| 唯一维度 | `store_id` + `date_type` + `weekday` + `time_bucket` |
| `weekday` | ISO 星期，周一 1 至周日 7，不使用 Go 的周日 0 或 Python 的周一 0 |
| `time_bucket` | `HH:MM`，半小时窗口起点，门店时区 |
| `date_type` | `workday` 调休工作日优先于 `holiday`，然后按现有周末窗口策略，否则 `weekday` |
| `sample_count` | 去重后的有效快照数，不是天数、人数或实际等待次数 |
| `open_rate`、`online_open_rate`、`busy_rate` | 0 至 1 的快照比例；繁忙为排队桌数大于 0 |
| `wait_typical_minutes`、`wait_safe_minutes` | 等待分钟 P50、P80，缺失可省略，不能伪造估计 |
| `queue_groups_typical`、`queue_groups_safe` | 桌数 P50、P80 |
| `called_no_slow`、`called_no_typical`、`called_no_fast` | 公开叫号号码 P20、P50、P80，**不是每分钟叫号速度** |
| `called_sample_count` | 有效正数公开叫号的快照数 |
| `confidence` | 现有样本阈值分类，不是已统计校准的概率 |

同店同一瞬间重复快照按输入顺序保留首条，时区表示不同但瞬间相同也去重；无效时间、未来时间、非法门店编号不参与本机基准聚合。`stores`、`rollups` 为空时仍为数组，`latest` 无样本可省略。

## 旧数据适配

- `queue_baseline.jsonl` 旧 `ts`、`wait` 字段仍兼容，读取时映射至 `collected_at`、`wait_minutes`。
- 私有 `QueueObservation.store_id` 是字符串，进入公开协议前要验证并转换成正整数；不能直接 JSON 拼接。
- 仓库独立 Python 采集器当前数据库仍使用 `weekday = 0..6`。本次不修改线上表或采集任务。未来输出本协议必须转换 `weekday + 1`，不可把旧行伪装成协议 1；迁移时要避免新旧主键混用。
- Python 数据库中 `source`、时区、缺失值和日型逻辑也必须经过适配与样例对照，保留字段名不代表已完成数据库迁移。

## 本机模型与采集状态

`queue_model.json` 外层 `version: 1`、`method: "local_empirical_quantiles"`，包含 `generated_at`、`sample_count`、`day_count`、原始数据/节假日表文件指纹，以及 `baseline` 聚合信封。模型外层只用于本机缓存；未来交换仍以 `baseline` 协议为准，不能把本机文件指纹当服务端游标。

样本数是去重后有效快照数，覆盖天数按 UTC+8 自然日去重，不代表每个门店或每个时段都有这些样本。达到 7 天、100 条只表示存在跨日历史，不代表统计显著性或预测准确率。每条聚合的 `sample_count` 仍是判断局部覆盖的依据。

`queue_collection_state.json` 保存共享的上次采集时间、错误、避让原因、门店集合和进程心跳。心跳超过两分钟或进程不存在时不能显示运行中。`queue_service.lock`、`queue_collection.lock`、`*.jsonl.lock` 和 `queue_alerts.lock` 均为本机 OS 锁文件，不包含凭证；正常释放只关句柄、不删文件。

## 个人记录保留与备份

`queue_baseline.jsonl` 和 `queue_observations.jsonl` 不再按总行数自动删除历史；维持原文件路径和格式。此前已被旧版本裁剪的数据无法自动恢复，长期分月存储另行设计。

`/api/records` 的 `export_record_count` 是当前门店、时间范围、日期和日型下的个人原始快照数，`total_record_count` 是全部有效去重个人快照数，均不含内置历史，不能与半小时曲线样本数混用。`stores` / `available_stores` 保留城市元数据供本地搜索使用。

图表导出携带当前筛选；设置的 `/api/records/export?days=all` 不附加其他筛选，备份所有有效个人记录。空导出返回 HTTP 409 和中文错误，不生成空附件。本轮未增加导入恢复协议。

## 隐私与重新开放条件

协议只能接收公开门店快照。`QueueSession`、个人票号、微信标识、手机号、Authorization、应用 session、设备 UA 和原始抓包不得进入信封。当前没有自动导出或上传端点。

未来开启在线提供者前必须完成：协议版本校验、门店白名单、样本新鲜度与覆盖提示、只读授权和限流、字段级脱敏，以及 Go/Python/前端共享样例测试。不能把数据库 token 下发给客户端。

回归测试入口：`TestLocalBaselineContractDeduplicatesAndUsesISOWeekdays`、`TestLocalHistoryIgnoresLegacyCloudConfiguration` 和 `scripts/test-webui-records.mjs`。
