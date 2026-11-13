package sub

import (
	"context"
	"fmt"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestAutoSyncGuard_ZeroNodesCircuitBreaker(t *testing.T) {
	guard := NewAutoSyncGuard()
	reloaded := false
	reloadFn := func(ctx context.Context, nodes []schema.ProxyNode) error {
		reloaded = true
		return nil
	}

	report, err := guard.EvaluateAndSync(context.Background(), nil, QuotaTelemetry{}, false, reloadFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Outcome != OutcomeCircuitBroken {
		t.Errorf("expected OutcomeCircuitBroken, got %v", report.Outcome)
	}
	if reloaded {
		t.Error("reloadFn should not have been called on 0 nodes")
	}
}

func TestAutoSyncGuard_DropOffCircuitBreaker(t *testing.T) {
	guard := NewAutoSyncGuard()
	reloaded := false
	reloadFn := func(ctx context.Context, nodes []schema.ProxyNode) error {
		reloaded = true
		return nil
	}

	// 模拟初始状态有 50 个节点
	initNodes := make([]schema.ProxyNode, 50)
	for i := 0; i < 50; i++ {
		initNodes[i] = schema.ProxyNode{Name: fmt.Sprintf("Node-%d", i), Server: "1.1.1.1", Port: 443}
	}
	guard.SetInitialState(initNodes)

	// 模拟拉取结果暴跌至 2 个节点
	droppedNodes := []schema.ProxyNode{
		{Name: "Node-0", Server: "1.1.1.1", Port: 443},
		{Name: "Node-1", Server: "1.1.1.1", Port: 443},
	}

	report, err := guard.EvaluateAndSync(context.Background(), droppedNodes, QuotaTelemetry{}, false, reloadFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Outcome != OutcomeCircuitBroken {
		t.Errorf("expected OutcomeCircuitBroken on severe drop, got %v", report.Outcome)
	}
	if reloaded {
		t.Error("reloadFn should not have been called on severe node drop")
	}

	// 验证手动强制刷新可以穿透熔断
	forceReport, err := guard.EvaluateAndSync(context.Background(), droppedNodes, QuotaTelemetry{}, true, reloadFn)
	if err != nil {
		t.Fatalf("unexpected error on force sync: %v", err)
	}
	if forceReport.Outcome != OutcomeAppliedAndReload {
		t.Errorf("expected OutcomeAppliedAndReload on forced reload, got %v", forceReport.Outcome)
	}
	if !reloaded {
		t.Error("reloadFn should have been called on forced reload")
	}
}

func TestAutoSyncGuard_SkippedUnchanged(t *testing.T) {
	guard := NewAutoSyncGuard()
	reloadedCount := 0
	reloadFn := func(ctx context.Context, nodes []schema.ProxyNode) error {
		reloadedCount++
		return nil
	}

	nodes := []schema.ProxyNode{
		{Name: "HK-01", Server: "1.2.3.4", Port: 443},
		{Name: "US-01", Server: "5.6.7.8", Port: 443},
	}

	// 第一次应用
	rep1, err := guard.EvaluateAndSync(context.Background(), nodes, QuotaTelemetry{}, false, reloadFn)
	if err != nil || rep1.Outcome != OutcomeAppliedAndReload {
		t.Fatalf("first sync failed: rep=%v, err=%v", rep1, err)
	}
	if reloadedCount != 1 {
		t.Fatalf("expected 1 reload, got %d", reloadedCount)
	}

	// 第二次同步相同节点列表
	rep2, err := guard.EvaluateAndSync(context.Background(), nodes, QuotaTelemetry{}, false, reloadFn)
	if err != nil {
		t.Fatalf("second sync error: %v", err)
	}
	if rep2.Outcome != OutcomeSkippedUnchanged {
		t.Errorf("expected OutcomeSkippedUnchanged on identical nodes, got %v", rep2.Outcome)
	}
	if reloadedCount != 1 {
		t.Errorf("reloadFn should not be invoked when nodes are unchanged, count=%d", reloadedCount)
	}
}
