package sub_test

import (
	"testing"

	"github.com/zyeyefyass/CubeX/internal/schema"
	"github.com/zyeyefyass/CubeX/internal/sub"
)

func TestEvaluateQuality_StandbyWhenUnsampled(t *testing.T) {
	nodes := []schema.ProxyNode{
		{Name: "Node-1"},
		{Name: "Node-2"},
		{Name: "Node-3"},
	}
	emptyLatency := make(map[string]int)

	report := sub.EvaluateQuality("sub-1", "Test Airport", nodes, emptyLatency)
	if report.State != "STANDBY" {
		t.Fatalf("Expected STANDBY state when unsampled, got: %s", report.State)
	}
	if report.HealthGrade != "待命就绪" {
		t.Fatalf("Expected 待命就绪 grade, got: %s", report.HealthGrade)
	}
	if report.OfflineNodes != 0 {
		t.Fatalf("Expected 0 offline nodes in standby, got: %d", report.OfflineNodes)
	}
}

func TestEvaluateQuality_Evaluated(t *testing.T) {
	nodes := []schema.ProxyNode{
		{Name: "Node-1"},
		{Name: "Node-2"},
		{Name: "Node-3"},
		{Name: "Node-4"},
	}
	latencies := map[string]int{
		"Node-1": 50,
		"Node-2": 80,
		"Node-3": 90,
		"Node-4": 9999, // offline
	}

	report := sub.EvaluateQuality("sub-1", "Test Airport", nodes, latencies)
	if report.OnlineNodes != 3 || report.OfflineNodes != 1 {
		t.Fatalf("Expected 3 online, 1 offline, got: %d online, %d offline", report.OnlineNodes, report.OfflineNodes)
	}
	if report.AvailabilityRate != 75.0 {
		t.Fatalf("Expected 75.0%% availability, got: %f", report.AvailabilityRate)
	}
	if report.State != "DEGRADED" {
		t.Fatalf("Expected DEGRADED for 75%%, got: %s", report.State)
	}
}
