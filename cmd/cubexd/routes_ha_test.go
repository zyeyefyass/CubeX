package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/host"
	"github.com/zyeyefyass/CubeX/internal/ros"
)

func TestIKuaiSelfHeal_NeverHitsRouterOSRestOrPort22(t *testing.T) {
	var rosRestHits atomic.Int32
	var ikuaiCallHits atomic.Int32

	mockRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/") {
			rosRestHits.Add(1)
			http.Error(w, "unexpected RouterOS REST call on iKuai host", http.StatusBadRequest)
			return
		}
		if r.URL.Path == "/Action/login" {
			http.SetCookie(w, &http.Cookie{Name: "sess_key", Value: "mock-ikuai-sess"})
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    0,
				"message": "Success",
				"results": map[string]any{"sess_key": "mock-ikuai-sess"},
			})
			return
		}
		if r.URL.Path == "/Action/call" {
			ikuaiCallHits.Add(1)
			var req struct {
				FuncName string `json:"func_name"`
				Action   string `json:"action"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			switch req.FuncName {
			case "sysstat":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code": 0,
					"results": map[string]any{
						"verinfo": map[string]any{"verstring": "4.0.317 x64", "arch": "x86_64"},
					},
				})
			case "lan":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    0,
					"results": map[string]any{"data": []any{}},
				})
			case "dhcp_server", "dhcp_option", "static_rt", "stream_ip":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    0,
					"results": map[string]any{"data": []any{}},
				})
			default:
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 0})
			}
			return
		}
		http.NotFound(w, r)
	}))
	defer mockRouter.Close()

	parts := strings.Split(strings.TrimPrefix(mockRouter.URL, "http://"), ":")
	hostIP := parts[0]
	portNum, _ := strconv.Atoi(parts[1])

	tmpDir := t.TempDir()
	srv := &Server{
		hostMgr: host.NewManager(tmpDir),
		rosMgr:  ros.NewManager(tmpDir),
	}

	// 1. Save iKuai config via /api/v1/host/config; also test port 22 does NOT pollute rosMgr
	cfgBody, _ := json.Marshal(host.HostConfig{
		Enabled:      true,
		Host:         hostIP,
		Port:         portNum,
		User:         "admin",
		Password:     "pass",
		PlatformType: "ikuai",
	})
	recCfg := httptest.NewRecorder()
	srv.handleHostConfig(recCfg, httptest.NewRequest(http.MethodPost, "/api/v1/host/config", bytes.NewReader(cfgBody)))
	if recCfg.Code != http.StatusOK {
		t.Fatalf("handleHostConfig failed: %d %s", recCfg.Code, recCfg.Body.String())
	}

	// Even if a rogue port 22 host-config save happens without platform_type, rosMgr must NOT inherit port 22
	rogueBody, _ := json.Marshal(host.HostConfig{
		Enabled:  true,
		Host:     hostIP,
		Port:     22,
		User:     "admin",
		Password: "pass",
	})
	recRogue := httptest.NewRecorder()
	srv.handleHostConfig(recRogue, httptest.NewRequest(http.MethodPost, "/api/v1/host/config", bytes.NewReader(rogueBody)))
	if srv.rosMgr.GetConfig()["port"] == "22" {
		t.Fatalf("handleHostConfig leaked SSH port 22 into rosMgr")
	}

	// Restore clean iKuai config
	recCfg2 := httptest.NewRecorder()
	srv.handleHostConfig(recCfg2, httptest.NewRequest(http.MethodPost, "/api/v1/host/config", bytes.NewReader(cfgBody)))

	// 2. Trigger /api/v1/ros/clean-and-reapply (the button that failed for zhengzongji)
	recHeal := httptest.NewRecorder()
	srv.handleROSCleanAndReapply(recHeal, httptest.NewRequest(http.MethodPost, "/api/v1/ros/clean-and-reapply", strings.NewReader(`{}`)))
	if recHeal.Code != http.StatusOK {
		t.Fatalf("handleROSCleanAndReapply returned %d", recHeal.Code)
	}
	var healResp ros.ApplyResult
	if err := json.Unmarshal(recHeal.Body.Bytes(), &healResp); err != nil {
		t.Fatalf("unmarshal healResp failed: %v", err)
	}
	if !healResp.Success || len(healResp.Errors) > 0 {
		t.Fatalf("expected iKuai clean-and-reapply to succeed cleanly, got success=%v errors=%v", healResp.Success, healResp.Errors)
	}

	// 3. Trigger /api/v1/ros/ipv6/setup right after (same frontend sequence)
	recV6 := httptest.NewRecorder()
	srv.handleROSIpv6Setup(recV6, httptest.NewRequest(http.MethodPost, "/api/v1/ros/ipv6/setup", strings.NewReader(`{}`)))
	if recV6.Code != http.StatusOK {
		t.Fatalf("handleROSIpv6Setup returned %d", recV6.Code)
	}

	if rosRestHits.Load() != 0 {
		t.Fatalf("expected ZERO RouterOS /rest/ calls on iKuai host, got %d", rosRestHits.Load())
	}
	if ikuaiCallHits.Load() == 0 {
		t.Fatalf("expected iKuai Web-RPC /Action/call to be invoked, got 0")
	}
}

func TestRouterOSSelfHealAndPort22Rejection(t *testing.T) {
	var rosRestHits atomic.Int32
	mockROS := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/") {
			rosRestHits.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/system/resource":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"platform":          "MikroTik",
				"architecture-name": "x86_64",
				"board-name":        "CHR",
				"version":           "7.25.2",
			})
		case "/rest/routing/table":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "cubex", ".id": "*1"}})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]any{".id": "*1"})
			}
		case "/rest/ip/route":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"dst-address": "0.0.0.0/0", "routing-table": "cubex", "gateway": "192.168.99.2", ".id": "*2"},
					{"dst-address": "198.18.0.0/15", "routing-table": "main", "gateway": "192.168.99.2", ".id": "*3"},
				})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]any{".id": "*4"})
			}
		default:
			_ = json.NewEncoder(w).Encode([]any{})
		}
	}))
	defer mockROS.Close()

	parts := strings.Split(strings.TrimPrefix(mockROS.URL, "https://"), ":")
	hostIP := parts[0]
	portNum, _ := strconv.Atoi(parts[1])

	tmpDir := t.TempDir()
	srv := &Server{
		hostMgr: host.NewManager(tmpDir),
		rosMgr:  ros.NewManager(tmpDir),
	}

	// Reject port 22 on /api/v1/host/test for iKuai and RouterOS
	for _, plat := range []string{"ikuai", "routeros"} {
		badBody, _ := json.Marshal(host.HostConfig{
			Host: hostIP, Port: 22, User: "admin", Password: "p", PlatformType: plat,
		})
		recBad := httptest.NewRecorder()
		srv.handleHostTest(recBad, httptest.NewRequest(http.MethodPost, "/api/v1/host/test", bytes.NewReader(badBody)))
		var res map[string]any
		_ = json.Unmarshal(recBad.Body.Bytes(), &res)
		if ok, _ := res["ok"].(bool); ok {
			t.Fatalf("expected handleHostTest to reject port 22 for %s", plat)
		}
	}

	// Save valid RouterOS config via /api/v1/host/config
	rosBody, _ := json.Marshal(host.HostConfig{
		Enabled: true, Host: hostIP, Port: portNum, User: "zyy", Password: "pwd", PlatformType: "routeros",
	})
	recSave := httptest.NewRecorder()
	srv.handleHostConfig(recSave, httptest.NewRequest(http.MethodPost, "/api/v1/host/config", bytes.NewReader(rosBody)))
	if recSave.Code != http.StatusOK {
		t.Fatalf("handleHostConfig ROS failed: %d", recSave.Code)
	}

	// Trigger handleROSCleanAndReapply on RouterOS platform -> MUST hit /rest/
	recHeal := httptest.NewRecorder()
	srv.handleROSCleanAndReapply(recHeal, httptest.NewRequest(http.MethodPost, "/api/v1/ros/clean-and-reapply", strings.NewReader(`{"target_ip":"192.168.99.2"}`)))
	if recHeal.Code != http.StatusOK {
		t.Fatalf("handleROSCleanAndReapply ROS failed: %d", recHeal.Code)
	}
	if rosRestHits.Load() == 0 {
		t.Fatalf("expected RouterOS /rest/ APIs to be called when platform is routeros")
	}
}

func TestHostBypassRemove_RouterOSSymmetry(t *testing.T) {
	var deletedPaths []string
	var patchedDNS bool
	var flushedDNS bool

	mockROS := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete {
			deletedPaths = append(deletedPaths, r.URL.Path)
			_ = json.NewEncoder(w).Encode(map[string]any{})
			return
		}
		switch r.URL.Path {
		case "/rest/system/resource":
			_ = json.NewEncoder(w).Encode(map[string]any{"platform": "MikroTik", "version": "7.25.2"})
		case "/rest/tool/netwatch":
			_ = json.NewEncoder(w).Encode([]map[string]any{{".id": "*NW1", "comment": "CubeX-HA-Failover", "host": "192.168.99.2"}})
		case "/rest/ip/route":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{".id": "*RT1", "dst-address": "0.0.0.0/0", "routing-table": "cubex", "gateway": "192.168.99.2", "comment": "CubeX Gateway Route"},
				{".id": "*RT2", "dst-address": "198.18.0.0/15", "routing-table": "main", "gateway": "192.168.99.2", "comment": "CubeX-FakeIP-Backtrack"},
			})
		case "/rest/routing/table":
			_ = json.NewEncoder(w).Encode([]map[string]any{{".id": "*TBL1", "name": "cubex"}})
		case "/rest/routing/rule":
			_ = json.NewEncoder(w).Encode([]map[string]any{{".id": "*RR1", "dst-address": "198.18.0.0/15", "table": "main", "comment": "CubeX-MultiWAN-FakeIP-Protect"}})
		case "/rest/ip/firewall/mangle":
			_ = json.NewEncoder(w).Encode([]map[string]any{{".id": "*MG1", "comment": "CubeX-PCC-Exempt-FakeIP"}})
		case "/rest/ip/firewall/nat":
			_ = json.NewEncoder(w).Encode([]map[string]any{{".id": "*NAT1", "comment": "CubeX-Client-DNS-Redirect"}})
		case "/rest/ip/firewall/filter":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{".id": "*FLT_FT", "chain": "forward", "action": "fasttrack-connection", "comment": "FastTrack Full Hardware Acceleration"},
				{".id": "*FLT1", "chain": "forward", "action": "accept", "comment": "CubeX Fake-IP Forward Accept"},
			})
		case "/rest/ip/firewall/address-list":
			_ = json.NewEncoder(w).Encode([]map[string]any{{".id": "*AL1", "list": "CUBEX_BYPASS", "address": "192.168.99.2"}})
		case "/rest/ipv6/firewall/filter":
			_ = json.NewEncoder(w).Encode([]map[string]any{{".id": "*V6F1", "comment": "CubeX Block Rogue IPv6 DNS UDP"}})
		case "/rest/ipv6/firewall/mangle":
			_ = json.NewEncoder(w).Encode([]map[string]any{{".id": "*V6M1", "comment": "CubeX_IPv6_MSS_Clamp"}})
		case "/rest/ip/dns":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode(map[string]any{"servers": "192.168.99.2"})
			} else {
				patchedDNS = true
				_ = json.NewEncoder(w).Encode(map[string]any{})
			}
		case "/rest/ip/dns/cache/flush":
			flushedDNS = true
			_ = json.NewEncoder(w).Encode(map[string]any{})
		default:
			_ = json.NewEncoder(w).Encode([]any{})
		}
	}))
	defer mockROS.Close()

	parts := strings.Split(strings.TrimPrefix(mockROS.URL, "https://"), ":")
	hostIP := parts[0]
	portNum, _ := strconv.Atoi(parts[1])

	tmpDir := t.TempDir()
	srv := &Server{
		hostMgr: host.NewManager(tmpDir),
		rosMgr:  ros.NewManager(tmpDir),
	}
	if err := srv.rosMgr.SaveConfig(map[string]string{
		"host": hostIP, "port": strconv.Itoa(portNum), "user": "admin", "password": "pwd",
	}); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/host/bypass/remove", strings.NewReader(`{"backup_dns":"223.5.5.5","target_ip":"192.168.99.2"}`))
	srv.handleHostBypassRemove(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("expected ok=true, got %v", resp)
	}
	if !patchedDNS || !flushedDNS {
		t.Fatalf("expected DNS restore & flush (patched=%v, flushed=%v)", patchedDNS, flushedDNS)
	}
	deletedSet := map[string]bool{}
	for _, p := range deletedPaths {
		deletedSet[p] = true
	}
	for _, want := range []string{
		"/rest/tool/netwatch/*NW1",
		"/rest/ip/route/*RT1",
		"/rest/ip/route/*RT2",
		"/rest/routing/table/*TBL1",
		"/rest/routing/rule/*RR1",
		"/rest/ip/firewall/mangle/*MG1",
		"/rest/ip/firewall/nat/*NAT1",
		"/rest/ip/firewall/filter/*FLT1",
		"/rest/ip/firewall/address-list/*AL1",
		"/rest/ipv6/firewall/filter/*V6F1",
		"/rest/ipv6/firewall/mangle/*V6M1",
	} {
		if !deletedSet[want] {
			t.Errorf("missing symmetric deletion of %s (deleted=%v)", want, deletedPaths)
		}
	}
	if deletedSet["/rest/ip/firewall/filter/*FLT_FT"] {
		t.Errorf("must NOT delete native RouterOS fasttrack-connection rule")
	}
}

func TestSwitchFromOpenWrtToRouterOS_CleanAndRemoveBypass(t *testing.T) {
	var rosRestHits atomic.Int32
	mockROS := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/") {
			rosRestHits.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete || r.Method == http.MethodPut || r.Method == http.MethodPost || r.Method == http.MethodPatch {
			_ = json.NewEncoder(w).Encode(map[string]any{".id": "*1"})
			return
		}
		switch r.URL.Path {
		case "/rest/system/resource":
			_ = json.NewEncoder(w).Encode(map[string]any{"platform": "MikroTik", "version": "7.25.2"})
		case "/rest/routing/table":
			_ = json.NewEncoder(w).Encode([]map[string]any{{".id": "*T1", "name": "cubex"}})
		case "/rest/ip/route":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{".id": "*R1", "dst-address": "0.0.0.0/0", "routing-table": "cubex", "gateway": "192.168.99.10"},
				{".id": "*R2", "dst-address": "198.18.0.0/15", "routing-table": "main", "gateway": "192.168.99.10", "comment": "CubeX-FakeIP-Backtrack"},
			})
		case "/rest/ip/dns":
			_ = json.NewEncoder(w).Encode(map[string]any{"servers": "192.168.99.10"})
		default:
			_ = json.NewEncoder(w).Encode([]any{})
		}
	}))
	defer mockROS.Close()

	parts := strings.Split(strings.TrimPrefix(mockROS.URL, "https://"), ":")
	hostIP := parts[0]
	portNum, _ := strconv.Atoi(parts[1])

	tmpDir := t.TempDir()
	srv := &Server{
		hostMgr: host.NewManager(tmpDir),
		rosMgr:  ros.NewManager(tmpDir),
	}

	// 1. Seed stale OpenWrt host config on an unreachable SSH port
	if err := srv.hostMgr.SaveConfig(host.HostConfig{
		Enabled:      true,
		Host:         hostIP,
		Port:         22,
		User:         "root",
		Password:     "password",
		PlatformType: "openwrt",
	}); err != nil {
		t.Fatal(err)
	}

	// 2. Save RouterOS config via /api/v1/ros/config -> must detect live ROS and sync hostMgr
	rosBody, _ := json.Marshal(map[string]string{
		"host":     hostIP,
		"port":     strconv.Itoa(portNum),
		"user":     "admin",
		"password": "pwd",
	})
	recCfg := httptest.NewRecorder()
	srv.handleROSConfig(recCfg, httptest.NewRequest(http.MethodPost, "/api/v1/ros/config", bytes.NewReader(rosBody)))
	if recCfg.Code != http.StatusOK {
		t.Fatalf("handleROSConfig returned %d: %s", recCfg.Code, recCfg.Body.String())
	}
	if gotPlat := srv.hostMgr.GetConfig().PlatformType; gotPlat != "routeros" {
		t.Fatalf("expected hostMgr PlatformType synced to routeros, got %q", gotPlat)
	}

	// 3. Even if hostCfg is disabled with stale openwrt PlatformType, resolveActiveHostPlatform must see routeros
	_ = srv.hostMgr.SaveConfig(host.HostConfig{Enabled: false, Host: "", PlatformType: "openwrt"})
	plat, _, _ := srv.resolveActiveHostPlatform()
	if plat != "routeros" {
		t.Fatalf("expected resolveActiveHostPlatform() == routeros when hostCfg disabled & rosMgr connected, got %q", plat)
	}

	// 4. CleanAndReapply & RemoveBypass must succeed via RouterOS REST without dialing SSH port 22
	recHeal := httptest.NewRecorder()
	srv.handleROSCleanAndReapply(recHeal, httptest.NewRequest(http.MethodPost, "/api/v1/ros/clean-and-reapply", strings.NewReader(`{"target_ip":"192.168.99.10"}`)))
	var healRes ros.ApplyResult
	if err := json.Unmarshal(recHeal.Body.Bytes(), &healRes); err != nil || !healRes.Success {
		t.Fatalf("handleROSCleanAndReapply failed: %v body=%s", err, recHeal.Body.String())
	}

	recRem := httptest.NewRecorder()
	srv.handleHostBypassRemove(recRem, httptest.NewRequest(http.MethodPost, "/api/v1/host/bypass/remove", strings.NewReader(`{"target_ip":"192.168.99.10","backup_dns":"223.5.5.5"}`)))
	var remRes map[string]any
	if err := json.Unmarshal(recRem.Body.Bytes(), &remRes); err != nil || remRes["ok"] != true {
		t.Fatalf("handleHostBypassRemove failed: %v body=%s", err, recRem.Body.String())
	}
	if rosRestHits.Load() == 0 {
		t.Fatalf("expected RouterOS REST API calls")
	}
}
