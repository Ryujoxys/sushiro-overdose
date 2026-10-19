# 验证记录

日期：2026-10-19。基于 v4.0.1 之后的 `a2514e3`，发布 v4.0.2。

## 已通过

- `gofmt -l .` 为空，`go vet ./...`、`go mod tidy` 无变化，`go test -race ./...` 全部通过；`-tags desktop,production` 下桥与 Windows 资源测试通过。
- 新增 `TestRecordingFlowEndToEnd`：经真实 handler 完成选店、开始、一轮采集写盘、模型重建、多日与按日曲线、近 7 天、筛选导出、全量备份、改间隔、暂停、清空门店及“无门店不可开始”，全程不生成账号凭证。仅替换网络抓取。
- 新增 `TestRecordsRestoreIsAdditiveAndValidated`：首次恢复追加 4 行并重建模型，坏行/未来时间/门店为空计为无效；重复恢复只跳过；空文件、无可用行和超长行返回 400 且记录文件逐字节不变。
- 桌面桥：`/api/update` 只读、`OpenReleasePage` 只打开常量地址、页面不能直接 POST 导入接口、取消系统打开框不发请求。
- 公开接口错误：403 拦截页、限流、5xx、连接重置、代理连接失败、EOF 均转为可读文案，页面不出现 HTML 或请求地址。
- Node 行为测试 9 项、四个页面脚本 `node --check`、MCP 适配测试 7 项通过。
- 浏览器 QA（模拟 API、隔离 `SUSHIRO_DATA_HOME`）：三步引导、开始记录不触发自启动、今天曲线与读数、门店行叫号、手动取号与进度、取消、结果未确认、恢复备份、检查更新、深色模式、手机宽度无横向滚动。真实内置历史 QA 通过，未产生外部请求。
- Linux / Windows / macOS（amd64、arm64）CLI 交叉编译，Windows 两种架构的 desktop 构建通过；Windows 资源重新生成为 4.0.2.0，清单检查通过。
- `govulncheck` 未发现已知漏洞；staticcheck 剩余项为 `health.go`（待单独移除）、Windows 专用解析函数的误报和 `sampling.go` 的有意重读配置。

## 未验证

- 本环境访问寿司郎公开接口返回 403 / 连接被重置，未用真实官方数据采集；该路径由上面的集成测试与既有单元测试覆盖。
- 未在 macOS / Windows 实机打开原生窗口检查深色外观、系统打开框和浏览器跳转；macOS desktop 构建需要 CGO，由发布工作流完成。
- 未做真实认证、取号或取消。
