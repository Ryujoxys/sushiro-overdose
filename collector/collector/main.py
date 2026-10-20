"""CLI 入口。

用法：
  python -m collector.main init-schema        建全部表
  python -m collector.main bootstrap          冷启动：从 stores? 拉全量写 store_dimension
  python -m collector.main run-once           跑一轮采集（stores + 每店 getStoreById）
  python -m collector.main seed-holidays      初始化节假日日历
  python -m collector.main migrate-v2         v1 原始快照/聚合表 → queue_records（加 --drop 删除旧表）
  python -m collector.main run                常驻：内置循环，每 15min 采集 + 每日清理
"""
from __future__ import annotations

import argparse
import logging
import sys
from typing import List, Optional

from .config import load_config, redact_for_log, require_credential
from .turso import TursoClient

log = logging.getLogger("collector")


def _setup_logging(verbose: bool) -> None:
    logging.basicConfig(
        level=logging.DEBUG if verbose else logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )


def _turso_from_cfg(cfg) -> TursoClient:
    url = require_credential(cfg, "turso", "url")
    token = require_credential(cfg, "turso", "auth_token")
    return TursoClient(url, token)


def cmd_init_schema(args, cfg) -> None:
    from .schema import SCHEMA_STATEMENTS

    turso = _turso_from_cfg(cfg)
    log.info("建表（%d 条 DDL）…", len(SCHEMA_STATEMENTS))
    # 逐条执行，任一失败立即报错（DDL 不支持参数）
    for stmt in SCHEMA_STATEMENTS:
        turso.execute(stmt)
    rows = turso.execute(
        "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name"
    )
    tables = [r.get("name") for r in rows]
    log.info("✅ 建表完成。现有表：%s", tables)


def cmd_bootstrap(args, cfg) -> None:
    """冷启动：拉一次 stores? 写 store_dimension（不采快照）。"""
    from .sushiro_client import SushiroClient
    from . import collector as coll

    token = require_credential(cfg, "sushiro", "token")
    coll_cfg = cfg.get("collect", {})
    client = SushiroClient(
        token=token,
        base_url=cfg["sushiro"].get("base_url"),
        referer=cfg["sushiro"].get("referer"),
        user_agent=cfg["sushiro"].get("user_agent"),
        timeout=float(coll_cfg.get("per_call_timeout_seconds", 15)),
    )
    turso = _turso_from_cfg(cfg)
    stores = client.list_stores(
        float(coll_cfg.get("list_latitude", 23.13)),
        float(coll_cfg.get("list_longitude", 113.26)),
    )
    log.info("拉到 %d 家店，写入 store_dimension", len(stores))
    coll._upsert_store_dimension(turso, stores, coll._fmt_dt())
    rows = turso.execute("SELECT COUNT(*) AS c FROM store_dimension")
    log.info("✅ store_dimension 现有 %s 行", rows[0].get("c") if rows else 0)


def cmd_run_once(args, cfg) -> None:
    from .collector import collect_once

    store_filter = [int(x) for x in args.stores] if args.stores else None
    stats = collect_once(cfg, store_id_filter=store_filter, skip_store_detail=args.no_detail)
    print(stats)


def cmd_migrate_v2(args, cfg) -> None:
    from .migrate_v2 import copy_records, drop_legacy

    turso = _turso_from_cfg(cfg)
    if args.drop:
        log.info("✅ %s", drop_legacy(turso))
    else:
        log.info("✅ %s", copy_records(turso))


def cmd_seed_holidays(args, cfg) -> None:
    from .seed_holidays import seed_holidays

    seed_holidays(_turso_from_cfg(cfg), year=args.year)


def cmd_run(args, cfg) -> None:
    from .runner import run_loop

    run_loop(cfg)


COMMANDS = {
    "init-schema": cmd_init_schema,
    "bootstrap": cmd_bootstrap,
    "run-once": cmd_run_once,
    "migrate-v2": cmd_migrate_v2,
    "seed-holidays": cmd_seed_holidays,
    "run": cmd_run,
}


def main(argv: Optional[List[str]] = None) -> int:
    parser = argparse.ArgumentParser(prog="collector", description="sushiro Turso 采集器")
    parser.add_argument("command", choices=sorted(COMMANDS.keys()))
    parser.add_argument("--config", help="config.json 所在目录")
    parser.add_argument("-v", "--verbose", action="store_true")
    parser.add_argument("--stores", nargs="*", type=int, help="run-once: 只采这些店 id")
    parser.add_argument("--no-detail", action="store_true", help="run-once: 跳过 getStoreById 叫号")
    parser.add_argument("--drop", action="store_true", help="migrate-v2: 删除 v1 遗留表并 VACUUM（先备份）")
    parser.add_argument("--year", type=int, help="seed-holidays: 年份")
    args = parser.parse_args(argv)

    _setup_logging(args.verbose)
    cfg = load_config(args.config)
    log.debug("配置（已脱敏）: %s", redact_for_log(cfg))

    try:
        COMMANDS[args.command](args, cfg)
    except SystemExit:
        raise
    except Exception as e:
        log.error("命令 %s 失败: %s", args.command, e)
        if args.verbose:
            raise
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
