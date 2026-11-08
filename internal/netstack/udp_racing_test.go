package netstack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUDPRacingManager(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "udp_racing.json")

	mgr := NewUDPRacingManager(storePath)
	status := mgr.GetStatus()
	if !status.Enabled {
		t.Fatalf("expected enabled by default")
	}
	if status.ClampedMSS != 1420 {
		t.Fatalf("expected clamped MSS 1420, got %d", status.ClampedMSS)
	}

	// Test probe
	res := mgr.ProbeDomain("googlevideo.com")
	if res.Domain != "googlevideo.com" {
		t.Fatalf("expected googlevideo.com, got %s", res.Domain)
	}

	// Test MSS clamp change
	newMSS := mgr.SetClampedMSS(1400)
	if newMSS != 1400 {
		t.Fatalf("expected 1400, got %d", newMSS)
	}

	// Test toggle
	toggled := mgr.Toggle(false)
	if toggled != false {
		t.Fatalf("expected false")
	}

	// Verify persistence
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		t.Fatalf("expected persisted file to exist")
	}
}
