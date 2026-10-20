#!/usr/bin/env python3
"""寿司郎排队数据查询：读 GitHub data 分支的公开数据，可选读本机应用的实时叫号。

只读、纯标准库。不取号、不取消、不带任何凭证。口径与应用图表一致：
半小时代表值，只统计营业中的值；等待 P50/P80，叫号 P20/P50/P80，线性插值。
"""
from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import os
import sys
import urllib.error
import urllib.request
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional

CST = timezone(timedelta(hours=8))
SCHEMA = "sushiro-public-data"
DEFAULT_URL = "https://raw.githubusercontent.com/Ryujoxys/sushiro-overdose/data/"
DATE_TYPES = ("weekday", "weekend", "holiday", "workday")
TYPE_LABEL = {"weekday": "工作日", "weekend": "周末", "holiday": "节假日", "workday": "调休工作日", "all": "全部日子"}
FEW_DAYS = 3
WAIT_CAP = 180  # 官方显示上限：公开数据里 180 表示“180 分钟及以上”
BUNDLE = Path(__file__).resolve().parents[4] / "internal/app/data/queue-history-v1.json.gz"


class DataError(Exception):
    pass


# ---------- 数据源 ----------

class Source:
    """data 分支（URL 或本地目录），失败时回退仓库内置历史包。"""

    def __init__(self, base: Optional[str] = None, cache_dir: Optional[Path] = None, bundle: Path = BUNDLE):
        self.base = base or os.environ.get("SUSHIRO_DATA_URL") or os.environ.get("SUSHIRO_DATA_DIR") or DEFAULT_URL
        self.cache = cache_dir or Path(os.environ.get("SUSHIRO_DATA_CACHE") or Path.home() / ".cache/sushiro-queue")
        self.bundle_path = bundle
        self._meta: Optional[Dict[str, Any]] = None
        self._bundle: Optional[Dict[str, Any]] = None
        self.fallback_reason = ""

    def _fetch(self, name: str) -> bytes:
        if not self.base.startswith(("http://", "https://")):
            return (Path(self.base) / name).read_bytes()
        req = urllib.request.Request(self.base.rstrip("/") + "/" + name, headers={"User-Agent": "sushiro-queue-skill"})
        with urllib.request.urlopen(req, timeout=20) as resp:
            return resp.read()

    def meta(self) -> Dict[str, Any]:
        if self._meta is None:
            try:
                meta = json.loads(self._fetch("meta.json"))
                if meta.get("schema") != SCHEMA or meta.get("version") != 1:
                    raise DataError("未知的数据格式版本")
                self._meta = meta
            except (OSError, urllib.error.URLError, ValueError, DataError) as e:
                self.fallback_reason = f"公开数据读取失败（{e}），改用仓库内置历史包"
                self._meta = self._bundle_meta()
        return self._meta

    def json(self, name: str) -> Any:
        meta = self.meta()
        if meta["source"] == "repo-bundle":
            raise DataError("bundle")
        info = meta["files"].get(name)
        if not info:
            raise DataError(f"没有 {name}")
        cached = self.cache / (info["sha256"] + ".json")
        if cached.exists():
            return json.loads(cached.read_bytes())
        raw = self._fetch(name)
        if hashlib.sha256(raw).hexdigest() != info["sha256"]:
            raise DataError(f"{name} 校验失败，数据分支可能正在更新，稍后重试")
        try:
            self.cache.mkdir(parents=True, exist_ok=True)
            cached.write_bytes(raw)
        except OSError:
            pass
        return json.loads(raw)

    # 回退：仓库内置历史包（与应用图表同一份数据）
    def _load_bundle(self) -> Dict[str, Any]:
        if self._bundle is None:
            if not self.bundle_path.exists():
                raise DataError("公开数据和内置历史包都不可用")
            self._bundle = json.loads(gzip.decompress(self.bundle_path.read_bytes()))
        return self._bundle

    def _bundle_meta(self) -> Dict[str, Any]:
        b = self._load_bundle()
        return {"schema": SCHEMA, "version": 1, "source": "repo-bundle", "first_at": b["first_at"],
                "cutoff_at": b["cutoff_at"], "store_count": len(b["stores"]), "record_count": len(b["records"]),
                "day_count": len({r["collected_at"][:10] for r in b["records"]}), "quality_note": b.get("quality_note", ""),
                "collector": None, "calendar": {}, "files": {}}

    def stores(self) -> List[Dict[str, Any]]:
        try:
            return self.json("stores.json")
        except DataError:
            return self._load_bundle()["stores"]

    def daily(self, store_id: int) -> List[Dict[str, Any]]:
        try:
            data = self.json(f"daily/{store_id}.json")
            return [dict(zip(data["fields"], row)) for row in data["rows"]]
        except DataError as e:
            if str(e) != "bundle":
                raise
            return [r for r in self._load_bundle()["records"] if r["store_id"] == store_id]

    def latest(self) -> Dict[int, Dict[str, Any]]:
        try:
            return {r["store_id"]: r for r in self.json("latest.json")["stores"]}
        except DataError:
            return {}


# ---------- 统计（与 scripts/export-public-data.mjs 一致） ----------

def quantile(values: List[float], q: float) -> Optional[float]:
    if not values:
        return None
    s = sorted(values)
    pos = q * (len(s) - 1)
    lo, hi = int(pos), min(int(pos) + 1, len(s) - 1)
    return s[lo] + (s[hi] - s[lo]) * (pos - lo)


def half_hour(stamp: str) -> str:
    t = datetime.fromisoformat(stamp).astimezone(CST)
    return f"{t.hour:02d}:{'00' if t.minute < 30 else '30'}"


def date_type_at(day: str, hhmm: str, calendar: Dict[str, str]) -> str:
    if calendar.get(day) in ("holiday", "workday"):
        return calendar[day]
    wd = datetime.fromisoformat(day).weekday()  # 周一 0
    minutes = int(hhmm[:2]) * 60 + int(hhmm[3:5])
    if wd == 5 or (wd == 4 and minutes >= 990) or (wd == 6 and minutes < 1320):
        return "weekend"
    return "weekday"


def patterns(rows: List[Dict[str, Any]], date_type: str = "all", since: str = "") -> Dict[str, Dict[str, Any]]:
    """半小时 → {wait_days, wait_p50, wait_p80, called_days, called_p20, called_p50, called_p80}。"""
    groups: Dict[str, Dict[str, list]] = {}
    for r in rows:
        if r["store_status"] != "OPEN" or r["collected_at"][:10] < since:
            continue
        if date_type != "all" and r["date_type"] != date_type:
            continue
        g = groups.setdefault(half_hour(r["collected_at"]), {"w": [], "c": []})
        if r["wait_minutes"] is not None:
            g["w"].append(r["wait_minutes"])
        if r["display_called_no"] is not None:
            g["c"].append(r["display_called_no"])
    out = {}
    for t in sorted(groups):
        w, c = groups[t]["w"], groups[t]["c"]
        out[t] = {"time": t, "wait_days": len(w), "wait_p50": quantile(w, .5), "wait_p80": quantile(w, .8),
                  "called_days": len(c), "called_p20": quantile(c, .2), "called_p50": quantile(c, .5), "called_p80": quantile(c, .8)}
    return out


def day_patterns(rows, day: str, calendar: Dict[str, str], since: str = "") -> Dict[str, Dict[str, Any]]:
    """目标日期每个半小时用它自己的日期类型（周五 16:30 前后不同）。"""
    by_type = {t: patterns(rows, t, since) for t in DATE_TYPES}
    merged = {}
    for t in sorted({k for p in by_type.values() for k in p}):
        dt = date_type_at(day, t, calendar)
        if t in by_type[dt]:
            merged[t] = dict(by_type[dt][t], date_type=dt)
    return merged


# ---------- 实时（本机应用，只读 GET） ----------

def live_from_app(store_id: int, port: Optional[int] = None) -> Optional[Dict[str, Any]]:
    port = port or int(os.environ.get("SUSHIRO_APP_PORT") or 39871)
    url = f"http://127.0.0.1:{port}/api/queue/live?store={store_id}"
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    try:
        with opener.open(urllib.request.Request(url, headers={"Host": f"127.0.0.1:{port}"}), timeout=8) as resp:
            data = json.loads(resp.read())
        return data if isinstance(data, dict) and data.get("store_id") else None
    except (OSError, urllib.error.URLError, ValueError):
        return None


# ---------- 业务 ----------

def now_cst() -> datetime:
    return datetime.now(CST)


def freshness(meta: Dict[str, Any], now: datetime) -> List[str]:
    notes = []
    cutoff = datetime.fromisoformat(meta["cutoff_at"])
    age = (now - cutoff).days
    if age >= 3:
        notes.append(f"历史数据截止 {meta['cutoff_at'][:16].replace('T', ' ')}，已有 {age} 天没有新数据；规律仍可参考，但不反映最近变化。")
    col = meta.get("collector") or {}
    if col.get("recent_runs") and not col.get("recent_ok"):
        notes.append("线上采集最近全部失败（公开接口可能拒绝了采集服务器）。")
    if meta.get("quality_note"):
        notes.append(meta["quality_note"])
    return notes


def find_store(src: Source, query: str) -> Dict[str, Any]:
    stores = src.stores()
    if query.isdigit():
        for s in stores:
            if s["id"] == int(query):
                return s
        raise DataError(f"没有门店 {query} 的数据")
    hits = search_stores(stores, query)
    if not hits:
        raise DataError(f"找不到“{query}”，先用 stores 搜索")
    if len(hits) > 1 and not any(s["name"] == query for s in hits):
        names = "、".join(f"{s['name']}({s['id']})" for s in hits[:8])
        raise DataError(f"“{query}”匹配多家：{names}，请用门店 ID")
    return next((s for s in hits if s["name"] == query), hits[0])


def search_stores(stores, query: str):
    q = query.strip().lower()
    return [s for s in stores if any(q in str(s.get(k) or "").lower() for k in ("name", "city", "area", "address"))]


def minutes_of(hhmm: str) -> int:
    return int(hhmm[:2]) * 60 + int(hhmm[3:5])


def clock(m: float) -> str:
    m = int(round(m))
    return f"{m // 60:02d}:{m % 60:02d}"


def build_plan(pattern: Dict[str, Dict[str, Any]], eat: str, travel: int = 0) -> Dict[str, Any]:
    """与应用 buildQueueMealPlan 相同：10:00 起每 10 分钟一个候选取号时刻，
    按该半小时的等待 P50/P80 推算入座时间；best 让 P50 入座最接近目标，latest 是 P80 仍不晚于目标的最晚取号。"""
    target = minutes_of(eat) - max(0, travel)
    best = latest = None
    best_gap = None
    for t in range(600, minutes_of(eat) + 1, 10):
        b = pattern.get(clock(t - t % 30))
        if not b or b["wait_p50"] is None:
            continue
        low, high = round(b["wait_p50"]), round(b["wait_p80"])
        if t + high <= target:
            latest = t
        gap = abs(t + low - target)
        if best_gap is None or gap < best_gap:
            best_gap, best = gap, (t, low, high, b["wait_days"])
    if best is None:
        return {"ok": False, "message": "这个时段之前没有可用的等待记录，没法倒推取号时间。"}
    early, late = sorted([best[0], latest if latest is not None else best[0]])
    return {"ok": True, "eat": eat, "travel": travel, "pickup": clock(best[0]), "latest_safe": clock(latest) if latest is not None else None,
            "range": [clock(early), clock(late)], "wait": [best[1], best[2]], "wait_days": best[3]}


def build_eta(pattern: Dict[str, Dict[str, Any]], ticket: int, after: str = "00:00") -> Dict[str, Any]:
    """号码 N 通常几点叫到：某半小时叫号 P80≥N 说明约两成日子已叫到，P50 一半日子，P20 八成日子。"""
    out = {"ticket": ticket, "fast": None, "typical": None, "slow": None}
    first = True
    for t, b in pattern.items():
        if t < after[:5] or b["called_p50"] is None:
            continue
        for key, q in (("fast", "called_p80"), ("typical", "called_p50"), ("slow", "called_p20")):
            if out[key] is None and b[q] >= ticket:
                out[key] = {"time": t, "days": b["called_days"], "already": first}
        first = False
    return out


# ---------- 输出 ----------

def fmt_num(v, step: int = 1) -> str:
    return "—" if v is None else str(int(round(v / step) * step))


def fmt_wait(v) -> str:
    return "—" if v is None else ("180+" if v >= WAIT_CAP else fmt_num(v, 5))


def emit(args, data: Dict[str, Any], lines: List[str]) -> None:
    if args.json:
        print(json.dumps(data, ensure_ascii=False, indent=1))
    else:
        print("\n".join(lines))


def cmd_status(src: Source, args) -> None:
    meta, now = src.meta(), now_cst()
    col = meta.get("collector") or {}
    lines = [f"数据来源：{'GitHub data 分支（' + meta['source'] + '）' if meta['source'] != 'repo-bundle' else '仓库内置历史包'}",
             f"范围：{meta['first_at'][:10]} 至 {meta['cutoff_at'][:16].replace('T', ' ')}，{meta['day_count']} 天，{meta['store_count']} 家店",
             f"导出时间：{meta.get('generated_at', '—')}"]
    if col:
        lines.append(f"线上采集：最近 {col.get('recent_runs')} 轮成功 {col.get('recent_ok')} 轮，上次成功 {col.get('last_ok_at') or '近期没有'}")
    if src.fallback_reason:
        lines.append(src.fallback_reason + "；内置包里超过 3 小时的等待记为未知，高峰等待会偏短。")
    lines += freshness(meta, now)
    emit(args, {"meta": {k: v for k, v in meta.items() if k != "files"}, "notes": freshness(meta, now), "fallback": src.fallback_reason}, lines)


def cmd_stores(src: Source, args) -> None:
    hits = search_stores(src.stores(), args.query or "")
    lines = [f"{s['id']:>5}  {s['name']}  {s.get('city') or ''}{s.get('area') or ''}  {s.get('days', '')}{'天' if s.get('days') else ''}" for s in hits[: args.limit]]
    emit(args, {"stores": hits[: args.limit], "total": len(hits)}, lines + ([f"共 {len(hits)} 家，只列前 {args.limit} 家"] if len(hits) > args.limit else []))


def resolve_day(meta: Dict[str, Any], day: Optional[str]) -> str:
    return day or now_cst().strftime("%Y-%m-%d")


def cmd_pattern(src: Source, args) -> None:
    meta, store = src.meta(), find_store(src, args.store)
    rows = src.daily(store["id"])
    since = (datetime.fromisoformat(meta["cutoff_at"][:10]) - timedelta(days=args.recent - 1)).strftime("%Y-%m-%d") if args.recent else ""
    if args.type == "auto":
        day = resolve_day(meta, args.date)
        pat, title = day_patterns(rows, day, meta.get("calendar") or {}, since), f"{day}（按当天每个时段的日期类型）"
    else:
        pat, title = patterns(rows, args.type, since), TYPE_LABEL[args.type]
    pat = {t: b for t, b in pat.items() if args.start <= t <= args.end}
    lines = [f"{store['name']}（{store['id']}）· {title}" + (f" · 最近 {args.recent} 天" if args.recent else ""),
             "时段   通常要等  八成日子不超过  通常叫到   常见范围      天数(等待/叫号)"]
    for t, b in pat.items():
        rng = "—" if b["called_p20"] is None else f"{fmt_num(b['called_p20'])}–{fmt_num(b['called_p80'])}"
        few = " 少" if 0 < min(b["wait_days"] or 99, b["called_days"] or 99) < FEW_DAYS else ""
        lines.append(f"{t}  {fmt_wait(b['wait_p50']):>5} 分  {fmt_wait(b['wait_p80']):>8} 分   {fmt_num(b['called_p50']):>6}   {rng:<12}  {b['wait_days']}/{b['called_days']}{few}")
    blind = [t for t, b in pat.items() if b["called_days"] >= FEW_DAYS and b["wait_days"] < FEW_DAYS]
    notes = freshness(meta, now_cst())
    if blind:
        notes.append(f"{blind[0]}–{blind[-1]} 有叫号但多数日子没有等待记录，等待规律缺这段，实际可能更挤。")
    if any(b["wait_p80"] is not None and b["wait_p80"] >= WAIT_CAP for b in pat.values()):
        notes.append("180+ 表示达到官方显示上限 3 小时，实际可能更久。")
    lines += [""] + notes if notes else []
    emit(args, {"store": store, "title": title, "buckets": list(pat.values()), "notes": notes}, lines)


def cmd_day(src: Source, args) -> None:
    store = src.daily(find_store(src, args.store)["id"])
    rows = [r for r in store if r["collected_at"][:10] == args.date]
    lines = [f"{r['collected_at'][11:16]}  {r['store_status']:<7} 等待 {fmt_num(r['wait_minutes'])} 分  叫号 {fmt_num(r['display_called_no'])}" for r in rows] or ["这天没有记录。"]
    emit(args, {"date": args.date, "rows": rows}, lines)


def cmd_live(src: Source, args) -> None:
    store = find_store(src, args.store)
    live = live_from_app(store["id"], args.port)
    if live:
        lines = [f"{store['name']} 实时（本机应用 {live.get('observed_at', '')}）：{live.get('store_status')}，叫到 {live.get('called_no') or '—'} 号，"
                 f"前面 {live.get('wait_groups')} 桌，官方预估 {live.get('server_wait_minutes')} 分钟"
                 + (f"，近 15 分钟叫了 {live['called_15m']} 个号" if live.get("called_15m") is not None else "")]
        emit(args, {"source": "app", "live": live}, lines)
        return
    last = src.latest().get(store["id"])
    if not last:
        emit(args, {"source": None}, ["拿不到实时数据：本机应用没在运行（默认端口 39871），公开数据里也没有这家店的最近状态。"])
        return
    lines = [f"本机应用没在运行，以下是线上最后一次采集（{last['collected_at'][:16].replace('T', ' ')}，不是实时）：",
             f"{last['store_status']}，叫到 {last['display_called_no'] or '—'} 号，前面 {last['group_queues_count']} 桌，官方预估 {last['wait_minutes']} 分钟"]
    emit(args, {"source": "latest", "latest": last}, lines)


def cmd_plan(src: Source, args) -> None:
    meta, store = src.meta(), find_store(src, args.store)
    day = resolve_day(meta, args.date)
    pat = day_patterns(src.daily(store["id"]), day, meta.get("calendar") or {})
    plan = build_plan(pat, args.eat, args.travel)
    notes = freshness(meta, now_cst())
    if not plan["ok"]:
        emit(args, {"store": store, "plan": plan, "notes": notes}, [plan["message"]] + notes)
        return
    wait = f"{plan['wait'][0]}–{plan['wait'][1]} 分钟" if plan["wait"][1] < WAIT_CAP else f"{min(plan['wait'][0], WAIT_CAP)} 分钟到 3 小时以上"
    lines = [f"{store['name']} · {day} 想 {args.eat} 吃" + (f"（路上 {args.travel} 分钟）" if args.travel else ""),
             f"建议 {plan['range'][0]}–{plan['range'][1]} 取号；最贴近的是 {plan['pickup']}，那时通常要等 {wait}（{plan['wait_days']} 天记录）。"]
    if plan["wait"][1] >= WAIT_CAP:
        lines.append("等待常到官方上限 3 小时，按上限倒推只是“最晚”参考，宁可更早取号。")
    if plan["latest_safe"] is None:
        lines.append("八成日子都会晚于目标时间，想稳妥就再早一些或换个时段。")
    elif plan["wait_days"] < FEW_DAYS:
        lines.append("这个时段记录很少，只能当参考。")
    live = live_from_app(store["id"], args.port) if not args.no_live else None
    if live and live.get("server_wait_minutes") is not None:
        lines.append(f"现在官方预估 {live['server_wait_minutes']} 分钟、前面 {live.get('wait_groups')} 桌；取号前以实时为准。")
    lines += notes
    emit(args, {"store": store, "date": day, "plan": plan, "live": live, "notes": notes}, lines)


def cmd_eta(src: Source, args) -> None:
    meta, store = src.meta(), find_store(src, args.store)
    day = resolve_day(meta, args.date)
    pat = day_patterns(src.daily(store["id"]), day, meta.get("calendar") or {})
    eta = build_eta(pat, args.ticket, args.after)
    def say(v):
        if not v:
            return "按历史当天叫不到"
        when = f"{v['time']} 之前就已叫过" if v["already"] else f"{v['time']} 前后"
        return when + f"（{v['days']} 天记录" + ("，太少只能参考）" if v["days"] < FEW_DAYS else "）")
    lines = [f"{store['name']} · {day} · {args.ticket} 号",
             f"快的日子（约两成）：{say(eta['fast'])}", f"一般（一半日子）：{say(eta['typical'])}", f"慢的日子（八成日子之前）：{say(eta['slow'])}"]
    live = live_from_app(store["id"], args.port) if not args.no_live else None
    if live and live.get("called_no"):
        left = args.ticket - live["called_no"]
        line = f"现在叫到 {live['called_no']} 号，还差 {left} 个号" if left > 0 else f"现在叫到 {live['called_no']} 号，已经过了 {args.ticket} 号"
        if left > 0 and live.get("rate_per_min"):
            line += f"，按本机均速约 {round(left / live['rate_per_min'])} 分钟"
        lines.append(line + "。")
    lines.append("叫号是号码不是人数，会因桌型、过号、批量叫号跳动；只能给区间。")
    lines += freshness(meta, now_cst())
    emit(args, {"store": store, "date": day, "eta": eta, "live": live}, lines)


def main(argv=None) -> int:
    p = argparse.ArgumentParser(prog="sushiro_queue", description="寿司郎排队规律、实时状态与取号时间建议（只读）")
    p.add_argument("--json", action="store_true", help="输出 JSON")
    p.add_argument("--port", type=int, help="本机应用端口，默认 39871 或 SUSHIRO_APP_PORT")
    sub = p.add_subparsers(dest="cmd", required=True)
    sub.add_parser("status", help="数据范围、新鲜度、采集健康")
    s = sub.add_parser("stores", help="按名称/城市/区域/地址搜门店"); s.add_argument("query", nargs="?"); s.add_argument("--limit", type=int, default=20)
    s = sub.add_parser("pattern", help="半小时规律")
    s.add_argument("store"); s.add_argument("--type", default="auto", choices=["auto", "all", *DATE_TYPES]); s.add_argument("--date")
    s.add_argument("--recent", type=int, help="只用截止前最近 N 天"); s.add_argument("--from", dest="start", default="00:00"); s.add_argument("--to", dest="end", default="23:59")
    s = sub.add_parser("day", help="某天的半小时记录"); s.add_argument("store"); s.add_argument("date")
    s = sub.add_parser("live", help="实时状态（本机应用），否则最近一次采集"); s.add_argument("store")
    s = sub.add_parser("plan", help="想几点吃 → 几点取号"); s.add_argument("store"); s.add_argument("--eat", required=True)
    s.add_argument("--travel", type=int, default=0); s.add_argument("--date"); s.add_argument("--no-live", action="store_true")
    s = sub.add_parser("eta", help="手里的号大概几点叫到"); s.add_argument("store"); s.add_argument("--ticket", type=int, required=True)
    s.add_argument("--date"); s.add_argument("--after", default="00:00", help="只看这个时间之后"); s.add_argument("--no-live", action="store_true")
    args = p.parse_args(argv)
    try:
        {"status": cmd_status, "stores": cmd_stores, "pattern": cmd_pattern, "day": cmd_day,
         "live": cmd_live, "plan": cmd_plan, "eta": cmd_eta}[args.cmd](Source(), args)
    except DataError as e:
        print(f"错误：{e}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
