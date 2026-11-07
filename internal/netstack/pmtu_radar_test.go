package netstack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPMTUManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_pmtu_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := GetPMTUManager(filepath.Join(tempDir, "data"))

	report := mgr.GetStatus()
	if report.DetectedPMTU <= 0 {
		t.Errorf("expected valid detected PMTU, got %d", report.DetectedPMTU)
	}
	if report.ClampedMSS != report.DetectedPMTU-40 {
		t.Errorf("expected clamped MSS %d, got %d", report.DetectedPMTU-40, report.ClampedMSS)
	}

	// Test SetAutoClamping
	if err := mgr.SetAutoClamping(false); err != nil {
		t.Fatalf("failed to disable auto clamping: %v", err)
	}
	if mgr.GetStatus().AutoClampingEnabled {
		t.Errorf("expected auto clamping disabled")
	}

	// Test RunPMTUProbe
	newReport := mgr.RunPMTUProbe()
	if len(newReport.Hops) == 0 {
		t.Errorf("expected hops in report")
	}
}
