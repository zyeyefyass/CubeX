package compiler

import (
	"encoding/json"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/schema"
	"gopkg.in/yaml.v3"
)

func TestLegacyLinuxKernel44Compatibility(t *testing.T) {
	t.Setenv("CUBEX_KERNEL_RELEASE", "4.4.60")
	t.Setenv("CUBEX_DISABLE_IPV6_DETECT", "1")

	legacy, rel := IsLegacyLinuxKernel()
	if !legacy || rel != "4.4.60" {
		t.Fatalf("expected legacy=true rel=4.4.60, got legacy=%v rel=%q", legacy, rel)
	}

	ir := schema.NewDefaultIR()
	ir.Network.IPv6 = true
	ir.Network.TunStack = "mixed"

	// 1. Verify sing-box 4.4 compat config
	sbBytes, err := BuildSingBox(ir)
	if err != nil {
		t.Fatalf("BuildSingBox failed: %v", err)
	}
	var sbCfg map[string]any
	if err := json.Unmarshal(sbBytes, &sbCfg); err != nil {
		t.Fatalf("unmarshal singbox json failed: %v", err)
	}
	inbounds, _ := sbCfg["inbounds"].([]any)
	var tunIn map[string]any
	for _, raw := range inbounds {
		if m, ok := raw.(map[string]any); ok && m["type"] == "tun" {
			tunIn = m
			break
		}
	}
	if tunIn == nil {
		t.Fatalf("expected tun inbound in sing-box config (runtime test on non-windows or host)")
	}
	if tunIn["stack"] != "gvisor" {
		t.Fatalf("expected legacy 4.4 sing-box tun stack=gvisor, got %v", tunIn["stack"])
	}
	if tunIn["endpoint_independent_nat"] != false {
		t.Fatalf("expected legacy 4.4 sing-box endpoint_independent_nat=false, got %v", tunIn["endpoint_independent_nat"])
	}
	if tunIn["auto_redirect"] != false {
		t.Fatalf("expected sing-box auto_redirect=false, got %v", tunIn["auto_redirect"])
	}
	if addrs, ok := tunIn["route_address"].([]any); !ok || len(addrs) != 1 || addrs[0] != "198.18.0.0/15" {
		t.Fatalf("expected IPv4-only route_address [198.18.0.0/15] when /proc/net/if_inet6 disabled on 4.4, got %v", tunIn["route_address"])
	}

	// 2. Verify mihomo 4.4 compat config
	mhBytes, err := BuildMihomo(ir)
	if err != nil {
		t.Fatalf("BuildMihomo failed: %v", err)
	}
	var mhCfg map[string]any
	if err := yaml.Unmarshal(mhBytes, &mhCfg); err != nil {
		t.Fatalf("unmarshal mihomo yaml failed: %v", err)
	}
	tunYaml, _ := mhCfg["tun"].(map[string]any)
	if tunYaml == nil {
		t.Fatalf("expected tun block in mihomo config")
	}
	if tunYaml["stack"] != "gvisor" {
		t.Fatalf("expected legacy 4.4 mihomo tun stack=gvisor, got %v", tunYaml["stack"])
	}
	if tunYaml["gso"] != false {
		t.Fatalf("expected legacy 4.4 mihomo tun gso=false, got %v", tunYaml["gso"])
	}
}

func TestModernLinuxKernelPreservesMixedAndGSO(t *testing.T) {
	t.Setenv("CUBEX_KERNEL_RELEASE", "6.12.112-amd64")

	legacy, _ := IsLegacyLinuxKernel()
	if legacy {
		t.Fatalf("expected 6.12.112-amd64 NOT to be flagged as legacy kernel")
	}

	ir := schema.NewDefaultIR()
	ir.Network.TunStack = "mixed"

	sbBytes, err := BuildSingBox(ir)
	if err != nil {
		t.Fatalf("BuildSingBox failed: %v", err)
	}
	var sbCfg map[string]any
	_ = json.Unmarshal(sbBytes, &sbCfg)
	inbounds, _ := sbCfg["inbounds"].([]any)
	for _, raw := range inbounds {
		if m, ok := raw.(map[string]any); ok && m["type"] == "tun" {
			if m["stack"] != "mixed" {
				t.Fatalf("expected modern kernel sing-box stack=mixed, got %v", m["stack"])
			}
			if m["endpoint_independent_nat"] != true {
				t.Fatalf("expected modern kernel sing-box endpoint_independent_nat=true, got %v", m["endpoint_independent_nat"])
			}
		}
	}
}
