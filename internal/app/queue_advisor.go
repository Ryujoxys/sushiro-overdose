package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// queue_advisor.go 把实时快照 + 本机采样 + 历史趋势合成「排队压力 + 时间答案」。
// 用户侧统一用「排队压力 / 预计等待 / 叫号速度 / 消化趋势」，不引入「最新发出号」之类口径。
// 大部分推算复用 queue_live_panel.go / queue_alert_status.go / queue_trends.go 的现成 helper。

const (
	queueAdvisorWindow15 = 15 * time.Minute
	queueAdvisorWindow30 = 30 * time.Minute
	queueAdvisorWindow60 = 60 * time.Minute

	queuePressureCurveLocalPreferredPoints = 8
)

type QueueTimeRange struct {
	Early string `json:"early,omitempty"`
	Late  string `json:"late,omitempty"`
}

type QueueWaitRange struct {
	Low  int `json:"low"`
	High int `json:"high"`
}

type QueueAdvisorCurrent struct {
	CalledNo            int    `json:"called_no"`
	WaitingGroups       int    `json:"waiting_groups"`
	OfficialWaitMinutes int    `json:"official_wait_minutes"`
	StoreStatus         string `json:"store_status"`
	NetTicketStatus     string `json:"net_ticket_status"`
	OnlineOpen          bool   `json:"online_open"`
}

type QueueAdvisorPressure struct {
	Level      string `json:"level"`       // low/medium/high/extreme/unknown
	Label      string `json:"label"`       // 低/中/高/极高/数据不足
	Score      int    `json:"score"`       // 0-100
	Trend      string `json:"trend"`       // improving/stable/worsening/stalled/unknown
	TrendLabel string `json:"trend_label"` // 正在变好/基本稳定/正在变差/叫号停滞/数据不足
	Reason     string `json:"reason"`
}

type QueueAdvisorSpeed struct {
	CalledPerMin15 *float64 `json:"called_per_min_15,omitempty"`
	CalledPerMin30 *float64 `json:"called_per_min_30,omitempty"`
	CalledPerMin60 *float64 `json:"called_per_min_60,omitempty"`
}

type QueueAdvisorEta struct {
	TargetNo               int             `json:"target_no"`
	RemainingGroups        int             `json:"remaining_groups"`
	EstimatedCalledAt      string          `json:"estimated_called_at,omitempty"`
	EstimatedCalledAtRange *QueueTimeRange `json:"estimated_called_at_range,omitempty"`
	WaitMinutesRange       *QueueWaitRange `json:"wait_minutes_range,omitempty"`
	ArrivalSuggestion      string          `json:"arrival_suggestion,omitempty"`
	Source                 string          `json:"source,omitempty"`
	SourceLabel            string          `json:"source_label,omitempty"`
	SourceNote             string          `json:"source_note,omitempty"`
	Risk                   string          `json:"risk"` // low/medium/high/unknown
}

type QueueAdvisor struct {
	StoreID        string               `json:"store_id"`
	StoreName      string               `json:"store_name"`
	GeneratedAt    string               `json:"generated_at"`
	Current        QueueAdvisorCurrent  `json:"current"`
	Pressure       QueueAdvisorPressure `json:"pressure"`
	Speed          QueueAdvisorSpeed    `json:"speed"`
	Eta            *QueueAdvisorEta     `json:"eta,omitempty"`
	SamplingPoints int                  `json:"sampling_points"`
	Warnings       []string             `json:"warnings,omitempty"`
}

// ---------- 排队压力模型（纯函数，便于测试） ----------

// queuePressureLevel 用当前等待桌数与官方等待分钟综合判档；两者都缺为 unknown。
func queuePressureLevel(waitGroups, waitMinutes int) string {
	if waitGroups <= 0 && waitMinutes <= 0 {
		return "unknown"
	}
	switch {
	case (waitGroups > 0 && waitGroups <= 20) || (waitMinutes > 0 && waitMinutes <= 20):
		return "low"
	case (waitGroups > 0 && waitGroups <= 60) || (waitMinutes > 0 && waitMinutes <= 60):
		return "medium"
	case (waitGroups > 0 && waitGroups <= 120) || (waitMinutes > 0 && waitMinutes <= 120):
		return "high"
	default:
		return "extreme"
	}
}

func queuePressureLabel(level string) string {
	switch level {
	case "low":
		return "低"
	case "medium":
		return "中"
	case "high":
		return "高"
	case "extreme":
		return "极高"
	default:
		return "数据不足"
	}
}

// queuePressureScore 把等待桌数/分钟单调映射到 0-100，用于面积/柱高度。
func queuePressureScore(waitGroups, waitMinutes int) int {
	if waitGroups <= 0 && waitMinutes <= 0 {
		return 0
	}
	byGroups := float64(waitGroups) / 160.0 * 100.0
	byMinutes := float64(waitMinutes) / 160.0 * 100.0
	score := math.Max(byGroups, byMinutes)
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return int(math.Round(score))
}

// queuePressureTrend 用近窗（默认 15 分钟）等待桌数首尾差 + 叫号速度判断消化趋势。
// stalled 优先级最高：监控判定叫号停滞时直接报 stalled，覆盖其它趋势（连号都不动时趋势无意义）。
// delta>=8 桌算 worsening、<=-8 桌算 improving，中间若无有效叫号速度（rate<=0）报 unknown
// 而非 stable——避免在没推进证据时误判「基本稳定」。
func queuePressureTrend(recent []QueueObservation, rate float64, stalled bool) (string, string) {
	if stalled {
		return "stalled", "叫号停滞"
	}
	if len(recent) < 2 {
		return "unknown", "数据不足"
	}
	first := recent[0].GroupQueuesCount
	last := recent[len(recent)-1].GroupQueuesCount
	if first <= 0 && last <= 0 {
		return "unknown", "数据不足"
	}
	delta := last - first
	switch {
	case delta <= -8:
		return "improving", "正在变好"
	case delta >= 8:
		return "worsening", "正在变差"
	case rate <= 0:
		return "unknown", "数据不足"
	default:
		return "stable", "基本稳定"
	}
}

func queuePressureReason(level, trend string, waitGroups int, rate float64, called15 *int) string {
	parts := []string{}
	if waitGroups > 0 {
		parts = append(parts, fmt.Sprintf("当前在等约 %d 桌", waitGroups))
	}
	if called15 != nil {
		parts = append(parts, fmt.Sprintf("近 15 分钟叫了 %d 桌", *called15))
	} else if rate > 0 {
		parts = append(parts, fmt.Sprintf("近期约 %.1f 桌/分钟", rate))
	}
	switch trend {
	case "improving":
		parts = append(parts, "队伍在加速消化")
	case "worsening":
		parts = append(parts, "积压还在变多")
	case "stalled":
		parts = append(parts, "叫号暂时没有推进")
	}
	if len(parts) == 0 {
		return "实时数据不足，先按官方等待参考"
	}
	return strings.Join(parts, "，") + "。"
}

// ---------- 叫号速度（多窗口） ----------

// calledRateOverWindow 复用 recentStoreObservations + calledRatePerMinute 算某窗口（15/30/60min）的叫号速度。
// 返回 nil 的两种情况：窗口内有效采样点不足 2 个，或叫号速度<=0（首尾叫号没推进/出现回退，见 calledRatePerMinute 的忽略回退逻辑）。
func calledRateOverWindow(all []QueueObservation, storeID string, now time.Time, window time.Duration) *float64 {
	recent := recentStoreObservations(all, storeID, now, window)
	if len(recent) < 2 {
		return nil
	}
	rate, ok := calledRatePerMinute(recent)
	if !ok || rate <= 0 {
		return nil
	}
	r := math.Round(rate*10) / 10
	return &r
}

// ---------- 预计等待区间 ----------

// estimateWaitRange 预计等待分钟区间，按可信度从高到低分三档回退：
//  1. recent_speed：有叫号速度时，用 remaining/rate 推算，并让官方等待修正下界；
//  2. official：没有叫号速度但有官方等待分钟，按官方值上下浮动（只能反映门店整体压力，无法定位到用户号码）；
//  3. history：前两者都缺，回退到同门店同日型的历史 P50/P80；
//  4. unknown：三者都没有，返回 nil 让上游提示数据不足。
//
// 为什么用 officialWait 抬高低界：官方等待是「门店整体」的等位分钟，用户号码靠后时
// 本机 remaining/rate 算出的值可能反而比官方小（号码靠后反而算得快），但这在物理上不可能——
// 用户不可能比门店整体更快叫到号。所以把 low 钳到 max(low, officialWait)，并同步抬高 high，
// 避免给用户一个与门店实际压力自相矛盾的过短区间。
func estimateWaitRange(waitGroups, officialWait int, rate, cv float64, realtimeN int, trend float64, rates []float64, hist *QueueWaitRange) (*QueueWaitRange, string) {
	if rate > 0 && waitGroups > 0 {
		// 档位 1：有叫号速度。base = 剩余桌数 / 叫号速度（组/分）。
		//
		// 【C 实时/历史融合】把历史等待分钟转成隐含速率做先验，与实时加权。
		// 单位统一：历史是分钟、实时是组/分，把历史按 waitGroups 锚成 rateHist=waitGroups/histMid。
		effRate := rate
		source := "recent_speed"
		hasHist := hist != nil && (hist.Low > 0 || hist.High > 0)
		if hasHist {
			histMid := float64(hist.Low+hist.High) / 2.0
			if histMid > 0 {
				w := realtimeBlendWeight(realtimeN, cv, true)
				rateHist := float64(waitGroups) / histMid
				effRate = w*rate + (1-w)*rateHist
				if effRate <= 0 {
					effRate = rate
				}
				// source：w>=0.5 实时为主(recent_speed)；0<w<0.5 历史为主(blended)；
				// w==0 完全历史(history，走下面 base 计算仍用 effRate=rateHist)。
				if w == 0 {
					source = "history"
				} else if w < 0.5 {
					source = "blended"
				}
			}
		}

		// 【趋势前瞻校正】叫号节奏在加速(trend>1)时，未来会比过去更快，纯用历史速度外推
		// 会系统性高估等待（用户看到的「预计时间」偏晚）。用 sqrt(trend) 温和校正：
		// 加速2倍只让预测速度×1.41（而非×2），平衡「修正渐变高峰高估」与「不过拟合噪声/不过校正」。
		// trend=1（稳定/低峰）sqrt(1)=1，预测不受影响——满足「低峰期基本不偏移」。
		// 守卫：trend 由实时短窗算出（rateTrendRatio 要求 n>=4 才非 1）。realtimeN<4 时趋势是
		// 1~2 个间隔驱动的噪声（钳到[0.5,2.0]的极端值），此时 source 常为 blended（实时权重 w 很小、
		// effRate 几乎纯历史），把噪声趋势乘到稳定历史上会让 ETA 被偶发抖动系统性偏移。
		// 故要求 realtimeN>=4（与 rateTrendRatio 自身门槛对齐）才套用前瞻校正。
		// 均值回归收缩(shrinkTrend)：观测趋势先向 1 收缩（样本越多越采信），避免把饭点周期性
		// 加速当成持续加速、系统性高估未来速度（ETA 偏早）。
		if source != "history" && trend != 1.0 && realtimeN >= 4 {
			effRate = effRate * math.Sqrt(shrinkTrend(trend, realtimeN))
		}

		// base = 剩余桌数 / 融合后速度。
		base := float64(waitGroups) / effRate
		// soft floor 仅在融合路径（历史参与）时生效：防止 histMid 极小（历史说该时段 2min）
		// 导致 rateHist 爆炸、算出「2 分钟叫到 200 桌」的离谱值。纯实时路径（无历史）不该触发——
		// 实时叫号速度本身就反映真实节奏，高峰门店能到 4~5 组/分，硬性 2 组/分上限会误伤。
		// floor 取 0.25min/组（4 组/分上限），覆盖真实门店高峰，同时仍兜住离谱值。
		if hasHist {
			if floorBase := float64(waitGroups) * 0.25; base < floorBase {
				base = floorBase
			}
		}

		// 【B 动态区间宽度】优先用经验分位数（非参数、非对称，天然贴合右偏等待分布）；
		// 样本不足时回退到旧的 CV→对称系数查表（仍可用，只是尾部覆盖不如分位数稳健）。
		//
		// 分位数区间思路：叫号等待 = 剩余桌数 / 速度，速度大→等待短。窗口内瞬时速率 rates 的
		// 分位数直接刻画「快慢分布」：q90（快）对应下界、q10（慢）对应上界。为保留 effRate 已做的
		// 历史融合/趋势校正，不直接用 remaining/分位数（会丢掉融合），而是以 base=remaining/effRate
		// 为中心锚，用 rates 的「相对离散度」(q10/q50, q90/q50) 决定非对称宽度——这样区间形状来自
		// 实测分布（右偏：慢尾拉长上界），中心仍锚在融合后的最佳估计上。
		var low, high int
		if lowRaw, highRaw, ok := quantileWaitBounds(base, rates); ok {
			low = int(math.Floor(lowRaw))
			high = int(math.Ceil(highRaw))
		} else {
			lowMul, highMul := waitRangeMultipliers(cv)
			low = int(math.Floor(base * lowMul))
			high = int(math.Ceil(base * highMul))
		}
		if officialWait > 0 {
			// 官方等待是门店整体等待分钟数，用户号码靠后时本机 remaining/rate 可能远大于它；
			// 但用户不可能比门店整体更快叫到，所以用它抬高下界（max），并校正上界，避免区间与门店压力矛盾。
			// 这是物理硬约束，统计融合(B/C)无权推翻。
			if officialWait > low {
				low = officialWait
			}
			if officialWait > high {
				high = officialWait
			}
		}
		if high < low {
			high = low
		}
		return &QueueWaitRange{Low: max(0, low), High: max(0, high)}, source
	}
	if officialWait > 0 {
		// 档位 2：没有叫号速度，只能用官方等待做粗估。下游会标记 source=official 并提示
		// 这只是门店整体压力、不能可靠定位到用户号码。
		low := int(math.Floor(float64(officialWait) * 0.9))
		high := int(math.Ceil(float64(officialWait) * 1.15))
		return &QueueWaitRange{Low: low, High: high}, "official"
	}
	if hist != nil && (hist.Low > 0 || hist.High > 0) {
		// 档位 3：历史 P50/P80。防御区间反序：脏基准可能 safe<typical 导致 P80<P50，
		// 这时把 high 钳到 low，避免上界小于下界（下游出发时间计算会出错）。
		low, high := hist.Low, hist.High
		if high < low {
			high = low
		}
		return &QueueWaitRange{Low: max(0, low), High: max(0, high)}, "history"
	}
	return nil, "unknown"
}

// ---------- advisor 主入口 ----------

// buildQueueAdvisor 组装「排队压力 + 时间答案」的对外结构。
// 数据来源：实时快照（snapshot，当前一次 GetStore）+ 本机历史采样（all）+ 官方等待/封顶（store）。
// 注意 history/recent15 都把实时 snapshot 临时 append 进去（不落盘），让速率/趋势窗口能用到「当前」这一刻。
func buildQueueAdvisor(ctx context.Context, storeID string, targetNo, travelMinutes int, now time.Time) (QueueAdvisor, error) {
	if now.IsZero() {
		now = time.Now()
	}
	store, err := NewQueueLiveClient().GetStore(ctx, storeID)
	if err != nil {
		return QueueAdvisor{}, err
	}
	snapshot := queueObservationFromLiveStore(store, now)

	all := loadQueueObservations()
	history := recentStoreObservations(all, snapshot.StoreID, now, queuePanelRateWindow)
	if snapshot.DisplayCalledNo > 0 {
		history = append(history, snapshot)
	}
	recent15 := recentStoreObservations(all, snapshot.StoreID, now, queueAdvisorWindow15)
	if snapshot.DisplayCalledNo > 0 {
		recent15 = append(recent15, snapshot)
	}

	warnings := queueAlertStoreWarnings(all, snapshot.StoreID, now)
	// stalled：告警里出现「没有推进」即认定叫号停滞。它会让 queuePressureTrend 直接判 stalled，
	// 优先级高于速度/桌数差，避免叫号卡死时还误报「稳定/变好」。
	stalled := false
	for _, w := range warnings {
		if strings.Contains(w, "没有推进") {
			stalled = true
		}
	}

	rate60 := calledRateOverWindow(all, snapshot.StoreID, now, queueAdvisorWindow60)
	// ETA 速度用近 30 分钟窗：门店叫号是短期节奏，近窗更能反映「当前多快」，
	// 不会被窗口边缘的旧慢速稀释（2h 窗下即使时间加权，几十分钟前的慢段仍会拖低速度）。
	// 仍顺带取 cv 和有效间隔数 n，供 ETA 的区间宽度(B)与实时/历史融合(C)使用。
	recent30obs := recentStoreObservations(all, snapshot.StoreID, now, queueAdvisorWindow30)
	if snapshot.DisplayCalledNo > 0 {
		recent30obs = append(recent30obs, snapshot)
	}
	rate := 0.0
	etaCV := -1.0
	etaN := 0
	etaTrend := 1.0
	var etaRates []float64 // IQR 剔除后的瞬时速率序列，供经验分位数区间使用
	if r, cv, n, trend, rates, ok := calledRatePerMinuteWeighted(recent30obs, now, queueAdvisorWindow30); ok && r > 0 {
		rate = r
		etaCV = cv
		etaN = n
		etaTrend = trend
		etaRates = rates
	}
	// 趋势用更长的 2h 窗单独检测：30min 短窗对全窗渐变不敏感（如午高峰临近，整窗都在缓慢加速，
	// 短窗内近/远差别小），2h 长窗能捕捉到「持续加速/减速」的长期趋势，修正早期高估。
	// 仅当长窗样本充足、且长窗趋势与短窗「同向」（都加速或都减速）且更显著时才采纳长窗。
	// 旧逻辑是无条件「让 trend 离 1 更远」（加速取 max、减速取 min），会把长窗的历史渐变信号
	// 覆盖到短窗反映「现在」的稳定节奏上：午高峰已过、近期已减速，但长窗含爬升段仍显示加速时，
	// 会误判为持续加速、ETA 系统性偏早。同向要求确保只在「长期与近期一致地变快/变慢」时才放大。
	if _, _, _, longTrend, _, ok := calledRatePerMinuteWeighted(history, now, queuePanelRateWindow); ok && len(history) >= 8 {
		if longTrend >= 1.0 && etaTrend >= 1.0 && longTrend > etaTrend {
			etaTrend = longTrend // 同向加速且长窗更显著：采用长窗更强信号
		} else if longTrend < 1.0 && etaTrend < 1.0 && longTrend < etaTrend {
			etaTrend = longTrend // 同向减速且长窗更显著：采用长窗更强信号
		}
	}

	var called15 *int
	if c, ok := calledAdvanceWithin(history, now, queueAdvisorWindow15); ok {
		called15 = &c
	}

	level := queuePressureLevel(snapshot.GroupQueuesCount, store.Wait)
	trend, trendLabel := queuePressureTrend(recent15, rate, stalled)

	advisor := QueueAdvisor{
		StoreID:     snapshot.StoreID,
		StoreName:   store.Name,
		GeneratedAt: now.Format(time.RFC3339),
		Current: QueueAdvisorCurrent{
			CalledNo:            snapshot.DisplayCalledNo,
			WaitingGroups:       snapshot.GroupQueuesCount,
			OfficialWaitMinutes: store.Wait,
			StoreStatus:         store.StoreStatus,
			NetTicketStatus:     store.NetTicketStatus,
			OnlineOpen:          snapshot.OnlineOpen,
		},
		Pressure: QueueAdvisorPressure{
			Level:      level,
			Label:      queuePressureLabel(level),
			Score:      queuePressureScore(snapshot.GroupQueuesCount, store.Wait),
			Trend:      trend,
			TrendLabel: trendLabel,
			Reason:     queuePressureReason(level, trend, snapshot.GroupQueuesCount, rate, called15),
		},
		Speed: QueueAdvisorSpeed{
			CalledPerMin15: calledRateOverWindow(all, snapshot.StoreID, now, queueAdvisorWindow15),
			CalledPerMin30: calledRateOverWindow(all, snapshot.StoreID, now, queueAdvisorWindow30),
			CalledPerMin60: rate60,
		},
		SamplingPoints: len(history),
		Warnings:       warnings,
	}

	if targetNo > 0 {
		advisor.Eta = buildQueueAdvisorEta(targetNo, travelMinutes, snapshot, store, rate, etaCV, etaN, etaTrend, etaRates, now, storeID)
		// 合理性检查：号码小于当前叫号即为过号（寿司郎过号不会自动叫到，需补号或重新取号）。
		// 这里和 computeQueueEta 内部的过号分支互为兜底，任何 targetNo < called 都给出显式警告。
		if called := snapshot.DisplayCalledNo; called > 0 && targetNo < called {
			advisor.Warnings = append(advisor.Warnings,
				fmt.Sprintf("你输入的 %d 号比当前叫到的 %d 号还小，可能已经过号或号码输错了，请到小程序核对后再判断。", targetNo, called))
			if advisor.Eta != nil {
				advisor.Eta.Risk = "high"
			}
		}
	}
	return advisor, nil
}

// buildQueueAdvisorEta 取今日同日型、同门店、当前半小时 bucket 的历史 P50/P80 作为 hist 兜底，
// 再交给 computeQueueEta 做纯函数估算（实时速度+历史融合、CV 动态区间）。
// rates 是 IQR 剔除后的瞬时速率序列，供经验分位数区间使用（nil/不足时回退 CV 查表）。
func buildQueueAdvisorEta(targetNo, travelMinutes int, snapshot QueueObservation, store QueueLiveStore, rate, cv float64, realtimeN int, trend float64, rates []float64, now time.Time, storeID string) *QueueAdvisorEta {
	var hist *QueueWaitRange
	if m, _ := historicalWaitByBucket(storeID, now); m != nil {
		if r, ok := m[queueTrendBucket(now, 30)]; ok {
			hist = &r
		}
	}
	return computeQueueEta(targetNo, snapshot.DisplayCalledNo, travelMinutes, store.Wait, store.WaitTimeCap, rate, cv, realtimeN, trend, rates, hist, now)
}

// computeQueueEta 是 ETA 估算的纯函数核心（不读磁盘），便于单测。
// 入参：targetNo 用户票号、calledNo 当前叫到号、travelMinutes 路程分钟、officialWait 官方等位分钟、
// waitCap 官方等位封顶（store.WaitTimeCap，如 180）、rate 本机叫号速度（组/分）、
// cv 速度变异系数（<0 表示样本不足）、realtimeN 有效叫号间隔数（用于实时/历史融合权重）、
// trend 速度趋势比（>1=加速、<1=减速，用于前瞻校正）、rates IQR 剔除后的瞬时速率序列（经验分位数区间）、
// hist 历史区间、now 当前时间。
func computeQueueEta(targetNo, calledNo, travelMinutes, officialWait, waitCap int, rate, cv float64, realtimeN int, trend float64, rates []float64, hist *QueueWaitRange, now time.Time) *QueueAdvisorEta {
	// remaining 一定钳到 >=0；targetNo<calledNo（过号）也会落在这里，但语义不同，见下分支。
	remaining := max(0, targetNo-calledNo)
	eta := &QueueAdvisorEta{
		TargetNo:        targetNo,
		RemainingGroups: remaining,
		Risk:            "unknown",
	}
	waitRange, source := estimateWaitRange(remaining, officialWait, rate, cv, realtimeN, trend, rates, hist)
	if remaining <= 0 {
		// remaining==0 有两种含义：targetNo==calledNo 是真的即将轮到；
		// targetNo<calledNo 是已经过号——寿司郎过号不会自动叫到，需要补号或重新取号，
		// 必须显式提示，不能再说“已轮到、低风险”（否则与 dashboard 路径自相矛盾）。
		if targetNo < calledNo {
			// 过号分支：等待区间设 0、风险置 high，并给出明确的「请补号」指引，
			// 不走正常 ETA 计算（没有 remaining 可以推时间）。
			eta.WaitMinutesRange = &QueueWaitRange{Low: 0, High: 0}
			eta.Risk = "high"
			eta.ArrivalSuggestion = "这个号可能已经过号了，请到小程序确认是否需要补号或重新取号。"
			return eta
		}
		// 即将轮到分支：targetNo==calledNo，号码正好在叫。
		eta.EstimatedCalledAt = now.Format(time.RFC3339)
		eta.WaitMinutesRange = &QueueWaitRange{Low: 0, High: 0}
		eta.Risk = "low"
		eta.ArrivalSuggestion = "已轮到或即将轮到你，请尽快到店。"
		return eta
	}
	eta.Source = source
	eta.SourceLabel = queueEtaSourceLabel(source)
	eta.SourceNote = queueEtaSourceNote(source)
	if source == "official" {
		// 只有官方等待、没有叫号速度时：无法定位到用户号码的叫到时间，标记 high 风险并提示，
		// 直接返回，不推具体时间区间（estimateWaitRange 虽然给了区间，但基于门店整体，不可靠）。
		eta.Risk = "high"
		eta.ArrivalSuggestion = "当前缺少叫号速度，官方等待只能代表门店排队压力，暂时不能可靠判断你的号码几点叫到。"
		return eta
	}
	if waitRange == nil {
		// 三档全部数据不足（recent/official/history 都拿不到）。
		eta.ArrivalSuggestion = "实时和历史数据都不足，暂时无法预估叫到时间。"
		return eta
	}
	eta.WaitMinutesRange = waitRange
	// 用官方等位封顶值（waitTimeCap，如 180 分钟）做软上限钳制：模型算出的高位若明显
	// 超过官方封顶，多半是异常高峰/叫号停滞，钳到 cap 并提示，避免给用户离谱的预测。
	// 注意只钳 high 不动 low（low 已被 officialWait 抬高过），钳完若 low>high 再把 low 拉回 high。
	note := eta.SourceNote
	if waitCap > 0 && waitRange.High > waitCap {
		capped := &QueueWaitRange{Low: waitRange.Low, High: waitCap}
		if capped.Low > capped.High {
			capped.Low = capped.High
		}
		eta.WaitMinutesRange = capped
		waitRange = capped
		note = "模型估算超过官方等位封顶 " + strconv.Itoa(waitCap) + " 分钟，已按封顶值收敛；可能为异常高峰，建议到小程序确认。"
	}
	// CV 大（叫号忽快忽慢）时追加节奏不稳提示（与封顶提示拼接，不互相覆盖）。
	if cv >= 0.5 {
		volatileNote := "近期叫号节奏不稳，区间已加宽覆盖不确定性。"
		if note == "" {
			note = volatileNote
		} else {
			note = note + " " + volatileNote
		}
	}
	eta.SourceNote = note
	// 时间区间：早=now+low，晚=now+high，中点取 (low+high)/2 作为单点估算。
	early := now.Add(time.Duration(waitRange.Low) * time.Minute)
	late := now.Add(time.Duration(waitRange.High) * time.Minute)
	mid := now.Add(time.Duration((waitRange.Low+waitRange.High)/2) * time.Minute)
	eta.EstimatedCalledAt = mid.Format(time.RFC3339)
	eta.EstimatedCalledAtRange = &QueueTimeRange{Early: early.Format(time.RFC3339), Late: late.Format(time.RFC3339)}
	// 建议出发：按偏早叫到时间倒减路程（travelMinutes 钳到 >=0 防负）。
	depart := early.Add(-time.Duration(max(0, travelMinutes)) * time.Minute)
	if depart.Before(now) {
		eta.ArrivalSuggestion = "建议现在就出发。"
	} else {
		eta.ArrivalSuggestion = fmt.Sprintf("建议 %s 前后出发。", depart.Format("15:04"))
	}
	eta.Risk = queueEtaRisk(source, officialWait, remaining, rate)
	return eta
}

func queueEtaSourceLabel(source string) string {
	switch source {
	case "recent_speed":
		return "近实时叫号速度"
	case "blended":
		return "综合近期与历史"
	case "history":
		return "同门店历史"
	case "official":
		return "官方等待参考"
	default:
		return "数据不足"
	}
}

func queueEtaSourceNote(source string) string {
	switch source {
	case "recent_speed":
		return "按本机近期采样的叫号推进速度估算，适合判断你手里号码的大概叫到时间。"
	case "blended":
		return "综合近期叫号速度与同门店历史规律估算；实时样本尚少，已用历史做参照加权。"
	case "history":
		return "按同门店同日型历史等待估算，适合作参考，建议配合实时叫号一起看。"
	case "official":
		return "官方等待不等同于到你号码的等待时间，只能说明当前门店压力。"
	default:
		return "缺少叫号速度和历史样本。"
	}
}

// queueEtaRisk 按 source + 剩余桌数/官方等待给风险等级。
// recent_speed 用剩余桌数（>200 high、>80 medium，否则 low）；
// blended 复用 recent_speed 规则（含实时成分，按剩余桌数判档）；
// official 用官方等待分钟（>=120 high、>=60 medium）；history 一律 medium（历史不反映今日突变）；
// 都没有则 unknown。
func queueEtaRisk(source string, officialWait, remaining int, rate float64) string {
	switch source {
	case "recent_speed", "blended":
		if remaining > 200 {
			return "high"
		}
		if remaining > 80 {
			return "medium"
		}
		return "low"
	case "official":
		if officialWait >= 120 {
			return "high"
		}
		if officialWait >= 60 {
			return "medium"
		}
		return "low"
	case "history":
		return "medium"
	default:
		return "unknown"
	}
}

// ---------- 压力曲线 ----------

type QueuePressureCurvePoint struct {
	Time                string   `json:"time"`
	CalledNo            int      `json:"called_no"`
	WaitingGroups       int      `json:"waiting_groups"`
	OfficialWaitMinutes int      `json:"official_wait_minutes"`
	PressureLevel       string   `json:"pressure_level"`
	PressureScore       int      `json:"pressure_score"`
	CalledSpeed15       *float64 `json:"called_speed_15,omitempty"`
	Source              string   `json:"source,omitempty"`
	SampleCount         int      `json:"sample_count,omitempty"`
	Confidence          string   `json:"confidence,omitempty"`
}

type QueuePressureCurve struct {
	StoreID      string                    `json:"store_id"`
	Date         string                    `json:"date"`
	DateType     string                    `json:"date_type,omitempty"`
	DateTypeName string                    `json:"date_type_name,omitempty"`
	GeneratedAt  string                    `json:"generated_at"`
	Source       string                    `json:"source,omitempty"`
	LocalPoints  int                       `json:"local_points"`
	RemotePoints int                       `json:"remote_points"`
	Baseline     QueueBaselineRemoteStatus `json:"baseline"`
	Points       []QueuePressureCurvePoint `json:"points"`
	Message      string                    `json:"message,omitempty"`
}

func buildQueuePressureCurve(ctx context.Context, storeID, date string, now time.Time) QueuePressureCurve {
	if ctx == nil {
		ctx = context.Background()
	}
	if now.IsZero() {
		now = time.Now()
	}
	date, day := queuePressureCurveDate(date, now)
	holidays, workdays, _ := loadQueueHolidayDates()
	dateType := queueTrendDateType(day, holidays, workdays)
	out := QueuePressureCurve{
		StoreID:      storeID,
		Date:         date,
		DateType:     dateType,
		DateTypeName: queueTrendDateTypeName(dateType),
		GeneratedAt:  now.Format(time.RFC3339),
	}

	localPoints := buildLocalQueuePressureCurvePoints(storeID, date)
	out.LocalPoints = len(localPoints)

	baseline, baselineStatus, baselineErr := loadRemoteQueuePressureBaseline(ctx, storeID, now)
	out.Baseline = baselineStatus
	remotePoints := buildRemoteQueuePressureCurvePoints(storeID, date, dateType, baseline)
	out.RemotePoints = len(remotePoints)

	// 按本机点数选择数据来源，优先级：本机足够 → 混合 → 仅远端 → 仅本机 → 无数据。
	// queuePressureCurveLocalPreferredPoints（8）是「本机可信」的阈值；不足 8 点时混入远端基准补全，
	// 让新用户第一次打开就能看到曲线，而不是一片空白。
	switch {
	case len(localPoints) >= queuePressureCurveLocalPreferredPoints:
		out.Points = localPoints
		out.Source = "local"
		if len(remotePoints) > 0 {
			out.Message = "当前曲线使用本机实际采样；线上 Turso 基准已连接，本机样本不足时会自动兜底。"
		}
	case len(localPoints) > 0 && len(remotePoints) > 0:
		out.Points = mergeQueuePressureCurvePoints(remotePoints, localPoints)
		out.Source = "mixed"
		out.Message = fmt.Sprintf("本机今天只有 %d 个采样点，已用线上 Turso 基准补全排队压力；带“本机采样”的点按实际数据覆盖，远端基准不等同实时叫号。", len(localPoints))
	case len(remotePoints) > 0:
		out.Points = remotePoints
		out.Source = "remote_baseline"
		out.Message = "本机今天还没有足够采样，当前使用线上 Turso 基准的排队压力；实时叫号判断仍以上方官方当前状态为准。"
	case len(localPoints) > 0:
		out.Points = localPoints
		out.Source = "local"
		if baselineErr != nil {
			out.Message = "线上 Turso 基准暂时不可用，当前只显示本机采样曲线。"
		}
	default:
		out.Source = "none"
		out.Message = "还没有这家店今天的本机采样曲线。开启本机数据收集后会逐步补齐。"
		if baselineErr != nil {
			out.Message += " 线上 Turso 基准暂时不可用。"
		} else if baselineStatus.Used {
			out.Message += " 线上 Turso 基准已连接，但这家店暂时没有可用基准数据。"
		} else if baselineStatus.Configured && !baselineStatus.Used {
			out.Message += " 线上 Turso 基准未返回可用数据。"
		} else if !baselineStatus.Configured {
			out.Message += " 未配置线上 Turso 基准。"
		}
	}
	return out
}

func queuePressureCurveDate(raw string, now time.Time) (string, time.Time) {
	if now.IsZero() {
		now = time.Now()
	}
	// 用寿司郎门店时区（UTC+8）解释日期，不依赖机器本地时区。
	now = now.In(SushiroTimezone)
	loc := SushiroTimezone
	if day, ok := parseTrendDateParam(raw, loc); ok {
		return day.Format("2006-01-02"), day
	}
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	return day.Format("2006-01-02"), day
}

func buildLocalQueuePressureCurvePoints(storeID, date string) []QueuePressureCurvePoint {
	all := loadQueueObservations()
	dayObservations := make([]QueueObservation, 0, len(all))
	for _, o := range all {
		if o.StoreID != storeID {
			continue
		}
		at, ok := parseRFC3339Local(queueObservationCollectedAt(o))
		if !ok || at.Format("2006-01-02") != date {
			continue
		}
		dayObservations = append(dayObservations, o)
	}
	if len(dayObservations) == 0 {
		return nil
	}
	sort.SliceStable(dayObservations, func(i, j int) bool {
		li, _ := parseRFC3339Local(queueObservationCollectedAt(dayObservations[i]))
		lj, _ := parseRFC3339Local(queueObservationCollectedAt(dayObservations[j]))
		return li.Before(lj)
	})
	points := make([]QueuePressureCurvePoint, 0, len(dayObservations))
	for _, o := range dayObservations {
		at, _ := parseRFC3339Local(queueObservationCollectedAt(o))
		level := queuePressureLevel(o.GroupQueuesCount, o.WaitMinutes)
		points = append(points, QueuePressureCurvePoint{
			Time:                at.Format("15:04"),
			CalledNo:            o.DisplayCalledNo,
			WaitingGroups:       o.GroupQueuesCount,
			OfficialWaitMinutes: o.WaitMinutes,
			PressureLevel:       level,
			PressureScore:       queuePressureScore(o.GroupQueuesCount, o.WaitMinutes),
			CalledSpeed15:       calledRateOverWindow(dayObservations, storeID, at, queueAdvisorWindow15),
			Source:              "local",
			SampleCount:         1,
			Confidence:          "live",
		})
	}
	return points
}

// queuePressureCurveRemoteAcc 把同一时段(bucket)的多条远端 rollup 按各自样本数加权累加，
// 最后除以总权重得到加权均值。不同 rollup 的 SampleCount 可能差很多，等权平均会偏向小样本。
type queuePressureCurveRemoteAcc struct {
	samples   int
	calledSum float64
	calledN   int
	groupsSum float64
	groupsN   int
	waitSum   float64
	waitN     int
}

// buildRemoteQueuePressureCurvePoints 把远端基准的 rollups（聚合后典型值）和 latest（当日实时快照）
// 转成压力曲线点。rollup 按 SampleCount 加权合并到同一 bucket；latest 作为单独的高可信点追加。
// 这样「远端基准」反映长期典型压力、「当日实时」反映今日突变，前端按 source rank 取最新覆盖。
func buildRemoteQueuePressureCurvePoints(storeID, date, dateType string, baseline QueueBaselineExport) []QueuePressureCurvePoint {
	storeID = strings.TrimSpace(storeID)
	if storeID == "" || len(baseline.Rollups) == 0 && len(baseline.Latest) == 0 {
		return nil
	}
	byBucket := map[string]*queuePressureCurveRemoteAcc{}
	for _, rollup := range baseline.Rollups {
		if strconv.Itoa(rollup.StoreID) != storeID {
			continue
		}
		if !queueDashboardRollupDateTypeAllowed(rollup.DateType, dateType, false) || !queueDashboardCalledBucketInRange(rollup.TimeBucket) {
			continue
		}
		if rollup.CalledNoTypical == nil && rollup.QueueGroupsTypical == nil && rollup.WaitTypicalMinutes == nil {
			continue
		}
		acc := byBucket[rollup.TimeBucket]
		if acc == nil {
			acc = &queuePressureCurveRemoteAcc{}
			byBucket[rollup.TimeBucket] = acc
		}
		acc.samples += maxInt(rollup.SampleCount, 1)
		if rollup.CalledNoTypical != nil {
			weight := maxInt(rollup.CalledSampleCount, 1)
			acc.calledSum += *rollup.CalledNoTypical * float64(weight)
			acc.calledN += weight
		}
		if rollup.QueueGroupsTypical != nil {
			weight := maxInt(rollup.SampleCount, 1)
			acc.groupsSum += *rollup.QueueGroupsTypical * float64(weight)
			acc.groupsN += weight
		}
		if rollup.WaitTypicalMinutes != nil {
			weight := maxInt(rollup.SampleCount, 1)
			acc.waitSum += *rollup.WaitTypicalMinutes * float64(weight)
			acc.waitN += weight
		}
	}

	buckets := make([]string, 0, len(byBucket))
	for bucket := range byBucket {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)
	points := make([]QueuePressureCurvePoint, 0, len(buckets)+1)
	for _, bucket := range buckets {
		acc := byBucket[bucket]
		point := QueuePressureCurvePoint{
			Time:        bucket,
			Source:      "remote_baseline",
			SampleCount: acc.samples,
			Confidence:  queueDashboardConfidence(acc.samples),
		}
		if acc.calledN > 0 {
			point.CalledNo = int(math.Round(acc.calledSum / float64(acc.calledN)))
		}
		if acc.groupsN > 0 {
			point.WaitingGroups = int(math.Round(acc.groupsSum / float64(acc.groupsN)))
		}
		if acc.waitN > 0 {
			point.OfficialWaitMinutes = int(math.Round(acc.waitSum / float64(acc.waitN)))
		}
		point.PressureLevel = queuePressureLevel(point.WaitingGroups, point.OfficialWaitMinutes)
		point.PressureScore = queuePressureScore(point.WaitingGroups, point.OfficialWaitMinutes)
		points = append(points, point)
	}

	for _, latest := range baseline.Latest {
		if strconv.Itoa(latest.StoreID) != storeID {
			continue
		}
		at, ok := parseRFC3339Local(latest.CollectedAt)
		if !ok || at.Format("2006-01-02") != date || !queueDashboardCalledTimeInRange(at) {
			continue
		}
		level := queuePressureLevel(latest.GroupQueuesCount, latest.WaitMinutes)
		points = append(points, QueuePressureCurvePoint{
			Time:                at.Format("15:04"),
			CalledNo:            latest.DisplayCalledNo,
			WaitingGroups:       latest.GroupQueuesCount,
			OfficialWaitMinutes: latest.WaitMinutes,
			PressureLevel:       level,
			PressureScore:       queuePressureScore(latest.GroupQueuesCount, latest.WaitMinutes),
			Source:              "remote_latest",
			SampleCount:         1,
			Confidence:          "live",
		})
	}
	return sortAndDedupQueuePressureCurvePoints(points)
}

func mergeQueuePressureCurvePoints(remotePoints, localPoints []QueuePressureCurvePoint) []QueuePressureCurvePoint {
	points := make([]QueuePressureCurvePoint, 0, len(remotePoints)+len(localPoints))
	points = append(points, remotePoints...)
	points = append(points, localPoints...)
	return sortAndDedupQueuePressureCurvePoints(points)
}

// sortAndDedupQueuePressureCurvePoints 按时间升序排，同一时刻只保留可信度最高的点。
// 同 minute 的点按 source rank 降序排（local>remote_latest>remote_baseline），稳定排序后第一个即为最高优先，
// 再用 seen map 按时间键去重，保证每个时刻只剩一个点，避免混排时双点叠加。
func sortAndDedupQueuePressureCurvePoints(points []QueuePressureCurvePoint) []QueuePressureCurvePoint {
	sort.SliceStable(points, func(i, j int) bool {
		mi := queueDashboardBucketMinute(points[i].Time)
		mj := queueDashboardBucketMinute(points[j].Time)
		if mi != mj {
			return mi < mj
		}
		return queuePressureCurveSourceRank(points[i].Source) > queuePressureCurveSourceRank(points[j].Source)
	})
	out := make([]QueuePressureCurvePoint, 0, len(points))
	seen := map[string]bool{}
	for _, point := range points {
		minute := queueDashboardBucketMinute(point.Time)
		if minute < 0 {
			continue
		}
		key := point.Time
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, point)
	}
	return out
}

// queuePressureCurveSourceRank 给数据来源打可信度分：local(本机实际采样) 最高、
// remote_latest(当日远端实时) 次之、remote_baseline(长期聚合) 最低。用于同时刻去重时保留更可信的点。
func queuePressureCurveSourceRank(source string) int {
	switch source {
	case "local":
		return 3
	case "remote_latest":
		return 2
	case "remote_baseline":
		return 1
	default:
		return 0
	}
}

// ---------- 时间互推：取号 <-> 就餐 ----------

// historicalWaitByBucket 取某店今日日期类型的历史等待 P50/P80（按半小时 bucket）。
func historicalWaitByBucket(storeID string, now time.Time) (map[string]QueueWaitRange, string) {
	dt := queueAdvisorDateTypeForToday(now)
	resp := BuildQueueTrends(QueueTrendQuery{StoreIDs: []string{storeID}, DateType: dt, BucketMinutes: 30}, now)
	out := map[string]QueueWaitRange{}
	for _, p := range resp.Series {
		if p.StoreID != storeID || p.WaitP50Minutes == nil {
			continue
		}
		low := int(math.Round(*p.WaitP50Minutes))
		high := low
		if p.WaitP80Minutes != nil {
			high = int(math.Round(*p.WaitP80Minutes))
		}
		if high < low {
			high = low
		}
		out[p.Bucket] = QueueWaitRange{Low: low, High: high}
	}
	return out, queueTrendDateTypeName(dt)
}

func queueAdvisorDateTypeForToday(now time.Time) string {
	holidays, workdays, _ := loadQueueHolidayDates()
	return queueTrendDateType(now, holidays, workdays)
}

type QueuePickupPlan struct {
	StoreID          string          `json:"store_id"`
	Pickup           string          `json:"pickup"`
	WaitMinutesRange *QueueWaitRange `json:"wait_minutes_range,omitempty"`
	MealRange        *QueueTimeRange `json:"meal_range,omitempty"`
	Risk             string          `json:"risk"`
	Basis            string          `json:"basis"`
	Message          string          `json:"message,omitempty"`
}

type QueueMealPlan struct {
	StoreID              string          `json:"store_id"`
	TargetMeal           string          `json:"target_meal"`
	RecommendPickupRange *QueueTimeRange `json:"recommend_pickup_range,omitempty"`
	StablePickup         string          `json:"stable_pickup,omitempty"`
	LatestPickup         string          `json:"latest_pickup,omitempty"`
	WaitMinutesRange     *QueueWaitRange `json:"wait_minutes_range,omitempty"`
	Risk                 string          `json:"risk"`
	Basis                string          `json:"basis"`
	Message              string          `json:"message,omitempty"`
}

// parseHHMM 把 "1210"/"12:10" 解析为今天对应的本地时间。
func parseHHMM(raw string, now time.Time) (time.Time, bool) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, ":", ""))
	if len(raw) != 4 {
		return time.Time{}, false
	}
	hh, ok1 := atoiStrict(raw[:2])
	mm, ok2 := atoiStrict(raw[2:])
	if !ok1 || !ok2 || hh > 23 || mm > 59 {
		return time.Time{}, false
	}
	// 用门店时区构造时刻，避免机器本地时区偏移导致叫号时间范围判定错位。
	now = now.In(SushiroTimezone)
	return time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, SushiroTimezone), true
}

func atoiStrict(s string) (int, bool) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

// liveWaitEstimate 用公开实时接口取“现在等多久”的粗估区间，作为没有历史样本时的兜底，
// 让只读用户第一次打开就能得到答案。返回 false 表示连实时数据也拿不到。
func liveWaitEstimate(ctx context.Context, storeID string, now time.Time) (QueueWaitRange, bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	panel, err := buildQueueLivePanel(timeoutCtx, storeID, now)
	if err != nil {
		return QueueWaitRange{}, false
	}
	// 等待分钟优先级：ServerWaitMin(官方) > EtaMinutes(接口算的) > WaitGroups*2(粗估)。
	// 每组约 2 分钟是经验值，只在接口完全不给等待分钟时兜底。
	wait := panel.ServerWaitMin
	if panel.EtaMinutes != nil && *panel.EtaMinutes > 0 {
		wait = *panel.EtaMinutes
	}
	if wait <= 0 && panel.WaitGroups > 0 {
		wait = panel.WaitGroups * 2 // 接口没给等待分钟时按每组约 2 分钟粗估
	}
	if wait <= 0 {
		// 连粗估都没有：营业且无排队就给一个 0-10 的低区间，否则放弃（返回 false）。
		if panel.OnlineOpen && panel.WaitGroups == 0 {
			return QueueWaitRange{Low: 0, High: 10}, true
		}
		return QueueWaitRange{}, false
	}
	// 上界 = wait*1.5，但至少 wait+10，保证 low/high 之间有足够缓冲表达不确定性。
	high := wait + wait/2
	if high < wait+10 {
		high = wait + 10
	}
	return QueueWaitRange{Low: wait, High: high}, true
}

const queuePlanLiveBasis = "按当前实时等待粗估（这家店还没有历史样本）"
const queuePlanLiveHint = "先按此刻的实时等待粗估；开启「预测准确度」积累几天后，会自动换成更准的历史曲线。"

func buildQueuePickupPlan(ctx context.Context, storeID, pickupRaw string, now time.Time) QueuePickupPlan {
	if now.IsZero() {
		now = time.Now()
	}
	out := QueuePickupPlan{StoreID: storeID, Pickup: compactTrendTime(pickupRaw)}
	pickup, ok := parseHHMM(pickupRaw, now)
	if !ok {
		out.Message = "请提供有效的取号时间，例如 1210 或 12:10。"
		return out
	}
	out.Pickup = pickup.Format("15:04")
	hist, basis := historicalWaitByBucket(storeID, now)
	wr, found := hist[queueTrendBucket(pickup, 30)]
	if !found {
		if lw, ok := liveWaitEstimate(ctx, storeID, now); ok {
			out.WaitMinutesRange = &lw
			out.MealRange = &QueueTimeRange{
				Early: pickup.Add(time.Duration(lw.Low) * time.Minute).Format("15:04"),
				Late:  pickup.Add(time.Duration(lw.High) * time.Minute).Format("15:04"),
			}
			out.Basis = queuePlanLiveBasis
			out.Risk = queuePlanRisk(storeID, now, lw)
			out.Message = queuePlanLiveHint
			return out
		}
		out.Message = "这家店该时段的历史样本不足，先开启本机数据收集积累几次。"
		out.Basis = basis
		return out
	}
	out.WaitMinutesRange = &wr
	out.MealRange = &QueueTimeRange{
		Early: pickup.Add(time.Duration(wr.Low) * time.Minute).Format("15:04"),
		Late:  pickup.Add(time.Duration(wr.High) * time.Minute).Format("15:04"),
	}
	out.Basis = "同门店同日型历史等待；今日实时压力仅修正风险"
	out.Risk = queuePlanRisk(storeID, now, wr)
	return out
}

// buildQueueMealPlan 计算“想几点吃→几点取号”。allowLiveFallback 为 true 时（Web 只读查询），
// 没有历史样本会退回当前实时等待粗估；Routine 提醒必须传 false，维持“样本不足不乱提醒”的承诺。
func buildQueueMealPlan(ctx context.Context, storeID, mealRaw string, travelMinutes int, now time.Time, allowLiveFallback bool) QueueMealPlan {
	if now.IsZero() {
		now = time.Now()
	}
	out := QueueMealPlan{StoreID: storeID, TargetMeal: compactTrendTime(mealRaw)}
	meal, ok := parseHHMM(mealRaw, now)
	if !ok {
		out.Message = "请提供有效的目标就餐时间，例如 1300 或 13:00。"
		return out
	}
	out.TargetMeal = meal.Format("15:04")
	target := meal.Add(-time.Duration(max(0, travelMinutes)) * time.Minute)
	hist, _ := historicalWaitByBucket(storeID, now)
	if len(hist) == 0 {
		if allowLiveFallback {
			if lw, ok := liveWaitEstimate(ctx, storeID, now); ok {
				stable := target.Add(-time.Duration(lw.High) * time.Minute)
				latest := target.Add(-time.Duration(lw.Low) * time.Minute)
				out.StablePickup = stable.Format("15:04")
				out.LatestPickup = latest.Format("15:04")
				out.RecommendPickupRange = &QueueTimeRange{Early: stable.Format("15:04"), Late: latest.Format("15:04")}
				out.WaitMinutesRange = &lw
				out.Basis = queuePlanLiveBasis
				out.Risk = queuePlanRisk(storeID, now, lw)
				out.Message = queuePlanLiveHint
				return out
			}
		}
		out.Message = "这家店历史样本不足，先开启本机数据收集积累几次。"
		return out
	}
	// 枚举候选取号时间（每 10 分钟），预测就餐时间，挑最接近目标的窗口。
	var best, latest time.Time
	var bestWR QueueWaitRange
	bestGap := math.MaxFloat64
	// 从门店当地 10:00 起按 10 分钟步进找最接近 meal 的历史桶，时区必须用门店时区。
	nowS := now.In(SushiroTimezone)
	dayStart := time.Date(nowS.Year(), nowS.Month(), nowS.Day(), 10, 0, 0, 0, SushiroTimezone)
	for t := dayStart; !t.After(meal); t = t.Add(10 * time.Minute) {
		wr, found := hist[queueTrendBucket(t, 30)]
		if !found {
			continue
		}
		predEarly := t.Add(time.Duration(wr.Low) * time.Minute)
		predLate := t.Add(time.Duration(wr.High) * time.Minute)
		// 推荐取号窗口：偏稳就餐（P80）不晚于目标。
		if !predLate.After(target) {
			latest = t
		}
		gap := math.Abs(predEarly.Sub(target).Minutes())
		if gap < bestGap {
			bestGap = gap
			best = t
			bestWR = wr
		}
	}
	if best.IsZero() {
		out.Message = "按历史样本，今天恐怕赶不上这个就餐时间，建议提早取号或换时段。"
		return out
	}
	out.StablePickup = best.Format("15:04")
	out.WaitMinutesRange = &bestWR
	// 推荐取号区间：best（最接近目标就餐时间的取号点）与 latest（不晚于目标的稳妥最晚取号点）
	// 是独立选取的，best 可能晚于 latest，导致 Early>Late 倒序（前端显示成 12:30-12:00）。
	// 这里校正为早→晚。
	early, latePick := best, best
	if !latest.IsZero() {
		out.LatestPickup = latest.Format("15:04")
		latePick = latest
	}
	if latePick.Before(early) {
		early, latePick = latePick, early
	}
	out.RecommendPickupRange = &QueueTimeRange{Early: early.Format("15:04"), Late: latePick.Format("15:04")}
	out.Basis = "同门店同日型历史等待；今日实时压力仅修正风险"
	out.Risk = queuePlanRisk(storeID, now, bestWR)
	return out
}

// queuePlanRisk 用今日实时排队压力修正历史规划的风险。
func queuePlanRisk(storeID string, now time.Time, wr QueueWaitRange) string {
	all := loadQueueObservations()
	latest := latestQueueObservationsByStore(all)[storeID]
	level := queuePressureLevel(latest.GroupQueuesCount, latest.WaitMinutes)
	switch level {
	case "extreme":
		return "high"
	case "high":
		if wr.High >= 90 {
			return "high"
		}
		return "medium"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		if wr.High >= 90 {
			return "medium"
		}
		return "low"
	}
}
