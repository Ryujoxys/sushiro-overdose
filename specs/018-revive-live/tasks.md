# Tasks: 实时数据恢复与离线可用

- [x] 共享接口：`ErrLiveAuthRequired`、`QueueLiveStore.ObservedAt/Source`、`liveErrorCode` / `writeLiveError` / `writeErrorCode`，401/403/451 不再提 VPN
- [x] A 会员连接：`live_session.go`（解析 curl / 请求头 / HAR、只保存更新的连接、0600、JWT 到期）与 `/api/live-session*`
- [x] A 网关客户端：有连接走新网关、无连接零请求、`SUSHIRO_TOKEN` 调试旧地址；解码加固；30 秒单店缓存
- [x] A 采集熔断：认证类失败停到换新连接或 6 小时，其余退避 30s→15m；状态字段；共享写入器 `recordQueueLiveSnapshots`
- [x] A 进程边界：只有界面进程注入 provider；守卫测试
- [x] B 离线叫号速度与叫到时间（回测 1 小时 MAE 18.1 分钟）
- [x] B 想几点吃历史兜底、候选不早于现在、路程透传、实时失败快速跳过
- [x] B 离线门店目录、`/api/queue/reading`、观测未知等待
- [x] 集成：路由与桥白名单；保存新连接时清除实时失败记忆
- [x] 技能 `eta --called`、MCP `arrival_advice(called_no, called_at)`
- [x] 宪章、AGENTS.md、内置历史规则、README、线上现状文档
- [ ] C 代理解密 sapi、捕获会员连接、被动记录、取号前状态查询失败即停
- [ ] D 界面：实时数据卡片、离线输入与叫到时间、门店离线目录、记录状态文案
- [ ] 全量评审与修复、验证记录、版本说明
