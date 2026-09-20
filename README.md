<div align="center">

<img src="assets/icons/sushiro-1024.png" width="88" height="88" alt="Sushiro Overdose 应用图标">

# 寿司郎排队记录

**常去的店，几点通常叫到几号？**

记下每天的排队情况，慢慢看出规律。数据留在自己电脑上。

[![最新版本](https://img.shields.io/github/v/release/Ryujoxys/sushiro-overdose?style=flat-square&color=B81C22&label=lite)](https://github.com/Ryujoxys/sushiro-overdose/releases/latest) [![支持平台](https://img.shields.io/badge/macOS%20%2F%20Windows-桌面版-555555?style=flat-square)](#下载) [![MIT 许可](https://img.shields.io/badge/license-MIT-777777?style=flat-square)](LICENSE)

[**下载最新版**](https://github.com/Ryujoxys/sushiro-overdose/releases/latest) · [使用方式](#怎么用) · [反馈问题](https://github.com/Ryujoxys/sushiro-overdose/issues)

<br>

<img src="docs/images/records.png" width="100%" alt="排队规律：查看不同时间通常叫到的号码、等待时长和历史范围">

<sub>点一个时段，看通常叫到几号、常见等待多久。历史规律不是今天的实时叫号。</sub>

</div>

## 下载

| Windows | macOS |
| :--- | :--- |
| **[下载 64 位 EXE](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0.1/Sushiro-Overdose-4.0.1-windows-amd64.exe)** | **[下载 DMG](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0.1/Sushiro-Overdose-4.0.1-macOS.dmg)** |
| Windows 10 / 11，下载后双击 | macOS 12+，拖入「应用程序」 |
| [ARM64 版本](https://github.com/Ryujoxys/sushiro-overdose/releases/download/v4.0.1/Sushiro-Overdose-4.0.1-windows-arm64.exe) | 同一个安装包支持 Intel / Apple 芯片 |

Linux、命令行版和文件校验值见[完整下载列表](https://github.com/Ryujoxys/sushiro-overdose/releases/latest)。不用部署网站。

## 怎么用

<table>
<tr>
<td width="50%" valign="top">

### 平时，记下排队情况

选常去的门店 → 开始记录 → 按日期看曲线。

**不用登录，也不用装证书。** 后台记录启动后，关窗口也能继续；登录自启动单独开启，电脑休眠时暂停。

</td>
<td width="50%" valign="top">

### 要去吃，再手动取号

选店、填人数 → 连接本人账号 → 核对后取号。

和记录分开用。「想几点吃」只是时间参考，能否入座以门店实际叫号为准。

</td>
</tr>
</table>

<img src="docs/images/recording.png" width="100%" alt="本地记录：选择常去的门店，设置记录间隔，按需开启后台与自启动">

**刚打开也有曲线可看。** 内置历史截至 **2026-09-06**，随应用附带，不会联网更新。取消「包含历史数据」，就只看自己的记录；导出也只包含自己的数据。

<details>
<summary><strong>安装、升级和账号连接</strong></summary>

- Windows 缺少 WebView2 时会引导安装。macOS 若提示来源未知，请先核对下载来源，再到「隐私与安全性」允许打开。
- 升级前关闭旧窗口、停止旧后台，再替换程序，本机记录会保留。
- 只有取号需要认证。macOS 可用电脑微信；Windows 按引导使用手机微信，安卓需导入本人抓包凭证。凭证只保存在本机。[认证与安全说明](SECURITY.md)

</details>

<details>
<summary><strong>数据与开发文档</strong></summary>

[数据与历史包](docs/bundled-history.md) · [开发与构建](CONTRIBUTING.md) · [代码架构](ARCHITECTURE.md) · [更新说明](docs/release-notes-4.0.1.md)

应用需要联网读取门店公开排队数据，不连接共享数据库，也不上传个人记录。

</details>

## Star 趋势

<a href="https://www.star-history.com/?repos=Ryujoxys%2Fsushiro-overdose&amp;type=date">
  <img src="https://api.star-history.com/chart?repos=Ryujoxys/sushiro-overdose&amp;type=date" width="100%" alt="Sushiro Overdose 的 GitHub Star 趋势">
</a>

---

<div align="center">
<sub>Sushiro Overdose · lite正式版 · <a href="LICENSE">MIT</a><br>非官方工具，与寿司郎官方无隶属关系。</sub>
</div>
