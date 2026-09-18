"""sushiro-overdose MCP server（FastMCP，stdio transport）。

注册所有 tool（查数据/联动桌面端/到店建议/教学）+ resource（FAQ）。
启动时只连接本机桌面端；历史数据由桌面端本地计算。

用法：
  uv run sushiro-mcp          # stdio，配 Claude Desktop
  mcp dev mcp_server/server.py  # inspector 调试
"""
from __future__ import annotations

import logging
from contextlib import asynccontextmanager
from typing import Optional

from mcp.server.fastmcp import FastMCP
from mcp.types import ToolAnnotations

from .config import load_config
from .desktop import DesktopClient
from . import tools_queue, tools_desktop, tools_advice, resources

log = logging.getLogger("mcp")

# 模块级客户端：lifespan 建连后赋值，tool 函数直接读。MCP 是单进程，无线程安全问题。
_desktop: Optional[DesktopClient] = None


def _require_desktop() -> DesktopClient:
    if _desktop is None:
        raise RuntimeError("本机助手尚未初始化，请重新启动 MCP 客户端。")
    return _desktop


@asynccontextmanager
async def lifespan(app: FastMCP):
    """启动时建连，退出时清理。"""
    global _desktop
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
    cfg = load_config()
    _desktop = DesktopClient(cfg.desktop_port)
    log.info("sushiro MCP server 启动（local desktop=127.0.0.1:%d）", cfg.desktop_port)
    try:
        yield
    finally:
        _desktop = None


mcp = FastMCP("sushiro", lifespan=lifespan)
_READONLY = ToolAnnotations(readOnlyHint=True)


# ===== A. 查排队数据（本机历史 + 官方实时只读接口）=====

@mcp.tool(annotations=_READONLY, description="搜索门店。city 匹配城市/区域，q 模糊匹配店名/城市/区域。返回 store_id/name/city/area。")
def list_stores(city: Optional[str] = None, q: Optional[str] = None, limit: int = 20):
    """例：list_stores(city='广州') 或 list_stores(q='太阳宫')"""
    return tools_queue.list_stores(_require_desktop(), city, q, limit)


@mcp.tool(annotations=_READONLY, description="某店历史叫号曲线 + 各时段忙率/等位。date_type 可选 weekday/workday/weekend/holiday（默认 weekday）。返回'按历史几点叫到几号'。")
def store_queue_history(store_id: int, date_type: str = "weekday") -> dict:
    """例：store_queue_history(3006, 'holiday')"""
    return tools_queue.store_queue_history(_require_desktop(), store_id, date_type)


@mcp.tool(annotations=_READONLY, description="某店各时段排队压力（忙率/等位/桌数），突出'几点最挤'。date_type 默认 weekday。")
def store_pressure(store_id: int, date_type: str = "weekday") -> dict:
    return tools_queue.store_pressure(_require_desktop(), store_id, date_type)


@mcp.tool(annotations=_READONLY, description="某店实时叫号、等位和本机历史叫号速度。速率单位及样本覆盖以返回字段为准；无样本不做推测。")
def called_speed(store_id: int) -> dict:
    return tools_queue.called_speed(_require_desktop(), store_id)


@mcp.tool(annotations=_READONLY, description="多店本机历史趋势对比。store_ids 是门店 id 列表；time_bucket 为 HH:MM，查询从该时间开始的半小时，否则用默认全天窗口。返回样本与可信度，不承诺有数据。")
def compare_stores(store_ids: list, date_type: str = "weekday", time_bucket: Optional[str] = None) -> dict:
    return tools_queue.compare_stores(_require_desktop(), store_ids, date_type, time_bucket)


# ===== B. 联动桌面端（调本地 API）=====

@mcp.tool(annotations=_READONLY, description="桌面端总状态：凭证是否有效(auth_status)、引擎状态、采样。AI 先调这个判断桌面端在不在跑、凭证有没有效。")
def desktop_status() -> dict:
    return tools_desktop.desktop_status(_desktop)


@mcp.tool(annotations=_READONLY, description="我的预约 + 排队号（需凭证）。桌面端没跑或无凭证时返回友好提示。")
def my_reservations() -> dict:
    return tools_desktop.my_reservations(_desktop)


@mcp.tool(annotations=_READONLY, description="手里排队号实时状态 + 取号计划。")
def my_ticket_status() -> dict:
    return tools_desktop.my_ticket_status(_desktop)


@mcp.tool(annotations=_READONLY, description="全套环境诊断：凭证完整性/证书/代理/网络。AI 排障利器——用户遇到问题时调这个看哪里出问题。")
def diagnose() -> dict:
    return tools_desktop.diagnose(_desktop)


@mcp.tool(annotations=_READONLY, description="查可约时段（需凭证+绑定门店）。period 可选 lunch/dinner/all。")
def available_slots(store: Optional[str] = None, period: Optional[str] = None) -> dict:
    return tools_desktop.available_slots(_desktop, store, period)


# ===== C. 智能到店建议（综合实时+历史）=====

@mcp.tool(annotations=_READONLY, description="综合到店建议：实时 ETA + 历史规律。给 target_no 算'这个号几点叫到/几点出发'；给 want_meal_time(如'18:30')算'想这个点吃几点取号'。travel_minutes 是路程分钟。")
def arrival_advice(
    store_id: int,
    target_no: Optional[int] = None,
    want_meal_time: Optional[str] = None,
    travel_minutes: Optional[int] = None,
) -> dict:
    return tools_advice.arrival_advice(_require_desktop(), store_id, target_no, want_meal_time, travel_minutes)


# ===== D. 教学资源 =====

@mcp.tool(annotations=_READONLY, description="教用户用 sushiro。topic 可选: get_passport(拿通行证)/auth_expired(凭证失效)/queue_chart(叫号曲线)/eta_prediction(预测)/notification(通知)/mcp。")
def explain_usage(topic: str) -> str:
    return resources.explain_usage(topic)


@mcp.resource("docs://faq", description="sushiro 使用 FAQ：通行证/凭证失效/叫号曲线/通知/隐私")
def faq_resource() -> str:
    return resources.load_faq()


def main() -> None:
    """入口（pyproject scripts: sushiro-mcp）。"""
    mcp.run(transport="stdio")


if __name__ == "__main__":
    main()
