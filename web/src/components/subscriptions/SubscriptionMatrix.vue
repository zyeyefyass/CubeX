<template>
  <div class="bg-[#00223e]/60 backdrop-blur-xl border border-cyan-500/20 rounded-2xl p-5 mb-6 shadow-2xl relative overflow-hidden">
    <!-- Ambient Glow Background -->
    <div class="absolute -right-20 -top-20 w-64 h-64 bg-cyan-500/10 rounded-full blur-3xl pointer-events-none"></div>
    <div class="absolute -left-20 -bottom-20 w-64 h-64 bg-purple-500/10 rounded-full blur-3xl pointer-events-none"></div>

    <!-- Header Section -->
    <div class="flex items-center justify-between mb-4 relative z-10">
      <div class="flex items-center space-x-3">
        <div class="w-9 h-9 rounded-xl bg-gradient-to-br from-cyan-500/20 to-purple-500/20 border border-cyan-400/30 flex items-center justify-center text-cyan-300 shadow-inner">
          <svg class="w-5 h-5 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
        </div>
        <div>
          <h3 class="text-white font-semibold text-base tracking-wide flex items-center gap-2">
            全息订阅源 SLA 质量矩阵
          </h3>
          <p class="text-xs text-slate-400 mt-0.5">多源可用率比对 · 中位数延迟透视 · 故障节点自动筛查</p>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <button
          @click="refreshQuality"
          :disabled="loading"
          class="px-3 py-1.5 text-xs rounded-lg bg-cyan-500/10 hover:bg-cyan-500/20 text-cyan-300 border border-cyan-500/30 transition-all flex items-center gap-1.5 disabled:opacity-50"
        >
          <svg class="w-3.5 h-3.5" :class="{ 'animate-spin': loading }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          {{ loading ? '评估中...' : '刷新质量矩阵' }}
        </button>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="!loading && reports.length === 0" class="text-center py-6 text-slate-400 text-xs">
      暂未检测到有效订阅源数据，请先添加并更新订阅
    </div>

    <!-- Matrix Cards Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 relative z-10">
      <div
        v-for="sub in reports"
        :key="sub.subscription_id"
        class="bg-[#001529]/70 border border-slate-700/50 hover:border-cyan-500/40 rounded-xl p-4 transition-all duration-300 hover:shadow-lg hover:shadow-cyan-500/10 group flex flex-col justify-between"
      >
        <div>
          <!-- Sub Title & Grade -->
          <div class="flex items-center justify-between mb-3">
            <span class="text-sm font-medium text-slate-200 truncate max-w-[160px]" :title="sub.subscription_name">
              {{ sub.subscription_name }}
            </span>
            <span
              class="text-xs px-2 py-0.5 rounded-md font-medium border"
              :class="getGradeBadgeClass(sub.health_grade)"
            >
              {{ sub.health_grade }}
            </span>
          </div>

          <!-- Availability Progress Bar -->
          <div class="mb-3">
            <div class="flex justify-between text-xs mb-1">
              <span class="text-slate-400">可用率 (SLA)</span>
              <span class="font-mono text-cyan-300 font-bold">{{ sub.availability_rate.toFixed(1) }}%</span>
            </div>
            <div class="w-full h-1.5 bg-slate-800 rounded-full overflow-hidden">
              <div
                class="h-full rounded-full transition-all duration-500"
                :class="getProgressBarClass(sub.availability_rate)"
                :style="{ width: `${Math.min(sub.availability_rate, 100)}%` }"
              ></div>
            </div>
          </div>

          <!-- Stats Grid -->
          <div class="grid grid-cols-3 gap-2 text-center bg-[#002d50]/40 rounded-lg p-2 border border-slate-700/30">
            <div>
              <div class="text-[10px] text-slate-400">健康节点</div>
              <div class="text-xs font-mono font-bold text-emerald-400 mt-0.5">
                {{ sub.online_nodes }} / {{ sub.total_nodes }}
              </div>
            </div>
            <div>
              <div class="text-[10px] text-slate-400">中位延迟</div>
              <div class="text-xs font-mono font-bold text-cyan-300 mt-0.5">
                {{ sub.median_latency_ms > 0 ? `${sub.median_latency_ms}ms` : '--' }}
              </div>
            </div>
            <div>
              <div class="text-[10px] text-slate-400">失联节点</div>
              <div class="text-xs font-mono font-bold mt-0.5" :class="sub.offline_nodes > 0 ? 'text-amber-400' : 'text-slate-500'">
                {{ sub.offline_nodes }}
              </div>
            </div>
          </div>
        </div>

        <!-- Footer / Warning -->
        <div v-if="sub.offline_nodes > 0" class="mt-3 pt-2 border-t border-slate-800 text-[11px] text-amber-300/80 flex items-center justify-between">
          <span>⚠️ 存在 {{ sub.offline_nodes }} 个失联节点</span>
          <span class="text-[10px] text-slate-500">已智能旁路隔离</span>
        </div>
        <div v-else class="mt-3 pt-2 border-t border-slate-800 text-[11px] text-emerald-400/80 flex items-center gap-1">
          <span>✨ 全节点 100% 满分可用</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const loading = ref(false)
const reports = ref([])

const refreshQuality = async () => {
  loading.value = true
  try {
    const res = await fetch('/api/v1/config/subscriptions/quality')
    const data = await res.json()
    if (data.code === 0 && Array.isArray(data.data)) {
      reports.value = data.data
    }
  } catch (err) {
    console.error('Failed to fetch subscription quality matrix:', err)
  } finally {
    loading.value = false
  }
}

const getGradeBadgeClass = (grade) => {
  if (!grade) return 'bg-slate-700/50 text-slate-300 border-slate-600'
  if (grade.includes('A+')) return 'bg-cyan-500/20 text-cyan-300 border-cyan-400/40 shadow-sm shadow-cyan-500/20'
  if (grade.includes('A')) return 'bg-emerald-500/20 text-emerald-300 border-emerald-400/40'
  if (grade.includes('B')) return 'bg-blue-500/20 text-blue-300 border-blue-400/40'
  return 'bg-amber-500/20 text-amber-300 border-amber-400/40'
}

const getProgressBarClass = (rate) => {
  if (rate >= 90) return 'bg-gradient-to-r from-cyan-500 to-emerald-400'
  if (rate >= 75) return 'bg-gradient-to-r from-blue-500 to-cyan-400'
  if (rate >= 50) return 'bg-gradient-to-r from-amber-500 to-yellow-400'
  return 'bg-gradient-to-r from-rose-500 to-red-400'
}

onMounted(() => {
  refreshQuality()
})
</script>
