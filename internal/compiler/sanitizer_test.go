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
