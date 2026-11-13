package compiler

import (
	"testing"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestSanitizeIR(t *testing.T) {
	ir := &schema.ConfigIR{
		Rules: []schema.Rule{
			{Type: "domain", Value: "google.com", Target: "Google"},
			{Type: "domain", Value: "bad domain with space.com", Target: "Google"},
			{Type: "ip_cidr", Value: "1.2.3.4/24", Target: "direct"},
			{Type: "ip_cidr", Value: "999.999.999.999/24", Target: "direct"},
			{Type: "port", Value: "443", Target: "direct"},
			{Type: "port", Value: "999999", Target: "direct"},
			{Type: "port", Value: "100-200", Target: "direct"},
			{Type: "port", Value: "500-100", Target: "direct"},
			{Type: "domain_suffix", Value: "  youtube.com  ", Target: "YouTube"},
			{Type: "domain", Value: "", Target: "YouTube"},
		},
		Proxies: []schema.ProxyNode{
			{Name: "Node-1", Server: "1.1.1.1", Port: 443, Type: "shadowsocks"},
			{Name: "Node-1", Server: "2.2.2.2", Port: 443, Type: "shadowsocks"}, // Duplicate name
			{Name: "Node-Bad-Port", Server: "3.3.3.3", Port: -1, Type: "shadowsocks"},
			{Name: "Node-Empty-Server", Server: "", Port: 443, Type: "shadowsocks"},
		},
		DNS: schema.DNSConfig{
			DirectDNS:   []string{"127.0.0.1", "127.0.0.53:53", "localhost", "223.5.5.5"},
			RemoteDNS:   []string{"::1", "0.0.0.0", "tls://1.1.1.1"},
			FallbackDNS: []string{"127.0.0.1:53"},
		},
	}

	SanitizeIR(ir)

	// Validate Rules
	if len(ir.Rules) != 5 {
		t.Fatalf("Expected 5 valid rules, got %d: %+v", len(ir.Rules), ir.Rules)
	}

	// Validate Proxies
	if len(ir.Proxies) != 2 {
		t.Fatalf("Expected 2 valid proxies, got %d", len(ir.Proxies))
	}
	if ir.Proxies[1].Name != "Node-1 2" {
		t.Errorf("Expected deduplicated name 'Node-1 2', got %q", ir.Proxies[1].Name)
	}

	// Validate DNS Loopback Guard
	if len(ir.DNS.DirectDNS) != 1 || ir.DNS.DirectDNS[0] != "223.5.5.5" {
		t.Errorf("Expected DirectDNS sanitized to [223.5.5.5], got %+v", ir.DNS.DirectDNS)
	}
	if len(ir.DNS.RemoteDNS) != 1 || ir.DNS.RemoteDNS[0] != "tls://1.1.1.1" {
		t.Errorf("Expected RemoteDNS sanitized to [tls://1.1.1.1], got %+v", ir.DNS.RemoteDNS)
	}
	if len(ir.DNS.FallbackDNS) == 0 || ir.DNS.FallbackDNS[0] == "127.0.0.1:53" {
		t.Errorf("Expected FallbackDNS loopback stripped and defaulted, got %+v", ir.DNS.FallbackDNS)
	}
}

func TestIsLoopbackOrDeadlockDNS(t *testing.T) {
	deadlocks := []string{
		"127.0.0.1",
		"127.0.0.53",
		"127.0.0.1:53",
		"udp://127.0.0.1:53",
		"::1",
		"[::1]:53",
		"0.0.0.0",
		"0.0.0.0:53",
		"localhost",
		"localhost:53",
		"https://127.0.0.1/dns-query",
		"",
	}
	for _, d := range deadlocks {
		if !IsLoopbackOrDeadlockDNS(d) {
			t.Errorf("Expected IsLoopbackOrDeadlockDNS(%q) == true, got false", d)
		}
	}

	valids := []string{
		"223.5.5.5",
		"119.29.29.29:53",
		"tls://1.1.1.1",
		"https://cloudflare-dns.com/dns-query",
		"8.8.8.8",
	}
	for _, v := range valids {
		if IsLoopbackOrDeadlockDNS(v) {
			t.Errorf("Expected IsLoopbackOrDeadlockDNS(%q) == false, got true", v)
		}
	}
}

func TestSanitizeAndDeduplicateNodes(t *testing.T) {
	rawNodes := []schema.ProxyNode{
		{Name: "剩余流量：500GB", Server: "1.1.1.1", Port: 443, Type: "shadowsocks"},
		{Name: "到期时间：2027-01-01", Server: "1.1.1.1", Port: 443, Type: "shadowsocks"},
		{Name: "官网：https://airport.com", Server: "1.1.1.1", Port: 443, Type: "shadowsocks"},
		{Name: "香港 01 高速专线", Server: "hk01.node.com", Port: 443, Type: "shadowsocks"},
		{Name: "香港 01 高速专线", Server: "hk02.node.com", Port: 443, Type: "shadowsocks"}, // Duplicate
		{Name: "本地伪节点", Server: "127.0.0.1", Port: 1080, Type: "socks5"},
		{Name: "日本 01 优质节点", Server: "jp01.node.com", Port: 443, Type: "shadowsocks"},
	}

	clean := SanitizeAndDeduplicateNodes(rawNodes)
	if len(clean) != 3 {
		t.Fatalf("expected 3 clean nodes after filtering pseudo/duplicate nodes, got %d", len(clean))
	}
	if clean[0].Name != "香港 01 高速专线" {
		t.Errorf("unexpected first node: %s", clean[0].Name)
	}
	if clean[1].Name != "香港 01 高速专线 2" {
		t.Errorf("expected deduplicated name '香港 01 高速专线 2', got %s", clean[1].Name)
	}
	if clean[2].Name != "日本 01 优质节点" {
		t.Errorf("unexpected third node: %s", clean[2].Name)
	}
}

func TestReconcilePolicyGroups(t *testing.T) {
	validNodes := []schema.ProxyNode{
		{Name: "香港 01", Server: "hk01.com", Port: 443, Type: "shadowsocks"},
		{Name: "日本 01", Server: "jp01.com", Port: 443, Type: "shadowsocks"},
	}

	groups := []schema.StrategyGroup{
		{
			Name:    "PROXY",
			Type:    "select",
			Proxies: []string{"香港 01", "香港 01", "剩余流量：500GB", "日本 01", "不存在的失效节点"},
			Now:     "剩余流量：500GB",
		},
		{
			Name:    "EMPTY_GROUP",
			Type:    "select",
			Proxies: []string{"剩余流量：500GB", "到期时间：2027-01-01"},
			Now:     "到期时间：2027-01-01",
		},
		{
			Name:    "NESTED_GROUP",
			Type:    "select",
			Proxies: []string{"PROXY", "DIRECT"},
			Now:     "PROXY",
		},
	}

	reconciled := ReconcilePolicyGroups(groups, validNodes)
	if len(reconciled) != 3 {
		t.Fatalf("expected 3 reconciled groups, got %d", len(reconciled))
	}

	// 1. PROXY 策略组应剔除重复节点与悬空脏节点
	if len(reconciled[0].Proxies) != 2 || reconciled[0].Proxies[0] != "香港 01" || reconciled[0].Proxies[1] != "日本 01" {
		t.Errorf("unexpected PROXY proxies: %+v", reconciled[0].Proxies)
	}
	if reconciled[0].Now != "香港 01" {
		t.Errorf("expected Now healed to '香港 01', got: %s", reconciled[0].Now)
	}

	// 2. EMPTY_GROUP 策略组全部为脏节点时，必须自动注入 DIRECT 兜底，严防内核崩溃
	if len(reconciled[1].Proxies) != 1 || reconciled[1].Proxies[0] != "DIRECT" {
		t.Errorf("expected EMPTY_GROUP to fallback to [DIRECT], got: %+v", reconciled[1].Proxies)
	}
	if reconciled[1].Now != "DIRECT" {
		t.Errorf("expected EMPTY_GROUP Now to fallback to DIRECT, got: %s", reconciled[1].Now)
	}

	// 3. NESTED_GROUP 嵌套引用 PROXY 与 DIRECT 应正常保留
	if len(reconciled[2].Proxies) != 2 || reconciled[2].Proxies[0] != "PROXY" || reconciled[2].Proxies[1] != "DIRECT" {
		t.Errorf("unexpected NESTED_GROUP proxies: %+v", reconciled[2].Proxies)
	}
}
