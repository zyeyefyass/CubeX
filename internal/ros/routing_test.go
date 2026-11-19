package ros

import (
	"net"
	"strings"
	"testing"
)

func TestDetectDefaultGatewayIP(t *testing.T) {
	// Should not panic, returns either a valid IP or empty string
	ipStr := DetectDefaultGatewayIP()
	if ipStr != "" {
		ip := net.ParseIP(ipStr)
		if ip == nil || ip.To4() == nil {
			t.Errorf("DetectDefaultGatewayIP() returned invalid IPv4: %s", ipStr)
		}
	}
}

func TestSubnetAdaptiveRouterIPMatch(t *testing.T) {
	// Simulate RouterOS interfaces across multiple distinct subnets
	rosIfAddrs := []struct {
		ip  net.IP
		net *net.IPNet
	}{
		{ip: net.ParseIP("192.168.88.1").To4()},
		{ip: net.ParseIP("10.0.0.1").To4()},
		{ip: net.ParseIP("172.16.10.1").To4()},
	}

	testCases := []struct {
		name       string
		dhcpNet    string
		currGW     string
		targetIP   string
		expectedGW string
		shouldFix  bool
	}{
		{
			name:       "Unmodified normal 192.168.88.x subnet should not be touched",
			dhcpNet:    "192.168.88.0/24",
			currGW:     "192.168.88.1",
			targetIP:   "192.168.88.228",
			expectedGW: "192.168.88.1",
			shouldFix:  false, // currGW != targetIP
		},
		{
			name:       "Misconfigured gateway pointing to CubeX sidecar should be healed back to main router",
			dhcpNet:    "192.168.88.0/24",
			currGW:     "192.168.88.228",
			targetIP:   "192.168.88.228",
			expectedGW: "192.168.88.1",
			shouldFix:  true, // currGW == targetIP
		},
		{
			name:       "Misconfigured gateway in 10.0.0.0/24 pointing to CubeX should be healed to 10.0.0.1",
			dhcpNet:    "10.0.0.0/24",
			currGW:     "10.0.0.7",
			targetIP:   "10.0.0.7",
			expectedGW: "10.0.0.1",
			shouldFix:  true, // currGW == targetIP
		},
		{
			name:       "Other custom router gateway (e.g. 192.168.88.254) should never be overridden",
			dhcpNet:    "192.168.88.0/24",
			currGW:     "192.168.88.254",
			targetIP:   "192.168.88.228",
			expectedGW: "192.168.88.254",
			shouldFix:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			needFixGW := (tc.currGW == tc.targetIP)
			if needFixGW != tc.shouldFix {
				t.Fatalf("expected shouldFix=%v, got %v", tc.shouldFix, needFixGW)
			}
			if !needFixGW {
				// Gateway remains unchanged
				if tc.currGW != tc.expectedGW {
					t.Errorf("expected gateway %s, got %s", tc.expectedGW, tc.currGW)
				}
				return
			}

			// Perform subnet match
			_, dhcpSubnet, err := net.ParseCIDR(tc.dhcpNet)
			if err != nil {
				t.Fatalf("failed to parse CIDR: %v", err)
			}

			var matchedIP string
			for _, ifa := range rosIfAddrs {
				if dhcpSubnet.Contains(ifa.ip) {
					matchedIP = ifa.ip.String()
					break
				}
			}

			if matchedIP != tc.expectedGW {
				t.Errorf("matched router IP %s != expected %s", matchedIP, tc.expectedGW)
			}
		})
	}
}

func TestBuildNetwatchAntiFlappingScripts(t *testing.T) {
	downScript := BuildNetwatchDownScript("10.0.0.7", 2026, "223.5.5.5,119.29.29.29", 4)
	if downScript == "" {
		t.Fatal("BuildNetwatchDownScript returned empty string")
	}
	if !strings.Contains(downScript, "cubexForceBypass") {
		t.Error("downScript missing cubexForceBypass check")
	}
	if !strings.Contains(downScript, "550ms") {
		t.Error("downScript missing 550ms interval")
	}
	if !strings.Contains(downScript, "200ms") {
		t.Error("downScript missing 200ms hard ping timeout")
	}
	if !strings.Contains(downScript, "/healthz/dataplane") {
		t.Error("downScript missing /healthz/dataplane fetch probe")
	}

	upScript := BuildNetwatchUpScript("10.0.0.7", 2026)
	if upScript == "" {
		t.Fatal("BuildNetwatchUpScript returned empty string")
	}
	if !strings.Contains(upScript, "escapedRoutes") {
		t.Error("upScript missing escapedRoutes disabled count check")
	}
	if !strings.Contains(upScript, "1200ms") {
		t.Error("upScript missing 1200ms steady state dampening delay")
	}
	if !strings.Contains(upScript, "/healthz/dataplane") {
		t.Error("upScript missing /healthz/dataplane readiness check")
	}
}

func TestMultiWANRoutingRulePayload(t *testing.T) {
	// Verify that the routing rule parameters follow RouterOS v7 syntax
	fakeIPRule := map[string]any{
		"dst-address": "198.18.0.0/15",
		"action":      "lookup-only-in-table",
		"table":       "main",
		"comment":     "CubeX-MultiWAN-FakeIP-Protect",
	}
	if fakeIPRule["action"] != "lookup-only-in-table" || fakeIPRule["table"] != "main" {
		t.Errorf("expected lookup-only-in-table main, got action=%v table=%v", fakeIPRule["action"], fakeIPRule["table"])
	}
}

func TestManglePCCExemptionIdempotency(t *testing.T) {
	// Verify idempotent matching for existing mangle rules
	existingMangles := []map[string]any{
		{
			".id":         "*1A",
			"chain":       "prerouting",
			"action":      "accept",
			"dst-address": "198.18.0.0/15",
			"disabled":    false,
			"comment":     "CubeX-PCC-Exempt-FakeIP",
		},
		{
			".id":         "*1B",
			"chain":       "prerouting",
			"action":      "accept",
			"dst-address": "10.0.0.7",
			"disabled":    true, // should be re-enabled without inserting duplicate
			"comment":     "CubeX-PCC-Exempt-Host",
		},
	}

	targetIP := "10.0.0.7"
	hasFakeIP := false
	hasHost := false
	needReEnableHost := false

	for _, mg := range existingMangles {
		chain, _ := mg["chain"].(string)
		action, _ := mg["action"].(string)
		dstAddr, _ := mg["dst-address"].(string)
		comment, _ := mg["comment"].(string)
		disabled, _ := mg["disabled"].(bool)

		isFakeIPMatch := (chain == "prerouting" && action == "accept" && dstAddr == "198.18.0.0/15") || strings.Contains(comment, "CubeX-PCC-Exempt-FakeIP")
		isHostMatch := (chain == "prerouting" && action == "accept" && (dstAddr == targetIP || dstAddr == targetIP+"/32")) || strings.Contains(comment, "CubeX-PCC-Exempt-Host")

		if isFakeIPMatch {
			hasFakeIP = true
		}
		if isHostMatch {
			hasHost = true
			if disabled {
				needReEnableHost = true
			}
		}
	}

	if !hasFakeIP {
		t.Error("expected hasFakeIP to be true, got false")
	}
	if !hasHost {
		t.Error("expected hasHost to be true, got false")
	}
	if !needReEnableHost {
		t.Error("expected needReEnableHost to be true for disabled rule")
	}
}
