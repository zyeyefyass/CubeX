package device

import (
	"os"
	"testing"
)

func TestDeviceQoSManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cubex-device-qos-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := GetDeviceQoSManager(tmpDir)

	ip := "10.0.0.88"
	mac := "00:11:22:33:44:55"

	// Check default
	p := mgr.GetDevicePriority(ip, mac)
	if p != PriorityNormal {
		t.Errorf("expected normal priority, got %s", p)
	}

	// Set VIP
	err = mgr.SetDevicePriority(ip, mac, PriorityVIP, 0, 0)
	if err != nil {
		t.Fatalf("set priority err: %v", err)
	}

	p = mgr.GetDevicePriority(ip, mac)
	if p != PriorityVIP {
		t.Errorf("expected VIP priority, got %s", p)
	}

	// Sample flows
	devs := []Device{
		{IP: ip, MAC: mac, HostName: "test-pc", Online: true, IsProxied: true},
	}
	flows := mgr.SampleDeviceFlows(devs)
	if len(flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(flows))
	}
	if flows[0].Priority != PriorityVIP {
		t.Errorf("expected flow priority VIP, got %s", flows[0].Priority)
	}
}
