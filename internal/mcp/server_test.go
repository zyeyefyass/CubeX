package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockProvider struct {
	telemetry map[string]any
	devices   []map[string]any
	healMsg   string
	engine    string
}

func (m *mockProvider) GetTelemetry() (map[string]any, error) {
	return m.telemetry, nil
}

func (m *mockProvider) ListDevices() ([]map[string]any, error) {
	return m.devices, nil
}

func (m *mockProvider) SetDeviceMode(ip, mode string) error {
	for _, d := range m.devices {
		if d["ip"] == ip {
			d["mode"] = mode
			return nil
		}
	}
	return nil
}

func (m *mockProvider) TriggerSelfHeal() (string, error) {
	return m.healMsg, nil
}

func (m *mockProvider) SwitchEngine(engine string) (string, error) {
	m.engine = engine
	return "740 毫秒", nil
}

func TestMCPServer(t *testing.T) {
	mock := &mockProvider{
		telemetry: map[string]any{
			"active_engine":      "mihomo",
			"connections_count":  12,
			"inbound_speed_bps":  int64(1024 * 1024 * 3),
			"outbound_speed_bps": int64(1024 * 50),
			"memory_mb":          55.2,
			"month_total_gb":     0.25,
		},
		devices: []map[string]any{
			{"name": "Apple TV 4K", "ip": "10.0.0.120", "mac": "aa:bb:cc:dd:ee:01", "mode": "rule", "online": true},
			{"name": "三星回音壁", "ip": "10.0.0.191", "mac": "aa:bb:cc:dd:ee:02", "mode": "bypass", "online": true},
		},
		healMsg: "已自动修复策略路由与连接池",
	}

	server := NewServer(mock, "v4.2")

	// 1. Test GET discovery
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mcp", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected GET 200, got %d", w.Code)
	}
	var disco map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &disco); err != nil || disco["name"] != "CubeX Gateway MCP Server" {
		t.Fatalf("invalid discovery response: %v", w.Body.String())
	}

	// 2. Test initialize
	initPayload := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/mcp", bytes.NewReader(initPayload))
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected init 200, got %d", w.Code)
	}
	var initResp RPCResponse
	_ = json.Unmarshal(w.Body.Bytes(), &initResp)
	if initResp.Error != nil {
		t.Fatalf("init error: %v", initResp.Error)
	}

	// 3. Test tools/list
	listPayload := []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/mcp", bytes.NewReader(listPayload))
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	var listResp struct {
		Result struct {
			Tools []ToolDefinition `json:"tools"`
		} `json:"result"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if len(listResp.Result.Tools) != 5 {
		t.Fatalf("expected 5 tools, got %d", len(listResp.Result.Tools))
	}

	// 4. Test tools/call: get_gateway_telemetry
	callPayload := []byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_gateway_telemetry","arguments":{}}}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/mcp", bytes.NewReader(callPayload))
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	var callResp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &callResp)
	if len(callResp.Result.Content) == 0 || !strings.Contains(callResp.Result.Content[0].Text, "CubeX 旁路由网关实时运行指标") {
		t.Fatalf("invalid telemetry call response: %v", w.Body.String())
	}

	// 5. Test tools/call: set_device_policy
	setPayload := []byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"set_device_policy","arguments":{"ip":"10.0.0.191","mode":"rule"}}}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/mcp", bytes.NewReader(setPayload))
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if mock.devices[1]["mode"] != "rule" {
		t.Fatalf("expected device mode updated to rule, got %v", mock.devices[1]["mode"])
	}
}
