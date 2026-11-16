package probe

import (
	"sync"
	"testing"
	"time"
)

func TestPassiveCircuitBreaker_TripAndRecovery(t *testing.T) {
	tripCh := make(chan string, 1)
	recoverCh := make(chan string, 1)

	cb := NewPassiveCircuitBreaker(
		3,             // 3 次失败即熔断
		50*time.Millisecond, // 50ms 极短冷却期便于单测
		func(node, reason string) {
			tripCh <- node
		},
		func(node string) {
			recoverCh <- node
		},
	)

	node := "HK-BGP-01"

	// 初始状态非隔离
	if cb.IsNodeQuarantined(node) {
		t.Fatal("expected node not to be quarantined initially")
	}

	// 连续 2 次失败，未达阈值 3
	cb.RecordFailure(node, "timeout 1")
	cb.RecordFailure(node, "timeout 2")
	if cb.IsNodeQuarantined(node) {
		t.Fatal("node should not be quarantined after 2 failures")
	}

	// 第 3 次失败，触发熔断
	tripped := cb.RecordFailure(node, "timeout 3")
	if !tripped {
		t.Fatal("expected breaker to trip on 3rd failure")
	}

	select {
	case trippedNode := <-tripCh:
		if trippedNode != node {
			t.Fatalf("expected trip callback for %s, got %s", node, trippedNode)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for trip callback")
	}

	// 此时应处于隔离状态
	if !cb.IsNodeQuarantined(node) {
		t.Fatal("expected node to be quarantined")
	}

	// 等待 60ms 冷却期过
	time.Sleep(60 * time.Millisecond)

	// 再次查询应自动转为 Half-Open，放行试探
	if cb.IsNodeQuarantined(node) {
		t.Fatal("expected node to allow half-open probing after quarantine expiry")
	}

	// 收到试探成功响应
	cb.RecordSuccess(node, 1024)

	select {
	case recoveredNode := <-recoverCh:
		if recoveredNode != node {
			t.Fatalf("expected recover callback for %s, got %s", node, recoveredNode)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for recover callback")
	}

	// 最终恢复 Closed
	if cb.IsNodeQuarantined(node) {
		t.Fatal("expected node to be fully recovered")
	}
}

func TestPassiveCircuitBreaker_ConcurrencyRace(t *testing.T) {
	cb := NewPassiveCircuitBreaker(5, 100*time.Millisecond, nil, nil)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			node := "JP-Tokyo-02"
			for j := 0; j < 50; j++ {
				if j%3 == 0 {
					cb.RecordFailure(node, "simulated RST")
				} else {
					cb.RecordSuccess(node, 512)
				}
				_ = cb.IsNodeQuarantined(node)
			}
		}(i)
	}

	wg.Wait()
}
