package probe

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// NodeEWMA tracks the smoothed moving average, jitter variance, and stability rating for a proxy node.
type NodeEWMA struct {
	NodeName         string    `json:"node_name"`
	Samples          []int     `json:"samples"`           // Up to 10 recent delay samples
	SRTT             float64   `json:"srtt"`              // Smoothed RTT (EWMA) in ms
	RTTVAR           float64   `json:"rttvar"`            // RTT Variance (Jitter) in ms
	EffectiveDelay   int       `json:"effective_delay"`   // SRTT + 1.5*RTTVAR + LossPenalty (anti-flapping effective metric)
	ConsecutiveFails int       `json:"consecutive_fails"` // Current continuous timeout/failure count
	TotalSamples     int       `json:"total_samples"`     // Total lifetime probes
	LossRate         float64   `json:"loss_rate"`         // 0.0 ~ 1.0 packet loss/timeout ratio
	StabilityScore   float64   `json:"stability_score"`   // 0.0 ~ 100.0 composite stability grade
	Grade            string    `json:"grade"`             // "S 极品稳态", "A+ 优质平稳", "A 常规可用", "B 波动较大", "C 离线/丢包"
	LastSampleAt     time.Time `json:"last_sample_at"`
	LastSwitchAt     time.Time `json:"last_switch_at"`    // Timestamp of last selection change for this node
}

// EWMAManager provides thread-safe sliding window tracking, jitter suppression, and anti-flap failover arbitration.
type EWMAManager struct {
	mu            sync.RWMutex
	nodes         map[string]*NodeEWMA
	alpha         float64       // EWMA smoothing factor for RTT (default 0.25)
	beta          float64       // EWMA smoothing factor for Jitter (default 0.25)
	maxWindowSize int           // Maximum recent samples to retain (default 10)
	defaultMargin int           // Minimum latency improvement in ms required to switch (default 25ms)
	defaultRatio  float64       // Minimum percentage improvement required to switch (default 0.15 / 15%)
	cooldown      time.Duration // Minimum duration between switches for the same node/group (default 20s)
}

// NewEWMAManager constructs an initialized EWMAManager with production anti-flap defaults.
func NewEWMAManager() *EWMAManager {
	return &EWMAManager{
		nodes:         make(map[string]*NodeEWMA),
		alpha:         0.25,
		beta:          0.25,
		maxWindowSize: 10,
		defaultMargin: 25,
		defaultRatio:  0.15,
		cooldown:      20 * time.Second,
	}
}

// RecordSample registers a latency observation for the specified node, updating SRTT, RTTVAR, and stability metrics.
func (m *EWMAManager) RecordSample(nodeName string, rawDelayMs int, isSuccess bool) NodeEWMA {
	if nodeName == "" {
		return NodeEWMA{}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	item, exists := m.nodes[nodeName]
	if !exists {
		item = &NodeEWMA{
			NodeName: nodeName,
			Samples:  make([]int, 0, m.maxWindowSize),
		}
		m.nodes[nodeName] = item
	}

	now := time.Now()
	item.LastSampleAt = now
	item.TotalSamples++

	if isSuccess && rawDelayMs > 0 {
		// Append to sliding window
		if len(item.Samples) >= m.maxWindowSize {
			item.Samples = append(item.Samples[1:], rawDelayMs)
		} else {
			item.Samples = append(item.Samples, rawDelayMs)
		}

		delayF := float64(rawDelayMs)
		if item.SRTT <= 0 {
			// First sample: initialize SRTT and estimate initial variance
			item.SRTT = delayF
			item.RTTVAR = delayF / 2.0
		} else {
			// RFC 6298 / TCP Jacobson-Karels EWMA smoothing
			diff := math.Abs(delayF - item.SRTT)
			item.RTTVAR = (1.0-m.beta)*item.RTTVAR + m.beta*diff
			item.SRTT = (1.0-m.alpha)*item.SRTT + m.alpha*delayF
		}

		item.ConsecutiveFails = 0
	} else {
		// Timeout / failure penalty
		item.ConsecutiveFails++
		if len(item.Samples) >= m.maxWindowSize {
			item.Samples = append(item.Samples[1:], 0)
		} else {
			item.Samples = append(item.Samples, 0)
		}
	}

	// Compute LossRate in sliding window
	if len(item.Samples) > 0 {
		failCount := 0
		for _, s := range item.Samples {
			if s <= 0 {
				failCount++
			}
		}
		item.LossRate = float64(failCount) / float64(len(item.Samples))
	}

	// Calculate EffectiveDelay: SRTT + 1.5 * Jitter + Failure Penalty
	// Failure penalty adds 200ms per consecutive timeout, up to 3000ms
	failPenalty := float64(item.ConsecutiveFails * 200)
	if failPenalty > 3000 {
		failPenalty = 3000
	}

	eff := item.SRTT + (1.5 * item.RTTVAR) + failPenalty
	if eff < 1 {
		eff = 1
	}
	item.EffectiveDelay = int(math.Round(eff))

	// Stability score calculation (0 - 100)
	score := 100.0

	// Latency deduction
	if item.SRTT > 300 {
		score -= (item.SRTT - 300) * 0.1
	} else if item.SRTT > 100 {
		score -= (item.SRTT - 100) * 0.05
	}

	// Jitter deduction (high variance penalizes stability)
	if item.RTTVAR > 30 {
		score -= (item.RTTVAR - 30) * 0.5
	}

	// Loss penalty (heavy penalty for packet loss/timeout)
	score -= item.LossRate * 50.0
	score -= float64(item.ConsecutiveFails) * 15.0

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	item.StabilityScore = math.Round(score*10) / 10

	// Grade classification
	if item.ConsecutiveFails >= 2 || item.LossRate >= 0.5 {
		item.Grade = "C 离线/丢包"
	} else if item.StabilityScore >= 90.0 {
		item.Grade = "S 极品稳态"
	} else if item.StabilityScore >= 80.0 {
		item.Grade = "A+ 优质平稳"
	} else if item.StabilityScore >= 65.0 {
		item.Grade = "A 常规可用"
	} else {
		item.Grade = "B 波动较大"
	}

	return *item
}

// GetNodeScore returns the current EWMA evaluation for a given node.
func (m *EWMAManager) GetNodeScore(nodeName string) (NodeEWMA, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.nodes[nodeName]
	if !exists {
		return NodeEWMA{NodeName: nodeName}, false
	}
	return *item, true
}

// GetAllScores returns a snapshot copy of all tracked nodes' EWMA states.
func (m *EWMAManager) GetAllScores() map[string]NodeEWMA {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make(map[string]NodeEWMA, len(m.nodes))
	for k, v := range m.nodes {
		res[k] = *v
	}
	return res
}

// ShouldSwitch arbitrates whether a group selection should transition from currentActive to candidate.
// It applies dual-threshold anti-flapping (absolute margin in ms AND relative percentage improvement)
// as well as switching cooldown to suppress rapid flap oscillation.
func (m *EWMAManager) ShouldSwitch(
	currentActive string,
	candidate string,
	customMarginMs int,
	customMarginRatio float64,
	customCooldown time.Duration,
) (bool, string) {
	if currentActive == "" || candidate == "" || currentActive == candidate {
		return false, "相同节点无需切换"
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	act, actExists := m.nodes[currentActive]
	cand, candExists := m.nodes[candidate]

	// 1. If active node does not exist or has failed continuously, switch immediately to valid candidate
	if !actExists || act.ConsecutiveFails >= 2 || (act.LossRate >= 0.6 && act.TotalSamples >= 3) {
		if candExists && cand.ConsecutiveFails == 0 && cand.EffectiveDelay > 0 {
			if cand != nil {
				cand.LastSwitchAt = time.Now()
			}
			return true, "当前活跃节点异常失效，执行紧急无感避障切流"
		}
		return true, "当前节点不可用，切至候选节点"
	}

	// 2. Candidate must be healthy
	if !candExists || cand.ConsecutiveFails > 0 || cand.EffectiveDelay <= 0 {
		return false, "候选节点未就绪或存在丢包，放弃切线"
	}

	// 3. Cooldown check: prevent rapid switching back and forth
	cd := m.cooldown
	if customCooldown > 0 {
		cd = customCooldown
	}
	if !act.LastSwitchAt.IsZero() && time.Since(act.LastSwitchAt) < cd {
		remaining := cd - time.Since(act.LastSwitchAt)
		return false, fmt.Sprintf("防抖冷却期生效中 (剩余 %.1fs)，抑制频繁切线", remaining.Seconds())
	}

	// 4. Threshold evaluation
	marginMs := m.defaultMargin
	if customMarginMs > 0 {
		marginMs = customMarginMs
	}
	marginRatio := m.defaultRatio
	if customMarginRatio > 0 {
		marginRatio = customMarginRatio
	}

	diffMs := act.EffectiveDelay - cand.EffectiveDelay
	if diffMs <= 0 {
		return false, fmt.Sprintf("候选节点等效延迟 (%dms) 未优于当前节点 (%dms)", cand.EffectiveDelay, act.EffectiveDelay)
	}

	ratio := float64(diffMs) / float64(act.EffectiveDelay)

	// Both absolute margin and relative ratio must be satisfied to trigger a switch
	if diffMs >= marginMs && ratio >= marginRatio {
		cand.LastSwitchAt = time.Now()
		if act != nil {
			act.LastSwitchAt = time.Now()
		}
		return true, fmt.Sprintf("候选节点综合防抖时延优于当前节点 %dms (改善 %.1f%%)，超越防抖阈值 (%dms, %.0f%%)",
			diffMs, ratio*100.0, marginMs, marginRatio*100.0)
	}

	return false, fmt.Sprintf("候选节点微小优势 %dms (改善 %.1f%%) 处于防抖抑制区间 (阈值: %dms, %.0f%%)，维持稳态",
		diffMs, ratio*100.0, marginMs, marginRatio*100.0)
}

// MarkSwitch registers an intentional manual or external switch event to initialize cooldown.
func (m *EWMAManager) MarkSwitch(nodeName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if item, ok := m.nodes[nodeName]; ok {
		item.LastSwitchAt = time.Now()
	}
}

// CandidateRank packages a real-time candidate ranking snapshot.
type CandidateRank struct {
	NodeName         string
	EffectiveDelay   int
	StabilityScore   float64
	ConsecutiveFails int
	HasSample        bool
}

// RankNodes sorts candidate node names according to EWMA quality metrics.
// Sorting priority: 1. ConsecutiveFails == 0  2. HasSample == true  3. EffectiveDelay asc  4. StabilityScore desc
func (m *EWMAManager) RankNodes(nodeNames []string) []CandidateRank {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ranks := make([]CandidateRank, 0, len(nodeNames))
	for _, name := range nodeNames {
		stat, ok := m.nodes[name]
		if !ok || len(stat.Samples) == 0 {
			ranks = append(ranks, CandidateRank{
				NodeName:       name,
				EffectiveDelay: 9999,
				HasSample:      false,
			})
			continue
		}
		ranks = append(ranks, CandidateRank{
			NodeName:         name,
			EffectiveDelay:   stat.EffectiveDelay,
			StabilityScore:   stat.StabilityScore,
			ConsecutiveFails: stat.ConsecutiveFails,
			HasSample:        true,
		})
	}

	sort.Slice(ranks, func(i, j int) bool {
		a, b := ranks[i], ranks[j]
		aHealthy := a.ConsecutiveFails == 0
		bHealthy := b.ConsecutiveFails == 0
		if aHealthy != bHealthy {
			return aHealthy
		}
		if a.HasSample != b.HasSample {
			return a.HasSample
		}
		if a.EffectiveDelay != b.EffectiveDelay {
			return a.EffectiveDelay < b.EffectiveDelay
		}
		return a.StabilityScore > b.StabilityScore
	})

	return ranks
}

// GetNodeSnapshot safely copies a single node's EWMA state.
func (m *EWMAManager) GetNodeSnapshot(nodeName string) (NodeEWMA, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.nodes[nodeName]
	if !ok || n == nil {
		return NodeEWMA{}, false
	}
	cp := *n
	return cp, true
}

// MarkGroupSwitched records node takeover timestamp to activate cooldown dampening window.
func (m *EWMAManager) MarkGroupSwitched(newActiveNode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n, ok := m.nodes[newActiveNode]; ok && n != nil {
		n.LastSwitchAt = time.Now()
	}
}

// PruneStaleNodes removes departed nodes to prevent map memory expansion.
func (m *EWMAManager) PruneStaleNodes(validNodes map[string]struct{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name := range m.nodes {
		if _, ok := validNodes[name]; !ok {
			delete(m.nodes, name)
		}
	}
}

// ResetPenalty decays or clears the passive blackhole penalty for a node after cooldown expires,
// restoring its candidacy and resetting failure counters.
func (m *EWMAManager) ResetPenalty(nodeName string) {
	if nodeName == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	item, exists := m.nodes[nodeName]
	if !exists || item == nil {
		return
	}

	if item.ConsecutiveFails > 0 {
		item.ConsecutiveFails = 0
		eff := item.SRTT + (1.5 * item.RTTVAR)
		if eff < 1 {
			eff = 1
		}
		item.EffectiveDelay = int(math.Round(eff))
		if item.StabilityScore < 60 {
			item.StabilityScore = 60
		}
		item.Grade = "A 常规可用"
	}
}

