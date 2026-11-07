package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLANGuardianManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_guardian_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := GetLANGuardianManager(filepath.Join(tempDir, "data"))

	status := mgr.GetStatus()
	if !status.ARPDefenseActive {
		t.Errorf("expected ARPDefenseActive to be true")
	}
	if status.TotalMonitoredIPs <= 0 {
		t.Errorf("expected positive total monitored IPs")
	}

	// Test SetDefenseActive
	if err := mgr.SetDefenseActive(false); err != nil {
		t.Fatalf("failed to disable defense: %v", err)
	}
	if mgr.GetStatus().ARPDefenseActive {
		t.Errorf("expected defense disabled")
	}

	// Re-enable
	_ = mgr.SetDefenseActive(true)

	// Test RunScan
	scanned := mgr.RunScan()
	if scanned.TotalMonitoredIPs == 0 {
		t.Errorf("expected scanned monitored IPs")
	}

	// Test IsolateAttacker
	if err := mgr.IsolateAttacker("D8:32:14:AA:BB:CC"); err != nil {
		t.Fatalf("failed to isolate attacker: %v", err)
	}
}
