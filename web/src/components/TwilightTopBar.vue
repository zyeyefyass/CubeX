<template>
  <header
    class="macos-topbar shrink-0 h-14 md:h-16 px-3 sm:px-5 md:px-6 flex items-center justify-between border-b border-white/[0.08] select-none z-30 transition-all duration-200"
    style="contain: layout style paint;"
  >
    <!-- 左侧：CubeX 品牌微光 -->
    <div class="flex items-center space-x-2 sm:space-x-3 shrink-0">
      <div class="flex items-center space-x-2 sm:space-x-2.5">
        <img
          src="/logo-cubex.svg"
          class="w-7 h-7 sm:w-8 sm:h-8 shrink-0 hover:scale-105 transition-transform duration-200 cursor-pointer"
          alt="CubeX Logo"
        />
        <span class="text-sm md:text-base font-bold tracking-wider text-[#0f172a] font-mono">
          CubeX
        </span>
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
            class="w-5.5 h-5.5 sm:w-6 sm:h-6 object-contain shrink-0  transition-transform duration-200 group-hover:scale-105"
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
            class="w-5.5 h-5.5 sm:w-6 sm:h-6 object-contain shrink-0  transition-transform duration-200 group-hover:scale-105"
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

    <!-- 右侧：主路由 Logo 与主路由固件版本号 + 👁️观景入口 -->
    <div class="flex items-center space-x-1.5 sm:space-x-2.5 md:space-x-3 shrink-0">
      <!-- 主路由 Logo 与固件版本 -->
      <div
        class="flex items-center space-x-1.5 px-3 py-1 rounded-full bg-white/70 hover:bg-white/90 border border-slate-300/60 shadow-sm text-xs select-none backdrop-blur-md transition-all cursor-default"
        :title="`主路由: ${routerName} ${routerVersion}`"
      >
        <img :src="routerLogo" class="w-4 h-4 object-contain shrink-0" :alt="routerName" />
        <span class="font-bold text-[#0f2744]">{{ routerName }}</span>
        <span class="text-slate-400 font-mono text-[10px]">·</span>
        <span class="font-mono text-[11px] font-extrabold text-cyan-800">{{ routerVersion }}</span>
      </div>

      <!-- 👁️ 观景模式切换按钮 (Alt+Z 亦可触发，可点击来回切换) -->
      <button
        @click.stop="$emit('toggle-peek')"
        class="w-8 h-8 sm:w-9 sm:h-9 rounded-full bg-[#f1f4f9]/90 hover:bg-[#e2e8f0] border border-slate-300/60 shadow-sm flex items-center justify-center text-slate-800 transition-all duration-200 hover:scale-105 active:scale-95 cursor-pointer select-none pointer-events-auto z-50"
        :title="isPeekMode ? '退出背景展示 (或按 ESC / Alt+Z)' : '隐藏面板（观景模式）· 快捷键 Alt+Z'"
      >
        <span class="text-sm select-none">👁️</span>
      </button>
    </div>
  </header>
</template>

<script setup>
defineProps({
  routerName: {
    type: String,
    default: 'RouterOS'
  },
  routerVersion: {
    type: String,
    default: 'v7.x'
  },
  routerLogo: {
    type: String,
    default: '/host-routeros-icon.svg'
  },
  appVersion: {
    type: String,
    default: 'v9.9.7'
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

defineEmits(['switch-engine', 'toggle-peek'])
</script>
