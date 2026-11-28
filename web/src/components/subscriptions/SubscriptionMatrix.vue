<template>
  <div class="cbx-card rounded-2xl p-4 sm:p-5 mb-5 shadow-lg relative overflow-hidden transition-all duration-300">
    <!-- Header Section / Collapsed Capsule -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 relative z-10" :class="{ 'mb-4': isExpanded }">
      <div class="flex items-center space-x-3 cursor-pointer select-none min-w-0" @click="isExpanded = !isExpanded">
        <div class="w-8 h-8 rounded-xl cbx-badge-info border flex items-center justify-center shadow-inner shrink-0">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
        </div>
        <div class="min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <h3 class="cbx-text-main font-semibold text-sm sm:text-base tracking-wide flex items-center gap-1.5">
              全息订阅源 SLA 质量透视
            </h3>
            <span class="text-[11px] font-mono px-2 py-0.5 rounded-full border font-bold" :class="summaryBadgeClass">
              {{ summaryText }}
            </span>
          </div>
          <p v-if="isExpanded" class="text-xs cbx-text-sub mt-0.5">多源可用率比对 · 中位数延迟透视 · 故障节点自动筛查</p>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center gap-2 self-end sm:self-auto shrink-0">
        <button
          @click="refreshQuality"
          :disabled="loading"
          class="px-3 py-1.5 text-xs rounded-lg cbx-badge-info border font-semibold transition-all flex items-center gap-1.5 disabled:opacity-50 cursor-pointer"
          title="重新评估所有订阅源质量"
        >
          <svg class="w-3.5 h-3.5" :class="{ 'animate-spin': loading }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span>{{ loading ? '评估中...' : '刷新质量' }}</span>
        </button>

        <button
          @click="isExpanded = !isExpanded"
          class="px-3 py-1.5 text-xs rounded-lg cbx-btn-ghost transition flex items-center gap-1 cursor-pointer font-mono font-semibold"
        >
          <span>{{ isExpanded ? '收起矩阵' : '展开详情' }}</span>
          <span class="text-[10px] transform transition-transform" :class="{ 'rotate-180': isExpanded }">▼</span>
        </button>
      </div>
    </div>

    <!-- Expanded Body Content -->
    <div v-show="isExpanded" class="mt-4 transition-all duration-300">
      <div v-if="!loading && reports.length === 0" class="text-center py-6 cbx-text-muted text-xs">
        暂未检测到有效订阅源数据，请先添加并更新订阅
      </div>

      <!-- Matrix Cards Grid -->
      <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 relative z-10">
        <div
          v-for="sub in enhancedReports"
          :key="sub.subscription_id"
          class="cbx-sub-card rounded-xl p-4 transition-all duration-300 hover:shadow-lg flex flex-col justify-between"
        >
          <div>
            <!-- Sub Title & Grade -->
            <div class="flex items-center justify-between gap-2 mb-3">
              <span class="text-sm font-bold cbx-text-main truncate max-w-[170px]" :title="sub.subscription_name">
                {{ sub.subscription_name }}
              </span>
              <span
                class="text-xs px-2 py-0.5 rounded-md font-semibold border shrink-0"
                :class="sub.displayState.toneClass"
              >
                {{ sub.displayState.badgeText }}
              </span>
            </div>

            <!-- Availability Progress Bar -->
            <div class="mb-3">
              <div class="flex justify-between text-xs mb-1">
                <span class="cbx-text-sub">可用率 (SLA)</span>
                <span class="font-mono font-bold text-sky-600 dark:text-cyan-300 font-tabular">{{ sub.displayState.rateText }}</span>
              </div>
              <div class="w-full h-1.5 cbx-progress-track rounded-full overflow-hidden">
                <div
                  class="h-full rounded-full transition-all duration-500"
                  :class="sub.displayState.barClass"
                  :style="{ width: `${sub.displayState.progressWidth}%` }"
                ></div>
              </div>
            </div>

            <!-- Stats Grid -->
            <div class="grid grid-cols-3 gap-2 text-center cbx-sub-card rounded-lg p-2">
              <div>
                <div class="text-[10px] cbx-text-muted">健康节点</div>
                <div class="text-xs font-mono font-bold text-emerald-600 dark:text-emerald-400 mt-0.5 font-tabular">
                  {{ sub.online_nodes }} / {{ sub.total_nodes }}
                </div>
              </div>
              <div>
                <div class="text-[10px] cbx-text-muted">中位延迟</div>
                <div class="text-xs font-mono font-bold text-sky-600 dark:text-cyan-300 mt-0.5 font-tabular">
                  {{ sub.median_latency_ms > 0 ? `${sub.median_latency_ms}ms` : '--' }}
                </div>
              </div>
              <div>
                <div class="text-[10px] cbx-text-muted">失联节点</div>
                <div class="text-xs font-mono font-bold mt-0.5 font-tabular" :class="sub.offline_nodes > 0 && sub.displayState.state !== 'STANDBY' ? 'text-amber-600 dark:text-amber-400' : 'cbx-text-sub'">
                  {{ sub.displayState.state === 'STANDBY' ? 0 : sub.offline_nodes }}
                </div>
              </div>
            </div>
          </div>

          <div v-if="sub.displayState.state === 'STANDBY'" class="mt-3 pt-2 border-t text-[11px] text-sky-600 dark:text-cyan-300 flex items-center justify-between" style="border-color: var(--cbx-border-soft);">
            <span>✨ 待命就绪</span>
            <span class="text-[10px] cbx-text-muted font-mono">按需发起测速评估</span>
          </div>
          <div v-else-if="sub.offline_nodes > 0" class="mt-3 pt-2 border-t text-[11px] text-amber-600 dark:text-amber-300 flex items-center justify-between" style="border-color: var(--cbx-border-soft);">
            <span>⚠️ 存在 {{ sub.offline_nodes }} 个失联节点</span>
            <span class="text-[10px] cbx-text-muted">已智能旁路隔离</span>
          </div>
          <div v-else class="mt-3 pt-2 border-t text-[11px] text-emerald-600 dark:text-emerald-400 flex items-center gap-1" style="border-color: var(--cbx-border-soft);">
            <span>✨ 全节点 100% 满分可用</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, inject, onMounted } from 'vue'
import { apiFetch } from '../../utils/apiFetch'

const app = inject('cubexApp', null)
const loading = ref(false)
const reports = ref([])
const isExpanded = ref(false)

function resolveSlaDisplayState(totalProbes, successProbes, avgRttMs, rawState, healthGrade) {
  if (rawState === 'STANDBY' || healthGrade === '待命就绪' || !totalProbes || totalProbes <= 0) {
    return {
      state: 'STANDBY',
      badgeText: '已就绪 · 待测速',
      rateText: '待命自适应',
      toneClass: 'cbx-badge-info',
      barClass: 'bg-gradient-to-r from-sky-500 to-cyan-400',
      progressWidth: 100
    }
  }

  const ratio = (successProbes / totalProbes) * 100
  if (ratio >= 80) {
    return {
      state: 'HEALTHY',
      badgeText: `健康 (${avgRttMs || 0}ms)`,
      rateText: `${ratio.toFixed(1)}%`,
      toneClass: 'cbx-badge-ok',
      barClass: 'bg-gradient-to-r from-cyan-500 to-emerald-400',
      progressWidth: Math.min(ratio, 100)
    }
  }
  if (ratio >= 30) {
    return {
      state: 'DEGRADED',
      badgeText: '部分节点波动',
      rateText: `${ratio.toFixed(1)}%`,
      toneClass: 'cbx-badge-warn',
      barClass: 'bg-gradient-to-r from-amber-500 to-yellow-400',
      progressWidth: Math.min(ratio, 100)
    }
  }
  return {
    state: 'OFFLINE',
    badgeText: '节点不可达',
    rateText: `${ratio.toFixed(1)}%`,
    toneClass: 'cbx-badge-danger',
    barClass: 'bg-gradient-to-r from-rose-500 to-red-400',
    progressWidth: Math.min(ratio, 100)
  }
}

const enhancedReports = computed(() => {
  return reports.value.map(sub => ({
    ...sub,
    displayState: resolveSlaDisplayState(
      sub.total_nodes,
      sub.online_nodes,
      sub.median_latency_ms,
      sub.state,
      sub.health_grade
    )
  }))
})

const summaryText = computed(() => {
  if (reports.value.length === 0) return '未接入订阅'
  const isAllStandby = enhancedReports.value.every(r => r.displayState.state === 'STANDBY')
  if (isAllStandby) {
    return `${reports.value.length} 个订阅源 · 待命就绪`
  }
  const avgRate = enhancedReports.value.reduce((acc, cur) => acc + (cur.availability_rate || 0), 0) / (reports.value.length || 1)
  return `${reports.value.length} 个订阅源 · 平均可用率 ${avgRate.toFixed(1)}%`
})

const summaryBadgeClass = computed(() => {
  if (reports.value.length === 0) return 'cbx-btn-ghost'
  const isAllStandby = enhancedReports.value.every(r => r.displayState.state === 'STANDBY')
  if (isAllStandby) return 'cbx-badge-info'
  return 'cbx-badge-ok'
})

const refreshQuality = async () => {
  loading.value = true
  try {
    const res = await apiFetch('/api/v1/config/subscriptions/quality')
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

onMounted(() => {
  if (!app?.needsAuthUnlock) {
    refreshQuality()
  }
})
</script>
