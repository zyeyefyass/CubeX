package netstack

import (
	"testing"
)

func TestQUICPassthroughManager(t *testing.T) {
	mgr := NewQUICPassthroughManager(nil)

	// Pristine network condition -> passthrough
	dec1 := mgr.RecordPacket("flow-1", "10.0.0.100", "quic.cloudflare.com", 443, 1420, 0.01, 15.2, 2.1)
	if dec1 != QUICModePassthrough {
		t.Fatalf("expected passthrough for pristine flow, got: %s", dec1)
	}

	// Degraded network condition (high loss) -> automatic fallback to TCP
	dec2 := mgr.RecordPacket("flow-2", "10.0.0.101", "youtube-ui.l.google.com", 443, 1420, 0.08, 65.4, 25.0)
	if dec2 != QUICModeTCPFallback {
		t.Fatalf("expected tcp_fallback for high loss flow, got: %s", dec2)
	}

	metrics := mgr.GetMetrics()
	if metrics.ActiveFlowsCount != 2 {
		t.Fatalf("expected 2 active flows, got: %d", metrics.ActiveFlowsCount)
	}
	if metrics.TotalFallbackCount != 1 {
		t.Fatalf("expected 1 fallback event, got: %d", metrics.TotalFallbackCount)
	}

	// Change mode to forced passthrough
	mgr.SetMode(QUICModePassthrough)
	dec3 := mgr.RecordPacket("flow-2", "10.0.0.101", "youtube-ui.l.google.com", 443, 1420, 0.12, 80.0, 30.0)
	if dec3 != QUICModePassthrough {
		t.Fatalf("expected forced passthrough, got: %s", dec3)
	}
}
