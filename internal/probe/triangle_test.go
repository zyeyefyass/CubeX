package probe

import (
	"testing"
)

func TestTriangleEngine(t *testing.T) {
	te := GetTriangleEngine()

	// 1. Test Latency & Jitter ring buffer
	node := "Test-Node-1"
	// Feed 10 samples
	delays := []int{60, 65, 55, 70, 62, 58, 63, 61, 59, 64}
	for _, d := range delays {
		te.RecordLatency(node, d)
	}

	snap := te.GetNodeScore(node)
	if snap.DelayCount != 10 {
		t.Fatalf("expected 10 delay samples, got %d", snap.DelayCount)
	}
	if snap.MedianDelayMs < 58 || snap.MedianDelayMs > 65 {
		t.Fatalf("unexpected median delay: %d", snap.MedianDelayMs)
	}
	if snap.MeanJitterMs > 15 {
		t.Fatalf("unexpected mean jitter: %d", snap.MeanJitterMs)
	}
	if snap.TotalScore < 70 {
		t.Fatalf("expected high score, got %f", snap.TotalScore)
	}

	// 2. Test Passive Bandwidth: Peak-Hold
	// Small packet (< 256KB) should not update peak
	te.RecordPassiveTraffic(node, 100*1024, 500)
	snap2 := te.GetNodeScore(node)
	if snap2.PeakBandwidthKbps != 0 {
		t.Fatalf("small packet should not update peak bandwidth, got %d", snap2.PeakBandwidthKbps)
	}

	// Large burst (10MB in 500ms = ~163840 Kbps = 160Mbps)
	te.RecordPassiveTraffic(node, 10*1024*1024, 500)
	snap3 := te.GetNodeScore(node)
	if snap3.PeakBandwidthKbps == 0 {
		t.Fatalf("expected peak bandwidth update, got 0")
	}
	if snap3.BandwidthScore < 90 {
		t.Fatalf("expected high bandwidth score for 160Mbps, got %f", snap3.BandwidthScore)
	}

	// 3. Test Manual Burst Speed (2MB)
	te.SetBurstSpeed(node, 120.0) // 120Mbps
	snap4 := te.GetNodeScore(node)
	if snap4.TotalScore < 80 {
		t.Fatalf("expected total score >= 80, got %f", snap4.TotalScore)
	}
}
