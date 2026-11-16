package dataplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestStaleFlowKiller_KillNodeConnections(t *testing.T) {
	var mu sync.Mutex
	deletedIDs := make([]string, 0)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/connections" {
			resp := clashConnectionsResponse{
				Connections: []struct {
					ID       string   `json:"id"`
					Upload   int64    `json:"upload"`
					Download int64    `json:"download"`
					Chains   []string `json:"chains"`
					Rule     string   `json:"rule"`
					Metadata struct {
						Network       string `json:"network"`
						SourceIP      string `json:"sourceIP"`
						DestinationIP string `json:"destinationIP"`
						Host          string `json:"host"`
					} `json:"metadata"`
				}{
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
			resp := clashConnectionsResponse{
				Connections: []struct {
					ID       string   `json:"id"`
					Upload   int64    `json:"upload"`
					Download int64    `json:"download"`
					Chains   []string `json:"chains"`
					Rule     string   `json:"rule"`
					Metadata struct {
						Network       string `json:"network"`
						SourceIP      string `json:"sourceIP"`
						DestinationIP string `json:"destinationIP"`
						Host          string `json:"host"`
					} `json:"metadata"`
				}{
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
