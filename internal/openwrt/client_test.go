package openwrt

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestNewClient(t *testing.T) {
	cfg := Config{
		Host:     "192.168.1.1",
		Port:     0,
		User:     "",
		Password: "password123",
	}
	c := NewClient(cfg)
	if c.cfg.Port != 80 {
		t.Errorf("expected default port 80, got %d", c.cfg.Port)
	}
	if c.cfg.User != "root" {
		t.Errorf("expected default user root, got %s", c.cfg.User)
	}
	if c.getBaseURL() != "http://192.168.1.1:80/ubus" {
		t.Errorf("unexpected base URL: %s", c.getBaseURL())
	}
}

func TestUbusSessionMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req UbusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Method == "call" && len(req.Params) >= 2 && req.Params[1] == "session" {
			// Login response
			resp := UbusResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  json.RawMessage(`[0, {"ubus_rpc_session": "test-sid-12345"}]`),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if req.Method == "call" && len(req.Params) >= 2 && req.Params[1] == "system" {
			// system.info response
			resp := UbusResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: json.RawMessage(`[0, {
					"hostname": "OpenWrt-Router",
					"model": "x86_64 VirtualBox",
					"release": {
						"distribution": "OpenWrt",
						"version": "23.05.3"
					},
					"memory": {
						"total": 1073741824,
						"free": 536870912
					}
				}]`),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse test server url: %v", err)
	}
	port, _ := strconv.Atoi(u.Port())

	c := NewClient(Config{
		Host:     u.Hostname(),
		Port:     port,
		User:     "root",
		Password: "pwd",
	})

	info, err := c.probeViaUbus()
	if err != nil {
		t.Fatalf("probeViaUbus failed: %v", err)
	}
	if info.Hostname != "OpenWrt-Router" {
		t.Errorf("unexpected hostname: %s", info.Hostname)
	}
	if info.Release != "OpenWrt 23.05.3" {
		t.Errorf("unexpected release: %s", info.Release)
	}
}
