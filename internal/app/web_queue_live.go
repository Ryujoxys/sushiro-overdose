package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func handleQueueLiveStores(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	query := queueLiveStoreQueryFromRequest(r)
	stores, err := NewQueueLiveClient().ListStores(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusBadGateway, friendlyNetworkError(err))
		return
	}
	writeJSON(w, map[string]any{
		"stores": stores,
		"count":  len(stores),
	})
}

func handleQueueLiveStore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	storeID := strings.TrimSpace(r.URL.Query().Get("id"))
	if storeID == "" {
		storeID = strings.TrimSpace(r.URL.Query().Get("store"))
	}
	if storeID == "" {
		writeError(w, http.StatusBadRequest, "缺少门店 ID")
		return
	}
	store, err := NewQueueLiveClient().GetStore(r.Context(), storeID)
	if err != nil {
		writeError(w, http.StatusBadGateway, friendlyNetworkError(err))
		return
	}
	writeJSON(w, store)
}

func handleQueueLivePanel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	storeID := strings.TrimSpace(r.URL.Query().Get("store"))
	if storeID == "" {
		storeID = strings.TrimSpace(r.URL.Query().Get("id"))
	}
	if storeID == "" {
		writeError(w, http.StatusBadRequest, "缺少门店 ID")
		return
	}
	panel, err := buildQueueLivePanel(r.Context(), storeID, time.Now())
	if err != nil {
		writeError(w, http.StatusBadGateway, friendlyNetworkError(err))
		return
	}
	writeJSON(w, panel)
}

func handleQueueAdvisor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	q := r.URL.Query()
	storeID := strings.TrimSpace(q.Get("store"))
	if storeID == "" {
		storeID = strings.TrimSpace(q.Get("id"))
	}
	if storeID == "" {
		writeError(w, http.StatusBadRequest, "缺少门店 ID")
		return
	}
	advisor, err := buildQueueAdvisor(r.Context(), storeID, atoiDefault(q.Get("target_no"), 0), atoiDefault(q.Get("travel_minutes"), 0), time.Now())
	if err != nil {
		writeError(w, http.StatusBadGateway, friendlyNetworkError(err))
		return
	}
	writeJSON(w, advisor)
}

// handleQueueAccuracy 返回基于实测回测的「预测 vs 实际」精度报告（按店）。
func handleQueueAccuracy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	writeJSON(w, getEtaAccuracyReport())
}

func handleQueuePressureCurve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	q := r.URL.Query()
	storeID := strings.TrimSpace(q.Get("store"))
	if storeID == "" {
		storeID = strings.TrimSpace(q.Get("id"))
	}
	if storeID == "" {
		writeError(w, http.StatusBadRequest, "缺少门店 ID")
		return
	}
	writeJSON(w, buildQueuePressureCurve(r.Context(), storeID, strings.TrimSpace(q.Get("date")), time.Now()))
}

// handleQueuePlan 时间互推：?pickup=HHMM 算几点能吃；?target_meal=HHMM 算几点取号。
func handleQueuePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	q := r.URL.Query()
	storeID := strings.TrimSpace(q.Get("store"))
	if storeID == "" {
		storeID = strings.TrimSpace(q.Get("id"))
	}
	if storeID == "" {
		writeError(w, http.StatusBadRequest, "缺少门店 ID")
		return
	}
	now := time.Now()
	if pickup := strings.TrimSpace(q.Get("pickup")); pickup != "" {
		writeJSON(w, buildQueuePickupPlan(r.Context(), storeID, pickup, now))
		return
	}
	if meal := strings.TrimSpace(q.Get("target_meal")); meal != "" {
		writeJSON(w, buildQueueMealPlan(r.Context(), storeID, meal, 0, now, false))
		return
	}
	writeError(w, http.StatusBadRequest, "缺少 pickup 或 target_meal 参数")
}

func handleQueueAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, LoadQueueAlertConfig())
	case http.MethodPost:
		var cfg QueueAlertConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		cfg = normalizeQueueAlertConfig(cfg)
		if err := SaveQueueAlertConfig(cfg); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, cfg)
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST only")
	}
}

func handleQueueAlertStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	writeJSON(w, BuildQueueAlertStatus(time.Now()))
}

func handleQueueLiveAreas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	areas, err := NewQueueLiveClient().ListAreas(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, friendlyNetworkError(err))
		return
	}
	writeJSON(w, map[string]any{
		"areas": areas,
		"count": len(areas),
	})
}

func queueLiveStoreQueryFromRequest(r *http.Request) QueueLiveStoreQuery {
	q := r.URL.Query()
	return QueueLiveStoreQuery{
		StoreIDs:    queueTrendRequestStores(q["store"], q.Get("stores")),
		City:        strings.TrimSpace(q.Get("city")),
		Area:        strings.TrimSpace(q.Get("area")),
		Keyword:     strings.TrimSpace(q.Get("q")),
		Near:        strings.TrimSpace(q.Get("near")),
		OpenOnly:    truthyQuery(q.Get("open")),
		WaitingOnly: truthyQuery(q.Get("waiting")),
		Limit:       atoiDefault(q.Get("limit"), 0),
	}
}

func truthyQuery(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
