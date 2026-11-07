package healing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActiveHealerManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_healer_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := GetActiveHealerManager(filepath.Join(tempDir, "data"))

	status := mgr.GetStatus()
	if !status.IsRunning {
		t.Errorf("expected is_running to be true")
	}
	if len(status.RecentEvents) == 0 {
		t.Errorf("expected default sample events to be present")
	}

	// Test SetMode
	if err := mgr.SetMode(ModeBalanced); err != nil {
		t.Fatalf("failed to set mode: %v", err)
	}
	if mgr.GetStatus().Mode != ModeBalanced {
		t.Errorf("expected mode to be balanced, got %s", mgr.GetStatus().Mode)
	}

	// Test RegisterCallbacks and TriggerRemediation
	callbackInvoked := false
	mgr.RegisterCallbacks(
		func() (int, error) {
			callbackInvoked = true
			return 15, nil
		},
		nil, nil, nil,
	)

	event := mgr.TriggerRemediation(Tier1ConntrackFlush, "测试悬挂流表清洗", 18.5, 0.0)
	if event.Tier != Tier1ConntrackFlush {
		t.Errorf("unexpected tier: %s", event.Tier)
	}
	if !callbackInvoked {
		t.Errorf("expected conntrack flusher callback to be called")
	}
	if event.Result != "RESOLVED" {
		t.Errorf("expected result RESOLVED, got %s", event.Result)
	}

	// Test IngestTelemetry
	triggered, ev := mgr.IngestTelemetry(45.0, 1.0, 0)
	if !triggered || ev == nil {
		t.Errorf("expected telemetry ingestion to trigger remediation for 45ms jitter in balanced mode")
	}
}
