package main

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
		writeError(w, http.StatusBadGateway, err.Error())
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
		writeError(w, http.StatusBadGateway, err.Error())
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
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, panel)
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
		if err := SaveQueueAlertConfig(cfg); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, cfg)
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST only")
	}
}

func handleQueueLiveAreas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	areas, err := NewQueueLiveClient().ListAreas(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
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
