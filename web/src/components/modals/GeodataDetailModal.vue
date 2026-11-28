<template>
          <div v-if="app.showGeodataDetailModal" class="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-black/60 animate-in fade-in" @click.self="app.showGeodataDetailModal = false">

            <div class="cbx-modal-shell rounded-3xl p-5 sm:p-6 max-w-4xl w-full max-h-[88dvh] flex flex-col relative space-y-4">

              <!-- Modal Header -->

              <div class="flex items-center justify-between border-b pb-3 gap-3" style="border-color: var(--cbx-border-sub);">

                <div class="flex items-center gap-3 min-w-0">

                  <span class="text-2xl p-2 rounded-2xl cbx-badge-ok shrink-0">📚</span>

                  <div class="min-w-0">

                    <h3 class="text-base sm:text-lg font-bold cbx-text-main flex items-center gap-2 flex-wrap">

                      <span>Geo 数据库 & 217heidai 规则包全景清单</span>

                      <span class="text-[11px] px-2 py-0.5 rounded-full cbx-badge-ok font-mono font-bold">

                        {{ app.geodataStatus.rules_count || 105 }} 套规则 · 已全量装载

                      </span>

                    </h3>

                    <p class="text-xs cbx-text-muted mt-0.5">

                      全套二进制 MRS/SRS 离线分流体系与 217heidai 去广告权威拦截规则详细清单

                    </p>

                  </div>

                </div>

                <button @click="app.showGeodataDetailModal = false" aria-label="关闭弹窗" class="cbx-btn-ghost min-w-[36px] min-h-[36px] px-2.5 py-1.5 rounded-xl text-sm font-bold transition cursor-pointer shrink-0 flex items-center justify-center cbx-touch-extend">✕</button>

              </div>

              <!-- Summary Metrics Bar -->

              <div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5 text-xs font-mono">

                <div class="p-2.5 rounded-xl cbx-sub-card">

                  <div class="text-[10px] cbx-text-muted">拦截规则总规模</div>

                  <div class="font-bold text-sm mt-0.5 font-tabular" style="color: var(--cbx-ok-fg);">{{ (app.geodataStatus.total_rules || 240005).toLocaleString() }} 条</div>

                </div>

                <div class="p-2.5 rounded-xl cbx-sub-card">

                  <div class="text-[10px] cbx-text-muted">规则库版本</div>

                  <div class="font-bold text-sm mt-0.5" style="color: var(--cbx-ok-fg);">{{ app.geodataStatus.version || '最新稳定版' }}</div>

                </div>

                <div class="p-2.5 rounded-xl cbx-sub-card">

                  <div class="text-[10px] cbx-text-muted">装载规则文件</div>

                  <div class="font-bold text-sm mt-0.5 font-tabular" style="color: var(--cbx-purple-fg);">{{ app.geodataStatus.files_count || app.geodataStatus.rules_count || 105 }} 个</div>

                </div>

                <div class="p-2.5 rounded-xl cbx-sub-card">

                  <div class="text-[10px] cbx-text-muted">最近校验/同步</div>

                  <div class="cbx-text-main font-bold text-xs mt-1 truncate">{{ app.geodataStatus.last_updated || '已装载' }}</div>

                </div>

              </div>

              <!-- Filter & Search Bar -->

              <div class="flex flex-col gap-2.5 text-xs">

                <div class="flex flex-wrap items-center justify-between gap-2">

                  <!-- Category Pills -->

                  <div class="flex items-center gap-1.5 overflow-x-auto py-0.5">

                    <button @click="app.geodataFilterCategory = 'all'" :class="['px-2.5 py-1.5 rounded-lg font-semibold transition cursor-pointer shrink-0', app.geodataFilterCategory === 'all' ? 'bg-emerald-600 text-white font-bold' : 'cbx-btn-ghost']">

                      全部分类 ({{ app.geodataStatus.files?.length || 0 }})

                    </button>

                    <button @click="app.geodataFilterCategory = '基础数据库'" :class="['px-2.5 py-1.5 rounded-lg font-semibold transition cursor-pointer shrink-0', app.geodataFilterCategory === '基础数据库' ? 'bg-emerald-600 text-white font-bold' : 'cbx-btn-ghost']">

                      基础库 ({{ app.geodataStatus.files?.filter(f => f.category === '基础数据库').length || 0 }})

                    </button>

                    <button @click="app.geodataFilterCategory = '广告与恶意追踪拦截'" :class="['px-2.5 py-1.5 rounded-lg font-semibold transition cursor-pointer shrink-0', app.geodataFilterCategory === '广告与恶意追踪拦截' ? 'bg-emerald-600 text-white font-bold' : 'cbx-btn-ghost']">

                      去广告 ({{ app.geodataStatus.files?.filter(f => f.category === '广告与恶意追踪拦截').length || 0 }})

                    </button>

                    <button @click="app.geodataFilterCategory = '基础路由分流'" :class="['px-2.5 py-1.5 rounded-lg font-semibold transition cursor-pointer shrink-0', app.geodataFilterCategory === '基础路由分流' ? 'bg-emerald-600 text-white font-bold' : 'cbx-btn-ghost']">

                      基础路由 ({{ app.geodataStatus.files?.filter(f => f.category === '基础路由分流').length || 0 }})

                    </button>

                    <button @click="app.geodataFilterCategory = '应用专线分流'" :class="['px-2.5 py-1.5 rounded-lg font-semibold transition cursor-pointer shrink-0', app.geodataFilterCategory === '应用专线分流' ? 'bg-emerald-600 text-white font-bold' : 'cbx-btn-ghost']">

                      核心专线 ({{ app.geodataStatus.files?.filter(f => f.category === '应用专线分流').length || 0 }})

                    </button>

                    <button @click="app.geodataFilterCategory = '常用生态与自定义扩展'" :class="['px-2.5 py-1.5 rounded-lg font-semibold transition cursor-pointer shrink-0', app.geodataFilterCategory === '常用生态与自定义扩展' ? 'bg-emerald-600 text-white font-bold' : 'cbx-btn-ghost']">

                      常用生态扩展 ({{ app.geodataStatus.files?.filter(f => f.category === '常用生态与自定义扩展').length || 0 }})

                    </button>

                  </div>

                  <!-- Engine Toggles -->

                  <div class="flex items-center gap-1 cbx-sub-card p-1 rounded-xl text-[11px]">

                    <button @click="app.geodataFilterEngine = 'all'" :class="['px-2.5 py-1 rounded-lg transition cursor-pointer', app.geodataFilterEngine === 'all' ? 'bg-sky-600 text-white font-bold' : 'cbx-text-sub hover:cbx-text-main']">

                      全部引擎

                    </button>

                    <button @click="app.geodataFilterEngine = 'Mihomo'" :class="['px-2.5 py-1 rounded-lg transition cursor-pointer flex items-center gap-1', app.geodataFilterEngine === 'Mihomo' ? 'cbx-badge-ok font-bold' : 'cbx-text-sub hover:cbx-text-main']">

                      <span>Mihomo (MRS)</span>

                    </button>

                    <button @click="app.geodataFilterEngine = 'sing-box'" :class="['px-2.5 py-1 rounded-lg transition cursor-pointer flex items-center gap-1', app.geodataFilterEngine === 'sing-box' ? 'cbx-badge-purple font-bold' : 'cbx-text-sub hover:cbx-text-main']">

                      <span>sing-box (SRS)</span>

                    </button>

                  </div>

                </div>

                <!-- Search Input -->

                <div class="relative w-full">

                  <input v-model="app.geodataSearchQuery" placeholder="搜索规则名称、策略组、文件名（如 openai, apple, telegram, srs, mrs）..." class="w-full rounded-xl px-3.5 py-2 pr-9 text-xs" />

                  <button v-if="app.geodataSearchQuery" @click="app.geodataSearchQuery = ''" aria-label="清除搜索" class="absolute right-2.5 top-1/2 -translate-y-1/2 cbx-text-muted hover:cbx-text-main cursor-pointer text-xs px-1">✕</button>

                </div>

              </div>

              <!-- Rule Files Table (Scrollable) -->

              <div class="flex-1 overflow-auto border rounded-2xl cbx-sub-card max-h-[380px]" tabindex="0" role="region" aria-label="Geo 规则文件列表">

                <table class="w-full text-left text-xs min-w-[600px]">

                  <thead class="cbx-modal-header cbx-text-sub font-mono sticky top-0 z-10">

                    <tr>

                      <th class="py-2.5 px-3">规则名称与用途</th>

                      <th class="py-2.5 px-2.5 text-center">适配引擎</th>

                      <th class="py-2.5 px-2.5">对应策略组</th>

                      <th class="py-2.5 px-2.5">规则文件</th>

                      <th class="py-2.5 px-2.5 text-right">文件体积</th>

                      <th class="py-2.5 px-3 text-center">运行状态</th>

                    </tr>

                  </thead>

                  <tbody class="divide-y" style="border-color: var(--cbx-border-sub);">

                    <tr v-for="(f, idx) in app.filteredGeodataFiles" :key="idx" class="hover:bg-sky-500/5 transition">

                      <td class="py-2.5 px-3 max-w-[260px]">

                        <div class="font-bold cbx-text-main">{{ f.name }}</div>

                        <div class="text-[11px] cbx-text-muted truncate">{{ f.description }}</div>

                      </td>

                      <td class="py-2.5 px-2.5 text-center">

                        <span class="text-[10px] font-mono px-2 py-0.5 rounded font-bold" :class="{

                          'cbx-badge-info': f.engine === 'Mihomo',

                          'cbx-badge-purple': f.engine === 'sing-box',

                          'cbx-badge-ok': f.engine === '双核通用' || !f.engine

                        }">

                          {{ f.engine || '双核通用' }}

                        </span>

                      </td>

                      <td class="py-2.5 px-2.5">

                        <span class="text-[10px] px-2 py-0.5 rounded cbx-btn-ghost font-mono">

                          {{ f.group || '通用分流' }}

                        </span>

                      </td>

                      <td class="py-2.5 px-2.5 font-mono text-[11px] font-semibold" style="color: var(--cbx-ok-fg);">

                        <code>{{ f.filename }}</code>

                      </td>

                      <td class="py-2.5 px-2.5 font-mono text-right cbx-text-sub font-semibold font-tabular">{{ f.size_str || '-' }}</td>

                      <td class="py-2.5 px-3 text-center">

                        <span class="text-[10px] px-2 py-0.5 rounded-full cbx-badge-ok font-mono font-bold">

                          ✓ 活跃生效

                        </span>

                      </td>

                    </tr>

                    <tr v-if="app.filteredGeodataFiles.length === 0">

                      <td colspan="6" class="py-8 text-center cbx-text-muted">未找到匹配的规则文件</td>

                    </tr>

                  </tbody>

                </table>

              </div>

              <!-- Modal Footer -->

              <div class="flex flex-wrap items-center justify-between gap-2 pt-2 border-t" style="border-color: var(--cbx-border-sub);">

                <div class="text-[11px] cbx-text-muted">

                  规则存储于 <code>/app/etc/rules</code> · 格式支持 Mihomo MRS 与 sing-box SRS

                </div>

                <div class="flex items-center gap-2 ml-auto">

                  <button @click="app.updateGeoData" :disabled="app.updatingGeo" class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs transition flex items-center gap-1.5 shadow-sm cursor-pointer disabled:opacity-50">

                    <span v-if="app.updatingGeo" class="inline-block animate-spin mr-1">↻</span>

                    <span>{{ app.updatingGeo ? '正在更新中...' : '🔄 立即一键更新' }}</span>

                  </button>

                  <button @click="app.showGeodataDetailModal = false" class="cbx-btn-ghost px-4 py-2 rounded-xl text-xs font-semibold cursor-pointer">

                    关闭

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
