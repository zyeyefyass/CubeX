package lifecycle

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zyeyefyass/CubeX/internal/dataplane"
	"github.com/zyeyefyass/CubeX/internal/telemetry"
)

// CoreApplyFunc 抽象加载指定配置并拉起双擎内核的回调
type CoreApplyFunc func(ctx context.Context, rawConfigIR []byte) error

// LTSDaemonSupervisor 统筹 v9.0 LTS 启动自愈、LKG 代数晋升、内核快转降级与零残留停机
type LTSDaemonSupervisor struct {
	mu          sync.Mutex
	configGen   atomic.Uint64
	lkgTimer    *time.Timer
	lkgWindow   time.Duration
	lkgGuard    *telemetry.LKGConfigGuard
	recorder    *telemetry.FlightRecorder
	broadcaster *telemetry.SSEBroadcaster
	nftActive   atomic.Bool
}

func NewLTSDaemonSupervisor(
	lkgGuard *telemetry.LKGConfigGuard,
	recorder *telemetry.FlightRecorder,
	broadcaster *telemetry.SSEBroadcaster,
	lkgPromotionWindow time.Duration,
) *LTSDaemonSupervisor {
	if lkgPromotionWindow <= 0 {
		lkgPromotionWindow = 15 * time.Second
	}
	return &LTSDaemonSupervisor{
		lkgGuard:    lkgGuard,
		recorder:    recorder,
		broadcaster: broadcaster,
		lkgWindow:   lkgPromotionWindow,
	}
}

// BootOrReloadConfig 执行带 LKG 自动回退与代数防抖晋升的配置加载
func (s *LTSDaemonSupervisor) BootOrReloadConfig(
	ctx context.Context,
	activeIRBytes []byte,
	applyFn CoreApplyFunc,
) error {
	// 递增配置代数并取消上一轮尚未到期的 LKG 晋升定时器，防止 ABA 误晋升
	gen := s.configGen.Add(1)
	s.cancelPendingLKGTimer()

	if err := applyFn(ctx, activeIRBytes); err != nil {
		if s.recorder != nil {
			s.recorder.Record(telemetry.CatConfigLKG, "WARN",
				fmt.Sprintf("Active config (gen=%d) failed to start: %v, attempting LKG fallback", gen, err), 0)
		}

		// 启动失败：立即从 LKG 快照回退自愈
		if s.lkgGuard == nil {
			return err
		}
		lkgBytes, lkgErr := s.lkgGuard.FallbackToLKG()
		if lkgErr != nil {
			return fmt.Errorf("active config failed (%v) and LKG fallback unavailable (%w)", err, lkgErr)
		}
		if retryErr := applyFn(ctx, lkgBytes); retryErr != nil {
			return fmt.Errorf("both active and LKG config failed to boot: %w", retryErr)
		}
		return nil
	}

	// 启动成功：开启严格绑定当前代数 gen 的 15s 稳态观察定时器
	s.scheduleLKGPromotion(gen, activeIRBytes)
	return nil
}

func (s *LTSDaemonSupervisor) scheduleLKGPromotion(expectedGen uint64, snapshot []byte) {
	if s.lkgGuard == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lkgTimer != nil {
		s.lkgTimer.Stop()
	}

	cp := make([]byte, len(snapshot))
	copy(cp, snapshot)

	s.lkgTimer = time.AfterFunc(s.lkgWindow, func() {
		// 仅当 15 秒内未发生任何新配置变更时，才将当前配置晋升为 LKG
		if s.configGen.Load() != expectedGen {
			return
		}
		_ = s.lkgGuard.PromoteAfterStable(cp)
	})
}

func (s *LTSDaemonSupervisor) cancelPendingLKGTimer() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lkgTimer != nil {
		s.lkgTimer.Stop()
		s.lkgTimer = nil
	}
}

// EnableKernelFastPathSafely 尝试启用 nftables 内核态中国 IP 零绕路直通；若容器缺乏特权则平滑降级
func (s *LTSDaemonSupervisor) EnableKernelFastPathSafely(ctx context.Context, spec dataplane.FastPathSpec) {
	start := time.Now()
	if err := dataplane.ApplyAtomicNftRules(ctx, spec); err != nil {
		s.nftActive.Store(false)
		if s.recorder != nil {
			s.recorder.Record(telemetry.CatEngineSwitch, "WARN",
				"Kernel nftables fastpath unavailable, degraded to user-space routing: "+err.Error(),
				time.Since(start).Milliseconds())
		}
		return
	}

	s.nftActive.Store(true)
	if s.recorder != nil {
		s.recorder.Record(telemetry.CatEngineSwitch, "INFO",
			"Kernel nftables fastpath & DNS leak guard activated",
			time.Since(start).Milliseconds())
	}
}

// DeterministicShutdown 执行严格的 5 步无阻塞停机序列（保证 3 秒内 100% 干净退出）
func (s *LTSDaemonSupervisor) DeterministicShutdown(
	ctx context.Context,
	drainHAFn func(context.Context) error,
	stopEngineFn func() error,
	stopHTTPServerFn func(context.Context) error,
) {
	// Step 1: 取消未决的 LKG 晋升定时器
	s.cancelPendingLKGTimer()

	// Step 2: 立即关闭所有 SSE 遥测长连接，解除 http.Server.Shutdown 的阻塞锁
	if s.broadcaster != nil {
		s.broadcaster.CloseAll()
	}

	// Step 3: 触发宿主共生旁路保护并卸载内核 nftables 表，确保宿主机零残留
	if drainHAFn != nil {
		_ = drainHAFn(ctx)
	}
	if s.nftActive.Swap(false) {
		_ = dataplane.TeardownAtomicNftRules(ctx)
	}

	// Step 4: 停止双擎子进程组 (SIGTERM -> SIGKILL)
	if stopEngineFn != nil {
		_ = stopEngineFn()
	}

	// Step 5: 优雅关闭控制面 HTTP Server 并归还物理内存
	if stopHTTPServerFn != nil {
		_ = stopHTTPServerFn(ctx)
	}
	runtime.GC()
	debug.FreeOSMemory()
}
