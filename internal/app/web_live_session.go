package app

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// 实时数据连接的三个接口。响应只有状态和剩余时间，从不包含请求头的值。

const liveSessionImportMaxBytes = 16 << 20 // HAR 文件可能较大

type liveSessionImportResponse struct {
	liveSessionStatus
	Saved   bool   `json:"saved"`
	Message string `json:"message"`
}

// GET /api/live-session
func handleLiveSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, currentLiveSessionStatus(time.Now()))
}

// POST /api/live-session/import {text}
func handleLiveSessionImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, liveSessionImportMaxBytes)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "没读到粘贴的内容，请重新粘贴一次")
		return
	}
	now := time.Now()
	session, err := ParseLiveSessionText(body.Text, now)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	saved, err := SaveLiveSession(session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "连接没有保存成功，请重试")
		return
	}
	stored, _ := LoadLiveSession()
	response := liveSessionImportResponse{liveSessionStatus: liveSessionStatusAt(stored, now), Saved: saved, Message: "实时数据已连接"}
	if !response.Connected { // 刚好在保存时到期，或这条连接已被寿司郎拒绝过
		writeError(w, http.StatusBadRequest, errLiveImportExpired.Error())
		return
	}
	if !saved {
		response.Message = "已经连接着一条有效期更长的连接，继续使用它"
		if stored != nil && stored.Authorization == session.Authorization {
			response.Message = "这条连接已经在用了"
		}
	}
	writeJSON(w, response)
}

// POST /api/live-session/clear
func handleLiveSessionClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if err := ClearLiveSession(); err != nil {
		writeError(w, http.StatusInternalServerError, "没能断开连接，请重试")
		return
	}
	writeJSON(w, currentLiveSessionStatus(time.Now()))
}
