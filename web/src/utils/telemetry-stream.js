// web/src/utils/telemetry-stream.js
// 使用单条 SSE 推送流取代高频 HTTP 轮询，并直接对接 v8.9 的 UIFrameScheduler
export class SymbioticStreamClient {
  constructor(frameScheduler, handlers = {}) {
    this.scheduler = frameScheduler
    this.handlers = handlers
    this.es = null
  }

  connect() {
    if (typeof window === 'undefined' || typeof EventSource === 'undefined') return
    if (this.es) this.es.close()
    this.es = new EventSource('/api/v1/telemetry/stream')

    // 接收后端单次序列化广播的实时遥测帧
    this.es.addEventListener('telemetry', (e) => {
      if (this.scheduler && this.scheduler.isSilenced()) return
      try {
        const data = JSON.parse(e.data)
        if (this.scheduler) {
          this.scheduler.enqueue('sse-telemetry', () => {
            if (this.handlers.onTelemetry) this.handlers.onTelemetry(data)
          })
        } else if (this.handlers.onTelemetry) {
          this.handlers.onTelemetry(data)
        }
      } catch (err) {
        console.warn('[SymbioticStreamClient] Parse error:', err)
      }
    })

    // 接收黑匣子关键状态跃迁推送（如 EWMA 发生切线或双擎热切完成）
    this.es.addEventListener('flight_event', (e) => {
      try {
        const ev = JSON.parse(e.data)
        if (this.scheduler) {
          this.scheduler.enqueue('sse-flight-log', () => {
            if (this.handlers.onFlightEvent) this.handlers.onFlightEvent(ev)
          })
        } else if (this.handlers.onFlightEvent) {
          this.handlers.onFlightEvent(ev)
        }
      } catch (err) {
        console.warn('[SymbioticStreamClient] Flight event parse error:', err)
      }
    })
  }

  close() {
    if (this.es) {
      this.es.close()
      this.es = null
    }
  }
}

// 注册 Ctrl+K / Cmd+K 疯狂动物城极客指令舱快捷键
export function bindGeekCommandPalette(toggleModalFn) {
  if (typeof window === 'undefined') return () => {}
  const handler = (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault()
      toggleModalFn()
    }
  }
  window.addEventListener('keydown', handler)
  return () => window.removeEventListener('keydown', handler)
}
