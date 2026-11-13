package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestClientInspector_TopLANActivityAndKick(t *testing.T) {
	var mu sync.Mutex
	deletedIDs := make([]string, 0)

	// 构造 50 条模拟连接数据
	mockConnections := make([]map[string]any, 0, 50)
	for i := 0; i < 50; i++ {
		srcIP := fmt.Sprintf("192.168.1.%d", (i%5)+10) // 5 台设备: 192.168.1.10 ~ 14
		targetHost := fmt.Sprintf("site%d.com", i%8)   // 8 个域名
		mockConnections = append(mockConnections, map[string]any{
			"id":       fmt.Sprintf("conn-%d", i),
			"upload":   int64(i * 1024),
			"download": int64(i * 1024 * 1024), // 下载流量递增
			"rule":     "RuleSet",
			"chains":   []string{"ProxyGroup"},
			"metadata": map[string]any{
				"sourceIP":      srcIP,
				"destinationIP": "1.2.3.4",
				"host":          targetHost,
			},
		})
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodGet && r.URL.Path == "/connections" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"connections": mockConnections,
			})
			return
		}

		if r.Method == http.MethodDelete && len(r.URL.Path) > len("/connections/") {
			connID := r.URL.Path[len("/connections/"):]
			deletedIDs = append(deletedIDs, connID)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	ci := NewClientInspector(server.URL, "test-secret")

	// 1. 验证 Top 聚合画像
	report, err := ci.InspectTopLANActivity(context.Background())
	if err != nil {
		t.Fatalf("InspectTopLANActivity failed: %v", err)
	}

	if report.TotalActiveConns != 50 {
		t.Errorf("expected 50 total conns, got %d", report.TotalActiveConns)
	}
	if len(report.TopClients) != 5 {
		t.Errorf("expected 5 unique clients, got %d", len(report.TopClients))
	}

	// 检查排序：流量最大的应该排在第 0 位
	for i := 0; i < len(report.TopClients)-1; i++ {
		if report.TopClients[i].DownloadTotal < report.TopClients[i+1].DownloadTotal {
			t.Errorf("clients not sorted by download total descending")
		}
	}

	// 2. 验证精准断流指定 IP
	targetKickIP := "192.168.1.14"
	kickedCount, err := ci.KickClientConnections(context.Background(), targetKickIP)
	if err != nil {
		t.Fatalf("KickClientConnections failed: %v", err)
	}

	// 192.168.1.14 应该有 10 条连接 (i % 5 == 4: 4, 9, 14, 19, 24, 29, 34, 39, 44, 49)
	if kickedCount != 10 {
		t.Errorf("expected 10 kicked connections for %s, got %d", targetKickIP, kickedCount)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(deletedIDs) != 10 {
		t.Errorf("expected 10 DELETE calls, got %d", len(deletedIDs))
	}
}
