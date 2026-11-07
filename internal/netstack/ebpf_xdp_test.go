package netstack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestXDPBypassManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_xdp_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := GetXDPBypassManager(filepath.Join(tempDir, "data"))

	status := mgr.GetStatus()
	if !status.Enabled {
		t.Errorf("expected enabled to be true")
	}
	if status.Mode != XDPModeNative {
		t.Errorf("expected native mode, got %s", status.Mode)
	}

	// Test RecordBypass
	initPkts := status.PacketsBypassed
	mgr.RecordBypass(100, 150000)
	newStatus := mgr.GetStatus()
	if newStatus.PacketsBypassed != initPkts+100 {
		t.Errorf("expected bypassed pkts to increment by 100")
	}

	// Test SetEnabled
	if err := mgr.SetEnabled(false); err != nil {
		t.Fatalf("failed to disable XDP: %v", err)
	}
	if mgr.GetStatus().Enabled {
		t.Errorf("expected XDP to be disabled")
	}

	// Re-enable
	if err := mgr.SetEnabled(true, XDPModeGeneric); err != nil {
		t.Fatalf("failed to enable XDP generic: %v", err)
	}
	if mgr.GetStatus().Mode != XDPModeGeneric {
		t.Errorf("expected mode generic, got %s", mgr.GetStatus().Mode)
	}

	// Test SyncDirectRoutes
	mgr.SyncDirectRoutes([]string{"10.0.0.0/8", "192.168.1.0/24", "180.101.49.12/32"})
	if mgr.GetStatus().DirectRoutesCount != 3 {
		t.Errorf("expected direct routes count to be 3, got %d", mgr.GetStatus().DirectRoutesCount)
	}
}
