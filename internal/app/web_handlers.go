package app

import (
	"encoding/json"
	"net/http"
	"strings"
)

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// no-store：页面里嵌着本次进程的 CSRF token，绝不能被缓存复用——
	// 否则浏览器展示旧 token 会导致后续写请求 CSRF 校验失败。
	w.Header().Set("Cache-Control", "no-store")
	// 把当前进程的 CSRF token 注入到首页 HTML 模板占位符里，前端 JS 取它放进写请求的 X-Sushiro-CSRF 头。
	w.Write([]byte(strings.Replace(indexHTML, "{{CSRF_TOKEN}}", getWebCSRFToken(), 1)))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeJSONStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// writeErrorCode 在中文错误之外附带机器可读的 code，页面据此分支，不去匹配中文文案。
func writeErrorCode(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg, "code": code})
}

func mustJSON(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}
