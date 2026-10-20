import contextlib
import hashlib
import io
import json
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "scripts"))
import sushiro_queue as sq  # noqa: E402


def row(stamp, wait=30, called=500, status="OPEN", date_type="weekday"):
    return {"collected_at": stamp, "date_type": date_type, "store_status": status, "wait_minutes": wait, "display_called_no": called}


def write_dataset(root: Path, rows, calendar=None, cutoff="2026-09-05T21:30:00+08:00"):
    fields = ["collected_at", "date_type", "store_status", "wait_minutes", "display_called_no", "group_queues_count", "online_open"]
    files = {
        "stores.json": [{"id": 1, "name": "东方宝泰店", "city": "", "area": "广州天河区"}, {"id": 2, "name": "正佳广场店", "city": "", "area": "广州天河区"}],
        "daily/1.json": {"store_id": 1, "fields": fields, "rows": [[r[f] if f in r else None for f in fields] for r in rows]},
        "latest.json": {"source": "store_latest", "stores": [{"store_id": 1, "collected_at": cutoff, "store_status": "OPEN", "wait_minutes": 20, "group_queues_count": 4, "display_called_no": 321}]},
    }
    meta = {"schema": sq.SCHEMA, "version": 1, "source": "turso", "first_at": rows[0]["collected_at"], "cutoff_at": cutoff,
            "store_count": 2, "record_count": len(rows), "day_count": 2, "quality_note": "", "calendar": calendar or {},
            "collector": {"recent_runs": 4, "recent_ok": 4, "last_ok_at": cutoff}, "files": {}}
    for name, value in files.items():
        raw = (json.dumps(value, ensure_ascii=False) + "\n").encode()
        (root / name).parent.mkdir(parents=True, exist_ok=True)
        (root / name).write_bytes(raw)
        meta["files"][name] = {"bytes": len(raw), "sha256": hashlib.sha256(raw).hexdigest()}
    (root / "meta.json").write_text(json.dumps(meta))


class StatsTests(unittest.TestCase):
    def test_quantile_matches_app_interpolation(self):
        self.assertEqual(25, sq.quantile([40, 10, 30, 20], .5))
        self.assertAlmostEqual(34, sq.quantile([10, 20, 30, 40], .8))
        self.assertIsNone(sq.quantile([], .5))

    def test_half_hour_uses_china_time(self):
        self.assertEqual("18:30", sq.half_hour("2026-09-04T10:44:00Z"))

    def test_date_type_rules(self):
        cal = {"2026-10-01": "holiday", "2026-10-11": "workday"}
        self.assertEqual("weekday", sq.date_type_at("2026-10-23", "16:00", cal))  # Friday afternoon
        self.assertEqual("weekend", sq.date_type_at("2026-10-23", "16:30", cal))
        self.assertEqual("weekend", sq.date_type_at("2026-10-25", "21:30", cal))  # Sunday
        self.assertEqual("weekday", sq.date_type_at("2026-10-25", "22:00", cal))
        self.assertEqual("holiday", sq.date_type_at("2026-10-01", "12:00", cal))
        self.assertEqual("workday", sq.date_type_at("2026-10-11", "12:00", cal))  # Sunday made a workday

    def test_patterns_skip_closed_and_unknown_and_keep_zero(self):
        p = sq.patterns([row("2026-09-04T18:05:00+08:00"), row("2026-09-05T18:10:00+08:00", wait=0, called=600),
                         row("2026-09-06T18:15:00+08:00", wait=None, called=None), row("2026-09-07T18:20:00+08:00", status="CLOSED", wait=99)])
        b = p["18:00"]
        self.assertEqual((2, 15, 2, 550), (b["wait_days"], b["wait_p50"], b["called_days"], b["called_p50"]))

    def test_friday_switches_date_type_at_1630(self):
        rows = [row("2026-09-03T16:00:00+08:00", wait=10), row("2026-09-03T17:00:00+08:00", wait=10),
                row("2026-09-05T16:00:00+08:00", wait=90, date_type="weekend"), row("2026-09-05T17:00:00+08:00", wait=90, date_type="weekend")]
        p = sq.day_patterns(rows, "2026-10-23", {})
        self.assertEqual(10, p["16:00"]["wait_p50"])
        self.assertEqual(90, p["17:00"]["wait_p50"])

    def test_bundle_parity_with_published_patterns(self):
        src = sq.Source(base="/nonexistent")
        rows = src.daily(1)
        self.assertIn("改用仓库内置历史包", src.fallback_reason)
        b = sq.patterns(rows)["18:00"]
        self.assertEqual([676, 535, 1244, 75], [b["called_p50"], int(b["called_p20"]), -int(-b["called_p80"] // 1), b["called_days"]])
        self.assertEqual([165, 175, 14], [b["wait_p50"], b["wait_p80"], b["wait_days"]])


class PlanTests(unittest.TestCase):
    def bucket(self, t, p50, p80, days=10):
        return {"time": t, "wait_p50": p50, "wait_p80": p80, "wait_days": days, "called_days": days, "called_p20": None, "called_p50": None, "called_p80": None}

    def test_plan_matches_app_algorithm(self):
        pat = {t: self.bucket(t, 60, 90) for t in ("17:00", "17:30", "18:00", "18:30")}
        plan = sq.build_plan(pat, "19:00", travel=10)
        # target 18:50: P50 lands exactly from 17:50; P80 still on time up to 17:20.
        self.assertEqual(("17:50", "17:20", ["17:20", "17:50"], [60, 90]), (plan["pickup"], plan["latest_safe"], plan["range"], plan["wait"]))

    def test_plan_without_history(self):
        self.assertFalse(sq.build_plan({}, "19:00")["ok"])

    def test_eta_bands_and_already_called(self):
        pat = {"17:00": {"called_p20": 700, "called_p50": 900, "called_p80": 1000, "called_days": 9},
               "18:00": {"called_p20": 950, "called_p50": 1100, "called_p80": 1300, "called_days": 9},
               "19:00": {"called_p20": 1200, "called_p50": 1400, "called_p80": 1500, "called_days": 2}}
        eta = sq.build_eta(pat, 1000, after="17:00")
        self.assertEqual(("17:00", True), (eta["fast"]["time"], eta["fast"]["already"]))
        self.assertEqual(("18:00", False), (eta["typical"]["time"], eta["typical"]["already"]))
        self.assertEqual("19:00", eta["slow"]["time"])
        self.assertIsNone(sq.build_eta(pat, 5000)["slow"])


class SourceAndCliTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name) / "data"
        self.cache = Path(self.tmp.name) / "cache"
        write_dataset(self.root, [row("2026-09-04T18:05:00+08:00", wait=40, called=500), row("2026-09-05T18:10:00+08:00", wait=180, called=700, date_type="weekend")])
        self.env = mock.patch.dict("os.environ", {"SUSHIRO_DATA_DIR": str(self.root), "SUSHIRO_DATA_CACHE": str(self.cache), "SUSHIRO_APP_PORT": "1"})
        self.env.start()

    def tearDown(self):
        self.env.stop()
        self.tmp.cleanup()

    def run_cli(self, *argv):
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            code = sq.main(list(argv))
        return code, out.getvalue()

    def test_hash_mismatch_is_rejected(self):
        (self.root / "daily/1.json").write_text("{}")
        with self.assertRaises(sq.DataError):
            sq.Source().daily(1)

    def test_store_lookup(self):
        src = sq.Source()
        self.assertEqual(1, sq.find_store(src, "东方宝泰").get("id"))
        with self.assertRaises(sq.DataError):
            sq.find_store(src, "广州")  # matches two stores
        with self.assertRaises(sq.DataError):
            sq.find_store(src, "99")

    def test_pattern_json_and_cap_display(self):
        code, out = self.run_cli("--json", "pattern", "1", "--type", "all")
        self.assertEqual(0, code)
        data = json.loads(out)
        self.assertEqual([2, 152], [data["buckets"][0]["wait_days"], data["buckets"][0]["wait_p80"]])
        code, out = self.run_cli("pattern", "1", "--type", "weekend")
        self.assertIn("180+", out)

    def test_live_falls_back_to_latest_when_app_is_down(self):
        code, out = self.run_cli("live", "1")
        self.assertEqual(0, code)
        self.assertIn("不是实时", out)
        self.assertIn("321", out)

    def test_status_warns_about_stale_data(self):
        code, out = self.run_cli("status")
        self.assertIn("没有新数据", out)

    def test_cli_never_writes_outside_cache(self):
        before = sorted(p.name for p in self.root.rglob("*"))
        self.run_cli("plan", "1", "--eat", "19:00", "--no-live")
        self.assertEqual(before, sorted(p.name for p in self.root.rglob("*")))


if __name__ == "__main__":
    unittest.main()
