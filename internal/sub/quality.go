package sub

import (
	"sort"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

// SubQualityReport represents an aggregated SLA quality matrix for a subscription source.
type SubQualityReport struct {
	SubscriptionID   string    `json:"subscription_id"`
	SubscriptionName string    `json:"subscription_name"`
	TotalNodes       int       `json:"total_nodes"`
	OnlineNodes      int       `json:"online_nodes"`
	OfflineNodes     int       `json:"offline_nodes"`
	AvailabilityRate float64   `json:"availability_rate"` // e.g., 96.5%
	MedianLatencyMs  int       `json:"median_latency_ms"`
	AvgLatencyMs     int       `json:"avg_latency_ms"`
	HealthGrade      string    `json:"health_grade"` // "A+ 卓越", "A 优秀", "B 良好", "C 需关注"
	OfflineNodeNames []string  `json:"offline_node_names"`
	EvaluatedAt      time.Time `json:"evaluated_at"`
}

// EvaluateQuality computes the SLA quality report for a given subscription and its node latency map.
func EvaluateQuality(subID, subName string, nodes []schema.ProxyNode, latencyMap map[string]int) SubQualityReport {
	report := SubQualityReport{
		SubscriptionID:   subID,
		SubscriptionName: subName,
		TotalNodes:       len(nodes),
		OfflineNodeNames: make([]string, 0),
		EvaluatedAt:      time.Now(),
	}

	if len(nodes) == 0 {
		report.HealthGrade = "暂无节点"
		return report
	}

	var validLatencies []int
	var latencySum int

	for _, n := range nodes {
		lat, ok := latencyMap[n.Name]
		if ok && lat > 0 && lat < 9999 {
			report.OnlineNodes++
			validLatencies = append(validLatencies, lat)
			latencySum += lat
		} else {
			report.OfflineNodes++
			report.OfflineNodeNames = append(report.OfflineNodeNames, n.Name)
		}
	}

	if report.TotalNodes > 0 {
		report.AvailabilityRate = (float64(report.OnlineNodes) / float64(report.TotalNodes)) * 100.0
	}

	if len(validLatencies) > 0 {
		sort.Ints(validLatencies)
		report.MedianLatencyMs = validLatencies[len(validLatencies)/2]
		report.AvgLatencyMs = latencySum / len(validLatencies)
	}

	// Grade evaluation
	switch {
	case report.AvailabilityRate >= 95.0 && report.MedianLatencyMs > 0 && report.MedianLatencyMs <= 100:
		report.HealthGrade = "A+ 暮光卓越"
	case report.AvailabilityRate >= 85.0 && report.MedianLatencyMs <= 200:
		report.HealthGrade = "A 稳定优质"
	case report.AvailabilityRate >= 70.0:
		report.HealthGrade = "B 基本可用"
	default:
		report.HealthGrade = "C 节点严重降级"
	}

	return report
}
