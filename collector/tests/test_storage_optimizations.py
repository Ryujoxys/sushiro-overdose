import gzip
import json
import sqlite3
import unittest
from unittest import mock

from collector import aggregator
from collector.aggregator import (
    aggregate_all,
    _load_snapshot_rows,
    _select_daily_writes,
    _write_daily_rollups,
)
from collector.archive import archive_old
from collector.collector import (
    _select_snapshots,
    _upsert_store_dimension,
)
from collector.models import StoreInfo
from collector.schema import SCHEMA_STATEMENTS
from collector.turso import TursoClient, TursoError


class SQLiteTurso:
    def __init__(self):
        self.conn = sqlite3.connect(":memory:")
        self.conn.row_factory = sqlite3.Row
        for statement in SCHEMA_STATEMENTS:
            self.conn.execute(statement)

    def execute(self, sql, params=None, *, retry_safe=None):
        cursor = self.conn.execute(sql, params or ())
        self.conn.commit()
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
        self.conn.commit()
        return rows


def daily_row(snapshot_date, store_id=1, bucket="19:30", updated_at=None):
    return {
        "snapshot_date": snapshot_date,
        "store_id": store_id,
        "date_type": "weekday",
        "weekday": 0,
        "time_bucket": bucket,
        "sample_count": 1,
        "open_count": 1,
        "online_open_count": 1,
        "busy_count": 1,
        "open_rate": 1.0,
        "online_open_rate": 1.0,
        "busy_rate": 1.0,
        "wait_typical_minutes": 30.0,
        "wait_safe_minutes": 30.0,
        "wait_max_minutes": 30,
        "queue_groups_typical": 8.0,
        "queue_groups_safe": 8.0,
        "called_sample_count": 1,
        "called_no_slow": 100,
        "called_no_typical": 100,
        "called_no_fast": 100,
        "confidence": "low",
        "updated_at": updated_at or f"{snapshot_date}T19:45:00+08:00",
    }


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


class SQLiteWriteTests(unittest.TestCase):
    def setUp(self):
        self.db = SQLiteTurso()

    def tearDown(self):
        self.db.conn.close()

    def test_dimension_only_refreshes_when_changed_or_on_new_day(self):
        store = StoreInfo(store_id=1, name="one", city="Beijing")
        _upsert_store_dimension(
            self.db, [store], "2026-07-27T10:00:00+08:00"
        )
        _upsert_store_dimension(
            self.db, [store], "2026-07-27T10:15:00+08:00"
        )
        row = self.db.execute(
            "SELECT last_seen_at FROM store_dimension WHERE store_id = 1"
        )[0]
        self.assertEqual("2026-07-27T10:00:00+08:00", row["last_seen_at"])

        _upsert_store_dimension(
            self.db, [store], "2026-07-28T10:00:00+08:00"
        )
        row = self.db.execute(
            "SELECT last_seen_at FROM store_dimension WHERE store_id = 1"
        )[0]
        self.assertEqual("2026-07-28T10:00:00+08:00", row["last_seen_at"])

    def test_daily_upsert_skips_equal_values_and_updates_all_changed_fields(self):
        first = daily_row("2026-07-27")
        _write_daily_rollups(self.db, [first])

        same = daily_row(
            "2026-07-27", updated_at="2026-07-28T02:00:00+08:00"
        )
        _write_daily_rollups(self.db, [same])
        row = self.db.execute(
            "SELECT updated_at FROM daily_store_bucket_rollups"
        )[0]
        self.assertEqual(first["updated_at"], row["updated_at"])

        changed = dict(same)
        changed["wait_safe_minutes"] = 45.0
        changed["called_no_fast"] = 120
        _write_daily_rollups(self.db, [changed])
        row = self.db.execute(
            "SELECT wait_safe_minutes, called_no_fast, updated_at "
            "FROM daily_store_bucket_rollups"
        )[0]
        self.assertEqual(45.0, row["wait_safe_minutes"])
        self.assertEqual(120.0, row["called_no_fast"])
        self.assertEqual(changed["updated_at"], row["updated_at"])

    def test_incremental_daily_refreshes_latest_and_incomplete_dates(self):
        existing = [
            daily_row("2026-07-25", store_id=1),
            daily_row("2026-07-25", store_id=2),
            daily_row("2026-07-26", store_id=1),
        ]
        _write_daily_rollups(self.db, existing)
        computed = [
            daily_row("2026-07-25", store_id=1),
            daily_row("2026-07-25", store_id=2),
            daily_row("2026-07-26", store_id=1),
            daily_row("2026-07-26", store_id=2),
            daily_row("2026-07-27", store_id=1),
            daily_row("2026-07-27", store_id=2),
        ]

        selected = _select_daily_writes(self.db, computed)

        self.assertEqual(
            {"2026-07-26", "2026-07-27"},
            {row["snapshot_date"] for row in selected},
        )

    def test_snapshot_reader_uses_keyset_pages_without_losing_rows(self):
        sql = """
        INSERT INTO queue_snapshots
          (collected_at, store_id, dq_source)
        VALUES (?, ?, ?)
        """
        for i in range(12):
            self.db.execute(
                sql,
                (
                    f"2026-07-27T10:{i:02d}:00+08:00",
                    i + 1,
                    "store_detail",
                ),
            )

        with mock.patch.object(aggregator, "READ_PAGE", 5):
            rows = _load_snapshot_rows(self.db, days=None)

        self.assertEqual(12, len(rows))
        self.assertEqual(list(range(1, 13)), [row["id"] for row in rows])

    def test_full_aggregate_executes_all_rollup_upserts(self):
        _upsert_store_dimension(
            self.db,
            [StoreInfo(store_id=1, name="one", tables_capacity=20)],
            "2026-07-27T10:00:00+08:00",
        )
        sql = """
        INSERT INTO queue_snapshots
          (collected_at, store_id, wait_minutes, group_queues_count,
           store_status, online_open, wait_time_cap, display_called_no,
           dq_source)
        VALUES (?, 1, ?, ?, 'OPEN', 1, 180, ?, 'store_detail')
        """
        self.db.execute(
            sql, ("2026-07-27T10:00:00+08:00", 20, 5, 10)
        )
        self.db.execute(
            sql, ("2026-07-27T10:15:00+08:00", 25, 7, 12)
        )

        stats = aggregate_all(self.db, days=30)

        self.assertEqual(
            {"rollups": 1, "daily": 1, "intervals": 1},
            stats,
        )
        self.assertEqual(
            1,
            self.db.execute(
                "SELECT COUNT(*) AS c FROM called_intervals_rollups"
            )[0]["c"],
        )

    def test_archive_skips_full_remaining_count_and_log_is_idempotent(self):
        sql = """
        INSERT INTO queue_snapshots
          (collected_at, store_id, dq_source)
        VALUES (?, ?, 'store_detail')
        """
        self.db.execute(sql, ("2026-05-01T10:00:00+08:00", 1))
        self.db.execute(sql, ("2026-07-27T10:00:00+08:00", 1))

        first = archive_old(self.db, retention_days=30)
        second = archive_old(self.db, retention_days=30)

        self.assertEqual({"deleted": 1, "remaining": -1}, first)
        self.assertEqual({"deleted": 0, "remaining": -1}, second)
        self.assertEqual(
            1,
            self.db.execute(
                "SELECT COUNT(*) AS c FROM queue_snapshots"
            )[0]["c"],
        )
        self.assertEqual(
            1,
            self.db.execute("SELECT COUNT(*) AS c FROM archive_runs")[0]["c"],
        )


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


if __name__ == "__main__":
    unittest.main()


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
