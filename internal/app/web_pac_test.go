package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleProxyPACOnlyProxiesSushiroHost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/proxy.pac?proxy=8082", nil)
	rr := httptest.NewRecorder()

	handleProxyPAC(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `if (host === "crm-cn-prd.sushiro.com.cn" || host === "sapi.sushiro.com.cn") {`) {
		t.Fatalf("PAC does not match exactly the two sushiro API hosts: %s", body)
	}
	if strings.Contains(body, "dnsDomainIs") || strings.Contains(body, "shExpMatch") || strings.Contains(body, `".sushiro.com.cn"`) {
		t.Fatalf("PAC must not proxy the whole sushiro domain: %s", body)
	}
	if !strings.Contains(body, `host = host.split(":")[0]`) {
		t.Fatalf("PAC does not normalize hosts that include a port: %s", body)
	}
	if !strings.Contains(body, `PROXY 127.0.0.1:8082`) {
		t.Fatalf("PAC does not point to proxy port: %s", body)
	}
	if !strings.Contains(body, `return "DIRECT"`) {
		t.Fatalf("PAC does not direct non-sushiro hosts: %s", body)
	}
}

func TestHandleProxyPACRejectsInvalidPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/proxy.pac?proxy=70000", nil)
	rr := httptest.NewRecorder()

	handleProxyPAC(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}
