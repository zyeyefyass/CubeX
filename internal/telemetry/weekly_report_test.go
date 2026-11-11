package telemetry

import (
	"testing"
)

func TestWeeklyReportManager(t *testing.T) {
	mgr := GetWeeklyReportManager()

	rep := mgr.GenerateWeeklyReport()
	if rep.StabilityScore < 99.0 {
		t.Errorf("expected high stability score, got %f", rep.StabilityScore)
	}

	if rep.TopDevices == nil {
		t.Errorf("expected non-nil top devices slice")
	}

	if len(rep.Highlights) == 0 {
		t.Errorf("expected non-empty highlights")
	}

	rep2 := mgr.GetReport()
	if rep2.DateRange != rep.DateRange {
		t.Errorf("mismatch in date range")
	}
}
