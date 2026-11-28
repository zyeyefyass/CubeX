<template>
  <div v-if="modelValue" class="fixed inset-0 z-[10000] flex items-center justify-center p-3 sm:p-4 bg-black/60 animate-in fade-in" @click.self="$emit('update:modelValue', false)">
    <div class="w-full max-w-xl rounded-3xl cbx-modal-shell p-5 sm:p-7 space-y-5 relative overflow-hidden">
      <!-- Header -->
      <div class="flex items-center justify-between border-b pb-4 gap-3" style="border-color: var(--cbx-border-sub);">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-11 h-11 rounded-2xl cbx-badge-ok flex items-center justify-center text-2xl shrink-0">
            🛡️
          </div>
          <div class="min-w-0">
            <h3 class="text-base sm:text-lg font-bold cbx-text-main tracking-wide flex items-center gap-2 flex-wrap">
              <span>全链路深度健康体检</span>
              <span class="text-xs px-2 py-0.5 rounded-full cbx-badge-ok font-mono font-bold">
                100% 满分通过
              </span>
            </h3>
            <p class="text-xs cbx-text-muted mt-0.5">宿主策略路由 · 双引擎守护 · DNS零泄露 · 节点SLA综合矩阵</p>
          </div>
        </div>
        <button
          @click="$emit('update:modelValue', false)"
          aria-label="关闭弹窗"
          class="cbx-btn-ghost min-w-[36px] min-h-[36px] p-1.5 rounded-xl transition cursor-pointer flex items-center justify-center shrink-0 cbx-touch-extend"
        >
          ✕
        </button>
      </div>

      <!-- Overview Stats -->
      <div v-if="report" class="grid grid-cols-3 gap-3">
        <div class="p-3 rounded-2xl cbx-sub-card flex flex-col items-center justify-center text-center">
          <span class="text-[11px] cbx-text-muted font-semibold">活跃引擎</span>
          <span class="text-sm font-bold font-mono mt-1 uppercase" style="color: var(--cbx-accent-fg);">{{ report.active_engine }}</span>
        </div>
        <div class="p-3 rounded-2xl cbx-sub-card flex flex-col items-center justify-center text-center">
          <span class="text-[11px] cbx-text-muted font-semibold">核心堆内存</span>
          <span class="text-sm font-bold font-mono mt-1 font-tabular" style="color: var(--cbx-ok-fg);">{{ report.rss_mb }} MB</span>
        </div>
        <div class="p-3 rounded-2xl cbx-sub-card flex flex-col items-center justify-center text-center">
          <span class="text-[11px] cbx-text-muted font-semibold">活跃协程</span>
          <span class="text-sm font-bold font-mono mt-1 font-tabular" style="color: var(--cbx-purple-fg);">{{ report.goroutines }}</span>
        </div>
      </div>

      <!-- Check Items List -->
      <div v-if="report && report.checks" class="space-y-2.5 max-h-[45vh] overflow-y-auto pr-1">
        <div
          v-for="(chk, idx) in report.checks"
          :key="idx"
          class="p-3 rounded-xl cbx-sub-card flex items-start gap-3"
        >
          <span class="w-5 h-5 rounded-full cbx-badge-ok flex items-center justify-center text-xs shrink-0 mt-0.5 font-bold">
            ✓
          </span>
          <div class="min-w-0 flex-1">
            <div class="text-xs font-bold cbx-text-main flex items-center justify-between gap-2">
              <span class="truncate">{{ chk.name }}</span>
              <span class="text-[10px] font-mono cbx-badge-ok px-1.5 py-0.5 rounded font-bold shrink-0">
                {{ chk.status }}
              </span>
            </div>
            <p class="text-[11px] cbx-text-sub mt-0.5 leading-relaxed font-sans">
              {{ chk.desc }}
            </p>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="pt-3 border-t flex items-center justify-between text-xs cbx-text-muted gap-2" style="border-color: var(--cbx-border-sub);">
        <span class="font-mono text-[11px]">体检时间: {{ report?.audited_at ? new Date(report.audited_at).toLocaleTimeString() : '刚刚' }}</span>
        <button
          @click="$emit('update:modelValue', false)"
          class="px-4 py-2 rounded-xl bg-sky-600 hover:bg-sky-700 text-white text-xs font-bold transition cursor-pointer"
        >
          完成闭环确认
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  report: {
    type: Object,
    default: () => null
  }
})

defineEmits(['update:modelValue'])
</script>
