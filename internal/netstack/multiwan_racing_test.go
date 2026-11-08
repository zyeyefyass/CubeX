package netstack

import (
	"testing"
)

func TestMultiWANRacingManager(t *testing.T) {
	mgr := NewMultiWANRacingManager(nil)

	metrics := mgr.GetMetrics()
	if len(metrics.Interfaces) < 2 {
		t.Fatalf("expected at least 2 WAN interfaces, got: %d", len(metrics.Interfaces))
	}

	initialPrimary := mgr.SelectBestWAN()
	if initialPrimary != "wan1" {
		t.Fatalf("expected initial primary wan1, got: %s", initialPrimary)
	}

	// Update wan2 with good metrics
	mgr.UpdateTelemetry("wan2", 18.5, 1.2, 0.0)

	// Inject heavy degradation on wan1 (high loss & jitter)
	newPrimary := mgr.UpdateTelemetry("wan1", 120.0, 35.0, 0.08)
	if newPrimary != "wan2" {
		t.Fatalf("expected failover to wan2, got: %s", newPrimary)
	}

	if mgr.SelectBestWAN() != "wan2" {
		t.Fatalf("expected SelectBestWAN to return wan2, got: %s", mgr.SelectBestWAN())
	}

	res := mgr.GetMetrics()
	if res.TotalSwitches != 1 {
		t.Fatalf("expected 1 switch event, got: %d", res.TotalSwitches)
	}
}
