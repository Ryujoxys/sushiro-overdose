# 云端数据入口下线

GitHub 登录和共享数据库访问已从桌面端与 MCP 源码移除。当前历史数据只在本机积累；实时排队仍请求寿司郎官方公开接口。这不是离线运行模式。

## 当前行为

- 桌面端不再读取 `cloud_auth.json`、`queue_baseline_remote.json` 或数据库环境变量，也不会上传历史。
- `/api/cloud` 及其子路径统一返回 HTTP 410，不跳转 GitHub、不接受 OAuth 回调、不保存 session。
- MCP 只连接本机桌面端，不再依赖 `libsql-client` 或数据库 token。
- 旧本地配置文件不自动删除；其中旧凭证不再使用。`mcp_config.json` 下次保存时会丢弃旧数据库字段。
- GitHub Release 更新检查保留，它与 GitHub 登录无关。

## 当前处理范围

维护者已说明网站自定义域名过期。本轮按要求只下线客户端云入口，线上只读 key 暂不处理，不执行 Worker 部署、凭证撤销或数据库删除，也不将这些操作作为本次改动的发布前置条件。

**域名过期不等于已确认 Worker 停服。** 仓库的 `wrangler.toml` 仍配置 `workers_dev = true`，不能只凭自定义域名失效判断所有入口的实际可达性。本轮不做线上探测，实际部署状态保持未核实；桌面端和 MCP 无论线上是否可达，都不再请求旧数据入口。

## 后续可选清理

以下清单仅在维护者后续决定彻底清理线上服务时执行，本轮暂缓：

1. 将 `cloudflare/sushiro-cloud/src/index.js` 中的停用版本发布到所有实际入口，或禁用对应 Worker、路由和自定义域名。
2. 确认旧登录路径、回调、`/api/me`、基准查询路径均不再返回数据；停用版本统一返回 410 且不访问上游。
3. 撤销旧数据库只读 token，停用 GitHub OAuth 应用或撤销其 client secret，移除 Worker 上遗留的数据库、OAuth、会话签名 secrets。
4. 检查是否存在其他历史部署、代理或直连 token。仅下掉页面不能撤回已发给客户端的凭证或已经复制的数据。
5. 如不再采集线上数据，单独停掉服务器上的 `collector.service` 及相应定时任务。仓库中的采集器是独立运维工具，不会由桌面端启动。

不删除原数据库，避免丢失历史；是否保留备份和继续服务端采集由维护者决定。未来重新开放前按 [数据协议](local-data-contract.md) 做适配、隐私审查、限流和授权设计，不恢复共享直连 token。

## 无部署回归测试

```bash
go test ./internal/app -run 'TestRetiredCloud|TestLocalHistory|TestMCPDrops|TestWebUIDoesNot'
node --test cloudflare/sushiro-cloud/test/retired.test.mjs
```
