"""Local desktop connection only; legacy database environment variables are ignored."""
from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass
class Config:
    desktop_port: int = 39871


def load_config() -> Config:
    try:
        port = int(os.environ.get("SUSHIRO_MCP_DESKTOP_PORT", "39871"))
    except ValueError:
        port = 39871
    return Config(desktop_port=port if 1 <= port <= 65535 else 39871)
