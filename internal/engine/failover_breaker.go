package engine

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

const (
	defaultFailoverCircuitWindow      = 60 * time.Second
	defaultFailoverCircuitMaxFailures = 4
	maxCoreDiagnosticLines            = 12
)

// EngineFaultStatus captures observable engine failure and circuit-breaker telemetry.
type EngineFaultStatus struct {
	CircuitBroken       bool   `json:"circuit_broken"`
	CircuitBrokenReason string `json:"circuit_broken_reason,omitempty"`
	CircuitBrokenAt     string `json:"circuit_broken_at,omitempty"`
	LastCoreError       string `json:"last_core_error,omitempty"`
	TunReady            bool   `json:"tun_ready"`
	TunDetail           string `json:"tun_detail,omitempty"`
	FailureCount        int    `json:"failure_count"`
}

// GetEngineFaultStatus returns a thread-safe snapshot of engine circuit-breaker, recent stderr, and TUN preflight status.
func (m *Manager) GetEngineFaultStatus() EngineFaultStatus {
	tunOK, tunMsg := CheckTunDevice()

	m.faultMu.Lock()
	broken := m.circuitBroken
	reason := m.circuitBrokenReason
	brokenAt := m.circuitBrokenAt
	lastErr := m.lastCoreError
	failCnt := len(m.failoverHistory)
	m.faultMu.Unlock()

	st := EngineFaultStatus{
		CircuitBroken:       broken,
		CircuitBrokenReason: reason,
		LastCoreError:       lastErr,
		TunReady:            tunOK,
		TunDetail:           tunMsg,
		FailureCount:        failCnt,
	}
	if !brokenAt.IsZero() {
		st.CircuitBrokenAt = brokenAt.Format(time.RFC3339)
	}
	return st
}

// IsCircuitBroken reports whether automatic dual-engine failover has tripped its circuit breaker.
func (m *Manager) IsCircuitBroken() bool {
	m.faultMu.Lock()
	defer m.faultMu.Unlock()
	return m.circuitBroken
}

// ResetCircuitBreaker clears any tripped dual-engine circuit breaker and failure window.
func (m *Manager) ResetCircuitBreaker() {
	m.faultMu.Lock()
	m.circuitBroken = false
	m.circuitBrokenReason = ""
	m.circuitBrokenAt = time.Time{}
	m.failoverHistory = nil
	m.faultMu.Unlock()
}

func (m *Manager) setLastCoreError(msg string) {
	trimmed := strings.TrimSpace(msg)
	if trimmed == "" {
		return
	}
	m.faultMu.Lock()
	m.lastCoreError = trimmed
	m.faultMu.Unlock()
}

func (m *Manager) getLastCoreError() string {
	m.faultMu.Lock()
	defer m.faultMu.Unlock()
	return m.lastCoreError
}

func (m *Manager) captureCoreOutput(pipeType, cleanLine string) {
	line := strings.TrimSpace(cleanLine)
	if line == "" {
		return
	}
	low := strings.ToLower(line)
	isImportant := pipeType == "stderr" ||
		strings.Contains(low, "fatal") ||
		strings.Contains(low, "panic") ||
		strings.Contains(low, "error") ||
		strings.Contains(low, "microarchitecture") ||
		strings.Contains(low, "permission denied") ||
		strings.Contains(low, "operation not permitted") ||
		strings.Contains(low, "tun")

	m.faultMu.Lock()
	m.recentStderr = append(m.recentStderr, line)
	if len(m.recentStderr) > maxCoreDiagnosticLines {
		m.recentStderr = m.recentStderr[len(m.recentStderr)-maxCoreDiagnosticLines:]
	}
	if isImportant {
		m.lastCoreError = line
	}
	m.faultMu.Unlock()
}

func (m *Manager) clearRecentCoreOutput() {
	m.faultMu.Lock()
	m.recentStderr = nil
	m.faultMu.Unlock()
}

func (m *Manager) summarizeRecentStderr() string {
	m.faultMu.Lock()
	defer m.faultMu.Unlock()
	if len(m.recentStderr) == 0 {
		return m.lastCoreError
	}
	start := len(m.recentStderr) - 3
	if start < 0 {
		start = 0
	}
	return strings.Join(m.recentStderr[start:], " | ")
}

func (m *Manager) tripFailoverCircuit(reason string) {
	now := time.Now()
	m.faultMu.Lock()
	already := m.circuitBroken
	m.circuitBroken = true
	m.circuitBrokenReason = reason
	m.circuitBrokenAt = now
	m.lastCoreError = reason
	m.faultMu.Unlock()

	if !already {
		m.log("ERROR", "SUPERVISOR", fmt.Sprintf("🛑 双核灾备保护已触发熔断，停止自动循环切换！原因: %s", reason))
		log.Printf("🛑 [Supervisor] Dual-engine failover circuit breaker TRIPPED: %s", reason)
	}
}

func (m *Manager) recordFailoverFailureAndMaybeTrip(now time.Time, detail string) bool {
	win := m.failoverWindow
	if win <= 0 {
		win = defaultFailoverCircuitWindow
	}
	maxFail := m.failoverMaxFailures
	if maxFail <= 0 {
		maxFail = defaultFailoverCircuitMaxFailures
	}

	m.faultMu.Lock()
	cutoff := now.Add(-win)
	kept := m.failoverHistory[:0]
	for _, ts := range m.failoverHistory {
		if !ts.Before(cutoff) {
			kept = append(kept, ts)
		}
	}
	kept = append(kept, now)
	m.failoverHistory = kept
	failCount := len(kept)
	m.faultMu.Unlock()

	if failCount >= maxFail {
		reason := fmt.Sprintf("短时 (%s) 连续发生 %d 次内核异常或切换失败，已中止死循环重试。最后报错: %s", win, failCount, detail)
		m.tripFailoverCircuit(reason)
		return true
	}
	return false
}

// runSupervisorFailover executes one supervisor failure-recovery step and trips the circuit breaker
// immediately if both active and standby engines fail or rapid thrashing exceeds threshold.
func (m *Manager) runSupervisorFailover(active, standby schema.EngineType, count int) {
	if m.IsCircuitBroken() {
		return
	}

	switchFn := m.switchHook
	if switchFn == nil {
		switchFn = m.SwitchEngine
	}

	if count == 1 {
		m.log("INFO", "SUPERVISOR", fmt.Sprintf("🛡️ 自愈中枢正在极速重启核心 %s...", active))
		if _, err := switchFn(active); err == nil {
			m.log("INFO", "SUPERVISOR", fmt.Sprintf("✅ 核心 %s 快速复位自愈成功", active))
			log.Printf("✅ [Supervisor] Successfully revived engine %s on quick restart", active)
			m.recordFailoverFailureAndMaybeTrip(time.Now(), fmt.Sprintf("%s 发生瞬时故障并快速复位", active))
			return
		} else {
			m.log("WARN", "SUPERVISOR", fmt.Sprintf("⚠️ 核心 %s 快速复位失败 (%v)，尝试切换至备用核心 %s...", active, err, standby))
			res, sErr := switchFn(standby)
			if sErr == nil {
				m.log("INFO", "SUPERVISOR", fmt.Sprintf("✅ 容灾切换成功: 当前主控平滑降级至 %s (耗时 %d ms)", standby, res.ElapsedMs))
				m.mu.Lock()
				m.crashCount = 0
				m.mu.Unlock()
				m.recordFailoverFailureAndMaybeTrip(time.Now(), fmt.Sprintf("主核 %s 失败 (%v)，已切至 %s", active, err, standby))
				return
			}
			// Both active quick-restart and standby switch failed -> trip circuit immediately!
			combined := fmt.Sprintf("主核 %s 复位失败 (%v)；备用核心 %s 切换失败 (%v)", active, err, standby, sErr)
			m.tripFailoverCircuit(combined)
			return
		}
	}

	m.log("WARN", "SUPERVISOR", fmt.Sprintf("🚨 核心 %s 连续异常，启动双核容灾切换 ➔ %s...", active, standby))
	res, err := switchFn(standby)
	if err == nil {
		m.log("INFO", "SUPERVISOR", fmt.Sprintf("✅ 双核容灾自愈完成: 已平滑切换至 %s (耗时 %d ms, 恢复策略组: %d)", standby, res.ElapsedMs, res.RestoredGroups))
		log.Printf("✅ [Supervisor] Auto-failover to %s succeeded in %d ms", standby, res.ElapsedMs)
		m.mu.Lock()
		m.crashCount = 0
		m.mu.Unlock()
		m.recordFailoverFailureAndMaybeTrip(time.Now(), fmt.Sprintf("主核 %s 连续异常已切换至 %s", active, standby))
		return
	}

	detail := fmt.Sprintf("主核 %s 异常且容灾切换 %s 失败: %v", active, standby, err)
	if !m.recordFailoverFailureAndMaybeTrip(time.Now(), detail) {
		// Even on first failure of standby switch during crash count >= 2, trip immediately because both engines are unhealthy.
		m.tripFailoverCircuit(detail)
	}
}
