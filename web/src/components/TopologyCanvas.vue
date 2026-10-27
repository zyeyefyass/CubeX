<template>
  <div class="rounded-3xl p-5 sm:p-7 macos-glass-card shadow-2xl relative overflow-hidden transition-all duration-300 border border-white/10 select-none">
    
    <!-- Top Ambient Glow Aura -->
    <div class="absolute -top-24 -left-24 w-96 h-96 bg-cyan-500/10 rounded-full blur-3xl pointer-events-none"></div>
    <div class="absolute -bottom-24 -right-24 w-96 h-96 bg-[#FF9E5E]/10 rounded-full blur-3xl pointer-events-none"></div>

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
            <span class="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-ping"></span>
            Twilight Crystal 2.0
          </span>
          <span class="text-[11px] font-mono px-2 py-0.5 rounded-full bg-white/10 text-[#FEF08A] border border-white/20 font-medium hidden sm:inline-flex">
            微秒级多流感知
          </span>
        </div>
        <p class="text-xs text-slate-400 font-medium">
          三层拓扑感知 · 客户端终端集群 ➔ CubeX 智能双核中枢 ➔ 真实国内直连与跨境分流落地
        </p>
      </div>

      <!-- Right Toolbar Controls -->
      <div class="flex items-center gap-2.5 flex-wrap">
        <!-- Filter Tabs -->
        <div class="flex items-center p-1 rounded-xl bg-black/40 border border-white/10 text-xs">
          <button @click="pathFilter = 'all'"
            :class="['px-2.5 py-1 rounded-lg font-bold transition cursor-pointer', pathFilter === 'all' ? 'bg-gradient-to-r from-[#FF9E5E] to-[#F59E0B] text-[#1E1106] shadow' : 'text-slate-400 hover:text-white']">
            全部链路
          </button>
          <button @click="pathFilter = 'proxy'"
            :class="['px-2.5 py-1 rounded-lg font-bold transition cursor-pointer', pathFilter === 'proxy' ? 'bg-cyan-500 text-black shadow' : 'text-slate-400 hover:text-white']">
            跨境代理
          </button>
          <button @click="pathFilter = 'direct'"
            :class="['px-2.5 py-1 rounded-lg font-bold transition cursor-pointer', pathFilter === 'direct' ? 'bg-emerald-500 text-black shadow' : 'text-slate-400 hover:text-white']">
            国内直连
          </button>
        </div>

        <!-- Particle Speed Toggle -->
        <button @click="toggleParticleSpeed" title="调节粒子流速"
          class="h-8 px-2.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/15 text-xs text-slate-300 hover:text-white flex items-center gap-1.5 transition active:scale-95 cursor-pointer">
          <span class="text-[13px]">⚡</span>
          <span>{{ flowSpeedLabel }}</span>
        </button>

        <!-- Refresh Topology -->
        <button @click="refreshTopology" title="刷新拓扑状态"
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
          <filter id="glowEffect" x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur stdDeviation="3" result="blur" />
            <feComposite in="SourceGraphic" in2="blur" operator="over" />
          </filter>
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
              'transition-all duration-300',
              flowSpeed !== 'paused' ? (p.type === 'direct' ? 'flow-dash-direct' : 'flow-dash-proxy') : ''
            ]"
            :style="{
              animationDuration: flowAnimationDuration,
              filter: p.isHighlighted ? 'url(#glowEffect)' : 'none'
            }"
          />
        </g>
      </svg>

      <!-- ── 3-Tier Grid Layout ── -->
      <div class="grid grid-cols-1 lg:grid-cols-12 gap-5 items-stretch relative z-10">

        <!-- ══════════════════════════════════════════════════════════════ -->
        <!-- TIER 1: 客户端终端集群 (LAN Clients) [lg:col-span-4]           -->
        <!-- ══════════════════════════════════════════════════════════════ -->
        <div class="lg:col-span-4 flex flex-col space-y-3">
          <div class="flex items-center justify-between pb-1 px-1">
            <span class="text-xs font-bold text-slate-300 tracking-wide flex items-center gap-1.5">
              <span>📱</span>
              <span>接入终端集群</span>
              <span class="text-[11px] font-mono px-2 py-0.2 rounded-full bg-cyan-950/80 text-cyan-300 border border-cyan-500/30 font-bold">
                {{ activeDisplayDevices.length }} 在线
              </span>
            </span>
            <span class="text-[11px] text-slate-400">点击设备高亮专属链路</span>
          </div>

          <!-- Device Cards Scroll List -->
          <div class="space-y-2.5 max-h-[440px] overflow-y-auto pr-1 custom-scrollbar">
            <div v-for="dev in activeDisplayDevices" :key="dev.ip"
              @click="toggleSelectDevice(dev)"
              :id="'client-node-' + dev.ip.replace(/\./g, '-')"
              :class="[
                'p-3 sm:p-3.5 rounded-2xl transition-all duration-200 cursor-pointer border relative overflow-hidden group',
                selectedDevice?.ip === dev.ip
                  ? 'bg-cyan-950/70 border-cyan-400 shadow-[0_0_20px_rgba(0,210,255,0.25)] ring-1 ring-cyan-400/50'
                  : 'bg-slate-950/60 hover:bg-slate-900/80 border-slate-800/80 hover:border-cyan-500/40'
              ]">
              
              <!-- Left Color Indicator Accent -->
              <div :class="['absolute left-0 top-0 bottom-0 w-1', dev.mode === 'bypass' ? 'bg-sky-400' : 'bg-emerald-400']"></div>

              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-2.5 min-w-0">
                  <div class="w-8 h-8 rounded-xl bg-white/10 flex items-center justify-center text-sm shrink-0 border border-white/10 group-hover:scale-105 transition-transform">
                    <span>{{ getDeviceIcon(dev) }}</span>
                  </div>
                  <div class="min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="text-xs sm:text-[13px] font-bold text-white truncate max-w-[140px]" :title="dev.name || dev.hostname || dev.ip">
                        {{ dev.name || dev.hostname || dev.ip }}
                      </span>
                      <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse shrink-0"></span>
                    </div>
                    <div class="text-[11px] font-mono text-slate-400 truncate">
                      {{ dev.ip }}
                    </div>
                  </div>
                </div>

                <!-- Right Mode & Traffic Badge -->
                <div class="flex flex-col items-end gap-1 shrink-0">
                  <span :class="[
                    'text-[10px] font-mono px-2 py-0.5 rounded-full font-bold border',
                    dev.mode === 'bypass'
                      ? 'bg-sky-500/15 text-sky-300 border-sky-400/30'
                      : 'bg-emerald-500/15 text-emerald-300 border-emerald-400/30'
                  ]">
                    {{ dev.mode === 'bypass' ? '直连绕过' : '智能分流' }}
                  </span>
                  <span class="text-[10px] font-mono text-slate-400 flex items-center gap-1">
                    <span class="text-emerald-400">↑{{ dev.trafficUp || '12KB' }}</span>
                    <span class="text-cyan-400">↓{{ dev.trafficDown || '380KB' }}</span>
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ══════════════════════════════════════════════════════════════ -->
        <!-- TIER 2: CubeX 旁路由双核中枢 (Gateway Core) [lg:col-span-4]    -->
        <!-- ══════════════════════════════════════════════════════════════ -->
        <div class="lg:col-span-4 flex flex-col justify-center items-center my-auto px-2 py-4">
          <div id="cubex-core-center"
            class="w-full max-w-[320px] rounded-3xl p-5 sm:p-6 bg-gradient-to-b from-[#00223e]/90 via-[#001529]/95 to-[#000d1a]/95 border border-cyan-400/40 shadow-[0_0_35px_rgba(0,210,255,0.2)] text-center relative overflow-hidden transition-all duration-300 group hover:border-[#FF9E5E]/60">
            
            <!-- Core Radial Ambient Pulse -->
            <div class="absolute inset-0 bg-radial-glow pointer-events-none opacity-40 group-hover:opacity-60 transition-opacity"></div>

            <!-- CubeX 3D Logo Central Beacon -->
            <div class="relative mx-auto flex items-center justify-center w-20 h-20 mb-3 group-hover:scale-105 transition-transform duration-300">
              <div class="absolute inset-0 rounded-full bg-cyan-400/20 blur-xl animate-pulse"></div>
              <CubeXLogo :status="systemStatus?.status === 'running' ? 'running' : 'online'" :size="64" />
            </div>

            <!-- Brand Identity -->
            <div class="space-y-1 mb-4 relative z-10">
              <h4 class="text-base sm:text-lg font-black tracking-widest bg-gradient-to-r from-white via-cyan-200 to-[#FF9E5E] bg-clip-text text-transparent">
                CubeX 零感中枢
              </h4>
              <div class="flex items-center justify-center gap-1.5 flex-wrap">
                <span class="text-[11px] font-mono px-2.5 py-0.5 rounded-full bg-cyan-500/20 text-cyan-300 border border-cyan-400/30 font-bold">
                  {{ activeEngine === 'singbox' ? 'sing-box v1.14' : 'Mihomo v1.19' }}
                </span>
                <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-300 border border-emerald-400/30 font-semibold">
                  TUN 协议栈
                </span>
              </div>
            </div>

            <!-- Core Live Metrics Grid -->
            <div class="grid grid-cols-2 gap-2 text-left relative z-10 pt-2 border-t border-white/10">
              <div class="p-2.5 rounded-xl bg-black/40 border border-white/5 space-y-0.5">
                <span class="text-[10px] text-slate-400 block font-medium">🛡️ 防 DNS 泄露</span>
                <span class="text-xs font-mono font-bold text-emerald-300">:53 0明文回退</span>
              </div>
              <div class="p-2.5 rounded-xl bg-black/40 border border-white/5 space-y-0.5">
                <span class="text-[10px] text-slate-400 block font-medium">⚡ 吞吐总速率</span>
                <span class="text-xs font-mono font-bold text-cyan-300 truncate block">
                  {{ formatThroughput(telemetryMetrics?.inboundTotalMB || 0) }}
                </span>
              </div>
              <div class="p-2.5 rounded-xl bg-black/40 border border-white/5 space-y-0.5">
                <span class="text-[10px] text-slate-400 block font-medium">🌐 活跃并发连接</span>
                <span class="text-xs font-mono font-bold text-amber-300">
                  {{ telemetryMetrics?.connections || 128 }} 会话
                </span>
              </div>
              <div class="p-2.5 rounded-xl bg-black/40 border border-white/5 space-y-0.5">
                <span class="text-[10px] text-slate-400 block font-medium">🔒 安全规则拦截</span>
                <span class="text-xs font-mono font-bold text-teal-300">101 套 MRS/SRS</span>
              </div>
            </div>

            <!-- Interactive Hint -->
            <div class="mt-3 text-[10px] text-cyan-400/80 font-mono tracking-tight">
              ● 双核热重载无感容灾保护就绪
            </div>
          </div>
        </div>

        <!-- ══════════════════════════════════════════════════════════════ -->
        <!-- TIER 3: 出口与分流策略 (Outbound & Target Hub) [lg:col-span-4]  -->
        <!-- ══════════════════════════════════════════════════════════════ -->
        <div class="lg:col-span-4 flex flex-col space-y-3">
          <div class="flex items-center justify-between pb-1 px-1">
            <span class="text-xs font-bold text-slate-300 tracking-wide flex items-center gap-1.5">
              <span>🚀</span>
              <span>出站与策略落地</span>
              <span class="text-[11px] font-mono px-2 py-0.2 rounded-full bg-amber-500/15 text-amber-300 border border-amber-400/30 font-bold">
                双轨直达
              </span>
            </span>
            <span class="text-[11px] text-slate-400">实时延迟优选</span>
          </div>

          <div class="space-y-2.5 max-h-[440px] overflow-y-auto pr-1 custom-scrollbar">

            <!-- 1. 🇨🇳 真实国内直连极速通道 (Direct Pipeline) -->
            <div id="outbound-direct-node"
              class="p-3.5 rounded-2xl bg-slate-950/60 border border-emerald-500/40 hover:border-emerald-400 transition-all duration-200 relative overflow-hidden group cursor-pointer shadow-md"
              @click="highlightOutbound('direct')">
              <div class="absolute left-0 top-0 bottom-0 w-1 bg-gradient-to-b from-emerald-400 to-teal-400"></div>
              
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-2.5 min-w-0">
                  <div class="w-8 h-8 rounded-xl bg-emerald-500/15 border border-emerald-400/30 flex items-center justify-center text-sm shrink-0">
                    <span>🇨🇳</span>
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

            <!-- 2. 🌐 跨境智能分流落地策略列表 (Proxy Pipelines) -->
            <div v-for="(route, ri) in displayOutboundRoutes" :key="ri"
              :id="'outbound-proxy-node-' + ri"
              @click="highlightOutbound(route.id)"
              class="p-3 sm:p-3.5 rounded-2xl bg-slate-950/60 border border-slate-800/80 hover:border-cyan-500/40 transition-all duration-200 relative overflow-hidden group cursor-pointer shadow-md">
              <div class="absolute left-0 top-0 bottom-0 w-1 bg-gradient-to-b from-cyan-400 to-purple-400"></div>

              <div class="flex items-center justify-between gap-2">
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

      <!-- ── Bottom Interactive Inspection HUD Bar ── -->
      <div v-if="selectedDevice" class="mt-4 p-3.5 rounded-2xl bg-cyan-950/40 border border-cyan-400/30 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs animate-in fade-in">
        <div class="flex items-center gap-2.5 min-w-0">
          <span class="text-base">🔍</span>
          <div class="min-w-0">
            <span class="text-slate-300">当前聚焦终端:</span>
            <span class="font-bold text-white mx-1.5 font-mono">{{ selectedDevice.name || selectedDevice.ip }}</span>
            <span class="text-slate-400 font-mono">({{ selectedDevice.ip }})</span>
            <span class="text-cyan-300 mx-1 font-mono">➔</span>
            <span class="text-emerald-300 font-bold font-mono">{{ selectedDevice.mode === 'bypass' ? '国内物理直连' : 'CubeX 智能双核穿透' }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <button @click="selectedDevice = null" class="px-2.5 py-1 rounded-lg bg-white/10 hover:bg-white/20 text-slate-300 hover:text-white transition cursor-pointer">
            ✕ 清除聚焦
          </button>
        </div>
      </div>

    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import CubeXLogo from './CubeXLogo.vue'

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
  }
})

const emit = defineEmits(['refresh', 'select-device'])

// Toolbar controls
const pathFilter = ref('all')
const flowSpeed = ref('normal') // 'normal', 'fast', 'paused'
const isRefreshing = ref(false)
const selectedDevice = ref(null)

const flowSpeedLabel = computed(() => {
  if (flowSpeed.value === 'fast') return '极速流'
  if (flowSpeed.value === 'paused') return '已暂停'
  return '标准流'
})

const flowAnimationDuration = computed(() => {
  if (flowSpeed.value === 'fast') return '1.2s'
  return '2.4s'
})

function toggleParticleSpeed() {
  if (flowSpeed.value === 'normal') flowSpeed.value = 'fast'
  else if (flowSpeed.value === 'fast') flowSpeed.value = 'paused'
  else flowSpeed.value = 'normal'
}

function refreshTopology() {
  isRefreshing.value = true
  emit('refresh')
  setTimeout(() => {
    isRefreshing.value = false
    calculateSvgPaths()
  }, 600)
}

function toggleSelectDevice(dev) {
  if (selectedDevice.value?.ip === dev.ip) {
    selectedDevice.value = null
  } else {
    selectedDevice.value = dev
    emit('select-device', dev)
  }
  nextTick(() => calculateSvgPaths())
}

function highlightOutbound(id) {
  // Can trigger specific route selection or toast
}

// ── Device Icon Resolver ──
function getDeviceIcon(dev) {
  const str = `${dev.name || ''} ${dev.hostname || ''} ${dev.source || ''}`.toLowerCase()
  if (str.includes('tv') || str.includes('电视') || str.includes('tcl') || str.includes('box')) return '📺'
  if (str.includes('phone') || str.includes('samsung') || str.includes('ultra') || str.includes('iphone') || str.includes('mi')) return '📱'
  if (str.includes('nas') || str.includes('fnos') || str.includes('飞牛') || str.includes('synology')) return '🗄️'
  if (str.includes('ros') || str.includes('router') || str.includes('路由') || str.includes('gateway')) return '🌐'
  if (str.includes('pve') || str.includes('server') || str.includes('linux')) return '🖥️'
  return '💻'
}

// Fallback high-fidelity sample devices if lanDevices is empty
const defaultSampleDevices = [
  { ip: '10.0.0.191', name: 'Samsung Galaxy S24', mode: 'rule', trafficUp: '24KB', trafficDown: '1.2MB' },
  { ip: '10.0.0.189', name: '客厅 TCL 智能电视', mode: 'rule', trafficUp: '15KB', trafficDown: '8.4MB' },
  { ip: '10.0.0.3', name: '飞牛 NAS 存储中枢', mode: 'bypass', trafficUp: '3.4MB', trafficDown: '4.2MB' },
  { ip: '10.0.0.1', name: 'MikroTik 主路由', mode: 'bypass', trafficUp: '450KB', trafficDown: '12MB' },
  { ip: '10.0.0.192', name: 'jun-de-iPhone-Pro', mode: 'rule', trafficUp: '18KB', trafficDown: '620KB' }
]

const activeDisplayDevices = computed(() => {
  let list = props.lanDevices && props.lanDevices.length > 0 ? props.lanDevices : defaultSampleDevices
  // Filter out internal non-client IPs like 198.18.0.1 or offline
  list = list.filter(d => d.ip !== '198.18.0.1' && d.online !== false)
  
  if (pathFilter.value === 'proxy') {
    return list.filter(d => d.mode !== 'bypass')
  }
  if (pathFilter.value === 'direct') {
    return list.filter(d => d.mode === 'bypass')
  }
  return list.slice(0, 6)
})

// Outbound live metrics
const directLatency = computed(() => {
  return props.healthRadar?.metrics?.direct?.latency_ms || 22
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

  if (pathFilter.value === 'direct') {
    return []
  }
  return list
})

function formatThroughput(mb) {
  if (mb >= 1024) return (mb / 1024).toFixed(2) + ' GB'
  return mb.toFixed(1) + ' MB'
}

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

// ── SVG Dynamic Bézier Lines Calculation ──
const topologyContainer = ref(null)
const svgPaths = ref([])

function calculateSvgPaths() {
  if (!topologyContainer.value || typeof window === 'undefined') return
  if (window.innerWidth < 1024) {
    svgPaths.value = []
    return
  }

  const containerRect = topologyContainer.value.getBoundingClientRect()
  const coreEl = document.getElementById('cubex-core-center')
  if (!coreEl) return

  const coreRect = coreEl.getBoundingClientRect()
  const coreLeftX = coreRect.left - containerRect.left
  const coreRightX = coreRect.right - containerRect.left
  const coreCenterY = coreRect.top + coreRect.height / 2 - containerRect.top

  const paths = []

  // 1. Connect Active Clients -> CubeX Core
  activeDisplayDevices.value.forEach((dev, idx) => {
    const devEl = document.getElementById('client-node-' + dev.ip.replace(/\./g, '-'))
    if (!devEl) return
    const devRect = devEl.getBoundingClientRect()
    const startX = devRect.right - containerRect.left
    const startY = devRect.top + devRect.height / 2 - containerRect.top
    const endX = coreLeftX
    const endY = coreCenterY + (idx - 2) * 12

    const cp1X = startX + (endX - startX) * 0.5
    const cp1Y = startY
    const cp2X = startX + (endX - startX) * 0.5
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

  // 2. Connect CubeX Core -> Outbound Hub
  // 2.1 Direct outbound path
  const directEl = document.getElementById('outbound-direct-node')
  if (directEl && pathFilter.value !== 'proxy') {
    const dRect = directEl.getBoundingClientRect()
    const startX = coreRightX
    const startY = coreCenterY - 15
    const endX = dRect.left - containerRect.left
    const endY = dRect.top + dRect.height / 2 - containerRect.top

    const cp1X = startX + (endX - startX) * 0.5
    const cp1Y = startY
    const cp2X = startX + (endX - startX) * 0.5
    const cp2Y = endY

    const isHigh = selectedDevice.value?.mode === 'bypass'
    paths.push({
      d: `M ${startX} ${startY} C ${cp1X} ${cp1Y}, ${cp2X} ${cp2Y}, ${endX} ${endY}`,
      type: 'direct',
      isActive: true,
      isHighlighted: isHigh,
      isMatchSelected: !selectedDevice.value || isHigh
    })
  }

  // 2.2 Proxy outbound paths
  if (pathFilter.value !== 'direct') {
    displayOutboundRoutes.value.forEach((route, ri) => {
      const pEl = document.getElementById('outbound-proxy-node-' + ri)
      if (!pEl) return
      const pRect = pEl.getBoundingClientRect()
      const startX = coreRightX
      const startY = coreCenterY + (ri - 1) * 12
      const endX = pRect.left - containerRect.left
      const endY = pRect.top + pRect.height / 2 - containerRect.top

      const cp1X = startX + (endX - startX) * 0.5
      const cp1Y = startY
      const cp2X = startX + (endX - startX) * 0.5
      const cp2Y = endY

      const isHigh = selectedDevice.value && selectedDevice.value.mode !== 'bypass'
      paths.push({
        d: `M ${startX} ${startY} C ${cp1X} ${cp1Y}, ${cp2X} ${cp2Y}, ${endX} ${endY}`,
        type: 'proxy',
        isActive: true,
        isHighlighted: isHigh,
        isMatchSelected: !selectedDevice.value || isHigh
      })
    })
  }

  svgPaths.value = paths
}

let resizeObserver = null
onMounted(() => {
  nextTick(() => {
    calculateSvgPaths()
    setTimeout(calculateSvgPaths, 300)
    setTimeout(calculateSvgPaths, 1000)
  })

  if (typeof ResizeObserver !== 'undefined' && topologyContainer.value) {
    resizeObserver = new ResizeObserver(() => calculateSvgPaths())
    resizeObserver.observe(topologyContainer.value)
  }
  window.addEventListener('resize', calculateSvgPaths)
})

onUnmounted(() => {
  if (resizeObserver) resizeObserver.disconnect()
  window.removeEventListener('resize', calculateSvgPaths)
})
</script>

<style scoped>
.bg-radial-glow {
  background: radial-gradient(circle at center, rgba(0, 245, 212, 0.25) 0%, rgba(0, 210, 255, 0.1) 45%, transparent 70%);
}

/* Animated SVG Particle Streams */
.flow-dash-proxy {
  stroke-dasharray: 8 12;
  animation: flowStream 2.4s linear infinite;
}

.flow-dash-direct {
  stroke-dasharray: 6 10;
  animation: flowStream 2s linear infinite;
}

@keyframes flowStream {
  from {
    stroke-dashoffset: 80;
  }
  to {
    stroke-dashoffset: 0;
  }
}
</style>
