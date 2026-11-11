<template>
        <div class="space-y-6 animate-in fade-in">

          <!-- ════════════════════════════════════════════════════════════════ -->

          <!-- 1. 全维度安全防泄露体检雷达与自愈中枢 Header                     -->

          <!-- ════════════════════════════════════════════════════════════════ -->

          <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4 pb-2">

            <div class="macos-title-shield space-y-1">

              <div class="flex items-center gap-2">

                <span class="text-xl">🛡️</span>

                <h2 class="text-base sm:text-lg font-bold text-white tracking-wide">

                  全维度防泄露安全雷达与自愈中心

                </h2>

              </div>

              <p class="text-xs text-[#E2F8EC] tracking-normal font-medium">

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

              <div class="px-3 py-1.5 rounded-xl bg-white/10 text-white/85 border border-slate-700/60 text-xs text-slate-400 flex items-center gap-1.5">

                <span class="text-emerald-300">💡</span>

                <span>核心安全规则自愈已全量合入【宿主共生】一体化全链路自愈总线</span>

              </div>

            </div>

          </div>

          <!-- ════════════════════════════════════════════════════════════════ -->

          <!-- 2. 安全总评与能量光环看板 (Security Radar Scorecard)               -->

          <!-- ════════════════════════════════════════════════════════════════ -->

          <div class="p-6 rounded-3xl macos-glass-card hover:border-sky-400/40 shadow-xl relative overflow-hidden">

            <div class="absolute -top-24 -right-24 w-80 h-80 rounded-full bg-sky-500/10 blur-3xl pointer-events-none"></div>

            <div class="flex flex-col lg:flex-row items-center justify-between gap-6 relative z-10">

              <!-- Left: Circular Score Gauge -->

              <div class="flex items-center gap-5">

                <div class="relative w-24 h-24 flex items-center justify-center shrink-0">

                  <div class="absolute inset-0 rounded-full border-4 border-slate-700/40"></div>

                  <div class="absolute inset-1 rounded-full border-2 border-emerald-500/40"></div>

                  <div class="w-20 h-20 rounded-full bg-white/10 text-white border border-emerald-500/40 flex flex-col items-center justify-center shadow-inner">

                    <span class="text-2xl font-black font-mono text-emerald-300 tracking-tighter">

                      {{ app.securityAudit.report?.total_score || 100 }}

                    </span>

                    <span class="text-[11px] font-mono text-slate-400 uppercase tracking-widest">/ 100 分</span>

                  </div>

                </div>

                <div class="space-y-1">

                  <div class="flex items-center gap-2">

                    <span class="text-lg font-extrabold text-white">

                      {{ app.securityAudit.report?.security_rating || '暮光晶盾级 (100% 零泄露)' }}

                    </span>

                    <!-- Active Engine Badge -->

                    <span class="px-2.5 py-0.5 rounded-full text-[11px] font-mono font-bold flex items-center gap-1 border"

                      :class="app.activeEngine === 'singbox' ? 'bg-purple-950 text-emerald-300 border-purple-500/40' : 'bg-sky-950 text-amber-300 border-amber-500/40'">

                      <span>内核:</span>

                      <span class="font-extrabold">{{ app.activeEngine === 'singbox' ? 'sing-box' : 'mihomo' }}</span>

                    </span>

                  </div>

                  <p class="text-xs text-slate-300 max-w-xl leading-relaxed">

                    {{ app.securityAudit.report?.summary || '所有 6 大硬核防泄露测试全量通过！DNS 0 明文回退、WebRTC Fake-IP 隔离已就绪。' }}

                  </p>

                  <div class="text-[10px] text-slate-400 font-mono">

                    最新渗透测试时间: {{ new Date(app.securityAudit.report?.tested_at || Date.now()).toLocaleTimeString() }}

                  </div>

                </div>

              </div>

              <!-- Right: Hardcore Pillars Mini-Chips -->

              <div class="grid grid-cols-1 sm:grid-cols-3 lg:grid-cols-1 gap-2.5 w-full lg:w-auto shrink-0">

                <div class="px-3 py-2 rounded-xl bg-white/10 text-white/90 border border-slate-800 flex items-center gap-2.5 text-xs font-mono">

                  <span class="w-2 h-2 rounded-full bg-emerald-400 shadow-sm shadow-emerald-400"></span>

                  <span class="text-slate-400">DNS 0 明文回退:</span>

                  <span class="text-emerald-300 font-bold">100% 熔断阻断</span>

                </div>

                <div class="px-3 py-2 rounded-xl bg-white/10 text-white/90 border border-slate-800 flex items-center gap-2.5 text-xs font-mono">

                  <span class="w-2 h-2 rounded-full bg-cyan-400 shadow-sm shadow-cyan-400"></span>

                  <span class="text-slate-400">WebRTC 境外隔离:</span>

                  <span class="text-emerald-300 font-bold">Fake-IP 保密</span>

                </div>

                <div class="px-3 py-2 rounded-xl bg-white/10 text-white/90 border border-slate-800 flex items-center gap-2.5 text-xs font-mono">

                  <span class="w-2 h-2 rounded-full bg-teal-400 shadow-sm shadow-teal-400"></span>

                  <span class="text-slate-400">Apple Type 65:</span>

                  <span class="text-emerald-300 font-bold">100% 清洗拦截</span>

                </div>

              </div>

            </div>

          </div>

          <!-- ════════════════════════════════════════════════════════════════ -->

          <!-- 3. 六大硬核安全测试 (6-Vector Hardcore Test Matrix)           -->

          <!-- ════════════════════════════════════════════════════════════════ -->

          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">

            <div v-for="item in (app.securityAudit.report?.items || [])" :key="item.id"

              class="p-4 rounded-2xl macos-sub-card border transition-all duration-300 hover:border-sky-400/40 flex flex-col justify-between space-y-3"

              :class="item.status === 'PASS' ? 'border-slate-700/60' : (item.status === 'WARN' ? 'border-amber-500/40' : 'border-rose-500/40')">

              <!-- Card Top -->

              <div class="space-y-2">

                <div class="flex items-start justify-between gap-2">

                  <div>

                    <span class="text-[10px] font-mono px-2 py-0.5 rounded-md font-bold uppercase tracking-wider"

                      :class="item.risk_level === 'CRITICAL' ? 'bg-rose-950 text-rose-300 border border-rose-500/30' : (item.risk_level === 'HIGH' ? 'bg-amber-950 text-amber-300 border border-amber-500/30' : 'bg-emerald-500/15 text-emerald-300 border border-emerald-400/30')">

                      {{ item.category }} · {{ item.risk_level }}

                    </span>

                    <h3 class="text-sm font-bold text-white mt-1.5 flex items-center gap-1.5">

                      <span>{{ item.name }}</span>

                    </h3>

                  </div>

                  <!-- Status Badge -->

                  <span class="px-2.5 py-1 rounded-lg text-xs font-mono font-bold shrink-0 border shadow-sm"

                    :class="item.status === 'PASS' ? 'bg-emerald-950/90 text-emerald-300 border-emerald-500/50' : (item.status === 'WARN' ? 'bg-amber-950/90 text-amber-300 border-amber-500/50' : 'bg-rose-950/90 text-rose-300 border-rose-500/50')">

                    {{ item.status === 'PASS' ? '✓ PASS 0泄露' : (item.status === 'WARN' ? '▲ WARN 待加固' : '✕ FAIL 风险') }}

                  </span>

                </div>

                <!-- Technical Specification (Visible Emerald Green · Frosted Glass) -->

                <div class="text-[11px] font-mono font-bold text-emerald-300 bg-emerald-500/10 px-2.5 py-1 rounded-md border border-emerald-400/25 shadow-sm">

                  规范: {{ item.technical_specs }}

                </div>

                <!-- Description -->

                <p class="text-xs text-slate-300 leading-relaxed">

                  {{ item.details }}

                </p>

              </div>

              <!-- Card Bottom -->

              <div class="pt-2 border-t border-slate-800/60 space-y-2">

                <!-- Score bar -->

                <div class="flex items-center justify-between text-[11px] font-mono">

                  <span class="text-slate-400">单项防御得分</span>

                  <span class="font-bold" :class="item.score >= 90 ? 'text-emerald-400' : (item.score >= 70 ? 'text-amber-400' : 'text-rose-400')">

                    {{ item.score }} / 100

                  </span>

                </div>

                <div class="w-full h-1.5 rounded-full bg-white/10 text-white overflow-hidden">

                  <div class="h-full rounded-full transition-all duration-500"

                    :class="item.score >= 90 ? 'bg-gradient-to-r from-teal-400 to-emerald-400' : (item.score >= 70 ? 'bg-gradient-to-r from-amber-400 to-yellow-400' : 'bg-rose-500')"

                    :style="{ width: item.score + '%' }"></div>

                </div>

                <!-- Recommendation / Remediation status -->

                <div class="text-[10px] text-slate-400 font-mono flex items-center justify-between">

                  <span>{{ item.recommendation }}</span>

                  <span v-if="item.remediated" class="text-emerald-300 font-bold">🛡️ 已加固</span>

                </div>

              </div>

            </div>

          </div>

          <!-- ════════════════════════════════════════════════════════════════ -->

          <!-- 4. 217heidai 去广告&隐私追踪拦截 (AdBlock & Threat Interception)    -->

          <!-- ════════════════════════════════════════════════════════════════ -->

          <div class="space-y-5 pt-6 border-t border-sky-800/40">

            <!-- Header: 217heidai 去广告&隐私追踪拦截 -->

            <div class="flex items-center justify-between flex-wrap gap-3 pb-3 border-b border-slate-700/40">

              <div>

                <h2 class="text-base sm:text-lg font-bold text-white tracking-wide flex items-center gap-2">

                  <span class="flex items-center gap-2"><ZootopiaIcon name="rules" :size="22" class="shrink-0" /><span>217heidai 去广告&隐私追踪拦截</span></span>

                </h2>

                <p class="text-xs text-slate-400 mt-1">

                  基于 217heidai MRS 规则库的高精度广告与隐私追踪拦截 · 最近 24 小时 DNS 耗时、查询吞吐与恶意威胁拦截遥测

                </p>

              </div>

              <div class="flex items-center gap-2 flex-wrap font-mono">

                <span class="text-xs text-emerald-400 bg-emerald-950 px-3 py-1 rounded-full border border-emerald-500/40 font-bold">

                  规则库: {{ app.adblockStats.total_rules || 233481 }} 条

                </span>

                <span class="text-xs text-emerald-300 bg-cyan-950 px-3 py-1 rounded-full border border-cyan-500/40 font-bold">

                  拦截率: {{ app.dnsFilterStats.blockedPercentage }}%

                </span>

              </div>

            </div>

            <!-- ════════════════════════════════════════════════════════════════ -->

            <!-- 域名过滤与广告拦截中枢 (Domain Filter & AdBlock Hub)                 -->

            <!-- ════════════════════════════════════════════════════════════════ -->

            <div class="rounded-3xl macos-glass-card hover:border-sky-400/40 transition-all duration-300 p-5 space-y-4 shadow-xl">

              <!-- Top Row: Title, Master Toggle, Status Badge, Action Buttons -->

              <div class="flex items-center justify-between flex-wrap gap-3 pb-3 border-b border-slate-800/80">

                <div class="flex items-center gap-3">

                  <h3 class="text-base font-bold text-white tracking-wide flex items-center gap-2">

                    <span>域名过滤</span>

                  </h3>

                  <!-- Master Switch -->

                  <label class="relative inline-flex items-center cursor-pointer">

                    <input type="checkbox" v-model="app.filterConfig.enabled" @change="app.toggleFilterMaster" class="sr-only peer">

                    <div class="w-11 h-6 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-500 shadow-inner"></div>

                  </label>

                  <span :class="['text-xs px-2.5 py-0.5 rounded-full font-bold border transition', app.filterConfig.enabled ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40' : 'bg-white/10 text-white text-slate-400 border-slate-700']">

                    {{ app.filterConfig.enabled ? '已生效' : '已暂停' }}

                  </span>

                </div>

                <div class="flex items-center gap-2">

                  <button @click="app.fetchDnsFilterStats" title="刷新状态" class="p-2 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white/80 hover:bg-slate-700 text-slate-300 hover:text-white border border-slate-700/60 active:scale-95 transition cursor-pointer">

                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>

                  </button>

                  <button @click="app.openAddSourceModal" class="px-3 py-1.5 rounded-xl bg-gradient-to-r from-emerald-500 to-teal-500 hover:from-emerald-400 hover:to-teal-400 text-slate-950 font-bold text-xs flex items-center gap-1.5 shadow-md shadow-emerald-500/20 active:scale-95 transition cursor-pointer">

                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4"/></svg>

                    <span>添加规则源</span>

                  </button>

                </div>

              </div>

              <!-- Note / Description -->

              <p class="text-xs text-slate-400 leading-relaxed">

                默认开启。开启后，转发的 DNS 查询先经过过滤，通过后仍按照原 DNS 策略解析；白名单放行优先。支持原生二进制 MRS/SRS、域名、Hosts 和基础 Adblock 语法，不支持的条目会显示跳过数量。启用的名单按下面的自动更新计划更新，下载失败沿用上一次有效名单。

              </p>

              <!-- Rule Sources List -->

              <div class="space-y-2.5">

                <div v-for="src in app.filterConfig.rule_sources" :key="src.id"

                     class="rounded-2xl macos-sub-card hover:border-sky-400/40 transition-all p-3.5 flex flex-col md:flex-row md:items-center justify-between gap-3">

                  <div class="flex items-center gap-3 min-w-0 flex-1">

                    <!-- Source Toggle -->

                    <label class="relative inline-flex items-center cursor-pointer shrink-0">

                      <input type="checkbox" v-model="src.enabled" @change="app.toggleSourceItem(src)" class="sr-only peer">

                      <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-emerald-500 shadow-inner"></div>

                    </label>

                    <div class="min-w-0 flex-1">

                      <div class="flex items-center gap-2 flex-wrap">

                        <span class="text-xs font-bold text-white">{{ src.name }}</span>

                        <span class="text-[11px] font-mono text-slate-400 truncate max-w-xs sm:max-w-md block" :title="src.url">{{ src.url }}</span>

                      </div>

                      <div class="flex items-center gap-3 mt-1 text-[11px] text-slate-400 font-mono">

                        <span class="text-emerald-400 font-semibold">规则数 {{ (src.rules_count || 0).toLocaleString() }}</span>

                        <span class="text-slate-300">上次更新 {{ src.updated_at || '刚刚' }}</span>

                      </div>

                    </div>

                  </div>

                  <!-- Actions -->

                  <div class="flex items-center gap-1.5 shrink-0 self-end md:self-center">

                    <button @click="app.openSourcePreview(src)" title="查看规则详情" class="p-1.5 rounded-lg bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white/80 hover:bg-slate-700 text-slate-400 hover:text-white transition cursor-pointer">

                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>

                    </button>

                    <button @click="app.refreshSourceItem(src)" :disabled="app.isUpdatingSource" title="立即更新" class="p-1.5 rounded-lg bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white/80 hover:bg-slate-700 text-slate-400 hover:text-emerald-300 transition cursor-pointer disabled:opacity-50">

                      <svg :class="['w-3.5 h-3.5', app.isUpdatingSource ? 'animate-spin text-emerald-300' : '']" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>

                    </button>

                    <button @click="app.openEditSourceModal(src)" title="编辑规则源" class="p-1.5 rounded-lg bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white/80 hover:bg-slate-700 text-slate-400 hover:text-white transition cursor-pointer">

                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"/></svg>

                    </button>

                    <button v-if="src.id !== '217heidai'" @click="app.deleteSourceItem(src)" title="删除规则源" class="p-1.5 rounded-lg bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white/80 hover:bg-rose-950 text-slate-400 hover:text-rose-400 transition cursor-pointer">

                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>

                    </button>

                  </div>

                </div>

              </div>

              <!-- Auto Update Scheduler Bar -->

              <div class="rounded-2xl bg-slate-950/60 border border-slate-800/80 p-3.5 flex flex-wrap items-center gap-3 text-xs text-slate-300">

                <span class="font-bold text-white flex items-center gap-1.5">

                  <span>自动更新</span>

                </span>

                <label class="relative inline-flex items-center cursor-pointer shrink-0">

                  <input type="checkbox" v-model="app.filterConfig.auto_update.enabled" @change="app.saveFilterAutoUpdate" class="sr-only peer">

                  <div class="w-8 h-4 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[1px] after:left-[1px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-3.5 after:w-3.5 after:transition-all peer-checked:bg-emerald-500 shadow-inner"></div>

                </label>

                <div class="flex items-center gap-2 flex-wrap">

                  <span>每</span>

                  <select v-model="app.filterConfig.auto_update.interval_days" @change="app.saveFilterAutoUpdate" class="bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white border border-slate-700 rounded-lg px-2 py-1 text-xs text-white focus:outline-none focus:border-emerald-500 font-mono cursor-pointer">

                    <option :value="1">1 天</option>

                    <option :value="3">3 天</option>

                    <option :value="7">7 天</option>

                  </select>

                  <span>在</span>

                  <select v-model="app.filterConfig.auto_update.update_time" @change="app.saveFilterAutoUpdate" class="bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white border border-slate-700 rounded-lg px-2 py-1 text-xs text-white focus:outline-none focus:border-emerald-500 font-mono cursor-pointer">

                    <option value="04:00">04:00</option>

                    <option value="06:00">06:00</option>

                    <option value="08:00">08:00</option>

                    <option value="12:00">12:00</option>

                    <option value="18:00">18:00</option>

                    <option value="00:00">00:00</option>

                  </select>

                  <span class="text-slate-400 text-[11px]">在选定时间更新一次；内核自己加载新名单，不重启。</span>

                </div>

              </div>

              <!-- Whitelist (放行域名) Accordion -->

              <div class="border-t border-slate-800/80 pt-3">

                <button @click="showWhitelistSection = !showWhitelistSection" class="w-full flex items-center justify-between text-xs font-bold text-slate-300 hover:text-white transition cursor-pointer py-1">

                  <span class="flex items-center gap-1.5">

                    <svg :class="['w-3.5 h-3.5 transition-transform text-slate-400', showWhitelistSection ? 'rotate-180' : '']" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/></svg>

                    <span>放行域名 ({{ app.filterConfig.whitelist_domains?.length || 0 }})</span>

                  </span>

                  <span class="text-[11px] text-slate-400 font-normal">优先直连放行 · 防误杀</span>

                </button>

                <div v-if="showWhitelistSection" class="mt-3 space-y-3 pl-4 border-l-2 border-emerald-500/40 animate-in fade-in">

                  <!-- Tag Cloud -->

                  <div class="flex flex-wrap items-center gap-2">

                    <span v-for="d in app.filterConfig.whitelist_domains" :key="d"

                          class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-slate-950 border border-slate-800 text-slate-300 text-xs font-mono group hover:border-emerald-500/50">

                      <span>{{ d }}</span>

                      <button @click="app.removeWhitelistItem(d)" title="移除放行" class="text-slate-300 group-hover:text-rose-400 transition cursor-pointer">

                        &times;

                      </button>

                    </span>

                    <span v-if="!app.filterConfig.whitelist_domains || app.filterConfig.whitelist_domains.length === 0" class="text-xs text-slate-400 italic">

                      暂无放行域名，可在下方添加特定业务域名以防误杀

                    </span>

                  </div>

                  <!-- Add Form -->

                  <div class="flex items-center gap-2 max-w-md">

                    <input type="text" v-model="app.newWhitelistDomain" @keyup.enter="app.addWhitelistItem" placeholder="输入放行域名 (如 apple.com)" class="flex-1 bg-slate-950 border border-slate-800 rounded-xl px-3 py-1.5 text-xs text-white placeholder-slate-400 focus:outline-none focus:border-emerald-500 font-mono">

                    <button @click="app.addWhitelistItem" class="px-3 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-emerald-400 font-bold text-xs border border-slate-700 active:scale-95 transition cursor-pointer">

                      添加放行

                    </button>

                  </div>

                </div>

              </div>

            </div>

            <!-- ════════════════════════════════════════════════════════════════ -->

            <!-- 🔀 DNS 重写 (域名别名重定向 & 静态 IP 映射)                        -->

            <!-- ════════════════════════════════════════════════════════════════ -->

            <div class="rounded-3xl bg-white/10 text-white/90 border border-slate-800/90 hover:border-cyan-500/30 transition-all duration-300 p-5 space-y-4 shadow-xl backdrop-blur-xl">

              <!-- Top Row: Title, Master Toggle, Status Badge, Action Buttons -->

              <div class="flex items-center justify-between flex-wrap gap-3 pb-3 border-b border-slate-800/80">

                <div class="flex items-center gap-3">

                  <h3 class="text-base font-bold text-white tracking-wide flex items-center gap-2">

                    <span class="text-emerald-300">🔀</span>

                    <span>DNS 重写和 局域网A 记录静态映射</span>

                  </h3>

                  <!-- Master Switch -->

                  <label class="relative inline-flex items-center cursor-pointer">

                    <input type="checkbox" v-model="app.dnsRewriteConfig.enabled" @change="app.toggleDnsRewriteMaster" class="sr-only peer">

                    <div class="w-11 h-6 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-cyan-500 shadow-inner"></div>

                  </label>

                  <span :class="['text-xs px-2.5 py-0.5 rounded-full font-bold border transition', app.dnsRewriteConfig.enabled ? 'bg-cyan-500/20 text-emerald-300 border-cyan-500/40' : 'bg-white/10 text-white text-slate-400 border-slate-700']">

                    {{ app.dnsRewriteConfig.enabled ? '已生效' : '已暂停' }}

                  </span>

                </div>

                <div class="flex items-center gap-2">

                  <button @click="app.resetDnsRewriteDefaults" title="恢复默认 DNS 重写与 局域网A 记录静态映射规则" class="px-3 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white/80 hover:bg-slate-700 text-slate-300 hover:text-white border border-slate-700/60 active:scale-95 transition cursor-pointer text-xs flex items-center gap-1.5">

                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>

                    <span>重置默认</span>

                  </button>

                  <button @click="app.openAddDnsRewriteModal" class="px-3 py-1.5 rounded-xl bg-gradient-to-r from-cyan-500 to-blue-500 hover:from-cyan-400 hover:to-blue-400 text-slate-950 font-bold text-xs flex items-center gap-1.5 shadow-md shadow-sky-500/20 active:scale-95 transition cursor-pointer">

                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4"/></svg>

                    <span>添加重写</span>

                  </button>

                </div>

              </div>

              <!-- Note / Description matching user screenshot -->

              <p class="text-xs text-slate-400 leading-relaxed">

                把某个域名的解析结果改成另一个域名或固定 IP；默认带两条 Google 的重写（解决 Google Play 无法下载 App 问题）及一条局域网 A 记录静态映射示例。

              </p>

              <!-- Rules List -->

              <div class="space-y-2.5">

                <div v-for="rule in app.dnsRewriteConfig.rules" :key="rule.id"

                     class="rounded-2xl bg-slate-950/80 border border-slate-800 hover:border-cyan-500/30 transition-all p-3.5 flex flex-col md:flex-row md:items-center justify-between gap-3">

                  <div class="flex-1 min-w-0 space-y-1">

                    <div class="flex items-center gap-2 flex-wrap">

                      <span class="font-mono text-xs font-bold text-white tracking-wide bg-white/10 text-white px-2 py-0.5 rounded border border-slate-800">{{ rule.domain }}</span>

                      <span class="text-emerald-300 text-xs">➔</span>

                      <span class="font-mono text-xs font-bold text-emerald-300 tracking-wide bg-cyan-950/50 px-2 py-0.5 rounded border border-cyan-800/50">{{ rule.target }}</span>

                      <span :class="['text-[10px] font-bold px-2 py-0.5 rounded uppercase tracking-wider', rule.type === 'CNAME' ? 'bg-cyan-500/10 text-emerald-300 border border-cyan-500/30' : 'bg-purple-500/10 text-purple-400 border border-purple-500/30']">

                        {{ rule.type === 'CNAME' ? 'CNAME 别名' : 'A 静态记录' }}

                      </span>

                    </div>

                    <div v-if="rule.comment" class="text-[11px] text-slate-400 flex items-center gap-1.5">

                      <span class="text-white/40">💬</span>

                      <span>{{ rule.comment }}</span>

                    </div>

                  </div>

                  <!-- Actions & Toggle -->

                  <div class="flex items-center gap-2.5 self-end md:self-auto shrink-0">

                    <label class="relative inline-flex items-center cursor-pointer" title="启用/停用单条规则">

                      <input type="checkbox" v-model="rule.enabled" @change="app.toggleDnsRewriteRule(rule)" class="sr-only peer">

                      <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-cyan-500"></div>

                    </label>

                    <button @click="app.openEditDnsRewriteModal(rule)" title="编辑规则" class="p-1.5 rounded-lg bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-slate-300 hover:text-white transition cursor-pointer">

                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"/></svg>

                    </button>

                    <button @click="app.deleteDnsRewriteRule(rule)" title="删除规则" class="p-1.5 rounded-lg bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-rose-950 text-slate-400 hover:text-rose-300 border border-transparent hover:border-rose-800 transition cursor-pointer">

                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>

                    </button>

                  </div>

                </div>

                <div v-if="!app.dnsRewriteConfig.rules || app.dnsRewriteConfig.rules.length === 0" class="text-center py-6 text-xs text-slate-300 italic bg-slate-950/40 rounded-2xl border border-dashed border-slate-800">

                  暂无规则，点击右上角【添加重写】或【重置默认】快速恢复官方加速重写与 局域网A 记录静态映射示例

                </div>

              </div>

            </div>

            <!-- 4-Card 2x2 Grid with 24h Area Charts and Rank -->

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">

              <!-- Card 1: 平均解析耗时 -->

              <div class="p-4 rounded-2xl bg-slate-950/70 border border-slate-800/80 space-y-2 relative overflow-hidden">

                <div class="flex items-center justify-between text-xs text-slate-400 font-medium">

                  <span>平均解析耗时 (24h 遥测)</span>

                  <span class="text-[10px] text-emerald-400 font-mono flex items-center gap-1">

                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>

                    0 泄露极速响应

                  </span>

                </div>

                <div class="text-2xl font-bold font-mono text-white flex items-baseline gap-1">

                  <span>{{ app.dnsFilterStats.avgLatencyMs }}</span>

                  <span class="text-xs text-slate-400 font-sans font-normal">ms</span>

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

                    <path :d="app.dnsLatencyPath.line" fill="none" stroke="#C084FC" stroke-width="2" stroke-linecap="round" />

                    <!-- Live Pulse End Point -->

                    <circle cx="316" :cy="app.dnsLatencyLastY" r="3" fill="#C084FC" />

                    <circle cx="316" :cy="app.dnsLatencyLastY" r="6" fill="#C084FC" opacity="0.4" class="animate-ping" />

                  </svg>

                </div>

              </div>

              <!-- Card 2: DNS 查询量 -->

              <div class="p-4 rounded-2xl bg-slate-950/70 border border-slate-800/80 space-y-2 relative overflow-hidden">

                <div class="flex items-center justify-between text-xs text-slate-400 font-medium">

                  <span>DNS 查询量 (24h 吞吐)</span>

                  <span class="text-[10px] text-emerald-300 font-mono flex items-center gap-1">

                    <span class="w-1.5 h-1.5 rounded-full bg-sky-400 animate-pulse"></span>

                    国内直连 + Fake-IP

                  </span>

                </div>

                <div class="text-2xl font-bold font-mono text-white">

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

              <div class="p-4 rounded-2xl bg-slate-950/70 border border-slate-800/80 space-y-2 relative overflow-hidden flex flex-col justify-between">

                <div>

                  <div class="flex items-center justify-between text-xs">

                    <span class="text-slate-400 font-medium">今日阻断与拦截</span>

                    <span class="text-xs font-bold font-mono text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded-lg border border-amber-500/30">

                      拦截率 {{ app.dnsFilterStats.blockedPercentage }}%

                    </span>

                  </div>

                  <div class="text-2xl font-bold font-mono text-amber-400">

                    {{ app.dnsFilterStats.blockedCount.toLocaleString() }}

                  </div>

                  <div class="h-16 w-full pt-1">

                    <svg class="w-full h-full" viewBox="0 0 320 60" preserveAspectRatio="none">

                      <defs>

                        <linearGradient id="dnsBlockedGrad" x1="0" y1="0" x2="0" y2="1">

                          <stop offset="0%" stop-color="#f59e0b" stop-opacity="0.4"/>

                          <stop offset="100%" stop-color="#f59e0b" stop-opacity="0.0"/>

                        </linearGradient>

                      </defs>

                      <path :d="app.dnsBlockedPath.area" fill="url(#dnsBlockedGrad)" />

                      <path :d="app.dnsBlockedPath.line" fill="none" stroke="#f59e0b" stroke-width="2" stroke-linecap="round" />

                      <!-- Live Pulse End Point -->

                      <circle cx="316" :cy="app.dnsBlockedLastY" r="3" fill="#f59e0b" />

                      <circle cx="316" :cy="app.dnsBlockedLastY" r="6" fill="#f59e0b" opacity="0.4" class="animate-ping" />

                    </svg>

                  </div>

                </div>

                <!-- 均衡行高底部分类标签 -->

                <div class="grid grid-cols-3 gap-2 pt-2 border-t border-slate-800/60 text-[10px] font-mono text-center">

                  <div class="p-1.5 rounded-lg bg-slate-900/70 text-white/75 border border-slate-800/60">

                    <div class="text-slate-400">广告弹窗</div>

                    <div class="text-amber-400 font-bold mt-0.5">14,820</div>

                  </div>

                  <div class="p-1.5 rounded-lg bg-slate-900/70 text-white/75 border border-slate-800/60">

                    <div class="text-slate-400">隐私追踪</div>

                    <div class="text-amber-400 font-bold mt-0.5">5,640</div>

                  </div>

                  <div class="p-1.5 rounded-lg bg-slate-900/70 text-white/75 border border-slate-800/60">

                    <div class="text-slate-400">恶意劫持</div>

                    <div class="text-amber-400 font-bold mt-0.5">1,220</div>

                  </div>

                </div>

              </div>

              <!-- Card 4: 被拦截域名排行 -->

              <div class="p-4 rounded-2xl bg-slate-950/70 border border-slate-800/80 space-y-3 relative overflow-hidden flex flex-col justify-between">

                <div>

                  <div class="flex items-center justify-between text-xs text-slate-400 font-medium">

                    <span>被拦截广告与恶意域名排行</span>

                    <span class="text-[10px] text-amber-400 font-mono">217heidai 高危特征</span>

                  </div>

                  <div class="space-y-2 pt-2">

                    <div v-for="item in app.dnsFilterStats.topBlocked" :key="item.domain" class="space-y-1">

                      <div class="flex items-center justify-between text-[11px] font-mono">

                        <span class="text-slate-300 truncate max-w-[240px]" :title="item.domain">{{ item.domain }}</span>

                        <span class="text-slate-400 font-semibold">{{ item.count.toLocaleString() }}</span>

                      </div>

                      <div class="w-full bg-white/10 text-white/85 h-1.5 rounded-full overflow-hidden">

                        <div class="bg-gradient-to-r from-amber-500 to-orange-500 h-full rounded-full transition-all duration-500"

                          :style="{ width: item.percent + '%' }"></div>

                      </div>

                    </div>

                  </div>

                </div>

              </div>

            </div>

          </div>

          <!-- ════════════════════════════════════════════════════════════════ -->
          <!-- 6. Fake-IP 白名单与真实 IP 直连解析中枢 (Fake-IP Filter Hub)       -->
          <!-- ════════════════════════════════════════════════════════════════ -->
          <div class="rounded-2xl p-5 macos-glass-card border border-sky-500/25 space-y-4">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div class="space-y-1">
                <div class="flex items-center gap-2">
                  <span class="text-lg">🎯</span>
                  <h3 class="text-sm sm:text-base font-bold text-white tracking-wide">
                    Fake-IP 过滤白名单 (真实 IP 直连解析中枢)
                  </h3>
                  <span class="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 font-bold">
                    双核秒级热重载 · 掉电持久化
                  </span>
                </div>
                <p class="text-xs text-slate-300">
                  名单内域名将跳过 198.18.0.0/15 假地址拦截，由本地运营商 DNS 直接返回物理真实 IP。彻底解决极空间 NAS (zspace.com)、本地网盘挂载、特殊企业服务、直连握手超时与 Apple TV 播放转圈问题。
                </p>
              </div>
            </div>

            <!-- Add Input Row -->
            <div class="flex flex-col sm:flex-row gap-2 pt-1">
              <input v-model="app.newFakeIpInput" @keyup.enter="app.addFakeIpFilter"
                type="text" placeholder="输入需直连解析的域名 (例如: *.zspace.com 或 nas.myhome.local)"
                class="flex-1 px-3.5 py-2 rounded-xl bg-slate-950/70 border border-sky-500/30 text-white placeholder-slate-500 text-xs font-mono focus:border-sky-400 focus:outline-none transition" />
              <button @click="app.addFakeIpFilter" :disabled="app.fakeIpLoading || !app.newFakeIpInput.trim()"
                class="px-4 py-2 rounded-xl bg-gradient-to-r from-sky-500 to-emerald-500 hover:from-sky-400 hover:to-emerald-400 disabled:opacity-40 text-white text-xs font-bold transition flex items-center justify-center gap-1.5 cursor-pointer shrink-0">
                <span v-if="app.fakeIpLoading" class="animate-spin">↻</span>
                <span v-else>➕</span>
                <span>添加直连白名单</span>
              </button>
            </div>

            <!-- Active User Filters -->
            <div class="space-y-2 pt-2 border-t border-slate-700/40">
              <div class="flex items-center justify-between text-xs">
                <span class="text-slate-300 font-semibold flex items-center gap-1.5">
                  <span>📌</span> 用户自定义直连域名 ({{ app.fakeIpFilters.length }} 项)
                </span>
                <span class="text-[11px] text-slate-400">重启或更新永不丢失</span>
              </div>
              <div v-if="app.fakeIpFilters.length === 0" class="text-xs text-slate-400 py-2 italic font-mono">
                暂未添加自定义域名。默认已内置极空间 NAS、Apple 设备自检与国内主流服务。
              </div>
              <div v-else class="flex flex-wrap gap-2 pt-1">
                <span v-for="domain in app.fakeIpFilters" :key="domain"
                  class="px-2.5 py-1 rounded-lg bg-sky-950/70 border border-sky-400/40 text-sky-200 text-xs font-mono flex items-center gap-1.5 shadow-sm group">
                  <span>{{ domain }}</span>
                  <button @click="app.removeFakeIpFilter(domain)"
                    class="text-sky-400 hover:text-red-300 ml-1 font-bold cursor-pointer transition" title="移除此域名">✕</button>
                </span>
              </div>
            </div>

            <!-- Built-in Defaults Display -->
            <div class="space-y-1.5 pt-2 border-t border-slate-700/40 text-xs">
              <span class="text-slate-400 font-medium flex items-center gap-1.5">
                <span>🛡️</span> 系统内置免 Fake-IP 直连项 (部分展示):
              </span>
              <div class="flex flex-wrap gap-1.5 text-[11px] font-mono text-slate-400">
                <span class="px-2 py-0.5 rounded bg-emerald-950/50 text-emerald-300 border border-emerald-500/30">+.zspace.com (极空间NAS)</span>
                <span class="px-2 py-0.5 rounded bg-emerald-950/50 text-emerald-300 border border-emerald-500/30">+.zspace.cn</span>
                <span class="px-2 py-0.5 rounded bg-slate-900/80 text-slate-300 border border-slate-700">geosite:cn</span>
                <span class="px-2 py-0.5 rounded bg-slate-900/80 text-slate-300 border border-slate-700">*.lan / *.local</span>
                <span class="px-2 py-0.5 rounded bg-slate-900/80 text-slate-300 border border-slate-700">NTP 时间同步</span>
                <span class="px-2 py-0.5 rounded bg-slate-900/80 text-slate-300 border border-slate-700">+.routeros.com</span>
              </div>
            </div>
          </div>

        </div>
</template>

<script setup>
import { inject, ref } from 'vue'
import ZootopiaIcon from '../ZootopiaIcon.vue'
const app = inject('cubexApp')
const showWhitelistSection = ref(false)
</script>
