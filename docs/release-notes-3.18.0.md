# v3.18.0

## feat: `sushiro version` 命令

- `sushiro version` / `sushiro -v` / `sushiro --version` 打印版本号并退出。

## fix: 趋势分桶时区归一（非 UTC+8 机器）

同一时刻的等待样本与叫号推进会因时区不一致落到不同日期/日型桶：

- session 循环、观测第一遍、`queueSessionBucket` 三处统一先归一到门店时区（UTC+8）再切日期/日型/半小时桶，与 advisor 压力曲线既有做法对齐。
- UTC 容器/海外机器上，周五晚餐高峰不再被劈成「上午的另一个桶」。

## perf: 取号不再被后台 tick 卡住

- armed 状态的「每天想几点吃」Routine 每 20s tick 都在重跑全量重算（读全部趋势文件、可能打远端基准，秒级），全程持锁，把用户手动取号/改计划卡在后面。
- 现在 armed 直接用已落盘的取号窗口判断是否到提醒时刻：持锁时间从秒级降到微秒级，窗口结论一天内不变也不浪费请求。
- `needs_notify` 且通知渠道仍未配置时同样跳过重算。

## perf: 排队数据读取缓存 + 行数裁剪

- `queue_observations.jsonl` / `queue_baseline.jsonl` 的 loader 加 (size, mtime) 内存缓存：advisor / trends / dashboard 单个请求内多次读盘的路径现在只解析一次。
- 两个文件加 10 万行上限的周期裁剪（与 `history.jsonl` 同模式），长期运行不再膨胀到几十 MB 拖慢曲线响应。
- loader 不再静默吞掉 `scanner.Err()`：磁盘错误/半截数据现在有日志。

## fix: collector 读写加固（云端基准链路）

- 采集配置启动校验：`interval_seconds` 非法（≤60s、负数、乱字符串）回退默认 900s，不再有「每秒一轮」hammer 上游的风险；`active_hours` 容忍字符串写法，非法结构回退 `[10, 22]`，不再崩溃循环。
- 每店每轮只写一帧：详情成功写 `store_detail`，失败才用列表帧兜底，原始快照量减半。
- 关店退役：完整采集轮里没见到的门店置 `is_active=0`（Worker 不再永久导出关店门店的过期数据）；列表疑似截断（<30 家）时跳过，门店回归自动复活。
- 聚合按主键分页读取（每页 5000 行），超大响应中断后不再整批重读；rollup UPSERT 只有业务字段变化才更新；daily 表只刷新最新日和缺行日期。
- 维护任务（聚合/归档）幂等：完成状态写 `collector_runs`，进程重启不重复执行；只在 02:00 到营业开始前补跑。

## fix: Worker 导出桶宽契约

- Cloudflare Worker 导出的 `bucket_minutes` 从 10 修正为 30，与 collector 实际的 30 分钟 rollup 桶一致（此前桌面端会拿错误值覆盖默认桶宽）。
- 需 `wrangler deploy` 生效。

## 验证

- `go test ./...`（6 包）、`go vet`、`gofmt`、`go mod tidy` 检查通过。
- 新增回归测试：时区分桶、周末窗口边界、armed 快路径状态机、读缓存命中/失效/缺文件、JSONL 裁剪、runner 配置校验、关店退役（Go + Python 共 28 个新用例）。
- collector 测试 24 个全部通过。
