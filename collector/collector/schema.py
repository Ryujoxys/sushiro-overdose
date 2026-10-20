"""Turso 库 schema（v2，精简版）。

只存一份事实表：每店每半小时一条公开状态（queue_records），与桌面端内置历史包、
data 分支公开数据同一粒度和语义。规律（P50/P80 等）由读取方按需计算，库里不再存聚合表。

- slot：门店时区（UTC+8）半小时起点，'YYYY-MM-DDTHH:MM'。同一半小时内后采到的一帧覆盖前一帧。
- 缺失或未知一律 NULL，不写 0 冒充：wait_minutes 为接口原值（读取方把超过 180 分钟的
  官方封顶外数值当作未知），display_called_no 只存大于 0 的堂食叫号。
- 日期类型（工作日/周末/节假日）不入库，读取时由 holiday_calendar + 周末窗口规则推出。
"""
from __future__ import annotations

from typing import List

SCHEMA_VERSION = 2

# 按顺序执行，全部幂等。
SCHEMA_STATEMENTS: List[str] = [
    # 1. 门店维度
    """
    CREATE TABLE IF NOT EXISTS store_dimension (
      store_id           INTEGER PRIMARY KEY,
      name               TEXT NOT NULL,
      city               TEXT, area TEXT, address TEXT,
      latitude           REAL, longitude REAL,
      open_date          TEXT,
      tables_capacity    INTEGER, counters_capacity INTEGER,
      first_seen_at      TEXT, last_seen_at TEXT,
      is_active          INTEGER NOT NULL DEFAULT 1
    )
    """,
    # 2. 每店最新一帧（实时总览）
    """
    CREATE TABLE IF NOT EXISTS store_latest (
      store_id           INTEGER PRIMARY KEY,
      collected_at       TEXT NOT NULL,
      store_status       TEXT NOT NULL,
      net_ticket_status  TEXT,
      online_open        INTEGER,
      wait_minutes       INTEGER,
      group_queues_count INTEGER,
      display_called_no  INTEGER
    )
    """,
    # 3. 半小时公开状态（唯一事实表）。slot 在前：按时间追加写入、按时间范围读取都走主键。
    """
    CREATE TABLE IF NOT EXISTS queue_records (
      slot               TEXT NOT NULL,
      store_id           INTEGER NOT NULL,
      collected_at       TEXT NOT NULL,
      store_status       TEXT NOT NULL,
      online_open        INTEGER,
      wait_minutes       INTEGER,
      group_queues_count INTEGER,
      display_called_no  INTEGER,
      PRIMARY KEY (slot, store_id)
    ) WITHOUT ROWID
    """,
    # 4. 节假日日历（holiday / workday 调休）
    """
    CREATE TABLE IF NOT EXISTS holiday_calendar (
      date_key           TEXT PRIMARY KEY,
      date_type          TEXT NOT NULL,
      name               TEXT,
      region             TEXT
    )
    """,
    # 5. 采集与维护日志（只保留最近 30 天）
    """
    CREATE TABLE IF NOT EXISTS collector_runs (
      run_id             TEXT PRIMARY KEY,
      started_at TEXT, finished_at TEXT,
      endpoint TEXT, stores_seen INTEGER, records_written INTEGER,
      ok INTEGER, error_message TEXT, api_profile_version TEXT
    )
    """,
]

# v1 遗留表：原始逐帧快照、各类聚合与归档日志。由 migrate-v2 迁移后删除。
LEGACY_TABLES = [
    "queue_snapshots",
    "store_bucket_rollups",
    "daily_store_bucket_rollups",
    "called_intervals_rollups",
    "archive_runs",
]
