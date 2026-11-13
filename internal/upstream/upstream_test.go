package upstream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type mockProvider struct {
	id        string
	dim       Dimension
	probeFunc func(ctx context.Context) (*VersionStatus, error)
}

func (m *mockProvider) ID() string             { return m.id }
func (m *mockProvider) Dimension() Dimension   { return m.dim }
func (m *mockProvider) Probe(ctx context.Context) (*VersionStatus, error) {
	if m.probeFunc != nil {
		return m.probeFunc(ctx)
	}
	return &VersionStatus{
		ProviderID:     m.id,
		Dimension:      m.dim,
		CurrentVersion: "v1.0",
		LatestVersion:  "v1.1",
		HasUpdate:      true,
	}, nil
}
func (m *mockProvider) Upgrade(ctx context.Context, tx *AtomicInstaller) (*UpgradeResult, error) {
	return &UpgradeResult{ProviderID: m.id, ToVersion: "v1.1"}, nil
}

func TestProbeAllTimeoutIsolation(t *testing.T) {
	reg := NewRegistry(4, 5*time.Second)

	// D1: Normal fast provider
	reg.Register(&mockProvider{id: "control.cubexd", dim: DimControlPlane})

	// D2: Normal fast provider
	reg.Register(&mockProvider{id: "core.mihomo", dim: DimEngineCore})

	// D3: Normal fast provider
	reg.Register(&mockProvider{id: "rules.217heidai", dim: DimRuleCatalog})

	// D4: Intentionally slow/hanging provider to test timeout isolation
	reg.Register(&mockProvider{
		id:  "host.ros",
		dim: DimHostSymbiosis,
		probeFunc: func(ctx context.Context) (*VersionStatus, error) {
			select {
			case <-time.After(2 * time.Second):
				return &VersionStatus{LatestVersion: "v7.18"}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	})

	ctx := context.Background()
	// ProbeAll with 100ms per-provider timeout
	start := time.Now()
	report := reg.ProbeAll(ctx, 100*time.Millisecond)
	elapsed := time.Since(start)

	if elapsed > 1*time.Second {
		t.Fatalf("ProbeAll took %v, expected under 1s due to timeout isolation", elapsed)
	}

	// D1, D2, D3 must be present
	if len(report.Dimensions[DimControlPlane]) != 1 {
		t.Errorf("expected 1 result in DimControlPlane")
	}
	if len(report.Dimensions[DimEngineCore]) != 1 {
		t.Errorf("expected 1 result in DimEngineCore")
	}
	if len(report.Dimensions[DimRuleCatalog]) != 1 {
		t.Errorf("expected 1 result in DimRuleCatalog")
	}

	// D4 should have reported an error in report.Errors
	if _, ok := report.Errors["host.ros"]; !ok {
		t.Errorf("expected host.ros to record timeout error in report.Errors")
	}
}

func TestAtomicInstallerRollbackOnFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cubex-upstream-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	targetFile := filepath.Join(tmpDir, "my_binary")
	originalContent := []byte("ORIGINAL_STABLE_BINARY_V1")
	if err := os.WriteFile(targetFile, originalContent, 0755); err != nil {
		t.Fatal(err)
	}

	newContent := []byte("NEW_TEST_BINARY_V2")
	hash := sha256.Sum256(newContent)
	expectedHash := hex.EncodeToString(hash[:])

	// Mock HTTP download server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(newContent)
	}))
	defer srv.Close()

	installer := NewAtomicInstaller(5 * time.Second)

	// Case 1: PostReload fails -> must automatically roll back to original content
	ctx := context.Background()
	err = installer.InstallFile(
		ctx,
		srv.URL,
		targetFile,
		expectedHash,
		0755,
		nil, // no pre-check
		func(ctx context.Context) error {
			// Simulate health check failure during post-reload
			return errors.New("core crashed during reload probe")
		},
	)

	if err == nil {
		t.Fatal("expected InstallFile to return error on postReload failure")
	}

	// Verify targetFile was restored to original content!
	restored, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read restored target file: %v", err)
	}
	if string(restored) != string(originalContent) {
		t.Errorf("rollback failed! content got %q, want %q", string(restored), string(originalContent))
	}

	// Case 2: PostReload succeeds -> target file is updated
	err = installer.InstallFile(
		ctx,
		srv.URL,
		targetFile,
		expectedHash,
		0755,
		nil,
		func(ctx context.Context) error {
			return nil
		},
	)

	if err != nil {
		t.Fatalf("expected successful install, got: %v", err)
	}

	updated, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read updated target file: %v", err)
	}
	if string(updated) != string(newContent) {
		t.Errorf("update failed! content got %q, want %q", string(updated), string(newContent))
	}
}
