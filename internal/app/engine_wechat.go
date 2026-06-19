package app

// engine_wechat.go —— runCapture 轮询期间 PC 微信进程探测的纯逻辑（无副作用、可单测）。
//
// 业务目的：用户在「拿通行证」流程被要求「彻底关闭 PC 微信后重开」，但工具此前完全无感知
// 用户做没做。本文件提供「从两轮微信进程快照判断是否发生了先关再开」的判定逻辑，
// 供 runCapture ticker 调用，结果经 CaptureStatusJSON.WeChat 推给前端做信号灯。

import . "github.com/Ryujoxys/sushiro-overdose/internal/platform"

// weChatSignature 是用于比对的进程指纹：PID + StartTime。
// 只比 PID 不够（PID 会被复用）；只比 StartTime 在拿不到时不稳。组合最稳。
type weChatSignature struct {
	PID       int
	StartTime string
}

// weChatSignatures 提取一组进程的指纹集合（用于集合比对，忽略顺序）。
func weChatSignatures(procs []WeChatProcessInfo) []weChatSignature {
	if len(procs) == 0 {
		return nil
	}
	sigs := make([]weChatSignature, 0, len(procs))
	for _, p := range procs {
		sigs = append(sigs, weChatSignature{PID: p.PID, StartTime: p.StartTime})
	}
	return sigs
}

// sigContains 报告指纹 s 是否在列表里（PID+StartTime 同时匹配）。
func sigContains(list []weChatSignature, s weChatSignature) bool {
	for _, x := range list {
		if x.PID == s.PID && x.StartTime == s.StartTime {
			return true
		}
	}
	return false
}

// weChatAppearsRestarted 判断从 baseline 到 current 是否发生了「先全关、再重开」。
// 已ClosedOnce（传入，runCapture 锁存）为 true 表示之前已观察到微信被关停；
// 此时若 current 又非空，即视为重启。
//
// 单次调用只看「baseline 非空 + current 空」→ 标记本轮「已关」（由调用方锁存 closedOnce）；
// 调用方在 closedOnce=true 且 current 非空时判 restarted。
// 这里把两种判定合一：closedOnce 由调用方维护，本函数只回答「当前快照是否构成重启证据」。
//
// 返回 (restarted bool, nowClosed bool)：
//   - nowClosed：baseline 非空且 current 为空（用户把微信关了）。
//   - restarted：已ClosedOnce（曾关过）且 current 又有进程（重开了）。
func weChatAppearsRestarted(baseline, current []WeChatProcessInfo, alreadyClosed bool) (restarted, nowClosed bool) {
	if len(current) == 0 {
		// 当前没微信在跑：若 baseline 有，说明本轮观察到了「关闭」。
		return false, len(baseline) > 0
	}
	// current 非空：若之前已关过（alreadyClosed），现在是重开 → restarted。
	// 或：同名进程出现更晚的 StartTime（同 PID 被复用+新启动时间）也算。
	if alreadyClosed {
		return true, false
	}
	// 兜底：current 里有进程的指纹不在 baseline 里（新增的），且 baseline 非空，
	// 也视作重启证据（覆盖「没观察到空窗就直接重启」的边界）。
	if len(baseline) > 0 {
		baseSigs := weChatSignatures(baseline)
		for _, c := range current {
			if !sigContains(baseSigs, weChatSignature{PID: c.PID, StartTime: c.StartTime}) {
				return true, false
			}
		}
	}
	return false, false
}

// buildWeChatStatus 从当前进程快照 + 锁存的 restarted 标志构造前端状态。
func buildWeChatStatus(current []WeChatProcessInfo, restarted bool) *WeChatStatusJSON {
	st := &WeChatStatusJSON{Running: len(current) > 0, Restarted: restarted}
	for _, p := range current {
		st.PIDs = append(st.PIDs, p.PID)
		if p.Name != "" {
			st.Names = append(st.Names, p.Name)
		}
	}
	return st
}
