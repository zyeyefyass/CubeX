<template>
  <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-md">
    <div class="bg-[#001b33] border border-cyan-500/30 rounded-2xl w-full max-w-4xl max-h-[85vh] flex flex-col shadow-2xl overflow-hidden">
      <!-- Header -->
      <div class="px-5 py-4 bg-[#001529]/80 border-b border-cyan-500/20 flex items-center justify-between">
        <div class="flex items-center space-x-3">
          <div class="w-8 h-8 rounded-lg bg-cyan-500/10 border border-cyan-400/30 flex items-center justify-center text-cyan-300">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h7" />
            </svg>
          </div>
          <div>
            <h3 class="text-white font-semibold text-sm">CubeX 全栈实时日志流</h3>
            <p class="text-[11px] text-slate-400">双引擎内核 / 控制面守护进程实时输出</p>
          </div>
        </div>

        <div class="flex items-center space-x-2">
          <button
            @click="clearLogs"
            class="px-2.5 py-1 text-xs rounded bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 transition-colors"
          >
            清空面板
          </button>
          <button
            @click="$emit('close')"
            class="w-7 h-7 rounded-lg bg-slate-800 hover:bg-rose-500/20 hover:text-rose-400 text-slate-400 flex items-center justify-center transition-colors"
          >
            ✕
          </button>
        </div>
      </div>

      <!-- Logs Content Area -->
      <div ref="logContainer" class="p-4 flex-1 overflow-y-auto font-mono text-xs bg-[#000d1a] space-y-1 select-text">
        <div v-if="logs.length === 0" class="text-slate-500 text-center py-10">
          等待日志数据接入...
        </div>
        <div
          v-for="(log, idx) in logs"
          :key="idx"
          class="leading-relaxed hover:bg-white/5 px-2 py-0.5 rounded transition-colors"
          :class="getLogLineClass(log)"
        >
          {{ log }}
        </div>
      </div>

      <!-- Footer Bar -->
      <div class="px-5 py-2.5 bg-[#001529]/80 border-t border-cyan-500/10 flex items-center justify-between text-xs text-slate-400">
        <div class="flex items-center space-x-2">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-ping"></span>
          <span>实时日志管道已就绪</span>
        </div>
        <div>
          共 {{ logs.length }} 条记录
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

defineProps({
  show: {
    type: Boolean,
    default: false
  },
  logs: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['close', 'clear'])

const clearLogs = () => {
  emit('clear')
}

const getLogLineClass = (line) => {
  if (line.includes('ERROR') || line.includes('error') || line.includes('panic') || line.includes('Fatal')) {
    return 'text-rose-400'
  }
  if (line.includes('WARN') || line.includes('warning')) {
    return 'text-amber-300'
  }
  if (line.includes('INFO') || line.includes('info')) {
    return 'text-cyan-300/90'
  }
  return 'text-slate-300'
}
</script>
