"""归档：删除超过保留期的原始 queue_snapshots（已聚合进 rollups）。

原始快照会持续增长（每 15min × 125 店 ≈ 每天约 6000 行），必须定期裁剪。
保留期默认 60 天，足够聚合统计 + 回溯近期异常。更老的靠 daily_store_bucket_rollups。
"""
from __future__ import annotations

import logging
from typing import Any, Dict

from .collector import _fmt_dt
from .turso import TursoClient, as_int

log = logging.getLogger("collector.archive")


def archive_old(turso: TursoClient, retention_days: int = 60) -> Dict[str, int]:
    """删除 retention_days 天前的 queue_snapshots。返回 {deleted, remaining}。"""
    # 找最新快照时间，往前推 retention_days 作为裁剪线
    rows = turso.execute("SELECT MAX(collected_at) AS mx FROM queue_snapshots")
    mx = rows[0].get("mx") if rows else None
    if not mx:
        log.info("无快照可归档")
        return {"deleted": 0, "remaining": 0}

    from datetime import timedelta
    from .datetype import parse_iso_cst

    cutoff_dt = parse_iso_cst(mx) - timedelta(days=retention_days)
    cutoff = cutoff_dt.strftime("%Y-%m-%dT%H:%M:%S+08:00")

    # 先走 collected_at 索引看最早一帧；保留期内无旧数据时不做全表 COUNT。
    min_rows = turso.execute(
        "SELECT MIN(collected_at) AS mn FROM queue_snapshots"
    )
    mn = min_rows[0].get("mn") if min_rows else None
    to_delete = 0
    if mn and mn < cutoff:
        cnt_rows = turso.execute(
            "SELECT COUNT(*) AS c FROM queue_snapshots WHERE collected_at < ?",
            (cutoff,),
        )
        to_delete = as_int(cnt_rows[0].get("c")) if cnt_rows else 0

    if to_delete:
        turso.execute(
            "DELETE FROM queue_snapshots WHERE collected_at < ?",
            (cutoff,),
            retry_safe=True,
        )
    # 精确剩余量只用于日志，却需要扫描整张原始表；-1 表示本轮主动跳过该昂贵计数。
    remaining = -1

    log.info("归档：删除 %d 行（< %s），剩余量未全表计数", to_delete, cutoff)

    # 写归档日志
    _record_archive(turso, to_delete, remaining, retention_days, cutoff)
    return {"deleted": to_delete, "remaining": remaining}


def _record_archive(
    turso: TursoClient, deleted: int, remaining: int, retention: int, cutoff: str
) -> None:
    now = _fmt_dt()
    sql = """
    INSERT INTO archive_runs
      (archive_date, snapshots, rollups_written, global_rollups,
       raw_deleted, raw_remaining, retention_days, prune_before, ok,
       error_message, created_at, updated_at)
    SELECT ?, ?, 0, 0, ?, ?, ?, ?, 1, '', ?, ?
    WHERE NOT EXISTS (
      SELECT 1 FROM archive_runs WHERE archive_date = ?
    )
    """
    from datetime import datetime
    try:
        turso.execute(
            sql,
            (
                now[:10], deleted, deleted, remaining, retention, cutoff,
                now, now, now[:10],
            ),
            retry_safe=True,
        )
    except Exception as e:
        log.warning("写 archive_runs 失败（不影响归档）: %s", e)
