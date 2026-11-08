package diagnostics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootCauseRadar(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "rootcause_report.json")

	radar := NewRootCauseRadar(storePath)
	rep := radar.GetLatestReport()
	if rep.OverallScore < 90 {
		t.Fatalf("expected high overall score, got %d", rep.OverallScore)
	}
	if len(rep.Hops) != 5 {
		t.Fatalf("expected 5 topology hops, got %d", len(rep.Hops))
	}

	// Test RunDiagnosis
	fresh := radar.RunDiagnosis("192.168.1.1", "192.168.1.1", "mihomo")
	if len(fresh.Hops) != 5 {
		t.Fatalf("expected 5 hops in fresh diagnosis")
	}

	// Test TriggerHeal
	healRes := radar.TriggerHeal()
	if !healRes.Success {
		t.Fatalf("expected heal success")
	}
	if healRes.HealedItemsCount != 5 {
		t.Fatalf("expected 5 healed items, got %d", healRes.HealedItemsCount)
	}

	// Verify persistence
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		t.Fatalf("expected persisted file to exist")
	}
}
