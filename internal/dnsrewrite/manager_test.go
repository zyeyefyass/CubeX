package dnsrewrite

import (
	"os"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestDNSRewriteManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_dns_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewManager(tempDir)
	if !mgr.IsEnabled() {
		t.Errorf("Expected default Enabled=true, got false")
	}

	rules := mgr.GetAllRules()
	if len(rules) < 3 {
		t.Errorf("Expected at least 3 default rules, got %d", len(rules))
	}

	// Duplicate domain test on pre-existing default rule nas.lan
	_, err = mgr.AddRule("nas.lan", schema.DNSRewriteTypeA, "10.0.0.101", "")
	if err == nil {
		t.Errorf("Expected duplicate error for existing nas.lan, got nil")
	}

	// Add an A-record rule
	added, err := mgr.AddRule("pve.lan", schema.DNSRewriteTypeA, "10.0.0.200", "Home PVE")
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}
	if added.Domain != "pve.lan" || added.Target != "10.0.0.200" || added.Type != schema.DNSRewriteTypeA {
		t.Errorf("Unexpected rule content: %+v", added)
	}

	// Duplicate domain test
	_, err = mgr.AddRule("pve.lan", schema.DNSRewriteTypeA, "10.0.0.201", "")
	if err == nil {
		t.Errorf("Expected duplicate error, got nil")
	}

	// Update rule
	err = mgr.UpdateRule(added.ID, "pve.lan", schema.DNSRewriteTypeA, "10.0.0.205", "Updated PVE", true)
	if err != nil {
		t.Fatalf("Failed to update rule: %v", err)
	}

	// Verify persistence across reload
	mgr2 := NewManager(tempDir)
	active := mgr2.GetActiveRules()
	found := false
	for _, r := range active {
		if r.Domain == "pve.lan" && r.Target == "10.0.0.205" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Did not find updated rule in reloaded manager")
	}

	// Delete rule
	err = mgr2.DeleteRule(added.ID)
	if err != nil {
		t.Fatalf("Failed to delete rule: %v", err)
	}
	if len(mgr2.GetAllRules()) != len(rules) {
		t.Errorf("Expected %d rules after deletion, got %d", len(rules), len(mgr2.GetAllRules()))
	}
}
