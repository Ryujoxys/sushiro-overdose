"""Reuse desktop predictions and their uncertainty, without cloud enrichment."""
from __future__ import annotations


def arrival_advice(desktop, store_id, target_no=None, want_meal_time=None, travel_minutes=None, called_no=None, called_at=None):
    params = {"store": str(store_id)}
    if travel_minutes is not None:
        params["travel_minutes"] = str(max(0, int(travel_minutes)))
    # 没有实时连接时，用户在小程序里看到的“现在叫到几号”可让应用按以往叫号速度推算。
    if called_no is not None:
        params["called_no"] = str(int(called_no))
        if called_at:
            params["called_at"] = str(called_at).replace(":", "")
    if target_no is not None:
        params["target_no"] = str(int(target_no))
        return desktop.get("/api/queue/advisor", params)
    if want_meal_time:
        params["target_meal"] = want_meal_time.replace(":", "")
        return desktop.get("/api/queue/plan", params)
    return desktop.get("/api/queue/advisor", params)
