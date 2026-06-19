package app

import (
	"context"
	"math"
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
	StoreID          string   `json:"store_id"`
	StoreName        string   `json:"store_name"`
	StoreStatus      string   `json:"store_status"`
	NetTicketStatus  string   `json:"net_ticket_status"`
	OnlineOpen       bool     `json:"online_open"`
	CalledNo         int      `json:"called_no"`               // 当前叫号
	WaitGroups       int      `json:"wait_groups"`             // 当前需等待（组/桌）
	WaitTimeCap      int      `json:"wait_time_cap,omitempty"` // 接口给的等待上限（分钟）
	TablesCapacity   int      `json:"tables_capacity,omitempty"`
	CountersCapacity int      `json:"counters_capacity,omitempty"`
	ServerWaitMin    int      `json:"server_wait_minutes"`    // 接口直接给的预估等待（分钟）
	Called15m        *int     `json:"called_15m,omitempty"`   // 近15分钟叫号推进
	RatePerMin       *float64 `json:"rate_per_min,omitempty"` // 历史均速（组/分）
	EtaMinutes       *int     `json:"eta_minutes,omitempty"`  // 综合预估等待（基于均速+在等组数）
	ObservedAt       string   `json:"observed_at"`            // 本次快照时间
	HistoryPoints    int      `json:"history_points"`         // 参与计算的历史观测数
	Spark            []int    `json:"spark,omitempty"`        // 最近一段叫号序列（时间升序），用于画推进小图
}

const queuePanelSparkMax = 40

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
		StoreID:          snapshot.StoreID,
		StoreName:        store.Name,
		StoreStatus:      store.StoreStatus,
		NetTicketStatus:  store.NetTicketStatus,
		OnlineOpen:       snapshot.OnlineOpen,
		CalledNo:         snapshot.DisplayCalledNo,
		WaitGroups:       snapshot.GroupQueuesCount,
		WaitTimeCap:      store.WaitTimeCap,
		TablesCapacity:   store.TablesCapacity,
		CountersCapacity: store.CountersCapacity,
		ServerWaitMin:    store.Wait,
		ObservedAt:       now.Format(time.RFC3339),
	}

	history := recentStoreObservations(loadQueueObservations(), snapshot.StoreID, now, queuePanelRateWindow)
	panel.HistoryPoints = len(history)
	if snapshot.DisplayCalledNo > 0 {
		// 把刚拿到的实时快照接到历史末尾，作为“现在”的叫号。
		history = append(history, snapshot)
	}
	panel.Spark = queuePanelSpark(history)
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

// queuePanelSpark 从时间升序的观测里取最近 queuePanelSparkMax 个叫号，用于前端画推进小图。
func queuePanelSpark(sorted []QueueObservation) []int {
	if len(sorted) < 2 {
		return nil
	}
	start := 0
	if len(sorted) > queuePanelSparkMax {
		start = len(sorted) - queuePanelSparkMax
	}
	out := make([]int, 0, len(sorted)-start)
	for _, o := range sorted[start:] {
		out = append(out, o.DisplayCalledNo)
	}
	return out
}

// recentStoreObservations 取某店在 [now-window, now] 内、按时间升序、带有效叫号的观测。
func recentStoreObservations(all []QueueObservation, storeID string, now time.Time, window time.Duration) []QueueObservation {
	cutoff := now.Add(-window)
	out := make([]QueueObservation, 0, len(all))
	for _, o := range all {
		if o.StoreID != storeID || o.DisplayCalledNo <= 0 {
			continue
		}
		at, ok := parseRFC3339Local(queueObservationCollectedAt(o))
		if !ok || at.Before(cutoff) || at.After(now) {
			continue
		}
		out = append(out, o)
	}
	sort.SliceStable(out, func(i, j int) bool {
		li, _ := parseRFC3339Local(queueObservationCollectedAt(out[i]))
		lj, _ := parseRFC3339Local(queueObservationCollectedAt(out[j]))
		return li.Before(lj)
	})
	return out
}

// calledAdvanceWithin 返回最近 window 内叫号推进了多少（取窗口内最早点到最后点的差值）。
func calledAdvanceWithin(sorted []QueueObservation, now time.Time, window time.Duration) (int, bool) {
	cutoff := now.Add(-window)
	first, last := -1, len(sorted)-1
	for i, o := range sorted {
		at, ok := parseRFC3339Local(queueObservationCollectedAt(o))
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

// calledRatePerMinute 估算叫号速度（组/分）。兼容包装：以最后一个观测时刻为 now、
// 2 小时窗口调用 calledRatePerMinuteWeighted。保留老签名是为了让未传 now/window 的调用方
// （queue_alert_status、buildQueueLivePanel 等）不破。新代码应直接调 weighted 版拿 cv/n。
func calledRatePerMinute(sorted []QueueObservation) (float64, bool) {
	if len(sorted) == 0 {
		return 0, false
	}
	now, ok := parseRFC3339Local(queueObservationCollectedAt(sorted[len(sorted)-1]))
	if !ok {
		return 0, false
	}
	rate, _, _, _, _, okw := calledRatePerMinuteWeighted(sorted, now, queuePanelRateWindow)
	if !okw {
		return 0, false
	}
	return rate, true
}

// callRateInterval 是两个相邻观测之间的有效叫号间隔：瞬时速率 + 间隔中点距 now 的分钟数。
type callRateInterval struct {
	rate   float64
	midMin float64
}

// calledRatePerMinuteWeighted 估算叫号速度（组/分），并返回速度的变异系数 cv 与有效间隔数 n。
//
// 流水线（顺序重要）：
//  1. 遍历相邻观测对，算每个有效推进间隔的瞬时速率 rate_i（叫号推进/间隔分钟）+ 间隔中点时间；
//     跳过停滞/回退间隔（diff<=0，午休/换班/重置）。
//  2. 【D 异常剔除】filterOutlierRates：IQR 剔除补号跳变等离群点。
//  3. 【A 时间加权】线性时间衰减：越近的间隔权重越高（最近权重 1.0，window 前权重 0.2），
//     反映「当前」节奏而非整个窗口的平均。对不均匀采样（间隔 30s~几分钟混杂）天然友好。
//  4. 【B 离散度】用剔除后的原始 rate_i 算变异系数 cv（σ/mean），供下游动态调节区间宽度。
//
// 返回：rate=A 加权结果；cv=变异系数（有效间隔<2 时返回 -1 哨兵）；n=剔除后有效间隔数；
// rates=IQR 剔除后的瞬时速率序列（供下游经验分位数区间使用；空表示无可用样本）；
// ok=是否算出可用速度。
//
// 退化：密集采样（间隔中点都≈now）→ 权重≈1 → 退化为等权 = 旧行为（回归保护）；
// 有效间隔为 0 → 退回首尾两点法兜底（不因丢速度而返回 false）。
func calledRatePerMinuteWeighted(sorted []QueueObservation, now time.Time, window time.Duration) (rate float64, cv float64, n int, trend float64, rates []float64, ok bool) {
	if len(sorted) < 2 {
		return 0, -1, 0, 1, nil, false
	}

	// 观测点的 CollectedAt 全链路是秒级 RFC3339（见 queueObservationFromLiveStore 等），
	// 而 now 常为纳秒精度的 time.Now()。把 now 截断到秒级与观测对齐，消除 ageMin/now 上 ≤1s
	// 的系统性偏差；同时避免密集采样（间隔接近 1 分钟）时最后一个间隔因 now 高出几秒而被
	// 错误地计入/丢弃。截断只影响内部差值计算，不改变调用方传入的 now。
	now = now.Truncate(time.Second)

	windowMin := window.Minutes()
	if windowMin <= 0 {
		windowMin = queuePanelRateWindow.Minutes()
	}

	var ivs []callRateInterval
	for i := 1; i < len(sorted); i++ {
		atPrev, ok1 := parseRFC3339Local(queueObservationCollectedAt(sorted[i-1]))
		atCur, ok2 := parseRFC3339Local(queueObservationCollectedAt(sorted[i]))
		if !ok1 || !ok2 {
			continue
		}
		spanMin := atCur.Sub(atPrev).Minutes()
		diff := sorted[i].DisplayCalledNo - sorted[i-1].DisplayCalledNo
		if spanMin <= 0 || diff <= 0 {
			continue
		}
		mid := atPrev.Add(atCur.Sub(atPrev) / 2)
		ageMin := now.Sub(mid).Minutes()
		if ageMin < 0 {
			ageMin = 0 // 间隔中点在未来（时钟漂移）按 0 处理
		}
		ivs = append(ivs, callRateInterval{rate: float64(diff) / spanMin, midMin: ageMin})
	}

	// 【D】IQR 剔除离群点（补号跳变等）。对 interval 列表按 rate 做 IQR 过滤，保留 midMin。
	rateVals := make([]float64, len(ivs))
	for i, v := range ivs {
		rateVals[i] = v.rate
	}
	if loF, hiF := iqrFences(rateVals); !math.IsNaN(loF) && !math.IsNaN(hiF) {
		filtered := make([]callRateInterval, 0, len(ivs))
		for _, v := range ivs {
			if v.rate >= loF && v.rate <= hiF {
				filtered = append(filtered, v)
			}
		}
		if len(filtered) >= 2 {
			ivs = filtered
		} // 否则剔除过狠，保留原 ivs。
	}

	if len(ivs) == 0 {
		// 有效间隔为 0：退回首尾两点法兜底（仍优于无速度）。无趋势信息、无速率序列。
		firstAt, ok := parseRFC3339Local(queueObservationCollectedAt(sorted[0]))
		if !ok {
			return 0, -1, 0, 1, nil, false
		}
		lastAt, ok := parseRFC3339Local(queueObservationCollectedAt(sorted[len(sorted)-1]))
		if !ok {
			return 0, -1, 0, 1, nil, false
		}
		span := lastAt.Sub(firstAt).Minutes()
		diff := sorted[len(sorted)-1].DisplayCalledNo - sorted[0].DisplayCalledNo
		if span <= 0 || diff <= 0 {
			return 0, -1, 0, 1, nil, false
		}
		return float64(diff) / span, -1, 0, 1, nil, true
	}

	// 【A】指数时间衰减加权：近期叫号间隔权重高，反映「当前」节奏。
	// 半衰期 = window/6（30min 窗→半衰期5min，2h 窗→半衰期20min），自适应窗口：
	// 短窗对几分钟内的变化敏感，长窗平滑更久的历史。底 0.05 保留弱先验（不全杀边缘间隔）。
	// 相比线性衰减，指数衰减对「最近几分钟的速度突变」反应更陡——门店叫号「现在多快」
	// 本质是短期节奏，几十分钟前的慢段不应显著拖低当前速度估计。
	halfLife := windowMin / 6.0
	if halfLife < 3 {
		halfLife = 3
	}
	var wsum, wrsum float64
	for _, v := range ivs {
		w := math.Pow(0.5, v.midMin/halfLife)
		if w < 0.05 {
			w = 0.05
		}
		wsum += w
		wrsum += w * v.rate
	}
	weightedRate := wrsum / wsum

	// 【B】变异系数（用剔除后的原始 rate_i，分母用加权均值）。plainRates 同时作为下游
	// 经验分位数区间的样本序列返回（已 IQR 剔除补号跳变）。
	plainRates := make([]float64, len(ivs))
	for i, v := range ivs {
		plainRates[i] = v.rate
	}
	cv = coefficientOfVariation(plainRates)

	// 【趋势】近半窗 vs 远半窗的加权速度比，>1=加速、<1=减速、≈1=稳定。
	// 用于 ETA 前瞻校正：节奏在变快时，未来叫号会比过去更快，纯用历史速度外推会系统性高估等待。
	trend = rateTrendRatio(ivs, halfLife)

	return weightedRate, cv, len(ivs), trend, plainRates, true
}

// rateTrendRatio 算速度趋势比 = 近半窗加权速度 / 远半窗加权速度。
// halfLife 与加权速度用的半衰期一致。样本不足以稳定分割时返回 1（无趋势）。
// 结果钳到 [0.5, 2.0]，防止单次抖动产生极端校正。
func rateTrendRatio(ivs []callRateInterval, halfLife float64) float64 {
	if len(ivs) < 4 {
		return 1
	}
	// 找 midMin 的中位数作为分割点（近/远各半）。
	// 用三分位把样本分近/中/远三段，取最近 1/3 vs 最远 1/3 的加权速度比。
	// 相比中位数二分，三分位对端点（最新与最旧）更敏感，能捕捉整窗内的渐变趋势
	// （如午高峰临近，整窗速度都在缓慢上升），而二分只比较两半、对全窗渐变不敏感。
	mids := make([]float64, len(ivs))
	for i, v := range ivs {
		mids[i] = v.midMin
	}
	sort.Float64s(mids)
	third := len(mids) / 3
	if third < 1 {
		third = 1
	}
	nearCutoff := mids[third]          // 最近 1/3 的边界
	farCutoff := mids[len(mids)-third] // 最远 1/3 的边界

	var nearW, nearR, farW, farR float64
	for _, v := range ivs {
		w := math.Pow(0.5, v.midMin/halfLife)
		if w < 0.05 {
			w = 0.05
		}
		if v.midMin <= nearCutoff {
			nearW += w
			nearR += w * v.rate
		} else if v.midMin >= farCutoff {
			farW += w
			farR += w * v.rate
		}
	}
	if farW <= 0 || nearW <= 0 {
		return 1
	}
	nearRate := nearR / nearW
	farRate := farR / farW
	if farRate <= 0 {
		return 1
	}
	ratio := nearRate / farRate
	// 钳制：避免噪声产生极端校正。
	if ratio < 0.5 {
		ratio = 0.5
	}
	if ratio > 2.0 {
		ratio = 2.0
	}
	return ratio
}
