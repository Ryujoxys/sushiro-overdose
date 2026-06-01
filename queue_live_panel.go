package main

import (
	"context"
	"sort"
	"time"
)

// 近15分钟叫号的统计窗口，以及历史均速的回看窗口。
const (
	queuePanelRecentWindow = 15 * time.Minute
	queuePanelRateWindow   = 2 * time.Hour
)

// QueueLivePanel 是单店实时排队面板，对应寿司小助手首页那几个指标。
// 实时快照来自 getStoreById；近15分钟叫号/历史均速/预估等待来自本机采样历史，
// 没有历史时这些字段为 nil，前端据此提示“数据收集中”。
type QueueLivePanel struct {
	StoreID         string   `json:"store_id"`
	StoreName       string   `json:"store_name"`
	StoreStatus     string   `json:"store_status"`
	NetTicketStatus string   `json:"net_ticket_status"`
	OnlineOpen      bool     `json:"online_open"`
	CalledNo        int      `json:"called_no"`               // 当前叫号
	WaitGroups      int      `json:"wait_groups"`             // 当前需等待（组/桌）
	WaitTimeCap     int      `json:"wait_time_cap,omitempty"` // 接口给的等待上限（分钟）
	ServerWaitMin   int      `json:"server_wait_minutes"`     // 接口直接给的预估等待（分钟）
	Called15m       *int     `json:"called_15m,omitempty"`    // 近15分钟叫号推进
	RatePerMin      *float64 `json:"rate_per_min,omitempty"`  // 历史均速（组/分）
	EtaMinutes      *int     `json:"eta_minutes,omitempty"`   // 综合预估等待（基于均速+在等组数）
	ObservedAt      string   `json:"observed_at"`             // 本次快照时间
	HistoryPoints   int      `json:"history_points"`          // 参与计算的历史观测数
}

func buildQueueLivePanel(ctx context.Context, storeID string, now time.Time) (QueueLivePanel, error) {
	store, err := NewQueueLiveClient().GetStore(ctx, storeID)
	if err != nil {
		return QueueLivePanel{}, err
	}
	if now.IsZero() {
		now = time.Now()
	}
	snapshot := queueObservationFromLiveStore(store, now)

	panel := QueueLivePanel{
		StoreID:         snapshot.StoreID,
		StoreName:       store.Name,
		StoreStatus:     store.StoreStatus,
		NetTicketStatus: store.NetTicketStatus,
		OnlineOpen:      snapshot.OnlineOpen,
		CalledNo:        snapshot.DisplayCalledNo,
		WaitGroups:      snapshot.GroupQueuesCount,
		WaitTimeCap:     store.WaitTimeCap,
		ServerWaitMin:   store.Wait,
		ObservedAt:      now.Format(time.RFC3339),
	}

	history := recentStoreObservations(loadQueueObservations(), snapshot.StoreID, now, queuePanelRateWindow)
	panel.HistoryPoints = len(history)
	if snapshot.DisplayCalledNo > 0 {
		// 把刚拿到的实时快照接到历史末尾，作为“现在”的叫号。
		history = append(history, snapshot)
	}
	if len(history) < 2 {
		return panel, nil
	}

	if c15, ok := calledAdvanceWithin(history, now, queuePanelRecentWindow); ok {
		panel.Called15m = &c15
	}
	if rate, ok := calledRatePerMinute(history); ok && rate > 0 {
		panel.RatePerMin = &rate
		eta := int(float64(panel.WaitGroups) / rate)
		if eta > 0 {
			panel.EtaMinutes = &eta
		}
	}
	return panel, nil
}

// recentStoreObservations 取某店在 [now-window, now] 内、按时间升序、带有效叫号的观测。
func recentStoreObservations(all []QueueObservation, storeID string, now time.Time, window time.Duration) []QueueObservation {
	cutoff := now.Add(-window)
	out := make([]QueueObservation, 0, len(all))
	for _, o := range all {
		if o.StoreID != storeID || o.DisplayCalledNo <= 0 {
			continue
		}
		at, ok := parseRFC3339Local(o.Timestamp)
		if !ok || at.Before(cutoff) || at.After(now) {
			continue
		}
		out = append(out, o)
	}
	sort.SliceStable(out, func(i, j int) bool {
		li, _ := parseRFC3339Local(out[i].Timestamp)
		lj, _ := parseRFC3339Local(out[j].Timestamp)
		return li.Before(lj)
	})
	return out
}

// calledAdvanceWithin 返回最近 window 内叫号推进了多少（取窗口内最早点到最后点的差值）。
func calledAdvanceWithin(sorted []QueueObservation, now time.Time, window time.Duration) (int, bool) {
	cutoff := now.Add(-window)
	first, last := -1, len(sorted)-1
	for i, o := range sorted {
		at, ok := parseRFC3339Local(o.Timestamp)
		if !ok || at.Before(cutoff) {
			continue
		}
		first = i
		break
	}
	if first < 0 || first >= last {
		return 0, false
	}
	diff := sorted[last].DisplayCalledNo - sorted[first].DisplayCalledNo
	if diff < 0 {
		return 0, false
	}
	return diff, true
}

// calledRatePerMinute 用首尾观测算平均叫号速度（组/分），忽略叫号回退（跨日/重置）。
func calledRatePerMinute(sorted []QueueObservation) (float64, bool) {
	firstAt, ok := parseRFC3339Local(sorted[0].Timestamp)
	if !ok {
		return 0, false
	}
	lastAt, ok := parseRFC3339Local(sorted[len(sorted)-1].Timestamp)
	if !ok {
		return 0, false
	}
	span := lastAt.Sub(firstAt).Minutes()
	diff := sorted[len(sorted)-1].DisplayCalledNo - sorted[0].DisplayCalledNo
	if span <= 0 || diff <= 0 {
		return 0, false
	}
	return float64(diff) / span, true
}
