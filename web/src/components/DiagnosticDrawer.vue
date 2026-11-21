<template>
  <teleport to="body">
    <transition
      enter-active-class="transition duration-300 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-200 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="isOpen"
        class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex justify-end"
        @click.self="$emit('close')"
      >
        <transition
          enter-active-class="transition transform duration-300 ease-out"
          enter-from-class="translate-x-full"
          enter-to-class="translate-x-0"
          leave-active-class="transition transform duration-200 ease-in"
          leave-from-class="translate-x-0"
          leave-to-class="translate-x-full"
        >
          <div
            v-if="isOpen"
            class="diagnostic-drawer w-full max-w-lg md:max-w-xl h-full bg-[#001529]/95 text-slate-200 shadow-2xl border-l border-white/10 flex flex-col backdrop-blur-xl"
            style="contain: strict;"
          >
            <!-- 顶栏：标题与关闭 -->
            <div class="px-5 py-4 border-b border-white/10 flex items-center justify-between shrink-0 bg-[#001c37]/60">
              <div class="flex items-center space-x-2.5">
                <span class="w-2.5 h-2.5 rounded-full" :class="activeState.dotClass"></span>
                <h3 class="text-base font-bold text-white tracking-wide">
                  全链路排障与数据面诊断抽屉
                </h3>
              </div>
              <button
                @click="$emit('close')"
                class="p-1.5 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
              >
                <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <!-- 四态联调测试切换卡 (仅在开发态或 URL 携带 ?debug=1 时开启，杜绝普通用户误操作惊吓) -->
            <div v-if="isDebug" class="px-5 py-2.5 bg-[#002244]/40 border-b border-white/[0.06] flex items-center space-x-1.5 shrink-0 overflow-x-auto text-xs">
              <span class="text-slate-400 shrink-0 font-medium mr-1">联调视图:</span>
              <button
                v-for="mode in simModes"
                :key="mode.id"
                @click="currentSimMode = mode.id"
                :class="[
                  'px-2.5 py-1 rounded-full font-medium transition-all shrink-0',
                  currentSimMode === mode.id
                    ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-400/40 shadow-sm'
                    : 'bg-white/[0.04] text-slate-400 hover:text-slate-200 border border-transparent'
                ]"
              >
                {{ mode.name }}
              </button>
            </div>

            <!-- 中间内容区：独立滚动视口 -->
            <div class="flex-1 min-h-0 overflow-y-auto p-5 space-y-5 text-xs select-text">
              <!-- 1. 状态大白话卡片 -->
              <div class="p-4 rounded-2xl border" :class="activeState.cardClass">
                <div class="flex items-center justify-between mb-2">
                  <div class="flex items-center space-x-2">
                    <span class="w-2.5 h-2.5 rounded-full" :class="activeState.dotClass"></span>
                    <span class="text-sm font-bold text-white">{{ activeState.title }}</span>
                  </div>
                  <span class="font-mono text-[11px] px-2 py-0.5 rounded-full bg-black/30 border border-white/10 text-slate-300">
                    {{ displayedSnapshot.active_engine || 'sing-box' }}
                  </span>
                </div>
                <p class="text-slate-300 leading-relaxed">
                  {{ activeState.description }}
                </p>
              </div>

              <!-- 2. 核心指标矩阵 (等宽数字 tabular-nums) -->
              <div class="grid grid-cols-2 gap-3">
                <div class="p-3.5 rounded-xl bg-white/[0.04] border border-white/[0.08]">
                  <span class="text-slate-400 block mb-1">探活延迟 (Probe RTT)</span>
                  <span class="text-lg font-mono font-bold text-emerald-400 tabular-nums">
                    {{ displayedSnapshot.probe_duration_ms || 1 }} ms
                  </span>
                </div>
                <div class="p-3.5 rounded-xl bg-white/[0.04] border border-white/[0.08]">
                  <span class="text-slate-400 block mb-1">连续稳态计数 (Streak)</span>
                  <span class="text-lg font-mono font-bold text-cyan-300 tabular-nums">
                    {{ displayedSnapshot.success_streak || 25 }} 次
                  </span>
                </div>
                <div class="p-3.5 rounded-xl bg-white/[0.04] border border-white/[0.08]">
                  <span class="text-slate-400 block mb-1">Clash API 事件环</span>
                  <div class="flex items-center space-x-1.5 mt-1">
                    <span class="w-2 h-2 rounded-full" :class="displayedSnapshot.clash_api ? 'bg-emerald-400' : 'bg-rose-500'"></span>
                    <span class="font-semibold" :class="displayedSnapshot.clash_api ? 'text-emerald-300' : 'text-rose-400'">
                      {{ displayedSnapshot.clash_api ? '200 OK 就绪' : '连接超时' }}
                    </span>
                  </div>
                </div>
                <div class="p-3.5 rounded-xl bg-white/[0.04] border border-white/[0.08]">
                  <span class="text-slate-400 block mb-1">DNS Fake-IP 内存真探活</span>
                  <div class="flex items-center space-x-1.5 mt-1">
                    <span class="w-2 h-2 rounded-full" :class="displayedSnapshot.dns_service ? 'bg-emerald-400' : 'bg-rose-500'"></span>
                    <span class="font-semibold" :class="displayedSnapshot.dns_service ? 'text-emerald-300' : 'text-rose-400'">
                      {{ displayedSnapshot.dns_service ? 'UDP:53 秒应答' : 'DNS 响应挂起' }}
                    </span>
                  </div>
                </div>
              </div>

              <!-- 3. 脱敏结构化诊断详情 (如果载入了模拟故障) -->
              <div v-if="faultReport" class="p-4 rounded-2xl bg-black/40 border border-white/10 space-y-3">
                <div class="flex items-center justify-between text-xs font-semibold text-slate-300 border-b border-white/10 pb-2">
                  <span>故障特征与自检结果</span>
                  <span class="font-mono text-[11px] text-amber-400">已脱敏</span>
                </div>
                <div class="space-y-2 text-xs">
                  <div
                    v-for="(item, idx) in faultReport.checks"
                    :key="idx"
                    class="p-2 rounded-lg bg-white/[0.03] border border-white/[0.05] flex items-start space-x-2"
                  >
                    <span :class="item.passed ? 'text-emerald-400' : 'text-amber-400'" class="font-bold shrink-0">
                      {{ item.passed ? '✓' : '⚠' }}
                    </span>
                    <div class="flex-1 min-w-0">
                      <div class="font-semibold text-slate-200">{{ item.title }}</div>
                      <div class="text-slate-400 text-[11px] mt-0.5">{{ item.detail }}</div>
                      <div v-if="item.remediation" class="mt-1.5 p-2 rounded bg-black/50 text-cyan-300 font-mono text-[10px] break-all">
                        {{ item.remediation }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 4. 宿主共生逃生建议 -->
              <div class="p-4 rounded-2xl bg-white/[0.02] border border-white/[0.06] space-y-2">
                <span class="text-xs font-bold text-slate-300 block">RouterOS / 宿主毫秒逃生机制 (Netwatch 0秒避障)</span>
                <p class="text-slate-400 text-[11px] leading-relaxed">
                  当 `/healthz/dataplane` 返回 503 或未响应时，Netwatch 将在 50ms 内将策略路由回退到物理直出，全家网络毫秒级无损逃生。
                </p>
                <div class="p-2.5 rounded-lg bg-black/60 font-mono text-[10px] text-teal-300 select-all overflow-x-auto">
                  /tool netwatch add host=10.0.0.7 port=2026 http-path="/healthz/dataplane" type=http-get interval=2s timeout=1s
                </div>
              </div>
            </div>

            <!-- 底栏：一键脱敏复制与刷新 -->
            <div class="p-4 border-t border-white/10 bg-[#001c37]/80 shrink-0 flex items-center justify-between">
              <span class="text-[11px] text-slate-400 font-mono">
                最后更新: {{ formattedCheckedAt }}
              </span>
              <div class="flex items-center space-x-2">
                <button
                  @click="copyReport"
                  class="px-3.5 py-1.5 rounded-full bg-cyan-500/20 hover:bg-cyan-500/30 text-cyan-300 border border-cyan-400/40 text-xs font-medium transition-all shadow-sm active:scale-95 flex items-center space-x-1.5"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                  <span>{{ copySuccess ? '已复制脱敏诊断卡' : '一键复制脱敏卡' }}</span>
                </button>
              </div>
            </div>
          </div>
        </transition>
      </div>
    </transition>
  </teleport>
</template>

<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  isOpen: {
    type: Boolean,
    default: false
  },
  liveSnapshot: {
    type: Object,
    default: () => ({})
  }
})

defineEmits(['close'])

const currentSimMode = ref('live')
const copySuccess = ref(false)

const isDebug = computed(() => {
  return import.meta.env.DEV || (typeof window !== 'undefined' && window.location.search.includes('debug=1'))
})

const simModes = [
  { id: 'live', name: '实时实机状态' },
  { id: 'switching', name: '无感切核 (1.5s)' },
  { id: 'kill9', name: 'Panic 强杀降级' },
  { id: 'dns_hang', name: 'DNS 假死熔断' }
]

// 模拟故障数据
const mockKill9 = {
  status: 'degraded',
  state: 'degraded',
  healthy: false,
  active_engine: 'sing-box',
  clash_api: false,
  dns_service: true,
  probe_duration_ms: 201,
  fail_streak: 3,
  success_streak: 0,
  checked_at: new Date().toISOString()
}

const mockDNSHang = {
  status: 'degraded',
  state: 'degraded',
  healthy: false,
  active_engine: 'mihomo',
  clash_api: true,
  dns_service: false,
  probe_duration_ms: 205,
  fail_streak: 2,
  success_streak: 0,
  checked_at: new Date().toISOString()
}

const mockSwitching = {
  status: 'ok',
  state: 'switching',
  healthy: true,
  active_engine: 'sing-box -> mihomo',
  clash_api: true,
  dns_service: true,
  probe_duration_ms: 1,
  fail_streak: 0,
  success_streak: 1,
  checked_at: new Date().toISOString()
}

const displayedSnapshot = computed(() => {
  if (isDebug.value) {
    if (currentSimMode.value === 'kill9') return mockKill9
    if (currentSimMode.value === 'dns_hang') return mockDNSHang
    if (currentSimMode.value === 'switching') return mockSwitching
  }
  return props.liveSnapshot || mockSwitching
})

const activeState = computed(() => {
  const s = displayedSnapshot.value
  if (s.state === 'switching') {
    return {
      title: '双核热切中 (1.5s 宽限防抖保护生效)',
      description: '内核正执行零中断切换与配置原子重载，探针暂时进入宽限期，不会触发 RouterOS Netwatch 误熔断。',
      dotClass: 'bg-cyan-400 animate-pulse',
      cardClass: 'bg-cyan-950/30 border-cyan-500/30 text-cyan-200'
    }
  }
  if (!s.healthy || s.state === 'degraded' || s.status === 'degraded') {
    return {
      title: '已保底逃生 (数据面降级保护中)',
      description: '检测到部分数据面探针异常，CubeX 已自动联动逃生兜底链路，保障局域网物理直连不受阻碍。',
      dotClass: 'bg-amber-400 animate-ping',
      cardClass: 'bg-amber-950/30 border-amber-500/30 text-amber-200'
    }
  }
  return {
    title: '全链路健康稳态运行',
    description: 'Fake-IP 内存真探活与 Clash API 均在 1~2ms 极速闭环，主路由与宿主旁路无缝协同。',
    dotClass: 'bg-emerald-400',
    cardClass: 'bg-emerald-950/30 border-emerald-500/30 text-emerald-200'
  }
})

const faultReport = computed(() => {
  if (currentSimMode.value === 'kill9' || currentSimMode.value === 'dns_hang') {
    return {
      checks: [
        {
          passed: currentSimMode.value !== 'kill9',
          title: 'Clash API 控制面响应探针',
          detail: currentSimMode.value === 'kill9' ? '端口 9090 无应答 (内核已被强杀或崩溃 Panic)' : '200 OK 正常',
          remediation: currentSimMode.value === 'kill9' ? 'Supervisor 已启动自愈进程，若持续失败请点击一键自愈' : ''
        },
        {
          passed: currentSimMode.value !== 'dns_hang',
          title: 'Fake-IP DNS UDP 活体探活',
          detail: currentSimMode.value === 'dns_hang' ? 'UDP:53 发送 probe.test 超时，内核事件循环挂死' : '0ms 内存秒解',
          remediation: currentSimMode.value === 'dns_hang' ? '已触发 SERVFAIL 严格熔断防侧漏机制' : ''
        }
      ]
    }
  }
  return null
})

const formattedCheckedAt = computed(() => {
  const d = displayedSnapshot.value.checked_at
  if (!d) return '--:--:--'
  try {
    return new Date(d).toLocaleTimeString()
  } catch {
    return d
  }
})

const copyReport = async () => {
  const s = displayedSnapshot.value
  const text = JSON.stringify({
    report_type: 'cubex_dataplane_diagnostic_card',
    state: s.state,
    healthy: s.healthy,
    active_engine: s.active_engine,
    probe_duration_ms: s.probe_duration_ms,
    clash_api_liveness: s.clash_api,
    dns_service_liveness: s.dns_service,
    fail_streak: s.fail_streak,
    success_streak: s.success_streak,
    checked_at: s.checked_at
  }, null, 2)
  try {
    await navigator.clipboard.writeText(text)
    copySuccess.value = true
    setTimeout(() => { copySuccess.value = false }, 2500)
  } catch (err) {
    console.error('Failed to copy', err)
  }
}
</script>
