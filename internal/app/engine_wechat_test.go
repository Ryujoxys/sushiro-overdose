package app

import (
	"testing"

	. "github.com/Ryujoxys/sushiro-overdose/internal/platform"
)

// engine_wechat_test.go —— 微信重启检测信号灯的纯逻辑测试。

func proc(pid int, name, start string) WeChatProcessInfo {
	return WeChatProcessInfo{PID: pid, Name: name, StartTime: start}
}

func TestWeChatAppearsRestarted(t *testing.T) {
	// 场景1：baseline 空、current 空 → 未关、未重启。
	r, c := weChatAppearsRestarted(nil, nil, false)
	if r || c {
		t.Errorf("都空应 (false,false)，got (%v,%v)", r, c)
	}
	// 场景2：baseline 有、current 空（用户关了微信）→ nowClosed=true。
	r, c = weChatAppearsRestarted(
		[]WeChatProcessInfo{proc(100, "WeChat.exe", "t0")},
		nil, false)
	if r || !c {
		t.Errorf("关掉微信应 nowClosed=true，got (%v,%v)", r, c)
	}
	// 场景3：已关过（closedOnce=true）、current 又有 → restarted=true。
	r, c = weChatAppearsRestarted(
		[]WeChatProcessInfo{proc(100, "WeChat.exe", "t0")},
		[]WeChatProcessInfo{proc(200, "WeChat.exe", "t1")},
		true)
	if !r || c {
		t.Errorf("关过又重开应 restarted=true，got (%v,%v)", r, c)
	}
	// 场景4：一直开着没动（baseline==current 指纹）→ 都 false。
	same := []WeChatProcessInfo{proc(100, "WeChat.exe", "t0")}
	r, c = weChatAppearsRestarted(same, same, false)
	if r || c {
		t.Errorf("指纹不变应都 false，got (%v,%v)", r, c)
	}
	// 场景5：没观察到空窗，但出现新指纹（PID 变了）→ restarted=true（兜底分支）。
	r, c = weChatAppearsRestarted(
		[]WeChatProcessInfo{proc(100, "WeChat.exe", "t0")},
		[]WeChatProcessInfo{proc(999, "WeChat.exe", "t0")},
		false)
	if !r {
		t.Errorf("新增指纹应 restarted=true（兜底），got (%v,%v)", r, c)
	}
}

func TestBuildWeChatStatus(t *testing.T) {
	// current 空 → Running=false。
	st := buildWeChatStatus(nil, false)
	if st == nil || st.Running || st.Restarted {
		t.Errorf("空快照应 Running=false，got %+v", st)
	}
	if len(st.PIDs) != 0 {
		t.Errorf("空快照 PIDs 应空，got %+v", st.PIDs)
	}
	// current 非空 + restarted → 正确填充。
	st = buildWeChatStatus(
		[]WeChatProcessInfo{proc(100, "WeChat.exe", "t0"), proc(101, "WeChatAppEx.exe", "t0")},
		true)
	if !st.Running || !st.Restarted {
		t.Errorf("应 Running+Restarted，got %+v", st)
	}
	if len(st.PIDs) != 2 || st.PIDs[0] != 100 {
		t.Errorf("PIDs 填充错误：got %+v", st.PIDs)
	}
	if len(st.Names) != 2 || st.Names[1] != "WeChatAppEx.exe" {
		t.Errorf("Names 填充错误：got %+v", st.Names)
	}
}
