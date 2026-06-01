package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	queueAlertConfigFile = "queue_alerts.json"
	queueAlertStateFile  = "queue_alert_state.json"

	queueAlertWaitBelow   = "wait_below"   // 预估等待低于阈值时提醒（该去取号了）
	queueAlertCalledReach = "called_reach" // 叫号接近我手里的号时提醒

	// wait_below 的迟滞缓冲：等待回升超过阈值+该值才会重新武装，避免临界点反复推送。
	queueAlertWaitHysteresis = 15
)

// QueueAlertRule 是一条叫号提醒规则。
type QueueAlertRule struct {
	StoreID     string `json:"store_id"`
	StoreName   string `json:"store_name,omitempty"`
	Type        string `json:"type"`
	WaitMinutes int    `json:"wait_minutes,omitempty"` // wait_below：预估等待 ≤ 此值时提醒
	TargetNo    int    `json:"target_no,omitempty"`    // called_reach：我手里的号
	LeadGroups  int    `json:"lead_groups,omitempty"`  // called_reach：提前多少号提醒
	Enabled     bool   `json:"enabled"`
}

func (r QueueAlertRule) key() string {
	return fmt.Sprintf("%s|%s|%d|%d", r.StoreID, r.Type, r.WaitMinutes, r.TargetNo)
}

type QueueAlertConfig struct {
	Rules []QueueAlertRule `json:"rules"`
}

// queueAlertRuleState 是单条规则的去重状态。
type queueAlertRuleState struct {
	Armed     bool   `json:"armed"`      // wait_below 是否已武装（等待曾高于阈值）
	FiredAt   string `json:"fired_at"`   // 上次推送时间
	FiredOnce bool   `json:"fired_once"` // called_reach 是否已推送过
}

var queueAlertMu sync.Mutex

func queueAlertConfigPath() string { return filepath.Join(appDirPath(), queueAlertConfigFile) }
func queueAlertStatePath() string  { return filepath.Join(appDirPath(), queueAlertStateFile) }

func LoadQueueAlertConfig() QueueAlertConfig {
	data, err := os.ReadFile(queueAlertConfigPath())
	if err != nil {
		return QueueAlertConfig{}
	}
	var cfg QueueAlertConfig
	if json.Unmarshal(data, &cfg) != nil {
		return QueueAlertConfig{}
	}
	return cfg
}

func SaveQueueAlertConfig(cfg QueueAlertConfig) error {
	os.MkdirAll(appDirPath(), 0o755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(queueAlertConfigPath(), data, 0o600)
}

func loadQueueAlertState() map[string]queueAlertRuleState {
	out := map[string]queueAlertRuleState{}
	data, err := os.ReadFile(queueAlertStatePath())
	if err != nil {
		return out
	}
	_ = json.Unmarshal(data, &out)
	return out
}

func saveQueueAlertState(state map[string]queueAlertRuleState) {
	os.MkdirAll(appDirPath(), 0o755)
	if data, err := json.MarshalIndent(state, "", "  "); err == nil {
		_ = os.WriteFile(queueAlertStatePath(), data, 0o600)
	}
}

// evaluateQueueAlerts 在每轮采样写入一条排队观测后调用，评估该门店的提醒规则并推送。
// obs 是 getStoreById 解析出的观测（含当前叫号/预估等待/在等组数）。
func evaluateQueueAlerts(ctx context.Context, obs QueueObservation, storeName string) {
	cfg := LoadQueueAlertConfig()
	if len(cfg.Rules) == 0 {
		return
	}
	storeID := strings.TrimSpace(obs.StoreID)

	queueAlertMu.Lock()
	defer queueAlertMu.Unlock()
	state := loadQueueAlertState()
	changed := false

	for _, rule := range cfg.Rules {
		if !rule.Enabled || rule.StoreID != storeID {
			continue
		}
		title, body, fire := queueAlertEvaluateRule(rule, obs, state)
		if !fire {
			continue
		}
		changed = true
		name := strings.TrimSpace(rule.StoreName)
		if name == "" {
			name = strings.TrimSpace(storeName)
		}
		if name == "" {
			name = storeID
		}
		sendQueueAlert(ctx, title, name+"："+body)
	}
	if changed {
		saveQueueAlertState(state)
	}
}

// queueAlertEvaluateRule 评估单条规则，更新 state，返回是否需要推送及内容。
func queueAlertEvaluateRule(rule QueueAlertRule, obs QueueObservation, state map[string]queueAlertRuleState) (string, string, bool) {
	key := rule.key()
	st := state[key]

	switch rule.Type {
	case queueAlertWaitBelow:
		wait := obs.WaitMinutes
		if wait > rule.WaitMinutes+queueAlertWaitHysteresis {
			// 等待较高，武装规则，等它落下来再提醒。
			if !st.Armed {
				st.Armed = true
				state[key] = st
			}
			return "", "", false
		}
		if wait <= rule.WaitMinutes && st.Armed {
			st.Armed = false
			st.FiredAt = time.Now().Format(time.RFC3339)
			state[key] = st
			return "🍣 可以去取号了",
				fmt.Sprintf("预计等待已降到约 %d 分钟（阈值 %d），在等 %d 桌。", wait, rule.WaitMinutes, obs.GroupQueuesCount),
				true
		}
		return "", "", false

	case queueAlertCalledReach:
		calledNo := obs.DisplayCalledNo
		if st.FiredOnce || calledNo <= 0 {
			return "", "", false
		}
		if calledNo >= rule.TargetNo-rule.LeadGroups {
			st.FiredOnce = true
			st.FiredAt = time.Now().Format(time.RFC3339)
			state[key] = st
			return "🔔 快叫到你了",
				fmt.Sprintf("当前叫号 %d，你的号 %d，还差 %d 桌，请尽快回店。", calledNo, rule.TargetNo, max(0, rule.TargetNo-calledNo)),
				true
		}
		return "", "", false
	}
	return "", "", false
}

func sendQueueAlert(ctx context.Context, title, content string) {
	logMessage(time.Now(), fmt.Sprintf("[排队提醒] %s — %s", title, content))
	BuildNotifierFromConfig().Send(ctx, title, content)
}
