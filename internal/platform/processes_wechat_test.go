package platform

import (
	"testing"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

// processes_wechat_test.go —— 微信进程枚举/杀进程输出的纯函数解析测试。
// 进程枚举本身依赖真实系统（PowerShell/ps），无法单测；这里测的是输出解析逻辑。

func TestParseWeChatProcessLines(t *testing.T) {
	// 空输入 → nil。
	if got := parseWeChatProcessLines(""); got != nil {
		t.Errorf("空输入应返回 nil，got %+v", got)
	}
	// 单行合法（路径用合法 JSON 转义）。
	got := parseWeChatProcessLines(`{"pid":1234,"name":"WeChat.exe","start_time":"2026-06-19T10:00:00+08:00","path":"C:\\WeChat.exe"}`)
	if len(got) != 1 || got[0].PID != 1234 || got[0].Name != "WeChat.exe" || got[0].StartTime == "" {
		t.Errorf("单行解析错误：got %+v", got)
	}
	// 多行 + 坏行跳过（不整体失败）。
	got = parseWeChatProcessLines(`
{"pid":1,"name":"WeChat.exe","start_time":"","path":""}
not-json-garbage
{"pid":2,"name":"WeChatAppEx.exe","start_time":"2026-06-19T10:01:00+08:00","path":""}
`)
	if len(got) != 2 {
		t.Fatalf("坏行应跳过，期望 2 个，got %d： %+v", len(got), got)
	}
	if got[0].PID != 1 || got[1].PID != 2 {
		t.Errorf("PID 解析错误：got %d %d", got[0].PID, got[1].PID)
	}
	// StartTime 缺失时留空（不报错）。
	if got[0].StartTime != "" {
		t.Errorf("start_time 缺失应留空，got %q", got[0].StartTime)
	}
	// PID<=0 跳过。
	got = parseWeChatProcessLines(`{"pid":0,"name":"x"}`)
	if len(got) != 0 {
		t.Errorf("PID<=0 应跳过，got %+v", got)
	}
	// name 缺失填默认。
	got = parseWeChatProcessLines(`{"pid":5,"name":"","start_time":"","path":""}`)
	if len(got) != 1 || got[0].Name != "wechat" {
		t.Errorf("name 缺失应填 wechat，got %+v", got)
	}
}

func TestParseWeChatKillOutput(t *testing.T) {
	// 空输出 → 单个 missing。
	got := parseWeChatKillOutput("")
	if len(got) != 1 || got[0].Status != MaintenanceStatusMissing {
		t.Errorf("空输出应回退 missing，got %+v", got)
	}
	// 正常 ok + error 混合。
	got = parseWeChatKillLines(t, `{"pid":1,"name":"WeChat.exe","status":"ok","error":""}`+"\n"+
		`{"pid":2,"name":"WeChatAppEx.exe","status":"error","error":"access denied"}`)
	if len(got) != 2 {
		t.Fatalf("期望 2 个结果，got %d", len(got))
	}
	if got[0].Status != MaintenanceStatusOK || got[1].Status != MaintenanceStatusError {
		t.Errorf("status 映射错误：got %s %s", got[0].Status, got[1].Status)
	}
	if got[0].Action != "kill_wechat" {
		t.Errorf("Action 应为 kill_wechat，got %q", got[0].Action)
	}
	if got[1].Name != "WeChatAppEx.exe" {
		t.Errorf("name 解析错误：got %q", got[1].Name)
	}
}

func parseWeChatKillLines(t *testing.T, out string) []MaintenanceResult {
	t.Helper()
	return parseWeChatKillOutput(out)
}
