package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestHandleNetstackAdvantages(t *testing.T) {
	cfgIR := &schema.ConfigIR{}
	engMgr := engine.NewManager(t.TempDir(), cfgIR)
	s := &Server{
		engMgr: engMgr,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/netstack/advantages", nil)
	w := httptest.NewRecorder()

	s.handleNetstackAdvantages(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data["success"] != true {
		t.Fatalf("expected success to be true, got %v", data["success"])
	}

	advs, ok := data["advantages"].([]any)
	if !ok || len(advs) != 5 {
		t.Fatalf("expected 5 advantage dimensions, got %d", len(advs))
	}

	for _, raw := range advs {
		adv, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("invalid item type")
		}
		if adv["probe_status"] != "VERIFIED_ACTIVE" {
			t.Errorf("expected probe_status to be VERIFIED_ACTIVE, got %v", adv["probe_status"])
		}
		if adv["dimension"] == "" {
			t.Errorf("empty dimension")
		}
	}
}
