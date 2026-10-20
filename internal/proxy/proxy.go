package proxy

import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ProxyPort 是代理首选监听端口；若被占用则向上探查 ProxyPortSearchLimit 个端口。
// 8080 是 HTTP 代理的常见端口，便于系统代理一键设置。
const ProxyPort = 8080
const ProxyPortSearchLimit = 100

// 会被 MITM 解密的寿司郎主机只有下面两个（精确匹配，不是整个 sushiro.com.cn 域）。
// 解密、证书签发、PAC 与 Windows QUIC 拦截都以 SushiroInterceptHosts 为准。
const (
	// SushiroHost 是旧接口主机：取号凭证（/wechat/api_auth/）仍从这里抓。
	SushiroHost = "crm-cn-prd.sushiro.com.cn"
	// SushiroGatewayHost 是小程序 2026-10 起查门店实时数据用的新网关（spec 018）。
	SushiroGatewayHost = "sapi.sushiro.com.cn"
	// SushiroGatewayAPIPath 是新网关只读接口的路径前缀。
	SushiroGatewayAPIPath = "/gateway/wechat/api/2.0/"
	// SushiroGatewayStorePath / SushiroGatewayStoresPath 是会缓冲响应、把响应体交给 OnGatewayExchange 的两个只读接口。
	SushiroGatewayStorePath  = SushiroGatewayAPIPath + "getStoreById"
	SushiroGatewayStoresPath = SushiroGatewayAPIPath + "stores"
	// sushiroGatewayMarker 标出网关请求：只有新网关主机上路径以它开头的请求才会调用 OnGatewayExchange。
	sushiroGatewayMarker = "/gateway/"
)

// SushiroInterceptHosts 返回会被解密的主机，顺序固定（旧接口在前）。
func SushiroInterceptHosts() []string {
	return []string{SushiroHost, SushiroGatewayHost}
}

const proxyErrorBodyLogLimit = 4096

// proxyTunnelDialTimeout 限制 pass-through 隧道连真实上游的等待时间，避免卡死在已下线的主机。
const proxyTunnelDialTimeout = 3 * time.Second

// maxSushiroBufferedResponseBytes 限制被缓冲进内存做 API 发现的寿司郎响应体大小（8 MiB）。
// 超过即判定异常并中断，防止恶意/异常大响应把内存吃爆。
const maxSushiroBufferedResponseBytes = 8 << 20

// normalizeProxyHost 去端口、去末尾点、转小写。
func normalizeProxyHost(host string) string {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.TrimSuffix(strings.ToLower(host), ".")
}

// isSushiroTargetHost 判断目标 host 是否是 SushiroInterceptHosts 之一（去端口、去末尾点、忽略大小写，精确匹配）。
// 命中即代表「这条 CONNECT 要做 MITM」，与 allowedProxyTarget 的白名单语义不同：
// 后者放行 sushiro.com.cn 整域做 pass-through，但只有这两个精确的 API 主机才被解密。
func isSushiroTargetHost(host string) bool {
	host = normalizeProxyHost(host)
	for _, intercept := range SushiroInterceptHosts() {
		if host == intercept {
			return true
		}
	}
	return false
}

// isSushiroGatewayHost 判断目标 host 是否是新网关主机（精确匹配）。
func isSushiroGatewayHost(host string) bool {
	return normalizeProxyHost(host) == SushiroGatewayHost
}

// isGatewayReadPath 判断是否是要缓冲响应的两个网关只读接口（getStoreById / stores）。
func isGatewayReadPath(path string) bool {
	path = strings.TrimSuffix(path, "/")
	return path == SushiroGatewayStorePath || path == SushiroGatewayStoresPath
}

// isGatewayExchangeTarget 判断这次请求是否交给 OnGatewayExchange：新网关主机上路径以 /gateway/ 开头的请求。
func isGatewayExchangeTarget(target *url.URL) bool {
	return target != nil && isSushiroGatewayHost(target.Host) && strings.HasPrefix(target.Path, sushiroGatewayMarker)
}

// allowedProxyTarget reports whether a CONNECT tunnel / plain HTTP proxy
// forward may be opened to the given host. The mobile capture proxy binds
// 0.0.0.0 so a phone on the LAN can route sushiro traffic through it; without
// this gate any LAN peer could turn the capture proxy into an open proxy /
// SSRF hop by CONNECTing to arbitrary hosts. Only sushiro-owned hosts are
// permitted — that is the sole purpose of the capture session. Everything
// else is rejected before any outbound dial.
//
// 中文要点：这是「防止本进程变成开放代理/SSRF 跳板」的核心闸门。
// 移动抓包代理监听 0.0.0.0（局域网可连），若不限制目标，同网段任意设备都能拿它当中转连任何主机。
// 这里只放行 sushiro.com.cn 域（精确 API 主机 + 防御性子域后缀匹配，覆盖 CDN/静态资源），
// 其余一律在拨号前拒绝。注意：放行 ≠ 解密，只有 SushiroInterceptHosts 里的两个精确主机才会被 MITM（见 isSushiroTargetHost）。
func allowedProxyTarget(host string) bool {
	host = normalizeProxyHost(host)
	if host == "" {
		return false
	}
	// Exact sushiro API hosts (the only things we actually MITM).
	if isSushiroTargetHost(host) {
		return true
	}
	// Defensive suffix match so a sushiro subdomain the mobile client hits
	// (e.g. static/CDN) still passes, but nothing outside sushiro.com.cn.
	return strings.HasSuffix(host, ".sushiro.com.cn")
}

// shouldBufferSushiroAPIResponse 决定是否把响应体整块读进内存（API 发现记录、OnGatewayExchange）。
// 旧接口主机下的 /wechat/api/ 与 /wechat/api_auth/ 路径，以及新网关的 getStoreById / stores 才缓冲——
// 这些是含业务语义的接口；其它响应一律流式透传，避免无谓占用内存。
func shouldBufferSushiroAPIResponse(target *url.URL) bool {
	if target == nil {
		return false
	}
	switch normalizeProxyHost(target.Host) {
	case SushiroHost:
		return strings.HasPrefix(target.Path, "/wechat/api/") ||
			strings.HasPrefix(target.Path, "/wechat/api_auth/")
	case SushiroGatewayHost:
		return isGatewayReadPath(target.Path)
	}
	return false
}

// ---- MITM Proxy Server ----

type ProxyServer struct {
	listener      net.Listener
	port          int
	listenHost    string
	done          chan struct{}
	caCert        tls.Certificate
	caKey         *rsa.PrivateKey
	tokens        *CapturedTokens
	transport     *http.Transport
	logf          func(string)
	traceMu       sync.Mutex
	seenHosts     map[string]struct{}
	patchRequests bool
	onGateway     GatewayExchangeFunc
}

// GatewayExchangeFunc 接收一次经代理转发的新网关请求：req 是转发出去的请求（含头部），status 是上游状态码，
// body 只在 getStoreById / stores 两个只读接口时给出（已解压的响应体），其余为 nil。
// 实现方不得把 body 或请求头的值写进磁盘或日志。它在这条连接的转发循环里同步调用，
// 返回前代理不会读小程序在同一连接上的下一条请求，所以耗时的工作要交给别的 goroutine。
type GatewayExchangeFunc func(req *http.Request, status int, body []byte)

type ProxyOptions struct {
	ListenHost    string
	PatchRequests bool
	// OnGatewayExchange 只在新网关主机（sapi.sushiro.com.cn）上路径以 /gateway/ 开头的请求转发完成后调用，
	// 从不为其它主机调用；为 nil 时不调用。proxy 包不依赖 app，由界面进程注入处理函数。
	OnGatewayExchange GatewayExchangeFunc
}

// DesktopCaptureOptions 是电脑抓包代理的默认设置：只监听本机，并把 PC 微信请求改成手机兼容形态。
func DesktopCaptureOptions() ProxyOptions {
	return ProxyOptions{ListenHost: "127.0.0.1", PatchRequests: true}
}

// MobileCaptureOptions 是手机抓包代理的默认设置：监听局域网，保持手机原始请求不改。
func MobileCaptureOptions() ProxyOptions {
	return ProxyOptions{ListenHost: "0.0.0.0", PatchRequests: false}
}

// BufferedConn 把一个底层 net.Conn 连同一个 bufio.Reader 包成统一的 Read 来源。
// MITM 路径里，读 CONNECT 头用的 bufio.Reader 可能已经「多读」进了若干 TLS 握手字节，
// 直接把裸 conn 交给 tls.Server 会丢这些字节；用 BufferedConn 让 Read 优先消费缓冲区，再回退到底层 conn。
type BufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *BufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func StartProxy(caCert tls.Certificate, caKey *rsa.PrivateKey, tokens *CapturedTokens, logger ...func(string)) (*ProxyServer, error) {
	return StartProxyWithOptions(caCert, caKey, tokens, DesktopCaptureOptions(), logger...)
}

func StartMobileCaptureProxy(caCert tls.Certificate, caKey *rsa.PrivateKey, tokens *CapturedTokens, logger ...func(string)) (*ProxyServer, error) {
	return StartProxyWithOptions(caCert, caKey, tokens, MobileCaptureOptions(), logger...)
}

func StartProxyWithOptions(caCert tls.Certificate, caKey *rsa.PrivateKey, tokens *CapturedTokens, options ProxyOptions, logger ...func(string)) (*ProxyServer, error) {
	listenHost := strings.TrimSpace(options.ListenHost)
	if listenHost == "" {
		listenHost = "127.0.0.1"
	}
	listener, port, err := ListenOnAvailableHostPort(listenHost, ProxyPort, ProxyPortSearchLimit)
	if err != nil {
		return nil, fmt.Errorf("listen on %d-%d: %w", ProxyPort, ProxyPort+ProxyPortSearchLimit-1, err)
	}
	var logf func(string)
	if len(logger) > 0 {
		logf = logger[0]
	}

	ps := &ProxyServer{
		listener:      listener,
		port:          port,
		listenHost:    listenHost,
		done:          make(chan struct{}),
		caCert:        caCert,
		caKey:         caKey,
		tokens:        tokens,
		transport:     newProxyUpstreamTransport(),
		logf:          logf,
		seenHosts:     map[string]struct{}{},
		patchRequests: options.PatchRequests,
		onGateway:     options.OnGatewayExchange,
	}

	go ps.serve()
	return ps, nil
}

func (ps *ProxyServer) addLog(msg string) {
	if ps.logf != nil {
		ps.logf(msg)
		return
	}
	LogMessage(time.Now(), msg)
}

func (ps *ProxyServer) traceConnect(hostPort string, mitm bool) {
	host := requestTraceHost(hostPort)
	if host == "" {
		return
	}
	ps.traceMu.Lock()
	defer ps.traceMu.Unlock()
	if ps.seenHosts == nil {
		ps.seenHosts = map[string]struct{}{}
	}
	if _, ok := ps.seenHosts[host]; ok {
		return
	}
	ps.seenHosts[host] = struct{}{}
	mode := "passthrough"
	if mitm {
		mode = "mitm"
	}
	ps.addLog(fmt.Sprintf("Request address: CONNECT %s (%s)", host, mode))
}

func (ps *ProxyServer) Close() {
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

func (ps *ProxyServer) serve() {
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

func (ps *ProxyServer) handleConn(clientConn net.Conn) {
	defer clientConn.Close()

	br := bufio.NewReader(clientConn)

	// Read the request line
	firstLine, err := br.ReadString('\n')
	if err != nil {
		return
	}
	firstLine = strings.TrimSpace(firstLine)

	if !strings.HasPrefix(firstLine, "CONNECT ") {
		// 非 CONNECT：明文 HTTP 代理请求，走另一条转发路径（同样有白名单闸门）。
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

	// 关键分支：只有两个精确的寿司郎 API 主机走 MITM（解密 + 抓凭证），其余 sushiro.com.cn 子域走 pass-through 透传，
	// 其它一切 host 直接 403 拒绝。这决定了下游两条完全不同的连接处理路径。
	isSushiro := isSushiroTargetHost(hostPort)
	ps.traceConnect(hostPort, isSushiro)

	if !isSushiro {
		// Security gate: the mobile capture proxy binds 0.0.0.0 so a phone on
		// the LAN can route sushiro traffic here. Without this check any LAN
		// peer could CONNECT to an arbitrary host:port and use this process as
		// an open proxy / SSRF relay. Reject anything that is not sushiro.
		if !allowedProxyTarget(hostPort) {
			ps.addLog(fmt.Sprintf("Proxy tunnel blocked (non-allowlisted target): %s", sanitizeProxyHost(hostPort)))
			fmt.Fprintf(clientConn, "HTTP/1.1 403 Forbidden\r\nConnection: close\r\nContent-Length: 0\r\n\r\n")
			return
		}
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
		// The MITM path instead wraps br in BufferedConn before the TLS handshake.
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
	// 这一段对寿司郎 API 做「中间人」：先伪造一张该域名的 TLS 证书（由本地 CA 签发）骗过客户端握手，
	// 随后在解密后的明文里读 HTTP 请求、抓取凭证参数、按需 patch 后转发到真服务器，再把响应回传给客户端。

	// Tell client the tunnel is ready. The real upstream connection is opened by
	// the HTTP transport when forwarding each captured request.
	fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")

	serverName := hostPort
	if idx := strings.LastIndex(hostPort, ":"); idx != -1 {
		serverName = hostPort[:idx]
	}

	// TLS handshake with client (using forged cert)
	// 用本地 CA 即时签发一张 serverName 的叶子证书，才能在 tls.Server 端完成与客户端的握手。
	// 前提是客户端系统已信任本 CA（见 cert.go 的安装流程），否则客户端会报证书不受信任。
	hostCert, err := generateHostCert(ps.caCert, ps.caKey, serverName)
	if err != nil {
		return
	}
	tlsClient := tls.Server(&BufferedConn{Conn: clientConn, reader: br}, &tls.Config{
		Certificates: []tls.Certificate{hostCert},
		MinVersion:   tls.VersionTLS12,
		// 锁定 ALPN 为 http/1.1：本代理只解析 HTTP/1.1 文本，强制不与客户端协商 h2，否则后续 ReadRequest 会失败。
		NextProtos: []string{"http/1.1"},
	})
	defer tlsClient.Close()
	if err := tlsClient.Handshake(); err != nil {
		ps.addLog(fmt.Sprintf("TLS handshake from client failed: %v", err))
		return
	}

	state := tlsClient.ConnectionState()
	ps.addLog(fmt.Sprintf("MITM established for %s (client_tls=%s alpn=%s)", serverName, tlsVersionName(state.Version), DefaultString(state.NegotiatedProtocol, "http/1.1")))
	gateway := isSushiroGatewayHost(hostPort)

	// Read-Forward-Relay loop
	// 在同一条 TLS 连接上循环：读一条请求 → 抓凭证/patch → 转发真服务器 → 回传响应。
	// 客户端主动关闭（io.EOF）即结束；任何读/转发失败都结束本连接。
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
		requestHeaderKeys := APIDiscoveryHeaderKeys(req.Header)
		requestBodyKeys := APIDiscoveryPayloadKeys(bodyBytes)
		requestBodyFields := APIDiscoveryPayloadFieldKinds(bodyBytes)
		// patchRequests 仅桌面代理启用：把 PC 微信发的请求体改造成手机端兼容形态再转发，
		// 移动抓包代理（PatchRequests=false）不改请求，保持手机原始流量。
		// 新网关只旁观、不改写：小程序发什么就转发什么，连接里存下的也是它自己的请求头。
		if ps.patchRequests && !gateway {
			var patches []string
			bodyBytes, patches = patchSushiroRequestForForward(req, bodyBytes)
			if len(patches) > 0 {
				ps.addLog(fmt.Sprintf("MITM request patched for mobile compatibility: %s %s (%s)", req.Method, sanitizedProxyURL(requestURLForTrace(req, "https", hostPort)), strings.Join(patches, ", ")))
			}
		}

		// Capture tokens
		// 抓取凭证的核心一步：从请求头/体里识别 wechatId、token、门店等参数写入 tokens。
		// 取号凭证只来自旧接口主机；新网关的会员 token 只交给 OnGatewayExchange（实时数据连接），
		// 不写进取号凭证，网关请求只顺手记下门店编号。
		if gateway {
			captureGatewayStoreID(ps.tokens, req)
		} else if ps.tokens != nil {
			ps.tokens.CaptureFromRequest(req, bodyBytes)
		}
		ps.addLog(fmt.Sprintf("Request address: %s %s", req.Method, sanitizedProxyURL(requestURLForTrace(req, "https", hostPort))))

		// Rebuild request for forwarding
		// 把代理收到的请求改写成「直连真服务器」的绝对 URL 请求，再剥掉逐跳头交给 transport。
		req.URL.Scheme = "https"
		req.URL.Host = hostPort
		req.RequestURI = ""
		if req.Host == "" {
			req.Host = hostWithoutPort(hostPort)
		}
		removeHopByHopHeaders(req.Header)
		if gateway && isGatewayReadPath(req.URL.Path) {
			// 让上游 transport 自己协商 gzip 并透明解压：缓冲下来的门店数据才是可读的 JSON（小程序可能要 br）。
			req.Header.Del("Accept-Encoding")
		}
		setForwardRequestBody(req, bodyBytes)

		// Forward
		resp, err := ps.transport.RoundTrip(req)
		if err != nil {
			ps.addLog(fmt.Sprintf("MITM upstream request failed: %s %s: %v", req.Method, sanitizedProxyURL(req.URL), err))
			fmt.Fprintf(tlsClient, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
			return
		}

		// Relay response
		bufferedBody, err := ps.relayResponse(tlsClient, resp, req.Method, req.URL, requestHeaderKeys, requestBodyKeys, requestBodyFields)
		if err != nil {
			return
		}
		// 响应已回给小程序之后才交给界面进程：不拖慢小程序，也不改它看到的内容。
		if gateway {
			ps.notifyGatewayExchange(req, resp, bufferedBody)
		}
	}
}

// captureGatewayStoreID 只从新网关请求里记下门店编号（取号凭证的“门店”一项），不读任何头部。
func captureGatewayStoreID(tokens *CapturedTokens, req *http.Request) {
	if tokens == nil || req == nil || req.URL == nil {
		return
	}
	storeID := strings.TrimSpace(req.URL.Query().Get("storeId"))
	if storeID == "" || len(storeID) > 12 || strings.Trim(storeID, "0123456789") != "" {
		return
	}
	tokens.Lock()
	defer tokens.Unlock()
	for _, existing := range tokens.StoreIDs {
		if existing == storeID {
			return
		}
	}
	tokens.StoreIDs = append(tokens.StoreIDs, storeID)
}

// notifyGatewayExchange 把一次新网关请求交给注入的处理函数。只对新网关主机调用；
// 响应体按 Content-Encoding 解压后给出，解不开时给 nil。处理函数出问题不影响代理继续工作。
func (ps *ProxyServer) notifyGatewayExchange(req *http.Request, resp *http.Response, bufferedBody []byte) {
	if ps.onGateway == nil || req == nil || resp == nil || !isGatewayExchangeTarget(req.URL) {
		return
	}
	var body []byte
	if bufferedBody != nil && isGatewayReadPath(req.URL.Path) {
		body = decodeGatewayBody(resp.Header.Get("Content-Encoding"), bufferedBody)
	}
	defer func() {
		if r := recover(); r != nil {
			ps.addLog(fmt.Sprintf("Gateway exchange handler panicked: %s %s", req.Method, sanitizedProxyURL(req.URL)))
		}
	}()
	ps.onGateway(req, resp.StatusCode, body)
}

// decodeGatewayBody 返回可读的响应体：未压缩原样返回，gzip / deflate 解压（同样受缓冲上限约束），其它编码返回 nil。
func decodeGatewayBody(encoding string, body []byte) []byte {
	var reader io.Reader
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "", "identity":
		return body
	case "gzip", "x-gzip":
		zr, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil
		}
		defer zr.Close()
		reader = zr
	case "deflate":
		if zr, err := zlib.NewReader(bytes.NewReader(body)); err == nil {
			defer zr.Close()
			reader = zr
		} else {
			reader = flate.NewReader(bytes.NewReader(body))
		}
	default:
		return nil
	}
	decoded, err := io.ReadAll(io.LimitReader(reader, int64(maxSushiroBufferedResponseBytes)+1))
	if err != nil || len(decoded) > maxSushiroBufferedResponseBytes {
		return nil
	}
	return decoded
}

func (ps *ProxyServer) handlePlainHTTPProxy(clientConn net.Conn, firstLine string, br *bufio.Reader) {
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
		requestHeaderKeys := APIDiscoveryHeaderKeys(req.Header)
		requestBodyKeys := APIDiscoveryPayloadKeys(bodyBytes)
		requestBodyFields := APIDiscoveryPayloadFieldKinds(bodyBytes)

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
		// Security gate (same rationale as the CONNECT path): the mobile
		// capture proxy must not become an open forward proxy for arbitrary
		// hosts. Only sushiro targets may be relayed over plain HTTP.
		if !allowedProxyTarget(req.URL.Host) {
			ps.addLog(fmt.Sprintf("HTTP proxy blocked (non-allowlisted target): %s %s", req.Method, sanitizeProxyHost(req.URL.Host)))
			fmt.Fprintf(clientConn, "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
			return
		}
		removeHopByHopHeaders(req.Header)
		setForwardRequestBody(req, bodyBytes)
		ps.addLog(fmt.Sprintf("Request address: %s %s", req.Method, sanitizedProxyURL(req.URL)))

		resp, err := ps.transport.RoundTrip(req)
		if err != nil {
			LogMessage(time.Now(), fmt.Sprintf("HTTP proxy request failed: %s %s: %v", req.Method, req.URL.String(), err))
			fmt.Fprintf(clientConn, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
			return
		}
		if _, err := ps.relayResponse(clientConn, resp, req.Method, req.URL, requestHeaderKeys, requestBodyKeys, requestBodyFields); err != nil {
			return
		}

		if closeAfterResponse {
			return
		}
	}
}

// removeHopByHopHeaders 转发前剥掉「逐跳」头：这些头只在相邻一跳的连接上有意义，
// 透传到上游会破坏 HTTP 语义（如让上游误以为客户端要 Upgrade、或被 Keep-Alive 带偏连接复用）。
// 先按 Connection 头里点名的字段删，再删固定的逐跳头集合；最后删 Content-Length，
// 因为下面 setForwardRequestBody 会用已知 bodyBytes 重新设置。
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

// relayResponse 把上游响应写回客户端；对需要缓冲的寿司郎接口返回缓冲下来的响应体（原样，可能仍是压缩的），其余返回 nil。
func (ps *ProxyServer) relayResponse(w io.Writer, resp *http.Response, method string, target *url.URL, requestHeaderKeys, requestBodyKeys []string, requestBodyFields map[string]string) ([]byte, error) {
	defer resp.Body.Close()
	upstreamProto := resp.Proto
	var bufferedBody []byte
	if shouldBufferSushiroAPIResponse(target) {
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxSushiroBufferedResponseBytes)+1))
		if err != nil {
			err = fmt.Errorf("读取寿司郎 API 响应失败: %w", err)
			ps.addLog(fmt.Sprintf("proxy buffer response failed: %s %s: %v", method, sanitizedProxyURL(target), err))
			return nil, err
		}
		if len(bodyBytes) > maxSushiroBufferedResponseBytes {
			err := fmt.Errorf("寿司郎 API 响应超过缓冲上限 %d bytes", maxSushiroBufferedResponseBytes)
			ps.addLog(fmt.Sprintf("proxy buffer response failed: %s %s: %v", method, sanitizedProxyURL(target), err))
			return nil, err
		}
		bufferedBody = bodyBytes
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		resp.ContentLength = int64(len(bodyBytes))
		resp.TransferEncoding = nil
		resp.Header.Del("Transfer-Encoding")
		resp.Header.Del("Content-Length")
		if err := RecordAPIDiscovery(BuildAPIDiscoveryRecord(method, target, resp.StatusCode, upstreamProto, requestHeaderKeys, requestBodyKeys, requestBodyFields, bodyBytes)); err != nil {
			ps.addLog(fmt.Sprintf("API discovery record failed: %s %s: %v", method, sanitizedProxyURL(target), err))
		}
	}
	resp.Proto = "HTTP/1.1"
	resp.ProtoMajor = 1
	resp.ProtoMinor = 1
	bodySample := ""
	// 新网关的错误响应体不进日志：它可能原样带回请求里的会员 token，脱敏规则认不出来。
	gatewayResponse := target != nil && isSushiroGatewayHost(target.Host)
	if resp.StatusCode >= 400 && !gatewayResponse && resp.ContentLength >= 0 && resp.ContentLength <= proxyErrorBodyLogLimit {
		bodyBytes := bufferedBody
		var err error
		if bodyBytes == nil {
			bodyBytes, err = io.ReadAll(resp.Body)
		}
		if err == nil {
			bodySample = SanitizeDiagnosticLine(strings.TrimSpace(string(bodyBytes)))
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
		return nil, err
	}
	return bufferedBody, nil
}

// stripQUICAdvertisement 删除会让客户端升级到 HTTP/3 的响应头。
func stripQUICAdvertisement(header http.Header) {
	header.Del("Alt-Svc")
	header.Del("Alternate-Protocol")
}

// sanitizedProxyURL 把 URL 里的敏感查询参数脱敏后再用于日志/展示。
// 命中 token/auth/phone/wechat/code/openid/unionid/session/secret/ticket/sign/key/sid 等关键字的参数值
// 一律替换为 ***，防止把微信凭证、手机号写进日志外泄。返回的是拷贝，不改动原 URL。
func sanitizedProxyURL(u *url.URL) string {
	if u == nil {
		return "-"
	}
	out := *u
	if out.RawQuery != "" {
		values := out.Query()
		for key := range values {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "token") || strings.Contains(lower, "auth") ||
				strings.Contains(lower, "phone") || strings.Contains(lower, "wechat") ||
				strings.Contains(lower, "code") || strings.Contains(lower, "openid") ||
				strings.Contains(lower, "unionid") || strings.Contains(lower, "session") ||
				strings.Contains(lower, "secret") || strings.Contains(lower, "ticket") ||
				strings.Contains(lower, "sign") || lower == "key" || lower == "sid" {
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

func WaitForCapture(ctx context.Context, tokens *CapturedTokens, skip <-chan struct{}) error {
	fmt.Println("等待捕获凭证参数...")
	fmt.Println("请按以下步骤捕获凭证参数：")
	fmt.Println("  1) 在任务管理器里彻底关闭 PC 微信（包括 WeChat.exe / WeChatAppEx.exe）")
	fmt.Println("  2) 重新打开 PC 微信，进入寿司郎小程序")
	fmt.Println("  3) 选择任意一家门店，点击「排队取号」或「立即预约」一次（不必真的提交/支付）")
	fmt.Println("  这样可以同时捕获查询和预约两种凭证参数")
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

func SelectStores(ctx context.Context, client *Client, tokens *CapturedTokens) ([]string, error) {
	tokens.Lock()
	storeIDs := make([]string, len(tokens.StoreIDs))
	copy(storeIDs, tokens.StoreIDs)
	tokens.Unlock()

	if len(storeIDs) == 0 {
		fmt.Print("未捕获到门店ID，请手动输入门店编号: ")
		return []string{ReadInput()}, nil
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
	input := ReadInput()

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

func (c SlotConfig) ShouldTarget(slot Slot, loc *time.Location) bool {
	day, err := ParseCompactDate(slot.Date, loc)
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

func ConfigureSlots() SlotConfig {
	opts := []SlotPref{Pref1930to2030, PrefBefore2000, Pref1030to1300, PrefNone}

	choose := func(label string) SlotPref {
		for {
			fmt.Printf("\n%s:\n", label)
			for i, p := range opts {
				fmt.Printf("  %d. %s\n", i+1, prefNames[p])
			}
			fmt.Print("请选择: ")
			input := ReadInput()
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

func SlotPrefToRanges(pref SlotPref) []TimeRange {
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

// Port 返回代理监听端口。
func (ps *ProxyServer) Port() int { return ps.port }

// NewBufferedConn 构造带预读缓冲的连接（供外部包用）。
func NewBufferedConn(conn net.Conn, r *bufio.Reader) *BufferedConn {
	return &BufferedConn{Conn: conn, reader: r}
}
