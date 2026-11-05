package ros

import (
	"net"
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
