<template>
  <header
    class="shrink-0 h-14 md:h-16 px-3 sm:px-5 md:px-6 flex items-center justify-between border-b border-white/[0.08] bg-[#001529]/80 backdrop-blur-md shadow-[inset_0_1px_1px_rgba(255,255,255,0.12),0_4px_24px_rgba(0,0,0,0.35)] select-none z-30 transition-all duration-200"
    style="contain: layout style paint;"
  >
    <!-- 左侧：CubeX 动态版本、品牌微光与活跃宿主底座 -->
    <div class="flex items-center space-x-2 sm:space-x-3 shrink-0">
      <div class="flex items-center space-x-1.5 sm:space-x-2">
        <span class="text-sm md:text-base font-bold tracking-wider text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 via-teal-300 to-indigo-300 font-mono">
          CubeX
        </span>
        <!-- 动态版本号，无缝绑定后端 meta.version -->
        <span class="px-2 py-0.5 text-[11px] font-mono font-medium rounded-full bg-cyan-950/60 text-cyan-300 border border-cyan-500/25 shadow-inner">
          {{ appVersion || 'v9.9.4' }}
        </span>
      </div>

      <!-- 当前接入宿主原生底座徽章 (仅展示真实活跃底座，不摆未激活灰标) -->
      <div v-if="activeHostBadge" class="hidden md:flex items-center space-x-1.5 px-2.5 py-0.5 rounded-full bg-white/[0.05] border border-white/10 text-[11px] text-slate-300 shadow-inner">
        <span class="w-1.5 h-1.5 rounded-full bg-cyan-400 shadow-[0_0_6px_#00d2ff]"></span>
        <span class="font-medium">{{ activeHostBadge }}</span>
      </div>
    </div>

    <!-- 中间：极致留白，通透展现动物城全景与暮光冰晶背景微光 (彻底剔除吃灰假需求大药丸) -->
    <div class="flex-1"></div>

    <!-- 右侧：双核热切胶囊 + 大白话数据面脉冲 (单点呼出诊断) + 赏景入口 (彻底去除重复齿轮图标) -->
    <div class="flex items-center space-x-1.5 sm:space-x-2.5 md:space-x-3 shrink-0">
      <!-- 满血双核秒级热切胶囊 (Windhawk 药丸风格) -->
      <div class="hidden sm:flex items-center p-0.5 rounded-full bg-black/40 border border-white/10 shadow-inner">
        <button
          @click="$emit('switch-engine', 'singbox')"
          :disabled="isSwitchingEngine"
          :class="[
            'px-2.5 py-0.5 rounded-full text-xs font-mono transition-all flex items-center space-x-1 cursor-pointer select-none',
            activeEngine === 'singbox'
              ? 'bg-gradient-to-r from-sky-500 to-cyan-500 text-white font-bold shadow-[0_0_10px_rgba(0,210,255,0.4)]'
              : 'text-slate-400 hover:text-slate-200'
          ]"
          title="切换至 sing-box v1.14+ 极速内核"
        >
          <img src="/logo-singbox.svg" class="w-3.5 h-3.5 object-contain" alt="sing-box" />
          <span>sing-box</span>
        </button>
        <button
          @click="$emit('switch-engine', 'mihomo')"
          :disabled="isSwitchingEngine"
          :class="[
            'px-2.5 py-0.5 rounded-full text-xs font-mono transition-all flex items-center space-x-1 cursor-pointer select-none',
            activeEngine === 'mihomo'
              ? 'bg-gradient-to-r from-sky-500 to-cyan-500 text-white font-bold shadow-[0_0_10px_rgba(0,210,255,0.4)]'
              : 'text-slate-400 hover:text-slate-200'
          ]"
          title="切换至 Mihomo 规则完备内核"
        >
          <img src="/logo-mihomo.svg" class="w-3.5 h-3.5 object-contain" alt="Mihomo" />
          <span>Mihomo</span>
        </button>
      </div>

      <!-- 热切中微脉冲动画 -->
      <span v-if="isSwitchingEngine" class="text-[11px] font-mono text-cyan-300 animate-pulse hidden lg:inline">
        🔄 切换中...
      </span>

      <!-- 大白话三态健康脉冲胶囊 (唯一且纯净的排障诊断抽屉入口，彻底杜绝重复按钮) -->
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

      <!-- 🐰🦊 赏景模式切换按钮 (Alt+Z 亦可触发) -->
      <button
        @click="$emit('toggle-peek')"
        class="flex p-1.5 rounded-full bg-white/[0.05] hover:bg-white/[0.1] border border-white/10 text-slate-300 hover:text-amber-300 transition-colors cursor-pointer select-none"
        :title="isPeekMode ? '退出背景展示 (或按 ESC)' : '隐藏面板（查看背景）· 快捷键 Alt+Z'"
      >
        <span class="text-xs">🐰🦊</span>
      </button>
    </div>
  </header>
</template>

<script setup>
defineProps({
  appVersion: {
    type: String,
    default: 'v9.9.4'
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

defineEmits(['open-diag', 'switch-engine', 'toggle-peek'])
</script>
