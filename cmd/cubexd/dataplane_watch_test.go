package main

import (
	"encoding/json"
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
