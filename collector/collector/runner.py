"""常驻运行循环。

每 interval_seconds 采一轮（营业时段内）。每天凌晨：
- 02:00 跑一次 30 天滚动聚合（产出 rollups + 叫号三档 + 间隔/吞吐）
- 03:00 跑一次归档（裁剪超期原始快照）

优雅退出：SIGTERM/SIGINT 触发后完成当前轮再退出（systemd restart 不丢数据）。
"""
from __future__ import annotations

import logging
import signal
import threading
import time
from datetime import datetime, timedelta, timezone
from typing import Any, Dict

from .aggregator import aggregate_all
from .archive import archive_old
from .collector import _fmt_dt, _record_run, collect_once
from .config import require_credential
from .turso import TursoClient, as_int

log = logging.getLogger("collector.run")

CST = timezone(timedelta(hours=8))

_STOP = threading.Event()


def _handle_signal(signum, _frame):
    log.info("收到信号 %s，准备退出（完成当前轮）", signum)
    _STOP.set()


def _seconds_until_next_boundary(now: datetime, interval: int) -> int:
    """算到下一个对齐边界的秒数。

    当 interval 能整除 3600（如 900=15min）时，对齐到整点边界（:00/:15/:30/:45）。
    不能整除则退化为固定 interval。返回值至少 1s。
    """
    if interval <= 0 or 3600 % interval != 0:
        return max(1, interval)
    # 当前分钟在小时内的秒偏移
    cur = now.minute * 60 + now.second
    boundary = interval
    # 下一个边界：向上取整到 interval 的倍数
    wait = boundary - (cur % boundary)
    if wait <= 0:
        wait = boundary
    return max(1, wait)


def _maintenance_done(turso: TursoClient, job: str, date_key: str) -> bool:
    run_id = f"maintenance-{job}-{date_key}"
    rows = turso.execute(
        "SELECT ok FROM collector_runs WHERE run_id = ?",
        (run_id,),
    )
    return bool(rows and as_int(rows[0].get("ok")) == 1)


def _record_maintenance(
    turso: TursoClient,
    job: str,
    date_key: str,
    started_at: str,
    records_written: int,
) -> None:
    _record_run(
        turso,
        f"maintenance-{job}-{date_key}",
        started_at,
        job,
        0,
        records_written,
        True,
        "",
    )


def _normalize_collect_cfg(
    coll_cfg: Dict[str, Any],
) -> tuple:
    """启动时校验采集配置，返回 (interval, active_hours)。

    非法值回退默认并打错误日志——比带病运行好：interval<=0 会退化成
    每秒一轮 hammer 上游 API；active_hours 类型/取值错会在主循环里
    TypeError 崩溃（systemd Restart=always 下变成 30s 崩溃循环）。
    """
    default_interval = 900
    default_hours = [10, 22]

    try:
        interval = int(coll_cfg.get("interval_seconds", default_interval))
    except (TypeError, ValueError):
        interval = -1
    # 下限 60s：再快就是对上游 API 的 DoS；负数/0 曾把循环变成每秒一采
    if interval < 60:
        log.error(
            "interval_seconds=%r 非法（需 >=60），回退默认 %ds",
            coll_cfg.get("interval_seconds"), default_interval,
        )
        interval = default_interval

    raw_hours = coll_cfg.get("active_hours", default_hours)
    active_hours = default_hours
    try:
        hours = [int(h) for h in raw_hours] if raw_hours else []
        if len(hours) == 2 and 0 <= hours[0] < hours[1] <= 24:
            active_hours = hours
        else:
            raise ValueError("结构不是 [lo, hi] 或越界")
    except (TypeError, ValueError) as e:
        log.error("active_hours=%r 非法（%s），回退默认 %s", raw_hours, e, default_hours)
    return interval, active_hours


def run_loop(cfg: Dict[str, Any]) -> None:
    signal.signal(signal.SIGTERM, _handle_signal)
    signal.signal(signal.SIGINT, _handle_signal)

    coll_cfg = cfg.get("collect", {})
    interval, active_hours = _normalize_collect_cfg(coll_cfg)
    retention = int(cfg.get("archive", {}).get("retention_days", 60))
    maintenance_end_hour = 10
    if active_hours and len(active_hours) == 2:
        active_start = int(active_hours[0])
        if active_start > 3:
            maintenance_end_hour = active_start

    last_aggregate_date = ""
    last_archive_date = ""

    log.info(
        "采集器启动：interval=%ds active=%s retention=%dd（对齐整点 :00/:15/:30/:45）",
        interval, active_hours, retention,
    )

    # 启动后先睡到下一个整点边界，让采集对齐 :00/:15/:30/:45（与 30min 聚合桶一致）
    first_wait = _seconds_until_next_boundary(datetime.now(CST), interval)
    log.info("首次采集等待 %ds 对齐到整点边界", first_wait)
    if _STOP.wait(first_wait):
        log.info("采集器已退出（启动等待期收到信号）")
        return

    while not _STOP.is_set():
        now = datetime.now(CST)
        today = now.strftime("%Y-%m-%d")

        # 每日聚合仅在 02:00 到营业开始前补跑，避免白天重启后抢占采集。
        if 2 <= now.hour < maintenance_end_hour and last_aggregate_date != today:
            log.info("每日聚合 %s", today)
            try:
                turso = TursoClient(
                    require_credential(cfg, "turso", "url"),
                    require_credential(cfg, "turso", "auth_token"),
                )
                if _maintenance_done(turso, "aggregate", today):
                    log.info("每日聚合 %s 已完成，跳过", today)
                else:
                    started_at = _fmt_dt()
                    daily_v2_ready = _maintenance_done(
                        turso, "daily-rollup-v2", "baseline"
                    )
                    # 只聚合最近 30 天，并按主键分页读取，避免单个超大响应中断。
                    stats = aggregate_all(
                        turso,
                        days=30,
                        incremental_daily=daily_v2_ready,
                    )
                    if not daily_v2_ready:
                        _record_maintenance(
                            turso,
                            "daily-rollup-v2",
                            "baseline",
                            started_at,
                            stats["daily"],
                        )
                    _record_maintenance(
                        turso,
                        "aggregate",
                        today,
                        started_at,
                        sum(stats.values()),
                    )
                last_aggregate_date = today
            except Exception as e:
                log.error("聚合失败: %s", e)

        # 每日归档同样限制在凌晨维护窗口。
        if 3 <= now.hour < maintenance_end_hour and last_archive_date != today:
            log.info("每日归档 %s", today)
            try:
                turso = TursoClient(
                    require_credential(cfg, "turso", "url"),
                    require_credential(cfg, "turso", "auth_token"),
                )
                if _maintenance_done(turso, "archive", today):
                    log.info("每日归档 %s 已完成，跳过", today)
                else:
                    started_at = _fmt_dt()
                    stats = archive_old(turso, retention)
                    _record_maintenance(
                        turso,
                        "archive",
                        today,
                        started_at,
                        stats["deleted"],
                    )
                last_archive_date = today
            except Exception as e:
                log.error("归档失败: %s", e)

        # 营业时段判断（active_hours=[10,22] 表示 10≤hour<22 才采）。
        # 只作用于采集，不影响上面的聚合/归档。
        if active_hours and len(active_hours) == 2:
            lo, hi = active_hours
            if lo >= hi:
                log.error("active_hours 配置错误 lo>=hi: %s，跳过采集", active_hours)
                _STOP.wait(_seconds_until_next_boundary(datetime.now(CST), interval))
                continue
            if not (lo <= now.hour < hi):
                log.debug("非营业时段 %s，跳过采集", now.strftime("%H:%M"))
                _STOP.wait(_seconds_until_next_boundary(datetime.now(CST), interval))
                continue

        # 跑一轮采集
        try:
            collect_once(cfg)
        except Exception as e:
            log.error("采集轮失败（下轮重试）: %s", e)

        # 等下一轮：对齐到整点边界（:00/:15/:30/:45），让快照时间规整、与 30min 聚合桶对齐
        _STOP.wait(_seconds_until_next_boundary(datetime.now(CST), interval))

    log.info("采集器已退出")
