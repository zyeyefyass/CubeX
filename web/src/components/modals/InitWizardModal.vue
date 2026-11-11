<template>
    <div v-if="app.showInitWizardModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-2xl animate-in fade-in overflow-y-auto">

      <div class="relative w-full max-w-3xl rounded-3xl macos-glass-card shadow-2xl p-6 sm:p-8 space-y-6 text-slate-200 overflow-hidden">

        <!-- Canvas for Celebratory Aurora Fireworks -->

        <canvas v-show="app.initWizardStep === 4" id="init-fireworks-canvas" class="absolute inset-0 z-20 pointer-events-none w-full h-full"></canvas>

        <!-- Hidden input for snapshot file upload -->

        <input type="file" :ref="(el) => { app.initBackupFileInput = el }" @change="app.handleInitBackupUpload" accept=".json,.bak" class="hidden" />

        <!-- Modal Header -->

        <div class="flex items-center justify-between border-b border-slate-700/60 pb-4 relative z-10">

          <div class="flex items-center gap-3">

            <span class="text-2xl">✨</span>

            <div>

              <h3 class="text-lg font-black text-white flex items-center gap-2">

                <span>CubeX 宿主共生系统初始化向导</span>

                <span class="text-[10px] px-2 py-0.5 rounded-full bg-white/10 text-sky-200 border border-sky-400/30 font-bold drop-shadow-sm font-mono">

                  STEP {{ app.initWizardStep }} / 4

                </span>

              </h3>

              <p class="text-xs text-slate-400 mt-0.5">宿主基座直连 · 环境自动探测 · 策略路由双向编排</p>

            </div>

          </div>

          <button @click="app.showInitWizardModal = false; app.stopFireworks()" class="text-slate-400 hover:text-white text-lg font-bold p-1 rounded-lg hover:bg-white/20 border border-transparent hover:border-sky-400/50 text-white/80 transition">✕</button>

        </div>

        <!-- Wizard Step Indicator Pills (4 Steps) -->

        <div class="grid grid-cols-4 gap-2 text-xs font-mono relative z-10">

          <div :class="['p-2 rounded-xl border text-center transition', app.initWizardStep === 1 ? 'bg-cyan-500/20 border-cyan-400 text-emerald-200 font-bold shadow-sm' : (app.initWizardStep > 1 ? 'bg-white/10 text-white border-emerald-500/40 text-emerald-300' : 'bg-slate-950/60 border-slate-800 text-slate-300')]">

            <span>① 模式选择</span>

          </div>

          <div :class="['p-2 rounded-xl border text-center transition', app.initWizardStep === 2 ? 'bg-cyan-500/20 border-cyan-400 text-emerald-200 font-bold shadow-sm' : (app.initWizardStep > 2 ? 'bg-white/10 text-white border-emerald-500/40 text-emerald-300' : 'bg-slate-950/60 border-slate-800 text-slate-300')]">

            <span>② 连通宿主</span>

          </div>

          <div :class="['p-2 rounded-xl border text-center transition', app.initWizardStep === 3 ? 'bg-cyan-500/20 border-cyan-400 text-emerald-200 font-bold shadow-sm' : (app.initWizardStep > 3 ? 'bg-white/10 text-white border-emerald-500/40 text-emerald-300' : 'bg-slate-950/60 border-slate-800 text-slate-300')]">

            <span>③ 探测编排</span>

          </div>

          <div :class="['p-2 rounded-xl border text-center transition', app.initWizardStep === 4 ? 'bg-gradient-to-r from-teal-500/30 to-cyan-500/30 border-cyan-400 text-emerald-200 font-bold shadow-sm' : 'bg-slate-950/60 border-slate-800 text-slate-300']">

            <span>④ 导入与启航</span>

          </div>

        </div>

        <!-- ──────────────────────────────────────────────────────────── -->

        <!-- STEP 1: 选择导入备份 或 全新开始                               -->

        <!-- ──────────────────────────────────────────────────────────── -->

        <div v-if="app.initWizardStep === 1" class="space-y-4 animate-in fade-in relative z-10">

          <div class="p-3.5 rounded-2xl bg-cyan-950/30 border border-cyan-800/40 text-xs text-emerald-200 flex items-center gap-2">

            <span class="text-base shrink-0">💡</span>

            <p class="text-slate-300 leading-relaxed">

              欢迎使用 CubeX！请选择初始化方式：若已有备份快照可一键快速还原；若为初次安装部署或需彻底重置请选择全新开始。

            </p>

          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">

            <!-- Option 1: 全新开始 -->

            <div class="p-5 rounded-2xl bg-gradient-to-b from-[#00223e] to-[#00172e] border border-cyan-500/40 hover:border-cyan-400 transition-all flex flex-col justify-between space-y-4 shadow-lg group">

              <div class="space-y-2.5">

                <div class="flex items-center justify-between">

                  <div class="w-10 h-10 rounded-xl bg-cyan-500/20 border border-cyan-400/50 flex items-center justify-center text-xl shadow-[0_0_15px_rgba(0,210,255,0.3)]">

                    ✨

                  </div>

                  <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-white/10 text-sky-200 border border-sky-400/30 font-bold drop-shadow-sm font-bold">

                    推荐 · 全新部署

                  </span>

                </div>

                <div>

                  <h4 class="text-sm font-bold text-white group-hover:text-emerald-300 transition">全新开始 (Clean Setup)</h4>

                  <p class="text-xs text-slate-400 mt-1 leading-relaxed">

                    清除旧有代理与网络配置，恢复洁净纯净状态。向导将引导填入宿主凭据，自动探测底座并下发策略路由。

                  </p>

                </div>

                <div class="space-y-1.5 pt-1 text-[11px] text-slate-300 font-mono">

                  <div class="flex items-center gap-1.5 text-emerald-300">

                    <span>✓</span> <span>0 残留纯净出厂重置</span>

                  </div>

                  <div class="flex items-center gap-1.5 text-emerald-300">

                    <span>✓</span> <span>向导引导建立宿主共生</span>

                  </div>

                  <div class="flex items-center gap-1.5 text-emerald-300">

                    <span>✓</span> <span>一键自动编排隔离策略路由表 (table cubex)</span>

                  </div>

                </div>

              </div>

              <button @click="app.chooseFreshStart" class="w-full py-2.5 px-4 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white font-extrabold font-bold text-xs shadow-md shadow-sky-500/25 active:scale-95 transition flex items-center justify-center gap-1.5">

                <span>开始全新初始化 ➔</span>

              </button>

            </div>

            <!-- Option 2: 导入备份 -->

            <div class="p-5 rounded-2xl bg-gradient-to-b from-[#180e36] to-[#00172e] border border-purple-500/40 hover:border-purple-400 transition-all flex flex-col justify-between space-y-4 shadow-lg group">

              <div class="space-y-2.5">

                <div class="flex items-center justify-between">

                  <div class="w-10 h-10 rounded-xl bg-purple-500/20 border border-purple-400/50 flex items-center justify-center text-xl shadow-[0_0_15px_rgba(123,44,191,0.3)]">

                    📦

                  </div>

                  <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-purple-950 text-purple-300 border border-purple-500/40 font-bold">

                    快速还原 · 快照迁移

                  </span>

                </div>

                <div>

                  <h4 class="text-sm font-bold text-white group-hover:text-purple-200 transition">导入已有备份 (Restore Snapshot)</h4>

                  <p class="text-xs text-slate-400 mt-1 leading-relaxed">

                    从本地已导出的 CubeX 备份文件（.json 快照）一键还原全量订阅源、代理节点、分流规则与宿主凭据。

                  </p>

                </div>

                <!-- Dropzone / File Picker Area -->

                <div v-if="!app.initImportSuccess"

                  @click="app.triggerInitBackupSelect"

                  @dragover.prevent

                  @drop="app.handleInitFileDrop"

                  :class="[

                    'border-2 border-dashed rounded-xl p-4 text-center cursor-pointer transition select-none',

                    app.initImporting ? 'border-amber-400 bg-amber-950/20' : 'border-purple-500/40 hover:border-purple-400 bg-slate-950/60 hover:bg-purple-950/20'

                  ]">

                  <div v-if="app.initImporting" class="flex flex-col items-center gap-2 py-2">

                    <span class="inline-block animate-spin text-2xl text-purple-400">↻</span>

                    <span class="text-xs font-mono text-purple-200">正在解析快照并还原全量配置...</span>

                  </div>

                  <div v-else class="space-y-1">

                    <div class="text-2xl">📁</div>

                    <div class="text-xs font-bold text-purple-200">点击选择或拖拽 .json 快照文件</div>

                    <div class="text-[10px] text-slate-300 font-mono">支持 CubeX 导出的全量备份文件 (*.json)</div>

                  </div>

                </div>

                <!-- Import Success Notice & Action -->

                <div v-else class="p-3 rounded-xl bg-emerald-950/60 border border-emerald-500/50 space-y-1.5 animate-in zoom-in-95">

                  <div class="flex items-center gap-1.5 text-xs font-bold text-emerald-300">

                    <span>✓</span>

                    <span>快照已成功恢复装载！</span>

                  </div>

                  <div class="text-[11px] font-mono text-slate-300 space-y-0.5">

                    <div>已装载网络订阅: <b class="text-emerald-300">{{ app.initImportStats.sub_count }}</b> 条</div>

                    <div>已装载出站节点: <b class="text-emerald-300">{{ app.initImportStats.proxy_count }}</b> 个</div>

                    <div class="text-[10px] text-slate-400 truncate">文件: {{ app.initImportStats.filename }}</div>

                  </div>

                </div>

              </div>

              <!-- Action buttons for Option 2 -->

              <div v-if="app.initImportSuccess" class="grid grid-cols-2 gap-2 pt-1">

                <button @click="app.initWizardStep = 3; app.startInitStep3Execution()" class="py-2.5 px-3 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 text-white font-bold text-xs shadow-md shadow-sky-500/20 active:scale-95 transition flex items-center justify-center gap-1">

                  <span>探测并同步宿主 ➔</span>

                </button>

                <button @click="app.initWizardStep = 4; app.launchFireworks()" class="py-2.5 px-3 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white text-emerald-300 border border-cyan-500/40 font-bold text-xs active:scale-95 transition flex items-center justify-center gap-1">

                  <span>跳过直接启航 🚀</span>

                </button>

              </div>

              <div v-else>

                <button @click="app.triggerInitBackupSelect" class="w-full py-2.5 px-4 rounded-xl bg-gradient-to-r from-purple-500 to-indigo-500 hover:from-purple-400 hover:to-indigo-400 text-white font-bold text-xs shadow-md shadow-purple-500/25 active:scale-95 transition flex items-center justify-center gap-1.5">

                  <span>选择本地备份快照 ➔</span>

                </button>

              </div>

            </div>

          </div>

        </div>

        <!-- ──────────────────────────────────────────────────────────── -->

        <!-- STEP 2: 填入宿主连通凭据                                       -->

        <!-- ──────────────────────────────────────────────────────────── -->

        <div v-else-if="app.initWizardStep === 2" class="space-y-4 animate-in fade-in relative z-10">

          <div class="p-4 rounded-2xl bg-cyan-950/30 border border-cyan-800/40 text-xs text-emerald-200 space-y-1">

            <div class="font-bold flex items-center gap-1.5 text-emerald-300">

              <span>💡</span>

              <span>第二步：建立 CubeX 与宿主底座的安全连通通道</span>

            </div>

            <p class="text-slate-300 leading-relaxed">

              填入宿主管理地址与管理员凭据。对于 <b>MikroTik RouterOS</b>，账号密码完全指你平时登录 WinBox 客户端所用的账号（默认 admin）与登录密码（无密码直接留空）。

            </p>

          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5 text-xs font-mono">

            <div>

              <label class="block text-slate-400 mb-1">宿主管理地址 (Host IP)</label>

              <input v-model="app.initHostForm.host_ip" placeholder="请输入或自动探测主路由 IP"

                class="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 focus:border-cyan-400 outline-none text-white shadow-inner font-mono" />

            </div>

            <div>

              <label class="block text-slate-400 mb-1">Web / REST 端口 (Port)</label>

              <input v-model="app.initHostForm.port" placeholder="ROS: 80/443, Linux: 22"

                class="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 focus:border-cyan-400 outline-none text-white shadow-inner font-mono" />

            </div>

            <div>

              <label class="block text-slate-400 mb-1">管理员账号 (Username)</label>

              <input v-model="app.initHostForm.username" placeholder="例: admin"

                class="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 focus:border-cyan-400 outline-none text-white shadow-inner font-mono" />

            </div>

            <div>

              <div class="flex items-center justify-between mb-1">

                <label class="block text-slate-400">管理员密码 (Password)</label>

                <button type="button" @click="app.initHostForm.showPassword = !app.initHostForm.showPassword" class="text-[10px] text-emerald-300 hover:underline">

                  {{ app.initHostForm.showPassword ? '隐藏' : '显示' }}

                </button>

              </div>

              <input :type="app.initHostForm.showPassword ? 'text' : 'password'" v-model="app.initHostForm.password" placeholder="WinBox登录密码 (无密码留空)"

                class="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 focus:border-cyan-400 outline-none text-white shadow-inner font-mono" />

            </div>

          </div>

          <div class="flex items-center justify-between pt-4 border-t border-sky-900/40">

            <button @click="app.initWizardStep = 1" class="px-4 py-2 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white border border-slate-800 text-slate-300 text-xs font-semibold active:scale-95 transition">

              ⬅ 返回模式选择

            </button>

            <button @click="app.startInitStep3Execution" class="px-5 py-2.5 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 text-white font-bold text-xs shadow-md shadow-sky-500/20 active:scale-95 transition flex items-center gap-1.5">

              <span>下一步：探测宿主并自动编排 ➔</span>

            </button>

          </div>

        </div>

        <!-- ──────────────────────────────────────────────────────────── -->

        <!-- STEP 3: 系统判断宿主类型并自动执行两条                         -->

        <!-- ──────────────────────────────────────────────────────────── -->

        <div v-else-if="app.initWizardStep === 3" class="space-y-4 animate-in fade-in relative z-10">

          <div class="p-4 rounded-2xl bg-white/10 text-white/90 border border-slate-800 flex items-center justify-between">

            <div class="flex items-center gap-3">

              <span class="text-2xl">{{ app.initDetectedPlatform === 'routeros' ? '👑' : '🐧' }}</span>

              <div>

                <div class="text-xs text-slate-400">识别到的宿主环境平台</div>

                <div class="text-sm font-bold text-emerald-300">

                  {{ app.initDetectedPlatform === 'routeros' ? 'MikroTik RouterOS v7.x (官方硬件微核)' : 'Linux 宿主底座 (fnOS / PVE / Docker)' }}

                </div>

              </div>

            </div>

            <span :class="['text-xs font-mono px-2.5 py-1 rounded-full border', app.initExecuting ? 'bg-amber-950/80 text-amber-300 border-amber-500/40 animate-pulse' : 'bg-emerald-950 text-emerald-300 border-emerald-500/40']">

              {{ app.initExecuting ? '⚡ 正在自动编排...' : '✓ 编排就绪' }}

            </span>

          </div>

          <!-- Live Execution Terminal -->

          <div class="p-4 rounded-2xl bg-slate-950 border border-slate-800 space-y-2.5 font-mono text-xs max-h-56 overflow-y-auto shadow-inner">

            <div class="text-[10px] text-slate-300 uppercase tracking-widest border-b border-slate-900 pb-1 flex justify-between">

              <span>EXECUTION DISPATCHER</span>

              <span>AUTO-PILOT</span>

            </div>

            <div v-for="(log, idx) in app.initLogs" :key="idx" class="flex items-start gap-2 leading-relaxed">

              <span v-if="log.status === 'running'" class="inline-block animate-spin text-emerald-300 shrink-0">↻</span>

              <span v-else class="text-emerald-400 shrink-0">✓</span>

              <span class="text-slate-200">{{ log.text }}</span>

            </div>

          </div>

          <div class="p-3 rounded-xl bg-slate-900/70 text-white/75 border border-slate-800 text-[11px] text-slate-400 flex items-center justify-between">

            <span>正在执行：巡检隔离策略路由表 (table cubex)、下发 Mangle 防火墙规则与配置防 DNS 泄露...</span>

            <span v-if="app.initExecuting" class="text-emerald-300 font-mono animate-pulse">处理中...</span>

          </div>

        </div>

        <!-- ──────────────────────────────────────────────────────────── -->

        <!-- STEP 4: 节点订阅导入与启航就绪                                  -->

        <!-- ──────────────────────────────────────────────────────────── -->

        <div v-else-if="app.initWizardStep === 4" class="space-y-4 animate-in zoom-in-95 duration-500 relative z-30">

          <div class="flex items-center gap-3 p-3.5 rounded-2xl bg-cyan-950/40 border border-cyan-500/40 shadow-sm">

            <div class="w-10 h-10 rounded-2xl bg-gradient-to-tr from-cyan-500 to-emerald-400 flex items-center justify-center text-xl shadow-[0_0_15px_rgba(0,245,212,0.5)] shrink-0 animate-bounce">

              🎉

            </div>

            <div class="flex-1 min-w-0">

              <h3 class="text-sm font-black text-white flex items-center gap-2">

                <span>宿主底座与策略路由已原生打通！</span>

                <span class="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-300 border border-emerald-400/30 font-mono">

                  ✓ READY

                </span>

              </h3>

              <p class="text-xs text-slate-300 mt-0.5 truncate">

                最后一步：可直接粘贴订阅源或节点链接一键启航，也可跳过进入主控台后续再导。

              </p>

            </div>

          </div>

          <!-- Feature Readiness Matrix -->

          <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs font-mono text-left">

            <div class="p-2 rounded-xl bg-slate-950/80 border border-cyan-500/30 space-y-0.5">

              <span class="text-[10px] text-emerald-300">宿主平台</span>

              <div class="font-bold text-white truncate">{{ app.initDetectedPlatform === 'routeros' ? 'RouterOS v7' : 'Linux / fnOS' }}</div>

            </div>

            <div class="p-2 rounded-xl bg-slate-950/80 border border-emerald-500/30 space-y-0.5">

              <span class="text-[10px] text-emerald-400">策略路由表</span>

              <div class="font-bold text-white">table cubex 就绪</div>

            </div>

            <div class="p-2 rounded-xl bg-slate-950/80 border border-purple-500/30 space-y-0.5">

              <span class="text-[10px] text-purple-400">防 DNS 泄露</span>

              <div class="font-bold text-white">217heidai 权威规则</div>

            </div>

            <div class="p-2 rounded-xl bg-slate-950/80 border border-amber-500/30 space-y-0.5">

              <span class="text-[10px] text-amber-400">双擎混动</span>

              <div class="font-bold text-white">Mihomo / sing-box</div>

            </div>

          </div>

          <!-- Embedded Batch Multi-Import Card -->

          <div class="space-y-3 p-4 rounded-2xl bg-slate-950/90 border border-cyan-500/30 shadow-inner">

            <!-- 2-Segment Switcher -->

            <div class="flex p-1 bg-white/10 text-white rounded-xl border border-slate-800 text-xs font-mono">

              <button type="button" @click="app.initWizardImportTab = 'sub'" :class="['flex-1 py-1.5 rounded-lg font-bold transition flex items-center justify-center gap-1.5', app.initWizardImportTab === 'sub' ? 'bg-cyan-500 text-slate-950 shadow-md shadow-sky-500/20' : 'text-slate-400 hover:text-slate-200']">

                <span>🌐 批量导入订阅</span>

                <span v-if="app.parsedBatchSubs.length > 0" class="px-1.5 py-0.2 rounded-full text-[10px] bg-slate-950 text-emerald-300 font-mono">

                  {{ app.parsedBatchSubs.length }}

                </span>

              </button>

              <button type="button" @click="app.initWizardImportTab = 'node'" :class="['flex-1 py-1.5 rounded-lg font-bold transition flex items-center justify-center gap-1.5', app.initWizardImportTab === 'node' ? 'bg-cyan-500 text-slate-950 shadow-md shadow-sky-500/20' : 'text-slate-400 hover:text-slate-200']">

                <span>🔗 批量导入自建节点</span>

                <span v-if="app.parsedBatchNodes.length > 0" class="px-1.5 py-0.2 rounded-full text-[10px] bg-slate-950 text-emerald-300 font-mono">

                  {{ app.parsedBatchNodes.length }}

                </span>

              </button>

            </div>

            <!-- Tab 1: Subscriptions -->

            <div v-if="app.initWizardImportTab === 'sub'" class="space-y-2">

              <div class="flex items-center justify-between text-xs font-semibold text-slate-300">

                <span class="text-[11px] text-slate-400">支持 Clash / sing-box / V2Ray 订阅链接，每行一条（支持直接指定名称）：</span>

                <span v-if="app.parsedBatchSubs.length > 0" class="text-emerald-300 font-mono text-[11px] font-bold">

                  ⚡ 实时识别: {{ app.parsedBatchSubs.length }} 条订阅

                </span>

              </div>

              <textarea v-model="app.batchSubText" rows="3"

                placeholder="直接粘贴订阅源链接，每行一条。例如：&#10;香港主力专线, https://sub.network1.com/api/v1/client/subscribe?token=abc&#10;备用海外节点 | https://sub.network2.com/clash/token123"

                class="w-full bg-white/10 text-white border border-slate-700/80 rounded-xl p-2.5 text-xs text-slate-100 font-mono outline-none focus:border-cyan-400 focus:ring-1 focus:ring-cyan-400 shadow-inner resize-y leading-relaxed" />

              <!-- Parsed Preview -->

              <div v-if="app.parsedBatchSubs.length > 0" class="max-h-24 overflow-y-auto space-y-1 font-mono text-[11px] bg-slate-900/70 text-white/75 p-2 rounded-xl border border-cyan-500/20">

                <div v-for="(item, idx) in app.parsedBatchSubs" :key="idx" class="flex items-center gap-2 truncate text-slate-300">

                  <span class="text-emerald-300 font-bold">✈️ {{ item.name }}</span>

                  <span class="text-slate-300 truncate text-[10px]">{{ item.url }}</span>

                </div>

              </div>

            </div>

            <!-- Tab 2: Custom Nodes -->

            <div v-else class="space-y-2">

              <div class="flex items-center justify-between text-xs font-semibold text-slate-300">

                <span class="text-[11px] text-slate-400">支持 ss, vmess, vless, trojan, hysteria2, tuic 节点链接，每行一条：</span>

                <span v-if="app.parsedBatchNodes.length > 0" class="text-purple-400 font-mono text-[11px] font-bold">

                  ⚡ 实时识别: {{ app.parsedBatchNodes.length }} 个节点

                </span>

              </div>

              <textarea v-model="app.batchNodeText" rows="3"

                placeholder="直接粘贴节点链接，每行一条。例如：&#10;东京落地 01, ss://YWVzLTI1Ni1nY206cGFzc0AxLjIuMy40OjgzODg=#东京&#10;vless://uuid@us.example.com:443?security=reality&sni=yahoo.com#美西"

                class="w-full bg-white/10 text-white border border-slate-700/80 rounded-xl p-2.5 text-xs text-slate-100 font-mono outline-none focus:border-purple-400 focus:ring-1 focus:ring-purple-400 shadow-inner resize-y leading-relaxed" />

              <!-- Parsed Preview -->

              <div v-if="app.parsedBatchNodes.length > 0" class="max-h-24 overflow-y-auto space-y-1 font-mono text-[11px] bg-slate-900/70 text-white/75 p-2 rounded-xl border border-purple-500/20">

                <div v-for="(item, idx) in app.parsedBatchNodes" :key="idx" class="flex items-center gap-2 truncate text-slate-300">

                  <span class="text-purple-400 font-bold uppercase text-[10px]">[{{ item.protocol }}]</span>

                  <span class="text-white font-bold">🔗 {{ item.name }}</span>

                  <span class="text-slate-300 truncate text-[10px]">{{ item.link }}</span>

                </div>

              </div>

            </div>

          </div>

          <!-- Bottom Actions -->

          <div class="pt-3 border-t border-sky-800/40 flex flex-col sm:flex-row items-center justify-between gap-3">

            <button @click="app.completeInitWizard" class="w-full sm:w-auto px-4 py-2.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white border border-slate-700 text-slate-300 text-xs font-semibold active:scale-95 transition">

              跳过直接进入主控台 ➔

            </button>

            <button v-if="(app.initWizardImportTab === 'sub' && app.parsedBatchSubs.length > 0) || (app.initWizardImportTab === 'node' && app.parsedBatchNodes.length > 0)"

              @click="app.submitInitWizardImportAndLaunch"

              :disabled="app.batchImportingSubs || app.batchAddingNodes"

              class="w-full sm:w-auto px-6 py-2.5 rounded-xl bg-gradient-to-r from-teal-400 via-cyan-400 to-blue-500 hover:from-teal-300 hover:to-cyan-300 text-slate-950 font-black text-xs shadow-[0_0_20px_rgba(0,210,255,0.4)] active:scale-95 transition flex items-center justify-center gap-1.5 disabled:opacity-50">

              <span v-if="app.batchImportingSubs || app.batchAddingNodes" class="inline-block animate-spin">↻</span>

              <span>{{ app.batchImportingSubs || app.batchAddingNodes ? '正在导入并重载核心...' : '🚀 立即导入并启航' }}</span>

            </button>

            <button v-else

              @click="app.completeInitWizard"

              class="w-full sm:w-auto px-6 py-2.5 rounded-xl bg-gradient-to-r from-teal-400 via-cyan-400 to-blue-500 hover:from-teal-300 hover:to-cyan-300 text-slate-950 font-black text-xs shadow-[0_0_20px_rgba(0,210,255,0.4)] active:scale-95 transition flex items-center justify-center gap-1.5">

              <span>🚀 立即启航，进入系统主控台</span>

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
