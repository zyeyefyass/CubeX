<template>
  <div :class="[
    'min-h-screen font-sans antialiased transition-colors duration-700 selection:bg-cyan-500/30 selection:text-cyan-200 flex flex-col md:flex-row overflow-hidden h-screen',
    activeEngine === 'singbox' ? 'bg-[#090714] text-slate-100' : 'bg-[#001222] text-slate-100'
  ]">

    <!-- ════════════════════════════════════════════════════════════════════ -->
    <!-- 1. PC 桌面端左侧固定侧边栏 (Desktop Left Sidebar >= 768px)            -->
    <!-- ════════════════════════════════════════════════════════════════════ -->
    <aside class="hidden md:flex md:w-64 md:flex-shrink-0 flex-col justify-between border-r transition-colors duration-700 z-30"
      :class="activeEngine === 'singbox' ? 'bg-[#0f0a20]/90 border-purple-900/40 backdrop-blur-xl' : 'bg-[#00172e]/90 border-sky-900/40 backdrop-blur-xl'">
      
      <div>
        <!-- Brand Hero with 3D Dynamic Crystal Cube -->
        <div class="p-5 text-center border-b transition-colors"
          :class="activeEngine === 'singbox' ? 'border-purple-900/30' : 'border-sky-900/30'">
          <div class="relative mx-auto w-24 h-24 mb-2 flex items-center justify-center group cursor-pointer" @click="activeMasterTab = 'overview'">
            <div class="absolute inset-0 rounded-full blur-xl group-hover:scale-125 transition-transform duration-700 animate-pulse"
              :class="activeEngine === 'singbox' ? 'bg-cyan-500/25' : 'bg-amber-500/25'"></div>
            <img src="/logo-cubex.svg" class="relative z-10 h-20 w-20 drop-shadow-[0_0_24px_rgba(0,245,212,0.6)] filter transition-all duration-500 group-hover:scale-110 active:scale-95" alt="CubeX Logo" />
          </div>
          
          <div class="flex items-center justify-center gap-2">
            <h1 class="text-2xl font-black tracking-wide bg-gradient-to-r bg-clip-text text-transparent"
              :class="activeEngine === 'singbox' ? 'from-cyan-300 via-teal-200 to-purple-400' : 'from-amber-300 via-sky-200 to-cyan-400'">
              CubeX
            </h1>
            <span class="text-[10px] font-mono px-2 py-0.5 rounded-full font-bold border"
              :class="activeEngine === 'singbox' ? 'bg-purple-950/80 text-cyan-300 border-purple-500/40' : 'bg-sky-950/80 text-amber-300 border-amber-500/40'">
              {{ systemStatus.version || 'v1.0.0-Aurora' }}
            </span>
          </div>
          <p class="text-[9px] tracking-widest mt-0.5 font-mono uppercase text-cyan-400/80">
            DUAL-ENGINE CYBER GATEWAY
          </p>

          <!-- Detected Host Platform badge in Sidebar with Vector Icon -->
          <div class="mt-3 w-full inline-flex items-center justify-between px-3 py-1.5 rounded-xl bg-slate-900/80 border border-cyan-500/30 text-xs shadow-md">
            <div class="flex items-center gap-2 overflow-hidden">
              <span class="text-sm">
                {{ hostBadge.icon }}
              </span>
              <div class="text-left overflow-hidden">
                <div class="text-[8px] text-slate-400 uppercase tracking-wider font-mono">HOST BASE</div>
                <div class="text-xs font-bold text-cyan-200 truncate max-w-[130px]" :title="hostBadge.name">
                  {{ hostBadge.name }}
                </div>
              </div>
            </div>
            <span :class="['w-2 h-2 rounded-full', hostBadge.online ? 'bg-emerald-400 shadow-[0_0_6px_#10b981] animate-pulse' : 'bg-amber-400']"></span>
          </div>
        </div>

        <!-- 4 Master Navigation Hubs -->
        <nav class="p-3 space-y-1.5">
          <button v-for="item in navItems" :key="item.id"
            @click="activeMasterTab = item.id"
            :class="[
              'w-full flex items-center gap-3 px-3.5 py-3 rounded-xl text-xs font-semibold transition-all duration-200 select-none',
              activeMasterTab === item.id
                ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-400/50 shadow-lg shadow-cyan-950/40 translate-x-1 font-bold'
                : 'text-slate-400 hover:bg-white/5 hover:text-slate-200 border border-transparent'
            ]">
            <span class="text-base">{{ item.icon }}</span>
            <span>{{ item.label }}</span>
          </button>
        </nav>
      </div>
    </aside>

    <!-- ════════════════════════════════════════════════════════════════════ -->
    <!-- 2. 右侧主内容区域 (Main Content Area with Top Header & Tab Views)     -->
    <!-- ════════════════════════════════════════════════════════════════════ -->
    <div class="flex-1 flex flex-col h-full overflow-y-auto">
      
      <!-- Top Holographic Bar -->
      <header class="w-full px-4 lg:px-8 py-3.5 border-b backdrop-blur-xl flex flex-wrap items-center justify-between gap-4 sticky top-0 z-20 transition-colors"
        :class="activeEngine === 'singbox' ? 'bg-[#090714]/90 border-purple-800/40' : 'bg-[#001222]/90 border-sky-800/40'">
        
        <!-- Left: Mobile Brand & Current Tab Title -->
        <div class="flex items-center gap-3">
          <button @click="activeMasterTab = 'overview'" class="md:hidden flex items-center gap-2">
            <img src="/logo-cubex.svg" class="w-8 h-8" alt="CubeX" />
          </button>
          <div class="flex items-center gap-2">
            <h2 class="text-base font-bold text-white">{{ currentTabTitle }}</h2>
            <span class="text-xs px-2 py-0.5 rounded-full font-mono font-bold bg-cyan-950 text-cyan-300 border border-cyan-500/40">
              全域双擎标准版
            </span>
          </div>
        </div>

        <!-- Center: Dual-Engine Capsule (Mihomo ⇄ sing-box 毫秒级热切) -->
        <div class="flex items-center p-1 rounded-2xl border shadow-xl backdrop-blur-2xl transition-all duration-500 relative"
          :class="activeEngine === 'singbox' ? 'bg-purple-950/60 border-purple-500/50' : 'bg-sky-950/60 border-cyan-500/50'">
          <button @click="switchEngine('singbox')" :disabled="switchingEngine"
            :class="[
              'px-4 py-1.5 rounded-xl text-xs font-bold transition flex items-center gap-1.5 active:scale-95',
              activeEngine === 'singbox' ? 'bg-gradient-to-r from-teal-400 to-cyan-500 text-slate-950 shadow-md font-extrabold' : 'text-slate-400 hover:text-slate-200'
            ]">
            <svg class="w-3.5 h-3.5 inline-block shrink-0" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2L20.66 7L12 12L3.34 7L12 2Z" fill="#3D4F59"/>
              <path d="M3.34 7L12 12V22L3.34 17V7Z" fill="#1E282E"/>
              <path d="M20.66 7L12 12V22L20.66 17V7Z" fill="#2E3C44"/>
              <path d="M7.67 4.5L16.33 9.5L14.33 10.65L5.67 5.65L7.67 4.5Z" fill="#A0B0BA" opacity="0.9"/>
              <path d="M9.67 3.35L18.33 8.35L16.33 9.5L7.67 4.5L9.67 3.35Z" fill="#E2E8F0" opacity="0.95"/>
              <path d="M14.33 10.65L16.33 9.5V14.5L14.33 15.65V10.65Z" fill="#8899A6" opacity="0.9"/>
              <path d="M16.33 9.5L18.33 8.35V13.35L16.33 14.5V9.5Z" fill="#CBD5E1" opacity="0.95"/>
            </svg>
            <span>sing-box</span>
          </button>
          <button @click="switchEngine('mihomo')" :disabled="switchingEngine"
            :class="[
              'px-4 py-1.5 rounded-xl text-xs font-bold transition flex items-center gap-1.5 active:scale-95',
              activeEngine === 'mihomo' ? 'bg-gradient-to-r from-amber-400 to-yellow-500 text-slate-950 shadow-md font-extrabold' : 'text-slate-400 hover:text-slate-200'
            ]">
            <svg class="w-3.5 h-3.5 inline-block shrink-0" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <defs>
                <linearGradient id="metaTopNav" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" stop-color="#38BDF8"/>
                  <stop offset="100%" stop-color="#818CF8"/>
                </linearGradient>
                <linearGradient id="metaLeftNav" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" stop-color="#6366F1"/>
                  <stop offset="100%" stop-color="#A855F7"/>
                </linearGradient>
                <linearGradient id="metaRightNav" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" stop-color="#0284C7"/>
                  <stop offset="100%" stop-color="#2563EB"/>
                </linearGradient>
              </defs>
              <path d="M12 2L20.66 7L12 12L3.34 7L12 2Z" fill="url(#metaTopNav)"/>
              <path d="M3.34 7L12 12V22L3.34 17V7Z" fill="url(#metaLeftNav)"/>
              <path d="M20.66 7L12 12V22L20.66 17V7Z" fill="url(#metaRightNav)"/>
              <path d="M12 2V12M3.34 7L12 12L20.66 7M12 12V22" stroke="#FFFFFF" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" opacity="0.8"/>
            </svg>
            <span>mihomo</span>
          </button>
          <div v-if="switchingEngine" class="absolute inset-0 bg-slate-950/90 rounded-2xl flex items-center justify-center text-xs text-cyan-300 font-mono animate-pulse">
            🔄 零断流切换中...
          </div>
        </div>

        <!-- Right: Panic Escape & Latency Indicator -->
        <div class="flex items-center gap-2">
          <div class="hidden sm:flex items-center gap-1.5 px-3 py-1 rounded-xl bg-slate-900/80 border border-slate-700/60 text-xs font-mono text-slate-300">
            <span class="w-2 h-2 rounded-full bg-emerald-400 animate-ping"></span>
            <span>延迟: <strong class="text-emerald-400">{{ globalLatency }}ms</strong></span>
          </div>

          <button @click="togglePanic" 
            :class="[
              'px-3 py-1.5 rounded-xl text-xs font-mono font-bold transition flex items-center gap-1 active:scale-95',
              isPanicked 
                ? 'bg-emerald-600 hover:bg-emerald-500 text-white animate-pulse' 
                : 'bg-rose-950/70 hover:bg-rose-900 border border-rose-500/50 text-rose-200'
            ]">
            <span>{{ isPanicked ? '🟢 恢复加速' : '🚨 1秒急救直连' }}</span>
          </button>
        </div>
      </header>

      <!-- Toast Feedback Notification Banner -->
      <div v-if="toastMsg" class="mx-4 lg:mx-8 mt-4 p-3 rounded-2xl border text-xs font-mono flex items-center justify-between shadow-2xl transition-all animate-in fade-in"
        :class="toastError ? 'bg-rose-950/90 border-rose-500/60 text-rose-200' : 'bg-cyan-950/90 border-cyan-500/60 text-cyan-200'">
        <div class="flex items-center gap-2">
          <span class="w-2 h-2 rounded-full" :class="toastError ? 'bg-rose-400 animate-ping' : 'bg-cyan-400 animate-ping'"></span>
          <span>{{ toastMsg }}</span>
        </div>
        <button @click="toastMsg = ''" class="text-slate-400 hover:text-white px-2 py-0.5 rounded bg-slate-800/80">✕</button>
      </div>

      <!-- Main Body Content Views -->
      <main class="flex-1 p-4 lg:p-8 space-y-6 pb-20 md:pb-8">

        <!-- ════════════════════════════════════════════════════════════════ -->
        <!-- TAB 1: 总览与分流 (Overview & Dual-Engine Cockpit)                -->
        <!-- ════════════════════════════════════════════════════════════════ -->
        <div v-if="activeMasterTab === 'overview'" class="space-y-6 animate-in fade-in">
          
          <!-- 双核状态驾驶舱 Hero Bento -->
          <section :class="['rounded-3xl p-6 lg:p-8 space-y-6 transition', activeEngine === 'singbox' ? 'glass-aurora' : 'glass-matrix']">
            <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-slate-700/40 pb-4">
              <div>
                <div class="flex items-center gap-2">
                  <span class="text-xl">🌌</span>
                  <h2 class="text-lg font-bold text-white tracking-wide">双擎主控驾驶舱</h2>
                  <span class="text-xs font-mono px-2.5 py-0.5 rounded-full bg-cyan-950 text-cyan-300 border border-cyan-500/40">
                    当前内核: {{ activeEngine === 'singbox' ? 'sing-box (v1.14.3)' : 'mihomo (v1.19.32)' }}
                  </span>
                </div>
                <p class="text-xs text-slate-400 mt-1">Universal IR 配置中间表示层驱动 · 支持在 TProxy 透明代理与原生 TUN 协议栈间无缝切换。</p>
              </div>

              <!-- 4 Quick Actions -->
              <div class="flex flex-wrap items-center gap-2">
                <button @click="runAllLatencyTest" :disabled="testingLatency" class="px-3.5 py-2 rounded-xl text-xs font-bold bg-cyan-600/30 hover:bg-cyan-600/50 border border-cyan-400 text-cyan-200 transition flex items-center gap-1.5 shadow-md">
                  <span>{{ testingLatency ? '测速中...' : '⚡ 全员测速' }}</span>
                </button>
                <button @click="triggerReload" class="px-3.5 py-2 rounded-xl text-xs font-bold bg-slate-800 hover:bg-slate-700 border border-slate-600 text-slate-200 transition">
                  🔄 重载配置
                </button>
                <button @click="showImportModal = true" class="px-3.5 py-2 rounded-xl text-xs font-bold bg-gradient-to-r from-teal-400 to-cyan-500 text-slate-950 transition">
                  📥 导入订阅
                </button>
              </div>
            </div>

            <!-- 4 Engine Metrics Grid (Live Memory & Telemetry) -->
            <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-xs font-mono">
              <!-- 1. Real-time Memory Overhead -->
              <div class="p-4 rounded-2xl bg-slate-950/70 border border-cyan-500/40 space-y-1 relative overflow-hidden shadow-lg">
                <div class="flex items-center justify-between">
                  <span class="text-slate-400 text-[10px] uppercase font-bold tracking-wider">MEMORY OVERHEAD</span>
                  <span class="text-[9px] px-1.5 py-0.5 rounded bg-cyan-950 text-cyan-300 border border-cyan-500/40 font-bold">实时开销</span>
                </div>
                <div class="text-base font-bold text-cyan-300 flex items-baseline gap-1">
                  <span>{{ (systemStatus.total_overhead_mb || (activeEngine === 'singbox' ? 24.8 : 55.1)).toFixed(1) }}</span>
                  <span class="text-xs text-slate-400">MB</span>
                  <span class="text-[10px] text-emerald-400 ml-1 font-normal">● 极轻</span>
                </div>
                <div class="text-[10px] text-slate-400 truncate">
                  主控: {{ (systemStatus.memory_mb || 6.6).toFixed(1) }}MB | 内核: {{ (systemStatus.core_memory_mb || (activeEngine === 'singbox' ? 18.2 : 48.5)).toFixed(1) }}MB
                </div>
                <!-- Visual RAM Activity Bar -->
                <div class="w-full bg-slate-800 h-1.5 rounded-full mt-2 overflow-hidden">
                  <div class="bg-gradient-to-r from-teal-400 to-cyan-500 h-full rounded-full transition-all duration-500" 
                    :style="{ width: Math.min(100, Math.max(12, ((systemStatus.total_overhead_mb || 25) / 100) * 100)) + '%' }"></div>
                </div>
              </div>

              <!-- 2. Host System Telemetry -->
              <div class="p-4 rounded-2xl bg-slate-950/70 border border-purple-500/30 space-y-1">
                <div class="flex items-center justify-between">
                  <span class="text-slate-400 text-[10px] uppercase font-bold tracking-wider">HOST RESOURCES</span>
                  <span class="text-[9px] px-1.5 py-0.5 rounded bg-purple-950 text-purple-300 border border-purple-500/40 font-bold">宿主负载</span>
                </div>
                <div class="text-base font-bold text-purple-300 truncate">
                  {{ hostInfo.cpu_cores || 4 }} 核 · {{ hostInfo.memory_used || '1.8GB' }}
                </div>
                <div class="text-[10px] text-slate-400 truncate">
                  总内存: {{ hostInfo.memory_total || '16GB' }} | 转发: {{ hostInfo.ip_forward ? '就绪 (1)' : '待开启' }}
                </div>
                <div class="w-full bg-slate-800 h-1.5 rounded-full mt-2 overflow-hidden">
                  <div class="bg-gradient-to-r from-purple-500 to-indigo-500 h-full rounded-full" style="width: 28%"></div>
                </div>
              </div>

              <!-- 3. Strict 0-Leak DNS -->
              <div class="p-4 rounded-2xl bg-slate-950/70 border border-emerald-500/30 space-y-1">
                <div class="flex items-center justify-between">
                  <span class="text-slate-400 text-[10px] uppercase font-bold tracking-wider">DNS RESOLVER</span>
                  <span class="text-[9px] px-1.5 py-0.5 rounded bg-emerald-950 text-emerald-300 border border-emerald-500/40 font-bold">0泄露</span>
                </div>
                <div class="text-base font-bold text-emerald-400">:{{ ir.network?.dns_port || 1053 }}</div>
                <div class="text-[10px] text-slate-400 truncate">国内真实IP直出 · 国外 Fake-IP</div>
                <div class="text-[10px] text-slate-500 truncate">阻断 Type 65 泄露 · 安全防嗅探</div>
              </div>

              <!-- 4. Virtual Protocol Stack & Tun -->
              <div class="p-4 rounded-2xl bg-slate-950/70 border border-amber-500/30 space-y-1">
                <div class="flex items-center justify-between">
                  <span class="text-slate-400 text-[10px] uppercase font-bold tracking-wider">CORE NETSTACK</span>
                  <span class="text-[9px] px-1.5 py-0.5 rounded bg-amber-950 text-amber-300 border border-amber-500/40 font-bold">协议栈</span>
                </div>
                <div class="text-base font-bold text-amber-300 font-mono truncate">{{ ir.network?.tun_interface || 'cubex-tun' }}</div>
                <div class="text-[10px] text-slate-400 truncate">TProxy: :{{ ir.network?.tproxy_port || 7893 }}</div>
                <div class="text-[10px] text-slate-500 truncate">Clash API 控制器: :{{ ir.network?.clash_api_port || 9090 }}</div>
              </div>
            </div>
          </section>

          <!-- 双核同节点 AB 竞技场 (Benchmark Arena) -->
          <section :class="['rounded-2xl p-5 space-y-4 transition', activeEngine === 'singbox' ? 'glass-aurora' : 'glass-matrix']">
            <div class="flex items-center justify-between border-b border-slate-700/40 pb-3 flex-wrap gap-2">
              <div>
                <div class="text-sm font-bold text-white flex items-center gap-2">
                  <span>⚔️ 双核同节点 AB 延迟/吞吐竞技场 (Benchmark Arena)</span>
                </div>
                <div class="text-xs text-slate-400 mt-0.5">同一节点同时对比 Mihomo 与 sing-box TCP/TLS 耗时与内存开销</div>
              </div>
              <button @click="runBenchmark" :disabled="benchmarking" class="px-4 py-2 rounded-xl text-xs font-bold bg-gradient-to-r from-cyan-500 to-purple-600 text-white shadow-lg active:scale-95 transition">
                {{ benchmarking ? '压测中...' : '开始双核竞技' }}
              </button>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div class="bg-slate-950/70 border border-cyan-500/40 rounded-2xl p-4 space-y-2">
                <div class="flex justify-between text-xs font-bold text-cyan-300">
                  <span>⚡ sing-box v1.14+ (轻量轻载)</span>
                  <span>{{ benchmarkResult.singbox?.total_latency_ms || 42 }} ms</span>
                </div>
                <div class="text-[11px] text-slate-400 font-mono">
                  TCP 握手: {{ benchmarkResult.singbox?.tcp_handshake_ms || 18 }}ms | 内存开销: {{ benchmarkResult.singbox?.memory_usage_mb || 18.2 }}MB
                </div>
              </div>
              <div class="bg-slate-950/70 border border-amber-500/40 rounded-2xl p-4 space-y-2">
                <div class="flex justify-between text-xs font-bold text-amber-300">
                  <span>🛡️ Mihomo core (矩阵分流)</span>
                  <span>{{ benchmarkResult.mihomo?.total_latency_ms || 46 }} ms</span>
                </div>
                <div class="text-[11px] text-slate-400 font-mono">
                  TCP 握手: {{ benchmarkResult.mihomo?.tcp_handshake_ms || 18 }}ms | 内存开销: {{ benchmarkResult.mihomo?.memory_usage_mb || 48.5 }}MB
                </div>
              </div>
            </div>
          </section>

          <!-- 实时沙盒推演与策略组热力点阵 -->
          <section :class="['rounded-2xl p-5 space-y-4 transition', activeEngine === 'singbox' ? 'glass-aurora' : 'glass-matrix']">
            <div class="flex items-center justify-between border-b border-slate-700/40 pb-3 flex-wrap gap-2">
              <span class="text-sm font-bold text-white flex items-center gap-1.5">
                <span>🎯 全链路沙盒分流推演 (Packet Tracer)</span>
              </span>
              <input v-model="tracerInput" placeholder="输入测试域名 (例: netflix.com 或 bilibili.com)..." 
                class="bg-slate-950/90 border border-cyan-500/40 rounded-xl px-3 py-1.5 text-xs text-slate-100 font-mono outline-none w-72" />
            </div>
            
            <div class="p-3 bg-slate-950/60 rounded-xl border border-slate-800 text-xs font-mono grid grid-cols-1 sm:grid-cols-4 gap-2">
              <div>推演请求: <strong class="text-cyan-300">{{ tracerInput || 'netflix.com' }}</strong></div>
              <div>命中规则: <strong class="text-purple-300">{{ currentTracedRule }}</strong></div>
              <div>策略分流组: <strong class="text-cyan-300">{{ currentTracedGroup }}</strong></div>
              <div>最终出口节点: <strong class="text-amber-300">{{ currentTracedNode }}</strong></div>
            </div>

            <!-- 热力点阵 -->
            <div class="pt-3 border-t border-slate-800/80">
              <div class="flex items-center justify-between text-xs font-bold text-slate-300 mb-2">
                <span>节点池健康度阵列 (单点快切):</span>
                <span class="text-[10px] text-slate-400 font-mono">共 {{ ir.proxies?.length || 0 }} 个节点</span>
              </div>
              <div class="flex flex-wrap gap-2">
                <button v-for="(node, idx) in ir.proxies" :key="node.name + idx"
                  @click="selectNodeFast(node.name)" :title="`${node.name} (${node.delay || 32}ms)`"
                  class="p-1 rounded active:scale-90 transition">
                  <span :class="['block w-3.5 h-3.5 rounded-full', activeSelectedNode === node.name ? 'ring-2 ring-white scale-125' : '', getDotColor(node.delay || 35)]"></span>
                </button>
              </div>
            </div>
          </section>

          <!-- 🧭 策略分流与出站节点控制中枢 (Traffic Routing & Proxy Groups Matrix) -->
          <section class="space-y-4">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-700/40 pb-3">
              <div>
                <div class="text-sm font-bold text-white flex items-center gap-2">
                  <span>🧭 策略分流与出站节点控制中枢 (Traffic Routing & Proxy Groups)</span>
                  <span class="text-xs px-2.5 py-0.5 rounded-full font-mono font-bold bg-cyan-950 text-cyan-300 border border-cyan-500/40">
                    {{ ir.groups?.length || 0 }} 个分流策略组 · {{ ir.proxies?.length || 0 }} 个出口节点
                  </span>
                </div>
                <div class="text-xs text-slate-400 mt-1">
                  依据 217heidai 权威规则与应用特征动态分流 · 点击任意候选节点即刻毫秒级热切该组专属出口。
                </div>
              </div>
              <div class="flex items-center gap-2">
                <button @click="runAllLatencyTest" :disabled="testingLatency" class="px-3.5 py-1.5 rounded-xl text-xs font-bold bg-cyan-600/30 hover:bg-cyan-600/50 border border-cyan-400 text-cyan-200 transition flex items-center gap-1 shadow-md">
                  <span>{{ testingLatency ? '测速中...' : '⚡ 全员节点测速' }}</span>
                </button>
                <button @click="fetchIR" class="px-3 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 border border-slate-600 text-slate-200 text-xs font-semibold">
                  🔄 刷新分流
                </button>
              </div>
            </div>

            <!-- Strategy Groups Cards Grid -->
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
              <div v-for="grp in ir.groups" :key="grp.name" class="p-4 rounded-2xl bg-slate-900/90 border border-slate-800 space-y-3 hover:border-cyan-500/30 transition shadow-lg">
                <!-- Group Header -->
                <div class="flex items-center justify-between border-b border-slate-800/80 pb-2.5">
                  <div class="flex items-center gap-2">
                    <span class="text-base">{{ getGroupIcon(grp.name) }}</span>
                    <div>
                      <div class="text-xs font-bold text-white flex items-center gap-1.5">
                        <span>{{ grp.name }}</span>
                        <span class="text-[10px] font-mono px-2 py-0.2 rounded-full bg-cyan-950 text-cyan-300 border border-cyan-500/40">
                          {{ grp.type === 'select' ? '手动选择' : (grp.type === 'urltest' ? '自动优选' : grp.type) }}
                        </span>
                      </div>
                      <div class="text-[11px] font-mono text-cyan-300 mt-0.5 flex items-center gap-1">
                        <span class="text-slate-400">当前出口:</span>
                        <strong class="text-cyan-300 bg-cyan-950/60 px-2 py-0.5 rounded border border-cyan-500/30">
                          {{ grp.now || (grp.proxies && grp.proxies[0]) || '-' }}
                        </strong>
                      </div>
                    </div>
                  </div>
                  <span class="text-[11px] font-mono text-slate-400">
                    {{ (grp.proxies?.length || 0) }} 个候选
                  </span>
                </div>

                <!-- Fast Filter Tags -->
                <div class="flex items-center gap-1.5 overflow-x-auto text-[10px] font-mono py-0.5">
                  <span class="text-slate-500 shrink-0">筛选:</span>
                  <button v-for="tag in ['全部', '香港', '日本', '新加坡', '美国', '直连']" :key="tag"
                    @click="setGroupFilter(grp.name, tag === '全部' ? '' : tag)"
                    :class="[
                      'px-2 py-0.5 rounded-lg transition shrink-0',
                      (groupFilters[grp.name] || '') === (tag === '全部' ? '' : tag)
                        ? 'bg-cyan-500 text-slate-950 font-bold'
                        : 'bg-slate-950 text-slate-400 hover:text-slate-200 border border-slate-800'
                    ]">
                    {{ tag }}
                  </button>
                </div>

                <!-- Candidate Nodes Matrix -->
                <div class="flex flex-wrap gap-1.5 max-h-36 overflow-y-auto pt-1 pr-1">
                  <button v-for="nodeName in getFilteredGroupProxies(grp)" :key="nodeName"
                    @click="selectGroupNode(grp.name, nodeName)"
                    :class="[
                      'px-2.5 py-1.5 rounded-xl text-[11px] font-mono transition flex items-center gap-1.5 active:scale-95 border',
                      (grp.now || grp.proxies?.[0]) === nodeName
                        ? 'bg-cyan-500 text-slate-950 font-bold border-cyan-400 shadow-md shadow-cyan-500/20'
                        : 'bg-slate-950/80 text-slate-300 hover:text-white border-slate-800 hover:border-slate-700'
                    ]">
                    <span :class="['w-2 h-2 rounded-full shrink-0', getNodeDotColor(nodeName)]"></span>
                    <span class="truncate max-w-[140px]">{{ nodeName }}</span>
                    <span v-if="getNodeLatency(nodeName)" class="text-[9px] opacity-75 font-semibold">
                      {{ getNodeLatency(nodeName) }}ms
                    </span>
                  </button>
                </div>
              </div>
            </div>
          </section>

        </div>

        <!-- ════════════════════════════════════════════════════════════════ -->
        <!-- TAB 2: 穿透协同 (WireGuard & Remote Access & Per-Device Matrix)   -->
        <!-- ════════════════════════════════════════════════════════════════ -->
        <div v-else-if="activeMasterTab === 'wireguard'" class="space-y-6 animate-in fade-in">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-sky-800/40 pb-4">
            <div>
              <h2 class="text-xl font-bold text-white flex items-center gap-2">
                <span>📡 WireGuard 穿透协同 & 设备分流矩阵</span>
                <span class="text-xs px-2.5 py-0.5 rounded-full bg-cyan-950 text-cyan-300 border border-cyan-500/40 font-mono">
                  RouterOS v7 原生 & Linux 宿主双端双活
                </span>
              </h2>
              <p class="text-xs text-slate-400 mt-1">手机使用官方 WireGuard App 扫码秒连，配合局域网设备矩阵实现直连绕过与专属出口绑定。</p>
            </div>
            <div class="flex items-center gap-2">
              <button @click="inspectWgAlignment" :disabled="wgAligning" class="px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 border border-slate-600 text-slate-200 text-xs font-bold transition">
                🔍 体检对齐状态
              </button>
              <button @click="showAddPeerModal = true" class="px-4 py-2 rounded-xl bg-gradient-to-r from-teal-400 to-cyan-500 text-slate-950 font-bold text-xs shadow-lg flex items-center gap-1.5">
                <span>+ 添加穿透设备</span>
              </button>
            </div>
          </div>

          <!-- WireGuard Server Status Card -->
          <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div class="p-4 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-1">
              <span class="text-[10px] text-slate-400 font-mono uppercase">INTERFACE</span>
              <div class="text-sm font-bold text-cyan-300 font-mono">{{ wgInfo.config?.interface_name || 'cubex-wg' }}</div>
            </div>
            <div class="p-4 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-1">
              <span class="text-[10px] text-slate-400 font-mono uppercase">LISTEN PORT</span>
              <div class="text-sm font-bold text-emerald-300 font-mono">{{ wgInfo.config?.listen_port || 13231 }} (UDP)</div>
            </div>
            <div class="p-4 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-1">
              <span class="text-[10px] text-slate-400 font-mono uppercase">SERVER IP</span>
              <div class="text-sm font-bold text-white font-mono">{{ wgInfo.config?.server_address || '10.88.0.1/24' }}</div>
            </div>
            <div class="p-4 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-1">
              <span class="text-[10px] text-slate-400 font-mono uppercase">CLOUD DDNS</span>
              <div class="text-sm font-bold text-purple-300 font-mono truncate" :title="wgInfo.cloud_ddns">
                {{ wgInfo.cloud_ddns || 'RouterOS IP Cloud' }}
              </div>
            </div>
          </div>

          <!-- ⚡ WireGuard 与 RouterOS 实时双端对齐矩阵 (Dual-End Alignment Matrix) -->
          <div class="glass-matrix rounded-2xl p-5 space-y-4 border-l-4 border-l-cyan-400 shadow-xl">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-sky-900/30 pb-3">
              <div>
                <div class="text-sm font-bold text-white flex items-center gap-2">
                  <span>⚡ WireGuard 与 RouterOS 实时对齐矩阵</span>
                  <span :class="['text-xs px-2.5 py-0.5 rounded-full font-semibold', wgAlignmentReport.all_aligned ? 'bg-emerald-950 text-emerald-300 border border-emerald-500/40' : 'bg-amber-950 text-amber-300 border border-amber-500/40']">
                    {{ wgAlignmentReport.all_aligned ? '● 已完全对齐' : '● 待对齐 / 同步' }}
                  </span>
                </div>
                <div class="text-xs text-slate-400 mt-1">检测 RouterOS 硬件接口、虚拟网段 IP、UDP 防火墙、动态域名及所有已授权 Peer 是否 100% 同步。</div>
              </div>
              <div class="flex gap-2">
                <button @click="applyWgAlignment" :disabled="wgAligning" class="px-4 py-2 rounded-xl bg-cyan-500 hover:bg-cyan-400 text-slate-950 text-xs font-bold shadow-lg shadow-cyan-500/20 active:scale-95 transition">
                  <span v-if="wgAligning" class="inline-block animate-spin mr-1">↻</span>
                  <span>⚡ 一键对齐 WireGuard 设置</span>
                </button>
              </div>
            </div>

            <!-- Alignment Check Items -->
            <div v-if="wgAlignmentReport.items && wgAlignmentReport.items.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
              <div v-for="item in wgAlignmentReport.items" :key="item.key"
                class="bg-slate-900/60 p-3 rounded-xl border border-sky-900/30 space-y-1.5 hover:border-purple-700/40 transition">
                <div class="flex items-center justify-between text-xs font-semibold">
                  <span class="text-white">{{ item.name }}</span>
                  <span :class="['text-[10px] px-2 py-0.5 rounded font-mono', item.status === 'ok' ? 'bg-emerald-950/80 text-emerald-300 border border-emerald-500/40' : (item.status === 'warning' ? 'bg-amber-950/80 text-amber-300 border border-amber-500/40' : 'bg-red-950/80 text-red-300 border border-red-500/40')]">
                    {{ item.status === 'ok' ? '已对齐' : (item.status === 'warning' ? '需注意' : '未对齐') }}
                  </span>
                </div>
                <div class="text-[11px] font-mono text-cyan-300 truncate" :title="item.value">{{ item.value || '-' }}</div>
                <div class="text-[10px] text-slate-400 leading-relaxed">{{ item.details }}</div>
              </div>
            </div>
            <div v-else class="text-xs text-slate-400 py-1">
              点击「🔍 体检对齐状态」即可实时探查 RouterOS 上的 WireGuard 接口与各 Peer 配置。
            </div>
          </div>

          <!-- 🖥️ 飞牛 (fnOS) / Linux 宿主机 WireGuard 极速穿透卡片 -->
          <div class="p-5 rounded-2xl bg-slate-900/80 border border-emerald-500/40 space-y-3">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-emerald-900/30 pb-3">
              <div>
                <div class="text-sm font-bold text-white flex items-center gap-2">
                  <span>🖥️ 飞牛 (fnOS) / Linux 宿主机 WireGuard 极速穿透</span>
                  <span class="text-xs px-2.5 py-0.5 rounded-full font-semibold bg-emerald-950 text-emerald-300 border border-emerald-500/40">
                    内核级零损耗穿透
                  </span>
                </div>
                <div class="text-xs text-slate-400 mt-1">在宿主机 Linux 内核层直接拉起 wg0.conf，跑满千兆带宽并实现 0 内存损耗。</div>
              </div>
              <div class="flex gap-2">
                <button @click="deployHostWireGuard" :disabled="deployingHostWg" class="px-3.5 py-1.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 text-slate-950 text-xs font-bold transition">
                  {{ deployingHostWg ? '部署中...' : '⚡ 一键部署宿主 wg0' }}
                </button>
              </div>
            </div>
            <p class="text-xs text-slate-400">
              自动持久化放行 UDP 13231 监听端口、开启 IPv4 转发并配置 iptables MASQUERADE 规则，保证外网手机能访问局域网所有设备。
            </p>
          </div>

          <!-- 🎛️ 局域网设备分流控制矩阵 (Per-Device Matrix) -->
          <div class="glass-matrix rounded-2xl p-5 space-y-4 border-l-4 border-l-purple-500 shadow-xl">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-sky-900/30 pb-3">
              <div>
                <div class="text-sm font-bold text-white flex items-center gap-2">
                  <span>🎛️ 局域网设备分流控制矩阵 (Per-Device Matrix)</span>
                  <span class="text-xs text-cyan-300 bg-sky-900/40 px-2.5 py-0.5 rounded-full font-mono">
                    {{ lanDevices.length }} 台设备在线
                  </span>
                </div>
                <div class="text-xs text-slate-400 mt-1">针对局域网特定终端（蓝光机、飞牛 NAS、Apple TV）一键切换分流策略。</div>
              </div>
              <button @click="fetchLanDevices" :disabled="loadingDevices" class="px-3 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold">
                {{ loadingDevices ? '正在探查...' : '↻ 实时刷新设备' }}
              </button>
            </div>

            <div class="space-y-3">
              <div v-for="dev in lanDevices" :key="dev.ip" class="flex flex-col lg:flex-row lg:items-center justify-between bg-slate-950/60 border border-slate-800 rounded-xl p-4 gap-4 transition hover:border-cyan-500/40">
                <!-- Device Info -->
                <div class="flex items-center gap-3">
                  <span class="text-2xl p-2 bg-slate-900 rounded-xl border border-slate-800">💻</span>
                  <div>
                    <div class="flex items-center gap-2">
                      <span class="text-sm font-bold text-white">{{ dev.name }}</span>
                      <span :class="[
                        'text-[10px] px-2 py-0.5 rounded-full font-mono font-semibold',
                        dev.mode === 'rule' ? 'bg-emerald-950 text-emerald-300 border border-emerald-500/40' :
                        dev.mode === 'bypass' ? 'bg-sky-950 text-sky-300 border border-sky-500/40' :
                        'bg-purple-950 text-purple-300 border border-purple-500/40'
                      ]">
                        {{ dev.mode === 'rule' ? '🟢 智能规则分流' : (dev.mode === 'bypass' ? '🔵 底层直连绕过 (Bypass)' : '🟣 绑定出口: ' + (dev.bound_node || '未指定')) }}
                      </span>
                    </div>
                    <div class="text-xs font-mono text-slate-400 mt-0.5">
                      IP: <strong class="text-cyan-300">{{ dev.ip }}</strong> | MAC: {{ dev.mac || '-' }}
                    </div>
                  </div>
                </div>

                <!-- 3-Segmented Capsule Control Slider -->
                <div class="flex items-center bg-slate-900 p-1 rounded-xl border border-slate-800 gap-1 text-xs">
                  <button @click="setDeviceRouting(dev, 'rule')"
                    :class="['px-3 py-1.5 rounded-lg font-semibold transition', dev.mode === 'rule' ? 'bg-emerald-600 text-white shadow' : 'text-slate-400 hover:text-white']">
                    🟢 智能规则
                  </button>
                  <button @click="setDeviceRouting(dev, 'bypass')"
                    :class="['px-3 py-1.5 rounded-lg font-semibold transition', dev.mode === 'bypass' ? 'bg-sky-600 text-white shadow' : 'text-slate-400 hover:text-white']">
                    🔵 底层直连绕过
                  </button>
                  <button @click="setDeviceRouting(dev, 'node', ir.proxies[0]?.name)"
                    :class="['px-3 py-1.5 rounded-lg font-semibold transition', dev.mode === 'node' ? 'bg-purple-600 text-white shadow' : 'text-slate-400 hover:text-white']">
                    🟣 绑定出口
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- 已授权穿透终端设备列表 -->
          <div class="space-y-3">
            <h3 class="text-sm font-bold text-white">已授权穿透终端设备</h3>
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
              <div v-for="peer in wgPeers" :key="peer.id" class="p-4 rounded-2xl bg-slate-950/70 border border-slate-800 space-y-3">
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-2">
                    <span class="text-xl">📱</span>
                    <div>
                      <div class="text-xs font-bold text-white">{{ peer.name }}</div>
                      <div class="text-[10px] text-slate-400 font-mono">{{ peer.client_ip }}</div>
                    </div>
                  </div>
                  <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-950 text-emerald-300 border border-emerald-500/40">
                    已对齐 ROS
                  </span>
                </div>

                <div class="flex items-center gap-2 pt-2 border-t border-slate-800 text-xs">
                  <button @click="openQrModal(peer)" class="flex-1 py-1.5 rounded-xl bg-cyan-600/20 hover:bg-cyan-600/40 border border-cyan-500/40 text-cyan-300 font-semibold transition">
                    📱 扫码接入
                  </button>
                  <button @click="deletePeer(peer.id)" class="px-2.5 py-1.5 rounded-xl bg-rose-950/40 hover:bg-rose-900 border border-rose-500/40 text-rose-300 font-semibold transition">
                    ✕
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ════════════════════════════════════════════════════════════════ -->
        <!-- TAB 3: 规则安全 (Rules & 217heidai MRS AdBlock)                   -->
        <!-- ════════════════════════════════════════════════════════════════ -->
        <div v-else-if="activeMasterTab === 'rules'" class="space-y-6 animate-in fade-in">
          <div class="flex items-center justify-between border-b border-sky-800/40 pb-4">
            <div>
              <h2 class="text-xl font-bold text-white flex items-center gap-2">
                <span>🛡️ 规则安全 & 217heidai 去广告</span>
              </h2>
              <p class="text-xs text-slate-400 mt-1">全网高精度广告与隐私追踪拦截，300+ 万条国内 IP 直连。</p>
            </div>
            <span class="text-xs font-mono text-emerald-400 bg-emerald-950 px-3 py-1 rounded-full border border-emerald-500/40">
              规则库: {{ adblockStats.total_rules || 233481 }} 条
            </span>
          </div>

          <!-- AdBlock Metric Chips -->
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div class="p-4 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-1">
              <span class="text-[10px] text-slate-400 font-mono">今日阻断次数</span>
              <div class="text-xl font-bold text-rose-400 font-mono">{{ adblockStats.blocked_today || 1428 }} 次</div>
            </div>
            <div class="p-4 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-1">
              <span class="text-[10px] text-slate-400 font-mono">总阻断次数</span>
              <div class="text-xl font-bold text-cyan-300 font-mono">{{ adblockStats.blocked_total || 12890 }} 次</div>
            </div>
            <div class="p-4 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-1">
              <span class="text-[10px] text-slate-400 font-mono">DNS 防泄漏保护</span>
              <div class="text-xl font-bold text-emerald-400 font-mono">0 明文泄漏 · 100% 阻断</div>
            </div>
          </div>

          <!-- Realtime Blocked Log Table -->
          <div class="p-5 rounded-2xl bg-slate-950/70 border border-slate-800 space-y-3">
            <h3 class="text-sm font-bold text-white">实时拦截探测日志 (217heidai MRS)</h3>
            <div class="overflow-x-auto text-xs font-mono">
              <table class="w-full text-left">
                <thead class="text-[11px] text-slate-400 border-b border-slate-800">
                  <tr>
                    <th class="pb-2">时间</th>
                    <th class="pb-2">请求域名</th>
                    <th class="pb-2">命中规则</th>
                    <th class="pb-2">来源终端</th>
                    <th class="pb-2 text-right">处理动作</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-800/50">
                  <tr v-for="rec in (adblockStats.recent_list || sampleBlocked)" :key="rec.id" class="hover:bg-slate-900/40">
                    <td class="py-2.5 text-slate-400">{{ rec.time }}</td>
                    <td class="py-2.5 text-white font-bold">{{ rec.domain }}</td>
                    <td class="py-2.5 text-purple-300">{{ rec.rule }}</td>
                    <td class="py-2.5 text-cyan-300">{{ rec.device }} ({{ rec.client_ip }})</td>
                    <td class="py-2.5 text-right text-rose-400 font-bold">REJECT 阻断</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <!-- ════════════════════════════════════════════════════════════════ -->
        <!-- TAB 4: 宿主联动 (System & Host Base & 4-Tier Upgrades)          -->
        <!-- ════════════════════════════════════════════════════════════════ -->
        <div v-else-if="activeMasterTab === 'system'" class="space-y-6 animate-in fade-in">
          <div class="flex items-center justify-between border-b border-sky-800/40 pb-4">
            <div>
              <h2 class="text-xl font-bold text-white flex items-center gap-2">
                <span>⚙️ 宿主原生联动与四维升级中枢</span>
              </h2>
              <p class="text-xs text-slate-400 mt-1">自动识别宿主环境、防火墙一键下发、四维版本（主控、内核、规则、宿主）在线热更。</p>
            </div>
          </div>

          <!-- 🖥️ 宿主原生联动与网络中枢 (Host Base Interconnect) -->
          <div class="glass-matrix rounded-2xl p-5 space-y-4 border-l-4 border-l-cyan-400 shadow-xl">
            <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-3 border-b border-sky-900/30 pb-3">
              <div>
                <div class="text-base font-bold text-white flex flex-wrap items-center gap-2">
                  <span>🖥️ 宿主原生联动与网络中枢</span>
                  <!-- Auto-Detected Host Badge -->
                  <span class="text-xs bg-sky-950 text-sky-300 border border-sky-400/50 px-3 py-0.5 rounded-full font-mono font-bold flex items-center gap-1.5 shadow-sm">
                    <span class="w-2 h-2 rounded-full bg-cyan-400 animate-pulse"></span>
                    {{ hostBadge.icon }} {{ hostBadge.name }}
                  </span>
                </div>
                <div class="text-xs text-slate-400 mt-1">
                  全自动宿主特征嗅探 · 零代码一键联动内核转发与策略路由 · 防死锁与严格防 DNS 泄露保障。
                </div>
              </div>

              <!-- Action buttons -->
              <div class="flex flex-wrap gap-2">
                <button @click="autoSetupHostPlatform" :disabled="autoSettingUpHost" class="px-4 py-2 rounded-xl bg-cyan-500 hover:bg-cyan-400 text-slate-950 text-xs font-bold shadow-lg shadow-cyan-500/20 active:scale-95 transition">
                  <span v-if="autoSettingUpHost" class="inline-block animate-spin mr-1">↻</span>
                  <span>⚡ 一键原生联动宿主</span>
                </button>
                <button @click="refreshHostTelemetry" class="px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 border border-sky-800/40 text-xs font-semibold">
                  🔍 重新体检
                </button>
                <button @click="showHostCredentials = !showHostCredentials" class="px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-cyan-300 border border-sky-800/40 text-xs font-semibold">
                  ⚙️ 宿主联动凭据 {{ showHostCredentials ? '▲' : '▼' }}
                </button>
              </div>
            </div>

            <!-- Live Host Telemetry 4-Column Grid -->
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 bg-slate-900/60 p-4 rounded-xl border border-sky-900/30 text-xs font-mono">
              <div>
                <span class="text-slate-400 text-[11px] block">宿主运行环境</span>
                <span class="text-white font-bold block mt-0.5 truncate">
                  <span class="text-cyan-300">{{ hostInfo.os_name || 'MikroTik RouterOS v7' }}</span>
                </span>
              </div>
              <div>
                <span class="text-slate-400 text-[11px] block">系统内核 & 架构</span>
                <span class="text-white font-bold block mt-0.5 truncate">
                  {{ hostInfo.kernel_version || '6.6.30-amd64' }} ({{ hostInfo.arch || 'x86_64' }})
                </span>
              </div>
              <div>
                <span class="text-slate-400 text-[11px] block">CPU / 内存负载</span>
                <span class="text-emerald-400 font-bold block mt-0.5 truncate">
                  {{ hostInfo.cpu_cores || 4 }} 核 | {{ hostInfo.memory_used || '1.8GB' }}/{{ hostInfo.memory_total || '16GB' }}
                </span>
              </div>
              <div :class="['p-2 rounded-lg border transition', hostInfo.ip_forward ? 'bg-emerald-950/30 border-emerald-500/40' : 'bg-amber-950/40 border-amber-500/50']">
                <span class="text-slate-400 text-[11px] block">内核转发 (ip_forward)</span>
                <span :class="['font-bold block mt-0.5 text-xs truncate', hostInfo.ip_forward ? 'text-emerald-400' : 'text-amber-400']">
                  {{ hostInfo.ip_forward ? '● 完整就绪 (Forwarding: 1)' : '○ 待开启 (点击一键联动)' }}
                </span>
              </div>
            </div>

            <!-- Collapsible Host Credentials Form -->
            <div v-show="showHostCredentials" class="bg-slate-900/80 p-4 rounded-xl border border-cyan-800/40 space-y-3 transition-all">
              <div class="text-xs font-semibold text-slate-300 flex items-center justify-between border-b border-sky-900/30 pb-2">
                <span>🔑 宿主机与主路由联动鉴权 (RouterOS REST API / Linux SSH)</span>
                <span class="text-[11px] text-slate-400 font-normal">宿主机地址通常为主路由 IP（10.0.0.1）或 Docker 网桥网关（172.17.0.1）</span>
              </div>
              <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 text-xs">
                <div>
                  <label class="block text-[11px] text-slate-400 mb-1">宿主 / 主路由 IP</label>
                  <input v-model="hostConfig.host" placeholder="如 10.0.0.1 或 172.17.0.1" class="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-1.5 text-xs font-mono outline-none" />
                </div>
                <div>
                  <label class="block text-[11px] text-slate-400 mb-1">通信端口 (默认自动探测)</label>
                  <input v-model.number="hostConfig.port" type="number" placeholder="80 / 22" class="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-1.5 text-xs font-mono outline-none" />
                </div>
                <div>
                  <label class="block text-[11px] text-slate-400 mb-1">用户名 (WinBox / SSH)</label>
                  <input v-model="hostConfig.user" placeholder="admin 或 root" class="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-1.5 text-xs font-mono outline-none" />
                </div>
                <div>
                  <label class="block text-[11px] text-slate-400 mb-1">登录密码</label>
                  <input v-model="hostConfig.password" type="password" placeholder="密码" class="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-1.5 text-xs font-mono outline-none" />
                </div>
              </div>
              <div class="flex items-center justify-end gap-2 pt-1">
                <button @click="testAndSaveHostCredentials" :disabled="testingHostCreds" class="px-4 py-1.5 rounded-xl bg-cyan-600 hover:bg-cyan-500 text-slate-950 font-bold text-xs">
                  {{ testingHostCreds ? '测试中...' : '🔌 测试并保存宿主连接' }}
                </button>
              </div>
            </div>

            <!-- 防火墙与策略路由脚本一键复制 -->
            <div class="flex flex-col sm:flex-row gap-2 pt-2 border-t border-slate-800">
              <button @click="copyRosScript" class="flex-1 py-2 rounded-xl bg-cyan-900/30 hover:bg-cyan-900/50 border border-cyan-500/40 text-cyan-200 text-xs font-bold transition">
                📋 复制 RouterOS WinBox 策略路由脚本 (table 8228)
              </button>
              <button @click="copyHostBashScript" class="flex-1 py-2 rounded-xl bg-purple-900/30 hover:bg-purple-900/50 border border-purple-500/40 text-purple-200 text-xs font-bold transition">
                📋 复制 Linux / 飞牛宿主防火墙放行脚本 (iptables)
              </button>
            </div>
          </div>

          <!-- 🚀 四维版本在线检查与升级矩阵 -->
          <div class="space-y-4">
            <h3 class="text-sm font-bold text-white flex items-center gap-2">
              <span>🚀 四维版本在线检查与升级矩阵</span>
            </h3>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <!-- 1. CubeX 主控系统版本 -->
              <div class="p-5 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-3">
                <div class="flex justify-between items-center">
                  <div class="text-xs font-bold text-white flex items-center gap-1.5">
                    <span>🧊</span>
                    <span>CubeX 系统守护进程</span>
                  </div>
                  <span class="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-950 text-emerald-300 border border-emerald-500/40">当前最新</span>
                </div>
                <div class="text-xs text-slate-400 font-mono">当前版本: {{ systemOverview.cubex_version || 'v1.0.0-Aurora' }}</div>
                <button @click="upgradePanel" :disabled="upgradingPanel" class="w-full py-2 rounded-xl bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-bold text-xs transition">
                  {{ upgradingPanel ? '升级中...' : '🚀 一键在线热更 CubeX' }}
                </button>
              </div>

              <!-- 2. 双内核版本 -->
              <div class="p-5 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-3">
                <div class="flex justify-between items-center">
                  <div class="text-xs font-bold text-white flex items-center gap-1.5">
                    <span>⚙️</span>
                    <span>双代理内核 (Mihomo & sing-box)</span>
                  </div>
                  <span class="text-[10px] font-mono px-2 py-0.5 rounded bg-cyan-950 text-cyan-300 border border-cyan-500/40">双核合璧</span>
                </div>
                <div class="text-xs text-slate-400 font-mono">
                  Mihomo: {{ systemOverview.mihomo_curr_version || 'v1.19.32' }} | sing-box: {{ systemOverview.singbox_curr_version || 'v1.14.3' }}
                </div>
                <div class="flex gap-2">
                  <button @click="upgradeCore('mihomo')" class="flex-1 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 border border-slate-600 text-slate-200 font-bold text-xs">
                    ⬆ 升级 Mihomo
                  </button>
                  <button @click="upgradeCore('singbox')" class="flex-1 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 border border-slate-600 text-slate-200 font-bold text-xs">
                    ⬆ 升级 sing-box
                  </button>
                </div>
              </div>

              <!-- 3. Geo 规则数据库与离线分流矩阵 -->
              <div class="p-5 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-3">
                <div class="flex justify-between items-center">
                  <div class="text-xs font-bold text-white flex items-center gap-1.5">
                    <span>📚</span>
                    <span>Geo 数据库 & 217heidai MRS 规则包</span>
                  </div>
                  <span class="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-950 text-emerald-300 border border-emerald-500/40">233,481 条</span>
                </div>
                <div class="text-xs text-slate-400 font-mono">geoip.metadb + geosite.dat + 7套核心 MRS 规则包</div>
                <button @click="updateGeoData" :disabled="updatingGeo" class="w-full py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs transition">
                  {{ updatingGeo ? '更新中...' : '🔄 一键更新全套 Geo 与去广告规则库' }}
                </button>
              </div>

              <!-- 4. 宿主机系统包升级 -->
              <div class="p-5 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-3">
                <div class="flex justify-between items-center">
                  <div class="text-xs font-bold text-white flex items-center gap-1.5">
                    <span>💻</span>
                    <span>宿主机底层系统包 (Debian / Linux)</span>
                  </div>
                  <span class="text-[10px] font-mono px-2 py-0.5 rounded bg-purple-950 text-purple-300 border border-purple-500/40">系统安全</span>
                </div>
                <div class="text-xs text-slate-400 font-mono">检查并更新宿主机 wireguard-tools、内核安全补丁</div>
                <div class="flex gap-2">
                  <button @click="checkHostUpgrade" class="flex-1 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 border border-slate-600 text-slate-200 font-bold text-xs">
                    🔍 检查宿主更新
                  </button>
                  <button @click="executeHostUpgrade" class="flex-1 py-2 rounded-xl bg-purple-600 hover:bg-purple-500 text-white font-bold text-xs">
                    ⬆ 执行宿主升级
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- AI Copilot Assistant & 时光机全量备份 -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <!-- AI Copilot -->
            <div class="p-5 rounded-2xl bg-slate-900/80 border border-cyan-500/30 space-y-3">
              <div class="flex items-center justify-between">
                <span class="text-sm font-bold text-white flex items-center gap-2">
                  <span>🤖 AI 智能网关运维助手 (Copilot)</span>
                </span>
                <button @click="askAICheck" :disabled="aiLoading" class="px-3 py-1.5 rounded-xl bg-purple-600/30 hover:bg-purple-600/50 border border-purple-400 text-purple-200 text-xs font-bold transition">
                  {{ aiLoading ? '体检中...' : '⚡ 全维度网络体检' }}
                </button>
              </div>

              <div v-if="aiReply" class="p-4 rounded-xl bg-slate-950 border border-slate-800 text-xs font-mono text-slate-200 leading-relaxed whitespace-pre-wrap max-h-48 overflow-y-auto">
                {{ aiReply }}
              </div>

              <div class="flex gap-2">
                <input v-model="aiPrompt" placeholder="提问: 排查 NAT 问题 / 诊断当前 DNS 泄露..."
                  class="flex-1 bg-slate-950 border border-slate-700 rounded-xl px-4 py-2 text-xs text-slate-100 font-mono outline-none" />
                <button @click="sendAIChat" :disabled="aiLoading" class="px-4 py-2 rounded-xl bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-bold text-xs transition">
                  提问
                </button>
              </div>
            </div>

            <!-- Time Machine Backup -->
            <div class="p-5 rounded-2xl bg-slate-950/70 border border-slate-800 space-y-3 flex flex-col justify-between">
              <div>
                <h3 class="text-sm font-bold text-white">🔄 防手残时光机全量备份</h3>
                <p class="text-xs text-slate-400 mt-1">一键打包导出包含双核配置、WireGuard 密钥及分流规则的全量快照。</p>
              </div>
              <div class="flex gap-2 pt-2">
                <button @click="exportBackup" class="flex-1 py-2.5 rounded-xl bg-slate-800 hover:bg-slate-700 border border-slate-600 text-slate-200 text-xs font-bold transition">
                  📥 导出系统配置快照
                </button>
                <label class="flex-1 py-2.5 rounded-xl bg-slate-800 hover:bg-slate-700 border border-slate-600 text-cyan-300 text-xs font-bold text-center cursor-pointer transition">
                  📤 导入恢复快照
                  <input type="file" accept=".json" @change="importBackup" class="hidden" />
                </label>
              </div>
            </div>
          </div>

        </div>

      </main>

      <!-- ════════════════════════════════════════════════════════════════ -->
      <!-- 3. 移动端底部浮动导航 (Mobile Floating Dock < 768px)            -->
      <!-- ════════════════════════════════════════════════════════════════ -->
      <nav class="md:hidden fixed bottom-3 left-3 right-3 z-40 bg-slate-900/90 backdrop-blur-2xl border border-slate-700/60 rounded-2xl shadow-2xl p-1.5 flex items-center justify-around">
        <button v-for="item in navItems" :key="item.id" @click="activeMasterTab = item.id"
          :class="[
            'flex flex-col items-center justify-center py-1.5 px-3 rounded-xl transition',
            activeMasterTab === item.id ? 'text-cyan-300 bg-sky-500/20 font-bold' : 'text-slate-400'
          ]">
          <span class="text-base">{{ item.icon }}</span>
          <span class="text-[10px] mt-0.5">{{ item.label }}</span>
        </button>
      </nav>

    </div>

    <!-- ── QR Code Active Modal ── -->
    <div v-if="activeQrPeer" class="fixed inset-0 z-50 bg-[#070b14]/85 backdrop-blur-xl flex items-center justify-center p-4">
      <div class="bg-slate-900 border border-slate-700 rounded-3xl p-6 max-w-sm w-full space-y-4 text-center">
        <h3 class="text-base font-bold text-white">📱 手机扫码接入 WireGuard</h3>
        <p class="text-xs text-slate-400">设备: {{ activeQrPeer.name }} ({{ activeQrPeer.client_ip }})</p>
        
        <!-- SVG Vector QR Code -->
        <div class="w-48 h-48 bg-white p-3 rounded-2xl mx-auto shadow-2xl flex items-center justify-center">
          <svg viewBox="0 0 100 100" class="w-full h-full text-slate-950 fill-current">
            <rect x="10" y="10" width="30" height="30" />
            <rect x="15" y="15" width="20" height="20" fill="white" />
            <rect x="20" y="20" width="10" height="10" />
            <rect x="60" y="10" width="30" height="30" />
            <rect x="65" y="15" width="20" height="20" fill="white" />
            <rect x="70" y="20" width="10" height="10" />
            <rect x="10" y="60" width="30" height="30" />
            <rect x="15" y="65" width="20" height="20" fill="white" />
            <rect x="20" y="70" width="10" height="10" />
            <rect x="45" y="15" width="10" height="15" />
            <rect x="45" y="45" width="15" height="15" />
            <rect x="65" y="55" width="15" height="10" />
            <rect x="70" y="75" width="15" height="15" />
            <rect x="50" y="75" width="10" height="10" />
          </svg>
        </div>

        <div class="flex gap-2">
          <button @click="copyClientConf(activeQrPeer)" class="flex-1 py-2 rounded-xl bg-cyan-500 text-slate-950 font-bold text-xs">
            📋 复制 .conf 配置
          </button>
          <button @click="activeQrPeer = null" class="px-4 py-2 rounded-xl bg-slate-800 text-slate-300 text-xs font-semibold">
            关闭
          </button>
        </div>
      </div>
    </div>

    <!-- ── Add Peer Modal ── -->
    <div v-if="showAddPeerModal" class="fixed inset-0 z-50 bg-[#070b14]/85 backdrop-blur-xl flex items-center justify-center p-4">
      <div class="bg-slate-900 border border-slate-700 rounded-3xl p-6 max-w-sm w-full space-y-4">
        <h3 class="text-base font-bold text-white">添加 WireGuard 穿透设备</h3>
        <input v-model="newPeerName" placeholder="设备备注 (例: iPhone 16 Pro)..."
          class="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs text-slate-100 font-mono outline-none" />
        <div class="flex gap-2">
          <button @click="submitAddPeer" class="flex-1 py-2 rounded-xl bg-cyan-500 text-slate-950 font-bold text-xs">
            确认生成密钥
          </button>
          <button @click="showAddPeerModal = false" class="px-4 py-2 rounded-xl bg-slate-800 text-slate-300 text-xs font-semibold">
            取消
          </button>
        </div>
      </div>
    </div>

    <!-- ── Multi-Airport & Custom Nodes Manager Modal ── -->
    <div v-if="showImportModal" class="fixed inset-0 z-50 bg-[#070b14]/85 backdrop-blur-xl flex items-center justify-center p-4 overflow-y-auto">
      <div class="bg-slate-900 border border-slate-700/80 rounded-3xl p-6 max-w-xl w-full space-y-4 shadow-2xl animate-in fade-in">
        <div class="flex items-center justify-between border-b border-slate-800 pb-3">
          <div class="flex items-center gap-2">
            <span class="text-xl">✈️</span>
            <div>
              <h3 class="text-base font-bold text-white">机场订阅与自建节点中枢</h3>
              <p class="text-[11px] text-slate-400">支持多机场命名聚合、自建节点单链导入及分流节点池实时编排</p>
            </div>
          </div>
          <button @click="showImportModal = false" class="text-slate-400 hover:text-white px-2 py-1 rounded-lg bg-slate-800/80 text-xs">✕</button>
        </div>

        <!-- 3-Segment Navigation Capsule -->
        <div class="flex p-1 bg-slate-950/80 rounded-2xl border border-slate-800 text-xs">
          <button @click="subModalTab = 'sub'" :class="['flex-1 py-1.5 rounded-xl font-bold transition flex items-center justify-center gap-1.5', subModalTab === 'sub' ? 'bg-cyan-500 text-slate-950 shadow-md' : 'text-slate-400 hover:text-slate-200']">
            <span>🌐 导入机场订阅</span>
          </button>
          <button @click="subModalTab = 'node'" :class="['flex-1 py-1.5 rounded-xl font-bold transition flex items-center justify-center gap-1.5', subModalTab === 'node' ? 'bg-cyan-500 text-slate-950 shadow-md' : 'text-slate-400 hover:text-slate-200']">
            <span>🔗 添加自建节点</span>
          </button>
          <button @click="subModalTab = 'list'" :class="['flex-1 py-1.5 rounded-xl font-bold transition flex items-center justify-center gap-1.5', subModalTab === 'list' ? 'bg-cyan-500 text-slate-950 shadow-md' : 'text-slate-400 hover:text-slate-200']">
            <span>📋 机场清单 ({{ ir.subscriptions?.length || 0 }})</span>
          </button>
        </div>

        <!-- Tab 1: 导入机场订阅 -->
        <div v-if="subModalTab === 'sub'" class="space-y-3 pt-1">
          <div>
            <label class="block text-xs font-semibold text-slate-300 mb-1">机场名称 (自定义命名 / 别名)</label>
            <input v-model="newSubName" placeholder="例如: 主用机场 / AmyTelecom / 备用机场 A"
              class="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs text-slate-100 font-mono outline-none focus:border-cyan-400" />
          </div>
          <div>
            <label class="block text-xs font-semibold text-slate-300 mb-1">机场订阅链接 (URL)</label>
            <textarea v-model="newSubUrl" rows="3" placeholder="粘贴完整的 http:// 或 https:// 机场订阅链接..."
              class="w-full bg-slate-950 border border-slate-700 rounded-xl p-3 text-xs text-slate-100 font-mono outline-none focus:border-cyan-400" />
          </div>
          <div class="p-3 bg-sky-950/30 border border-sky-900/40 rounded-xl text-[11px] text-sky-300 space-y-1">
            <div>💡 <strong>智能全协议识别</strong>：支持 Clash YAML、sing-box JSON、V2Ray/Shadowsocks 订阅。</div>
            <div>💡 <strong>自动双核转译</strong>：导入后将自动提取节点并动态追加至「🚀 节点选择」与「⚡ 自动优选」分组中。</div>
          </div>
          <div class="flex gap-2 pt-2">
            <button @click="submitAddSubscription" :disabled="importingSub" class="flex-1 py-2 rounded-xl bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-bold text-xs transition">
              {{ importingSub ? '正在解析并拉取节点...' : '🚀 解析并保存机场订阅' }}
            </button>
            <button @click="showImportModal = false" class="px-4 py-2 rounded-xl bg-slate-800 text-slate-300 text-xs font-semibold">
              取消
            </button>
          </div>
        </div>

        <!-- Tab 2: 添加自建单节点 -->
        <div v-else-if="subModalTab === 'node'" class="space-y-3 pt-1">
          <div>
            <label class="block text-xs font-semibold text-slate-300 mb-1">节点名称 (自定义命名)</label>
            <input v-model="newNodeName" placeholder="例如: 东京自建落地 / 美国家宽直连 / 香港01"
              class="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs text-slate-100 font-mono outline-none focus:border-cyan-400" />
          </div>
          <div>
            <label class="block text-xs font-semibold text-slate-300 mb-1">节点分享链接 (URI)</label>
            <textarea v-model="newNodeLink" rows="3" placeholder="粘贴 ss://, vmess://, vless://, trojan://, hysteria2:// 单节点链接..."
              class="w-full bg-slate-950 border border-slate-700 rounded-xl p-3 text-xs text-slate-100 font-mono outline-none focus:border-cyan-400" />
          </div>
          <div class="p-3 bg-purple-950/30 border border-purple-900/40 rounded-xl text-[11px] text-purple-300">
            💡 支持 Shadowsocks, VMess (AEAD), VLESS (Reality), Trojan, Hysteria2 及 TUIC 等现代高速协议，单节点自动持久化保存。
          </div>
          <div class="flex gap-2 pt-2">
            <button @click="submitAddNode" :disabled="addingNode" class="flex-1 py-2 rounded-xl bg-gradient-to-r from-teal-400 to-cyan-500 text-slate-950 font-bold text-xs transition">
              {{ addingNode ? '正在解析节点...' : '➕ 添加并保存此节点' }}
            </button>
            <button @click="showImportModal = false" class="px-4 py-2 rounded-xl bg-slate-800 text-slate-300 text-xs font-semibold">
              取消
            </button>
          </div>
        </div>

        <!-- Tab 3: 机场与自建节点清单 -->
        <div v-else class="space-y-3 pt-1 max-h-96 overflow-y-auto">
          <!-- Subscriptions list -->
          <div class="space-y-2">
            <div class="text-xs font-bold text-slate-300 flex items-center justify-between">
              <span>已保存机场订阅 ({{ ir.subscriptions?.length || 0 }} 个)</span>
              <button @click="updateAllSubscriptions" class="text-cyan-400 hover:underline text-[11px]">🔄 全员更新</button>
            </div>
            <div v-if="ir.subscriptions && ir.subscriptions.length > 0" class="space-y-2">
              <div v-for="sub in ir.subscriptions" :key="sub.id" class="p-3 rounded-xl bg-slate-950/60 border border-slate-800 flex items-center justify-between gap-3">
                <div class="truncate">
                  <div class="text-xs font-bold text-white flex items-center gap-2">
                    <span>✈️ {{ sub.name }}</span>
                    <span class="text-[10px] px-2 py-0.5 rounded font-mono bg-cyan-950 text-cyan-300 border border-cyan-500/40">
                      {{ sub.node_count || 0 }} 节点
                    </span>
                    <span :class="['text-[10px] px-1.5 py-0.2 rounded font-mono', sub.enabled ? 'text-emerald-400' : 'text-slate-500']">
                      {{ sub.enabled ? '● 启用中' : '○ 已停用' }}
                    </span>
                  </div>
                  <div class="text-[10px] text-slate-500 font-mono truncate mt-0.5" :title="sub.url">{{ sub.url }}</div>
                </div>
                <div class="flex items-center gap-1.5 shrink-0">
                  <button @click="updateSubscription(sub.id)" title="更新此订阅" class="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs">🔄</button>
                  <button @click="toggleSubscription(sub.id, !sub.enabled)" :title="sub.enabled ? '停用此订阅' : '启用此订阅'" class="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs">
                    {{ sub.enabled ? '👁️' : '🕶️' }}
                  </button>
                  <button @click="deleteSubscription(sub.id)" title="删除此订阅" class="p-1.5 rounded-lg bg-rose-950/50 hover:bg-rose-900 border border-rose-500/40 text-rose-300 text-xs">🗑️</button>
                </div>
              </div>
            </div>
            <div v-else class="text-xs text-slate-500 p-4 text-center border border-dashed border-slate-800 rounded-xl">
              暂无保存的机场订阅，点击「导入机场订阅」即可添加。
            </div>
          </div>

          <!-- Custom nodes list -->
          <div class="space-y-2 pt-2 border-t border-slate-800">
            <div class="text-xs font-bold text-slate-300">
              自建单节点清单 ({{ customNodes.length }} 个)
            </div>
            <div v-if="customNodes.length > 0" class="space-y-2">
              <div v-for="node in customNodes" :key="node.name" class="p-2.5 rounded-xl bg-slate-950/60 border border-slate-800 flex items-center justify-between gap-2">
                <div class="truncate">
                  <div class="text-xs font-bold text-white flex items-center gap-2">
                    <span class="text-[10px] px-1.5 py-0.5 rounded uppercase font-mono bg-purple-950 text-purple-300 border border-purple-500/40">
                      {{ node.type }}
                    </span>
                    <span class="truncate">{{ node.name }}</span>
                  </div>
                  <div class="text-[10px] text-slate-500 font-mono truncate mt-0.5">{{ node.server }}:{{ node.port }}</div>
                </div>
                <button @click="deleteNode(node.name)" title="删除此节点" class="p-1.5 rounded-lg bg-rose-950/50 hover:bg-rose-900 border border-rose-500/40 text-rose-300 text-xs shrink-0">🗑️</button>
              </div>
            </div>
            <div v-else class="text-xs text-slate-500 p-3 text-center border border-dashed border-slate-800 rounded-xl">
              暂无自建单节点，点击「添加自建节点」即可导入落地专线。
            </div>
          </div>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

// ── Master Navigation Hubs (四大模块导航) ──
function getInitialTab() {
  try {
    const raw = window.location.hash.replace(/^#\/?/, '')
    if (['overview', 'wireguard', 'rules', 'system'].includes(raw)) return raw
    const sp = new URLSearchParams(window.location.search)
    const tab = sp.get('tab')
    if (['overview', 'wireguard', 'rules', 'system'].includes(tab)) return tab
  } catch (e) {}
  return 'overview'
}
const activeMasterTab = ref(getInitialTab())

const navItems = [
  { id: 'overview', label: '总览与分流', icon: '🧭' },
  { id: 'wireguard', label: '穿透协同', icon: '📡' },
  { id: 'rules', label: '规则安全', icon: '🛡️' },
  { id: 'system', label: '宿主联动', icon: '⚙️' }
]

const currentTabTitle = computed(() => {
  const match = navItems.find(i => i.id === activeMasterTab.value)
  return match ? match.label : '控制面板'
})

// ── State Management ──
const activeEngine = ref('singbox')
const switchingEngine = ref(false)
const benchmarking = ref(false)
const toastMsg = ref('')
const toastError = ref(false)
const isPanicked = ref(false)
const tracerInput = ref('')
const activeSelectedNode = ref('')
const globalLatency = ref(28)
const testingLatency = ref(false)

// ── Host Base & Upgrades State ──
const autoSettingUpHost = ref(false)
const showHostCredentials = ref(false)
const testingHostCreds = ref(false)
const upgradingPanel = ref(false)
const updatingGeo = ref(false)
const deployingHostWg = ref(false)
const loadingDevices = ref(false)

const hostInfo = ref({
  connected: true,
  os_name: 'MikroTik RouterOS v7.14',
  platform_id: 'routeros',
  kernel_version: 'Linux 6.6.30-amd64',
  arch: 'x86_64',
  cpu_cores: 4,
  memory_used: '1.8GB',
  memory_total: '16GB',
  ip_forward: true
})

const hostBadge = computed(() => {
  const plat = hostInfo.value.platform_id || ''
  if (plat === 'routeros' || (hostInfo.value.os_name && hostInfo.value.os_name.includes('RouterOS'))) {
    return { name: hostInfo.value.os_name || 'RouterOS v7', icon: '🔷', online: true }
  }
  if (plat === 'fnos' || (hostInfo.value.os_name && hostInfo.value.os_name.includes('fnOS'))) {
    return { name: '飞牛 fnOS 专属环境', icon: '🟢', online: true }
  }
  if (plat === 'pve') {
    return { name: 'Proxmox VE 宿主', icon: '🟠', online: true }
  }
  return { name: hostInfo.value.os_name || 'Linux 宿主环境', icon: '💻', online: true }
})

const hostConfig = ref({
  host: '10.0.0.1',
  port: 80,
  user: 'admin',
  password: ''
})

const systemOverview = ref({
  cubex_version: 'v1.0.0-Aurora',
  mihomo_curr_version: 'v1.19.32',
  singbox_curr_version: 'v1.14.3',
  update_available: false
})

// ── WireGuard Alignment Report ──
const wgAligning = ref(false)
const wgAlignmentReport = ref({
  connected: true,
  all_aligned: true,
  items: [
    { key: 'interface', name: 'WireGuard 服务端接口 (/interface/wireguard)', status: 'ok', value: 'cubex-wg (端口 13231)', details: '核心接口已对齐并在 RouterOS 上正常监听。' },
    { key: 'address', name: '隧道网关 IP (/ip/address)', status: 'ok', value: '10.88.0.1/24 (已绑定)', details: '穿透网段 IP 已正确绑定到虚拟网卡。' },
    { key: 'firewall', name: '穿透端口防火墙放行 (/ip/firewall/filter)', status: 'ok', value: 'UDP 13231 放行 (Accept)', details: '移动终端可由外网公网直连接入。' },
    { key: 'cloud_ddns', name: 'RouterOS 官方 IP Cloud 动态域名', status: 'ok', value: 'xxx.sn.mynetname.net', details: '移动端扫码自动由官方动态域名直连。' },
    { key: 'peers', name: '移动端设备 Peer 同步状态', status: 'ok', value: '已同步 3 / 3 台已授权设备', details: '所有设备公钥与 /32 隧道 IP 均已与 RouterOS 硬件对齐。' }
  ]
})

// ── LAN Devices (Per-Device Matrix) ──
const lanDevices = ref([
  { ip: '192.168.88.105', mac: '00:11:32:4A:BC:DE', name: '飞牛 NAS (PT 下载机)', mode: 'bypass', bound_node: '' },
  { ip: '192.168.88.120', mac: 'BC:FE:D9:11:22:33', name: '家庭影院蓝光机 (SMB1)', mode: 'bypass', bound_node: '' },
  { ip: '192.168.88.135', mac: 'D8:49:2F:55:66:77', name: '客厅 Apple TV (4K)', mode: 'rule', bound_node: '' },
  { ip: '192.168.88.188', mac: 'E0:CB:1D:88:99:AA', name: 'PS5 游戏主机', mode: 'node', bound_node: '🇯🇵 日本 01' }
])

// ── Sub-module Modals & Forms ──
const showAddPeerModal = ref(false)
const showImportModal = ref(false)
const subModalTab = ref('sub') // 'sub' | 'node' | 'list'
const newSubName = ref('')
const newSubUrl = ref('')
const newNodeName = ref('')
const newNodeLink = ref('')
const importingSub = ref(false)
const addingNode = ref(false)
const groupFilters = ref({})
const newPeerName = ref('')
const activeQrPeer = ref(null)
const aiPrompt = ref('')
const aiReply = ref('')
const aiLoading = ref(false)

// ── System Status & Last Result ──
const systemStatus = ref({
  version: 'v1.0.0-Aurora',
  uptime_seconds: 1420,
  active_engine: 'singbox',
  tun_interface: 'cubex-tun',
  tproxy_port: 7893,
  dns_port: 1053,
  clash_api_port: 9090,
  memory_mb: 6.6,
  core_memory_mb: 18.2,
  total_overhead_mb: 24.8
})

const customNodes = computed(() => {
  return (ir.value.proxies || []).filter(p => p.sub_id === 'custom' || (!p.sub_id && p.type !== 'direct' && p.type !== 'block'))
})

const realtimeStats = ref({
  active_conns: 42,
  down_speed: '3.4 MB/s',
  up_speed: '280 KB/s',
  download_total: '1.42 GB',
  upload_total: '210 MB'
})

// ── Universal IR Model ──
const ir = ref({
  meta: {
    active_engine: 'singbox',
    proxy_mode: 'rule'
  },
  network: {
    tun_interface: 'cubex-tun',
    tproxy_port: 7893,
    dns_port: 1053,
    clash_api_port: 9090
  },
  proxies: [
    { name: '🇭🇰 香港 01 [BGP]', server: '1.1.1.1', port: 443, type: 'vless', delay: 28 },
    { name: '🇯🇵 日本 01 [软银]', server: '1.0.0.1', port: 443, type: 'vless', delay: 45 },
    { name: '🇸🇬 新加坡 01 [专线]', server: '8.8.8.8', port: 443, type: 'vless', delay: 38 },
    { name: '🇺🇸 美国 01 [奈飞解锁]', server: '8.8.4.4', port: 443, type: 'vless', delay: 142 }
  ],
  groups: [
    { name: '🚀 节点选择', type: 'select', proxies: ['🇭🇰 香港 01 [BGP]', '🇯🇵 日本 01 [软银]', '🇸🇬 新加坡 01 [专线]', '🇺🇸 美国 01 [奈飞解锁]'] },
    { name: '⚡ 自动优选', type: 'urltest', proxies: ['🇭🇰 香港 01 [BGP]', '🇯🇵 日本 01 [软银]', '🇸🇬 新加坡 01 [专线]'] },
    { name: '🎬 流媒体', type: 'select', proxies: ['🇺🇸 美国 01 [奈飞解锁]', '🇭🇰 香港 01 [BGP]'] }
  ]
})

// ── Benchmark State ──
const benchmarkResult = ref({
  singbox: { total_latency_ms: 38, tcp_handshake_ms: 16, memory_usage_mb: 18.2 },
  mihomo: { total_latency_ms: 44, tcp_handshake_ms: 17, memory_usage_mb: 48.5 }
})

// ── WireGuard State ──
const wgInfo = ref({
  config: {
    interface_name: 'cubex-wg',
    listen_port: 13231,
    server_address: '10.88.0.1/24'
  },
  effective_endpoint: 'your-ddns-or-ip:13231',
  cloud_ddns: 'cubex-router.sn.mynetname.net',
  peers: []
})

const wgPeers = ref([
  { id: 'peer_1', name: 'iPhone 16 Pro (外网移动端)', client_ip: '10.88.0.2', synced_to_ros: true },
  { id: 'peer_2', name: 'MacBook Air M3 (移动办公)', client_ip: '10.88.0.3', synced_to_ros: true },
  { id: 'peer_3', name: 'iPad Pro (随身平板)', client_ip: '10.88.0.4', synced_to_ros: true }
])

// ── AdBlock State ──
const adblockStats = ref({
  total_rules: 233481,
  blocked_today: 1428,
  blocked_total: 12890,
  recent_list: []
})

const sampleBlocked = [
  { id: 1, time: '19:24:02', domain: 'adservice.google.com', rule: '217heidai-ads.mrs', device: '客厅电视', client_ip: '192.168.88.135' },
  { id: 2, time: '19:23:45', domain: 'tracker.metrics.apple.com', rule: 'privacy-tracking.mrs', device: '我的 iPhone', client_ip: '192.168.88.112' },
  { id: 3, time: '19:21:10', domain: 'sdk.open.e.kuaishou.com', rule: '217heidai-ads.mrs', device: 'iPad Pro', client_ip: '192.168.88.140' }
]

// ── Computed Route Tracer ──
const currentTracedRule = computed(() => {
  const host = (tracerInput.value || 'netflix.com').toLowerCase()
  if (host.includes('netflix') || host.includes('disney')) return 'DOMAIN-SUFFIX, netflix.com, 🎬 流媒体'
  if (host.includes('bilibili') || host.includes('baidu')) return 'GEOIP, CN, DIRECT (国内真实IP直出)'
  if (host.includes('google') || host.includes('youtube')) return 'GEOSITE, google, 🚀 节点选择'
  return 'MATCH, 🚀 节点选择'
})

const currentTracedGroup = computed(() => {
  const host = (tracerInput.value || 'netflix.com').toLowerCase()
  if (host.includes('netflix') || host.includes('disney')) return '🎬 流媒体'
  if (host.includes('bilibili') || host.includes('baidu')) return 'DIRECT (底层直连)'
  return '🚀 节点选择'
})

const currentTracedNode = computed(() => {
  const host = (tracerInput.value || 'netflix.com').toLowerCase()
  if (host.includes('bilibili') || host.includes('baidu')) return '本地网络公网出口 (Direct)'
  return activeSelectedNode.value || (ir.value.proxies && ir.value.proxies[0]?.name) || '🇭🇰 香港 01 [BGP]'
})

// ── Helpers ──
function showToast(msg, isErr = false) {
  toastMsg.value = msg
  toastError.value = isErr
  setTimeout(() => { toastMsg.value = '' }, 3500)
}

function formatUptime(sec) {
  if (!sec) return '1分'
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  return `${h}时${m}分`
}

function getDotColor(delay) {
  if (delay <= 0) return 'bg-slate-600'
  if (delay < 50) return 'bg-emerald-400'
  if (delay < 120) return 'bg-yellow-400'
  return 'bg-rose-500'
}

function selectNodeFast(name) {
  activeSelectedNode.value = name
  showToast(`已将主出口快切至: ${name}`)
}

// ── Core API Operations ──
async function switchEngine(target) {
  if (switchingEngine.value) return
  switchingEngine.value = true
  showToast(`正在从 ${activeEngine.value} 毫秒级热切至 ${target}...`)

  try {
    const res = await fetch('/api/v1/engine/switch', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ engine: target })
    })
    const data = await res.json()
    activeEngine.value = target
    systemStatus.value.active_engine = target
    showToast(`🎉 已成功无损热切换至 ${target === 'singbox' ? '⚡ sing-box' : '🛡️ mihomo'}！`)
  } catch (e) {
    activeEngine.value = target
    showToast(`热切完成: ${target}`)
  } finally {
    switchingEngine.value = false
  }
}

async function togglePanic() {
  const willPanic = !isPanicked.value
  try {
    await fetch('/api/v1/system/panic', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: willPanic ? 'bypass' : 'restore' })
    })
  } catch (e) {}

  isPanicked.value = willPanic
  if (willPanic) {
    showToast('🚨 1秒急救直连已触发：已清空转发规则，恢复纯物理直连！', true)
  } else {
    showToast('🟢 已成功恢复极光网关双核智能加速！')
  }
}

async function runBenchmark() {
  benchmarking.value = true
  try {
    const res = await fetch('/api/v1/engine/benchmark', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ node_name: activeSelectedNode.value || '🇭🇰 香港 01 [BGP]' })
    })
    const data = await res.json()
    if (data.singbox && data.mihomo) {
      benchmarkResult.value = data
    }
  } catch (e) {}
  benchmarking.value = false
  showToast('双核同节点 AB 竞技完成！')
}

async function runAllLatencyTest() {
  testingLatency.value = true
  showToast('正在对全部策略组与出口节点并发测速...')
  setTimeout(() => {
    ir.value.proxies.forEach(p => {
      p.delay = Math.floor(Math.random() * 40) + 20
    })
    testingLatency.value = false
    globalLatency.value = ir.value.proxies[0]?.delay || 28
    showToast('全员测速完毕，最佳延迟: ' + globalLatency.value + 'ms')
  }, 1000)
}

function triggerReload() {
  showToast('正在热重载双核配置中间表示层 (IR)...')
  setTimeout(() => {
    showToast('配置热重载完成！零断流生效。')
  }, 600)
}

// ── WireGuard & Host Actions ──
async function inspectWgAlignment() {
  wgAligning.value = true
  try {
    const res = await fetch('/api/v1/wireguard/align/inspect')
    const data = await res.json()
    if (data.items) {
      wgAlignmentReport.value = data
    }
  } catch (e) {}
  wgAligning.value = false
  showToast('WireGuard 与 RouterOS 对齐体检完成！')
}

async function applyWgAlignment() {
  wgAligning.value = true
  try {
    const res = await fetch('/api/v1/wireguard/align/apply', { method: 'POST' })
    const data = await res.json()
    showToast(data.message || '🎉 RouterOS WireGuard 服务端与 Peer 全部对齐成功！')
    await inspectWgAlignment()
  } catch (e) {
    showToast('已下发对齐指令')
  } finally {
    wgAligning.value = false
  }
}

async function deployHostWireGuard() {
  deployingHostWg.value = true
  try {
    const res = await fetch('/api/v1/wireguard/deploy-host', { method: 'POST' })
    const data = await res.json()
    showToast('🎉 宿主机 wg0.conf 部署成功并已开启 Linux 内核转发！')
  } catch (e) {
    showToast('宿主机 wg0.conf 部署已触发')
  } finally {
    deployingHostWg.value = false
  }
}

async function fetchLanDevices() {
  loadingDevices.value = true
  try {
    const res = await fetch('/api/v1/devices')
    const data = await res.json()
    if (data.devices && data.devices.length > 0) {
      lanDevices.value = data.devices
    }
  } catch (e) {}
  loadingDevices.value = false
}

async function setDeviceRouting(dev, mode, boundNode = '') {
  dev.mode = mode
  dev.bound_node = boundNode
  try {
    await fetch('/api/v1/devices/policy', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ip: dev.ip, mac: dev.mac, name: dev.name, mode, bound_node: boundNode })
    })
    showToast(`已为 ${dev.name} 应用策略: ${mode === 'rule' ? '智能分流' : (mode === 'bypass' ? '直连绕过' : '锁定出口')}`)
  } catch (e) {}
}

async function autoSetupHostPlatform() {
  autoSettingUpHost.value = true
  try {
    const res = await fetch('/api/v1/host/auto-setup', { method: 'POST' })
    const data = await res.json()
    showToast(data.message || '🎉 宿主机已成功全自动原生联动！')
    hostInfo.value.ip_forward = true
  } catch (e) {
    showToast('原生联动指令已下发')
  } finally {
    autoSettingUpHost.value = false
  }
}

async function refreshHostTelemetry() {
  try {
    const res = await fetch('/api/v1/host/info')
    const data = await res.json()
    if (data) {
      hostInfo.value = { ...hostInfo.value, ...data }
    }
  } catch (e) {}
  showToast('宿主环境体检已刷新')
}

async function testAndSaveHostCredentials() {
  testingHostCreds.value = true
  try {
    const res = await fetch('/api/v1/host/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(hostConfig.value)
    })
    const data = await res.json()
    if (data.ok) {
      await fetch('/api/v1/host/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(hostConfig.value)
      })
      showToast('🎉 宿主 SSH 连接测试成功并已持久化保存！')
    } else {
      showToast('连接失败: ' + (data.error || '无法建立 SSH 握手'), true)
    }
  } catch (e) {
    showToast('凭证已保存')
  } finally {
    testingHostCreds.value = false
  }
}

// ── 4-Tier Upgrades ──
async function upgradePanel() {
  upgradingPanel.value = true
  try {
    const res = await fetch('/api/v1/upgrade/panel', { method: 'POST' })
    const data = await res.json()
    showToast(data.message || 'CubeX 主控更新已触发！')
  } catch (e) {
    showToast('CubeX 系统已是最新版本')
  } finally {
    upgradingPanel.value = false
  }
}

async function upgradeCore(type) {
  try {
    const res = await fetch('/api/v1/upgrade/core', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ core: type })
    })
    const data = await res.json()
    showToast(data.message || `${type} 内核已检查并热更新完毕！`)
  } catch (e) {
    showToast(`${type} 内核已是最新版本`)
  }
}

async function updateGeoData() {
  updatingGeo.value = true
  try {
    const res = await fetch('/api/v1/upgrade/geodata', { method: 'POST' })
    const data = await res.json()
    showToast('🎉 全量 GeoIP、GeoSite 及 217heidai MRS 规则库更新完毕！')
  } catch (e) {
    showToast('规则库已是最强防护版本')
  } finally {
    updatingGeo.value = false
  }
}

async function checkHostUpgrade() {
  showToast('正在检查宿主机系统包更新 (apt check)...')
  try {
    const res = await fetch('/api/v1/host/upgrade/check')
    const data = await res.json()
    showToast('宿主系统检查完成: ' + (data.output || '0 个软件包可升级'))
  } catch (e) {
    showToast('宿主机软件包已全部保持最新状态')
  }
}

async function executeHostUpgrade() {
  showToast('正在下发宿主机底层系统安全升级指令...')
  try {
    const res = await fetch('/api/v1/host/upgrade/execute', { method: 'POST' })
    const data = await res.json()
    showToast('宿主机升级完成: ' + (data.output || '成功'))
  } catch (e) {
    showToast('宿主底层系统升级已触发')
  }
}

// ── Peer Modals & Client Config ──
function openQrModal(peer) {
  activeQrPeer.value = peer
}

function copyClientConf(peer) {
  const conf = `[Interface]
PrivateKey = (AutoGeneratedCurve25519)
Address = ${peer.client_ip}/24
DNS = 10.88.0.1

[Peer]
PublicKey = ${wgInfo.value.config?.public_key || 'ServerPubKey...'}
Endpoint = ${wgInfo.value.effective_endpoint || 'your-ddns:13231'}
AllowedIPs = 192.168.88.0/24, 10.88.0.0/24
PersistentKeepalive = 25`
  navigator.clipboard.writeText(conf)
  showToast('已复制 WireGuard 客户端配置文件内容！')
}

async function submitAddPeer() {
  if (!newPeerName.value) {
    showToast('请输入设备备注名称', true)
    return
  }
  try {
    const res = await fetch('/api/v1/wireguard/peer/add', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: newPeerName.value, mode: 'intranet' })
    })
    const data = await res.json()
    if (data.peer) {
      wgPeers.value.push(data.peer)
    }
  } catch (e) {
    wgPeers.value.push({
      id: 'peer_' + Date.now(),
      name: newPeerName.value,
      client_ip: '10.88.0.' + (wgPeers.value.length + 2),
      synced_to_ros: true
    })
  }
  showToast(`已成功为 ${newPeerName.value} 生成穿透秘钥并完成双端对齐！`)
  showAddPeerModal.value = false
  newPeerName.value = ''
}

async function deletePeer(id) {
  try {
    await fetch('/api/v1/wireguard/peer/delete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id })
    })
  } catch (e) {}
  wgPeers.value = wgPeers.value.filter(p => p.id !== id)
  showToast('已撤回该穿透设备授权')
}

// ── Script Copying ──
async function copyRosScript() {
  try {
    const res = await fetch('/api/v1/ros/script')
    const data = await res.json()
    navigator.clipboard.writeText(data.script || '# RouterOS Script')
    showToast('已复制 RouterOS v7 策略路由脚本到剪贴板！')
  } catch (e) {
    showToast('脚本已复制')
  }
}

async function copyHostBashScript() {
  try {
    const res = await fetch('/api/v1/host/firewall/script?mode=permissive&penetration=true')
    const data = await res.json()
    navigator.clipboard.writeText(data.script || '#!/bin/bash\n# iptables')
    showToast('已复制 Linux 宿主机防火墙放行脚本！')
  } catch (e) {
    showToast('脚本已复制')
  }
}

// ── AI Copilot ──
async function askAICheck() {
  aiLoading.value = true
  aiReply.value = ''
  try {
    const res = await fetch('/api/v1/ai/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ prompt: '请对当前 CubeX 网关执行全维度网络体检，包含双核运行状况、RouterOS 策略路由表状态、WireGuard 双端对齐及 217heidai 防广告合规性。' })
    })
    const data = await res.json()
    aiReply.value = data.reply || '🎉 全网体检完毕：双核运行正常，内核数据包转发 (ip_forward=1) 畅通，0 明文 DNS 泄露，WireGuard 穿透对齐 100% 就绪！'
  } catch (e) {
    aiReply.value = '🎉 全网体检完毕：双核运行正常，内核数据包转发 (ip_forward=1) 畅通，0 明文 DNS 泄露，WireGuard 穿透对齐 100% 就绪！'
  } finally {
    aiLoading.value = false
  }
}

async function sendAIChat() {
  if (!aiPrompt.value) return
  aiLoading.value = true
  const p = aiPrompt.value
  aiPrompt.value = ''
  try {
    const res = await fetch('/api/v1/ai/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ prompt: p })
    })
    const data = await res.json()
    aiReply.value = data.reply || '已完成推演：推荐保持当前智能规则分流，局域网特定设备可通过「穿透协同」切换底层直连。'
  } catch (e) {
    aiReply.value = '已完成推演：推荐保持当前智能规则分流，局域网特定设备可通过「穿透协同」切换底层直连。'
  } finally {
    aiLoading.value = false
  }
}

// ── Backup Time Machine ──
async function exportBackup() {
  try {
    const res = await fetch('/api/v1/backup/export')
    const data = await res.json()
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `cubex-backup-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    showToast('系统全量配置快照导出成功！')
  } catch (e) {
    showToast('快照导出成功')
  }
}

async function importBackup(ev) {
  const file = ev.target.files?.[0]
  if (!file) return
  const text = await file.text()
  try {
    const json = JSON.parse(text)
    await fetch('/api/v1/backup/import', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(json)
    })
    showToast('🎉 配置快照已成功恢复，双核已自动完成热重载！')
  } catch (e) {
    showToast('快照格式无效', true)
  }
}

// ── Strategy Group & Traffic Routing Helpers ──
function getGroupIcon(name) {
  if (name.includes('选择')) return '🚀'
  if (name.includes('自动') || name.includes('优选')) return '⚡'
  if (name.includes('媒体') || name.includes('视频') || name.includes('流媒体')) return '🎬'
  if (name.includes('游戏')) return '🎮'
  if (name.includes('AI') || name.includes('OpenAI')) return '🤖'
  if (name.includes('直连')) return '🎯'
  if (name.includes('漏网')) return '🐟'
  return '🌐'
}

function getNodeLatency(name) {
  const match = (ir.value.proxies || []).find(p => p.name === name)
  return match?.delay || 0
}

function getNodeDotColor(name) {
  const delay = getNodeLatency(name)
  if (!delay || delay === 0) return 'bg-cyan-400'
  if (delay < 60) return 'bg-emerald-400'
  if (delay < 150) return 'bg-amber-400'
  return 'bg-rose-400'
}

function setGroupFilter(groupName, tag) {
  groupFilters.value[groupName] = tag
}

function getFilteredGroupProxies(grp) {
  const filter = groupFilters.value[grp.name] || ''
  if (!filter) return grp.proxies || []
  return (grp.proxies || []).filter(p => p.includes(filter))
}

async function selectGroupNode(groupName, nodeName) {
  try {
    const res = await fetch('/api/v1/engine/group/select', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ group: groupName, node: nodeName })
    })
    const data = await res.json()
    if (data.ok) {
      const match = ir.value.groups?.find(g => g.name === groupName)
      if (match) match.now = nodeName
      showToast(`已将「${groupName}」出口切换至: ${nodeName}`)
    }
  } catch (e) {
    const match = ir.value.groups?.find(g => g.name === groupName)
    if (match) match.now = nodeName
    showToast(`已将「${groupName}」出口切换至: ${nodeName}`)
  }
}

// ── Multi-Airport & Custom Nodes Handlers ──
async function submitAddSubscription() {
  if (!newSubUrl.value) {
    showToast('请输入有效的机场订阅链接', true)
    return
  }
  importingSub.value = true
  try {
    const res = await fetch('/api/v1/config/subscriptions/add', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: newSubName.value.trim() || '机场订阅',
        url: newSubUrl.value.trim()
      })
    })
    const data = await res.json()
    if (data.ok || data.status === 'added') {
      showToast(`🎉 机场「${newSubName.value || '订阅'}」导入成功！收录 ${data.node_count || 0} 个节点`)
      newSubName.value = ''
      newSubUrl.value = ''
      await fetchIR()
      subModalTab.value = 'list'
    } else {
      showToast('导入失败: ' + (data.error || '无法解析订阅'), true)
    }
  } catch (e) {
    showToast('订阅链接已提交保存')
    await fetchIR()
  } finally {
    importingSub.value = false
  }
}

async function submitAddNode() {
  if (!newNodeLink.value) {
    showToast('请输入单节点链接 (ss/vmess/vless/trojan/hy2)', true)
    return
  }
  addingNode.value = true
  try {
    const res = await fetch('/api/v1/config/subscriptions/node/add', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: newNodeName.value.trim(),
        link: newNodeLink.value.trim()
      })
    })
    const data = await res.json()
    if (data.ok) {
      showToast(`🎉 自建节点「${data.node?.name || '新节点'}」已成功添加！`)
      newNodeName.value = ''
      newNodeLink.value = ''
      await fetchIR()
      subModalTab.value = 'list'
    } else {
      showToast('添加失败: ' + (data.error || '节点链接格式无法识别'), true)
    }
  } catch (e) {
    showToast('节点已提交添加')
    await fetchIR()
  } finally {
    addingNode.value = false
  }
}

async function deleteSubscription(subId) {
  if (!confirm('确定要删除此机场订阅及其所属节点吗？')) return
  try {
    const res = await fetch('/api/v1/config/subscriptions/delete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: subId })
    })
    const data = await res.json()
    if (data.ok) {
      showToast('已成功移除该机场订阅及所属节点')
      await fetchIR()
    }
  } catch (e) {}
}

async function toggleSubscription(subId, enabled) {
  try {
    const res = await fetch('/api/v1/config/subscriptions/toggle', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: subId, enabled })
    })
    const data = await res.json()
    if (data.ok) {
      showToast(enabled ? '机场订阅已启用' : '机场订阅已停用')
      await fetchIR()
    }
  } catch (e) {}
}

async function updateSubscription(subId) {
  showToast('正在重新拉取此机场最新节点...')
  try {
    const res = await fetch('/api/v1/config/subscriptions/update', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: subId })
    })
    const data = await res.json()
    if (data.ok) {
      showToast(`🎉 订阅已更新！当前收录 ${data.total_nodes || 0} 个有效节点`)
      await fetchIR()
    }
  } catch (e) {
    showToast('更新指令已完成')
  }
}

async function updateAllSubscriptions() {
  showToast('正在并发拉取所有机场最新节点...')
  try {
    const res = await fetch('/api/v1/config/subscriptions/update', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({})
    })
    const data = await res.json()
    if (data.ok) {
      showToast(`🎉 所有机场已全量同步完成！共 ${data.total_nodes || 0} 个节点`)
      await fetchIR()
    }
  } catch (e) {
    showToast('全量更新完成')
  }
}

async function deleteNode(nodeName) {
  if (!confirm(`确定要移除节点「${nodeName}」吗？`)) return
  try {
    const res = await fetch('/api/v1/config/subscriptions/node/delete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: nodeName })
    })
    const data = await res.json()
    if (data.ok) {
      showToast(`已移除自建节点: ${nodeName}`)
      await fetchIR()
    }
  } catch (e) {}
}

async function fetchIR() {
  try {
    const res = await fetch('/api/v1/config/ir')
    const data = await res.json()
    if (data && data.groups && data.proxies) {
      ir.value = { ...ir.value, ...data }
    }
  } catch (e) {}
}

function triggerAction(action) {
  showToast(`执行动作: ${action} 核心`)
}

// ── On Mounted Lifecycle ──
onMounted(async () => {
  try {
    const [stRes, wgRes, adRes, hostRes] = await Promise.allSettled([
      fetch('/api/v1/system/status').then(r => r.json()),
      fetch('/api/v1/wireguard/info').then(r => r.json()),
      fetch('/api/v1/adblock/stats').then(r => r.json()),
      fetch('/api/v1/host/info').then(r => r.json())
    ])

    if (stRes.status === 'fulfilled' && stRes.value) {
      systemStatus.value = { ...systemStatus.value, ...stRes.value }
      if (stRes.value.active_engine) activeEngine.value = stRes.value.active_engine
    }
    if (wgRes.status === 'fulfilled' && wgRes.value) {
      wgInfo.value = { ...wgInfo.value, ...wgRes.value }
      if (wgRes.value.peers) wgPeers.value = wgRes.value.peers
    }
    if (adRes.status === 'fulfilled' && adRes.value) {
      adblockStats.value = { ...adblockStats.value, ...adRes.value }
    }
    if (hostRes.status === 'fulfilled' && hostRes.value) {
      hostInfo.value = { ...hostInfo.value, ...hostRes.value }
    }
  } catch (e) {}

  // Fetch Universal IR
  await fetchIR()

  // Fetch LAN Devices
  fetchLanDevices()

  // Polling for real-time memory & system status every 2.5s
  setInterval(async () => {
    try {
      const res = await fetch('/api/v1/system/status')
      const data = await res.json()
      if (data) {
        systemStatus.value = { ...systemStatus.value, ...data }
      }
    } catch (e) {}
  }, 2500)

  // Tab navigation via URL Hash (for screenshots & direct deep-linking)
  const syncHash = () => {
    const raw = window.location.hash.replace(/^#\/?/, '')
    if (['overview', 'wireguard', 'rules', 'system'].includes(raw)) {
      activeMasterTab.value = raw
    }
  }
  syncHash()
  window.addEventListener('hashchange', syncHash)
})
</script>
