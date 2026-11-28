package perf

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DetectTotalRAMMb 统一从 cgroups v2/v1 与 /proc/meminfo 探测容器或宿主机可用内存上限（MiB）。
// 取有效正值的最小值以防止容器内存限制被宿主机物理内存掩盖；全不可读时安全回退 1024。
func DetectTotalRAMMb() int64 {
	totalMb := int64(0)
	updateMin := func(mb int64) {
		if mb > 0 && (totalMb == 0 || mb < totalMb) {
			totalMb = mb
		}
	}

	// 1. cgroups v2 memory.max
	if data, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		str := strings.TrimSpace(string(data))
		if str != "" && str != "max" {
			if bytesVal, err := strconv.ParseInt(str, 10, 64); err == nil && bytesVal > 0 {
				updateMin(bytesVal / (1024 * 1024))
			}
		}
	}

	// 2. cgroups v1 memory.limit_in_bytes
	if data, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil {
		str := strings.TrimSpace(string(data))
		if str != "" {
			if bytesVal, err := strconv.ParseInt(str, 10, 64); err == nil && bytesVal > 0 && bytesVal < (1<<60) {
				updateMin(bytesVal / (1024 * 1024))
			}
		}
	}

	// 3. /proc/meminfo MemTotal
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if totalKb, err := strconv.ParseInt(fields[1], 10, 64); err == nil && totalKb > 0 {
						updateMin(totalKb / 1024)
					}
				}
				break
			}
		}
	}

	if totalMb <= 0 {
		return 1024
	}
	return totalMb
}

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
	limitBytes        int64
	idleScavengeCount atomic.Uint64
	lastScavengeMilli atomic.Int64
	isWebUIActiveFunc func() bool
}

// NewMemoryGuard 初始化内存护栏。
// 当 limitMiB <= 0 时不覆盖外部已配置的 GOMEMLIMIT，仅读取当前生效上限，避免与自适应限制互相覆盖。
func NewMemoryGuard(limitMiB int, isWebUIActive func() bool) *MemoryGuard {
	var limitBytes int64
	if limitMiB > 0 {
		limitBytes = int64(limitMiB) * 1024 * 1024
		debug.SetMemoryLimit(limitBytes)
	} else {
		cur := debug.SetMemoryLimit(-1)
		if cur > 0 && cur < (1<<50) {
			limitBytes = cur
		} else {
			limitBytes = 24 * 1024 * 1024
			debug.SetMemoryLimit(limitBytes)
		}
	}

	return &MemoryGuard{
		limitBytes:        limitBytes,
		isWebUIActiveFunc: isWebUIActive,
	}
}

// LimitMiB 返回当前生效的内存软上限（MiB）。
func (mg *MemoryGuard) LimitMiB() int64 {
	if mg == nil || mg.limitBytes <= 0 {
		return 24
	}
	return mg.limitBytes / (1024 * 1024)
}

// ScavengeNow 立即触发一次强制 GC 与物理页归还 (FreeOSMemory)
func (mg *MemoryGuard) ScavengeNow() {
	runtime.GC()
	debug.FreeOSMemory()
	now := time.Now().UnixMilli()
	mg.lastScavengeMilli.Store(now)
	mg.idleScavengeCount.Add(1)
}

// StartIdleScavenger 启动后台闲时物理页归还协程 (首次 15s 冷启动收敛，随后每 interval 检查一次)
func (mg *MemoryGuard) StartIdleScavenger(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 3 * time.Minute
	}

	// 冷启动 15s 后触发一次温和归还，释放初始化期临时堆页
	warmupDelay := 15 * time.Second
	if interval < warmupDelay {
		warmupDelay = interval
	}
	select {
	case <-ctx.Done():
		return
	case <-time.After(warmupDelay):
		if mg.isWebUIActiveFunc == nil || !mg.isWebUIActiveFunc() {
			mg.ScavengeNow()
		}
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
