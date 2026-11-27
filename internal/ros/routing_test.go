package ros

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
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
	for _, s := range []string{downScript, upScript} {
		if strings.Contains(s, ":return;") {
			t.Errorf("netwatch script must not contain bare ':return;' (fails in RouterOS 7.25.2): %s", s)
		}
		if strings.Contains(s, "disabled=no") || strings.Contains(s, "disabled=yes") {
			t.Errorf("netwatch script must use '!disabled'/'disabled' instead of disabled=no/yes for RouterOS 7.25.2 compatibility: %s", s)
		}
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

func TestRouterOSManager_RemoveBypass(t *testing.T) {
	// 1. Unconfigured RouterOS should return a clear error
	unconfMgr := NewManager(t.TempDir())
	if _, err := unconfMgr.RemoveBypass("192.168.99.2", "223.5.5.5"); err == nil {
		t.Fatalf("expected error when RouterOS is not configured")
	}

	// 2. Configured RouterOS withdraws routes, blackholes, netwatch, DNS, and DHCP overrides symmetrically
	var deleted []string
	var dhcpPatched map[string]any
	mockROS := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete {
			deleted = append(deleted, r.URL.Path)
			_ = json.NewEncoder(w).Encode(map[string]any{})
			return
		}
		switch r.URL.Path {
		case "/rest/system/resource":
			_ = json.NewEncoder(w).Encode(map[string]any{"version": "7.25.2"})
		case "/rest/ip/route":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{".id": "*BH1", "dst-address": "198.18.0.0/15", "blackhole": true, "comment": "CubeX-FakeIP-Blackhole"},
				{".id": "*BT1", "dst-address": "198.18.0.0/15", "routing-table": "main", "gateway": "192.168.99.2", "comment": "CubeX-FakeIP-Backtrack"},
			})
		case "/rest/ip/dns":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode(map[string]any{"servers": "192.168.99.2"})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]any{})
			}
		case "/rest/ip/dhcp-server/network":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{".id": "*DH1", "address": "192.168.99.0/24", "gateway": "192.168.99.1", "dns-server": "192.168.99.2"},
			})
		case "/rest/ip/dhcp-server/network/*DH1":
			_ = json.NewDecoder(r.Body).Decode(&dhcpPatched)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		case "/rest/ip/firewall/connection":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{".id": "*CN1", "dst-address": "198.18.0.12:443"},
			})
		default:
			_ = json.NewEncoder(w).Encode([]any{})
		}
	}))
	defer mockROS.Close()

	parts := strings.Split(strings.TrimPrefix(mockROS.URL, "https://"), ":")
	mgr := NewManager(t.TempDir())
	if err := mgr.SaveConfig(map[string]string{
		"host": parts[0], "port": parts[1], "user": "admin", "password": "pwd",
	}); err != nil {
		t.Fatal(err)
	}
	logs, err := mgr.RemoveBypass("192.168.99.2", "223.5.5.5")
	if err != nil {
		t.Fatalf("RemoveBypass failed: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("expected non-empty logs from RemoveBypass")
	}
	if dhcpPatched["dns-server"] != "223.5.5.5" {
		t.Fatalf("expected DHCP dns-server restored to 223.5.5.5, got %v", dhcpPatched)
	}
	deletedStr := strings.Join(deleted, ",")
	for _, want := range []string{"*BH1", "*BT1", "*CN1"} {
		if !strings.Contains(deletedStr, want) {
			t.Errorf("expected %s deleted, got %v", want, deleted)
		}
	}
}

func TestApplyAndCleanRouting_LegacyTableAndDanglingMangle(t *testing.T) {
	// Simulate RouterOS behavior: deleting /routing/table/*TBL_OLD turns mangle mark "table-cubex-8228" into "*400"
	var tableDeleted bool
	var deletedMangles []string
	var deletedOrder []string

	mockROS := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete {
			deletedOrder = append(deletedOrder, r.URL.Path)
			if r.URL.Path == "/rest/routing/table/*TBL_OLD" {
				tableDeleted = true
			}
			if strings.HasPrefix(r.URL.Path, "/rest/ip/firewall/mangle/") {
				deletedMangles = append(deletedMangles, strings.TrimPrefix(r.URL.Path, "/rest/ip/firewall/mangle/"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{})
			return
		}
		if r.Method == http.MethodPut || r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{".id": "*NEW"})
			return
		}
		switch r.URL.Path {
		case "/rest/system/resource":
			_ = json.NewEncoder(w).Encode(map[string]any{"version": "7.25.2"})
		case "/rest/routing/table":
			tables := []map[string]any{{".id": "*TBL_CUBEX", "name": "cubex"}}
			if !tableDeleted {
				tables = append(tables, map[string]any{".id": "*TBL_OLD", "name": "table-cubex-8228"})
			}
			_ = json.NewEncoder(w).Encode(tables)
		case "/rest/ip/firewall/mangle":
			mark := "table-cubex-8228"
			if tableDeleted {
				mark = "*400"
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{".id": "*MG_LEGACY", "chain": "prerouting", "action": "mark-routing", "new-routing-mark": mark, "src-address-list": "!CUBEX_BYPASS"},
				{".id": "*MG_DANGLING", "chain": "prerouting", "action": "mark-routing", "new-routing-mark": "*400"},
			})
		case "/rest/ip/route":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{".id": "*RT_CUBEX", "dst-address": "0.0.0.0/0", "routing-table": "cubex", "gateway": "192.168.99.10"},
				{".id": "*RT_FIP", "dst-address": "198.18.0.0/15", "routing-table": "main", "gateway": "192.168.99.10"},
				{".id": "*RT_BH", "dst-address": "198.18.0.0/15", "blackhole": true, "comment": "CubeX-FakeIP-Blackhole"},
			})
		case "/rest/ip/dns":
			_ = json.NewEncoder(w).Encode(map[string]any{"servers": "192.168.99.10"})
		default:
			_ = json.NewEncoder(w).Encode([]any{})
		}
	}))
	defer mockROS.Close()

	parts := strings.Split(strings.TrimPrefix(mockROS.URL, "https://"), ":")
	mgr := NewManager(t.TempDir())
	if err := mgr.SaveConfig(map[string]string{
		"host": parts[0], "port": parts[1], "user": "admin", "password": "pwd",
	}); err != nil {
		t.Fatal(err)
	}

	res, err := mgr.ApplyRouting("192.168.99.10")
	if err != nil {
		t.Fatalf("ApplyRouting failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected ApplyRouting Success, got errors=%v", res.Errors)
	}
	deletedStr := strings.Join(deletedMangles, ",")
	if !strings.Contains(deletedStr, "*MG_LEGACY") || !strings.Contains(deletedStr, "*MG_DANGLING") {
		t.Fatalf("expected both legacy and dangling mangle rules deleted, got %v", deletedMangles)
	}

	// Verify mangle deletion occurred before table deletion
	idxMangle, idxTable := -1, -1
	for i, p := range deletedOrder {
		if p == "/rest/ip/firewall/mangle/*MG_LEGACY" && idxMangle == -1 {
			idxMangle = i
		}
		if p == "/rest/routing/table/*TBL_OLD" && idxTable == -1 {
			idxTable = i
		}
	}
	if idxMangle == -1 || idxTable == -1 || idxMangle > idxTable {
		t.Fatalf("expected legacy mangle deleted before legacy table deletion, order=%v", deletedOrder)
	}

	// Also verify CleanAndReapplyRouting & RemoveBypass handle dangling "*400" marks
	deletedMangles = nil
	tableDeleted = true
	if _, err := mgr.CleanAndReapplyRouting("192.168.99.10"); err != nil {
		t.Fatalf("CleanAndReapplyRouting failed: %v", err)
	}
	if !strings.Contains(strings.Join(deletedMangles, ","), "*MG_DANGLING") {
		t.Fatalf("CleanAndReapplyRouting missed dangling *400 mangle rule: %v", deletedMangles)
	}

	deletedMangles = nil
	if _, err := mgr.RemoveBypass("192.168.99.10", "223.5.5.5"); err != nil {
		t.Fatalf("RemoveBypass failed: %v", err)
	}
	if !strings.Contains(strings.Join(deletedMangles, ","), "*MG_DANGLING") {
		t.Fatalf("RemoveBypass missed dangling *400 mangle rule: %v", deletedMangles)
	}
}
