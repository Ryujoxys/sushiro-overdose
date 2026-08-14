package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBuildQueueLocalStatsAggregatesPrivateSessions(t *testing.T) {
	sessions := []QueueSession{
		{StoreID: "001", TakenAt: "2026-05-16T18:00:00+08:00", TicketNo: 101, DisplayCalledNoAtTake: 80, CalledForUserAt: "2026-05-16T18:40:00+08:00", CalledNoWhenUserCalled: 101, ActualWaitMinutes: 40, CheckedInAt: "2026-05-16T18:20:00+08:00", PartySize: 2, TableType: "T"},
		{StoreID: "001", TakenAt: "2026-05-23T18:10:00+08:00", TicketNo: 102, DisplayCalledNoAtTake: 84, CalledForUserAt: "2026-05-23T19:00:00+08:00", CalledNoWhenUserCalled: 102, ActualWaitMinutes: 50, CheckedInAt: "2026-05-23T18:35:00+08:00", PartySize: 2, TableType: "T"},
		{StoreID: "001", TakenAt: "2026-05-30T18:25:00+08:00", TicketNo: 103, DisplayCalledNoAtTake: 88, CalledForUserAt: "2026-05-30T19:25:00+08:00", CalledNoWhenUserCalled: 103, ActualWaitMinutes: 60, CheckedInAt: "2026-05-30T18:55:00+08:00", PartySize: 2, TableType: "T", ExpiredOrMissed: true},
		{StoreID: "002", TakenAt: "2026-05-16T18:00:00+08:00", ActualWaitMinutes: 20, PartySize: 5, TableType: "C"},
	}

	stats, usable := BuildQueueLocalStats(sessions, 3)
	if usable != 4 {
		t.Fatalf("usable sessions = %d, want 4", usable)
	}
	if len(stats) != 1 {
		t.Fatalf("stats len = %d, want 1", len(stats))
	}
	got := stats[0]
	if got.StoreID != "001" || got.Weekday != 6 || got.TimeBucket != "18:00" || got.PartySizeBucket != "1-2" || got.TableType != "T" {
		t.Fatalf("unexpected bucket: %+v", got)
	}
	if got.Samples != 3 {
		t.Fatalf("samples = %d, want 3", got.Samples)
	}
	if got.WaitP50Minutes == nil || *got.WaitP50Minutes != 50 {
		t.Fatalf("wait p50 = %v, want 50", got.WaitP50Minutes)
	}
	if got.WaitP80Minutes == nil || *got.WaitP80Minutes != 56 {
		t.Fatalf("wait p80 = %v, want 56", got.WaitP80Minutes)
	}
	if got.MissedRate < 0.33 || got.MissedRate > 0.34 {
		t.Fatalf("missed rate = %.4f, want about 0.3333", got.MissedRate)
	}
}

func TestQueueTrendActualPassedRequiresConfirmedCalledNumber(t *testing.T) {
	if _, ok := queueActualPassed(QueueSession{TicketNo: 130, DisplayCalledNoAtTake: 100}); ok {
		t.Fatal("actual passed should not be inferred without confirmed called number")
	}
	passed, ok := queueActualPassed(QueueSession{DisplayCalledNoAtTake: 100, CalledNoWhenUserCalled: 128})
	if !ok || passed != 28 {
		t.Fatalf("actual passed = %d/%v, want 28/true", passed, ok)
	}
}

func TestQueueTrendResponseDoesNotExposeRawTicketFields(t *testing.T) {
	sessions := []QueueSession{
		{StoreID: "001", TakenAt: "2026-05-16T18:00:00+08:00", TicketNo: 123, DisplayCalledNoAtTake: 100, CalledNoWhenUserCalled: 123, ActualWaitMinutes: 30, PartySize: 2, TableType: "T"},
	}
	stats, _ := BuildQueueLocalStats(sessions, 1)
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"ticket_no", "display_called_no", "wechat", "authorization", "phone", "called_no_when_user_called"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("local aggregate contains raw field %q: %s", forbidden, data)
		}
	}
}

func TestQueueTrendDateTypeUsesLocalHolidayOverrides(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	holiday := time.Date(2026, 5, 1, 18, 0, 0, 0, loc)
	workday := time.Date(2026, 5, 2, 18, 0, 0, 0, loc)
	if got := queueTrendDateType(holiday, map[string]bool{"2026-05-01": true}, nil); got != "holiday" {
		t.Fatalf("holiday type = %s, want holiday", got)
	}
	if got := queueTrendDateType(workday, nil, map[string]bool{"2026-05-02": true}); got != "workday" {
		t.Fatalf("adjusted workday type = %s, want workday", got)
	}
}

func TestQueueTrendDateTypeUsesWeekendWindow(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	cases := []struct {
		at   time.Time
		want string
	}{
		{time.Date(2026, 6, 5, 16, 29, 59, 0, loc), "weekday"},
		{time.Date(2026, 6, 5, 16, 30, 0, 0, loc), "weekend"},
		{time.Date(2026, 6, 6, 12, 0, 0, 0, loc), "weekend"},
		{time.Date(2026, 6, 7, 21, 59, 59, 0, loc), "weekend"},
		{time.Date(2026, 6, 7, 22, 0, 0, 0, loc), "weekday"},
	}
	for _, c := range cases {
		if got := queueTrendDateType(c.at, nil, nil); got != c.want {
			t.Fatalf("queueTrendDateType(%s) = %s, want %s", c.at.Format(time.RFC3339), got, c.want)
		}
	}
}

func TestQueueTrendGlobalPassedUsesForwardObservationDelta(t *testing.T) {
	query := normalizeQueueTrendQuery(QueueTrendQuery{
		StoreIDs:      []string{"001"},
		DateType:      "weekend",
		From:          "2026-05-16",
		To:            "2026-05-16",
		Start:         "18:00",
		End:           "20:00",
		BucketMinutes: 30,
	}, time.Date(2026, 5, 16, 20, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	holidays, workdays := map[string]bool{}, map[string]bool{}
	storeFilter := stringSet(query.StoreIDs)
	series := map[string]*queueTrendAccumulator{}
	summary := QueueTrendSummary{}
	observations := []QueueObservation{
		{StoreID: "001", Timestamp: "2026-05-16T18:00:00+08:00", DisplayCalledNo: 80},
		{StoreID: "001", Timestamp: "2026-05-16T18:20:00+08:00", DisplayCalledNo: 91},
		{StoreID: "001", Timestamp: "2026-05-16T18:40:00+08:00", DisplayCalledNo: 90},
	}
	for i := 1; i < len(observations); i++ {
		prev, curr := observations[i-1], observations[i]
		currAt, _ := parseRFC3339Local(curr.Timestamp)
		if !queueTrendMatches(query, currAt, storeFilter, curr.StoreID, holidays, workdays) {
			continue
		}
		diff := curr.DisplayCalledNo - prev.DisplayCalledNo
		if diff <= 0 {
			continue
		}
		acc := queueTrendAcc(series, curr.StoreID, curr.StoreID, queueTrendDateType(currAt, holidays, workdays), queueTrendBucket(currAt, query.BucketMinutes))
		acc.point.GlobalPassed += diff
		summary.GlobalPassedTotal += diff
	}
	points := finalizeQueueTrendPoints(series)
	if len(points) != 1 {
		t.Fatalf("points len = %d, want 1", len(points))
	}
	if points[0].GlobalPassed != 11 || summary.GlobalPassedTotal != 11 {
		t.Fatalf("global passed = point %d summary %d, want 11", points[0].GlobalPassed, summary.GlobalPassedTotal)
	}
}

func TestQueueBaselineRollupsMergeIntoTrend(t *testing.T) {
	query := normalizeQueueTrendQuery(QueueTrendQuery{
		StoreIDs:      []string{"3015"},
		DateType:      "weekend",
		From:          "2026-06-06",
		To:            "2026-06-06",
		Start:         "18:00",
		End:           "20:00",
		BucketMinutes: 60,
	}, time.Date(2026, 6, 6, 20, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	waitA, waitB, safe := 30.0, 50.0, 70.0
	series := map[string]*queueTrendAccumulator{}
	summary := addQueueBaselineToTrend(series, QueueTrendSummary{}, query, QueueBaselineExport{
		Stats: QueueBaselineStats{SourceUpdatedAt: "2026-06-03T22:30:00+08:00"},
		Stores: []QueueBaselineStore{
			{StoreID: 3015, Name: "深圳店"},
		},
		Rollups: []QueueBaselineRollup{
			{StoreID: 3015, DateType: "weekend", Weekday: 6, TimeBucket: "18:00", SampleCount: 2, WaitTypicalMinutes: &waitA, WaitSafeMinutes: &safe, BusyRate: 0.5, OnlineOpenRate: 1},
			{StoreID: 3015, DateType: "weekend", Weekday: 6, TimeBucket: "18:30", SampleCount: 2, WaitTypicalMinutes: &waitB, WaitSafeMinutes: &safe, BusyRate: 1, OnlineOpenRate: 0.5},
		},
	}, map[string]string{}, stringSet([]string{"3015"}))
	points := finalizeQueueTrendPoints(series)
	if len(points) != 1 {
		t.Fatalf("points len = %d, want 1", len(points))
	}
	point := points[0]
	if point.Bucket != "18:00" || point.BaselineSamples != 4 {
		t.Fatalf("unexpected point bucket/samples: %+v", point)
	}
	if point.WaitP50Minutes == nil || *point.WaitP50Minutes != 40 {
		t.Fatalf("baseline p50 = %v, want 40", point.WaitP50Minutes)
	}
	if point.OnlineOpenRate == nil || *point.OnlineOpenRate != 0.75 {
		t.Fatalf("online rate = %v, want 0.75", point.OnlineOpenRate)
	}
	if summary.BaselineSamples != 4 || summary.BaselineRecords != 2 || summary.BaselineUpdatedAt == "" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestQueueObservationFromStoreInfoCapturesPublicWait(t *testing.T) {
	now := time.Date(2026, 5, 26, 10, 30, 0, 0, time.FixedZone("CST", 8*3600))
	observation, ok := queueObservationFromStoreInfo("3015", StoreInfo{
		Wait:              20,
		WaitTimeCounter:   3,
		StoreStatus:       "OPEN",
		NetTicketStatus:   "ONLINE_OPEN",
		GroupQueuesCount:  19,
		RemoteTicketing:   "ON",
		ReservationStatus: "ON",
	}, now)
	if !ok {
		t.Fatal("observation should be captured")
	}
	if observation.StoreID != "3015" || observation.WaitMinutes != 20 || !observation.OnlineOpen || observation.GroupQueuesCount != 19 {
		t.Fatalf("unexpected observation: %+v", observation)
	}
	if observation.CollectedAt != now.Format(time.RFC3339) || observation.Timestamp != "" {
		t.Fatalf("unexpected observation time fields: %+v", observation)
	}
	if observation.ReservationStatus != "ON" || observation.WaitTimeCounter != 3 || observation.SourceEndpoint != queueSourceEndpointStoreByID || observation.APIProfileVersion != queueAPIProfileStoreDetailV1 {
		t.Fatalf("unexpected database-aligned fields: %+v", observation)
	}
}

func TestNormalizeQueueObservationWritesDatabaseShape(t *testing.T) {
	observation := QueueObservation{
		Timestamp:       "2026-05-26T10:30:00+08:00",
		StoreID:         "3015",
		DisplayCalledNo: 3,
		WaitMinutes:     20,
	}
	normalizeQueueObservationForWrite(&observation, time.Time{})
	data, err := json.Marshal(observation)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(data)
	if !strings.Contains(raw, `"collected_at":"2026-05-26T10:30:00+08:00"`) || !strings.Contains(raw, `"wait_minutes":20`) {
		t.Fatalf("database fields missing: %s", raw)
	}
	if strings.Contains(raw, `"ts"`) {
		t.Fatalf("legacy ts should not be written: %s", raw)
	}
	if !strings.Contains(raw, `"source_endpoint":"getStoreById"`) || !strings.Contains(raw, `"api_profile_version":"store-detail-profile-v1"`) {
		t.Fatalf("source fields missing: %s", raw)
	}
}

func TestAppendQueueObservationTightensPrivateObservationFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose owner-only POSIX file modes")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if err := os.MkdirAll(AppDirPath(), 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}
	path := queueObservationPath()
	if err := os.WriteFile(path, []byte(`{"store_id":"old","display_called_no":1}`+"\n"), 0o644); err != nil {
		t.Fatalf("write existing observation file: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod existing observation file: %v", err)
	}

	err := appendQueueObservation(QueueObservation{
		Timestamp:       "2026-05-26T10:30:00+08:00",
		StoreID:         "3015",
		DisplayCalledNo: 3,
	})
	if err != nil {
		t.Fatalf("append queue observation: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat observation file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("observation file mode = %o, want 0600", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read observation file: %v", err)
	}
	if got := strings.Count(string(data), "\n"); got != 2 {
		t.Fatalf("observation lines = %d, want 2: %s", got, data)
	}
	if !strings.Contains(string(data), `"store_id":"3015"`) {
		t.Fatalf("appended observation missing: %s", data)
	}
	if !strings.Contains(string(data), `"collected_at":"2026-05-26T10:30:00+08:00"`) {
		t.Fatalf("appended observation should use collected_at: %s", data)
	}
}

func TestQueueObservationFromStoreInfoCapturesGroupQueues(t *testing.T) {
	now := time.Date(2026, 5, 26, 10, 30, 0, 0, time.FixedZone("CST", 8*3600))
	var store StoreInfo
	if err := json.Unmarshal([]byte(`{
		"wait": 30,
		"groupQueuesCount": 33,
		"groupQueues": {
			"mixedQueue": ["001", "002", "003"],
			"reservationQueue": ["101"],
			"counterQueue": [],
			"boothQueue": ["001", "002", "003"]
		}
	}`), &store); err != nil {
		t.Fatal(err)
	}
	observation, ok := queueObservationFromStoreInfo("3015", store, now)
	if !ok {
		t.Fatal("observation should be captured")
	}
	if observation.DisplayCalledNo != 3 {
		t.Fatalf("display called no = %d, want 3", observation.DisplayCalledNo)
	}
	if strings.Join(observation.GroupQueues.MixedQueue, ",") != "001,002,003" {
		t.Fatalf("mixed queue = %#v, want 001/002/003", observation.GroupQueues.MixedQueue)
	}
	if strings.Join(observation.GroupQueues.ReservationQueue, ",") != "101" {
		t.Fatalf("reservation queue = %#v, want 101", observation.GroupQueues.ReservationQueue)
	}
}

func TestStoreInfoGroupQueuesAcceptsMixedArraysAndSingleValues(t *testing.T) {
	var store StoreInfo
	if err := json.Unmarshal([]byte(`{
		"groupQueues": {
			"mixedQueue": [" 001 ", 2, true, null, "", "   "],
			"reservationQueue": " 101 ",
			"counterQueue": 202,
			"boothQueue": false,
			"unknownQueue": ["999"]
		}
	}`), &store); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(store.GroupQueues.MixedQueue, ","); got != "001,2,true" {
		t.Fatalf("mixed queue = %q, want 001,2,true", got)
	}
	if got := strings.Join(store.GroupQueues.ReservationQueue, ","); got != "101" {
		t.Fatalf("reservation queue = %q, want 101", got)
	}
	if got := strings.Join(store.GroupQueues.CounterQueue, ","); got != "202" {
		t.Fatalf("counter queue = %q, want 202", got)
	}
	if got := strings.Join(store.GroupQueues.BoothQueue, ","); got != "false" {
		t.Fatalf("booth queue = %q, want false", got)
	}
}

func TestStoreInfoGroupQueuesNullAndNonObjectParseAsEmpty(t *testing.T) {
	cases := []struct {
		name        string
		groupQueues string
	}{
		{name: "null", groupQueues: `null`},
		{name: "empty object", groupQueues: `{}`},
		{name: "number", groupQueues: `123`},
		{name: "string", groupQueues: `"unexpected"`},
		{name: "array", groupQueues: `["001"]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var store StoreInfo
			data := []byte(`{"id":3015,"groupQueues":` + tc.groupQueues + `}`)
			if err := json.Unmarshal(data, &store); err != nil {
				t.Fatalf("StoreInfo unmarshal failed: %v", err)
			}
			if queueGroupQueuesHasAny(store.GroupQueues) {
				t.Fatalf("group queues = %#v, want empty", store.GroupQueues)
			}
		})
	}
}

func TestQueueTrendUsesMixedQueueWhenDisplayCalledNoMissing(t *testing.T) {
	query := normalizeQueueTrendQuery(QueueTrendQuery{
		StoreIDs:      []string{"001"},
		DateType:      "weekday",
		From:          "2026-05-26",
		To:            "2026-05-26",
		Start:         "10:00",
		End:           "12:00",
		BucketMinutes: 30,
	}, time.Date(2026, 5, 26, 12, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	series := map[string]*queueTrendAccumulator{}
	summary := addQueueObservationsToTrend(series, QueueTrendSummary{}, query, []QueueObservation{
		{StoreID: "001", Timestamp: "2026-05-26T10:05:00+08:00", GroupQueues: QueueGroupQueues{MixedQueue: []string{"001", "002", "003"}}},
		{StoreID: "001", Timestamp: "2026-05-26T10:25:00+08:00", GroupQueues: QueueGroupQueues{MixedQueue: []string{"004", "005", "006"}}},
	}, map[string]string{"001": "门店"}, map[string]bool{}, map[string]bool{}, map[string]bool{})
	points := finalizeQueueTrendPoints(series)
	if len(points) != 1 {
		t.Fatalf("points len = %d, want 1", len(points))
	}
	if points[0].GlobalPassed != 3 || summary.GlobalPassedTotal != 3 {
		t.Fatalf("global passed = point %d summary %d, want 3", points[0].GlobalPassed, summary.GlobalPassedTotal)
	}
}

func TestQueueTrendWaitOnlyObservationDoesNotResetCalledNumberBaseline(t *testing.T) {
	query := normalizeQueueTrendQuery(QueueTrendQuery{
		StoreIDs:      []string{"001"},
		DateType:      "weekday",
		From:          "2026-05-26",
		To:            "2026-05-26",
		Start:         "10:00",
		End:           "12:00",
		BucketMinutes: 30,
	}, time.Date(2026, 5, 26, 12, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	series := map[string]*queueTrendAccumulator{}
	summary := addQueueObservationsToTrend(series, QueueTrendSummary{}, query, []QueueObservation{
		{StoreID: "001", Timestamp: "2026-05-26T10:00:00+08:00", DisplayCalledNo: 100},
		{StoreID: "001", Timestamp: "2026-05-26T10:10:00+08:00", WaitMinutes: 25},
		{StoreID: "001", Timestamp: "2026-05-26T10:20:00+08:00", DisplayCalledNo: 110},
	}, map[string]string{"001": "门店"}, map[string]bool{}, map[string]bool{}, map[string]bool{})

	points := finalizeQueueTrendPoints(series)
	if len(points) != 1 {
		t.Fatalf("points len = %d, want 1", len(points))
	}
	if points[0].GlobalPassed != 10 || summary.GlobalPassedTotal != 10 {
		t.Fatalf("global passed = point %d summary %d, want 10", points[0].GlobalPassed, summary.GlobalPassedTotal)
	}
	if points[0].GlobalSamples != 1 || summary.GlobalSamples != 1 {
		t.Fatalf("global samples = point %d summary %d, want 1", points[0].GlobalSamples, summary.GlobalSamples)
	}
	if points[0].ObservationSamples != 1 {
		t.Fatalf("observation samples = %d, want 1", points[0].ObservationSamples)
	}
	if points[0].WaitP50Minutes == nil || *points[0].WaitP50Minutes != 25 {
		t.Fatalf("wait p50 = %v, want 25", points[0].WaitP50Minutes)
	}
}

func TestQueueTrendUsesWaitMinutesFromPublicObservations(t *testing.T) {
	query := normalizeQueueTrendQuery(QueueTrendQuery{
		StoreIDs:      []string{"001"},
		DateType:      "weekday",
		From:          "2026-05-26",
		To:            "2026-05-26",
		Start:         "10:00",
		End:           "12:00",
		BucketMinutes: 30,
	}, time.Date(2026, 5, 26, 12, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	series := map[string]*queueTrendAccumulator{}
	observations := []QueueObservation{
		{StoreID: "001", Timestamp: "2026-05-26T10:05:00+08:00", WaitMinutes: 20},
		{StoreID: "001", Timestamp: "2026-05-26T10:25:00+08:00", WaitMinutes: 40},
	}
	addQueueObservationsToTrend(series, QueueTrendSummary{}, query, observations, map[string]string{"001": "门店"}, map[string]bool{}, map[string]bool{}, map[string]bool{})
	points := finalizeQueueTrendPoints(series)
	if len(points) != 1 {
		t.Fatalf("points len = %d, want 1", len(points))
	}
	if points[0].ObservationSamples != 2 {
		t.Fatalf("observation samples = %d, want 2", points[0].ObservationSamples)
	}
	if points[0].WaitP50Minutes == nil || *points[0].WaitP50Minutes != 30 {
		t.Fatalf("wait p50 = %v, want 30", points[0].WaitP50Minutes)
	}
}

func TestBuildQueueTrendRecommendationsPrioritizesUsableStoreTime(t *testing.T) {
	waitFast := 25.0
	waitSlow := 70.0
	points := []QueueTrendPoint{
		{
			StoreID:        "slow",
			StoreName:      "慢店",
			DateType:       "weekend",
			DateTypeName:   "周末",
			Bucket:         "18:30",
			ActualSamples:  8,
			GlobalSamples:  6,
			WaitP50Minutes: &waitSlow,
			Confidence:     "high",
			MissedRate:     0.1,
		},
		{
			StoreID:        "fast",
			StoreName:      "快店",
			DateType:       "weekend",
			DateTypeName:   "周末",
			Bucket:         "17:30",
			ActualSamples:  4,
			GlobalSamples:  5,
			WaitP50Minutes: &waitFast,
			Confidence:     "medium",
			MissedRate:     0.0,
		},
	}

	recommendations := BuildQueueTrendRecommendations(points, 2)
	if len(recommendations) != 2 {
		t.Fatalf("recommendations len = %d, want 2", len(recommendations))
	}
	if recommendations[0].StoreID != "fast" {
		t.Fatalf("top store = %s, want fast: %+v", recommendations[0].StoreID, recommendations)
	}
	if recommendations[0].ActionLabel != "优先考虑" {
		t.Fatalf("action = %s, want 优先考虑", recommendations[0].ActionLabel)
	}
	if !strings.Contains(recommendations[0].Reason, "P50") {
		t.Fatalf("reason should mention P50 wait: %s", recommendations[0].Reason)
	}
}

func TestBuildQueueTrendRecommendationsSkipsEmptyPoints(t *testing.T) {
	recommendations := BuildQueueTrendRecommendations([]QueueTrendPoint{
		{StoreID: "empty", StoreName: "空", DateType: "weekday", Bucket: "12:00"},
	}, 5)
	if len(recommendations) != 0 {
		t.Fatalf("recommendations len = %d, want 0", len(recommendations))
	}
}

func TestQueueSessionBucketNormalizesToSushiroTimezone(t *testing.T) {
	// 2026-05-16T10:00:00Z 与 18:00 CST 是同一时刻。桶键按 CST 语义定义，
	// UTC 机器上不能把周六晚餐高峰劈成"周六 10:00"。
	session := QueueSession{
		StoreID:  "001",
		TakenAt:  "2026-05-16T10:00:00Z",
		TicketNo: 100, DisplayCalledNoAtTake: 80, CalledNoWhenUserCalled: 100,
		ActualWaitMinutes: 30, PartySize: 2, TableType: "T",
	}
	bucket, ok := queueSessionBucket(session)
	if !ok {
		t.Fatal("expected bucket to be usable")
	}
	if bucket.timeBucket != "18:00" {
		t.Fatalf("timeBucket = %s, want 18:00 (CST)", bucket.timeBucket)
	}
	if bucket.weekday != 6 {
		t.Fatalf("weekday = %d, want 6 (Saturday in CST)", bucket.weekday)
	}
}

func TestQueueTrendWeekendWindowBoundaries(t *testing.T) {
	cst := SushiroTimezone
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"周五16:29 仍是工作日节奏", time.Date(2026, 8, 14, 16, 29, 0, 0, cst), false},
		{"周五16:30 进入周末窗口", time.Date(2026, 8, 14, 16, 30, 0, 0, cst), true},
		{"周六全天周末", time.Date(2026, 8, 15, 9, 0, 0, 0, cst), true},
		{"周日21:59 仍周末", time.Date(2026, 8, 16, 21, 59, 0, 0, cst), true},
		{"周日22:00 回落工作日", time.Date(2026, 8, 16, 22, 0, 0, 0, cst), false},
	}
	for _, tc := range cases {
		if got := queueTrendWeekendWindow(tc.at); got != tc.want {
			t.Errorf("%s: weekendWindow = %v, want %v", tc.name, got, tc.want)
		}
	}
}
