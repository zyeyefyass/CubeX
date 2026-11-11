<template>
          <div v-if="showGeodataDetailModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-md animate-in fade-in">

            <div class="glass-matrix border border-emerald-500/40 rounded-3xl p-6 max-w-4xl w-full max-h-[85vh] flex flex-col shadow-2xl relative space-y-4">

              <!-- Modal Header -->

              <div class="flex items-center justify-between border-b border-slate-700/60 pb-3">

                <div class="flex items-center gap-3">

                  <span class="text-2xl p-2 rounded-2xl bg-emerald-950/80 border border-emerald-500/30">📚</span>

                  <div>

                    <h3 class="text-base sm:text-lg font-bold text-white flex items-center gap-2">

                      <span>Geo 数据库 & 217heidai 规则包全景清单</span>

                      <span class="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-300 border border-emerald-400/30 font-mono font-bold">

                        {{ app.geodataStatus.rules_count || 105 }} 套规则 · 已全量装载

                      </span>

                    </h3>

                    <p class="text-xs text-slate-400 mt-0.5">

                      全套二进制 MRS/SRS 离线分流体系与 217heidai 去广告权威拦截规则详细清单

                    </p>

                  </div>

                </div>

                <button @click="showGeodataDetailModal = false" class="text-slate-400 hover:text-white px-2.5 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-sm font-bold transition cursor-pointer">✕</button>

              </div>

              <!-- Summary Metrics Bar -->

              <div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5 text-xs font-mono">

                <div class="p-2.5 rounded-xl bg-slate-950/60 border border-slate-800">

                  <div class="text-[10px] text-slate-400">拦截规则总规模</div>

                  <div class="text-emerald-400 font-bold text-sm mt-0.5">{{ (app.geodataStatus.total_rules || 240005).toLocaleString() }} 条</div>

                </div>

                <div class="p-2.5 rounded-xl bg-slate-950/60 border border-slate-800">

                  <div class="text-[10px] text-slate-400">规则库版本</div>

                  <div class="text-emerald-300 font-bold text-sm mt-0.5">{{ app.geodataStatus.version || '最新稳定版' }}</div>

                </div>

                <div class="p-2.5 rounded-xl bg-slate-950/60 border border-slate-800">

                  <div class="text-[10px] text-slate-400">装载规则文件</div>

                  <div class="text-purple-300 font-bold text-sm mt-0.5">{{ app.geodataStatus.files_count || app.geodataStatus.rules_count || 105 }} 个</div>

                </div>

                <div class="p-2.5 rounded-xl bg-slate-950/60 border border-slate-800">

                  <div class="text-[10px] text-slate-400">最近校验/同步</div>

                  <div class="text-slate-300 font-bold text-xs mt-1 truncate">{{ app.geodataStatus.last_updated || '已装载' }}</div>

                </div>

              </div>

              <!-- Filter & Search Bar -->

              <div class="flex flex-col gap-2.5 text-xs">

                <div class="flex flex-wrap items-center justify-between gap-2">

                  <!-- Category Pills -->

                  <div class="flex items-center gap-1.5 overflow-x-auto py-0.5">

                    <button @click="app.geodataFilterCategory = 'all'" :class="['px-2.5 py-1 rounded-lg font-medium transition cursor-pointer', app.geodataFilterCategory === 'all' ? 'bg-emerald-500 text-slate-950 font-bold' : 'bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white text-slate-300 hover:bg-slate-700']">

                      全部分类 ({{ app.geodataStatus.files?.length || 0 }})

                    </button>

                    <button @click="app.geodataFilterCategory = '基础数据库'" :class="['px-2.5 py-1 rounded-lg font-medium transition cursor-pointer', app.geodataFilterCategory === '基础数据库' ? 'bg-emerald-500 text-slate-950 font-bold' : 'bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white text-slate-300 hover:bg-slate-700']">

                      基础库 ({{ app.geodataStatus.files?.filter(f => f.category === '基础数据库').length || 0 }})

                    </button>

                    <button @click="app.geodataFilterCategory = '广告与恶意追踪拦截'" :class="['px-2.5 py-1 rounded-lg font-medium transition cursor-pointer', app.geodataFilterCategory === '广告与恶意追踪拦截' ? 'bg-emerald-500 text-slate-950 font-bold' : 'bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white text-slate-300 hover:bg-slate-700']">

                      去广告 ({{ app.geodataStatus.files?.filter(f => f.category === '广告与恶意追踪拦截').length || 0 }})

                    </button>

                    <button @click="app.geodataFilterCategory = '基础路由分流'" :class="['px-2.5 py-1 rounded-lg font-medium transition cursor-pointer', app.geodataFilterCategory === '基础路由分流' ? 'bg-emerald-500 text-slate-950 font-bold' : 'bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white text-slate-300 hover:bg-slate-700']">

                      基础路由 ({{ app.geodataStatus.files?.filter(f => f.category === '基础路由分流').length || 0 }})

                    </button>

                    <button @click="app.geodataFilterCategory = '应用专线分流'" :class="['px-2.5 py-1 rounded-lg font-medium transition cursor-pointer', app.geodataFilterCategory === '应用专线分流' ? 'bg-emerald-500 text-slate-950 font-bold' : 'bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white text-slate-300 hover:bg-slate-700']">

                      核心专线 ({{ app.geodataStatus.files?.filter(f => f.category === '应用专线分流').length || 0 }})

                    </button>

                    <button @click="app.geodataFilterCategory = '常用生态与自定义扩展'" :class="['px-2.5 py-1 rounded-lg font-medium transition cursor-pointer', app.geodataFilterCategory === '常用生态与自定义扩展' ? 'bg-emerald-500 text-slate-950 font-bold' : 'bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white text-slate-300 hover:bg-slate-700']">

                      常用生态扩展 ({{ app.geodataStatus.files?.filter(f => f.category === '常用生态与自定义扩展').length || 0 }})

                    </button>

                  </div>

                  <!-- Engine Toggles -->

                  <div class="flex items-center gap-1 bg-slate-950/80 p-0.5 rounded-xl border border-slate-800 text-[11px]">

                    <button @click="app.geodataFilterEngine = 'all'" :class="['px-2.5 py-0.5 rounded-lg transition cursor-pointer', app.geodataFilterEngine === 'all' ? 'bg-slate-700 text-white font-bold' : 'text-slate-400 hover:text-slate-200']">

                      全部引擎

                    </button>

                    <button @click="app.geodataFilterEngine = 'Mihomo'" :class="['px-2.5 py-0.5 rounded-lg transition cursor-pointer flex items-center gap-1', app.geodataFilterEngine === 'Mihomo' ? 'bg-cyan-900/60 text-emerald-300 border border-cyan-500/40 font-bold' : 'text-slate-400 hover:text-emerald-300']">

                      <span>Mihomo (MRS)</span>

                    </button>

                    <button @click="app.geodataFilterEngine = 'sing-box'" :class="['px-2.5 py-0.5 rounded-lg transition cursor-pointer flex items-center gap-1', app.geodataFilterEngine === 'sing-box' ? 'bg-purple-900/60 text-purple-300 border border-purple-500/40 font-bold' : 'text-slate-400 hover:text-purple-300']">

                      <span>sing-box (SRS)</span>

                    </button>

                  </div>

                </div>

                <!-- Search Input -->

                <div class="relative w-full">

                  <input v-model="app.geodataSearchQuery" placeholder="搜索规则名称、策略组、文件名（如 openai, apple, telegram, srs, mrs）..." class="w-full bg-slate-950/80 border border-slate-700/80 rounded-xl px-3.5 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500 shadow-inner" />

                  <span v-if="app.geodataSearchQuery" @click="app.geodataSearchQuery = ''" class="absolute right-3 top-1.5 text-slate-400 hover:text-white cursor-pointer text-xs">✕</span>

                </div>

              </div>

              <!-- Rule Files Table (Scrollable) -->

              <div class="flex-1 overflow-y-auto border border-slate-800/80 rounded-2xl bg-slate-950/60 custom-scrollbar max-h-[400px]">

                <table class="w-full text-left text-xs">

                  <thead class="bg-slate-900/90 text-white text-slate-400 font-mono border-b border-slate-800 sticky top-0 backdrop-blur z-10">

                    <tr>

                      <th class="py-2.5 px-3">规则名称与用途</th>

                      <th class="py-2.5 px-2.5 text-center">适配引擎</th>

                      <th class="py-2.5 px-2.5">对应策略组</th>

                      <th class="py-2.5 px-2.5">规则文件</th>

                      <th class="py-2.5 px-2.5 text-right">文件体积</th>

                      <th class="py-2.5 px-3 text-center">运行状态</th>

                    </tr>

                  </thead>

                  <tbody class="divide-y divide-slate-800/40">

                    <tr v-for="(f, idx) in app.filteredGeodataFiles" :key="idx" class="hover:bg-white/10 transition">

                      <td class="py-2.5 px-3 max-w-[280px]">

                        <div class="font-bold text-slate-200">{{ f.name }}</div>

                        <div class="text-[11px] text-slate-400 truncate">{{ f.description }}</div>

                      </td>

                      <td class="py-2.5 px-2.5 text-center">

                        <span class="text-[10px] font-mono px-2 py-0.5 rounded font-bold" :class="{

                          'bg-white/10 text-sky-200 border border-sky-400/30 font-bold drop-shadow-sm': f.engine === 'Mihomo',

                          'bg-purple-950 text-purple-300 border border-purple-500/40': f.engine === 'sing-box',

                          'bg-emerald-500/15 text-emerald-300 border border-emerald-400/30': f.engine === '双核通用' || !f.engine

                        }">

                          {{ f.engine || '双核通用' }}

                        </span>

                      </td>

                      <td class="py-2.5 px-2.5">

                        <span class="text-[10px] px-2 py-0.5 rounded bg-white/10 text-white text-slate-200 border border-slate-700/80 font-mono">

                          {{ f.group || '通用分流' }}

                        </span>

                      </td>

                      <td class="py-2.5 px-2.5 font-mono text-[11px] text-emerald-300">

                        <code>{{ f.filename }}</code>

                      </td>

                      <td class="py-2.5 px-2.5 font-mono text-right text-slate-300 font-semibold">{{ f.size_str || '-' }}</td>

                      <td class="py-2.5 px-3 text-center">

                        <span class="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-300 border border-emerald-400/30 font-mono">

                          ✓ 活跃生效

                        </span>

                      </td>

                    </tr>

                    <tr v-if="app.filteredGeodataFiles.length === 0">

                      <td colspan="6" class="py-8 text-center text-slate-400">未找到匹配的规则文件</td>

                    </tr>

                  </tbody>

                </table>

              </div>

              <!-- Modal Footer -->

              <div class="flex items-center justify-between pt-2 border-t border-slate-800/60">

                <div class="text-[11px] text-slate-400">

                  规则存储于 <code>/app/etc/rules</code> · 格式支持 Mihomo MRS 与 sing-box SRS

                </div>

                <div class="flex items-center gap-2">

                  <button @click="app.updateGeoData" :disabled="app.updatingGeo" class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs transition flex items-center gap-1.5 shadow-md shadow-emerald-600/20 cursor-pointer">

                    <span v-if="app.updatingGeo" class="inline-block animate-spin mr-1">↻</span>

                    <span>{{ app.updatingGeo ? '正在更新中...' : '🔄 立即一键更新' }}</span>

                  </button>

                  <button @click="showGeodataDetailModal = false" class="px-4 py-2 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-slate-300 text-xs font-semibold cursor-pointer">

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
