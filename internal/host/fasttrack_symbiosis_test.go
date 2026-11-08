package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFastTrackSymbiosisManager(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "fasttrack_symbiosis.json")

	mgr := NewFastTrackSymbiosisManager(storePath)
	status := mgr.GetStatus()
	if !status.Enabled {
		t.Fatalf("expected enabled by default")
	}

	// Test Optimize
	opt, err := mgr.Optimize()
	if err != nil {
		t.Fatalf("optimize failed: %v", err)
	}
	if opt.FastTrackRulesCount != 4 {
		t.Fatalf("expected 4 rules, got %d", opt.FastTrackRulesCount)
	}

	// Test Generate RSC
	rsc := mgr.GenerateROSFastTrackRSC("10.0.0.7")
	if !strings.Contains(rsc, "cubex_direct") {
		t.Fatalf("expected cubex_direct in RSC script")
	}

	// Test Toggle
	tog := mgr.Toggle(false)
	if tog != false {
		t.Fatalf("expected false")
	}

	// Verify persistence
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		t.Fatalf("expected store file to exist")
	}
}
