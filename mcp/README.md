# 本机排队助手（可选）

这是 Sushiro Overdose 的可选 MCP 模块，供 Claude Desktop、Cursor 等客户端查询排队、读取本机历史、获得到店建议。日常记录和手动取号不需要安装它。

不登录 GitHub，不直连数据库，不需要数据库 token。本机历史不足时会说明，不用内置历史包补齐个人预测。

## 前提

- Python 3.9 或以上版本。
- sushiro 桌面端正在运行；默认本机端口为 `39871`，端口被占用时请使用桌面端实际端口。
- 实时门店查询仍由桌面端访问寿司郎公开接口；读个人单据、查预约时段仍需要寿司郎凭证。

## 工具

| 工具 | 数据来源 |
|---|---|
| `list_stores` | 本机接口转发官方公开门店查询 |
| `store_queue_history` | 本机历史曲线与画像 |
| `store_pressure` | 本机历史趋势 |
| `called_speed` | 官方实时状态和本机速度统计 |
| `compare_stores` | 本机多店趋势，可选半小时时段 |
| `arrival_advice` | 桌面端预测，保留样本和可信度 |
| `desktop_status` | 桌面端状态 |
| `my_reservations` | 本机接口查询个人单据 |
| `my_ticket_status` | 本机接口查询当前取号状态 |
| `diagnose` | 桌面端诊断 |
| `available_slots` | 官方可预约时段 |
| `explain_usage` | 本地使用说明 |

MCP 只发送 GET，不提供取号、预约或取消工具。v4.0 已移除自动计划；查询可能更新本机状态缓存，但不会创建或取消号码。

## 启用

lite正式版不在主界面展示 MCP 设置。需要使用时，按下面的方式手动安装；首次安装依赖需要联网。

手动安装示例：

```bash
cd mcp
python3 -m venv venv
venv/bin/python -m pip install -e .
```

Claude Desktop 或 Cursor 配置示例，路径应换成实际安装位置；Windows 使用 `venv/Scripts/python.exe`：

```json
{
  "mcpServers": {
    "sushiro": {
      "command": "/绝对路径/mcp/venv/bin/python",
      "args": ["-m", "mcp_server.server"],
      "env": {"SUSHIRO_MCP_DESKTOP_PORT": "39871"}
    }
  }
}
```

旧数据库环境变量与 `.env` 不再读取。可从客户端配置中移除旧 URL/token；已发出的 token 还需要在服务端撤销，仅删本机配置不能使其失效。

## 测试

无需安装依赖或启动桌面服务即可运行接口适配测试：

```bash
cd mcp
python3 -m unittest discover -s tests -v
```

测试使用模拟桌面客户端，不等同于实际 MCP 握手、安装或桌面联调验收。
