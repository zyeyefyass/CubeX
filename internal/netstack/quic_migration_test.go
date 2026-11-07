package netstack

import (
	"testing"
	"time"
)

func TestQUICMigrationManager(t *testing.T) {
	mgr := NewQUICMigrationManager()

	// 1. Initial Packet
	migrated := mgr.RecordPacket("cid-123456", "192.168.1.100", 54321, 15.2)
	if migrated {
		t.Errorf("expected false for initial packet, got true")
	}

	status := mgr.GetStatus()
	if status.ActiveSessions != 1 {
		t.Errorf("expected 1 active session, got %d", status.ActiveSessions)
	}

	// 2. Wi-Fi Roaming migration (IP change from 192.168.1.100 to 192.168.1.101)
	migrated = mgr.RecordPacket("cid-123456", "192.168.1.101", 54322, 12.0)
	if !migrated {
		t.Errorf("expected true on roaming migration, got false")
	}

	status = mgr.GetStatus()
	if status.TotalMigrations != 1 {
		t.Errorf("expected 1 total migration, got %d", status.TotalMigrations)
	}

	// 3. 0-RTT Ticket Cache & Retrieval
	sampleTicket := []byte("tls13-quic-sample-ticket-data")
	mgr.CacheZeroRTTTicket("example.com", sampleTicket, 1*time.Hour)

	cached, ok := mgr.GetZeroRTTTicket("example.com")
	if !ok || string(cached) != string(sampleTicket) {
		t.Errorf("failed to retrieve cached 0-RTT ticket")
	}

	status = mgr.GetStatus()
	if status.ZeroRTTHits != 1 {
		t.Errorf("expected 1 zero RTT hit, got %d", status.ZeroRTTHits)
	}

	// 4. FEC recovery
	mgr.RecordFECRecovery(5)
	status = mgr.GetStatus()
	if status.FECRecoveredPackets != 5 {
		t.Errorf("expected 5 FEC recovered packets, got %d", status.FECRecoveredPackets)
	}
}
