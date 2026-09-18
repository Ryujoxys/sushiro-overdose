package core

import (
	"fmt"
	"net"
	"strconv"
)

// ListenOnAvailableLocalPort 在 127.0.0.1 上从 preferred 端口开始顺序探测，
// 返回第一个能成功 listen 的端口（同时返回已建立的 listener，调用方可直接用）。
// 调用方应直接使用返回的 listener，不要关闭后重新 bind；持有监听器可避免端口抢占竞态。
func ListenOnAvailableLocalPort(preferred, attempts int) (net.Listener, int, error) {
	return ListenOnAvailableHostPort("127.0.0.1", preferred, attempts)
}

// ListenOnAvailableHostPort 是 ListenOnAvailableLocalPort 的多 host 版本，host 为空时回退到 127.0.0.1。
// attempts 是探测窗口大小：从 preferred 起连续试 attempts 个端口（含 preferred）。
func ListenOnAvailableHostPort(host string, preferred, attempts int) (net.Listener, int, error) {
	var lastErr error
	for port := preferred; port < preferred+attempts; port++ {
		ln, err := listenHostPort(host, port)
		if err == nil {
			return ln, port, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		// 理论上 attempts<=0 时一次都没试，给个明确的错误而不是静默成功。
		lastErr = fmt.Errorf("no ports checked")
	}
	return nil, 0, lastErr
}

// FirstAvailableLocalPort 只探测端口号、不保留 listener：拿到可用端口后立即 Close。
// 用于「我只想知道哪个端口空着，等会儿再 bind」的场景。
// 注意这有 TOCTOU 竞态：close 之后到真正 bind 之间端口可能被占，调用方需容忍 bind 失败重试。
func FirstAvailableLocalPort(preferred, attempts int) (int, bool) {
	ln, port, err := ListenOnAvailableLocalPort(preferred, attempts)
	if err != nil {
		return 0, false
	}
	_ = ln.Close()
	return port, true
}

// listenLocalPort 在 127.0.0.1 上尝试 listen 单个端口（未导出，仅本包内部用）。
func listenLocalPort(port int) (net.Listener, error) {
	return listenHostPort("127.0.0.1", port)
}

// listenHostPort 在指定 host:port 上 listen。host 为空时默认 127.0.0.1（只绑本地，避免暴露到外网）。
func listenHostPort(host string, port int) (net.Listener, error) {
	if host == "" {
		host = "127.0.0.1"
	}
	return net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
}
