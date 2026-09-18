package app

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type QueueAlertStatusResponse struct {
	GeneratedAt      string                       `json:"generated_at"`
	Config           QueueAlertConfig             `json:"config"`
	Rules            []QueueAlertRuleStatus       `json:"rules"`
	Sampling         QueueSamplingStatus          `json:"sampling"`
	PublicCollection PublicQueueCollectionStatus  `json:"public_collection"`
	Notifications    QueueAlertNotificationStatus `json:"notifications"`
	Warnings         []string                     `json:"warnings,omitempty"`
}

type QueueAlertNotificationStatus struct {
	Configured bool     `json:"configured"`
	Channels   []string `json:"channels"`
}

type QueueAlertRuleStatus struct {
	EvaluatedAt                string         `json:"evaluated_at,omitempty"`
	Key                        string         `json:"key"`
	Rule                       QueueAlertRule `json:"rule"`
	Label                      string         `json:"label,omitempty"`
	Threshold                  int            `json:"threshold,omitempty"`
	CurrentCalledNo            int            `json:"current_called_no,omitempty"`
	RemainingToThreshold       int            `json:"remaining_to_threshold,omitempty"`
	RemainingToTicket          int            `json:"remaining_to_ticket,omitempty"`
	EstimateToThresholdMinutes *int           `json:"estimate_to_threshold_minutes,omitempty"`
	EstimateToTicketMinutes    *int           `json:"estimate_to_ticket_minutes,omitempty"`
	LatestAt                   string         `json:"latest_at,omitempty"`
	Fired                      bool           `json:"fired"`
	FiredAt                    string         `json:"fired_at,omitempty"`
	Next                       bool           `json:"next"`
	Status                     string         `json:"status"`
	StatusText                 string         `json:"status_text"`
	StoreWarnings              []string       `json:"store_warnings,omitempty"`
}

func BuildQueueAlertStatus(now time.Time) QueueAlertStatusResponse {
	if now.IsZero() {
		now = time.Now()
	}
	// 加读锁与 evaluateQueueAlerts/SaveQueueAlertConfig 的写互斥：状态接口被 HTTP 调用时
	// 后台采样可能正并发写 config/state。无锁 + 非原子写并发可能读到「新 config + 旧 state」
	// 中间态，甚至两个写者并发写同一文件产生损坏 JSON（下一轮 loadQueueAlertState 解析失败
	// 返回空 map → 全部 FiredOnce 丢失 → 所有阈值一次性重新触发）。
	queueAlertMu.Lock()
	defer queueAlertMu.Unlock()
	cfg := LoadQueueAlertConfig()
	observations := loadQueueObservations()
	latestByStore := latestQueueObservationsByStore(observations)
	state := loadQueueAlertState()
	summary := queueAlertSamplingSummary(observations)
	response := QueueAlertStatusResponse{
		GeneratedAt:      now.Format(time.RFC3339),
		Config:           cfg,
		Sampling:         buildQueueSamplingStatus(now, summary),
		PublicCollection: queueBaselineCollector.status(),
		Notifications:    queueAlertNotificationStatus(),
	}

	rates := map[string]float64{}
	warnings := map[string][]string{}
	for _, rule := range cfg.Rules {
		storeID := strings.TrimSpace(rule.StoreID)
		if _, ok := rates[storeID]; !ok {
			if rate, hasRate := queueAlertStoreRate(observations, storeID, now); hasRate {
				rates[storeID] = rate
			}
		}
		if _, ok := warnings[storeID]; !ok {
			warnings[storeID] = queueAlertStoreWarnings(observations, storeID, now)
			response.Warnings = append(response.Warnings, warnings[storeID]...)
		}
		status := buildQueueAlertRuleStatus(rule, latestByStore[storeID], state[rule.key()], rates[storeID])
		status.StoreWarnings = warnings[storeID]
		response.Rules = append(response.Rules, status)
	}
	markNextQueueAlertRules(response.Rules)
	return response
}

func queueAlertSamplingSummary(observations []QueueObservation) QueueTrendSummary {
	summary := QueueTrendSummary{ObservationRecords: len(observations)}
	for _, observation := range observations {
		if at, ok := parseRFC3339Local(queueObservationCollectedAt(observation)); ok {
			updateLatest(&summary.LastObservationAt, at)
		}
	}
	return summary
}

func queueAlertNotificationStatus() QueueAlertNotificationStatus {
	notifier := configuredNotifiers()
	list := notifier.List()
	out := QueueAlertNotificationStatus{Configured: len(list) > 0}
	for _, item := range list {
		out.Channels = append(out.Channels, item.Name())
	}
	sort.Strings(out.Channels)
	return out
}

func latestQueueObservationsByStore(observations []QueueObservation) map[string]QueueObservation {
	out := map[string]QueueObservation{}
	for _, observation := range observations {
		if strings.TrimSpace(observation.StoreID) == "" {
			continue
		}
		currentAt, ok := parseRFC3339Local(queueObservationCollectedAt(observation))
		if !ok {
			continue
		}
		existing, exists := out[observation.StoreID]
		if !exists {
			out[observation.StoreID] = observation
			continue
		}
		existingAt, ok := parseRFC3339Local(queueObservationCollectedAt(existing))
		if !ok || currentAt.After(existingAt) {
			out[observation.StoreID] = observation
		}
	}
	return out
}

func queueAlertStoreRate(observations []QueueObservation, storeID string, now time.Time) (float64, bool) {
	recent := recentStoreObservations(observations, storeID, now, queuePanelRateWindow)
	return calledRatePerMinute(recent)
}

func buildQueueAlertRuleStatus(rule QueueAlertRule, latest QueueObservation, state queueAlertRuleState, rate float64) QueueAlertRuleStatus {
	threshold := rule.calledReachThreshold()
	calledNo := latest.DisplayCalledNo
	remainingToThreshold := max(0, threshold-calledNo)
	remainingToTicket := max(0, rule.TargetNo-calledNo)
	status := QueueAlertRuleStatus{
		EvaluatedAt:          state.EvaluatedAt,
		Key:                  rule.key(),
		Rule:                 rule,
		Label:                queueAlertLabel(rule),
		Threshold:            threshold,
		CurrentCalledNo:      calledNo,
		RemainingToThreshold: remainingToThreshold,
		RemainingToTicket:    remainingToTicket,
		LatestAt:             queueObservationCollectedAt(latest),
		Fired:                state.FiredOnce,
		FiredAt:              state.FiredAt,
	}
	if rate > 0 {
		status.EstimateToThresholdMinutes = estimateMinutes(remainingToThreshold, rate)
		status.EstimateToTicketMinutes = estimateMinutes(remainingToTicket, rate)
	}
	switch {
	case !rule.Enabled:
		status.Status = "disabled"
		status.StatusText = "已停用"
	case state.FiredOnce:
		status.Status = "fired"
		status.StatusText = "已提醒"
	case calledNo <= 0:
		status.Status = "waiting_data"
		status.StatusText = "等待采样"
	case threshold > 0 && calledNo >= threshold:
		status.Status = "due"
		status.StatusText = "已到提醒点，等待下一轮推送"
	default:
		status.Status = "armed"
		status.StatusText = "监控中"
	}
	return status
}

func estimateMinutes(gap int, rate float64) *int {
	if gap <= 0 || rate <= 0 {
		value := 0
		return &value
	}
	value := int(math.Ceil(float64(gap) / rate))
	return &value
}

func markNextQueueAlertRules(rules []QueueAlertRuleStatus) {
	nextByStoreTicket := map[string]int{}
	for i, rule := range rules {
		if rule.Fired || rule.Status == "disabled" {
			continue
		}
		key := rule.Rule.StoreID + "|" + rule.Rule.Type + "|" + strconv.Itoa(rule.Rule.TargetNo)
		current, ok := nextByStoreTicket[key]
		if !ok || rule.Threshold < rules[current].Threshold {
			nextByStoreTicket[key] = i
		}
	}
	for _, index := range nextByStoreTicket {
		rules[index].Next = true
	}
}

func queueAlertStoreWarnings(observations []QueueObservation, storeID string, now time.Time) []string {
	recent := recentStoreObservations(observations, storeID, now, queuePanelRecentWindow)
	if len(recent) < 2 {
		return nil
	}
	first := recent[0]
	last := recent[len(recent)-1]
	if last.DisplayCalledNo < first.DisplayCalledNo {
		return []string{"叫号出现回退，可能是官方队列重置或口径变化。"}
	}
	advance := last.DisplayCalledNo - first.DisplayCalledNo
	switch {
	case advance == 0:
		return []string{"近15分钟叫号没有推进，建议不要只按历史均速判断。"}
	case advance >= 80:
		return []string{"近15分钟叫号跳号较大，可能会比预计更快到号。"}
	default:
		return nil
	}
}
