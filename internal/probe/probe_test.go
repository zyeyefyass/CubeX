package probe

import (
	"context"
	"testing"
	"time"
)

func TestWatchdogPoolExecution(t *testing.T) {
	pool := NewWatchdogPool(4, 500*time.Millisecond)

	// Test normal execution
	err := pool.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Test panic recovery
	err = pool.Execute(context.Background(), func(ctx context.Context) error {
		panic("simulated worker crash")
	})
	if err == nil {
		t.Fatalf("expected error from panic, got nil")
	}

	// Test timeout
	err = pool.Execute(context.Background(), func(ctx context.Context) error {
		time.Sleep(1 * time.Second)
		return nil
	})
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}

func TestBandwidthQuotaManager(t *testing.T) {
	cfg := QuotaConfig{
		MaxChunkBytes:   100,
		DailyQuotaBytes: 250,
		CacheTTL:        1 * time.Hour,
	}
	p := NewBandwidthProber(cfg)

	used, limit := p.GetDailyUsage()
	if used != 0 || limit != 250 {
		t.Fatalf("initial usage mismatch: got %d, %d", used, limit)
	}
}

func TestEvaluateNodeScore(t *testing.T) {
	// S-Grade Node: low latency, 100% SLA, high throughput, residential IP
	resS := EvaluateNodeScore("HK-VIP-01", 45, 100.0, 150.0, "Residential", 0)
	if resS.Grade != "S 极品金牌" || !resS.IsRecommended {
		t.Fatalf("expected S grade and recommended, got %s (score: %.1f)", resS.Grade, resS.TotalScore)
	}

	// C-Grade Node: high latency, poor SLA, datacenter with high risk
	resC := EvaluateNodeScore("US-Slow-09", 1200, 20.0, 1.0, "Datacenter", 80)
	if resC.Grade != "C 备用降级" || resC.IsRecommended {
		t.Fatalf("expected C grade and not recommended, got %s (score: %.1f)", resC.Grade, resC.TotalScore)
	}
}
