<template>
  <dialog ref="dialogRef" class="modern-dialog select-none" @click="handleBackdropClick" @cancel="closeModal">
    <div class="fixed inset-0 bg-black/80 backdrop-blur-xl flex items-center justify-center p-3 sm:p-5 overflow-y-auto animate-in fade-in" @click.self="closeModal">
      <div class="border border-white/20 rounded-3xl max-w-6xl w-full shadow-2xl flex flex-col text-slate-100 overflow-hidden relative" style="background-color: #080c16 !important;">

        <!-- Header -->
        <div class="px-4 sm:px-6 py-3 bg-white/[0.04] border-b border-white/10 flex items-center justify-between gap-3 select-none flex-wrap sm:flex-nowrap">
          <!-- Left: Status + Tabs Capsule -->
          <div class="flex items-center gap-2 sm:gap-3 shrink-0 flex-wrap">
            <div class="flex items-center gap-1.5">
              <span class="w-2.5 h-2.5 rounded-full bg-rose-500/80 inline-block shadow-sm"></span>
              <span class="w-2.5 h-2.5 rounded-full bg-amber-500/80 inline-block shadow-sm"></span>
              <span class="w-2.5 h-2.5 rounded-full bg-emerald-500/80 inline-block shadow-sm"></span>
            </div>
            <div class="h-3.5 w-px bg-slate-700/80 mx-0.5"></div>

            <!-- Tab Switcher: Logs vs Connections Tree -->
            <div class="flex items-center macos-capsule p-0.5 rounded-xl border border-white/10">
              <button @click="activeModalTab = 'logs'"
                :class="['px-3 py-1 rounded-lg text-xs font-bold transition flex items-center gap-1.5 cursor-pointer',
                  activeModalTab === 'logs' ? 'bg-gradient-to-r from-sky-600 to-cyan-500 text-white shadow-sm' : 'text-slate-400 hover:text-white']">
                <span>📜 系统日志</span>
                <span class="text-[10px] px-1.5 py-0.2 rounded-full bg-black/40 font-mono font-tabular">{{ app.displayedLogs.length }}</span>
              </button>
              <button @click="switchToConnections"
                :class="['px-3 py-1 rounded-lg text-xs font-bold transition flex items-center gap-1.5 cursor-pointer',
                  activeModalTab === 'connections' ? 'bg-gradient-to-r from-sky-600 to-cyan-500 text-white shadow-sm' : 'text-slate-400 hover:text-white']">
                <span>🌲 活跃连接树</span>
                <span class="text-[10px] px-1.5 py-0.2 rounded-full bg-black/40 font-mono text-cyan-300 font-bold font-tabular">{{ connections.length }}</span>
                <span v-if="connections.length > 0" class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-ping"></span>
              </button>
            </div>

            <!-- Issue #9 当月代理出站流量素雅概览 (已剥离 DIRECT / REJECT) -->
            <div class="hidden lg:flex items-center gap-1.5 px-2.5 py-1 rounded-xl bg-slate-800/80 border border-slate-700/80 text-[11px] font-mono text-slate-300 select-none">
              <span class="text-sky-400 font-bold">📊 当月代理出站:</span>
              <span class="text-white font-bold">{{ monthlyTraffic.total_human || '0 B' }}</span>
              <span class="text-slate-400 text-[10px]">(↑ {{ monthlyTraffic.uplink_human || '0 B' }} / ↓ {{ monthlyTraffic.downlink_human || '0 B' }})</span>
            </div>
          </div>

          <!-- Right Controls for Logs Tab -->
          <div v-if="activeModalTab === 'logs'" class="flex items-center gap-2 text-xs font-mono shrink-0 flex-nowrap">
            <div class="flex items-center macos-capsule p-0.5 rounded-xl">
              <button v-for="lvl in ['ALL', 'INFO', 'WARN', 'ERROR']" :key="lvl"
                @click="app.logFilterLevel = lvl"
                :class="['px-2.5 py-1 rounded-lg text-[10px] font-bold transition-all cursor-pointer',
                  app.logFilterLevel === lvl ? 'bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 text-white font-extrabold shadow-sm' : 'text-slate-300 hover:text-white hover:bg-white/10']">
                {{ lvl }}
              </button>
            </div>
            <button @click="app.clearSystemLogs" class="px-2.5 py-1 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-slate-300 hover:text-white text-[11px] font-bold active:scale-95 transition cursor-pointer" title="清空日志">
              🗑 清屏
            </button>
            <button @click="app.fetchSystemLogs(true)" class="p-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-slate-300 hover:text-white active:scale-95 transition cursor-pointer" title="立即刷新日志">
              🔄
            </button>
            <button @click="app.copyDiagnosticMarkdown" class="px-3 py-1 rounded-xl text-[11px] font-bold bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-slate-200 active:scale-95 transition flex items-center gap-1 cursor-pointer" title="复制脱敏报告">
              📋 复制报告
            </button>
            <button @click="app.uploadToGithubIssue" :disabled="app.creatingGithubIssue" class="px-3.5 py-1 rounded-xl text-[11px] font-extrabold bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white shadow-md active:scale-95 transition flex items-center gap-1.5 cursor-pointer shrink-0">
              <span v-if="app.creatingGithubIssue" class="inline-block animate-spin mr-0.5">↻</span>
              <span v-else>🚀</span>
              <span>问题反馈</span>
            </button>
            <button @click="closeModal" class="px-2.5 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 text-slate-300 hover:text-white text-xs font-bold transition cursor-pointer">
              ✕
            </button>
          </div>

          <!-- Right Controls for Connections Tab -->
          <div v-else class="flex items-center gap-2 text-xs font-mono shrink-0 flex-nowrap">
            <!-- View Mode Toggle -->
            <div class="flex items-center macos-capsule p-0.5 rounded-xl border border-white/10">
              <button @click="connViewMode = 'device'"
                :class="['px-2.5 py-1 rounded-lg text-[10px] font-bold transition cursor-pointer',
                  connViewMode === 'device' ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-400/40 shadow-sm' : 'text-slate-400 hover:text-white']">
                📱 终端设备 (MAC)
              </button>
              <button @click="connViewMode = 'domain'"
                :class="['px-2.5 py-1 rounded-lg text-[10px] font-bold transition cursor-pointer',
                  connViewMode === 'domain' ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-400/40 shadow-sm' : 'text-slate-400 hover:text-white']">
                🌐 目标域名
              </button>
              <button @click="connViewMode = 'flat'"
                :class="['px-2.5 py-1 rounded-lg text-[10px] font-bold transition cursor-pointer',
                  connViewMode === 'flat' ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-400/40 shadow-sm' : 'text-slate-400 hover:text-white']">
                📋 平铺明细
              </button>
            </div>

            <!-- Search Filter -->
            <input v-model="connSearch" placeholder="过滤域名/IP/节点..."
              class="px-2.5 py-1 rounded-xl bg-black/40 border border-white/15 text-slate-200 placeholder-slate-500 text-[11px] focus:outline-none focus:border-cyan-400/50 w-32 sm:w-40" />

            <!-- Refresh Button -->
            <button @click="fetchConnections(true)" class="p-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 text-slate-300 hover:text-white active:scale-95 transition cursor-pointer" title="刷新连接">
              <span :class="{'inline-block animate-spin': loadingConns}">🔄</span>
            </button>
            <button @click="closeModal" class="px-2.5 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 text-slate-300 hover:text-white text-xs font-bold transition cursor-pointer">
              ✕
            </button>
          </div>
        </div>

        <!-- Inline Toast Notification (Modern Web INP-optimized Feedback) -->
        <div v-if="toastMessage" class="px-4 py-1.5 bg-gradient-to-r from-cyan-900/90 via-sky-900/90 to-blue-900/90 border-b border-cyan-400/30 flex items-center justify-between text-xs font-mono animate-in fade-in">
          <div class="flex items-center gap-2 text-cyan-200">
            <span>⚡</span>
            <span>{{ toastMessage }}</span>
          </div>
          <button @click="toastMessage = ''" class="text-cyan-400 hover:text-white font-bold text-xs">✕</button>
        </div>

        <!-- Tab Content 1: System Logs -->
        <div v-if="activeModalTab === 'logs'" :ref="(el) => { app.logContainerRef = el }" class="h-[520px] max-h-[70vh] overflow-y-auto no-scrollbar p-3.5 sm:p-4 font-mono text-xs select-text transition-all" style="background-color: #05070e !important; color: #f8fafc !important;">
          <div class="space-y-1">
            <div v-if="app.displayedLogs.length === 0" class="h-full flex flex-col items-center justify-center text-slate-400 space-y-2 py-12">
              <span class="text-3xl">📜</span>
              <span class="text-xs">当前过滤器下暂无日志记录</span>
              <button v-if="app.logFilterLevel !== 'ALL' || app.logFilterModule !== 'ALL' || app.logSearchQuery"
                @click="app.logFilterLevel = 'ALL'; app.logFilterModule = 'ALL'; app.logSearchQuery = ''"
                class="text-[11px] text-cyan-400 underline hover:text-cyan-300 cursor-pointer">
                重置所有过滤器
              </button>
            </div>

            <div v-for="entry in app.displayedLogs" :key="entry.id" class="flex items-start gap-2.5 leading-normal hover:bg-white/[0.08] px-2 py-1 rounded transition font-mono group modern-cv-auto">
              <span class="text-amber-400/90 text-[11px] shrink-0 select-none font-tabular font-semibold">{{ entry.time }}</span>
              <span :class="['px-1.5 py-0.2 rounded text-[10px] font-bold shrink-0 select-none uppercase tracking-wider',
                entry.level === 'ERROR' ? 'bg-rose-950 text-rose-300 border border-rose-500/60 animate-pulse' :
                entry.level === 'WARN' ? 'bg-amber-950 text-amber-300 border border-amber-500/60' :
                entry.level === 'DEBUG' ? 'bg-purple-950 text-purple-300 border border-purple-500/50' :
                'bg-cyan-950 text-cyan-300 border border-cyan-500/50']">
                {{ entry.level }}
              </span>
              <span class="px-1.5 py-0.2 rounded bg-slate-800 text-sky-200 border border-slate-700 text-[10px] font-bold shrink-0 select-none">
                {{ entry.module }}
              </span>
              <span :class="['flex-1 break-all text-xs font-mono font-medium',
                entry.level === 'ERROR' ? 'text-rose-300 font-bold' :
                entry.level === 'WARN' ? 'text-amber-300' : 'text-slate-100']">
                {{ entry.msg }}
              </span>
            </div>
          </div>

          <div class="pt-2 sm:pt-2.5 flex items-center gap-2 text-emerald-300 text-xs select-none">
            <span>cubex@host:~$</span>
            <span class="inline-block w-2 h-4 bg-cyan-400 animate-pulse"></span>
          </div>
        </div>

        <!-- Tab Content 2: Three-Level Connections Tree & One-Click Override (Zero-CLS & Baseline CSS Containment) -->
        <div v-else class="h-[520px] max-h-[70vh] overflow-y-auto no-scrollbar p-3.5 sm:p-4 font-mono text-xs select-text transition-all space-y-3" style="background-color: #05070e !important; color: #f8fafc !important;">
          <!-- Stats Summary Bar (Zero CLS with font-tabular) -->
          <div class="p-3 rounded-2xl bg-white/[0.03] border border-white/10 flex items-center justify-between text-xs flex-wrap gap-2">
            <div class="flex items-center gap-3">
              <span class="text-slate-400">活跃连接: <strong class="text-cyan-300 font-bold font-tabular">{{ filteredConnections.length }}</strong> 条</span>
              <span class="text-slate-400">总下行: <strong class="text-emerald-300 font-bold font-tabular">{{ formatBytes(downloadTotal) }}</strong></span>
              <span class="text-slate-400">总上行: <strong class="text-sky-300 font-bold font-tabular">{{ formatBytes(uploadTotal) }}</strong></span>
            </div>
            <div class="text-[11px] text-slate-400 flex items-center gap-2">
              <span>🛡️ 自定义分流置顶已生效 (仅对新连接生效，长连接0震荡)</span>
              <button @click="expandAll" class="text-cyan-400 hover:underline cursor-pointer">全部展开</button>
              <span class="text-slate-600">|</span>
              <button @click="collapseAll" class="text-slate-400 hover:underline cursor-pointer">全部折叠</button>
            </div>
          </div>

          <!-- Empty State -->
          <div v-if="filteredConnections.length === 0" class="flex flex-col items-center justify-center py-16 text-slate-400 space-y-2">
            <span class="text-3xl">🌲</span>
            <span>当前暂无匹配的活跃网络连接</span>
            <span class="text-[11px] text-slate-500">当内网设备产生访问流量时将在此实时聚合显示</span>
          </div>

          <!-- View 1: Grouped by Client Device (MAC / SourceIP) -->
          <div v-else-if="connViewMode === 'device'" class="space-y-2.5">
            <div v-for="devGroup in deviceTree" :key="devGroup.key"
              class="rounded-2xl bg-white/[0.02] border border-white/10 overflow-hidden transition modern-cv-card">
              <!-- Level 1: Client Device Card Header -->
              <div @click="toggleExpand(devGroup.key)"
                class="px-4 py-2.5 bg-white/[0.04] hover:bg-white/[0.07] flex items-center justify-between cursor-pointer select-none">
                <div class="flex items-center gap-2.5">
                  <span class="text-xs text-cyan-400 font-bold transition-transform"
                    :class="{'rotate-90': isExpanded(devGroup.key)}">▶</span>
                  <span class="text-sm font-black text-white flex items-center gap-2">
                    <span>{{ devGroup.icon || '📱' }}</span>
                    <span>{{ devGroup.device_name }}</span>
                  </span>
                  <span class="text-[10px] font-mono px-2 py-0.2 rounded-full bg-slate-800 text-slate-300 border border-slate-700 font-tabular">
                    {{ devGroup.ip }}
                  </span>
                  <span v-if="devGroup.mac" class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-cyan-950/70 text-cyan-300 border border-cyan-500/30 font-tabular">
                    MAC: {{ devGroup.mac }}
                  </span>
                </div>
                <div class="flex items-center gap-3 text-xs font-tabular">
                  <span class="text-slate-400 font-mono">{{ devGroup.conns.length }} 连接</span>
                  <span class="text-emerald-300 font-mono">↓ {{ formatBytes(devGroup.down) }}</span>
                  <span class="text-sky-300 font-mono">↑ {{ formatBytes(devGroup.up) }}</span>
                </div>
              </div>

              <!-- Level 2 & 3: Collapsible Target Domains & Conns -->
              <div v-if="isExpanded(devGroup.key)" class="p-3 pl-6 space-y-2 bg-black/20 border-t border-white/5">
                <div v-for="domGroup in devGroup.domainGroups" :key="domGroup.domain"
                  class="rounded-xl bg-white/[0.02] border border-white/5 p-2.5 modern-cv-auto">
                  <!-- Level 2: Target Domain Header with Override Action -->
                  <div class="flex items-center justify-between gap-2 flex-wrap sm:flex-nowrap">
                    <div @click="toggleExpand(devGroup.key + ':' + domGroup.domain)"
                      class="flex items-center gap-2 cursor-pointer select-none truncate">
                      <span class="text-[10px] text-slate-400 font-bold"
                        :class="{'rotate-90': isExpanded(devGroup.key + ':' + domGroup.domain)}">▶</span>
                      <span class="font-bold text-white text-xs truncate max-w-[280px]">
                        {{ domGroup.domain }}
                      </span>
                      <span class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-slate-800 text-slate-400 font-tabular">
                        {{ domGroup.conns.length }} 流
                      </span>
                      <span v-if="domGroup.isOverridden"
                        class="text-[9px] px-1.5 py-0.2 rounded font-bold"
                        :class="domGroup.overridePolicy === 'DIRECT' ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-400/40' : (domGroup.overridePolicy === 'PROXIES' ? 'bg-sky-500/20 text-sky-300 border border-sky-400/40' : 'bg-rose-500/20 text-rose-300 border border-rose-400/40')">
                        {{ domGroup.overridePolicy === 'DIRECT' ? '已覆写直连' : (domGroup.overridePolicy === 'PROXIES' ? '已覆写代理' : '已拦截') }}
                      </span>
                    </div>

                    <!-- One-Click Override Control -->
                    <div class="flex items-center gap-1.5 shrink-0">
                      <button @click.stop="quickOverride(domGroup.domain, 'DIRECT')"
                        class="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-300 border border-emerald-400/30 active:scale-95 transition cursor-pointer"
                        title="一键改分流为直连 (仅对新连接生效)">
                        ⚡ 改直连
                      </button>
                      <button @click.stop="quickOverride(domGroup.domain, 'PROXIES')"
                        class="px-2 py-0.5 rounded text-[10px] font-bold bg-sky-500/20 hover:bg-sky-500/30 text-sky-300 border border-sky-400/30 active:scale-95 transition cursor-pointer"
                        title="一键改分流为代理 (仅对新连接生效)">
                        🚀 改代理
                      </button>
                      <button @click.stop="quickOverride(domGroup.domain, 'REJECT')"
                        class="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-500/20 hover:bg-rose-500/30 text-rose-300 border border-rose-400/30 active:scale-95 transition cursor-pointer"
                        title="一键拦截屏蔽">
                        🛡️ 屏蔽
                      </button>
                      <button v-if="domGroup.isOverridden" @click.stop="quickOverride(domGroup.domain, 'CLEAR')"
                        class="px-1.5 py-0.5 rounded text-[10px] text-slate-400 hover:text-white cursor-pointer" title="清除覆写">
                        ✕
                      </button>
                    </div>
                  </div>

                  <!-- Level 3: Individual Connection Details -->
                  <div v-if="isExpanded(devGroup.key + ':' + domGroup.domain)" class="mt-2 space-y-1 pl-4 border-l border-white/10">
                    <div v-for="conn in domGroup.conns" :key="conn.id"
                      class="flex items-center justify-between text-[11px] font-mono text-slate-300 hover:bg-white/[0.04] px-2 py-1 rounded modern-cv-auto">
                      <div class="flex items-center gap-2 truncate">
                        <span class="text-[10px] px-1 py-0.2 rounded bg-slate-800 text-slate-400 font-tabular">{{ conn.network }}/{{ conn.type }}</span>
                        <span class="text-white font-tabular">{{ conn.destination_ip }}:{{ conn.destination_port }}</span>
                        <span class="text-slate-500">➔</span>
                        <span class="text-cyan-300 font-bold truncate max-w-[140px]">{{ conn.chains && conn.chains.length ? conn.chains[conn.chains.length - 1] : 'DIRECT' }}</span>
                        <span class="text-[10px] text-slate-400">({{ conn.rule }})</span>
                      </div>
                      <div class="flex items-center gap-3 shrink-0 text-slate-400 font-tabular">
                        <span>↓ {{ formatBytes(conn.download_bytes) }}</span>
                        <span>↑ {{ formatBytes(conn.upload_bytes) }}</span>
                        <span>{{ conn.duration_sec }}s</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- View 2: Grouped by Target Domain -->
          <div v-else-if="connViewMode === 'domain'" class="space-y-2.5">
            <div v-for="domGroup in domainTree" :key="domGroup.domain"
              class="rounded-2xl bg-white/[0.02] border border-white/10 overflow-hidden transition modern-cv-card">
              <!-- Level 1: Domain Card Header with One-Click Override -->
              <div @click="toggleExpand('dom:' + domGroup.domain)"
                class="px-4 py-2.5 bg-white/[0.04] hover:bg-white/[0.07] flex items-center justify-between cursor-pointer select-none">
                <div class="flex items-center gap-2.5 truncate">
                  <span class="text-xs text-cyan-400 font-bold transition-transform"
                    :class="{'rotate-90': isExpanded('dom:' + domGroup.domain)}">▶</span>
                  <span class="text-sm font-black text-white truncate max-w-[320px]">
                    🌐 {{ domGroup.domain }}
                  </span>
                  <span class="text-[10px] font-mono px-2 py-0.2 rounded-full bg-slate-800 text-slate-300 font-tabular">
                    {{ domGroup.conns.length }} 连接
                  </span>
                  <span v-if="domGroup.isOverridden"
                    class="text-[9px] px-1.5 py-0.2 rounded font-bold"
                    :class="domGroup.overridePolicy === 'DIRECT' ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-400/40' : (domGroup.overridePolicy === 'PROXIES' ? 'bg-sky-500/20 text-sky-300 border border-sky-400/40' : 'bg-rose-500/20 text-rose-300 border border-rose-400/40')">
                    {{ domGroup.overridePolicy === 'DIRECT' ? '已覆写直连' : (domGroup.overridePolicy === 'PROXIES' ? '已覆写代理' : '已拦截') }}
                  </span>
                </div>
                <div class="flex items-center gap-2 shrink-0">
                  <button @click.stop="quickOverride(domGroup.domain, 'DIRECT')"
                    class="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-300 border border-emerald-400/30 active:scale-95 transition cursor-pointer">
                    ⚡ 改直连
                  </button>
                  <button @click.stop="quickOverride(domGroup.domain, 'PROXIES')"
                    class="px-2 py-0.5 rounded text-[10px] font-bold bg-sky-500/20 hover:bg-sky-500/30 text-sky-300 border border-sky-400/30 active:scale-95 transition cursor-pointer">
                    🚀 改代理
                  </button>
                  <button @click.stop="quickOverride(domGroup.domain, 'REJECT')"
                    class="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-500/20 hover:bg-rose-500/30 text-rose-300 border border-rose-400/30 active:scale-95 transition cursor-pointer">
                    🛡️ 屏蔽
                  </button>
                  <button v-if="domGroup.isOverridden" @click.stop="quickOverride(domGroup.domain, 'CLEAR')"
                    class="px-1.5 py-0.5 rounded text-[10px] text-slate-400 hover:text-white cursor-pointer" title="清除覆写">
                    ✕
                  </button>
                </div>
              </div>

              <!-- Level 2 & 3: Clients & Connections under Domain -->
              <div v-if="isExpanded('dom:' + domGroup.domain)" class="p-3 pl-6 space-y-1.5 bg-black/20 border-t border-white/5">
                <div v-for="conn in domGroup.conns" :key="conn.id"
                  class="flex items-center justify-between text-[11px] font-mono text-slate-300 hover:bg-white/[0.04] px-2 py-1 rounded modern-cv-auto">
                  <div class="flex items-center gap-2 truncate">
                    <span class="text-white font-bold">{{ conn.client_device }}</span>
                    <span class="text-slate-400 font-tabular">({{ conn.source_ip }})</span>
                    <span class="text-slate-500">➔</span>
                    <span class="text-cyan-300 font-bold truncate max-w-[160px]">{{ conn.chains && conn.chains.length ? conn.chains[conn.chains.length - 1] : 'DIRECT' }}</span>
                  </div>
                  <div class="flex items-center gap-3 shrink-0 text-slate-400 font-tabular">
                    <span>↓ {{ formatBytes(conn.download_bytes) }}</span>
                    <span>↑ {{ formatBytes(conn.upload_bytes) }}</span>
                    <span>{{ conn.duration_sec }}s</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- View 3: Flat List -->
          <div v-else class="space-y-1">
            <div v-for="conn in filteredConnections" :key="conn.id"
              class="flex items-center justify-between p-2 rounded-xl bg-white/[0.02] hover:bg-white/[0.05] border border-white/5 text-[11px] font-mono text-slate-300 modern-cv-auto">
              <div class="flex items-center gap-2 truncate">
                <span class="text-[10px] px-1.5 py-0.2 rounded bg-slate-800 text-slate-400 font-tabular">{{ conn.network }}</span>
                <span class="text-white font-bold truncate max-w-[120px]">{{ conn.client_device }}</span>
                <span class="text-slate-500">➔</span>
                <span class="text-cyan-200 truncate max-w-[200px]">{{ conn.host || conn.destination_ip }}</span>
                <span class="text-slate-500">via</span>
                <span class="text-emerald-300 truncate max-w-[120px]">{{ conn.chains && conn.chains.length ? conn.chains[conn.chains.length - 1] : 'DIRECT' }}</span>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <span class="text-slate-400 font-tabular">↓ {{ formatBytes(conn.download_bytes) }}</span>
                <button @click="quickOverride(conn.host || conn.destination_ip, 'DIRECT')"
                  class="px-1.5 py-0.5 rounded text-[9px] font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-400/30 hover:bg-emerald-500/30 cursor-pointer">
                  改直连
                </button>
                <button @click="quickOverride(conn.host || conn.destination_ip, 'PROXIES')"
                  class="px-1.5 py-0.5 rounded text-[9px] font-bold bg-sky-500/20 text-sky-300 border border-sky-400/30 hover:bg-sky-500/30 cursor-pointer">
                  改代理
                </button>
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>
  </dialog>
</template>

<script setup>
import { inject, ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'

const app = inject('cubexApp')

const dialogRef = ref(null)
const activeModalTab = ref('logs') // 'logs' | 'connections'
const connViewMode = ref('device') // 'device' | 'domain' | 'flat'
const connSearch = ref('')
const connections = ref([])
const downloadTotal = ref(0)
const uploadTotal = ref(0)
const loadingConns = ref(false)
const toastMessage = ref('')
let toastTimer = null

const monthlyTraffic = ref({
  year_month: '',
  uplink_human: '0 B',
  downlink_human: '0 B',
  total_human: '0 B'
})

async function fetchMonthlyTraffic() {
  try {
    const res = await fetch('/api/v1/traffic/monthly')
    const data = await res.json()
    if (data && data.ok) {
      monthlyTraffic.value = data
    }
  } catch (e) {}
}

// Native Dialog Open/Close watcher
watch(() => app?.showLogsModal, async (val) => {
  await nextTick()
  if (val) {
    fetchMonthlyTraffic()
    if (dialogRef.value && !dialogRef.value.open) {
      try {
        dialogRef.value.showModal()
      } catch (e) {}
    }
  } else {
    if (dialogRef.value && dialogRef.value.open) {
      dialogRef.value.close()
    }
  }
}, { immediate: true })

function closeModal() {
  if (app) app.showLogsModal = false
}

function handleBackdropClick(e) {
  if (e.target === dialogRef.value) {
    closeModal()
  }
}

// Independent Set for collapse/expand state persistence (Zero DOM Rebuild, 60FPS)
const expandedKeys = ref(new Set())

function isExpanded(key) {
  return expandedKeys.value.has(key)
}

function toggleExpand(key) {
  if (expandedKeys.value.has(key)) {
    expandedKeys.value.delete(key)
  } else {
    expandedKeys.value.add(key)
  }
}

function expandAll() {
  deviceTree.value.forEach(d => {
    expandedKeys.value.add(d.key)
    d.domainGroups.forEach(dg => expandedKeys.value.add(d.key + ':' + dg.domain))
  })
  domainTree.value.forEach(d => expandedKeys.value.add('dom:' + d.domain))
}

function collapseAll() {
  expandedKeys.value.clear()
}

function switchToConnections() {
  activeModalTab.value = 'connections'
  fetchConnections(true)
}

async function fetchConnections(manual = false) {
  if (manual) loadingConns.value = true
  try {
    const res = await fetch('/api/v1/telemetry/connections')
    if (res.ok) {
      const data = await res.json()
      if (data.ok && Array.isArray(data.connections)) {
        connections.value = data.connections
        downloadTotal.value = data.download_total || 0
        uploadTotal.value = data.upload_total || 0
        // Auto expand first 2 devices if nothing expanded yet
        if (expandedKeys.value.size === 0 && deviceTree.value.length > 0) {
          expandedKeys.value.add(deviceTree.value[0].key)
        }
      }
    }
  } catch (e) {
  } finally {
    loadingConns.value = false
  }
}

// Filtered Connections by Search Query
const filteredConnections = computed(() => {
  const q = (connSearch.value || '').trim().toLowerCase()
  if (!q) return connections.value
  return connections.value.filter(c => {
    return (c.host && c.host.toLowerCase().includes(q)) ||
      (c.source_ip && c.source_ip.toLowerCase().includes(q)) ||
      (c.client_mac && c.client_mac.toLowerCase().includes(q)) ||
      (c.client_device && c.client_device.toLowerCase().includes(q)) ||
      (c.destination_ip && c.destination_ip.toLowerCase().includes(q)) ||
      (c.chains && c.chains.some(ch => ch.toLowerCase().includes(q)))
  })
})

// Level 1-2-3 Tree Grouped by Client Device (MAC with SourceIP Fallback)
const deviceTree = computed(() => {
  const map = {}
  filteredConnections.value.forEach(c => {
    const key = c.client_mac || c.source_ip || 'unknown'
    if (!map[key]) {
      let devName = c.client_device || c.source_ip
      let icon = '📱'
      const isInternal = c.source_ip === '172.19.0.1' || (c.source_ip && c.source_ip.startsWith('172.19.'))
      const isLoopback = c.source_ip === '127.0.0.1' || c.source_ip === '::1'
      if (isInternal) {
        devName = 'CubeX 旁路由系统内部 (TUN网卡回环)'
        icon = '🖥️'
      } else if (isLoopback) {
        devName = 'CubeX 控制面服务 (Localhost)'
        icon = '🖥️'
      }
      map[key] = {
        key: key,
        ip: c.source_ip,
        mac: c.client_mac,
        device_name: devName,
        icon: icon,
        is_internal: isInternal || isLoopback,
        conns: [],
        down: 0,
        up: 0,
        domainMap: {}
      }
    }
    map[key].conns.push(c)
    map[key].down += c.download_bytes || 0
    map[key].up += c.upload_bytes || 0

    const dom = c.host || c.destination_ip || '未知目标'
    if (!map[key].domainMap[dom]) {
      map[key].domainMap[dom] = {
        domain: dom,
        isOverridden: c.is_overridden,
        overridePolicy: c.override_policy,
        conns: []
      }
    }
    map[key].domainMap[dom].conns.push(c)
  })

  return Object.values(map).map(dev => ({
    ...dev,
    domainGroups: Object.values(dev.domainMap)
  }))
})

// Grouped by Target Domain
const domainTree = computed(() => {
  const map = {}
  filteredConnections.value.forEach(c => {
    const dom = c.host || c.destination_ip || '未知目标'
    if (!map[dom]) {
      map[dom] = {
        domain: dom,
        isOverridden: c.is_overridden,
        overridePolicy: c.override_policy,
        conns: []
      }
    }
    map[dom].conns.push(c)
  })
  return Object.values(map)
})

function showToast(msg) {
  toastMessage.value = msg
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toastMessage.value = ''
  }, 4000)
}

// One-Click Non-Destructive Override (Modern Web Zero-Jank & INP < 50ms)
async function quickOverride(target, outbound) {
  if (!target) return
  try {
    const res = await fetch('/api/v1/rules/override-target', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ target, outbound })
    })
    const data = await res.json()
    if (data.ok) {
      // Immediate local state patch
      connections.value.forEach(c => {
        if ((c.host && c.host.toLowerCase() === target.toLowerCase()) || c.destination_ip === target) {
          c.is_overridden = (outbound !== 'CLEAR')
          c.override_policy = (outbound === 'CLEAR' ? 'NONE' : outbound)
        }
      })
      showToast(`已成功将 [${target}] 分流调整为: ${outbound} (仅对新连接生效，现有长连接保持稳定零震荡)`)
    } else {
      showToast(`改分流失败: ${data.error || '未知错误'}`)
    }
  } catch (e) {
    showToast('改分流请求异常: ' + e.message)
  }
}

function formatBytes(bytes) {
  if (!bytes || bytes <= 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

let connTimer = null
onMounted(() => {
  if (dialogRef.value && !dialogRef.value.open) {
    try {
      dialogRef.value.showModal()
    } catch (e) {}
  }
  connTimer = setInterval(() => {
    if (app && app.showLogsModal && activeModalTab.value === 'connections') {
      fetchConnections()
    }
  }, 2500)
})

onUnmounted(() => {
  if (connTimer) clearInterval(connTimer)
  if (toastTimer) clearTimeout(toastTimer)
})
</script>
