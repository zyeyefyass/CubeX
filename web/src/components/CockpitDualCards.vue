<template>
  <section class="grid grid-cols-1 lg:grid-cols-2 gap-4">
    <!-- 卡片 1: 一键连通性验收（三灯状态条 · 无代理验证） -->
    <div class="cbx-card p-5 sm:p-6 space-y-4 flex flex-col justify-between">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b pb-3.5" style="border-color: var(--cbx-border-soft);">
        <div class="space-y-1 min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <h3 class="text-sm sm:text-base font-extrabold cbx-text-main tracking-wide">一键连通性验收</h3>
            <span class="px-2 py-0.5 rounded-full text-[11px] font-bold cbx-badge-info border">
              无代理验证
            </span>
          </div>
          <p class="text-xs cbx-text-sub font-medium">
            本地直出三路探针并发验收：国内直连、国外加速与 Fake-IP DNS 映射
          </p>
        </div>

        <button
          type="button"
          @click="runThreeLightCheck"
          :disabled="checking"
          class="px-3.5 py-2 rounded-xl bg-sky-600 hover:bg-sky-700 text-white text-xs font-extrabold transition active:scale-95 shadow-sm cursor-pointer disabled:opacity-50 flex items-center gap-1.5 shrink-0"
        >
          <span v-if="checking" class="inline-block animate-spin">↻</span>
          <span v-else aria-hidden="true">🚦</span>
          <span>{{ checking ? '验收探测中...' : (hasChecked ? '重新验收' : '开始验收') }}</span>
        </button>
      </div>

      <!-- 三灯状态条 -->
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
        <div
          v-for="light in threeLights"
          :key="light.key"
          class="cbx-sub-card p-3 rounded-xl space-y-1.5 transition"
          :class="light.cardClass"
        >
          <div class="flex items-center justify-between gap-2">
            <span class="text-xs font-bold cbx-text-main flex items-center gap-1.5">
              <span
                class="w-2.5 h-2.5 rounded-full shrink-0"
                :class="light.dotClass"
                aria-hidden="true"
              ></span>
              <span>{{ light.label }}</span>
            </span>
            <span class="text-[11px] font-bold font-mono tabular-nums px-1.5 py-0.5 rounded" :class="light.badgeClass">
              <span aria-hidden="true">{{ light.icon }}</span>
              <span>{{ light.statusLabel }}</span>
            </span>
          </div>

          <div class="text-xs sm:text-sm font-black font-mono tabular-nums cbx-text-main truncate">
            {{ light.metricText }}
          </div>
          <div class="text-[11px] cbx-text-sub truncate" :title="light.detailText">
            {{ light.detailText }}
          </div>
        </div>
      </div>

      <!-- 双重代理防呆提示 -->
      <div
        v-if="threeLightResult.dual_proxy_hint"
        role="status"
        class="p-3 rounded-xl cbx-badge-warn border text-xs font-bold flex items-center gap-2"
      >
        <span aria-hidden="true" class="shrink-0">⚠️</span>
        <span>请确认电脑或手机已关闭 Clash/小火箭等客户端</span>
      </div>
      <div
        v-else-if="checkError"
        role="alert"
        class="p-3 rounded-xl cbx-badge-danger border text-xs font-semibold flex items-center gap-2"
      >
        <span aria-hidden="true" class="shrink-0">✕</span>
        <span>{{ checkError }}</span>
      </div>
      <div v-else class="text-[11px] cbx-text-sub flex items-center justify-between gap-2 font-mono">
        <span>验证模式：独立网关原生探针（不经过浏览器代理）</span>
        <span v-if="lastCheckedAt" class="tabular-nums cbx-text-muted">上次验收 {{ lastCheckedAt }}</span>
      </div>
    </div>

    <!-- 卡片 2: 宿主性能与速率（活跃出站吞吐 Sparkline） -->
    <div class="cbx-card p-5 sm:p-6 space-y-4 flex flex-col justify-between">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b pb-3.5" style="border-color: var(--cbx-border-soft);">
        <div class="space-y-1 min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <h3 class="text-sm sm:text-base font-extrabold cbx-text-main tracking-wide">宿主性能与速率</h3>
            <span class="px-2 py-0.5 rounded-full text-[11px] font-bold font-mono tabular-nums cbx-badge-ok border">
              {{ totalTrafficActive ? '实时吞吐活跃' : '空闲静默' }}
            </span>
          </div>
          <p class="text-xs cbx-text-sub font-medium">
            活跃出站节点实时上下行吞吐 · 后台标签页自动挂起省电
          </p>
        </div>

        <div class="flex items-center gap-2 text-xs font-mono tabular-nums shrink-0">
          <span class="cbx-sub-card px-2.5 py-1 rounded-lg font-bold cbx-text-main">
            CPU {{ hostCpu }}%
          </span>
          <span class="cbx-sub-card px-2.5 py-1 rounded-lg font-bold cbx-text-main">
            {{ hostMemoryFree }}
          </span>
          <span class="cbx-sub-card px-2.5 py-1 rounded-lg font-bold cbx-text-sub">
            {{ connectionsCount }} 连接
          </span>
        </div>
      </div>

      <!-- 活跃出站节点速率列表 -->
      <div v-if="activeRateRows.length > 0" class="space-y-2">
        <div
          v-for="row in activeRateRows"
          :key="row.node"
          class="cbx-sub-card px-3.5 py-2.5 rounded-xl flex items-center justify-between gap-3"
        >
          <div class="flex items-center gap-2 min-w-0 flex-1">
            <span
              class="w-2 h-2 rounded-full shrink-0"
              :class="row.hasTraffic ? 'bg-emerald-500 animate-pulse' : 'bg-slate-400'"
              aria-hidden="true"
            ></span>
            <span class="text-xs sm:text-[13px] font-bold cbx-text-main truncate" :title="row.node">
              {{ row.node }}
            </span>
          </div>

          <div v-if="row.hasTraffic" class="flex items-center gap-3 shrink-0">
            <svg viewBox="0 0 64 18" class="w-16 h-[18px] shrink-0 overflow-visible" aria-hidden="true">
              <path
                v-if="row.fillPath"
                :d="row.fillPath"
                fill="currentColor"
                class="text-sky-500/15"
              />
              <polyline
                fill="none"
                stroke="currentColor"
                stroke-width="1.6"
                stroke-linecap="round"
                stroke-linejoin="round"
                class="text-sky-500"
                :points="row.sparkPoints"
              />
            </svg>
            <span class="text-xs font-mono font-bold tabular-nums cbx-text-main whitespace-nowrap">
              <span class="text-emerald-600 dark:text-emerald-400">↓ {{ row.downText }}</span>
              <span class="mx-1.5 cbx-text-muted">/</span>
              <span class="text-sky-600 dark:text-cyan-300">↑ {{ row.upText }}</span>
            </span>
          </div>

          <div v-else class="flex items-center gap-2 shrink-0 text-xs font-mono tabular-nums cbx-text-muted">
            <span>静默</span>
            <span>↓ 0.0 MB/s · ↑ 0.0 MB/s</span>
          </div>
        </div>
      </div>

      <!-- 空闲静默态 -->
      <div v-else class="cbx-sub-card p-4 rounded-xl flex items-center justify-between gap-3">
        <div class="flex items-center gap-2.5 min-w-0">
          <span class="w-2 h-2 rounded-full bg-slate-400 shrink-0" aria-hidden="true"></span>
          <div class="min-w-0">
            <div class="text-xs font-bold cbx-text-main truncate">
              当前主线：{{ fallbackNodeName }}
            </div>
            <div class="text-[11px] cbx-text-sub">暂无活跃出站数据流，已优雅退回静默低功耗态</div>
          </div>
        </div>
        <span class="text-xs font-mono tabular-nums cbx-text-muted shrink-0">
          ↓ 0.0 MB/s  ↑ 0.0 MB/s
        </span>
      </div>

      <div class="text-[11px] cbx-text-sub flex items-center justify-between gap-2 font-mono tabular-nums">
        <span>总内存 {{ hostMemoryTotal }} · 引擎 {{ activeEngineLabel }}</span>
        <span>{{ activeRateRows.length }} 个活跃出站</span>
      </div>
    </div>
  </section>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { apiFetch } from '../utils/apiFetch.js'

const props = defineProps({
  activeTab: { type: String, default: 'overview' },
  needsAuthUnlock: { type: Boolean, default: false },
  activeEngine: { type: String, default: 'singbox' },
  hostCpu: { type: [Number, String], default: 0 },
  hostMemoryFree: { type: String, default: '充足' },
  hostMemoryTotal: { type: String, default: '--' },
  connectionsCount: { type: [Number, String], default: 0 },
  fallbackNodeName: { type: String, default: '智能自动优选' }
})

const checking = ref(false)
const hasChecked = ref(false)
const checkError = ref('')
const lastCheckedAt = ref('')
const threeLightResult = ref({
  domestic: null,
  proxy: null,
  fakeip_dns: null,
  dual_proxy_hint: false
})

const activeNodeRates = ref({})
const rateHistoryMap = ref({})
let ratesTimer = null

const activeEngineLabel = computed(() =>
  props.activeEngine === 'mihomo' ? 'Mihomo' : 'sing-box'
)

function formatRateMBps(bps) {
  const num = Number(bps) || 0
  if (num <= 0) return '0.0 MB/s'
  if (num < 1024 * 100) {
    return `${(num / 1024).toFixed(1)} KB/s`
  }
  return `${(num / (1024 * 1024)).toFixed(1)} MB/s`
}

function buildSparklineGeometry(samples) {
  const list = Array.isArray(samples) && samples.length > 0 ? samples : [0, 0]
  const padded = list.length === 1 ? [0, list[0]] : list
  const maxVal = Math.max(...padded, 1)
  const w = 64
  const h = 16
  const coords = padded.map((v, idx) => {
    const x = padded.length > 1 ? (idx / (padded.length - 1)) * w : w
    const y = h - Math.max(1, (v / maxVal) * (h - 2))
    return [Number(x.toFixed(1)), Number(y.toFixed(1))]
  })
  const points = coords.map(([x, y]) => `${x},${y}`).join(' ')
  const firstX = coords[0][0]
  const lastX = coords[coords.length - 1][0]
  const fillPath = `M ${firstX},${h} L ${coords.map(([x, y]) => `${x},${y}`).join(' L ')} L ${lastX},${h} Z`
  return { points, fillPath }
}

const totalTrafficActive = computed(() => {
  const rates = activeNodeRates.value || {}
  return Object.values(rates).some(r => (Number(r?.up_bps) || 0) > 0 || (Number(r?.down_bps) || 0) > 0)
})

const activeRateRows = computed(() => {
  const rates = activeNodeRates.value || {}
  const entries = Object.entries(rates)
  if (entries.length === 0) return []
  entries.sort((a, b) => {
    const ta = (Number(a[1]?.down_bps) || 0) + (Number(a[1]?.up_bps) || 0)
    const tb = (Number(b[1]?.down_bps) || 0) + (Number(b[1]?.up_bps) || 0)
    return tb - ta
  })
  return entries.slice(0, 4).map(([node, stat]) => {
    const up = Number(stat?.up_bps) || 0
    const down = Number(stat?.down_bps) || 0
    const hasTraffic = up > 0 || down > 0
    const samples = rateHistoryMap.value[node] || [down + up]
    const geom = buildSparklineGeometry(samples)
    return {
      node,
      upText: formatRateMBps(up),
      downText: formatRateMBps(down),
      hasTraffic,
      sparkPoints: geom.points,
      fillPath: geom.fillPath
    }
  })
})

function buildLightState(key, label, item, isDns = false) {
  if (!hasChecked.value || !item) {
    return {
      key,
      label,
      icon: '○',
      statusLabel: '待验收',
      metricText: isDns ? '—' : '-- ms',
      detailText: '点击右上角按钮开始验收',
      dotClass: 'bg-slate-400',
      badgeClass: 'cbx-btn-ghost',
      cardClass: ''
    }
  }
  const ok = Boolean(item.ok)
  return {
    key,
    label,
    icon: ok ? '✓' : '✕',
    statusLabel: ok ? '畅通' : '异常',
    metricText: isDns
      ? (item.resolved_ip || (ok ? '198.18.x.x' : '未命中'))
      : (item.latency_ms > 0 ? `${item.latency_ms} ms` : (ok ? '正常' : '超时')),
    detailText: item.detail || (ok ? '链路连通正常' : '探测失败'),
    dotClass: ok ? 'bg-emerald-500' : 'bg-rose-500',
    badgeClass: ok ? 'cbx-badge-ok border' : 'cbx-badge-danger border',
    cardClass: ok ? 'border-emerald-500/30' : 'border-rose-500/40'
  }
}

const threeLights = computed(() => [
  buildLightState('domestic', '国内', threeLightResult.value.domestic, false),
  buildLightState('proxy', '国外', threeLightResult.value.proxy, false),
  buildLightState('fakeip_dns', 'DNS', threeLightResult.value.fakeip_dns, true)
])

async function runThreeLightCheck() {
  if (checking.value) return
  checking.value = true
  checkError.value = ''
  try {
    const res = await apiFetch('/api/v1/diagnostics/three-light-check', {
      method: 'POST',
      timeoutMs: 12000
    })
    if (!res.ok) {
      throw new Error(`验收请求失败 (${res.status})`)
    }
    const data = await res.json()
    threeLightResult.value = {
      domestic: data?.domestic || null,
      proxy: data?.proxy || null,
      fakeip_dns: data?.fakeip_dns || null,
      dual_proxy_hint: Boolean(data?.dual_proxy_hint)
    }
    hasChecked.value = true
    lastCheckedAt.value = new Date().toLocaleTimeString()
  } catch (err) {
    checkError.value = err?.message || '三灯连通性验收请求异常'
  } finally {
    checking.value = false
  }
}

async function fetchActiveRates() {
  if (props.needsAuthUnlock || props.activeTab !== 'overview') return
  if (typeof document !== 'undefined' && document.visibilityState === 'hidden') return
  try {
    const res = await apiFetch('/api/v1/proxy/active-rates', {
      skipWhenHidden: true,
      timeoutMs: 4000
    })
    if (!res.ok) return
    const data = await res.json()
    const rates = (data && typeof data.active_node_rates === 'object' && data.active_node_rates) || {}
    activeNodeRates.value = rates
    const nextHistory = { ...rateHistoryMap.value }
    for (const [node, stat] of Object.entries(rates)) {
      const total = (Number(stat?.down_bps) || 0) + (Number(stat?.up_bps) || 0)
      const prev = Array.isArray(nextHistory[node]) ? nextHistory[node] : []
      nextHistory[node] = [...prev, total].slice(-16)
    }
    rateHistoryMap.value = nextHistory
  } catch (e) {
    // 静默容错，避免轮询瞬断打扰用户
  }
}

function startPolling() {
  stopPolling()
  if (props.needsAuthUnlock || props.activeTab !== 'overview') return
  if (typeof document !== 'undefined' && document.visibilityState === 'hidden') return
  fetchActiveRates()
  ratesTimer = setInterval(fetchActiveRates, 2500)
}

function stopPolling() {
  if (ratesTimer) {
    clearInterval(ratesTimer)
    ratesTimer = null
  }
}

function onVisibilityChange() {
  if (typeof document === 'undefined') return
  if (document.visibilityState === 'hidden') {
    stopPolling()
  } else {
    startPolling()
  }
}

watch(
  () => [props.activeTab, props.needsAuthUnlock],
  ([tab, locked]) => {
    if (tab === 'overview' && !locked) {
      startPolling()
    } else {
      stopPolling()
    }
  }
)

onMounted(() => {
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', onVisibilityChange)
  }
  startPolling()
})

onUnmounted(() => {
  stopPolling()
  if (typeof document !== 'undefined') {
    document.removeEventListener('visibilitychange', onVisibilityChange)
  }
})
</script>
