<template>
  <div v-if="app.showInitWizardModal" @click.self="app.showInitWizardModal = false; app.stopFireworks()" class="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-black/60 backdrop-blur-sm animate-in fade-in overflow-y-auto">
    <div class="relative w-full max-w-3xl rounded-3xl cbx-modal-shell shadow-2xl p-5 sm:p-8 space-y-5 overflow-hidden max-h-[94vh] overflow-y-auto">
      <!-- Canvas for Celebratory Aurora Fireworks -->
      <canvas v-show="app.initWizardStep === 4" id="init-fireworks-canvas" class="absolute inset-0 z-20 pointer-events-none w-full h-full"></canvas>

      <!-- Hidden input for snapshot file upload -->
      <input type="file" :ref="(el) => { app.initBackupFileInput = el }" @change="app.handleInitBackupUpload" accept=".json,.bak" class="hidden" />

      <!-- Modal Header -->
      <div class="flex items-center justify-between gap-2 pb-4 border-b relative z-10" style="border-color: var(--cbx-border-soft);">
        <div class="flex items-center gap-3 min-w-0">
          <span class="text-2xl shrink-0">✨</span>
          <div class="min-w-0">
            <h3 class="text-base sm:text-lg font-black cbx-text-main flex items-center gap-2 flex-wrap">
              <span>CubeX 宿主共生系统初始化向导</span>
              <span class="text-[10px] px-2 py-0.5 rounded-full cbx-badge-info border font-mono font-bold">
                STEP {{ app.initWizardStep }} / 4
              </span>
            </h3>
            <p class="text-xs cbx-text-sub mt-0.5">宿主基座直连 · 环境自动探测 · 策略路由双向编排</p>
          </div>
        </div>
        <button @click="app.showInitWizardModal = false; app.stopFireworks()" class="cbx-btn-ghost text-sm font-bold px-3 py-1.5 rounded-xl shrink-0 cursor-pointer" aria-label="关闭初始化向导">✕</button>
      </div>

      <!-- Wizard Step Indicator Pills (4 Steps) -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs font-mono relative z-10">
        <div :class="['p-2 rounded-xl border text-center transition', app.initWizardStep === 1 ? 'cbx-badge-info font-bold shadow-sm' : (app.initWizardStep > 1 ? 'cbx-badge-ok font-semibold' : 'cbx-sub-card cbx-text-muted')]">
          <span>① 模式选择</span>
        </div>
        <div :class="['p-2 rounded-xl border text-center transition', app.initWizardStep === 2 ? 'cbx-badge-info font-bold shadow-sm' : (app.initWizardStep > 2 ? 'cbx-badge-ok font-semibold' : 'cbx-sub-card cbx-text-muted')]">
          <span>② 连通宿主</span>
        </div>
        <div :class="['p-2 rounded-xl border text-center transition', app.initWizardStep === 3 ? 'cbx-badge-info font-bold shadow-sm' : (app.initWizardStep > 3 ? 'cbx-badge-ok font-semibold' : 'cbx-sub-card cbx-text-muted')]">
          <span>③ 探测编排</span>
        </div>
        <div :class="['p-2 rounded-xl border text-center transition', app.initWizardStep === 4 ? 'cbx-badge-ok font-bold shadow-sm' : 'cbx-sub-card cbx-text-muted']">
          <span>④ 导入与启航</span>
        </div>
      </div>

      <!-- STEP 1: 选择导入备份 或 全新开始 -->
      <div v-if="app.initWizardStep === 1" class="space-y-4 animate-in fade-in relative z-10">
        <div class="p-3.5 rounded-2xl cbx-sub-card text-xs flex items-center gap-2.5">
          <span class="text-base shrink-0">💡</span>
          <p class="cbx-text-sub leading-relaxed">
            欢迎使用 CubeX！请选择初始化方式：若已有备份快照可一键快速还原；若为初次安装部署或需彻底重置请选择全新开始。
          </p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <!-- Option 1: 全新开始 -->
          <div class="p-5 rounded-2xl cbx-sub-card border-2 border-sky-500/35 hover:border-sky-500/70 transition-all flex flex-col justify-between space-y-4 shadow-sm group">
            <div class="space-y-2.5">
              <div class="flex items-center justify-between gap-2">
                <div class="w-10 h-10 rounded-xl cbx-badge-info border flex items-center justify-center text-xl">
                  ✨
                </div>
                <span class="text-[10px] font-mono px-2 py-0.5 rounded-full cbx-badge-info border font-bold">
                  推荐 · 全新部署
                </span>
              </div>
              <div>
                <h4 class="text-sm font-bold cbx-text-main group-hover:text-sky-500 transition">全新开始 (Clean Setup)</h4>
                <p class="text-xs cbx-text-sub mt-1 leading-relaxed">
                  清除旧有代理与网络配置，恢复洁净纯净状态。向导将引导填入宿主凭据，自动探测底座并下发策略路由。
                </p>
              </div>
              <div class="space-y-1.5 pt-1 text-[11px] font-mono text-emerald-600 dark:text-emerald-300">
                <div class="flex items-center gap-1.5"><span>✓</span> <span>0 残留纯净出厂重置</span></div>
                <div class="flex items-center gap-1.5"><span>✓</span> <span>向导引导建立宿主共生</span></div>
                <div class="flex items-center gap-1.5"><span>✓</span> <span>一键自动编排隔离策略路由表 (table cubex)</span></div>
              </div>
            </div>
            <button @click="app.chooseFreshStart" class="w-full py-2.5 px-4 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white font-bold text-xs shadow-md shadow-sky-500/25 active:scale-95 transition flex items-center justify-center gap-1.5 cursor-pointer">
              <span>开始全新初始化 ➔</span>
            </button>
          </div>

          <!-- Option 2: 导入备份 -->
          <div class="p-5 rounded-2xl cbx-sub-card border-2 border-purple-500/35 hover:border-purple-500/70 transition-all flex flex-col justify-between space-y-4 shadow-sm group">
            <div class="space-y-2.5">
              <div class="flex items-center justify-between gap-2">
                <div class="w-10 h-10 rounded-xl cbx-badge-purple border flex items-center justify-center text-xl">
                  📦
                </div>
                <span class="text-[10px] font-mono px-2 py-0.5 rounded-full cbx-badge-purple border font-bold">
                  快速还原 · 快照迁移
                </span>
              </div>
              <div>
                <h4 class="text-sm font-bold cbx-text-main group-hover:text-purple-500 transition">导入已有备份 (Restore Snapshot)</h4>
                <p class="text-xs cbx-text-sub mt-1 leading-relaxed">
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
                  app.initImporting ? 'cbx-badge-warn' : 'cbx-sub-card hover:border-purple-400'
                ]">
                <div v-if="app.initImporting" class="flex flex-col items-center gap-2 py-2">
                  <span class="inline-block animate-spin text-2xl text-purple-500">↻</span>
                  <span class="text-xs font-mono cbx-text-main">正在解析快照并还原全量配置...</span>
                </div>
                <div v-else class="space-y-1">
                  <div class="text-2xl">📁</div>
                  <div class="text-xs font-bold cbx-text-main">点击选择或拖拽 .json 快照文件</div>
                  <div class="text-[10px] cbx-text-muted font-mono">支持 CubeX 导出的全量备份文件 (*.json)</div>
                </div>
              </div>

              <!-- Import Success Notice & Action -->
              <div v-else class="p-3 rounded-xl cbx-badge-ok border space-y-1.5 animate-in zoom-in-95">
                <div class="flex items-center gap-1.5 text-xs font-bold">
                  <span>✓</span>
                  <span>快照已成功恢复装载！</span>
                </div>
                <div class="text-[11px] font-mono cbx-text-main space-y-0.5">
                  <div>已装载网络订阅: <b>{{ app.initImportStats.sub_count }}</b> 条</div>
                  <div>已装载出站节点: <b>{{ app.initImportStats.proxy_count }}</b> 个</div>
                  <div class="text-[10px] cbx-text-sub truncate">文件: {{ app.initImportStats.filename }}</div>
                </div>
              </div>
            </div>

            <!-- Action buttons for Option 2 -->
            <div v-if="app.initImportSuccess" class="grid grid-cols-1 sm:grid-cols-2 gap-2 pt-1">
              <button @click="app.initWizardStep = 3; app.startInitStep3Execution()" class="py-2.5 px-3 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 text-white font-bold text-xs shadow-md shadow-sky-500/20 active:scale-95 transition flex items-center justify-center gap-1 cursor-pointer">
                <span>探测并同步宿主 ➔</span>
              </button>
              <button @click="app.initWizardStep = 4; app.launchFireworks()" class="py-2.5 px-3 rounded-xl cbx-btn-ghost font-bold text-xs active:scale-95 transition flex items-center justify-center gap-1 cursor-pointer">
                <span>跳过直接启航 🚀</span>
              </button>
            </div>
            <div v-else>
              <button @click="app.triggerInitBackupSelect" class="w-full py-2.5 px-4 rounded-xl bg-gradient-to-r from-purple-600 to-indigo-600 hover:from-purple-500 hover:to-indigo-500 text-white font-bold text-xs shadow-md shadow-purple-500/25 active:scale-95 transition flex items-center justify-center gap-1.5 cursor-pointer">
                <span>选择本地备份快照 ➔</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- STEP 2: 填入宿主连通凭据 -->
      <div v-else-if="app.initWizardStep === 2" class="space-y-4 animate-in fade-in relative z-10">
        <div class="p-4 rounded-2xl cbx-sub-card text-xs space-y-1">
          <div class="font-bold flex items-center gap-1.5 cbx-text-main">
            <span>💡</span>
            <span>第二步：建立 CubeX 与宿主底座的安全连通通道</span>
          </div>
          <p class="cbx-text-sub leading-relaxed">
            填入宿主管理地址与管理员凭据。对于 <b>MikroTik RouterOS</b>，账号密码完全指你平时登录 WinBox 客户端所用的账号（默认 admin）与登录密码（无密码直接留空）。
          </p>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5 text-xs font-mono">
          <div>
            <label class="block cbx-text-sub font-semibold mb-1">宿主管理地址 (Host IP)</label>
            <input v-model="app.initHostForm.host_ip" placeholder="请输入或自动探测主路由 IP"
              class="w-full px-3.5 py-2.5 rounded-xl outline-none shadow-inner font-mono" />
          </div>
          <div>
            <label class="block cbx-text-sub font-semibold mb-1">Web / REST 端口 (Port)</label>
            <input v-model="app.initHostForm.port" placeholder="ROS: 80/443, Linux: 22"
              class="w-full px-3.5 py-2.5 rounded-xl outline-none shadow-inner font-mono" />
          </div>
          <div>
            <label class="block cbx-text-sub font-semibold mb-1">管理员账号 (Username)</label>
            <input v-model="app.initHostForm.username" placeholder="例: admin"
              class="w-full px-3.5 py-2.5 rounded-xl outline-none shadow-inner font-mono" />
          </div>
          <div>
            <div class="flex items-center justify-between mb-1">
              <label class="block cbx-text-sub font-semibold">管理员密码 (Password)</label>
              <button type="button" @click="app.initHostForm.showPassword = !app.initHostForm.showPassword" class="text-[10px] text-sky-600 dark:text-cyan-300 font-bold hover:underline cursor-pointer">
                {{ app.initHostForm.showPassword ? '隐藏' : '显示' }}
              </button>
            </div>
            <input :type="app.initHostForm.showPassword ? 'text' : 'password'" v-model="app.initHostForm.password" placeholder="WinBox登录密码 (无密码留空)"
              class="w-full px-3.5 py-2.5 rounded-xl outline-none shadow-inner font-mono" />
          </div>
        </div>

        <div class="flex items-center justify-between gap-2 pt-4 border-t" style="border-color: var(--cbx-border-soft);">
          <button @click="app.initWizardStep = 1" class="px-4 py-2.5 rounded-xl cbx-btn-ghost text-xs font-semibold active:scale-95 transition cursor-pointer">
            ⬅ 返回模式选择
          </button>
          <button @click="app.startInitStep3Execution" class="px-5 py-2.5 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 text-white font-bold text-xs shadow-md shadow-sky-500/20 active:scale-95 transition flex items-center gap-1.5 cursor-pointer">
            <span>下一步：探测宿主并自动编排 ➔</span>
          </button>
        </div>
      </div>

      <!-- STEP 3: 系统判断宿主类型并自动执行两条 -->
      <div v-else-if="app.initWizardStep === 3" class="space-y-4 animate-in fade-in relative z-10">
        <div class="p-4 rounded-2xl cbx-sub-card flex items-center justify-between gap-2 flex-wrap">
          <div class="flex items-center gap-3">
            <span class="text-2xl">{{ app.initDetectedPlatform === 'routeros' ? '👑' : '🐧' }}</span>
            <div>
              <div class="text-xs cbx-text-sub">识别到的宿主环境平台</div>
              <div class="text-sm font-bold text-emerald-600 dark:text-emerald-300">
                {{ app.initDetectedPlatform === 'routeros' ? 'MikroTik RouterOS v7.x (官方硬件微核)' : 'Linux 宿主底座 (fnOS / PVE / Docker)' }}
              </div>
            </div>
          </div>
          <span :class="['text-xs font-mono px-2.5 py-1 rounded-full border', app.initExecuting ? 'cbx-badge-warn animate-pulse' : 'cbx-badge-ok']">
            {{ app.initExecuting ? '⚡ 正在自动编排...' : '✓ 编排就绪' }}
          </span>
        </div>

        <!-- Live Execution Terminal -->
        <div class="p-4 rounded-2xl cbx-terminal-box border space-y-2 font-mono text-xs max-h-56 overflow-y-auto shadow-inner">
          <div class="text-[10px] text-slate-400 uppercase tracking-widest border-b border-slate-800 pb-1 flex justify-between">
            <span>EXECUTION DISPATCHER</span>
            <span>AUTO-PILOT</span>
          </div>
          <div v-for="(log, idx) in app.initLogs" :key="idx" class="flex items-start gap-2 leading-relaxed">
            <span v-if="log.status === 'running'" class="inline-block animate-spin text-cyan-300 shrink-0">↻</span>
            <span v-else class="text-emerald-400 shrink-0">✓</span>
            <span class="text-slate-100">{{ log.text }}</span>
          </div>
        </div>

        <div class="p-3 rounded-xl cbx-sub-card text-[11px] cbx-text-sub flex items-center justify-between gap-2 flex-wrap">
          <span>正在执行：巡检隔离策略路由表 (table cubex)、下发 Mangle 防火墙规则与配置防 DNS 泄露...</span>
          <span v-if="app.initExecuting" class="cbx-badge-info border px-2 py-0.5 rounded font-mono animate-pulse">处理中...</span>
        </div>
      </div>

      <!-- STEP 4: 节点订阅导入与启航就绪 -->
      <div v-else-if="app.initWizardStep === 4" class="space-y-4 animate-in zoom-in-95 duration-500 relative z-30">
        <div class="flex items-center gap-3 p-3.5 rounded-2xl cbx-sub-card shadow-sm">
          <div class="w-10 h-10 rounded-2xl bg-gradient-to-tr from-sky-500 to-emerald-500 text-white flex items-center justify-center text-xl shadow-md shrink-0">
            🎉
          </div>
          <div class="flex-1 min-w-0">
            <h3 class="text-sm font-black cbx-text-main flex items-center gap-2 flex-wrap">
              <span>宿主底座与策略路由已原生打通！</span>
              <span class="text-[10px] px-2 py-0.5 rounded-full cbx-badge-ok border font-mono">
                ✓ READY
              </span>
            </h3>
            <p class="text-xs cbx-text-sub mt-0.5 truncate">
              最后一步：可直接粘贴订阅源或节点链接一键启航，也可跳过进入主控台后续再导。
            </p>
          </div>
        </div>

        <!-- Feature Readiness Matrix -->
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs font-mono text-left">
          <div class="p-2.5 rounded-xl cbx-sub-card space-y-0.5">
            <span class="text-[10px] text-sky-600 dark:text-cyan-300 font-bold">宿主平台</span>
            <div class="font-bold cbx-text-main truncate">{{ app.initDetectedPlatform === 'routeros' ? 'RouterOS v7' : 'Linux / fnOS' }}</div>
          </div>
          <div class="p-2.5 rounded-xl cbx-sub-card space-y-0.5">
            <span class="text-[10px] text-emerald-600 dark:text-emerald-400 font-bold">策略路由表</span>
            <div class="font-bold cbx-text-main">table cubex 就绪</div>
          </div>
          <div class="p-2.5 rounded-xl cbx-sub-card space-y-0.5">
            <span class="text-[10px] text-purple-600 dark:text-purple-400 font-bold">防 DNS 泄露</span>
            <div class="font-bold cbx-text-main">217heidai 权威规则</div>
          </div>
          <div class="p-2.5 rounded-xl cbx-sub-card space-y-0.5">
            <span class="text-[10px] text-amber-600 dark:text-amber-400 font-bold">双擎混动</span>
            <div class="font-bold cbx-text-main">Mihomo / sing-box</div>
          </div>
        </div>

        <!-- Embedded Batch Multi-Import Card -->
        <div class="space-y-3 p-4 rounded-2xl cbx-sub-card shadow-inner">
          <!-- 2-Segment Switcher -->
          <div class="flex flex-wrap sm:flex-nowrap gap-1 p-1 cbx-sub-card rounded-xl text-xs font-mono">
            <button type="button" @click="app.initWizardImportTab = 'sub'" :class="['flex-1 py-1.5 px-2 rounded-lg font-bold transition flex items-center justify-center gap-1.5 cursor-pointer', app.initWizardImportTab === 'sub' ? 'bg-gradient-to-r from-sky-600 to-cyan-500 text-white shadow-md' : 'cbx-text-sub hover:cbx-text-main']">
              <span>🌐 批量导入订阅</span>
              <span v-if="app.parsedBatchSubs.length > 0" class="px-1.5 py-0.2 rounded-full text-[10px] cbx-badge-ok font-mono">
                {{ app.parsedBatchSubs.length }}
              </span>
            </button>
            <button type="button" @click="app.initWizardImportTab = 'node'" :class="['flex-1 py-1.5 px-2 rounded-lg font-bold transition flex items-center justify-center gap-1.5 cursor-pointer', app.initWizardImportTab === 'node' ? 'bg-gradient-to-r from-sky-600 to-cyan-500 text-white shadow-md' : 'cbx-text-sub hover:cbx-text-main']">
              <span>🔗 批量导入自建节点</span>
              <span v-if="app.parsedBatchNodes.length > 0" class="px-1.5 py-0.2 rounded-full text-[10px] cbx-badge-purple font-mono">
                {{ app.parsedBatchNodes.length }}
              </span>
            </button>
          </div>

          <!-- Tab 1: Subscriptions -->
          <div v-if="app.initWizardImportTab === 'sub'" class="space-y-2">
            <div class="flex items-center justify-between gap-2 flex-wrap text-xs font-semibold cbx-text-sub">
              <span class="text-[11px]">支持 Clash / sing-box / V2Ray 订阅链接，每行一条（支持直接指定名称）：</span>
              <span v-if="app.parsedBatchSubs.length > 0" class="cbx-badge-ok border px-2 py-0.5 rounded font-mono text-[11px] font-bold">
                ⚡ 实时识别: {{ app.parsedBatchSubs.length }} 条订阅
              </span>
            </div>
            <textarea v-model="app.batchSubText" rows="3"
              placeholder="直接粘贴订阅源链接，每行一条。例如：&#10;香港主力专线, https://example.com/api/v1/client/subscribe?token=abc&#10;备用海外节点 | https://example.com/clash/token123"
              class="w-full rounded-xl p-2.5 text-xs font-mono outline-none shadow-inner resize-y leading-relaxed" />
            <div v-if="app.parsedBatchSubs.length > 0" class="max-h-24 overflow-y-auto space-y-1 font-mono text-[11px] cbx-sub-card p-2 rounded-xl">
              <div v-for="(item, idx) in app.parsedBatchSubs" :key="idx" class="flex items-center gap-2 truncate">
                <span class="text-emerald-600 dark:text-emerald-300 font-bold shrink-0">✈️ {{ item.name }}</span>
                <span class="cbx-text-muted truncate text-[10px]">{{ item.url }}</span>
              </div>
            </div>
          </div>

          <!-- Tab 2: Custom Nodes -->
          <div v-else class="space-y-2">
            <div class="flex items-center justify-between gap-2 flex-wrap text-xs font-semibold cbx-text-sub">
              <span class="text-[11px]">支持 ss, vmess, vless, trojan, hysteria2, tuic 节点链接，每行一条：</span>
              <span v-if="app.parsedBatchNodes.length > 0" class="cbx-badge-purple border px-2 py-0.5 rounded font-mono text-[11px] font-bold">
                ⚡ 实时识别: {{ app.parsedBatchNodes.length }} 个节点
              </span>
            </div>
            <textarea v-model="app.batchNodeText" rows="3"
              placeholder="直接粘贴节点链接，每行一条。例如：&#10;节点A, 协议://凭据@域名:端口#别名&#10;协议://凭据@域名:端口#节点B"
              class="w-full rounded-xl p-2.5 text-xs font-mono outline-none shadow-inner resize-y leading-relaxed" />
            <div v-if="app.parsedBatchNodes.length > 0" class="max-h-24 overflow-y-auto space-y-1 font-mono text-[11px] cbx-sub-card p-2 rounded-xl">
              <div v-for="(item, idx) in app.parsedBatchNodes" :key="idx" class="flex items-center gap-2 truncate">
                <span class="cbx-badge-purple border px-1.5 rounded font-bold uppercase text-[10px]">[{{ item.protocol }}]</span>
                <span class="cbx-text-main font-bold">🔗 {{ item.name }}</span>
                <span class="cbx-text-muted truncate text-[10px]">{{ item.link }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Bottom Actions -->
        <div class="pt-3 border-t flex flex-col sm:flex-row items-center justify-between gap-3" style="border-color: var(--cbx-border-soft);">
          <button @click="app.completeInitWizard" class="w-full sm:w-auto px-4 py-2.5 rounded-xl cbx-btn-ghost text-xs font-semibold active:scale-95 transition cursor-pointer">
            跳过直接进入主控台 ➔
          </button>
          <button v-if="(app.initWizardImportTab === 'sub' && app.parsedBatchSubs.length > 0) || (app.initWizardImportTab === 'node' && app.parsedBatchNodes.length > 0)"
            @click="app.submitInitWizardImportAndLaunch"
            :disabled="app.batchImportingSubs || app.batchAddingNodes"
            class="w-full sm:w-auto px-6 py-2.5 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white font-black text-xs shadow-lg shadow-sky-500/25 active:scale-95 transition flex items-center justify-center gap-1.5 disabled:opacity-50 cursor-pointer">
            <span v-if="app.batchImportingSubs || app.batchAddingNodes" class="inline-block animate-spin">↻</span>
            <span>{{ app.batchImportingSubs || app.batchAddingNodes ? '正在导入并重载核心...' : '🚀 立即导入并启航' }}</span>
          </button>
          <button v-else
            @click="app.completeInitWizard"
            class="w-full sm:w-auto px-6 py-2.5 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white font-black text-xs shadow-lg shadow-sky-500/25 active:scale-95 transition flex items-center justify-center gap-1.5 cursor-pointer">
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
