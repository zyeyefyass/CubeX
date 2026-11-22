<template>
  <dialog ref="dialogRef" class="modern-dialog select-none" @click="handleBackdropClick" @cancel="close">
    <div class="fixed inset-0 bg-black/60 flex items-center justify-center p-3 sm:p-5 overflow-y-auto animate-in fade-in" @click.self="close">
      <div class="bg-[#E5EAF1] border border-slate-300/60 rounded-3xl max-w-2xl w-full shadow-2xl flex flex-col text-slate-800 overflow-hidden relative">
        <!-- Modal Header -->
        <div class="px-5 py-4 bg-[#F1F4F9] border-b border-slate-300/60 flex items-center justify-between gap-3 select-none">
          <div class="flex items-center gap-2.5">
            <div class="w-8 h-8 rounded-xl bg-gradient-to-tr from-cyan-500/20 to-sky-500/20 border border-cyan-400/30 flex items-center justify-center text-cyan-300 font-bold">
              🔍
            </div>
            <div>
              <div class="flex items-center gap-2">
                <h3 class="text-sm sm:text-base font-extrabold text-slate-900 tracking-wide truncate max-w-[280px] sm:max-w-[360px]">
                  {{ nodeName }}
                </h3>
                <span class="text-[10px] font-mono px-2 py-0.5 rounded-full font-bold border font-tabular"
                  :class="overallScore >= 80 ? 'bg-emerald-500/15 text-emerald-300 border-emerald-400/30' : (overallScore < 60 ? 'bg-rose-500/20 text-rose-300 border-rose-500/40 animate-pulse' : 'bg-amber-500/15 text-amber-300 border-amber-400/30')">
                  {{ overallScore >= 90 ? 'S 极致稳定' : (overallScore >= 80 ? 'A+ 极高品质' : (overallScore >= 60 ? 'A 优质可用' : 'C 波动预警')) }}
                </span>
              </div>
              <p class="text-[11px] text-slate-500 font-mono mt-0.5">
                稳定三角评分 (40% 延时中位数 × 30% 抖动平均 × 30% 被动吞吐)
              </p>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <button @click="triggerBurst" :disabled="testingBurst"
              class="px-3 py-1 rounded-xl text-xs font-bold bg-amber-500/20 hover:bg-amber-500/30 text-amber-300 border border-amber-400/40 flex items-center gap-1.5 transition active:scale-95 cursor-pointer font-tabular"
              title="单次 ≤2MB 闪电突发测速，零额外流量损耗">
              <span v-if="testingBurst" class="inline-block animate-spin">↻</span>
              <span v-else>⚡</span>
              <span>{{ testingBurst ? '测速中...' : '闪电测速' }}</span>
            </button>
            <button @click="close" class="px-2.5 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 text-slate-700 hover:text-white text-xs font-bold transition cursor-pointer">
              ✕
            </button>
          </div>
        </div>

        <!-- Modal Body (Twilight Crystal 4-Quadrant Perspective) -->
        <div class="p-5 space-y-4">
          <!-- Top Score Banner -->
          <div class="p-4 rounded-2xl bg-gradient-to-r from-sky-950/40 via-cyan-950/30 to-purple-950/40 border border-white/10 flex items-center justify-between">
            <div>
              <div class="text-[11px] font-mono text-slate-500">稳定三角综合得分</div>
              <div class="text-3xl font-black font-mono tracking-tight font-tabular"
                :class="overallScore >= 80 ? 'text-cyan-300' : (overallScore < 60 ? 'text-rose-400' : 'text-amber-300')">
                {{ Math.round(overallScore) }} <span class="text-sm font-normal text-slate-500">/ 100</span>
              </div>
            </div>
            <div class="text-right text-[11px] font-mono space-y-1 text-slate-700 font-tabular">
              <div>延时中位数: <span class="font-bold text-white">{{ q1.latency_median_ms || '--' }} ms</span></div>
              <div>相邻抖动差: <span class="font-bold text-white">{{ q1.jitter_avg_ms || '--' }} ms</span></div>
              <div>峰值吞吐量: <span class="font-bold text-white">{{ q2.bandwidth_mbps ? q2.bandwidth_mbps.toFixed(1) + ' Mbps' : '75分中性基准' }}</span></div>
            </div>
          </div>

          <!-- 4-Quadrant Grid (Modern CSS Containment) -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
            <!-- Quadrant 1: Latency & Jitter -->
            <div class="p-4 rounded-2xl bg-white/[0.03] border border-white/10 flex flex-col justify-between hover:border-cyan-500/40 transition modern-cv-card">
              <div class="flex items-center justify-between text-xs font-bold text-cyan-300">
                <span class="flex items-center gap-1.5">⏱ 延时中位数 & 抖动</span>
                <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-cyan-950/80 text-cyan-300 border border-cyan-500/30 font-tabular">
                  {{ q1.samples || 0 }}/10 采样
                </span>
              </div>
              <div class="mt-3 space-y-2 text-xs font-mono font-tabular">
                <div class="flex justify-between items-center text-slate-700">
                  <span>10次探测中位数:</span>
                  <span class="font-bold text-white">{{ q1.latency_median_ms || 0 }} ms</span>
                </div>
                <div class="flex justify-between items-center text-slate-700">
                  <span>相邻抖动均值:</span>
                  <span class="font-bold text-emerald-300">{{ q1.jitter_avg_ms || 0 }} ms</span>
                </div>
                <div class="flex justify-between items-center text-slate-700">
                  <span>线路稳定性:</span>
                  <span class="font-bold text-cyan-200">{{ q1.stability_grade || '普通' }}</span>
                </div>
              </div>
            </div>

            <!-- Quadrant 2: Throughput & Passive Bandwidth -->
            <div class="p-4 rounded-2xl bg-white/[0.03] border border-white/10 flex flex-col justify-between hover:border-sky-500/40 transition modern-cv-card">
              <div class="flex items-center justify-between text-xs font-bold text-sky-300">
                <span class="flex items-center gap-1.5">⚡ 被动吞吐 & 闪电测速</span>
                <span class="text-[10px] font-mono px-1.5 py-0.5 rounded font-tabular"
                  :class="q2.unmeasured ? 'bg-slate-800 text-slate-500' : 'bg-sky-950/80 text-sky-300 border border-sky-500/30'">
                  {{ q2.unmeasured ? '未跑大流量' : '业务流已采样' }}
                </span>
              </div>
              <div class="mt-3 space-y-2 text-xs font-mono font-tabular">
                <div class="flex justify-between items-center text-slate-700">
                  <span>峰值带宽估算:</span>
                  <span class="font-bold text-white">{{ q2.bandwidth_mbps ? q2.bandwidth_mbps.toFixed(1) + ' Mbps' : '未跑大流(赋75分)' }}</span>
                </div>
                <div class="flex justify-between items-center text-slate-700">
                  <span>采样触发门槛:</span>
                  <span class="text-slate-500">单次传输 ≥ 256KB</span>
                </div>
                <div class="flex justify-between items-center text-slate-700">
                  <span>微量突发限流:</span>
                  <span class="text-amber-300 font-bold">≤ 2MB 强制熔断</span>
                </div>
              </div>
            </div>

            <!-- Quadrant 3: IP & ASN Metadata -->
            <div class="p-4 rounded-2xl bg-white/[0.03] border border-white/10 flex flex-col justify-between hover:border-purple-500/40 transition modern-cv-card">
              <div class="flex items-center justify-between text-xs font-bold text-purple-300">
                <span class="flex items-center gap-1.5">🌐 出口 IP 与 ASN 归属</span>
                <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-purple-950/80 text-purple-300 border border-purple-500/30">
                  {{ q3.country_code || 'UN' }}
                </span>
              </div>
              <div class="mt-3 space-y-2 text-xs font-mono">
                <div class="flex justify-between items-center text-slate-700">
                  <span>地理位置:</span>
                  <span class="font-bold text-white">{{ q3.region || '国际' }}</span>
                </div>
                <div class="flex justify-between items-center text-slate-700">
                  <span>自治系统:</span>
                  <span class="font-bold text-purple-200 font-tabular">{{ q3.asn || 'AS13335' }}</span>
                </div>
                <div class="flex justify-between items-center text-slate-700">
                  <span>运营商 ISP:</span>
                  <span class="font-bold text-slate-800 truncate max-w-[150px]">{{ q3.isp || 'Tier-1 Transit' }}</span>
                </div>
              </div>
            </div>

            <!-- Quadrant 4: Cleanliness & Native Risk Assessment -->
            <div class="p-4 rounded-2xl bg-white/[0.03] border border-white/10 flex flex-col justify-between hover:border-emerald-500/40 transition modern-cv-card">
              <div class="flex items-center justify-between text-xs font-bold text-emerald-300">
                <span class="flex items-center gap-1.5">🛡️ 纯净度与欺诈风险评估</span>
                <span class="text-[10px] font-mono px-1.5 py-0.5 rounded"
                  :class="q4.ip_type === 'Residential' ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-400/30' : (q4.ip_type === 'Commercial' ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-400/30' : 'bg-slate-800 text-slate-700')">
                  {{ q4.ip_type === 'Residential' ? '原生住宅宽带' : (q4.ip_type === 'Commercial' ? '企业BGP专线' : '机房数据中心') }}
                </span>
              </div>
              <div class="mt-3 space-y-2 text-xs font-mono font-tabular">
                <div class="flex justify-between items-center text-slate-700">
                  <span>欺诈风险指数:</span>
                  <span class="font-bold" :class="q4.fraud_score < 15 ? 'text-emerald-400' : 'text-amber-400'">
                    {{ q4.fraud_score || 0 }} / 100 (低风险)
                  </span>
                </div>
                <div class="flex justify-between items-center text-slate-700">
                  <span>信誉级别:</span>
                  <span class="font-bold text-emerald-300">{{ q4.trust_level === 'Native' ? '98% 原生极高信誉' : '85% 优质可靠' }}</span>
                </div>
                <div class="flex flex-wrap gap-1 mt-1">
                  <span v-for="tag in (q4.risk_flags || [])" :key="tag"
                    class="text-[9px] px-1.5 py-0.2 rounded bg-emerald-950/70 text-emerald-300 border border-emerald-500/30">
                    {{ tag }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </dialog>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'

const props = defineProps({
  visible: Boolean,
  nodeName: String
})

const emit = defineEmits(['update:visible', 'burst-done'])

const dialogRef = ref(null)
const perspectiveData = ref(null)
const testingBurst = ref(false)

const q1 = computed(() => perspectiveData.value?.quadrant_1_latency || {})
const q2 = computed(() => perspectiveData.value?.quadrant_2_bandwidth || {})
const q3 = computed(() => perspectiveData.value?.quadrant_3_asn || {})
const q4 = computed(() => perspectiveData.value?.quadrant_4_cleanliness || {})
const overallScore = computed(() => perspectiveData.value?.overall_score || 75.0)

watch(() => props.visible, async (val) => {
  await nextTick()
  if (val) {
    if (dialogRef.value && !dialogRef.value.open) {
      try {
        dialogRef.value.showModal()
      } catch (e) {
        // Fallback if already opened
      }
    }
  } else {
    if (dialogRef.value && dialogRef.value.open) {
      dialogRef.value.close()
    }
  }
}, { immediate: true })

function handleBackdropClick(e) {
  if (e.target === dialogRef.value) {
    close()
  }
}

async function fetchPerspective(node, force = false) {
  if (!node) return
  try {
    const res = await fetch(`/api/v1/config/proxies/ip-perspective?node=${encodeURIComponent(node)}${force ? '&refresh=1' : ''}&_t=${Date.now()}`)
    if (res.ok) {
      const data = await res.json()
      if (data.ok) {
        perspectiveData.value = data
      }
    }
  } catch (e) {}
}

async function triggerBurst() {
  if (!props.nodeName || testingBurst.value) return
  testingBurst.value = true
  try {
    const res = await fetch('/api/v1/config/proxies/burst-speedtest', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ node: props.nodeName })
    })
    const data = await res.json()
    if (data.ok) {
      await fetchPerspective(props.nodeName, true)
      emit('burst-done', data)
    }
  } catch (e) {
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
