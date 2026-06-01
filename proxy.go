package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const proxyPort = 8080
const proxyPortSearchLimit = 100
const sushiroHost = "crm-cn-prd.sushiro.com.cn"
const proxyErrorBodyLogLimit = 4096
const proxyTunnelDialTimeout = 3 * time.Second
const maxSushiroBufferedResponseBytes = 8 << 20

func (t *CapturedTokens) captureFromRequest(req *http.Request, bodyBytes []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	host := req.URL.Host
	if host == "" {
		host = req.Host
	}
	if !isSushiroTargetHost(host) {
		return
	}

	if v := req.Header.Get("X-App-Code"); v != "" && t.XAppCode == "" {
		t.XAppCode = v
	}
	if v := req.Header.Get("User-Agent"); v != "" && t.UserAgent == "" {
		t.UserAgent = v
	}
	if v := req.Header.Get("Referer"); v != "" && t.Referer == "" {
		t.Referer = v
	}
	if v := req.Header.Get("X-App-Client"); v != "" && t.XAppClient == "" {
		t.XAppClient = v
	}

	authHeader := req.Header.Get("Authorization")
	if authHeader != "" {
		path := req.URL.Path
		if strings.Contains(path, "/api_auth/") || strings.Contains(path, "createReservation") {
			if t.ReservationAuth == "" {
				t.ReservationAuth = authHeader
			}
		} else {
			if t.QueryAuth == "" {
				t.QueryAuth = authHeader
			}
		}
	}

	if sid := req.URL.Query().Get("storeId"); sid != "" {
		found := false
		for _, existing := range t.StoreIDs {
			if existing == sid {
				found = true
				break
			}
		}
		if !found {
			t.StoreIDs = append(t.StoreIDs, sid)
		}
	}

	// Parse POST body for wechatId and phoneNumber
	if req.Method == http.MethodPost && len(bodyBytes) > 0 {
		var body map[string]any
		if json.Unmarshal(bodyBytes, &body) == nil {
			if wid, ok := body["wechatId"].(string); ok && t.WechatID == "" {
				t.WechatID = wid
			}
			if pn, ok := body["phoneNumber"].(string); ok && t.PhoneNumber == "" {
				t.PhoneNumber = pn
			}
		}
	}
}

func isSushiroTargetHost(host string) bool {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	return host == sushiroHost
}

func shouldBufferSushiroAPIResponse(target *url.URL) bool {
	if target == nil || !isSushiroTargetHost(target.Host) {
		return false
	}
	return strings.HasPrefix(target.Path, "/wechat/api/") ||
		strings.HasPrefix(target.Path, "/wechat/api_auth/")
}

// ---- MITM Proxy Server ----

type proxyServer struct {
	listener  net.Listener
	port      int
	done      chan struct{}
	caCert    tls.Certificate
	caKey     *rsa.PrivateKey
	tokens    *CapturedTokens
	transport *http.Transport
	logf      func(string)
	traceMu   sync.Mutex
	seenHosts map[string]struct{}
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func startProxy(caCert tls.Certificate, caKey *rsa.PrivateKey, tokens *CapturedTokens, logger ...func(string)) (*proxyServer, error) {
	listener, port, err := listenOnAvailableLocalPort(proxyPort, proxyPortSearchLimit)
	if err != nil {
		return nil, fmt.Errorf("listen on %d-%d: %w", proxyPort, proxyPort+proxyPortSearchLimit-1, err)
	}
	var logf func(string)
	if len(logger) > 0 {
		logf = logger[0]
	}

	ps := &proxyServer{
		listener:  listener,
		port:      port,
		done:      make(chan struct{}),
		caCert:    caCert,
		caKey:     caKey,
		tokens:    tokens,
		transport: newProxyUpstreamTransport(),
		logf:      logf,
		seenHosts: map[string]struct{}{},
	}

	go ps.serve()
	return ps, nil
}

func (ps *proxyServer) addLog(msg string) {
	if ps.logf != nil {
		ps.logf(msg)
		return
	}
	logMessage(time.Now(), msg)
}

func (ps *proxyServer) traceConnect(hostPort string, mitm bool) {
	host := requestTraceHost(hostPort)
	if host == "" {
		return
	}
	ps.traceMu.Lock()
	if _, ok := ps.seenHosts[host]; ok {
		ps.traceMu.Unlock()
		return
	}
	ps.seenHosts[host] = struct{}{}
	ps.traceMu.Unlock()
	mode := "passthrough"
	if mitm {
		mode = "mitm"
	}
	ps.addLog(fmt.Sprintf("Request address: CONNECT %s (%s)", host, mode))
}

func (ps *proxyServer) close() {
	if ps.listener != nil {
		ps.listener.Close()
	}
	if ps.transport != nil {
		ps.transport.CloseIdleConnections()
	}
	select {
	case <-ps.done:
	default:
		close(ps.done)
	}
}

func (ps *proxyServer) serve() {
	for {
		conn, err := ps.listener.Accept()
		if err != nil {
			select {
			case <-ps.done:
				return
			default:
				continue
			}
		}
		go ps.handleConn(conn)
	}
}

func (ps *proxyServer) handleConn(clientConn net.Conn) {
	defer clientConn.Close()

	br := bufio.NewReader(clientConn)

	// Read the request line
	firstLine, err := br.ReadString('\n')
	if err != nil {
		return
	}
	firstLine = strings.TrimSpace(firstLine)

	if !strings.HasPrefix(firstLine, "CONNECT ") {
		ps.handlePlainHTTPProxy(clientConn, firstLine, br)
		return
	}

	// Parse target host
	parts := strings.SplitN(firstLine, " ", 3)
	if len(parts) < 2 {
		return
	}
	hostPort := parts[1]

	// Drain remaining CONNECT headers
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}

	isSushiro := isSushiroTargetHost(hostPort)
	ps.traceConnect(hostPort, isSushiro)

	if !isSushiro {
		// Dial the real server for pass-through tunnels.
		serverConn, err := net.DialTimeout("tcp", hostPort, proxyTunnelDialTimeout)
		if err != nil {
			ps.addLog(fmt.Sprintf("Proxy tunnel dial failed: %s: %v", sanitizeProxyHost(hostPort), err))
			fmt.Fprintf(clientConn, "HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\n\r\n")
			return
		}
		defer serverConn.Close()

		// Tell client the tunnel is ready.
		fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")

		// Flush any buffered data from br that read ahead past the CONNECT headers.
		// The MITM path instead wraps br in bufferedConn before the TLS handshake.
		if br.Buffered() > 0 {
			buf := make([]byte, br.Buffered())
			n, _ := br.Read(buf)
			if n > 0 {
				serverConn.Write(buf[:n])
			}
		}
		// Plain tunnel — just relay bytes both ways
		go func() {
			io.Copy(serverConn, clientConn)
			serverConn.Close()
		}()
		io.Copy(clientConn, serverConn)
		return
	}

	// ---- MITM for sushiro ----

	// Tell client the tunnel is ready. The real upstream connection is opened by
	// the HTTP transport when forwarding each captured request.
	fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")

	serverName := hostPort
	if idx := strings.LastIndex(hostPort, ":"); idx != -1 {
		serverName = hostPort[:idx]
	}

	// TLS handshake with client (using forged cert)
	hostCert, err := generateHostCert(ps.caCert, ps.caKey, serverName)
	if err != nil {
		return
	}
	tlsClient := tls.Server(&bufferedConn{Conn: clientConn, reader: br}, &tls.Config{
		Certificates: []tls.Certificate{hostCert},
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"http/1.1"},
	})
	defer tlsClient.Close()
	if err := tlsClient.Handshake(); err != nil {
		ps.addLog(fmt.Sprintf("TLS handshake from client failed: %v", err))
		return
	}

	state := tlsClient.ConnectionState()
	ps.addLog(fmt.Sprintf("MITM established for %s (client_tls=%s alpn=%s)", serverName, tlsVersionName(state.Version), defaultString(state.NegotiatedProtocol, "http/1.1")))

	// Read-Forward-Relay loop
	clientReader := bufio.NewReader(tlsClient)
	for {
		req, err := http.ReadRequest(clientReader)
		if err != nil {
			if err != io.EOF {
				ps.addLog(fmt.Sprintf("MITM read request failed for %s: %v", serverName, err))
			}
			return // client closed connection
		}

		// Read body bytes
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body.Close()
		}

		// Capture tokens
		ps.tokens.captureFromRequest(req, bodyBytes)
		ps.addLog(fmt.Sprintf("Request address: %s %s", req.Method, sanitizedProxyURL(requestURLForTrace(req, "https", hostPort))))

		// Rebuild request for forwarding
		req.URL.Scheme = "https"
		req.URL.Host = hostPort
		req.RequestURI = ""
		if req.Host == "" {
			req.Host = hostWithoutPort(hostPort)
		}
		removeHopByHopHeaders(req.Header)
		setForwardRequestBody(req, bodyBytes)

		// Forward
		resp, err := ps.transport.RoundTrip(req)
		if err != nil {
			ps.addLog(fmt.Sprintf("MITM upstream request failed: %s %s: %v", req.Method, sanitizedProxyURL(req.URL), err))
			fmt.Fprintf(tlsClient, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
			return
		}

		// Relay response
		if err := ps.relayResponse(tlsClient, resp, req.Method, req.URL); err != nil {
			return
		}
	}
}

func (ps *proxyServer) handlePlainHTTPProxy(clientConn net.Conn, firstLine string, br *bufio.Reader) {
	reader := bufio.NewReader(io.MultiReader(strings.NewReader(firstLine+"\r\n"), br))
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			return
		}
		closeAfterResponse := req.Close

		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body.Close()
		}

		if !req.URL.IsAbs() {
			if req.Host == "" {
				fmt.Fprintf(clientConn, "HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
				return
			}
			req.URL.Scheme = "http"
			req.URL.Host = req.Host
		}
		req.RequestURI = ""
		if req.Host == "" {
			req.Host = req.URL.Host
		}
		removeHopByHopHeaders(req.Header)
		setForwardRequestBody(req, bodyBytes)
		ps.addLog(fmt.Sprintf("Request address: %s %s", req.Method, sanitizedProxyURL(req.URL)))

		resp, err := ps.transport.RoundTrip(req)
		if err != nil {
			logMessage(time.Now(), fmt.Sprintf("HTTP proxy request failed: %s %s: %v", req.Method, req.URL.String(), err))
			fmt.Fprintf(clientConn, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
			return
		}
		if err := ps.relayResponse(clientConn, resp, req.Method, req.URL); err != nil {
			return
		}

		if closeAfterResponse {
			return
		}
	}
}

func removeHopByHopHeaders(header http.Header) {
	for _, value := range header.Values("Connection") {
		for _, name := range strings.Split(value, ",") {
			if name = strings.TrimSpace(name); name != "" {
				header.Del(name)
			}
		}
	}
	for _, name := range []string{
		"Proxy-Connection",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Connection",
		"Keep-Alive",
		"TE",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	} {
		header.Del(name)
	}
	header.Del("Content-Length")
}

func newProxyUpstreamTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = func(*http.Request) (*url.URL, error) { return nil, nil }
	transport.ForceAttemptHTTP2 = true
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"},
	}
	return transport
}

func setForwardRequestBody(req *http.Request, bodyBytes []byte) {
	if len(bodyBytes) == 0 {
		req.Body = http.NoBody
		req.ContentLength = 0
		req.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }
	} else {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}
	req.TransferEncoding = nil
	req.Close = false
}

func hostWithoutPort(hostPort string) string {
	if host, _, err := net.SplitHostPort(hostPort); err == nil {
		return host
	}
	return hostPort
}

func (ps *proxyServer) relayResponse(w io.Writer, resp *http.Response, method string, target *url.URL) error {
	defer resp.Body.Close()
	upstreamProto := resp.Proto
	if shouldBufferSushiroAPIResponse(target) {
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxSushiroBufferedResponseBytes)+1))
		if err != nil {
			err = fmt.Errorf("读取寿司郎 API 响应失败: %w", err)
			ps.addLog(fmt.Sprintf("proxy buffer response failed: %s %s: %v", method, sanitizedProxyURL(target), err))
			return err
		}
		if len(bodyBytes) > maxSushiroBufferedResponseBytes {
			err := fmt.Errorf("寿司郎 API 响应超过缓冲上限 %d bytes", maxSushiroBufferedResponseBytes)
			ps.addLog(fmt.Sprintf("proxy buffer response failed: %s %s: %v", method, sanitizedProxyURL(target), err))
			return err
		}
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		resp.ContentLength = int64(len(bodyBytes))
		resp.TransferEncoding = nil
		resp.Header.Del("Transfer-Encoding")
		resp.Header.Del("Content-Length")
	}
	resp.Proto = "HTTP/1.1"
	resp.ProtoMajor = 1
	resp.ProtoMinor = 1
	bodySample := ""
	if resp.StatusCode >= 400 && resp.ContentLength >= 0 && resp.ContentLength <= proxyErrorBodyLogLimit {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			bodySample = sanitizeDiagnosticLine(strings.TrimSpace(string(bodyBytes)))
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			resp.ContentLength = int64(len(bodyBytes))
		}
	}
	if resp.StatusCode >= 400 {
		msg := fmt.Sprintf("MITM upstream response: %s %s HTTP %d", method, sanitizedProxyURL(target), resp.StatusCode)
		if bodySample != "" {
			msg += ": " + bodySample
		}
		ps.addLog(msg)
	} else if target != nil && isSushiroTargetHost(target.Host) {
		ps.addLog(fmt.Sprintf("MITM upstream response: %s %s HTTP %d upstream=%s", method, sanitizedProxyURL(target), resp.StatusCode, upstreamProto))
	}
	// 剥掉 HTTP/3 升级提示，避免客户端（尤其 Windows 微信的 Chromium/XWeb 内核）
	// 据此把后续请求切到 QUIC（UDP 443）——那会绕过只管 TCP 的本代理，导致小程序报网络错误。
	stripQUICAdvertisement(resp.Header)
	if err := resp.Write(w); err != nil {
		ps.addLog(fmt.Sprintf("proxy relay response failed: %s %s: %v", method, sanitizedProxyURL(target), err))
		return err
	}
	return nil
}

// stripQUICAdvertisement 删除会让客户端升级到 HTTP/3 的响应头。
func stripQUICAdvertisement(header http.Header) {
	header.Del("Alt-Svc")
	header.Del("Alternate-Protocol")
}

func sanitizedProxyURL(u *url.URL) string {
	if u == nil {
		return "-"
	}
	out := *u
	if out.RawQuery != "" {
		values := out.Query()
		for key := range values {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "token") || strings.Contains(lower, "auth") || strings.Contains(lower, "phone") || strings.Contains(lower, "wechat") {
				values.Set(key, "***")
			}
		}
		out.RawQuery = values.Encode()
	}
	return out.String()
}

func requestURLForTrace(req *http.Request, scheme, hostPort string) *url.URL {
	if req == nil {
		return nil
	}
	out := *req.URL
	out.Scheme = scheme
	out.Host = hostPort
	if out.Host == "" {
		out.Host = req.Host
	}
	return &out
}

func requestTraceHost(hostPort string) string {
	host := strings.TrimSpace(hostPort)
	if h, _, err := net.SplitHostPort(hostPort); err == nil {
		host = h
	}
	return strings.TrimSuffix(strings.ToLower(host), ".")
}

func sanitizeProxyHost(hostPort string) string {
	host := requestTraceHost(hostPort)
	if host == "" {
		return "-"
	}
	return host
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS13:
		return "TLS1.3"
	case tls.VersionTLS12:
		return "TLS1.2"
	case tls.VersionTLS11:
		return "TLS1.1"
	case tls.VersionTLS10:
		return "TLS1.0"
	default:
		return fmt.Sprintf("0x%x", version)
	}
}

// ---- Capture wait loop ----

func waitForCapture(ctx context.Context, tokens *CapturedTokens, skip <-chan struct{}) error {
	fmt.Println("等待捕获认证参数...")
	fmt.Println("请按以下步骤捕获认证参数：")
	fmt.Println("  1) 在任务管理器里彻底关闭 PC 微信（包括 WeChat.exe / WeChatAppEx.exe）")
	fmt.Println("  2) 重新打开 PC 微信，进入寿司郎小程序")
	fmt.Println("  3) 选择任意一家门店，点击「排队取号」或「立即预约」一次（不必真的提交/支付）")
	fmt.Println("  这样可以同时捕获查询和预约两种认证参数")
	fmt.Println("按回车跳过等待（手动模式）...")
	fmt.Println()

	var lastStatus string
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-skip:
			return nil
		case <-ticker.C:
			currentStatus := strings.Join(tokens.Status(), "\n")
			if currentStatus != lastStatus {
				lastStatus = currentStatus
				for _, line := range tokens.Status() {
					fmt.Println(line)
				}
				fmt.Println()
			}

			if tokens.IsComplete() {
				fmt.Println("所有必要参数已捕获!")
				return nil
			}
		}
	}
}

// ---- Interactive selection ----

func selectStores(ctx context.Context, client *Client, tokens *CapturedTokens) ([]string, error) {
	tokens.mu.Lock()
	storeIDs := make([]string, len(tokens.StoreIDs))
	copy(storeIDs, tokens.StoreIDs)
	tokens.mu.Unlock()

	if len(storeIDs) == 0 {
		fmt.Print("未捕获到门店ID，请手动输入门店编号: ")
		return []string{readInput()}, nil
	}

	fmt.Println("\n--- 可选门店 ---")
	reg := GetStoreRegistry()
	for i, storeID := range storeIDs {
		storeInfo, err := client.GetStoreInfo(ctx, storeID)
		if err != nil {
			fmt.Printf("  %d. 门店 %s（获取详情失败: %v）\n", i+1, storeID, err)
			continue
		}
		displayName := reg.DisplayName(storeID, storeInfo.Name)
		nicknameTag := ""
		if displayName != storeInfo.Name {
			nicknameTag = fmt.Sprintf(" [%s]", displayName)
		}
		fmt.Printf("  %d. %s%s（%s）- %s\n", i+1, storeInfo.Name, nicknameTag, storeID, storeInfo.Address)
	}

	fmt.Print("\n请选择门店编号（多个用逗号分隔，直接回车选全部）: ")
	input := readInput()

	if input == "" {
		return storeIDs, nil
	}

	var selected []string
	for _, s := range strings.Split(input, ",") {
		s = strings.TrimSpace(s)
		var idx int
		if _, err := fmt.Sscanf(s, "%d", &idx); err == nil && idx >= 1 && idx <= len(storeIDs) {
			selected = append(selected, storeIDs[idx-1])
		}
	}
	if len(selected) == 0 {
		return storeIDs, nil
	}
	return selected, nil
}

// SlotPref defines what time range to target for a day type.
type SlotPref int

const (
	PrefNone       SlotPref = iota // 不预约
	Pref1930to2030                 // 19:30-20:30
	PrefBefore2000                 // 20:00前
	Pref1030to1300                 // 10:30-13:00
)

var prefNames = map[SlotPref]string{
	PrefNone:       "不预约",
	Pref1930to2030: "19:30-20:30",
	PrefBefore2000: "20:00前",
	Pref1030to1300: "10:30-13:00",
}

// SlotConfig holds per-day-type slot preferences.
type SlotConfig struct {
	Weekday  SlotPref
	Saturday SlotPref
	Sunday   SlotPref
}

func (c SlotConfig) shouldTarget(slot Slot, loc *time.Location) bool {
	day, err := parseCompactDate(slot.Date, loc)
	if err != nil {
		return false
	}
	weekday := day.Weekday()

	var pref SlotPref
	switch weekday {
	case time.Saturday:
		pref = c.Saturday
	case time.Sunday:
		pref = c.Sunday
	default:
		pref = c.Weekday
	}

	if pref == PrefNone {
		return false
	}

	start := slot.Start
	end := slot.End

	switch pref {
	case Pref1930to2030:
		return start >= "193000" && start < "203000" && end <= "203000"
	case PrefBefore2000:
		return start < "200000" && end <= "200000"
	case Pref1030to1300:
		return start >= "103000" && start < "130000" && end <= "130000"
	}
	return false
}

func configureSlots() SlotConfig {
	opts := []SlotPref{Pref1930to2030, PrefBefore2000, Pref1030to1300, PrefNone}

	choose := func(label string) SlotPref {
		for {
			fmt.Printf("\n%s:\n", label)
			for i, p := range opts {
				fmt.Printf("  %d. %s\n", i+1, prefNames[p])
			}
			fmt.Print("请选择: ")
			input := readInput()
			var idx int
			if _, err := fmt.Sscanf(input, "%d", &idx); err == nil && idx >= 1 && idx <= len(opts) {
				return opts[idx-1]
			}
			fmt.Printf("  无效输入，请输入 1-%d\n", len(opts))
		}
	}

	fmt.Println("\n--- 时段配置 ---")
	return SlotConfig{
		Weekday:  choose("工作日 (周一-周五)"),
		Saturday: choose("周六"),
		Sunday:   choose("周日"),
	}
}

func slotPrefToRanges(pref SlotPref) []TimeRange {
	switch pref {
	case Pref1930to2030:
		return []TimeRange{{Start: "1930", End: "2030"}}
	case PrefBefore2000:
		return []TimeRange{{Start: "0000", End: "2000"}}
	case Pref1030to1300:
		return []TimeRange{{Start: "1030", End: "1300"}}
	default:
		return nil
	}
}
