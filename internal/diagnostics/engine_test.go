package diagnostics

import (
	"testing"
)

func TestDiagnosticsEngine(t *testing.T) {
	eng := NewEngine()
	rep := eng.RunFullDiagnostic("127.0.0.1", "127.0.0.1:7890", "mihomo")

	if len(rep.Checks) < 6 {
		t.Fatalf("expected at least 6 diagnostic checks, got %d", len(rep.Checks))
	}
	if rep.Score <= 0 || rep.Score > 100 {
		t.Fatalf("unexpected score: %d", rep.Score)
	}
	if rep.Summary == "" {
		t.Fatalf("summary should not be empty")
	}

	cached := eng.GetCachedReport()
	if cached == nil || cached.Score != rep.Score {
		t.Fatalf("cached report mismatch")
	}
}
