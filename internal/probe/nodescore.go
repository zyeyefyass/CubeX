package probe

import (
	"math"
	"strings"
	"time"
)

// NodeScoreResult represents the multi-dimensional comprehensive quality evaluation for a proxy node.
type NodeScoreResult struct {
	NodeName        string    `json:"node_name"`
	TotalScore      float64   `json:"total_score"`      // 0 - 100
	Grade           string    `json:"grade"`            // "S 极品金牌", "A+ 卓越首选", "A 优质推荐", "B 普通可用", "C 慢速备用"
	LatencyScore    float64   `json:"latency_score"`    // 0 - 100 (Weight: 40%)
	SLAScore        float64   `json:"sla_score"`        // 0 - 100 (Weight: 30%)
	ThroughputScore float64   `json:"throughput_score"` // 0 - 100 (Weight: 20%)
	IPQualityScore  float64   `json:"ip_quality_score"` // 0 - 100 (Weight: 10%)
	IsRecommended   bool      `json:"is_recommended"`   // True if Score >= 80 (S or A+)
	EvaluatedAt     time.Time `json:"evaluated_at"`
}

// EvaluateNodeScore calculates a balanced, weighted composite score across 4 key dimensions:
// 1. Latency (40%): Real network round-trip time.
// 2. SLA Availability (30%): Historical uptime and stability.
// 3. Throughput / Bandwidth (20%): Measured speed slice or baseline capacity.
// 4. IP Intelligence & Purity (10%): Residential ISP bonus vs Datacenter risk penalty.
func EvaluateNodeScore(
	nodeName string,
	latencyMs int,
	availRate float64,
	throughputMbps float64,
	ipType string,
	riskScore int,
) NodeScoreResult {
	latScore := calcLatencyScore(latencyMs)
	slaScore := calcSLAScore(availRate)
	tpScore := calcThroughputScore(throughputMbps)
	ipScore := calcIPQualityScore(ipType, riskScore)

	// Composite weighted calculation
	total := (0.40 * latScore) + (0.30 * slaScore) + (0.20 * tpScore) + (0.10 * ipScore)
	total = math.Round(total*10) / 10 // 1 decimal place

	grade := "C 慢速备用"
	recommended := false

	if total >= 90.0 {
		grade = "S 极品金牌"
		recommended = true
	} else if total >= 80.0 {
		grade = "A+ 卓越首选"
		recommended = true
	} else if total >= 70.0 {
		grade = "A 优质推荐"
	} else if total >= 50.0 {
		grade = "B 普通可用"
	}

	return NodeScoreResult{
		NodeName:        nodeName,
		TotalScore:      total,
		Grade:           grade,
		LatencyScore:    math.Round(latScore*10) / 10,
		SLAScore:        math.Round(slaScore*10) / 10,
		ThroughputScore: math.Round(tpScore*10) / 10,
		IPQualityScore:  math.Round(ipScore*10) / 10,
		IsRecommended:   recommended,
		EvaluatedAt:     time.Now(),
	}
}

func calcLatencyScore(lat int) float64 {
	if lat <= 0 || lat >= 2500 {
		return 0.0
	}
	if lat <= 60 {
		return 100.0
	}
	if lat <= 150 {
		// 60~150ms: 100 -> 80
		return 100.0 - float64(lat-60)*(20.0/90.0)
	}
	if lat <= 300 {
		// 150~300ms: 80 -> 50
		return 80.0 - float64(lat-150)*(30.0/150.0)
	}
	if lat <= 800 {
		// 300~800ms: 50 -> 20
		return 50.0 - float64(lat-300)*(30.0/500.0)
	}
	// 800~2500ms: 20 -> 0
	score := 20.0 - float64(lat-800)*(20.0/1700.0)
	if score < 0 {
		return 0
	}
	return score
}

func calcSLAScore(avail float64) float64 {
	if avail < 0 {
		return 0.0
	}
	if avail > 100 {
		return 100.0
	}
	return avail
}

func calcThroughputScore(mbps float64) float64 {
	// Untested throughput returns a neutral baseline score
	if mbps <= 0 {
		return 60.0
	}
	if mbps >= 100.0 {
		return 100.0
	}
	if mbps >= 50.0 {
		return 85.0 + (mbps-50.0)*0.3 // 85 -> 100
	}
	if mbps >= 20.0 {
		return 70.0 + (mbps-20.0)*0.5 // 70 -> 85
	}
	if mbps >= 5.0 {
		return 50.0 + (mbps-5.0)*1.33 // 50 -> 70
	}
	score := 30.0 + mbps*4.0
	if score < 10.0 {
		return 10.0
	}
	return score
}

func calcIPQualityScore(ipType string, riskScore int) float64 {
	base := 75.0
	t := strings.ToLower(ipType)
	if strings.Contains(t, "residential") || strings.Contains(t, "原生") {
		base = 95.0
	} else if strings.Contains(t, "business") || strings.Contains(t, "专线") {
		base = 88.0
	} else if strings.Contains(t, "datacenter") || strings.Contains(t, "机房") {
		base = 70.0
	}

	// Penalty for fraud / blacklist risk score (0=clean, 100=bad)
	penalty := float64(riskScore) * 0.4
	score := base - penalty
	if score < 10.0 {
		score = 10.0
	}
	if score > 100.0 {
		score = 100.0
	}
	return score
}
