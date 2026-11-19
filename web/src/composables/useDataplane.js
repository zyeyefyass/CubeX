// web/src/composables/useDataplane.js
// CubeX v9.9.4 数据面健康契约与大白话诊断中枢
import { ref, shallowRef, computed, onMounted, onUnmounted } from 'vue'

export function useDataplane() {
  const snapshot = shallowRef({
    status: 'ok',
    state: 'healthy',
    healthy: true,
    active_engine: 'sing-box',
    clash_api: true,
    dns_service: true,
    fail_streak: 0,
    success_streak: 25,
    probe_duration_ms: 1,
    checked_at: new Date().toISOString()
  })

  const isDiagDrawerOpen = ref(false)
  const isFetching = ref(false)
  const lastError = ref(null)
  let pollTimer = null

  // 大白话三态胶囊计算
  const pulseState = computed(() => {
    const s = snapshot.value
    if (s.state === 'switching') {
      return {
        type: 'switching',
        label: '无感切核中',
        dotClass: 'bg-cyan-400 animate-pulse shadow-[0_0_8px_#00d2ff]',
        badgeClass: 'bg-cyan-950/40 text-cyan-300 border-cyan-500/30 hover:border-cyan-400/50',
        textClass: 'text-cyan-300'
      }
    }
    if (!s.healthy || s.state === 'degraded' || s.status === 'degraded') {
      return {
        type: 'degraded',
        label: '已保底逃生',
        dotClass: 'bg-amber-400 animate-ping shadow-[0_0_8px_#f59e0b]',
        badgeClass: 'bg-amber-950/40 text-amber-300 border-amber-500/30 hover:border-amber-400/50',
        textClass: 'text-amber-300'
      }
    }
    return {
      type: 'healthy',
      label: '数据面健康',
      dotClass: 'bg-emerald-400 shadow-[0_0_8px_#10b981]',
      badgeClass: 'bg-emerald-950/40 text-emerald-300 border-emerald-500/30 hover:border-emerald-400/50',
      textClass: 'text-emerald-300'
    }
  })

  // 毫秒级防抖探针拉取
  const fetchDataplaneHealth = async () => {
    if (typeof window === 'undefined' || isFetching.value) return
    isFetching.value = true
    try {
      const res = await fetch('/healthz/dataplane', {
        headers: { 'Accept': 'application/json' },
        cache: 'no-store'
      })
      if (res.ok || res.status === 503) {
        const data = await res.json()
        snapshot.value = Object.freeze(data)
        lastError.value = null
      }
    } catch (err) {
      lastError.value = err.message
    } finally {
      isFetching.value = false
    }
  }

  // 页面可见性感知与省电节流轮询
  const handleVisibilityChange = () => {
    if (document.visibilityState === 'hidden') {
      clearInterval(pollTimer)
      pollTimer = setInterval(fetchDataplaneHealth, 15000)
    } else if (document.visibilityState === 'visible') {
      clearInterval(pollTimer)
      // 切回前台即时唤醒，杜绝 10 多秒陈旧状态
      fetchDataplaneHealth()
      pollTimer = setInterval(fetchDataplaneHealth, 3000)
    }
  }

  onMounted(() => {
    fetchDataplaneHealth()
    pollTimer = setInterval(fetchDataplaneHealth, 3000)
    document.addEventListener('visibilitychange', handleVisibilityChange)
  })

  onUnmounted(() => {
    clearInterval(pollTimer)
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  })

  // 一键复制脱敏诊断卡
  const copyDesensitizedReport = async () => {
    const s = snapshot.value
    const report = {
      timestamp: new Date().toISOString(),
      dataplane_state: s.state,
      healthy: s.healthy,
      active_engine: s.active_engine,
      clash_api_liveness: s.clash_api,
      dns_liveness: s.dns_service,
      fail_streak: s.fail_streak,
      success_streak: s.success_streak,
      probe_duration_ms: s.probe_duration_ms,
      diagnosis: s.healthy ? '全链路健康，Fake-IP 与网关转发处于最优稳态' : '触发无损降级逃生网，建议检查双核配置或上游链路'
    }
    const text = JSON.stringify(report, null, 2)
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      return false
    }
  }

  return {
    snapshot,
    pulseState,
    isDiagDrawerOpen,
    isFetching,
    fetchDataplaneHealth,
    copyDesensitizedReport
  }
}
