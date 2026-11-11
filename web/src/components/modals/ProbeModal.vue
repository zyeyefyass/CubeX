<template>
    <div v-if="app.showProbeModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-md animate-in fade-in">

      <div class="glass-matrix border border-cyan-500/40 rounded-3xl p-6 sm:p-7 max-w-4xl w-full max-h-[90vh] flex flex-col shadow-2xl relative space-y-5 text-slate-200">

        <!-- Header -->

        <div class="flex items-center justify-between pb-3 border-b border-slate-700/60">

          <div class="flex items-center gap-3">

            <span class="text-3xl p-2 rounded-2xl bg-cyan-950/60 border border-cyan-500/40" :class="{'animate-spin': app.isProbingAll}">🛰️</span>

            <div>

              <h3 class="text-lg font-bold text-white flex items-center gap-2">

                <span>CubeX 四维全息智能体检中心</span>

                <span v-if="app.isProbingAll" class="text-xs px-2.5 py-0.5 rounded-full bg-white/10 text-sky-200 border border-sky-400/30 font-bold drop-shadow-sm animate-pulse">正在并发探测...</span>

                <span v-else-if="app.hasAnyUpdate" class="text-xs px-2.5 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-400/40 font-semibold">发现可用更新</span>

                <span v-else class="text-xs px-2.5 py-0.5 rounded-full bg-emerald-500/15 text-emerald-300 border border-emerald-400/30 font-semibold">✓ 全维度最新稳定版</span>

              </h3>

              <p class="text-xs text-slate-400 mt-0.5">主控守护 · 双核引擎 · 217heidai 规则库 · 宿主固件 4 维度全栈健康实时诊断</p>

            </div>

          </div>

          <button @click="app.showProbeModal = false" class="p-2 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white/80 hover:bg-slate-700 text-slate-400 hover:text-white transition cursor-pointer">

            <span class="text-lg leading-none">✕</span>

          </button>

        </div>

        <!-- Body: 4 Dimension Grid -->

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 overflow-y-auto max-h-[60vh] pr-1">

          <!-- 维度 1: CubeX 主控守护进程 -->

          <div class="p-4 rounded-2xl bg-white/10 text-white/85 border border-slate-800 space-y-3 flex flex-col justify-between">

            <div class="flex justify-between items-center">

              <span class="text-xs font-bold text-white flex items-center gap-2">

                <img src="/logo-cubex.svg" class="w-5 h-5 min-w-[20px] max-w-[20px] h-[20px] max-h-[20px] object-contain shrink-0 drop-shadow-[0_0_5px_rgba(56,189,248,0.4)]" alt="CubeX" />

                <span>CubeX 系统守护进程 (cubexd)</span>

              </span>

              <span v-if="app.systemOverview.panel_update_avail" class="text-[10px] font-mono px-2 py-0.5 rounded bg-amber-500/20 text-amber-300 border border-amber-400/40">发现新版 {{ app.systemOverview.panel_latest_version }}</span>

              <span v-else class="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-300 border border-emerald-400/30">✓ 已是最新</span>

            </div>

            <div class="text-xs font-mono bg-slate-950/60 p-2.5 rounded-xl border border-slate-800/60 space-y-1">

              <div class="flex justify-between text-slate-400">

                <span>当前运行:</span>

                <strong class="text-emerald-300">{{ app.systemOverview.cubex_version || 'Test v8.4' }}</strong>

              </div>

              <div class="flex justify-between text-slate-400">

                <span>云端 Release:</span>

                <span class="text-slate-200">{{ app.systemOverview.panel_latest_version || app.systemOverview.cubex_version || 'Test v8.4' }}</span>

              </div>

            </div>

            <button v-if="app.systemOverview.panel_update_avail" @click="app.openUpgradeModal" :disabled="app.upgradingPanel" class="w-full py-1.5 rounded-xl bg-cyan-600 hover:bg-cyan-500 text-slate-950 text-xs font-bold transition cursor-pointer">

              {{ app.upgradingPanel ? '正在热更...' : '一键平滑热更 CubeX' }}

            </button>

            <div v-else class="text-[11px] text-center text-slate-300 py-1">当前主控版本处于稳定就绪状态</div>

          </div>

          <!-- 维度 2: 双核驱动引擎 -->

          <div class="p-4 rounded-2xl bg-white/10 text-white/85 border border-slate-800 space-y-3 flex flex-col justify-between">

            <div class="flex justify-between items-center">

              <span class="text-xs font-bold text-white flex items-center gap-2">

                <div class="flex items-center gap-2 shrink-0">

                  <img src="/logo-mihomo.svg" class="w-5 h-5 object-contain shrink-0 drop-shadow-[0_0_6px_rgba(56,189,248,0.4)]" alt="Mihomo" />

                  <img src="/logo-singbox.svg" class="w-5 h-5 object-contain shrink-0 drop-shadow-[0_0_6px_rgba(56,189,248,0.4)]" alt="sing-box" />

                </div>

                <span>双核引擎 (Mihomo × sing-box)</span>

              </span>

              <span class="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-300 border border-emerald-400/30">✓ 双核就绪</span>

            </div>

            <div class="text-xs font-mono bg-slate-950/60 p-2.5 rounded-xl border border-slate-800/60 space-y-1">

              <div class="flex justify-between text-slate-400">

                <span>Mihomo 核心:</span>

                <span class="text-slate-200">{{ app.systemOverview.mihomo_curr_version || 'v1.19.16' }}</span>

              </div>

              <div class="flex justify-between text-slate-400">

                <span>sing-box 核心:</span>

                <span class="text-slate-200">{{ app.systemOverview.singbox_curr_version || 'v1.14.4' }}</span>

              </div>

            </div>

            <div class="text-[11px] text-center text-slate-300 py-1">双引擎深度混动，支持毫秒级无感热容灾切换</div>

          </div>

          <!-- 维度 3: 217heidai 规则库 -->

          <div class="p-4 rounded-2xl bg-white/10 text-white/85 border border-slate-800 space-y-3 flex flex-col justify-between">

            <div class="flex justify-between items-center">

              <span class="text-xs font-bold text-white flex items-center gap-1.5">

                <span>🛡️</span>

                <span>217heidai 权威规则体系</span>

              </span>

              <span class="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-300 border border-emerald-400/30">✓ {{ (app.geodataStatus.total_rules || 236255).toLocaleString() }} 条规则</span>

            </div>

            <div class="text-xs font-mono bg-slate-950/60 p-2.5 rounded-xl border border-slate-800/60 space-y-1">

              <div class="flex justify-between text-slate-400">

                <span>版本标识:</span>

                <span class="text-slate-200">{{ app.geodataStatus.version || '20261021' }}</span>

              </div>

              <div class="flex justify-between text-slate-400">

                <span>规则文件:</span>

                <span class="text-emerald-400">{{ app.geodataStatus.rules_count || 105 }} 套 MRS/SRS 二进制规则</span>

              </div>

            </div>

            <button @click="app.updateGeoData" :disabled="app.updatingGeo" class="w-full py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition">

              {{ app.updatingGeo ? '正在下载更新...' : '↻ 立即更新规则库' }}

            </button>

          </div>

          <!-- 维度 4: 宿主底座与固件 -->

          <div class="p-4 rounded-2xl bg-white/10 text-white/85 border border-slate-800 space-y-3 flex flex-col justify-between">

            <div class="flex justify-between items-center">

              <span class="text-xs font-bold text-white flex items-center gap-2">

                <img :src="app.hostBadge.svg" class="w-5 h-5 min-w-[20px] max-w-[20px] h-[20px] max-h-[20px] object-contain shrink-0 drop-shadow-[0_0_5px_rgba(56,189,248,0.35)]" :alt="app.hostBadge.shortName" />

                <span>宿主底层固件 ({{ app.isRouterOS ? 'RouterOS v7' : app.hostBadge.name }})</span>

              </span>

              <span v-if="app.hostUpgradeDetails.hasUpdate" class="text-[10px] font-mono px-2 py-0.5 rounded bg-purple-950 text-purple-300 border border-purple-500/40">可升级 {{ app.hostUpgradeDetails.latestVersion }}</span>

              <span v-else class="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-300 border border-emerald-400/30">✓ 系统健康</span>

            </div>

            <div class="text-xs font-mono bg-slate-950/60 p-2.5 rounded-xl border border-slate-800/60 space-y-1">

              <div class="flex justify-between text-slate-400">

                <span>宿主版本:</span>

                <span class="text-slate-200">{{ app.hostInfo.os_version || '7.18+' }} ({{ app.hostInfo.arch || 'x86_64' }})</span>

              </div>

              <div class="flex justify-between text-slate-400">

                <span>容器底座:</span>

                <span class="text-purple-300">VETH 隔离 · 宿主零损伤</span>

              </div>

            </div>

            <button v-if="app.hostUpgradeDetails.hasUpdate" @click="app.executeHostUpgrade" :disabled="upgradingHost" class="w-full py-1.5 rounded-xl bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold transition">

              {{ upgradingHost ? '正在升级宿主...' : '一键升级宿主固件' }}

            </button>

            <div v-else class="text-[11px] text-center text-slate-300 py-1">宿主固件处于最佳稳定运行通道</div>

          </div>

        </div>

        <!-- Footer Actions -->

        <div class="pt-3 border-t border-slate-800 flex items-center justify-between">

          <div class="text-xs text-slate-400">

            上次体检时间: <span class="font-mono text-emerald-300">{{ app.probeLastTime || '刚刚' }}</span>

          </div>

          <div class="flex items-center gap-2">

            <button @click="app.probeAllFourDimensions" :disabled="app.isProbingAll" class="px-3.5 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-slate-200 text-xs font-bold transition cursor-pointer flex items-center gap-1.5">

              <span :class="{'animate-spin': app.isProbingAll}">↻</span>

              <span>重新体检</span>

            </button>

            <button @click="app.showProbeModal = false" class="px-4 py-1.5 rounded-xl bg-cyan-600 hover:bg-cyan-500 text-slate-950 text-xs font-bold transition cursor-pointer shadow">

              完成并关闭

            </button>

          </div>

        </div>

      </div>

    </div>
</template>

<script setup>
import { inject } from 'vue'
const app = inject('cubexApp')
</script>
