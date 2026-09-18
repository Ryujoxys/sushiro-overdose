# 实施方案

## 模块

- `webui/index.html`、`app.css`：记录、手动取号、设置三个页面，延续原型的红色与暖白。
- `webui/app.js`：记录服务、历史分布、门店选择及只读建议。
- `webui/auth_ticket.js`：认证、单次确认、查号及显式取消。
- `local_records.go`：按时间和日期类型读取本机 JSONL，分店聚合、去重和导出。
- `automation_retired.go`：旧自动接口统一 410；删除自动引擎与调度器。

## 数据与调用

- `queue_baseline.jsonl` 仍由公开采集器写；记录页面和导出只读。
- `queue_baseline.json` 由用户保存门店和间隔；后台与自启动仍由服务入口控制。
- `queue_model.json` 由本机模型重建维护；不得混入凭证、票号。
- `netticket_plan.json` 只兼容旧结果，不再调度；历史计划不删除。
- 唯一远端取号写调用来自显式 `handleQueueTicket`，取消排队与预约分别走原有独立入口。
- 旧命令给出迁移说明，不唤醒旧后台；不调整用户操作系统中的现存进程。

## 验证

- Go 单元测试覆盖历史范围、去重、无数据、错误输入、导出及退役路由。
- 架构守卫禁止应用层调用 `CreateReservation`，仅允许手动 handler 调用 `CreateNetTicket`。
- Node 行为测试覆盖只读进入、认证确认、重复点击、未知结果、过时请求。
- 隔离数据目录预览桌面与手机页面；认证和订单只用模拟结果，不发真实请求。
- 全量 Go 测试、静态检查和 Windows/Linux 交叉编译。
