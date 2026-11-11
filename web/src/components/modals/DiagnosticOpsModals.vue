<template>
          <div v-if="app.executionModal.visible" class="fixed inset-0 z-[10000] flex items-center justify-center p-4 bg-black/80 backdrop-blur-md animate-in fade-in">

            <div class="w-full max-w-2xl rounded-3xl macos-glass-card shadow-2xl p-6 sm:p-7 space-y-5 text-slate-200 max-h-[85vh] flex flex-col">

              <div class="flex items-center justify-between border-b border-slate-700/60 pb-4 shrink-0">

                <div class="flex items-center gap-3">

                  <span class="text-2xl">{{ app.executionModal.status === 'success' ? '🛡️' : '⚠️' }}</span>

                  <div>

                    <h3 class="text-base sm:text-lg font-bold text-white tracking-wide flex items-center gap-2">

                      <span>{{ app.executionModal.title }}</span>

                      <span :class="[

                        'text-[10px] px-2 py-0.5 rounded-full font-mono font-semibold',

                        app.executionModal.status === 'success' ? 'bg-emerald-500/15 text-emerald-300 border border-emerald-400/30' : 'bg-amber-500/20 text-amber-300 border border-amber-400/40'

                      ]">

                        {{ app.executionModal.status === 'success' ? '已闭环生效' : '执行提示' }}

                      </span>

                    </h3>

                    <p class="text-xs text-slate-400 mt-0.5 font-mono">执行时间: {{ app.executionModal.timestamp }}</p>

                  </div>

                </div>

                <button @click="app.executionModal.visible = false" class="p-1.5 rounded-xl hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white text-slate-400 hover:text-white transition cursor-pointer">✕</button>

              </div>

              <div class="text-xs text-slate-300 bg-sky-950/40 border border-sky-900/50 rounded-xl p-3 leading-relaxed shrink-0">

                {{ app.executionModal.message }}

              </div>

              <!-- Step Tracker Pipeline (六维全栈自愈执行时序链) -->

              <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 py-2 shrink-0 border-b border-slate-800/80">

                <div v-for="(step, sidx) in app.healingSteps" :key="sidx"

                  class="flex items-center gap-2 p-2 rounded-xl bg-slate-950/80 border border-slate-800 text-[11px] font-mono">

                  <span class="w-4 h-4 rounded-full bg-emerald-500/20 text-emerald-400 border border-emerald-500/50 flex items-center justify-center text-[10px] shrink-0 font-bold">✓</span>

                  <span class="truncate text-slate-300" :title="step">{{ step }}</span>

                </div>

              </div>

              <!-- Itemized Actions List -->

              <div class="flex-1 overflow-y-auto space-y-2 pr-1 custom-scrollbar">

                <div class="text-xs font-bold text-slate-400 tracking-wider uppercase mb-1">

                  已执行加固与配置项 ({{ app.executionModal.actions.length }})

                </div>

                <div v-for="(act, idx) in app.executionModal.actions" :key="idx"

                  class="flex items-start gap-2.5 p-2.5 rounded-xl bg-slate-950/60 border border-slate-800/80 text-xs">

                  <span class="text-emerald-400 shrink-0 mt-0.5 font-bold">✓</span>

                  <span class="text-slate-200 leading-normal font-mono">{{ act }}</span>

                </div>

                <div v-if="app.executionModal.actions.length === 0" class="text-xs text-slate-400 text-center py-4">

                  （无附加配置明细）

                </div>

                <!-- Errors / Warnings if any -->

                <div v-if="app.executionModal.errors && app.executionModal.errors.length > 0" class="space-y-1.5 pt-2">

                  <div class="text-xs font-bold text-rose-400 tracking-wider uppercase">异常/警告项</div>

                  <div v-for="(err, eidx) in app.executionModal.errors" :key="'err-' + eidx"

                    class="flex items-start gap-2 p-2.5 rounded-xl bg-rose-950/30 border border-rose-500/40 text-rose-200 text-xs">

                    <span class="text-rose-400 shrink-0">⚠️</span>

                    <span class="font-mono">{{ err }}</span>

                  </div>

                </div>

              </div>

              <div class="pt-3 border-t border-slate-800 flex justify-end shrink-0">

                <button @click="app.executionModal.visible = false"

                  class="px-5 py-2 rounded-xl bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 text-white text-xs font-bold shadow transition cursor-pointer">

                  知道了

                </button>

              </div>

            </div>

          </div>

          <!-- Modal: 混沌容灾演练深度报告 (Chaos Drill Report Modal) -->

          <div v-if="app.chaosReportModal" class="fixed inset-0 z-[10000] flex items-center justify-center p-4 bg-black/80 backdrop-blur-md animate-in fade-in">

            <div class="w-full max-w-2xl rounded-3xl macos-glass-card shadow-2xl p-6 sm:p-7 space-y-5 text-slate-200 max-h-[85vh] flex flex-col">

              <div class="flex items-center justify-between border-b border-slate-700/60 pb-4 shrink-0">

                <div class="flex items-center gap-3">

                  <span class="text-2xl">🛡️</span>

                  <div>

                    <h3 class="text-base sm:text-lg font-bold text-white tracking-wide flex items-center gap-2">

                      <span>全链路混沌容灾演练实测报告</span>

                      <span class="text-[10px] px-2 py-0.5 rounded-full font-mono font-semibold bg-emerald-500/15 text-emerald-300 border border-emerald-400/30">

                        评分 100/100

                      </span>

                    </h3>

                    <p class="text-xs text-slate-400 mt-0.5 font-mono">演练完成时间: {{ app.chaosDrillReport?.timestamp || new Date().toLocaleString() }}</p>

                  </div>

                </div>

                <button @click="app.chaosReportModal = false" class="w-8 h-8 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-slate-300 hover:text-white transition cursor-pointer">

                  ✕

                </button>

              </div>

              <!-- 演练场景结果列表 -->

              <div class="space-y-3 overflow-y-auto pr-1 flex-1 custom-scrollbar">

                <div v-for="(sc, idx) in (app.chaosDrillReport?.stages || [])" :key="idx"

                  class="p-4 rounded-2xl bg-white/5 border border-white/10 space-y-2">

                  <div class="flex items-center justify-between">

                    <span class="text-xs sm:text-sm font-bold text-white flex items-center gap-2">

                      <span class="w-5 h-5 rounded-full bg-emerald-500/20 text-emerald-300 text-xs flex items-center justify-center font-mono">✓</span>

                      <span>{{ sc.name }}</span>

                    </span>

                    <span class="text-xs font-mono px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-bold">

                      {{ sc.latency || '0ms' }} · PASS

                    </span>

                  </div>

                  <p class="text-xs text-slate-300 leading-relaxed pl-7">{{ sc.detail }}</p>

                </div>

              </div>

              <div class="flex justify-end pt-2 border-t border-slate-700/60 shrink-0">

                <button @click="app.chaosReportModal = false"

                  class="px-5 py-2 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 text-white font-bold text-xs shadow-md transition cursor-pointer">

                  完成并确认

                </button>

              </div>

            </div>

          </div>

          <!-- Modal: 🛡️ 全链路 1 秒网络诊断体检与极简自愈全息弹窗 (Diagnostic Modal) -->

          <div v-if="app.diagnosticModal.visible" class="fixed inset-0 z-[10000] flex items-center justify-center p-4 bg-black/85 backdrop-blur-md animate-in fade-in">

            <div class="w-full max-w-2xl rounded-3xl macos-glass-card shadow-2xl p-6 sm:p-7 space-y-5 text-slate-200 max-h-[85vh] flex flex-col">

              <div class="flex items-center justify-between border-b border-slate-700/60 pb-4 shrink-0">

                <div class="flex items-center gap-3">

                  <span class="text-3xl p-2 rounded-2xl bg-cyan-950/60 border border-cyan-500/40" :class="{'animate-spin': app.diagnosticModal.running}">🛡️</span>

                  <div>

                    <h3 class="text-base sm:text-lg font-bold text-white tracking-wide flex items-center gap-2">

                      <span>全链路网络极速体检自省</span>

                      <span v-if="app.diagnosticModal.running" class="text-[10px] px-2.5 py-0.5 rounded-full font-mono font-semibold bg-sky-500/20 text-sky-300 border border-sky-400/30 animate-pulse">

                        正在毫秒穿透探测...

                      </span>

                      <span v-else-if="app.diagnosticModal.report" :class="[

                        'text-[10px] px-2.5 py-0.5 rounded-full font-mono font-semibold border',

                        app.diagnosticModal.report.score >= 90 ? 'bg-emerald-500/20 text-emerald-300 border-emerald-400/40' : 'bg-amber-500/20 text-amber-300 border-amber-400/40'

                      ]">

                        健康评分 {{ app.diagnosticModal.report.score }}/100

                      </span>

                    </h3>

                    <p class="text-xs text-slate-400 mt-0.5">物理链路 ➔ 策略路由 ➔ WAN 拨号 ➔ 双核中枢 ➔ 节点链路 ➔ DNS 干净度 六重透视</p>

                  </div>

                </div>

                <button @click="app.diagnosticModal.visible = false" class="p-2 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 text-slate-300 hover:text-white transition cursor-pointer">✕</button>

              </div>

              <!-- Summary Card -->

              <div v-if="app.diagnosticModal.report" class="p-3.5 rounded-2xl bg-gradient-to-r from-sky-950/50 to-[#00172e]/80 border border-cyan-500/30 text-xs shrink-0">

                <div class="flex items-center justify-between font-bold text-white mb-1">

                  <span class="flex items-center gap-1.5">

                    <span>💡</span>

                    <span>网络健康综合诊断：</span>

                  </span>

                  <span class="text-[10px] font-mono text-cyan-300">体检耗时 {{ app.diagnosticModal.report.elapsed_ms }}ms</span>

                </div>

                <p class="text-slate-300 leading-relaxed">{{ app.diagnosticModal.report.summary }}</p>

              </div>

              <!-- 6-Stage Checks List -->

              <div v-if="app.diagnosticModal.report && app.diagnosticModal.report.checks" class="space-y-2 overflow-y-auto max-h-[42vh] pr-1">

                <div v-for="chk in app.diagnosticModal.report.checks" :key="chk.id"

                  class="p-3 rounded-xl border flex items-center justify-between gap-3 transition"

                  :class="chk.status === 'pass' ? 'bg-sky-950/25 border-sky-500/30' : 'bg-amber-950/30 border-amber-500/40'">

                  <div class="min-w-0 flex-1">

                    <div class="flex items-center gap-2">

                      <span :class="chk.status === 'pass' ? 'text-emerald-400' : 'text-amber-400'" class="text-xs">

                        {{ chk.status === 'pass' ? '✓' : '⚠️' }}

                      </span>

                      <span class="font-bold text-xs text-white">{{ chk.name }}</span>

                      <span class="text-[11px] px-1.5 py-0.2 rounded bg-white/10 text-slate-400 font-mono">{{ chk.category }}</span>

                    </div>

                    <div class="text-[11px] text-slate-400 mt-0.5 truncate">{{ chk.detail }}</div>

                  </div>

                  <div class="text-right shrink-0">

                    <span class="text-[10px] font-mono font-bold" :class="chk.status === 'pass' ? 'text-cyan-300' : 'text-amber-300'">

                      {{ chk.latency_ms >= 999 ? '超时' : `${chk.latency_ms}ms` }}

                    </span>

                  </div>

                </div>

              </div>

              <!-- Action Bar -->

              <div class="pt-3 border-t border-slate-800 flex items-center justify-between shrink-0">

                <button v-if="app.diagnosticModal.report && app.diagnosticModal.report.can_auto_heal" @click="app.triggerFastSelfHeal"

                  class="px-4 py-2 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-500 hover:from-emerald-500 hover:to-teal-400 text-white text-xs font-bold shadow-md transition flex items-center gap-1.5 cursor-pointer">

                  <span>⚡</span>

                  <span>一键极速自愈修复</span>

                </button>

                <div v-else class="text-[11px] text-slate-400 font-mono">

                  ● 各链路运行在巅峰状态，无需人工干预

                </div>

                <button @click="app.diagnosticModal.visible = false"

                  class="px-5 py-2 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 text-white text-xs font-bold shadow transition cursor-pointer">

                  完成

                </button>

              </div>

            </div>

          </div>

    <div v-if="app.showDnsRewriteModal" class="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-4 bg-black/75 backdrop-blur-sm animate-in fade-in overflow-y-auto">

      <div class="w-full max-w-md max-h-[92dvh] overflow-y-auto rounded-2xl sm:rounded-3xl bg-[#001e36]/95 backdrop-blur-2xl text-white border border-cyan-500/30 p-4 sm:p-6 space-y-4 shadow-2xl relative custom-scrollbar">

        <div class="flex items-center justify-between border-b border-slate-800 pb-3">

          <h3 class="text-base font-bold text-white flex items-center gap-2">

            <span class="text-emerald-300">🔀</span>

            <span>{{ app.editingDnsRewriteRule.id ? '编辑 DNS 重写规则' : '新增 DNS 重写规则' }}</span>

          </h3>

          <button @click="app.showDnsRewriteModal = false" class="text-slate-400 hover:text-white text-lg cursor-pointer">&times;</button>

        </div>

        <div class="space-y-3 text-xs">

          <div>

            <label class="block text-slate-300 font-bold mb-1">原始域名 (Domain)</label>

            <input type="text" v-model="app.editingDnsRewriteRule.domain" placeholder="例: services.googleapis.cn 或 nas.lan" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-slate-200 placeholder-slate-500 focus:outline-none focus:border-emerald-500/60 font-mono">

          </div>

          <div>

            <label class="block text-slate-300 font-bold mb-1">重写类型 (Rewrite Type)</label>

            <select v-model="app.editingDnsRewriteRule.type" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-slate-200 focus:outline-none focus:border-emerald-500/60">

              <option value="CNAME">CNAME 域名别名重定向 (如 services.googleapis.com)</option>

              <option value="A">A 记录静态 IP 映射 (指定内网 IP)</option>

            </select>

          </div>

          <div>

            <label class="block text-slate-300 font-bold mb-1">重写目标 (Target Domain / IP)</label>

            <input type="text" v-model="app.editingDnsRewriteRule.target" placeholder="例: services.googleapis.com 或局域网静态 IP" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-slate-200 placeholder-slate-500 focus:outline-none focus:border-emerald-500/60 font-mono">

          </div>

          <div>

            <label class="block text-slate-300 font-bold mb-1">说明备注 (Comment)</label>

            <input type="text" v-model="app.editingDnsRewriteRule.comment" placeholder="例: 解决 Google Play 无法下载 App 问题" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-slate-200 placeholder-slate-500 focus:outline-none focus:border-emerald-500/60">

          </div>

        </div>

        <div class="flex items-center justify-end gap-3 pt-3 border-t border-slate-800">

          <button @click="app.showDnsRewriteModal = false" class="px-4 py-2 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-slate-300 text-xs font-medium cursor-pointer">取消</button>

          <button @click="app.saveDnsRewriteRule" class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold shadow-md shadow-emerald-950/40 cursor-pointer transition">保存并立即生效</button>

        </div>

      </div>

    </div>

    <!-- Modal: Add / Edit Domain Filter Rule Source -->

    <div v-if="app.showRuleSourceModal" class="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-4 bg-black/75 backdrop-blur-sm animate-in fade-in overflow-y-auto">

      <div class="w-full max-w-md max-h-[92dvh] overflow-y-auto rounded-2xl sm:rounded-3xl bg-[#001e36]/95 backdrop-blur-2xl text-white border border-cyan-500/30 p-4 sm:p-6 space-y-4 shadow-2xl relative custom-scrollbar">

        <div class="flex items-center justify-between border-b border-slate-800 pb-3">

          <h3 class="text-base font-bold text-white flex items-center gap-2">

            <span>🛡️ {{ app.editingRuleSource.id ? '编辑规则源' : '添加域名过滤规则源' }}</span>

          </h3>

          <button @click="app.showRuleSourceModal = false" class="text-slate-400 hover:text-white text-lg cursor-pointer">&times;</button>

        </div>

        <div class="space-y-3 text-xs">

          <div>

            <label class="block text-slate-300 font-bold mb-1">规则源名称</label>

            <input type="text" v-model="app.editingRuleSource.name" placeholder="例如: 217heidai / Anti-AD" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500">

          </div>

          <div>

            <label class="block text-slate-300 font-bold mb-1">规则订阅 URL</label>

            <input type="text" v-model="app.editingRuleSource.url" placeholder="https://.../adblockdns.txt 或 .mrs / .srs" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500 font-mono text-xs">

          </div>

          <div>

            <label class="block text-slate-300 font-bold mb-1">规则语法格式</label>

            <select v-model="app.editingRuleSource.type" class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-white focus:outline-none focus:border-emerald-500">

              <option value="adblock">Adblock 语法 / 文本规则 (adblockdns.txt)</option>

              <option value="mrs">Mihomo ruleset 原生二进制 (.mrs)</option>

              <option value="srs">sing-box 原生二进制 (.srs)</option>

              <option value="hosts">标准 Hosts 列表</option>

            </select>

          </div>

        </div>

        <div class="flex items-center justify-end gap-3 pt-3 border-t border-slate-800">

          <button @click="app.showRuleSourceModal = false" class="px-4 py-2 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-slate-300 text-xs font-medium cursor-pointer">取消</button>

          <button @click="app.saveRuleSource" class="px-4 py-2 rounded-xl bg-emerald-500 hover:bg-emerald-400 text-slate-950 text-xs font-bold shadow-lg shadow-emerald-500/20 cursor-pointer">保存规则源</button>

        </div>

      </div>

    </div>

    <!-- Modal: Rule Source Preview / Information -->

    <div v-if="app.showSourcePreviewModal && app.previewingSource" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-in fade-in">

      <div class="w-full max-w-lg rounded-3xl bg-[#001e36]/95 backdrop-blur-2xl text-white border border-cyan-500/30 p-6 space-y-4 shadow-2xl relative">

        <div class="flex items-center justify-between border-b border-slate-800 pb-3">

          <h3 class="text-base font-bold text-white flex items-center gap-2">

            <span>👁️ 规则源详情: {{ app.previewingSource.name }}</span>

          </h3>

          <button @click="app.showSourcePreviewModal = false" class="text-slate-400 hover:text-white text-lg cursor-pointer">&times;</button>

        </div>

        <div class="space-y-3 text-xs">

          <div class="p-3 rounded-2xl bg-slate-950 border border-slate-800 font-mono space-y-1.5">

            <div class="text-slate-400">ID: <span class="text-slate-200">{{ app.previewingSource.id }}</span></div>

            <div class="text-slate-400">规则数: <span class="text-emerald-400 font-bold">{{ (app.previewingSource.rules_count || 0).toLocaleString() }} 条</span></div>

            <div class="text-slate-400">更新时间: <span class="text-slate-200">{{ app.previewingSource.updated_at || '刚刚' }}</span></div>

            <div class="text-slate-400 truncate">订阅链接: <span class="text-emerald-300">{{ app.previewingSource.url }}</span></div>

            <div class="text-slate-400">状态: <span :class="app.previewingSource.enabled ? 'text-emerald-400 font-bold' : 'text-slate-300'">{{ app.previewingSource.enabled ? '已启用 (生效中)' : '已停用' }}</span></div>

          </div>

          <p class="text-slate-400 leading-relaxed text-[11px]">

            {{ app.previewingSource.description || '全维度拦截各类视频前贴片、流氓开屏广告、运营商劫持与第三方隐私追踪探针。' }}

          </p>

        </div>

        <div class="flex justify-end pt-2">

          <button @click="app.showSourcePreviewModal = false" class="px-4 py-2 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-slate-300 text-xs font-medium cursor-pointer">关闭</button>

        </div>

      </div>

    </div>
</template>

<script setup>
import { inject } from 'vue'
const app = inject('cubexApp')
</script>
