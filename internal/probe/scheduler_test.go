package probe

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSelectTieredBatch100Nodes(t *testing.T) {
	ewma := NewEWMAManager()
	sched := NewEWMAScheduler(SchedulerConfig{
		HotStandbyCount: 3,
		ColdBatchSize:   2,
	}, ewma, nil)

	// Build 100 nodes
	candidates := make([]string, 100)
	for i := 0; i < 100; i++ {
		candidates[i] = fmt.Sprintf("Node-%03d", i)
		// Give some nodes EWMA scores
		if i < 10 {
			ewma.RecordSample(candidates[i], 50+i*5, true)
		}
	}

	active := "Node-000"

	// Round 1
	batch1 := sched.SelectTieredBatch("ProxyGroup", active, candidates)
	// Expect active (1) + hot (top 3) + cold (2) <= 6
	if len(batch1) > 6 {
		t.Fatalf("batch1 size %d > 6", len(batch1))
	}
	if batch1[0] != active {
		t.Errorf("batch1 active sentinel mismatch: got %s, want %s", batch1[0], active)
	}

	// Verify cursor rotation over multiple rounds
	seenCold := make(map[string]bool)
	for round := 0; round < 50; round++ {
		batch := sched.SelectTieredBatch("ProxyGroup", active, candidates)
		for _, n := range batch {
			seenCold[n] = true
		}
	}

	// Over 50 rounds, 50 * 2 = 100 cold candidates should all have been touched
	if len(seenCold) < 80 {
		t.Errorf("expected at least 80 unique nodes sampled across 50 rounds, got %d", len(seenCold))
	}
}

func TestDualModeFailoverAndHysteresis(t *testing.T) {
	ewma := NewEWMAManager()
	var switchedEvents []SwitchEvent
	sched := NewEWMAScheduler(SchedulerConfig{
		SwitchMarginMs:    25,
		SwitchMarginRatio: 0.18,
		SwitchCooldown:    60 * time.Second,
	}, ewma, func(ev SwitchEvent) {
		switchedEvents = append(switchedEvents, ev)
	})

	ctx := context.Background()

	// Scenario 1: Active node is 80ms, Candidate is 72ms (only 8ms diff < 25ms margin)
	// Hysteresis must firmly maintain stability without switching
	ewma.RecordSample("Node-Active", 80, true)
	ewma.RecordSample("Node-Cand", 72, true)

	sched.EvaluateAndSwitchGroup(ctx, "ProxyGroup", "Node-Active", []string{"Node-Active", "Node-Cand"})
	if len(switchedEvents) > 0 {
		t.Fatalf("expected 0 switches for minor 8ms advantage, got %d", len(switchedEvents))
	}

	// Scenario 2: Active node suffers 2 consecutive timeouts (Emergency Fast-Failover)
	ewma.RecordSample("Node-Active", 0, false)
	ewma.RecordSample("Node-Active", 0, false)

	sched.EvaluateAndSwitchGroup(ctx, "ProxyGroup", "Node-Active", []string{"Node-Active", "Node-Cand"})
	if len(switchedEvents) != 1 {
		t.Fatalf("expected 1 emergency fast-failover, got %d", len(switchedEvents))
	}
	if !switchedEvents[0].Emergency {
		t.Errorf("expected switch event to be Emergency=true")
	}
	if switchedEvents[0].ToNode != "Node-Cand" {
		t.Errorf("expected switch to Node-Cand, got %s", switchedEvents[0].ToNode)
	}
}
