package dnsradar

import (
	"os"
	"testing"
)

func TestDNSRadarManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_dnsradar_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewDNSRadarManager(tempDir)

	// 1. Initial Report
	report := mgr.GetReport()
	if len(report.Probes) == 0 {
		t.Errorf("expected at least 1 probe result")
	}

	// 2. Device Registration and Sandboxing
	isNew := mgr.RegisterDevice("aa:bb:cc:dd:ee:ff", "192.168.1.188", "Unknown-Phone")
	if !isNew {
		t.Errorf("expected isNew=true for first registration")
	}

	report = mgr.RunProbe()
	if report.SandboxedDevicesCount != 1 {
		t.Errorf("expected 1 sandboxed device, got %d", report.SandboxedDevicesCount)
	}

	// 3. Approve Device
	approved := mgr.ApproveDevice("aa:bb:cc:dd:ee:ff")
	if !approved {
		t.Errorf("expected approve=true")
	}

	report = mgr.RunProbe()
	if report.SandboxedDevicesCount != 0 {
		t.Errorf("expected 0 sandboxed devices after approve, got %d", report.SandboxedDevicesCount)
	}
}
