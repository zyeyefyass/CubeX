package sentinel

import (
	"os"
	"testing"
	"time"
)

func TestSentinelEngine(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cubex-sentinel-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s := GetSentinel(tmpDir)
	s.Start(100 * time.Millisecond)

	s.RegisterHealers(
		func() error { return nil },
		func() error { return nil },
		func() error { return nil },
	)

	s.RunCheck()

	rec := s.TriggerManualHeal()
	if rec.Result != "SUCCESS" {
		t.Errorf("expected SUCCESS heal, got %s", rec.Result)
	}

	status := s.GetStatus()
	if status.TotalSelfHeals == 0 {
		t.Errorf("expected TotalSelfHeals > 0")
	}
	if len(status.RecentEvents) == 0 {
		t.Errorf("expected at least 1 recent event")
	}

	s.Stop()
}
