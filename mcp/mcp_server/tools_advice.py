"""Reuse desktop predictions and their uncertainty, without cloud enrichment."""
from __future__ import annotations


def arrival_advice(desktop, store_id, target_no=None, want_meal_time=None, travel_minutes=None):
    params = {"store": str(store_id)}
    if travel_minutes is not None:
        params["travel_minutes"] = str(max(0, int(travel_minutes)))
    if target_no is not None:
        params["target_no"] = str(int(target_no))
        return desktop.get("/api/queue/advisor", params)
    if want_meal_time:
        params["target_meal"] = want_meal_time.replace(":", "")
        return desktop.get("/api/queue/plan", params)
    return desktop.get("/api/queue/advisor", params)
