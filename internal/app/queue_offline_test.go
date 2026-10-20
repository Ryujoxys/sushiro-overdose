package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

func useTempHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

// stubQueueLive 替换实时接口调用点；测试里从不向任何主机发请求。
func stubQueueLive(t *testing.T, getStore func(context.Context, string) (QueueLiveStore, error)) {
	t.Helper()
	oldGet, oldList, oldAreas := queueLiveGetStore, queueLiveListStores, queueLiveListAreas
	queueLiveGetStore = getStore
	queueLiveListStores = func(context.Context, QueueLiveStoreQuery) ([]QueueLiveStore, error) {
		return nil, ErrLiveAuthRequired
	}
	queueLiveListAreas = func(context.Context) ([]string, error) { return nil, ErrLiveAuthRequired }
	noteQueueLiveResult(nil)
	t.Cleanup(func() {
		queueLiveGetStore, queueLiveListStores, queueLiveListAreas = oldGet, oldList, oldAreas
		noteQueueLiveResult(nil)
	})
}

func liveMustNotBeCalled(t *testing.T) func(context.Context, string) (QueueLiveStore, error) {
	return func(context.Context, string) (QueueLiveStore, error) {
		t.Fatal("live data must not be requested")
		return QueueLiveStore{}, nil
	}
}

func liveAuthRequired(context.Context, string) (QueueLiveStore, error) {
	return QueueLiveStore{}, ErrLiveAuthRequired
}

func cst(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, SushiroTimezone)
}

func expectedHistoryLabel(t *testing.T) string {
	t.Helper()
	pack, err := loadBundledHistory()
	if err != nil {
		t.Fatal(err)
	}
	cutoff, err := time.Parse(time.RFC3339, pack.CutoffAt)
	if err != nil {
		t.Fatal(err)
	}
	return "历史截至 " + cutoff.In(SushiroTimezone).Format("2006-01-02")
}

func calledSample(storeID int, at time.Time, called int, open, history bool) recordViewSample {
	c := called
	return recordViewSample{storeID: storeID, at: at, dateType: queueTrendDateType(at, nil, nil), called: &c, open: open, history: history}
}

func flatSpeedProfile(mean, p20, p80 float64) queueSpeedProfile {
	stat := queueSpeedStat{mean: mean, p20: p20, p80: p80, days: 10, level: "store_date_type"}
	p := queueSpeedProfile{storeID: 1, byType: map[string]queueSpeedStat{}, byStore: map[string]queueSpeedStat{}, global: map[string]queueSpeedStat{}, openSlots: map[string]bool{}, historyPairs: 100, known: true}
	for h := 10; h < 23; h++ {
		for _, m := range []int{0, 30} {
			slot := fmt.Sprintf("%02d:%02d", h, m)
			p.byStore[slot] = stat
			p.openSlots[slot] = true
		}
	}
	return p
}

func weekdayType(time.Time) string { return "weekday" }

func TestQueueHistoryLabelComesFromBundleCutoff(t *testing.T) {
	if got, want := queueHistoryLabel(), expectedHistoryLabel(t); got != want {
		t.Fatalf("label = %q, want %q", got, want)
	}
}

func TestQueueSpeedPairsFollowRules(t *testing.T) {
	day := cst(2026, 10, 13, 0, 0)
	samples := []recordViewSample{
		calledSample(7, day.Add(18*time.Hour+15*time.Minute), 100, true, true),
		calledSample(7, day.Add(18*time.Hour+45*time.Minute), 160, true, true),
		// 叫号回退（重置）不计入。
		calledSample(7, day.Add(19*time.Hour+15*time.Minute), 150, true, true),
		// 不营业不计入。
		calledSample(7, day.Add(19*time.Hour+45*time.Minute), 200, false, true),
		// 间隔太短（3 分钟）不计入。
		calledSample(7, day.Add(20*time.Hour+28*time.Minute), 210, true, true),
		calledSample(7, day.Add(20*time.Hour+31*time.Minute), 240, true, true),
		// 跨天不配对。
		calledSample(7, day.Add(23*time.Hour+45*time.Minute), 900, true, true),
		calledSample(7, day.Add(24*time.Hour+15*time.Minute), 950, true, true),
		// 增量为 0 是有效配对（叫号不动）。
		calledSample(7, day.Add(22*time.Hour+15*time.Minute), 300, true, true),
		calledSample(7, day.Add(22*time.Hour+45*time.Minute), 300, true, true),
	}
	pairs := queueSpeedPairs(samples)
	if len(pairs) != 2 {
		t.Fatalf("pairs = %+v, want two valid pairs", pairs)
	}
	if pairs[0].slot != "18:30" || pairs[0].speed != 2 || pairs[0].dateType != "weekday" {
		t.Fatalf("first pair = %+v, want 2/min centred on 18:30", pairs[0])
	}
	if pairs[1].slot != "22:30" || pairs[1].speed != 0 {
		t.Fatalf("second pair = %+v, want a zero-speed pair at 22:30", pairs[1])
	}
}

func TestQueueSpeedRepresentativesPreferLocal(t *testing.T) {
	at := cst(2026, 10, 13, 18, 15)
	reps := queueSpeedRepresentatives([]recordViewSample{
		calledSample(7, at, 100, true, true),
		calledSample(7, at.Add(10*time.Minute), 130, true, false),
		calledSample(7, at.Add(-12*time.Minute), 90, true, false),
	})
	got := reps[queueSpeedRepKey(7, at)]
	if got.history || *got.called != 130 {
		t.Fatalf("representative = %+v, want the local sample closest to mid-bucket", got)
	}
}

func TestQueueSpeedProfileFallbacks(t *testing.T) {
	var pairs []queueSpeedPair
	add := func(day, dateType, slot string, speed float64, local bool) {
		pairs = append(pairs, queueSpeedPair{storeID: 7, day: day, dateType: dateType, slot: slot, speed: speed, local: local})
	}
	// 18:30 工作日只有 2 天，周末 2 天：回退到同店全部日期类型。
	add("2026-10-12", "weekday", "18:30", 1, false)
	add("2026-10-13", "weekday", "18:30", 1, false)
	add("2026-10-17", "weekend", "18:30", 3, false)
	add("2026-10-18", "weekend", "18:30", 3, false)
	// 19:00 工作日：历史 3 天 + 本机 3 天，本机较新，只用本机。
	for i, d := range []string{"2026-09-01", "2026-09-02", "2026-09-03"} {
		add(d, "weekday", "19:00", 1, false)
		add([]string{"2026-10-14", "2026-10-15", "2026-10-16"}[i], "weekday", "19:00", 4, true)
	}
	// 叫号不动的半小时不进速度。
	add("2026-10-12", "weekday", "20:00", 0, false)
	global := buildQueueSpeedGlobal([]queueSpeedPair{
		{storeID: 8, day: "a", dateType: "weekday", slot: "12:00", speed: 1},
		{storeID: 8, day: "b", dateType: "weekday", slot: "12:00", speed: 2},
		{storeID: 9, day: "c", dateType: "weekday", slot: "12:00", speed: 3},
	})
	profile := buildQueueSpeedProfile(7, pairs, global)
	if s, ok := profile.at(cst(2026, 10, 20, 18, 30), "weekday"); !ok || s.level != "store" || s.mean != 2 {
		t.Fatalf("18:30 = %+v %v, want store-level mean 2", s, ok)
	}
	if s, ok := profile.at(cst(2026, 10, 20, 19, 0), "weekday"); !ok || s.level != "store_date_type" || s.mean != 4 {
		t.Fatalf("19:00 = %+v %v, want local-only mean 4", s, ok)
	}
	if s, ok := profile.at(cst(2026, 10, 20, 12, 0), "weekday"); !ok || s.level != "all_stores" || s.mean != 2 {
		t.Fatalf("12:00 = %+v %v, want all-store median 2", s, ok)
	}
	if _, ok := profile.byType["weekday|20:00"]; ok || !profile.openSlots["20:00"] {
		t.Fatalf("zero-speed half hours count as open but not as speed: %+v", profile)
	}
	profile.known = true
	if _, ok := profile.at(cst(2026, 10, 20, 12, 0), "weekday"); ok {
		t.Fatal("a known store with no calls at 12:00 should be treated as not calling then")
	}
}

func TestQueueTrimmedMeanDropsTails(t *testing.T) {
	values := []float64{0.1, 1, 1, 1, 1, 1, 1, 1, 1, 50}
	if got := queueTrimmedMean(values, queueSpeedTrim); got != 1 {
		t.Fatalf("trimmed mean = %v, want 1", got)
	}
}

func TestEstimateCalledAtIntegratesSpeeds(t *testing.T) {
	now := cst(2026, 10, 20, 18, 0)
	eta := estimateCalledAtWithProfile(flatSpeedProfile(2, 1, 3), 560, 500, now, now, weekdayType)
	if eta.Source != queueEtaSourceHistorySpeed || eta.RemainingGroups != 60 {
		t.Fatalf("eta = %+v", eta)
	}
	if eta.EstimatedCalledAtHHMM != "18:30" || eta.RangeHHMM == nil || eta.RangeHHMM.Early != "18:20" || eta.RangeHHMM.Late != "19:00" {
		t.Fatalf("times = %q %+v, want 18:30 within 18:20–19:00", eta.EstimatedCalledAtHHMM, eta.RangeHHMM)
	}
	if eta.EstimatedCalledAtRange == nil || !strings.HasPrefix(eta.EstimatedCalledAtRange.Early, "2026-10-20T18:20:00") {
		t.Fatalf("RFC3339 fields must stay for existing callers: %+v", eta.EstimatedCalledAtRange)
	}
	if eta.WaitMinutesRange == nil || eta.WaitMinutesRange.Low != 20 || eta.WaitMinutesRange.High != 60 {
		t.Fatalf("wait range = %+v, want 20–60", eta.WaitMinutesRange)
	}
	if !strings.Contains(eta.Basis, expectedHistoryLabel(t)) || !strings.Contains(eta.Basis, "叫号速度") {
		t.Fatalf("basis = %q, want plain speed basis with the history cutoff", eta.Basis)
	}
}

func TestEstimateCalledAtFromEarlierReading(t *testing.T) {
	now := cst(2026, 10, 20, 18, 20)
	// 18:00 叫到 500，按每分钟 2 号，18:30 叫到 560；现在 18:20，还需约 10 分钟。
	eta := estimateCalledAtWithProfile(flatSpeedProfile(2, 1, 3), 560, 500, cst(2026, 10, 20, 18, 0), now, weekdayType)
	if eta.EstimatedCalledAtHHMM != "18:30" || eta.WaitMinutesRange.Low != 0 || eta.WaitMinutesRange.High != 40 {
		t.Fatalf("eta = %+v", eta)
	}
	if !strings.Contains(eta.ArrivalSuggestion, "随时") {
		t.Fatalf("suggestion = %q, want an arrive-now hint once the early end has passed", eta.ArrivalSuggestion)
	}
	late := estimateCalledAtWithProfile(flatSpeedProfile(2, 1, 3), 560, 500, cst(2026, 10, 20, 17, 0), now, weekdayType)
	if late.Risk != "high" || !strings.Contains(late.ArrivalSuggestion, "可能已经叫到") {
		t.Fatalf("eta = %+v, want a likely-called warning", late)
	}
}

func TestEstimateCalledAtEdgeCases(t *testing.T) {
	now := cst(2026, 10, 20, 18, 0)
	passed := estimateCalledAtWithProfile(flatSpeedProfile(2, 1, 3), 480, 500, now, now, weekdayType)
	if passed.Risk != "high" || !strings.Contains(passed.ArrivalSuggestion, "过号") {
		t.Fatalf("passed = %+v", passed)
	}
	due := estimateCalledAtWithProfile(flatSpeedProfile(2, 1, 3), 500, 500, now, now, weekdayType)
	if due.Risk != "low" || due.EstimatedCalledAtHHMM != "18:00" {
		t.Fatalf("due = %+v", due)
	}
	// 慢端推不到时按中间值对称放宽，区间不倒序。
	stalled := estimateCalledAtWithProfile(flatSpeedProfile(2, 0, 3), 560, 500, now, now, weekdayType)
	if stalled.RangeHHMM == nil || stalled.RangeHHMM.Late != "18:40" {
		t.Fatalf("stalled = %+v", stalled.RangeHHMM)
	}
	empty := queueSpeedProfile{byType: map[string]queueSpeedStat{}, byStore: map[string]queueSpeedStat{}, global: map[string]queueSpeedStat{}, openSlots: map[string]bool{}}
	none := estimateCalledAtWithProfile(empty, 560, 500, now, now, weekdayType)
	if none.Risk != "high" || none.RangeHHMM != nil || none.ArrivalSuggestion == "" {
		t.Fatalf("no speed = %+v", none)
	}
}

func TestQueueSpeedIntegrationCarriesSpeedPastClosing(t *testing.T) {
	p := flatSpeedProfile(2, 1, 3)
	for _, slot := range []string{"21:30", "22:00", "22:30"} {
		delete(p.byStore, slot)
		delete(p.openSlots, slot)
	}
	got := p.integrate(cst(2026, 10, 20, 21, 0), 120, queueSpeedMean, weekdayType)
	if !got.ok || queueClockHHMM(got.at) != "22:00" {
		t.Fatalf("integration = %+v, want 22:00 at the last known speed", got)
	}
}

func TestLiveEtaHasHHMMInStoreTimezone(t *testing.T) {
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC) // 18:00 门店时间
	eta := computeQueueEta(160, 100, 0, 0, 0, 2, -1, 5, 1, nil, nil, now)
	if eta.EstimatedCalledAtHHMM == "" || eta.RangeHHMM == nil {
		t.Fatalf("eta = %+v, want HH:MM fields", eta)
	}
	early, _ := time.Parse(time.RFC3339, eta.EstimatedCalledAtRange.Early)
	if eta.RangeHHMM.Early != early.In(SushiroTimezone).Format("15:04") || !strings.HasPrefix(eta.RangeHHMM.Early, "18:") {
		t.Fatalf("range %+v does not match %s in store time", eta.RangeHHMM, eta.EstimatedCalledAtRange.Early)
	}
}

func TestQueueAdvisorWithCalledNoSkipsLive(t *testing.T) {
	useTempHome(t)
	stubQueueLive(t, liveMustNotBeCalled(t))
	now := cst(2026, 10, 20, 18, 0)
	reading := &queueCalledReading{calledNo: 500, at: now, source: "input"}
	advisor, err := buildQueueAdvisorWithReading(context.Background(), "1", 560, 15, reading, now)
	if err != nil {
		t.Fatal(err)
	}
	if !advisor.Offline || advisor.Source != "input" || advisor.Current.CalledNo != 500 || advisor.StoreName == "" || advisor.StoreName == "1" {
		t.Fatalf("advisor = %+v", advisor)
	}
	if advisor.Eta == nil || advisor.Eta.Source != queueEtaSourceHistorySpeed || advisor.Eta.RangeHHMM == nil || advisor.Eta.EstimatedCalledAtHHMM == "" {
		t.Fatalf("eta = %+v", advisor.Eta)
	}
	if advisor.History != expectedHistoryLabel(t) || !strings.Contains(advisor.Eta.Basis, advisor.History) {
		t.Fatalf("offline answer must name the history cutoff: %q / %q", advisor.History, advisor.Eta.Basis)
	}
	if advisor.Eta.ArrivalSuggestion == "" {
		t.Fatal("missing arrival suggestion")
	}
}

func TestQueueAdvisorUsesRecentManualReadingWhenLiveFails(t *testing.T) {
	useTempHome(t)
	stubQueueLive(t, liveAuthRequired)
	now := time.Now().In(SushiroTimezone)
	if _, err := recordQueueReading(context.Background(), "1", 500, nil, now.Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	advisor, err := buildQueueAdvisorWithReading(context.Background(), "1", 560, 0, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if advisor.NeedsCalledNo || advisor.Source != "manual" || advisor.Current.CalledNo != 500 || advisor.Code != liveErrAuthRequired {
		t.Fatalf("advisor = %+v", advisor)
	}
	if advisor.Eta == nil || advisor.Eta.Source != queueEtaSourceHistorySpeed {
		t.Fatalf("eta = %+v", advisor.Eta)
	}
	if len(advisor.Warnings) == 0 || !strings.Contains(advisor.Warnings[0], "记下") {
		t.Fatalf("warnings = %v, want a note about the manual reading", advisor.Warnings)
	}
}

func TestQueueAdvisorAsksForCalledNoWhenNothingKnown(t *testing.T) {
	useTempHome(t)
	stubQueueLive(t, liveAuthRequired)
	now := time.Now().In(SushiroTimezone)
	// 超过 30 分钟的记录不再作为起点。
	if _, err := recordQueueReading(context.Background(), "1", 500, nil, now.Add(-45*time.Minute)); err != nil {
		t.Fatal(err)
	}
	advisor, err := buildQueueAdvisorWithReading(context.Background(), "1", 560, 0, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if !advisor.NeedsCalledNo || advisor.Eta != nil || advisor.Code != liveErrAuthRequired {
		t.Fatalf("advisor = %+v", advisor)
	}
	for _, banned := range []string{"HTTP", "403", "VPN", "代理"} {
		if strings.Contains(advisor.Message, banned) {
			t.Fatalf("message %q must stay plain", advisor.Message)
		}
	}
	if !strings.Contains(advisor.Message, "叫到几号") || !strings.Contains(advisor.Message, expectedHistoryLabel(t)) {
		t.Fatalf("message = %q", advisor.Message)
	}
}

func TestHandleQueueAdvisorCalledNoAndNeedsCalledNo(t *testing.T) {
	useTempHome(t)
	stubQueueLive(t, liveAuthRequired)
	get := func(target string) (*httptest.ResponseRecorder, map[string]any) {
		rec := httptest.NewRecorder()
		handleQueueAdvisor(rec, httptest.NewRequest(http.MethodGet, target, nil))
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		return rec, body
	}
	rec, body := get("/api/queue/advisor?store=1&target_no=560")
	if rec.Code != http.StatusOK || body["needs_called_no"] != true || body["code"] != liveErrAuthRequired {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	stubQueueLive(t, liveMustNotBeCalled(t))
	rec, body = get("/api/queue/advisor?store=1&target_no=560&called_no=500&travel_minutes=10")
	eta, _ := body["eta"].(map[string]any)
	if rec.Code != http.StatusOK || body["offline"] != true || eta == nil || eta["source"] != queueEtaSourceHistorySpeed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	for _, bad := range []string{"called_no=abc", "called_no=0", "called_no=500&called_at=25:00"} {
		rec, body = get("/api/queue/advisor?store=1&target_no=560&" + bad)
		if rec.Code != http.StatusBadRequest || !strings.Contains(fmt.Sprint(body["error"]), "叫") {
			t.Fatalf("%s: status=%d body=%s", bad, rec.Code, rec.Body.String())
		}
	}
}

func TestQueueReadingFromQueryValidatesTime(t *testing.T) {
	now := cst(2026, 10, 20, 18, 0)
	r, err := queueReadingFromQuery("500", "17:40", now)
	if err != nil || r.calledNo != 500 || queueClockHHMM(r.at) != "17:40" {
		t.Fatalf("reading = %+v err = %v", r, err)
	}
	if r, err := queueReadingFromQuery("", "", now); r != nil || err != nil {
		t.Fatalf("empty reading = %+v %v", r, err)
	}
	if _, err := queueReadingFromQuery("500", "18:30", now); err == nil {
		t.Fatal("a future reading time must be rejected")
	}
}

func TestHandleQueueReadingWritesManualObservationAndAlerts(t *testing.T) {
	useTempHome(t)
	stubQueueLive(t, liveMustNotBeCalled(t))
	reach := QueueAlertRule{StoreID: "1", Type: queueAlertCalledReach, TargetNo: 120, NotifyAtNo: 100, Enabled: true}
	below := QueueAlertRule{StoreID: "1", Type: queueAlertWaitBelow, WaitMinutes: 15, Enabled: true}
	if err := SaveQueueAlertConfig(QueueAlertConfig{Rules: []QueueAlertRule{reach, below}}); err != nil {
		t.Fatal(err)
	}
	cfg := LoadQueueAlertConfig()
	belowKey := ""
	for _, rule := range cfg.Rules {
		if rule.Type == queueAlertWaitBelow {
			belowKey = rule.key()
		}
	}
	if err := saveQueueAlertState(map[string]queueAlertRuleState{belowKey: {Armed: true}}); err != nil {
		t.Fatal(err)
	}
	before := notificationCountForTest("desktop")

	rec := httptest.NewRecorder()
	handleQueueReading(rec, httptest.NewRequest(http.MethodPost, "/api/queue/reading", strings.NewReader(`{"store_id":"1","called_no":105}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["ok"] != true || body["observed_at"] == "" || body["observed_at_hhmm"] == "" {
		t.Fatalf("body = %s", rec.Body.String())
	}
	raw, err := os.ReadFile(queueObservationPath())
	if err != nil {
		t.Fatal(err)
	}
	var obs QueueObservation
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(raw))), &obs); err != nil {
		t.Fatal(err)
	}
	if obs.SourceEndpoint != queueSourceEndpointManual || obs.WaitMinutes >= 0 || obs.DisplayCalledNo != 105 || obs.StoreID != "1" {
		t.Fatalf("observation = %+v, want a manual reading with unknown wait", obs)
	}
	if _, err := os.Stat(queueBaselineRecordsPath()); !os.IsNotExist(err) {
		t.Fatalf("a manual reading must not be written to the personal record baseline: %v", err)
	}
	if got := notificationCountForTest("desktop") - before; got != 1 {
		t.Fatalf("notifications = %d, want only the called_reach alert", got)
	}
	state := loadQueueAlertState()
	if !state[belowKey].Armed || state[belowKey].FiredAt != "" {
		t.Fatalf("wait_below state = %+v, unknown wait must neither fire nor disarm", state[belowKey])
	}
	fired := false
	for key, st := range state {
		if key != belowKey && st.FiredOnce {
			fired = true
		}
	}
	if !fired {
		t.Fatalf("called_reach should fire for the manual reading: %+v", state)
	}
}

func TestHandleQueueReadingRejectsBadInput(t *testing.T) {
	useTempHome(t)
	for _, body := range []string{`{}`, `{"store_id":"abc","called_no":5}`, `{"store_id":"1","called_no":0}`, `{"store_id":"1","called_no":5,"wait_groups":-1}`, `{"store_id":1,"called_no":5,"extra":1}`} {
		rec := httptest.NewRecorder()
		handleQueueReading(rec, httptest.NewRequest(http.MethodPost, "/api/queue/reading", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d", body, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	handleQueueReading(rec, httptest.NewRequest(http.MethodGet, "/api/queue/reading", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d", rec.Code)
	}
	rec = httptest.NewRecorder()
	handleQueueReading(rec, httptest.NewRequest(http.MethodPost, "/api/queue/reading", strings.NewReader(`{"store_id":1012,"called_no":5,"wait_groups":3}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("numeric store id: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUnknownWaitIsSkippedByReaders(t *testing.T) {
	useTempHome(t)
	now := cst(2026, 10, 20, 18, 0)
	writeQueueObservationLines(t,
		`{"collected_at":"2026-10-20T17:50:00+08:00","store_id":"1","wait_minutes":-1,"group_queues_count":30,"display_called_no":480,"source_endpoint":"manual"}`,
	)
	points := buildLocalQueuePressureCurvePoints("1", "2026-10-20")
	if len(points) != 1 || points[0].OfficialWaitMinutes != 0 || points[0].PressureLevel != "medium" {
		t.Fatalf("points = %+v, unknown wait must not show as a negative wait", points)
	}
	trends := BuildQueueTrends(QueueTrendQuery{StoreIDs: []string{"1"}, BucketMinutes: 30}, now)
	for _, p := range trends.Series {
		if p.WaitP50Minutes != nil {
			t.Fatalf("unknown wait entered the wait distribution: %+v", p)
		}
	}
	rule := QueueAlertRule{StoreID: "1", Type: queueAlertWaitBelow, WaitMinutes: 15, Enabled: true}
	state := map[string]queueAlertRuleState{rule.key(): {Armed: true}}
	if _, _, fire := queueAlertEvaluateRule(rule, QueueObservation{StoreID: "1", WaitMinutes: -1}, state); fire || !state[rule.key()].Armed {
		t.Fatalf("wait_below fired on unknown wait: %+v", state)
	}
}

func TestLiveWaitEstimateSkipsAfterRecentFailure(t *testing.T) {
	useTempHome(t)
	stubQueueLive(t, liveMustNotBeCalled(t))
	noteQueueLiveResult(errors.New("timeout"))
	if _, ok := liveWaitEstimate(context.Background(), "1", time.Now()); ok {
		t.Fatal("live estimate should be skipped right after a live failure")
	}
	noteQueueLiveResult(context.Canceled)
	if !queueLiveRecentlyFailed() {
		t.Fatal("a cancelled request must not clear the failure memory")
	}
}

func TestQueueMealPlanUsesBundledHistoryWhenLocalThin(t *testing.T) {
	useTempHome(t)
	stubQueueLiveWaitEstimate(t, func(context.Context, string, time.Time) (QueueWaitRange, bool) {
		t.Fatal("meal plan must not wait for live data")
		return QueueWaitRange{}, false
	})
	now := cst(2026, 10, 20, 10, 5)
	got := buildQueueMealPlan(context.Background(), "1", "13:00", 10, now, false)
	if got.RecommendPickupRange == nil || got.WaitMinutesRange == nil {
		t.Fatalf("plan = %+v", got)
	}
	if got.Source != "history" || got.History != expectedHistoryLabel(t) || !strings.Contains(got.Basis, got.History) || got.TravelMinutes != 10 {
		t.Fatalf("plan must name the bundled history: %+v", got)
	}
	if got.RecommendPickupRange.Early < "10:10" || got.StablePickup < "10:10" {
		t.Fatalf("pickup %+v is before now", got.RecommendPickupRange)
	}
	stable, _ := parseHHMM(got.StablePickup, now)
	want := queueHistoryWaitProfile("1", "weekday", now)[halfHourBucket(stable)]
	if *got.WaitMinutesRange != want.rng {
		t.Fatalf("wait = %+v, want bundled P50/P80 %+v", got.WaitMinutesRange, want.rng)
	}
}

func TestQueueMealPlanCandidatesNeverBeforeNow(t *testing.T) {
	useTempHome(t)
	now := cst(2026, 10, 20, 12, 43)
	got := buildQueueMealPlan(context.Background(), "1", "14:30", 0, now, false)
	if got.RecommendPickupRange == nil || got.RecommendPickupRange.Early < "12:50" || got.StablePickup < "12:50" {
		t.Fatalf("plan = %+v, candidates must start at 12:50", got)
	}
	past := buildQueueMealPlan(context.Background(), "1", "12:00", 0, now, false)
	if past.RecommendPickupRange != nil || past.Message == "" {
		t.Fatalf("a meal time already passed should ask for a later time: %+v", past)
	}
}

func TestQueueMealPlanIncludesZeroWaitsFromSnapshots(t *testing.T) {
	useTempHome(t)
	var rows []QueueBaselineRecord
	for i, wait := range []int{0, 0, 0, 40} {
		at := cst(2026, 9, 22, 15, 15).AddDate(0, 0, 7*i)
		rows = append(rows, QueueBaselineRecord{StoreID: 990002, Name: "本机门店", CollectedAt: at.Format(time.RFC3339), StoreStatus: "OPEN", WaitMinutes: wait})
	}
	if err := appendQueueBaselineRecords(rows); err != nil {
		t.Fatal(err)
	}
	now := cst(2026, 10, 20, 14, 0)
	got := buildQueueMealPlan(context.Background(), "990002", "15:30", 0, now, false)
	if got.WaitMinutesRange == nil || got.WaitMinutesRange.Low != 0 || got.WaitMinutesRange.High != 16 {
		t.Fatalf("plan = %+v, want zero waits counted (P50 0, P80 16)", got)
	}
	if got.Source != "local" || got.History != "" {
		t.Fatalf("a store without bundled history must not be labelled as bundled: %+v", got)
	}
}

func TestQueuePickupPlanUsesBundledHistoryWithoutLive(t *testing.T) {
	useTempHome(t)
	stubQueueLiveWaitEstimate(t, func(context.Context, string, time.Time) (QueueWaitRange, bool) { return QueueWaitRange{}, false })
	now := cst(2026, 10, 20, 12, 10)
	got := buildQueuePickupPlan(context.Background(), "1", "12:10", now)
	if got.MealRange == nil || got.Source != "history" || got.History != expectedHistoryLabel(t) {
		t.Fatalf("plan = %+v", got)
	}
}

func TestHandleQueuePlanPassesTravelMinutes(t *testing.T) {
	useTempHome(t)
	rec := httptest.NewRecorder()
	handleQueuePlan(rec, httptest.NewRequest(http.MethodGet, "/api/queue/plan?store=1&target_meal=2330&travel_minutes=25", nil))
	var plan QueueMealPlan
	if err := json.Unmarshal(rec.Body.Bytes(), &plan); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if plan.TravelMinutes != 25 {
		t.Fatalf("travel = %d, want 25", plan.TravelMinutes)
	}
}

func TestOfflineStoreDirectoryAndHandlers(t *testing.T) {
	useTempHome(t)
	stubQueueLive(t, liveAuthRequired)
	if err := appendQueueBaselineRecords([]QueueBaselineRecord{{StoreID: 990001, Name: "本机门店", City: "深圳", Area: "深圳南山区", CollectedAt: cst(2026, 10, 1, 12, 0).Format(time.RFC3339), StoreStatus: "OPEN"}}); err != nil {
		t.Fatal(err)
	}
	writeQueueObservationLines(t, `{"collected_at":"2026-10-01T12:00:00+08:00","store_id":"990003","display_called_no":10,"source_endpoint":"manual","wait_minutes":-1}`)

	get := func(target string, handler http.HandlerFunc) map[string]any {
		rec := httptest.NewRecorder()
		handler(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status=%d body=%s", target, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["offline"] != true || body["code"] != liveErrAuthRequired || !strings.Contains(fmt.Sprint(body["message"]), expectedHistoryLabel(t)) {
			t.Fatalf("%s: offline answer must be labelled: %v", target, body)
		}
		return body
	}
	stores := func(body map[string]any) []map[string]any {
		var out []map[string]any
		for _, raw := range body["stores"].([]any) {
			out = append(out, raw.(map[string]any))
		}
		return out
	}
	body := get("/api/queue/stores?q="+url.QueryEscape("天河")+"&limit=100", handleQueueLiveStores)
	list := stores(body)
	if len(list) == 0 || int(body["count"].(float64)) != len(list) {
		t.Fatalf("q=天河 found %v", body)
	}
	for _, s := range list {
		if !strings.Contains(fmt.Sprint(s["area"])+fmt.Sprint(s["name"]), "天河") {
			t.Fatalf("store %v does not match 天河", s)
		}
	}
	for _, s := range stores(get("/api/queue/stores?city="+url.QueryEscape("广州"), handleQueueLiveStores)) {
		if !strings.HasPrefix(fmt.Sprint(s["area"]), "广州") || s["nameKana"] != "广州" {
			t.Fatalf("city=广州 returned %v", s)
		}
	}
	shenzhen := stores(get("/api/queue/stores?city="+url.QueryEscape("深圳市")+"&q="+url.QueryEscape("本机"), handleQueueLiveStores))
	if len(shenzhen) != 1 || shenzhen[0]["name"] != "本机门店" {
		t.Fatalf("local record store not found: %v", shenzhen)
	}
	picked := stores(get("/api/queue/stores?stores=1,990003", handleQueueLiveStores))
	if len(picked) != 2 {
		t.Fatalf("stores=1,990003 returned %v", picked)
	}
	for _, s := range picked {
		if fmt.Sprint(s["id"]) == "990003" && s["name"] != "门店 990003" {
			t.Fatalf("observation-only store should get a readable name: %v", s)
		}
	}
	areas := get("/api/queue/areas", handleQueueLiveAreas)
	if !strings.Contains(fmt.Sprint(areas["areas"]), "广州天河区") || !strings.Contains(fmt.Sprint(areas["areas"]), "深圳南山区") {
		t.Fatalf("areas = %v", areas["areas"])
	}
	if got := queueOfflineCityFromArea("石家庄桥西区"); got != "石家庄" {
		t.Fatalf("city = %q", got)
	}
}

func TestHandleStoresFallsBackToOfflineName(t *testing.T) {
	useTempHome(t)
	stubQueueLive(t, liveAuthRequired)
	webSettingsMu.Lock()
	webSettings, webClient = Settings{StoreIDs: []string{"1"}}, nil
	webSettingsMu.Unlock()
	t.Cleanup(clearWebSettings)
	rec := httptest.NewRecorder()
	handleStores(rec, httptest.NewRequest(http.MethodGet, "/api/stores", nil))
	var list []map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 1 {
		t.Fatalf("body=%s", rec.Body.String())
	}
	if list[0]["name"] == "1" || list[0]["name"] == "" || rec.Header().Get("X-Sushiro-Offline") != "1" {
		t.Fatalf("store = %v header=%q", list[0], rec.Header().Get("X-Sushiro-Offline"))
	}
}

func TestDashboardTreatsManualReadingWaitAsUnknown(t *testing.T) {
	useTempHome(t)
	now := cst(2026, 10, 20, 18, 0)
	observations := []QueueObservation{
		{CollectedAt: cst(2026, 10, 20, 17, 40).Format(time.RFC3339), StoreID: "1", WaitMinutes: 30, GroupQueuesCount: 20, StoreStatus: "OPEN", DisplayCalledNo: 450},
		{CollectedAt: cst(2026, 10, 20, 17, 50).Format(time.RFC3339), StoreID: "1", WaitMinutes: -1, DisplayCalledNo: 480, SourceEndpoint: queueSourceEndpointManual},
	}
	query := normalizeQueueDashboardQuery(QueueDashboardQuery{StoreIDs: []string{"1"}})
	filter := stringSet(query.StoreIDs)
	names := map[string]string{"1": "测试店"}
	rows := buildQueueDashboardLatestRows(query, QueueBaselineExport{}, nil, observations, names, filter)
	if len(rows) != 1 || rows[0].WaitMinutes != 30 || rows[0].QueueGroups != 20 {
		t.Fatalf("latest rows = %+v, a manual reading must not replace the store's queue state", rows)
	}
	for _, p := range buildQueueDashboardLocalTrend(query, nil, observations, names, filter, now) {
		if p.TotalWaitMinutes < 0 {
			t.Fatalf("trend point %+v carries an unknown wait", p)
		}
	}
	summary, curve := buildQueueDashboardCalledCurve(query, nil, observations, names, filter, now)
	if summary.LatestCalledNo != 480 || summary.LatestWaitMinutes != 0 {
		t.Fatalf("called summary = %+v, want the manual called number with no negative wait", summary)
	}
	for _, p := range curve {
		if p.LatestWaitMinutes < 0 || p.WaitMinutes < 0 {
			t.Fatalf("called point %+v shows a negative wait", p)
		}
	}
}

func TestOfflineAdvisorRecentReadingIsLabelledAndInStoreTime(t *testing.T) {
	useTempHome(t)
	stubQueueLive(t, liveAuthRequired)
	now := cst(2026, 10, 20, 18, 0)
	// 自动记录的观测时间带 Z（机器在 UTC），在等桌数不是用户记下的。
	writeQueueObservationLines(t,
		`{"collected_at":"2026-10-20T09:50:00Z","store_id":"1","wait_minutes":35,"group_queues_count":25,"store_status":"OPEN","display_called_no":500}`,
	)
	advisor, err := buildQueueAdvisorWithReading(context.Background(), "1", 560, 0, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if advisor.Source != "recent" || advisor.Current.CalledNo != 500 || advisor.Eta == nil {
		t.Fatalf("advisor = %+v", advisor)
	}
	if strings.Contains(advisor.Pressure.Reason, "你记下") || !strings.Contains(advisor.Pressure.Reason, "最近一次记录") {
		t.Fatalf("pressure reason = %q, an automatic record must not be described as the user's note", advisor.Pressure.Reason)
	}
	if advisor.CalledAtHHMM != "17:50" || !strings.HasSuffix(advisor.CalledAt, "+08:00") {
		t.Fatalf("called_at = %q / %q, want store time", advisor.CalledAt, advisor.CalledAtHHMM)
	}
	if len(advisor.Warnings) == 0 || !strings.Contains(advisor.Warnings[0], "最近一次记录") {
		t.Fatalf("warnings = %v", advisor.Warnings)
	}
}

func TestOfflineAdvisorWithoutTargetOrWithPassedNumber(t *testing.T) {
	useTempHome(t)
	stubQueueLive(t, liveMustNotBeCalled(t))
	now := cst(2026, 10, 20, 18, 0)
	reading := &queueCalledReading{calledNo: 500, at: now, source: "input"}
	noTarget, err := buildQueueAdvisorWithReading(context.Background(), "1", 0, 0, reading, now)
	if err != nil {
		t.Fatal(err)
	}
	if noTarget.Eta != nil || !strings.Contains(noTarget.Message, "号码") || noTarget.Current.CalledNo != 500 {
		t.Fatalf("without a ticket number = %+v", noTarget)
	}
	passed, err := buildQueueAdvisorWithReading(context.Background(), "1", 480, 0, reading, now)
	if err != nil {
		t.Fatal(err)
	}
	if passed.Eta == nil || passed.Eta.Risk != "high" || len(passed.Warnings) == 0 || !strings.Contains(passed.Warnings[len(passed.Warnings)-1], "过号") {
		t.Fatalf("passed number = %+v", passed)
	}
}

func TestQueueReadingFromQueryAcceptsRFC3339AndRejectsOldTimes(t *testing.T) {
	now := cst(2026, 10, 20, 18, 0)
	r, err := queueReadingFromQuery("500", "2026-10-20T09:40:00Z", now)
	if err != nil || queueClockHHMM(r.at) != "17:40" || r.at.Location() != SushiroTimezone {
		t.Fatalf("reading = %+v err = %v, want 17:40 store time", r, err)
	}
	if _, err := queueReadingFromQuery("500", "05:30", now); err == nil {
		t.Fatal("a reading more than 12 hours old must be rejected")
	}
	if _, err := queueReadingFromQuery("500", "18:30:99", now); err == nil {
		t.Fatal("a malformed time must be rejected")
	}
}

func TestQueueMealPlanCloseToMealStillConsidersPickingUpNow(t *testing.T) {
	useTempHome(t)
	now := cst(2026, 10, 20, 12, 51)
	got := buildQueueMealPlan(context.Background(), "1", "12:58", 0, now, false)
	if got.StablePickup != "12:51" || got.WaitMinutesRange == nil {
		t.Fatalf("plan = %+v, want picking up now as the only candidate", got)
	}
	if strings.Contains(got.Message, "规律还不够") {
		t.Fatalf("message = %q must not blame missing history", got.Message)
	}
}

func TestQueueMealPlanOnAdjustedWorkdayUsesWeekdayHistory(t *testing.T) {
	useTempHome(t)
	if err := os.MkdirAll(AppDirPath(), 0o755); err != nil {
		t.Fatal(err)
	}
	// 2026-10-10 是周六，但在 holidays.json 里标为调休上班。
	if err := os.WriteFile(queueHolidayPath(), []byte(`{"workdays":["2026-10-10"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	now := cst(2026, 10, 10, 10, 5)
	got := buildQueueMealPlan(context.Background(), "1", "13:00", 0, now, false)
	if got.RecommendPickupRange == nil || got.Source != "history" {
		t.Fatalf("plan = %+v, want weekday history on an adjusted workday", got)
	}
	if !strings.Contains(got.Basis, "工作日") || strings.Contains(got.Basis, "调休") {
		t.Fatalf("basis = %q, want the weekday pattern named plainly", got.Basis)
	}
}

func TestQueueMealPlanWarnsWhenTargetIsOutOfReach(t *testing.T) {
	useTempHome(t)
	var rows []QueueBaselineRecord
	for _, day := range []int{0, 7, 14, 21} {
		for _, minute := range []int{15, 45} {
			at := cst(2026, 9, 22, 12, minute).AddDate(0, 0, day)
			rows = append(rows, QueueBaselineRecord{StoreID: 990004, Name: "排长队的店", CollectedAt: at.Format(time.RFC3339), StoreStatus: "OPEN", WaitMinutes: 90})
		}
	}
	if err := appendQueueBaselineRecords(rows); err != nil {
		t.Fatal(err)
	}
	now := cst(2026, 10, 20, 12, 0)
	got := buildQueueMealPlan(context.Background(), "990004", "12:40", 0, now, false)
	if got.StablePickup != "12:00" || got.Risk != "high" || !strings.Contains(got.Message, "可能赶不上 12:40") {
		t.Fatalf("plan = %+v, want a plain out-of-reach warning", got)
	}
}

func TestQueuePlansSayWhenNothingIsKnown(t *testing.T) {
	useTempHome(t)
	stubQueueLiveWaitEstimate(t, func(context.Context, string, time.Time) (QueueWaitRange, bool) { return QueueWaitRange{}, false })
	now := cst(2026, 10, 20, 12, 0)
	pickup := buildQueuePickupPlan(context.Background(), "990009", "12:00", now)
	if pickup.WaitMinutesRange != nil || !strings.Contains(pickup.Message, "估不出来") || strings.Contains(pickup.Message, "未开放线上取号") {
		t.Fatalf("pickup = %+v", pickup)
	}
	meal := buildQueueMealPlan(context.Background(), "990009", "13:00", 0, now, false)
	if meal.RecommendPickupRange != nil || !strings.Contains(meal.Message, "估不出来") {
		t.Fatalf("meal = %+v", meal)
	}
}
