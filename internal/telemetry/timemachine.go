package telemetry

import (
	"sync"
	"time"
)

// TimeMachinePoint 24小时流量时光机时间切片样本点
type TimeMachinePoint struct {
	Timestamp         string  `json:"timestamp"`
	DownloadBps       int64   `json:"download_bps"`
	UploadBps         int64   `json:"upload_bps"`
	DirectBps         int64   `json:"direct_bps"`
	ProxyBps          int64   `json:"proxy_bps"`
	ActiveConnections int     `json:"active_connections"`
	DNSQueries        int64   `json:"dns_queries"`
	LatencyMs         float64 `json:"latency_ms"`
}

// TimeMachineSummary 24小时时光机汇总与时间序列
type TimeMachineSummary struct {
	PeakDownloadBps   int64              `json:"peak_download_bps"`
	PeakUploadBps     int64              `json:"peak_upload_bps"`
	TotalDownloadedMB float64            `json:"total_downloaded_mb"`
	TotalUploadedMB   float64            `json:"total_uploaded_mb"`
	AverageLatencyMs  float64            `json:"average_latency_ms"`
	TotalPoints       int                `json:"total_points"`
	Points            []TimeMachinePoint `json:"points"`
}

// TimeMachine 环形缓冲采样管理器
type TimeMachine struct {
	mu       sync.RWMutex
	maxSize  int
	points   []TimeMachinePoint
	cursor   int
	isFull   bool
}

// NewTimeMachine 实例化时光机管理器 (默认 288 个样本点，覆盖 24 小时 5 分钟采样)
func NewTimeMachine(maxPoints int) *TimeMachine {
	if maxPoints <= 0 {
		maxPoints = 288 // 24h * 12 points/hr = 288
	}
	return &TimeMachine{
		maxSize: maxPoints,
		points:  make([]TimeMachinePoint, maxPoints),
	}
}

// RecordPoint 写入一个 5 分钟周期的流量切片
func (tm *TimeMachine) RecordPoint(dlBps, ulBps, dirBps, prxBps int64, conns int, dns int64, latMs float64) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	pt := TimeMachinePoint{
		Timestamp:         time.Now().Format("15:04"),
		DownloadBps:       dlBps,
		UploadBps:         ulBps,
		DirectBps:         dirBps,
		ProxyBps:          prxBps,
		ActiveConnections: conns,
		DNSQueries:        dns,
		LatencyMs:         latMs,
	}

	tm.points[tm.cursor] = pt
	tm.cursor++
	if tm.cursor >= tm.maxSize {
		tm.cursor = 0
		tm.isFull = true
	}
}

// Get24HHistory 获取按时间顺序排列的 24 小时历史时间序列与统计
func (tm *TimeMachine) Get24HHistory() TimeMachineSummary {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var ordered []TimeMachinePoint
	if !tm.isFull {
		ordered = make([]TimeMachinePoint, tm.cursor)
		copy(ordered, tm.points[:tm.cursor])
	} else {
		ordered = make([]TimeMachinePoint, tm.maxSize)
		// 从最老的数据复制到 cursor-1
		copy(ordered, tm.points[tm.cursor:])
		copy(ordered[tm.maxSize-tm.cursor:], tm.points[:tm.cursor])
	}

	summary := TimeMachineSummary{
		Points:      ordered,
		TotalPoints: len(ordered),
	}

	if len(ordered) == 0 {
		return summary
	}

	var totalDl, totalUl int64
	var sumLat float64

	for _, p := range ordered {
		if p.DownloadBps > summary.PeakDownloadBps {
			summary.PeakDownloadBps = p.DownloadBps
		}
		if p.UploadBps > summary.PeakUploadBps {
			summary.PeakUploadBps = p.UploadBps
		}
		// 5 分钟 (300 秒) 积分计算流量总量
		totalDl += p.DownloadBps * 300
		totalUl += p.UploadBps * 300
		sumLat += p.LatencyMs
	}

	summary.TotalDownloadedMB = float64(totalDl) / (8 * 1024 * 1024)
	summary.TotalUploadedMB = float64(totalUl) / (8 * 1024 * 1024)
	summary.AverageLatencyMs = sumLat / float64(len(ordered))

	return summary
}
