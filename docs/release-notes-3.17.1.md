# v3.17.1

## fix: 移动端顶部导航切页后当前入口不可见

390px 宽度下顶部导航会横向滚动，但切到「设置」「我的单据」这类靠右入口时，active tab 仍停在滚动区域外。

- 切页 / 切模式后，用相对 `.nav` 的视口几何把当前高亮项滚进可视区（不再用易错的 `offsetLeft`）。

## fix: 交互细节

- 切页后自动滚回顶部，避免从长设置页跳走仍停在半屏。
- 「设置通知」会展开通知折叠卡再聚焦输入框。
- 全国门店选择器：自动聚焦搜索、回车确认、打开时锁定背景滚动。

## fix: 线上 Turso 基准「看起来没拿到」

本机当天采样 ≥ 8 点时，旧逻辑会整段丢掉线上基准曲线，UI 只剩本机采样。

- 本机 + 线上都有时始终 **mixed 叠加**（重叠时刻本机优先，空档用线上补全）。
- 云端路径：GitHub 登录 → Cloudflare Worker → Turso；**密钥只在 Worker secrets**。

## fix: 云端默认地址改 workers.dev

- 默认 `https://sushiro-cloud.sushiro-ryujoxys.workers.dev`，降低自定义域名到期后登录/基准全挂的风险。
- 自定义域 `ryujo.online` 变为可选过渡。

## refactor: Web UI 静态资源拆分

- HTML/CSS/JS/logo 从巨型 Go 字符串迁到 `internal/app/webui/`，`//go:embed` 组装，行为与入口不变。

## fix: Windows 双击版并行配置（SxS）风险

- 重建 `resource_windows_*.syso`：图标 + 干净 application manifest。
- **禁止**声明 VC CRT / Common-Controls（纯 Go 不需要，坏 WinSxS 机器上会报「并行配置不正确」）。
- 发版 CI 增加 syso 清单守卫与 exe 体积检查；可用 `./scripts/gen-windows-resources.sh` 再生资源。

## 验证

- `gofmt` / `go vet` / `go test ./...` / `go build` 通过。
- 压力曲线在本机密采样下仍 `source=mixed` 且含 `remote_baseline` 点。
- Windows 资源测试 `TestWindowsResourceSysoHasCleanManifest` 通过。
