package security

import (
	"testing"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestSecurityRadarAudit(t *testing.T) {
	ir := schema.NewDefaultIR()

	report := RunSecurityAudit(ir)
	if report == nil {
		t.Fatalf("expected non-nil report")
	}

	if len(report.Items) != 6 {
		t.Errorf("expected 6 security test items, got %d", len(report.Items))
	}

	if report.TotalScore <= 0 || report.TotalScore > 100 {
		t.Errorf("expected total score between 1 and 100, got %d", report.TotalScore)
	}

	if len(report.AuditLogs) == 0 {
		t.Errorf("expected audit logs to be populated")
	}
}

func TestAutoRemediateAll(t *testing.T) {
	ir := schema.NewDefaultIR()
	ir.Network.IPv6 = true // artificially make it vulnerable

	res := AutoRemediateAll(ir)
	if !res.Success {
		t.Errorf("expected remediation to succeed")
	}

	if res.NewTotalScore != 100 {
		t.Errorf("expected new score 100, got %d", res.NewTotalScore)
	}

	if ir.Network.IPv6 != false {
		t.Errorf("expected IPv6 to be disabled after remediation")
	}

	// Re-audit to verify 100% pass
	report := RunSecurityAudit(ir)
	if report.TotalScore != 100 {
		t.Errorf("expected 100 score after remediation, got %d", report.TotalScore)
	}
}
