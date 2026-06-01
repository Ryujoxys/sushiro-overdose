package main

import "testing"

func TestQueueAlertWaitBelowHysteresis(t *testing.T) {
	rule := QueueAlertRule{StoreID: "3006", Type: queueAlertWaitBelow, WaitMinutes: 60, Enabled: true}
	state := map[string]queueAlertRuleState{}

	// 等待高于阈值+迟滞，只武装不推送。
	if _, _, fire := queueAlertEvaluateRule(rule, QueueObservation{WaitMinutes: 100}, state); fire {
		t.Fatal("high wait should not fire")
	}
	if !state[rule.key()].Armed {
		t.Fatal("rule should be armed after high wait")
	}
	// 等待落到阈值内且已武装，推送一次并解除武装。
	if _, _, fire := queueAlertEvaluateRule(rule, QueueObservation{WaitMinutes: 50}, state); !fire {
		t.Fatal("wait dropping below threshold should fire")
	}
	if state[rule.key()].Armed {
		t.Fatal("rule should disarm after firing")
	}
	// 仍然低位但未重新武装，不再重复推送。
	if _, _, fire := queueAlertEvaluateRule(rule, QueueObservation{WaitMinutes: 50}, state); fire {
		t.Fatal("should not re-fire while disarmed")
	}
}

func TestQueueAlertCalledReachOnce(t *testing.T) {
	rule := QueueAlertRule{StoreID: "3006", Type: queueAlertCalledReach, TargetNo: 900, LeadGroups: 5, Enabled: true}
	state := map[string]queueAlertRuleState{}

	if _, _, fire := queueAlertEvaluateRule(rule, QueueObservation{DisplayCalledNo: 890}, state); fire {
		t.Fatal("called 890 vs target-lead 895 should not fire")
	}
	if _, _, fire := queueAlertEvaluateRule(rule, QueueObservation{DisplayCalledNo: 896}, state); !fire {
		t.Fatal("called 896 should fire")
	}
	if !state[rule.key()].FiredOnce {
		t.Fatal("called_reach should mark FiredOnce")
	}
	if _, _, fire := queueAlertEvaluateRule(rule, QueueObservation{DisplayCalledNo: 905}, state); fire {
		t.Fatal("called_reach should fire only once")
	}
}
