# v4.0 · lite正式版

这一版做了减法：以记录自己的排队数据为主，需要吃饭时再手动取号。

## 记录自己的数据

选几家常去的店，点击「开始记录」就能积累数据，不需要账号、证书或代理。可以单独开启登录电脑后自动记录；后台服务运行时，关闭界面仍可继续。电脑休眠、退出登录或关机时暂停。

记录和模型都保存在本机，支持导出 JSONL。没有 GitHub 登录，不连接共享数据库，也不上传自己的记录。

## 打开就有曲线

内置固定历史包，覆盖 138 家门店、81 个记录日，截止 **2026-09-06 21:30:17（UTC+8）**。数据随安装包分发，不会每次打开都去线上拉取。

曲线可以看「叫到几号」或「等多久」。点选或拖动时段，查看通常叫到的号码、常见范围和样本天数；也能按日期类型或具体日期筛选。

关闭「包含历史数据」后，只看自己的记录，重启保持选择。历史包不会写入个人模型、个人记录数量或导出文件。同店同日同半小时，优先采用本机记录。

历史叫号是堂食队列的最大号码，不分桌型，不是当天实时叫号。缺失不当作零，样本少时会注明。

## 取号回到手动

选店、核对人数和桌型、必要时连接本人账号，再确认提交。认证成功不会直接取号；结果不明确时先查询，不自动重复提交。

「想几点吃」保留为时间建议，不预约未来入座时间。狙击、循环抢号、定时取号和每日自动计划已移除，旧计划不再执行。

Windows、Linux 默认引导手机微信认证，macOS 默认电脑微信；也可手动导入凭证。认证仍受微信版本、证书和网络环境影响，不承诺免配置或所有设备都可登录。只记录数据可以跳过认证。

## 界面更简单

主要入口收成「我的记录」「手动取号」「设置」，保留原来的红白视觉。叫号图支持鼠标、键盘和手机拖动，数据来源与历史截止时间直接显示在图旁。

同时整理了后台记录、代理恢复和本地数据边界。检查工作流改为手动触发；自动测试隔离通知，不再弹真实系统消息。

## 下载

| 系统 | 安装包 |
|------|------|
| Windows 常见电脑 | [Sushiro-Overdose-4.0-windows-amd64.exe](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0/Sushiro-Overdose-4.0-windows-amd64.exe) |
| Windows ARM | [Sushiro-Overdose-4.0-windows-arm64.exe](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0/Sushiro-Overdose-4.0-windows-arm64.exe) |
| macOS Intel / Apple 芯片 | [Sushiro-Overdose-4.0-macOS.dmg](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0/Sushiro-Overdose-4.0-macOS.dmg) |
| Linux x86-64 | [sushiro-overdose_4.0_linux_amd64.tar.gz](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0/sushiro-overdose_4.0_linux_amd64.tar.gz) |
| Linux ARM64 | [sushiro-overdose_4.0_linux_arm64.tar.gz](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0/sushiro-overdose_4.0_linux_arm64.tar.gz) |

Windows 控制台压缩包、macOS 通用压缩包和 `checksums.txt` 见下方附件。

## 升级前

先停止旧版后台进程，再替换应用。新版不会删除已有记录，也不会继续执行旧自动计划，但替换文件不会停止仍在运行的旧进程。自启动请使用固定安装路径。

macOS 与 Windows 可能提示签名或信誉检查，请确认下载来源后再决定是否运行，不要关闭系统整体安全检查。

本版已完成隔离自动测试、桌面/手机浏览器流程检查和跨平台编译。Windows 真实设备认证、真实取号未做本次实机验收。

[完整说明](https://github.com/Ryujoxys/sushiro-overdose#readme) · [反馈问题](https://github.com/Ryujoxys/sushiro-overdose/issues)
