<template>
    <div v-if="app.showAISettingsModal" class="fixed inset-0 z-50 bg-black/80 backdrop-blur-xl flex items-center justify-center p-3 sm:p-4 overflow-y-auto animate-in fade-in" @click.self="app.showAISettingsModal = false">

      <div class="bg-[#0e091c]/95 border border-white/20 rounded-3xl p-5 sm:p-7 max-w-2xl w-full space-y-5 shadow-2xl text-slate-200 my-auto">

        <!-- 模态框标题 -->

        <div class="flex items-center justify-between border-b border-white/10 pb-3.5">

          <div class="flex items-center gap-2.5">

            <span class="text-2xl p-2 rounded-2xl bg-purple-500/15 border border-purple-500/30">🤖</span>

            <div>

              <h3 class="text-base font-black text-white">AI 管家大模型引擎设置</h3>

              <p class="text-[11px] text-slate-400 mt-0.5">支持 DeepSeek、通义千问、OpenAI、Claude、Gemini、Kimi、GLM、豆包及本地离线私有化</p>

            </div>

          </div>

          <button @click="app.showAISettingsModal = false" class="text-slate-400 hover:text-white px-2.5 py-1 rounded-xl bg-white/10 hover:bg-white/20 text-xs font-bold transition cursor-pointer">✕</button>

        </div>

        <!-- 表单 -->

        <div class="space-y-4 text-xs font-mono">

          <!-- 10 大模型提供商选择网格 -->

          <div class="space-y-2">

            <div class="flex items-center justify-between">

              <label class="block font-bold text-slate-300">选择大模型提供商 (点击一键载入推荐配置)</label>

              <span class="text-[10px] text-emerald-400 font-mono">● 即选即填</span>

            </div>

            <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">

              <button type="button" v-for="prov in app.aiProviders" :key="prov.id"

                @click="app.selectAIProvider(prov)"

                :class="[

                  'p-2.5 rounded-xl font-bold transition border cursor-pointer text-left flex items-start gap-2 select-none',

                  app.aiConfig.provider === prov.id ? 'bg-sky-500/20 text-white border-sky-400/40 shadow-md ring-1 ring-sky-400/50' : 'bg-white/5 border-white/10 text-slate-300 hover:bg-white/10 hover:border-white/20'

                ]">

                <span class="text-lg shrink-0 mt-0.5">{{ prov.icon }}</span>

                <div class="min-w-0">

                  <div class="font-extrabold text-xs truncate flex items-center gap-1">

                    <span>{{ prov.name }}</span>

                    <span v-if="app.aiConfig.provider === prov.id" class="text-[10px] text-emerald-300">✔</span>

                  </div>

                  <div class="text-[10px] text-slate-400 truncate font-sans font-normal mt-0.5">{{ prov.desc }}</div>

                </div>

              </button>

            </div>

          </div>

          <!-- 模型名称 -->

          <div class="space-y-1.5">

            <div class="flex items-center justify-between">

              <label class="block font-bold text-slate-300">模型名称 (Model)</label>

              <span class="text-[10px] text-slate-400">已自动为您预设推荐模型，支持手动修改</span>

            </div>

            <input v-model="app.aiConfig.model" placeholder="例如 deepseek-chat, qwen-plus, gpt-4o-mini, gemini-2.0-flash"

              class="w-full px-3.5 py-2.5 rounded-xl bg-black/40 border border-white/15 text-white font-mono focus:outline-none focus:border-sky-400/60 text-xs" />

          </div>

          <!-- API Key -->

          <div class="space-y-1.5">

            <div class="flex items-center justify-between">

              <label class="block font-bold text-slate-300">API Key 凭证</label>

              <span class="text-[10px] text-emerald-400">本地 Ollama 可不填 · 留空启用内置启发式体检</span>

            </div>

            <input v-model="app.aiConfig.api_key" type="password" placeholder="sk-..."

              class="w-full px-3.5 py-2.5 rounded-xl bg-black/40 border border-white/15 text-white font-mono focus:outline-none focus:border-sky-400/60 text-xs" />

          </div>

          <!-- 自定义 BaseURL -->

          <div class="space-y-1.5" v-if="app.aiConfig.provider !== 'gemini'">

            <div class="flex items-center justify-between">

              <label class="block font-bold text-slate-300">API 端点 (Base URL)</label>

              <span class="text-[10px] text-slate-400">可使用官方端点或您的私有中转/局域网端点</span>

            </div>

            <input v-model="app.aiConfig.base_url" placeholder="https://api.deepseek.com/v1"

              class="w-full px-3.5 py-2.5 rounded-xl bg-black/40 border border-white/15 text-white font-mono focus:outline-none focus:border-sky-400/60 text-xs" />

          </div>

        </div>

        <!-- 底部按钮 -->

        <div class="flex items-center justify-between pt-3 border-t border-white/10">

          <div class="text-[11px] text-slate-400 flex items-center gap-1 font-sans">

            <span>💡 提示：零 Key 即时可用，配置 Key 解锁完整多轮对话</span>

          </div>

          <div class="flex items-center gap-2.5">

            <button @click="app.showAISettingsModal = false" class="px-4 py-2 rounded-xl bg-white/10 hover:bg-white/20 text-slate-300 font-bold text-xs transition cursor-pointer">取消</button>

            <button @click="app.saveAIConfig" class="px-5 py-2 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 text-white font-black text-xs shadow-md shadow-sky-500/20 transition active:scale-95 cursor-pointer">保存生效</button>

          </div>

        </div>

      </div>

    </div>
</template>

<script setup>
import { inject } from 'vue'
const app = inject('cubexApp')
</script>
