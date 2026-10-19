# 实施计划

## 页面

  `app.css` 以变量承载颜色，深色模式在 `prefers-color-scheme: dark` 下覆盖同名变量。按钮底色 `--red` 与文字强调色 `--red-text` 分开，深色背景上的红字和焦点圈使用更亮的 `--red-text`。图表 SVG 只输出类名，颜色由 CSS 决定。

  记录页引导根据 `/api/queue/service` 已返回的门店、启用状态和样本数渲染。取号卡片在 `loadTicketForecast` 中复用 advisor 的 `current.called_no` 计算剩余号数；进度条起点为本次查看到的首个叫号，只随叫号前进。

## 记录与恢复

  “今天”曲线复用 `/api/records?date=今天`，仅在多日视图、日期类型为全部且当天有记录时请求。恢复由 `local_records_import.go` 实现：先读已有键，再完整扫描上传内容（单行 4 MB、总计 256 MB 上限），全部通过后按时间排序一次追加，复用采集的文件锁；`filterLocalRecords` 与恢复共用同一去重键。桌面桥新增 `ImportRecords`，经系统打开框选择文件后在 Go 侧上传，页面无法直接调用导入接口。

## 桌面与更新

  Wails Windows 主题改为跟随系统，macOS 不再强制浅色外观。桌面桥白名单增加只读 `/api/update`；新增无参数的 `OpenReleasePage`，只打开常量发布页。`CheckLatestRelease` 成功结果缓存 30 分钟，失败结果 1 分钟。

## 维护

  删除自动抢号下线后无调用的函数（staticcheck U1000 / SA4006），交叉编译各平台与 desktop 标签确认无平台文件引用。`health.go` 仍无调用方，留待单独移除。

## 验证

  `TestRecordingFlowEndToEnd` 通过真实 handler 串起选店、开始、采集写盘、模型、曲线、按日、导出、改间隔、暂停与清空门店，仅替换网络抓取；`TestRecordsRestoreIsAdditiveAndValidated` 覆盖恢复。Go 单元测试覆盖桥白名单、固定发布页、导入只走系统对话框和更新缓存重试。Node 行为测试与浏览器 QA 覆盖引导、进度、更新检查和深色截图；浏览器 QA 使用独立 `SUSHIRO_DATA_HOME` 和模拟 API，不触发认证或真实取号。
