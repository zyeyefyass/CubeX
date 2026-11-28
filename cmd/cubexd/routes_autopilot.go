package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/zyeyefyass/CubeX/internal/dataplane"
	"github.com/zyeyefyass/CubeX/internal/dnsradar"
	"github.com/zyeyefyass/CubeX/internal/perf"
	"github.com/zyeyefyass/CubeX/internal/probe"
)

// registerV98Routes registers CubeX v9.8.0 AutoPilot endpoints
func (s *Server) registerV98Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/autopilot/status", s.handleAutoPilotStatus)
	mux.HandleFunc("/api/v1/autopilot/failopen/toggle", s.handleFailOpenToggle)
	mux.HandleFunc("/api/v1/autopilot/circuit-breaker/reset", s.handleCircuitBreakerReset)
	mux.HandleFunc("/api/v1/autopilot/dns/resolve", s.handleServeStaleDNSResolve)
}

func (s *Server) handleAutoPilotStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	buf := perf.GetBuffer()
	defer perf.PutBuffer(buf)

	var quarantined map[string]int64
	if s.circuitBreaker != nil {
		quarantined = s.circuitBreaker.GetQuarantinedNodes()
	}

	var failOpenRep dataplane.FailOpenStatusReport
	if s.failOpenCtrl != nil {
		failOpenRep = s.failOpenCtrl.GetStatusReport()
	}

	var scavengeCount uint64
	var lastScavenge int64
	var softLimitMiB int64 = 24
	if s.memGuard != nil {
		scavengeCount, lastScavenge = s.memGuard.GetScavengeStats()
		softLimitMiB = s.memGuard.LimitMiB()
	}

	status := map[string]any{
		"autopilot_enabled": true,
		"circuit_breaker": map[string]any{
			"quarantined_nodes": quarantined,
			"quarantined_count": len(quarantined),
		},
		"fail_open": failOpenRep,
		"memory_guard": map[string]any{
			"soft_limit_mib":      softLimitMiB,
			"idle_scavenge_count": scavengeCount,
			"last_scavenge_milli": lastScavenge,
		},
		"timestamp": time.Now().UnixMilli(),
	}

	_ = json.NewEncoder(w).Encode(status)
}

func (s *Server) handleFailOpenToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.failOpenCtrl == nil {
		http.Error(w, "FailOpen controller not initialized", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Enable bool   `json:"enable"`
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Reason == "" {
		req.Reason = "控制台手动操作"
	}

	if req.Enable {
		_ = s.failOpenCtrl.TriggerFailOpen(r.Context(), req.Reason)
	} else {
		_ = s.failOpenCtrl.RestoreNormal(r.Context(), req.Reason)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.failOpenCtrl.GetStatusReport())
}

func (s *Server) handleCircuitBreakerReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.circuitBreaker == nil {
		http.Error(w, "Circuit breaker not initialized", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		NodeName string `json:"node_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NodeName == "" {
		http.Error(w, "Invalid node_name", http.StatusBadRequest)
		return
	}

	s.circuitBreaker.ResetNode(req.NodeName)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Node circuit breaker reset successfully",
		"node":    req.NodeName,
	})
}

func (s *Server) handleServeStaleDNSResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	domain := r.URL.Query().Get("domain")
	if domain == "" {
		http.Error(w, "domain parameter required", http.StatusBadRequest)
		return
	}

	if s.staleDNSResolver == nil {
		s.staleDNSResolver = dnsradar.NewServeStaleResolver("223.5.5.5:53", "119.29.29.29:53", 4096)
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	ips, isStale, err := s.staleDNSResolver.Resolve(ctx, domain)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"domain":     domain,
		"ips":        ips,
		"is_stale":   isStale,
		"latency_ms": elapsed,
	})
}

// 确保引用 probe，防止编译警告
var _ = probe.StateClosed
