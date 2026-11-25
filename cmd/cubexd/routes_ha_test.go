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
	mockROS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	parts := strings.Split(strings.TrimPrefix(mockROS.URL, "http://"), ":")
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
