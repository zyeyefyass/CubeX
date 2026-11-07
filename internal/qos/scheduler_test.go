package qos

import (
	"testing"
)

func TestSchedulerClassification(t *testing.T) {
	s := NewScheduler("")

	// Test Critical: DNS port 53
	p := s.ClassifyPacket(53, "udp", 64, "10.0.0.100")
	if p != PriorityCritical {
		t.Fatalf("expected critical for port 53, got %s", p)
	}

	// Test Critical: Game UDP small packet
	p = s.ClassifyPacket(3074, "udp", 120, "10.0.0.100")
	if p != PriorityCritical {
		t.Fatalf("expected critical for small UDP packet, got %s", p)
	}

	// Test Interactive: HTTPS small packet
	p = s.ClassifyPacket(443, "tcp", 512, "10.0.0.100")
	if p != PriorityInteractive {
		t.Fatalf("expected interactive for port 443 small packet, got %s", p)
	}

	// Test Bulk: Large payload
	p = s.ClassifyPacket(8080, "tcp", 1460, "10.0.0.100")
	if p != PriorityBulk {
		t.Fatalf("expected bulk for large payload, got %s", p)
	}

	// Test Custom Device Policy
	s.SetDevicePolicy(DevicePolicy{
		IP:       "10.0.0.88",
		Name:     "Geek Rig",
		Priority: PriorityCritical,
	})
	p = s.ClassifyPacket(8080, "tcp", 1460, "10.0.0.88")
	if p != PriorityCritical {
		t.Fatalf("expected device override to critical, got %s", p)
	}

	stats := s.GetStats()
	if stats.TotalPackets != 5 {
		t.Fatalf("expected 5 packets, got %d", stats.TotalPackets)
	}
	if !stats.BufferbloatMitigated {
		t.Fatalf("expected bufferbloat mitigation to be active")
	}
}
