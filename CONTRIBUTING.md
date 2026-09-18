# 参与贡献

项目当前以本地排队记录和手动取号为主。不恢复自动抢号，也不把个人数据上传共享数据库。

## 反馈问题

请说明系统、应用版本、操作步骤、预期与实际结果。有截图或脱敏诊断包更容易定位；不要上传账号凭证、CA 私钥或完整个人数据目录。

功能建议先说清楚使用场景。涉及认证、取号、取消或数据协议的大改动，先开 Issue 对齐范围。

## 开发检查

需要 Go 1.25+。业务核心只用标准库；原生桌面适配使用固定版本 Wails v2。页面和历史包内嵌，不需要前端打包或 Wails CLI。

普通 `go build -o sushiro .` 生成 CLI/浏览器版。macOS 和 Windows 原生版：

```bash
go build -tags desktop,production -o sushiro .
```

macOS 需要 Xcode Command Line Tools 和 CGO，运行系统最低 macOS 12；Windows 使用 WebView2。发布构建在 macOS 执行 `bash scripts/build-desktop.sh 4.0.1 dist`，同时产出 Universal 原生二进制及两种 Windows GUI EXE。Windows 发布标签包含 `wv2runtime.browser`，缺少运行时时引导用户到微软官网安装。

依赖升级后，在 macOS 执行 `node scripts/generate-third-party-notices.mjs` 更新内嵌许可；用户在「设置 → 开源许可」查看。README 截图可用隔离预览配合 `node scripts/capture-readme.mjs` 生成，只读取界面，不启动采集或认证。

```bash
gofmt -l .
go vet ./...
go test -race ./...
go build ./...
node --test scripts/test-webui-records.mjs scripts/test-history-export.mjs cloudflare/sushiro-cloud/test/retired.test.mjs
# macOS / Windows：不打开窗口的原生适配检查
go test -tags desktop,production ./internal/app -run 'TestDesktop|TestLocalWeb|TestWindowsResource'
```

MCP 适配测试可在 `mcp/` 中运行 `python3 -m unittest discover -s tests -v`。

检查工作流仅手动触发；push、PR 和发布不运行全量测试。自动测试必须隔离用户目录，通过内存通知器替换真实通知，不得弹系统消息、发送外部推送或提交真实订单。

## 隔离预览

```bash
go build -o /tmp/sushiro-preview .
SUSHIRO_DATA_HOME="$(mktemp -d /tmp/sushiro-data.XXXXXX)" /tmp/sushiro-preview web
```

不要覆盖 `HOME` 或 `USERPROFILE`。独立目录不迁移旧凭证、不修改正式自启动，但不是系统沙箱；主动认证仍可能安装证书、设置代理。

浏览器检查需要 Playwright。用 `SUSHIRO_QA_BROWSER` 指定 Chromium 路径，`SUSHIRO_QA_URL` 指定隔离预览地址。

- `node scripts/check-webui-browser.mjs`：模拟 API 检查界面与手动取号交互，不提交真实订单。
- `node scripts/check-history-browser.mjs`：使用无个人记录的新数据目录，检查真实内置历史、日期筛选和开关持久化；不启动采集、认证或取号。

## 提交约定

1. Fork 后创建分支，PR 提交到 `master`。
2. 只提交当前改动需要的文件，不带个人配置、构建产物或凭证。
3. 使用中文 conventional commits 提交信息，并说明验证结果与未验证范围。
4. 涉及核心流程或协议时，同步 `specs/`、测试和文档。

包依赖方向为 `core` → `api/proxy/platform/notify` → `app`。新后台记录生命周期放在 `queue_service.go`，`engine.go` 仅管理认证；不要重新引入自动预约调度。详见 [ARCHITECTURE.md](ARCHITECTURE.md)。

## 发布

维护者先完成本地检查，编写 `docs/release-notes-<版本>.md`，同步 Windows 资源版本，重新生成 syso，再提交和推送 tag。

版本名称固定为「lite正式版」。v4.0 系列修复使用 `v4.0.x`，本次为 `v4.0.1`；安装包版本与 tag 去掉 v 后完全一致。

发布工作流先创建草稿，确认各平台安装包与完整 SHA-256 校验表均已上传，再公开发布。不要手动把未完成的草稿设为 Latest。

Release 页面仅保留最新正式版。新版本完全公开并验证下载后，维护者再删除旧 Release 及附件；保留所有 Git tag 和源代码历史。不要在新版本构建完成前清理旧下载。
