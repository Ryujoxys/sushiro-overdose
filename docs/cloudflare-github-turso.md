# 云端入口下线与线上现状

## 旧公开数据网站：已放弃

旧网站是 Cloudflare Worker `sushiro-cloud`（`sushiro-cloud.sushiro-ryujoxys.workers.dev`，自定义域名 `sushiro-cloud.ryujo.online` 已过期），数据导出需要 GitHub 登录。维护者 2026-10-20 决定不再保留，Worker 源码已从仓库删除。它读取的 v1 数据表也已删除，线上那份旧部署已无数据可返回。

仓库删除源码不会停掉线上部署。彻底清理需要维护者在自己的 Cloudflare / GitHub 账号里操作：

1. Cloudflare 控制台 → Workers → `sushiro-cloud` → 删除（或 `wrangler delete sushiro-cloud`），连同路由和自定义域名。
2. GitHub → Settings → Developer settings → OAuth Apps：删除旧网站用的 OAuth 应用。
3. Turso：撤销当年下发给 Worker 的数据库 token（只保留采集器的读写 token 和导出工作流的只读 token）。

替代入口是仓库 `data` 分支，无需登录，见 [public-data.md](public-data.md)。

## 桌面端

GitHub 登录和共享数据库访问已从桌面端与 MCP 移除：

- 不读取 `cloud_auth.json`、`queue_baseline_remote.json` 或数据库环境变量，不上传历史；旧本地配置文件不自动删除，其中凭证不再使用。
- `/api/cloud` 及其子路径统一返回 HTTP 410。
- MCP 只连接本机桌面端；`mcp_config.json` 下次保存时丢弃旧数据库字段。
- 桌面端不读 `data` 分支；历史曲线只用本机记录和内置历史包。GitHub Release 更新检查保留。

## 数据库与采集（2026-10-20）

- 数据库已精简为 v2：`queue_records` 等 5 张表，数据约 16MB，见 [查询指南](turso-readonly-query-guide.md)。
- 公开数据从 2026-09-06 21:30 起没有更新。采集器在那之后停了六周，10-18 恢复后每轮都返回 HTTP 403。
- 403 的原因：寿司郎小程序本身正常，实时叫号服务也没有下线，但小程序已改用 `sapi.sushiro.com.cn/gateway/...`。它先发一次带 `js_code`、设备信息和签名的初始化请求，换到绑定会员账号、有效期一小时的 token，到期后自动重新获取。本项目仍用旧地址 `crm-cn-prd.sushiro.com.cn/wechat/api/2.0` 加旧固定 token，所以被拒绝。用有效的新 token 仍能拿到叫号、在等桌数和门店数据。
- 已完成：抓包、新旧接口对照、续期流程排查。未完成：业务代码适配和自动续期；独立程序能否脱离微信完成续期尚未验证。
- 影响：桌面端的实时叫号刷新、开始记录 / 后台记录、叫号提醒，以及服务器采集器，目前都取不到新数据；已有本机记录和内置历史仍可查看。应用遇到这类拒绝时会直接说明原因，不再提示关 VPN。
- 修复方向：本机授权采集（用户授权会员 token 后按新接口请求），这是方向，还不是已恢复的功能；长期无人值守的采集暂时不能保证。

## 回归测试

```bash
go test ./internal/app -run 'TestRetiredCloud|TestLocalHistory|TestMCPDrops|TestWebUIDoesNot'
```
