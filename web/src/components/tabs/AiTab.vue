<template>
        <div class="space-y-6 animate-in fade-in">

          <!-- 模块标题盾牌 -->

          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2">

            <div class="macos-title-shield space-y-1">

              <h2 class="text-base sm:text-lg font-bold text-white tracking-wide flex items-center gap-2">

                <span class="flex items-center gap-2"><ZootopiaIcon name="ai" :size="24" class="shrink-0" /><span>CubeX AI 伴生管家</span></span>

              </h2>

              <p class="text-xs text-[#E2F8EC] tracking-normal font-medium">端云融合 AI 智能体 · 零感网络健康体检 · 策略路由智能推演 · 双核容灾诊断顾问</p>

            </div>

            <!-- 右侧状态胶囊 -->

            <div class="flex items-center gap-2 shrink-0">

              <span class="h-7 px-3 inline-flex items-center gap-2 text-xs font-mono font-bold rounded-full bg-white/10 text-sky-200 border border-sky-400/30 shadow-sm">

                <span class="w-2 h-2 rounded-full bg-emerald-400 shadow-[0_0_8px_#10b981] animate-pulse"></span>

                <span>全天候伴生守护</span>

              </span>

              <button @click="app.showAISettingsModal = true"

                class="px-3.5 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/60 text-slate-200 hover:text-white font-bold text-xs flex items-center gap-1.5 transition active:scale-95 cursor-pointer"

                title="配置 AI 大模型 API Key 与提供商">

                <span>⚙️</span>

                <span>模型配置</span>

              </button>

            </div>

          </div>

          <!-- AI 对话式全息交互视窗 (Interactive Chat & Diagnostic Console) -->

          <div class="rounded-3xl macos-glass-card p-5 sm:p-7 space-y-5 border border-white/15 relative overflow-hidden">

            <div class="flex items-center justify-between border-b border-white/10 pb-4 flex-wrap gap-2">

              <div class="flex items-center gap-3">

                <div class="flex items-center justify-center shrink-0">

                  <ZootopiaIcon name="ai" :size="28" class="drop-shadow-[0_0_12px_rgba(0,245,212,0.6)]" />

                </div>

                <div>

                  <h3 class="text-sm sm:text-base font-black text-white flex items-center gap-2">

                    <span>CubeX Copilot 智能问答与诊断中枢</span>

                    <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-300 border border-emerald-400/30 font-bold">

                      端云自适应

                    </span>

                  </h3>

                  <p class="text-[11px] text-slate-400 mt-0.5">可直接提问任何关于软路由配置、RouterOS 策略、DNS 优化及双核网关的问题</p>

                </div>

              </div>

              <!-- 右侧快捷操作 -->

              <div class="flex items-center gap-2">

                <button v-if="app.aiReply" @click="app.copyText(app.aiReply, '已复制 AI 诊断报告')"

                  class="px-3 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 text-slate-200 hover:text-white text-xs font-bold transition flex items-center gap-1.5 cursor-pointer">

                  <span>📋</span>

                  <span>复制回答</span>

                </button>

                <button v-if="app.aiReply" @click="app.aiReply = ''; app.aiActions = []; app.aiActionResult = ''"

                  class="px-2.5 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 text-slate-400 hover:text-white text-xs transition cursor-pointer">

                  清空

                </button>

              </div>

            </div>

            <!-- AI 回复内容展示视窗 -->

            <div class="min-h-[220px] max-h-[500px] overflow-y-auto custom-scrollbar rounded-2xl bg-black/35 border border-white/10 p-4 sm:p-5 text-slate-200 text-xs sm:text-[13px] leading-relaxed relative">

              <div v-if="app.aiLoading" class="h-48 flex flex-col items-center justify-center space-y-3">

                <div class="w-10 h-10 border-3 border-sky-400/40 border-t-transparent rounded-full animate-spin"></div>

                <div class="text-xs font-mono text-slate-300 animate-pulse">CubeX AI 伴生管家正在全维推演与体检中...</div>

              </div>

              <div v-else-if="!app.aiReply" class="space-y-4 py-3">

                <div class="flex items-start gap-3">

                  <span class="text-2xl shrink-0 select-none leading-none drop-shadow-sm">🤖</span>

                  <div class="space-y-2">

                    <p class="font-bold text-white text-sm">你好！我是 CubeX 零感网络伴生 AI 管家。</p>

                    <p class="text-slate-300">我已深度接入当前软路由运行时底层：实时监控 RouterOS / 飞牛 / Linux 宿主底座、双核引擎健康度、策略路由表 table cubex 与 217heidai 防广告拦截中枢。</p>

                    <p class="text-slate-400 text-xs">点击上方快捷作业卡片，或直接在下方输入框提出您关心的网络问题！</p>

                  </div>

                </div>

                <!-- 推荐热点提问气泡 -->

                <div class="pt-2 border-t border-white/5 space-y-2">

                  <span class="text-[11px] font-bold text-slate-400">💡 推荐快捷提问：</span>

                  <div class="flex flex-wrap gap-2">

                    <button v-for="q in app.aiPresetQuestions" :key="q"

                      @click="app.triggerQuickAIPrompt(q)"

                      class="px-3 py-1.5 rounded-xl bg-white/5 hover:bg-white/15 border border-white/10 hover:border-sky-400/40 text-slate-300 hover:text-white text-xs transition-colors cursor-pointer text-left">

                      {{ q }}

                    </button>

                  </div>

                </div>

              </div>

              <!-- AI 输出结果 (支持 Markdown 排版与实操执行卡片) -->

              <div v-else class="space-y-4 whitespace-pre-wrap select-text font-sans">

                <div class="flex items-center justify-between pb-2 border-b border-white/10 text-[11px] font-mono">

                  <div class="flex items-center gap-2 text-cyan-300">

                    <span class="w-2 h-2 rounded-full bg-emerald-400 shadow-[0_0_8px_#10b981] animate-pulse"></span>

                    <span>AI 管家推演完成 · 具备真实自愈运维权限</span>

                  </div>

                  <span class="text-slate-400 hidden sm:inline">端云协同 · 本地安全沙盒</span>

                </div>

                <div class="leading-relaxed text-slate-100 text-xs sm:text-[13px]">{{ app.aiReply }}</div>

                <!-- 🛠️ AI 管家专属交互操作胶囊卡片流 (Interactive Action Pills) -->

                <div v-if="app.aiActions && app.aiActions.length > 0" class="pt-3 border-t border-white/10 space-y-2">

                  <div class="flex items-center justify-between">

                    <span class="text-xs font-bold text-sky-300 flex items-center gap-1.5">

                      <span>⚡ AI 推荐立即执行的操作 (点击直接修网):</span>

                    </span>

                    <span class="text-[10px] text-emerald-400 font-mono">闭环自愈就绪</span>

                  </div>

                  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">

                    <button v-for="act in app.aiActions" :key="act.id"

                      @click="app.executeAIAction(act)"

                      :disabled="app.executingAIActionId === act.id"

                      class="p-2.5 rounded-xl bg-white/5 hover:bg-white/15 border border-white/15 hover:border-amber-400/50 text-left transition flex items-start gap-2.5 cursor-pointer group active:scale-98 disabled:opacity-60 select-none shadow-sm">

                      <span class="text-xl shrink-0 group-hover:scale-110 transition-transform select-none leading-none mt-0.5">

                        {{ app.executingAIActionId === act.id ? '⏳' : act.icon }}

                      </span>

                      <div class="min-w-0 flex-1">

                        <div class="font-bold text-xs text-white group-hover:text-amber-200 flex items-center gap-1 truncate">

                          <span>{{ app.cleanActionLabelMap[act.label] || act.label }}</span>

                        </div>

                        <div class="text-[11px] text-slate-400 font-normal truncate mt-0.5">{{ act.desc }}</div>

                      </div>

                    </button>

                  </div>

                </div>

                <!-- 📜 AI 执行动作后的实时操作回执日志 -->

                <div v-if="app.aiActionResult" class="p-3.5 rounded-xl bg-emerald-950/40 border border-emerald-500/30 text-emerald-200 text-xs space-y-1.5 animate-in fade-in">

                  <div class="font-bold flex items-center gap-1.5 text-emerald-300">

                    <span>✅ AI 运维指令执行成功报告</span>

                  </div>

                  <div class="text-[11px] text-slate-300 whitespace-pre-line leading-relaxed font-mono">{{ app.aiActionResult }}</div>

                </div>

              </div>

            </div>

            <!-- 输入栏与控制栏 -->

            <div class="space-y-2">

              <form @submit.prevent="app.sendAIChat" class="flex items-center gap-2">

                <div class="relative flex-1">

                  <input v-model="app.aiPrompt"

                    :disabled="app.aiLoading"

                    type="text"

                    placeholder="问 AI 管家：例如“如何配置 RouterOS 策略路由？”或“帮我体检当前网络”..."

                    class="w-full px-4 py-3 rounded-2xl bg-black/40 border border-white/15 text-white placeholder-slate-500 text-xs sm:text-[13px] focus:outline-none focus:border-sky-400/60 focus:ring-1 focus:ring-sky-400/30 transition" />

                  <span v-if="app.aiPrompt" @click="app.aiPrompt = ''" class="absolute right-3.5 top-3 text-slate-400 hover:text-white cursor-pointer text-xs">✕</span>

                </div>

                <button type="submit"

                  :disabled="app.aiLoading || !app.aiPrompt.trim()"

                  class="px-5 py-3 rounded-2xl font-black text-xs sm:text-[13px] bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white shadow-lg shadow-sky-500/20 active:scale-95 disabled:opacity-40 disabled:cursor-not-allowed transition flex items-center gap-1.5 shrink-0 cursor-pointer">

                  <span v-if="app.aiLoading" class="inline-block animate-spin mr-1">↻</span>

                  <span v-else>🚀</span>

                  <span>发送提问</span>

                </button>

              </form>

            </div>

          </div>

        </div>
</template>

<script setup>
import { inject } from 'vue'
import ZootopiaIcon from '../ZootopiaIcon.vue'
const app = inject('cubexApp')
</script>
