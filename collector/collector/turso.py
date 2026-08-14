"""Turso v2 pipeline HTTP 客户端。

移植自 Go internal/app/queue_baseline_turso.go 的协议层（libsql→https、pipeline body、
值解析），但这里是双向：既能 SELECT 也能 DDL/INSERT/UPDATE。零 SDK，纯 urllib。
"""
from __future__ import annotations

import gzip
import http.client
import json
import logging
import urllib.error
import urllib.request
from typing import Any, Dict, List, Optional, Sequence

log = logging.getLogger("collector.turso")

CLIENT_VERSION = "sushiro-collector"
HTTP_TIMEOUT = 30


class TursoError(Exception):
    pass


def pipeline_url(raw: str) -> str:
    """libsql:// → https://.../v2/pipeline。

    WHATWG URL 禁止把 libsql（非特殊协议）直接改写成 https，必须在解析前字符串替换
    （与 worker index.js tursoPipelineURL 同样的处理）。
    """
    s = (raw or "").strip()
    if s.startswith("libsql://"):
        s = "https://" + s[len("libsql://"):]
    if not (s.startswith("https://") or s.startswith("http://")):
        raise TursoError(f"unsupported Turso URL scheme: {raw}")
    # 去掉 path/query/fragment
    for sep in ("/", "?", "#"):
        idx = s.find(sep, 8)  # 跳过 https://
        if idx > 0:
            s = s[:idx]
    return s.rstrip("/") + "/v2/pipeline"


def _cell_to_py(cell: Any) -> Any:
    """Turso pipeline 的 cell 形如 {"type":"integer","value":"123"} 或 {"type":"null"}。"""
    if not isinstance(cell, dict):
        return None
    if cell.get("type") == "null":
        return None
    v = cell.get("value")
    return v


class TursoClient:
    """Turso HTTP pipeline 客户端。execute 跑单条 SQL；execute_many 批量（一个请求多 statement）。"""

    def __init__(self, url: str, auth_token: str):
        if not url or not auth_token:
            raise TursoError("TursoClient 需要 url 和 auth_token")
        self._url = pipeline_url(url)
        self._token = auth_token
        # Turso 基于 host 路由，必须显式设 Host 头（从 URL 取）
        # self._url 已是 https://host/v2/pipeline，urllib 默认 Host 正确，无需额外处理。

    def execute(
        self,
        sql: str,
        params: Optional[Sequence[Any]] = None,
        *,
        retry_safe: Optional[bool] = None,
    ) -> List[Dict[str, Any]]:
        """执行单条 SQL，返回 rows（每行 {col: value}）。DDL/INSERT 返回空 list。"""
        return self.execute_many([(sql, params)], retry_safe=retry_safe)

    def execute_many(
        self,
        statements: Sequence[tuple],
        *,
        retry_safe: Optional[bool] = None,
    ) -> List[Dict[str, Any]]:
        """批量执行。statements 是 [(sql, params), ...] 或 [(sql), ...]。

        返回最后一条语句的 rows（采集器场景：批量 INSERT 后通常不需要结果）。
        只读 SQL 默认可重试；写 SQL 默认不重试，避免服务端已提交但客户端丢响应时重复写。
        调用方确认 SQL 幂等后可显式传 retry_safe=True。
        """
        requests_body = []
        sql_texts = []
        for stmt in statements:
            if isinstance(stmt, str):
                sql, params = stmt, None
            else:
                sql, params = stmt[0], (stmt[1] if len(stmt) > 1 else None)
            sql_texts.append(sql)
            entry: Dict[str, Any] = {
                "type": "execute",
                "stmt": {"sql": sql, "want_rows": True},
            }
            if params:
                entry["stmt"]["args"] = _encode_args(params)
            requests_body.append(entry)

        body = json.dumps({"requests": requests_body}).encode("utf-8")
        if retry_safe is None:
            retry_safe = all(_is_read_only_sql(sql) for sql in sql_texts)
        attempts = 5 if retry_safe else 1

        # Turso 偶发 SSL EOF / 连接重置，指数退避重试缓解（瞬态错误）
        import time as _time

        last_exc = None
        raw = None
        for attempt in range(attempts):
            req = urllib.request.Request(
                self._url,
                data=body,
                method="POST",
                headers={
                    "authorization": f"Bearer {self._token}",
                    "content-type": "application/json",
                    "accept-encoding": "gzip",
                    "x-libsql-client-version": CLIENT_VERSION,
                    "connection": "close",  # 避免复用坏连接
                },
            )
            try:
                with urllib.request.urlopen(req, timeout=HTTP_TIMEOUT) as resp:
                    raw_bytes = resp.read()
                    if (resp.headers.get("content-encoding") or "").lower() == "gzip":
                        raw_bytes = gzip.decompress(raw_bytes)
                    raw = raw_bytes.decode("utf-8")
                break
            except urllib.error.HTTPError as e:
                # 5xx 可能是瞬态限流，重试；4xx 是业务错误（含 401），不重试
                if retry_safe and 500 <= e.code < 600 and attempt < attempts - 1:
                    detail = e.read().decode("utf-8", "replace")[:200]
                    last_exc = TursoError(f"Turso HTTP {e.code}: {detail}")
                    log.debug("Turso %d（attempt %d，重试）", e.code, attempt + 1)
                    _time.sleep(0.5 * (2 ** attempt))
                    continue
                detail = e.read().decode("utf-8", "replace")[:400]
                raise TursoError(f"Turso HTTP {e.code}: {detail}") from None
            except (
                urllib.error.URLError,
                http.client.IncompleteRead,
                http.client.RemoteDisconnected,
                OSError,
            ) as e:
                last_exc = e
                wait = 0.5 * (2 ** attempt)  # 0.5, 1, 2, 4, 8 秒
                if not retry_safe:
                    raise TursoError(
                        f"Turso 写请求响应失败，提交状态未知且未自动重试: {e}"
                    ) from e
                log.debug("Turso 网络错误（attempt %d，%0.1fs 后重试）: %s", attempt + 1, wait, e)
                if attempt < attempts - 1:
                    _time.sleep(wait)
                continue
        if raw is None:
            raise TursoError(
                f"Turso 网络错误（重试 {attempts} 次仍失败）: {last_exc}"
            ) from last_exc

        try:
            data = json.loads(raw)
        except json.JSONDecodeError as e:
            raise TursoError(f"Turso 响应非 JSON: {raw[:200]}") from e

        results = data.get("results") or []
        rows_out: List[Dict[str, Any]] = []
        last_error = None
        for i, r in enumerate(results):
            if r.get("type") == "error":
                err = r.get("error", {})
                last_error = f"stmt#{i}: {err.get('code', 'TURSO_ERROR')}: {err.get('message', '')}"
                break
            resp_obj = r.get("response", {})
            res = resp_obj.get("result") or {}
            cols = [c.get("name", "") for c in (res.get("cols") or [])]
            rows = []
            for row in (res.get("rows") or []):
                cells = [_cell_to_py(c) for c in row]
                rows.append(dict(zip(cols, cells)))
            rows_out = rows
        if last_error:
            raise TursoError(last_error)
        return rows_out


def _is_read_only_sql(sql: str) -> bool:
    head = (sql or "").lstrip().upper()
    return head.startswith(("SELECT", "PRAGMA", "EXPLAIN"))


def _encode_args(params: Sequence[Any]) -> List[Dict[str, Any]]:
    """把 Python 值编码成 Turso pipeline 的 positional args。

    Turso 协议的别扭之处（实测确认）：
    - integer 的 value 必须是字符串 "10"（给数字会报 expected a borrowed string）
    - float 的 value 必须是 JSON 数字 23.12（给字符串会报 expected f64）
    - null 的 value 用 None；text 的 value 用字符串
    """
    out = []
    for p in params:
        if p is None:
            out.append({"type": "null", "value": None})
        elif isinstance(p, bool):
            out.append({"type": "integer", "value": "1" if p else "0"})
        elif isinstance(p, int):
            out.append({"type": "integer", "value": str(p)})
        elif isinstance(p, float):
            out.append({"type": "float", "value": p})
        else:
            out.append({"type": "text", "value": str(p)})
    return out


# 查询结果值规范化（SELECT 出来都是字符串，转回 Python 类型）
def as_int(v: Any, default: int = 0) -> int:
    if v is None:
        return default
    try:
        return int(float(v))
    except (TypeError, ValueError):
        return default


def as_float(v: Any) -> Optional[float]:
    if v is None or v == "":
        return None
    try:
        return float(v)
    except (TypeError, ValueError):
        return None


def as_str(v: Any, default: str = "") -> str:
    if v is None:
        return default
    return str(v)
