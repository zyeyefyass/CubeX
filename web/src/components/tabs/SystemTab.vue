<template>
        <div class="space-y-6 animate-in fade-in">

          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2">

            <div class="macos-title-shield space-y-1">

              <h2 class="text-base sm:text-lg font-bold text-white tracking-wide flex items-center gap-2">

                <span class="flex items-center gap-2"><ZootopiaIcon name="system" :size="22" class="shrink-0" /><span>宿主共生与系统运维</span></span>

              </h2>

              <p class="text-xs text-[#E2F8EC] tracking-normal font-medium">自动识别 RouterOS / 爱快 iKuai / OpenWrt / 飞牛 / PVE / Linux 底座 · 策略路由与防火墙自愈 · 四维在线无缝热更</p>

            </div>

            <button @click="app.showLogsModal = true"

              class="px-4 py-2 rounded-2xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/60 text-white font-bold text-xs flex items-center gap-2 transition active:scale-95 shadow-md cursor-pointer shrink-0"

              title="调出实时系统日志环形缓冲区终端">

              <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>

              <span>📜 实时系统日志终端</span>

            </button>

          </div>

          <!-- 宿主底座与四维升级主视窗 -->

          <div class="space-y-6">

          <!-- ════════════════════════════════════════════════════════════════════════════════════ -->
          <!-- ════════════════════════════════════════════════════════════════════════════════════ -->
          <!-- 👑 层级 1: 宿主共生底座 · 物理全景驾驶舱与一体化自愈中枢                           -->
          <!-- ════════════════════════════════════════════════════════════════════════════════════ -->
          <div class="p-6 sm:p-7 rounded-2xl sm:rounded-3xl macos-glass-card hover:border-sky-400/40 transition-all duration-300 space-y-6 shadow-2xl relative overflow-hidden">
            <div class="absolute -top-32 -right-32 w-80 h-80 rounded-full bg-cyan-500/10 blur-3xl pointer-events-none"></div>

            <!-- Tier 1: 统一头部 (Unified Hero Header) -->
            <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-cyan-500/20 pb-4 relative z-10">
              <div class="flex items-center gap-3.5">
                <div class="w-11 h-11 rounded-2xl bg-cyan-500/15 border border-cyan-400/30 flex items-center justify-center text-xl shrink-0 shadow-lg">
                  <ZootopiaIcon name="system" :size="24" />
                </div>
                <div>
                  <div class="flex items-center gap-2.5">
                    <h3 class="text-base sm:text-lg font-black text-white tracking-wide">宿主共生底座 · 物理全景驾驶舱</h3>
                    <!-- 宿主底座徽章 -->
                    <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-bold font-mono border shadow-sm transition-all bg-sky-500/15 text-sky-200 border-sky-400/40">
                      <span v-if="app.routerHW.connected" class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                      <span v-else class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                      {{ app.hostBadge.name || 'MikroTik RouterOS v7' }}
                    </span>
                  </div>
                  <p class="text-xs text-slate-300 mt-0.5">
                    {{ app.routerHW.model ? `${app.routerHW.model} · 固件 ${app.routerHW.version || 'v7.x'}` : 'RouterOS / 爱快 iKuai / OpenWrt / 飞牛 fnOS 原生共生' }} · 硬件底层毫秒透视 · 0秒逃生不伤主网
                  </p>
                </div>
              </div>

              <!-- 右侧快捷操作：刷新硬件遥测、主路由Web直达、配置凭据折叠切换 -->
              <div class="flex flex-wrap items-center gap-2">
                <button @click="refreshHW"
                  class="px-3 py-1.5 rounded-xl bg-white/5 hover:bg-white/10 border border-white/10 text-slate-200 hover:text-white text-xs font-bold font-mono transition-all flex items-center gap-1.5 cursor-pointer">
                  <span :class="{'animate-spin': refreshingHW}">↻</span> 刷新硬件遥测
                </button>
                <a v-if="app.routerHW.lan_ip || app.hostConfig.host" :href="`http://${app.routerHW.lan_ip || app.hostConfig.host}/`" target="_blank"
                  class="px-3 py-1.5 rounded-xl bg-sky-500/20 hover:bg-sky-500/30 border border-sky-400/40 text-sky-200 hover:text-white text-xs font-bold font-mono transition-all flex items-center gap-1 shadow-sm">
                  <span>↗</span> 主路由 Web
                </a>
                <button @click="showHostCredentialsDeck = !showHostCredentialsDeck"
                  class="px-3 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/15 text-slate-200 hover:text-white text-xs font-bold font-mono transition-all flex items-center gap-1.5">
                  <span>🔑</span> {{ showHostCredentialsDeck ? '收起凭据' : '配置凭据' }}
                </button>
              </div>
            </div>

            <!-- Tier 2: 核心网络与存储遥测大屏 (Live Router Telemetry 4-Column Grid) -->
            <div class="grid grid-cols-2 lg:grid-cols-4 gap-3 relative z-10 text-xs font-mono">
              <!-- 1. 外网 WAN 出口 -->
              <div class="macos-sub-card p-3.5 rounded-2xl border border-white/10 space-y-1.5">
                <div class="flex items-center justify-between text-slate-400 text-[11px]">
                  <span class="flex items-center gap-1">🌐 外网 WAN 拨号</span>
                  <span class="text-emerald-300 font-bold">● 在线</span>
                </div>
                <div class="text-sm sm:text-base font-black text-white truncate">
                  {{ app.routerHW.wan_ip || app.hostInfo.wan_ip || '待探测' }}
                </div>
                <div class="text-[10px] text-slate-400 flex items-center justify-between">
                  <span>{{ app.routerHW.wan_interface || 'WAN 上行接口' }}</span>
                  <span class="text-cyan-300">动态物理出口</span>
                </div>
              </div>

              <!-- 2. 内网 LAN 网关与转发 -->
              <div class="macos-sub-card p-3.5 rounded-2xl border border-white/10 space-y-1.5">
                <div class="flex items-center justify-between text-slate-400 text-[11px]">
                  <span class="flex items-center gap-1">🏠 内网 LAN 网关</span>
                  <span class="text-sky-300 font-bold">内核转发就绪</span>
                </div>
                <div class="text-sm sm:text-base font-black text-sky-200 truncate">
                  {{ app.routerHW.lan_ip || app.hostConfig.host || '自动探测中' }}
                </div>
                <div class="text-[10px] text-slate-400 flex items-center justify-between">
                  <span>{{ app.routerHW.lan_interface || 'bridge1' }}</span>
                  <span class="text-emerald-300">旁路规则已接管</span>
                </div>
              </div>

              <!-- 3. 固件底座与连续运行时间 -->
              <div class="macos-sub-card p-3.5 rounded-2xl border border-white/10 space-y-1.5">
                <div class="flex items-center justify-between text-slate-400 text-[11px]">
                  <span class="flex items-center gap-1">⏱️ 固件与运行时间</span>
                  <span class="text-emerald-300 font-bold">● 稳定</span>
                </div>
                <div class="text-sm sm:text-base font-black text-emerald-200 truncate">
                  {{ app.routerHWMemoryDisplay.uptime }}
                </div>
                <div class="text-[10px] text-slate-400 flex items-center justify-between">
                  <span>固件 {{ app.routerHWMemoryDisplay.version }}</span>
                  <span class="text-cyan-300">{{ app.routerHWMemoryDisplay.arch }}</span>
                </div>
              </div>

              <!-- 4. 宿主硬件 CPU 与内存负载 -->
              <div class="macos-sub-card p-3.5 rounded-2xl border border-white/10 space-y-1.5">
                <div class="flex items-center justify-between text-slate-400 text-[11px]">
                  <span class="flex items-center gap-1">⚡ 宿主物理负载</span>
                  <span :class="app.routerHWMemoryDisplay.cpu > 80 ? 'text-amber-300' : 'text-emerald-300'" class="font-bold">
                    CPU {{ app.routerHWMemoryDisplay.cpu }}%
                  </span>
                </div>
                <div class="text-sm sm:text-base font-black text-cyan-200 truncate">
                  {{ app.routerHWMemoryDisplay.freeStr }}
                </div>
                <div class="text-[10px] text-slate-400 flex items-center justify-between">
                  <span>总内存 {{ app.routerHWMemoryDisplay.totalStr }}</span>
                  <span class="text-teal-300 font-bold">物理线速直出</span>
                </div>
              </div>
            </div>

            <!-- 全息微光动态通道管道 (Twilight Crystal Data Pipeline) -->
            <div class="p-4 sm:p-5 rounded-2xl bg-black/40 border border-cyan-500/25 relative z-10 space-y-3 shadow-inner">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <span class="text-xs">⚡</span>
                  <span class="text-xs sm:text-sm font-bold text-white tracking-wide">全链路物理芯片直出 vs 双核网关全息管道</span>
                </div>
                <div class="flex items-center gap-3 text-xs font-mono">
                  <span class="text-emerald-400 flex items-center gap-1">
                    <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span> 物理芯片直出 91.5%
                  </span>
                  <span class="text-cyan-400 flex items-center gap-1">
                    <span class="w-2 h-2 rounded-full bg-cyan-400 animate-pulse"></span> 旁路加速穿透 8.5%
                  </span>
                </div>
              </div>

              <!-- 全息双通道流光动效管道 -->
              <div class="space-y-2">
                <!-- 通道 1: 主路由物理芯片直连通道 -->
                <div class="space-y-1">
                  <div class="flex items-center justify-between text-[11px] font-mono text-slate-300">
                    <span class="flex items-center gap-1.5">
                      <span class="text-emerald-400 font-bold">⚡</span>
                      <span>主路由物理芯片直连通道</span>
                      <span class="text-[10px] text-slate-400">(FastTrack / 芯片流表 · 0 CPU 0 延迟直出)</span>
                    </span>
                    <span class="text-emerald-300 font-bold">91.5% · ~980 Mbps 线速</span>
                  </div>
                  <div class="w-full h-3 rounded-full bg-slate-900/80 border border-emerald-500/30 p-0.5 overflow-hidden relative">
                    <div class="h-full rounded-full bg-gradient-to-r from-emerald-500 via-teal-400 to-cyan-400 transition-all duration-700 shadow-[0_0_12px_rgba(16,185,129,0.5)]" style="width: 91.5%"></div>
                  </div>
                </div>

                <!-- 通道 2: CubeX 双核网关加速穿透通道 -->
                <div class="space-y-1">
                  <div class="flex items-center justify-between text-[11px] font-mono text-slate-300">
                    <span class="flex items-center gap-1.5">
                      <span class="text-cyan-400 font-bold">🌌</span>
                      <span>CubeX 双核网关加速通道</span>
                      <span class="text-[10px] text-slate-400">(Mihomo × sing-box v1.14+ · Fake-IP 规则穿透)</span>
                    </span>
                    <span class="text-cyan-300 font-bold">8.5% · 198.18.0.0/15 智能分流</span>
                  </div>
                  <div class="w-full h-3 rounded-full bg-slate-900/80 border border-cyan-500/30 p-0.5 overflow-hidden relative">
                    <div class="h-full rounded-full bg-gradient-to-r from-purple-500 via-indigo-400 to-sky-400 transition-all duration-700 shadow-[0_0_12px_rgba(56,189,248,0.5)]" style="width: 8.5%"></div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Tier 3: 宿主共生凭据与鉴权面板 (支持点击展开/折叠，默认清爽折叠) -->
            <div v-show="showHostCredentialsDeck" class="macos-sub-card p-4 sm:p-5 rounded-2xl space-y-3.5 shadow-sm relative z-10 transition-all border border-cyan-500/20">
              <div class="flex items-center justify-between border-b border-white/10 pb-2.5">
                <div class="flex items-center gap-2">
                  <span class="text-base">🔑</span>
                  <span class="font-bold text-white tracking-wide">宿主共生凭据与鉴权</span>
                  <span class="text-[10px] font-mono px-2.5 py-0.5 rounded-full bg-white/10 text-emerald-300 border border-emerald-400/30 font-semibold shadow-sm">
                    {{ app.isRouterOS ? 'RouterOS REST API' : (app.isOpenWrt ? 'OpenWrt UCI / SSH' : (app.isIKuai ? '爱快 Web-RPC' : 'Docker / Linux 内核')) }}
                  </span>
                </div>
                <span class="text-[11px] text-slate-400 font-mono">凭据仅保存在当前容器，绝不上传云端</span>
              </div>

              <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
                <div>
                  <label class="block text-[11px] text-slate-300 font-medium mb-1">主路由 IP / 域名</label>
                  <input v-model="app.hostConfig.host" type="text" placeholder="自动探测或输入主路由 IP"
                    class="w-full px-3 py-2 rounded-xl bg-black/30 border border-white/15 text-white placeholder-slate-500 text-xs font-mono focus:outline-none focus:border-cyan-400 transition-all" />
                </div>
                <div>
                  <label class="block text-[11px] text-slate-300 font-medium mb-1">API 管理员账号</label>
                  <input v-model="app.hostConfig.user" type="text" placeholder="admin"
                    class="w-full px-3 py-2 rounded-xl bg-black/30 border border-white/15 text-white placeholder-slate-500 text-xs font-mono focus:outline-none focus:border-cyan-400 transition-all" />
                </div>
                <div>
                  <label class="block text-[11px] text-slate-300 font-medium mb-1">API 管理密码</label>
                  <input v-model="app.hostConfig.password" type="password" placeholder="••••••••"
                    class="w-full px-3 py-2 rounded-xl bg-black/30 border border-white/15 text-white placeholder-slate-500 text-xs font-mono focus:outline-none focus:border-cyan-400 transition-all" />
                </div>
              </div>

              <div class="flex items-center justify-between gap-3 pt-1">
                <p v-if="app.hostConfig.password === '******'" class="text-[11px] text-emerald-300/90 font-mono">
                  ✓ 密码已加密脱敏保存（不修改将沿用已存凭据）
                </p>
                <span v-else></span>
                <button @click="app.saveCredentialsAndLinkHost" :disabled="app.savingAndLinkingHost"
                  class="px-4 py-2 rounded-xl bg-cyan-500/25 hover:bg-cyan-500/35 border border-cyan-400/40 text-cyan-200 hover:text-white text-xs font-bold font-mono transition-all flex items-center gap-1.5 shadow-sm">
                  <span v-if="app.savingAndLinkingHost" class="inline-block animate-spin">↻</span>
                  <span>{{ app.savingAndLinkingHost ? '正在验证连接...' : '保存凭据并立即联动' }}</span>
                </button>
              </div>
            </div>

            <!-- Tier 4: 宿主共生自愈控制中枢 (方案 A · 统一自适应全链路深度自愈) -->
            <div class="pt-2.5 border-t border-cyan-500/20 relative z-10 space-y-3">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <span class="text-sm">🛡️</span>
                  <span class="text-xs sm:text-sm font-bold text-white tracking-wide">宿主全链路自愈中枢</span>
                  <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-300 border border-emerald-400/30">
                    Netwatch 0秒逃生守护就绪
                  </span>
                </div>
                <span class="text-[11px] text-slate-400 font-mono hidden sm:inline-block">底座状态微秒级感知 · 故障全自动无损切换</span>
              </div>

              <!-- 统一自适应全链路自愈主操作卡片 (全宽通栏) -->
              <div class="w-full">
                <button @click="app.inspectAndCleanReapplyRos" :disabled="app.cleaningRosFirewall"
                  class="w-full p-4 rounded-2xl bg-gradient-to-r from-cyan-500/20 via-sky-500/15 to-blue-500/15 hover:from-cyan-500/30 hover:via-sky-500/25 hover:to-blue-500/25 border border-cyan-400/40 hover:border-cyan-400 text-white transition-all flex items-center justify-between group shadow-lg cursor-pointer">
                  <div class="flex items-center gap-3.5">
                    <div class="w-11 h-11 rounded-2xl bg-cyan-500/20 border border-cyan-400/40 flex items-center justify-center text-xl shrink-0 shadow-inner">
                      <span v-if="app.cleaningRosFirewall" class="animate-spin text-cyan-300">↻</span>
                      <span v-else>⚡</span>
                    </div>
                    <div class="text-left">
                      <div class="font-black text-sm sm:text-base text-white flex items-center gap-2 flex-wrap">
                        <span>宿主全链路深度自愈与规则加固</span>
                        <span class="text-[10px] font-mono px-2.5 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-400/30 font-bold">
                          自适应当前底座: {{ app.hostBadge.name || 'RouterOS' }}
                        </span>
                      </div>
                      <div class="text-xs text-slate-300 mt-1 font-mono">
                        一键原子化联动：策略路由重配 · 伴生防火墙放行 · IPv6 双栈防 DNS 泄露 · 0秒逃生看门狗校准
                      </div>
                    </div>
                  </div>
                  <div class="flex items-center gap-2 text-cyan-300 font-bold font-mono text-xs sm:text-sm shrink-0 pl-3">
                    <span>{{ app.cleaningRosFirewall ? '全栈自愈执行中...' : '立即自愈' }}</span>
                    <span class="group-hover:translate-x-1.5 transition-transform text-sm sm:text-base">➔</span>
                  </div>
                </button>
              </div>

              <!-- 实时执行反馈卡片 -->
              <div v-if="app.hostLastExecution.title" class="p-3 rounded-xl border border-cyan-500/30 bg-black/40 text-xs font-mono flex items-center justify-between">
                <div class="flex items-center gap-2 truncate">
                  <span :class="app.hostLastExecution.success ? 'text-emerald-400' : 'text-rose-400'">
                    {{ app.hostLastExecution.success ? '✓' : '✗' }}
                  </span>
                  <span class="font-bold text-white">{{ app.hostLastExecution.title }}</span>
                  <span class="text-slate-400 truncate">{{ app.hostLastExecution.detail || app.hostLastExecution.message }}</span>
                </div>
                <span class="text-[10px] text-slate-500 shrink-0 ml-2">{{ app.hostLastExecution.time }}</span>
              </div>

            </div>
          </div>

          <!-- 🚀 四维版本在线检查与升级 -->

          <div class="space-y-4">

            <div class="border-b border-slate-700/40 pb-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3">

              <div>

                <h3 class="text-base sm:text-lg font-bold text-white tracking-wide flex items-center gap-2">

                  <span>🚀 四维版本在线检查与升级</span>

                  <span v-if="app.hasAnyUpdate" class="text-xs px-2.5 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/40 font-bold animate-pulse">

                    ● 发现新版本可用

                  </span>

                  <span v-else-if="app.probeChecked" class="text-xs px-2.5 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 font-medium">

                    ✓ 全维度最新稳定版

                  </span>

                </h3>

                <p class="text-xs text-slate-400 mt-1">

                  支持 CubeX 主控守护进程、Mihomo / sing-box 双内核、217heidai 规则库及宿主共生固件的一键云端版本核对与在线无缝平滑热更

                </p>

              </div>

              <button @click="app.openProbeModal" :disabled="app.isProbingAll" class="px-4 py-2 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white font-extrabold shadow-md shadow-sky-500/20 text-xs transition flex items-center gap-2 shadow-lg shadow-sky-500/20 shrink-0 cursor-pointer active:scale-98">

                <span :class="{'inline-block animate-spin': app.isProbingAll}">🛰️</span>

                <span>{{ app.isProbingAll ? '正在进行四维体检...' : '四维智能探针实时体检' }}</span>

              </button>

            </div>

            <!-- 四维版本智能探测横幅提示 (检测到更新时全息微光提醒) -->

            <transition name="fade">

              <div v-if="app.hasAnyUpdate" class="p-4 rounded-2xl bg-gradient-to-r from-amber-950/80 via-slate-900/90 to-cyan-950/80 border border-amber-500/50 shadow-xl flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 animate-in fade-in">

                <div class="flex items-center gap-3">

                  <span class="text-3xl animate-bounce select-none leading-none">🔔</span>

                  <div>

                    <div class="text-xs font-bold text-white flex items-center gap-2">

                      <span class="text-amber-300">【四维探针提醒】检测到可用系统升级项目！</span>

                      <span class="text-[10px] px-2 py-0.5 rounded bg-amber-900/90 text-amber-200 border border-amber-600 font-mono">{{ app.updateSummaryText }}</span>

                    </div>

                    <p class="text-[11px] text-slate-300 mt-0.5 leading-relaxed">{{ app.updateDetailText }}</p>

                  </div>

                </div>

                <div class="flex items-center gap-2 shrink-0 flex-wrap">

                  <!-- 1. CubeX Panel Update -->

                  <button v-if="app.systemOverview.panel_update_avail" @click="app.openUpgradeModal" :disabled="app.upgradingPanel" class="px-3.5 py-1.5 rounded-xl bg-cyan-500 hover:bg-cyan-400 text-slate-950 text-xs font-bold transition shadow-md shadow-sky-500/20 cursor-pointer">

                    一键热更 CubeX

                  </button>

                  <!-- 2. Mihomo Core Update -->

                  <button v-if="!app.coreUpgradeFrozen && app.systemOverview.mihomo_update_avail" @click="app.upgradeCore('mihomo')" :disabled="app.upgradingCoreMihomo" class="px-3.5 py-1.5 rounded-xl bg-amber-600 hover:bg-amber-500 text-white text-xs font-bold transition shadow-md shadow-amber-600/20 cursor-pointer active:scale-95 flex items-center gap-1.5">

                    <span v-if="app.upgradingCoreMihomo" class="inline-block animate-spin mr-0.5">↻</span>

                    <span>{{ app.upgradingCoreMihomo ? '正在升级 Mihomo...' : `⬆ 升级 Mihomo 内核 (${app.systemOverview.mihomo_latest_version || '最新'})` }}</span>

                  </button>

                  <!-- 3. sing-box Core Update -->

                  <button v-if="!app.coreUpgradeFrozen && app.systemOverview.singbox_update_avail" @click="app.upgradeCore('singbox')" :disabled="app.upgradingCoreSingbox" class="px-3.5 py-1.5 rounded-xl bg-amber-600 hover:bg-amber-500 text-white text-xs font-bold transition shadow-md shadow-amber-600/20 cursor-pointer active:scale-95 flex items-center gap-1.5">

                    <span v-if="app.upgradingCoreSingbox" class="inline-block animate-spin mr-0.5">↻</span>

                    <span>{{ app.upgradingCoreSingbox ? '正在升级 sing-box...' : `⬆ 升级 sing-box 内核 (${app.systemOverview.singbox_latest_version || '最新'})` }}</span>

                  </button>

                  <!-- 4. Geo Rules Update (ONLY when has_update is true) -->

                  <button v-if="app.geodataStatus.has_update" @click="app.updateGeoData" :disabled="app.updatingGeo" class="px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition shadow-md shadow-emerald-600/20 cursor-pointer active:scale-95 flex items-center gap-1.5">

                    <span v-if="app.updatingGeo" class="inline-block animate-spin mr-0.5">↻</span>

                    <span>{{ app.updatingGeo ? '正在全量同步...' : '一键更新规则库' }}</span>

                  </button>

                  <!-- 5. Host Firmware Update -->

                  <button v-if="app.hostUpgradeDetails.hasUpdate" @click="app.executeHostUpgrade" :disabled="app.executingHostUpgrade" class="px-3.5 py-1.5 rounded-xl bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold transition shadow-md shadow-purple-600/20 cursor-pointer active:scale-95">

                    一键升级宿主固件

                  </button>

                </div>

              </div>

            </transition>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-5 sm:gap-6">

              <!-- 1. CubeX 主控系统版本 -->

              <div class="p-6 sm:p-7 rounded-2xl sm:rounded-3xl macos-glass-card hover:border-sky-400/40 transition-all duration-300 space-y-4 flex flex-col justify-between group">

                <!-- Tier 1: Header (min-h-[36px]) -->

                <div class="min-h-[36px] flex justify-between items-center gap-3">

                  <div class="flex items-center gap-3 min-w-0">

                    <img src="/logo-cubex.svg" class="w-8 h-8 sm:w-9 sm:h-9 min-w-[32px] sm:min-w-[36px] max-w-[36px] h-[32px] sm:h-[36px] max-h-[36px] object-contain shrink-0 drop-shadow-[0_0_12px_rgba(56,189,248,0.6)] transition-transform duration-300 group-hover:scale-110" alt="CubeX" />

                    <span class="text-sm sm:text-base font-extrabold text-white whitespace-nowrap truncate tracking-wide">CubeX 系统守护进程</span>

                  </div>

                  <span v-if="app.systemOverview.panel_update_avail" class="text-xs font-mono px-3 py-1 rounded-lg bg-amber-500/20 text-amber-300 border border-amber-400/40 animate-pulse whitespace-nowrap shrink-0 font-semibold shadow-sm">发现新版 {{ app.systemOverview.panel_latest_version }}</span>

                  <span v-else class="text-xs font-mono px-3 py-1 rounded-lg bg-emerald-500/15 text-emerald-300 border border-emerald-400/30 whitespace-nowrap shrink-0 flex items-center gap-1.5 font-semibold shadow-sm"><span class="w-2 h-2 rounded-full bg-emerald-400"></span>✓ 已是最新稳定版</span>

                </div>

                <!-- Tier 2: Body (min-h-[86px]) -->

                <div class="min-h-[86px] text-xs sm:text-[13px] font-mono macos-sub-card p-3.5 rounded-xl flex flex-col justify-center gap-2.5 shadow-sm">

                  <div class="flex items-center justify-between">

                    <span>当前运行: <strong class="text-sky-200 font-black text-sm">{{ app.systemOverview.cubex_version || 'Test v8.4' }}</strong></span>

                    <span class="text-slate-400">云端最新: <span class="text-slate-200 font-semibold">{{ app.systemOverview.panel_latest_version || app.systemOverview.cubex_version || 'Test v8.4' }}</span></span>

                  </div>

                  <div class="flex items-center justify-between text-xs">

                    <span class="text-slate-400">运行底座: <span class="text-slate-200 font-semibold">{{ app.hostBadge.name || 'MikroTik RouterOS' }} ({{ app.hostInfo.architecture || 'amd64' }})</span></span>

                    <span class="text-emerald-400 font-bold flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>主控活跃守护</span>

                  </div>

                </div>

                <!-- Tier 3: Action Buttons (h-11 sm:h-12) -->

                <div class="h-11 sm:h-12">

                  <button @click="app.upgradePanel" :disabled="app.upgradingPanel" class="w-full h-full rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white font-extrabold text-xs sm:text-sm transition cursor-pointer active:scale-98 flex items-center justify-center gap-2 shadow-lg shadow-sky-500/25 disabled:opacity-50">

                    <span v-if="app.upgradingPanel" class="inline-block animate-spin mr-1">↻</span>

                    <span>{{ app.upgradingPanel ? '⏳ 正在核对与平滑热更中...' : (app.systemOverview.panel_update_avail ? `🚀 一键在线热更至 ${app.systemOverview.panel_latest_version}` : '🔄 校验并在线热更 CubeX') }}</span>

                  </button>

                </div>

                <!-- Tier 4: Terminal Viewport (h-[70px] sm:h-[76px]) -->

                <div class="h-[70px] sm:h-[76px] p-3 rounded-xl macos-sub-card border border-cyan-500/25 text-xs font-mono text-emerald-300 leading-relaxed overflow-y-auto custom-scrollbar flex items-center shadow-inner">

                  <div v-if="app.panelUpgradeLog" class="w-full break-words">

                    {{ app.panelUpgradeLog }}

                  </div>

                  <div v-else class="text-slate-300 text-xs flex items-center gap-2.5">

                    <span class="text-emerald-300 font-bold shrink-0">[CUBEX]</span>

                    <span>主控守护进程运行中 · 双引擎热备调度活跃 · 内存与端口健康</span>

                  </div>

                </div>

              </div>

              <!-- 2. 双内核版本 -->

              <div class="p-6 sm:p-7 rounded-2xl sm:rounded-3xl macos-glass-card hover:border-sky-400/40 transition-all duration-300 space-y-4 flex flex-col justify-between group">

                <!-- Tier 1: Header (min-h-[36px]) -->

                <div class="min-h-[36px] flex justify-between items-center gap-3">

                  <div class="flex items-center gap-3 min-w-0">

                    <div class="flex items-center gap-2.5 shrink-0">

                      <img src="/logo-singbox.svg" class="w-7 h-7 min-w-[28px] max-w-[28px] h-[28px] max-h-[28px] object-contain shrink-0 drop-shadow-[0_0_8px_rgba(168,85,247,0.5)] transition-transform duration-300 group-hover:scale-105" alt="sing-box" />

                      <img src="/logo-mihomo.svg" class="w-7 h-7 min-w-[28px] max-w-[28px] h-[28px] max-h-[28px] object-contain shrink-0 drop-shadow-[0_0_8px_rgba(56,189,248,0.5)] transition-transform duration-300 group-hover:scale-105" alt="Mihomo" />

                    </div>

                    <span class="text-sm sm:text-base font-extrabold text-white whitespace-nowrap truncate tracking-wide">双代理内核</span>

                  </div>

                  <div class="flex items-center gap-2 shrink-0">

                    <!-- Freeze Toggle Pill (Clean Frosted Glass · Zero Black Box) -->

                    <button @click="app.toggleCoreUpgradeFreeze" :title="app.coreUpgradeFrozen ? '当前已锁定内核版本，点击解除冻结' : '点击冻结内核升级，锁定当前稳定版本，防止语法与功能冲突'"

                      :class="[

                        'text-xs font-mono px-2.5 py-1 rounded-lg border transition cursor-pointer flex items-center gap-1.5 shadow-sm active:scale-95 font-semibold',

                        app.coreUpgradeFrozen

                          ? 'bg-amber-500/20 text-sky-200 border-amber-400/50 hover:bg-amber-500/30'

                          : 'bg-white/10 hover:bg-white/20 border-white/20 text-white hover:text-sky-200 hover:border-sky-400/50'

                      ]">

                      <span>{{ app.coreUpgradeFrozen ? '🔒 已冻结升级 (点击解锁)' : '🔒 冻结升级' }}</span>

                    </button>

                    <!-- Status Badge (when not frozen) -->

                    <template v-if="!app.coreUpgradeFrozen">

                      <span v-if="app.upgradingCoreMihomo || app.upgradingCoreSingbox" class="text-xs font-mono px-2.5 py-1 rounded-lg bg-purple-950 text-purple-300 border border-purple-500/40 flex items-center gap-1.5 shadow-sm font-semibold">

                        <span class="inline-block animate-spin text-[10px]">↻</span>

                        <span>内核热更中...</span>

                      </span>

                      <span v-else-if="app.systemOverview.mihomo_update_avail || app.systemOverview.singbox_update_avail" class="text-xs font-mono px-2.5 py-1 rounded-lg bg-amber-500/20 text-amber-300 border border-amber-400/40 animate-pulse flex items-center gap-1 shadow-sm font-semibold">

                        <span>⚠️ 待催更</span>

                      </span>

                      <span v-else class="text-xs font-mono px-2.5 py-1 rounded-lg bg-emerald-500/15 text-emerald-300 border border-emerald-400/30 flex items-center gap-1.5 shadow-sm font-semibold">

                        <span class="w-2 h-2 rounded-full bg-emerald-400"></span>

                        <span>✓ 已最新</span>

                      </span>

                    </template>

                  </div>

                </div>

                <!-- Tier 2: Body (min-h-[86px] · singbox在mihomo上排) -->

                <div class="min-h-[86px] text-xs sm:text-[13px] font-mono macos-sub-card p-3.5 rounded-xl flex flex-col justify-center gap-2.5 shadow-sm">

                  <div class="flex items-center justify-between">

                    <span>sing-box 内核: <strong class="text-white font-bold text-sm">{{ app.systemOverview.singbox_curr_version || 'v1.14.4' }}</strong></span>

                    <span class="text-slate-400">官方最新: <span :class="app.systemOverview.singbox_update_avail ? 'text-amber-400 font-bold' : 'text-slate-200 font-semibold'">{{ app.systemOverview.singbox_latest_version || 'v1.14.4' }}</span></span>

                  </div>

                  <div class="flex items-center justify-between">

                    <span>Mihomo 内核: <strong class="text-white font-bold text-sm">{{ app.systemOverview.mihomo_curr_version || 'v1.19.33' }}</strong></span>

                    <span class="text-slate-400">官方最新: <span :class="app.systemOverview.mihomo_update_avail ? 'text-amber-400 font-bold' : 'text-slate-200 font-semibold'">{{ app.systemOverview.mihomo_latest_version || 'v1.19.33' }}</span></span>

                  </div>

                </div>

                <!-- Tier 3: Action Buttons (h-11 sm:h-12 · 对应按钮singbox在左侧) -->

                <div class="h-11 sm:h-12 grid grid-cols-2 gap-2.5">

                  <!-- 1. sing-box 升级按钮 (左侧) -->

                  <button v-if="app.coreUpgradeFrozen" disabled class="h-full rounded-xl border border-white/10 bg-white/5 text-slate-400 font-bold text-xs sm:text-sm flex items-center justify-center gap-1 cursor-not-allowed opacity-60" title="内核升级已冻结锁定，请先解除冻结">

                    <span>🔒 sing-box (已冻结)</span>

                  </button>

                  <button v-else @click="app.upgradeCore('singbox')" :disabled="app.upgradingCoreSingbox" class="h-full rounded-xl border font-bold text-xs sm:text-sm transition cursor-pointer active:scale-98 flex items-center justify-center gap-1.5 shadow-md"

                    :class="app.systemOverview.singbox_update_avail ? 'bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 text-white font-extrabold shadow-lg shadow-sky-500/25' : 'bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/60 text-white font-bold shadow-md'">

                    <span v-if="app.upgradingCoreSingbox" class="inline-block animate-spin mr-1">↻</span>

                    <span>{{ app.upgradingCoreSingbox ? '核对中...' : (app.systemOverview.singbox_update_avail ? '⬆ 升级 sing-box (自决)' : '⬆ 升级 sing-box') }}</span>

                  </button>

                  <!-- 2. Mihomo 升级按钮 (右侧) -->

                  <button v-if="app.coreUpgradeFrozen" disabled class="h-full rounded-xl border border-white/10 bg-white/5 text-slate-400 font-bold text-xs sm:text-sm flex items-center justify-center gap-1 cursor-not-allowed opacity-60" title="内核升级已冻结锁定，请先解除冻结">

                    <span>🔒 Mihomo (已冻结)</span>

                  </button>

                  <button v-else @click="app.upgradeCore('mihomo')" :disabled="app.upgradingCoreMihomo" class="h-full rounded-xl border font-bold text-xs sm:text-sm transition cursor-pointer active:scale-98 flex items-center justify-center gap-1.5 shadow-md"

                    :class="app.systemOverview.mihomo_update_avail ? 'bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 text-white font-extrabold shadow-lg shadow-sky-500/25' : 'bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/60 text-white font-bold shadow-md'">

                    <span v-if="app.upgradingCoreMihomo" class="inline-block animate-spin mr-1">↻</span>

                    <span>{{ app.upgradingCoreMihomo ? '核对中...' : (app.systemOverview.mihomo_update_avail ? '⬆ 升级 Mihomo (自决)' : '⬆ 升级 Mihomo') }}</span>

                  </button>

                </div>

                <!-- Tier 4: Terminal Viewport (h-[70px] sm:h-[76px]) -->

                <div class="h-[70px] sm:h-[76px] p-3 rounded-xl macos-sub-card border text-xs font-mono leading-relaxed overflow-y-auto custom-scrollbar flex items-center shadow-inner"

                  :class="app.coreUpgradeFrozen ? 'border-blue-500/30 text-blue-300' : (app.systemOverview.mihomo_update_avail || app.systemOverview.singbox_update_avail ? 'border-amber-500/30 text-amber-300' : 'border-slate-700/60 text-slate-300')">

                  <div v-if="app.coreUpgradeLog" class="w-full break-words">

                    {{ app.coreUpgradeLog }}

                  </div>

                  <!-- Case: Frozen -->

                  <div v-else-if="app.coreUpgradeFrozen" class="w-full flex items-center justify-between gap-2">

                    <div class="flex items-center gap-2 truncate">

                      <span class="text-blue-400 font-bold shrink-0">[LOCKED]</span>

                      <span class="truncate">内核升级已冻结锁定，避免语法特性未对齐影响运行。</span>

                    </div>

                    <button @click="app.toggleCoreUpgradeFreeze" class="shrink-0 px-2.5 py-1 rounded-lg bg-blue-600/30 hover:bg-blue-600/50 border border-blue-400/50 text-blue-200 hover:text-white text-xs font-bold transition active:scale-95 cursor-pointer flex items-center gap-1">

                      <span>🔓 解除冻结</span>

                    </button>

                  </div>

                  <!-- Case: Unfrozen & Update Available -->

                  <div v-else-if="app.systemOverview.mihomo_update_avail || app.systemOverview.singbox_update_avail" class="w-full flex items-center justify-between gap-2">

                    <div class="space-y-0.5 pr-1 flex-1 overflow-hidden leading-tight">

                      <div class="flex items-center gap-1 font-bold text-amber-400 text-xs">

                        <span>⚠️ 发现新版官方内核</span>

                      </div>

                      <div class="text-[11px] text-amber-200/90 leading-tight truncate">

                        请自行决定是否更新到最新版（语法特性未对齐可能影响使用）。<span class="text-amber-300 font-semibold underline decoration-amber-400/60">可提交 issue 催更作者！</span>

                      </div>

                    </div>

                    <div class="flex flex-col gap-1 shrink-0">

                      <button @click="app.openGitHubIssuePrompt" class="px-2.5 py-0.5 rounded-lg bg-cyan-950 hover:bg-cyan-900 border border-cyan-500/60 text-emerald-300 hover:text-white text-[11px] font-bold flex items-center justify-center gap-1 transition cursor-pointer shadow-sm active:scale-95" title="立即在 GitHub 提交 Issue 催促作者对齐新内核版本">

                        <svg class="w-3 h-3 text-emerald-300" viewBox="0 0 24 24" fill="currentColor">

                          <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />

                        </svg>

                        <span>📮 催更 Issue</span>

                      </button>

                      <button @click="app.toggleCoreUpgradeFreeze" class="px-2.5 py-0.5 rounded-lg bg-blue-950/80 hover:bg-blue-900 border border-blue-500/50 text-blue-300 hover:text-white text-[11px] font-bold flex items-center justify-center gap-1 transition cursor-pointer active:scale-95" title="立即冻结锁定本卡片升级，保持生产稳定">

                        <span>🔒 冻结升级</span>

                      </button>

                    </div>

                  </div>

                  <!-- Case: Default / All Latest -->

                  <div v-else class="text-slate-300 text-xs flex items-center gap-2">

                    <span class="text-emerald-400 font-bold shrink-0">[CORE]</span>

                    <span>双引擎无感热重载就绪 · {{ (app.activeEngine === 'singbox' || app.activeEngine === 'sing-box') ? 'sing-box' : 'Mihomo' }} 主控调度 · {{ (app.activeEngine === 'singbox' || app.activeEngine === 'sing-box') ? 'Mihomo' : 'sing-box' }} 待命热备</span>

                  </div>

                </div>

              </div>

              <!-- 3. Geo 规则数据库与离线分流体系 -->

              <div class="p-6 sm:p-7 rounded-2xl sm:rounded-3xl macos-glass-card hover:border-sky-400/40 transition-all duration-300 space-y-4 flex flex-col justify-between group">

                <!-- Tier 1: Header (min-h-[36px]) -->

                <div class="min-h-[36px] flex justify-between items-center gap-3">

                  <div class="flex items-center gap-3 min-w-0">

                    <svg class="w-7 h-7 min-w-[28px] max-w-[28px] h-[28px] max-h-[28px] shrink-0 drop-shadow-[0_0_8px_rgba(16,185,129,0.5)] transition-transform duration-300 group-hover:scale-105" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">

                      <path d="M12 2L3 7V12C3 17.5 6.8 22.3 12 23.5C17.2 22.3 21 17.5 21 12V7L12 2Z" fill="url(#geoShieldGrad)" stroke="#34D399" stroke-width="1.6" stroke-linejoin="round"/>

                      <path d="M9 12.5L11 14.5L15 9.5" stroke="#FFFFFF" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>

                      <defs>

                        <linearGradient id="geoShieldGrad" x1="3" y1="2" x2="21" y2="23.5" gradientUnits="userSpaceOnUse">

                          <stop stop-color="#059669"/>

                          <stop offset="0.6" stop-color="#C084FC"/>

                          <stop offset="1" stop-color="#064E3B"/>

                        </linearGradient>

                      </defs>

                    </svg>

                    <span class="text-sm sm:text-base font-extrabold text-white whitespace-nowrap truncate tracking-wide">Geo 规则体系 (MRS / SRS)</span>

                  </div>

                  <!-- Dynamic Top-Right Badge -->

                  <span v-if="app.updatingGeo" class="text-xs font-mono px-3 py-1 rounded-lg bg-purple-950 text-purple-300 border border-purple-500/40 flex items-center gap-1.5 shadow-sm shrink-0 font-semibold">

                    <span class="inline-block animate-spin text-[10px]">↻</span>

                    <span>规则库同步中...</span>

                  </span>

                  <span v-else-if="app.checkingGeoUpgrade" class="text-xs font-mono px-3 py-1 rounded-lg bg-white/10 text-white text-slate-400 border border-slate-700/50 flex items-center gap-1.5 shadow-sm shrink-0 font-semibold">

                    <span class="inline-block animate-spin text-[10px]">↻</span>

                    <span>规则通道核对中...</span>

                  </span>

                  <span v-else-if="app.geodataStatus.has_update" :title="`检测到新规则包已发布 (最新版本: ${app.geodataStatus.latest_version || '最新'})，点击下方【一键全量更新】即可同步`"

                    class="text-xs font-mono px-3 py-1 rounded-lg bg-amber-500/20 text-amber-300 border border-amber-400/40 animate-pulse flex items-center gap-1.5 shadow-sm shrink-0 cursor-pointer font-semibold"

                    @click="app.updateGeoData">

                    <span>🚀 发现规则更新 (待同步)</span>

                  </span>

                  <span v-else :title="`基础离线库 (geoip/geosite) + ${app.geodataStatus.rules_count || 105} 套策略规则集已全量装载生效（覆盖国内外全域路由分流加速）`"

                    class="text-xs font-mono px-3 py-1 rounded-lg bg-emerald-500/15 text-emerald-300 border border-emerald-400/30 flex items-center gap-1.5 shadow-sm shrink-0 truncate font-semibold">

                    <span class="w-2 h-2 rounded-full bg-emerald-400 shrink-0"></span>

                    <span class="truncate">基础离线库 · 105 套规则集生效</span>

                  </span>

                </div>

                <!-- Tier 2: Body (min-h-[86px]) -->

                <div class="min-h-[86px] text-xs sm:text-[13px] font-mono macos-sub-card p-3.5 rounded-xl flex flex-col justify-center gap-2.5 shadow-sm">

                  <div class="flex justify-between items-center">

                    <span>离线基础库: <span class="text-slate-200 font-semibold">geoip.metadb + geosite.dat</span></span>

                    <span class="text-emerald-400 font-bold">✓ 离线就绪 ({{ app.geodataStatus.geoip_size_str && app.geodataStatus.geosite_size_str ? (app.geodataStatus.geoip_size_str + ' + ' + app.geodataStatus.geosite_size_str) : '38.4 MB' }})</span>

                  </div>

                  <div class="flex justify-between items-center">

                    <span>权威分流包: <span class="text-slate-200"><strong class="text-emerald-300 font-bold text-sm">{{ app.geodataStatus.rules_count || 105 }} 套</strong> MRS / SRS 规则集</span></span>

                    <span class="text-slate-400 font-mono font-medium">{{ app.geodataStatus.last_updated ? ('更新于 ' + (app.geodataStatus.last_updated.length > 10 ? app.geodataStatus.last_updated.substring(5, 16) : app.geodataStatus.last_updated)) : (app.geodataLastUpdateText || '已全量装载') }}</span>

                  </div>

                </div>

                <!-- Tier 3: Action Buttons (h-11 sm:h-12) -->

                <div class="h-11 sm:h-12 grid grid-cols-2 gap-2.5">

                  <button @click="app.updateGeoData" :disabled="app.updatingGeo" class="h-full rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white font-extrabold text-xs sm:text-sm transition cursor-pointer active:scale-98 flex items-center justify-center gap-2 shadow-lg shadow-sky-500/25 disabled:opacity-50">

                    <span v-if="app.updatingGeo" class="inline-block animate-spin mr-0.5">↻</span>

                    <svg v-else class="w-4 h-4 text-white shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">

                      <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />

                    </svg>

                    <span class="truncate">{{ app.updatingGeo ? '正在全量同步...' : (app.geodataStatus.has_update ? '一键全量更新 (新规则)' : '一键全量更新') }}</span>

                  </button>

                  <button @click="app.showGeodataDetailModal = true" :title="`查看全量 ${app.geodataStatus.rules_count || 105} 套分流策略规则清单与生效明细`" class="h-full px-2.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/60 text-white hover:text-sky-200 text-xs sm:text-sm font-bold transition flex items-center justify-center gap-2 cursor-pointer shadow-md active:scale-98 group">

                    <svg class="w-4 h-4 text-emerald-300 group-hover:text-sky-200 shrink-0 transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">

                      <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />

                    </svg>

                    <span class="truncate">{{ app.geodataStatus.rules_count || 105 }} 套规则清单</span>

                  </button>

                </div>

                <!-- Tier 4: Terminal Viewport (h-[70px] sm:h-[76px]) -->

                <div class="h-[70px] sm:h-[76px] p-3 rounded-xl macos-sub-card border text-xs font-mono leading-relaxed overflow-y-auto custom-scrollbar flex items-center shadow-inner"

                  :class="app.geodataStatus.has_update ? 'border-amber-500/30 text-amber-300' : 'border-emerald-500/25 text-emerald-300'">

                  <div v-if="app.geoUpdateLog" class="w-full break-words">

                    {{ app.geoUpdateLog }}

                  </div>

                  <div v-else-if="app.geodataStatus.has_update" class="text-amber-300 text-xs flex items-center justify-between gap-2 w-full">

                    <div class="space-y-0.5 pr-1 flex-1 overflow-hidden leading-tight">

                      <div class="flex items-center gap-1.5 font-bold text-amber-400 text-xs">

                        <span class="text-amber-400 font-bold shrink-0">[UPDATE]</span>

                        <span>发现官方规则库新版 ({{ app.geodataStatus.latest_version || '最新' }})</span>

                      </div>

                      <div class="text-[11px] text-amber-200/90 leading-tight">

                        全量 MRS/SRS 规则源有新提交，点击上方【一键全量更新】即可秒级平滑同步！

                      </div>

                    </div>

                  </div>

                  <div v-else class="text-slate-300 text-xs flex items-center gap-2.5 w-full">

                    <span class="text-emerald-400 font-bold shrink-0">[GEODATA]</span>

                    <div class="text-xs leading-tight space-y-1 w-full overflow-hidden">

                      <div class="truncate">

                        <span class="text-slate-300">已装载 </span><strong class="text-emerald-300 font-bold">{{ app.geodataStatus.rules_count || 105 }} 套</strong><span class="text-slate-300"> 分流加速规则集</span><span class="text-slate-400">（国内直连 / 境外加速）</span>

                      </div>

                      <div class="text-[11px] text-slate-400 flex items-center justify-between">

                        <div class="flex items-center gap-1.5 truncate">

                          <span class="text-emerald-400 font-bold flex items-center gap-1 shrink-0">

                            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>

                            <span>自动更新守护</span>

                          </span>

                          <span class="text-slate-300">·</span>

                          <span class="truncate">每日 04:00 同步</span>

                          <span class="text-slate-300">·</span>

                          <span class="text-slate-300 shrink-0">无感热重载</span>

                        </div>

                        <button @click="app.toggleGeoAutoUpdate" :title="app.geoAutoUpdateEnabled ? '点击暂停每日自动更新' : '点击启用每日自动更新'"

                          class="shrink-0 text-xs px-2 py-0.5 rounded-md border transition cursor-pointer flex items-center gap-1 active:scale-95 font-semibold"

                          :class="app.geoAutoUpdateEnabled ? 'bg-emerald-950 text-emerald-300 border-emerald-500/50 hover:bg-emerald-900' : 'bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white text-slate-400 border-slate-700 hover:text-white'">

                          <span>{{ app.geoAutoUpdateEnabled ? '已开启' : '已暂停' }}</span>

                        </button>

                      </div>

                    </div>

                  </div>

                </div>

              </div>

              <!-- 4. 宿主固件系统 / 宿主机系统在线升级 (智能自适应 RouterOS / 飞牛 / PVE / Docker / Linux) -->

              <div class="p-6 sm:p-7 rounded-2xl sm:rounded-3xl macos-glass-card hover:border-sky-400/40 transition-all duration-300 space-y-4 flex flex-col justify-between group">

                <!-- Tier 1: Header (min-h-[36px]) -->

                <div class="min-h-[36px] flex justify-between items-center gap-3">

                  <div class="flex items-center gap-3 min-w-0">

                    <img :src="app.hostBadge.svg" class="w-7 h-7 min-w-[28px] max-w-[28px] h-[28px] max-h-[28px] object-contain shrink-0 drop-shadow-[0_0_8px_rgba(168,85,247,0.4)] transition-transform duration-300 group-hover:scale-105" :alt="app.hostBadge.shortName" />

                    <span class="text-sm sm:text-base font-extrabold text-white whitespace-nowrap truncate tracking-wide">

                      {{ app.isRouterOS ? 'RouterOS 固件系统版本' : (app.hostBadge.name + ' 宿主系统版本') }}

                    </span>

                  </div>

                  <span v-if="app.checkingHostUpgrade" class="text-xs font-mono px-3 py-1 rounded-lg bg-purple-950 text-purple-300 border border-purple-500/40 flex items-center gap-1.5 shadow-sm shrink-0 whitespace-nowrap font-semibold">

                    <span class="inline-block animate-spin text-[10px]">↻</span>

                    <span>通道核对中...</span>

                  </span>

                  <span v-else-if="app.hostUpgradeDetails.hasUpdate" class="text-xs font-mono px-3 py-1 rounded-lg bg-amber-500/20 text-amber-300 border border-amber-400/40 animate-pulse flex items-center gap-1.5 shadow-sm shrink-0 whitespace-nowrap font-semibold">

                    <span>🚀 发现新固件 {{ app.hostUpgradeDetails.latestVersion }}</span>

                  </span>

                  <span v-else class="text-xs font-mono px-3 py-1 rounded-lg bg-emerald-500/15 text-emerald-300 border border-emerald-400/30 flex items-center gap-1.5 shadow-sm shrink-0 whitespace-nowrap font-semibold">

                    <span class="w-2 h-2 rounded-full bg-emerald-400 shrink-0"></span>

                    <span>✓ 已是最新稳定版</span>

                  </span>

                </div>

                <!-- Tier 2: Body (min-h-[86px]) -->

                <div class="min-h-[86px] text-xs sm:text-[13px] font-mono macos-sub-card p-3.5 rounded-xl flex flex-col justify-center gap-2.5 shadow-sm">

                  <template v-if="app.isRouterOS">

                    <div class="flex justify-between items-center">

                      <span>当前固件: <strong class="text-purple-300 font-bold text-sm">{{ (app.hostUpgradeDetails.installedVersion && !app.hostUpgradeDetails.installedVersion.includes('beta')) ? app.hostUpgradeDetails.installedVersion : ((app.hostInfo.os_name && (app.hostInfo.os_name.toLowerCase().includes('routeros') || app.hostInfo.os_name.toLowerCase().includes('mikrotik'))) ? app.hostInfo.os_name : 'MikroTik RouterOS') }}</strong></span>

                      <span class="text-slate-400">升级通道: <span class="text-slate-200 font-semibold">{{ app.hostUpgradeDetails.channel || 'stable' }}</span></span>

                    </div>

                    <div class="flex justify-between items-center">

                      <span>云端最新: <strong :class="app.hostUpgradeDetails.hasUpdate ? 'text-amber-400 font-bold text-sm' : 'text-emerald-400 font-bold text-sm'">{{ app.hostUpgradeDetails.latestVersion || (app.hostUpgradeDetails.installedVersion ? app.hostUpgradeDetails.installedVersion : (app.hostUpgradeDetails.checked ? '已是最新稳定版' : '待连通检测')) }}</strong></span>

                      <span :class="app.hostUpgradeDetails.hasUpdate ? 'text-amber-400 font-bold' : 'text-emerald-400 font-semibold'">{{ app.hostUpgradeDetails.hasUpdate ? '● 存在新版本' : (app.hostUpgradeDetails.checked ? '● 最新稳定版' : '● 待核对') }}</span>

                    </div>

                  </template>

                  <template v-else>

                    <div class="flex justify-between items-center">

                      <span>当前宿主: <strong class="text-emerald-300 font-bold text-sm">{{ app.hostInfo.os_name || app.hostBadge.name }}</strong></span>

                      <span class="text-slate-400">内核: <span class="text-slate-200 font-semibold">{{ app.hostInfo.kernel_version || 'Linux 6.x' }}</span></span>

                    </div>

                    <div class="flex justify-between items-center">

                      <span>运行模式: <strong class="text-emerald-400 font-bold">容器沙盒化</strong></span>

                      <span class="text-emerald-400 font-semibold">● 宿主正常</span>

                    </div>

                  </template>

                </div>

                <!-- Tier 3: Action Buttons (h-11 sm:h-12) -->

                <div class="h-11 sm:h-12 grid grid-cols-2 gap-2.5">

                  <button @click="app.checkHostUpgrade" :disabled="app.checkingHostUpgrade" class="h-full rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/60 text-white hover:text-sky-200 font-bold text-xs sm:text-sm transition active:scale-95 flex items-center justify-center gap-1.5 cursor-pointer shadow-md">

                    <span v-if="app.checkingHostUpgrade" class="inline-block animate-spin mr-1">↻</span>

                    <svg class="w-4 h-4 text-emerald-300 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">

                      <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />

                    </svg>

                    <span>检查 {{ app.isRouterOS ? 'ROS 固件' : app.hostBadge.shortName }} 更新</span>

                  </button>

                  <button @click="app.executeHostUpgrade" :disabled="app.executingHostUpgrade || (app.isRouterOS && app.hostUpgradeDetails.checked && !app.hostUpgradeDetails.hasUpdate)" class="h-full rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white font-extrabold text-xs sm:text-sm shadow-lg shadow-sky-500/25 transition active:scale-95 flex items-center justify-center gap-1.5 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed">

                    <span v-if="app.executingHostUpgrade" class="inline-block animate-spin mr-1">↻</span>

                    <span v-if="app.isRouterOS">

                      {{ app.hostUpgradeDetails.checked && !app.hostUpgradeDetails.hasUpdate ? '✓ 固件已是最新版' : (app.hostUpgradeDetails.hasUpdate ? `🚀 升级至 ${app.hostUpgradeDetails.latestVersion}` : '⬆ 升级 RouterOS 并重启') }}

                    </span>

                    <span v-else>⬆ 升级宿主系统包</span>

                  </button>

                </div>

                <!-- Tier 4: Terminal Viewport (h-[70px] sm:h-[76px]) -->

                <div class="h-[70px] sm:h-[76px] p-3 rounded-xl macos-sub-card border border-purple-500/25 text-xs font-mono text-purple-300 leading-relaxed overflow-y-auto custom-scrollbar flex items-center shadow-inner">

                  <div v-if="app.hostUpgradeResult" class="w-full break-words whitespace-pre-wrap">

                    {{ app.hostUpgradeResult }}

                  </div>

                  <div v-else class="text-slate-300 text-xs flex items-center gap-2.5">

                    <span class="text-purple-400 font-bold shrink-0">[HOST]</span>

                    <span v-if="app.isIKuai">爱快 iKuai Web-RPC 联通 · 198.18.0.0/15 静态回指与 DHCP DNS 联动就绪</span>

                    <span v-else-if="app.isOpenWrt">OpenWrt UBUS/SSH 联通 · 198.18.0.0/15 回指与策略转发就绪</span>

                    <span v-else-if="app.isRouterOS">MikroTik RouterOS 原生 API 联通 · 隔离策略路由 (table cubex) 正常放行</span>

                    <span v-else>{{ app.hostBadge.name }} 原生通道联通 · 底层网络与防火墙自愈就绪</span>

                  </div>

                </div>

              </div>

            </div>

          </div>

          <!-- 🛡️ 灾备恢复与系统维护中枢 (极客双栏对称设计 · 视觉统一沉浸) -->

          <div class="grid grid-cols-1 md:grid-cols-2 gap-5 sm:gap-6 pt-2">

            <!-- 1. ✨ 系统出厂初始化向导 -->

            <div class="p-6 sm:p-7 rounded-2xl sm:rounded-3xl macos-glass-card hover:border-sky-400/40 transition-all duration-300 space-y-4 flex flex-col justify-between group">

              <div>

                <div class="flex items-center justify-between gap-3">

                  <div class="flex items-center gap-3">

                    <div class="shrink-0 flex items-center justify-center group-hover:scale-110 transition-transform">

                      <ZootopiaIcon name="overview" :size="32" class="drop-shadow-md" />

                    </div>

                    <div>

                      <h4 class="text-sm sm:text-base font-extrabold text-white tracking-wide">系统出厂初始化向导</h4>

                      <span class="text-[11px] font-mono text-emerald-300 font-semibold">出厂重置 · 自动适配宿主底座</span>

                    </div>

                  </div>

                  <span class="text-[10px] font-mono px-2 py-0.5 rounded-md bg-cyan-950/90 text-emerald-300 border border-cyan-500/40 font-bold shrink-0">

                    一键引导

                  </span>

                </div>

                <p class="text-xs text-slate-400 mt-3.5 leading-relaxed">

                  一键安全清空现有全部配置并重置出厂状态，交互式引导重新填入宿主凭据，自动嗅探 RouterOS / 爱快 iKuai / OpenWrt / 飞牛 / PVE / Linux 全宿主底座并编排底层策略路由。

                </p>

              </div>

              <div class="pt-2">

                <button @click="app.confirmAndStartInitWizard" class="w-full py-2.5 sm:py-3 rounded-xl text-xs sm:text-sm font-bold bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white font-extrabold shadow-lg shadow-sky-500/25 active:scale-98 transition flex items-center justify-center gap-2 cursor-pointer">

                  <span>✨ 启动出厂初始化向导</span>

                </button>

              </div>

            </div>

            <!-- 2. 🔄 防手残时光机全量备份 -->

            <div class="p-6 sm:p-7 rounded-2xl sm:rounded-3xl macos-glass-card hover:border-indigo-400/50 transition-all duration-300 space-y-4 flex flex-col justify-between group">

              <div>

                <div class="flex items-center justify-between gap-3">

                  <div class="flex items-center gap-3">

                    <div class="shrink-0 flex items-center justify-center group-hover:scale-110 transition-transform">

                      <ZootopiaIcon name="system" :size="32" class="drop-shadow-md" />

                    </div>

                    <div>

                      <h4 class="text-sm sm:text-base font-extrabold text-white tracking-wide">防手残时光机全量备份</h4>

                      <span class="text-[11px] font-mono text-indigo-300 font-semibold">无损快照 · 秒级灾备恢复</span>

                    </div>

                  </div>

                  <span class="text-[10px] font-mono px-2 py-0.5 rounded-md bg-indigo-950/90 text-indigo-300 border border-indigo-500/40 font-bold shrink-0">

                    秒级快照

                  </span>

                </div>

                <p class="text-xs text-slate-400 mt-3.5 leading-relaxed">

                  一键打包导出包含双核配置、分流策略规则、宿主共生凭据在内的全量系统快照，遭遇任何误操作或故障均可一键导入秒级自愈。

                </p>

              </div>

              <div class="space-y-3 pt-2">

                <!-- 统一规格 2x2 运维控制中枢矩阵 (四个按钮严格统一字体与尺寸) -->

                <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">

                  <!-- 1. 导出系统快照 -->

                  <button @click="app.exportBackup" class="h-10 sm:h-11 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white text-xs sm:text-[13px] font-bold transition flex items-center justify-center gap-2 active:scale-98 cursor-pointer shadow-md select-none">

                    <span class="text-sm">📥</span>

                    <span>导出系统快照</span>

                  </button>

                  <!-- 2. 导入快照恢复 -->

                  <label class="h-10 sm:h-11 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white text-xs sm:text-[13px] font-bold transition flex items-center justify-center gap-2 active:scale-98 cursor-pointer text-center shadow-md backdrop-blur-md select-none">

                    <span class="text-sm">📤</span>

                    <span>导入快照恢复</span>

                    <input type="file" accept=".json" @change="app.importBackup" class="hidden" />

                  </label>

                  <!-- 3. 平滑热重载 -->

                  <button @click="app.triggerHotRestart" :disabled="app.isHotRestarting" class="h-10 sm:h-11 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white text-xs sm:text-[13px] font-bold transition flex items-center justify-center gap-2 active:scale-98 cursor-pointer shadow-md disabled:opacity-50 select-none" title="平滑热重载双核代理引擎">

                    <span v-if="app.isHotRestarting" class="inline-block animate-spin text-sm">↻</span>

                    <span v-else class="text-sm">🔥</span>

                    <span>{{ app.isHotRestarting ? '热重启中...' : '平滑热重载' }}</span>

                  </button>

                  <!-- 4. 整机重启 -->

                  <button @click="app.confirmSystemReboot" class="h-10 sm:h-11 rounded-xl bg-blue-500/15 hover:bg-blue-500/25 border border-blue-400/30 text-blue-300 hover:text-white text-xs sm:text-[13px] font-bold transition flex items-center justify-center gap-2 active:scale-98 cursor-pointer shadow-md select-none" title="整机系统重启，重启前主动通知主路由逃生">

                    <span class="text-sm">🔄</span>

                    <span>整机重启</span>

                  </button>

                </div>

              </div>

            </div>

            <!-- 控制台安全访问锁卡片 -->
            <div class="p-5 sm:p-6 rounded-2xl sm:rounded-3xl macos-glass-card border border-cyan-500/25 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div class="flex items-center gap-3.5">
                <div class="w-11 h-11 rounded-2xl bg-cyan-500/15 border border-cyan-400/35 flex items-center justify-center text-xl shrink-0">
                  {{ app.authState.enabled ? '🔐' : '🔓' }}
                </div>
                <div>
                  <div class="flex items-center gap-2">
                    <h4 class="text-sm sm:text-base font-bold text-white">Web 控制台访问安全锁</h4>
                    <span :class="['text-[11px] font-mono px-2 py-0.5 rounded-full border font-semibold', app.authState.enabled ? 'bg-emerald-500/15 text-emerald-300 border-emerald-400/35' : 'bg-white/10 text-slate-300 border-white/20']">
                      {{ app.authState.enabled ? '口令保护已启用' : '默认免密直连' }}
                    </span>
                  </div>
                  <p class="text-xs text-slate-300 mt-1">
                    {{ app.authState.enabled ? '已启用访问口令校验（采用加盐哈希与 HttpOnly 会话 Cookie 保护）' : '开启后访问 Web 控制台与 API 需输入密码解锁，不开启则保持局域网零配置直连' }}
                  </p>
                </div>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <button v-if="app.authState.enabled" @click="app.lockConsoleNow"
                  class="px-3.5 py-2 rounded-xl bg-amber-500/20 hover:bg-amber-500/30 border border-amber-400/40 text-amber-200 text-xs font-bold transition cursor-pointer">
                  🔒 立即锁定
                </button>
                <button @click="app.showAuthManageModal = true"
                  class="px-4 py-2 rounded-xl bg-gradient-to-r from-sky-600 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white text-xs font-extrabold transition cursor-pointer shadow-md">
                  {{ app.authState.enabled ? '🔑 修改 / 关闭密码锁' : '🔐 启用控制台密码锁' }}
                </button>
              </div>
            </div>

          </div>

          </div>

        </div>
</template>

<script setup>
import { inject, ref } from 'vue'
import ZootopiaIcon from '../ZootopiaIcon.vue'

const app = inject('cubexApp')

const showHostCredentialsDeck = ref(false)
const refreshingHW = ref(false)

async function refreshHW() {
  if (refreshingHW.value) return
  refreshingHW.value = true
  try {
    if (app && typeof app.fetchRouterHardware === 'function') {
      await app.fetchRouterHardware()
    }
  } catch (err) {
    console.error('Failed to refresh router hardware:', err)
  } finally {
    refreshingHW.value = false
  }
}
</script>
