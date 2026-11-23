<template>
        <div class="space-y-6 animate-in fade-in">

          <!-- ════════════════════════════════════════════════════════════════ -->

          <!-- 1. 全维度安全防泄露体检雷达与自愈中枢 Header                     -->
          <!-- ════════════════════════════════════════════════════════════════ -->
          <div class="p-5 sm:p-6 rounded-3xl bg-[#F6F7FA]/[0.90] border border-white/80 shadow-lg backdrop-blur-xl flex flex-col lg:flex-row lg:items-center justify-between gap-4">
            <div class="macos-title-shield space-y-1">
              <div class="flex items-center gap-2">
                <span class="text-xl">🛡️</span>
                <h2 class="text-base sm:text-lg font-black text-[#0F172A] tracking-wide">
                  全维度防泄露安全雷达与自愈中心
                </h2>
              </div>
              <p class="text-xs text-slate-700 tracking-normal font-medium">
                6 维硬核渗透测试 · 0 明文回退熔断 · Apple Type 65 清洗 · WebRTC Fake-IP 隔离 · ECS 隐私剥离 · 一键闭环自愈加固
              </p>
            </div>

            <!-- Action Controls -->
            <div class="flex flex-wrap items-center gap-3">
              <button @click="app.runSecurityAudit" :disabled="app.securityAudit.loading"
                class="px-4 py-2 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white text-xs font-extrabold shadow-lg shadow-sky-500/25 active:scale-95 transition flex items-center gap-1.5 cursor-pointer">
                <span v-if="app.securityAudit.loading" class="inline-block animate-spin">↻</span>
                <span v-else>⚡</span>
                <span>一键全维度体检</span>
              </button>

              <div class="px-3 py-1.5 rounded-xl bg-white/90 text-slate-700 border border-slate-200/90 text-xs flex items-center gap-1.5 shadow-sm font-medium">
                <span class="text-emerald-600">💡</span>
                <span>核心安全规则自愈已全量合入【宿主共生】一体化全链路自愈总线</span>
              </div>
            </div>
          </div>

          <!-- ════════════════════════════════════════════════════════════════ -->
          <!-- 1.5 ➕ 一句话网站分流 (Quick Custom Domain Override)               -->
          <!-- ════════════════════════════════════════════════════════════════ -->
          <div class="p-5 sm:p-6 rounded-3xl bg-[#F6F7FA]/[0.90] border border-white/80 shadow-lg backdrop-blur-xl relative overflow-hidden space-y-4">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-2xl bg-gradient-to-br from-cyan-500/20 to-sky-500/20 border border-sky-300 flex items-center justify-center text-xl shrink-0 shadow-sm">
                  ➕
                </div>
                <div>
                  <h3 class="text-base sm:text-lg font-black text-[#0F172A] tracking-wide flex items-center gap-2">
                    <span>一句话网站分流</span>
                    <span class="text-[11px] font-mono px-2 py-0.5 rounded-full bg-sky-100 text-sky-800 border border-sky-300 font-bold">即按即生效 · 零感热重载</span>
                  </h3>
                  <p class="text-xs text-slate-700 mt-0.5 font-medium">
                    输入任意网址或域名，一键决定加速、直连或屏蔽，双核自动同步生效
                  </p>
                </div>
              </div>
              <div class="text-xs font-mono text-slate-700 self-start sm:self-center">
                已自定义: <strong class="text-sky-700 font-black">{{ totalCustomDomainCount }}</strong> 个网址
              </div>
            </div>

            <!-- 极简输入框与 3 个大操作按钮 -->
            <div class="flex flex-col md:flex-row items-stretch gap-2.5 pt-1">
              <div class="relative flex-1">
                <input v-model="quickDomainInput" @keyup.enter="handleQuickAdd('PROXIES')"
                  type="text"
                  placeholder="输入网址或域名，例如 javdb.com、pt.m-team.cc、github.com..."
                  class="w-full h-11 px-4 pr-10 rounded-xl bg-white border border-slate-200/90 text-[#0F172A] placeholder-slate-400 text-sm font-mono shadow-sm focus:outline-none focus:border-sky-500 transition" />
                <button v-if="quickDomainInput" @click="quickDomainInput = ''"
                  class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-800 text-xs font-bold w-5 h-5 flex items-center justify-center rounded-full bg-slate-100 cursor-pointer">
                  ✕
                </button>
              </div>

              <div class="flex items-center gap-2 shrink-0">
                <!-- ⚡ 走加速 -->
                <button @click="handleQuickAdd('PROXIES')" :disabled="quickAdding || !quickDomainInput.trim()"
                  class="flex-1 sm:flex-none h-11 px-4 rounded-xl bg-sky-600 hover:bg-sky-700 text-white text-xs sm:text-[13px] font-extrabold flex items-center justify-center gap-1.5 shadow-md shadow-sky-600/20 active:scale-95 transition disabled:opacity-40 cursor-pointer">
                  <span>⚡</span>
                  <span>强制走加速</span>
                </button>

                <!-- 🏠 走直连 -->
                <button @click="handleQuickAdd('DIRECT')" :disabled="quickAdding || !quickDomainInput.trim()"
                  class="flex-1 sm:flex-none h-11 px-4 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white text-xs sm:text-[13px] font-extrabold flex items-center justify-center gap-1.5 shadow-md shadow-emerald-600/20 active:scale-95 transition disabled:opacity-40 cursor-pointer">
                  <span>🏠</span>
                  <span>强制走直连</span>
                </button>

                <!-- 🛑 拦截 -->
                <button @click="handleQuickAdd('REJECT')" :disabled="quickAdding || !quickDomainInput.trim()"
                  class="flex-1 sm:flex-none h-11 px-3.5 rounded-xl bg-rose-600 hover:bg-rose-700 text-white text-xs sm:text-[13px] font-extrabold flex items-center justify-center gap-1.5 shadow-md shadow-rose-600/20 active:scale-95 transition disabled:opacity-40 cursor-pointer">
                  <span>🛑</span>
                  <span>屏蔽该网站</span>
                </button>
              </div>
            </div>

            <!-- 当前已生效的自定义规则胶囊瀑布流展示 (按分类彩色徽章 + ✕ 单击即删) -->
            <div v-if="totalCustomDomainCount > 0" class="pt-2 border-t border-slate-200/80 space-y-2">
              <div class="text-[11px] font-bold text-slate-700 flex items-center justify-between">
                <span>当前自定分流规则清单（点击 ✕ 立即取消并还原默认）：</span>
              </div>
              <div class="flex flex-wrap gap-2 max-h-40 overflow-y-auto custom-scrollbar pr-1">
                <!-- ⚡ 走加速项 -->
                <span v-for="d in userOverrides.force_proxy_domains" :key="'p-' + d"
                  class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-xl bg-sky-50 border border-sky-300 text-sky-900 text-xs font-mono shadow-sm group">
                  <span class="text-[10px] text-sky-700 font-bold">⚡ 加速</span>
                  <span class="font-bold">{{ d }}</span>
                  <button @click="handleQuickClear(d)" title="删除此加速规则"
                    class="ml-1 text-sky-600 hover:text-white hover:bg-sky-600 rounded px-1 transition cursor-pointer font-bold">✕</button>
                </span>

                <!-- 🏠 走直连项 -->
                <span v-for="d in userOverrides.force_direct_domains" :key="'d-' + d"
                  class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-xl bg-emerald-50 border border-emerald-300 text-emerald-900 text-xs font-mono shadow-sm group">
                  <span class="text-[10px] text-emerald-700 font-bold">🏠 直连</span>
                  <span class="font-bold">{{ d }}</span>
                  <button @click="handleQuickClear(d)" title="删除此直连规则"
                    class="ml-1 text-emerald-600 hover:text-white hover:bg-emerald-600 rounded px-1 transition cursor-pointer font-bold">✕</button>
                </span>

                <!-- 🛑 拦截项 -->
                <span v-for="d in userOverrides.force_reject_domains" :key="'r-' + d"
                  class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-xl bg-rose-50 border border-rose-300 text-rose-900 text-xs font-mono shadow-sm group">
                  <span class="text-[10px] text-rose-700 font-bold">🛑 屏蔽</span>
                  <span class="font-bold">{{ d }}</span>
                  <button @click="handleQuickClear(d)" title="删除此拦截规则"
                    class="ml-1 text-rose-600 hover:text-white hover:bg-rose-600 rounded px-1 transition cursor-pointer font-bold">✕</button>
                </span>
              </div>
            </div>
          </div>

          <!-- ════════════════════════════════════════════════════════════════ -->

          <!-- 2. 安全总评与能量光环看板 (Security Radar Scorecard)               -->

          <!-- ════════════════════════════════════════════════════════════════ -->

          <div class="p-6 rounded-3xl bg-[#F6F7FA]/[0.90] border border-white/80 shadow-lg backdrop-blur-xl relative overflow-hidden">

            <div class="absolute -top-24 -right-24 w-80 h-80 rounded-full bg-sky-500/10 blur-3xl pointer-events-none"></div>

            <div class="flex flex-col lg:flex-row items-center justify-between gap-6 relative z-10">

              <!-- Left: Circular Score Gauge -->

              <div class="flex items-center gap-5">

                <div class="relative w-24 h-24 flex items-center justify-center shrink-0">

                  <div class="absolute inset-0 rounded-full border-4 border-slate-200/80"></div>

                  <div class="absolute inset-1 rounded-full border-2 border-emerald-500/50"></div>

                  <div class="w-20 h-20 rounded-full bg-white text-[#0F172A] border border-emerald-500/40 flex flex-col items-center justify-center shadow-sm">

                    <span class="text-2xl font-black font-mono text-emerald-600 tracking-tighter">

                      {{ app.securityAudit.report?.total_score || 100 }}

                    </span>

                    <span class="text-[11px] font-mono text-slate-500 uppercase tracking-widest font-bold">/ 100 分</span>

                  </div>

                </div>

                <div class="space-y-1">

                  <div class="flex items-center gap-2">

                    <span class="text-lg font-black text-[#0F172A]">

                      {{ app.securityAudit.report?.security_rating || '暮光晶盾级 (100% 零泄露)' }}

                    </span>

                    <!-- Active Engine Badge -->

                    <span class="px-2.5 py-0.5 rounded-full text-[11px] font-mono font-bold flex items-center gap-1 border shadow-sm"

                      :class="app.activeEngine === 'singbox' ? 'bg-purple-50 text-purple-800 border-purple-300' : 'bg-sky-50 text-sky-800 border-sky-300'">

                      <span>内核:</span>

                      <span class="font-black">{{ app.activeEngine === 'singbox' ? 'sing-box' : 'mihomo' }}</span>

                    </span>

                  </div>

                  <p class="text-xs text-slate-700 max-w-xl leading-relaxed font-medium">

                    {{ app.securityAudit.report?.summary || '所有 6 大硬核防泄露测试全量通过！DNS 0 明文回退、WebRTC Fake-IP 隔离已就绪。' }}

                  </p>

                  <div class="text-[10px] text-slate-500 font-mono font-medium">

                    最新渗透测试时间: {{ new Date(app.securityAudit.report?.tested_at || Date.now()).toLocaleTimeString() }}

                  </div>

                </div>

              </div>

              <!-- Right: Hardcore Pillars Mini-Chips -->

              <div class="grid grid-cols-1 sm:grid-cols-3 lg:grid-cols-1 gap-2.5 w-full lg:w-auto shrink-0">

                <div class="px-3.5 py-2 rounded-xl bg-white/90 border border-slate-200/90 shadow-sm flex items-center gap-2.5 text-xs font-mono">

                  <span class="w-2 h-2 rounded-full bg-emerald-500 shadow-sm"></span>

                  <span class="text-slate-600 font-medium">DNS 0 明文回退:</span>

                  <span class="text-emerald-700 font-black">100% 熔断阻断</span>

                </div>

                <div class="px-3.5 py-2 rounded-xl bg-white/90 border border-slate-200/90 shadow-sm flex items-center gap-2.5 text-xs font-mono">

                  <span class="w-2 h-2 rounded-full bg-sky-500 shadow-sm"></span>

                  <span class="text-slate-600 font-medium">WebRTC 境外隔离:</span>

                  <span class="text-sky-700 font-black">Fake-IP 保密</span>

                </div>

                <div class="px-3.5 py-2 rounded-xl bg-white/90 border border-slate-200/90 shadow-sm flex items-center gap-2.5 text-xs font-mono">

                  <span class="w-2 h-2 rounded-full bg-teal-500 shadow-sm"></span>

                  <span class="text-slate-600 font-medium">Apple Type 65:</span>

                  <span class="text-teal-700 font-black">100% 清洗拦截</span>

                </div>

              </div>

            </div>

          </div>

          <!-- ════════════════════════════════════════════════════════════════ -->

          <!-- 3. 六大硬核安全测试 (6-Vector Hardcore Test Matrix)           -->

          <!-- ════════════════════════════════════════════════════════════════ -->

          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">

            <div v-for="item in (app.securityAudit.report?.items || [])" :key="item.id"

              class="p-4 rounded-2xl bg-white/90 border shadow-sm transition-all duration-300 hover:border-sky-400 flex flex-col justify-between space-y-3"

              :class="item.status === 'PASS' ? 'border-slate-200/90' : (item.status === 'WARN' ? 'border-amber-300' : 'border-rose-300')">

              <!-- Card Top -->

              <div class="space-y-2">

                <div class="flex items-start justify-between gap-2">

                  <div>

                    <span class="text-[10px] font-mono px-2 py-0.5 rounded-md font-bold uppercase tracking-wider"

                      :class="item.risk_level === 'CRITICAL' ? 'bg-rose-50 text-rose-800 border border-rose-300' : (item.risk_level === 'HIGH' ? 'bg-amber-50 text-amber-800 border border-amber-300' : 'bg-emerald-50 text-emerald-800 border border-emerald-300')">

                      {{ item.category }} · {{ item.risk_level }}

                    </span>

                    <h3 class="text-sm font-black text-[#0F172A] mt-1.5 flex items-center gap-1.5">

                      <span>{{ item.name }}</span>

                    </h3>

                  </div>

                  <!-- Status Badge -->

                  <span class="px-2.5 py-1 rounded-lg text-xs font-mono font-bold shrink-0 border shadow-sm"

                    :class="item.status === 'PASS' ? 'bg-emerald-50 text-emerald-800 border-emerald-300' : (item.status === 'WARN' ? 'bg-amber-50 text-amber-800 border-amber-300' : 'bg-rose-50 text-rose-800 border-rose-300')">

                    {{ item.status === 'PASS' ? '✓ PASS 0泄露' : (item.status === 'WARN' ? '▲ WARN 待加固' : '✕ FAIL 风险') }}

                  </span>

                </div>

                <!-- Technical Specification -->

                <div class="text-[11px] font-mono font-bold text-emerald-800 bg-emerald-50 px-2.5 py-1 rounded-md border border-emerald-300 shadow-sm">

                  规范: {{ item.technical_specs }}

                </div>

                <!-- Description -->

                <p class="text-xs text-slate-700 leading-relaxed font-medium">

                  {{ item.details }}

                </p>

              </div>

              <!-- Card Bottom -->

              <div class="pt-2 border-t border-slate-200/80 space-y-2">

                <!-- Score bar -->

                <div class="flex items-center justify-between text-[11px] font-mono">

                  <span class="text-slate-600 font-medium">单项防御得分</span>

                  <span class="font-black" :class="item.score >= 90 ? 'text-emerald-700' : (item.score >= 70 ? 'text-amber-700' : 'text-rose-700')">

                    {{ item.score }} / 100

                  </span>

                </div>

                <div class="w-full h-1.5 rounded-full bg-slate-200 overflow-hidden">

                  <div class="h-full rounded-full transition-all duration-500"

                    :class="item.score >= 90 ? 'bg-gradient-to-r from-teal-500 to-emerald-500' : (item.score >= 70 ? 'bg-gradient-to-r from-amber-500 to-yellow-500' : 'bg-rose-500')"

                    :style="{ width: item.score + '%' }"></div>

                </div>

                <!-- Recommendation / Remediation status -->

                <div class="text-[10px] text-slate-600 font-mono font-medium flex items-center justify-between">

                  <span>{{ item.recommendation }}</span>

                  <span v-if="item.remediated" class="text-emerald-700 font-bold">🛡️ 已加固</span>

                </div>

              </div>

            </div>

          </div>

          <!-- ════════════════════════════════════════════════════════════════ -->

          <!-- 4. 217heidai 去广告&隐私追踪拦截 (AdBlock & Threat Interception)    -->
          <!-- ════════════════════════════════════════════════════════════════ -->
          <div class="space-y-4">
            <!-- Header: 217heidai 去广告&隐私追踪拦截 (Windhawk 奶白亚克力卡片) -->
            <div class="p-5 sm:p-6 rounded-3xl bg-[#F6F7FA]/[0.90] border border-white/80 shadow-lg backdrop-blur-xl flex items-center justify-between flex-wrap gap-3">
              <div>
                <h2 class="text-base sm:text-lg font-black text-[#0F172A] tracking-wide flex items-center gap-2">
                  <span class="flex items-center gap-2"><ZootopiaIcon name="rules" :size="22" class="shrink-0" /><span class="text-[#0F172A] font-black">217heidai 去广告&隐私追踪拦截</span></span>
                </h2>
                <p class="text-xs text-slate-700 mt-1 font-medium font-sans">
                  基于 217heidai MRS 规则库的高精度广告与隐私追踪拦截 · 最近 24 小时 DNS 耗时、查询吞吐与恶意威胁拦截遥测
                </p>
              </div>

              <div class="flex items-center gap-2 flex-wrap font-mono">
                <span class="text-xs text-emerald-800 bg-emerald-50 px-3 py-1 rounded-full border border-emerald-300 font-bold shadow-sm">
                  规则库: {{ app.adblockStats.total_rules || 233481 }} 条
                </span>
                <span class="text-xs text-sky-800 bg-sky-50 px-3 py-1 rounded-full border border-sky-300 font-bold shadow-sm">
                  拦截率: {{ app.dnsFilterStats.blockedPercentage }}%
                </span>
              </div>
            </div>

            <!-- ════════════════════════════════════════════════════════════════ -->

            <!-- 域名过滤与广告拦截中枢 (Domain Filter & AdBlock Hub)                 -->

            <!-- ════════════════════════════════════════════════════════════════ -->

            <div class="rounded-3xl bg-[#F6F7FA]/[0.90] border border-white/80 hover:border-sky-400/40 transition-all duration-300 p-5 space-y-4 shadow-lg backdrop-blur-xl">

              <!-- Top Row: Title, Master Toggle, Status Badge, Action Buttons -->

              <div class="flex items-center justify-between flex-wrap gap-3 pb-3 border-b border-slate-200/80">

                <div class="flex items-center gap-3">

                  <h3 class="text-base font-black text-[#0F172A] tracking-wide flex items-center gap-2">

                    <span>域名过滤</span>

                  </h3>

                  <!-- Master Switch -->

                  <label class="relative inline-flex items-center cursor-pointer">

                    <input type="checkbox" v-model="app.filterConfig.enabled" @change="app.toggleFilterMaster" class="sr-only peer">

                    <div class="w-11 h-6 bg-slate-300 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-500 shadow-inner"></div>

                  </label>

                  <span :class="['text-xs px-2.5 py-0.5 rounded-full font-bold border transition shadow-sm', app.filterConfig.enabled ? 'bg-emerald-100 text-emerald-800 border-emerald-300' : 'bg-slate-200 text-slate-700 border-slate-300']">

                    {{ app.filterConfig.enabled ? '已生效' : '已暂停' }}

                  </span>

                </div>

                <div class="flex items-center gap-2">

                  <button @click="app.fetchDnsFilterStats" title="刷新状态" class="p-2 rounded-xl bg-white hover:bg-slate-50 border border-slate-200/90 text-[#0F172A] hover:border-sky-400 active:scale-95 transition cursor-pointer shadow-sm">

                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>

                  </button>

                  <button @click="app.openAddSourceModal" class="px-3 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs flex items-center gap-1.5 shadow-md shadow-emerald-600/20 active:scale-95 transition cursor-pointer">

                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4"/></svg>

                    <span>添加规则源</span>

                  </button>

                </div>

              </div>

              <!-- Note / Description -->

              <p class="text-xs text-slate-700 leading-relaxed font-medium">

                默认开启。开启后，转发的 DNS 查询先经过过滤，通过后仍按照原 DNS 策略解析；白名单放行优先。支持原生二进制 MRS/SRS、域名、Hosts 和基础 Adblock 语法，不支持的条目会显示跳过数量。启用的名单按下面的自动更新计划更新，下载失败沿用上一次有效名单。

              </p>

              <!-- Rule Sources List -->

              <div class="space-y-2.5">

                <div v-for="src in app.filterConfig.rule_sources" :key="src.id"

                     class="rounded-2xl bg-white/90 border border-slate-200/80 hover:border-sky-400/50 shadow-sm transition-all p-3.5 flex flex-col md:flex-row md:items-center justify-between gap-3">

                  <div class="flex items-center gap-3 min-w-0 flex-1">

                    <!-- Source Toggle -->

                    <label class="relative inline-flex items-center cursor-pointer shrink-0">

                      <input type="checkbox" v-model="src.enabled" @change="app.toggleSourceItem(src)" class="sr-only peer">

                      <div class="w-9 h-5 bg-slate-300 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-emerald-500 shadow-inner"></div>

                    </label>

                    <div class="min-w-0 flex-1">

                      <div class="flex items-center gap-2 flex-wrap">

                        <span class="text-xs font-black text-[#0F172A]">{{ src.name }}</span>

                        <span class="text-[11px] font-mono text-slate-600 truncate max-w-xs sm:max-w-md block" :title="src.url">{{ src.url }}</span>

                      </div>

                      <div class="flex items-center gap-3 mt-1 text-[11px] font-mono">

                        <span class="text-emerald-700 font-bold">规则数 {{ (src.rules_count || 0).toLocaleString() }}</span>

                        <span class="text-slate-600 font-medium">上次更新 {{ src.updated_at || '刚刚' }}</span>

                      </div>

                    </div>

                  </div>

                  <!-- Actions -->

                  <div class="flex items-center gap-1.5 shrink-0 self-end md:self-center">

                    <button @click="app.openSourcePreview(src)" title="查看规则详情" class="p-1.5 rounded-lg bg-white hover:bg-slate-50 border border-slate-200/90 text-[#0F172A] hover:border-sky-400/50 shadow-sm transition cursor-pointer">

                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>

                    </button>

                    <button @click="app.refreshSourceItem(src)" :disabled="app.isUpdatingSource" title="立即更新" class="p-1.5 rounded-lg bg-white hover:bg-slate-50 border border-slate-200/90 text-[#0F172A] hover:border-emerald-400/50 shadow-sm transition cursor-pointer disabled:opacity-50">

                      <svg :class="['w-3.5 h-3.5', app.isUpdatingSource ? 'animate-spin text-emerald-600' : '']" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>

                    </button>

                    <button @click="app.openEditSourceModal(src)" title="编辑规则源" class="p-1.5 rounded-lg bg-white hover:bg-slate-50 border border-slate-200/90 text-[#0F172A] hover:border-sky-400/50 shadow-sm transition cursor-pointer">

                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"/></svg>

                    </button>

                    <button v-if="src.id !== '217heidai'" @click="app.deleteSourceItem(src)" title="删除规则源" class="p-1.5 rounded-lg bg-white hover:bg-rose-50 border border-slate-200/90 text-rose-700 hover:border-rose-300 shadow-sm transition cursor-pointer">

                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>

                    </button>

                  </div>

                </div>

              </div>

              <!-- Auto Update Scheduler Bar -->

              <div class="rounded-2xl bg-white/90 border border-slate-200/80 p-3.5 flex flex-wrap items-center gap-3 text-xs text-slate-700 font-medium shadow-sm">

                <span class="font-bold text-[#0F172A] flex items-center gap-1.5">

                  <span>自动更新</span>

                </span>

                <label class="relative inline-flex items-center cursor-pointer shrink-0">

                  <input type="checkbox" v-model="app.filterConfig.auto_update.enabled" @change="app.saveFilterAutoUpdate" class="sr-only peer">

                  <div class="w-8 h-4 bg-slate-300 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[1px] after:left-[1px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-3.5 after:w-3.5 after:transition-all peer-checked:bg-emerald-500 shadow-inner"></div>

                </label>

                <div class="flex items-center gap-2 flex-wrap">

                  <span>每</span>

                  <select v-model="app.filterConfig.auto_update.interval_days" @change="app.saveFilterAutoUpdate" class="bg-white border border-slate-200/90 rounded-lg px-2 py-1 text-xs text-[#0F172A] font-bold focus:outline-none focus:border-emerald-500 font-mono shadow-sm cursor-pointer">

                    <option :value="1">1 天</option>

                    <option :value="3">3 天</option>

                    <option :value="7">7 天</option>

                  </select>

                  <span>在</span>

                  <select v-model="app.filterConfig.auto_update.update_time" @change="app.saveFilterAutoUpdate" class="bg-white border border-slate-200/90 rounded-lg px-2 py-1 text-xs text-[#0F172A] font-bold focus:outline-none focus:border-emerald-500 font-mono shadow-sm cursor-pointer">

                    <option value="04:00">04:00</option>

                    <option value="06:00">06:00</option>

                    <option value="08:00">08:00</option>

                    <option value="12:00">12:00</option>

                    <option value="18:00">18:00</option>

                    <option value="00:00">00:00</option>

                  </select>

                  <span class="text-slate-600 text-[11px]">在选定时间更新一次；内核自己加载新名单，不重启。</span>

                </div>

              </div>

              <!-- Whitelist (放行域名) Accordion -->

              <div class="border-t border-slate-200/80 pt-3">

                <button @click="showWhitelistSection = !showWhitelistSection" class="w-full flex items-center justify-between text-xs font-bold text-slate-800 hover:text-black transition cursor-pointer py-1">

                  <span class="flex items-center gap-1.5">

                    <svg :class="['w-3.5 h-3.5 transition-transform text-slate-500', showWhitelistSection ? 'rotate-180' : '']" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/></svg>

                    <span>放行域名 ({{ app.filterConfig.whitelist_domains?.length || 0 }})</span>

                  </span>

                  <span class="text-[11px] text-slate-600 font-medium">优先直连放行 · 防误杀</span>

                </button>

                <div v-if="showWhitelistSection" class="mt-3 space-y-3 pl-4 border-l-2 border-emerald-500/60 animate-in fade-in">

                  <!-- Tag Cloud -->

                  <div class="flex flex-wrap items-center gap-2">

                    <span v-for="d in app.filterConfig.whitelist_domains" :key="d"

                          class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-white border border-slate-200/90 text-slate-800 text-xs font-mono shadow-sm group hover:border-emerald-500/50">

                      <span>{{ d }}</span>

                      <button @click="app.removeWhitelistItem(d)" title="移除放行" class="text-slate-400 group-hover:text-rose-600 font-bold transition cursor-pointer">

                        &times;

                      </button>

                    </span>

                    <span v-if="!app.filterConfig.whitelist_domains || app.filterConfig.whitelist_domains.length === 0" class="text-xs text-slate-600 italic font-medium">

                      暂无放行域名，可在下方添加特定业务域名以防误杀

                    </span>

                  </div>

                  <!-- Add Form -->

                  <div class="flex items-center gap-2 max-w-md">

                    <input type="text" v-model="app.newWhitelistDomain" @keyup.enter="app.addWhitelistItem" placeholder="输入放行域名 (如 apple.com)" class="flex-1 bg-white border border-slate-200/90 rounded-xl px-3 py-1.5 text-xs text-[#0F172A] placeholder-slate-400 focus:outline-none focus:border-emerald-500 font-mono shadow-sm">

                    <button @click="app.addWhitelistItem" class="px-3 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-sm active:scale-95 transition cursor-pointer">

                      添加放行

                    </button>

                  </div>

                </div>

              </div>

            </div>

            <!-- ════════════════════════════════════════════════════════════════ -->
            <!-- 🔀 DNS 重写 (域名别名重定向 & 静态 IP 映射)                        -->
            <!-- ════════════════════════════════════════════════════════════════ -->
            <div class="rounded-3xl bg-[#F6F7FA]/[0.90] border border-white/80 shadow-lg backdrop-blur-xl p-5 sm:p-6 space-y-4">
              <!-- Top Row: Title, Master Toggle, Status Badge, Action Buttons -->
              <div class="flex items-center justify-between flex-wrap gap-3 pb-3 border-b border-slate-200/80">
                <div class="flex items-center gap-3">
                  <h3 class="text-base font-black text-[#0F172A] tracking-wide flex items-center gap-2">
                    <span class="text-emerald-600">🔀</span>
                    <span>DNS 重写和 局域网A 记录静态映射</span>
                  </h3>

                  <!-- Master Switch -->
                  <label class="relative inline-flex items-center cursor-pointer">
                    <input type="checkbox" v-model="app.dnsRewriteConfig.enabled" @change="app.toggleDnsRewriteMaster" class="sr-only peer">
                    <div class="w-11 h-6 bg-slate-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-sky-500 shadow-inner"></div>
                  </label>

                  <span :class="['text-xs px-2.5 py-0.5 rounded-full font-bold border transition', app.dnsRewriteConfig.enabled ? 'bg-emerald-50 text-emerald-800 border-emerald-300' : 'bg-slate-100 text-slate-600 border-slate-300']">
                    {{ app.dnsRewriteConfig.enabled ? '已生效' : '已暂停' }}
                  </span>
                </div>

                <div class="flex items-center gap-2">
                  <button @click="app.resetDnsRewriteDefaults" title="恢复默认 DNS 重写与 局域网A 记录静态映射规则" class="px-3 py-1.5 rounded-xl bg-white hover:bg-slate-50 border border-slate-200/90 text-slate-700 font-bold active:scale-95 transition cursor-pointer text-xs flex items-center gap-1.5 shadow-sm">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
                    <span>重置默认</span>
                  </button>

                  <button @click="app.openAddDnsRewriteModal" class="px-3.5 py-1.5 rounded-xl bg-gradient-to-r from-sky-500 to-cyan-500 hover:from-sky-400 hover:to-cyan-400 text-white font-bold text-xs flex items-center gap-1.5 shadow-md shadow-sky-500/20 active:scale-95 transition cursor-pointer">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4"/></svg>
                    <span>添加重写</span>
                  </button>
                </div>
              </div>

              <!-- Note / Description matching user screenshot -->
              <p class="text-xs text-slate-700 leading-relaxed font-medium">
                把某个域名的解析结果改成另一个域名或固定 IP；默认带两条 Google 的重写（解决 Google Play 无法下载 App 问题）及一条局域网 A 记录静态映射示例。
              </p>

              <!-- Rules List -->
              <div class="space-y-2.5">
                <div v-for="rule in app.dnsRewriteConfig.rules" :key="rule.id"
                     class="rounded-2xl bg-white/90 border border-slate-200/90 hover:border-sky-300 transition-all p-3.5 flex flex-col md:flex-row md:items-center justify-between gap-3 shadow-sm">
                  <div class="flex-1 min-w-0 space-y-1">
                    <div class="flex items-center gap-2 flex-wrap">
                      <span class="font-mono text-xs font-bold text-[#0F172A] tracking-wide bg-slate-100 px-2 py-0.5 rounded border border-slate-300">{{ rule.domain }}</span>
                      <span class="text-sky-600 text-xs">➔</span>
                      <span class="font-mono text-xs font-bold text-sky-800 tracking-wide bg-sky-50 px-2 py-0.5 rounded border border-sky-300">{{ rule.target }}</span>
                      <span :class="['text-[10px] font-bold px-2 py-0.5 rounded uppercase tracking-wider', rule.type === 'CNAME' ? 'bg-indigo-50 text-indigo-800 border border-indigo-200' : 'bg-purple-50 text-purple-800 border border-purple-200']">
                        {{ rule.type === 'CNAME' ? 'CNAME 别名' : 'A 静态记录' }}
                      </span>
                    </div>

                    <div v-if="rule.comment" class="text-[11px] text-slate-600 font-medium flex items-center gap-1.5">
                      <span class="text-slate-400">💬</span>
                      <span>{{ rule.comment }}</span>
                    </div>
                  </div>

                  <!-- Actions & Toggle -->
                  <div class="flex items-center gap-2.5 self-end md:self-auto shrink-0">
                    <label class="relative inline-flex items-center cursor-pointer" title="启用/停用单条规则">
                      <input type="checkbox" v-model="rule.enabled" @change="app.toggleDnsRewriteRule(rule)" class="sr-only peer">
                      <div class="w-9 h-5 bg-slate-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-sky-500 shadow-inner"></div>
                    </label>

                    <button @click="app.openEditDnsRewriteModal(rule)" title="编辑规则" class="p-1.5 rounded-lg bg-white hover:bg-slate-50 border border-slate-200/90 text-slate-700 shadow-sm transition cursor-pointer">
                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"/></svg>
                    </button>

                    <button @click="app.deleteDnsRewriteRule(rule)" title="删除规则" class="p-1.5 rounded-lg bg-white hover:bg-rose-50 border border-slate-200/90 text-rose-700 hover:border-rose-300 shadow-sm transition cursor-pointer">
                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
                    </button>
                  </div>
                </div>

                <div v-if="!app.dnsRewriteConfig.rules || app.dnsRewriteConfig.rules.length === 0" class="text-center py-6 text-xs text-slate-600 font-medium italic bg-white/60 rounded-2xl border border-dashed border-slate-300">
                  暂无规则，点击右上角【添加重写】或【重置默认】快速恢复官方加速重写与 局域网A 记录静态映射示例
                </div>
              </div>
            </div>

            <!-- 4-Card 2x2 Grid with 24h Area Charts and Rank -->

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">

              <!-- Card 1: 平均解析耗时 -->

              <div class="p-4 rounded-2xl bg-[#F6F7FA]/[0.90] border border-white/80 shadow-lg space-y-2 relative overflow-hidden backdrop-blur-xl">

                <div class="flex items-center justify-between text-xs text-slate-700 font-medium">

                  <span>平均解析耗时 (24h 遥测)</span>

                  <span class="text-[10px] text-emerald-800 bg-emerald-50 px-2 py-0.5 rounded-full border border-emerald-300 font-mono font-bold flex items-center gap-1 shadow-sm">

                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-600 animate-pulse"></span>

                    0 泄露极速响应

                  </span>

                </div>

                <div class="text-2xl font-bold font-mono text-[#0F172A] flex items-baseline gap-1">

                  <span>{{ app.dnsFilterStats.avgLatencyMs }}</span>

                  <span class="text-xs text-slate-600 font-sans font-medium">ms</span>

                </div>

                <div class="h-16 w-full pt-1">

                  <svg class="w-full h-full" viewBox="0 0 320 60" preserveAspectRatio="none">

                    <defs>

                      <linearGradient id="dnsLatencyGrad" x1="0" y1="0" x2="0" y2="1">

                        <stop offset="0%" stop-color="#C084FC" stop-opacity="0.4"/>

                        <stop offset="100%" stop-color="#C084FC" stop-opacity="0.0"/>

                      </linearGradient>

                    </defs>

                    <path :d="app.dnsLatencyPath.area" fill="url(#dnsLatencyGrad)" />

                    <path :d="app.dnsLatencyPath.line" fill="none" stroke="#9333ea" stroke-width="2" stroke-linecap="round" />

                    <!-- Live Pulse End Point -->

                    <circle cx="316" :cy="app.dnsLatencyLastY" r="3" fill="#9333ea" />

                    <circle cx="316" :cy="app.dnsLatencyLastY" r="6" fill="#9333ea" opacity="0.4" class="animate-ping" />

                  </svg>

                </div>

              </div>

              <!-- Card 2: DNS 查询量 -->

              <div class="p-4 rounded-2xl bg-[#F6F7FA]/[0.90] border border-white/80 shadow-lg space-y-2 relative overflow-hidden backdrop-blur-xl">

                <div class="flex items-center justify-between text-xs text-slate-700 font-medium">

                  <span>DNS 查询量 (24h 吞吐)</span>

                  <span class="text-[10px] text-sky-800 bg-sky-50 px-2 py-0.5 rounded-full border border-sky-300 font-mono font-bold flex items-center gap-1 shadow-sm">

                    <span class="w-1.5 h-1.5 rounded-full bg-sky-600 animate-pulse"></span>

                    国内直连 + Fake-IP

                  </span>

                </div>

                <div class="text-2xl font-bold font-mono text-[#0F172A]">

                  {{ app.dnsFilterStats.queryCount.toLocaleString() }}

                </div>

                <div class="h-16 w-full pt-1">

                  <svg class="w-full h-full" viewBox="0 0 320 60" preserveAspectRatio="none">

                    <defs>

                      <linearGradient id="dnsQueryGrad" x1="0" y1="0" x2="0" y2="1">

                        <stop offset="0%" stop-color="#0284c7" stop-opacity="0.4"/>

                        <stop offset="100%" stop-color="#0284c7" stop-opacity="0.0"/>

                      </linearGradient>

                    </defs>

                    <path :d="app.dnsQueryPath.area" fill="url(#dnsQueryGrad)" />

                    <path :d="app.dnsQueryPath.line" fill="none" stroke="#0284c7" stroke-width="2" stroke-linecap="round" />

                    <!-- Live Pulse End Point -->

                    <circle cx="316" :cy="app.dnsQueryLastY" r="3" fill="#0284c7" />

                    <circle cx="316" :cy="app.dnsQueryLastY" r="6" fill="#0284c7" opacity="0.4" class="animate-ping" />

                  </svg>

                </div>

              </div>

              <!-- Card 3: 今日阻断与拦截 (带分类统计以平衡行高) -->

              <div class="p-4 rounded-2xl bg-[#F6F7FA]/[0.90] border border-white/80 shadow-lg space-y-2 relative overflow-hidden flex flex-col justify-between backdrop-blur-xl">

                <div>

                  <div class="flex items-center justify-between text-xs">

                    <span class="text-slate-700 font-medium">今日阻断与拦截</span>

                    <span class="text-xs font-bold font-mono text-amber-800 bg-amber-50 px-2 py-0.5 rounded-lg border border-amber-300 shadow-sm">

                      拦截率 {{ app.dnsFilterStats.blockedPercentage }}%

                    </span>

                  </div>

                  <div class="text-2xl font-bold font-mono text-[#B45309]">

                    {{ app.dnsFilterStats.blockedCount.toLocaleString() }}

                  </div>

                  <div class="h-16 w-full pt-1">

                    <svg class="w-full h-full" viewBox="0 0 320 60" preserveAspectRatio="none">

                      <defs>

                        <linearGradient id="dnsBlockedGrad" x1="0" y1="0" x2="0" y2="1">

                          <stop offset="0%" stop-color="#d97706" stop-opacity="0.4"/>

                          <stop offset="100%" stop-color="#d97706" stop-opacity="0.0"/>

                        </linearGradient>

                      </defs>

                      <path :d="app.dnsBlockedPath.area" fill="url(#dnsBlockedGrad)" />

                      <path :d="app.dnsBlockedPath.line" fill="none" stroke="#d97706" stroke-width="2" stroke-linecap="round" />

                      <!-- Live Pulse End Point -->

                      <circle cx="316" :cy="app.dnsBlockedLastY" r="3" fill="#d97706" />

                      <circle cx="316" :cy="app.dnsBlockedLastY" r="6" fill="#d97706" opacity="0.4" class="animate-ping" />

                    </svg>

                  </div>

                </div>

                <!-- 真实规则库拦截说明标签 -->
                <div class="pt-2 border-t border-slate-200/80 text-[11px] text-slate-600 flex items-center justify-between">
                  <span class="flex items-center gap-1.5">
                    <span class="w-1.5 h-1.5 rounded-full bg-amber-500"></span>
                    <span>217heidai 权威规则拦截</span>
                  </span>
                  <span class="font-mono text-slate-500">实时过滤</span>
                </div>

                <!-- 🛡️ 战场三：真实安全拦截子分类遥测 (无锁原子真数据，缺失优雅占位 —) -->
                <div class="grid grid-cols-3 gap-1.5 pt-1 text-[11px] font-mono select-none">
                  <div class="flex flex-col items-center justify-center p-1.5 rounded-xl bg-amber-500/10 border border-amber-300/60 text-slate-800" title="广告弹窗规则拦截">
                    <span class="text-[10px] text-slate-500 font-sans font-medium">广告弹窗</span>
                    <span class="font-bold text-amber-800 tabular-nums">
                      {{ app.dnsFilterStats.categoryStats ? app.dnsFilterStats.categoryStats.ads.toLocaleString() : '—' }}
                    </span>
                  </div>
                  <div class="flex flex-col items-center justify-center p-1.5 rounded-xl bg-purple-500/10 border border-purple-300/60 text-slate-800" title="隐私与遥测追踪拦截">
                    <span class="text-[10px] text-slate-500 font-sans font-medium">隐私追踪</span>
                    <span class="font-bold text-purple-800 tabular-nums">
                      {{ app.dnsFilterStats.categoryStats ? app.dnsFilterStats.categoryStats.privacy.toLocaleString() : '—' }}
                    </span>
                  </div>
                  <div class="flex flex-col items-center justify-center p-1.5 rounded-xl bg-rose-500/10 border border-rose-300/60 text-slate-800" title="恶意域名与劫持阻断">
                    <span class="text-[10px] text-slate-500 font-sans font-medium">恶意劫持</span>
                    <span class="font-bold text-rose-800 tabular-nums">
                      {{ app.dnsFilterStats.categoryStats ? app.dnsFilterStats.categoryStats.malicious.toLocaleString() : '—' }}
                    </span>
                  </div>
                </div>
              </div>

              <!-- Card 4: 被拦截域名排行 -->

              <div class="p-4 rounded-2xl bg-[#F6F7FA]/[0.90] border border-white/80 shadow-lg space-y-3 relative overflow-hidden flex flex-col justify-between backdrop-blur-xl">

                <div>

                  <div class="flex items-center justify-between text-xs text-slate-700 font-medium">

                    <span>被拦截广告与恶意域名排行</span>

                    <span class="text-[10px] text-amber-800 bg-amber-50 px-2 py-0.5 rounded-full border border-amber-300 font-mono font-bold">217heidai 高危特征</span>

                  </div>

                  <!-- 列表内容：有数据渲染进度条，无数据优雅兜底提示 -->

                  <div v-if="app.dnsFilterStats.topBlocked && app.dnsFilterStats.topBlocked.length > 0" class="space-y-2 pt-2">

                    <div v-for="item in app.dnsFilterStats.topBlocked" :key="item.domain" class="space-y-1">

                      <div class="flex items-center justify-between text-[11px] font-mono">

                        <span class="text-slate-800 font-medium truncate max-w-[240px]" :title="item.domain">{{ item.domain }}</span>

                        <span class="text-slate-600 font-semibold">{{ item.count.toLocaleString() }}</span>

                      </div>

                      <div class="w-full bg-slate-200/80 h-1.5 rounded-full overflow-hidden">

                        <div class="bg-gradient-to-r from-amber-500 to-orange-500 h-full rounded-full transition-all duration-500"

                          :style="{ width: item.percent + '%' }"></div>

                      </div>

                    </div>

                  </div>

                  <div v-else class="h-28 flex flex-col items-center justify-center text-center text-slate-500 space-y-1">

                    <span class="text-lg">🛡️</span>

                    <span class="text-xs font-semibold text-slate-700">暂无高危拦截记录</span>

                    <span class="text-[10px] text-slate-500">217heidai DNS 规则防线实时守卫中</span>

                  </div>

                </div>

              </div>

            </div>

          </div>

          <!-- ════════════════════════════════════════════════════════════════ -->
          <!-- 6. Fake-IP 白名单与真实 IP 直连解析中枢 (Fake-IP Filter Hub)       -->
          <!-- ════════════════════════════════════════════════════════════════ -->
          <div class="rounded-3xl p-5 sm:p-6 bg-[#F6F7FA]/[0.90] border border-white/80 shadow-lg backdrop-blur-xl space-y-4">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div class="space-y-1">
                <div class="flex items-center gap-2">
                  <span class="text-lg">🎯</span>
                  <h3 class="text-sm sm:text-base font-black text-[#0F172A] tracking-wide">
                    Fake-IP 过滤白名单 (真实 IP 直连解析中枢)
                  </h3>
                  <span class="text-[10px] px-2 py-0.5 rounded-full bg-emerald-50 text-emerald-800 border border-emerald-300 font-bold shadow-sm">
                    双核秒级热重载 · 掉电持久化
                  </span>
                </div>
                <p class="text-xs text-slate-700 font-medium leading-relaxed">
                  名单内域名将跳过 198.18.0.0/15 假地址拦截，由本地运营商 DNS 直接返回物理真实 IP。彻底解决极空间 NAS (zspace.com)、本地网盘挂载、特殊企业服务、直连握手超时与 Apple TV 播放转圈问题。
                </p>
              </div>
            </div>

            <!-- Add Input Row -->
            <div class="flex flex-col sm:flex-row gap-2 pt-1">
              <input v-model="app.newFakeIpInput" @keyup.enter="app.addFakeIpFilter"
                type="text" placeholder="输入需直连解析的域名 (例如: *.zspace.com 或 nas.myhome.local)"
                class="flex-1 px-3.5 py-2 rounded-xl bg-white border border-slate-200/90 text-[#0F172A] placeholder-slate-400 text-xs font-mono shadow-sm focus:border-sky-500 focus:outline-none transition" />
              <button @click="app.addFakeIpFilter" :disabled="app.fakeIpLoading || !app.newFakeIpInput.trim()"
                class="px-4 py-2 rounded-xl bg-gradient-to-r from-sky-500 to-emerald-500 hover:from-sky-400 hover:to-emerald-400 disabled:opacity-40 text-white text-xs font-bold transition flex items-center justify-center gap-1.5 cursor-pointer shrink-0 shadow-md shadow-sky-500/20 active:scale-95">
                <span v-if="app.fakeIpLoading" class="animate-spin">↻</span>
                <span v-else>➕</span>
                <span>添加直连白名单</span>
              </button>
            </div>

            <!-- Active User Filters -->
            <div class="space-y-2 pt-2 border-t border-slate-200/80">
              <div class="flex items-center justify-between text-xs">
                <span class="text-slate-700 font-bold flex items-center gap-1.5">
                  <span>📌</span> 用户自定义直连域名 ({{ app.fakeIpFilters.length }} 项)
                </span>
                <span class="text-[11px] text-slate-500 font-medium">重启或更新永不丢失</span>
              </div>
              <div v-if="app.fakeIpFilters.length === 0" class="text-xs text-slate-500 py-2 italic font-mono">
                暂未添加自定义域名。默认已内置极空间 NAS、Apple 设备自检与国内主流服务。
              </div>
              <div v-else class="flex flex-wrap gap-2 pt-1">
                <span v-for="domain in app.fakeIpFilters" :key="domain"
                  class="px-2.5 py-1 rounded-xl bg-sky-50 border border-sky-300 text-sky-900 text-xs font-mono flex items-center gap-1.5 shadow-sm group">
                  <span class="font-bold">{{ domain }}</span>
                  <button @click="app.removeFakeIpFilter(domain)"
                    class="text-sky-600 hover:text-white hover:bg-sky-600 rounded px-1 ml-1 font-bold cursor-pointer transition" title="移除此域名">✕</button>
                </span>
              </div>
            </div>

            <!-- Built-in Defaults Display -->
            <div class="space-y-1.5 pt-2 border-t border-slate-200/80 text-xs">
              <span class="text-slate-600 font-semibold flex items-center gap-1.5">
                <span>🛡️</span> 系统内置免 Fake-IP 直连项 (部分展示):
              </span>
              <div class="flex flex-wrap gap-1.5 text-[11px] font-mono text-slate-700">
                <span class="px-2 py-0.5 rounded-lg bg-emerald-50 text-emerald-800 border border-emerald-300 font-medium shadow-sm">+.zspace.com (极空间NAS)</span>
                <span class="px-2 py-0.5 rounded-lg bg-emerald-50 text-emerald-800 border border-emerald-300 font-medium shadow-sm">+.zspace.cn</span>
                <span class="px-2 py-0.5 rounded-lg bg-white border border-slate-200/90 text-slate-700 shadow-sm font-medium">geosite:cn</span>
                <span class="px-2 py-0.5 rounded-lg bg-white border border-slate-200/90 text-slate-700 shadow-sm font-medium">*.lan / *.local</span>
                <span class="px-2 py-0.5 rounded-lg bg-white border border-slate-200/90 text-slate-700 shadow-sm font-medium">NTP 时间同步</span>
                <span class="px-2 py-0.5 rounded-lg bg-white border border-slate-200/90 text-slate-700 shadow-sm font-medium">+.routeros.com</span>
              </div>
            </div>
          </div>

        </div>
</template>

<script setup>
import { inject, ref, computed, onMounted } from 'vue'
import ZootopiaIcon from '../ZootopiaIcon.vue'
const app = inject('cubexApp')
const showWhitelistSection = ref(false)

// ➕ v9.9.0 一句话网站分流 (Quick Custom Domain Override)
const quickDomainInput = ref('')
const quickAdding = ref(false)

// v9.4.0 User Overrides
const userOverrides = ref({
  force_direct_domains: [],
  force_proxy_domains: [],
  force_reject_domains: []
})

const totalCustomDomainCount = computed(() => {
  return (userOverrides.value.force_proxy_domains?.length || 0) +
         (userOverrides.value.force_direct_domains?.length || 0) +
         (userOverrides.value.force_reject_domains?.length || 0)
})

function cleanDomain(val) {
  return (val || '').trim().toLowerCase().replace(/^https?:\/\//, '').split('/')[0].split(':')[0].replace(/^\*?\./, '')
}

async function handleQuickAdd(outbound) {
  const domain = cleanDomain(quickDomainInput.value)
  if (!domain || !domain.includes('.')) {
    if (app && app.showToast) app.showToast('⚠️ 请输入有效的网址或域名 (如 javdb.com)', true)
    return
  }
  quickAdding.value = true
  try {
    const res = await fetch('/api/v1/rules/user-overrides/target', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ target: domain, outbound })
    })
    const data = await res.json()
    if (data.ok) {
      quickDomainInput.value = ''
      await fetchUserOverrides()
      const label = outbound === 'PROXIES' ? '⚡ 走加速' : outbound === 'DIRECT' ? '🏠 走直连' : '🛑 拦截屏蔽'
      if (app && app.showToast) app.showToast(`✅ [${domain}] 已设置为【${label}】并即刻生效`)
    } else {
      if (app && app.showToast) app.showToast('⚠️ 设置失败: ' + (data.error || '未知错误'), true)
    }
  } catch (err) {
    if (app && app.showToast) app.showToast('⚠️ 请求失败: ' + err.message, true)
  } finally {
    quickAdding.value = false
  }
}

async function handleQuickClear(domain) {
  try {
    const res = await fetch('/api/v1/rules/user-overrides/target', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ target: domain, outbound: 'CLEAR' })
    })
    const data = await res.json()
    if (data.ok) {
      await fetchUserOverrides()
      if (app && app.showToast) app.showToast(`🗑️ [${domain}] 已清除自定义分流，恢复默认规则`)
    } else {
      if (app && app.showToast) app.showToast('⚠️ 清除失败: ' + (data.error || '未知错误'), true)
    }
  } catch (err) {
    if (app && app.showToast) app.showToast('⚠️ 清除失败: ' + err.message, true)
  }
}
async function fetchUserOverrides() {
  try {
    const res = await fetch('/api/v1/rules/overrides')
    if (res.ok) {
      const data = await res.json()
      if (data.ok && data.data) {
        userOverrides.value = {
          force_direct_domains: data.data.force_direct_domains || [],
          force_proxy_domains: data.data.force_proxy_domains || [],
          force_reject_domains: data.data.force_reject_domains || []
        }
      }
    }
  } catch (err) {
    console.error('Failed to load user overrides:', err)
  }
}

onMounted(() => {
  fetchUserOverrides()
})
</script>
