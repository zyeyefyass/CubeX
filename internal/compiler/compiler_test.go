package compiler_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/compiler"
	"github.com/zyeyefyass/CubeX/internal/schema"
	"gopkg.in/yaml.v3"
)

func TestBuildMihomo(t *testing.T) {
	ir := schema.NewDefaultIR()
	ir.DevicePolicies = append(ir.DevicePolicies, schema.DevicePolicy{
		Name: "Test-NAS",
		IP:   "192.168.1.50",
		Mode: "bypass",
	})
	ir.DevicePolicies = append(ir.DevicePolicies, schema.DevicePolicy{
		Name:      "Test-PS5",
		IP:        "192.168.1.99",
		Mode:      "node",
		BoundNode: "JP-Gaming-01",
	})

	data, err := compiler.BuildMihomo(ir)
	if err != nil {
		t.Fatalf("BuildMihomo failed: %v", err)
	}

	// Verify valid YAML
	var rawMap map[string]any
	if err := yaml.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("Generated Mihomo YAML is invalid: %v", err)
	}

	content := string(data)

	// Verify ports
	if !strings.Contains(content, "tproxy-port: 7893") {
		t.Errorf("Expected tproxy-port: 7893 in YAML")
	}
	if !strings.Contains(content, "device: cubex-tun") {
		t.Errorf("Expected device: cubex-tun in YAML")
	}

	// Verify top-priority device rules
	if !strings.Contains(content, "SRC-IP-CIDR,192.168.1.50/32,DIRECT") {
		t.Errorf("Expected bypass rule for NAS in YAML")
	}
	if !strings.Contains(content, "SRC-IP-CIDR,192.168.1.99/32,JP-Gaming-01") {
		t.Errorf("Expected bound node rule for PS5 in YAML")
	}
}

func TestBuildSingBox(t *testing.T) {
	ir := schema.NewDefaultIR()
	ir.DevicePolicies = append(ir.DevicePolicies, schema.DevicePolicy{
		Name: "Test-NAS",
		IP:   "192.168.1.50",
		Mode: "bypass",
	})
	ir.DevicePolicies = append(ir.DevicePolicies, schema.DevicePolicy{
		Name:      "Test-PS5",
		IP:        "192.168.1.99",
		Mode:      "node",
		BoundNode: "JP-Gaming-01",
	})

	data, err := compiler.BuildSingBox(ir)
	if err != nil {
		t.Fatalf("BuildSingBox failed: %v", err)
	}

	// Verify valid JSON
	var rawMap map[string]any
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("Generated sing-box JSON is invalid: %v", err)
	}

	content := string(data)

	// Verify inbounds
	if !strings.Contains(content, "cubex-tun") {
		t.Errorf("Expected cubex-tun in sing-box config")
	}
	if !strings.Contains(content, "7893") {
		t.Errorf("Expected 7893 tproxy port in sing-box config")
	}

	// Verify device rules
	if !strings.Contains(content, "192.168.1.50/32") {
		t.Errorf("Expected NAS CIDR in route rules")
	}
	if !strings.Contains(content, "JP-Gaming-01") {
		t.Errorf("Expected PS5 bound node in route rules")
	}
}

func TestSingboxDNSLeakPrevention(t *testing.T) {
	ir := schema.NewDefaultIR()
	data, err := compiler.BuildSingBox(ir)
	if err != nil {
		t.Fatalf("BuildSingBox failed: %v", err)
	}

	var sbCfg map[string]any
	if err := json.Unmarshal(data, &sbCfg); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	dnsSection, ok := sbCfg["dns"].(map[string]any)
	if !ok {
		t.Fatalf("dns section missing in sing-box config")
	}

	// 1. Verify Fake-IP server in servers list (sing-box 1.14+ compliant)
	servers, _ := dnsSection["servers"].([]any)
	hasDirectServer := false
	hasFakeIPServer := false
	hasRemoteServer := false

	for _, s := range servers {
		sMap, _ := s.(map[string]any)
		tag := sMap["tag"]
		if tag == "dns-direct" {
			hasDirectServer = true
		}
		if tag == "dns-fakeip" && sMap["type"] == "fakeip" {
			hasFakeIPServer = true
			if sMap["inet4_range"] != "198.18.0.0/15" {
				t.Errorf("Fake-IP inet4_range expected 198.18.0.0/15, got: %v", sMap["inet4_range"])
			}
		}
		if tag == "dns-remote" {
			hasRemoteServer = true
		}
	}

	if !hasDirectServer || !hasFakeIPServer || !hasRemoteServer {
		t.Errorf("sing-box missing core DNS servers (direct, fakeip, remote)")
	}

	// 3. Verify final fallback server is encrypted dns-remote (0 plaintext fallback)
	if dnsSection["final"] != "dns-remote" {
		t.Errorf("sing-box final DNS fallback must be dns-remote (encrypted proxy), got: %v", dnsSection["final"])
	}

	// 4. Verify rules order: Type 65 block -> domestic direct -> A/AAAA fakeip
	rules, _ := dnsSection["rules"].([]any)
	hasType65Rule := false
	hasDomesticRule := false
	hasFakeIPRule := false

	for _, r := range rules {
		rMap, _ := r.(map[string]any)
		qTypes, _ := rMap["query_type"].([]any)
		for _, qt := range qTypes {
			if qt == "HTTPS" && rMap["action"] == "reject" {
				hasType65Rule = true
			}
			if qt == "A" && rMap["server"] == "dns-fakeip" {
				hasFakeIPRule = true
			}
		}
		if rMap["server"] == "dns-direct" {
			hasDomesticRule = true
		}
	}

	if !hasType65Rule {
		t.Errorf("sing-box missing HTTPS Type 65 block rule")
	}
	if !hasDomesticRule {
		t.Errorf("sing-box missing domestic direct DNS rule")
	}
	if !hasFakeIPRule {
		t.Errorf("sing-box missing fake-ip rule for foreign domains")
	}
}

func TestMihomoDNSLeakPrevention(t *testing.T) {
	ir := schema.NewDefaultIR()
	data, err := compiler.BuildMihomo(ir)
	if err != nil {
		t.Fatalf("BuildMihomo failed: %v", err)
	}

	var mCfg map[string]any
	if err := yaml.Unmarshal(data, &mCfg); err != nil {
		t.Fatalf("Invalid YAML: %v", err)
	}

	dnsSection, ok := mCfg["dns"].(map[string]any)
	if !ok {
		t.Fatalf("dns section missing in Mihomo config")
	}

	// 1. Verify Fake-IP enhanced mode
	if dnsSection["enhanced-mode"] != "fake-ip" {
		t.Errorf("Mihomo enhanced-mode must be fake-ip, got: %v", dnsSection["enhanced-mode"])
	}

	// 2. Verify fake-ip-filter exists and contains domestic domains
	filters, _ := dnsSection["fake-ip-filter"].([]any)
	hasCN := false
	for _, f := range filters {
		if f == "geosite:cn" || f == "+.cn" {
			hasCN = true
			break
		}
	}
	if !hasCN {
		t.Errorf("Mihomo fake-ip-filter must include domestic domains to prevent fake-ip leakage on direct sites")
	}

	// 3. Verify proxy-server-nameserver and direct-nameserver (prevents circular deadlocks and direct leaks)
	proxyDNS, _ := dnsSection["proxy-server-nameserver"].([]any)
	if len(proxyDNS) == 0 {
		t.Errorf("Mihomo missing proxy-server-nameserver")
	}
	directDNS, _ := dnsSection["direct-nameserver"].([]any)
	if len(directDNS) == 0 {
		t.Errorf("Mihomo missing direct-nameserver")
	}

	// 4. Verify disable-qtype-65 on DoH endpoints
	nameservers, _ := dnsSection["nameserver"].([]any)
	for _, ns := range nameservers {
		nsStr, _ := ns.(string)
		if strings.HasPrefix(nsStr, "https://") && !strings.Contains(nsStr, "disable-qtype-65=true") {
			t.Errorf("Mihomo remote DoH must have #disable-qtype-65=true: %s", nsStr)
		}
	}

	// 5. Verify nameserver-policy for domestic domains
	nsPolicy, _ := dnsSection["nameserver-policy"].(map[string]any)
	if nsPolicy["geosite:cn,private"] == nil {
		t.Errorf("Mihomo missing nameserver-policy for geosite:cn,private")
	}

	// 6. Verify respect-rules
	if dnsSection["respect-rules"] != true {
		t.Errorf("Mihomo respect-rules must be true")
	}
}
