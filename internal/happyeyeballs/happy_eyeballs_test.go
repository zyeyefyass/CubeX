package happyeyeballs

import (
	"testing"
)

func TestHappyEyeballsManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	st := mgr.GetStatus()
	if !st.Enabled {
		t.Errorf("expected enabled=true, got %v", st.Enabled)
	}

	// Domestic test: IPv6 is faster or comparable
	winnerDom := mgr.RecordRace("baidu.com", true, 20.0, 16.0)
	if winnerDom != "ipv6" {
		t.Errorf("expected domestic winner to be ipv6, got %s", winnerDom)
	}

	// Overseas test: IPv6 is significantly slower (> +50ms)
	winnerOverseas := mgr.RecordRace("github.com", false, 45.0, 120.0)
	if winnerOverseas != "ipv4" {
		t.Errorf("expected overseas winner to be ipv4 (fallback), got %s", winnerOverseas)
	}

	// Test Leakage block
	initialBlocks := mgr.GetStatus().LeakageBlockedCount
	mgr.RecordLeakBlock()
	if mgr.GetStatus().LeakageBlockedCount != initialBlocks+1 {
		t.Errorf("expected LeakageBlockedCount to increment")
	}

	// Test Toggle
	toggled := mgr.Toggle(false)
	if toggled.Enabled {
		t.Errorf("expected toggle(false) to disable")
	}

	// Test Probe
	res := mgr.RunProbe("bilibili.com")
	if res.Domain != "bilibili.com" || res.Winner == "" {
		t.Errorf("unexpected probe result: %+v", res)
	}
}
