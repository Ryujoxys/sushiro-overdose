import unittest

from collector.runner import _normalize_collect_cfg, _seconds_until_next_boundary


class NormalizeCollectCfgTests(unittest.TestCase):
    def test_defaults(self):
        interval, hours = _normalize_collect_cfg({})
        self.assertEqual(interval, 900)
        self.assertEqual(hours, [10, 22])

    def test_valid_values_pass_through(self):
        interval, hours = _normalize_collect_cfg(
            {"interval_seconds": 300, "active_hours": [9, 23]}
        )
        self.assertEqual(interval, 300)
        self.assertEqual(hours, [9, 23])

    def test_string_hours_are_coerced(self):
        # 曾因 ["10","22"] 的字符串比较 TypeError 直接崩 run_loop
        _, hours = _normalize_collect_cfg({"active_hours": ["10", "22"]})
        self.assertEqual(hours, [10, 22])

    def test_zero_interval_falls_back(self):
        # interval<=0 曾把循环变成每秒一轮 hammer 上游
        interval, _ = _normalize_collect_cfg({"interval_seconds": 0})
        self.assertEqual(interval, 900)

    def test_negative_interval_falls_back(self):
        interval, _ = _normalize_collect_cfg({"interval_seconds": -300})
        self.assertEqual(interval, 900)

    def test_tiny_interval_falls_back(self):
        interval, _ = _normalize_collect_cfg({"interval_seconds": 5})
        self.assertEqual(interval, 900)

    def test_non_numeric_interval_falls_back(self):
        interval, _ = _normalize_collect_cfg({"interval_seconds": "15min"})
        self.assertEqual(interval, 900)

    def test_bad_hours_fall_back(self):
        for bad in (["10"], [22, 10], [-1, 22], [10, 25], "10-22", None):
            with self.subTest(bad=bad):
                _, hours = _normalize_collect_cfg({"active_hours": bad})
                self.assertEqual(hours, [10, 22])


class BoundaryTests(unittest.TestCase):
    def test_non_divisor_interval_returns_interval(self):
        # 不能整除 3600 → 固定间隔，不能退化成 1s
        self.assertEqual(_seconds_until_next_boundary(_now(), 700), 700)

    def test_divisor_interval_aligns(self):
        wait = _seconds_until_next_boundary(_now(), 900)
        self.assertGreaterEqual(wait, 1)
        self.assertLessEqual(wait, 900)


def _now():
    from datetime import datetime, timedelta, timezone

    return datetime(2026, 8, 14, 12, 3, 21, tzinfo=timezone(timedelta(hours=8)))


if __name__ == "__main__":
    unittest.main()
