<template>
  <dialog
    ref="dialogRef"
    closedby="any"
    aria-labelledby="ip-perspective-title"
    class="modern-dialog select-none p-3 sm:p-5 w-full max-w-2xl"
    @click="handleBackdropClick"
    @cancel.prevent="close"
  >
    <div class="cbx-modal-shell rounded-3xl w-full flex flex-col overflow-hidden relative" @click.stop>
      <!-- Modal Header -->
      <div class="cbx-modal-header px-4 sm:px-5 py-3.5 sm:py-4 flex items-center justify-between gap-3 select-none">
        <div class="flex items-center gap-2.5 min-w-0">
          <div class="w-9 h-9 rounded-xl cbx-badge-info flex items-center justify-center font-bold shrink-0">
            🔍
          </div>
          <div class="min-w-0">
            <div class="flex items-center gap-2 flex-wrap">
              <h3 id="ip-perspective-title" class="text-sm sm:text-base font-extrabold cbx-text-main tracking-wide truncate max-w-[200px] sm:max-w-[340px]">
                {{ nodeName || '节点透视' }}
              </h3>
              <span
                class="text-[11px] font-mono px-2 py-0.5 rounded-full font-bold font-tabular shrink-0"
                :class="overallBadgeClass"
              >
                {{ overallScore >= 90 ? 'S 极致稳定' : (overallScore >= 80 ? 'A+ 极高品质' : (overallScore >= 60 ? 'A 优质可用' : 'C 波动预警')) }}
              </span>
            </div>
            <p class="text-[11px] cbx-text-muted font-mono mt-0.5 truncate">
              稳定三角评分 (40% 延时中位数 × 30% 抖动平均 × 30% 被动吞吐)
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2 shrink-0">
          <button
            @click="triggerBurst"
            :disabled="testingBurst"
            class="min-h-[36px] sm:min-h-0 px-3 py-1.5 rounded-xl text-xs font-bold cbx-badge-warn flex items-center gap-1.5 transition active:scale-95 cursor-pointer font-tabular disabled:opacity-50"
            title="单次 ≤2MB 闪电突发测速，零额外流量损耗"
          >
            <span v-if="testingBurst" class="inline-block animate-spin">↻</span>
            <span v-else>⚡</span>
            <span>{{ testingBurst ? '测速中...' : '闪电测速' }}</span>
          </button>
          <button
            @click="close"
            aria-label="关闭节点详情"
            class="min-w-[36px] min-h-[36px] px-2.5 py-1.5 rounded-xl cbx-btn-ghost text-xs font-bold transition cursor-pointer flex items-center justify-center cbx-touch-extend"
          >
            ✕
          </button>
        </div>
      </div>

      <!-- Modal Body -->
      <div class="p-4 sm:p-5 space-y-4 max-h-[78dvh] overflow-y-auto">
        <div v-if="loading && !perspectiveData" class="py-10 text-center text-xs cbx-text-muted font-mono">
          正在探测节点实时透视指标...
        </div>

        <div v-else-if="loadError && !perspectiveData" class="p-4 rounded-2xl cbx-badge-danger text-xs font-mono flex items-center justify-between gap-3">
          <span>⚠️ {{ loadError }}</span>
          <button @click="fetchPerspective(nodeName, true)" class="px-3 py-1 rounded-lg cbx-btn-ghost text-xs font-bold cursor-pointer">
            重试
          </button>
        </div>

        <template v-else>
          <!-- Top Score Banner: 单色高对比卡片，无脏渐变与多色混排 -->
          <div class="p-4 rounded-2xl cbx-sub-card flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <div class="text-[11px] font-mono cbx-text-muted font-semibold">稳定三角综合得分</div>
              <div class="text-3xl font-black font-mono tracking-tight font-tabular mt-0.5" :style="{ color: overallScoreColor }">
                {{ Math.round(overallScore) }}
                <span class="text-sm font-normal cbx-text-muted">/ 100</span>
              </div>
            </div>
            <div class="sm:text-right text-xs font-mono space-y-1 font-tabular">
              <div class="cbx-text-sub">延时中位数: <span class="font-bold cbx-text-main">{{ formatMetric(q1.latency_median_ms, 'ms') }}</span></div>
              <div class="cbx-text-sub">相邻抖动差: <span class="font-bold cbx-text-main">{{ formatMetric(q1.jitter_avg_ms, 'ms') }}</span></div>
              <div class="cbx-text-sub">峰值吞吐量: <span class="font-bold cbx-text-main">{{ q2.bandwidth_mbps ? q2.bandwidth_mbps.toFixed(1) + ' Mbps' : '75分中性基准' }}</span></div>
            </div>
          </div>

          <!-- 4-Quadrant Grid -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
            <!-- Quadrant 1: Latency & Jitter -->
            <div class="p-4 rounded-2xl cbx-sub-card flex flex-col justify-between modern-cv-card">
              <div class="flex items-center justify-between gap-2 text-xs font-bold" style="color: var(--cbx-accent-fg);">
                <span class="flex items-center gap-1.5">⏱ 延时中位数 & 抖动</span>
                <span class="text-[11px] font-mono px-2 py-0.5 rounded-md cbx-badge-info font-tabular">
                  {{ q1.samples || 0 }}/10 采样
                </span>
              </div>
              <div class="mt-3 space-y-2 text-xs font-mono font-tabular">
                <div class="flex justify-between items-center cbx-text-sub">
                  <span>10次探测中位数:</span>
                  <span class="font-bold cbx-text-main">{{ formatMetric(q1.latency_median_ms, 'ms') }}</span>
                </div>
                <div class="flex justify-between items-center cbx-text-sub">
                  <span>相邻抖动均值:</span>
                  <span class="font-bold" style="color: var(--cbx-ok-fg);">{{ formatMetric(q1.jitter_avg_ms, 'ms') }}</span>
                </div>
                <div class="flex justify-between items-center cbx-text-sub">
                  <span>线路稳定性:</span>
                  <span class="font-bold" style="color: var(--cbx-accent-fg);">{{ q1.stability_grade || '普通' }}</span>
                </div>
              </div>
            </div>

            <!-- Quadrant 2: Throughput & Passive Bandwidth -->
            <div class="p-4 rounded-2xl cbx-sub-card flex flex-col justify-between modern-cv-card">
              <div class="flex items-center justify-between gap-2 text-xs font-bold" style="color: var(--cbx-accent-fg);">
                <span class="flex items-center gap-1.5">⚡ 被动吞吐 & 闪电测速</span>
                <span
                  class="text-[11px] font-mono px-2 py-0.5 rounded-md font-tabular"
                  :class="q2.unmeasured ? 'cbx-btn-ghost cbx-text-muted' : 'cbx-badge-info'"
                >
                  {{ q2.unmeasured ? '未跑大流量' : '业务流已采样' }}
                </span>
              </div>
              <div class="mt-3 space-y-2 text-xs font-mono font-tabular">
                <div class="flex justify-between items-center cbx-text-sub">
                  <span>峰值带宽估算:</span>
                  <span class="font-bold cbx-text-main">{{ q2.bandwidth_mbps ? q2.bandwidth_mbps.toFixed(1) + ' Mbps' : '未跑大流(赋75分)' }}</span>
                </div>
                <div class="flex justify-between items-center cbx-text-sub">
                  <span>采样触发门槛:</span>
                  <span class="cbx-text-muted font-semibold">单次传输 ≥ 256KB</span>
                </div>
                <div class="flex justify-between items-center cbx-text-sub">
                  <span>微量突发限流:</span>
                  <span class="font-bold" style="color: var(--cbx-warn-fg);">≤ 2MB 强制熔断</span>
                </div>
              </div>
            </div>

            <!-- Quadrant 3: IP & ASN Metadata -->
            <div class="p-4 rounded-2xl cbx-sub-card flex flex-col justify-between modern-cv-card">
              <div class="flex items-center justify-between gap-2 text-xs font-bold" style="color: var(--cbx-purple-fg);">
                <span class="flex items-center gap-1.5">🌐 出口 IP 与 ASN 归属</span>
                <span class="text-[11px] font-mono px-2 py-0.5 rounded-md cbx-badge-purple">
                  {{ q3.country_code || 'UN' }}
                </span>
              </div>
              <div class="mt-3 space-y-2 text-xs font-mono">
                <div class="flex justify-between items-center cbx-text-sub">
                  <span>地理位置:</span>
                  <span class="font-bold cbx-text-main">{{ q3.region || '国际' }}</span>
                </div>
                <div class="flex justify-between items-center cbx-text-sub">
                  <span>自治系统:</span>
                  <span class="font-bold font-tabular" style="color: var(--cbx-purple-fg);">{{ q3.asn || 'AS13335' }}</span>
                </div>
                <div class="flex justify-between items-center cbx-text-sub">
                  <span>运营商 ISP:</span>
                  <span class="font-bold cbx-text-main truncate max-w-[160px]">{{ q3.isp || 'Tier-1 Transit' }}</span>
                </div>
              </div>
            </div>

            <!-- Quadrant 4: Cleanliness & Native Risk Assessment -->
            <div class="p-4 rounded-2xl cbx-sub-card flex flex-col justify-between modern-cv-card">
              <div class="flex items-center justify-between gap-2 text-xs font-bold" style="color: var(--cbx-ok-fg);">
                <span class="flex items-center gap-1.5">🛡️ 纯净度与欺诈风险评估</span>
                <span
                  class="text-[11px] font-mono px-2 py-0.5 rounded-md"
                  :class="q4.ip_type === 'Residential' ? 'cbx-badge-ok' : (q4.ip_type === 'Commercial' ? 'cbx-badge-info' : 'cbx-btn-ghost cbx-text-sub')"
                >
                  {{ q4.ip_type === 'Residential' ? '原生住宅宽带' : (q4.ip_type === 'Commercial' ? '企业BGP专线' : '机房数据中心') }}
                </span>
              </div>
              <div class="mt-3 space-y-2 text-xs font-mono font-tabular">
                <div class="flex justify-between items-center cbx-text-sub">
                  <span>欺诈风险指数:</span>
                  <span class="font-bold" :style="{ color: (q4.fraud_score || 0) < 15 ? 'var(--cbx-ok-fg)' : 'var(--cbx-warn-fg)' }">
                    {{ q4.fraud_score ?? 0 }} / 100 ({{ (q4.fraud_score || 0) < 15 ? '低风险' : '常规机房' }})
                  </span>
                </div>
                <div class="flex justify-between items-center cbx-text-sub">
                  <span>信誉级别:</span>
                  <span class="font-bold" style="color: var(--cbx-ok-fg);">{{ q4.trust_level === 'Native' ? '98% 原生极高信誉' : '85% 优质可靠' }}</span>
                </div>
                <div v-if="(q4.risk_flags || []).length > 0" class="flex flex-wrap gap-1.5 pt-1">
                  <span
                    v-for="tag in q4.risk_flags"
                    :key="tag"
                    class="text-[10px] px-2 py-0.5 rounded-md cbx-badge-ok font-semibold"
                  >
                    {{ tag }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </dialog>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { apiFetch } from '../../utils/apiFetch'

const props = defineProps({
  visible: Boolean,
  nodeName: String
})

const emit = defineEmits(['update:visible', 'burst-done'])

const dialogRef = ref(null)
const perspectiveData = ref(null)
const testingBurst = ref(false)
const loading = ref(false)
const loadError = ref('')

const q1 = computed(() => perspectiveData.value?.quadrant_1_latency || {})
const q2 = computed(() => perspectiveData.value?.quadrant_2_bandwidth || {})
const q3 = computed(() => perspectiveData.value?.quadrant_3_asn || {})
const q4 = computed(() => perspectiveData.value?.quadrant_4_cleanliness || {})
const overallScore = computed(() => perspectiveData.value?.overall_score ?? 75.0)

const overallBadgeClass = computed(() => {
  if (overallScore.value >= 80) return 'cbx-badge-ok'
  if (overallScore.value < 60) return 'cbx-badge-danger'
  return 'cbx-badge-warn'
})

const overallScoreColor = computed(() => {
  if (overallScore.value >= 80) return 'var(--cbx-accent-fg)'
  if (overallScore.value < 60) return 'var(--cbx-danger-fg)'
  return 'var(--cbx-warn-fg)'
})

function formatMetric(val, unit) {
  if (val === undefined || val === null || val === '') return '--'
  return `${val} ${unit}`
}

watch(
  () => [props.visible, props.nodeName],
  async ([vis, node], [prevVis, prevNode] = []) => {
    await nextTick()
    if (vis) {
      if (node !== prevNode || !prevVis) {
        if (node !== prevNode) {
          perspectiveData.value = null
        }
        if (node) {
          fetchPerspective(node)
        }
      }
      if (dialogRef.value && !dialogRef.value.open) {
        try {
          dialogRef.value.showModal()
        } catch (e) {}
      }
    } else {
      if (dialogRef.value && dialogRef.value.open) {
        dialogRef.value.close()
      }
    }
  },
  { immediate: true }
)

function handleBackdropClick(e) {
  if (!dialogRef.value || e.target !== dialogRef.value) return
  if ('closedBy' in HTMLDialogElement.prototype) {
    close()
    return
  }
  // Fallback for browsers without closedby="any" support (e.g., Safari)
  const rect = dialogRef.value.getBoundingClientRect()
  const inside =
    rect.top <= e.clientY &&
    e.clientY <= rect.top + rect.height &&
    rect.left <= e.clientX &&
    e.clientX <= rect.left + rect.width
  if (!inside || e.target === dialogRef.value) {
    close()
  }
}

async function fetchPerspective(node, force = false) {
  if (!node) return
  loading.value = true
  loadError.value = ''
  try {
    const res = await apiFetch(`/api/v1/config/proxies/ip-perspective?node=${encodeURIComponent(node)}${force ? '&refresh=1' : ''}&_t=${Date.now()}`)
    const data = await res.json().catch(() => null)
    if (res.ok && data?.ok) {
      perspectiveData.value = data
    } else {
      loadError.value = data?.error || `获取节点透视数据失败 (${res.status})`
    }
  } catch (e) {
    loadError.value = e?.message || '网络请求异常'
  } finally {
    loading.value = false
  }
}

async function triggerBurst() {
  if (!props.nodeName || testingBurst.value) return
  testingBurst.value = true
  try {
    const res = await apiFetch('/api/v1/config/proxies/burst-speedtest', {
      method: 'POST',
      json: { node: props.nodeName },
      timeoutMs: 30000
    })
    const data = await res.json().catch(() => null)
    if (res.ok && data?.ok) {
      await fetchPerspective(props.nodeName, true)
      emit('burst-done', data)
    } else {
      loadError.value = data?.error || '闪电测速失败'
    }
  } catch (e) {
    loadError.value = e?.message || '测速请求异常'
  } finally {
    testingBurst.value = false
  }
}

function close() {
  emit('update:visible', false)
}

defineExpose({
  fetchPerspective
})
</script>
