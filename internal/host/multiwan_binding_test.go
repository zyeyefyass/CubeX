package host

import (
	"os"
	"testing"
)

func TestMultiWANBindingManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_multiwan_binding_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := newMultiWANBindingManager(tempDir)

	// 1. Initial State
	status := mgr.GetStatus()
	if status.ActiveMode != PolicyLatency {
		t.Errorf("expected initial mode latency, got %s", status.ActiveMode)
	}
	if len(status.Paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(status.Paths))
	}

	// 2. Set Mode
	if err := mgr.SetPolicyMode(PolicyBandwidth); err != nil {
		t.Fatalf("set policy mode failed: %v", err)
	}
	status = mgr.GetStatus()
	if status.ActiveMode != PolicyBandwidth {
		t.Errorf("expected mode bandwidth, got %s", status.ActiveMode)
	}

	// 3. Failover Trigger
	mgr.TriggerFailover("pppoe-out1", "wan2")
	status = mgr.GetStatus()
	if status.PrimaryWAN != "wan2" || status.BackupWAN != "pppoe-out1" {
		t.Errorf("failover failed: primary=%s, backup=%s", status.PrimaryWAN, status.BackupWAN)
	}
	if status.FailoverCount != 1 {
		t.Errorf("expected failover count 1, got %d", status.FailoverCount)
	}

	// 4. Persistence Test
	mgr2 := newMultiWANBindingManager(tempDir)
	status2 := mgr2.GetStatus()
	if status2.ActiveMode != PolicyBandwidth || status2.PrimaryWAN != "wan2" {
		t.Errorf("persisted state mismatch: mode=%s, primary=%s", status2.ActiveMode, status2.PrimaryWAN)
	}
}
