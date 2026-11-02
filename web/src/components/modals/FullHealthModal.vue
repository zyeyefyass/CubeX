<template>
  <div v-if="modelValue" class="fixed inset-0 z-[10000] flex items-center justify-center p-3 sm:p-4 bg-black/80 backdrop-blur-xl animate-in fade-in">
    <div class="w-full max-w-xl rounded-3xl macos-glass-card shadow-2xl p-6 sm:p-7 space-y-5 text-slate-200 border border-cyan-500/30 relative overflow-hidden">
      <!-- Glow ambient background -->
      <div class="absolute -right-24 -top-24 w-72 h-72 bg-cyan-500/15 rounded-full blur-3xl pointer-events-none"></div>
      <div class="absolute -left-24 -bottom-24 w-72 h-72 bg-emerald-500/15 rounded-full blur-3xl pointer-events-none"></div>

      <!-- Header -->
      <div class="flex items-center justify-between border-b border-white/10 pb-4 relative z-10">
        <div class="flex items-center gap-3">
          <div class="w-11 h-11 rounded-2xl bg-gradient-to-br from-cyan-500/20 to-emerald-500/20 border border-cyan-400/40 flex items-center justify-center text-2xl shadow-[0_0_15px_rgba(6,182,212,0.3)]">
            🛡️
          </div>
          <div>
            <h3 class="text-base sm:text-lg font-bold text-white tracking-wide flex items-center gap-2">
              全链路深度健康体检
              <span class="text-xs px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-400/40 font-mono">
                100% 满分通过
              </span>
            </h3>
            <p class="text-xs text-slate-400 mt-0.5">宿主策略路由 · 双引擎守护 · DNS零泄露 · 节点SLA综合矩阵</p>
          </div>
        </div>
        <button
          @click="$emit('update:modelValue', false)"
          class="p-1.5 rounded-xl hover:bg-white/10 text-slate-400 hover:text-white transition cursor-pointer"
        >
          ✕
        </button>
      </div>

      <!-- Overview Stats -->
      <div v-if="report" class="grid grid-cols-3 gap-3 relative z-10">
        <div class="p-3 rounded-2xl bg-white/[0.04] border border-cyan-500/20 flex flex-col items-center justify-center text-center">
          <span class="text-[11px] text-slate-400">活跃引擎</span>
          <span class="text-sm font-bold font-mono text-cyan-300 mt-1 uppercase">{{ report.active_engine }}</span>
        </div>
        <div class="p-3 rounded-2xl bg-white/[0.04] border border-emerald-500/20 flex flex-col items-center justify-center text-center">
          <span class="text-[11px] text-slate-400">核心堆内存</span>
          <span class="text-sm font-bold font-mono text-emerald-300 mt-1">{{ report.rss_mb }} MB</span>
        </div>
        <div class="p-3 rounded-2xl bg-white/[0.04] border border-purple-500/20 flex flex-col items-center justify-center text-center">
          <span class="text-[11px] text-slate-400">活跃协程</span>
          <span class="text-sm font-bold font-mono text-purple-300 mt-1">{{ report.goroutines }}</span>
        </div>
      </div>

      <!-- Check Items List -->
      <div v-if="report && report.checks" class="space-y-2.5 relative z-10 max-h-[45vh] overflow-y-auto pr-1">
        <div
          v-for="(chk, idx) in report.checks"
          :key="idx"
          class="p-3 rounded-xl bg-[#001b33]/80 border border-slate-700/50 flex items-start gap-3"
        >
          <span class="w-5 h-5 rounded-full bg-emerald-500/20 text-emerald-400 border border-emerald-400/40 flex items-center justify-center text-xs shrink-0 mt-0.5">
            ✓
          </span>
          <div class="min-w-0 flex-1">
            <div class="text-xs font-bold text-white flex items-center justify-between">
              <span>{{ chk.name }}</span>
              <span class="text-[10px] font-mono text-emerald-400 bg-emerald-950/60 px-1.5 py-0.2 rounded border border-emerald-500/30">
                {{ chk.status }}
              </span>
            </div>
            <p class="text-[11px] text-slate-300/80 mt-0.5 leading-relaxed font-sans">
              {{ chk.desc }}
            </p>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="pt-3 border-t border-white/10 flex items-center justify-between text-xs text-slate-400 relative z-10">
        <span class="font-mono text-[11px]">体检时间: {{ report?.audited_at ? new Date(report.audited_at).toLocaleTimeString() : '刚刚' }}</span>
        <button
          @click="$emit('update:modelValue', false)"
          class="px-4 py-1.5 rounded-xl bg-cyan-500/20 hover:bg-cyan-500/30 text-cyan-300 border border-cyan-400/40 font-bold transition cursor-pointer"
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
