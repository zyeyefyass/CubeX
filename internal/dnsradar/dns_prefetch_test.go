package dnsradar

import (
	"os"
	"testing"
)

func TestDNSPrefetchManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cubex_prefetch_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewDNSPrefetchManager(tmpDir)
	defer mgr.Close()

	status := mgr.GetStatus()
	if !status.Enabled {
		t.Errorf("expected prefetch to be enabled by default")
	}
	if status.HotDomainsCount == 0 {
		t.Errorf("expected default hot domains count > 0")
	}
	if status.AvgMemoryLatencyUs <= 0 {
		t.Errorf("expected positive average memory latency")
	}

	// Test Toggle
	statusToggled := mgr.Toggle(false)
	if statusToggled.Enabled {
		t.Errorf("expected prefetch to be disabled after toggle")
	}

	statusReToggled := mgr.Toggle(true)
	if !statusReToggled.Enabled {
		t.Errorf("expected prefetch to be enabled after re-toggle")
	}

	// Test Flush
	statusFlushed := mgr.Flush()
	if statusFlushed.HotDomainsCount == 0 {
		t.Errorf("expected re-initialized hot domains after flush")
	}
}
