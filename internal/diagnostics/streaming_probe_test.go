package diagnostics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStreamingDiagnosticManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_streaming_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewStreamingDiagnosticManager(tempDir)
	rep := mgr.GetReport()
	if rep == nil {
		t.Fatal("expected initial report to be non-nil")
	}
	if len(rep.Probes) == 0 {
		t.Fatal("expected initial probes to have entries")
	}
	if rep.OverallScore < 80 {
		t.Fatalf("unexpected score: %d", rep.OverallScore)
	}

	// Test RunDiagnostic
	newRep, err := mgr.RunDiagnostic()
	if err != nil {
		t.Fatalf("RunDiagnostic failed: %v", err)
	}
	if newRep == nil || len(newRep.Probes) == 0 {
		t.Fatal("expected new report with probes")
	}

	// Test ExecuteHeal
	healRes, err := mgr.ExecuteHeal()
	if err != nil {
		t.Fatalf("ExecuteHeal failed: %v", err)
	}
	if !healRes.Success || len(healRes.ActionsExecuted) == 0 {
		t.Fatal("expected successful heal result")
	}

	healedRep := mgr.GetReport()
	if healedRep.OverallScore != 100 {
		t.Fatalf("expected score 100 after heal, got: %d", healedRep.OverallScore)
	}

	// Verify persistence
	filePath := filepath.Join(tempDir, "streaming_diagnostic.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("expected file to be persisted")
	}
}
