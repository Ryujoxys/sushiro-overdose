package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func obsAt(storeID string, called, groups, wait int, at time.Time) QueueObservation {
	return QueueObservation{
		StoreID:          storeID,
		DisplayCalledNo:  called,
		GroupQueuesCount: groups,
		WaitMinutes:      wait,
		CollectedAt:      at.Format(time.RFC3339),
	}
}

func TestQueuePressureLevel(t *testing.T) {
	cases := []struct {
		groups, wait int
		want         string
	}{
		{0, 0, "unknown"},
		{10, 0, "low"},
		{0, 15, "low"},
		{50, 0, "medium"},
		{0, 55, "medium"},
		{100, 0, "high"},
		{0, 110, "high"},
		{200, 0, "extreme"},
		{0, 300, "extreme"},
	}
	for _, c := range cases {
		if got := queuePressureLevel(c.groups, c.wait); got != c.want {
			t.Errorf("queuePressureLevel(%d,%d)=%q want %q", c.groups, c.wait, got, c.want)
		}
	}
}

func TestQueuePressureScore(t *testing.T) {
	if s := queuePressureScore(0, 0); s != 0 {
		t.Errorf("empty score=%d want 0", s)
	}
	if s := queuePressureScore(500, 0); s != 100 {
		t.Errorf("huge score=%d want capped 100", s)
	}
	low := queuePressureScore(20, 0)
	high := queuePressureScore(120, 0)
	if !(low < high) {
		t.Errorf("score not monotonic: low=%d high=%d", low, high)
	}
}

func TestBuildRemoteQueuePressureCurvePoints(t *testing.T) {
	storeID := "3006"
	baseline := QueueBaselineExport{
		Rollups: []QueueBaselineRollup{
			{
				StoreID:            3006,
				DateType:           "weekday",
				TimeBucket:         "10:00",
				SampleCount:        12,
				WaitTypicalMinutes: floatPtr(55),
				QueueGroupsTypical: floatPtr(32),
				CalledSampleCount:  10,
				CalledNoTypical:    floatPtr(120),
			},
			{
				StoreID:            3006,
				DateType:           "weekend",
				TimeBucket:         "10:00",
				SampleCount:        12,
				WaitTypicalMinutes: floatPtr(120),
				QueueGroupsTypical: floatPtr(90),
				CalledSampleCount:  10,
				CalledNoTypical:    floatPtr(900),
			},
			{
				StoreID:            3006,
				DateType:           "weekday",
				TimeBucket:         "10:30",
				SampleCount:        20,
				WaitTypicalMinutes: floatPtr(70),
				QueueGroupsTypical: floatPtr(64),
				CalledSampleCount:  18,
				CalledNoTypical:    floatPtr(180),
			},
		},
		Latest: []QueueBaselineLatest{
			{
				StoreID:          3006,
				CollectedAt:      "2026-06-08T11:03:00+08:00",
				WaitMinutes:      80,
				GroupQueuesCount: 70,
				DisplayCalledNo:  220,
			},
			{
				StoreID:          3006,
				CollectedAt:      "2026-06-09T11:03:00+08:00",
				WaitMinutes:      10,
				GroupQueuesCount: 5,
				DisplayCalledNo:  20,
			},
		},
	}

	points := buildRemoteQueuePressureCurvePoints(storeID, "2026-06-08", "weekday", baseline)
	if len(points) != 3 {
		t.Fatalf("points len=%d want 3: %+v", len(points), points)
	}
	if points[0].Time != "10:00" || points[0].CalledNo != 120 || points[0].WaitingGroups != 32 || points[0].OfficialWaitMinutes != 55 {
		t.Fatalf("first point mismatch: %+v", points[0])
	}
	if points[0].Source != "remote_baseline" || points[0].Confidence == "" {
		t.Fatalf("first point missing source/confidence: %+v", points[0])
	}
	if points[2].Time != "11:03" || points[2].Source != "remote_latest" || points[2].CalledNo != 220 {
		t.Fatalf("latest point mismatch: %+v", points[2])
	}
}

func TestBuildQueuePressureCurveExplainsConnectedCloudWithNoStoreData(t *testing.T) {
	resetQueueBaselineRemoteCacheForTest(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv(queueBaselineTursoURLEnv, "")
	t.Setenv(queueBaselineTursoTokenEnv, "")
	t.Setenv(queueBaselineTursoFallbackURL, "")
	t.Setenv(queueBaselineTursoFallbackAuth, "")
	t.Setenv(cloudAuthURLEnv, "")
	t.Setenv(cloudAuthSessionTokenEnv, "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/queue/baseline/store" {
			t.Fatalf("path = %s, want /api/queue/baseline/store", r.URL.Path)
		}
		writeJSON(w, QueueBaselineExport{
			Version:       1,
			GeneratedAt:   "2026-06-08T10:00:00+08:00",
			Source:        "turso-cloudflare",
			BucketMinutes: 10,
			DateTypes:     []string{"weekday"},
			Stats:         QueueBaselineStats{},
		})
	}))
	defer server.Close()
	if err := SaveCloudAuthConfig(CloudAuthConfig{BaseURL: server.URL, SessionToken: "cloud-session", UserLogin: "octocat"}); err != nil {
		t.Fatal(err)
	}

	curve := buildQueuePressureCurve(context.Background(), "3006", "2026-06-08", time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC))

	if curve.Baseline.Provider != "cloudflare" || !curve.Baseline.Used {
		t.Fatalf("baseline = %+v, want used cloudflare", curve.Baseline)
	}
	if !strings.Contains(curve.Message, "线上 Turso 基准已连接") {
		t.Fatalf("message = %q, want connected cloud no-data explanation", curve.Message)
	}
}

func TestMergeQueuePressureCurvePointsPrefersLocal(t *testing.T) {
	remote := []QueuePressureCurvePoint{{
		Time:          "10:00",
		CalledNo:      100,
		Source:        "remote_baseline",
		PressureLevel: "medium",
	}}
	local := []QueuePressureCurvePoint{{
		Time:          "10:00",
		CalledNo:      140,
		Source:        "local",
		PressureLevel: "low",
	}}

	points := mergeQueuePressureCurvePoints(remote, local)
	if len(points) != 1 {
		t.Fatalf("merged len=%d want 1: %+v", len(points), points)
	}
	if points[0].Source != "local" || points[0].CalledNo != 140 {
		t.Fatalf("local point should win: %+v", points[0])
	}
}

func TestCalledRateOverWindow(t *testing.T) {
	now := time.Now()
	store := "3006"
	// 30 分钟推进 60 个号 → 2 桌/分钟。
	obs := []QueueObservation{
		obsAt(store, 100, 50, 60, now.Add(-30*time.Minute)),
		obsAt(store, 160, 40, 50, now),
	}
	rate := calledRateOverWindow(obs, store, now, queueAdvisorWindow60)
	if rate == nil {
		t.Fatal("expected rate, got nil")
	}
	if *rate < 1.9 || *rate > 2.1 {
		t.Errorf("rate=%.2f want ~2.0", *rate)
	}

	// 只有一个点 → nil。
	one := []QueueObservation{obsAt(store, 100, 50, 60, now)}
	if r := calledRateOverWindow(one, store, now, queueAdvisorWindow60); r != nil {
		t.Errorf("single point should be nil, got %v", *r)
	}

	// 叫号回退（队列重置）→ diff<=0 → nil。
	back := []QueueObservation{
		obsAt(store, 900, 50, 60, now.Add(-20*time.Minute)),
		obsAt(store, 50, 40, 50, now),
	}
	if r := calledRateOverWindow(back, store, now, queueAdvisorWindow60); r != nil {
		t.Errorf("rollback should be nil, got %v", *r)
	}
}

func TestQueuePressureTrend(t *testing.T) {
	now := time.Now()
	store := "3006"
	improving := []QueueObservation{
		obsAt(store, 100, 80, 90, now.Add(-15*time.Minute)),
		obsAt(store, 140, 60, 70, now),
	}
	if level, _ := queuePressureTrend(improving, 2.0, false); level != "improving" {
		t.Errorf("improving trend=%q", level)
	}
	worsening := []QueueObservation{
		obsAt(store, 100, 60, 70, now.Add(-15*time.Minute)),
		obsAt(store, 110, 90, 100, now),
	}
	if level, _ := queuePressureTrend(worsening, 1.0, false); level != "worsening" {
		t.Errorf("worsening trend=%q", level)
	}
	if level, _ := queuePressureTrend(improving, 2.0, true); level != "stalled" {
		t.Errorf("stalled override trend=%q", level)
	}
	if level, _ := queuePressureTrend(nil, 0, false); level != "unknown" {
		t.Errorf("no data trend=%q", level)
	}
}

func TestEstimateWaitRange(t *testing.T) {
	// cv=-1（样本不足哨兵）→ 走默认宽度 0.85/1.20，数值与历史行为一致。
	const cvDefault = -1.0
	// 速度优先。
	if wr, src := estimateWaitRange(100, 0, 2.0, cvDefault, 0, 1.0, nil, nil); wr == nil || src != "recent_speed" {
		t.Fatalf("speed range nil/src=%q", src)
	}
	// 速度 + 官方等待：用户不可能比门店整体更快叫到，官方等待抬低下界而非压缩。
	// remaining=178/rate=2 → base≈89，low=floor(89*0.85)=75，high=ceil(89*1.2)=107。
	// officialWait=120 高于本机估算区间 → low/high 都被抬到 120。
	if wr, src := estimateWaitRange(178, 120, 2.0, cvDefault, 0, 1.0, nil, nil); wr == nil || src != "recent_speed" {
		t.Fatalf("official+speed range nil/src=%q", src)
	} else if wr.Low < 120 || wr.High < 120 {
		t.Errorf("officialWait should raise floor (max), got Low=%d High=%d want >=120", wr.Low, wr.High)
	}
	// officialWait 远小于本机估算（号码靠后）时不应压缩 low。
	if wr, _ := estimateWaitRange(178, 30, 2.0, cvDefault, 0, 1.0, nil, nil); wr == nil || wr.Low < 75 {
		t.Errorf("small officialWait must not shrink low, got Low=%d want >=75", wr.Low)
	}
	// 无速度退化到官方。
	wr, src := estimateWaitRange(0, 60, 0, cvDefault, 0, 1.0, nil, nil)
	if wr == nil || src != "official" {
		t.Fatalf("official range nil/src=%q", src)
	}
	// 无速度无官方退化到历史。
	hist := &QueueWaitRange{Low: 40, High: 70}
	wr2, src2 := estimateWaitRange(0, 0, 0, cvDefault, 0, 1.0, nil, hist)
	if wr2 == nil || src2 != "history" || wr2.Low != 40 || wr2.High != 70 {
		t.Fatalf("history range %+v src=%q", wr2, src2)
	}
	// 全缺 → unknown。
	if wr3, src3 := estimateWaitRange(0, 0, 0, cvDefault, 0, 1.0, nil, nil); wr3 != nil || src3 != "unknown" {
		t.Fatalf("unknown range %+v src=%q", wr3, src3)
	}
}

func TestComputeQueueEta(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 30, 0, 0, time.Local)
	const cv = -1.0 // 样本不足哨兵，走默认宽度
	const n = 0

	// 已轮到：targetNo==calledNo 是真的即将轮到。
	done := computeQueueEta(950, 950, 0, 90, 0, 2.0, cv, n, 1.0, nil, nil, now)
	if done.RemainingGroups != 0 || done.Risk != "low" {
		t.Errorf("done eta=%+v", done)
	}

	// 过号：targetNo<calledNo（哪怕只差一个号）也不能再说“已轮到、低风险”，
	// 寿司郎过号不会自动叫到，需补号或重新取号，与 dashboard 路径口径一致。
	passed := computeQueueEta(900, 950, 0, 90, 0, 2.0, cv, n, 1.0, nil, nil, now)
	if passed.Risk != "high" || !strings.Contains(passed.ArrivalSuggestion, "过号") {
		t.Errorf("passed eta should be high risk + 过号 hint, got %+v", passed)
	}
	passedBy1 := computeQueueEta(949, 950, 0, 90, 0, 2.0, cv, n, 1.0, nil, nil, now)
	if passedBy1.Risk != "high" || !strings.Contains(passedBy1.ArrivalSuggestion, "过号") {
		t.Errorf("passed-by-1 eta should be high risk + 过号 hint, got %+v", passedBy1)
	}

	// 正常：还差 178，速度 2/分 → ~89 分钟，带区间与出发建议。
	eta := computeQueueEta(1078, 900, 25, 90, 0, 2.0, cv, n, 1.0, nil, nil, now)
	if eta.RemainingGroups != 178 {
		t.Errorf("remaining=%d want 178", eta.RemainingGroups)
	}
	if eta.WaitMinutesRange == nil || eta.EstimatedCalledAtRange == nil {
		t.Fatalf("missing range: %+v", eta)
	}
	if eta.ArrivalSuggestion == "" {
		t.Error("missing arrival suggestion")
	}
	if eta.Risk == "unknown" {
		t.Error("risk should be resolved with speed source")
	}

	// 数据不足：无速度、无官方、无历史 → 无法预估。
	none := computeQueueEta(1078, 900, 0, 0, 0, 0, cv, n, 1.0, nil, nil, now)
	if none.EstimatedCalledAt != "" || none.WaitMinutesRange != nil {
		t.Errorf("insufficient data should not estimate: %+v", none)
	}

	// 只有官方等待：只能作为门店压力参考，不能包装成到号码的 ETA。
	officialOnly := computeQueueEta(1078, 900, 0, 90, 0, 0, cv, n, 1.0, nil, nil, now)
	if officialOnly.WaitMinutesRange != nil || officialOnly.Source != "official" || officialOnly.Risk != "high" {
		t.Fatalf("official-only ETA should not estimate target number: %+v", officialOnly)
	}
}

// TestComputeQueueEtaWaitCapGuard 验证官方等位封顶值（waitTimeCap）钳制离谱的高位估算。
func TestComputeQueueEtaWaitCapGuard(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	// 剩 200 桌、近窗速度偏慢 → 模型可能算出很高的高位；waitCap=180 应把高位钳到 180。
	// cv=-1、realtimeN=0 → 融合权重 w=0，走纯历史 source=history，base 经 soft floor 后高位仍超 180 触发钳制。
	eta := computeQueueEta(1100, 900, 0, 0, 180, 1.0, -1.0, 0, 1.0, nil, &QueueWaitRange{Low: 150, High: 320}, now)
	if eta.WaitMinutesRange == nil {
		t.Fatalf("应有等待区间")
	}
	if eta.WaitMinutesRange.High > 180 {
		t.Fatalf("高位 %d 超过官方封顶 180，应被钳制", eta.WaitMinutesRange.High)
	}
	if eta.SourceNote == "" {
		t.Fatalf("超过封顶时应给出 SourceNote 提示")
	}
	// waitCap=0 表示无封顶信息，不应触发钳制（SourceNote 不含封顶提示）。
	noCap := computeQueueEta(1100, 900, 0, 0, 0, 1.0, -1.0, 0, 1.0, nil, &QueueWaitRange{Low: 150, High: 320}, now)
	if strings.Contains(noCap.SourceNote, "官方等位封顶") {
		t.Fatalf("无封顶信息时不应给出封顶提示，实际 %q", noCap.SourceNote)
	}
}

func TestParseHHMM(t *testing.T) {
	now := time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local)
	for _, raw := range []string{"1210", "12:10"} {
		ts, ok := parseHHMM(raw, now)
		if !ok || ts.Hour() != 12 || ts.Minute() != 10 {
			t.Errorf("parseHHMM(%q)=%v ok=%v", raw, ts, ok)
		}
	}
	for _, bad := range []string{"", "9", "2510", "1260", "abcd"} {
		if _, ok := parseHHMM(bad, now); ok {
			t.Errorf("parseHHMM(%q) should fail", bad)
		}
	}
}

// TestQueuePressureCurveDateUsesSushiroTimezone 是 S5 回归：dateKey 必须按门店时区
// (UTC+8) 切片，不依赖机器本地时区。机器在 UTC 时，UTC 18:00 实为 CST 次日 02:00，
// raw 为空时默认日期应是 CST 的 2026-06-05，而非 UTC 的 2026-06-04。
func TestQueuePressureCurveDateUsesSushiroTimezone(t *testing.T) {
	utc := time.FixedZone("UTC", 0)
	// 2026-06-04 18:00 UTC == 2026-06-05 02:00 CST（跨日）。
	now := time.Date(2026, 6, 4, 18, 0, 0, 0, utc)
	dateKey, _ := queuePressureCurveDate("", now)
	if dateKey != "2026-06-05" {
		t.Fatalf("dateKey = %s, want 2026-06-05（按门店 UTC+8 切片，而非机器 UTC 当日）", dateKey)
	}
}

// TestQueuePressureCurveDateExplicitParamInSushiroTimezone 验证显式日期参数也按门店时区解释。
func TestQueuePressureCurveDateExplicitParamInSushiroTimezone(t *testing.T) {
	utc := time.FixedZone("UTC", 0)
	now := time.Date(2026, 6, 4, 18, 0, 0, 0, utc)
	dateKey, day := queuePressureCurveDate("2026-06-10", now)
	if dateKey != "2026-06-10" {
		t.Fatalf("dateKey = %s, want 2026-06-10", dateKey)
	}
	// 返回的 day 应落在门店时区（UTC+8），而非调用方的 UTC。用与 UTC 的偏移判断。
	_, offset := day.Zone()
	if offset != 8*60*60 {
		t.Fatalf("day zone offset = %d, want %d (CST UTC+8)", offset, 8*60*60)
	}
}

// ---------- 排队预测算法优化测试 (A 时间加权 / B CV动态区间 / C 实时历史融合 / D IQR剔除) ----------

func approxEqual(a, b, tol float64) bool { return a > b-tol && a < b+tol }

// D: IQR 异常剔除
func TestFilterOutlierRatesRemovesSpike(t *testing.T) {
	// [2,2,2,2,100]：100 是补号跳变离群点，应被剔除。
	got := filterOutlierRates([]float64{2, 2, 2, 2, 100})
	if len(got) != 4 {
		t.Fatalf("len=%d want 4 (spike removed): %v", len(got), got)
	}
	for _, v := range got {
		if v > 10 {
			t.Errorf("spike 100 should be removed, got %v", got)
		}
	}
}

func TestFilterOutlierRatesSmallNFallback(t *testing.T) {
	// n<4 不剔除（即使有离群点）。
	got := filterOutlierRates([]float64{2, 100})
	if len(got) != 2 {
		t.Fatalf("n<4 should not filter, got %v", got)
	}
}

func TestFilterOutlierRatesAllFilteredFallback(t *testing.T) {
	// 极端分散（每个点都落 fence 外）→ 剔除过狠 → 回退原值。
	extreme := []float64{1, 100, 10000, 1000000}
	got := filterOutlierRates(extreme)
	if len(got) < 2 {
		t.Fatalf("over-filtering should fall back to original, got %v", got)
	}
}

func TestFilterOutlierRatesIQRZero(t *testing.T) {
	// IQR=0（≥50% 样本相同导致 Q1==Q3）：特判用 mean 做尺度，只剔偏离 3*mean 的点。
	// [2,2,2,2,5]: Q1=Q3=2 → IQR=0 → mean=2.6, hiFence=2.6+7.8=10.4，5<10.4 → 保留。
	got := filterOutlierRates([]float64{2, 2, 2, 2, 5})
	if len(got) != 5 {
		t.Fatalf("IQR=0 mild outlier should be kept, got %v", got)
	}
	// [2,2,2,2,50]: mean=11.6, hiFence=11.6+34.8=46.4，50>46.4 → 剔除。
	got = filterOutlierRates([]float64{2, 2, 2, 2, 50})
	if len(got) != 4 {
		t.Fatalf("IQR=0 large outlier should be removed, got %v", got)
	}
}

// B: CV → 区间宽度系数
func TestWaitRangeMultipliers(t *testing.T) {
	cases := []struct {
		cv            float64
		wantLow, want float64
	}{
		{-1.0, 0.85, 1.20}, // 哨兵：默认
		{0.05, 0.90, 1.10}, // 很稳定：收窄
		{0.20, 0.85, 1.20}, // 正常：默认
		{0.40, 0.80, 1.35}, // 波动大：加宽
		{0.60, 0.75, 1.50}, // 极不稳：大幅加宽
	}
	for _, c := range cases {
		lo, hi := waitRangeMultipliers(c.cv)
		if !approxEqual(lo, c.wantLow, 1e-9) || !approxEqual(hi, c.want, 1e-9) {
			t.Errorf("cv=%.2f got (%.2f,%.2f) want (%.2f,%.2f)", c.cv, lo, hi, c.wantLow, c.want)
		}
		if lo > 1 || hi < 1 {
			t.Errorf("cv=%.2f multipliers violate low<=1<=high: (%.2f,%.2f)", c.cv, lo, hi)
		}
	}
}

func TestEstimateWaitRangeNarrowsWhenStable(t *testing.T) {
	// 稳定速率 cv<0.15 → high 系数 1.10，比默认 1.20 窄。
	stable, _ := estimateWaitRange(100, 0, 2.0, 0.05, 10, 1.0, nil, nil)
	def, _ := estimateWaitRange(100, 0, 2.0, -1.0, 10, 1.0, nil, nil) // -1 走默认 1.20
	if stable == nil || def == nil {
		t.Fatal("nil range")
	}
	if stable.High >= def.High {
		t.Errorf("stable cv should narrow high: stable=%d default=%d", stable.High, def.High)
	}
}

func TestEstimateWaitRangeWidensWhenVolatile(t *testing.T) {
	// 高 cv → high 系数 1.50，比默认宽。
	vol, _ := estimateWaitRange(100, 0, 2.0, 0.60, 10, 1.0, nil, nil)
	def, _ := estimateWaitRange(100, 0, 2.0, -1.0, 10, 1.0, nil, nil)
	if vol == nil || def == nil {
		t.Fatal("nil range")
	}
	if vol.High <= def.High {
		t.Errorf("volatile cv should widen high: volatile=%d default=%d", vol.High, def.High)
	}
}

// C: 实时/历史融合
func TestEstimateWaitRangeBlendedRealtimeDominant(t *testing.T) {
	// realtimeN 大、cv 小、有历史 → w≈1 → source=recent_speed，base≈waitGroups/rate。
	wr, src := estimateWaitRange(100, 0, 2.0, 0.1, 10, 1.0, nil, &QueueWaitRange{Low: 40, High: 80})
	if wr == nil || src != "recent_speed" {
		t.Fatalf("realtime-dominant should be recent_speed, got src=%q wr=%+v", src, wr)
	}
	// base=100/2=50，lowMul=0.90(稳定)→low≈45。
	if wr.Low > 50 {
		t.Errorf("realtime-dominant base should be ~50, low=%d", wr.Low)
	}
}

func TestEstimateWaitRangeBlendedHistoryDominant(t *testing.T) {
	// realtimeN=1（sampleW=0）→ w=0 → source=history，base≈waitGroups/histMid。
	wr, src := estimateWaitRange(100, 0, 2.0, 0.1, 1, 1.0, nil, &QueueWaitRange{Low: 40, High: 80})
	if wr == nil || src != "history" {
		t.Fatalf("history-dominant should be history, got src=%q wr=%+v", src, wr)
	}
}

func TestEstimateWaitRangeBlendedMiddle(t *testing.T) {
	// realtimeN 中等 → 0<w<0.5 → source=blended。
	wr, src := estimateWaitRange(100, 0, 2.0, 0.3, 4, 1.0, nil, &QueueWaitRange{Low: 40, High: 80})
	if wr == nil || src != "blended" {
		t.Fatalf("middle blend should be blended, got src=%q wr=%+v", src, wr)
	}
}

func TestEstimateWaitRangeOfficialFloorStillApplies(t *testing.T) {
	// 融合后 base 可能 < officialWait，但 low 仍被 officialWait 抬高（物理约束不破）。
	wr, _ := estimateWaitRange(100, 120, 2.0, 0.1, 10, 1.0, nil, &QueueWaitRange{Low: 40, High: 80})
	if wr == nil || wr.Low < 120 {
		t.Errorf("officialWait floor must still apply after blend, Low=%d want >=120", wr.Low)
	}
}

func TestEstimateWaitRangeHistMidTinySoftFloor(t *testing.T) {
	// hist P50 极小（2min）→ rateHist 巨大，但 soft floor（融合路径每组≥0.25min）兜住，
	// base=waitGroups*0.25=25（waitGroups=100），不产生「2分钟叫到100桌」的离谱小值。
	wr, _ := estimateWaitRange(100, 0, 2.0, 0.1, 1, 1.0, nil, &QueueWaitRange{Low: 1, High: 3})
	if wr == nil {
		t.Fatal("nil range")
	}
	// w=0(n=1)→history，cv=0.1→lowMul=0.90：base=max(100/50=2, 100*0.25=25)=25，
	// low=floor(25*0.90)=22。soft floor 保证 base>=25，不会是 1~2 分钟量级的离谱值。
	if wr.Low < 20 {
		t.Errorf("soft floor should prevent tiny estimate, Low=%d", wr.Low)
	}
}

func TestRealtimeBlendWeight(t *testing.T) {
	// hasHist=false → 永远 1。
	if w := realtimeBlendWeight(1, 0.5, false); w != 1.0 {
		t.Errorf("no hist → w=1, got %v", w)
	}
	// n=1 → sampleW=0 → w=0。
	if w := realtimeBlendWeight(1, 0.1, true); w != 0 {
		t.Errorf("n=1 → w=0, got %v", w)
	}
	// n 大、cv=0 → w=1。
	if w := realtimeBlendWeight(10, 0, true); w != 1.0 {
		t.Errorf("n=10,cv=0 → w=1, got %v", w)
	}
	// cv>=0.5 → stabilityW=0 → w=0。
	if w := realtimeBlendWeight(10, 0.5, true); w != 0 {
		t.Errorf("cv>=0.5 → w=0, got %v", w)
	}
}

// A: 时间加权速度（端到端，用 calledRatePerMinuteWeighted）
func TestCalledRateWeightedPrefersRecent(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	// 60min 窗口：清晰的两间隔：[-50min,-10min] 速率 1.0；[-10min,now] 速率 3.0。
	// 等权均值=(1+3)/2=2.0；时间加权应偏向近段 3.0 → rate > 2.0。
	obs2 := []QueueObservation{
		obsAt(store, 100, 50, 60, now.Add(-50*time.Minute)),
		obsAt(store, 140, 45, 55, now.Add(-10*time.Minute)), // 40min 推 40 → 1.0
		obsAt(store, 170, 40, 50, now),                      // 10min 推 30 → 3.0
	}
	rate, _, _, _, _, ok := calledRatePerMinuteWeighted(obs2, now, 60*time.Minute)
	if !ok {
		t.Fatal("expected rate")
	}
	// 等权均值=2.0；加权偏向近段应 > 2.0（3.0 方向）。
	if rate <= 2.0 {
		t.Errorf("weighted rate should prefer recent (>2.0), got %.2f", rate)
	}
}

func TestCalledRateWeightedDenseConvergesToEqual(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	// 全在近 2min 内密集采样（age≈0）→ 权重≈1 → ≈等权均值。
	obs := []QueueObservation{
		obsAt(store, 100, 50, 60, now.Add(-2*time.Minute)),
		obsAt(store, 110, 45, 55, now.Add(-1*time.Minute)), // 1min 推 10 → 10
		obsAt(store, 120, 40, 50, now),                     // 1min 推 10 → 10
	}
	rate, _, _, _, _, ok := calledRatePerMinuteWeighted(obs, now, 15*time.Minute)
	if !ok {
		t.Fatal("expected rate")
	}
	if !approxEqual(rate, 10.0, 0.5) {
		t.Errorf("dense samples should converge to ~10.0, got %.2f", rate)
	}
}

func TestCalledRateWeightedOutlierDoesNotInflate(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.Local)
	store := "3006"
	// 5 个间隔，其中一次补号跳变（瞬时 100/min）应被 IQR 剔除，不拉高均值。
	obs := []QueueObservation{
		obsAt(store, 100, 50, 60, now.Add(-6*time.Minute)),
		obsAt(store, 110, 45, 55, now.Add(-5*time.Minute)), // 1min 推 10 → 10
		obsAt(store, 210, 45, 55, now.Add(-4*time.Minute)), // 1min 推 100 → 100 (补号跳变)
		obsAt(store, 220, 45, 55, now.Add(-3*time.Minute)), // 1min 推 10 → 10
		obsAt(store, 230, 45, 55, now.Add(-2*time.Minute)), // 1min 推 10 → 10
	}
	rate, _, _, _, _, ok := calledRatePerMinuteWeighted(obs, now, 15*time.Minute)
	if !ok {
		t.Fatal("expected rate")
	}
	// 异常 100 应被剔；正常速率约 10/min，加权后不应被拉到几十。
	if rate > 30 {
		t.Errorf("outlier (100/min) should not inflate rate, got %.2f", rate)
	}
}

func TestCoefficientOfVariation(t *testing.T) {
	// n<2 → -1 哨兵。
	if cv := coefficientOfVariation([]float64{5}); cv != -1 {
		t.Errorf("n<2 → cv=-1, got %v", cv)
	}
	// 全相同 → cv=0。
	if cv := coefficientOfVariation([]float64{5, 5, 5}); cv != 0 {
		t.Errorf("identical → cv=0, got %v", cv)
	}
	// 有离散 → cv>0。
	if cv := coefficientOfVariation([]float64{2, 4, 6, 8}); cv <= 0 {
		t.Errorf("dispersed → cv>0, got %v", cv)
	}
}

// TestEstimateWaitRangeTrendGuardedByRealtimeN 验证趋势前瞻校正的 realtimeN 守卫：
// realtimeN<4 时（趋势来自 1~2 个间隔的噪声），即便 source=blended（effRate 几乎纯历史），
// 也不应套用 sqrt(trend) 校正——否则偶发抖动会系统性偏移稳定的历史先验。
func TestEstimateWaitRangeTrendGuardedByRealtimeN(t *testing.T) {
	hist := &QueueWaitRange{Low: 40, High: 80} // histMid=60 → rateHist=100/60≈1.67
	// realtimeN=2, cv 较小 → w=(2-1)/9*(1-cv/0.5) 很小 → source=blended，effRate≈rateHist。
	// trend=2.0（强加速噪声）。守卫生效时不校正，base≈100/rateHist≈60。
	wrGuarded, _ := estimateWaitRange(100, 0, 3.0, 0.2, 2, 2.0, nil, hist)
	// realtimeN=10 → 守卫放行，校正生效：effRate 上抬，base 变小。
	wrApplied, _ := estimateWaitRange(100, 0, 3.0, 0.2, 10, 2.0, nil, hist)
	if wrGuarded == nil || wrApplied == nil {
		t.Fatalf("两个区间都应非 nil: guarded=%+v applied=%+v", wrGuarded, wrApplied)
	}
	// 校正生效时 base 更小 → low 更低。若守卫失效（realtimeN=2 也校正），两者会接近相等。
	if wrApplied.Low >= wrGuarded.Low {
		t.Errorf("realtimeN=10 套趋势校正应让 base 更小（low 更低）：guarded.Low=%d applied.Low=%d",
			wrGuarded.Low, wrApplied.Low)
	}
}

// TestEstimateWaitRangeTrendOnlyWhenSufficient 直接验证守卫门槛：realtimeN<4 不校正、>=4 校正。
func TestEstimateWaitRangeTrendOnlyWhenSufficient(t *testing.T) {
	// 无历史、纯实时：source=recent_speed，base=waitGroups/rate。
	// rate=2.0, trend=2.0 → 校正后 effRate=2*sqrt(2)≈2.83, base≈35。
	// realtimeN=3（<4 守卫）：不校正，base=50。
	wrNo, _ := estimateWaitRange(100, 0, 2.0, 0.1, 3, 2.0, nil, nil)
	wrYes, _ := estimateWaitRange(100, 0, 2.0, 0.1, 4, 2.0, nil, nil)
	if wrNo == nil || wrYes == nil {
		t.Fatalf("区间都应非 nil: no=%+v yes=%+v", wrNo, wrYes)
	}
	if wrYes.Low >= wrNo.Low {
		t.Errorf("realtimeN>=4 时趋势校正应让 base 更小：no.Low=%d yes.Low=%d", wrNo.Low, wrYes.Low)
	}
}

// TestShrinkTrend 验证趋势均值回归收缩：trend_post = λ·trend + (1-λ)·1，
// λ 随 realtimeN 增长（样本少收缩回 1、样本足接近观测值）。
func TestShrinkTrend(t *testing.T) {
	// trend=1 时恒为 1（无趋势不动）。
	if got := shrinkTrend(1.0, 10); got != 1.0 {
		t.Errorf("trend=1 应返回 1，got %v", got)
	}
	// realtimeN<3（守卫之外）→ λ=0 → 完全收缩回 1。
	if got := shrinkTrend(2.0, 2); got != 1.0 {
		t.Errorf("小样本应完全收缩回 1，got %v", got)
	}
	// realtimeN=4 → λ≈0.11 → trend_post 介于 1 和 2 之间，且 >1。
	got4 := shrinkTrend(2.0, 4)
	if got4 <= 1.0 || got4 >= 2.0 {
		t.Errorf("n=4 应部分收缩（1<trend_post<2），got %v", got4)
	}
	// realtimeN>=12 → λ=1 → 完全采信观测 trend。
	if got := shrinkTrend(2.0, 12); got != 2.0 {
		t.Errorf("大样本应完全采信 trend=2，got %v", got)
	}
	if got := shrinkTrend(0.6, 20); got != 0.6 {
		t.Errorf("大样本减速也应完全采信 trend=0.6，got %v", got)
	}
	// 单调性：固定 trend=2，realtimeN 越大 trend_post 越大（越采信）。
	prev := 0.0
	for n := 4; n <= 12; n++ {
		cur := shrinkTrend(2.0, n)
		if cur < prev-1e-9 {
			t.Errorf("trend_post 应随 realtimeN 单调不减：n=%d cur=%v prev=%v", n, cur, prev)
		}
		prev = cur
	}
}

// TestQuantileWaitBounds 验证经验分位数区间的非对称性与回退。
func TestQuantileWaitBounds(t *testing.T) {
	// 样本不足（<3）→ ok=false，调用方回退 CV 查表。
	if _, _, ok := quantileWaitBounds(50, []float64{2.0, 2.5}); ok {
		t.Errorf("rates<3 应返回 ok=false")
	}
	// 匀速（所有 rate 相同）→ low==high==base（无离散）。
	low, high, ok := quantileWaitBounds(50, []float64{2.0, 2.0, 2.0})
	if !ok || low != 50 || high != 50 {
		t.Errorf("匀速应 low=high=base=50：low=%v high=%v ok=%v", low, high, ok)
	}
	// 右偏（慢尾）：速率分布含慢值 → high（慢档等待）应显著大于 low（快档等待），且 high>base>low。
	// base=50，rates 有快有慢。
	low, high, ok = quantileWaitBounds(50, []float64{1.0, 2.0, 4.0})
	if !ok {
		t.Fatalf("应有区间")
	}
	if !(low < 50 && high > 50) {
		t.Errorf("区间应跨 base 两侧：low=%v base=50 high=%v", low, high)
	}
	if high <= low {
		t.Errorf("high 应大于 low：low=%v high=%v", low, high)
	}
}

// TestEstimateWaitRangeQuantileWiderThanSymmetric 验证经验分位数区间在有 rates 时启用、
// 且对右偏（慢尾）分布给出比对称查表更宽的上界（更贴合真实长等待风险）。
func TestEstimateWaitRangeQuantileWiderThanSymmetric(t *testing.T) {
	// base=100/2=50。rates 右偏（含慢值 1.0）：慢档等待被推高。
	rates := []float64{1.0, 2.0, 2.0, 4.0}
	wrQ, _ := estimateWaitRange(100, 0, 2.0, 0.3, 10, 1.0, rates, nil) // 走分位数
	wrS, _ := estimateWaitRange(100, 0, 2.0, 0.3, 10, 1.0, nil, nil)   // 走 CV 查表回退
	if wrQ == nil || wrS == nil {
		t.Fatalf("区间都应非 nil：q=%+v s=%+v", wrQ, wrS)
	}
	// 分位数区间因右偏慢尾，上界应不低于对称查表（覆盖长等待更充分）。
	if wrQ.High < wrS.High {
		t.Errorf("右偏分布下分位数上界应>=对称查表：quantile.High=%d symmetric.High=%d", wrQ.High, wrS.High)
	}
}
