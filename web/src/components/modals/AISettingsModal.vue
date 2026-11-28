<template>
    <div v-if="app.showAISettingsModal" class="fixed inset-0 z-50 bg-black/60 flex items-center justify-center p-3 sm:p-4 overflow-y-auto animate-in fade-in" @click.self="app.showAISettingsModal = false">

      <div class="cbx-modal-shell rounded-3xl p-5 sm:p-7 max-w-2xl w-full space-y-5 my-auto">

        <!-- 模态框标题 -->

        <div class="flex items-center justify-between border-b pb-3.5" style="border-color: var(--cbx-border-sub);">

          <div class="flex items-center gap-2.5">

            <span class="text-2xl p-2 rounded-2xl cbx-badge-purple">🤖</span>

            <div>

              <h3 class="text-base font-black cbx-text-main">AI 管家大模型引擎设置</h3>

              <p class="text-[11px] cbx-text-muted mt-0.5">支持 DeepSeek、通义千问、OpenAI、Claude、Gemini、Kimi、GLM、豆包及本地离线私有化</p>

            </div>

          </div>

          <button @click="app.showAISettingsModal = false" aria-label="关闭弹窗" class="cbx-btn-ghost min-w-[36px] min-h-[36px] px-2.5 py-1 rounded-xl text-xs font-bold transition cursor-pointer flex items-center justify-center cbx-touch-extend">✕</button>

        </div>

        <!-- 表单 -->

        <div class="space-y-4 text-xs font-mono">

          <!-- 10 大模型提供商选择网格 -->

          <div class="space-y-2">

            <div class="flex items-center justify-between gap-2 flex-wrap">

              <label class="block font-bold cbx-text-sub">选择大模型提供商 (点击一键载入推荐配置)</label>

              <span class="text-[10px] font-mono font-bold" style="color: var(--cbx-ok-fg);">● 即选即填</span>

            </div>

            <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">

              <button type="button" v-for="prov in app.aiProviders" :key="prov.id"

                @click="app.selectAIProvider(prov)"

                :class="[

                  'p-2.5 rounded-xl font-bold transition border cursor-pointer text-left flex items-start gap-2 select-none',

                  app.aiConfig.provider === prov.id ? 'cbx-badge-info shadow-sm ring-1 ring-sky-400/40' : 'cbx-sub-card'

                ]">

                <span class="text-lg shrink-0 mt-0.5">{{ prov.icon }}</span>

                <div class="min-w-0">

                  <div class="font-extrabold text-xs truncate flex items-center gap-1 cbx-text-main">

                    <span>{{ prov.name }}</span>

                    <span v-if="app.aiConfig.provider === prov.id" class="text-[10px]" style="color: var(--cbx-ok-fg);">✔</span>

                  </div>

                  <div class="text-[10px] cbx-text-muted truncate font-sans font-normal mt-0.5">{{ prov.desc }}</div>

                </div>

              </button>

            </div>

          </div>

          <!-- 模型名称 -->

          <div class="space-y-1.5">

            <div class="flex items-center justify-between gap-2 flex-wrap">

              <label class="block font-bold cbx-text-sub">模型名称 (Model)</label>

              <span class="text-[10px] cbx-text-muted">已自动为您预设推荐模型，支持手动修改</span>

            </div>

            <input v-model="app.aiConfig.model" placeholder="例如 deepseek-chat, qwen-plus, gpt-4o-mini, gemini-2.0-flash"

              class="w-full px-3.5 py-2.5 rounded-xl font-mono text-xs" />

          </div>

          <!-- API Key -->

          <div class="space-y-1.5">

            <div class="flex items-center justify-between gap-2 flex-wrap">

              <label class="block font-bold cbx-text-sub">API Key 凭证</label>

              <span class="text-[10px] font-semibold" style="color: var(--cbx-ok-fg);">本地 Ollama 可不填 · 留空启用内置启发式体检</span>

            </div>

            <input v-model="app.aiConfig.api_key" type="password" placeholder="sk-..."

              class="w-full px-3.5 py-2.5 rounded-xl font-mono text-xs" />

          </div>

          <!-- 自定义 BaseURL -->

          <div class="space-y-1.5" v-if="app.aiConfig.provider !== 'gemini'">

            <div class="flex items-center justify-between gap-2 flex-wrap">

              <label class="block font-bold cbx-text-sub">API 端点 (Base URL)</label>

              <span class="text-[10px] cbx-text-muted">可使用官方端点或您的私有中转/局域网端点</span>

            </div>

            <input v-model="app.aiConfig.base_url" placeholder="https://api.deepseek.com/v1"

              class="w-full px-3.5 py-2.5 rounded-xl font-mono text-xs" />

          </div>

        </div>

        <!-- 底部按钮 -->

        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-3 border-t" style="border-color: var(--cbx-border-sub);">

          <div class="text-[11px] cbx-text-muted flex items-center gap-1 font-sans">

            <span>💡 提示：零 Key 即时可用，配置 Key 解锁完整多轮对话</span>

          </div>

          <div class="flex items-center gap-2.5 self-end sm:self-auto">

            <button @click="app.showAISettingsModal = false" class="cbx-btn-ghost px-4 py-2 rounded-xl font-bold text-xs transition cursor-pointer">取消</button>

            <button @click="app.saveAIConfig" class="px-5 py-2 rounded-xl bg-sky-600 hover:bg-sky-700 text-white font-black text-xs shadow-sm transition active:scale-95 cursor-pointer">保存生效</button>

          </div>

        </div>

      </div>

    </div>
</template>

<script setup>
import { inject } from 'vue'
const app = inject('cubexApp')
</script>
