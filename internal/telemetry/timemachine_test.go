package telemetry

import (
	"testing"
)

func TestTimeMachine(t *testing.T) {
	// Create small ring buffer of 5 points for testing
	tm := NewTimeMachine(5)

	// Record 3 points
	tm.RecordPoint(1000, 200, 800, 200, 10, 50, 12.5)
	tm.RecordPoint(2000, 400, 1600, 400, 20, 80, 11.0)
	tm.RecordPoint(1500, 300, 1200, 300, 15, 65, 13.0)

	hist := tm.Get24HHistory()
	if hist.TotalPoints != 3 {
		t.Errorf("expected 3 points, got %d", hist.TotalPoints)
	}
	if hist.PeakDownloadBps != 2000 {
		t.Errorf("expected peak dl 2000, got %d", hist.PeakDownloadBps)
	}

	// Overflow buffer to trigger ring wraparound
	tm.RecordPoint(3000, 500, 2500, 500, 25, 120, 10.0)
	tm.RecordPoint(4000, 600, 3200, 800, 30, 150, 9.5)
	tm.RecordPoint(5000, 700, 4000, 1000, 35, 180, 8.0) // 6th point, overwrites 1st

	hist2 := tm.Get24HHistory()
	if hist2.TotalPoints != 5 {
		t.Errorf("expected max 5 points, got %d", hist2.TotalPoints)
	}
	if hist2.PeakDownloadBps != 5000 {
		t.Errorf("expected peak dl 5000, got %d", hist2.PeakDownloadBps)
	}
	// Oldest should be point 2 (2000 bps)
	if hist2.Points[0].DownloadBps != 2000 {
		t.Errorf("expected oldest point 2000, got %d", hist2.Points[0].DownloadBps)
	}
	// Newest should be point 6 (5000 bps)
	if hist2.Points[4].DownloadBps != 5000 {
		t.Errorf("expected newest point 5000, got %d", hist2.Points[4].DownloadBps)
	}
}
