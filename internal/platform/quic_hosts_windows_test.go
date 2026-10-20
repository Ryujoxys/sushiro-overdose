//go:build windows

package platform

import (
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/Ryujoxys/sushiro-overdose/internal/proxy"
)

func TestQUICBlockResolvesExactlyTheInterceptHosts(t *testing.T) {
	asked := []string{}
	lookup := func(host string) ([]net.IP, error) {
		asked = append(asked, host)
		switch host {
		case proxy.SushiroHost:
			return []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2")}, nil
		case proxy.SushiroGatewayHost:
			return []net.IP{net.ParseIP("10.0.0.2"), net.ParseIP("10.0.0.3")}, nil
		}
		return nil, errors.New("unexpected host")
	}
	got := resolveHostIPs(proxy.SushiroInterceptHosts(), lookup)
	if strings.Join(asked, ",") != "crm-cn-prd.sushiro.com.cn,sapi.sushiro.com.cn" {
		t.Fatalf("looked up %v", asked)
	}
	if strings.Join(got, ",") != "10.0.0.1,10.0.0.2,10.0.0.3" {
		t.Fatalf("ips = %v", got)
	}
	// One host failing to resolve still blocks the other.
	only := resolveHostIPs([]string{"missing.invalid", proxy.SushiroGatewayHost}, lookup)
	if strings.Join(only, ",") != "10.0.0.2,10.0.0.3" {
		t.Fatalf("partial ips = %v", only)
	}
}
