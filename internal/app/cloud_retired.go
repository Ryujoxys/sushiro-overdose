package app

import "net/http"

// Keep old bookmarks and callbacks from falling through to the SPA.
func registerRetiredCloudRoutes(mux *http.ServeMux) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		writeError(w, http.StatusGone, "云端数据和 GitHub 登录已下线，请使用本机数据。")
	}
	mux.HandleFunc("/api/cloud", handler)
	mux.HandleFunc("/api/cloud/", handler)
}
