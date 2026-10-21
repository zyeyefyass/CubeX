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
}
