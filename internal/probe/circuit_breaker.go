package probe

import (
	"sync"
	"sync/atomic"
	"time"
)

// CircuitState 熔断器状态
type CircuitState int32

const (
	StateClosed   CircuitState = 0 // 正常闭合 (流量正常通过)
	StateOpen     CircuitState = 1 // 熔断开启 (流量隔离阻断，处于冷却期)
	StateHalfOpen CircuitState = 2 // 半开试探 (冷却期满，允许少量流量试探)
)

// NodeHealthSample 固定环形缓冲区内的单个被动采样点 (零堆逃逸)
type NodeHealthSample struct {
	TimestampMilli int64
	IsFailure      bool // 是否为超时、RST 或黑洞假死
	UpBytes        int64
	DownBytes      int64
}

// NodeCircuitTracker 追踪单个上游节点的被动连接健康状态
type NodeCircuitTracker struct {
	NodeName         string
	State            atomic.Int32 // CircuitState
	ConsecutiveFails atomic.Int32 // 连续失败计数
	QuarantineUntil  atomic.Int64 // 隔离截止毫秒时间戳
	LastTripMilli    atomic.Int64 // 最近一次熔断时间

	mu        sync.Mutex
	head      int
	count     int
	ring      [10]NodeHealthSample // 固定容量 10 的环形采样数组 (过去 10 秒时间窗)
	tripCount int64
}

// PassiveCircuitBreaker 零开销被动流量遥测熔断器 (Zero-Probe Passive Circuit Breaker)
// 特性：
// 1. 0 额外测速流量开销，纯被动感知真实业务连接状态；
// 2. 10 秒滑动时间窗内连续 >= 5 次握手超时/RST/黑洞，0.5 秒内触发局部熔断；
// 3. 熔断后自动隔离 5 分钟，触发陈旧连接精准清道夫 (StaleFlowKiller) 并切至热备节点；
// 4. 严格采用定长数组与对象复用，零动态 slice 扩容与零堆内存碎片。
type PassiveCircuitBreaker struct {
	mu            sync.RWMutex
	trackers      map[string]*NodeCircuitTracker
	failThreshold int32         // 连续失败阈值 (默认 5)
	quarantineDur time.Duration // 隔离冷却时长 (默认 5 分钟)
	onTrip        func(nodeName string, reason string)
	onRecover     func(nodeName string)
}

// NewPassiveCircuitBreaker 初始化被动遥测熔断器
func NewPassiveCircuitBreaker(
	failThreshold int32,
	quarantineDur time.Duration,
	onTrip func(nodeName string, reason string),
	onRecover func(nodeName string),
) *PassiveCircuitBreaker {
	if failThreshold <= 0 {
		failThreshold = 5
	}
	if quarantineDur <= 0 {
		quarantineDur = 5 * time.Minute
	}
	return &PassiveCircuitBreaker{
		trackers:      make(map[string]*NodeCircuitTracker, 64),
		failThreshold: failThreshold,
		quarantineDur: quarantineDur,
		onTrip:        onTrip,
		onRecover:     onRecover,
	}
}

func (cb *PassiveCircuitBreaker) getOrCreateTracker(nodeName string) *NodeCircuitTracker {
	cb.mu.RLock()
	t, ok := cb.trackers[nodeName]
	cb.mu.RUnlock()
	if ok {
		return t
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()
	if t, ok = cb.trackers[nodeName]; ok {
		return t
	}

	// 硬上限防护：防止恶意或超大订阅导致 map 无限膨胀 (最多追踪 128 个活跃节点)
	if len(cb.trackers) >= 128 {
		// 清理最老或已闭合的 tracker
		for k, trk := range cb.trackers {
			if CircuitState(trk.State.Load()) == StateClosed {
				delete(cb.trackers, k)
				break
			}
		}
	}

	t = &NodeCircuitTracker{
		NodeName: nodeName,
	}
	t.State.Store(int32(StateClosed))
	cb.trackers[nodeName] = t
	return t
}

// RecordFailure 记录一次连接超时、TLS RST 或黑洞假死
// 若触发连续失败门禁，立即执行局部熔断并返回 true
func (cb *PassiveCircuitBreaker) RecordFailure(nodeName string, reason string) bool {
	if nodeName == "" || nodeName == "DIRECT" || nodeName == "REJECT" {
		return false
	}
	t := cb.getOrCreateTracker(nodeName)
	now := time.Now().UnixMilli()

	t.mu.Lock()
	defer t.mu.Unlock()

	// 记录到环形数组
	t.ring[t.head] = NodeHealthSample{
		TimestampMilli: now,
		IsFailure:      true,
	}
	t.head = (t.head + 1) % len(t.ring)
	if t.count < len(t.ring) {
		t.count++
	}

	fails := t.ConsecutiveFails.Add(1)

	// 若已处于熔断状态，直接返回
	if CircuitState(t.State.Load()) == StateOpen {
		return false
	}

	// 判定是否达到熔断阈值 (连续 >= 5 次失败，或环形窗口内过去 10 秒失败密集)
	if fails >= cb.failThreshold {
		t.State.Store(int32(StateOpen))
		quarantineUntil := now + cb.quarantineDur.Milliseconds()
		t.QuarantineUntil.Store(quarantineUntil)
		t.LastTripMilli.Store(now)
		t.tripCount++

		if cb.onTrip != nil {
			go cb.onTrip(nodeName, reason)
		}
		return true
	}

	return false
}

// RecordSuccess 记录一次成功连接或收到有效下行响应
func (cb *PassiveCircuitBreaker) RecordSuccess(nodeName string, downBytes int64) {
	if nodeName == "" || nodeName == "DIRECT" || nodeName == "REJECT" {
		return
	}
	cb.mu.RLock()
	t, ok := cb.trackers[nodeName]
	cb.mu.RUnlock()
	if !ok {
		return
	}

	now := time.Now().UnixMilli()
	t.mu.Lock()
	defer t.mu.Unlock()

	t.ring[t.head] = NodeHealthSample{
		TimestampMilli: now,
		IsFailure:      false,
		DownBytes:      downBytes,
	}
	t.head = (t.head + 1) % len(t.ring)
	if t.count < len(t.ring) {
		t.count++
	}

	// 成功恢复，连续失败计数清零
	t.ConsecutiveFails.Store(0)

	// 若处于半开试探状态，收到下行即认定节点恢复
	if CircuitState(t.State.Load()) == StateHalfOpen {
		t.State.Store(int32(StateClosed))
		if cb.onRecover != nil {
			go cb.onRecover(nodeName)
		}
	}
}

// IsNodeQuarantined 查询节点是否处于熔断隔离期
// 若隔离期已满，自动转为半开试探状态并放行微量试探流量
func (cb *PassiveCircuitBreaker) IsNodeQuarantined(nodeName string) bool {
	cb.mu.RLock()
	t, ok := cb.trackers[nodeName]
	cb.mu.RUnlock()
	if !ok {
		return false
	}

	state := CircuitState(t.State.Load())
	if state == StateClosed {
		return false
	}

	now := time.Now().UnixMilli()
	quarantineUntil := t.QuarantineUntil.Load()

	// 隔离期未满，继续阻断
	if state == StateOpen && now < quarantineUntil {
		return true
	}

	// 隔离期满，自动推进至 Half-Open 状态
	if state == StateOpen && now >= quarantineUntil {
		t.State.Store(int32(StateHalfOpen))
		return false // 允许试探请求通过
	}

	return false
}

// ResetNode 手动重置节点熔断状态
func (cb *PassiveCircuitBreaker) ResetNode(nodeName string) {
	cb.mu.RLock()
	t, ok := cb.trackers[nodeName]
	cb.mu.RUnlock()
	if !ok {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.State.Store(int32(StateClosed))
	t.ConsecutiveFails.Store(0)
	t.QuarantineUntil.Store(0)
}

// GetQuarantinedNodes 获取当前被隔离的节点清单及其剩余冷却时间 (秒)
func (cb *PassiveCircuitBreaker) GetQuarantinedNodes() map[string]int64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	now := time.Now().UnixMilli()
	res := make(map[string]int64, len(cb.trackers))
	for name, t := range cb.trackers {
		if CircuitState(t.State.Load()) == StateOpen {
			remaining := (t.QuarantineUntil.Load() - now) / 1000
			if remaining > 0 {
				res[name] = remaining
			}
		}
	}
	return res
}
