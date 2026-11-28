<template>
          <div v-if="app.executionModal.visible" class="fixed inset-0 z-[10000] flex items-center justify-center p-3 sm:p-4 bg-black/60 animate-in fade-in" @click.self="app.executionModal.visible = false">

            <div class="w-full max-w-2xl rounded-3xl cbx-modal-shell p-5 sm:p-7 space-y-5 max-h-[88dvh] flex flex-col">

              <div class="flex items-center justify-between border-b pb-4 shrink-0 gap-3" style="border-color: var(--cbx-border-sub);">

                <div class="flex items-center gap-3 min-w-0">

                  <span class="text-2xl shrink-0">{{ app.executionModal.status === 'success' ? '🛡️' : '⚠️' }}</span>

                  <div class="min-w-0">

                    <h3 class="text-base sm:text-lg font-bold cbx-text-main tracking-wide flex items-center gap-2 flex-wrap">

                      <span>{{ app.executionModal.title }}</span>

                      <span :class="[

                        'text-[10px] px-2 py-0.5 rounded-full font-mono font-bold',

                        app.executionModal.status === 'success' ? 'cbx-badge-ok' : 'cbx-badge-warn'

                      ]">

                        {{ app.executionModal.status === 'success' ? '已闭环生效' : '执行提示' }}

                      </span>

                    </h3>

                    <p class="text-xs cbx-text-muted mt-0.5 font-mono">执行时间: {{ app.executionModal.timestamp }}</p>

                  </div>

                </div>

                <button @click="app.executionModal.visible = false" aria-label="关闭弹窗" class="cbx-btn-ghost min-w-[36px] min-h-[36px] p-1.5 rounded-xl transition cursor-pointer shrink-0 flex items-center justify-center cbx-touch-extend">✕</button>

              </div>

              <div class="text-xs cbx-badge-info rounded-xl p-3 leading-relaxed shrink-0 font-medium">

                {{ app.executionModal.message }}

              </div>

              <!-- Step Tracker Pipeline -->

              <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 py-2 shrink-0 border-b" style="border-color: var(--cbx-border-sub);">

                <div v-for="(step, sidx) in app.healingSteps" :key="sidx"

                  class="flex items-center gap-2 p-2 rounded-xl cbx-sub-card text-[11px] font-mono">

                  <span class="w-4 h-4 rounded-full cbx-badge-ok flex items-center justify-center text-[10px] shrink-0 font-bold">✓</span>

                  <span class="truncate cbx-text-sub" :title="step">{{ step }}</span>

                </div>

              </div>

              <!-- Itemized Actions List -->

              <div class="flex-1 overflow-y-auto space-y-2 pr-1 custom-scrollbar">

                <div class="text-xs font-bold cbx-text-muted tracking-wider uppercase mb-1">

                  已执行加固与配置项 ({{ app.executionModal.actions.length }})

                </div>

                <div v-for="(act, idx) in app.executionModal.actions" :key="idx"

                  class="flex items-start gap-2.5 p-2.5 rounded-xl cbx-sub-card text-xs">

                  <span class="shrink-0 mt-0.5 font-bold" style="color: var(--cbx-ok-fg);">✓</span>

                  <span class="cbx-text-main leading-normal font-mono break-all">{{ act }}</span>

                </div>

                <div v-if="app.executionModal.actions.length === 0" class="text-xs cbx-text-muted text-center py-4">

                  （无附加配置明细）

                </div>

                <div v-if="app.executionModal.errors && app.executionModal.errors.length > 0" class="space-y-1.5 pt-2">

                  <div class="text-xs font-bold tracking-wider uppercase" style="color: var(--cbx-danger-fg);">异常/警告项</div>

                  <div v-for="(err, eidx) in app.executionModal.errors" :key="'err-' + eidx"

                    class="flex items-start gap-2 p-2.5 rounded-xl cbx-badge-danger text-xs">

                    <span class="shrink-0">⚠️</span>

                    <span class="font-mono break-all">{{ err }}</span>

                  </div>

                </div>

              </div>

              <div class="pt-3 border-t flex justify-end shrink-0" style="border-color: var(--cbx-border-sub);">

                <button @click="app.executionModal.visible = false"

                  class="px-5 py-2 rounded-xl bg-sky-600 hover:bg-sky-700 text-white text-xs font-bold shadow-sm transition cursor-pointer">

                  知道了

                </button>

              </div>

            </div>

          </div>

          <!-- Modal: 混沌容灾演练深度报告 -->

          <div v-if="app.chaosReportModal" class="fixed inset-0 z-[10000] flex items-center justify-center p-3 sm:p-4 bg-black/60 animate-in fade-in" @click.self="app.chaosReportModal = false">

            <div class="w-full max-w-2xl rounded-3xl cbx-modal-shell p-5 sm:p-7 space-y-5 max-h-[88dvh] flex flex-col">

              <div class="flex items-center justify-between border-b pb-4 shrink-0 gap-3" style="border-color: var(--cbx-border-sub);">

                <div class="flex items-center gap-3 min-w-0">

                  <span class="text-2xl shrink-0">🛡️</span>

                  <div class="min-w-0">

                    <h3 class="text-base sm:text-lg font-bold cbx-text-main tracking-wide flex items-center gap-2 flex-wrap">

                      <span>全链路混沌容灾演练实测报告</span>

                      <span class="text-[10px] px-2 py-0.5 rounded-full font-mono font-bold cbx-badge-ok">

                        评分 100/100

                      </span>

                    </h3>

                    <p class="text-xs cbx-text-muted mt-0.5 font-mono">演练完成时间: {{ app.chaosDrillReport?.timestamp || new Date().toLocaleString() }}</p>

                  </div>

                </div>

                <button @click="app.chaosReportModal = false" aria-label="关闭弹窗" class="cbx-btn-ghost min-w-[36px] min-h-[36px] rounded-xl flex items-center justify-center transition cursor-pointer shrink-0 cbx-touch-extend">

                  ✕

                </button>

              </div>

              <!-- 演练场景结果列表 -->

              <div class="space-y-3 overflow-y-auto pr-1 flex-1 custom-scrollbar">

                <div v-for="(sc, idx) in (app.chaosDrillReport?.stages || [])" :key="idx"

                  class="p-4 rounded-2xl cbx-sub-card space-y-2">

                  <div class="flex items-center justify-between gap-2">

                    <span class="text-xs sm:text-sm font-bold cbx-text-main flex items-center gap-2 min-w-0">

                      <span class="w-5 h-5 rounded-full cbx-badge-ok text-xs flex items-center justify-center font-mono shrink-0">✓</span>

                      <span class="truncate">{{ sc.name }}</span>

                    </span>

                    <span class="text-xs font-mono px-2 py-0.5 rounded cbx-badge-ok font-bold shrink-0">

                      {{ sc.latency || '0ms' }} · PASS

                    </span>

                  </div>

                  <p class="text-xs cbx-text-sub leading-relaxed pl-7">{{ sc.detail }}</p>

                </div>

              </div>

              <div class="flex justify-end pt-2 border-t shrink-0" style="border-color: var(--cbx-border-sub);">

                <button @click="app.chaosReportModal = false"

                  class="px-5 py-2 rounded-xl bg-sky-600 hover:bg-sky-700 text-white font-bold text-xs shadow-sm transition cursor-pointer">

                  完成并确认

                </button>

              </div>

            </div>

          </div>

          <!-- Modal: 全链路 1 秒网络诊断体检 -->

          <div v-if="app.diagnosticModal.visible" class="fixed inset-0 z-[10000] flex items-center justify-center p-3 sm:p-4 bg-black/70 animate-in fade-in" @click.self="app.diagnosticModal.visible = false">

            <div class="w-full max-w-2xl rounded-3xl cbx-modal-shell p-5 sm:p-7 space-y-5 max-h-[88dvh] flex flex-col">

              <div class="flex items-center justify-between border-b pb-4 shrink-0 gap-3" style="border-color: var(--cbx-border-sub);">

                <div class="flex items-center gap-3 min-w-0">

                  <span class="text-2xl sm:text-3xl p-2 rounded-2xl cbx-badge-info shrink-0" :class="{'animate-spin': app.diagnosticModal.running}">🛡️</span>

                  <div class="min-w-0">

                    <h3 class="text-base sm:text-lg font-bold cbx-text-main tracking-wide flex items-center gap-2 flex-wrap">

                      <span>全链路网络极速体检自省</span>

                      <span v-if="app.diagnosticModal.running" class="text-[10px] px-2.5 py-0.5 rounded-full font-mono font-bold cbx-badge-info animate-pulse">

                        正在毫秒穿透探测...

                      </span>

                      <span v-else-if="app.diagnosticModal.report" :class="[

                        'text-[10px] px-2.5 py-0.5 rounded-full font-mono font-bold',

                        app.diagnosticModal.report.score >= 90 ? 'cbx-badge-ok' : 'cbx-badge-warn'

                      ]">

                        健康评分 {{ app.diagnosticModal.report.score }}/100

                      </span>

                    </h3>

                    <p class="text-xs cbx-text-muted mt-0.5">物理链路 ➔ 策略路由 ➔ WAN 拨号 ➔ 双核中枢 ➔ 节点链路 ➔ DNS 干净度 六重透视</p>

                  </div>

                </div>

                <button @click="app.diagnosticModal.visible = false" aria-label="关闭弹窗" class="cbx-btn-ghost min-w-[36px] min-h-[36px] p-2 rounded-xl transition cursor-pointer shrink-0 flex items-center justify-center cbx-touch-extend">✕</button>

              </div>

              <!-- Summary Card -->

              <div v-if="app.diagnosticModal.report" class="p-3.5 rounded-2xl cbx-sub-card text-xs shrink-0">

                <div class="flex items-center justify-between font-bold cbx-text-main mb-1 gap-2">

                  <span class="flex items-center gap-1.5">

                    <span>💡</span>

                    <span>网络健康综合诊断：</span>

                  </span>

                  <span class="text-[10px] font-mono font-bold" style="color: var(--cbx-accent-fg);">体检耗时 {{ app.diagnosticModal.report.elapsed_ms }}ms</span>

                </div>

                <p class="cbx-text-sub leading-relaxed">{{ app.diagnosticModal.report.summary }}</p>

              </div>

              <!-- 6-Stage Checks List -->

              <div v-if="app.diagnosticModal.report && app.diagnosticModal.report.checks" class="space-y-2 overflow-y-auto max-h-[42vh] pr-1">

                <div v-for="chk in app.diagnosticModal.report.checks" :key="chk.id"

                  class="p-3 rounded-xl cbx-sub-card flex items-center justify-between gap-3 transition"

                  :style="{ borderColor: chk.status === 'pass' ? 'var(--cbx-accent-border)' : 'var(--cbx-warn-border)' }">

                  <div class="min-w-0 flex-1">

                    <div class="flex items-center gap-2 flex-wrap">

                      <span :style="{ color: chk.status === 'pass' ? 'var(--cbx-ok-fg)' : 'var(--cbx-warn-fg)' }" class="text-xs font-bold">

                        {{ chk.status === 'pass' ? '✓' : '⚠️' }}

                      </span>

                      <span class="font-bold text-xs cbx-text-main">{{ chk.name }}</span>

                      <span class="text-[10px] px-1.5 py-0.5 rounded cbx-btn-ghost cbx-text-muted font-mono">{{ chk.category }}</span>

                    </div>

                    <div class="text-[11px] cbx-text-muted mt-0.5 truncate">{{ chk.detail }}</div>

                  </div>

                  <div class="text-right shrink-0">

                    <span class="text-[11px] font-mono font-bold font-tabular" :style="{ color: chk.status === 'pass' ? 'var(--cbx-accent-fg)' : 'var(--cbx-warn-fg)' }">

                      {{ chk.latency_ms >= 999 ? '超时' : `${chk.latency_ms}ms` }}

                    </span>

                  </div>

                </div>

              </div>

              <!-- Action Bar -->

              <div class="pt-3 border-t flex flex-wrap items-center justify-between gap-2 shrink-0" style="border-color: var(--cbx-border-sub);">

                <button v-if="app.diagnosticModal.report && app.diagnosticModal.report.can_auto_heal" @click="app.triggerFastSelfHeal"

                  class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold shadow-sm transition flex items-center gap-1.5 cursor-pointer">

                  <span>⚡</span>

                  <span>一键极速自愈修复</span>

                </button>

                <div v-else class="text-[11px] cbx-text-muted font-mono">

                  ● 各链路运行在巅峰状态，无需人工干预

                </div>

                <button @click="app.diagnosticModal.visible = false"

                  class="cbx-btn-ghost px-5 py-2 rounded-xl text-xs font-bold transition cursor-pointer ml-auto">

                  完成

                </button>

              </div>

            </div>

          </div>

    <!-- Modal: Add / Edit DNS Rewrite Rule -->

    <div v-if="app.showDnsRewriteModal" class="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-4 bg-black/60 animate-in fade-in overflow-y-auto" @click.self="app.showDnsRewriteModal = false">

      <div class="w-full max-w-md max-h-[92dvh] overflow-y-auto rounded-2xl sm:rounded-3xl cbx-modal-shell p-4 sm:p-6 space-y-4 relative custom-scrollbar">

        <div class="flex items-center justify-between border-b pb-3" style="border-color: var(--cbx-border-sub);">

          <h3 class="text-base font-bold cbx-text-main flex items-center gap-2">

            <span style="color: var(--cbx-ok-fg);">🔀</span>

            <span>{{ app.editingDnsRewriteRule.id ? '编辑 DNS 重写规则' : '新增 DNS 重写规则' }}</span>

          </h3>

          <button @click="app.showDnsRewriteModal = false" aria-label="关闭弹窗" class="cbx-btn-ghost min-w-[36px] min-h-[36px] rounded-xl text-base cursor-pointer flex items-center justify-center cbx-touch-extend">&times;</button>

        </div>

        <div class="space-y-3 text-xs">

          <div>

            <label class="block cbx-text-sub font-bold mb-1">原始域名 (Domain)</label>

            <input type="text" v-model="app.editingDnsRewriteRule.domain" placeholder="例: services.googleapis.cn 或 nas.lan" class="w-full rounded-xl px-3 py-2 font-mono">

          </div>

          <div>

            <label class="block cbx-text-sub font-bold mb-1">重写类型 (Rewrite Type)</label>

            <select v-model="app.editingDnsRewriteRule.type" class="w-full rounded-xl px-3 py-2">

              <option value="CNAME">CNAME 域名别名重定向 (如 services.googleapis.com)</option>

              <option value="A">A 记录静态 IP 映射 (指定内网 IP)</option>

            </select>

          </div>

          <div>

            <label class="block cbx-text-sub font-bold mb-1">重写目标 (Target Domain / IP)</label>

            <input type="text" v-model="app.editingDnsRewriteRule.target" placeholder="例: services.googleapis.com 或局域网静态 IP" class="w-full rounded-xl px-3 py-2 font-mono">

          </div>

          <div>

            <label class="block cbx-text-sub font-bold mb-1">说明备注 (Comment)</label>

            <input type="text" v-model="app.editingDnsRewriteRule.comment" placeholder="例: 解决 Google Play 无法下载 App 问题" class="w-full rounded-xl px-3 py-2">

          </div>

        </div>

        <div class="flex items-center justify-end gap-3 pt-3 border-t" style="border-color: var(--cbx-border-sub);">

          <button @click="app.showDnsRewriteModal = false" class="cbx-btn-ghost px-4 py-2 rounded-xl text-xs font-semibold cursor-pointer">取消</button>

          <button @click="app.saveDnsRewriteRule" class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold shadow-sm cursor-pointer transition">保存并立即生效</button>

        </div>

      </div>

    </div>

    <!-- Modal: Add / Edit Domain Filter Rule Source -->

    <div v-if="app.showRuleSourceModal" class="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-4 bg-black/60 animate-in fade-in overflow-y-auto" @click.self="app.showRuleSourceModal = false">

      <div class="w-full max-w-md max-h-[92dvh] overflow-y-auto rounded-2xl sm:rounded-3xl cbx-modal-shell p-4 sm:p-6 space-y-4 relative custom-scrollbar">

        <div class="flex items-center justify-between border-b pb-3" style="border-color: var(--cbx-border-sub);">

          <h3 class="text-base font-bold cbx-text-main flex items-center gap-2">

            <span>🛡️ {{ app.editingRuleSource.id ? '编辑规则源' : '添加域名过滤规则源' }}</span>

          </h3>

          <button @click="app.showRuleSourceModal = false" aria-label="关闭弹窗" class="cbx-btn-ghost min-w-[36px] min-h-[36px] rounded-xl text-base cursor-pointer flex items-center justify-center cbx-touch-extend">&times;</button>

        </div>

        <div class="space-y-3 text-xs">

          <div>

            <label class="block cbx-text-sub font-bold mb-1">规则源名称</label>

            <input type="text" v-model="app.editingRuleSource.name" placeholder="例如: 217heidai / Anti-AD" class="w-full rounded-xl px-3 py-2">

          </div>

          <div>

            <label class="block cbx-text-sub font-bold mb-1">规则订阅 URL</label>

            <input type="text" v-model="app.editingRuleSource.url" placeholder="https://.../adblockdns.txt 或 .mrs / .srs" class="w-full rounded-xl px-3 py-2 font-mono text-xs">

          </div>

          <div>

            <label class="block cbx-text-sub font-bold mb-1">规则语法格式</label>

            <select v-model="app.editingRuleSource.type" class="w-full rounded-xl px-3 py-2">

              <option value="adblock">Adblock 语法 / 文本规则 (adblockdns.txt)</option>

              <option value="mrs">Mihomo ruleset 原生二进制 (.mrs)</option>

              <option value="srs">sing-box 原生二进制 (.srs)</option>

              <option value="hosts">标准 Hosts 列表</option>

            </select>

          </div>

        </div>

        <div class="flex items-center justify-end gap-3 pt-3 border-t" style="border-color: var(--cbx-border-sub);">

          <button @click="app.showRuleSourceModal = false" class="cbx-btn-ghost px-4 py-2 rounded-xl text-xs font-semibold cursor-pointer">取消</button>

          <button @click="app.saveRuleSource" class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold shadow-sm cursor-pointer">保存规则源</button>

        </div>

      </div>

    </div>

    <!-- Modal: Rule Source Preview / Information -->

    <div v-if="app.showSourcePreviewModal && app.previewingSource" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/65 animate-in fade-in" @click.self="app.showSourcePreviewModal = false">

      <div class="w-full max-w-lg rounded-3xl cbx-modal-shell p-6 space-y-4 relative">

        <div class="flex items-center justify-between border-b pb-3" style="border-color: var(--cbx-border-sub);">

          <h3 class="text-base font-bold cbx-text-main flex items-center gap-2 truncate">

            <span>👁️ 规则源详情: {{ app.previewingSource.name }}</span>

          </h3>

          <button @click="app.showSourcePreviewModal = false" aria-label="关闭弹窗" class="cbx-btn-ghost min-w-[36px] min-h-[36px] rounded-xl text-base cursor-pointer flex items-center justify-center shrink-0 cbx-touch-extend">&times;</button>

        </div>

        <div class="space-y-3 text-xs">

          <div class="p-3 rounded-2xl cbx-sub-card font-mono space-y-1.5">

            <div class="cbx-text-muted">ID: <span class="cbx-text-main font-semibold">{{ app.previewingSource.id }}</span></div>

            <div class="cbx-text-muted">规则数: <span class="font-bold" style="color: var(--cbx-ok-fg);">{{ (app.previewingSource.rules_count || 0).toLocaleString() }} 条</span></div>

            <div class="cbx-text-muted">更新时间: <span class="cbx-text-main font-semibold">{{ app.previewingSource.updated_at || '刚刚' }}</span></div>

            <div class="cbx-text-muted truncate">订阅链接: <span class="font-semibold" style="color: var(--cbx-accent-fg);">{{ app.previewingSource.url }}</span></div>

            <div class="cbx-text-muted">状态: <span class="font-bold" :style="{ color: app.previewingSource.enabled ? 'var(--cbx-ok-fg)' : 'var(--cbx-text-muted)' }">{{ app.previewingSource.enabled ? '已启用 (生效中)' : '已停用' }}</span></div>

          </div>

          <p class="cbx-text-sub leading-relaxed text-[11px]">

            {{ app.previewingSource.description || '全维度拦截各类视频前贴片、流氓开屏广告、运营商劫持与第三方隐私追踪探针。' }}

          </p>

        </div>

        <div class="flex justify-end pt-2">

          <button @click="app.showSourcePreviewModal = false" class="cbx-btn-ghost px-4 py-2 rounded-xl text-xs font-semibold cursor-pointer">关闭</button>

        </div>

      </div>

    </div>
</template>

<script setup>
import { inject } from 'vue'
const app = inject('cubexApp')
</script>
