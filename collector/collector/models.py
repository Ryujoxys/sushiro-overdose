"""数据模型（dataclass）。对应寿司郎接口返回 + Turso queue_records 行。"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, Optional


@dataclass
class StoreInfo:
    """寿司郎接口返回的单店信息（stores? 和 getStoreById? 共有字段的超集）。"""

    store_id: int
    name: str = ""
    name_kana: str = ""
    city: str = ""
    area: str = ""
    address: str = ""
    latitude: Optional[float] = None
    longitude: Optional[float] = None
    distance: str = ""
    store_status: str = ""          # OPEN / CLOSED
    net_ticket_status: str = ""     # ONLINE / OFFLINE_*
    wait_minutes: Optional[int] = None   # 接口没给就是未知，不当 0
    wait_time_counter: Optional[int] = None
    wait_time_cap: Optional[int] = None
    tables_capacity: int = 0
    counters_capacity: int = 0
    group_queues_count: Optional[int] = None
    reservation_status: str = ""
    open_date: str = ""
    # group_queues 仅 getStoreById 返回
    group_queues: Optional[Dict[str, Any]] = None

    @property
    def online_open(self) -> bool:
        s = (self.net_ticket_status or "").upper()
        if "CLOSED" in s or "OFFLINE" in s:
            return False
        if s in ("ONLINE", "ON") or "OPEN" in s:
            return True
        return False

    @classmethod
    def from_api(cls, d: Dict[str, Any]) -> "StoreInfo":
        """从接口 JSON（驼峰键）构造。兼容 id / storeId 两种主键名。"""
        sid = d.get("storeId")
        if sid is None:
            sid = d.get("id")
        try:
            store_id = int(sid) if sid is not None else 0
        except (TypeError, ValueError):
            store_id = 0

        lat = d.get("latitude")
        lng = d.get("longitude")

        def _int(key: str) -> int:
            v = d.get(key)
            try:
                return int(v) if v is not None else 0
            except (TypeError, ValueError):
                return 0

        def _oint(key: str) -> Optional[int]:
            v = d.get(key)
            if v is None or v == "":
                return None
            try:
                return int(v)
            except (TypeError, ValueError):
                return None

        return cls(
            store_id=store_id,
            name=str(d.get("name") or ""),
            name_kana=str(d.get("nameKana") or ""),
            city=str(d.get("city") or ""),
            area=str(d.get("area") or ""),
            address=str(d.get("address") or ""),
            latitude=float(lat) if lat not in (None, "") else None,
            longitude=float(lng) if lng not in (None, "") else None,
            distance=str(d.get("distance") or ""),
            store_status=str(d.get("storeStatus") or ""),
            net_ticket_status=str(d.get("netTicketStatus") or ""),
            wait_minutes=_oint("wait"),
            wait_time_counter=_oint("waitTimeCounter"),
            wait_time_cap=_oint("waitTimeCap"),
            tables_capacity=_int("tablesCapacity"),
            counters_capacity=_int("countersCapacity"),
            group_queues_count=_oint("groupQueuesCount"),
            reservation_status=str(d.get("reservationStatus") or ""),
            open_date=str(d.get("openDate") or ""),
            group_queues=d.get("groupQueues") if isinstance(d.get("groupQueues"), dict) else None,
        )


@dataclass
class Snapshot:
    """一帧公开状态，写入 queue_records 的一行（同一半小时内后写的覆盖先写的）。

    缺失一律 None：display_called_no 只在详情接口给出大于 0 的堂食叫号时有值。
    dq_source 只在内存里区分详情帧/列表兜底帧，不入库。
    """

    collected_at: str
    store_id: int
    store_status: str = "UNKNOWN"
    online_open: int = 0
    wait_minutes: Optional[int] = None
    group_queues_count: Optional[int] = None
    display_called_no: Optional[int] = None
    dq_source: str = "stores_list"           # 'stores_list' | 'store_detail'

    @property
    def slot(self) -> str:
        """门店时区半小时起点 'YYYY-MM-DDTHH:MM'。collected_at 已是 +08:00 文本。"""
        minute = "00" if int(self.collected_at[14:16]) < 30 else "30"
        return self.collected_at[:14] + minute

    @classmethod
    def from_store(
        cls,
        store: StoreInfo,
        collected_at: str,
        dq_source: str,
        include_called: bool,
    ) -> "Snapshot":
        called_no: Optional[int] = None
        if include_called:
            from .called_no import current_called_no
            called_no = current_called_no(store.group_queues) or None

        def _non_negative(v: Optional[int]) -> Optional[int]:
            return v if v is not None and v >= 0 else None

        return cls(
            collected_at=collected_at,
            store_id=store.store_id,
            store_status=(store.store_status or "UNKNOWN").upper(),
            online_open=1 if store.online_open else 0,
            wait_minutes=_non_negative(store.wait_minutes),
            group_queues_count=_non_negative(store.group_queues_count),
            display_called_no=called_no,
            dq_source=dq_source,
        )
