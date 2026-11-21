// web/src/utils/scheduler.js
/**
 * UIFrameScheduler: 120FPS 零卡顿渲染调度器
 * 采用 requestAnimationFrame 聚合高频状态与波形更新，后台标签页自动休眠，切回前台单次合并渲染。
 * 融合双擎切核在途请求截断 (In-flight Abort) 与毫秒级提前唤醒刷新机制。
 */
export class UIFrameScheduler {
  constructor() {
    this.pendingTasks = new Map()
    this.rafId = null
    this.isPageVisible = typeof document !== 'undefined' ? !document.hidden : true
    this.silencedUntil = 0
    this.inFlightControllers = new Set()

    this.onWakeupCallbacks = new Set()
    this.isHibernated = false

    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', () => {
        this.isPageVisible = !document.hidden
        if (!this.isPageVisible) {
          // 页面切入后台：彻底冻结生命周期，清除积压帧任务，防止后台积压与切回雪崩
          this.isHibernated = true
          this.pendingTasks.clear()
          if (this.rafId !== null) {
            cancelAnimationFrame(this.rafId)
            this.rafId = null
          }
        } else {
          // 页面切回前台：解除冻结，在单帧内执行合并唤醒快照刷新
          this.isHibernated = false
          if (this.pendingTasks.size > 0) {
            this.scheduleFlush()
          }
          // 执行所有注册的唤醒快照对齐回调 (Single Coalesced Snapshot Refresh)
          for (const cb of this.onWakeupCallbacks) {
            try {
              cb()
            } catch (err) {
              console.warn('[UIFrameScheduler] onWakeup error:', err)
            }
          }
        }
      })
    }
  }

  // 注册前台唤醒快照对齐回调，返回注销函数
  onWakeup(cb) {
    if (typeof cb === 'function') {
      this.onWakeupCallbacks.add(cb)
      return () => this.onWakeupCallbacks.delete(cb)
    }
    return () => {}
  }

  // 注册受管遥测请求的 AbortController，便于切核瞬间一键静默截断在途请求
  trackFetchController() {
    const controller = new AbortController()
    this.inFlightControllers.add(controller)
    const cleanup = () => this.inFlightControllers.delete(controller)
    controller.signal.addEventListener('abort', cleanup, { once: true })
    return { signal: controller.signal, done: cleanup }
  }

  // 开启双擎切核静默保护窗：1. 立即静默取消所有在途遥测请求 2. 挂起新轮询最多 durationMs
  enterSwitchSilence(durationMs = 3000) {
    this.silencedUntil = performance.now() + durationMs
    for (const ctrl of this.inFlightControllers) {
      ctrl.abort('engine-switching-silence')
    }
    this.inFlightControllers.clear()
  }

  // 切核 API 返回 SwitchResult 后立即调用：提前结束 3s 静默窗，实现毫秒级无缝看板刷新
  exitSwitchSilenceEarly(immediateRefreshFn) {
    this.silencedUntil = 0
    if (typeof immediateRefreshFn === 'function') {
      this.enqueue('post-switch-refresh', immediateRefreshFn)
    }
  }

  isSilenced() {
    return performance.now() < this.silencedUntil
  }

  // 提交单帧合并渲染任务
  enqueue(taskKey, renderFn) {
    this.pendingTasks.set(taskKey, renderFn)
    if (!this.isPageVisible) return
    this.scheduleFlush()
  }

  // 兼容旧版 schedule 调用
  schedule(key, fn) {
    this.enqueue(key, fn)
  }

  // 组件卸载时清理未执行的帧任务，防止闭包内存泄漏
  cancel(taskKey) {
    this.pendingTasks.delete(taskKey)
  }

  scheduleFlush() {
    if (this.rafId !== null) return
    this.rafId = requestAnimationFrame(() => {
      this.rafId = null
      const tasks = Array.from(this.pendingTasks.values())
      this.pendingTasks.clear()
      for (let i = 0; i < tasks.length; i++) {
        try {
          tasks[i]()
        } catch (err) {
          console.warn('[UIFrameScheduler] Task failed:', err)
        }
      }
    })
  }
}

export const frameScheduler = new UIFrameScheduler()

// 一键赏景模式 (Peek Mode) 控制器：支持快捷键 (Alt+Z / ESC) 与点击图标双向来回切换
export function enablePeekModeWithGuard(onExitCallback) {
  if (typeof document === 'undefined') return () => {}
  document.body.classList.add('wallpaper-peek-mode')
  const enteredAt = performance.now()

  const cleanupAndExit = () => {
    document.body.classList.remove('wallpaper-peek-mode')
    window.removeEventListener('keydown', onKeyDown, true)
    window.removeEventListener('mousedown', onPointerDown, true)
    if (typeof onExitCallback === 'function') onExitCallback()
  }

  const onKeyDown = (e) => {
    if (e.key === 'Escape' || (e.altKey && (e.key === 'z' || e.key === 'Z'))) {
      cleanupAndExit()
    }
  }

  const onPointerDown = () => {
    if (performance.now() - enteredAt < 250) return // 过滤点击按钮本身的冒泡残余
    cleanupAndExit()
  }

  window.addEventListener('keydown', onKeyDown, true)
  window.addEventListener('mousedown', onPointerDown, true)
  return cleanupAndExit
}
