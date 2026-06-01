package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBufferedConnReadsDataAlreadyBufferedAfterConnect(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errCh := make(chan error, 1)
	go func() {
		_, err := client.Write([]byte("CONNECT crm-cn-prd.sushiro.com.cn:443 HTTP/1.1\r\nHost: crm-cn-prd.sushiro.com.cn:443\r\n\r\nHELLO"))
		errCh <- err
	}()

	reader := bufio.NewReader(server)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read CONNECT header: %v", err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}

	got := make([]byte, 5)
	if _, err := io.ReadFull(&bufferedConn{Conn: server, reader: reader}, got); err != nil {
		t.Fatalf("read buffered payload: %v", err)
	}
	if string(got) != "HELLO" {
		t.Fatalf("buffered payload = %q, want HELLO", string(got))
	}
	if err := <-errCh; err != nil {
		t.Fatalf("client write: %v", err)
	}
}

func TestPlainHTTPProxyForwardsAbsoluteFormRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ping" || r.URL.RawQuery != "x=1" {
			t.Fatalf("unexpected upstream URL: %s", r.URL.String())
		}
		if got := r.Header.Get("X-Test"); got != "yes" {
			t.Fatalf("upstream X-Test = %q, want yes", got)
		}
		w.Header().Set("X-Upstream", "ok")
		_, _ = w.Write([]byte("pong"))
	}))
	defer upstream.Close()

	client, server := net.Pipe()
	defer client.Close()

	ps := &proxyServer{transport: http.DefaultTransport.(*http.Transport).Clone()}
	done := make(chan struct{})
	go func() {
		ps.handleConn(server)
		close(done)
	}()

	if _, err := fmt.Fprintf(client, "GET %s/ping?x=1 HTTP/1.1\r\nHost: %s\r\nX-Test: yes\r\nConnection: close\r\n\r\n", upstream.URL, strings.TrimPrefix(upstream.URL, "http://")); err != nil {
		t.Fatalf("write request: %v", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(client), nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if resp.Header.Get("X-Upstream") != "ok" {
		t.Fatalf("missing upstream header")
	}
	if string(body) != "pong" {
		t.Fatalf("body = %q, want pong", string(body))
	}
	<-done
}

func TestProxyUpstreamTransportAllowsHTTP2Upstream(t *testing.T) {
	tr := newProxyUpstreamTransport()
	if !tr.ForceAttemptHTTP2 {
		t.Fatalf("ForceAttemptHTTP2 = false")
	}
	if tr.TLSClientConfig == nil {
		t.Fatalf("TLSClientConfig is nil")
	}
	got := strings.Join(tr.TLSClientConfig.NextProtos, ",")
	if !strings.Contains(got, "h2") || !strings.Contains(got, "http/1.1") {
		t.Fatalf("NextProtos = %q, want h2 and http/1.1", got)
	}
}

func TestRelayResponseNormalizesHTTPVersion(t *testing.T) {
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		Proto:         "HTTP/2.0",
		ProtoMajor:    2,
		ProtoMinor:    0,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader("ok")),
		ContentLength: 2,
	}
	target, err := url.Parse("https://crm-cn-prd.sushiro.com.cn/")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	ps := &proxyServer{}
	if err := ps.relayResponse(&out, resp, http.MethodGet, target); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "HTTP/1.1 200 OK\r\n") {
		t.Fatalf("response status line = %q", strings.SplitN(out.String(), "\r\n", 2)[0])
	}
}

func TestRelayResponseBuffersSushiroResponseWithUnknownLength(t *testing.T) {
	resp := &http.Response{
		StatusCode:       http.StatusOK,
		Status:           "200 OK",
		Proto:            "HTTP/2.0",
		ProtoMajor:       2,
		ProtoMinor:       0,
		Header:           make(http.Header),
		Body:             io.NopCloser(strings.NewReader("stores")),
		ContentLength:    -1,
		TransferEncoding: []string{"chunked"},
	}
	target, err := url.Parse("https://crm-cn-prd.sushiro.com.cn/wechat/api/2.0/stores")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	ps := &proxyServer{}
	if err := ps.relayResponse(&out, resp, http.MethodGet, target); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !strings.Contains(text, "Content-Length: 6\r\n") {
		t.Fatalf("response missing content length:\n%s", text)
	}
	if strings.Contains(strings.ToLower(text), "transfer-encoding: chunked") {
		t.Fatalf("response should not be chunked:\n%s", text)
	}
	if !strings.HasSuffix(text, "\r\n\r\nstores") {
		t.Fatalf("response body mismatch:\n%s", text)
	}
}

func TestRelayResponseStreamsNonAPISushiroResponseWithUnknownLength(t *testing.T) {
	resp := &http.Response{
		StatusCode:       http.StatusOK,
		Status:           "200 OK",
		Proto:            "HTTP/2.0",
		ProtoMajor:       2,
		ProtoMinor:       0,
		Header:           make(http.Header),
		Body:             io.NopCloser(strings.NewReader("asset")),
		ContentLength:    -1,
		TransferEncoding: []string{"chunked"},
	}
	target, err := url.Parse("https://crm-cn-prd.sushiro.com.cn/static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	ps := &proxyServer{}
	if err := ps.relayResponse(&out, resp, http.MethodGet, target); err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(out.String())
	if strings.Contains(text, "content-length: 5\r\n") {
		t.Fatalf("non-API response should not be force-buffered:\n%s", out.String())
	}
	if !strings.Contains(text, "transfer-encoding: chunked\r\n") {
		t.Fatalf("non-API response should keep streaming transfer encoding:\n%s", out.String())
	}
}

func TestRelayResponseRejectsOversizedSushiroAPIResponse(t *testing.T) {
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		Proto:         "HTTP/2.0",
		ProtoMajor:    2,
		ProtoMinor:    0,
		Header:        make(http.Header),
		Body:          io.NopCloser(io.LimitReader(zeroReader{}, int64(maxSushiroBufferedResponseBytes)+1)),
		ContentLength: -1,
	}
	target, err := url.Parse("https://crm-cn-prd.sushiro.com.cn/wechat/api_auth/2.0/reservations")
	if err != nil {
		t.Fatal(err)
	}
	ps := &proxyServer{}
	err = ps.relayResponse(io.Discard, resp, http.MethodGet, target)
	if err == nil {
		t.Fatal("relayResponse returned nil error for oversized Sushiro API response")
	}
	if !strings.Contains(err.Error(), "超过缓冲上限") {
		t.Fatalf("error = %q, want buffer limit message", err.Error())
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

func TestSetForwardRequestBodyUsesNoBodyForEmptyPayload(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.test/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	setForwardRequestBody(req, nil)
	if req.Body != http.NoBody {
		t.Fatalf("empty body = %#v, want http.NoBody", req.Body)
	}
	if req.ContentLength != 0 {
		t.Fatalf("ContentLength = %d, want 0", req.ContentLength)
	}
}

func TestSetForwardRequestBodyPreservesPayload(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.test/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"ok":true}`)
	setForwardRequestBody(req, payload)
	if req.ContentLength != int64(len(payload)) {
		t.Fatalf("ContentLength = %d, want %d", req.ContentLength, len(payload))
	}
	got, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("body = %q, want %q", got, payload)
	}
}

func TestStripQUICAdvertisement(t *testing.T) {
	h := http.Header{}
	h.Set("Alt-Svc", `h3=":443"; ma=86400`)
	h.Set("Alternate-Protocol", "443:quic")
	h.Set("Content-Type", "application/json")
	stripQUICAdvertisement(h)
	if h.Get("Alt-Svc") != "" || h.Get("Alternate-Protocol") != "" {
		t.Fatalf("QUIC advertisement headers should be stripped: %v", h)
	}
	if h.Get("Content-Type") != "application/json" {
		t.Fatalf("non-QUIC headers must be preserved: %v", h)
	}
}
