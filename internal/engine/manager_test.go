package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/compiler"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestManagerPreFlightAndBinaryDiscovery(t *testing.T) {
	workDir := filepath.Join("..", "..")
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		t.Fatalf("failed to get abs workdir: %v", err)
	}

	ir := schema.NewDefaultIR()
	mgr := NewManager(absWorkDir, ir)

	mihomoBin := mgr.findBinary("mihomo")
	if _, err := os.Stat(mihomoBin); err != nil {
		t.Errorf("failed to discover mihomo binary: %s (%v)", mihomoBin, err)
	}

	singboxBin := mgr.findBinary("sing-box")
	if _, err := os.Stat(singboxBin); err != nil {
		t.Errorf("failed to discover sing-box binary: %s (%v)", singboxBin, err)
	}

	etcDir := filepath.Join(absWorkDir, "etc")
	mihomoConfig := filepath.Join(etcDir, "config.yaml")
	mihomoData, err := compiler.BuildMihomo(ir)
	if err != nil {
		t.Fatalf("failed to compile mihomo: %v", err)
	}
	_ = os.WriteFile(mihomoConfig, mihomoData, 0644)

	if err := mgr.dryRunCheck(schema.EngineMihomo, mihomoBin, etcDir, mihomoConfig); err != nil {
		t.Errorf("mihomo dry-run validation failed: %v", err)
	}

	singboxConfig := filepath.Join(etcDir, "config.json")
	singboxData, err := compiler.BuildSingBox(ir)
	if err != nil {
		t.Fatalf("failed to compile sing-box: %v", err)
	}
	_ = os.WriteFile(singboxConfig, singboxData, 0644)

	if err := mgr.dryRunCheck(schema.EngineSingbox, singboxBin, etcDir, singboxConfig); err != nil {
		t.Errorf("sing-box dry-run validation failed: %v", err)
	}
}
