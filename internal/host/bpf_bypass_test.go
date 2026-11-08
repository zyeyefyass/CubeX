package host

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBpfBypassManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_bpf_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewBpfBypassManager(tempDir)
	status := mgr.GetStatus()
	if !status.KernelHookActive {
		t.Fatal("expected kernel hook to be active")
	}
	if len(status.ActiveRules) == 0 {
		t.Fatal("expected default bypass rules")
	}

	// Test ToggleRule
	firstID := status.ActiveRules[0].ID
	initialEnabled := status.ActiveRules[0].Enabled
	if err := mgr.ToggleRule(firstID); err != nil {
		t.Fatalf("ToggleRule failed: %v", err)
	}
	newStatus := mgr.GetStatus()
	if newStatus.ActiveRules[0].Enabled == initialEnabled {
		t.Fatal("expected rule state to be toggled")
	}

	// Test Optimize
	if err := mgr.Optimize(); err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}
	optStatus := mgr.GetStatus()
	if optStatus.TotalPacketsBypassed <= status.TotalPacketsBypassed {
		t.Fatal("expected TotalPacketsBypassed to increase")
	}

	// Verify persistence
	filePath := filepath.Join(tempDir, "bpf_bypass.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("expected status file to be written")
	}
}
