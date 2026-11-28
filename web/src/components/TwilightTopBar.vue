<template>
  <header
    class="macos-topbar shrink-0 h-14 md:h-16 px-2.5 sm:px-5 md:px-6 flex items-center justify-between gap-1.5 sm:gap-3 select-none z-30 transition-all duration-200"
    style="contain: layout style paint;"
  >
    <!-- 左侧：CubeX 品牌微光 -->
    <div class="flex items-center gap-1.5 sm:gap-2.5 shrink-0 min-w-0">
      <img
        src="/logo-cubex.svg"
        class="w-7 h-7 sm:w-8 sm:h-8 shrink-0 hover:scale-105 transition-transform duration-200 cursor-pointer"
        alt="CubeX Logo"
      />
      <span class="text-sm md:text-base font-bold tracking-wider cbx-text-main font-mono hidden min-[360px]:inline">
        CubeX
      </span>
    </div>

    <!-- 中间：双核热切胶囊 (全端常驻 · 手机端紧凑 SB / MH，桌面端完整展示) -->
    <div class="flex-1 flex items-center justify-center px-1 sm:px-2 min-w-0">
      <div
        id="cubex-topbar-engine-switch"
        class="flex items-center p-0.5 sm:p-1 rounded-full macos-capsule shadow-sm max-w-full"
      >
        <button
          @click="$emit('switch-engine', 'singbox')"
          :disabled="isSwitchingEngine"
          :class="[
            'min-h-[34px] sm:min-h-[36px] px-2.5 sm:px-4 py-1 rounded-full text-[11px] sm:text-[13px] font-mono transition-all flex items-center gap-1.5 sm:gap-2 cursor-pointer select-none font-bold shrink-0',
            activeEngine === 'singbox' ? 'active-engine shadow-sm' : ''
          ]"
          :title="coreVersions?.sing_box ? `切换至 sing-box ${coreVersions.sing_box} 极速内核` : '切换至 sing-box 极速内核'"
        >
          <img
            src="/logo-singbox.svg"
            class="w-4 h-4 sm:w-5 sm:h-5 object-contain shrink-0"
            alt="sing-box"
          />
          <span class="sm:hidden">SB</span>
          <span class="hidden sm:inline">sing-box</span>
        </button>
        <button
          @click="$emit('switch-engine', 'mihomo')"
          :disabled="isSwitchingEngine"
          :class="[
            'min-h-[34px] sm:min-h-[36px] px-2.5 sm:px-4 py-1 rounded-full text-[11px] sm:text-[13px] font-mono transition-all flex items-center gap-1.5 sm:gap-2 cursor-pointer select-none font-bold shrink-0',
            activeEngine === 'mihomo' ? 'active-engine shadow-sm' : ''
          ]"
          :title="coreVersions?.mihomo ? `切换至 Mihomo ${coreVersions.mihomo} 规则完备内核` : '切换至 Mihomo 规则完备内核'"
        >
          <img
            src="/logo-mihomo.svg"
            class="w-4 h-4 sm:w-5 sm:h-5 object-contain shrink-0"
            alt="Mihomo"
          />
          <span class="sm:hidden">MH</span>
          <span class="hidden sm:inline">Mihomo</span>
        </button>
      </div>

      <!-- 热切中微脉冲提示 -->
      <span
        v-if="isSwitchingEngine"
        class="text-[11px] font-mono font-bold animate-pulse ml-2 hidden lg:inline"
        style="color: var(--cbx-accent-fg);"
      >
        🔄 切换中...
      </span>
    </div>

    <!-- 右侧：主路由胶囊 + 主题切换 + 观景模式 -->
    <div class="flex items-center gap-1.5 sm:gap-2.5 shrink-0">
      <!-- 主路由 Logo 与固件版本 (<420px 自动折叠主路由长名称为图标+短版本号) -->
      <div
        class="flex items-center gap-1 sm:gap-1.5 px-2 sm:px-3 py-1 rounded-full macos-capsule text-xs select-none transition-all cursor-default max-w-[115px] sm:max-w-none"
        :title="`主路由: ${routerName} ${routerVersion}`"
      >
        <img :src="routerLogo" class="w-4 h-4 object-contain shrink-0" :alt="routerName" />
        <span class="font-bold cbx-text-main hidden min-[420px]:inline truncate">{{ routerName }}</span>
        <span class="cbx-text-muted font-mono text-[10px] hidden min-[420px]:inline">·</span>
        <span class="font-mono text-[11px] font-extrabold truncate" style="color: var(--cbx-accent-fg);">{{ routerVersion }}</span>
      </div>

      <!-- 🌗 深色暮光 / 柔光浅色 一键切换 -->
      <button
        @click.stop="$emit('toggle-color-mode')"
        class="w-9 h-9 rounded-full macos-capsule flex items-center justify-center transition-transform duration-200 hover:scale-105 active:scale-95 cursor-pointer select-none shrink-0 cbx-touch-extend"
        :aria-label="colorMode === 'dark' ? '切换为柔光浅色模式' : '切换为深色暮光模式'"
        :title="colorMode === 'dark' ? '当前：深色暮光 · 点击切换为柔光浅色' : '当前：柔光浅色 · 点击切换为深色暮光'"
      >
        <span class="text-sm select-none" aria-hidden="true">{{ colorMode === 'dark' ? '☀️' : '🌙' }}</span>
      </button>

      <!-- 👁️ 观景模式切换按钮 -->
      <button
        @click.stop="$emit('toggle-peek')"
        class="w-9 h-9 rounded-full macos-capsule flex items-center justify-center transition-transform duration-200 hover:scale-105 active:scale-95 cursor-pointer select-none pointer-events-auto z-50 shrink-0 cbx-touch-extend"
        :aria-label="isPeekMode ? '退出背景展示' : '隐藏面板进入观景模式'"
        :title="isPeekMode ? '退出背景展示 (或按 ESC / Alt+Z)' : '隐藏面板（观景模式）· 快捷键 Alt+Z'"
      >
        <span class="text-sm select-none" aria-hidden="true">👁️</span>
      </button>
    </div>
  </header>
</template>

<script setup>
defineProps({
  routerName: {
    type: String,
    default: '自适应网关'
  },
  routerVersion: {
    type: String,
    default: '--'
  },
  routerLogo: {
    type: String,
    default: '/logo-cubex.svg'
  },
  appVersion: {
    type: String,
    default: '--'
  },
  coreVersions: {
    type: Object,
    default: () => ({})
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
  },
  colorMode: {
    type: String,
    default: 'dark'
  }
})

defineEmits(['switch-engine', 'toggle-peek', 'toggle-color-mode', 'open-logs'])
</script>
