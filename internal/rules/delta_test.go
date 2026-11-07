package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeltaManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cubex-delta-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	manifestPath := filepath.Join(tmpDir, "manifest.json")
	cachePath := filepath.Join(tmpDir, "ir_cache.bin")

	dm := NewDeltaManager(manifestPath, cachePath)

	ruleTag := "mrs_direct"
	content1 := []byte("payload_version_1")

	if !dm.HasChanged(ruleTag, content1) {
		t.Fatalf("expected HasChanged=true for new rule")
	}

	dm.RecordUpdate(ruleTag, content1, 4.5)

	if dm.HasChanged(ruleTag, content1) {
		t.Fatalf("expected HasChanged=false for identical content")
	}

	content2 := []byte("payload_version_2_updated")
	if !dm.HasChanged(ruleTag, content2) {
		t.Fatalf("expected HasChanged=true for updated content")
	}

	// Test cache save and load
	mockCache := []byte("binary_precompiled_radix_tree_data")
	if err := dm.SavePrecompiledCache(mockCache); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	loaded, err := dm.LoadPrecompiledCache()
	if err != nil || string(loaded) != string(mockCache) {
		t.Fatalf("failed to load identical cache: %v", err)
	}

	status := dm.GetStatus()
	if !status.CacheFileExists || status.DeltaPatchesApplied != 1 {
		t.Fatalf("unexpected delta status: %+v", status)
	}
}
