package host

import (
	"os"
	"testing"
)

func TestShadowAuditorManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cubex_shadow_auditor_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewShadowAuditorManager(tmpDir)
	defer mgr.Close()

	status := mgr.GetStatus()
	if !status.Enabled {
		t.Errorf("expected shadow auditor to be enabled by default")
	}
	if len(status.Items) == 0 {
		t.Errorf("expected non-empty shadow audit items")
	}
	if status.ConsistencyScorePct < 100.0 {
		t.Errorf("expected 100.0 consistency score initially")
	}

	// Test RunImmediateAudit
	statusAudited := mgr.RunImmediateAudit()
	if statusAudited.TotalAuditsExecuted <= status.TotalAuditsExecuted {
		t.Errorf("expected total audits executed to increment")
	}

	// Test Toggle
	statusDisabled := mgr.Toggle(false)
	if statusDisabled.Enabled {
		t.Errorf("expected auditor to be disabled after toggle")
	}

	statusReEnabled := mgr.Toggle(true)
	if !statusReEnabled.Enabled {
		t.Errorf("expected auditor to be enabled after re-toggle")
	}
}
