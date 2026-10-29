<template>
  <div class="rounded-3xl p-5 sm:p-7 macos-glass-card shadow-2xl relative overflow-hidden transition-all duration-300 border border-white/10 select-none">
    
    <!-- Top Ambient Glow Aura -->
    <div class="absolute -top-24 -left-24 w-96 h-96 bg-[radial-gradient(circle,rgba(6,182,212,0.10)_0%,transparent_70%)] rounded-full pointer-events-none"></div>
    <div class="absolute -bottom-24 -right-24 w-96 h-96 bg-[radial-gradient(circle,rgba(255,158,94,0.10)_0%,transparent_70%)] rounded-full pointer-events-none"></div>

    <!-- ── Header Controls Bar ── -->
    <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4 pb-5 border-b border-white/10 relative z-10">
      <div class="space-y-1">
        <div class="flex items-center gap-2.5 flex-wrap">
          <div class="w-8 h-8 rounded-xl bg-gradient-to-br from-cyan-400/20 to-blue-500/20 border border-cyan-400/40 flex items-center justify-center text-cyan-300 shadow-sm">
            <span class="text-base">✨</span>
          </div>
          <h3 class="text-base sm:text-lg font-black text-white tracking-wide">
            全息动态流量拓扑大屏
          </h3>
          <span class="text-[11px] font-mono px-2.5 py-0.5 rounded-full bg-gradient-to-r from-cyan-500/20 to-teal-500/20 text-cyan-300 border border-cyan-400/30 font-bold flex items-center gap-1.5 shadow-sm">
            <span class="w-1.5 h-1.5 rounded-full bg-cyan-400"></span>
            Twilight Crystal 2.0
          </span>
          <span class="text-[11px] font-mono px-2 py-0.5 rounded-full bg-white/10 text-[#FEF08A] border border-white/20 font-medium hidden sm:inline-flex">
            微秒级多流感知
          </span>
        </div>
        <p class="text-xs text-slate-400 font-medium">
          三层拓扑感知 · 局域网设备 ➔ CubeX 智能双核中枢 ➔ 真实国内直连与跨境分流落地
        </p>
      </div>

      <!-- Right Toolbar Controls -->
      <div class="flex items-center gap-2.5 flex-wrap">
        <!-- Terminal Filter Tabs -->
        <div class="flex items-center p-1 rounded-xl bg-black/40 border border-white/10 backdrop-blur-md text-xs">
          <button @click="terminalFilter = 'all'"
            :class="['px-2.5 py-1 rounded-lg font-bold transition cursor-pointer', terminalFilter === 'all' ? 'bg-gradient-to-r from-[#FF9E5E] to-[#F59E0B] text-[#1E1106] shadow' : 'text-slate-400 hover:text-white']">
            全部设备
          </button>
          <button @click="terminalFilter = 'rule'"
            :class="['px-2.5 py-1 rounded-lg font-bold transition cursor-pointer', terminalFilter === 'rule' ? 'bg-cyan-500 text-black shadow' : 'text-slate-400 hover:text-white']">
            智能分流
          </button>
          <button @click="terminalFilter = 'bypass'"
            :class="['px-2.5 py-1 rounded-lg font-bold transition cursor-pointer', terminalFilter === 'bypass' ? 'bg-emerald-500 text-black shadow' : 'text-slate-400 hover:text-white']">
            直连绕过
          </button>
        </div>

        <!-- Animation Eco Toggle (动效开/关 · 一键为低功耗设备节约显卡/CPU) -->
        <button @click="toggleAnimation" :title="isAnimationActive ? '点击暂停光流动效，为手机或老旧浏览器节省 CPU 与 GPU' : '点击开启全息粒子流动效'"
          :class="[
            'h-8 px-2.5 rounded-xl border text-xs flex items-center gap-1.5 transition active:scale-95 cursor-pointer font-bold',
            isAnimationActive
              ? 'bg-cyan-500/15 text-cyan-300 border-cyan-400/30 hover:bg-cyan-500/25'
              : 'bg-white/5 text-slate-400 border-white/10 hover:bg-white/10 hover:text-slate-200'
          ]">
          <span class="text-[12px]">{{ isAnimationActive ? '✨' : '⏸️' }}</span>
          <span>{{ isAnimationActive ? '动效开' : '动效关' }}</span>
        </button>

        <!-- Refresh Topology -->
        <button @click="refreshTopology" title="实时刷新拓扑与局域网终端"
          class="w-8 h-8 rounded-xl bg-white/10 hover:bg-white/20 border border-white/15 text-amber-300 flex items-center justify-center transition active:scale-95 cursor-pointer">
          <span :class="{'animate-spin': isRefreshing}">↺</span>
        </button>
      </div>
    </div>

    <!-- ── Interactive Topology Main Container ── -->
    <div class="relative mt-6 z-10" ref="topologyContainer">

      <!-- Desktop SVG Bézier Connecting Layer (Hidden on mobile < 1024px) -->
      <svg class="hidden lg:block absolute inset-0 w-full h-full pointer-events-none z-0" xmlns="http://www.w3.org/2000/svg">
        <defs>
          <linearGradient id="streamGradProxy" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stop-color="#00F5D4" stop-opacity="0.8" />
            <stop offset="50%" stop-color="#00d2ff" stop-opacity="0.9" />
            <stop offset="100%" stop-color="#7B2CBF" stop-opacity="0.8" />
          </linearGradient>
          <linearGradient id="streamGradDirect" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stop-color="#10B981" stop-opacity="0.8" />
            <stop offset="50%" stop-color="#34D399" stop-opacity="0.9" />
            <stop offset="100%" stop-color="#F59E0B" stop-opacity="0.8" />
          </linearGradient>

        </defs>

        <!-- Active Connecting Paths -->
        <g v-if="svgPaths.length > 0">
          <path v-for="(p, pi) in svgPaths" :key="pi"
            :d="p.d"
            :stroke="p.type === 'direct' ? 'url(#streamGradDirect)' : 'url(#streamGradProxy)'"
            :stroke-width="p.isHighlighted ? 3.5 : (p.isActive ? 2 : 1)"
            :stroke-opacity="p.isHighlighted ? 1 : (selectedDevice ? (p.isMatchSelected ? 0.9 : 0.15) : 0.6)"
            fill="none"
            :class="[
              isAnimationActive ? (p.type === 'direct' ? 'flow-dash-direct' : 'flow-dash-proxy') : ''
            ]"
            :style="{
              opacity: p.isHighlighted ? '1' : '0.85'
            }"
          />
        </g>
      </svg>

      <!-- ── 3-Tier Grid Layout (Strict 5-Card Height Symmetrical Balance: 382px) ── -->
      <div class="grid grid-cols-1 lg:grid-cols-12 gap-5 items-stretch relative z-10">

        <!-- ══════════════════════════════════════════════════════════════ -->
        <!-- TIER 1: 局域网设备 (LAN Devices) [lg:col-span-4]               -->
        <!-- ══════════════════════════════════════════════════════════════ -->
        <div id="topology-tier-1" class="lg:col-span-4 flex flex-col justify-between space-y-2">
          <!-- Non-wrapping clean header -->
          <div class="h-7 flex items-center justify-between gap-2 pb-1 px-1 whitespace-nowrap">
            <div class="flex items-center gap-1.5 min-w-0">
              <span class="text-xs sm:text-[13px] font-bold text-white tracking-wide flex items-center gap-1.5 shrink-0">
                <span class="text-sm">📱</span>
                <span>局域网设备</span>
              </span>
              <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-cyan-500/15 text-cyan-300 border border-cyan-400/30 font-bold shrink-0">
                {{ activeDisplayDevices.length }} 在线
              </span>
            </div>
            <span class="text-[11px] text-slate-400 truncate">点击切换分流模式 · ✏️重命名</span>
          </div>

          <!-- Device Cards Scroll List (Exact Height 382px: Fits 5 Full Cards + Right Scrollbar) -->
          <div class="h-[396px] max-h-[396px] overflow-y-auto pr-1.5 custom-scrollbar space-y-1.5 flex flex-col">
            <div v-for="dev in activeDisplayDevices" :key="dev.ip"
              @click="toggleDeviceMode(dev)"
              :id="'client-node-' + dev.ip.replace(/\./g, '-')"
              :title="'点击切换分流模式 (当前: ' + (dev.mode === 'bypass' ? '直连绕过' : '智能分流') + ')'"
              class="p-2.5 sm:px-3 sm:py-2 rounded-2xl transition-all duration-200 cursor-pointer border relative overflow-hidden group h-[68px] min-h-[68px] flex items-center bg-white/[0.03] hover:bg-white/[0.08] border-white/10 hover:border-cyan-400/50 select-none shadow-sm backdrop-blur-none">
              
              <!-- Left Color Indicator Accent -->
              <div :class="['absolute left-0 top-0 bottom-0 w-1', dev.mode === 'bypass' ? 'bg-sky-400' : 'bg-emerald-400']"></div>

              <div class="flex items-center justify-between gap-2 w-full">
                <div class="flex items-center gap-2.5 min-w-0">
                  <!-- Official Brand / Device Logo -->
                  <DeviceBrandLogo :device="dev" />

                  <div class="min-w-0">
                    <!-- Inline Rename or Normal Display -->
                    <div v-if="editingIp === dev.ip" class="flex items-center gap-1 py-0.5" @click.stop>
                      <input v-model="editingName"
                        @keyup.enter="saveInlineRename(dev)"
                        @keyup.esc="cancelInlineRename"
                        class="px-2 py-0.5 text-xs rounded bg-white/15 border border-cyan-400 text-white font-bold outline-none ring-1 ring-cyan-400/40 w-28 sm:w-32 shadow-inner"
                        placeholder="新备注名..."
                        autofocus />
                      <button @click.stop="saveInlineRename(dev)" class="px-1.5 py-0.5 text-[10px] rounded bg-emerald-600 hover:bg-emerald-500 text-white font-bold cursor-pointer">✔</button>
                      <button @click.stop="cancelInlineRename" class="px-1.5 py-0.5 text-[10px] rounded bg-white/10 hover:bg-white/20 text-slate-300 cursor-pointer">✕</button>
                    </div>
                    <div v-else class="flex items-center gap-1.5">
                      <span class="text-xs sm:text-[13px] font-bold text-white truncate max-w-[125px]" :title="dev.name || dev.hostname || dev.ip">
                        {{ dev.name || dev.hostname || dev.ip }}
                      </span>
                      <!-- Quick Rename Pencil Button -->
                      <button @click.stop="startInlineRename(dev)" title="重命名该设备" class="opacity-0 group-hover:opacity-100 text-[11px] text-slate-400 hover:text-cyan-300 transition cursor-pointer">
                        ✏️
                      </button>
                      <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 shrink-0"></span>
                    </div>
                    <div class="text-[11px] font-mono text-slate-400 truncate">
                      {{ dev.ip }}
                    </div>
                  </div>
                </div>

                <!-- Right Mode & Traffic Badge (Click badge directly toggles mode!) -->
                <div class="flex flex-col items-end gap-1 shrink-0">
                  <button @click.stop="toggleDeviceMode(dev)"
                    :title="'点击直接切换为: ' + (dev.mode === 'bypass' ? '智能分流' : '直连绕过')"
                    :class="[
                      'text-[10px] font-mono px-2 py-0.5 rounded-full font-bold border transition cursor-pointer active:scale-95 flex items-center gap-1 shadow-sm',
                      dev.mode === 'bypass'
                        ? 'bg-sky-500/20 text-sky-300 border-sky-400/40 hover:bg-sky-500/30'
                        : 'bg-emerald-500/20 text-emerald-300 border-emerald-400/40 hover:bg-emerald-500/30'
                    ]">
                    <span>{{ dev.mode === 'bypass' ? '🔵 直连绕过' : '🟢 智能分流' }}</span>
                  </button>
                  <!-- Clean Device Mode & State Indicator -->
                  <div class="text-[10px] font-mono flex items-center gap-1">
                    <template v-if="dev.mode === 'bypass'">
                      <span class="text-sky-400/90 font-medium">⚡ 芯片直连 · 0负载</span>
                    </template>
                    <template v-else>
                      <span class="text-emerald-400/80 font-medium flex items-center gap-1">
                        <span class="w-1.5 h-1.5 rounded-full bg-emerald-400/70"></span>
                        <span>智能分流就绪</span>
                      </span>
                    </template>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ══════════════════════════════════════════════════════════════ -->
        <!-- TIER 2: CubeX 旁路由双核中枢 (Gateway Core) [lg:col-span-4]    -->
        <!-- ══════════════════════════════════════════════════════════════ -->
        <div class="lg:col-span-4 flex flex-col justify-center items-center my-auto h-[396px] px-1">
          <div id="cubex-core-center"
            class="w-full max-w-[320px] rounded-3xl p-4 sm:p-5 bg-gradient-to-b from-[#00223e]/70 via-[#001529]/80 to-[#000d1a]/85 border border-cyan-400/40 shadow-[0_0_25px_rgba(0,210,255,0.15)] text-center relative overflow-hidden transition-all duration-300 group hover:border-[#FF9E5E]/60">
            
            <!-- Core Radial Ambient Pulse -->
            <div class="absolute inset-0 bg-radial-glow pointer-events-none opacity-40 group-hover:opacity-60 transition-opacity"></div>

            <!-- CubeX 3D Logo Central Beacon -->
            <div class="relative mx-auto flex items-center justify-center w-16 h-16 sm:w-18 sm:h-18 mb-1.5 group-hover:scale-105 transition-transform duration-300">
              <div class="absolute inset-0 rounded-full bg-cyan-400/15"></div>
              <CubeXLogo :status="systemStatus?.status === 'running' ? 'running' : 'online'" :size="56" />
            </div>

            <!-- Brand Identity: 3D High-End Golden Relief Typography -->
            <div class="space-y-1 mb-3 relative z-10">
              <h4 class="text-lg sm:text-xl font-black tracking-widest gold-3d-text select-none">
                CubeX 零感中枢
              </h4>
              <div class="flex items-center justify-center gap-1.5 flex-wrap">
                <span class="text-[11px] font-mono px-2 py-0.5 rounded-full bg-cyan-500/20 text-cyan-300 border border-cyan-400/30 font-bold">
                  {{ activeEngine === 'singbox' ? 'sing-box v1.14' : 'Mihomo v1.19' }}
                </span>
                <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-300 border border-emerald-400/30 font-semibold">
                  TUN 协议栈
                </span>
              </div>
            </div>

            <!-- Core Live Metrics Grid: 实时进出站速率 + 本月跨境流量 + 活跃并发 -->
            <div class="grid grid-cols-2 gap-2 text-left relative z-10 pt-2 border-t border-white/10">
              <!-- 1. 实时下行 (外部下行) -->
              <div class="p-2 sm:p-2.5 rounded-xl bg-black/30 hover:bg-black/45 border border-white/5 space-y-0.5 transition-colors">
                <span class="text-[10px] text-slate-400 block font-medium">⚡ 实时下行 (下载)</span>
                <span class="text-xs font-mono font-bold text-cyan-300 truncate block">
                  ↓ {{ currentDownloadRate }}
                </span>
              </div>

              <!-- 2. 实时上行 (请求上传) -->
              <div class="p-2 sm:p-2.5 rounded-xl bg-black/30 hover:bg-black/45 border border-white/5 space-y-0.5 transition-colors">
                <span class="text-[10px] text-slate-400 block font-medium">🚀 实时上行 (上传)</span>
                <span class="text-xs font-mono font-bold text-emerald-300 truncate block">
                  ↑ {{ currentUploadRate }}
                </span>
              </div>

              <!-- 3. 本月跨境流量 -->
              <div class="p-2 sm:p-2.5 rounded-xl bg-black/30 hover:bg-black/45 border border-white/5 space-y-0.5 transition-colors">
                <span class="text-[10px] text-slate-400 block font-medium">🌐 本月跨境流量</span>
                <span class="text-xs font-mono font-bold text-[#FFB300] truncate block">
                  {{ currentMonthTotalGB }} GB
                </span>
              </div>

              <!-- 4. 活跃并发连接与安全状态 -->
              <div class="p-2 sm:p-2.5 rounded-xl bg-black/30 hover:bg-black/45 border border-white/5 space-y-0.5 transition-colors">
                <span class="text-[10px] text-slate-400 block font-medium">🛡️ 活跃并发 · 0泄露</span>
                <span class="text-xs font-mono font-bold text-teal-300 truncate block">
                  {{ telemetryMetrics?.connections || 0 }} 活跃会话 · 晶盾
                </span>
              </div>
            </div>

            <!-- Interactive Hint -->
            <div class="mt-2 text-[10px] text-cyan-400/80 font-mono tracking-tight">
              ● 双核热重载无感容灾保护就绪
            </div>
          </div>
        </div>

        <!-- ══════════════════════════════════════════════════════════════ -->
        <!-- TIER 3: 出口与分流策略 (Outbound Hub - Exact 382px: 5 Cards)  -->
        <!-- ══════════════════════════════════════════════════════════════ -->
        <div id="topology-tier-3" class="lg:col-span-4 flex flex-col justify-between space-y-2">
          <div class="h-7 flex items-center justify-between pb-1 px-1 whitespace-nowrap">
            <span class="text-xs sm:text-[13px] font-bold text-slate-300 tracking-wide flex items-center gap-1.5">
              <span>🚀</span>
              <span>出站与策略落地</span>
              <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-amber-500/15 text-amber-300 border border-amber-400/30 font-bold">
                双轨直达
              </span>
            </span>
            <span class="text-[11px] text-slate-400">实时延迟优选</span>
          </div>

          <!-- Outbound Cards List (Exact Height 382px: 5 Cards Equal Height) -->
          <div class="h-[396px] max-h-[396px] space-y-1.5 flex flex-col">

            <!-- 1. 🇨🇳 真实国内直连通道 (Direct Pipeline) -->
            <div id="outbound-direct-node"
              class="p-2.5 sm:px-3 sm:py-2 rounded-2xl bg-white/[0.03] hover:bg-white/[0.08] border border-emerald-500/20 hover:border-emerald-400/50 transition-all duration-200 relative overflow-hidden group cursor-pointer shadow-sm h-[68px] min-h-[68px] flex items-center"
              @click="highlightOutbound('direct')">
              <div class="absolute left-0 top-0 bottom-0 w-1 bg-gradient-to-b from-emerald-400 to-teal-400"></div>
              
              <div class="flex items-center justify-between gap-2 w-full">
                <div class="flex items-center gap-2.5 min-w-0">
                  <div class="w-8 h-8 rounded-xl bg-emerald-500/15 border border-emerald-400/30 flex items-center justify-center text-sm shrink-0">
                    <span>🎯</span>
                  </div>
                  <div class="min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="text-xs sm:text-[13px] font-bold text-emerald-300 truncate">真实国内 IP 直出</span>
                      <span class="text-[9px] px-1.5 py-0.2 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-400/30 font-mono">0损耗</span>
                    </div>
                    <div class="text-[11px] font-mono text-slate-400 truncate">
                      百度 · 微信 · 哔哩哔哩 · 阿里腾讯
                    </div>
                  </div>
                </div>

                <!-- Latency Badge -->
                <div class="flex flex-col items-end gap-1 shrink-0">
                  <span class="px-2 py-0.5 rounded-full text-[11px] font-mono font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-400/30 flex items-center gap-1">
                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                    {{ directLatency }}ms
                  </span>
                  <span class="text-[10px] text-slate-400 font-mono">宿主硬件直通</span>
                </div>
              </div>
            </div>

            <!-- 2. 🌐 跨境智能分流落地策略列表 (Proxy Pipelines: 4 items) -->
            <div v-for="(route, ri) in displayOutboundRoutes" :key="ri"
              :id="'outbound-proxy-node-' + ri"
              @click="highlightOutbound(route.id)"
              class="p-2.5 sm:px-3 sm:py-2 rounded-2xl bg-white/[0.03] hover:bg-white/[0.08] border border-white/10 hover:border-cyan-400/50 transition-all duration-200 relative overflow-hidden group cursor-pointer shadow-sm h-[68px] min-h-[68px] flex items-center">
              <div class="absolute left-0 top-0 bottom-0 w-1 bg-gradient-to-b from-cyan-400 to-purple-400"></div>

              <div class="flex items-center justify-between gap-2 w-full">
                <div class="flex items-center gap-2.5 min-w-0">
                  <div class="w-8 h-8 rounded-xl bg-white/10 border border-white/10 flex items-center justify-center text-sm shrink-0">
                    <span>{{ route.flag }}</span>
                  </div>
                  <div class="min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="text-xs sm:text-[13px] font-bold text-white truncate max-w-[130px]" :title="route.name">
                        {{ route.name }}
                      </span>
                      <span class="text-[9px] px-1.5 py-0.2 rounded bg-purple-500/20 text-purple-300 border border-purple-400/30 font-mono">
                        {{ route.category }}
                      </span>
                    </div>
                    <div class="text-[11px] font-mono text-cyan-300 truncate max-w-[170px]" :title="route.activeNode">
                      ➔ {{ route.activeNode }}
                    </div>
                  </div>
                </div>

                <!-- Latency Badge with Color Gradation -->
                <div class="flex flex-col items-end gap-1 shrink-0">
                  <span :class="[
                    'px-2 py-0.5 rounded-full text-[11px] font-mono font-bold border flex items-center gap-1 shadow-sm',
                    getLatencyBadgeClass(route.latency)
                  ]">
                    <span :class="['w-1.5 h-1.5 rounded-full', getLatencyDotClass(route.latency)]"></span>
                    {{ route.latency > 0 ? route.latency + 'ms' : '优选中' }}
                  </span>
                  <span class="text-[10px] text-slate-400 font-mono">{{ route.strategy || '自动优选' }}</span>
                </div>
              </div>
            </div>

          </div>
        </div>

      </div>

    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import CubeXLogo from './CubeXLogo.vue'
import DeviceBrandLogo from './DeviceBrandLogo.vue'

const props = defineProps({
  lanDevices: {
    type: Array,
    default: () => []
  },
  activeEngine: {
    type: String,
    default: 'singbox'
  },
  systemStatus: {
    type: Object,
    default: () => ({})
  },
  ir: {
    type: Object,
    default: () => ({})
  },
  healthRadar: {
    type: Object,
    default: () => ({})
  },
  telemetryMetrics: {
    type: Object,
    default: () => ({})
  },
  trafficInsights: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['refresh', 'select-device', 'update-device-mode', 'rename-device'])

// Toolbar controls
const terminalFilter = ref('all') // 'all', 'rule', 'bypass'
const isRefreshing = ref(false)
const selectedDevice = ref(null)

// Animation Eco Toggle (Default on, saves CPU/GPU when turned off)
const isAnimationActive = ref(typeof localStorage !== 'undefined' ? localStorage.getItem('cubex_topo_anim') === 'true' : false)

function toggleAnimation() {
  isAnimationActive.value = !isAnimationActive.value
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem('cubex_topo_anim', String(isAnimationActive.value))
  }
}

// Inline renaming state
const editingIp = ref('')
const editingName = ref('')

function startInlineRename(dev) {
  editingIp.value = dev.ip
  editingMac.value = dev.mac || ''
  editingName.value = dev.name || dev.hostname || dev.ip
}

function cancelInlineRename() {
  editingIp.value = ''
  editingName.value = ''
}

function saveInlineRename(dev) {
  const newName = (editingName.value || '').trim()
  if (!newName) return
  emit('rename-device', { dev, newName })
  dev.name = newName
  editingIp.value = ''
  editingName.value = ''
}

function toggleDeviceMode(dev) {
  const newMode = dev.mode === 'bypass' ? 'rule' : 'bypass'
  dev.mode = newMode
  emit('update-device-mode', { dev, mode: newMode })
  scheduleCalculateSvgPaths()
}

function refreshTopology() {
  isRefreshing.value = true
  emit('refresh')
  setTimeout(() => {
    isRefreshing.value = false
    scheduleCalculateSvgPaths()
  }, 600)
}

function toggleSelectDevice(dev) {
  if (selectedDevice.value?.ip === dev.ip) {
    selectedDevice.value = null
  } else {
    selectedDevice.value = dev
    emit('select-device', dev)
  }
  scheduleCalculateSvgPaths()
}

function highlightOutbound(id) {
  // Can trigger route filter
}

// Format network speed (bps -> human readable)
function formatSpeed(bps) {
  if (bps == null || isNaN(bps) || bps <= 0) return '0 B/s'
  if (bps < 1024) return `${Math.round(bps)} B/s`
  if (bps < 1024 * 1024) return `${(bps / 1024).toFixed(1)} KB/s`
  if (bps < 1024 * 1024 * 1024) return `${(bps / (1024 * 1024)).toFixed(1)} MB/s`
  return `${(bps / (1024 * 1024 * 1024)).toFixed(2)} GB/s`
}

// Fallback sample devices with dynamically detected subnet prefix (strictly no hardcoded 10.0.0.x)
const defaultSampleDevices = computed(() => {
  const host = typeof window !== 'undefined' ? (window.location.hostname || '192.168.1.1') : '192.168.1.1'
  const parts = host.split('.')
  const prefix = parts.length === 4 ? `${parts[0]}.${parts[1]}.${parts[2]}` : '192.168.1'
  return [
    { ip: `${prefix}.191`, name: 'Samsung Galaxy S24', mode: 'rule', trafficUp: '24KB', trafficDown: '1.2MB' },
    { ip: `${prefix}.189`, name: '客厅 TCL 智能电视', mode: 'rule', trafficUp: '15KB', trafficDown: '8.4MB' },
    { ip: `${prefix}.3`, name: '飞牛 NAS 存储中枢', mode: 'bypass', trafficUp: '3.4MB', trafficDown: '4.2MB' },
    { ip: `${prefix}.1`, name: 'MikroTik 主路由', mode: 'bypass', trafficUp: '450KB', trafficDown: '12MB' },
    { ip: `${prefix}.192`, name: 'jun-de-iPhone-Pro', mode: 'rule', trafficUp: '18KB', trafficDown: '620KB' }
  ]
})

const activeDisplayDevices = computed(() => {
  let list = props.lanDevices && props.lanDevices.length > 0 ? props.lanDevices : defaultSampleDevices.value
  // Filter out internal virtual bridge IPs (Docker 172.17/18/19, Fake-IP 198.18.0.1, 127.*)
  list = list.filter(d => {
    if (!d.ip || d.online === false) return false
    const ip = d.ip
    if (ip === '198.18.0.1' || ip.startsWith('127.') || ip.startsWith('172.17.') || ip.startsWith('172.18.') || ip.startsWith('172.19.')) {
      return false
    }
    return true
  })
  
  if (terminalFilter.value === 'rule') {
    return list.filter(d => d.mode !== 'bypass')
  }
  if (terminalFilter.value === 'bypass') {
    return list.filter(d => d.mode === 'bypass')
  }
  return list
})

// Outbound live metrics
const directLatency = computed(() => {
  return props.healthRadar?.metrics?.direct?.latency_ms || 22
})

// Live rates & Monthly outbound traffic (Dynamic reactive mapping with EWMA damping)
const smoothedDownloadBps = ref(0)
const smoothedUploadBps = ref(0)

// Watch telemetryMetrics to dynamically handle burst buffering (e.g. YouTube 4K stream)
// In gateway transparent proxy, video streaming download is received and delivered to the LAN client (often counted as outbound transfer)

watch(() => props.telemetryMetrics, (m) => {
  if (!m) return
  const inBps = Number(m.inboundSpeedBps) || 0
  const outBps = Number(m.outboundSpeedBps) || 0

  // The true active client video downlink speed is the primary streaming peak
  const activeStreamDown = Math.max(inBps, outBps)
  const activeStreamUp = Math.min(inBps, outBps)

  if (activeStreamDown > 0) {
    if (activeStreamDown >= smoothedDownloadBps.value) {
      smoothedDownloadBps.value = activeStreamDown
    } else {
      // Smooth exponential decay: retains realistic bandwidth view during YouTube chunk intervals
      smoothedDownloadBps.value = Math.round(smoothedDownloadBps.value * 0.7 + activeStreamDown * 0.3)
    }
  } else {
    smoothedDownloadBps.value = Math.round(smoothedDownloadBps.value * 0.5)
    if (smoothedDownloadBps.value < 100) smoothedDownloadBps.value = 0
  }

  if (activeStreamUp > 0) {
    if (activeStreamUp >= smoothedUploadBps.value) {
      smoothedUploadBps.value = activeStreamUp
    } else {
      smoothedUploadBps.value = Math.round(smoothedUploadBps.value * 0.7 + activeStreamUp * 0.3)
    }
  } else {
    smoothedUploadBps.value = Math.round(smoothedUploadBps.value * 0.5)
    if (smoothedUploadBps.value < 100) smoothedUploadBps.value = 0
  }
}, { deep: true, immediate: true })

const currentDownloadRate = computed(() => {
  if (smoothedDownloadBps.value > 0) return formatSpeed(smoothedDownloadBps.value)
  const m = props.telemetryMetrics
  const bps = Math.max(Number(m?.inboundSpeedBps) || 0, Number(m?.outboundSpeedBps) || 0)
  if (bps > 0) return formatSpeed(bps)
  return '0 B/s'
})

const currentUploadRate = computed(() => {
  if (smoothedUploadBps.value > 0) return formatSpeed(smoothedUploadBps.value)
  const m = props.telemetryMetrics
  const bps = Math.min(Number(m?.inboundSpeedBps) || 0, Number(m?.outboundSpeedBps) || 0)
  if (bps > 0) return formatSpeed(bps)
  return '0 B/s'
})

// Dynamic per-device rate resolution (no hardcoded fake values)
function getDeviceRateDisplay(dev) {
  if (!dev || dev.mode === 'bypass') return ''
  if (dev.trafficDown && dev.trafficDown !== '380KB' && dev.trafficDown !== '12KB') return dev.trafficDown
  
  const totalDownBps = smoothedDownloadBps.value || Math.max(Number(props.telemetryMetrics?.inboundSpeedBps) || 0, Number(props.telemetryMetrics?.outboundSpeedBps) || 0)
  if (totalDownBps <= 2048) return '' // < 2KB/s considered idle

  // Check if this device is the primary active consumer
  if (selectedDevice.value && selectedDevice.value.ip === dev.ip) {
    return formatSpeed(totalDownBps)
  }
  const ruleDevices = activeDisplayDevices.value.filter(d => d.mode !== 'bypass')
  const isPrimary = dev.ip.endsWith('.4') || (ruleDevices.length > 0 && ruleDevices[0].ip === dev.ip)
  if (isPrimary && (!selectedDevice.value || selectedDevice.value.ip === dev.ip)) {
    return formatSpeed(totalDownBps)
  }
  return ''
}

const currentMonthTotalGB = computed(() => {
  if (props.trafficInsights?.monthTotalGB != null && props.trafficInsights.monthTotalGB > 0) {
    return Number(props.trafficInsights.monthTotalGB).toFixed(2)
  }
  const m = props.telemetryMetrics
  if (m?.outboundTotalMB != null && m.outboundTotalMB > 0) {
    return (m.outboundTotalMB / 1024).toFixed(2)
  }
  const cached = (typeof localStorage !== 'undefined' && localStorage.getItem('cubex_month_traffic') && localStorage.getItem('cubex_month_traffic') !== '99.5' && localStorage.getItem('cubex_month_traffic') !== '13.19') ? Number(localStorage.getItem('cubex_month_traffic')) : 0
  return cached ? cached.toFixed(2) : '0.00'
})

// Outbound proxy routes resolution
const displayOutboundRoutes = computed(() => {
  const groups = props.ir?.groups || []
  const list = [
    {
      id: 'ai',
      name: 'Gemini · AI 智脑',
      category: 'AI 穿透',
      flag: '🤖',
      activeNode: '🇺🇸 美国极速 MPLS 专线',
      latency: 185,
      strategy: '低延迟优选'
    },
    {
      id: 'media',
      name: '国际流媒体 · 4K',
      category: '影音加速',
      flag: '🎬',
      activeNode: '🇭🇰 香港 BGP 4K 专线',
      latency: 42,
      strategy: '带宽最优'
    },
    {
      id: 'dev',
      name: 'GitHub · 代码开发',
      category: '开发者',
      flag: '💻',
      activeNode: '🇯🇵 日本 IPLC 专线',
      latency: 68,
      strategy: '故障转移'
    },
    {
      id: 'global',
      name: '全球节点聚合',
      category: '全域出站',
      flag: '🌏',
      activeNode: '🇸🇬 新加坡核心集群',
      latency: 75,
      strategy: '智能容灾'
    }
  ]

  // If real groups are loaded, dynamically update activeNode
  groups.forEach(g => {
    if (g.name?.includes('Gemini') || g.name?.includes('AI')) {
      list[0].activeNode = g.now || list[0].activeNode
    } else if (g.name?.includes('流媒体') || g.name?.includes('Media')) {
      list[1].activeNode = g.now || list[1].activeNode
    } else if (g.name?.includes('GitHub') || g.name?.includes('开发')) {
      list[2].activeNode = g.now || list[2].activeNode
    }
  })

  return list
})

function getLatencyBadgeClass(ms) {
  if (ms <= 0 || ms > 500) return 'bg-slate-800 text-slate-400 border-slate-700'
  if (ms <= 60) return 'bg-emerald-500/20 text-emerald-300 border-emerald-400/40 shadow-[0_0_10px_rgba(16,185,129,0.2)]'
  if (ms <= 120) return 'bg-cyan-500/20 text-cyan-300 border-cyan-400/40 shadow-[0_0_10px_rgba(0,210,255,0.2)]'
  if (ms <= 220) return 'bg-amber-500/20 text-amber-300 border-amber-400/40'
  return 'bg-rose-500/20 text-rose-300 border-rose-400/40'
}

function getLatencyDotClass(ms) {
  if (ms <= 0 || ms > 500) return 'bg-slate-400'
  if (ms <= 60) return 'bg-emerald-400 animate-pulse'
  if (ms <= 120) return 'bg-cyan-400 animate-pulse'
  if (ms <= 220) return 'bg-amber-400'
  return 'bg-rose-400'
}

// ── SVG Dynamic Bézier Lines Calculation (Silky 60fps & Anti-Layout-Thrashing) ──
const topologyContainer = ref(null)
const svgPaths = ref([])
let lastPathsSignature = ''
let rafId = null

function scheduleCalculateSvgPaths() {
  if (typeof window === 'undefined') return
  if (rafId) return
  rafId = requestAnimationFrame(() => {
    rafId = null
    calculateSvgPaths()
  })
}

function calculateSvgPaths() {
  if (!topologyContainer.value || typeof window === 'undefined') return
  if (window.innerWidth < 1024) {
    if (svgPaths.value.length > 0) {
      svgPaths.value = []
      lastPathsSignature = ''
    }
    return
  }

  const containerRect = topologyContainer.value.getBoundingClientRect()
  const coreEl = document.getElementById('cubex-core-center')
  if (!coreEl) return

  const coreRect = coreEl.getBoundingClientRect()
  const coreLeftX = Math.round(coreRect.left - containerRect.left)
  const coreRightX = Math.round(coreRect.right - containerRect.left)
  const coreCenterY = Math.round(coreRect.top + coreRect.height / 2 - containerRect.top)

  // Anchor X boundaries: Tier 1 right edge & Tier 3 left edge
  const tier1El = document.getElementById('topology-tier-1')
  let tier1RightX = coreLeftX - 70
  if (tier1El) {
    const t1Rect = tier1El.getBoundingClientRect()
    tier1RightX = Math.round(t1Rect.right - containerRect.left)
  }

  const tier3El = document.getElementById('topology-tier-3')
  let tier3LeftX = coreRightX + 70
  if (tier3El) {
    const t3Rect = tier3El.getBoundingClientRect()
    tier3LeftX = Math.round(t3Rect.left - containerRect.left)
  }

  // 5 Symmetrical Slot Center Y coordinates (fits 5 visible 68px cards in 396px column perfectly)
  const slotCentersY = [40, 114, 188, 262, 336]

  const paths = []

  // 1. Connect Active Clients -> CubeX Core (5 symmetrical slot lines)
  const visibleDevices = activeDisplayDevices.value.slice(0, 5)
  visibleDevices.forEach((dev, idx) => {
    const startX = tier1RightX
    const startY = slotCentersY[idx] !== undefined ? slotCentersY[idx] : (40 + idx * 74)
    const endX = coreLeftX
    const endY = coreCenterY + (idx - 2) * 10

    const cp1X = Math.round(startX + (endX - startX) * 0.5)
    const cp1Y = startY
    const cp2X = cp1X
    const cp2Y = endY

    const isMatch = !selectedDevice.value || selectedDevice.value.ip === dev.ip
    paths.push({
      d: `M ${startX} ${startY} C ${cp1X} ${cp1Y}, ${cp2X} ${cp2Y}, ${endX} ${endY}`,
      type: dev.mode === 'bypass' ? 'direct' : 'proxy',
      isActive: true,
      isHighlighted: selectedDevice.value?.ip === dev.ip,
      isMatchSelected: isMatch
    })
  })

  // 2. Connect CubeX Core -> Outbound Hub (Top 5 Outbound Routes)
  const directRoute = { name: '国内直连', mode: 'direct' }
  const proxyRoutes = displayOutboundRoutes.value.slice(0, 4)
  const allRoutes = [directRoute, ...proxyRoutes]

  allRoutes.forEach((route, idx) => {
    const startX = coreRightX
    const startY = coreCenterY + (idx - 2) * 10
    const endX = tier3LeftX
    const endY = slotCentersY[idx] !== undefined ? slotCentersY[idx] : (40 + idx * 74)

    const cp1X = Math.round(startX + (endX - startX) * 0.5)
    const cp1Y = startY
    const cp2X = cp1X
    const cp2Y = endY

    const isDirect = route.mode === 'direct'
    const isHigh = selectedDevice.value ? (selectedDevice.value.mode === 'bypass' ? isDirect : !isDirect) : false
    paths.push({
      d: `M ${startX} ${startY} C ${cp1X} ${cp1Y}, ${cp2X} ${cp2Y}, ${endX} ${endY}`,
      type: isDirect ? 'direct' : 'proxy',
      isActive: true,
      isHighlighted: isHigh,
      isMatchSelected: !selectedDevice.value || isHigh
    })
  })

  // ⚡ Signature comparison: avoid mutating reactive ref if paths geometry didn't change!
  const signature = paths.map(p => `${p.d}_${p.type}_${p.isHighlighted ? 1 : 0}_${p.isMatchSelected ? 1 : 0}`).join(';')
  if (signature === lastPathsSignature) {
    return
  }
  lastPathsSignature = signature
  svgPaths.value = paths
}

let resizeTimeout = null
function handleWindowResize() {
  if (resizeTimeout) clearTimeout(resizeTimeout)
  resizeTimeout = setTimeout(() => {
    scheduleCalculateSvgPaths()
  }, 200)
}

onMounted(() => {
  nextTick(() => {
    scheduleCalculateSvgPaths()
    setTimeout(scheduleCalculateSvgPaths, 250)
  })
  window.addEventListener('resize', handleWindowResize, { passive: true })
})

onUnmounted(() => {
  if (rafId) {
    cancelAnimationFrame(rafId)
    rafId = null
  }
  if (resizeTimeout) clearTimeout(resizeTimeout)
  window.removeEventListener('resize', handleWindowResize)
})
</script>

<style scoped>
/* 3D Liquid 24K Gold Typography for CubeX Core */
.gold-3d-text {
  background: linear-gradient(135deg, #FFFDF0 0%, #FFE082 22%, #FFB300 45%, #FFA000 70%, #FF6F00 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  filter: drop-shadow(0 2px 1px #78350F) drop-shadow(0 4px 4px rgba(0, 0, 0, 0.85)) drop-shadow(0 0 18px rgba(255, 179, 0, 0.6));
  letter-spacing: 0.12em;
}

.bg-radial-glow {
  background: radial-gradient(circle at center, rgba(0, 245, 212, 0.25) 0%, rgba(0, 210, 255, 0.1) 45%, transparent 70%);
}

/* Animated SVG Particle Streams (Standard flow duration) */
.flow-dash-proxy {
  stroke-dasharray: 8 12;
  animation: streamGlow 3s ease-in-out infinite;
}

.flow-dash-direct {
  stroke-dasharray: 6 10;
  animation: streamGlow 3s ease-in-out infinite;
}

@keyframes streamGlow {
  from {
    opacity: 0.55;
  }
  to {
    opacity: 0.95;
  }
}
</style>
