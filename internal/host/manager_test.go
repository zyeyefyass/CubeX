package host

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestTestSSH_RouterOSPort80_AuthFailure(t *testing.T) {
	// Mock HTTP server returning 401 Unauthorized like RouterOS REST API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/system/resource" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":   401,
				"message": "Unauthorized",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	// Extract port and host from ts.URL
	parts := strings.Split(strings.TrimPrefix(ts.URL, "http://"), ":")
	host := parts[0]
	port, _ := strconv.Atoi(parts[1])

	m := NewManager(t.TempDir())
	cfg := HostConfig{
		Host:     host,
		Port:     port,
		User:     "zyy",
		Password: "wrongpassword",
	}

	_, err := m.TestSSH(cfg)
	if err == nil {
		t.Fatalf("expected error on 401 Unauthorized, got nil")
	}

	// Must contain explicit authentication failure, NOT "ssh: handshake failed"
	if !strings.Contains(err.Error(), "鉴权失败") && !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected clear 401 auth error, got: %v", err)
	}
	if strings.Contains(err.Error(), "ssh: handshake failed") {
		t.Fatalf("unexpected misleading SSH error on HTTP port: %v", err)
	}
}

func TestTestSSH_RouterOSPort80_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/system/resource" {
			user, pass, ok := r.BasicAuth()
			if !ok || user != "zyy" || pass != "820130" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"platform":          "MikroTik",
				"architecture-name": "x86_64",
				"board-name":        "CHR",
				"version":           "7.24.5 (stable)",
				"cpu-count":         "4",
				"cpu-load":          "1",
				"total-memory":      "2080374784",
				"free-memory":       "1521696768",
				"uptime":            "2h43m9s",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	parts := strings.Split(strings.TrimPrefix(ts.URL, "http://"), ":")
	host := parts[0]
	port, _ := strconv.Atoi(parts[1])

	m := NewManager(t.TempDir())
	cfg := HostConfig{
		Host:     host,
		Port:     port,
		User:     "zyy",
		Password: "820130",
	}

	res, err := m.TestSSH(cfg)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if res["platform_id"] != "routeros" {
		t.Errorf("expected platform_id routeros, got: %v", res["platform_id"])
	}
	if res["kernel"] != "7.24.5 (stable)" {
		t.Errorf("expected kernel version 7.24.5 (stable), got: %v", res["kernel"])
	}
}

func TestProbeRouterOSWANList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/ip/address" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"address": "10.0.0.1/24", "interface": "bridge", "disabled": "false"},
				{"address": "116.23.45.67/24", "interface": "pppoe-out1", "disabled": "false"},
				{"address": "120.52.88.99/24", "interface": "wan2-unicom", "disabled": "false"},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	wanList := probeRouterOSWANList(ts.Client(), ts.URL, "admin", "admin")
	if len(wanList) != 2 {
		t.Fatalf("expected 2 WAN interfaces, got %d", len(wanList))
	}
	if wanList[0].Role != "primary" || wanList[1].Role != "secondary" {
		t.Errorf("unexpected roles: %s, %s", wanList[0].Role, wanList[1].Role)
	}
	if wanList[0].IP != "116.23.45.67" || wanList[1].IP != "120.52.88.99" {
		t.Errorf("unexpected IPs: %s, %s", wanList[0].IP, wanList[1].IP)
	}
}

func TestHostSaveConfigAndSyncROS_BlocksPort22Pollution(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	if err := m.SaveConfig(HostConfig{
		Host:         "192.168.2.253",
		User:         "admin",
		Password:     "pass",
		PlatformType: "ikuai",
		Port:         0,
	}); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}
	if m.GetConfig().Port != 80 {
		t.Fatalf("expected ikuai empty port to default to 80, got %d", m.GetConfig().Port)
	}

	if err := m.syncToROSConfig(HostConfig{
		Host:     "192.168.2.253",
		Port:     22,
		User:     "admin",
		Password: "pass",
	}); err != nil {
		t.Fatalf("syncToROSConfig failed: %v", err)
	}

	rawBytes, err := os.ReadFile(filepath.Join(tmpDir, "data", "ros-config.json"))
	if err != nil {
		t.Fatalf("read ros-config.json failed: %v", err)
	}
	var rosCfg map[string]string
	if err := json.Unmarshal(rawBytes, &rosCfg); err != nil {
		t.Fatalf("unmarshal ros-config.json failed: %v", err)
	}
	if rosCfg["port"] == "22" {
		t.Fatalf("syncToROSConfig must never write port 22 into ros-config.json")
	}
}

func TestValidateHostPortForPlatform(t *testing.T) {
	if _, err := ValidateHostPortForPlatform("ikuai", 22); err == nil {
		t.Fatalf("expected error for ikuai with port 22")
	}
	if _, err := ValidateHostPortForPlatform("routeros", 22); err == nil {
		t.Fatalf("expected error for routeros with port 22")
	}
	if p, err := ValidateHostPortForPlatform("ikuai", 0); err != nil || p != 80 {
		t.Fatalf("expected ikuai empty port -> 80, got %d err=%v", p, err)
	}
	if p, err := ValidateHostPortForPlatform("routeros", 0); err != nil || p != 80 {
		t.Fatalf("expected routeros empty port -> 80, got %d err=%v", p, err)
	}
	if p, err := ValidateHostPortForPlatform("openwrt", 0); err != nil || p != 22 {
		t.Fatalf("expected openwrt empty port -> 22, got %d err=%v", p, err)
	}
	if p, err := ValidateHostPortForPlatform("openwrt", 22); err != nil || p != 22 {
		t.Fatalf("expected openwrt port 22 valid, got %d err=%v", p, err)
	}
}
