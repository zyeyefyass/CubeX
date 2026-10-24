package ikuai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestIKuaiClientMock(t *testing.T) {
	// Mock iKuai server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Action/login" {
			http.SetCookie(w, &http.Cookie{Name: "sess_key", Value: "mock_session_key_12345"})
			resp := RPCResponse{Result: 10000, ErrMsg: "Success"}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path != "/Action/call" {
			http.NotFound(w, r)
			return
		}

		var req RPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.FuncName == "login" {
			http.SetCookie(w, &http.Cookie{Name: "sess_key", Value: "mock_session_key_12345"})
			resp := RPCResponse{Result: 30000, ErrMsg: "Success"}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if req.FuncName == "sysstat" {
			data := map[string]any{
				"verinfo": map[string]any{
					"version": "3.7.14 x64",
					"sysname": "iKuai8",
					"arch":    "x86_64",
				},
				"cpu": map[string]any{
					"load": "2%",
				},
				"memory": map[string]any{
					"used":  "512MB",
					"total": "4096MB",
				},
				"uptime": "120h",
			}
			dataBytes, _ := json.Marshal(data)
			resp := RPCResponse{Result: 30000, ErrMsg: "Success", Data: dataBytes}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if req.FuncName == "route_static" {
			data := map[string]any{
				"data": []map[string]any{
					{
						"id":        1,
						"ip":        "198.18.0.0",
						"netmask":   "255.254.0.0",
						"gateway":   "192.168.1.200",
						"interface": "lan1",
						"comment":   "CubeX-FakeIP-Bypass",
						"enabled":   "yes",
					},
				},
				"total": 1,
			}
			dataBytes, _ := json.Marshal(data)
			resp := RPCResponse{Result: 30000, ErrMsg: "Success", Data: dataBytes}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if req.FuncName == "dhcp" {
			data := map[string]any{
				"data": []map[string]any{
					{
						"id":        1,
						"interface": "lan1",
						"dns1":      "192.168.1.200",
						"gateway":   "192.168.1.1",
					},
				},
				"total": 1,
			}
			dataBytes, _ := json.Marshal(data)
			resp := RPCResponse{Result: 30000, ErrMsg: "Success", Data: dataBytes}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		resp := RPCResponse{Result: 30000, ErrMsg: "Success"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("failed to parse url: %v", err)
	}
	port, _ := strconv.Atoi(u.Port())

	client := NewClient(Config{
		Host:     u.Hostname(),
		Port:     port,
		User:     "admin",
		Password: "password123",
	})

	// Test Login
	if err := client.Login(); err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Test SysStat
	stat, err := client.GetSysStat()
	if err != nil {
		t.Fatalf("GetSysStat failed: %v", err)
	}
	if stat.Version != "3.7.14 x64" {
		t.Errorf("expected version 3.7.14 x64, got %s", stat.Version)
	}

	// Test InspectRouting
	hasRoute, hasDNS, details, err := client.InspectRouting("192.168.1.200")
	if err != nil {
		t.Fatalf("InspectRouting failed: %v", err)
	}
	if !hasRoute || !hasDNS {
		t.Errorf("expected hasRoute=true, hasDNS=true, got %v, %v (details: %s)", hasRoute, hasDNS, details)
	}

	// Test ApplyBypass
	logs, err := client.ApplyBypass("192.168.1.200")
	if err != nil {
		t.Fatalf("ApplyBypass failed: %v", err)
	}
	if len(logs) == 0 {
		t.Errorf("expected logs, got none")
	}
}
