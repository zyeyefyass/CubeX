<template>
  <header
    class="macos-topbar shrink-0 h-14 md:h-16 px-3 sm:px-5 md:px-6 flex items-center justify-between border-b border-white/[0.08] select-none z-30 transition-all duration-200"
    style="contain: layout style paint;"
  >
    <!-- 左侧：CubeX 动态版本、品牌微光与活跃宿主底座 -->
    <div class="flex items-center space-x-2 sm:space-x-3 shrink-0">
      <div class="flex items-center space-x-2 sm:space-x-2.5">
        <img
          src="/logo-cubex.svg"
          class="w-7 h-7 sm:w-8 sm:h-8 shrink-0 drop-shadow-[0_0_10px_rgba(56,189,248,0.55)] hover:scale-105 transition-transform duration-200 cursor-pointer"
          alt="CubeX Logo"
        />
        <span class="text-sm md:text-base font-bold tracking-wider text-[#0f172a] font-mono">
          CubeX
        </span>
        <!-- 动态版本号，无缝绑定后端 meta.version -->
        <span class="px-2 py-0.5 text-[11px] font-mono font-bold rounded-full bg-white/70 text-sky-700 border border-white/80 shadow-sm">
          {{ appVersion || 'v9.9.5' }}
        </span>
      </div>

      <!-- 当前接入宿主原生底座徽章 (仅展示真实活跃底座，不摆未激活灰标) -->
      <div v-if="activeHostBadge" class="hidden md:flex items-center space-x-1.5 px-2.5 py-0.5 rounded-full bg-white/60 border border-white/80 text-[11px] text-slate-700 shadow-sm">
        <span class="w-1.5 h-1.5 rounded-full bg-sky-500 shadow-[0_0_6px_rgba(2,132,199,0.8)]"></span>
        <span class="font-bold">{{ activeHostBadge }}</span>
      </div>
    </div>

    <!-- 中间：双核热切胶囊 (严格居中显示 · 饱满行高 · 高对比度清晰排版) -->
    <div class="flex-1 flex items-center justify-center px-2">
      <div id="cubex-topbar-engine-switch" class="hidden sm:flex items-center p-1 rounded-full bg-white/70 border border-white/90 shadow-sm backdrop-blur-md">
        <button
          @click="$emit('switch-engine', 'singbox')"
          :disabled="isSwitchingEngine"
          :class="[
            'px-4 py-1.5 rounded-full text-xs sm:text-[13px] font-mono transition-all flex items-center gap-2 cursor-pointer select-none font-bold',
            activeEngine === 'singbox'
              ? 'active-engine bg-[#0f172a] text-white shadow-md'
              : 'text-slate-700 hover:text-slate-950 hover:bg-white/50'
          ]"
          title="切换至 sing-box v1.14+ 极速内核"
        >
          <img
            src="/logo-singbox.svg"
            class="w-5.5 h-5.5 sm:w-6 sm:h-6 object-contain shrink-0 drop-shadow-[0_0_6px_rgba(232,121,249,0.7)] transition-transform duration-200 group-hover:scale-105"
            alt="sing-box"
          />
          <span>sing-box</span>
        </button>
        <button
          @click="$emit('switch-engine', 'mihomo')"
          :disabled="isSwitchingEngine"
          :class="[
            'px-4 py-1.5 rounded-full text-xs sm:text-[13px] font-mono transition-all flex items-center gap-2 cursor-pointer select-none font-bold',
            activeEngine === 'mihomo'
              ? 'active-engine bg-[#0f172a] text-white shadow-md'
              : 'text-slate-700 hover:text-slate-950 hover:bg-white/50'
          ]"
          title="切换至 Mihomo 规则完备内核"
        >
          <img
            src="/logo-mihomo.svg"
            class="w-5.5 h-5.5 sm:w-6 sm:h-6 object-contain shrink-0 drop-shadow-[0_0_6px_rgba(56,189,248,0.8)] transition-transform duration-200 group-hover:scale-105"
            alt="Mihomo"
          />
          <span>Mihomo</span>
        </button>
      </div>

      <!-- 热切中微脉冲动画 -->
      <span v-if="isSwitchingEngine" class="text-[11px] font-mono text-cyan-700 font-bold animate-pulse ml-2 hidden lg:inline">
        🔄 切换中...
      </span>
    </div>

    <!-- 右侧：大白话数据面脉冲 (单点呼出诊断) + 🎨主题中心 + 👁️赏景入口 -->
    <div class="flex items-center space-x-1.5 sm:space-x-2.5 md:space-x-3 shrink-0">
      <button
        @click="$emit('open-diag')"
        :class="[
          'group flex items-center space-x-1.5 sm:space-x-2 px-2.5 sm:px-3 py-1 rounded-full border text-xs font-medium transition-all duration-200 cursor-pointer shadow-sm hover:scale-[1.02] active:scale-[0.98]',
          pulseState.badgeClass
        ]"
        title="点击查看数据面实时指标与诊断卡"
      >
        <span :class="['w-2 h-2 rounded-full transition-transform', pulseState.dotClass]"></span>
        <span class="font-sans tracking-tight hidden xs:inline">{{ pulseState.label }}</span>
        <!-- 毫秒探针延迟 (等宽数字排版，杜绝抖动) -->
        <span v-if="probeDurationMs > 0" class="font-mono text-[11px] opacity-85 tabular-nums">
          {{ probeDurationMs }}ms
        </span>
      </button>

      <!-- 👁️ 赏景模式切换按钮 (Alt+Z 亦可触发，可点击来回切换) -->
      <button
        @click.stop="$emit('toggle-peek')"
        class="w-8 h-8 sm:w-9 sm:h-9 rounded-full bg-white/85 hover:bg-white border border-white/90 shadow-md flex items-center justify-center text-slate-800 transition-all duration-200 hover:scale-105 active:scale-95 cursor-pointer select-none pointer-events-auto z-50"
        :title="isPeekMode ? '退出背景展示 (或按 ESC / Alt+Z)' : '隐藏面板（查看背景）· 快捷键 Alt+Z'"
      >
        <span class="text-sm select-none">👁️</span>
      </button>
    </div>
  </header>
</template>

<script setup>
defineProps({
  appVersion: {
    type: String,
    default: 'v9.9.5'
  },
  activeHostBadge: {
    type: String,
    default: 'RouterOS v7'
  },
  searchQuery: {
    type: String,
    default: ''
  },
  pulseState: {
    type: Object,
    default: () => ({
      type: 'healthy',
      label: '数据面健康',
      dotClass: 'bg-emerald-400 shadow-[0_0_8px_#10b981]',
      badgeClass: 'bg-emerald-950/40 text-emerald-300 border-emerald-500/30',
      textClass: 'text-emerald-300'
    })
  },
  probeDurationMs: {
    type: Number,
    default: 1
  },
  activeEngine: {
    type: String,
    default: 'sing-box'
  },
  isSwitchingEngine: {
    type: Boolean,
    default: false
  },
  isPeekMode: {
    type: Boolean,
    default: false
  }
})

defineEmits(['open-diag', 'switch-engine', 'toggle-peek', 'open-theme'])
</script>
