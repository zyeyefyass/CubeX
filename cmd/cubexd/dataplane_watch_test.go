package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestDataplaneHealthGracePeriodAndSnapshot(t *testing.T) {
	tempDir := t.TempDir()
	engMgr := engine.NewManager(tempDir, schema.NewDefaultIR())

	srv := &Server{
		engMgr: engMgr,
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz/dataplane", nil)
	handler := srv.healthzDataplaneHandler()

	// 1. Initial starting snapshot fallback
	wInit := httptest.NewRecorder()
	handler.ServeHTTP(wInit, req)
	if wInit.Code != http.StatusOK {
		t.Errorf("expected 200 OK initially, got %d", wInit.Code)
	}

	// 2. Normal healthy snapshot
	srv.dataplaneSnapshot.Store(&DataplaneHealthSnapshot{
		Status:       "ok",
		State:        "healthy",
		Healthy:      true,
		ActiveEngine: "sing-box",
		ClashAPI:     true,
		DNSService:   true,
		CheckedAt:    time.Now().Format(time.RFC3339),
	})

	wNormal := httptest.NewRecorder()
	handler.ServeHTTP(wNormal, req)
	if wNormal.Code != http.StatusOK {
		t.Errorf("expected 200 OK for healthy snapshot, got %d", wNormal.Code)
	}

	// 3. Degraded snapshot (unhealthy) returns 503
	srv.dataplaneSnapshot.Store(&DataplaneHealthSnapshot{
		Status:       "degraded",
		State:        "degraded",
		Healthy:      false,
		ActiveEngine: "sing-box",
		ClashAPI:     false,
		DNSService:   false,
		CheckedAt:    time.Now().Format(time.RFC3339),
	})

	wDegraded := httptest.NewRecorder()
	handler.ServeHTTP(wDegraded, req)
	if wDegraded.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for degraded snapshot, got %d", wDegraded.Code)
	}

	// 4. In grace period, even if snapshot was degraded, it returns 200 OK with state=switching
	// Setting switching grace deadline in manager
	engMgr.SetSwitchingGraceDeadline(time.Now().Add(1500 * time.Millisecond))
	wGrace := httptest.NewRecorder()
	handler.ServeHTTP(wGrace, req)
	if wGrace.Code != http.StatusOK {
		t.Errorf("expected 200 OK during grace period, got %d", wGrace.Code)
	}
	var resGrace map[string]any
	_ = json.Unmarshal(wGrace.Body.Bytes(), &resGrace)
	if resGrace["state"] != "switching" {
		t.Errorf("expected state=switching during grace period, got %v", resGrace["state"])
	}
	if resGrace["healthy"] != true {
		t.Errorf("expected healthy=true during grace period, got %v", resGrace["healthy"])
	}
}

func TestDataplaneHysteresisRecoveryStrict(t *testing.T) {
	// Verify strict 3-success recovery & 2-failure hysteresis state machine
	isHealthy := false
	successStreak := 0
	failStreak := 2

	stepHysteresis := func(probePass bool) {
		if probePass {
			failStreak = 0
			successStreak++
			if !isHealthy && successStreak >= 3 {
				isHealthy = true
			}
		} else {
			successStreak = 0
			failStreak++
			if isHealthy && failStreak >= 2 {
				isHealthy = false
			}
		}
	}

	// 1st success after failure: must remain false
	stepHysteresis(true)
	if isHealthy {
		t.Error("expected isHealthy to remain false after 1st success")
	}
	if successStreak != 1 {
		t.Errorf("expected successStreak=1, got %d", successStreak)
	}

	// 2nd success: must remain false
	stepHysteresis(true)
	if isHealthy {
		t.Error("expected isHealthy to remain false after 2nd success")
	}
	if successStreak != 2 {
		t.Errorf("expected successStreak=2, got %d", successStreak)
	}

	// 3rd consecutive success: transitions to true
	stepHysteresis(true)
	if !isHealthy {
		t.Error("expected isHealthy to become true after 3rd consecutive success")
	}
	if successStreak != 3 {
		t.Errorf("expected successStreak=3, got %d", successStreak)
	}

	// Micro-jitter: 1 failure should NOT immediately trigger degraded
	stepHysteresis(false)
	if !isHealthy {
		t.Error("expected isHealthy to remain true after single probe jitter")
	}
	if failStreak != 1 {
		t.Errorf("expected failStreak=1, got %d", failStreak)
	}

	// 2nd consecutive failure: transitions to false
	stepHysteresis(false)
	if isHealthy {
		t.Error("expected isHealthy to become false after 2nd consecutive failure")
	}
	if failStreak != 2 {
		t.Errorf("expected failStreak=2, got %d", failStreak)
	}
}

func TestProbeDNSLiveness(t *testing.T) {
	// Start mock UDP DNS server
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen UDP: %v", err)
	}
	defer pc.Close()

	port := pc.LocalAddr().(*net.UDPAddr).Port

	// Mock server echo reply
	go func() {
		buf := make([]byte, 512)
		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			reply := make([]byte, n)
			copy(reply, buf[:n])
			if len(reply) >= 3 {
				reply[2] |= 0x80 // Set QR=1 (response)
			}
			_, _ = pc.WriteTo(reply, addr)
		}
	}()

	// Test 1: Live DNS server returns true
	if !probeDNSLiveness("127.0.0.1", port, 300*time.Millisecond) {
		t.Error("expected probeDNSLiveness=true for responding DNS server")
	}

	// Test 2: Closed port returns false
	if probeDNSLiveness("127.0.0.1", port+1000, 100*time.Millisecond) {
		t.Error("expected probeDNSLiveness=false for non-existent port")
	}
}
