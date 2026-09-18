"""教学资源：FAQ 作为 resource 暴露 + explain_usage tool。

resource 是 AI 主动读的上下文（docs://faq），tool 是 AI 调用的动作（explain_usage）。
两者都做，因为部分 MCP 客户端对 resource 自动注入弱，tool 更可靠。
"""
from __future__ import annotations

import os
from typing import Dict

# FAQ/explain 的主题 → 人话指引（比纯 resource 更可控、更精准）
USAGE_TOPICS: Dict[str, str] = {
    "get_passport": (
        "拿通行证（凭证）：设置页点「拿通行证」。macOS 推荐「PC 微信自动抓」（最省事）；"
        "Windows 用手机抓包导入（PC 微信抓不到小程序请求）。流程约 3 分钟。"
        "拿到后才能预约/远程取号/看我的单据。只看排队和叫号预测不需要通行证。"
    ),
    "auth_expired": (
        "凭证失效：寿司郎一账号同时只认一个活跃会话——手机用过小程序就把电脑顶失效。"
        "信号：官方返回 E010/error.server、401/403、取号/预约突然失败、手机重开过小程序。"
        "处理：设置页点「重置认证」再「重新获取通行证」即可。"
    ),
    "queue_chart": (
        "叫号曲线：「按历史几点叫到几号」来自本机采集的历史叫号聚合。"
        "绿线=典型（P50），蓝虚线=保守（慢，P20），黄虚线=偏快（P80）。"
        "曲线为空表示本机样本不足；选择常用门店并保持本机采集运行。"
    ),
    "eta_prediction": (
        "ETA 预测（几点叫到我）：把手里号码填进去，工具算大概多久叫到、几点出发。"
        "优先用实时叫号速度（最准），融合历史规律。号码靠前/叫号快时最准。"
        "数据少时给保守区间（范围大），数据多时收窄。"
    ),
    "notification": (
        "通知：设置页配通知渠道（飞书/Telegram/Bark/Server酱），快到你的号时推一条，"
        "不用一直盯着屏幕。可同时开多个。"
    ),
    "mcp": (
        "MCP 助手：装到 Claude Desktop/Cursor，AI 能帮你查排队数据、看预约状态、给到店建议。"
        "sushiro 设置页「MCP 助手」开启，无需数据库密钥；使用时保持桌面端运行。"
    ),
}


def load_faq() -> str:
    """读 docs/faq.md 内容（作为 resource 返回）。"""
    # docs/faq.md 相对于 mcp_server 包的上一级
    here = os.path.dirname(os.path.abspath(__file__))
    faq_path = os.path.join(os.path.dirname(here), "docs", "faq.md")
    try:
        with open(faq_path, "r", encoding="utf-8") as f:
            return f.read()
    except FileNotFoundError:
        return FAQ_FALLBACK


FAQ_FALLBACK = """# sushiro-overdose FAQ

通行证：只看排队不用登录；预约/取号/读单据需要通行证（设置页「拿通行证」，约3分钟）。
凭证失效：手机用小程序会顶掉电脑凭证，重新获取即可。
叫号曲线：来自本机历史数据，样本不足时不生成确定结论。
ETA：填手里号码，工具算几点叫到、几点出发。
通知：设置页配飞书/Telegram/Bark/Server酱，快叫到时提醒。
"""


def explain_usage(topic: str) -> str:
    """针对某主题返回人话指引。topic 见 USAGE_TOPICS keys。未知 topic 返回主题清单。"""
    t = (topic or "").strip().lower()
    if t in USAGE_TOPICS:
        return USAGE_TOPICS[t]
    return "未知主题。可用主题：" + "、".join(USAGE_TOPICS.keys())
