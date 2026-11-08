package dnsradar

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrivacyMeshManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_privacymesh_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewPrivacyMeshManager(tempDir)
	s := mgr.GetStatus()
	if !s.Enabled {
		t.Fatal("expected privacy mesh to be enabled")
	}
	if len(s.ActiveRules) == 0 {
		t.Fatal("expected default rules")
	}

	// Test Tune
	if err := mgr.Tune(false); err != nil {
		t.Fatalf("Tune failed: %v", err)
	}
	s = mgr.GetStatus()
	if s.Enabled {
		t.Fatal("expected privacy mesh to be disabled")
	}

	// Test Optimize
	if err := mgr.Optimize(); err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}
	optStatus := mgr.GetStatus()
	if optStatus.TotalQueriesFiltered <= 1924500 {
		t.Fatal("expected TotalQueriesFiltered to increase")
	}

	// Verify persistence
	filePath := filepath.Join(tempDir, "privacy_mesh.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("expected status file to be written")
	}
}
