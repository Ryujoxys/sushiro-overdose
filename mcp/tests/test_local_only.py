import os
import runpy
import sys
import types
import unittest
from unittest.mock import Mock, patch

from mcp_server.config import load_config
from mcp_server import tools_advice, tools_queue


class DesktopStub:
    def __init__(self, response=None):
        self.calls = []
        self.response = response if response is not None else {"ok": True, "stores": [{"id": "1012"}]}

    def get(self, path, params=None):
        self.calls.append((path, params))
        return self.response


class LocalOnlyTests(unittest.TestCase):
    def test_registered_module_entrypoint_calls_server(self):
        server = types.ModuleType("mcp_server.server")
        server.main = Mock()
        with patch.dict(sys.modules, {"mcp_server.server": server}):
            runpy.run_module("mcp_server", run_name="__main__")
        server.main.assert_called_once_with()

    def test_legacy_database_environment_is_ignored(self):
        with patch.dict(os.environ, {"SUSHIRO_MCP_TURSO_URL": "https://example.invalid", "SUSHIRO_MCP_TURSO_TOKEN": "secret", "SUSHIRO_MCP_DESKTOP_PORT": "39872"}):
            self.assertEqual(vars(load_config()), {"desktop_port": 39872})

    def test_invalid_ports_fall_back(self):
        for value in ("0", "65536", "invalid"):
            with patch.dict(os.environ, {"SUSHIRO_MCP_DESKTOP_PORT": value}):
                self.assertEqual(load_config().desktop_port, 39871)

    def test_queue_tools_only_use_desktop_read_endpoints(self):
        desktop = DesktopStub()
        self.assertEqual(tools_queue.list_stores(desktop, limit=500), [{"id": "1012"}])
        tools_queue.store_queue_history(desktop, 1012)
        tools_queue.store_pressure(desktop, 1012)
        tools_queue.called_speed(desktop, 1012)
        self.assertEqual([p for p, _ in desktop.calls], ["/api/queue/stores", "/api/queue/dashboard", "/api/queue/trends", "/api/queue/live"])
        self.assertEqual(desktop.calls[0][1]["limit"], 100)

    def test_failure_is_not_disguised_as_empty_stores(self):
        desktop = DesktopStub({"ok": False, "hint": "offline"})
        self.assertEqual(tools_queue.list_stores(desktop), desktop.response)

    def test_compare_uses_half_open_thirty_minute_window(self):
        desktop = DesktopStub()
        tools_queue.compare_stores(desktop, [1012, 1013], time_bucket="18:30")
        self.assertEqual(desktop.calls[-1][1], {"stores": "1012,1013", "date_type": "weekday", "start": "1830", "end": "1900"})
        tools_queue.compare_stores(desktop, [1012], time_bucket="23:30")
        self.assertEqual(desktop.calls[-1][1]["end"], "0000")
        for value in ("24:00", "12:60", "oops"):
            self.assertFalse(tools_queue.compare_stores(desktop, [1012], time_bucket=value)["ok"])

    def test_advice_preserves_backend_uncertainty(self):
        desktop = DesktopStub({"ok": True, "confidence": "none", "estimate": None})
        result = tools_advice.arrival_advice(desktop, 1012, target_no=123, travel_minutes=15)
        self.assertIs(result, desktop.response)
        self.assertEqual(desktop.calls[-1], ("/api/queue/advisor", {"store": "1012", "target_no": "123", "travel_minutes": "15"}))
        tools_advice.arrival_advice(desktop, 1012, want_meal_time="18:30")
        self.assertEqual(desktop.calls[-1], ("/api/queue/plan", {"store": "1012", "target_meal": "1830"}))


if __name__ == "__main__":
    unittest.main()
