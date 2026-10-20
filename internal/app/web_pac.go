package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/proxy"
import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	pacLogMu   sync.Mutex
	pacLogSeen = map[string]struct{}{}
)

func handleProxyPAC(w http.ResponseWriter, r *http.Request) {
	port, err := strconv.Atoi(r.URL.Query().Get("proxy"))
	if err != nil || port <= 0 || port > 65535 {
		http.Error(w, "invalid proxy port", http.StatusBadRequest)
		return
	}
	logProxyPACRequest(r, port)
	w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintf(w, `function FindProxyForURL(url, host) {
  host = String(host || "").toLowerCase().replace(/\.$/, "");
  if (host.charAt(0) !== "[" && host.indexOf(":") >= 0) {
    host = host.split(":")[0];
  }
  if (%s) {
    return "PROXY 127.0.0.1:%d";
  }
  return "DIRECT";
}
`, proxyPACHostCondition(), port)
}

// proxyPACHostCondition 只让会被解密的两个精确主机走本机代理，其余（含 sushiro.com.cn 其它子域）直连。
func proxyPACHostCondition() string {
	hosts := SushiroInterceptHosts()
	parts := make([]string, 0, len(hosts))
	for _, host := range hosts {
		parts = append(parts, strconv.Quote(host))
	}
	return "host === " + strings.Join(parts, " || host === ")
}

func logProxyPACRequest(r *http.Request, port int) {
	remote := strings.TrimSpace(r.RemoteAddr)
	ua := SanitizeDiagnosticLine(strings.TrimSpace(r.UserAgent()))
	if ua == "" {
		ua = "-"
	}
	key := remote + "|" + ua + "|" + strconv.Itoa(port)
	pacLogMu.Lock()
	if _, ok := pacLogSeen[key]; ok {
		pacLogMu.Unlock()
		return
	}
	pacLogSeen[key] = struct{}{}
	pacLogMu.Unlock()
	LogMessage(time.Now(), fmt.Sprintf("Windows PAC 已被读取: proxy=127.0.0.1:%d remote=%s ua=%s", port, DefaultString(remote, "-"), ua))
}
