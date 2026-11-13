package probe

import (
	"testing"
	"time"
)

func TestEWMARecordAndConvergence(t *testing.T) {
	mgr := NewEWMAManager()

	// Initial sample 100ms
	res1 := mgr.RecordSample("HK-01", 100, true)
	if res1.SRTT != 100.0 {
		t.Fatalf("expected initial SRTT 100, got %f", res1.SRTT)
	}
	if res1.RTTVAR != 50.0 {
		t.Fatalf("expected initial RTTVAR 50, got %f", res1.RTTVAR)
	}

	// Repeated identical 100ms samples should converge RTTVAR to near zero
	for i := 0; i < 20; i++ {
		res1 = mgr.RecordSample("HK-01", 100, true)
	}

	if res1.SRTT != 100.0 {
		t.Fatalf("expected converged SRTT 100, got %f", res1.SRTT)
	}
	if res1.RTTVAR > 1.0 {
		t.Fatalf("expected converged RTTVAR near 0, got %f", res1.RTTVAR)
	}
	if res1.Grade != "S 极品稳态" && res1.Grade != "A+ 优质平稳" {
		t.Fatalf("expected S or A+ grade, got %s", res1.Grade)
	}
}

func TestEWMAAntiFlappingMargin(t *testing.T) {
	mgr := NewEWMAManager()

	// Current active node: stable at 60ms
	for i := 0; i < 5; i++ {
		mgr.RecordSample("Active-HK", 60, true)
	}

	// Candidate node: 52ms (only 8ms faster, ratio ~13%, within default margin of 25ms and 15%)
	for i := 0; i < 5; i++ {
		mgr.RecordSample("Cand-JP", 52, true)
	}

	shouldSwitch, reason := mgr.ShouldSwitch("Active-HK", "Cand-JP", 25, 0.15, 0)
	if shouldSwitch {
		t.Fatalf("expected switch suppression due to anti-flapping margin, but got switch=true, reason: %s", reason)
	}

	// Significant improvement: candidate 25ms (35ms faster, ratio > 50%)
	for i := 0; i < 5; i++ {
		mgr.RecordSample("Cand-SG", 25, true)
	}

	shouldSwitch2, reason2 := mgr.ShouldSwitch("Active-HK", "Cand-SG", 25, 0.15, 0)
	if !shouldSwitch2 {
		t.Fatalf("expected switch=true for substantial improvement, but got false, reason: %s", reason2)
	}
}

func TestEWMACooldownSuppression(t *testing.T) {
	mgr := NewEWMAManager()

	for i := 0; i < 5; i++ {
		mgr.RecordSample("Node-A", 100, true)
		mgr.RecordSample("Node-B", 40, true)
	}

	// First switch from A to B should succeed
	sw1, _ := mgr.ShouldSwitch("Node-A", "Node-B", 20, 0.15, 5*time.Second)
	if !sw1 {
		t.Fatalf("expected first switch to succeed")
	}

	// Immediate next switch should be blocked by cooldown
	for i := 0; i < 5; i++ {
		mgr.RecordSample("Node-A", 20, true)
	}
	sw2, reason2 := mgr.ShouldSwitch("Node-B", "Node-A", 15, 0.15, 5*time.Second)
	if sw2 {
		t.Fatalf("expected immediate second switch to be blocked by cooldown, got reason: %s", reason2)
	}
}

func TestEWMAFailureImmediateFailover(t *testing.T) {
	mgr := NewEWMAManager()

	mgr.RecordSample("Failing-Node", 50, true)
	mgr.RecordSample("Failing-Node", 0, false)
	mgr.RecordSample("Failing-Node", 0, false) // 2 consecutive failures

	mgr.RecordSample("Backup-Node", 60, true)

	sw, reason := mgr.ShouldSwitch("Failing-Node", "Backup-Node", 25, 0.15, 10*time.Second)
	if !sw {
		t.Fatalf("expected immediate failover when active node fails, reason: %s", reason)
	}
}

func BenchmarkEWMARecordSample(b *testing.B) {
	mgr := NewEWMAManager()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		delay := 50 + (i % 20)
		mgr.RecordSample("Benchmark-Node", delay, true)
	}
}

func BenchmarkEWMAShouldSwitch(b *testing.B) {
	mgr := NewEWMAManager()
	for i := 0; i < 10; i++ {
		mgr.RecordSample("Node-1", 60, true)
		mgr.RecordSample("Node-2", 40, true)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = mgr.ShouldSwitch("Node-1", "Node-2", 25, 0.15, 0)
	}
}
