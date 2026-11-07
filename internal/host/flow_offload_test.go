package host

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFlowOffloadManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_flow_offload_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := GetFlowOffloadManager(filepath.Join(tempDir, "data"))

	status := mgr.GetStatus()
	if !status.Enabled {
		t.Errorf("expected enabled to be true")
	}
	if status.Mode != ModeFastTrackPassthrough {
		t.Errorf("expected ModeFastTrackPassthrough, got %s", status.Mode)
	}

	// Test SetEnabled
	if err := mgr.SetEnabled(false); err != nil {
		t.Fatalf("failed to set enabled: %v", err)
	}
	if mgr.GetStatus().Enabled {
		t.Errorf("expected enabled to be false")
	}

	// Re-enable with iKuai mode
	if err := mgr.SetEnabled(true, ModeIKuaiHardwareNAT); err != nil {
		t.Fatalf("failed to set enabled ikuai: %v", err)
	}
	if mgr.GetStatus().Mode != ModeIKuaiHardwareNAT {
		t.Errorf("expected ModeIKuaiHardwareNAT, got %s", mgr.GetStatus().Mode)
	}

	// Test OptimizeRules
	optStatus := mgr.OptimizeRules()
	if optStatus.OffloadedPercent <= 0 {
		t.Errorf("expected valid offloaded percent")
	}
}
