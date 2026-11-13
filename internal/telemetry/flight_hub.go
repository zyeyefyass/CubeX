package telemetry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// EventCategory 定义黑匣子事件分类
type EventCategory string

const (
	CatEngineSwitch EventCategory = "ENGINE_SWITCH" // 双擎热切事件
	CatHAFailover   EventCategory = "HA_WATCHDOG"   // 宿主共生逃生/恢复
	CatEWMARoute    EventCategory = "EWMA_ROUTE"    // 策略组选路切线
	CatConfigLKG    EventCategory = "CONFIG_LKG"    // 配置快照晋升与回滚
	CatUpstream     EventCategory = "UPSTREAM_UPD"  // 四维探针与升级
)

// FlightEvent 单个黑匣子环形槽位记录（定长轻量结构，零磁盘 I/O）
type FlightEvent struct {
	Seq       uint64        `json:"seq"`
	Timestamp int64         `json:"ts_ms"`
	Category  EventCategory `json:"category"`
	Level     string        `json:"level"` // "INFO" | "WARN" | "CRIT"
	Summary   string        `json:"summary"`
	CostMs    int64         `json:"cost_ms,omitempty"`
}

const RingBufferCapacity = 256 // 恒定 256 个槽位，总内存占用 < 48KB

// FlightRecorder 零闪存磨损的内存环形黑匣子
type FlightRecorder struct {
	mu     sync.RWMutex
	ring   [RingBufferCapacity]FlightEvent
	head   int
	count  int
	seqGen atomic.Uint64
}

// GlobalFlightRecorder 全局单例黑匣子
var (
	globalRecorder     *FlightRecorder
	globalRecorderOnce sync.Once
)

func GetGlobalFlightRecorder() *FlightRecorder {
	globalRecorderOnce.Do(func() {
		globalRecorder = NewFlightRecorder()
	})
	return globalRecorder
}

func NewFlightRecorder() *FlightRecorder {
	return &FlightRecorder{}
}

// Record 写入一条关键状态跃迁记录（O(1) 复杂度，无堆内存逃逸扩张）
func (r *FlightRecorder) Record(cat EventCategory, level, summary string, costMs int64) {
	seq := r.seqGen.Add(1)
	ev := FlightEvent{
		Seq:       seq,
		Timestamp: time.Now().UnixMilli(),
		Category:  cat,
		Level:     level,
		Summary:   summary,
		CostMs:    costMs,
	}

	r.mu.Lock()
	r.ring[r.head] = ev
	r.head = (r.head + 1) % RingBufferCapacity
	if r.count < RingBufferCapacity {
		r.count++
	}
	r.mu.Unlock()
}

// Snapshot 按时间倒序导出最近 N 条黑匣子记录，供前端时间线或离线诊断包使用
func (r *FlightRecorder) Snapshot(limit int) []FlightEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 || limit > r.count {
		limit = r.count
	}
	out := make([]FlightEvent, 0, limit)
	for i := 0; i < limit; i++ {
		idx := (r.head - 1 - i + RingBufferCapacity) % RingBufferCapacity
		out = append(out, r.ring[idx])
	}
	return out
}

// SSEBroadcaster 单次序列化、多客户端零拷贝广播的 SSE 遥测流中心
type SSEBroadcaster struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
	closed  bool
}

var (
	globalBroadcaster     *SSEBroadcaster
	globalBroadcasterOnce sync.Once
)

func GetGlobalSSEBroadcaster() *SSEBroadcaster {
	globalBroadcasterOnce.Do(func() {
		globalBroadcaster = NewSSEBroadcaster()
	})
	return globalBroadcaster
}

func NewSSEBroadcaster() *SSEBroadcaster {
	return &SSEBroadcaster{
		clients: make(map[chan []byte]struct{}),
	}
}

// CloseAll 在守护进程收到 SIGTERM 时调用，立即断开所有 SSE 客户端流，防止 http.Server.Shutdown 挂死
func (b *SSEBroadcaster) CloseAll() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}
	b.closed = true
	for ch := range b.clients {
		close(ch)
		delete(b.clients, ch)
	}
}

// BroadcastPayload 将任意遥测快照执行【单次 JSON 序列化】，随后零拷贝分发给所有在线浏览器标签页
func (b *SSEBroadcaster) BroadcastPayload(eventName string, payload any) {
	b.mu.RLock()
	if b.closed || len(b.clients) == 0 {
		b.mu.RUnlock()
		return // 无浏览器打开 WebUI 或已停机时，0 序列化开销、0 CPU 消耗！
	}
	b.mu.RUnlock()

	rawJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}
	frame := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventName, rawJSON))

	b.mu.RLock()
	for ch := range b.clients {
		select {
		case ch <- frame:
		default:
			// 慢客户端非阻塞丢弃旧帧，绝不阻塞控制面主协程
		}
	}
	b.mu.RUnlock()
}

// ServeHTTP 挂载于 GET /api/v1/telemetry/stream
func (b *SSEBroadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := make(chan []byte, 4)
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		http.Error(w, "server shutting down", http.StatusServiceUnavailable)
		return
	}
	b.clients[ch] = struct{}{}
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		delete(b.clients, ch)
		b.mu.Unlock()
	}()

	// 首帧发送连接确认
	_, _ = w.Write([]byte("event: connected\ndata: {\"status\":\"ok\"}\n\n"))
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-ch:
			if !ok {
				return
			}
			if _, err := w.Write(frame); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// LKGConfigGuard 保障 ConfigIR 具备“最后已知良好（Last-Known-Good）”自动回滚自愈能力
type LKGConfigGuard struct {
	activePath string
	lkgPath    string
	recorder   *FlightRecorder
}

func NewLKGConfigGuard(activePath, lkgPath string, rec *FlightRecorder) *LKGConfigGuard {
	return &LKGConfigGuard{activePath: activePath, lkgPath: lkgPath, recorder: rec}
}

// PromoteAfterStable 当新配置加载且双核稳定运行 15s 后调用，将 active 备份晋升为 LKG
func (g *LKGConfigGuard) PromoteAfterStable(data []byte) error {
	tmp := g.lkgPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, g.lkgPath); err != nil {
		return err
	}
	if g.recorder != nil {
		g.recorder.Record(CatConfigLKG, "INFO", "Active ConfigIR promoted to Last-Known-Good (LKG) snapshot", 0)
	}
	return nil
}

// FallbackToLKG 当检测到 active 配置导致启动连续崩溃时调用，秒级回退至 LKG 快照
func (g *LKGConfigGuard) FallbackToLKG() ([]byte, error) {
	raw, err := os.ReadFile(g.lkgPath)
	if err != nil {
		return nil, fmt.Errorf("read LKG snapshot failed: %w", err)
	}
	_ = os.WriteFile(g.activePath, raw, 0644)
	if g.recorder != nil {
		g.recorder.Record(CatConfigLKG, "CRIT", "Corrupted active config detected! Auto-healed from LKG snapshot", 0)
	}
	return raw, nil
}
