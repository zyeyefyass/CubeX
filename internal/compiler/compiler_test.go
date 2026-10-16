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
