package dataplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zyeyefyass/CubeX/internal/clashapi"
)

func TestStaleFlowKiller_KillNodeConnections(t *testing.T) {
	var mu sync.Mutex
	deletedIDs := make([]string, 0)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/connections" {
			resp := clashapi.ConnectionsResponse{
				Connections: []clashapi.ConnectionEntry{
					{
						ID:     "conn-1",
						Chains: []string{"Proxy", "HK-01"},
						Upload: 2048, Download: 4096,
					},
					{
						ID:     "conn-2",
						Chains: []string{"DIRECT"},
						Upload: 500, Download: 10000,
					},
					{
						ID:     "conn-3",
						Chains: []string{"Proxy", "JP-02"},
						Upload: 1200, Download: 3000,
					},
					{
						ID:     "conn-4",
						Chains: []string{"Proxy", "HK-01"},
						Upload: 4096, Download: 0,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == http.MethodDelete && len(r.URL.Path) > len("/connections/") {
			id := r.URL.Path[len("/connections/"):]
			mu.Lock()
			deletedIDs = append(deletedIDs, id)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	killer := NewStaleFlowKiller(ts.URL, "secret123")
	killed, err := killer.KillNodeConnections(context.Background(), "HK-01")
	if err != nil {
		t.Fatalf("KillNodeConnections failed: %v", err)
	}

	if killed != 2 {
		t.Fatalf("expected 2 killed connections, got %d", killed)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(deletedIDs) != 2 {
		t.Fatalf("expected 2 deleted IDs, got %d", len(deletedIDs))
	}
	deletedMap := make(map[string]bool)
	for _, id := range deletedIDs {
		deletedMap[id] = true
	}
	if !deletedMap["conn-1"] || !deletedMap["conn-4"] {
		t.Fatalf("unexpected deleted IDs: %v", deletedIDs)
	}
}

func TestStaleFlowKiller_KillStalledConnections(t *testing.T) {
	var mu sync.Mutex
	deletedIDs := make([]string, 0)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/connections" {
			resp := clashapi.ConnectionsResponse{
				Connections: []clashapi.ConnectionEntry{
					{
						ID:     "conn-stalled",
						Chains: []string{"Proxy", "HK-01"},
						Upload: 5000, Download: 0, // Stalled!
					},
					{
						ID:     "conn-normal",
						Chains: []string{"Proxy", "HK-01"},
						Upload: 5000, Download: 10000, // Normal
					},
					{
						ID:     "conn-direct",
						Chains: []string{"DIRECT"},
						Upload: 5000, Download: 0, // Direct should NOT be killed by proxy stall detector
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == http.MethodDelete && len(r.URL.Path) > len("/connections/") {
			id := r.URL.Path[len("/connections/"):]
			mu.Lock()
			deletedIDs = append(deletedIDs, id)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	killer := NewStaleFlowKiller(ts.URL, "secret123")
	killed, err := killer.KillStalledConnections(context.Background())
	if err != nil {
		t.Fatalf("KillStalledConnections failed: %v", err)
	}

	if killed != 1 {
		t.Fatalf("expected 1 killed connection, got %d", killed)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(deletedIDs) != 1 || deletedIDs[0] != "conn-stalled" {
		t.Fatalf("unexpected deleted IDs: %v", deletedIDs)
	}
}

func TestStaleFlowKiller_DeleteBatchCancelsCleanly(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	killer := NewStaleFlowKiller(ts.URL, "")
	ids := make([]string, 32)
	for i := range ids {
		ids[i] = "id"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()

	start := time.Now()
	_ = killer.deleteBatch(ctx, ids)
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("deleteBatch blocked too long after ctx cancel: %v", elapsed)
	}
}

func TestStaleFlowKiller_BareHostPortAndErrorPropagation(t *testing.T) {
	srvOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/connections" {
			_ = json.NewEncoder(w).Encode(clashapi.ConnectionsResponse{
				Connections: []clashapi.ConnectionEntry{{ID: "c-bare", Chains: []string{"HK-01"}}},
			})
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srvOK.Close()

	killer := NewStaleFlowKiller(strings.TrimPrefix(srvOK.URL, "http://"), "")
	killed, err := killer.KillNodeConnections(context.Background(), "HK-01")
	if err != nil || killed != 1 {
		t.Fatalf("expected bare host:port KillNodeConnections=1, got killed=%d err=%v", killed, err)
	}

	srvErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srvErr.Close()

	killerErr := NewStaleFlowKiller(srvErr.URL, "")
	if _, err := killerErr.KillNodeConnections(context.Background(), "HK-01"); err == nil {
		t.Fatalf("expected explicit error when Clash API returns 500")
	}
}
