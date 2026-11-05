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

		if req.FuncName == "dhcp" || req.FuncName == "dhcp_server" {
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

		if req.FuncName == "dhcp_option" || req.FuncName == "dhcp_opts" {
			data := map[string]any{
				"data": []map[string]any{
					{
						"id":        1,
						"opt":       "6",
						"ip":        "192.168.1.200",
						"interface": "lan1",
						"comment":   "CubeX-DNS-Option6",
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

	// Test SyncSelfHealing
	syncLogs, err := client.SyncSelfHealing("192.168.1.200")
	if err != nil {
		t.Fatalf("SyncSelfHealing failed: %v", err)
	}
	if len(syncLogs) == 0 {
		t.Errorf("expected syncLogs, got none")
	}
}

func TestIKuaiV4ClientMock(t *testing.T) {
	// Mock iKuai 4.0 server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Action/login" {
			http.SetCookie(w, &http.Cookie{Name: "sess_key", Value: "mock_v4_session_key"})
			resp := RPCResponse{Result: 10000, Code: 0, Message: "成功"}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path != "/Action/call" {
			http.NotFound(w, r)
			return
		}

		var req RPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.FuncName == "sysstat" {
			data := map[string]any{
				"verinfo": map[string]any{
					"version": "4.0.314-beta x64",
					"sysname": "iKuai8",
					"arch":    "x86_64",
				},
				"cpu": []map[string]any{
					{"load": "1%"},
				},
				"memory": map[string]any{
					"used":  536870912.0,
					"total": 2147483648.0,
				},
				"uptime": "24h",
			}
			dataBytes, _ := json.Marshal(data)
			resp := RPCResponse{Code: 0, Message: "Success", Results: dataBytes}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if req.FuncName == "static_rt" {
			if req.Action == "show" {
				data := map[string]any{
					"data": []map[string]any{
						{
							"rowid":     1,
							"dst_addr":  "198.18.0.0/15",
							"gateway":   "10.0.0.7",
							"interface": "lan1",
							"ip_type":   "4",
							"tagname":   "CubeX-FakeIP-Bypass",
						},
					},
					"total": 1,
				}
				dataBytes, _ := json.Marshal(data)
				resp := RPCResponse{Code: 0, Message: "Success", Results: dataBytes}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			resp := RPCResponse{Code: 0, Message: "Success", RowID: 1}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if req.FuncName == "dhcp_server" {
			if req.Action == "show" {
				data := map[string]any{
					"data": []map[string]any{
						{
							"id":        1,
							"interface": "lan1",
							"dns1":      "10.0.0.7",
							"gateway":   "10.0.0.188",
						},
					},
					"total": 1,
				}
				dataBytes, _ := json.Marshal(data)
				resp := RPCResponse{Code: 0, Message: "Success", Results: dataBytes}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			resp := RPCResponse{Code: 0, Message: "Success"}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if req.FuncName == "dhcp_option" || req.FuncName == "dhcp_opts" {
			if req.Action == "show" {
				data := map[string]any{
					"data": []map[string]any{
						{
							"rowid":     1,
							"opt":       "6",
							"ip":        "10.0.0.7",
							"interface": "lan1",
							"comment":   "CubeX-DNS-Option6",
						},
					},
					"total": 1,
				}
				dataBytes, _ := json.Marshal(data)
				resp := RPCResponse{Code: 0, Message: "Success", Results: dataBytes}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			resp := RPCResponse{Code: 0, Message: "Success"}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		resp := RPCResponse{Code: 0, Message: "Success"}
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

	if err := client.Login(); err != nil {
		t.Fatalf("V4 Login failed: %v", err)
	}

	stat, err := client.GetSysStat()
	if err != nil {
		t.Fatalf("V4 GetSysStat failed: %v", err)
	}
	if stat.Version != "4.0.314-beta x64" {
		t.Errorf("expected version 4.0.314-beta x64, got %s", stat.Version)
	}

	hasRoute, hasDNS, details, err := client.InspectRouting("10.0.0.7")
	if err != nil {
		t.Fatalf("V4 InspectRouting failed: %v", err)
	}
	if !hasRoute || !hasDNS {
		t.Errorf("expected hasRoute=true, hasDNS=true, got %v, %v (details: %s)", hasRoute, hasDNS, details)
	}

	logs, err := client.ApplyBypass("10.0.0.7")
	if err != nil {
		t.Fatalf("V4 ApplyBypass failed: %v", err)
	}
	if len(logs) == 0 {
		t.Errorf("expected logs, got none")
	}

	// Test V4 SyncSelfHealing
	syncLogs, err := client.SyncSelfHealing("10.0.0.7")
	if err != nil {
		t.Fatalf("V4 SyncSelfHealing failed: %v", err)
	}
	if len(syncLogs) == 0 {
		t.Errorf("expected V4 syncLogs, got none")
	}
}
