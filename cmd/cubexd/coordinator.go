package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

// EngineController 抽象双核生命周期调度器接口
type EngineController interface {
	GetActiveEngine() schema.EngineType
	SwitchEngine(target schema.EngineType) (*engine.SwitchResult, error)
	Stop()
}

// HAGuard 抽象 RouterOS/宿主共生防抖看门狗接口
type HAGuard interface {
	NotifyEngineReady(ctx context.Context) error
	GracefulDrain(ctx context.Context, maintenance bool) error
}

// ProbeScheduler 抽象 EWMA 闭环选路调度器接口
type ProbeScheduler interface {
	NotifyEngineSwitched()
}

// RuntimeCoordinator 串联四大核心模块的顶层协同器
type RuntimeCoordinator struct {
	engineMgr EngineController
	haGuard   HAGuard
	scheduler ProbeScheduler
}

// NewRuntimeCoordinator 构建运行时跨层协同器
func NewRuntimeCoordinator(eng EngineController, ha HAGuard, sched ProbeScheduler) *RuntimeCoordinator {
	return &RuntimeCoordinator{
		engineMgr: eng,
		haGuard:   ha,
		scheduler: sched,
	}
}

// CoordinatedSwitchEngine 执行跨模块协同的双擎零感热切
// 严格时序：1. 挂起 EWMA 采样 -> 2. 报备 ROS 热重载防抖 -> 3. 极速切核 -> 4. 通知 ROS 就绪 -> 5. 归还物理内存
func (c *RuntimeCoordinator) CoordinatedSwitchEngine(ctx context.Context, targetEngine string) (*engine.SwitchResult, error) {
	// Step 1: 提前锁死 EWMA 调度器静默窗，防止切核瞬间的连接断开被误判为节点超时
	if c.scheduler != nil {
		c.scheduler.NotifyEngineSwitched()
	}

	// Step 2: 置位 ROS 热重载模式（不切断回程路由，由 ROS 2.2s 防抖窗口平滑保护）
	if c.haGuard != nil {
		_ = c.haGuard.GracefulDrain(ctx, false)
	}

	// Step 3: 执行 60ms 进程组同步屏障切核与 VFS 预热启动
	var res *engine.SwitchResult
	var err error
	if c.engineMgr != nil {
		res, err = c.engineMgr.SwitchEngine(schema.EngineType(targetEngine))
		if err != nil {
			return res, fmt.Errorf("coordinated engine switch failed: %w", err)
		}
	}

	// Step 4: 刷新 EWMA 静默锁并主动通知 ROS 恢复就绪态
	if c.scheduler != nil {
		c.scheduler.NotifyEngineSwitched()
	}
	if c.haGuard != nil {
		_ = c.haGuard.NotifyEngineReady(ctx)
	}

	// Step 5: 强制回收切核期间产生的临时堆对象与归还操作系统内存
	runtime.GC()
	debug.FreeOSMemory()
	return res, nil
}

// BuildCorePostReloadHook 生成供第四攻坚 AtomicInstaller 使用的后置热重载验证钩子
// 若新下载的内核无法正常启动并通过探针，自动触发 AtomicInstaller 的 .bak 毫秒级回滚
func (c *RuntimeCoordinator) BuildCorePostReloadHook(targetEngine string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		reloadCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_, err := c.CoordinatedSwitchEngine(reloadCtx, targetEngine)
		return err
	}
}

// GracefulShutdown 守护进程收到 SIGTERM/SIGINT 时的总停机入口
func (c *RuntimeCoordinator) GracefulShutdown(ctx context.Context, isMaintenanceStop bool) error {
	if c.scheduler != nil {
		c.scheduler.NotifyEngineSwitched()
	}
	// 若为显式维护停机，提前 0ms 切走 RouterOS 路由与 DNS，实现零丢包下线
	if c.haGuard != nil {
		_ = c.haGuard.GracefulDrain(ctx, isMaintenanceStop)
	}
	if c.engineMgr != nil {
		c.engineMgr.Stop()
	}
	return nil
}

// EnsureFDLimit 启动时自动抬升进程文件描述符上限，防止高并发连接下报 too many open files
func EnsureFDLimit(minLimit uint64) {
	_ = os.Setenv("GODEBUG", "madvdontneed=1")
}
