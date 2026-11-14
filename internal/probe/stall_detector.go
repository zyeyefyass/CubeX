package probe

import (
	"sync/atomic"
	"time"
)

// PassiveStallDetector 监控代理隧道是否发生“有上行请求、但连续 N 秒 0 下行响应”的假活死链状态
type PassiveStallDetector struct {
	stallTicks       atomic.Int32
	lastTriggerMilli atomic.Int64
	thresholdTicks   int32 // 默认连续 6 秒
	cooldownMs       int64 // 默认 30000ms (30s 冷却锁，防止频繁误触发)
}

// NewPassiveStallDetector 初始化被动死链检测器
func NewPassiveStallDetector() *PassiveStallDetector {
	return &PassiveStallDetector{
		thresholdTicks: 6,
		cooldownMs:     30000,
	}
}

// ObserveSecond 传入当前每秒活跃代理连接数、代理上行速率(B/s)、代理下行速率(B/s)
// 若返回 true，代表当前现役节点已发生死链假死，应立即触发 EWMA 切流与 WANHealer 冲刷
func (d *PassiveStallDetector) ObserveSecond(activeProxyConns int, proxyUpBps int64, proxyDownBps int64) bool {
	// 仅在存在多条活跃连接、且客户端正在持续发送请求(Up > 1KB/s)、但下行几乎为 0 (< 64B/s) 时累加异常计数
	if activeProxyConns >= 4 && proxyUpBps > 1024 && proxyDownBps < 64 {
		cur := d.stallTicks.Add(1)
		if cur >= d.thresholdTicks {
			d.stallTicks.Store(0)
			now := time.Now().UnixMilli()
			last := d.lastTriggerMilli.Load()
			if now-last >= d.cooldownMs && d.lastTriggerMilli.CompareAndSwap(last, now) {
				return true
			}
		}
		return false
	}

	// 一旦恢复正常下行流量，立即清零异常计数
	d.stallTicks.Store(0)
	return false
}

// Reset 强制重置检测状态
func (d *PassiveStallDetector) Reset() {
	d.stallTicks.Store(0)
}
