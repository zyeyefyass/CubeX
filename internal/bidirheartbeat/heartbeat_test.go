package bidirheartbeat

import (
	"testing"
)

func TestBidirHeartbeatManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	st := mgr.GetStatus()
	if !st.Enabled {
		t.Errorf("expected enabled=true, got %v", st.Enabled)
	}
	if st.HeartbeatIntervalMs != 20 {
		t.Errorf("expected HeartbeatIntervalMs=20, got %d", st.HeartbeatIntervalMs)
	}
	if len(st.VerifiedRules) == 0 {
		t.Errorf("expected non-empty verified rules")
	}

	// Record heartbeat
	mgr.RecordHeartbeat(1.5, true)
	if mgr.GetStatus().HeartbeatStatus != "HEALTHY" {
		t.Errorf("expected status HEALTHY")
	}

	// Toggle
	toggled := mgr.Toggle(false)
	if toggled.Enabled {
		t.Errorf("expected toggle(false) to disable")
	}

	// RunProbe
	probeRes := mgr.RunProbe()
	if !probeRes.AllRulesActive || !probeRes.HeartbeatHealthy {
		t.Errorf("unexpected probe result: %+v", probeRes)
	}

	// ApplyInstantPatch
	patchRes := mgr.ApplyInstantPatch()
	if !patchRes.Success || len(patchRes.PatchedRules) == 0 {
		t.Errorf("unexpected patch result: %+v", patchRes)
	}
}
