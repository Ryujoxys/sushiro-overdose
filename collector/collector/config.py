"""配置加载：环境变量 > config.json > config.example.json 占位。

真实 token / Turso 凭证绝不写进代码或上传 git。日志里一律 redact。
"""
from __future__ import annotations

import json
import logging
import os
from copy import deepcopy
from typing import Any, Dict, Optional

log = logging.getLogger("collector.config")

CONFIG_FILENAME = "config.json"
EXAMPLE_FILENAME = "config.example.json"

# 环境变量名 → 配置路径（点分）
ENV_OVERRIDES = {
    "SUSHIRO_COLLECTOR_SUSHIRO_TOKEN": ("sushiro", "token"),
    "SUSHIRO_COLLECTOR_TURSO_URL": ("turso", "url"),
    "SUSHIRO_COLLECTOR_TURSO_AUTH_TOKEN": ("turso", "auth_token"),
}

# 这些值含敏感信息，日志里要 redact
SENSITIVE_PATHS = {
    ("sushiro", "token"),
    ("turso", "auth_token"),
}

DEFAULTS = {
    "sushiro": {
        "base_url": "https://crm-cn-prd.sushiro.com.cn/wechat/api/2.0",
        "token": "",
        "referer": "https://servicewechat.com/wx7ac31ef6c073a7ed/159/page-frame.html",
        "user_agent": "Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36 MicroMessenger/8.0",
    },
    "turso": {"url": "", "auth_token": ""},
    "collect": {
        "interval_seconds": 900,
        "store_ids": [],
        "active_hours": [10, 22],
        "concurrency": 8,
        "per_call_timeout_seconds": 15,
        "list_latitude": 23.13,
        "list_longitude": 113.26,
    },
    "retention": {"records_days": 0, "runs_days": 30},
}


def _find_config_dir(start: str) -> str:
    """从 start 往上找包含 config.json 或 config.example.json 的目录。"""
    cur = os.path.abspath(start)
    for _ in range(6):
        if os.path.exists(os.path.join(cur, CONFIG_FILENAME)) or os.path.exists(
            os.path.join(cur, EXAMPLE_FILENAME)
        ):
            return cur
        parent = os.path.dirname(cur)
        if parent == cur:
            break
        cur = parent
    return os.path.abspath(start)


def _deep_merge(base: Dict[str, Any], overlay: Dict[str, Any]) -> Dict[str, Any]:
    out = deepcopy(base)
    for k, v in (overlay or {}).items():
        if isinstance(v, dict) and isinstance(out.get(k), dict):
            out[k] = _deep_merge(out[k], v)
        else:
            out[k] = v
    return out


def _set_path(d: Dict[str, Any], path: tuple, value: Any) -> None:
    cur = d
    for p in path[:-1]:
        cur = cur.setdefault(p, {})
    cur[path[-1]] = value


def load_config(config_dir: Optional[str] = None) -> Dict[str, Any]:
    """加载配置。优先级：env > config.json > config.example.json > DEFAULTS。"""
    cfg_dir = config_dir or _find_config_dir(os.getcwd())
    cfg = deepcopy(DEFAULTS)

    # config.example.json 作为基线（含注释键，加载时过滤 _ 开头的键）
    example_path = os.path.join(cfg_dir, EXAMPLE_FILENAME)
    if os.path.exists(example_path):
        with open(example_path, "r", encoding="utf-8") as f:
            example = json.load(f)
        cfg = _deep_merge(cfg, _strip_comment_keys(example))

    # config.json 覆盖（真实凭证）
    real_path = os.path.join(cfg_dir, CONFIG_FILENAME)
    if os.path.exists(real_path):
        with open(real_path, "r", encoding="utf-8") as f:
            real = json.load(f)
        cfg = _deep_merge(cfg, _strip_comment_keys(real))
    else:
        log.warning("未找到 config.json（%s），使用 example 占位 + 环境变量。", real_path)

    # 环境变量最高优先级
    for env_key, path in ENV_OVERRIDES.items():
        val = os.environ.get(env_key)
        if val:
            _set_path(cfg, path, val)

    return cfg


def _strip_comment_keys(d: Any) -> Any:
    """递归删掉 _comment 开头的键，它们只是文档不是配置。"""
    if isinstance(d, dict):
        return {k: _strip_comment_keys(v) for k, v in d.items() if not k.startswith("_")}
    if isinstance(d, list):
        return [_strip_comment_keys(x) for x in d]
    return d


def redact_for_log(cfg: Dict[str, Any]) -> Dict[str, Any]:
    """返回一份脱敏副本，用于打印日志。敏感字段替换为 <redacted>。"""
    out = deepcopy(cfg)

    def _walk(d: Any, path: tuple) -> None:
        if isinstance(d, dict):
            for k, v in list(d.items()):
                p = path + (k,)
                if isinstance(v, dict):
                    _walk(v, p)
                elif p in SENSITIVE_PATHS and v:
                    d[k] = "<redacted>"
                elif k == "url" and isinstance(v, str) and "@" in v:
                    d[k] = v.split("@")[0] + "@<redacted>"

    _walk(out, ())
    return out


def require_credential(cfg: Dict[str, Any], *path: str) -> str:
    """取必填凭证，缺失就抛异常（启动时 fail fast）。"""
    cur: Any = cfg
    for p in path:
        if not isinstance(cur, dict):
            cur = None
            break
        cur = cur.get(p)
    val = (cur or "").strip() if isinstance(cur, str) else ""
    if not val or val.startswith("PUT_"):
        raise SystemExit(
            f"缺少配置 {'.'.join(path)}：请在 collector/config.json 或环境变量中填入真实值"
        )
    return val
