package netenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHexLittleEndianIPv4(t *testing.T) {
	tests := []struct {
		hexStr string
		wantIP string
		wantOk bool
	}{
		{"0100A8C0", "192.168.0.1", true},
		{"0101A8C0", "192.168.1.1", true},
		{"FE00000A", "10.0.0.254", true},
		{"00000000", "0.0.0.0", true},
		{"INVALID!", "", false},
		{"123", "", false},
	}

	for _, tt := range tests {
		gotIP, gotOk := parseHexLittleEndianIPv4(tt.hexStr)
		if gotOk != tt.wantOk || gotIP != tt.wantIP {
			t.Errorf("parseHexLittleEndianIPv4(%q) = (%q, %v), want (%q, %v)", tt.hexStr, gotIP, gotOk, tt.wantIP, tt.wantOk)
		}
	}
}

func TestParseProcNetDefaultRoute(t *testing.T) {
	tmpDir := t.TempDir()
	mockProcFile := filepath.Join(tmpDir, "route")

	content := "Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n" +
		"eth0\t00000000\t0101A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0\n" +
		"eth0\t0001A8C0\t00000000\t0001\t0\t0\t100\t00FFFFFF\t0\t0\t0\n"

	if err := os.WriteFile(mockProcFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write mock proc file: %v", err)
	}

	iface, gw := parseProcNetDefaultRoute(mockProcFile)
	if iface != "eth0" {
		t.Errorf("expected iface 'eth0', got %q", iface)
	}
	if gw != "192.168.1.1" {
		t.Errorf("expected gateway '192.168.1.1', got %q", gw)
	}
}

func TestDetectHostTopologyWithMockProc(t *testing.T) {
	tmpDir := t.TempDir()
	mockProcFile := filepath.Join(tmpDir, "route")

	// 1. 模拟正常 192.168.50.0/24 拓扑
	content := "Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n" +
		"eth0\t00000000\t0132A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0\n"

	_ = os.WriteFile(mockProcFile, []byte(content), 0644)

	topo := DetectHostTopologyWithProcRoute(mockProcFile, "192.168.50.100", "")
	if !topo.Ready {
		t.Fatalf("expected topo.Ready to be true, got warning: %s", topo.DiagnosticWarning)
	}
	if topo.GatewayIP != "192.168.50.1" {
		t.Errorf("expected GatewayIP 192.168.50.1, got %s", topo.GatewayIP)
	}
	if topo.LocalIP != "192.168.50.100" {
		t.Errorf("expected LocalIP 192.168.50.100, got %s", topo.LocalIP)
	}
	if topo.FakeIPCIDR != DefaultFakeIPCIDR {
		t.Errorf("expected FakeIPCIDR %s, got %s", DefaultFakeIPCIDR, topo.FakeIPCIDR)
	}

	// 2. 模拟路由自环 (LocalIP == GatewayIP)
	topoLoop := DetectHostTopologyWithProcRoute(mockProcFile, "192.168.50.1", "192.168.50.1")
	if topoLoop.Ready {
		t.Errorf("expected Ready=false on loop, got true")
	}
	if !strings.Contains(topoLoop.DiagnosticWarning, "路由自环") {
		t.Errorf("expected warning to mention 路由自环, got: %s", topoLoop.DiagnosticWarning)
	}

	// 3. 模拟无网关
	emptyProcFile := filepath.Join(tmpDir, "route_empty")
	_ = os.WriteFile(emptyProcFile, []byte("Iface\tDestination\tGateway\n"), 0644)
	topoNoGW := DetectHostTopologyWithProcRoute(emptyProcFile, "192.168.1.50", "")
	if topoNoGW.Ready {
		t.Errorf("expected Ready=false on missing gateway, got true")
	}
	if !strings.Contains(topoNoGW.DiagnosticWarning, "缺少默认网关") {
		t.Errorf("expected warning to mention 缺少默认网关, got: %s", topoNoGW.DiagnosticWarning)
	}
}
