package app

// queue_advisor_accuracy_test.go —— 算法准确性验证（非契约测试）。
//
// 这组测试不是验证「函数签名/边界」，而是验证「算法算得准不准」：
// 用确定性的叫号模拟器生成观测序列，让新算法（A时间加权+B CV区间+C融合+D IQR）
// 预测 ETA，再和解析真值对比，量化各优化是否真的让预测更准。
//
// 每个场景在失败时打印详细数值（不直接 fail），便于人眼审视偏差大小。

import (
	"testing"
	"time"
)

// scenarioObs 按给定的(相对分钟偏移, 叫号)序列生成观测。now 是「当前」时刻。
func scenarioObs(store string, now time.Time, points []struct {
	min int // 距 now 的分钟数（负数=过去）
	no  int // 该时刻的叫号
}) []QueueObservation {
	out := make([]QueueObservation, len(points))
	for i, p := range points {
		out[i] = obsAt(store, p.no, 0, 0, now.Add(time.Duration(p.min)*time.Minute))
	}
	return out
}

func reportScenario(t *testing.T, name string, rate float64, cv float64, n int, trend float64,
	waitGroups int, officialWait, waitCap int, hist *QueueWaitRange,
	now time.Time, trueWaitMin float64, trueCurrentRate float64) {
	t.Helper()

	// 用新算法端到端算 ETA（含融合/区间/钳制）。
	// targetNo = calledNo + waitGroups，让 remaining=waitGroups。
	calledNo := 1000
	targetNo := calledNo + waitGroups
	eta := computeQueueEta(targetNo, calledNo, 0, officialWait, waitCap, rate, cv, n, trend, nil, hist, now)

	// 单独看 estimateWaitRange 的 base（无 officialWait/cap 干扰）。
	wr, src := estimateWaitRange(waitGroups, 0, rate, cv, n, trend, nil, hist)

	// 解析预测区间和中点的等待分钟。
	midWait := -1.0
	if wr != nil {
		midWait = float64(wr.Low+wr.High) / 2.0
	}
	errPct := -1.0
	if midWait > 0 && trueWaitMin > 0 {
		errPct = (midWait - trueWaitMin) / trueWaitMin * 100
	}

	t.Logf("[%s] 真实等待≈%.0fmin | 真实当前速度=%.2f组/分 | 新算法: 速度=%.2f cv=%.2f n=%d base中点=%.0fmin 区间=[%d,%d] src=%s | 中点误差=%+.1f%% | 出发建议=%q",
		name, trueWaitMin, trueCurrentRate, rate, cv, n, midWait, lowOr(wr), highOr(wr), src, errPct, eta.ArrivalSuggestion)

	// 判定阈值（宽松，因为这些是真实噪声场景，不是数学恒等式）：
	if errPct > 25 || errPct < -25 {
		t.Errorf("[%s] 预测中点误差 %.1f%% 超出 ±25%%", name, errPct)
	}
}

func lowOr(wr *QueueWaitRange) int {
	if wr == nil {
		return -1
	}
	return wr.Low
}
func highOr(wr *QueueWaitRange) int {
	if wr == nil {
		return -1
	}
	return wr.High
}

// TestAccuracyOverall 对比新算法 vs 等权基线，验证各优化整体上让预测更准。
func TestAccuracyOverall(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	window := queueAdvisorWindow30 // 30min，与 buildQueueAdvisor 的 ETA 速度窗一致

	// ---------- 场景1：匀速 2组/分，还剩 60 组 → 真实等待 30min ----------
	{
		pts := []struct{ min, no int }{}
		for m := -27; m <= 0; m += 3 { // 每3分钟一个点（30min窗内）
			pts = append(pts, struct{ min, no int }{m, 1000 + (27+m)*2})
		}
		obs := scenarioObs(store, now, pts)
		obs = recentStoreObservations(obs, store, now, window)
		rate, cv, n, trend, _, ok := calledRatePerMinuteWeighted(obs, now, window)
		if !ok {
			t.Fatal("匀速场景应算出速度")
		}
		reportScenario(t, "匀速2组/分", rate, cv, n, trend, 60, 0, 0, nil, now, 30, 2.0)
		if !approxEqual(rate, 2.0, 0.3) {
			t.Errorf("匀速场景速度应≈2.0，实际 %.3f", rate)
		}
	}

	// ---------- 场景2：先慢后快（30min窗内）。[-25,-10] 1组/分、[-9,0] 4组/分 ----------
	// 真实「当前」速度=4组/分。还剩 40 组 → 真实等待 10min。
	// 等权会被慢段拖低；时间加权（指数衰减，半衰期5min）应更接近 4。
	// 用 recentStoreObservations 按 window 过滤，与 buildQueueAdvisor 真实路径一致。
	{
		pts := []struct{ min, no int }{}
		for m := -25; m <= -10; m += 5 { // 前15min: 15min叫15号 → 1组/分
			pts = append(pts, struct{ min, no int }{m, 1000 + (25+m)*1})
		}
		for m := -9; m <= 0; m++ { // 近10min: 10min叫40号 → 4组/分
			pts = append(pts, struct{ min, no int }{m, 1015 + (10+m)*4})
		}
		obs := scenarioObs(store, now, pts)
		obs = recentStoreObservations(obs, store, now, window) // 按窗过滤（真实路径）
		rate, cv, n, trend, _, ok := calledRatePerMinuteWeighted(obs, now, window)
		if !ok {
			t.Fatal("先慢后快场景应算出速度")
		}
		reportScenario(t, "先慢后快", rate, cv, n, trend, 40, 0, 0, nil, now, 10, 4.0)
		// 加权应偏向近期4组/分，明显 > 等权均值。
		if rate < 3.0 {
			t.Errorf("时间加权应偏向近期4组/分，rate=%.3f 偏低（未体现A优化）", rate)
		}
	}

	// ---------- 场景3：补号跳变。匀速2组/分，中间一次跳50号 ----------
	// 真实速度=2组/分，跳变是噪声。还剩 60 组 → 真实等待 30min。
	// 无 D 时跳变会拉高速度、低估等待；有 D 应剔除，速度回到≈2。
	{
		pts := []struct{ min, no int }{}
		for m := -27; m <= 0; m += 3 {
			no := 1000 + (27+m)*2
			pts = append(pts, struct{ min, no int }{m, no})
		}
		// 在 -15min 处插入一次跳变 +50（模拟补号）。
		for i := range pts {
			if pts[i].min == -15 {
				for j := i; j < len(pts); j++ {
					pts[j].no += 50
				}
				break
			}
		}
		obs := scenarioObs(store, now, pts)
		obs = recentStoreObservations(obs, store, now, window)
		rate, cv, n, trend, _, ok := calledRatePerMinuteWeighted(obs, now, window)
		if !ok {
			t.Fatal("补号跳变场景应算出速度")
		}
		reportScenario(t, "补号跳变", rate, cv, n, trend, 60, 0, 0, nil, now, 30, 2.0)
		// 跳变不应让速度远超 2.5（无 D 时会到 3~4）。
		if rate > 2.8 {
			t.Errorf("补号跳变应被 IQR 剔除，速度=%.3f 仍偏高（D 未生效）", rate)
		}
	}

	// ---------- 场景4：窗口内停滞（30min窗）。[-27,-15]叫号、[-15,-3]停滞、[-3,0]恢复 ----------
	// 真实有效速度=2组/分（叫号段）。停滞段的 diff<=0 间隔被过滤，速度不被稀释。
	// 还剩 40 组 → 真实等待 20min。
	{
		pts := []struct{ min, no int }{}
		// [-27,-15]: 12min叫24号 → 2组/分，叫到 1024
		for m := -27; m <= -15; m += 3 {
			pts = append(pts, struct{ min, no int }{m, 1000 + (27+m)*2})
		}
		// [-15,-3]: 停滞12min，叫号卡在 1024
		for m := -12; m <= -3; m += 3 {
			pts = append(pts, struct{ min, no int }{m, 1024})
		}
		// [-3,0]: 恢复叫号 2组/分
		for m := -2; m <= 0; m++ {
			pts = append(pts, struct{ min, no int }{m, 1024 + (2+m)*2})
		}
		obs := scenarioObs(store, now, pts)
		obs = recentStoreObservations(obs, store, now, window)
		rate, cv, n, trend, _, ok := calledRatePerMinuteWeighted(obs, now, window)
		if !ok {
			t.Fatal("停滞场景应算出速度")
		}
		reportScenario(t, "中途停滞", rate, cv, n, trend, 40, 0, 0, nil, now, 20, 2.0)
		// 加权速度应 > 1.0（不被停滞稀释）。
		if rate < 1.2 {
			t.Errorf("中途停滞不应被稀释，加权速度=%.3f 偏低", rate)
		}
	}

	// ---------- 场景5：小样本 + 历史。实时只有2个点(刚开采样)，历史说该时段≈40min ----------
	// 真实等待应以历史为主锚。还剩 80 组。
	// 实时2个点间隔5min叫10号→实时速度2组/分→纯实时会算40min；历史P50/P80=[35,50]中位42.5→隐含速度80/42.5≈1.88。
	// 两者接近，融合后应≈40min，不会因小样本被偶发带偏。
	{
		obs := []QueueObservation{
			obsAt(store, 1000, 80, 40, now.Add(-5*time.Minute)),
			obsAt(store, 1010, 75, 38, now),
		}
		rate, cv, n, trend, _, ok := calledRatePerMinuteWeighted(obs, now, window)
		if !ok {
			t.Fatal("小样本场景应算出速度")
		}
		hist := &QueueWaitRange{Low: 35, High: 50}
		// n=2 → sampleW 小 → 历史权重高。真实等待≈40。
		reportScenario(t, "小样本+历史", rate, cv, n, trend, 80, 40, 0, hist, now, 40, 2.0)
		// source 应体现融合（realtimeN=2 → w=(2-1)/9 * (1-cv/0.5)；cv 在2点时为0或很小）。
		_, src := estimateWaitRange(80, 40, rate, cv, n, trend, nil, hist)
		if src == "unknown" {
			t.Errorf("小样本+历史应给出融合结果，非 unknown")
		}
	}
}

// TestAccuracyWeightedVsEqual 直接量化：时间加权(A)相比等权在「速度变化」场景下的改进。
func TestAccuracyWeightedVsEqual(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	window := 60 * time.Minute

	// 60min窗：真实速度从 1组/分 线性加速到 5组/分（叫号越来越快）。
	// 「当前」真实速度≈5。等权会被早期的慢速拖低；时间加权应更接近 5。
	pts := []struct{ min, no int }{}
	called := 1000
	for m := -60; m <= 0; m += 2 {
		// 速度从1线性增到5：每2分钟推进 (1 + (60+m)/60*4)*2 个号
		speed := 1.0 + float64(60+m)/60.0*4.0
		called += int(speed * 2)
		pts = append(pts, struct{ min, no int }{m, called})
	}
	obs := scenarioObs(store, now, pts)

	rateWeighted, _, _, _, _, ok := calledRatePerMinuteWeighted(obs, now, window)
	if !ok {
		t.Fatal("应算出速度")
	}

	// 等权基线：首尾两点法（整个窗口的平均）。
	eqRate := float64(obs[len(obs)-1].DisplayCalledNo-obs[0].DisplayCalledNo) / 60.0

	t.Logf("加速场景：真实当前速度≈5.0 | 时间加权=%.3f | 等权(首尾两点)=%.3f", rateWeighted, eqRate)
	// 时间加权应比等权更接近 5.0（更高）。
	if rateWeighted <= eqRate+0.3 {
		t.Errorf("时间加权(%.3f)应明显高于等权(%.3f)，更接近真实当前速度5.0", rateWeighted, eqRate)
	}
	// 两者离真值的距离：加权应更近。
	distW := abs(rateWeighted - 5.0)
	distE := abs(eqRate - 5.0)
	t.Logf("离真值5.0的距离：加权=%.3f 等权=%.3f → 加权更准=%v", distW, distE, distW < distE)
	if distW >= distE {
		t.Errorf("时间加权离真值更远（加权=%.3f 等权=%.3f），A优化未体现", distW, distE)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
