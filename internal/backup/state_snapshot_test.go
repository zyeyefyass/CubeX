package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateSnapshotManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_snapshot_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write mock config.ir.json
	mockIR := `{"meta":{"active_engine":"mihomo","proxy_mode":"rule"},"subscriptions":[{"name":"sub1"}],"proxies":[{"name":"proxy1"}]}`
	if err := os.WriteFile(filepath.Join(tempDir, "config.ir.json"), []byte(mockIR), 0644); err != nil {
		t.Fatalf("failed to write mock IR: %v", err)
	}

	mgr := NewStateSnapshotManager(tempDir)

	// 1. Test Create Snapshot
	snap, rawBytes, err := mgr.CreateSnapshot("Test v8.0", "MikroTik RouterOS 7.25.1")
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}
	if snap.Metadata.TotalSubscriptions != 1 {
		t.Fatalf("expected 1 subscription, got: %d", snap.Metadata.TotalSubscriptions)
	}
	if snap.ChecksumSHA256 == "" {
		t.Fatalf("expected non-empty checksum")
	}

	// 2. Test Atomic Restore
	reloadCalled := false
	res, err := mgr.RestoreSnapshot(rawBytes, func() error {
		reloadCalled = true
		return nil
	})
	if err != nil {
		t.Fatalf("RestoreSnapshot failed: %v", err)
	}
	if !res.Success || !reloadCalled {
		t.Fatalf("expected successful restore and reloadFn invoked")
	}

	// 3. Test Invalid JSON
	_, err = mgr.RestoreSnapshot([]byte("invalid json"), nil)
	if err == nil {
		t.Fatalf("expected error for invalid json, got nil")
	}
}
