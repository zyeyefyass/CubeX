package probe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestEWMAScheduler_ReadinessGateAndStartupGrace(t *testing.T) {
	var ready atomic.Bool
	var hits atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proxies": map[string]any{
				"🇭🇰 香港自动": map[string]any{
					"name": "🇭🇰 香港自动",
					"type": "URLTest",
					"now":  "HK-01",
					"all":  []string{"HK-01"},
				},
				"HK-01": map[string]any{"name": "HK-01", "type": "VLESS"},
			},
		})
	}))
	defer srv.Close()

	sched := NewEWMAScheduler(SchedulerConfig{
		ClashAPIBase:   srv.URL,
		TickInterval:   20 * time.Millisecond,
		StartupGrace:   5 * time.Second,
		ReadinessCheck: ready.Load,
	}, NewEWMAManager(), nil)

	// 1. While readiness gate is false, RunSingleCycle must NOT hit Clash API
	sched.RunSingleCycle(context.Background())
	if hits.Load() != 0 {
		t.Fatalf("expected 0 Clash API requests before core is ready, got %d", hits.Load())
	}

	// 2. Once ready becomes true, RunSingleCycle proceeds normally
	ready.Store(true)
	sched.RunSingleCycle(context.Background())
	if hits.Load() == 0 {
		t.Fatalf("expected Clash API request once readiness gate opened")
	}

	// 3. During startup grace against closed port, transient errors enter exponential backoff without panicking
	closedSched := NewEWMAScheduler(SchedulerConfig{
		ClashAPIBase: "http://127.0.0.1:1",
		ProbeTimeout: 50 * time.Millisecond,
		StartupGrace: 30 * time.Second,
	}, NewEWMAManager(), nil)
	closedSched.RunSingleCycle(context.Background())

	closedSched.mu.Lock()
	fails := closedSched.consecutiveFails
	backoff := closedSched.backoffTicks
	closedSched.mu.Unlock()
	if fails != 1 || backoff != 1 {
		t.Fatalf("expected consecutiveFails=1 backoffTicks=1 after transient error, got fails=%d backoff=%d", fails, backoff)
	}
}
