package app

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMobileAuthGuideContainsCAAndProxyInstructions(t *testing.T) {
	rr := httptest.NewRecorder()

	writeMobileAuthGuide(rr, mobileAuthGuideData{
		Hosts:     []string{"192.168.1.20"},
		ProxyPort: 8080,
		CAURL:     "/mobile-auth/token/ca.crt",
	})

	body := rr.Body.String()
	for _, want := range []string{"/mobile-auth/token/ca.crt", "192.168.1.20:8080", "关闭手机 Wi-Fi 代理"} {
		if !strings.Contains(body, want) {
			t.Fatalf("guide page missing %q: %s", want, body)
		}
	}
	for _, removed := range []string{"我已装好证书，验证一下", "拿通行证", "第 4 步"} {
		if strings.Contains(body, removed) {
			t.Fatalf("guide references missing control %q", removed)
		}
	}
}

func TestMobileAuthRequiresUsableNetworkAddress(t *testing.T) {
	if hosts := usableMobileAuthHosts([]string{"127.0.0.1", "0.0.0.0", "169.254.1.2", "::1", "invalid"}); len(hosts) != 0 {
		t.Fatalf("unreachable phone addresses accepted: %v", hosts)
	}
	hosts := usableMobileAuthHosts([]string{"192.168.1.20", "10.0.0.5"})
	urls := mobileAuthGuideURLs(hosts, 1234, "test")
	if len(urls) != 2 || !strings.Contains(urls[1], "10.0.0.5:1234") {
		t.Fatalf("alternate addresses lost: %v", urls)
	}
}

func TestMobileAuthStatusOffersQRCodeForEachAddress(t *testing.T) {
	m := &mobileAuthCaptureManager{hosts: []string{"192.168.1.20", "10.0.0.5"}, urls: []string{"http://192.168.1.20:1234/mobile-auth/test", "http://10.0.0.5:1234/mobile-auth/test"}}
	addresses := m.status()["addresses"].([]map[string]string)
	if len(addresses) != 2 || addresses[1]["host"] != "10.0.0.5" || !strings.Contains(addresses[1]["qr_svg"], "<svg") {
		t.Fatalf("missing alternate QR code: %v", addresses)
	}
}
