package probe

import (
	"sort"
	"sync"
	"time"
)

// NodeTriangleMetrics tracks Latency Median, Jitter, and Passive Throughput.
// All sliding window samples use fixed-size arrays [10]uint16 with zero heap allocation.
type NodeTriangleMetrics struct {
	// Latency ring buffer (10 samples, in milliseconds, max 65535ms)
	Delays     [10]uint16 `json:"delays"`
	DelayIdx   uint8      `json:"delay_idx"`
	DelayCount uint8      `json:"delay_count"`
	lastDelay  uint16     // for jitter calculation

	// Jitter ring buffer (10 samples of |Delay_i - Delay_{i-1}|)
	Jitters     [10]uint16 `json:"jitters"`
	JitterIdx   uint8      `json:"jitter_idx"`
	JitterCount uint8      `json:"jitter_count"`

	// Passive throughput high-water mark (Kbps) with peak-hold & decay
	PeakBandwidthKbps uint32 `json:"peak_bandwidth_kbps"`
	LastPeakUpdate    int64  `json:"last_peak_update"` // unix timestamp in seconds
	SampledBytes      uint64 `json:"sampled_bytes"`    // total bytes observed passively

	// Evaluated scores (cached to avoid redundant computation)
	MedianDelayMs  uint16  `json:"median_delay_ms"`
	MeanJitterMs   uint16  `json:"mean_jitter_ms"`
	LatencyScore   float64 `json:"latency_score"`
	JitterScore    float64 `json:"jitter_score"`
	BandwidthScore float64 `json:"bandwidth_score"`
	TotalScore     float64 `json:"total_score"` // 0.0 ~ 100.0
	Grade          string  `json:"grade"`       // "S", "A+", "A", "B", "C"
	LastEvalAt     int64   `json:"last_eval_at"`
}

// TriangleEngine manages stability triangle scores across all proxy nodes.
type TriangleEngine struct {
	mu              sync.RWMutex
	nodes           map[string]*NodeTriangleMetrics
	latencyWeight   float64 // default 0.40
	jitterWeight    float64 // default 0.30
	bandwidthWeight float64 // default 0.30
}

var (
	globalTriangleOnce sync.Once
	globalTriangle     *TriangleEngine
)

// NewTriangleEngine returns a new TriangleEngine instance with specified weights.
func NewTriangleEngine(wLat, wJit, wBw float64) *TriangleEngine {
	sum := wLat + wJit + wBw
	if sum <= 0 {
		wLat, wJit, wBw = 0.40, 0.30, 0.30
		sum = 1.0
	}
	return &TriangleEngine{
		nodes:           make(map[string]*NodeTriangleMetrics, 256),
		latencyWeight:   wLat / sum,
		jitterWeight:    wJit / sum,
		bandwidthWeight: wBw / sum,
	}
}

// GetWeights returns current weights (latency, jitter, bandwidth).
func (te *TriangleEngine) GetWeights() (float64, float64, float64) {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.latencyWeight, te.jitterWeight, te.bandwidthWeight
}

// GetTriangleEngine returns the singleton instance of TriangleEngine.
func GetTriangleEngine() *TriangleEngine {
	globalTriangleOnce.Do(func() {
		globalTriangle = &TriangleEngine{
			nodes:           make(map[string]*NodeTriangleMetrics, 256),
			latencyWeight:   0.40,
			jitterWeight:    0.30,
			bandwidthWeight: 0.30,
		}
	})
	return globalTriangle
}

// RecordLatency logs a new latency probe measurement using a fixed [10]uint16 ring buffer.
func (te *TriangleEngine) RecordLatency(nodeName string, delayMs int) {
	if nodeName == "" {
		return
	}
	te.mu.Lock()
	defer te.mu.Unlock()

	m, ok := te.nodes[nodeName]
	if !ok {
		m = &NodeTriangleMetrics{}
		te.nodes[nodeName] = m
	}

	safeDelay := uint16(delayMs)
	if delayMs > 65535 {
		safeDelay = 65535
	} else if delayMs < 0 {
		safeDelay = 0
	}

	// 1. Calculate jitter if there was a previous valid delay
	if m.lastDelay > 0 && safeDelay > 0 {
		var diff uint16
		if safeDelay > m.lastDelay {
			diff = safeDelay - m.lastDelay
		} else {
			diff = m.lastDelay - safeDelay
		}
		m.Jitters[m.JitterIdx] = diff
		m.JitterIdx = (m.JitterIdx + 1) % 10
		if m.JitterCount < 10 {
			m.JitterCount++
		}
	}
	if safeDelay > 0 {
		m.lastDelay = safeDelay
	}

	// 2. Insert into Delays ring buffer
	m.Delays[m.DelayIdx] = safeDelay
	m.DelayIdx = (m.DelayIdx + 1) % 10
	if m.DelayCount < 10 {
		m.DelayCount++
	}

	// 3. Re-evaluate metrics immediately
	te.evaluateNodeLocked(m)
}

// RecordPassiveTraffic updates the node throughput using a Peak-Hold with Decay algorithm.
// Only bursts >= 256KB trigger peak bandwidth updates.
func (te *TriangleEngine) RecordPassiveTraffic(nodeName string, transferredBytes uint64, durationMs int64) {
	if nodeName == "" || transferredBytes == 0 {
		return
	}
	te.mu.Lock()
	defer te.mu.Unlock()

	m, ok := te.nodes[nodeName]
	if !ok {
		m = &NodeTriangleMetrics{}
		te.nodes[nodeName] = m
	}

	m.SampledBytes += transferredBytes

	// Only sample peak throughput if burst is >= 256KB and duration >= 100ms
	if transferredBytes >= 256*1024 && durationMs >= 100 {
		// rate in Kbps = (bytes * 8) / (durationMs)
		rateKbps := uint32((transferredBytes * 8) / uint64(durationMs))
		nowSec := time.Now().Unix()

		// Peak-Hold: update if higher
		if rateKbps > m.PeakBandwidthKbps {
			m.PeakBandwidthKbps = rateKbps
			m.LastPeakUpdate = nowSec
		} else if m.LastPeakUpdate > 0 && nowSec-m.LastPeakUpdate > 1800 {
			// Decay by 5% every 30 minutes of inactivity
			decayIntervals := (nowSec - m.LastPeakUpdate) / 1800
			decayFactor := 1.0 - (0.05 * float64(decayIntervals))
			if decayFactor < 0.20 {
				decayFactor = 0.20
			}
			m.PeakBandwidthKbps = uint32(float64(m.PeakBandwidthKbps) * decayFactor)
			m.LastPeakUpdate = nowSec
		}
	}

	te.evaluateNodeLocked(m)
}

// SetBurstSpeed explicitly sets the peak bandwidth from a manual micro burst test (e.g. 2MB burst).
func (te *TriangleEngine) SetBurstSpeed(nodeName string, speedMbps float64) {
	if nodeName == "" {
		return
	}
	te.mu.Lock()
	defer te.mu.Unlock()

	m, ok := te.nodes[nodeName]
	if !ok {
		m = &NodeTriangleMetrics{}
		te.nodes[nodeName] = m
	}

	kbps := uint32(speedMbps * 1024)
	if kbps > m.PeakBandwidthKbps {
		m.PeakBandwidthKbps = kbps
		m.LastPeakUpdate = time.Now().Unix()
	}
	m.SampledBytes += 2 * 1024 * 1024
	te.evaluateNodeLocked(m)
}

func (te *TriangleEngine) evaluateNodeLocked(m *NodeTriangleMetrics) {
	// 1. Calculate Median Delay
	if m.DelayCount == 0 {
		m.MedianDelayMs = 0
		m.LatencyScore = 0.0
	} else {
		validDelays := make([]int, 0, m.DelayCount)
		for i := 0; i < int(m.DelayCount); i++ {
			if m.Delays[i] > 0 {
				validDelays = append(validDelays, int(m.Delays[i]))
			}
		}
		if len(validDelays) == 0 {
			m.MedianDelayMs = 0
			m.LatencyScore = 0.0
		} else {
			sort.Ints(validDelays)
			mid := len(validDelays) / 2
			if len(validDelays)%2 == 1 {
				m.MedianDelayMs = uint16(validDelays[mid])
			} else {
				m.MedianDelayMs = uint16((validDelays[mid-1] + validDelays[mid]) / 2)
			}
			m.LatencyScore = scoreLatency(m.MedianDelayMs)
		}
	}

	// 2. Calculate Mean Jitter
	if m.JitterCount == 0 {
		m.MeanJitterMs = 0
		m.JitterScore = 100.0 // no jitter observed initially
	} else {
		var sum uint32 = 0
		for i := 0; i < int(m.JitterCount); i++ {
			sum += uint32(m.Jitters[i])
		}
		m.MeanJitterMs = uint16(sum / uint32(m.JitterCount))
		m.JitterScore = scoreJitter(m.MeanJitterMs)
	}

	// 3. Calculate Bandwidth Score
	if m.SampledBytes == 0 || m.PeakBandwidthKbps == 0 {
		// Neutral baseline (75.0) when no large traffic passed through
		m.BandwidthScore = 75.0
	} else {
		m.BandwidthScore = scoreBandwidth(m.PeakBandwidthKbps)
	}

	// 4. Calculate Weighted Total Score
	m.TotalScore = (m.LatencyScore * te.latencyWeight) +
		(m.JitterScore * te.jitterWeight) +
		(m.BandwidthScore * te.bandwidthWeight)

	if m.TotalScore > 100.0 {
		m.TotalScore = 100.0
	} else if m.TotalScore < 0.0 {
		m.TotalScore = 0.0
	}

	// 5. Determine Grade
	if m.TotalScore >= 90.0 {
		m.Grade = "S 极致稳定"
	} else if m.TotalScore >= 80.0 {
		m.Grade = "A+ 极高品质"
	} else if m.TotalScore >= 70.0 {
		m.Grade = "A 优质推荐"
	} else if m.TotalScore >= 60.0 {
		m.Grade = "B 普通可用"
	} else {
		m.Grade = "C 波动劣化 (建议避让)"
	}

	m.LastEvalAt = time.Now().Unix()
}

func scoreLatency(ms uint16) float64 {
	if ms == 0 || ms >= 9999 {
		return 0.0
	}
	if ms <= 30 {
		return 100.0
	}
	if ms <= 60 {
		return 95.0 - float64(ms-30)*(10.0/30.0)
	}
	if ms <= 120 {
		return 85.0 - float64(ms-60)*(15.0/60.0)
	}
	if ms <= 250 {
		return 70.0 - float64(ms-120)*(20.0/130.0)
	}
	if ms <= 500 {
		return 50.0 - float64(ms-250)*(30.0/250.0)
	}
	return 10.0
}

func scoreJitter(ms uint16) float64 {
	if ms <= 3 {
		return 100.0
	}
	if ms <= 10 {
		return 95.0 - float64(ms-3)*(10.0/7.0)
	}
	if ms <= 25 {
		return 85.0 - float64(ms-10)*(15.0/15.0)
	}
	if ms <= 60 {
		return 70.0 - float64(ms-25)*(25.0/35.0)
	}
	if ms <= 150 {
		return 45.0 - float64(ms-60)*(30.0/90.0)
	}
	return 10.0
}

func scoreBandwidth(kbps uint32) float64 {
	if kbps >= 500000 { // >= 500 Mbps
		return 100.0
	}
	if kbps >= 100000 { // >= 100 Mbps
		return 90.0 + float64(kbps-100000)*(10.0/400000.0)
	}
	if kbps >= 30000 { // >= 30 Mbps
		return 80.0 + float64(kbps-30000)*(10.0/70000.0)
	}
	if kbps >= 10000 { // >= 10 Mbps
		return 70.0 + float64(kbps-10000)*(10.0/20000.0)
	}
	if kbps >= 2048 { // >= 2 Mbps
		return 40.0 + float64(kbps-2048)*(30.0/8192.0)
	}
	return 20.0
}

// GetNodeScore returns a snapshot of the node triangle metrics.
func (te *TriangleEngine) GetNodeScore(nodeName string) NodeTriangleMetrics {
	te.mu.RLock()
	defer te.mu.RUnlock()

	m, ok := te.nodes[nodeName]
	if !ok || m == nil {
		return NodeTriangleMetrics{
			TotalScore:     75.0,
			BandwidthScore: 75.0,
			Grade:          "A 优质推荐",
		}
	}

	return *m
}

// GetAllScores returns snapshots of all evaluated nodes.
func (te *TriangleEngine) GetAllScores() map[string]NodeTriangleMetrics {
	te.mu.RLock()
	defer te.mu.RUnlock()

	out := make(map[string]NodeTriangleMetrics, len(te.nodes))
	for k, m := range te.nodes {
		out[k] = *m
	}
	return out
}

// UpdateWeights dynamically configures the weights of Latency, Jitter, and Bandwidth.
func (te *TriangleEngine) UpdateWeights(wLat, wJit, wBw float64) {
	sum := wLat + wJit + wBw
	if sum <= 0 {
		return
	}
	te.mu.Lock()
	defer te.mu.Unlock()

	te.latencyWeight = wLat / sum
	te.jitterWeight = wJit / sum
	te.bandwidthWeight = wBw / sum

	// Re-evaluate all nodes with new weights
	for _, m := range te.nodes {
		te.evaluateNodeLocked(m)
	}
}
