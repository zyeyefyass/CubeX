<template>
    <div v-if="app.showLogsModal" class="fixed inset-0 z-50 bg-black/80 backdrop-blur-xl flex items-center justify-center p-3 sm:p-5 overflow-y-auto animate-in fade-in" @click.self="app.showLogsModal = false">

      <div class="bg-[#0b0816]/95 border border-white/20 rounded-3xl max-w-5xl w-full shadow-2xl flex flex-col text-slate-200 overflow-hidden relative">

        <!-- Terminal Header -->

        <div class="px-4 sm:px-6 py-3 bg-white/[0.04] border-b border-white/10 flex items-center justify-between gap-3 select-none flex-wrap sm:flex-nowrap">

          <!-- Left Status -->

          <div class="flex items-center gap-2.5 shrink-0">

            <div class="flex items-center gap-1.5">

              <span class="w-2.5 h-2.5 rounded-full bg-rose-500/80 inline-block shadow-sm"></span>

              <span class="w-2.5 h-2.5 rounded-full bg-amber-500/80 inline-block shadow-sm"></span>

              <span class="w-2.5 h-2.5 rounded-full bg-emerald-500/80 inline-block shadow-sm"></span>

            </div>

            <div class="h-3.5 w-px bg-slate-700/80 mx-0.5"></div>

            <div class="flex items-center gap-2">

              <span class="text-xs sm:text-[13px] font-mono font-black text-white tracking-wide flex items-center gap-1.5">

                <ZootopiaIcon name="diagnostics" :size="16" class="shrink-0" />

                <span>CUBEX_LIVE_RING_BUFFER</span>

              </span>

              <span class="text-[10px] font-mono px-2 py-0.5 rounded-full font-bold flex items-center gap-1 bg-emerald-500/15 text-emerald-300 border border-emerald-400/30">

                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-ping"></span>

                实时捕获

              </span>

            </div>

          </div>

          <!-- Right Controls -->

          <div class="flex items-center gap-2 text-xs font-mono shrink-0 flex-nowrap">

            <!-- Level Filter Pills -->

            <div class="flex items-center macos-capsule p-0.5 rounded-xl">

              <button v-for="lvl in ['ALL', 'INFO', 'WARN', 'ERROR']" :key="lvl"

                @click="app.logFilterLevel = lvl"

                :class="[

                  'px-2.5 py-1 rounded-lg text-[10px] font-bold transition-all cursor-pointer',

                  app.logFilterLevel === lvl ? 'bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 text-white font-extrabold shadow-sm' : 'text-slate-300 hover:text-white hover:bg-white/10'

                ]">

                {{ lvl }}

              </button>

            </div>

            <!-- Clear Screen -->

            <button @click="app.clearSystemLogs" class="px-2.5 py-1 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-slate-300 hover:text-white text-[11px] font-bold active:scale-95 transition cursor-pointer" title="清空当前环形缓冲区日志">

              <span>🗑 清屏</span>

            </button>

            <!-- Refresh -->

            <button @click="app.fetchSystemLogs(true)" class="p-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-slate-300 hover:text-white active:scale-95 transition cursor-pointer" title="立即刷新日志">

              <span>🔄</span>

            </button>

            <!-- Copy Desensitized Markdown Report -->

            <button @click="app.copyDiagnosticMarkdown"

              class="px-3 py-1 rounded-xl text-[11px] font-bold bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-slate-200 active:scale-95 transition flex items-center gap-1 cursor-pointer"

              title="复制 Markdown 格式脱敏排障报告至剪贴板">

              <span>📋</span>

              <span>复制报告</span>

            </button>

            <!-- One-Click Upload to GitHub Issues -->

            <button @click="app.uploadToGithubIssue" :disabled="app.creatingGithubIssue"

              class="px-3.5 py-1 rounded-xl text-[11px] font-extrabold bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white shadow-md shadow-sky-500/20 active:scale-95 transition flex items-center gap-1.5 cursor-pointer shrink-0"

              title="在浏览器直达 GitHub Issues 预填脱敏排障报告">

              <span v-if="app.creatingGithubIssue" class="inline-block animate-spin mr-0.5">↻</span>

              <span v-else>🚀</span>

              <span>问题反馈</span>

            </button>

            <!-- Close Modal Button -->

            <button @click="app.showLogsModal = false" class="px-2.5 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 text-slate-300 hover:text-white text-xs font-bold transition cursor-pointer" title="关闭窗口">

              ✕

            </button>

          </div>

        </div>

        <!-- Terminal Output Window -->

        <div :ref="(el) => { app.logContainerRef = el }" class="h-[520px] max-h-[70vh] overflow-y-auto no-scrollbar p-3.5 sm:p-4 font-mono text-xs bg-black/40 select-text transition-all">

          <div class="space-y-1">

            <div v-if="app.displayedLogs.length === 0" class="h-full flex flex-col items-center justify-center text-slate-300 space-y-2 py-12">

              <span class="text-3xl">📜</span>

              <span class="text-xs">当前过滤器下暂无日志记录</span>

              <button v-if="app.logFilterLevel !== 'ALL' || app.logFilterModule !== 'ALL' || app.logSearchQuery"

                @click="app.logFilterLevel = 'ALL'; app.logFilterModule = 'ALL'; app.logSearchQuery = ''"

                class="text-[11px] text-emerald-300 underline hover:text-emerald-300 cursor-pointer">

                重置所有过滤器

              </button>

            </div>

            <div v-for="entry in app.displayedLogs" :key="entry.id" class="flex items-start gap-2.5 leading-normal hover:bg-white/[0.06] px-2 py-0.5 rounded transition font-mono group">

              <!-- Timestamp -->

              <span class="text-slate-400 text-[11px] shrink-0 select-none">{{ entry.time }}</span>

              <!-- Level Badge -->

              <span :class="[

                'px-1.5 py-0.2 rounded text-[10px] font-bold shrink-0 select-none uppercase tracking-wider',

                entry.level === 'ERROR' ? 'bg-rose-950 text-rose-300 border border-rose-500/50 animate-pulse' :

                entry.level === 'WARN' ? 'bg-amber-950 text-amber-300 border border-amber-500/50' :

                entry.level === 'DEBUG' ? 'bg-purple-950 text-purple-300 border border-purple-500/40' :

                'bg-[rgba(255,158,94,0.18)] text-sky-300 border border-[rgba(255,158,94,0.35)]'

              ]">

                {{ entry.level }}

              </span>

              <!-- Module Tag -->

              <span class="px-1.5 py-0.2 rounded bg-white/[0.06] text-slate-300 border border-white/10 text-[10px] font-bold shrink-0 select-none">

                {{ entry.module }}

              </span>

              <!-- Message Text -->

              <span :class="[

                'flex-1 break-all text-xs font-mono',

                entry.level === 'ERROR' ? 'text-rose-200 font-bold' :

                entry.level === 'WARN' ? 'text-amber-200' :

                'text-slate-200'

              ]">

                {{ entry.msg }}

              </span>

            </div>

          </div>

          <!-- Terminal Cursor Blink -->

          <div class="pt-2 sm:pt-2.5 flex items-center gap-2 text-emerald-300 text-xs select-none">

            <span>cubex@host:~$</span>

            <span class="inline-block w-2 h-4 bg-cyan-400 animate-pulse"></span>

          </div>

        </div>

      </div>

    </div>
</template>

<script setup>
import { inject } from 'vue'
import ZootopiaIcon from '../ZootopiaIcon.vue'
const app = inject('cubexApp')
</script>
