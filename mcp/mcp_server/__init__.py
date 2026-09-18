"""sushiro-overdose MCP server。

装到 AI agent（Claude Desktop / Cursor），让 AI：
- 通过本机桌面端查实时排队和本机历史
- 联动桌面端（查我的预约/凭证状态/实时预测）
- 给智能到店建议
- 教用户用 sushiro

只调用桌面端 GET 接口，不提供官方取号或取消工具。
"""
