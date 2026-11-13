package dataplane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWANHealer_FlushStaleConnections(t *testing.T) {
	deleteCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/connections" {
			deleteCalled = true
			if r.Header.Get("Authorization") != "Bearer test-secret" {
				t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	healedReason := ""
	healer := NewWANHealer(server.URL, "test-secret", func(reason string) {
		healedReason = reason
	})

	err := healer.FlushStaleConnections(context.Background(), "PPPoE Redial")
	if err != nil {
		t.Fatalf("FlushStaleConnections failed: %v", err)
	}
	if !deleteCalled {
		t.Error("expected DELETE /connections to be called")
	}
	if healedReason != "PPPoE Redial" {
		t.Errorf("expected reason 'PPPoE Redial', got %s", healedReason)
	}
}

func TestWANHealer_WatchGatewayTransition(t *testing.T) {
	flushCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/connections" {
			flushCount++
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	healer := NewWANHealer(server.URL, "", nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reachabilitySequence := []bool{true, false, true}
	callIdx := 0

	probeFn := func() bool {
		if callIdx < len(reachabilitySequence) {
			res := reachabilitySequence[callIdx]
			callIdx++
			return res
		}
		return true
	}

	// 启动并模拟快速调用
	go healer.WatchGatewayTransition(ctx, 10*time.Millisecond, probeFn)

	time.Sleep(50 * time.Millisecond)
	cancel()

	if flushCount == 0 {
		t.Error("expected FlushStaleConnections to be triggered on offline->online transition")
	}
}
