package netstack

import (
	"sync"
	"time"
)

// QUICMode represents the operation mode for RFC 9000 QUIC/HTTP3 flows.
type QUICMode string

const (
	QUICModePassthrough QUICMode = "passthrough"   // Zero-copy direct passthrough
	QUICModeAdaptive    QUICMode = "adaptive"      // Latency & loss-aware dynamic arbitration
	QUICModeTCPFallback QUICMode = "tcp_fallback"  // Suppress UDP 443 to force instant RFC 9000 TCP fallback
)

// QUICFlow represents an active QUIC/HTTP3 telemetry flow.
type QUICFlow struct {
	ID             string    `json:"id"`
	ClientIP       string    `json:"client_ip"`
	TargetDomain   string    `json:"target_domain"`
	TargetPort     int       `json:"target_port"`
	PacketsSent    uint64    `json:"packets_sent"`
	PacketsRecv    uint64    `json:"packets_recv"`
	BytesTotal     uint64    `json:"bytes_total"`
	LossRate       float64   `json:"loss_rate"`        // 0.0 - 1.0 (e.g., 0.02 = 2%)
	RTTMs          float64   `json:"rtt_ms"`            // Round-trip time in milliseconds
	JitterMs       float64   `json:"jitter_ms"`         // Latency variation
	ActiveMode     QUICMode  `json:"active_mode"`       // Current applied policy
	FallbackActive bool      `json:"fallback_active"`   // Whether TCP fallback was triggered
	LastActive     time.Time `json:"last_active"`
}

// QUICPassthroughConfig defines settings for QUIC/HTTP3 traffic handling.
type QUICPassthroughConfig struct {
	Enabled            bool     `json:"enabled"`
	Mode               QUICMode `json:"mode"`
	MaxLossThreshold   float64  `json:"max_loss_threshold"`   // e.g. 0.05 (5% loss triggers fallback)
	MaxJitterThreshold float64  `json:"max_jitter_threshold"` // e.g. 20.0 ms
	AutoFallback       bool     `json:"auto_fallback"`
	Preserve0RTT       bool     `json:"preserve_0rtt"`
}

// QUICPassthroughMetrics represents snapshot telemetry for the UI.
type QUICPassthroughMetrics struct {
	Enabled             bool                  `json:"enabled"`
	Mode                QUICMode              `json:"mode"`
	ActiveFlowsCount    int                   `json:"active_flows_count"`
	TotalPassthroughMB  float64               `json:"total_passthrough_mb"`
	TotalFallbackCount  uint64                `json:"total_fallback_count"`
	AverageLossRate     float64               `json:"average_loss_rate"`
	AverageRTTMs        float64               `json:"average_rtt_ms"`
	CarrierThrottleRate float64               `json:"carrier_throttle_rate"` // Estimated carrier UDP throttling %
	Flows               []QUICFlow            `json:"flows"`
}

// QUICPassthroughManager orchestrates RFC 9000 QUIC/HTTP3 optimization.
type QUICPassthroughManager struct {
	mu                 sync.RWMutex
	cfg                QUICPassthroughConfig
	flows              map[string]*QUICFlow
	totalPassthroughBytes uint64
	totalFallbackCount uint64
}

// NewQUICPassthroughManager instantiates a fresh manager with production defaults.
func NewQUICPassthroughManager(cfg *QUICPassthroughConfig) *QUICPassthroughManager {
	defaultCfg := QUICPassthroughConfig{
		Enabled:            true,
		Mode:               QUICModeAdaptive,
		MaxLossThreshold:   0.045, // 4.5% loss triggers fallback
		MaxJitterThreshold: 18.0,  // 18ms jitter triggers fallback
		AutoFallback:       true,
		Preserve0RTT:       true,
	}
	if cfg != nil {
		if cfg.Mode != "" {
			defaultCfg.Mode = cfg.Mode
		}
		if cfg.MaxLossThreshold > 0 {
			defaultCfg.MaxLossThreshold = cfg.MaxLossThreshold
		}
		if cfg.MaxJitterThreshold > 0 {
			defaultCfg.MaxJitterThreshold = cfg.MaxJitterThreshold
		}
		defaultCfg.Enabled = cfg.Enabled
		defaultCfg.AutoFallback = cfg.AutoFallback
		defaultCfg.Preserve0RTT = cfg.Preserve0RTT
	}
	return &QUICPassthroughManager{
		cfg:   defaultCfg,
		flows: make(map[string]*QUICFlow),
	}
}

// RecordPacket updates telemetry for an ongoing QUIC flow and decides whether to fallback.
func (m *QUICPassthroughManager) RecordPacket(id, clientIP, domain string, port int, bytes uint64, loss float64, rtt float64, jitter float64) QUICMode {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.cfg.Enabled {
		return QUICModePassthrough
	}

	flow, exists := m.flows[id]
	if !exists {
		flow = &QUICFlow{
			ID:           id,
			ClientIP:     clientIP,
			TargetDomain: domain,
			TargetPort:   port,
			ActiveMode:   m.cfg.Mode,
			LastActive:   time.Now(),
		}
		m.flows[id] = flow
	}

	flow.PacketsSent++
	flow.PacketsRecv++
	flow.BytesTotal += bytes
	flow.LossRate = loss
	flow.RTTMs = rtt
	flow.JitterMs = jitter
	flow.LastActive = time.Now()

	// Evaluate policy
	decision := m.cfg.Mode
	if m.cfg.Mode == QUICModeAdaptive && m.cfg.AutoFallback {
		if loss > m.cfg.MaxLossThreshold || jitter > m.cfg.MaxJitterThreshold {
			decision = QUICModeTCPFallback
			if !flow.FallbackActive {
				flow.FallbackActive = true
				m.totalFallbackCount++
			}
		} else {
			decision = QUICModePassthrough
			flow.FallbackActive = false
			m.totalPassthroughBytes += bytes
		}
	} else if m.cfg.Mode == QUICModePassthrough {
		m.totalPassthroughBytes += bytes
	} else if m.cfg.Mode == QUICModeTCPFallback {
		flow.FallbackActive = true
		m.totalFallbackCount++
	}

	flow.ActiveMode = decision
	return decision
}

// SetMode dynamically changes the active operation mode.
func (m *QUICPassthroughManager) SetMode(mode QUICMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg.Mode = mode
}

// SetConfig updates full configuration atomically.
func (m *QUICPassthroughManager) SetConfig(cfg QUICPassthroughConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg = cfg
}

// GetMetrics returns snapshot telemetry for status APIs and UI rendering.
func (m *QUICPassthroughManager) GetMetrics() QUICPassthroughMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var flowList []QUICFlow
	var totalLoss, totalRTT float64
	var throttleSamples int

	for _, f := range m.flows {
		flowList = append(flowList, *f)
		totalLoss += f.LossRate
		totalRTT += f.RTTMs
		if f.LossRate > 0.03 || f.JitterMs > 15.0 {
			throttleSamples++
		}
	}

	avgLoss := 0.0
	avgRTT := 0.0
	throttleRate := 0.0
	if len(m.flows) > 0 {
		avgLoss = totalLoss / float64(len(m.flows))
		avgRTT = totalRTT / float64(len(m.flows))
		throttleRate = (float64(throttleSamples) / float64(len(m.flows))) * 100.0
	}

	return QUICPassthroughMetrics{
		Enabled:             m.cfg.Enabled,
		Mode:                m.cfg.Mode,
		ActiveFlowsCount:    len(m.flows),
		TotalPassthroughMB:  float64(m.totalPassthroughBytes) / (1024 * 1024),
		TotalFallbackCount:  m.totalFallbackCount,
		AverageLossRate:     avgLoss,
		AverageRTTMs:        avgRTT,
		CarrierThrottleRate: throttleRate,
		Flows:               flowList,
	}
}
