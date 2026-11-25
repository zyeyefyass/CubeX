package ikuai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
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

func TestIKuaiMultiWANExemption(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Action/login" {
			http.SetCookie(w, &http.Cookie{Name: "sess_key", Value: "mock_session_key_12345"})
			resp := RPCResponse{Result: 10000, ErrMsg: "Success"}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/Action/call" {
			var req RPCRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			if req.FuncName == "login" {
				http.SetCookie(w, &http.Cookie{Name: "sess_key", Value: "mock_session_key_12345"})
				resp := RPCResponse{Result: 30000, ErrMsg: "Success"}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}

			if req.FuncName == "stream_ip" && req.Action == "show" {
				data := map[string]any{
					"data": []map[string]any{
						{
							"id":        1,
							"interface": "wan1",
							"dst_ip":    "0.0.0.0-255.255.255.255",
							"comment":   "All WAN1 Load Balance",
						},
					},
					"total": 1,
				}
				dataBytes, _ := json.Marshal(data)
				_ = json.NewEncoder(w).Encode(RPCResponse{Result: 30000, ErrMsg: "Success", Data: dataBytes})
				return
			}

			if req.FuncName == "stream_ip" && req.Action == "add" {
				_ = json.NewEncoder(w).Encode(RPCResponse{Result: 30000, ErrMsg: "Success"})
				return
			}
		}

		_ = json.NewEncoder(w).Encode(RPCResponse{Result: 30000, ErrMsg: "Success"})
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())
	client := NewClient(Config{
		Host: u.Hostname(),
		Port: port,
	})

	mwInfo, err := client.InspectMultiWANPolicy("10.0.0.7")
	if err != nil {
		t.Fatalf("InspectMultiWANPolicy failed: %v", err)
	}
	if !mwInfo.HasMultiWAN {
		t.Errorf("expected HasMultiWAN=true, got false")
	}
	if mwInfo.FakeIPProtected {
		t.Errorf("expected FakeIPProtected=false before exemption, got true")
	}

	logs, err := client.ApplyMultiWANExemption("10.0.0.7", "lan1")
	if err != nil {
		t.Fatalf("ApplyMultiWANExemption failed: %v", err)
	}
	if len(logs) == 0 {
		t.Errorf("expected exemption logs, got none")
	}
}

func TestDetectLANInterface(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/Action/login" {
			http.SetCookie(w, &http.Cookie{Name: "sess_key", Value: "mock_session_lan"})
			_ = json.NewEncoder(w).Encode(RPCResponse{Result: 30000, ErrMsg: "Success"})
			return
		}

		var req RPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.FuncName == "login" {
			http.SetCookie(w, &http.Cookie{Name: "sess_key", Value: "mock_session_lan"})
			_ = json.NewEncoder(w).Encode(RPCResponse{Result: 30000, ErrMsg: "Success"})
			return
		}

		if req.FuncName == "lan" && req.Action == "show" {
			data := map[string]any{
				"data": []map[string]any{
					{
						"interface": "lan1",
						"ip_addr":   "192.168.1.1",
						"netmask":   "255.255.255.0",
					},
					{
						"interface": "lan2",
						"ip_addr":   "10.0.0.1",
						"netmask":   "255.255.255.0",
					},
				},
				"total": 2,
			}
			dataBytes, _ := json.Marshal(data)
			_ = json.NewEncoder(w).Encode(RPCResponse{Result: 30000, ErrMsg: "Success", Data: dataBytes})
			return
		}
		_ = json.NewEncoder(w).Encode(RPCResponse{Result: 30000, ErrMsg: "Success"})
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())
	client := NewClient(Config{
		Host: u.Hostname(),
		Port: port,
	})

	// Case 1: 10.0.0.7 in lan2 subnet
	if iface := client.DetectLANInterface("10.0.0.7"); iface != "lan2" {
		t.Errorf("expected lan2 for 10.0.0.7, got %s", iface)
	}

	// Case 2: 192.168.1.100 in lan1 subnet
	if iface := client.DetectLANInterface("192.168.1.100"); iface != "lan1" {
		t.Errorf("expected lan1 for 192.168.1.100, got %s", iface)
	}

	// Case 3: Out-of-range IP defaults to lan1
	if iface := client.DetectLANInterface("172.16.50.2"); iface != "lan1" {
		t.Errorf("expected fallback lan1 for 172.16.50.2, got %s", iface)
	}
}

// 🛡️ 师父防雷补丁 4: 测试并发下的会话重登与双检锁机制
func TestIKuaiConcurrentSessionRelogin(t *testing.T) {
	loginCount := 0
	var mu sync.Mutex

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Action/login" {
			mu.Lock()
			loginCount++
			mu.Unlock()
			http.SetCookie(w, &http.Cookie{Name: "sess_key", Value: "valid_session"})
			_ = json.NewEncoder(w).Encode(RPCResponse{Result: 10000, Code: 0, Message: "Success"})
			return
		}

		if r.URL.Path == "/Action/call" {
			cookie, err := r.Cookie("sess_key")
			if err != nil || cookie.Value != "valid_session" {
				_ = json.NewEncoder(w).Encode(RPCResponse{Result: 10001, ErrMsg: "Session expired, please login"})
				return
			}
			dataBytes, _ := json.Marshal(map[string]any{"data": []any{}})
			_ = json.NewEncoder(w).Encode(RPCResponse{Result: 30000, ErrMsg: "Success", Data: dataBytes})
			return
		}
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())
	client := NewClient(Config{
		Host: u.Hostname(),
		Port: port,
	})

	// Inject expired session initially
	client.sessKey = "expired_cookie"
	client.lastLogin = time.Now().Add(-20 * time.Minute)

	var wg sync.WaitGroup
	workers := 10
	errChan := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Call("sysstat", "show", map[string]string{})
			if err != nil {
				errChan <- err
				return
			}
			if !resp.IsSuccess() {
				errChan <- fmt.Errorf("response not successful: %s", resp.ErrorMessage())
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Fatalf("unexpected error during concurrent call: %v", err)
	}

	mu.Lock()
	count := loginCount
	mu.Unlock()

	// Double-Checked Locking should ensure minimal logins (at most 2 in extreme race, far below workers count)
	if count > 2 {
		t.Errorf("expected at most 2 logins with double-checked locking, got %d", count)
	}
}

func TestIKuaiCircuitBreaker(t *testing.T) {
	// Point to non-routable port to trigger immediate connection failure
	client := NewClient(Config{
		Host: "127.0.0.1",
		Port: 65432,
	})

	// Trigger 3 failures
	for i := 0; i < 3; i++ {
		_, _ = client.Call("sysstat", "show", map[string]string{})
	}

	// 4th call should immediately trigger circuit breaker without network dial
	_, err := client.Call("sysstat", "show", map[string]string{})
	if err == nil {
		t.Fatalf("expected circuit breaker error, got nil")
	}
	if !strings.Contains(err.Error(), "熔断保护") {
		t.Errorf("expected circuit breaker error containing '熔断保护', got: %v", err)
	}
}

func TestIKuaiV4_ApplyBypassIncludesStreamIPAndRemoveBypass(t *testing.T) {
	var streamShowCalled, streamAddCalled, streamDelCalled, rtDelCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Action/login" {
			http.SetCookie(w, &http.Cookie{Name: "sess_key", Value: "v4_test_sess"})
			_ = json.NewEncoder(w).Encode(RPCResponse{Result: 10000, Code: 0})
			return
		}
		var req RPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		switch req.FuncName {
		case "lan":
			b, _ := json.Marshal(map[string]any{"data": []map[string]any{{"id": 1, "ip_addr": "192.168.99.8", "netmask": "255.255.255.0", "interface": "lan1"}}})
			_ = json.NewEncoder(w).Encode(RPCResponse{Code: 0, Results: b})
		case "dhcp_server":
			b, _ := json.Marshal(map[string]any{"data": []map[string]any{{"id": 1, "interface": "lan1", "dns1": "223.5.5.5", "gateway": "192.168.99.8"}}})
			_ = json.NewEncoder(w).Encode(RPCResponse{Code: 0, Results: b})
		case "dhcp_option":
			if req.Action == "show" {
				b, _ := json.Marshal(map[string]any{"data": []map[string]any{{"id": 7, "opt": "6", "ip": "192.168.99.10", "comment": "CubeX-DNS-Option6"}}})
				_ = json.NewEncoder(w).Encode(RPCResponse{Code: 0, Results: b})
				return
			}
			_ = json.NewEncoder(w).Encode(RPCResponse{Code: 0})
		case "static_rt":
			if req.Action == "show" {
				b, _ := json.Marshal(map[string]any{"data": []map[string]any{{"id": 9, "dst_addr": "198.18.0.0", "gateway": "192.168.99.10", "tagname": "CubeX-Bypass"}}})
				_ = json.NewEncoder(w).Encode(RPCResponse{Code: 0, Results: b})
				return
			}
			if req.Action == "del" {
				rtDelCalled = true
			}
			_ = json.NewEncoder(w).Encode(RPCResponse{Code: 0})
		case "stream_ip":
			if req.Action == "show" {
				streamShowCalled = true
				b, _ := json.Marshal(map[string]any{"data": []map[string]any{{"id": 11, "dst_ip": "198.18.0.0/15", "comment": "CubeX-FakeIP"}}})
				_ = json.NewEncoder(w).Encode(RPCResponse{Code: 0, Results: b})
				return
			}
			if req.Action == "add" {
				streamAddCalled = true
			}
			if req.Action == "del" {
				streamDelCalled = true
			}
			_ = json.NewEncoder(w).Encode(RPCResponse{Code: 0})
		default:
			_ = json.NewEncoder(w).Encode(RPCResponse{Code: 0})
		}
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())
	client := NewClient(Config{Host: u.Hostname(), Port: port, User: "admin", Password: "pwd"})
	if _, err := client.ApplyBypass("192.168.99.10"); err != nil {
		t.Fatalf("ApplyBypass 4.0 failed: %v", err)
	}
	if !streamShowCalled {
		t.Fatalf("expected iKuai 4.0 ApplyBypass to run Multi-WAN stream_ip exemption check instead of returning early")
	}
	_ = streamAddCalled
	if _, err := client.RemoveBypass("192.168.99.10", "223.5.5.5"); err != nil {
		t.Fatalf("RemoveBypass 4.0 failed: %v", err)
	}
	if !rtDelCalled || !streamDelCalled {
		t.Fatalf("expected RemoveBypass to delete both static_rt (%v) and stream_ip (%v)", rtDelCalled, streamDelCalled)
	}
}
