# 寿司郎排队记录

**Sushiro Overdose v4.0 · lite正式版**

记录常去门店的排队数据，看看什么时间通常叫到几号、要等多久。想去吃饭时，再手动取号。

[下载 v4.0](https://github.com/Ryujoxys/sushiro-overdose/releases/tag/v4.0) · [这次改了什么](docs/release-notes-4.0.md) · [使用与安全](SECURITY.md) · [反馈问题](https://github.com/Ryujoxys/sushiro-overdose/issues)

支持 macOS、Windows、Linux。下载即可运行，不用部署网站，也不用安装 Node.js 或数据库。

## 先记录，不用登录

1. 打开应用，在「我的记录」选择常去的门店。
2. 点击「开始记录」，默认每 5 分钟保存一次公开排队数据。
3. 按门店、日期和日期类型查看曲线，逐渐积累自己的记录。

**只记录数据，不需要账号、证书或代理。** 选店不会自动启动服务，是否记录由你决定。

可以单独开启「登录电脑后自动记录」。后台服务启动后，关闭界面仍会继续；电脑关机、退出登录或休眠时暂停。暂停不会删除已有数据。

## 看懂排队规律

| 想知道什么 | 怎么看 |
|------|------|
| 这个时间一般叫到几号？ | 切到「叫到几号」，点击曲线或拖动时间，查看通常叫到的号码、常见范围和样本天数 |
| 通常需要等多久？ | 切到「等多久」，看各时段的等待分布 |
| 周末和工作日有什么不同？ | 切换日期类型，也可以只看某一天 |
| 只想分析自己采的数据？ | 关闭「包含历史数据」，重启后仍保留选择 |
| 想自己继续分析？ | 导出自己的原始 JSONL 记录 |

首次打开也有曲线可看：程序内置一份 **截至 2026-09-06 的固定历史包**，覆盖 138 家门店。它随程序分发，不会自动联网更新，也不计入你自己的记录数量和个人模型。

图表会区分本机与历史样本。同店、同日、同半小时已有本机记录时，优先使用本机数据，不重复计数。缺失时段不会当成零，样本不足会直接注明。

历史叫号采用堂食队列的最大号码，不区分桌型。它是过去的参考，不是今天的实时叫号，也不保证某张排队号的入座时间。[历史包的范围与限制](docs/bundled-history.md)

## 需要时，手动取号

进入「手动取号」，选店、核对人数和桌型，必要时连接本人账号，然后确认提交。

认证完成后会回到确认框，**不会直接取号**。已有有效号码时优先展示；提交结果不明确时先查询，不自动重试。

「想几点吃」只给时间建议。确认后取的是现在的排队号，不是预约未来的入座时间；建议只参考本机数据，样本不足时会说明。

v4.0 不再提供狙击、循环抢号、定时取号或每日自动计划。

### 账号怎么连接

仅手动取号需要寿司郎账号。凭证从你本人的小程序请求中获取，保存在本机。

- macOS 默认使用电脑微信；Windows、Linux 默认使用手机微信，也支持手动导入凭证。
- 电脑连接会安装本机证书并临时设置代理，完成或停止后恢复代理。
- 手机与电脑需在同一 Wi-Fi，按引导安装证书和设置手机代理，结束后关闭手机 Wi-Fi 代理。
- 代理只解密寿司郎接口，其他 HTTPS 连接透传。

这不是免配置扫码登录。微信版本、证书策略、防火墙和路由器隔离都可能影响认证，凭证也会过期。**只想记录数据，可以完全跳过这一步。**

## 下载与升级

| 系统 | 推荐下载 | 使用方式 |
|------|------|------|
| Windows 常见电脑 | [windows-amd64.exe](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0/Sushiro-Overdose-4.0-windows-amd64.exe) | 双击运行 |
| Windows ARM | [windows-arm64.exe](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0/Sushiro-Overdose-4.0-windows-arm64.exe) | 双击运行 |
| macOS Intel / Apple 芯片 | [macOS.dmg](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0/Sushiro-Overdose-4.0-macOS.dmg) | 拖入 Applications 后打开 |
| Linux x86-64 | [linux_amd64.tar.gz](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0/sushiro-overdose_4.0_linux_amd64.tar.gz) | 解压后运行 `./sushiro` |
| Linux ARM64 | [linux_arm64.tar.gz](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0/sushiro-overdose_4.0_linux_arm64.tar.gz) | 解压后运行 `./sushiro` |

更多压缩包及 SHA-256 校验值见[发布页](https://github.com/Ryujoxys/sushiro-overdose/releases/tag/v4.0)。

macOS 和 Windows 可能提示应用未签名或来源未知。请先确认下载来源；macOS 可在「系统设置 → 隐私与安全性」中允许打开。不要关闭系统整体安全检查。

从旧版升级前，先停止旧版后台进程，再替换应用。新版保留本机记录，不执行旧自动计划；替换文件不会停止已经运行的旧版进程。开启自启动前，把应用放在固定位置。

本地界面默认位于 `127.0.0.1:39871`，端口占用时自动换端口。

## 数据留在自己电脑上

自己的记录、配置和模型保存在 `~/.sushiro/`，Windows 对应 `%USERPROFILE%\.sushiro\`。

- 记录服务只访问寿司郎公开接口，不读取个人凭证，不创建或取消号码。
- 不登录 GitHub，不连接共享数据库，不上传自己的排队记录。
- 内置历史包和自己的原始数据分开保存；导出不含历史包、凭证或个人号码。
- 手动取号只在你确认后访问寿司郎官方接口。
- 原始公开快照定期保留最近 10 万条，长期记录建议定期导出备份。

数据字段遵循[统一本地数据协议](docs/local-data-contract.md)。网络请求、凭证与证书处理详见[安全说明](SECURITY.md)。

## 命令行与开发

Go 1.23+，Go 端仅使用标准库。页面与历史包内嵌在程序里，无需前端构建。

```bash
git clone https://github.com/Ryujoxys/sushiro-overdose.git
cd sushiro-overdose
go build -o sushiro .
./sushiro
```

常用命令：

```bash
sushiro                       # 打开界面
sushiro collect status        # 查看记录状态
sushiro collect start         # 后台记录，需先选门店
sushiro collect stop          # 停止记录，保留数据
sushiro collect autostart on  # 开启登录自启动
sushiro collect autostart off # 关闭自启动
sushiro doctor                # 只读诊断
sushiro repair-proxy          # 恢复残留系统代理
sushiro version               # 查看版本和版本名称
```

开发与验证见 [CONTRIBUTING.md](CONTRIBUTING.md)，包职责见 [ARCHITECTURE.md](ARCHITECTURE.md)。MCP 是[可选模块](mcp/README.md)，普通使用不需要安装。

开发预览使用绝对路径 `SUSHIRO_DATA_HOME` 隔离文件，**不要改写 `HOME` 或 `USERPROFILE`**。它不是系统沙箱，主动认证仍可能安装证书、设置代理。检查工作流只手动触发，发布不执行全量测试，测试也不会发送真实系统通知。

## Star 趋势

<a href="https://www.star-history.com/?repos=Ryujoxys%2Fsushiro-overdose&type=date&legend=top-left">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=Ryujoxys/sushiro-overdose&type=date&theme=dark&legend=top-left">
    <img alt="Sushiro Overdose 的 GitHub Star 趋势" src="https://api.star-history.com/chart?repos=Ryujoxys/sushiro-overdose&type=date&legend=top-left">
  </picture>
</a>

图表由 [Star History](https://www.star-history.com/) 提供，仅在查看 README 时加载，不属于应用的数据采集。

## 许可与说明

[MIT](LICENSE)。本项目为非官方工具，与寿司郎官方无隶属关系；实时状态、取号结果和入座时间以官方小程序及门店为准。
