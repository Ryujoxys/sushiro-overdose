import gzip
import json
import sqlite3
import unittest
from unittest import mock

from collector.collector import (
    _select_snapshots,
    _upsert_store_dimension,
    _upsert_store_latest,
    _write_records,
)
from collector.migrate_v2 import copy_records, drop_legacy
from collector.models import Snapshot, StoreInfo
from collector.runner import prune, retention_days
from collector.schema import SCHEMA_STATEMENTS
from collector.turso import TursoClient, TursoError


class SQLiteTurso:
    def __init__(self, statements=SCHEMA_STATEMENTS):
        self.conn = sqlite3.connect(":memory:", isolation_level=None)
        self.conn.row_factory = sqlite3.Row
        for statement in statements:
            self.conn.execute(statement)

    def __del__(self):
        self.conn.close()

    def execute(self, sql, params=None, *, retry_safe=None):
        cursor = self.conn.execute(sql, params or ())
        return [dict(row) for row in cursor.fetchall()]

    def execute_many(self, statements, *, retry_safe=None):
        rows = []
        for statement in statements:
            if isinstance(statement, str):
                sql, params = statement, ()
            else:
                sql = statement[0]
                params = statement[1] if len(statement) > 1 else ()
            cursor = self.conn.execute(sql, params or ())
            rows = [dict(row) for row in cursor.fetchall()]
        return rows


def frame(at, store_id=1, wait=30, called=None, status="OPEN", source="store_detail"):
    return Snapshot(
        collected_at=at, store_id=store_id, store_status=status, online_open=1,
        wait_minutes=wait, group_queues_count=5, display_called_no=called, dq_source=source,
    )


class SnapshotSelectionTests(unittest.TestCase):
    def test_detail_replaces_list_snapshot_and_list_is_fallback(self):
        list_stores = [
            StoreInfo(store_id=1, name="one", city="Beijing", wait_minutes=10),
            StoreInfo(store_id=2, name="two", city="Beijing", wait_minutes=20),
        ]
        detail = StoreInfo(
            store_id=1,
            name="one",
            wait_minutes=12,
            group_queues={"boothQueue": ["88"]},
        )

        snapshots, stores = _select_snapshots(
            list_stores,
            {1: detail, 2: None},
            "2026-07-27T19:45:00+08:00",
            "2026-07-27T19:45:05+08:00",
        )

        self.assertEqual(2, len(snapshots))
        by_id = {row.store_id: row for row in snapshots}
        self.assertEqual("store_detail", by_id[1].dq_source)
        self.assertEqual(88, by_id[1].display_called_no)
        self.assertEqual("Beijing", stores[1].city)
        self.assertEqual("stores_list", by_id[2].dq_source)
        self.assertIsNone(by_id[2].display_called_no)

    def test_unknown_values_stay_none_and_zero_called_is_none(self):
        detail = StoreInfo.from_api({"storeId": 3, "storeStatus": "open", "groupQueues": {"boothQueue": []}})
        snap = Snapshot.from_store(detail, "2026-07-27T19:29:59+08:00", "store_detail", True)
        self.assertIsNone(snap.wait_minutes)
        self.assertIsNone(snap.group_queues_count)
        self.assertIsNone(snap.display_called_no)
        self.assertEqual("OPEN", snap.store_status)
        self.assertEqual("2026-07-27T19:00", snap.slot)
        self.assertEqual(0, StoreInfo.from_api({"storeId": 3, "wait": 0}).wait_minutes)


class RecordWriteTests(unittest.TestCase):
    def setUp(self):
        self.db = SQLiteTurso()

    def tearDown(self):
        self.db.conn.close()

    def rows(self):
        return self.db.execute("SELECT * FROM queue_records ORDER BY slot, store_id")

    def test_last_frame_in_half_hour_wins_and_late_old_frame_is_ignored(self):
        _write_records(self.db, [frame("2026-07-27T19:30:05+08:00", called=100)])
        _write_records(self.db, [frame("2026-07-27T19:45:05+08:00", wait=40, called=None, source="stores_list")])
        _write_records(self.db, [frame("2026-07-27T19:31:00+08:00", wait=99, called=1)])
        _write_records(self.db, [frame("2026-07-27T20:00:05+08:00", wait=0, called=120)])

        rows = self.rows()
        self.assertEqual(["2026-07-27T19:30", "2026-07-27T20:00"], [r["slot"] for r in rows])
        self.assertEqual(("2026-07-27T19:45:05+08:00", 40, None), (rows[0]["collected_at"], rows[0]["wait_minutes"], rows[0]["display_called_no"]))
        self.assertEqual(0, rows[1]["wait_minutes"], "zero wait is a real observation")

    def test_store_latest_uses_v2_columns(self):
        snaps = [frame("2026-07-27T19:45:05+08:00", called=88)]
        _upsert_store_latest(self.db, snaps, {1: "ONLINE"})
        _upsert_store_latest(self.db, [frame("2026-07-27T20:00:05+08:00", wait=None)], {1: ""})
        row = self.db.execute("SELECT * FROM store_latest")[0]
        self.assertEqual("2026-07-27T20:00:05+08:00", row["collected_at"])
        self.assertIsNone(row["wait_minutes"])
        self.assertIsNone(row["net_ticket_status"])

    def test_dimension_only_refreshes_when_changed_or_on_new_day(self):
        store = StoreInfo(store_id=1, name="one", city="Beijing")
        _upsert_store_dimension(self.db, [store], "2026-07-27T10:00:00+08:00")
        _upsert_store_dimension(self.db, [store], "2026-07-27T10:15:00+08:00")
        row = self.db.execute("SELECT last_seen_at FROM store_dimension WHERE store_id = 1")[0]
        self.assertEqual("2026-07-27T10:00:00+08:00", row["last_seen_at"])

        _upsert_store_dimension(self.db, [store], "2026-07-28T10:00:00+08:00")
        row = self.db.execute("SELECT last_seen_at FROM store_dimension WHERE store_id = 1")[0]
        self.assertEqual("2026-07-28T10:00:00+08:00", row["last_seen_at"])


class RetentionTests(unittest.TestCase):
    def test_v1_archive_setting_is_not_applied_to_records(self):
        self.assertEqual((0, 30), retention_days({"archive": {"retention_days": 60}}))
        self.assertEqual((400, 7), retention_days({"retention": {"records_days": 400, "runs_days": 7}}))
        self.assertEqual((0, 1), retention_days({"retention": {"records_days": "x", "runs_days": -1}}))

    def test_prune_keeps_records_by_default_and_trims_by_slot(self):
        db = SQLiteTurso()
        _write_records(db, [frame("2026-01-01T12:00:00+08:00"), frame("2026-07-27T12:00:00+08:00")])
        self.assertEqual(0, prune(db, 0, 30, "2026-07-28"))
        self.assertEqual(2, db.execute("SELECT COUNT(*) AS c FROM queue_records")[0]["c"])
        self.assertEqual(1, prune(db, 100, 30, "2026-07-28"))
        self.assertEqual(["2026-07-27T12:00"], [r["slot"] for r in db.execute("SELECT slot FROM queue_records")])


LEGACY = [
    """CREATE TABLE store_latest (store_id INTEGER PRIMARY KEY, collected_at TEXT NOT NULL, name TEXT,
       wait_minutes INTEGER NOT NULL DEFAULT 0, group_queues_count INTEGER NOT NULL DEFAULT 0,
       store_status TEXT, net_ticket_status TEXT, online_open INTEGER NOT NULL DEFAULT 0,
       display_called_no INTEGER, group_queues_json TEXT, updated_at TEXT NOT NULL)""",
    """CREATE TABLE queue_snapshots (id INTEGER PRIMARY KEY AUTOINCREMENT, collected_at TEXT NOT NULL,
       store_id INTEGER NOT NULL, wait_minutes INTEGER NOT NULL DEFAULT 0,
       group_queues_count INTEGER NOT NULL DEFAULT 0, store_status TEXT, online_open INTEGER NOT NULL DEFAULT 0,
       display_called_no INTEGER, dq_source TEXT NOT NULL)""",
    """CREATE TABLE daily_store_bucket_rollups (snapshot_date TEXT NOT NULL, store_id INTEGER NOT NULL,
       time_bucket TEXT, sample_count INTEGER, open_count INTEGER, online_open_count INTEGER,
       wait_typical_minutes REAL, queue_groups_typical REAL, called_sample_count INTEGER,
       called_no_typical REAL, updated_at TEXT NOT NULL, PRIMARY KEY (snapshot_date, store_id, time_bucket))""",
    "CREATE TABLE store_bucket_rollups (store_id INTEGER)",
    "CREATE TABLE called_intervals_rollups (store_id INTEGER)",
    "CREATE TABLE archive_runs (id INTEGER PRIMARY KEY)",
    [s for s in SCHEMA_STATEMENTS if "collector_runs" in s][0],
]


class MigrateV2Tests(unittest.TestCase):
    def setUp(self):
        self.db = SQLiteTurso(LEGACY)
        daily = "INSERT INTO daily_store_bucket_rollups VALUES (?, 1, ?, 1, ?, 1, ?, 8, 1, ?, ?)"
        # Archived day: only rollup representatives remain.
        self.db.execute(daily, ("2026-07-01", "12:00", 1, 45.0, 300.0, "2026-07-01T12:15:00+08:00"))
        self.db.execute(daily, ("2026-07-01", "12:30", 0, None, 0.0, "2026-07-01T12:45:00+08:00"))
        # Raw day: the rollup row is overridden by the last raw frame of the slot.
        self.db.execute(daily, ("2026-07-10", "19:00", 1, 60.0, 500.0, "2026-07-10T19:15:05+08:00"))
        raw = "INSERT INTO queue_snapshots (collected_at, store_id, wait_minutes, group_queues_count, store_status, online_open, display_called_no, dq_source) VALUES (?, 1, ?, 9, 'open', 1, ?, ?)"
        self.db.execute(raw, ("2026-07-10T19:00:05+08:00", 50, 480, "store_detail"))
        self.db.execute(raw, ("2026-07-10T19:15:05+08:00", 240, 0, "store_detail"))
        self.db.execute(raw, ("2026-07-10T19:15:05+08:00", 240, None, "stores_list"))
        self.db.execute("INSERT INTO store_latest (store_id, collected_at, name, wait_minutes, store_status, display_called_no, group_queues_json, updated_at) VALUES (1, '2026-07-10T19:15:05+08:00', 'one', 240, 'OPEN', 0, '{}', 'x')")

    def test_copy_is_idempotent_and_keeps_semantics(self):
        first = copy_records(self.db)
        second = copy_records(self.db)
        self.assertEqual(first, second)
        rows = {r["slot"]: r for r in self.db.execute("SELECT * FROM queue_records")}
        self.assertEqual(3, len(rows))
        self.assertEqual((45, 8, 300, "OPEN"), tuple(rows["2026-07-01T12:00"][k] for k in ("wait_minutes", "group_queues_count", "display_called_no", "store_status")))
        self.assertEqual(("CLOSED", None, None), tuple(rows["2026-07-01T12:30"][k] for k in ("store_status", "wait_minutes", "display_called_no")))
        raw = rows["2026-07-10T19:00"]
        self.assertEqual(("2026-07-10T19:15:05+08:00", 240, None, 9, "OPEN"), (raw["collected_at"], raw["wait_minutes"], raw["display_called_no"], raw["group_queues_count"], raw["store_status"]))

    def test_drop_requires_copy_then_removes_legacy_and_rebuilds_latest(self):
        with self.assertRaises(RuntimeError):
            drop_legacy(self.db)
        copy_records(self.db)
        drop_legacy(self.db)
        tables = {r["name"] for r in self.db.execute("SELECT name FROM sqlite_master WHERE type='table'")}
        self.assertEqual({"queue_records", "store_latest", "collector_runs"}, tables)
        latest = self.db.execute("SELECT * FROM store_latest")[0]
        self.assertEqual(["store_id", "collected_at", "store_status", "net_ticket_status", "online_open", "wait_minutes", "group_queues_count", "display_called_no"], list(latest))
        self.assertIsNone(latest["display_called_no"])
        drop_legacy(self.db)  # second run is a no-op

    def test_drop_refuses_when_frames_were_not_copied(self):
        self.db.execute("CREATE TABLE IF NOT EXISTS x (y)")
        copy_records(self.db)
        # the old collector kept writing after the copy
        self.db.execute("INSERT INTO queue_snapshots (collected_at, store_id, wait_minutes, store_status, dq_source) VALUES ('2026-07-10T19:25:00+08:00', 1, 5, 'OPEN', 'store_detail')")
        with self.assertRaises(RuntimeError):
            drop_legacy(self.db)
        self.assertIn("queue_snapshots", {r["name"] for r in self.db.execute("SELECT name FROM sqlite_master")})
        copy_records(self.db)
        drop_legacy(self.db)

    def test_copy_does_not_replace_a_newer_live_frame(self):
        copy_records(self.db)
        self.db.execute("UPDATE queue_records SET collected_at = '2026-07-10T19:29:00+08:00', wait_minutes = 1 WHERE slot = '2026-07-10T19:00'")
        copy_records(self.db)
        self.assertEqual(1, self.db.execute("SELECT wait_minutes FROM queue_records WHERE slot = '2026-07-10T19:00'")[0]["wait_minutes"])

    def test_collector_refuses_v1_schema(self):
        from collector import collector as coll

        coll._schema_checked = False
        with self.assertRaises(RuntimeError):
            coll.ensure_v2_schema(self.db)
        copy_records(self.db)
        with self.assertRaises(RuntimeError):
            coll.ensure_v2_schema(self.db)  # store_latest is still v1
        drop_legacy(self.db)
        coll.ensure_v2_schema(self.db)
        coll._schema_checked = False


class RetrySafetyTests(unittest.TestCase):
    def setUp(self):
        self.client = TursoClient("libsql://example.turso.io", "token")

    @mock.patch("time.sleep")
    @mock.patch("collector.turso.urllib.request.urlopen")
    def test_write_is_not_retried_by_default(self, urlopen, _sleep):
        urlopen.side_effect = OSError("connection reset")

        with self.assertRaises(TursoError):
            self.client.execute("INSERT INTO sample(value) VALUES (1)")

        self.assertEqual(1, urlopen.call_count)

    @mock.patch("time.sleep")
    @mock.patch("collector.turso.urllib.request.urlopen")
    def test_select_is_retried(self, urlopen, _sleep):
        urlopen.side_effect = OSError("connection reset")

        with self.assertRaises(TursoError):
            self.client.execute("SELECT 1")

        self.assertEqual(5, urlopen.call_count)

    @mock.patch("collector.turso.urllib.request.urlopen")
    def test_gzip_response_is_decoded(self, urlopen):
        payload = {
            "results": [
                {
                    "type": "ok",
                    "response": {
                        "result": {
                            "cols": [{"name": "value"}],
                            "rows": [[{"type": "integer", "value": "1"}]],
                        }
                    },
                }
            ]
        }

        class Response:
            headers = {"content-encoding": "gzip"}

            def __enter__(self):
                return self

            def __exit__(self, *_args):
                return False

            def read(self):
                return gzip.compress(json.dumps(payload).encode("utf-8"))

        urlopen.return_value = Response()

        rows = self.client.execute("SELECT 1")

        self.assertEqual([{"value": "1"}], rows)


class RetireMissingStoresTests(unittest.TestCase):
    def _db_with_stores(self, ids):
        db = SQLiteTurso()
        for sid in ids:
            db.execute(
                "INSERT INTO store_dimension (store_id, name, first_seen_at, last_seen_at, is_active)"
                " VALUES (?, ?, '2026-08-01T10:00:00+08:00', '2026-08-13T10:00:00+08:00', 1)",
                (sid, f"store-{sid}"),
            )
        return db

    def test_missing_store_is_retired(self):
        from collector.collector import _retire_missing_stores

        db = self._db_with_stores(list(range(1, 101)) + [999])
        retired = _retire_missing_stores(db, list(range(1, 101)))  # 999 没出现
        self.assertEqual(retired, 1)
        rows = db.execute("SELECT is_active FROM store_dimension WHERE store_id = 999")
        self.assertEqual(rows[0]["is_active"], 0)

    def test_seen_stores_stay_active(self):
        from collector.collector import _retire_missing_stores

        db = self._db_with_stores(list(range(1, 101)))
        retired = _retire_missing_stores(db, list(range(1, 101)))
        self.assertEqual(retired, 0)
        rows = db.execute("SELECT COUNT(*) AS c FROM store_dimension WHERE is_active = 1")
        self.assertEqual(rows[0]["c"], 100)

    def test_truncated_list_skips_retirement(self):
        # 上游截断（<30 家）不许触发批量下线
        from collector.collector import _retire_missing_stores

        db = self._db_with_stores(list(range(1, 101)))
        retired = _retire_missing_stores(db, [1, 2, 3])
        self.assertEqual(retired, 0)
        rows = db.execute("SELECT COUNT(*) AS c FROM store_dimension WHERE is_active = 1")
        self.assertEqual(rows[0]["c"], 100)

    def test_retired_store_revives_on_upsert(self):
        # 回归列表时，_upsert_store_dimension 的条件分支应把它复活
        from collector.collector import _retire_missing_stores
        from datetime import datetime, timedelta, timezone

        db = self._db_with_stores(list(range(1, 101)) + [999])
        _retire_missing_stores(db, list(range(1, 101)))
        now = datetime.now(timezone(timedelta(hours=8))).isoformat(timespec="seconds")
        _upsert_store_dimension(
            db,
            [StoreInfo(store_id=999, name="reborn", wait_minutes=5)],
            now,
        )
        rows = db.execute("SELECT is_active, name FROM store_dimension WHERE store_id = 999")
        self.assertEqual(rows[0]["is_active"], 1)
        self.assertEqual(rows[0]["name"], "reborn")


if __name__ == "__main__":
    unittest.main()
