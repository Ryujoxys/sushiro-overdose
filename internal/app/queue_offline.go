package app

import (
	"context"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

// queue_offline.go 是没有实时连接时的离线建议：按以往叫号速度推算叫到时间、
// 离线门店目录、手动记一笔的观测写入。数据来自内置历史包与本机观测，
// 每个结果都标明来源和历史截止日期，不写回个人模型。

const (
	queueEtaSourceHistorySpeed = "history_speed"
	queueSourceEndpointManual  = "manual"
	queueAPIProfileManualV1    = "manual-reading-v1"

	// queueSpeedMinDays：同店同日型同时段少于 3 天时依次回退到同店全部日期类型、全部门店。
	queueSpeedMinDays = 3
	// queueSpeedTrim：截尾均值两端各去掉 10%，压住停滞和补号跳变。
	queueSpeedTrim = 0.10
	// 超过每分钟 12 号多半是号段跳变或重置，不计入速度。
	queueSpeedMaxPerMinute = 12.0
	queueSpeedMinSpan      = 10 * time.Minute
	queueSpeedMaxSpan      = 50 * time.Minute
	// queueSpeedKnownPairs：一家店累计这么多配对后，认为已知道它的营业时段；
	// 从没出现过叫号的时段按不营业处理，不再借用其他门店的速度。
	queueSpeedKnownPairs = 30
	// queueSpeedHorizon：最多往后推算 8 小时，再远就是“今天恐怕叫不到”。
	queueSpeedHorizon = 8 * time.Hour
	// queueReadingMaxAge：实时失败时，30 分钟内的叫号记录仍可作为推算起点。
	queueReadingMaxAge = 30 * time.Minute
)

// ---------- 来源标注 ----------

// queueHistoryCutoffDate 返回内置历史包的截止日期（YYYY-MM-DD），读不到时返回空串。
func queueHistoryCutoffDate() string {
	pack, err := loadBundledHistory()
	if err != nil || pack == nil {
		return ""
	}
	cutoff, err := time.Parse(time.RFC3339, pack.CutoffAt)
	if err != nil {
		return ""
	}
	return cutoff.In(SushiroTimezone).Format("2006-01-02")
}

// queueHistoryLabel 是离线结果统一挂的来源标注，例如“历史截至 2026-09-06”。
func queueHistoryLabel() string {
	if date := queueHistoryCutoffDate(); date != "" {
		return "历史截至 " + date
	}
	return ""
}

// withQueueHistoryLabel 在说明文字后面括注历史截止日期。
func withQueueHistoryLabel(text string) string {
	if label := queueHistoryLabel(); label != "" {
		return text + "（" + label + "）"
	}
	return text
}

// ---------- 叫号速度画像 ----------

// queueSpeedStat 是某个时段的叫号速度（号/分钟）。mean 为 10% 截尾均值
// （全部门店兜底时为中位数），p20/p80 给出慢/快两端。
type queueSpeedStat struct {
	mean  float64
	p20   float64
	p80   float64
	days  int
	level string // store_date_type | store | all_stores
}

// queueSpeedPair 是同一家店同一天相邻两个半小时代表值之间的叫号速度。
type queueSpeedPair struct {
	storeID  int
	day      string
	dateType string
	slot     string
	speed    float64
	local    bool
}

// queueSpeedProfile 是一家店按日期类型 × 半小时的叫号速度，外加全部门店兜底。
type queueSpeedProfile struct {
	storeID      int
	byType       map[string]queueSpeedStat // dateType|slot
	byStore      map[string]queueSpeedStat // slot
	global       map[string]queueSpeedStat // dateType|slot 与 |slot
	openSlots    map[string]bool
	known        bool
	localPairs   int
	historyPairs int
}

// queueSpeedSlot 把时刻归到最近的整点/半点。半小时代表值大多落在 :15/:45，
// 相邻两点之间的速度描述的是以 :00/:30 为中心的半小时，按最近半点取速度与之对齐。
func queueSpeedSlot(t time.Time) string {
	return halfHourBucket(t.In(SushiroTimezone).Add(15 * time.Minute))
}

func queueSpeedBucketStart(t time.Time) time.Time {
	t = t.In(SushiroTimezone)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), (t.Minute()/30)*30, 0, 0, SushiroTimezone)
}

func queueSpeedMidDistance(t time.Time) time.Duration {
	d := t.Sub(queueSpeedBucketStart(t)) - 15*time.Minute
	if d < 0 {
		return -d
	}
	return d
}

// queueSpeedRepKey 与 recordViewSample.key 相同：门店/日期/半小时。
func queueSpeedRepKey(storeID int, t time.Time) string {
	t = t.In(SushiroTimezone)
	return strconv.Itoa(storeID) + "/" + t.Format("2006-01-02") + "/" + halfHourBucket(t)
}

// queueSpeedRepresentatives 每店每天每半小时保留一条代表值：本机记录优先于内置历史，
// 同一来源取最接近半小时中点的一条，与内置历史的取样方式对齐。
func queueSpeedRepresentatives(samples []recordViewSample) map[string]recordViewSample {
	reps := make(map[string]recordViewSample, len(samples))
	for _, s := range samples {
		s.at = s.at.In(SushiroTimezone)
		key := s.key()
		old, ok := reps[key]
		if !ok || (old.history && !s.history) || (old.history == s.history && queueSpeedMidDistance(s.at) < queueSpeedMidDistance(old.at)) {
			reps[key] = s
		}
	}
	return reps
}

// queueSpeedPairs 把半小时代表值按天配对：相邻两个半小时都营业、叫号已知、增量不为负才计入。
func queueSpeedPairs(samples []recordViewSample) []queueSpeedPair {
	reps := queueSpeedRepresentatives(samples)
	pairs := make([]queueSpeedPair, 0, len(reps))
	for _, a := range reps {
		if !a.open || a.called == nil {
			continue
		}
		next := queueSpeedBucketStart(a.at).Add(30 * time.Minute)
		b, ok := reps[queueSpeedRepKey(a.storeID, next)]
		day := a.at.Format("2006-01-02")
		if !ok || !b.open || b.called == nil || b.at.Format("2006-01-02") != day {
			continue
		}
		span := b.at.Sub(a.at)
		delta := *b.called - *a.called
		if span < queueSpeedMinSpan || span > queueSpeedMaxSpan || delta < 0 {
			continue
		}
		speed := float64(delta) / span.Minutes()
		if speed > queueSpeedMaxPerMinute {
			continue
		}
		pairs = append(pairs, queueSpeedPair{
			storeID:  a.storeID,
			day:      day,
			dateType: a.dateType,
			slot:     queueSpeedSlot(a.at.Add(span / 2)),
			speed:    speed,
			local:    !a.history || !b.history,
		})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].day != pairs[j].day {
			return pairs[i].day < pairs[j].day
		}
		return pairs[i].slot < pairs[j].slot
	})
	return pairs
}

// queueTrimmedMean 去掉两端各 trim 比例后取均值；样本太少时退回普通均值。
func queueTrimmedMean(values []float64, trim float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	k := int(float64(len(sorted)) * trim)
	if len(sorted)-2*k > 0 {
		sorted = sorted[k : len(sorted)-k]
	}
	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	return sum / float64(len(sorted))
}

func queueSpeedStatOf(speeds []float64, level string, median bool) queueSpeedStat {
	stat := queueSpeedStat{p20: queueQuantile(speeds, 0.2), p80: queueQuantile(speeds, 0.8), days: len(speeds), level: level}
	if median {
		stat.mean = queueQuantile(speeds, 0.5)
	} else {
		stat.mean = queueTrimmedMean(speeds, queueSpeedTrim)
	}
	return stat
}

// buildQueueSpeedGlobal 是全部门店的兜底速度：按日期类型 × 时段、再按时段取中位数。
func buildQueueSpeedGlobal(pairs []queueSpeedPair) map[string]queueSpeedStat {
	groups := map[string][]float64{}
	for _, p := range pairs {
		if p.speed <= 0 {
			continue
		}
		groups[p.dateType+"|"+p.slot] = append(groups[p.dateType+"|"+p.slot], p.speed)
		groups["|"+p.slot] = append(groups["|"+p.slot], p.speed)
	}
	out := make(map[string]queueSpeedStat, len(groups))
	for key, speeds := range groups {
		if len(speeds) >= queueSpeedMinDays {
			out[key] = queueSpeedStatOf(speeds, "all_stores", true)
		}
	}
	return out
}

// buildQueueSpeedProfile 汇总一家店的配对速度。同一时段本机配对够 3 天时只用本机（更新），
// 否则本机与内置历史合在一起算。
func buildQueueSpeedProfile(storeID int, pairs []queueSpeedPair, global map[string]queueSpeedStat) queueSpeedProfile {
	type group struct{ local, all []float64 }
	byType, byStore := map[string]*group{}, map[string]*group{}
	add := func(m map[string]*group, key string, p queueSpeedPair) {
		g := m[key]
		if g == nil {
			g = &group{}
			m[key] = g
		}
		g.all = append(g.all, p.speed)
		if p.local {
			g.local = append(g.local, p.speed)
		}
	}
	profile := queueSpeedProfile{storeID: storeID, byType: map[string]queueSpeedStat{}, byStore: map[string]queueSpeedStat{}, global: global, openSlots: map[string]bool{}}
	for _, p := range pairs {
		if p.storeID != storeID {
			continue
		}
		profile.openSlots[p.slot] = true
		if p.local {
			profile.localPairs++
		} else {
			profile.historyPairs++
		}
		// 叫号不动的半小时（增量为 0）仍算营业，但不进速度统计：它们多半是没人排队或
		// 接口里的叫号没更新；手里有号说明队伍在动，按“叫号推进时”的速度推算。
		// 回测里把它们算进均值会让推算整体偏晚、近 1 成情况推不到。
		if p.speed <= 0 {
			continue
		}
		add(byType, p.dateType+"|"+p.slot, p)
		add(byStore, p.slot, p)
	}
	pick := func(g *group, level string) (queueSpeedStat, bool) {
		if len(g.local) >= queueSpeedMinDays {
			return queueSpeedStatOf(g.local, level, false), true
		}
		if len(g.all) >= queueSpeedMinDays {
			return queueSpeedStatOf(g.all, level, false), true
		}
		return queueSpeedStat{}, false
	}
	for key, g := range byType {
		if stat, ok := pick(g, "store_date_type"); ok {
			profile.byType[key] = stat
		}
	}
	for key, g := range byStore {
		if stat, ok := pick(g, "store"); ok {
			profile.byStore[key] = stat
		}
	}
	profile.known = profile.localPairs+profile.historyPairs >= queueSpeedKnownPairs
	return profile
}

// at 返回某时刻的叫号速度：同店同日型 → 同店全部日期类型 → 全部门店中位数。
func (p queueSpeedProfile) at(t time.Time, dateType string) (queueSpeedStat, bool) {
	slot := queueSpeedSlot(t)
	if stat, ok := p.byType[dateType+"|"+slot]; ok {
		return stat, true
	}
	if stat, ok := p.byStore[slot]; ok {
		return stat, true
	}
	if p.known && !p.openSlots[slot] {
		return queueSpeedStat{}, false
	}
	if stat, ok := p.global[dateType+"|"+slot]; ok {
		return stat, true
	}
	if stat, ok := p.global["|"+slot]; ok {
		return stat, true
	}
	return queueSpeedStat{}, false
}

// queueSpeedIntegration 记录一次推算用到的速度来源。
type queueSpeedIntegration struct {
	at         time.Time
	ok         bool
	usedGlobal bool
}

// integrate 从 from 起逐分钟累加叫号速度（同一个半小时内速度不变，按段累加），
// 返回累计推进到 remaining 号的时刻；8 小时内推进不到时 ok=false。
// 以往这个时段没有叫号记录（例如快打烊）时沿用上一段的速度：手里的号总会被叫到，
// 回测里打烊前后叫号常常一下推进很多，停在 0 会让推算永远到不了。
func (p queueSpeedProfile) integrate(from time.Time, remaining int, pick func(queueSpeedStat) float64, dateTypeAt func(time.Time) string) queueSpeedIntegration {
	out := queueSpeedIntegration{}
	if remaining <= 0 {
		out.at, out.ok = from, true
		return out
	}
	need, last := float64(remaining), 0.0
	t := from.In(SushiroTimezone)
	end := t.Add(queueSpeedHorizon)
	for t.Before(end) {
		// 速度时段以 :15/:45 为界，下一段从下一个边界开始。
		segEnd := queueSpeedBucketStart(t.Add(-15 * time.Minute)).Add(45 * time.Minute)
		if segEnd.After(end) {
			segEnd = end
		}
		minutes := segEnd.Sub(t).Minutes()
		if minutes <= 0 {
			segEnd = t.Add(time.Minute)
			minutes = 1
		}
		speed := last
		if stat, ok := p.at(t, dateTypeAt(t)); ok {
			speed = pick(stat)
			if stat.level == "all_stores" && speed > 0 {
				out.usedGlobal = true
			}
		}
		if speed > 0 {
			last = speed
		}
		if speed > 0 && speed*minutes >= need {
			out.at = t.Add(time.Duration(need / speed * float64(time.Minute)))
			out.ok = true
			return out
		}
		need -= speed * minutes
		t = segEnd
	}
	return out
}

var (
	queueBundleSpeedOnce   sync.Once
	queueBundleSpeedGlobal map[string]queueSpeedStat
)

// queueBundledSpeedGlobal 用全部内置历史算一次全部门店兜底速度并缓存。
func queueBundledSpeedGlobal() map[string]queueSpeedStat {
	queueBundleSpeedOnce.Do(func() {
		queueBundleSpeedGlobal = map[string]queueSpeedStat{}
		pack, err := loadBundledHistory()
		if err != nil || pack == nil {
			return
		}
		var pairs []queueSpeedPair
		for _, samples := range pack.byStore {
			pairs = append(pairs, queueSpeedPairs(samples)...)
		}
		queueBundleSpeedGlobal = buildQueueSpeedGlobal(pairs)
	})
	return queueBundleSpeedGlobal
}

// queueObservationSpeedSamples 把本机观测转成半小时取样的输入。手动记一笔没有营业状态，
// 但能看到叫号说明门店在叫号，按营业处理。
func queueObservationSpeedSamples(observations []QueueObservation, storeID string, holidays, workdays map[string]bool, now time.Time) []recordViewSample {
	id, err := strconv.Atoi(strings.TrimSpace(storeID))
	if err != nil || id <= 0 {
		return nil
	}
	out := []recordViewSample{}
	for _, o := range observations {
		if strings.TrimSpace(o.StoreID) != strconv.Itoa(id) {
			continue
		}
		called := queueObservationCalledNo(o)
		at, ok := parseRFC3339Local(queueObservationCollectedAt(o))
		if called <= 0 || !ok || at.After(now) {
			continue
		}
		at = at.In(SushiroTimezone)
		open := strings.EqualFold(strings.TrimSpace(o.StoreStatus), "OPEN") || o.SourceEndpoint == queueSourceEndpointManual
		c := called
		out = append(out, recordViewSample{storeID: id, at: at, dateType: queueTrendDateType(at, holidays, workdays), called: &c, open: open})
	}
	return out
}

// offlineSpeedProfile 是某店的叫号速度画像：内置历史 + 本机观测（本机优先），按门店 × 日期类型 × 半小时。
func offlineSpeedProfile(storeID string, now time.Time) queueSpeedProfile {
	if now.IsZero() {
		now = time.Now()
	}
	id, _ := strconv.Atoi(strings.TrimSpace(storeID))
	samples := []recordViewSample{}
	if pack, err := loadBundledHistory(); err == nil && pack != nil {
		for _, s := range pack.byStore[id] {
			if !s.at.After(now) {
				samples = append(samples, s)
			}
		}
	}
	holidays, workdays, _ := loadQueueHolidayDates()
	samples = append(samples, queueObservationSpeedSamples(loadQueueObservations(), storeID, holidays, workdays, now)...)
	return buildQueueSpeedProfile(id, queueSpeedPairs(samples), queueBundledSpeedGlobal())
}

// ---------- 叫到时间推算 ----------

func queueSpeedMean(s queueSpeedStat) float64 { return s.mean }
func queueSpeedFast(s queueSpeedStat) float64 { return s.p80 }
func queueSpeedSlow(s queueSpeedStat) float64 { return s.p20 }

// queueClockHHMM 把时刻显示为门店时区的 HH:MM。
func queueClockHHMM(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(SushiroTimezone).Format("15:04")
}

// setQueueEtaTimes 同时写入 RFC3339 与 HH:MM 两套时间字段，页面直接显示 HH:MM。
func setQueueEtaTimes(eta *QueueAdvisorEta, mid, early, late time.Time) {
	if eta == nil {
		return
	}
	eta.EstimatedCalledAt = mid.Format(time.RFC3339)
	eta.EstimatedCalledAtHHMM = queueClockHHMM(mid)
	if early.IsZero() || late.IsZero() {
		eta.EstimatedCalledAtRange, eta.RangeHHMM = nil, nil
		return
	}
	eta.EstimatedCalledAtRange = &QueueTimeRange{Early: early.Format(time.RFC3339), Late: late.Format(time.RFC3339)}
	eta.RangeHHMM = &QueueTimeRange{Early: queueClockHHMM(early), Late: queueClockHHMM(late)}
}

// estimateCalledAt 按这家店以往的叫号速度，从 calledAt 时叫到 calledNo 起逐分钟累加，
// 推算 myNo 几点叫到。快的日子（P80 速度）给早端，慢的日子（P20 速度）给晚端。
func estimateCalledAt(storeID string, myNo, calledNo int, calledAt time.Time, now time.Time) QueueAdvisorEta {
	if now.IsZero() {
		now = time.Now()
	}
	holidays, workdays, _ := loadQueueHolidayDates()
	dateTypeAt := func(t time.Time) string { return queueTrendDateType(t.In(SushiroTimezone), holidays, workdays) }
	return estimateCalledAtWithProfile(offlineSpeedProfile(storeID, now), myNo, calledNo, calledAt, now, dateTypeAt)
}

func estimateCalledAtWithProfile(profile queueSpeedProfile, myNo, calledNo int, calledAt, now time.Time, dateTypeAt func(time.Time) string) QueueAdvisorEta {
	now = now.In(SushiroTimezone)
	if calledAt.IsZero() || calledAt.After(now) {
		calledAt = now
	}
	calledAt = calledAt.In(SushiroTimezone)
	remaining := myNo - calledNo
	eta := QueueAdvisorEta{
		TargetNo:        myNo,
		RemainingGroups: max(0, remaining),
		Source:          queueEtaSourceHistorySpeed,
		SourceLabel:     "以往叫号速度",
		SourceNote:      "没有实时叫号，按以往同类日子的叫号速度推算，只作参考，以小程序叫号为准。",
		Risk:            "medium",
		Basis:           queueSpeedBasis(profile, false),
	}
	if remaining < 0 {
		eta.WaitMinutesRange = &QueueWaitRange{}
		eta.Risk = "high"
		eta.ArrivalSuggestion = "这个号可能已经过号了，请到小程序确认是否需要补号或重新取号。"
		return eta
	}
	if remaining == 0 {
		setQueueEtaTimes(&eta, calledAt, calledAt, calledAt)
		eta.WaitMinutesRange = &QueueWaitRange{}
		eta.Risk = "low"
		eta.ArrivalSuggestion = "已轮到或即将轮到你，请尽快到店。"
		return eta
	}
	mid := profile.integrate(calledAt, remaining, queueSpeedMean, dateTypeAt)
	early := profile.integrate(calledAt, remaining, queueSpeedFast, dateTypeAt)
	late := profile.integrate(calledAt, remaining, queueSpeedSlow, dateTypeAt)
	if mid.usedGlobal || early.usedGlobal {
		eta.Basis = queueSpeedBasis(profile, true)
		eta.Risk = "high"
	}
	if !mid.ok || !early.ok {
		eta.Risk = "high"
		eta.ArrivalSuggestion = "按以往的叫号速度，这个号今天恐怕要很晚才叫到，或者门店快打烊了，请在小程序里确认。"
		return eta
	}
	lateAt := late.at
	if !late.ok || lateAt.Before(mid.at) {
		// 慢的日子里有些时段叫号几乎不动，推不到时按中间值与早端的差距对称放宽。
		lateAt = mid.at.Add(mid.at.Sub(early.at))
	}
	earlyAt := early.at
	if earlyAt.After(mid.at) {
		earlyAt = mid.at
	}
	setQueueEtaTimes(&eta, mid.at, earlyAt, lateAt)
	eta.WaitMinutesRange = &QueueWaitRange{
		Low:  max(0, int(math.Floor(earlyAt.Sub(now).Minutes()))),
		High: max(0, int(math.Ceil(lateAt.Sub(now).Minutes()))),
	}
	if remaining > 200 {
		eta.Risk = "high"
	}
	switch {
	case lateAt.Before(now):
		eta.Risk = "high"
		eta.ArrivalSuggestion = "按以往的叫号速度，这个号可能已经叫到了，请马上在小程序里确认。"
	case !earlyAt.After(now):
		eta.ArrivalSuggestion = "按以往的叫号速度，随时可能叫到，建议现在就回店。"
	}
	return eta
}

// queueSpeedBasis 用白话说明推算依据并标出历史截止日期。
func queueSpeedBasis(profile queueSpeedProfile, usedGlobal bool) string {
	switch {
	case usedGlobal:
		return withQueueHistoryLabel("这家店以往记录不多，按各门店常见的叫号速度推算")
	case profile.historyPairs == 0 && profile.localPairs > 0:
		return "按这家店本机记录里的叫号速度推算"
	default:
		return withQueueHistoryLabel("按这家店以往的叫号速度推算")
	}
}

// queueEtaArrivalSuggestion 按偏早叫到时间倒减路程给出发建议。
func queueEtaArrivalSuggestion(early time.Time, travelMinutes int, now time.Time) string {
	depart := early.Add(-time.Duration(max(0, travelMinutes)) * time.Minute)
	if depart.Before(now) {
		return "建议现在就出发。"
	}
	return "建议 " + queueClockHHMM(depart) + " 前后出发。"
}

// ---------- 实时接口调用点 ----------

// 实时接口的调用集中在这几个函数变量里：测试替换成本地桩，不向任何主机发请求。
var (
	queueLiveGetStore = func(ctx context.Context, storeID string) (QueueLiveStore, error) {
		return NewQueueLiveClient().GetStore(ctx, storeID)
	}
	queueLiveListStores = func(ctx context.Context, query QueueLiveStoreQuery) ([]QueueLiveStore, error) {
		return NewQueueLiveClient().ListStores(ctx, query)
	}
	queueLiveListAreas = func(ctx context.Context) ([]string, error) {
		return NewQueueLiveClient().ListAreas(ctx)
	}
)

// queueLiveFailureMemory：页面先读实时面板、再读“想几点吃”。实时刚失败过，
// 就不再为粗估等待去等一次超时，直接用历史规律。
const queueLiveFailureMemory = 2 * time.Minute

var queueLiveFailure struct {
	sync.Mutex
	at time.Time
}

func noteQueueLiveResult(err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	queueLiveFailure.Lock()
	defer queueLiveFailure.Unlock()
	if err == nil {
		queueLiveFailure.at = time.Time{}
		return
	}
	queueLiveFailure.at = time.Now()
}

// queueLiveRecentlyFailed 报告最近 2 分钟内实时读取是否失败过。
func queueLiveRecentlyFailed() bool {
	queueLiveFailure.Lock()
	defer queueLiveFailure.Unlock()
	return !queueLiveFailure.at.IsZero() && time.Since(queueLiveFailure.at) < queueLiveFailureMemory
}

// fetchQueueLiveStore 读一家店的实时状态并记下成败。
func fetchQueueLiveStore(ctx context.Context, storeID string) (QueueLiveStore, error) {
	store, err := queueLiveGetStore(ctx, storeID)
	noteQueueLiveResult(err)
	return store, err
}

// ---------- 离线门店目录 ----------

// queueOfflineLongCityNames 是名字超过两个字的城市；其余城市名取区域的前两个字。
var queueOfflineLongCityNames = []string{
	"石家庄", "哈尔滨", "呼和浩特", "乌鲁木齐", "齐齐哈尔", "张家口", "秦皇岛", "连云港",
	"驻马店", "平顶山", "景德镇", "马鞍山", "牡丹江", "佳木斯", "鄂尔多斯", "攀枝花",
	"六盘水", "张家界", "葫芦岛", "防城港", "嘉峪关", "石嘴山", "克拉玛依",
}

// queueOfflineCityFromArea 从“广州天河区”这类区域里取出城市名“广州”。
func queueOfflineCityFromArea(area string) string {
	area = strings.TrimSpace(area)
	for _, name := range queueOfflineLongCityNames {
		if strings.HasPrefix(area, name) {
			return name
		}
	}
	runes := []rune(area)
	if len(runes) <= 2 {
		return area
	}
	return string(runes[:2])
}

// offlineStoreDirectory 是连不上实时数据时的门店目录：内置历史门店 + 本机记录里的门店 + 门店昵称。
// 状态、等待一律未知，只用于选店和显示店名。
func offlineStoreDirectory() []QueueLiveStore {
	byID := map[int]*QueueLiveStore{}
	put := func(id int, name, city, area string) {
		if id <= 0 {
			return
		}
		s := byID[id]
		if s == nil {
			s = &QueueLiveStore{ID: id}
			byID[id] = s
		}
		if name = strings.TrimSpace(name); name != "" {
			s.Name = name
		}
		if city = strings.TrimSpace(city); city != "" {
			s.NameKana = city
		}
		if area = strings.TrimSpace(area); area != "" {
			s.Area = area
		}
	}
	if pack, err := loadBundledHistory(); err == nil && pack != nil {
		for _, s := range pack.Stores {
			put(s.ID, s.Name, s.City, s.Area)
		}
	}
	// 本机快照比内置历史新：按时间先后覆盖，留下最新的店名与区域。
	rows := append([]QueueBaselineRecord(nil), loadQueueBaselineRecords()...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].CollectedAt < rows[j].CollectedAt })
	for _, row := range rows {
		put(row.StoreID, row.Name, row.City, row.Area)
	}
	for _, o := range loadQueueObservations() {
		if id, err := strconv.Atoi(strings.TrimSpace(o.StoreID)); err == nil {
			put(id, "", "", "")
		}
	}
	reg := GetStoreRegistry()
	for _, entry := range reg.List() {
		if id, err := strconv.Atoi(strings.TrimSpace(entry.ID)); err == nil {
			put(id, "", "", "")
		}
	}
	out := make([]QueueLiveStore, 0, len(byID))
	for id, s := range byID {
		if s.Name == "" {
			s.Name = reg.DisplayName(strconv.Itoa(id), "门店 "+strconv.Itoa(id))
		}
		if s.NameKana == "" && s.Area != "" {
			s.NameKana = queueOfflineCityFromArea(s.Area)
		}
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// offlineStoreByID 在离线目录里找一家店。
func offlineStoreByID(storeID string) (QueueLiveStore, bool) {
	id, err := strconv.Atoi(strings.TrimSpace(storeID))
	if err != nil {
		return QueueLiveStore{}, false
	}
	for _, s := range offlineStoreDirectory() {
		if s.ID == id {
			return s, true
		}
	}
	return QueueLiveStore{}, false
}

// offlineStoreName 返回离线可知的店名，找不到时用昵称或门店号。
func offlineStoreName(storeID string) string {
	if s, ok := offlineStoreByID(storeID); ok {
		return s.Name
	}
	return GetStoreRegistry().DisplayName(storeID, storeID)
}

// filterOfflineQueueStores 沿用实时列表的筛选；城市既按城市名比，也按区域开头比
// （内置历史只有“广州天河区”这样的区域）。离线没有坐标，忽略 near。
func filterOfflineQueueStores(stores []QueueLiveStore, query QueueLiveStoreQuery) []QueueLiveStore {
	if city := queueLiveCityKey(query.City); city != "" {
		kept := make([]QueueLiveStore, 0, len(stores))
		for _, s := range stores {
			if queueLiveCityKey(s.NameKana) == city || strings.HasPrefix(s.Area, city) {
				kept = append(kept, s)
			}
		}
		stores = kept
		query.City = ""
	}
	query.Near = ""
	return filterQueueLiveStores(stores, query)
}

// offlineStoreAreas 是离线目录里出现过的区域，供按区域搜索。
func offlineStoreAreas(stores []QueueLiveStore) []string {
	areas := make([]string, 0, len(stores))
	for _, s := range stores {
		areas = append(areas, s.Area)
	}
	areas = UniqueNonEmptyStrings(areas)
	sort.Strings(areas)
	return areas
}

// queueOfflineStoresMessage 说明门店列表来自哪里。
func queueOfflineStoresMessage() string {
	return withQueueHistoryLabel("暂时连不上实时数据，门店列表来自内置历史和本机记录，排队状态未知")
}

// ---------- 手动记一笔 ----------

// queueCalledReading 是一次叫号读数：用户填写、手动记一笔或最近的本机观测。
type queueCalledReading struct {
	calledNo    int
	at          time.Time
	groupQueues int
	source      string // input | manual | recent
}

// recordQueueReading 把用户在小程序里看到的叫号写成一条本机观测：等待记为未知（-1），
// 只用于叫到时间推算和叫号提醒，不写基线，不进“我的记录”等待曲线。
func recordQueueReading(ctx context.Context, storeID string, calledNo int, waitGroups *int, now time.Time) (QueueObservation, error) {
	if now.IsZero() {
		now = time.Now()
	}
	obs := QueueObservation{
		CollectedAt:       now.In(SushiroTimezone).Format(time.RFC3339),
		StoreID:           strings.TrimSpace(storeID),
		WaitMinutes:       -1,
		SourceEndpoint:    queueSourceEndpointManual,
		APIProfileVersion: queueAPIProfileManualV1,
		DisplayCalledNo:   calledNo,
	}
	if waitGroups != nil {
		obs.GroupQueuesCount = *waitGroups
	}
	if err := appendQueueObservation(obs); err != nil {
		return obs, err
	}
	evaluateQueueAlerts(ctx, obs, offlineStoreName(storeID))
	return obs, nil
}

// latestQueueReading 找这家店 maxAge 内最近一条带叫号的本机观测（含手动记一笔）。
func latestQueueReading(storeID string, now time.Time, maxAge time.Duration) (queueCalledReading, bool) {
	storeID = strings.TrimSpace(storeID)
	best := queueCalledReading{}
	found := false
	for _, o := range loadQueueObservations() {
		if strings.TrimSpace(o.StoreID) != storeID {
			continue
		}
		called := queueObservationCalledNo(o)
		at, ok := parseRFC3339Local(queueObservationCollectedAt(o))
		if called <= 0 || !ok || at.After(now) || now.Sub(at) > maxAge {
			continue
		}
		if found && !at.After(best.at) {
			continue
		}
		source := "recent"
		if o.SourceEndpoint == queueSourceEndpointManual {
			source = "manual"
		}
		best = queueCalledReading{calledNo: called, at: at, groupQueues: max(0, o.GroupQueuesCount), source: source}
		found = true
	}
	return best, found
}

// queueKnownWait 返回观测里的等待分钟；手动记一笔的等待未知（负数），按缺失处理。
func queueKnownWait(o QueueObservation) (int, bool) {
	if o.WaitMinutes < 0 {
		return 0, false
	}
	return o.WaitMinutes, true
}

// ---------- “想几点吃”的等待规律 ----------

// queuePlanWait 是某个取号时刻的等待区间及其来源。
type queuePlanWait struct {
	rng      QueueWaitRange
	source   string // local | history
	dateType string
	days     int
}

type queueHistoryWait struct {
	rng  QueueWaitRange
	days int
	// bundled 表示这一档用到了内置历史；只有本机快照时按本机记录标注。
	bundled bool
}

// queuePlanWaitSource 为“想几点吃”提供等待规律：本机等待记录够 3 天的时段用本机；
// 否则用内置历史（与本机快照合并、零等待也算，与记录页曲线同一口径）；
// 都不够 3 天时先用本机已有的少量记录，再用历史。
type queuePlanWaitSource struct {
	storeID            string
	now                time.Time
	holidays, workdays map[string]bool
	local              map[string]map[string]QueueWaitRange
	localDays          map[string]map[string]int
	history            map[string]map[string]queueHistoryWait
}

func newQueuePlanWaitSource(storeID string, now time.Time) *queuePlanWaitSource {
	holidays, workdays, _ := loadQueueHolidayDates()
	return &queuePlanWaitSource{
		storeID:   storeID,
		now:       now,
		holidays:  holidays,
		workdays:  workdays,
		local:     map[string]map[string]QueueWaitRange{},
		localDays: map[string]map[string]int{},
		history:   map[string]map[string]queueHistoryWait{},
	}
}

func (s *queuePlanWaitSource) dateType(t time.Time) string {
	return queueTrendDateType(t.In(SushiroTimezone), s.holidays, s.workdays)
}

func (s *queuePlanWaitSource) at(t time.Time) (queuePlanWait, bool) {
	dt := s.dateType(t)
	bucket := halfHourBucket(t.In(SushiroTimezone))
	if _, ok := s.local[dt]; !ok {
		s.local[dt] = historicalWaitByDateType(s.storeID, dt, s.now)
		s.localDays[dt] = localObservationWaitDays(s.storeID, dt, s.now, s.holidays, s.workdays)
	}
	if _, ok := s.history[dt]; !ok {
		s.history[dt] = queueHistoryWaitProfile(s.storeID, dt, s.now)
	}
	local, hasLocal := s.local[dt][bucket]
	hist, hasHist := s.history[dt][bucket]
	days := s.localDays[dt][bucket]
	switch {
	case hasLocal && days >= queueSpeedMinDays:
		return queuePlanWait{rng: local, source: "local", dateType: dt, days: days}, true
	case hasHist && hist.days >= queueSpeedMinDays:
		return queuePlanWait{rng: hist.rng, source: hist.source(), dateType: dt, days: hist.days}, true
	case hasLocal:
		return queuePlanWait{rng: local, source: "local", dateType: dt, days: days}, true
	case hasHist:
		return queuePlanWait{rng: hist.rng, source: hist.source(), dateType: dt, days: hist.days}, true
	}
	return queuePlanWait{}, false
}

func (h queueHistoryWait) source() string {
	if h.bundled {
		return "history"
	}
	return "local"
}

// basis 用白话说明等待规律的来源；内置历史标出截止日期。
func (w queuePlanWait) basis() string {
	name := queueTrendDateTypeName(w.dateType)
	if w.source == "local" {
		return "按这家店本机记录里" + name + "的等待；今日实时压力仅修正风险"
	}
	return withQueueHistoryLabel("按这家店以往" + name + "的等待规律")
}

// localObservationWaitDays 统计本机观测里每个半小时有几天有等待记录，口径与
// historicalWaitByBucket 相同（最近 14 天、10:00–22:00、等待 > 0）。
func localObservationWaitDays(storeID, dateType string, now time.Time, holidays, workdays map[string]bool) map[string]int {
	now = now.In(SushiroTimezone)
	from := BeginningOfDay(now).AddDate(0, 0, -14)
	to := BeginningOfDay(now).AddDate(0, 0, 1)
	days := map[string]map[string]bool{}
	for _, o := range loadQueueObservations() {
		wait, known := queueKnownWait(o)
		if strings.TrimSpace(o.StoreID) != storeID || !known || wait <= 0 {
			continue
		}
		at, ok := parseRFC3339Local(queueObservationCollectedAt(o))
		if !ok {
			continue
		}
		at = at.In(SushiroTimezone)
		if at.Before(from) || !at.Before(to) || !queueTrendTimeInRange(at, "10:00", "22:00") || queueTrendDateType(at, holidays, workdays) != dateType {
			continue
		}
		bucket := halfHourBucket(at)
		if days[bucket] == nil {
			days[bucket] = map[string]bool{}
		}
		days[bucket][at.Format("2006-01-02")] = true
	}
	out := make(map[string]int, len(days))
	for bucket, set := range days {
		out[bucket] = len(set)
	}
	return out
}

// queueHistoryWaitProfile 是内置历史与本机快照合并后的等待规律（P50/P80，含零等待），
// 与记录页“通常要等”曲线同一口径；按日期类型筛选。
func queueHistoryWaitProfile(storeID, dateType string, now time.Time) map[string]queueHistoryWait {
	id, err := strconv.Atoi(strings.TrimSpace(storeID))
	if err != nil || id <= 0 {
		return map[string]queueHistoryWait{}
	}
	pack, _ := loadBundledHistory()
	view := buildRecordView(loadQueueBaselineRecords(), pack, recordViewSettings{IncludeHistory: true}, localRecordsQuery{store: id, dateType: dateType}, now)
	bundled := view.HistorySamples > 0
	out := make(map[string]queueHistoryWait, len(view.Points))
	for _, p := range view.Points {
		low := int(math.Round(p.Median))
		high := int(math.Round(p.Upper))
		if high < low {
			high = low
		}
		out[p.Time] = queueHistoryWait{rng: QueueWaitRange{Low: max(0, low), High: max(0, high)}, days: p.Days, bundled: bundled}
	}
	return out
}
