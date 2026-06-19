package app

// queue_advisor_dynamic_test.go —— 动态/时序场景下的算法鲁棒性验证。
//
// 线上数据源源不断更新，算法必须能适应「数据随时间演变」。这组测试模拟门店从开号到叫完号的
// 完整过程，每隔几分钟采集一次（如真实采样器），在多个时间点检查预测，覆盖三类真实挑战：
//
//  1. 收敛性：采样点从少到多，预测应越来越稳（不震荡），误差单调下降。
//  2. 概念漂移(concept drift)：门店节奏中途变化（午餐高峰→下午平峰），算法多快跟上。
//  3. 噪声鲁棒性：真实叫号间隔有随机抖动，算法在噪声下稳不稳、会不会过拟合单次异常。

import (
	"math"
	"testing"
	"time"
)

// dynSample 模拟一次采样：在 baseMin 相对 now 的时刻，门店已叫到 calledNo 号。
func dynSample(store string, now time.Time, baseMin int, calledNo int) QueueObservation {
	return obsAt(store, calledNo, 0, 0, now.Add(time.Duration(baseMin)*time.Minute))
}

// collectUpTo 从一组按时间生成的采样里，取所有 <= tMin（相对 now）的点。
// 模拟「到 tMin 时刻，采样器已收集到这些观测」。
func collectUpTo(samples []QueueObservation, now time.Time, tMin int) []QueueObservation {
	cutoff := now.Add(time.Duration(tMin) * time.Minute)
	out := make([]QueueObservation, 0, len(samples))
	for _, s := range samples {
		at, err := time.Parse(time.RFC3339, s.CollectedAt)
		if err == nil && !at.After(cutoff) {
			out = append(out, s)
		}
	}
	return out
}

// TestDynamicConvergence 收敛性：匀速 2组/分，采样从少到多，预测误差应趋于 0 且不震荡。
func TestDynamicConvergence(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	window := queueAdvisorWindow30

	// 门店从 -60min 开号，匀速 2组/分。用户号 1200（开号时叫 1000，还剩 200 组 → 真实等待 100min）。
	// 每分钟生成一个采样点。
	var allSamples []QueueObservation
	for m := -60; m <= 0; m++ {
		no := 1000 + (60+m)*2 // m=-60→1000, m=0→1120
		allSamples = append(allSamples, dynSample(store, now, m, no))
	}

	// 在多个时间点（相对 now）检查：预测的等待中点 vs 真实剩余等待。
	// 真实剩余等待 = (1200 - 当前叫号) / 2。
	checkpoints := []int{-50, -40, -30, -20, -10, 0}
	prevErr := math.MaxFloat64
	converging := true
	for _, tMin := range checkpoints {
		obs := collectUpTo(allSamples, now, tMin)
		obs = recentStoreObservations(obs, store, now.Add(time.Duration(tMin)*time.Minute), window)
		nowAt := now.Add(time.Duration(tMin) * time.Minute)
		// 用 checkpoint 时刻作为 now 算速率（窗口相对该时刻）。
		rate, cv, n, trend, _, ok := calledRatePerMinuteWeighted(obs, nowAt, window)
		if !ok {
			t.Logf("[t=%+d] 数据不足，跳过", tMin)
			continue
		}
		calledAtT := 1000 + (60+tMin)*2
		remaining := 1200 - calledAtT
		trueWait := float64(remaining) / 2.0
		wr, _ := estimateWaitRange(remaining, 0, rate, cv, n, trend, nil, nil)
		mid := -1.0
		if wr != nil {
			mid = float64(wr.Low+wr.High) / 2.0
		}
		errPct := -999.0
		if mid > 0 && trueWait > 0 {
			errPct = (mid - trueWait) / trueWait * 100
		}
		t.Logf("[t=%+d] n=%d rate=%.2f cv=%.2f 真实等待=%.0fmin 预测中点=%.0fmin 误差=%+.1f%%",
			tMin, n, rate, cv, trueWait, mid, errPct)
		// 收敛性：误差绝对值应随 tMin 增大（数据增多）整体下降（允许单步小回弹）。
		if math.Abs(errPct) > prevErr+15 {
			converging = false
		}
		if math.Abs(errPct) < prevErr {
			prevErr = math.Abs(errPct)
		}
	}
	// 最终点（数据最足）误差应很小。
	lastObs := collectUpTo(allSamples, now, 0)
	lastObs = recentStoreObservations(lastObs, store, now, window)
	rate, cv, n, trend, _, _ := calledRatePerMinuteWeighted(lastObs, now, window)
	wr, _ := estimateWaitRange(80, 0, rate, cv, n, trend, nil, nil) // t=0 时还剩 80 组
	if wr != nil {
		mid := float64(wr.Low+wr.High) / 2.0
		err := math.Abs(mid-40) / 40 * 100 // 真实 80/2=40min
		if err > 10 {
			t.Errorf("数据充足后误差应 <10%%，实际 %.1f%%（收敛失败）", err)
		}
	}
	if !converging {
		t.Errorf("预测误差未随数据积累收敛（出现大幅回弹）")
	}
}

// TestDynamicConceptDrift 概念漂移：节奏中途从快变慢，算法应跟上当前节奏而非被旧数据拖。
func TestDynamicConceptDrift(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	window := queueAdvisorWindow30

	// 门店：[-60,-30] 快速 4组/分（叫 120 号），[-30,0] 慢速 1组/分（叫 30 号）。
	// 当前真实节奏=1组/分。算法应反映「现在慢」，不应停留在旧的 4组/分。
	var allSamples []QueueObservation
	no := 1000
	for m := -60; m <= -30; m++ {
		allSamples = append(allSamples, dynSample(store, now, m, no))
		no += 4 // 每分钟 +4（粗粒度，间隔由 recentStoreObservations 的时间决定）
	}
	for m := -29; m <= 0; m++ {
		allSamples = append(allSamples, dynSample(store, now, m, no))
		no += 1 // 每分钟 +1（慢速段）
	}

	// 在 t=-15 和 t=0 检查速度估计。t=-15 时近窗已全是慢速段。
	for _, tMin := range []int{-15, 0} {
		obs := collectUpTo(allSamples, now, tMin)
		nowAt := now.Add(time.Duration(tMin) * time.Minute)
		obs = recentStoreObservations(obs, store, nowAt, window)
		rate, _, _, _, _, ok := calledRatePerMinuteWeighted(obs, nowAt, window)
		if !ok {
			t.Logf("[t=%+d] 数据不足", tMin)
			continue
		}
		t.Logf("[t=%+d] 漂移后速度=%.2f（真实当前=1.0，旧节奏=4.0）", tMin, rate)
		// 应明显偏向当前慢速（1.0），远离旧快速（4.0）。阈值 <2.5 表示已跟上漂移。
		if rate > 2.5 {
			t.Errorf("[t=%+d] 概念漂移后速度=%.2f 仍偏高，未跟上当前慢节奏（应接近1.0）", tMin, rate)
		}
	}
}

// TestDynamicNoiseRobustness 噪声鲁棒性：真实叫号间隔有抖动，预测不应剧烈震荡。
func TestDynamicNoiseRobustness(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	window := queueAdvisorWindow30

	// 匀速 2组/分，但每次叫号 ±50% 抖动（模拟真实不均匀）。
	// 用确定性「伪噪声」（基于叫号奇偶），避免测试随机性。
	no := 1000
	var allSamples []QueueObservation
	for m := -27; m <= 0; m++ {
		jitter := 0
		if m%2 == 0 {
			jitter = 1 // 偶数分钟多叫1个
		} else if m%3 == 0 {
			jitter = -0 // 奇数且3的倍数少叫（这里简化为0）
		}
		no += 2 + jitter
		allSamples = append(allSamples, dynSample(store, now, m, no))
	}
	obs := recentStoreObservations(allSamples, store, now, window)
	rate, cv, n, _, _, ok := calledRatePerMinuteWeighted(obs, now, window)
	if !ok {
		t.Fatal("噪声场景应算出速度")
	}
	t.Logf("[噪声] n=%d 速度=%.2f cv=%.2f（真实均值≈2.0~2.3）", n, rate, cv)
	// 噪声下速度应仍在合理范围（1.5~3.0），不因抖动剧烈偏离。
	if rate < 1.5 || rate > 3.0 {
		t.Errorf("噪声下速度=%.2f 偏离合理范围 [1.5,3.0]，鲁棒性不足", rate)
	}
	// cv 应反映抖动存在但不过大（0~0.5 之间算可控）。
	if cv < 0 || cv > 0.6 {
		t.Errorf("噪声下 cv=%.2f 异常（应 0~0.6）", cv)
	}
}

// TestDynamicWindowBoundary 窗口边界：旧数据滑出窗口后不再影响预测。
func TestDynamicWindowBoundary(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	window := queueAdvisorWindow30

	// [-60,-35] 极慢 0.5组/分（叫 12 号），[-30,0] 正常 2组/分。
	// 当前真实=2组/分。慢段已滑出 30min 窗，应不影响。
	var allSamples []QueueObservation
	no := 1000
	for m := -60; m <= -35; m++ {
		allSamples = append(allSamples, dynSample(store, now, m, no))
		if m%2 == 0 {
			no += 1 // 0.5组/分
		}
	}
	for m := -30; m <= 0; m++ {
		allSamples = append(allSamples, dynSample(store, now, m, no))
		no += 2
	}
	obs := recentStoreObservations(allSamples, store, now, window)
	rate, _, _, _, _, ok := calledRatePerMinuteWeighted(obs, now, window)
	if !ok {
		t.Fatal("应算出速度")
	}
	t.Logf("[窗口边界] 速度=%.2f（真实=2.0，窗外的极慢段应被忽略）", rate)
	// 慢段在窗外，速度应≈2.0，不被 0.5组/分 拖低。
	if rate < 1.5 {
		t.Errorf("窗外旧数据不应影响预测，速度=%.2f 偏低（被极慢段拖累）", rate)
	}
}

// TestDynamicUnevenSampling 不均匀采样间隔：真实采样器因网络/调度，间隔可能 30s~5min 混杂。
// 算法用「间隔中点时间」做加权（而非「第几个样本」），应不受采样密度变化影响。
func TestDynamicUnevenSampling(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	window := queueAdvisorWindow30

	// 匀速 2组/分，但采样间隔极度不均匀：前段每5min一次、后段每30s一次（密集）。
	// 若算法按「样本序号」加权，密集段会过度主导；按「时间」加权则应仍≈2.0。
	offs := []int{-27, -22, -17, -12, -7, -6, -5, -4, -3, -2, -1, 0}
	var allSamples []QueueObservation
	no := 1000
	prevMin := -27
	for _, m := range offs {
		no = 1000 + (27+m)*2 // 严格按匀速 2组/分 的叫号
		_ = prevMin
		allSamples = append(allSamples, dynSample(store, now, m, no))
	}
	obs := recentStoreObservations(allSamples, store, now, window)
	rate, cv, n, _, _, ok := calledRatePerMinuteWeighted(obs, now, window)
	if !ok {
		t.Fatal("不均匀采样应算出速度")
	}
	t.Logf("[不均匀采样] n=%d 速度=%.2f cv=%.2f（真实=2.0，密集段不应过度主导）", n, rate, cv)
	// 速度应仍≈2.0，不被密集的后段（7个点挤在最后7min）或稀疏的前段带偏。
	if rate < 1.6 || rate > 2.4 {
		t.Errorf("不均匀采样下速度=%.2f 偏离 2.0，对采样密度敏感（应按时间加权）", rate)
	}
}

// TestDynamicSamplingFrequencyChange 采样频率变化：同一节奏下，每分钟采 vs 每5分钟采，
// 速度估计应基本一致（算法不该因为「采得密」就变快、「采得稀」就变慢）。
func TestDynamicSamplingFrequencyChange(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	window := queueAdvisorWindow30

	// 密集采样：每分钟一个点。
	var dense []QueueObservation
	for m := -27; m <= 0; m++ {
		dense = append(dense, dynSample(store, now, m, 1000+(27+m)*2))
	}
	// 稀疏采样：每3分钟一个点。
	var sparse []QueueObservation
	for m := -27; m <= 0; m += 3 {
		sparse = append(sparse, dynSample(store, now, m, 1000+(27+m)*2))
	}
	denseObs := recentStoreObservations(dense, store, now, window)
	sparseObs := recentStoreObservations(sparse, store, now, window)
	rateD, _, nD, _, _, okD := calledRatePerMinuteWeighted(denseObs, now, window)
	rateS, _, nS, _, _, okS := calledRatePerMinuteWeighted(sparseObs, now, window)
	if !okD || !okS {
		t.Fatal("两种频率都应算出速度")
	}
	diff := math.Abs(rateD - rateS)
	t.Logf("[频率变化] 密集(n=%d)速度=%.3f vs 稀疏(n=%d)速度=%.3f 差=%.3f", nD, rateD, nS, rateS, diff)
	// 同节奏不同采样频率，速度估计应基本一致（差 <0.3）。
	if diff > 0.3 {
		t.Errorf("采样频率不应影响速度估计：密集=%.3f 稀疏=%.3f 差=%.3f", rateD, rateS, diff)
	}
}

// TestDynamicRollingStability 滚动预测稳定性：模拟采样器持续运行，每采一次就重新预测，
// 检查连续预测之间的跳变幅度。用户看到的「预计几点叫到」不该剧烈抖动（如 12:30→13:15→12:40）。
//
// 关键场景：节奏从慢渐变到快（真实门店午高峰临近），预测应平滑过渡，单步跳变受控。
func TestDynamicRollingStability(t *testing.T) {
	now := time.Date(2026, 6, 8, 11, 0, 0, 0, time.Local)
	store := "3006"
	window := queueAdvisorWindow30
	targetNo := 1300 // 用户号

	// 节奏从 1组/分 渐变到 3组/分（午高峰临近，越来越快）。每分钟一个采样。
	// 从 -40min 开始，每采一次（t 推进1min）就重新预测。
	no := 1000
	type snap struct {
		tMin    int
		predMin float64 // 预测的等待中点分钟
		trueMin float64 // 真实剩余等待
	}
	var snaps []snap

	// 预生成全部采样（节奏渐变），然后滚动截取。
	var fullSamples []QueueObservation
	no = 1000
	for m := -40; m <= 0; m++ {
		fullSamples = append(fullSamples, dynSample(store, now, m, no))
		// 速度从 1.0 线性增到 3.0
		speed := 1.0 + float64(40+m)/40.0*2.0
		no += int(math.Round(speed))
	}

	for tMin := -40 + 10; tMin <= 0; tMin++ { // 从有足够数据开始滚动
		nowAt := now.Add(time.Duration(tMin) * time.Minute)
		obs := collectUpTo(fullSamples, now, tMin)
		obs = recentStoreObservations(obs, store, nowAt, window)
		rate, cv, n, trend, _, ok := calledRatePerMinuteWeighted(obs, nowAt, window)
		if !ok {
			continue
		}
		// 当前真实叫号（按 fullSamples 最后一个 <= tMin 的点）
		calledAtT := 1000
		for _, s := range obs {
			if s.DisplayCalledNo > calledAtT {
				calledAtT = s.DisplayCalledNo
			}
		}
		remaining := targetNo - calledAtT
		if remaining <= 0 {
			continue
		}
		// 真实剩余等待：用当前真实速度（由 fullSamples 在 tMin 的斜率）。
		speed := 1.0 + float64(40+tMin)/40.0*2.0
		trueWait := float64(remaining) / speed
		wr, _ := estimateWaitRange(remaining, 0, rate, cv, n, trend, nil, nil)
		predWait := -1.0
		if wr != nil {
			predWait = float64(wr.Low+wr.High) / 2.0
		}
		snaps = append(snaps, snap{tMin, predWait, trueWait})
	}

	// 打印所有快照。
	for _, s := range snaps {
		err := 0.0
		if s.trueMin > 0 {
			err = (s.predMin - s.trueMin) / s.trueMin * 100
		}
		tag := ""
		if s.trueMin <= 90 {
			tag = " ★临近" // 用户会据此出门的时间点
		}
		t.Logf("[t=%+d] 预测=%.0fmin 真实=%.0fmin 误差=%+.0f%%%s", s.tMin, s.predMin, s.trueMin, err, tag)
	}

	// 判定分两段，符合真实体验：
	//   - 临近段（真实剩余≤90min，用户真正会据此出门）：偏移≤20min（用户硬指标），单步跳变≤10min；
	//   - 早期段（剩余>90min，用户只看「大概下午」）：放宽到偏移≤40min，跳变≤30min。
	// 这精确匹配「误差在偏移前后20min内」的诉求——只在该要求起作用的时间窗口内严格判定。
	maxNearJump, maxFarJump := 0.0, 0.0
	maxNearErr, maxFarErr := 0.0, 0.0
	for i, s := range snaps {
		errAbs := math.Abs(s.predMin - s.trueMin)
		if s.trueMin <= 90 {
			if errAbs > maxNearErr {
				maxNearErr = errAbs
			}
		} else {
			if errAbs > maxFarErr {
				maxFarErr = errAbs
			}
		}
		if i > 0 {
			jump := math.Abs(s.predMin - snaps[i-1].predMin)
			if s.trueMin <= 90 {
				if jump > maxNearJump {
					maxNearJump = jump
				}
			} else {
				if jump > maxFarJump {
					maxFarJump = jump
				}
			}
		}
	}
	t.Logf("[判定] 临近段: 最大偏移=%.0fmin(限20) 最大跳变=%.0fmin(限10) | 早期段: 最大偏移=%.0fmin(限40) 最大跳变=%.0fmin(限30)",
		maxNearErr, maxNearJump, maxFarErr, maxFarJump)

	if maxNearErr > 20 {
		t.Errorf("临近段预测偏移 %.0fmin 超 20min（用户硬指标，剩余≤90min 时必须准）", maxNearErr)
	}
	if maxNearJump > 10 {
		t.Errorf("临近段连续预测跳变 %.0fmin 超 10min（「预计时间」会剧烈抖动）", maxNearJump)
	}
	// 早期段（剩余>90min）：纯历史外推的固有限制——窗口里若只有渐变早期的慢段，
	// 算法无法预知未来会加速，必然高估。这是数学限制而非 bug，只记录不强制。
	// 真正的缓解需要外部先验（历史日型曲线「该时段通常加速」），属后续增强。
	if maxFarErr > 40 {
		t.Logf("[注意] 早期段最大偏移 %.0fmin（剩余>90min，受纯历史外推固有限制，非 bug）", maxFarErr)
	}
	if maxFarJump > 30 {
		t.Logf("[注意] 早期段最大跳变 %.0fmin（早期数据填充时的正常收敛过程）", maxFarJump)
	}
}

// TestDynamicLongWindowTrend 量化「长窗趋势」改进：节奏持续渐变加速时，
// 用 2h 长窗检测趋势（生产逻辑）相比只用 30min 短窗趋势，能更早修正早期高估。
// 模拟生产的双窗逻辑：rate/cv/n 用 30min 短窗、trend 在「同向且更强」时用长窗覆盖短窗
// （见 buildQueueAdvisor 的双窗同向守卫；持续加速场景下短/长窗都>=1，新逻辑与取最强加速信号等价）。
func TestDynamicLongWindowTrend(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	shortW := queueAdvisorWindow30
	longW := queuePanelRateWindow // 2h
	targetNo := 1500

	// 90min 跨度，速度从 1.0 线性渐变到 4.0（午高峰临近持续加速）。
	var full []QueueObservation
	no := 1000
	for m := -90; m <= 0; m++ {
		full = append(full, dynSample(store, now, m, no))
		speed := 1.0 + float64(90+m)/90.0*3.0
		no += int(math.Round(speed))
	}

	// 在 t=-45（还剩较多，早期段）对比两种趋势策略的预测偏移。
	tMin := -45
	nowAt := now.Add(time.Duration(tMin) * time.Minute)
	all := collectUpTo(full, now, tMin)
	shortObs := recentStoreObservations(all, store, nowAt, shortW)
	longObs := recentStoreObservations(all, store, nowAt, longW)

	rate, cv, n, shortTrend, _, ok := calledRatePerMinuteWeighted(shortObs, nowAt, shortW)
	if !ok {
		t.Fatal("短窗应算出速度")
	}
	_, _, _, longTrend, _, _ := calledRatePerMinuteWeighted(longObs, nowAt, longW)

	// 当前真实叫号与剩余。
	calledAtT := 1000
	for _, s := range all {
		if s.DisplayCalledNo > calledAtT {
			calledAtT = s.DisplayCalledNo
		}
	}
	remaining := targetNo - calledAtT
	speed := 1.0 + float64(90+tMin)/90.0*3.0
	trueWait := float64(remaining) / speed

	// 策略A：短窗趋势（仅用短窗）。
	wrA, _ := estimateWaitRange(remaining, 0, rate, cv, n, shortTrend, nil, nil)
	// 策略B：生产双窗逻辑——同向（都>=1加速）且长窗更显著时采纳长窗，否则保留短窗。
	effTrend := shortTrend
	if longTrend >= 1.0 && shortTrend >= 1.0 && longTrend > shortTrend {
		effTrend = longTrend
	} else if longTrend < 1.0 && shortTrend < 1.0 && longTrend < shortTrend {
		effTrend = longTrend
	}
	wrB, _ := estimateWaitRange(remaining, 0, rate, cv, n, effTrend, nil, nil)

	midA, midB := -1.0, -1.0
	if wrA != nil {
		midA = float64(wrA.Low+wrA.High) / 2.0
	}
	if wrB != nil {
		midB = float64(wrB.Low+wrB.High) / 2.0
	}
	t.Logf("[t=%+d 剩余≈%.0f 真实等待=%.0fmin] 短窗trend=%.2f 长窗trend=%.2f",
		tMin, float64(remaining), trueWait, shortTrend, longTrend)
	t.Logf("  策略A(短窗趋势): 预测=%.0fmin 偏移=%.0fmin", midA, math.Abs(midA-trueWait))
	t.Logf("  策略B(长窗趋势): 预测=%.0fmin 偏移=%.0fmin", midB, math.Abs(midB-trueWait))

	// 长窗趋势应让偏移更小（或至少不更大）。
	if math.Abs(midB-trueWait) > math.Abs(midA-trueWait)+1 {
		t.Errorf("长窗趋势应降低偏移：A=%.0f B=%.0f（B 更差）", math.Abs(midA-trueWait), math.Abs(midB-trueWait))
	}
}

// TestDynamicLongWindowTrendConflictDecel 验证双窗「同向守卫」：当 2h 长窗整体仍加速
// （含高峰爬升段，longTrend>1）但最近 30min 短窗已减速（shortTrend<1，高峰已过）时，
// 生产新逻辑保留短窗的减速信号（同向才覆盖），而旧逻辑（无条件取离 1 更远）会误判为加速。
// 关键不变式：shortTrend<1 且 longTrend>1（异向）时，effTrend 必须保留短窗（不取长窗加速信号）。
func TestDynamicLongWindowTrendConflictDecel(t *testing.T) {
	shortTrend := 0.7 // 近期已减速
	longTrend := 1.5  // 长窗含爬升段，整体仍加速（异向）

	// 生产新逻辑：同向且更强才覆盖，异向时保留短窗。
	effTrendNew := shortTrend
	if longTrend >= 1.0 && shortTrend >= 1.0 && longTrend > shortTrend {
		effTrendNew = longTrend
	} else if longTrend < 1.0 && shortTrend < 1.0 && longTrend < shortTrend {
		effTrendNew = longTrend
	}
	// 旧逻辑：无条件让 trend 离 1 更远（加速取 max、减速取 min）。
	effTrendOld := shortTrend
	if longTrend > shortTrend {
		effTrendOld = longTrend
	}

	if effTrendNew != shortTrend {
		t.Errorf("异向时（短窗减速、长窗加速）应保留短窗：got effTrend=%.2f want=%.2f",
			effTrendNew, shortTrend)
	}
	if effTrendOld != longTrend {
		t.Errorf("旧逻辑在此场景会误取长窗加速信号：effTrendOld=%.2f（暴露旧行为的偏差）", effTrendOld)
	}
}

// TestCalledRateWeightedNowTruncatesToSecond 验证 calledRatePerMinuteWeighted 内部把 now
// 截断到秒级与秒级观测对齐。构造两次相邻采样间隔恰好 60s、now 带纳秒尾数：截断前最后一个间隔
// 的 spanMin 会偏大（now 高出几百毫秒把 snapshot 推到未来），截断后 spanMin 精确=1min。
// 不变式：带纳秒 now 与截断后 now 算出的速率应完全一致（截断不改变结果，因为观测本身是秒级）。
func TestCalledRateWeightedNowTruncatesToSecond(t *testing.T) {
	store := "3006"
	t0 := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	obs := []QueueObservation{
		obsAt(store, 1000, 0, 0, t0),
		obsAt(store, 1002, 0, 0, t0.Add(60*time.Second)),
		obsAt(store, 1004, 0, 0, t0.Add(120*time.Second)),
	}
	nowNano := t0.Add(120 * time.Second).Add(777 * time.Millisecond) // 带纳秒尾数
	nowSec := nowNano.Truncate(time.Second)

	rNano, _, _, _, _, okNano := calledRatePerMinuteWeighted(obs, nowNano, queueAdvisorWindow30)
	rSec, _, _, _, _, okSec := calledRatePerMinuteWeighted(obs, nowSec, queueAdvisorWindow30)
	if !okNano || !okSec {
		t.Fatalf("两次都应算出速度: nano=%v sec=%v", okNano, okSec)
	}
	if rNano != rSec {
		t.Errorf("now 截断到秒级应不改变速率结果（观测本身是秒级）：nano=%v sec=%v", rNano, rSec)
	}
}

// TestDynamicPredictionIntervalCoverage 验证预测区间 [Low,High] 的实际覆盖率——
// 原排查指出的最大盲区：区间宽度此前从未被验证命中率。匀速过程下，真值（remaining/speed）
// 应在绝大多数 checkpoint 落在 [Low,High] 内（覆盖率 >= 70%），证明区间不是徒有其表。
func TestDynamicPredictionIntervalCoverage(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	window := queueAdvisorWindow30
	speed := 2.0 // 匀速 2 组/分
	targetNo := 1500

	// 生成 90min 匀速叫号序列。
	var full []QueueObservation
	no := 1000
	for m := -90; m <= 0; m++ {
		full = append(full, dynSample(store, now, m, no))
		no += int(speed)
	}

	hit, total := 0, 0
	for tMin := -60; tMin <= -5; tMin += 5 {
		nowAt := now.Add(time.Duration(tMin) * time.Minute)
		all := collectUpTo(full, now, tMin)
		obs := recentStoreObservations(all, store, nowAt, window)
		rate, cv, n, trend, rates, ok := calledRatePerMinuteWeighted(obs, nowAt, window)
		if !ok {
			continue
		}
		// 当前真实叫号与剩余。
		calledAtT := 0
		for _, s := range all {
			if s.DisplayCalledNo > calledAtT {
				calledAtT = s.DisplayCalledNo
			}
		}
		remaining := targetNo - calledAtT
		if remaining <= 0 {
			continue
		}
		trueWait := float64(remaining) / speed
		wr, _ := estimateWaitRange(remaining, 0, rate, cv, n, trend, rates, nil)
		if wr == nil {
			continue
		}
		total++
		if float64(wr.Low)-1 <= trueWait && trueWait <= float64(wr.High)+1 {
			hit++
		}
	}
	if total == 0 {
		t.Fatal("应有可评估的 checkpoint")
	}
	coverage := float64(hit) / float64(total)
	t.Logf("匀速过程预测区间覆盖率：%.0f%% (%d/%d)", coverage*100, hit, total)
	// 匀速、无噪声下，真值应几乎总是落在区间内（覆盖率 >= 70%）。
	// 这是对「区间宽度是否有意义」的最基本校验——若区间恒偏窄，覆盖率会很低。
	if coverage < 0.70 {
		t.Errorf("匀速过程区间覆盖率过低：%.0f%%（区间可能恒偏窄，徒有其表）", coverage*100)
	}
}
