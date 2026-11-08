package device

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIoTBehaviorGuard(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "iot_guard.json")

	guard := NewIoTBehaviorGuard(storePath)
	status := guard.GetStatus()
	if !status.Enabled {
		t.Fatalf("expected enabled by default")
	}

	// Test device classification
	cat, vendor := guard.ClassifyDevice("xiaomi-camera-hallway", "64:90:C1:22:33:44")
	if cat != CategoryCamera || vendor != "智能摄像头" {
		t.Fatalf("classification error: %v, %v", cat, vendor)
	}

	// Test Sync Device
	guard.SyncDevice("10.0.0.199", "AA:BB:CC:DD:EE:FF", "bedroom-tv-box", 1024*1024)
	st := guard.GetStatus()
	if st.TotalMonitoredDevices < 5 {
		t.Fatalf("expected at least 5 devices, got %d", st.TotalMonitoredDevices)
	}

	// Test InspectAndClip
	// 500 KB/s = 4,000,000 bps > limit (384Kbps = 48,000 Bps)
	clipped, allowed := guard.InspectAndClip("AA:BB:CC:DD:EE:FF", 500*1024)
	if !clipped {
		t.Fatalf("expected clipped")
	}
	if allowed >= 500*1024 {
		t.Fatalf("expected allowed < 500KB/s, got %d", allowed)
	}

	// Test SetDeviceSandbox
	err := guard.SetDeviceSandbox("AA:BB:CC:DD:EE:FF", false, 0)
	if err != nil {
		t.Fatalf("set sandbox failed: %v", err)
	}

	// After disabling sandbox, should not clip
	clipped2, _ := guard.InspectAndClip("AA:BB:CC:DD:EE:FF", 500*1024)
	if clipped2 {
		t.Fatalf("expected not clipped after un-sandboxing")
	}

	// Test Toggle
	tog := guard.Toggle(false)
	if tog != false {
		t.Fatalf("expected false")
	}

	// Verify persistence
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		t.Fatalf("expected persisted file to exist")
	}
}
