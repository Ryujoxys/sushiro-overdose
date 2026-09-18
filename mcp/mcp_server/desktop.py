"""桌面端联动客户端（httpx，调 127.0.0.1:39871 只读 GET）。

优雅降级：桌面端没跑/超时 → 返回 {"ok": False, "hint": "..."}，不抛异常（AI 能友好告知用户）。
只调 GET，绝不调写接口。
"""
from __future__ import annotations

import logging
from typing import Any, Dict, Optional

import httpx

log = logging.getLogger("mcp.desktop")

CONNECT_TIMEOUT = 3.0
READ_TIMEOUT = 5.0


class DesktopClient:
    def __init__(self, port: int = 39871):
        self._base = f"http://127.0.0.1:{port}"
        self._port = port

    def get(self, path: str, params: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """GET 桌面端 API。失败返回 {"ok": False, "hint": ...}。"""
        url = f"{self._base}{path}"
        try:
            with httpx.Client(timeout=httpx.Timeout(READ_TIMEOUT, connect=CONNECT_TIMEOUT), trust_env=False) as client:
                # Host 头用 127.0.0.1 通过桌面端白名单（防 DNS 重绑定）
                resp = client.get(url, params=params, headers={"Host": f"127.0.0.1:{self._port}"})
            if resp.status_code != 200:
                return {"ok": False, "hint": f"桌面端返回 HTTP {resp.status_code}", "path": path}
            data = resp.json()
            if isinstance(data, dict):
                data.setdefault("ok", True)
            return data if data is not None else {"ok": True}
        except httpx.ConnectError:
            return {"ok": False, "hint": "桌面端没在运行，请先启动 sushiro（双击运行）", "path": path}
        except httpx.TimeoutException:
            return {"ok": False, "hint": "桌面端响应超时，可能正忙", "path": path}
        except Exception as e:
            log.warning("桌面端调用失败 %s: %s", path, e)
            return {"ok": False, "hint": f"调用桌面端失败: {e}", "path": path}

    @property
    def base(self) -> str:
        return self._base
