// web/src/composables/useDataplane.js
// CubeX 数据面健康契约与大白话诊断中枢
import { ref, shallowRef, computed, onMounted, onUnmounted } from 'vue'
import { copyTextToClipboard } from '../utils/clipboard'
import { apiFetch } from '../utils/apiFetch'

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

  const isFetching = ref(false)
  const lastError = ref(null)
  let pollTimer = null

  // 大白话三态胶囊计算 (低饱和莫兰迪色，去霓虹光晕，适配雾面奶白底盘)
  const pulseState = computed(() => {
    const s = snapshot.value
    if (s.state === 'switching') {
      return {
        type: 'switching',
        label: '无感切核中',
        dotClass: 'bg-[#0284c7] animate-pulse',
        badgeClass: 'bg-[#f1f4f9]/90 text-[#0369a1] border-sky-300/60',
        textClass: 'text-[#0369a1]'
      }
    }
    if (!s.healthy || s.state === 'degraded' || s.status === 'degraded') {
      return {
        type: 'degraded',
        label: '已保底逃生',
        dotClass: 'bg-[#d97706] animate-ping',
        badgeClass: 'bg-[#f1f4f9]/90 text-[#b45309] border-amber-300/60',
        textClass: 'text-[#b45309]'
      }
    }
    return {
      type: 'healthy',
      label: '数据面健康',
      dotClass: 'bg-[#0f766e]',
      badgeClass: 'bg-[#f1f4f9]/90 text-[#0f766e] border-teal-300/60',
      textClass: 'text-[#0f766e]'
    }
  })

  // 毫秒级防抖探针拉取
  const fetchDataplaneHealth = async () => {
    if (typeof window === 'undefined' || isFetching.value) return
    isFetching.value = true
    try {
      const res = await apiFetch('/healthz/dataplane', {
        headers: { 'Accept': 'application/json' },
        cache: 'no-store',
        skipWhenHidden: true,
        timeoutMs: 4000
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
    return await copyTextToClipboard(text)
  }

  return {
    snapshot,
    pulseState,
    isFetching,
    fetchDataplaneHealth,
    copyDesensitizedReport
  }
}
