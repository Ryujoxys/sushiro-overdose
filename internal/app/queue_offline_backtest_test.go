package app

import (
	"math"
	"sort"
	"testing"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

// queueSpeedBacktestDateType 用应用的日期类型规则（周末窗口）判断，节假日与调休沿用内置历史的标注。
func queueSpeedBacktestDateType(anchor recordViewSample) func(time.Time) string {
	return func(t time.Time) string {
		switch anchor.dateType {
		case "holiday", "workday":
			return anchor.dateType
		}
		return queueTrendDateType(t.In(SushiroTimezone), nil, nil)
	}
}

type queueSpeedBacktestResult struct {
	cases, unreached int
	mae, bias        float64
	medianAbs        float64
	covered          float64
}

func runQueueSpeedBacktest(t *testing.T, pack *historyBundle, cutoff time.Time, horizon time.Duration) queueSpeedBacktestResult {
	t.Helper()
	train := map[int][]recordViewSample{}
	test := map[int][]recordViewSample{}
	for id, samples := range pack.byStore {
		for _, s := range samples {
			if s.at.Before(cutoff) {
				train[id] = append(train[id], s)
			} else {
				test[id] = append(test[id], s)
			}
		}
	}
	trainPairs := map[int][]queueSpeedPair{}
	var all []queueSpeedPair
	for id, samples := range train {
		trainPairs[id] = queueSpeedPairs(samples)
		all = append(all, trainPairs[id]...)
	}
	global := buildQueueSpeedGlobal(all)
	var errs []float64
	result := queueSpeedBacktestResult{}
	signed, inside := 0.0, 0
	for id, samples := range test {
		profile := buildQueueSpeedProfile(id, trainPairs[id], global)
		reps := queueSpeedRepresentatives(samples)
		for _, a := range reps {
			if !a.open || a.called == nil {
				continue
			}
			target := queueSpeedBucketStart(a.at).Add(horizon)
			b, ok := reps[queueSpeedRepKey(id, target)]
			if !ok || !b.open || b.called == nil || *b.called <= *a.called || b.at.Format("2006-01-02") != a.at.Format("2006-01-02") {
				continue
			}
			remaining := *b.called - *a.called
			actual := b.at.Sub(a.at).Minutes()
			// 与训练同一规则：每分钟超过 12 号是号段跳变，不是叫号。
			if float64(remaining)/actual > queueSpeedMaxPerMinute {
				continue
			}
			result.cases++
			dateTypeAt := queueSpeedBacktestDateType(a)
			mid := profile.integrate(a.at, remaining, queueSpeedMean, dateTypeAt)
			if !mid.ok {
				result.unreached++
				errs = append(errs, queueSpeedHorizon.Minutes()-actual)
				signed += queueSpeedHorizon.Minutes() - actual
				continue
			}
			pred := mid.at.Sub(a.at).Minutes()
			errs = append(errs, math.Abs(pred-actual))
			signed += pred - actual
			early := profile.integrate(a.at, remaining, queueSpeedFast, dateTypeAt)
			late := profile.integrate(a.at, remaining, queueSpeedSlow, dateTypeAt)
			lateAt := late.at
			if !late.ok || lateAt.Before(mid.at) {
				lateAt = mid.at.Add(mid.at.Sub(early.at))
			}
			if early.ok && !b.at.Before(early.at) && !b.at.After(lateAt) {
				inside++
			}
		}
	}
	if result.cases == 0 {
		t.Fatal("backtest found no test cases")
	}
	sum := 0.0
	for _, e := range errs {
		sum += e
	}
	sort.Float64s(errs)
	result.mae = sum / float64(len(errs))
	result.bias = signed / float64(len(errs))
	result.medianAbs = errs[len(errs)/2]
	result.covered = float64(inside) / float64(result.cases)
	return result
}

// TestOfflineCalledAtBacktest 用 2026-08-15 前的内置历史训练、之后的数据检验：
// 以 t 时刻真实叫号为起点，推算 60 分钟后真实叫到的号几点叫到，平均绝对误差不超过 20 分钟。
func TestOfflineCalledAtBacktest(t *testing.T) {
	pack, err := loadBundledHistory()
	if err != nil {
		t.Fatal(err)
	}
	cutoff := time.Date(2026, 8, 15, 0, 0, 0, 0, SushiroTimezone)
	got := runQueueSpeedBacktest(t, pack, cutoff, 60*time.Minute)
	t.Logf("backtest 60m: cases=%d unreached=%d MAE=%.1f bias=%.1f median=%.1f covered=%.2f", got.cases, got.unreached, got.mae, got.bias, got.medianAbs, got.covered)
	if got.mae > 20 {
		t.Fatalf("60-minute MAE = %.1f minutes, want <= 20", got.mae)
	}
}
