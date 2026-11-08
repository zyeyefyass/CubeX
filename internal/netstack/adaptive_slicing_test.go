package netstack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAdaptiveSlicingManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_slicing_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewAdaptiveSlicingManager(tempDir)
	s := mgr.GetStatus()
	if !s.Enabled {
		t.Fatal("expected slicing to be enabled by default")
	}
	if len(s.Nodes) == 0 {
		t.Fatal("expected nodes to be initialized")
	}

	// Test Tune
	if err := mgr.Tune(false, 45.0); err != nil {
		t.Fatalf("Tune failed: %v", err)
	}
	s = mgr.GetStatus()
	if s.Enabled || s.CongestionThresholdMbps != 45.0 {
		t.Fatal("Tune did not update parameters properly")
	}

	// Test SampleAndEvaluate
	updated := mgr.SampleAndEvaluate()
	if updated.TotalStreamsSampled <= 8640 {
		t.Fatal("expected TotalStreamsSampled to increase")
	}

	// Check persistence
	filePath := filepath.Join(tempDir, "adaptive_slicing.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("expected status file to exist")
	}
}
