package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/zyeyefyass/CubeX/internal/netutil"
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

	httpClient := netutil.NewBoundedHTTPClient(400 * time.Millisecond)

	failStreak := 0
	successStreak := 1
	isHealthy := true
	bootStart := time.Now()
	const bootGrace = 15 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// If server is draining, mark degraded immediately and terminate watcher
			if s.draining.Load() {
				s.dataplaneSnapshot.Store(&DataplaneHealthSnapshot{
					Status:       "degraded",
					State:        "draining",
					Healthy:      false,
					ActiveEngine: string(s.engMgr.GetActiveEngine()),
					CheckedAt:    time.Now().Format(time.RFC3339),
				})
				return
			}

			// During initial cold-boot readiness grace or intentional switching grace, maintain healthy optimistic snapshot
			if s.engMgr != nil && ((!s.engMgr.IsCoreReady() && time.Since(bootStart) < bootGrace) || s.engMgr.IsSwitchingInGracePeriod()) {
				continue
			}

			tStart := time.Now()
			activeEngine := s.engMgr.GetActiveEngine()
			clashPort := 9090
			dnsPort := 53
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
				netutil.DrainAndCloseBody(resp)
			}

			// 2. Probe DNS UDP Port via Active Query (ensures core event loop is truly processing packets)
			lanIP := getLocalLANIP()
			dnsOK := probeDNSLiveness("127.0.0.1", dnsPort, 50*time.Millisecond)
			if !dnsOK && dnsPort != 53 {
				dnsOK = probeDNSLiveness("127.0.0.1", 53, 50*time.Millisecond)
			}
			if !dnsOK && lanIP != "127.0.0.1" {
				dnsOK = probeDNSLiveness(lanIP, dnsPort, 50*time.Millisecond)
				if !dnsOK && dnsPort != 53 {
					dnsOK = probeDNSLiveness(lanIP, 53, 50*time.Millisecond)
				}
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

// probeDNSLiveness actively sends a DNS A query packet to 127.0.0.1:dnsPort and waits for a valid DNS reply.
// We query "probe.test", which is resolved in-memory by Fake-IP/core DNS tables without hitting external network,
// guaranteeing true application-level event-loop liveness with 0ms overhead.
// getLocalLANIP finds the first non-loopback, non-tun IPv4 address of the host.
func getLocalLANIP() string {
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ip4 := ipnet.IP.To4(); ip4 != nil {
						// Skip internal cubex-tun micro-subnet (172.19.x.x)
						if ip4[0] == 172 && ip4[1] == 19 {
							continue
						}
						return ip4.String()
					}
				}
			}
		}
	}
	return "127.0.0.1"
}

// probeDNSLiveness actively sends a DNS A query packet to the target IP:port and waits for a valid DNS reply.
// We query "probe.test", which is resolved in-memory by Fake-IP/core DNS tables without hitting external network,
// guaranteeing true application-level event-loop liveness with 0ms overhead.
func probeDNSLiveness(ip string, dnsPort int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:%d", ip, dnsPort), timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	// Minimal DNS A query for "probe.test" (ID 0x1234, RD=1)
	query := []byte{
		0x12, 0x34, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x05, 'p', 'r', 'o', 'b', 'e', 0x04, 't', 'e', 's', 't', 0x00,
		0x00, 0x01, 0x00, 0x01,
	}
	if _, err := conn.Write(query); err != nil {
		return false
	}
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	return err == nil && n >= 12
}
