"""Read queue information through the local desktop API, never a database."""
from __future__ import annotations


def list_stores(desktop, city=None, q=None, limit=20):
    result = desktop.get("/api/queue/stores", {
        "city": city or "", "q": q or "", "limit": max(1, min(int(limit), 100)),
    })
    if not result.get("ok"):
        return result
    return result.get("stores", [])


def store_queue_history(desktop, store_id, date_type="weekday"):
    return desktop.get("/api/queue/dashboard", {"store": str(store_id), "date_type": date_type})


def store_pressure(desktop, store_id, date_type="weekday"):
    return desktop.get("/api/queue/trends", {"store": str(store_id), "date_type": date_type})


def called_speed(desktop, store_id):
    return desktop.get("/api/queue/live", {"store": str(store_id)})


def compare_stores(desktop, store_ids, date_type="weekday", time_bucket=None):
    # Keep the backend's sample/confidence metadata intact rather than inventing averages.
    params = {"stores": ",".join(str(int(s)) for s in store_ids), "date_type": date_type}
    if not params["stores"]:
        return {"ok": False, "hint": "请至少选择一家门店"}
    if time_bucket:
        compact = time_bucket.replace(":", "")
        if len(compact) != 4 or not compact.isdigit():
            return {"ok": False, "hint": "时段需使用 HH:MM 格式"}
        hour, minute = int(compact[:2]), int(compact[2:])
        if hour > 23 or minute > 59:
            return {"ok": False, "hint": "时段需使用有效的 HH:MM 时间"}
        end = (hour * 60 + minute + 30) % 1440
        params["start"] = compact
        params["end"] = f"{end // 60:02d}{end % 60:02d}"
    return desktop.get("/api/queue/trends", params)
