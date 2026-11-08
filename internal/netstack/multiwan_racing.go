package netstack

import (
	"sync"
	"time"
)

// WANStatus represents the operational status of a physical/virtual WAN interface.
type WANStatus string

const (
	WANStatusOptimal   WANStatus = "optimal"   // Latency < 30ms, Loss = 0%
	WANStatusNormal    WANStatus = "normal"    // Latency < 60ms, Loss < 2%
	WANStatusDegraded  WANStatus = "degraded"  // High jitter / loss > 3%
	WANStatusOffline   WANStatus = "offline"   // Link down or 100% loss
)

// WANInterface represents a single monitored WAN link.
type WANInterface struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Gateway    string    `json:"gateway"`
	Weight     int       `json:"weight"`       // Weight percentage (e.g. 50, 30, 20)
	RTTMs      float64   `json:"rtt_ms"`       // Real-time RTT in milliseconds
	JitterMs   float64   `json:"jitter_ms"`    // Latency variation
	LossRate   float64   `json:"loss_rate"`    // 0.0 - 1.0
	Status     WANStatus `json:"status"`
	IsActive   bool      `json:"is_active"`    // Whether current primary for microflows
	BytesSent  uint64    `json:"bytes_sent"`
	BytesRecv  uint64    `json:"bytes_recv"`
	LastProbe  time.Time `json:"last_probe"`
}

// MultiWANRacingConfig defines policies for dynamic WAN selection.
type MultiWANRacingConfig struct {
	Enabled            bool    `json:"enabled"`
	Policy             string  `json:"policy"`               // "lowest_latency", "loss_avoidance", "weighted"
	LossThreshold      float64 `json:"loss_threshold"`       // e.g. 0.03 (3%)
	JitterThresholdMs  float64 `json:"jitter_threshold_ms"`  // e.g. 15.0 ms
	AutoFailover       bool    `json:"auto_failover"`
}

// MultiWANRacingMetrics is telemetry returned to the API and Twilight Crystal cards.
type MultiWANRacingMetrics struct {
	Enabled       bool           `json:"enabled"`
	Policy        string         `json:"policy"`
	PrimaryWAN    string         `json:"primary_wan"`
	TotalSwitches uint64         `json:"total_switches"`
	Interfaces    []WANInterface `json:"interfaces"`
}

// MultiWANRacingManager orchestrates millisecond-level dynamic WAN path selection.
type MultiWANRacingManager struct {
	mu            sync.RWMutex
	cfg           MultiWANRacingConfig
	interfaces    map[string]*WANInterface
	primaryWAN    string
	totalSwitches uint64
}

// NewMultiWANRacingManager instantiates the racing manager with resilient defaults.
func NewMultiWANRacingManager(cfg *MultiWANRacingConfig) *MultiWANRacingManager {
	defaultCfg := MultiWANRacingConfig{
		Enabled:           true,
		Policy:            "loss_avoidance",
		LossThreshold:     0.03, // 3% packet loss triggers failover
		JitterThresholdMs: 15.0, // 15ms jitter triggers failover
		AutoFailover:      true,
	}
	if cfg != nil {
		if cfg.Policy != "" {
			defaultCfg.Policy = cfg.Policy
		}
		if cfg.LossThreshold > 0 {
			defaultCfg.LossThreshold = cfg.LossThreshold
		}
		if cfg.JitterThresholdMs > 0 {
			defaultCfg.JitterThresholdMs = cfg.JitterThresholdMs
		}
		defaultCfg.Enabled = cfg.Enabled
		defaultCfg.AutoFailover = cfg.AutoFailover
	}

	mgr := &MultiWANRacingManager{
		cfg:        defaultCfg,
		interfaces: make(map[string]*WANInterface),
	}

	// Preset standard carrier WAN uplinks
	mgr.RegisterWAN("wan1", "主力宽带 (电信千兆)", "192.168.1.1", 60)
	mgr.RegisterWAN("wan2", "备用宽带 (联通/移动)", "192.168.2.1", 40)
	return mgr
}

// RegisterWAN adds or updates a monitored WAN interface.
func (m *MultiWANRacingManager) RegisterWAN(id, name, gateway string, weight int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.interfaces[id]; !exists {
		m.interfaces[id] = &WANInterface{
			ID:        id,
			Name:      name,
			Gateway:   gateway,
			Weight:    weight,
			Status:    WANStatusOptimal,
			LastProbe: time.Now(),
		}
		if m.primaryWAN == "" {
			m.primaryWAN = id
			m.interfaces[id].IsActive = true
		}
	}
}

// UpdateTelemetry updates real-time probe metrics for a WAN and arbitrates optimal path.
func (m *MultiWANRacingManager) UpdateTelemetry(id string, rtt, jitter, loss float64) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	wan, exists := m.interfaces[id]
	if !exists {
		return m.primaryWAN
	}

	wan.RTTMs = rtt
	wan.JitterMs = jitter
	wan.LossRate = loss
	wan.LastProbe = time.Now()

	// Assess status
	if loss >= 0.8 {
		wan.Status = WANStatusOffline
	} else if loss > m.cfg.LossThreshold || jitter > m.cfg.JitterThresholdMs {
		wan.Status = WANStatusDegraded
	} else if rtt < 35.0 && loss == 0 {
		wan.Status = WANStatusOptimal
	} else {
		wan.Status = WANStatusNormal
	}

	// If current primary WAN is degraded or offline, arbitrate to best candidate
	if m.cfg.AutoFailover && id == m.primaryWAN && (wan.Status == WANStatusDegraded || wan.Status == WANStatusOffline) {
		var bestCandidate string
		var bestScore float64 = 999999.0

		for cid, candidate := range m.interfaces {
			if cid == m.primaryWAN || candidate.Status == WANStatusOffline || candidate.Status == WANStatusDegraded {
				continue
			}
			score := candidate.RTTMs + (candidate.LossRate * 1000.0) + candidate.JitterMs
			if score < bestScore {
				bestScore = score
				bestCandidate = cid
			}
		}

		if bestCandidate != "" {
			wan.IsActive = false
			m.interfaces[bestCandidate].IsActive = true
			m.primaryWAN = bestCandidate
			m.totalSwitches++
		}
	}

	return m.primaryWAN
}

// SelectBestWAN returns the best active interface ID for an outbound microflow.
func (m *MultiWANRacingManager) SelectBestWAN() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.primaryWAN != "" {
		return m.primaryWAN
	}
	for id := range m.interfaces {
		return id
	}
	return "wan1"
}

// SetPolicy updates arbitration policy.
func (m *MultiWANRacingManager) SetPolicy(policy string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg.Policy = policy
}

// GetMetrics returns snapshot state for UI and API reporting.
func (m *MultiWANRacingManager) GetMetrics() MultiWANRacingMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []WANInterface
	for _, w := range m.interfaces {
		list = append(list, *w)
	}

	return MultiWANRacingMetrics{
		Enabled:       m.cfg.Enabled,
		Policy:        m.cfg.Policy,
		PrimaryWAN:    m.primaryWAN,
		TotalSwitches: m.totalSwitches,
		Interfaces:    list,
	}
}
