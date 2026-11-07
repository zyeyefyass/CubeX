package netstack

import (
	"testing"
)

func TestJitterRadarManager(t *testing.T) {
	mgr := GetJitterRadarManager()

	rep := mgr.RunHopProbe()
	if len(rep.Hops) != 4 {
		t.Errorf("expected 4 hops, got %d", len(rep.Hops))
	}

	if rep.HealthRating == "" {
		t.Errorf("expected non-empty health rating")
	}

	mgr.RecordMicroBurstDrop(3)
	rep2 := mgr.GetReport()
	if rep2.MicroBurstPacketDrops != 3 {
		t.Errorf("expected 3 micro burst drops, got %d", rep2.MicroBurstPacketDrops)
	}
}
