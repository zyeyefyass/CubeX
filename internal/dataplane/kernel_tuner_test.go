package dataplane

import (
	"testing"
)

func TestKernelTuner(t *testing.T) {
	report := OptimizeKernelForHighThroughput(t.TempDir())
	if report == nil {
		t.Fatal("OptimizeKernelForHighThroughput returned nil")
	}

	mihomoCfg := BuildMihomoCacheConfig(t.TempDir())
	if mihomoCfg["store-fake-ip"] != true {
		t.Errorf("expected store-fake-ip to be true, got %v", mihomoCfg["store-fake-ip"])
	}
	if mihomoCfg["store-selected"] != true {
		t.Errorf("expected store-selected to be true, got %v", mihomoCfg["store-selected"])
	}

	singboxCfg := BuildSingBoxCacheConfig(t.TempDir())
	if singboxCfg["enabled"] != true {
		t.Errorf("expected singbox cache enabled to be true, got %v", singboxCfg["enabled"])
	}
	if singboxCfg["store_fakeip"] != true {
		t.Errorf("expected store_fakeip to be true, got %v", singboxCfg["store_fakeip"])
	}
	if singboxCfg["store_rdrc"] != true {
		t.Errorf("expected store_rdrc to be true, got %v", singboxCfg["store_rdrc"])
	}
}
