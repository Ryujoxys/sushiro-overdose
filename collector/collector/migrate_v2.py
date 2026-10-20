"""v1 → v2 迁移：原始快照 + 逐日聚合 → queue_records（每店每半小时一条）。

两步，均可重复执行：
  copy：建 queue_records，先填 daily_store_bucket_rollups 的半小时代表值（覆盖原始快照已裁剪的日期），
        再用原始快照里每个半小时的最后一帧覆盖（同一时刻详情帧优先），然后核对行数。
  drop：重建 store_latest 为 v2 列，删除 v1 遗留表并 VACUUM。不可逆，执行前先完整备份。
迁移全部在服务端执行（INSERT … SELECT），不经本机传输数据。
"""
from __future__ import annotations

import logging
from datetime import date, timedelta
from typing import Dict

from .schema import LEGACY_TABLES, SCHEMA_STATEMENTS
from .turso import TursoClient, as_int

log = logging.getLogger("collector.migrate_v2")

SLOT = "substr({c},1,14)||CASE WHEN CAST(substr({c},15,2) AS INTEGER)<30 THEN '00' ELSE '30' END"

FROM_DAILY = """
INSERT OR IGNORE INTO queue_records
  (slot, store_id, collected_at, store_status, online_open, wait_minutes,
   group_queues_count, display_called_no)
SELECT snapshot_date||'T'||time_bucket, store_id, updated_at,
  CASE WHEN open_count=1 THEN 'OPEN' ELSE 'CLOSED' END,
  online_open_count,
  CASE WHEN wait_typical_minutes>=0 THEN CAST(wait_typical_minutes AS INTEGER) END,
  CASE WHEN queue_groups_typical>=0 THEN CAST(queue_groups_typical AS INTEGER) END,
  CASE WHEN called_sample_count=1 AND called_no_typical>0 THEN CAST(called_no_typical AS INTEGER) END
FROM daily_store_bucket_rollups
WHERE snapshot_date>=? AND snapshot_date<? AND sample_count=1
"""

# 与导出脚本、桌面端内置历史包相同的代表值：半小时内最后一帧，同一时刻详情帧优先。
FROM_RAW = """
WITH ranked AS (
  SELECT store_id, collected_at, store_status, online_open, wait_minutes,
    group_queues_count, display_called_no,
    ROW_NUMBER() OVER (
      PARTITION BY store_id, substr(collected_at,1,13), CAST(substr(collected_at,15,2) AS INTEGER)/30
      ORDER BY collected_at DESC, CASE dq_source WHEN 'store_detail' THEN 1 ELSE 0 END DESC, id DESC
    ) AS rk
  FROM queue_snapshots WHERE collected_at>=? AND collected_at<?
)
INSERT INTO queue_records
  (slot, store_id, collected_at, store_status, online_open, wait_minutes,
   group_queues_count, display_called_no)
SELECT """ + SLOT.format(c="collected_at") + """, store_id, collected_at,
  UPPER(COALESCE(NULLIF(store_status,''),'UNKNOWN')), online_open,
  CASE WHEN wait_minutes>=0 THEN wait_minutes END,
  CASE WHEN group_queues_count>=0 THEN group_queues_count END,
  CASE WHEN display_called_no>0 THEN display_called_no END
FROM ranked WHERE rk=1
ON CONFLICT(slot, store_id) DO UPDATE SET
  collected_at=excluded.collected_at, store_status=excluded.store_status,
  online_open=excluded.online_open, wait_minutes=excluded.wait_minutes,
  group_queues_count=excluded.group_queues_count, display_called_no=excluded.display_called_no
WHERE excluded.collected_at >= queue_records.collected_at
"""

# 删表前的覆盖核对：每个旧半小时都要有对应记录，且旧表里没有比记录更新的帧（迁移后旧采集器仍在写）。
MISSING_RAW = """
SELECT COUNT(*) AS n FROM queue_snapshots s
LEFT JOIN queue_records q ON q.store_id = s.store_id AND q.slot = """ + SLOT.format(c="s.collected_at") + """
WHERE q.slot IS NULL OR q.collected_at < s.collected_at
"""
MISSING_DAILY = """
SELECT COUNT(*) AS n FROM daily_store_bucket_rollups d
LEFT JOIN queue_records q ON q.store_id = d.store_id AND q.slot = d.snapshot_date||'T'||d.time_bucket
WHERE d.sample_count = 1 AND q.slot IS NULL
"""


# collector_runs 的时间是 '+08:00' 文本，用同样格式比较。
RUNS_PRUNE = "DELETE FROM collector_runs WHERE started_at < strftime('%Y-%m-%dT%H:%M:%S+08:00','now','+8 hours',?)"


def prune_runs(turso: TursoClient, keep_days: int = 30) -> None:
    turso.execute(RUNS_PRUNE, (f"-{int(keep_days)} days",), retry_safe=True)


def _tables(turso: TursoClient) -> set:
    return {r.get("name") for r in turso.execute("SELECT name FROM sqlite_master WHERE type='table'")}


def _weeks(first: str, last: str):
    day = date.fromisoformat(first)
    end = date.fromisoformat(last)
    while day <= end:
        nxt = day + timedelta(days=7)
        yield day.isoformat(), nxt.isoformat()
        day = nxt


def copy_records(turso: TursoClient) -> Dict[str, int]:
    for stmt in SCHEMA_STATEMENTS:
        if "queue_records" in stmt:
            turso.execute(stmt)
    tables = _tables(turso)
    stats = {"daily_rows": 0, "raw_slots": 0}
    if "daily_store_bucket_rollups" in tables:
        b = turso.execute("SELECT MIN(snapshot_date) AS a, MAX(snapshot_date) AS b, COUNT(*) AS n, SUM(sample_count=1) AS one FROM daily_store_bucket_rollups")[0]
        if b.get("a"):
            if as_int(b.get("n")) != as_int(b.get("one")):
                raise RuntimeError("daily_store_bucket_rollups 含多帧聚合行，不能当半小时代表值迁移")
            for start, end in _weeks(b["a"], b["b"]):
                turso.execute(FROM_DAILY, (start, end), retry_safe=True)
                log.info("逐日代表值 %s..%s 已迁移", start, end)
            stats["daily_rows"] = as_int(b.get("n"))
    if "queue_snapshots" in tables:
        b = turso.execute("SELECT MIN(collected_at) AS a, MAX(collected_at) AS b FROM queue_snapshots")[0]
        if b.get("a"):
            for start, end in _weeks(b["a"][:10], b["b"][:10]):
                turso.execute(FROM_RAW, (start + "T00:00:00+08:00", end + "T00:00:00+08:00"), retry_safe=True)
                log.info("原始快照 %s..%s 已迁移", start, end)
            stats["raw_slots"] = as_int(turso.execute(
                "SELECT COUNT(*) AS n FROM (SELECT 1 FROM queue_snapshots GROUP BY store_id, substr(collected_at,1,13), CAST(substr(collected_at,15,2) AS INTEGER)/30)"
            )[0].get("n"))
    stats["records"] = as_int(turso.execute("SELECT COUNT(*) AS n FROM queue_records")[0].get("n"))
    log.info("queue_records 迁移完成：%s", stats)
    return stats


def drop_legacy(turso: TursoClient) -> Dict[str, int]:
    tables = _tables(turso)
    if "queue_records" not in tables or not as_int(turso.execute("SELECT COUNT(*) AS n FROM queue_records")[0].get("n")):
        raise RuntimeError("queue_records 为空，先执行 copy")
    for name, sql in (("queue_snapshots", MISSING_RAW), ("daily_store_bucket_rollups", MISSING_DAILY)):
        if name in tables:
            missing = as_int(turso.execute(sql)[0].get("n"))
            if missing:
                raise RuntimeError(f"{name} 有 {missing} 帧没有迁移到 queue_records（或迁移后又写入了新帧），先停旧采集器并重新执行 copy")
    cols = {r.get("name") for r in turso.execute("SELECT name FROM pragma_table_info('store_latest')")}
    if "group_queues_json" in cols:
        turso.execute_many([
            ("BEGIN",),
            ("DROP TABLE IF EXISTS store_latest_v2",),
            ([s for s in SCHEMA_STATEMENTS if "store_latest" in s][0].replace("store_latest", "store_latest_v2"),),
            ("""INSERT INTO store_latest_v2 SELECT store_id, collected_at,
                 UPPER(COALESCE(NULLIF(store_status,''),'UNKNOWN')), net_ticket_status, online_open,
                 CASE WHEN wait_minutes>=0 THEN wait_minutes END,
                 CASE WHEN group_queues_count>=0 THEN group_queues_count END,
                 CASE WHEN display_called_no>0 THEN display_called_no END
               FROM store_latest""",),
            ("DROP TABLE store_latest",),
            ("ALTER TABLE store_latest_v2 RENAME TO store_latest",),
            ("COMMIT",),
        ])
    dropped = 0
    for name in LEGACY_TABLES:
        if name in tables:
            turso.execute(f"DROP TABLE {name}")
            dropped += 1
            log.info("已删除 %s", name)
    prune_runs(turso)
    try:
        turso.execute("VACUUM")
    except Exception as e:  # 托管库可能不允许 VACUUM；空闲页会被后续写入复用
        log.warning("VACUUM 未执行：%s", e)
    return {"dropped": dropped}
