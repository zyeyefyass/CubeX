package netstack

import (
	"context"
	"testing"
)

func TestBBRManager(t *testing.T) {
	mgr := GetBBRManager()
	status := mgr.Detect()
	if status.CurrentAlgorithm == "" {
		t.Errorf("expected non-empty CurrentAlgorithm, got empty")
	}

	optStatus, err := mgr.Optimize()
	if err != nil {
		t.Logf("optimize returned: %v (expected on non-root or non-linux)", err)
	}
	if optStatus.CurrentAlgorithm == "" {
		t.Errorf("expected non-empty optStatus.CurrentAlgorithm")
	}
}

func TestHappyEyeballsDialer(t *testing.T) {
	dialer := GetHappyEyeballsDialer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Direct IP test (loopback)
	conn, err := dialer.DialContext(ctx, "tcp", "127.0.0.1:2026")
	if err != nil {
		t.Logf("dial 127.0.0.1:2026 err: %v (expected if service not bound on 127.0.0.1 on Windows)", err)
	} else {
		conn.Close()
	}

	telemetry := dialer.GetTelemetry()
	if telemetry.TotalAttempts == 0 {
		t.Errorf("expected TotalAttempts > 0")
	}
}
