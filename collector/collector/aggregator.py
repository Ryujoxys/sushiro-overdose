"""聚合管线：从 queue_snapshots 聚合成 rollups。

产出三张表：
- store_bucket_rollups：时段聚合（压力 P50/P80 + 叫号 P20/P50/P80），worker 查它画图
- daily_store_bucket_rollups：按天细分的同结构
- called_intervals_rollups：叫号时序对的间隔/吞吐率（预测辅助）

核心分桶语义：
- store_bucket_rollups 主键 (store, date_type, weekday, bucket)，**每个 30min 桶一行**
- 同 date_type+weekday 的不同天会合并到同一行（如所有"工作日"的 17:30 合并）
- 同一天同一桶的多次采样先去重（取最后一条），避免高频放大单点
- 然后跨天合并：对该桶收集所有天的代表值，算 P50/P80（wait/groups）和 P20/P50/P80（called_no）
- busy_rate = 该桶"有排队"的样本比例
"""
from __future__ import annotations

import logging
from collections import defaultdict
from datetime import timedelta
from typing import Any, Dict, List, Optional, Tuple

from .datetype import date_type_for, half_hour_bucket, parse_iso_cst, snapshot_date
from .quantile import quantile
from .turso import TursoClient, as_int

log = logging.getLogger("collector.aggregate")

WRITE_BATCH = 200
READ_PAGE = 5000


def _load_holiday_sets(turso: TursoClient) -> Tuple[set, set]:
    holidays, workdays = set(), set()
    for r in turso.execute("SELECT date_key, date_type FROM holiday_calendar"):
        key, dt = r.get("date_key"), (r.get("date_type") or "").lower()
        if key and dt == "workday":
            workdays.add(key)
        elif key and dt == "holiday":
            holidays.add(key)
    return holidays, workdays


def aggregate_all(
    turso: TursoClient,
    days: Optional[int] = None,
    *,
    incremental_daily: bool = False,
) -> Dict[str, int]:
    """聚合快照。days=N 限制窗口；incremental_daily 只刷新最新或缺行日期。"""
    holidays, workdays = _load_holiday_sets(turso)
    log.info("节假日配置：holiday=%d workday=%d", len(holidays), len(workdays))

    rows = _load_snapshot_rows(turso, days)
    log.info("读入 %d 条快照聚合", len(rows))

    parsed = [_parse_row(r) for r in rows]
    parsed = [p for p in parsed if p]

    rollups = _build_bucket_rollups(parsed, holidays, workdays)
    daily = _build_daily_rollups(parsed, holidays, workdays)
    intervals = _build_called_intervals(parsed, holidays, workdays)
    daily_to_write = (
        _select_daily_writes(turso, daily) if incremental_daily else daily
    )

    _write_bucket_rollups(turso, rollups)
    _write_daily_rollups(turso, daily_to_write)
    _write_intervals(turso, intervals)

    log.info(
        "✅ 聚合完成：rollups=%d daily=%d/%d intervals=%d",
        len(rollups), len(daily_to_write), len(daily), len(intervals),
    )
    return {
        "rollups": len(rollups),
        "daily": len(daily_to_write),
        "intervals": len(intervals),
    }


def _load_snapshot_rows(
    turso: TursoClient, days: Optional[int]
) -> List[Dict[str, Any]]:
    """按 id 做 keyset 分页，避免大结果响应中断后整批重读。"""
    bounds = turso.execute(
        "SELECT MAX(id) AS max_id, MAX(collected_at) AS mx FROM queue_snapshots"
    )
    if not bounds:
        return []
    max_id = as_int(bounds[0].get("max_id"))
    mx = bounds[0].get("mx")
    if max_id <= 0 or not mx:
        return []

    cutoff = ""
    last_id = 0
    if days:
        cutoff_dt = parse_iso_cst(mx) - timedelta(days=days)
        cutoff = cutoff_dt.strftime("%Y-%m-%dT%H:%M:%S+08:00")
        first = turso.execute(
            "SELECT MIN(id) AS min_id FROM queue_snapshots WHERE collected_at >= ?",
            (cutoff,),
        )
        min_id = as_int(first[0].get("min_id")) if first else 0
        if min_id <= 0:
            return []
        last_id = min_id - 1

    out: List[Dict[str, Any]] = []
    pages = 0
    while last_id < max_id:
        where = "id > ? AND id <= ?"
        params: List[Any] = [last_id, max_id]
        if cutoff:
            where += " AND collected_at >= ?"
            params.append(cutoff)
        params.append(READ_PAGE)
        page = turso.execute(
            "SELECT id, collected_at, store_id, wait_minutes, group_queues_count, "
            "store_status, net_ticket_status, online_open, wait_time_cap, "
            "display_called_no, dq_source, dq_anomaly "
            f"FROM queue_snapshots WHERE {where} ORDER BY id LIMIT ?",
            tuple(params),
        )
        if not page:
            break
        out.extend(page)
        next_id = as_int(page[-1].get("id"))
        if next_id <= last_id:
            raise RuntimeError("queue_snapshots 分页游标未前进")
        last_id = next_id
        pages += 1
        if pages % 20 == 0:
            log.info("聚合分页读取：%d 条", len(out))
        if len(page) < READ_PAGE:
            break
    return out


def _select_daily_writes(turso: TursoClient, daily: List[dict]) -> List[dict]:
    """历史日不重写；最新日和库中缺行的日期需要刷新。"""
    if not daily:
        return []
    expected: Dict[str, int] = defaultdict(int)
    for row in daily:
        expected[row["snapshot_date"]] += 1
    min_date = min(expected)
    max_date = max(expected)
    existing_rows = turso.execute(
        "SELECT snapshot_date, COUNT(*) AS c "
        "FROM daily_store_bucket_rollups "
        "WHERE snapshot_date >= ? AND snapshot_date <= ? "
        "GROUP BY snapshot_date",
        (min_date, max_date),
    )
    existing = {
        str(row.get("snapshot_date")): as_int(row.get("c"))
        for row in existing_rows
    }
    refresh_dates = {
        date_key
        for date_key, count in expected.items()
        if date_key == max_date or existing.get(date_key, 0) != count
    }
    selected = [row for row in daily if row["snapshot_date"] in refresh_dates]
    log.info(
        "daily 增量写入：刷新 %d/%d 天、%d/%d 行",
        len(refresh_dates),
        len(expected),
        len(selected),
        len(daily),
    )
    return selected


# 寿司郎 wait_time_cap 恒为 180（系统硬封顶），wait 超过它就是接口异常脏数据。
# 物理上不可能让人等超过封顶值，聚合算分位数时一律丢弃。默认兜底 180。
WAIT_DIRTY_THRESHOLD = 180

# group_queues_count（在等桌数）的脏值上限。寿司郎单店同时排队超过这个数极不正常
# （实测脏值到 529，真实高峰一般 <100）。保守取 200，宁错杀少量真实高峰也不让脏值污染分位数。
GROUPS_DIRTY_THRESHOLD = 200


def _parse_row(r: Dict[str, Any]) -> Optional[dict]:
    try:
        dt = parse_iso_cst(r["collected_at"])
    except (KeyError, ValueError):
        return None
    called_raw = r.get("display_called_no")
    wait = as_int(r.get("wait_minutes"))
    cap = as_int(r.get("wait_time_cap"))
    if cap <= 0:
        cap = WAIT_DIRTY_THRESHOLD
    return {
        "dt": dt,
        "store_id": as_int(r.get("store_id")),
        "wait": wait,
        "wait_cap": cap,
        "groups": as_int(r.get("group_queues_count")),
        "store_status": (r.get("store_status") or "").upper(),
        "online_open": as_int(r.get("online_open")),
        "called_no": as_int(called_raw) if called_raw not in (None, "") else None,
        "has_called": called_raw is not None and called_raw != "",
        "dq_anomaly": as_int(r.get("dq_anomaly")),
    }


def _confidence(n: int) -> str:
    if n >= 20:
        return "high"
    if n >= 8:
        return "medium"
    return "low"


def _build_bucket_rollups(
    parsed: List[dict], holidays: set, workdays: set
) -> List[dict]:
    """按 (store, date_type, weekday, bucket) 聚合。先同天同桶去重，再跨天合并算分位数。"""
    # 第一步：同 (store, date, bucket) 去重取最后一条（collected_at 最新）
    day_bucket_latest: Dict[tuple, dict] = {}
    for p in parsed:
        bucket = half_hour_bucket(p["dt"])
        sdate = snapshot_date(p["dt"])
        key = (p["store_id"], sdate, bucket)
        prev = day_bucket_latest.get(key)
        if prev is None or p["dt"] >= prev["dt"]:
            day_bucket_latest[key] = p

    # 第二步：按 (store, date_type, weekday, bucket) 收集所有天的去重帧
    grouped: Dict[tuple, List[dict]] = defaultdict(list)
    for (store_id, sdate, bucket), p in day_bucket_latest.items():
        date_type, weekday = date_type_for(p["dt"], holidays, workdays)
        grouped[(store_id, date_type, weekday, bucket)].append(p)

    out: List[dict] = []
    now_iso = max(p["dt"] for p in parsed).isoformat() if parsed else ""
    updated_at = (now_iso or "")[:19] + "+08:00" if now_iso else ""
    for (store_id, date_type, weekday, bucket), frames in grouped.items():
        n = len(frames)
        # wait 过滤：>0 且不超过门店 cap（wait_time_cap，恒 180）。超过 cap 的是接口脏数据
        # （寿司郎不可能让人等超过封顶值），算分位数/最大值/busy 时一律丢弃。
        waits = [f["wait"] for f in frames if 0 < f["wait"] <= f["wait_cap"]]
        # groups 过滤：>0 且不超过脏值上限（实测脏值到 529，单店不可能排这么多桌）
        groups_vals = [
            f["groups"] for f in frames if 0 < f["groups"] <= GROUPS_DIRTY_THRESHOLD
        ]
        open_count = sum(1 for f in frames if f["store_status"] == "OPEN")
        online_count = sum(1 for f in frames if f["online_open"])
        # busy 用"合理 wait 或合理 groups"判断，脏值（wait>cap 或 groups>上限）不计入在排队
        busy_count = sum(
            1
            for f in frames
            if 0 < f["wait"] <= f["wait_cap"] or 0 < f["groups"] <= GROUPS_DIRTY_THRESHOLD
        )
        anomaly_count = sum(1 for f in frames if f["dq_anomaly"])

        # 叫号：只取 has_called 且 >0 的帧
        called_vals = [f["called_no"] for f in frames if f["has_called"] and (f["called_no"] or 0) > 0]

        out.append({
            "store_id": store_id,
            "date_type": date_type,
            "weekday": weekday,
            "time_bucket": bucket,
            "sample_count": n,
            "open_rate": open_count / n if n else 0.0,
            "online_open_rate": online_count / n if n else 0.0,
            "busy_rate": busy_count / n if n else 0.0,
            "wait_typical_minutes": quantile(waits, 0.5),
            "wait_safe_minutes": quantile(waits, 0.8),
            "wait_max_minutes": int(max(waits)) if waits else 0,
            "queue_groups_typical": quantile(groups_vals, 0.5),
            "queue_groups_safe": quantile(groups_vals, 0.8),
            "called_sample_count": len(called_vals),
            "called_no_slow": quantile(called_vals, 0.2),
            "called_no_typical": quantile(called_vals, 0.5),
            "called_no_fast": quantile(called_vals, 0.8),
            "dq_anomaly_rate": anomaly_count / n if n else 0.0,
            "confidence": _confidence(n),
            "updated_at": updated_at,
        })
    return out


def _build_daily_rollups(
    parsed: List[dict], holidays: set, workdays: set
) -> List[dict]:
    """按天细分：(snapshot_date, store, bucket)，同天同桶去重后直接取该天的代表值。"""
    day_bucket_latest: Dict[tuple, dict] = {}
    for p in parsed:
        bucket = half_hour_bucket(p["dt"])
        sdate = snapshot_date(p["dt"])
        key = (sdate, p["store_id"], bucket)
        prev = day_bucket_latest.get(key)
        if prev is None or p["dt"] >= prev["dt"]:
            day_bucket_latest[key] = p

    out: List[dict] = []
    for (sdate, store_id, bucket), f in day_bucket_latest.items():
        date_type, weekday = date_type_for(f["dt"], holidays, workdays)
        called = f["called_no"] if f["has_called"] and (f["called_no"] or 0) > 0 else None
        # 脏值过滤（与 _build_bucket_rollups 一致）：wait>cap 或 groups>上限视为无效
        wait_ok = 0 < f["wait"] <= f["wait_cap"]
        groups_ok = 0 < f["groups"] <= GROUPS_DIRTY_THRESHOLD
        busy = wait_ok or groups_ok
        out.append({
            "snapshot_date": sdate,
            "store_id": store_id,
            "date_type": date_type,
            "weekday": weekday,
            "time_bucket": bucket,
            "sample_count": 1,
            "open_count": 1 if f["store_status"] == "OPEN" else 0,
            "online_open_count": 1 if f["online_open"] else 0,
            "busy_count": 1 if busy else 0,
            "open_rate": 1.0 if f["store_status"] == "OPEN" else 0.0,
            "online_open_rate": 1.0 if f["online_open"] else 0.0,
            "busy_rate": 1.0 if busy else 0.0,
            "wait_typical_minutes": float(f["wait"]) if wait_ok else None,
            "wait_safe_minutes": float(f["wait"]) if wait_ok else None,
            "wait_max_minutes": f["wait"] if wait_ok else 0,
            "queue_groups_typical": float(f["groups"]) if groups_ok else None,
            "queue_groups_safe": float(f["groups"]) if groups_ok else None,
            "called_sample_count": 1 if called is not None else 0,
            "called_no_slow": called,
            "called_no_typical": called,
            "called_no_fast": called,
            "confidence": "low",
            "updated_at": f["dt"].isoformat(),
        })
    return out


def _build_called_intervals(
    parsed: List[dict], holidays: set, workdays: set
) -> List[dict]:
    """叫号时序对：连续两帧（同店、时间递增）都 >0 且号递增 → 算 interval/delta。

    落在"前一个观测点的 bucket"（叫号推进发生时所在时段）。按 (store, date_type, bucket) 聚合。
    throughput = P50(delta) / P50(interval_seconds) * 3600
    capacity_utilization = throughput_per_hour / tables_capacity（capacity 在调用处补）
    """
    by_store: Dict[int, List[dict]] = defaultdict(list)
    for p in parsed:
        if p["has_called"] and (p["called_no"] or 0) > 0:
            by_store[p["store_id"]].append(p)

    grouped: Dict[tuple, List[dict]] = defaultdict(list)
    for store_id, frames in by_store.items():
        frames.sort(key=lambda x: x["dt"])
        for i in range(1, len(frames)):
            prev, cur = frames[i - 1], frames[i]
            delta = (cur["called_no"] or 0) - (prev["called_no"] or 0)
            if delta <= 0:
                continue  # 号没推进或倒退（倒退是异常，不计入正常间隔）
            interval_sec = (cur["dt"] - prev["dt"]).total_seconds()
            if interval_sec <= 0 or interval_sec > 6 * 3600:
                continue  # 间隔异常（>6h 多半跨了采集断档）
            date_type, _ = date_type_for(prev["dt"], holidays, workdays)
            bucket = half_hour_bucket(prev["dt"])
            grouped[(store_id, date_type, bucket)].append({
                "delta": delta, "interval": interval_sec,
            })

    out: List[dict] = []
    for (store_id, date_type, bucket), pairs in grouped.items():
        deltas = [p["delta"] for p in pairs]
        intervals = [p["interval"] for p in pairs]
        delta_p50 = quantile(deltas, 0.5) or 0
        interval_p50 = quantile(intervals, 0.5) or 0
        throughput = (delta_p50 / interval_p50 * 3600) if interval_p50 > 0 else None
        out.append({
            "store_id": store_id,
            "date_type": date_type,
            "time_bucket": bucket,
            "pair_count": len(pairs),
            "interval_typical_seconds": interval_p50,
            "delta_typical": delta_p50,
            "throughput_per_hour": throughput,
            "capacity_utilization": None,  # 写入时按 store 的 tables_capacity 补
        })
    return out


def _write_bucket_rollups(turso: TursoClient, rollups: List[dict]) -> None:
    if not rollups:
        return
    sql = """
    INSERT INTO store_bucket_rollups
      (store_id, date_type, weekday, time_bucket, sample_count,
       open_rate, online_open_rate, busy_rate,
       wait_typical_minutes, wait_safe_minutes, wait_max_minutes,
       queue_groups_typical, queue_groups_safe,
       called_sample_count, called_no_slow, called_no_typical, called_no_fast,
       dq_anomaly_rate, confidence, updated_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(store_id, date_type, weekday, time_bucket) DO UPDATE SET
      sample_count=excluded.sample_count, open_rate=excluded.open_rate,
      online_open_rate=excluded.online_open_rate, busy_rate=excluded.busy_rate,
      wait_typical_minutes=excluded.wait_typical_minutes,
      wait_safe_minutes=excluded.wait_safe_minutes,
      wait_max_minutes=excluded.wait_max_minutes,
      queue_groups_typical=excluded.queue_groups_typical,
      queue_groups_safe=excluded.queue_groups_safe,
      called_sample_count=excluded.called_sample_count,
      called_no_slow=excluded.called_no_slow,
      called_no_typical=excluded.called_no_typical,
      called_no_fast=excluded.called_no_fast,
      dq_anomaly_rate=excluded.dq_anomaly_rate,
      confidence=excluded.confidence, updated_at=excluded.updated_at
    WHERE
      store_bucket_rollups.sample_count IS NOT excluded.sample_count
      OR store_bucket_rollups.open_rate IS NOT excluded.open_rate
      OR store_bucket_rollups.online_open_rate IS NOT excluded.online_open_rate
      OR store_bucket_rollups.busy_rate IS NOT excluded.busy_rate
      OR store_bucket_rollups.wait_typical_minutes IS NOT excluded.wait_typical_minutes
      OR store_bucket_rollups.wait_safe_minutes IS NOT excluded.wait_safe_minutes
      OR store_bucket_rollups.wait_max_minutes IS NOT excluded.wait_max_minutes
      OR store_bucket_rollups.queue_groups_typical IS NOT excluded.queue_groups_typical
      OR store_bucket_rollups.queue_groups_safe IS NOT excluded.queue_groups_safe
      OR store_bucket_rollups.called_sample_count IS NOT excluded.called_sample_count
      OR store_bucket_rollups.called_no_slow IS NOT excluded.called_no_slow
      OR store_bucket_rollups.called_no_typical IS NOT excluded.called_no_typical
      OR store_bucket_rollups.called_no_fast IS NOT excluded.called_no_fast
      OR store_bucket_rollups.dq_anomaly_rate IS NOT excluded.dq_anomaly_rate
      OR store_bucket_rollups.confidence IS NOT excluded.confidence
    """
    args = [
        (
            r["store_id"], r["date_type"], r["weekday"], r["time_bucket"], r["sample_count"],
            r["open_rate"], r["online_open_rate"], r["busy_rate"],
            r["wait_typical_minutes"], r["wait_safe_minutes"], r["wait_max_minutes"],
            r["queue_groups_typical"], r["queue_groups_safe"],
            r["called_sample_count"], r["called_no_slow"], r["called_no_typical"],
            r["called_no_fast"], r["dq_anomaly_rate"], r["confidence"], r["updated_at"],
        )
        for r in rollups
    ]
    _batch_write(turso, sql, args)


def _write_daily_rollups(turso: TursoClient, daily: List[dict]) -> None:
    if not daily:
        return
    sql = """
    INSERT INTO daily_store_bucket_rollups
      (snapshot_date, store_id, date_type, weekday, time_bucket,
       sample_count, open_count, online_open_count, busy_count,
       open_rate, online_open_rate, busy_rate,
       wait_typical_minutes, wait_safe_minutes, wait_max_minutes,
       queue_groups_typical, queue_groups_safe,
       called_sample_count, called_no_slow, called_no_typical, called_no_fast,
       confidence, updated_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(snapshot_date, store_id, time_bucket) DO UPDATE SET
      date_type=excluded.date_type, weekday=excluded.weekday,
      sample_count=excluded.sample_count, open_count=excluded.open_count,
      online_open_count=excluded.online_open_count, busy_count=excluded.busy_count,
      open_rate=excluded.open_rate, online_open_rate=excluded.online_open_rate,
      busy_rate=excluded.busy_rate,
      wait_typical_minutes=excluded.wait_typical_minutes,
      wait_safe_minutes=excluded.wait_safe_minutes,
      wait_max_minutes=excluded.wait_max_minutes,
      queue_groups_typical=excluded.queue_groups_typical,
      queue_groups_safe=excluded.queue_groups_safe,
      called_sample_count=excluded.called_sample_count,
      called_no_slow=excluded.called_no_slow,
      called_no_typical=excluded.called_no_typical,
      called_no_fast=excluded.called_no_fast,
      confidence=excluded.confidence,
      updated_at=excluded.updated_at
    WHERE
      daily_store_bucket_rollups.date_type IS NOT excluded.date_type
      OR daily_store_bucket_rollups.weekday IS NOT excluded.weekday
      OR daily_store_bucket_rollups.sample_count IS NOT excluded.sample_count
      OR daily_store_bucket_rollups.open_count IS NOT excluded.open_count
      OR daily_store_bucket_rollups.online_open_count IS NOT excluded.online_open_count
      OR daily_store_bucket_rollups.busy_count IS NOT excluded.busy_count
      OR daily_store_bucket_rollups.open_rate IS NOT excluded.open_rate
      OR daily_store_bucket_rollups.online_open_rate IS NOT excluded.online_open_rate
      OR daily_store_bucket_rollups.busy_rate IS NOT excluded.busy_rate
      OR daily_store_bucket_rollups.wait_typical_minutes IS NOT excluded.wait_typical_minutes
      OR daily_store_bucket_rollups.wait_safe_minutes IS NOT excluded.wait_safe_minutes
      OR daily_store_bucket_rollups.wait_max_minutes IS NOT excluded.wait_max_minutes
      OR daily_store_bucket_rollups.queue_groups_typical IS NOT excluded.queue_groups_typical
      OR daily_store_bucket_rollups.queue_groups_safe IS NOT excluded.queue_groups_safe
      OR daily_store_bucket_rollups.called_sample_count IS NOT excluded.called_sample_count
      OR daily_store_bucket_rollups.called_no_slow IS NOT excluded.called_no_slow
      OR daily_store_bucket_rollups.called_no_typical IS NOT excluded.called_no_typical
      OR daily_store_bucket_rollups.called_no_fast IS NOT excluded.called_no_fast
      OR daily_store_bucket_rollups.confidence IS NOT excluded.confidence
    """
    args = [
        (
            d["snapshot_date"], d["store_id"], d["date_type"], d["weekday"], d["time_bucket"],
            d["sample_count"], d["open_count"], d["online_open_count"], d["busy_count"],
            d["open_rate"], d["online_open_rate"], d["busy_rate"],
            d["wait_typical_minutes"], d["wait_safe_minutes"], d["wait_max_minutes"],
            d["queue_groups_typical"], d["queue_groups_safe"],
            d["called_sample_count"], d["called_no_slow"], d["called_no_typical"],
            d["called_no_fast"], d["confidence"], d["updated_at"],
        )
        for d in daily
    ]
    _batch_write(turso, sql, args)


def _write_intervals(turso: TursoClient, intervals: List[dict]) -> None:
    if not intervals:
        return
    # 取 store → tables_capacity 映射，补 capacity_utilization
    cap_map: Dict[int, int] = {}
    for r in turso.execute("SELECT store_id, tables_capacity FROM store_dimension"):
        cap_map[as_int(r.get("store_id"))] = as_int(r.get("tables_capacity"))

    sql = """
    INSERT INTO called_intervals_rollups
      (store_id, date_type, time_bucket, pair_count,
       interval_typical_seconds, delta_typical, throughput_per_hour,
       capacity_utilization, updated_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(store_id, date_type, time_bucket) DO UPDATE SET
      pair_count=excluded.pair_count,
      interval_typical_seconds=excluded.interval_typical_seconds,
      delta_typical=excluded.delta_typical,
      throughput_per_hour=excluded.throughput_per_hour,
      capacity_utilization=excluded.capacity_utilization, updated_at=excluded.updated_at
    WHERE
      called_intervals_rollups.pair_count IS NOT excluded.pair_count
      OR called_intervals_rollups.interval_typical_seconds IS NOT excluded.interval_typical_seconds
      OR called_intervals_rollups.delta_typical IS NOT excluded.delta_typical
      OR called_intervals_rollups.throughput_per_hour IS NOT excluded.throughput_per_hour
      OR called_intervals_rollups.capacity_utilization IS NOT excluded.capacity_utilization
    """
    args = []
    from .collector import _fmt_dt
    updated = _fmt_dt()
    for it in intervals:
        cap = cap_map.get(it["store_id"], 0)
        util = (it["throughput_per_hour"] / cap) if (cap > 0 and it["throughput_per_hour"]) else None
        args.append(
            (
                it["store_id"], it["date_type"], it["time_bucket"], it["pair_count"],
                it["interval_typical_seconds"], it["delta_typical"], it["throughput_per_hour"],
                util, updated,
            )
        )
    _batch_write(turso, sql, args)


def _batch_write(turso: TursoClient, sql: str, args: List[tuple]) -> None:
    for i in range(0, len(args), WRITE_BATCH):
        batch = args[i : i + WRITE_BATCH]
        turso.execute_many([(sql, a) for a in batch], retry_safe=True)
