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

	testClientIP := "192.168.50.194"
	mgr.UpsertClient(WiFiClientMetrics{
		IP:         testClientIP,
		MAC:        "F4:60:E2:44:55:66",
		HostName:   "Test-Cam-IoT",
		RSSIdBm:    -78,
		IsWeakLink: true,
		QueueClass: QueueAirtimePaced,
	})

	status := mgr.GetStatus()
	if !status.Enabled {
		t.Errorf("expected enabled to be true")
	}
	if status.TotalMonitoredClients == 0 {
		t.Errorf("expected monitored clients")
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
	if err := mgr.TuneClient(testClientIP, true, false); err != nil {
		t.Fatalf("failed to tune client: %v", err)
	}
	st := mgr.GetStatus()
	found := false
	for _, c := range st.MonitoredClients {
		if c.IP == testClientIP {
			found = true
			if !c.IsVIP || c.QueueClass != QueueExemptVIP {
				t.Errorf("expected %s to be VIP and QueueExemptVIP, got %s", testClientIP, c.QueueClass)
			}
		}
	}
	if !found {
		t.Errorf("client %s not found", testClientIP)
	}

	// Test InspectAndRecalculate
	recStatus := mgr.InspectAndRecalculate()
	if recStatus.ChannelCleanScore <= 0 {
		t.Errorf("expected valid channel clean score, got %d", recStatus.ChannelCleanScore)
	}
}
