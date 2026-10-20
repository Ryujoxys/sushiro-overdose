package app

import (
	"strings"
	"testing"
)

func TestNetworkDiagnosticsListBothHostsWithoutOverclaiming(t *testing.T) {
	hosts := diagnosticNetworkHosts()
	if len(hosts) != 2 || hosts[0].Host != liveGatewayHost || hosts[1].Host != "crm-cn-prd.sushiro.com.cn" {
		t.Fatalf("hosts = %+v", hosts)
	}
	hosts[0].Reachable, hosts[0].LatencyMS = true, 12
	hosts[1].Reachable, hosts[1].LatencyMS = true, 15
	var out strings.Builder
	printNetworkDiagnostics(&out, hosts)
	text := out.String()
	for _, want := range []string{"sapi.sushiro.com.cn:443 可达", "crm-cn-prd.sushiro.com.cn:443 可达", "能连上不代表实时数据能用", "需要在应用里连接"} {
		if !strings.Contains(text, want) {
			t.Fatalf("diagnostics text missing %q:\n%s", want, text)
		}
	}
	hosts[0].Reachable, hosts[0].Error = false, "DNS: no such host"
	out.Reset()
	printNetworkDiagnostics(&out, hosts)
	if !strings.Contains(out.String(), "sapi.sushiro.com.cn:443 不可达 (DNS: no such host)") {
		t.Fatalf("unreachable host: %s", out.String())
	}
}

func TestSystemProxyMentionsPort(t *testing.T) {
	cases := []struct {
		name    string
		summary []string
		port    int
		want    bool
	}{
		{
			name:    "windows proxy server",
			summary: []string{"ProxyEnable=1", "ProxyServer=127.0.0.1:8083"},
			port:    8083,
			want:    true,
		},
		{
			name:    "darwin compact service output",
			summary: []string{"Wi-Fi HTTP[Enabled: Yes; Server: 127.0.0.1; Port: 8084]"},
			port:    8084,
			want:    true,
		},
		{
			name:    "windows pac auto config",
			summary: []string{"AutoConfigURL=http://127.0.0.1:8081/proxy.pac?proxy=8085"},
			port:    8085,
			want:    true,
		},
		{
			name:    "different port",
			summary: []string{"ProxyServer=127.0.0.1:7897"},
			port:    8080,
			want:    false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := systemProxyMentionsPort(DiagnosticSystemProxy{Available: true, Summary: tc.summary}, tc.port)
			if got != tc.want {
				t.Fatalf("systemProxyMentionsPort() = %t, want %t", got, tc.want)
			}
		})
	}
}
