<template>
    <div v-if="app.showProbeModal" class="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-black/60 animate-in fade-in" @click.self="app.showProbeModal = false">

      <div class="cbx-modal-shell rounded-3xl p-5 sm:p-7 max-w-4xl w-full max-h-[90vh] flex flex-col relative space-y-5">

        <!-- Header -->

        <div class="flex items-center justify-between pb-3 border-b gap-3" style="border-color: var(--cbx-border-sub);">

          <div class="flex items-center gap-3 min-w-0">

            <span class="text-2xl sm:text-3xl p-2 rounded-2xl cbx-badge-info shrink-0" :class="{'animate-spin': app.isProbingAll}">🛰️</span>

            <div class="min-w-0">

              <h3 class="text-base sm:text-lg font-bold cbx-text-main flex items-center gap-2 flex-wrap">

                <span>CubeX 系统与规则全栈体检中心</span>

                <span v-if="app.isProbingAll" class="text-xs px-2.5 py-0.5 rounded-full cbx-badge-info font-bold animate-pulse">正在并发探测...</span>

                <span v-else-if="app.hasAnyUpdate" class="text-xs px-2.5 py-0.5 rounded-full cbx-badge-warn font-semibold">发现可用更新</span>

                <span v-else class="text-xs px-2.5 py-0.5 rounded-full cbx-badge-ok font-semibold">✓ 全维度最新稳定版</span>

              </h3>

              <p class="text-xs cbx-text-muted mt-0.5">主控守护 · 双核引擎 · 217heidai 规则库 · 宿主固件 4 维度全栈健康实时诊断</p>

            </div>

          </div>

          <button @click="app.showProbeModal = false" aria-label="关闭体检中心" class="cbx-btn-ghost min-w-[36px] min-h-[36px] p-2 rounded-xl transition cursor-pointer shrink-0 flex items-center justify-center cbx-touch-extend">

            <span class="text-sm leading-none font-bold">✕</span>

          </button>

        </div>

        <!-- Body: 4 Dimension Grid -->

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 overflow-y-auto max-h-[60vh] pr-1">

          <!-- 维度 1: CubeX 主控守护进程 -->

          <div class="p-4 rounded-2xl cbx-sub-card space-y-3 flex flex-col justify-between">

            <div class="flex justify-between items-center gap-2">

              <span class="text-xs font-bold cbx-text-main flex items-center gap-2 min-w-0">

                <img src="/logo-cubex.svg" class="w-5 h-5 object-contain shrink-0" alt="CubeX" />

                <span class="truncate">CubeX 系统守护进程 (cubexd)</span>

              </span>

              <span v-if="app.systemOverview.panel_update_avail" class="text-[10px] font-mono px-2 py-0.5 rounded cbx-badge-warn shrink-0 font-bold">发现新版 {{ app.systemOverview.panel_latest_version }}</span>

              <span v-else class="text-[10px] font-mono px-2 py-0.5 rounded cbx-badge-ok shrink-0 font-bold">✓ 已是最新</span>

            </div>

            <div class="text-xs font-mono cbx-card !rounded-xl p-2.5 space-y-1">

              <div class="flex justify-between cbx-text-muted">

                <span>当前运行:</span>

                <strong class="font-tabular" style="color: var(--cbx-ok-fg);">{{ app.systemOverview.cubex_version || '—' }}</strong>

              </div>

              <div class="flex justify-between cbx-text-muted">

                <span>云端 Release:</span>

                <span class="cbx-text-main font-semibold font-tabular">{{ app.systemOverview.panel_latest_version || app.systemOverview.cubex_version || '—' }}</span>

              </div>

            </div>

            <button v-if="app.systemOverview.panel_update_avail" @click="app.openUpgradeModal" :disabled="app.upgradingPanel" class="w-full py-2 rounded-xl bg-sky-600 hover:bg-sky-700 text-white text-xs font-bold transition cursor-pointer">

              {{ app.upgradingPanel ? '正在热更...' : '一键平滑热更 CubeX' }}

            </button>

            <div v-else class="text-[11px] text-center cbx-text-sub py-1">当前主控版本处于稳定就绪状态</div>

          </div>

          <!-- 维度 2: 双核驱动引擎 -->

          <div class="p-4 rounded-2xl cbx-sub-card space-y-3 flex flex-col justify-between">

            <div class="flex justify-between items-center gap-2">

              <span class="text-xs font-bold cbx-text-main flex items-center gap-2 min-w-0">

                <div class="flex items-center gap-1.5 shrink-0">

                  <img src="/logo-mihomo.svg" class="w-5 h-5 object-contain shrink-0" alt="Mihomo" />

                  <img src="/logo-singbox.svg" class="w-5 h-5 object-contain shrink-0" alt="sing-box" />

                </div>

                <span class="truncate">双核引擎 (Mihomo × sing-box)</span>

              </span>

              <span class="text-[10px] font-mono px-2 py-0.5 rounded cbx-badge-ok shrink-0 font-bold">✓ 双核就绪</span>

            </div>

            <div class="text-xs font-mono cbx-card !rounded-xl p-2.5 space-y-1">

              <div class="flex justify-between cbx-text-muted">

                <span>Mihomo 核心:</span>

                <span class="cbx-text-main font-semibold font-tabular">{{ app.systemOverview.mihomo_curr_version || '--' }}</span>

              </div>

              <div class="flex justify-between cbx-text-muted">

                <span>sing-box 核心:</span>

                <span class="cbx-text-main font-semibold font-tabular">{{ app.systemOverview.singbox_curr_version || '--' }}</span>

              </div>

            </div>

            <div class="text-[11px] text-center cbx-text-sub py-1">双引擎深度混动，支持毫秒级无感热容灾切换</div>

          </div>

          <!-- 维度 3: 217heidai 规则库 -->

          <div class="p-4 rounded-2xl cbx-sub-card space-y-3 flex flex-col justify-between">

            <div class="flex justify-between items-center gap-2">

              <span class="text-xs font-bold cbx-text-main flex items-center gap-1.5 min-w-0">

                <span>🛡️</span>

                <span class="truncate">217heidai 权威规则体系</span>

              </span>

              <span class="text-[10px] font-mono px-2 py-0.5 rounded cbx-badge-ok shrink-0 font-bold">✓ {{ (app.geodataStatus.total_rules || 0).toLocaleString() }} 条规则</span>

            </div>

            <div class="text-xs font-mono cbx-card !rounded-xl p-2.5 space-y-1">

              <div class="flex justify-between cbx-text-muted">

                <span>版本标识:</span>

                <span class="cbx-text-main font-semibold">{{ app.geodataStatus.version || '--' }}</span>

              </div>

              <div class="flex justify-between cbx-text-muted">

                <span>规则文件:</span>

                <span class="font-bold" style="color: var(--cbx-ok-fg);">{{ app.geodataStatus.rules_count || 0 }} 套 MRS/SRS 二进制规则</span>

              </div>

            </div>

            <button @click="app.updateGeoData" :disabled="app.updatingGeo" class="w-full py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold transition cursor-pointer disabled:opacity-50">

              {{ app.updatingGeo ? '正在下载更新...' : '↻ 立即更新规则库' }}

            </button>

          </div>

          <!-- 维度 4: 宿主底座与固件 -->

          <div class="p-4 rounded-2xl cbx-sub-card space-y-3 flex flex-col justify-between">

            <div class="flex justify-between items-center gap-2">

              <span class="text-xs font-bold cbx-text-main flex items-center gap-2 min-w-0">

                <img :src="app.hostBadge.svg" class="w-5 h-5 object-contain shrink-0" :alt="app.hostBadge.shortName" />

                <span class="truncate">宿主底层固件 ({{ app.isRouterOS ? 'RouterOS v7' : app.hostBadge.name }})</span>

              </span>

              <span v-if="app.hostUpgradeDetails.hasUpdate" class="text-[10px] font-mono px-2 py-0.5 rounded cbx-badge-purple shrink-0 font-bold">可升级 {{ app.hostUpgradeDetails.latestVersion }}</span>

              <span v-else class="text-[10px] font-mono px-2 py-0.5 rounded cbx-badge-ok shrink-0 font-bold">✓ 系统健康</span>

            </div>

            <div class="text-xs font-mono cbx-card !rounded-xl p-2.5 space-y-1">

              <div class="flex justify-between cbx-text-muted">

                <span>宿主版本:</span>

                <span class="cbx-text-main font-semibold">{{ app.hostInfo.os_version || '--' }} ({{ app.hostInfo.arch || 'amd64' }})</span>

              </div>

              <div class="flex justify-between cbx-text-muted">

                <span>容器底座:</span>

                <span class="font-bold" style="color: var(--cbx-purple-fg);">VETH 隔离 · 宿主零损伤</span>

              </div>

            </div>

            <button v-if="app.hostUpgradeDetails.hasUpdate" @click="app.executeHostUpgrade" :disabled="app.executingHostUpgrade" class="w-full py-2 rounded-xl bg-purple-600 hover:bg-purple-700 text-white text-xs font-bold transition cursor-pointer disabled:opacity-50">

              {{ app.executingHostUpgrade ? '正在升级宿主...' : '一键升级宿主固件' }}

            </button>

            <div v-else class="text-[11px] text-center cbx-text-sub py-1">宿主固件处于最佳稳定运行通道</div>

          </div>

        </div>

        <!-- Footer Actions -->

        <div class="pt-3 border-t flex flex-wrap items-center justify-between gap-2" style="border-color: var(--cbx-border-sub);">

          <div class="text-xs cbx-text-muted">

            上次体检时间: <span class="font-mono font-bold" style="color: var(--cbx-ok-fg);">{{ app.probeLastTime || '刚刚' }}</span>

          </div>

          <div class="flex items-center gap-2">

            <button @click="app.probeAllFourDimensions" :disabled="app.isProbingAll" class="cbx-btn-ghost px-3.5 py-2 rounded-xl text-xs font-bold transition cursor-pointer flex items-center gap-1.5">

              <span :class="{'animate-spin': app.isProbingAll}">↻</span>

              <span>重新体检</span>

            </button>

            <button @click="app.showProbeModal = false" class="px-4 py-2 rounded-xl bg-sky-600 hover:bg-sky-700 text-white text-xs font-bold transition cursor-pointer shadow-sm">

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
