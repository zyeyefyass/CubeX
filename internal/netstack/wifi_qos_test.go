package netstack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWiFiQoSManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_wifi_qos_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := GetWiFiQoSManager(filepath.Join(tempDir, "data"))

	status := mgr.GetStatus()
	if !status.Enabled {
		t.Errorf("expected enabled to be true")
	}
	if status.TotalMonitoredClients == 0 {
		t.Errorf("expected default monitored clients")
	}

	// Test SetEnabled
	if err := mgr.SetEnabled(false); err != nil {
		t.Fatalf("failed to disable: %v", err)
	}
	if mgr.GetStatus().Enabled {
		t.Errorf("expected enabled to be false")
	}
	_ = mgr.SetEnabled(true)

	// Test TuneClient (VIP override)
	if err := mgr.TuneClient("10.0.0.194", true, false); err != nil {
		t.Fatalf("failed to tune client: %v", err)
	}
	st := mgr.GetStatus()
	found := false
	for _, c := range st.MonitoredClients {
		if c.IP == "10.0.0.194" {
			found = true
			if !c.IsVIP || c.QueueClass != QueueExemptVIP {
				t.Errorf("expected 10.0.0.194 to be VIP and QueueExemptVIP, got %s", c.QueueClass)
			}
		}
	}
	if !found {
		t.Errorf("client 10.0.0.194 not found")
	}

	// Test InspectAndRecalculate
	recStatus := mgr.InspectAndRecalculate()
	if recStatus.ChannelCleanScore <= 0 {
		t.Errorf("expected valid channel clean score, got %d", recStatus.ChannelCleanScore)
	}
}
