package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"net/http"
	"strings"
	"time"
)

type calendarStoreResult struct {
	StoreID   string `json:"store_id"`
	StoreName string `json:"store_name"`
	Slots     []Slot `json:"slots"`
	Error     string `json:"error,omitempty"`
}

func handleCalendar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	ws := getWebSettings()
	if len(ws.StoreIDs) == 0 {
		writeError(w, http.StatusServiceUnavailable, "暂无配置，请先完成参数捕获")
		return
	}

	client := getWebClient()
	query := r.URL.Query()
	storeIDs := requestedCalendarStores(query["store"], query.Get("stores"), ws.StoreIDs)
	onlyAvailable := query.Get("available") == "1" || strings.EqualFold(query.Get("available"), "true")
	period := strings.ToLower(strings.TrimSpace(query.Get("period")))

	reg := GetStoreRegistry()
	results := make([]calendarStoreResult, 0, len(storeIDs))
	for _, storeID := range storeIDs {
		slots, err := client.GetTimeslots(r.Context(), storeID)
		storeInfo, _ := client.GetStoreInfo(r.Context(), storeID)
		displayName := reg.DisplayName(storeID, storeInfo.Name)
		if displayName == "" {
			displayName = storeID
		}
		result := calendarStoreResult{StoreID: storeID, StoreName: displayName}
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}
		appendHistory(slots, storeID)
		result.Slots = filterCalendarSlots(slots, onlyAvailable, period)
		results = append(results, result)
	}

	result := map[string]any{
		"store_id":   results[0].StoreID,
		"store_name": results[0].StoreName,
		"slots":      results[0].Slots,
		"stores":     results,
		"fetched_at": time.Now().Format(time.RFC3339),
		"filters": map[string]any{
			"available": onlyAvailable,
			"period":    period,
		},
	}
	writeJSON(w, result)
	bus.publish("calendar", mustJSON(result))
}

func handleStores(w http.ResponseWriter, r *http.Request) {
	ws := getWebSettings()
	if len(ws.StoreIDs) == 0 {
		writeJSON(w, []map[string]string{})
		return
	}
	client := getWebClient()
	reg := GetStoreRegistry()
	stores := make([]map[string]string, 0)
	offline := false

	for _, id := range ws.StoreIDs {
		name := id
		address := ""
		if client != nil {
			if info, err := client.GetStoreInfo(r.Context(), id); err == nil {
				name = info.Name
				address = info.Address
			}
		}
		if name == id { // 凭证缺失或未命中时，用公开接口兜底解析门店名
			if s, err := fetchQueueLiveStore(r.Context(), id); err == nil && s.Name != "" {
				name = s.Name
				address = s.Address
			} else if s, ok := offlineStoreByID(id); ok {
				// 实时也读不到时用离线目录（内置历史 + 本机记录）里的店名。
				name = s.Name
				offline = true
			}
		}
		stores = append(stores, map[string]string{
			"id":       id,
			"name":     name,
			"nickname": reg.DisplayName(id, name),
			"address":  address,
		})
	}
	if offline {
		// 响应仍是数组，离线兜底用响应头标明，原有调用方不受影响。
		w.Header().Set("X-Sushiro-Offline", "1")
	}
	writeJSON(w, stores)
}

func requestedCalendarStores(storeValues []string, storesValue string, allowed []string) []string {
	allowedSet := map[string]bool{}
	for _, storeID := range allowed {
		allowedSet[storeID] = true
	}
	var raw []string
	raw = append(raw, storeValues...)
	raw = append(raw, strings.Split(storesValue, ",")...)
	out := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] || !allowedSet[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	if len(out) == 0 && len(allowed) > 0 {
		out = append(out, allowed[0])
	}
	return out
}

func filterCalendarSlots(slots []Slot, onlyAvailable bool, period string) []Slot {
	out := make([]Slot, 0, len(slots))
	for _, slot := range slots {
		if onlyAvailable && strings.ToUpper(slot.Availability) != "AVAILABLE" {
			continue
		}
		if !calendarSlotMatchesPeriod(slot, period) {
			continue
		}
		out = append(out, slot)
	}
	return out
}

func calendarSlotMatchesPeriod(slot Slot, period string) bool {
	switch period {
	case "", "all":
		return true
	case "lunch":
		seconds := ParseTimeSeconds(slot.Start)
		return seconds >= 10*3600 && seconds < 16*3600
	case "dinner":
		return ParseTimeSeconds(slot.Start) >= 16*3600
	default:
		return true
	}
}
