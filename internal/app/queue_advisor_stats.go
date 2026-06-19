package app

import "math"

// queue_advisor_stats.go 存放排队预测(ETA)的纯统计函数：异常剔除、CV→区间宽度映射、
// 实时/历史融合权重。全部无副作用、不读磁盘，便于单测。
//
// CV 约定：变异系数（标准差/均值），无量纲，描述叫号速度的离散度。
// 当有效样本不足以算标准差时，调用方传入 cv=-1（哨兵），下游一律走默认/退化。

// filterOutlierRates 用四分位距(IQR)剔除叫号瞬时速率中的离群点（如一次性补号跳变：
// 一次叫号推进 50 个号、瞬时速率 100 组/分，会严重拉高均值）。纯函数，便于单测。
//
// 规则：
//   - n < 4：不剔（小样本下 IQR 极不稳，误剔风险高于收益），原样返回。
//   - n >= 4：算 Q1/Q3/IQR，丢弃 r < Q1-1.5*IQR 或 r > Q3+1.5*IQR 的点。
//   - IQR==0 特判（≥25% 样本相同时 fence 坍缩成 Q1，会把正常波动全剔光）：
//     改用均值做尺度，只剔偏离均值超过 3*mean 的点。
//   - 剔除过狠兜底：若剔除后剩余 <2 个点（极端分散），回退返回原 rates，宁可不剔也不丢数据。
func filterOutlierRates(rates []float64) []float64 {
	if len(rates) < 4 {
		return rates
	}
	lo, hi := iqrFences(rates)
	if math.IsNaN(lo) || math.IsNaN(hi) {
		return rates // 无法计算 fence（如全负），放弃剔除。
	}
	filtered := make([]float64, 0, len(rates))
	for _, r := range rates {
		if r >= lo && r <= hi {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) < 2 {
		return rates // 剔除过狠，回退原值。
	}
	return filtered
}

// iqrFences 算 IQR 离群点上下界。IQR==0 时改用均值做尺度（只剔偏离均值超过 3*mean 的点）。
// 返回 NaN 表示无法计算（如均值<=0）。n<4 时返回 (±Inf,∓Inf) 表示不剔除。
func iqrFences(rates []float64) (lo, hi float64) {
	n := len(rates)
	if n < 4 {
		return math.Inf(-1), math.Inf(1)
	}
	q1 := queueQuantile(rates, 0.25)
	q3 := queueQuantile(rates, 0.75)
	iqr := q3 - q1
	if iqr > 0 {
		return q1 - 1.5*iqr, q3 + 1.5*iqr
	}
	// IQR==0：用均值做尺度。
	sum := 0.0
	for _, r := range rates {
		sum += r
	}
	mean := sum / float64(n)
	if mean <= 0 {
		return math.NaN(), math.NaN()
	}
	return mean - 3*mean, mean + 3*mean
}

// shrinkTrend 对观测到的速度趋势比做均值回归收缩：trend_post = λ·trend + (1-λ)·1。
// λ 随 realtimeN 增长（样本少→λ 小→收缩回 1，即「不信趋势」；样本足→λ 大→接近观测 trend）。
//
// 动机：叫号节奏的「加速/减速」常是确定性日周期（饭点加速、峰后反转），而非持续漂移。
// 直接用 sqrt(trend) 外推会把周期性加速当成持续加速，系统性高估未来速度（ETA 偏早）。
// 收缩让小样本/近期偶发抖动不至于扭曲预测，样本累积后才逐步采信趋势信号。
//
// 阈值对齐 rateTrendRatio 自身门槛（n>=4 才有非 1 的 trend，且 estimateWaitRange 守卫也要求
// realtimeN>=4）：n=4→λ≈0.11（刚过门槛，谨慎采信）、n>=12→λ=1（完全采信）。中间线性过渡。
func shrinkTrend(trend float64, realtimeN int) float64 {
	if trend == 1.0 {
		return 1.0
	}
	lam := float64(realtimeN-3) / 9.0
	if lam < 0 {
		lam = 0
	}
	if lam > 1 {
		lam = 1
	}
	return lam*trend + (1-lam)*1.0
}

// quantileWaitBounds 用 IQR 剔除后的瞬时速率序列 rates 算经验分位数等待区间。
// base 是已融合历史先验/趋势校正后的中心估计（remaining/effRate）。区间宽度由 rates 的相对离散度
// 驱动：快档 q90（速度高→等待短）→ 下界，慢档 q10（速度低→等待长）→ 上界。
// 以 q50（中位速度）做归一化：low=base*(q50/q90)、high=base*(q50/q10)。这样区间中心锚在 base，
// 形状（右偏、非对称）来自实测速率分布，而非对称的「均值±k·σ」查表。
//
// 返回 ok=false 的情况：rates<3（分位数不稳）、q10/q50 非正（脏数据）。调用方应回退 CV 查表。
func quantileWaitBounds(base float64, rates []float64) (low, high float64, ok bool) {
	if len(rates) < 3 || base <= 0 {
		return 0, 0, false
	}
	q10 := queueQuantile(rates, 0.10)
	q50 := queueQuantile(rates, 0.50)
	q90 := queueQuantile(rates, 0.90)
	if q10 <= 0 || q50 <= 0 || q90 <= 0 {
		return 0, 0, false
	}
	// q10<=q50<=q90（速度分位数单调）。等待时间 = base * (q50/q_rate)，故：
	//   low  = base * (q50/q90) <= base（快档：等待短）
	//   high = base * (q50/q10) >= base（慢档：等待长）
	low = base * (q50 / q90)
	high = base * (q50 / q10)
	if high < low {
		low, high = high, low // 防御性：脏数据导致反转
	}
	return low, high, true
}

// waitRangeMultipliers 把叫号速度的变异系数 CV 映射成等待区间的下界/上界系数。
// CV 越大（叫号忽快忽慢）→ 区间越宽，覆盖不确定性；CV 越小（几乎匀速）→ 区间收窄，
// 给更紧的出发窗口。cv<0（哨兵：样本不足）走默认档。
//
// 返回 (lowMul, highMul)，满足 lowMul<=1、highMul>=1。
func waitRangeMultipliers(cv float64) (float64, float64) {
	switch {
	case cv < 0 || cv < 0.15:
		// 样本不足或很稳定：默认/收窄。
		// 注意 cv<0 是哨兵，应走默认 0.85/1.20（与历史行为一致）；cv∈[0,0.15) 才收窄。
		if cv < 0 {
			return 0.85, 1.20
		}
		return 0.90, 1.10
	case cv < 0.30:
		return 0.85, 1.20
	case cv < 0.50:
		return 0.80, 1.35
	default:
		return 0.75, 1.50
	}
}

// realtimeBlendWeight 计算实时叫号速度在「实时 vs 历史先验」融合中的权重 w∈[0,1]。
// w=1 表示完全信实时，w=0 表示完全信历史。
//
// 综合两个可信度因子：
//   - 样本数：realtimeN 越多越可信（n=1→0，n>=10→满）。
//   - 稳定性：cv 越小越可信（cv>=0.5→0，cv=0→满）；cv<0（哨兵）视为不稳→0。
//
// hasHist=false 时强制 w=1：没有历史先验可锚，只能信实时（或退化）。
func realtimeBlendWeight(realtimeN int, cv float64, hasHist bool) float64 {
	if !hasHist {
		return 1.0
	}
	sampleW := float64(realtimeN-1) / 9.0
	if sampleW < 0 {
		sampleW = 0
	}
	if sampleW > 1 {
		sampleW = 1
	}
	var stabilityW float64
	if cv < 0 {
		stabilityW = 0
	} else {
		stabilityW = 1 - cv/0.5
		if stabilityW < 0 {
			stabilityW = 0
		}
		if stabilityW > 1 {
			stabilityW = 1
		}
	}
	return sampleW * stabilityW
}

// coefficientOfVariation 计算一组速率的变异系数 CV = σ/mean。
// n<2 时返回 -1（哨兵：样本不足以算标准差）。
func coefficientOfVariation(rates []float64) float64 {
	n := len(rates)
	if n < 2 {
		return -1
	}
	sum := 0.0
	for _, r := range rates {
		sum += r
	}
	mean := sum / float64(n)
	if mean <= 0 {
		return -1
	}
	var sq float64
	for _, r := range rates {
		d := r - mean
		sq += d * d
	}
	std := math.Sqrt(sq / float64(n-1)) // 样本标准差（n-1 分母）
	return std / mean
}
