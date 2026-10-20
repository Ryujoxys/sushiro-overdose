package proxy

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Ryujoxys/sushiro-overdose/internal/core"
)

const (
	testGatewayToken   = "Bearer eyJhbGciOiJIUzI1NiJ9.eyJleHAiOjQxMDI0NDQ4MDB9.c2VjcmV0LXNpZ25hdHVyZQ"
	testGatewayAppCode = "gateway-app-code-secret"
)

func TestInterceptHostsAreExactlyTheTwoAPIHosts(t *testing.T) {
	if got := SushiroInterceptHosts(); len(got) != 2 || got[0] != "crm-cn-prd.sushiro.com.cn" || got[1] != "sapi.sushiro.com.cn" {
		t.Fatalf("intercept hosts = %v", got)
	}
	cases := map[string]bool{
		"crm-cn-prd.sushiro.com.cn":     true,
		"crm-cn-prd.sushiro.com.cn:443": true,
		"sapi.sushiro.com.cn":           true,
		"sapi.sushiro.com.cn:443":       true,
		" SAPI.Sushiro.com.cn. ":        true,
		// Allowed to pass through, never decrypted.
		"static.sushiro.com.cn:443":      false,
		"sushiro.com.cn":                 false,
		"x.sapi.sushiro.com.cn":          false,
		"sapi.sushiro.com.cn.evil.com":   false,
		"sapi-sushiro.com.cn":            false,
		"crm-cn-prd.sushiro.com.cn.evil": false,
		"":                               false,
	}
	for host, want := range cases {
		if got := isSushiroTargetHost(host); got != want {
			t.Errorf("isSushiroTargetHost(%q) = %v, want %v", host, got, want)
		}
	}
	// The open-proxy gate still admits the gateway host (and only sushiro.com.cn).
	for host, want := range map[string]bool{"sapi.sushiro.com.cn:443": true, "evil.example:443": false} {
		if got := allowedProxyTarget(host); got != want {
			t.Errorf("allowedProxyTarget(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestOnlyGatewayReadEndpointsAreBufferedOnTheGateway(t *testing.T) {
	cases := map[string]bool{
		"https://sapi.sushiro.com.cn/gateway/wechat/api/2.0/getStoreById?storeId=1012": true,
		"https://sapi.sushiro.com.cn/gateway/wechat/api/2.0/stores?latitude=1":         true,
		"https://sapi.sushiro.com.cn/gateway/wechat/api/2.0/stores/":                   true,
		"https://sapi.sushiro.com.cn/gateway/wechat/api/2.0/queue/takeTicket":          false,
		"https://sapi.sushiro.com.cn/wechat/api/2.0/getStoreById":                      false,
		"https://crm-cn-prd.sushiro.com.cn/wechat/api/2.0/getStoreById":                true,
		"https://crm-cn-prd.sushiro.com.cn/gateway/wechat/api/2.0/getStoreById":        false,
		"https://static.sushiro.com.cn/gateway/wechat/api/2.0/getStoreById":            false,
	}
	for raw, want := range cases {
		target, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if got := shouldBufferSushiroAPIResponse(target); got != want {
			t.Errorf("shouldBuffer(%s) = %v, want %v", raw, got, want)
		}
	}
	for raw, want := range map[string]bool{
		"https://sapi.sushiro.com.cn:443/gateway/wechat/api/2.0/getStoreById": true,
		"https://sapi.sushiro.com.cn/gateway/other":                           true,
		"https://sapi.sushiro.com.cn/static/app.js":                           false,
		"https://crm-cn-prd.sushiro.com.cn/gateway/wechat/api/2.0/stores":     false,
	} {
		target, _ := url.Parse(raw)
		if got := isGatewayExchangeTarget(target); got != want {
			t.Errorf("isGatewayExchangeTarget(%s) = %v, want %v", raw, got, want)
		}
	}
}

type gatewayCall struct {
	method        string
	path          string
	status        int
	body          []byte
	authorization string
}

type gatewayHarness struct {
	t        *testing.T
	ps       *ProxyServer
	caPool   *x509.CertPool
	upstream *httptest.Server
	calls    chan gatewayCall
	logsMu   sync.Mutex
	logs     []string
	upMu     sync.Mutex
	upSeen   []*http.Request
}

func newGatewayHarness(t *testing.T, handler http.HandlerFunc) *gatewayHarness {
	t.Helper()
	t.Setenv("SUSHIRO_DATA_HOME", t.TempDir())
	caCert, caKey, caPool := newTestCA(t)
	h := &gatewayHarness{t: t, caPool: caPool, calls: make(chan gatewayCall, 8)}
	h.upstream = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.upMu.Lock()
		h.upSeen = append(h.upSeen, r.Clone(context.Background()))
		h.upMu.Unlock()
		handler(w, r)
	}))
	t.Cleanup(h.upstream.Close)
	upstreamPool := x509.NewCertPool()
	upstreamPool.AddCert(h.upstream.Certificate())
	upstreamAddr := h.upstream.Listener.Addr().String()
	h.ps = &ProxyServer{
		caCert: caCert,
		caKey:  caKey,
		tokens: core.NewCapturedTokens(),
		// Every upstream connection goes to the local test server; nothing leaves this machine.
		transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "tcp", upstreamAddr)
			},
			TLSClientConfig: &tls.Config{RootCAs: upstreamPool, ServerName: "example.com"},
		},
		logf: func(msg string) {
			h.logsMu.Lock()
			h.logs = append(h.logs, msg)
			h.logsMu.Unlock()
		},
		onGateway: func(req *http.Request, status int, body []byte) {
			h.calls <- gatewayCall{method: req.Method, path: req.URL.Path, status: status, body: append([]byte(nil), body...), authorization: req.Header.Get("Authorization")}
		},
	}
	return h
}

// roundTrip opens one CONNECT tunnel to host through the proxy, sends each request over the
// decrypted connection and returns the response bodies.
func (h *gatewayHarness) roundTrip(host string, requests ...*http.Request) []string {
	h.t.Helper()
	client, server := net.Pipe()
	done := make(chan struct{})
	go func() {
		h.ps.handleConn(server)
		close(done)
	}()
	defer func() {
		client.Close()
		<-done
	}()
	_ = client.SetDeadline(time.Now().Add(20 * time.Second))
	if _, err := fmt.Fprintf(client, "CONNECT %s:443 HTTP/1.1\r\nHost: %s:443\r\n\r\n", host, host); err != nil {
		h.t.Fatalf("write CONNECT: %v", err)
	}
	reader := bufio.NewReader(client)
	status, err := reader.ReadString('\n')
	if err != nil || !strings.Contains(status, "200") {
		h.t.Fatalf("CONNECT status = %q, %v", status, err)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			h.t.Fatalf("read CONNECT headers: %v", err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}
	tlsConn := tls.Client(&BufferedConn{Conn: client, reader: reader}, &tls.Config{RootCAs: h.caPool, ServerName: host, NextProtos: []string{"http/1.1"}})
	if err := tlsConn.Handshake(); err != nil {
		h.t.Fatalf("TLS handshake with the proxy for %s: %v", host, err)
	}
	if got := tlsConn.ConnectionState().PeerCertificates[0].DNSNames; len(got) != 1 || got[0] != host {
		h.t.Fatalf("leaf certificate names = %v, want [%s]", got, host)
	}
	tlsReader := bufio.NewReader(tlsConn)
	bodies := []string{}
	for _, req := range requests {
		if err := req.Write(tlsConn); err != nil {
			h.t.Fatalf("write request: %v", err)
		}
		resp, err := http.ReadResponse(tlsReader, req)
		if err != nil {
			h.t.Fatalf("read response: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		bodies = append(bodies, string(body))
	}
	return bodies
}

func (h *gatewayHarness) nextCall() gatewayCall {
	h.t.Helper()
	select {
	case call := <-h.calls:
		return call
	case <-time.After(10 * time.Second):
		h.t.Fatal("OnGatewayExchange was not called")
	}
	return gatewayCall{}
}

func (h *gatewayHarness) noMoreCalls() {
	h.t.Helper()
	select {
	case call := <-h.calls:
		h.t.Fatalf("unexpected OnGatewayExchange call: %+v", call)
	case <-time.After(50 * time.Millisecond):
	}
}

func gatewayRequest(t *testing.T, host, path string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "https://"+host+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", testGatewayToken)
	req.Header.Set("X-App-Code", testGatewayAppCode)
	req.Header.Set("X-App-Client", "mini")
	req.Header.Set("Accept-Encoding", "br, gzip, deflate")
	return req
}

func TestProxyDecryptsGatewayAndReportsReadEndpoints(t *testing.T) {
	const storeJSON = `{"id":1012,"name":"南山店","storeStatus":"OPEN","wait":30,"groupQueues":{"boothQueue":["900"]}}`
	h := newGatewayHarness(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case SushiroGatewayStorePath:
			// The transport asked for gzip itself and decodes it; the mini program's "br" never reaches upstream.
			if r.Header.Get("Accept-Encoding") != "gzip" {
				t.Errorf("upstream Accept-Encoding = %q, want transport gzip", r.Header.Get("Accept-Encoding"))
			}
			w.Header().Set("Content-Encoding", "gzip")
			zw := gzip.NewWriter(w)
			_, _ = zw.Write([]byte(storeJSON))
			_ = zw.Close()
		default:
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	})

	bodies := h.roundTrip(SushiroGatewayHost,
		gatewayRequest(t, SushiroGatewayHost, SushiroGatewayStorePath+"?storeId=1012"),
		gatewayRequest(t, SushiroGatewayHost, "/gateway/wechat/api/2.0/queue/info"),
	)
	if bodies[0] != storeJSON || bodies[1] != `{"ok":true}` {
		t.Fatalf("client bodies = %q", bodies)
	}

	store := h.nextCall()
	if store.method != http.MethodGet || store.path != SushiroGatewayStorePath || store.status != http.StatusOK || string(store.body) != storeJSON {
		t.Fatalf("store exchange = %+v (body %q)", store, store.body)
	}
	if store.authorization != testGatewayToken {
		t.Fatal("handler did not receive the forwarded request headers")
	}
	other := h.nextCall()
	if other.path != "/gateway/wechat/api/2.0/queue/info" || other.body != nil {
		t.Fatalf("non-read gateway exchange should carry no body: %+v", other)
	}
	h.noMoreCalls()

	// The member token is never written into ticket credentials; only the store id is noted.
	h.ps.tokens.Lock()
	queryAuth, reservationAuth, appCode, stores := h.ps.tokens.QueryAuth, h.ps.tokens.ReservationAuth, h.ps.tokens.XAppCode, append([]string(nil), h.ps.tokens.StoreIDs...)
	h.ps.tokens.Unlock()
	if queryAuth != "" || reservationAuth != "" || appCode != "" {
		t.Fatalf("gateway headers leaked into ticket credentials: query=%q reservation=%q app=%q", queryAuth, reservationAuth, appCode)
	}
	if len(stores) != 1 || stores[0] != "1012" {
		t.Fatalf("store ids = %v", stores)
	}
	h.logsMu.Lock()
	logs := strings.Join(h.logs, "\n")
	h.logsMu.Unlock()
	for _, secret := range []string{"c2VjcmV0LXNpZ25hdHVyZQ", testGatewayAppCode} {
		if strings.Contains(logs, secret) {
			t.Fatalf("proxy log exposes a header value:\n%s", logs)
		}
	}
	if !strings.Contains(logs, "MITM established for sapi.sushiro.com.cn") {
		t.Fatalf("gateway host was not decrypted:\n%s", logs)
	}
}

func TestProxyDecryptsLegacyHostWithoutGatewayHandler(t *testing.T) {
	h := newGatewayHarness(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":1012}`))
	})
	req := gatewayRequest(t, SushiroHost, "/wechat/api/2.0/getStoreById?storeId=1012")
	req.Header.Set("Authorization", "legacy-query")
	bodies := h.roundTrip(SushiroHost, req)
	if bodies[0] != `{"id":1012}` {
		t.Fatalf("body = %q", bodies[0])
	}
	h.noMoreCalls()
	h.ps.tokens.Lock()
	queryAuth := h.ps.tokens.QueryAuth
	h.ps.tokens.Unlock()
	if queryAuth != "legacy-query" {
		t.Fatalf("legacy host should still feed ticket credentials, QueryAuth = %q", queryAuth)
	}
	// The legacy host keeps its original Accept-Encoding: only gateway read endpoints are rewritten.
	h.upMu.Lock()
	seen := h.upSeen[0].Header.Get("Accept-Encoding")
	h.upMu.Unlock()
	if seen != "br, gzip, deflate" {
		t.Fatalf("legacy upstream Accept-Encoding = %q", seen)
	}
}

func TestGatewayHandlerSeesErrorStatusAndSurvivesPanics(t *testing.T) {
	h := newGatewayHarness(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
	})
	bodies := h.roundTrip(SushiroGatewayHost, gatewayRequest(t, SushiroGatewayHost, SushiroGatewayStoresPath+"?latitude=1&longitude=1&numresults=10000"))
	if !strings.Contains(bodies[0], "unauthorized") {
		t.Fatalf("body = %q", bodies[0])
	}
	if call := h.nextCall(); call.status != http.StatusUnauthorized || call.path != SushiroGatewayStoresPath {
		t.Fatalf("call = %+v", call)
	}

	panicked := false
	h.ps.onGateway = func(*http.Request, int, []byte) {
		panicked = true
		panic("handler bug")
	}
	// A broken handler must not take the tunnel down: both requests still get answers.
	bodies = h.roundTrip(SushiroGatewayHost,
		gatewayRequest(t, SushiroGatewayHost, SushiroGatewayStorePath+"?storeId=1"),
		gatewayRequest(t, SushiroGatewayHost, SushiroGatewayStorePath+"?storeId=2"),
	)
	if len(bodies) != 2 || !panicked {
		t.Fatalf("bodies = %q panicked=%v", bodies, panicked)
	}
}

func TestDecodeGatewayBody(t *testing.T) {
	plain := []byte(`{"id":1}`)
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, _ = zw.Write(plain)
	_ = zw.Close()
	var zl bytes.Buffer
	zlw := zlib.NewWriter(&zl)
	_, _ = zlw.Write(plain)
	_ = zlw.Close()
	cases := []struct {
		encoding string
		body     []byte
		want     []byte
	}{
		{"", plain, plain},
		{"identity", plain, plain},
		{"gzip", gz.Bytes(), plain},
		{"deflate", zl.Bytes(), plain},
		{"br", []byte{0x1b, 0x02}, nil},
		{"gzip", []byte("not gzip"), nil},
	}
	for _, c := range cases {
		if got := decodeGatewayBody(c.encoding, c.body); !bytes.Equal(got, c.want) {
			t.Errorf("decode(%q) = %q, want %q", c.encoding, got, c.want)
		}
	}
}

func TestCaptureGatewayStoreIDAcceptsOnlyNumbers(t *testing.T) {
	tokens := core.NewCapturedTokens()
	for _, raw := range []string{"?storeId=1012", "?storeId=1012", "?storeId=abc", "?storeId=", "?storeId=3006", "?storeId=12345678901234"} {
		req := httptest.NewRequest(http.MethodGet, "https://sapi.sushiro.com.cn/gateway/wechat/api/2.0/getStoreById"+raw, nil)
		captureGatewayStoreID(tokens, req)
	}
	if got := strings.Join(tokens.StoreIDs, ","); got != "1012,3006" {
		t.Fatalf("store ids = %s", got)
	}
	captureGatewayStoreID(nil, httptest.NewRequest(http.MethodGet, "https://sapi.sushiro.com.cn/?storeId=1", nil))
}

func TestCaptureProxyOptionDefaults(t *testing.T) {
	if o := DesktopCaptureOptions(); o.ListenHost != "127.0.0.1" || !o.PatchRequests || o.OnGatewayExchange != nil {
		t.Fatalf("desktop options = %+v", o)
	}
	if o := MobileCaptureOptions(); o.ListenHost != "0.0.0.0" || o.PatchRequests || o.OnGatewayExchange != nil {
		t.Fatalf("mobile options = %+v", o)
	}
	called := false
	options := ProxyOptions{ListenHost: "127.0.0.1", OnGatewayExchange: func(*http.Request, int, []byte) { called = true }}
	ps, err := StartProxyWithOptions(tls.Certificate{}, nil, core.NewCapturedTokens(), options)
	if err != nil {
		t.Fatal(err)
	}
	defer ps.Close()
	if ps.onGateway == nil {
		t.Fatal("OnGatewayExchange was not wired into the server")
	}
	ps.onGateway(nil, 0, nil)
	if !called {
		t.Fatal("wired handler is not the injected one")
	}
}

// newTestCA makes an in-memory CA so tests never touch the user's ~/.sushiro-proxy.
func newTestCA(t *testing.T) (tls.Certificate, *rsa.PrivateKey, *x509.CertPool) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Proxy CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, key, pool
}
