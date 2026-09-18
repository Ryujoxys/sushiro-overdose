package app

import "net/http"

const retiredAutomationMessage = "自动抢号已移除。请在界面手动确认取号；日常记录请使用 collect。"

func registerRetiredAutomationRoutes(mux *http.ServeMux) {
	for _, path := range []string{"/api/sniper/", "/api/sniper", "/api/engine/booking", "/api/queue/ticket/plan", "/api/queue/ticket/routine"} {
		mux.HandleFunc(path, handleRetiredAutomation)
	}
}

func handleRetiredAutomation(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusGone, retiredAutomationMessage)
}
