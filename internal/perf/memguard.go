package perf

import (
	"bytes"
	"context"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// GlobalBufferPool 全链路零分配 Buffer 复用池
// 特性：统一从全局 sync.Pool 借用 bytes.Buffer，并在归还时拦截超过 16KB 的异常大 Buffer 防止池膨胀
var GlobalBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 1024))
	},
}

// GetBuffer 从池中获取 Buffer
func GetBuffer() *bytes.Buffer {
	buf := GlobalBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer 归还 Buffer (超过 16KB 自动丢弃防膨胀)
func PutBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	if buf.Cap() > 16*1024 {
		// 超过 16KB 的超大 Buffer 丢弃，交由 GC 回收，防止池长期霸占物理内存
		return
	}
	buf.Reset()
	GlobalBufferPool.Put(buf)
}

// MemoryGuard 运行时内存软上限护栏与闲时物理页归还管理器
type MemoryGuard struct {
	limitBytes         int64
	idleScavengeCount  atomic.Uint64
	lastScavengeMilli  atomic.Int64
	isWebUIActiveFunc  func() bool
}

// NewMemoryGuard 初始化内存护栏
func NewMemoryGuard(limitMiB int, isWebUIActive func() bool) *MemoryGuard {
	if limitMiB <= 0 {
		limitMiB = 24
	}
	limitBytes := int64(limitMiB) * 1024 * 1024
	debug.SetMemoryLimit(limitBytes)

	return &MemoryGuard{
		limitBytes:        limitBytes,
		isWebUIActiveFunc: isWebUIActive,
	}
}

// ScavengeNow 立即触发一次强制 GC 与物理页归还 (FreeOSMemory)
func (mg *MemoryGuard) ScavengeNow() {
	runtime.GC()
	debug.FreeOSMemory()
	now := time.Now().UnixMilli()
	mg.lastScavengeMilli.Store(now)
	mg.idleScavengeCount.Add(1)
}

// StartIdleScavenger 启动后台闲时物理页归还协程 (每 3 分钟检查一次)
func (mg *MemoryGuard) StartIdleScavenger(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 3 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 仅当 Web UI 处于非活跃/无客户端连入/后台休眠态时，执行深度物理页归还
			if mg.isWebUIActiveFunc == nil || !mg.isWebUIActiveFunc() {
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				// 仅在有空闲或待释放堆内存 (> 2MB) 时才主动触发，避免无意义的频繁 CPU 唤醒
				if m.HeapIdle > 2*1024*1024 || m.HeapReleased > 0 {
					mg.ScavengeNow()
				}
			}
		}
	}
}

// GetScavengeStats 获取巡航归还统计
func (mg *MemoryGuard) GetScavengeStats() (count uint64, lastTime int64) {
	return mg.idleScavengeCount.Load(), mg.lastScavengeMilli.Load()
}
