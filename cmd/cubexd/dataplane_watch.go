package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// DataplaneHealthSnapshot holds the atomic cached health status of the proxy data plane.
type DataplaneHealthSnapshot struct {
	Status          string `json:"status"` // "ok" or "degraded"
	State           string `json:"state"`  // "healthy", "switching", "degraded", "draining"
	Healthy         bool   `json:"healthy"`
	ActiveEngine    string `json:"active_engine"`
	ClashAPI        bool   `json:"clash_api"`
	DNSService      bool   `json:"dns_service"`
	CheckedAt       string `json:"checked_at"`
	FailStreak      int    `json:"fail_streak"`
	SuccessStreak   int    `json:"success_streak"`
	ProbeDurationMs int64  `json:"probe_duration_ms"`
}

// startDataplaneHealthLoop runs an independent background loop (1s interval) with hysteresis filtering
// to maintain an ultra-fast in-memory atomic health cache for /healthz/dataplane.
func (s *Server) startDataplaneHealthLoop(ctx context.Context) {
	// Initialize with a healthy optimistic snapshot on boot
	initSnap := &DataplaneHealthSnapshot{
		Status:        "ok",
		State:         "healthy",
		Healthy:       true,
		ActiveEngine:  string(s.engMgr.GetActiveEngine()),
		ClashAPI:      true,
		DNSService:    true,
		CheckedAt:     time.Now().Format(time.RFC3339),
		SuccessStreak: 1,
	}
	s.dataplaneSnapshot.Store(initSnap)

	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()

	httpClient := &http.Client{Timeout: 400 * time.Millisecond}

	failStreak := 0
	successStreak := 1
	isHealthy := true

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// If server is draining, mark degraded immediately
			if s.draining.Load() {
				s.dataplaneSnapshot.Store(&DataplaneHealthSnapshot{
					Status:       "degraded",
					State:        "draining",
					Healthy:      false,
					ActiveEngine: string(s.engMgr.GetActiveEngine()),
					CheckedAt:    time.Now().Format(time.RFC3339),
				})
				continue
			}

			// In intentional switching grace period, maintain healthy status and skip physical probe
			if s.engMgr.IsSwitchingInGracePeriod() {
				continue
			}

			tStart := time.Now()
			activeEngine := s.engMgr.GetActiveEngine()
			clashPort := 9090
			dnsPort := 1053
			if ir := s.engMgr.GetIR(); ir != nil {
				if ir.Network.ClashAPIPort > 0 {
					clashPort = ir.Network.ClashAPIPort
				}
				if ir.Network.DNSPort > 0 {
					dnsPort = ir.Network.DNSPort
				}
			}

			// 1. Probe Clash API
			clashOK := false
			if resp, err := httpClient.Get(fmt.Sprintf("http://127.0.0.1:%d/version", clashPort)); err == nil {
				if resp.StatusCode == http.StatusOK {
					clashOK = true
				}
				_ = resp.Body.Close()
			}

			// 2. Probe DNS UDP Port
			dnsOK := false
			if conn, err := net.DialTimeout("udp", fmt.Sprintf("127.0.0.1:%d", dnsPort), 300*time.Millisecond); err == nil {
				_ = conn.Close()
				dnsOK = true
			}

			probePass := clashOK && dnsOK

			// Hysteresis State Machine:
			// - Transition to Unhealthy requires 2 consecutive probe failures (avoids DNS micro-jitter false alarms)
			// - Transition to Healthy requires 3 consecutive probe successes (ensures rock-solid recovery before failback)
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

			stateStr := "healthy"
			statusStr := "ok"
			if !isHealthy {
				stateStr = "degraded"
				statusStr = "degraded"
			}

			snap := &DataplaneHealthSnapshot{
				Status:          statusStr,
				State:           stateStr,
				Healthy:         isHealthy,
				ActiveEngine:    string(activeEngine),
				ClashAPI:        clashOK,
				DNSService:      dnsOK,
				CheckedAt:       time.Now().Format(time.RFC3339),
				FailStreak:      failStreak,
				SuccessStreak:   successStreak,
				ProbeDurationMs: time.Since(tStart).Milliseconds(),
			}
			s.dataplaneSnapshot.Store(snap)
		}
	}
}

// healthzDataplaneHandler returns an HTTP handler for /healthz/dataplane.
// Microsecond in-memory atomic read with intentional switching grace period protection.
func (s *Server) healthzDataplaneHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if s.draining.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":    "draining",
				"state":     "draining",
				"healthy":   false,
				"dataplane": "unhealthy",
				"reason":    "server is draining",
			})
			return
		}

		// 1. 计划内切核宽限态 (Grace Period): 在主动热切换 1500ms 窗口内返回 200 OK + switching，杜绝 Netwatch 误熔断
		if s.engMgr != nil && s.engMgr.IsSwitchingInGracePeriod() {
			w.WriteHeader(http.StatusOK)
			activeEngine := ""
			if s.engMgr != nil {
				activeEngine = string(s.engMgr.GetActiveEngine())
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":        "ok",
				"state":         "switching",
				"healthy":       true,
				"active_engine": activeEngine,
				"notice":        "planned core hot-swap in progress (grace period protected)",
				"checked_at":    time.Now().Format(time.RFC3339),
			})
			return
		}

		// 2. 微秒级非阻塞读取后台独立协程维护的原子快照 (杜绝高频轮询打爆本地 DNS 与 goroutine 堆积)
		snap := s.dataplaneSnapshot.Load()
		if snap == nil {
			activeEngine := ""
			if s.engMgr != nil {
				activeEngine = string(s.engMgr.GetActiveEngine())
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":        "ok",
				"state":         "starting",
				"healthy":       true,
				"active_engine": activeEngine,
				"checked_at":    time.Now().Format(time.RFC3339),
			})
			return
		}

		if snap.Healthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(snap)
	}
}
