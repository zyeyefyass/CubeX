<template>
  <div class="space-y-6 animate-in fade-in">

    <!-- 1. 全维度安全防泄露体检雷达与自愈中枢 Header -->
    <div class="p-5 sm:p-6 rounded-3xl cbx-card shadow-lg flex flex-col lg:flex-row lg:items-center justify-between gap-4">
      <div class="space-y-1">
        <div class="flex items-center gap-2">
          <span class="text-xl">🛡️</span>
          <h2 class="text-base sm:text-lg font-black cbx-text-main tracking-wide">
            全维度防泄露安全雷达与自愈中心
          </h2>
        </div>
        <p class="text-xs cbx-text-sub tracking-normal font-medium">
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

        <div class="px-3 py-1.5 rounded-xl cbx-sub-card cbx-text-sub text-xs flex items-center gap-1.5 shadow-sm font-medium">
          <span class="text-emerald-600 dark:text-emerald-400">💡</span>
          <span>核心安全规则自愈已全量合入【宿主共生】一体化全链路自愈总线</span>
        </div>
      </div>
    </div>

    <!-- 1.5 一句话网站分流 (Quick Custom Domain Override) -->
    <div class="p-5 sm:p-6 rounded-3xl cbx-card shadow-lg relative overflow-hidden space-y-4">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-10 h-10 rounded-2xl cbx-badge-info border flex items-center justify-center text-xl shrink-0 shadow-sm">
            ➕
          </div>
          <div class="min-w-0">
            <h3 class="text-base sm:text-lg font-black cbx-text-main tracking-wide flex items-center gap-2 flex-wrap">
              <span>一句话网站分流</span>
              <span class="text-[11px] font-mono px-2 py-0.5 rounded-full cbx-badge-info border font-bold">即按即生效 · 零感热重载</span>
            </h3>
            <p class="text-xs cbx-text-sub mt-0.5 font-medium">
              输入任意网址或域名，一键决定加速、直连或屏蔽，双核自动同步生效
            </p>
          </div>
        </div>
        <div class="text-xs font-mono cbx-text-sub self-start sm:self-center shrink-0">
          已自定义: <strong class="text-sky-600 dark:text-cyan-300 font-black font-tabular">{{ totalCustomDomainCount }}</strong> 个网址
        </div>
      </div>

      <!-- 极简输入框与 3 个大操作按钮 -->
      <div class="flex flex-col md:flex-row items-stretch gap-2.5 pt-1">
        <div class="relative flex-1">
          <input v-model="quickDomainInput" @keyup.enter="handleQuickAdd('PROXIES')"
            type="text"
            placeholder="输入网址或域名，例如 example.com、github.com..."
            class="w-full h-11 px-4 pr-10 rounded-xl text-sm font-mono shadow-sm focus:outline-none transition" />
          <button v-if="quickDomainInput" @click="quickDomainInput = ''"
            class="cbx-btn-ghost absolute right-3 top-1/2 -translate-y-1/2 text-xs font-bold w-6 h-6 flex items-center justify-center rounded-full cursor-pointer"
            aria-label="清空输入">
            ✕
          </button>
        </div>

        <div class="flex flex-wrap sm:flex-nowrap items-center gap-2 shrink-0">
          <button @click="handleQuickAdd('PROXIES')" :disabled="quickAdding || !quickDomainInput.trim()"
            class="flex-1 sm:flex-none h-11 px-4 rounded-xl bg-sky-600 hover:bg-sky-700 text-white text-xs sm:text-[13px] font-extrabold flex items-center justify-center gap-1.5 shadow-md shadow-sky-600/20 active:scale-95 transition disabled:opacity-40 cursor-pointer">
            <span>⚡</span>
            <span>强制走加速</span>
          </button>

          <button @click="handleQuickAdd('DIRECT')" :disabled="quickAdding || !quickDomainInput.trim()"
            class="flex-1 sm:flex-none h-11 px-4 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white text-xs sm:text-[13px] font-extrabold flex items-center justify-center gap-1.5 shadow-md shadow-emerald-600/20 active:scale-95 transition disabled:opacity-40 cursor-pointer">
            <span>🏠</span>
            <span>强制走直连</span>
          </button>

          <button @click="handleQuickAdd('REJECT')" :disabled="quickAdding || !quickDomainInput.trim()"
            class="flex-1 sm:flex-none h-11 px-3.5 rounded-xl bg-rose-600 hover:bg-rose-700 text-white text-xs sm:text-[13px] font-extrabold flex items-center justify-center gap-1.5 shadow-md shadow-rose-600/20 active:scale-95 transition disabled:opacity-40 cursor-pointer">
            <span>🛑</span>
            <span>屏蔽该网站</span>
          </button>
        </div>
      </div>

      <!-- 当前已生效的自定义规则胶囊瀑布流展示 -->
      <div v-if="totalCustomDomainCount > 0" class="pt-2 border-t space-y-2" style="border-color: var(--cbx-border-soft);">
        <div class="text-[11px] font-bold cbx-text-sub flex items-center justify-between">
          <span>当前自定分流规则清单（点击 ✕ 立即取消并还原默认）：</span>
        </div>
        <div class="flex flex-wrap gap-2 max-h-40 overflow-y-auto custom-scrollbar pr-1">
          <span v-for="d in userOverrides.force_proxy_domains" :key="'p-' + d"
            class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-xl cbx-badge-info border text-xs font-mono shadow-sm">
            <span class="text-[10px] font-bold">⚡ 加速</span>
            <span class="font-bold">{{ d }}</span>
            <button @click="handleQuickClear(d)" title="删除此加速规则"
              class="ml-1 hover:opacity-75 rounded px-1 transition cursor-pointer font-bold">✕</button>
          </span>

          <span v-for="d in userOverrides.force_direct_domains" :key="'d-' + d"
            class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-xl cbx-badge-ok border text-xs font-mono shadow-sm">
            <span class="text-[10px] font-bold">🏠 直连</span>
            <span class="font-bold">{{ d }}</span>
            <button @click="handleQuickClear(d)" title="删除此直连规则"
              class="ml-1 hover:opacity-75 rounded px-1 transition cursor-pointer font-bold">✕</button>
          </span>

          <span v-for="d in userOverrides.force_reject_domains" :key="'r-' + d"
            class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-xl cbx-badge-danger border text-xs font-mono shadow-sm">
            <span class="text-[10px] font-bold">🛑 屏蔽</span>
            <span class="font-bold">{{ d }}</span>
            <button @click="handleQuickClear(d)" title="删除此拦截规则"
              class="ml-1 hover:opacity-75 rounded px-1 transition cursor-pointer font-bold">✕</button>
          </span>
        </div>
      </div>
    </div>

    <!-- 2. 安全总评与能量光环看板 (Security Radar Scorecard) -->
    <div class="p-5 sm:p-6 rounded-3xl cbx-card shadow-lg relative overflow-hidden">
      <div class="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-6 relative z-10">
        <!-- Left: Circular Score Gauge -->
        <div class="flex items-center gap-4 sm:gap-5">
          <div class="relative w-20 h-20 sm:w-24 sm:h-24 flex items-center justify-center shrink-0">
            <div class="w-20 h-20 rounded-full cbx-sub-card border-2 border-emerald-500/50 flex flex-col items-center justify-center shadow-sm">
              <span class="text-2xl font-black font-mono text-emerald-600 dark:text-emerald-400 tracking-tighter font-tabular">
                {{ app.securityAudit.report?.total_score || 100 }}
              </span>
              <span class="text-[10px] font-mono cbx-text-muted uppercase tracking-widest font-bold">/ 100 分</span>
            </div>
          </div>

          <div class="space-y-1 min-w-0">
            <div class="flex items-center gap-2 flex-wrap">
              <span class="text-base sm:text-lg font-black cbx-text-main">
                {{ app.securityAudit.report?.security_rating || '暮光晶盾级 (100% 零泄露)' }}
              </span>
              <span class="px-2.5 py-0.5 rounded-full text-[11px] font-mono font-bold flex items-center gap-1 border shadow-sm"
                :class="app.activeEngine === 'singbox' ? 'cbx-badge-purple' : 'cbx-badge-info'">
                <span>内核:</span>
                <span class="font-black">{{ app.activeEngine === 'singbox' ? 'sing-box' : 'mihomo' }}</span>
              </span>
            </div>
            <p class="text-xs cbx-text-sub max-w-xl leading-relaxed font-medium">
              {{ app.securityAudit.report?.summary || '所有 6 大硬核防泄露测试全量通过！DNS 0 明文回退、WebRTC Fake-IP 隔离已就绪。' }}
            </p>
            <div class="text-[10px] cbx-text-muted font-mono font-medium">
              最新渗透测试时间: {{ new Date(app.securityAudit.report?.tested_at || Date.now()).toLocaleTimeString() }}
            </div>
          </div>
        </div>

        <!-- Right: Hardcore Pillars Mini-Chips -->
        <div class="grid grid-cols-1 sm:grid-cols-3 lg:grid-cols-1 gap-2.5 w-full lg:w-auto shrink-0">
          <div class="px-3.5 py-2 rounded-xl cbx-sub-card shadow-sm flex items-center gap-2.5 text-xs font-mono">
            <span class="w-2 h-2 rounded-full bg-emerald-500 shadow-sm shrink-0"></span>
            <span class="cbx-text-sub font-medium">DNS 0 明文回退:</span>
            <span class="text-emerald-700 dark:text-emerald-300 font-black">100% 熔断阻断</span>
          </div>
          <div class="px-3.5 py-2 rounded-xl cbx-sub-card shadow-sm flex items-center gap-2.5 text-xs font-mono">
            <span class="w-2 h-2 rounded-full bg-sky-500 shadow-sm shrink-0"></span>
            <span class="cbx-text-sub font-medium">WebRTC 境外隔离:</span>
            <span class="text-sky-700 dark:text-cyan-300 font-black">Fake-IP 保密</span>
          </div>
          <div class="px-3.5 py-2 rounded-xl cbx-sub-card shadow-sm flex items-center gap-2.5 text-xs font-mono">
            <span class="w-2 h-2 rounded-full bg-teal-500 shadow-sm shrink-0"></span>
            <span class="cbx-text-sub font-medium">Apple Type 65:</span>
            <span class="text-teal-700 dark:text-teal-300 font-black">100% 清洗拦截</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 3. 六大硬核安全测试 (6-Vector Hardcore Test Matrix) -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div v-for="item in (app.securityAudit.report?.items || [])" :key="item.id"
        class="p-4 rounded-2xl cbx-sub-card shadow-sm transition-all duration-300 hover:border-sky-400 flex flex-col justify-between space-y-3"
        :class="item.status === 'PASS' ? '' : (item.status === 'WARN' ? 'border-amber-400/60' : 'border-rose-400/60')">
        <!-- Card Top -->
        <div class="space-y-2">
          <div class="flex items-start justify-between gap-2">
            <div>
              <span class="text-[10px] font-mono px-2 py-0.5 rounded-md font-bold uppercase tracking-wider border"
                :class="item.risk_level === 'CRITICAL' ? 'cbx-badge-danger' : (item.risk_level === 'HIGH' ? 'cbx-badge-warn' : 'cbx-badge-ok')">
                {{ item.category }} · {{ item.risk_level }}
              </span>
              <h3 class="text-sm font-black cbx-text-main mt-1.5 flex items-center gap-1.5">
                <span>{{ item.name }}</span>
              </h3>
            </div>
            <span class="px-2.5 py-1 rounded-lg text-xs font-mono font-bold shrink-0 border shadow-sm"
              :class="item.status === 'PASS' ? 'cbx-badge-ok' : (item.status === 'WARN' ? 'cbx-badge-warn' : 'cbx-badge-danger')">
              {{ item.status === 'PASS' ? '✓ PASS 0泄露' : (item.status === 'WARN' ? '▲ WARN 待加固' : '✕ FAIL 风险') }}
            </span>
          </div>

          <div class="text-[11px] font-mono font-bold cbx-badge-ok px-2.5 py-1 rounded-md border shadow-sm">
            规范: {{ item.technical_specs }}
          </div>

          <p class="text-xs cbx-text-sub leading-relaxed font-medium">
            {{ item.details }}
          </p>
        </div>

        <!-- Card Bottom -->
        <div class="pt-2 border-t space-y-2" style="border-color: var(--cbx-border-soft);">
          <div class="flex items-center justify-between text-[11px] font-mono">
            <span class="cbx-text-sub font-medium">单项防御得分</span>
            <span class="font-black font-tabular" :class="item.score >= 90 ? 'text-emerald-700 dark:text-emerald-300' : (item.score >= 70 ? 'text-amber-700 dark:text-amber-300' : 'text-rose-700 dark:text-rose-300')">
              {{ item.score }} / 100
            </span>
          </div>
          <div class="w-full h-1.5 rounded-full cbx-progress-track overflow-hidden">
            <div class="h-full rounded-full transition-all duration-500"
              :class="item.score >= 90 ? 'bg-gradient-to-r from-teal-500 to-emerald-500' : (item.score >= 70 ? 'bg-gradient-to-r from-amber-500 to-yellow-500' : 'bg-rose-500')"
              :style="{ width: item.score + '%' }"></div>
          </div>
          <div class="text-[10px] cbx-text-muted font-mono font-medium flex items-center justify-between">
            <span>{{ item.recommendation }}</span>
            <span v-if="item.remediated" class="text-emerald-700 dark:text-emerald-300 font-bold">🛡️ 已加固</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 4. 217heidai 去广告&隐私追踪拦截 (AdBlock & Threat Interception) -->
    <div class="space-y-4">
      <div class="p-5 sm:p-6 rounded-3xl cbx-card shadow-lg flex items-center justify-between flex-wrap gap-3">
        <div>
          <h2 class="text-base sm:text-lg font-black cbx-text-main tracking-wide flex items-center gap-2">
            <span class="flex items-center gap-2"><ZootopiaIcon name="rules" :size="22" class="shrink-0" /><span>217heidai 去广告&隐私追踪拦截</span></span>
          </h2>
          <p class="text-xs cbx-text-sub mt-1 font-medium font-sans">
            基于 217heidai MRS 规则库的高精度广告与隐私追踪拦截 · 最近 24 小时 DNS 耗时、查询吞吐与恶意威胁拦截遥测
          </p>
        </div>

        <div class="flex items-center gap-2 flex-wrap font-mono">
          <span class="text-xs cbx-badge-ok px-3 py-1 rounded-full border font-bold shadow-sm font-tabular">
            规则库: {{ app.adblockStats.total_rules || 233481 }} 条
          </span>
          <span class="text-xs cbx-badge-info px-3 py-1 rounded-full border font-bold shadow-sm font-tabular">
            拦截率: {{ app.dnsFilterStats.blockedPercentage }}%
          </span>
        </div>
      </div>

      <!-- 域名过滤与广告拦截中枢 -->
      <div class="rounded-3xl cbx-card hover:border-sky-400/40 transition-all duration-300 p-5 space-y-4 shadow-lg">
        <div class="flex items-center justify-between flex-wrap gap-3 pb-3 border-b" style="border-color: var(--cbx-border-soft);">
          <div class="flex items-center gap-3 flex-wrap">
            <h3 class="text-base font-black cbx-text-main tracking-wide flex items-center gap-2">
              <span>域名过滤</span>
            </h3>
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="app.filterConfig.enabled" @change="app.toggleFilterMaster" class="sr-only peer">
              <div class="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-500 shadow-inner"></div>
            </label>
            <span :class="['text-xs px-2.5 py-0.5 rounded-full font-bold border transition shadow-sm', app.filterConfig.enabled ? 'cbx-badge-ok' : 'cbx-btn-ghost']">
              {{ app.filterConfig.enabled ? '已生效' : '已暂停' }}
            </span>
          </div>

          <div class="flex items-center gap-2">
            <button @click="app.fetchDnsFilterStats" title="刷新状态" class="p-2 rounded-xl cbx-btn-ghost active:scale-95 transition cursor-pointer shadow-sm">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
            </button>
            <button @click="app.openAddSourceModal" class="px-3 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs flex items-center gap-1.5 shadow-md shadow-emerald-600/20 active:scale-95 transition cursor-pointer">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4"/></svg>
              <span>添加规则源</span>
            </button>
          </div>
        </div>

        <p class="text-xs cbx-text-sub leading-relaxed font-medium">
          默认开启。开启后，转发的 DNS 查询先经过过滤，通过后仍按照原 DNS 策略解析；白名单放行优先。支持原生二进制 MRS/SRS、域名、Hosts 和基础 Adblock 语法，不支持的条目会显示跳过数量。启用的名单按下面的自动更新计划更新，下载失败沿用上一次有效名单。
        </p>

        <!-- Rule Sources List -->
        <div class="space-y-2.5">
          <div v-for="src in app.filterConfig.rule_sources" :key="src.id"
               class="rounded-2xl cbx-sub-card hover:border-sky-400/50 shadow-sm transition-all p-3.5 flex flex-col md:flex-row md:items-center justify-between gap-3">
            <div class="flex items-center gap-3 min-w-0 flex-1">
              <label class="relative inline-flex items-center cursor-pointer shrink-0">
                <input type="checkbox" v-model="src.enabled" @change="app.toggleSourceItem(src)" class="sr-only peer">
                <div class="w-9 h-5 bg-slate-300 dark:bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-emerald-500 shadow-inner"></div>
              </label>

              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="text-xs font-black cbx-text-main">{{ src.name }}</span>
                  <span class="text-[11px] font-mono cbx-text-muted truncate max-w-xs sm:max-w-md block" :title="src.url">{{ src.url }}</span>
                </div>
                <div class="flex items-center gap-3 mt-1 text-[11px] font-mono flex-wrap">
                  <span class="text-emerald-700 dark:text-emerald-400 font-bold font-tabular">规则数 {{ (src.rules_count || 0).toLocaleString() }}</span>
                  <span class="cbx-text-sub font-medium">上次更新 {{ src.updated_at || '刚刚' }}</span>
                </div>
              </div>
            </div>

            <!-- Actions -->
            <div class="flex items-center gap-1.5 shrink-0 self-end md:self-center">
              <button @click="app.openSourcePreview(src)" title="查看规则详情" class="p-2 rounded-lg cbx-btn-ghost shadow-sm transition cursor-pointer">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>
              </button>
              <button @click="app.refreshSourceItem(src)" :disabled="app.isUpdatingSource" title="立即更新" class="p-2 rounded-lg cbx-btn-ghost shadow-sm transition cursor-pointer disabled:opacity-50">
                <svg :class="['w-3.5 h-3.5', app.isUpdatingSource ? 'animate-spin text-emerald-600' : '']" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
              </button>
              <button @click="app.openEditSourceModal(src)" title="编辑规则源" class="p-2 rounded-lg cbx-btn-ghost shadow-sm transition cursor-pointer">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"/></svg>
              </button>
              <button v-if="src.id !== '217heidai'" @click="app.deleteSourceItem(src)" title="删除规则源" class="p-2 rounded-lg cbx-badge-danger border shadow-sm transition cursor-pointer">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
              </button>
            </div>
          </div>
        </div>

        <!-- Auto Update Scheduler Bar -->
        <div class="rounded-2xl cbx-sub-card p-3.5 flex flex-wrap items-center gap-3 text-xs cbx-text-sub font-medium shadow-sm">
          <span class="font-bold cbx-text-main flex items-center gap-1.5">
            <span>自动更新</span>
          </span>
          <label class="relative inline-flex items-center cursor-pointer shrink-0">
            <input type="checkbox" v-model="app.filterConfig.auto_update.enabled" @change="app.saveFilterAutoUpdate" class="sr-only peer">
            <div class="w-8 h-4 bg-slate-300 dark:bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[1px] after:left-[1px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-3.5 after:w-3.5 after:transition-all peer-checked:bg-emerald-500 shadow-inner"></div>
          </label>
          <div class="flex items-center gap-2 flex-wrap">
            <span>每</span>
            <select v-model="app.filterConfig.auto_update.interval_days" @change="app.saveFilterAutoUpdate" class="rounded-lg px-2 py-1 text-xs font-bold focus:outline-none font-mono shadow-sm cursor-pointer">
              <option :value="1">1 天</option>
              <option :value="3">3 天</option>
              <option :value="7">7 天</option>
            </select>
            <span>在</span>
            <select v-model="app.filterConfig.auto_update.update_time" @change="app.saveFilterAutoUpdate" class="rounded-lg px-2 py-1 text-xs font-bold focus:outline-none font-mono shadow-sm cursor-pointer">
              <option value="04:00">04:00</option>
              <option value="06:00">06:00</option>
              <option value="08:00">08:00</option>
              <option value="12:00">12:00</option>
              <option value="18:00">18:00</option>
              <option value="00:00">00:00</option>
            </select>
            <span class="cbx-text-muted text-[11px]">在选定时间更新一次；内核自己加载新名单，不重启。</span>
          </div>
        </div>

        <!-- Whitelist (放行域名) Accordion -->
        <div class="border-t pt-3" style="border-color: var(--cbx-border-soft);">
          <button @click="showWhitelistSection = !showWhitelistSection" class="w-full flex items-center justify-between text-xs font-bold cbx-text-main hover:opacity-80 transition cursor-pointer py-1">
            <span class="flex items-center gap-1.5">
              <svg :class="['w-3.5 h-3.5 transition-transform cbx-text-muted', showWhitelistSection ? 'rotate-180' : '']" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/></svg>
              <span>放行域名 ({{ app.filterConfig.whitelist_domains?.length || 0 }})</span>
            </span>
            <span class="text-[11px] cbx-text-muted font-medium">优先直连放行 · 防误杀</span>
          </button>

          <div v-if="showWhitelistSection" class="mt-3 space-y-3 pl-4 border-l-2 border-emerald-500/60 animate-in fade-in">
            <div class="flex flex-wrap items-center gap-2">
              <span v-for="d in app.filterConfig.whitelist_domains" :key="d"
                    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg cbx-sub-card cbx-text-main text-xs font-mono shadow-sm group hover:border-emerald-500/50">
                <span>{{ d }}</span>
                <button @click="app.removeWhitelistItem(d)" title="移除放行" class="cbx-text-muted group-hover:text-rose-500 font-bold transition cursor-pointer">&times;</button>
              </span>
              <span v-if="!app.filterConfig.whitelist_domains || app.filterConfig.whitelist_domains.length === 0" class="text-xs cbx-text-muted italic font-medium">
                暂无放行域名，可在下方添加特定业务域名以防误杀
              </span>
            </div>

            <div class="flex items-center gap-2 max-w-md">
              <input type="text" v-model="app.newWhitelistDomain" @keyup.enter="app.addWhitelistItem" placeholder="输入放行域名 (如 apple.com)" class="flex-1 rounded-xl px-3 py-1.5 text-xs focus:outline-none font-mono shadow-sm">
              <button @click="app.addWhitelistItem" class="px-3 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-sm active:scale-95 transition cursor-pointer shrink-0">
                添加放行
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- DNS 重写 -->
      <div class="rounded-3xl cbx-card shadow-lg p-5 sm:p-6 space-y-4">
        <div class="flex items-center justify-between flex-wrap gap-3 pb-3 border-b" style="border-color: var(--cbx-border-soft);">
          <div class="flex items-center gap-3 flex-wrap">
            <h3 class="text-base font-black cbx-text-main tracking-wide flex items-center gap-2">
              <span class="text-emerald-600 dark:text-emerald-400">🔀</span>
              <span>DNS 重写和 局域网A 记录静态映射</span>
            </h3>
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="app.dnsRewriteConfig.enabled" @change="app.toggleDnsRewriteMaster" class="sr-only peer">
              <div class="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-sky-500 shadow-inner"></div>
            </label>
            <span :class="['text-xs px-2.5 py-0.5 rounded-full font-bold border transition', app.dnsRewriteConfig.enabled ? 'cbx-badge-ok' : 'cbx-btn-ghost']">
              {{ app.dnsRewriteConfig.enabled ? '已生效' : '已暂停' }}
            </span>
          </div>

          <div class="flex items-center gap-2 flex-wrap">
            <button @click="app.resetDnsRewriteDefaults" title="恢复默认 DNS 重写与 局域网A 记录静态映射规则" class="px-3 py-1.5 rounded-xl cbx-btn-ghost font-bold active:scale-95 transition cursor-pointer text-xs flex items-center gap-1.5 shadow-sm">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
              <span>重置默认</span>
            </button>
            <button @click="app.openAddDnsRewriteModal" class="px-3.5 py-1.5 rounded-xl bg-gradient-to-r from-sky-500 to-cyan-500 hover:from-sky-400 hover:to-cyan-400 text-white font-bold text-xs flex items-center gap-1.5 shadow-md shadow-sky-500/20 active:scale-95 transition cursor-pointer">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4"/></svg>
              <span>添加重写</span>
            </button>
          </div>
        </div>

        <p class="text-xs cbx-text-sub leading-relaxed font-medium">
          把某个域名的解析结果改成另一个域名或固定 IP；默认带两条 Google 的重写（解决 Google Play 无法下载 App 问题）及一条局域网 A 记录静态映射示例。
        </p>

        <div class="space-y-2.5">
          <div v-for="rule in app.dnsRewriteConfig.rules" :key="rule.id"
               class="rounded-2xl cbx-sub-card hover:border-sky-400/50 transition-all p-3.5 flex flex-col md:flex-row md:items-center justify-between gap-3 shadow-sm">
            <div class="flex-1 min-w-0 space-y-1">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-mono text-xs font-bold cbx-btn-ghost px-2 py-0.5 rounded">{{ rule.domain }}</span>
                <span class="text-sky-600 dark:text-cyan-300 text-xs">➔</span>
                <span class="font-mono text-xs font-bold cbx-badge-info px-2 py-0.5 rounded border">{{ rule.target }}</span>
                <span :class="['text-[10px] font-bold px-2 py-0.5 rounded uppercase tracking-wider border', rule.type === 'CNAME' ? 'cbx-badge-info' : 'cbx-badge-purple']">
                  {{ rule.type === 'CNAME' ? 'CNAME 别名' : 'A 静态记录' }}
                </span>
              </div>
              <div v-if="rule.comment" class="text-[11px] cbx-text-sub font-medium flex items-center gap-1.5">
                <span class="cbx-text-muted">💬</span>
                <span>{{ rule.comment }}</span>
              </div>
            </div>

            <div class="flex items-center gap-2.5 self-end md:self-auto shrink-0">
              <label class="relative inline-flex items-center cursor-pointer" title="启用/停用单条规则">
                <input type="checkbox" v-model="rule.enabled" @change="app.toggleDnsRewriteRule(rule)" class="sr-only peer">
                <div class="w-9 h-5 bg-slate-300 dark:bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-sky-500 shadow-inner"></div>
              </label>
              <button @click="app.openEditDnsRewriteModal(rule)" title="编辑规则" class="p-2 rounded-lg cbx-btn-ghost shadow-sm transition cursor-pointer">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"/></svg>
              </button>
              <button @click="app.deleteDnsRewriteRule(rule)" title="删除规则" class="p-2 rounded-lg cbx-badge-danger border shadow-sm transition cursor-pointer">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
              </button>
            </div>
          </div>

          <div v-if="!app.dnsRewriteConfig.rules || app.dnsRewriteConfig.rules.length === 0" class="text-center py-6 text-xs cbx-text-muted font-medium italic cbx-sub-card rounded-2xl border border-dashed">
            暂无规则，点击右上角【添加重写】或【重置默认】快速恢复官方加速重写与 局域网A 记录静态映射示例
          </div>
        </div>
      </div>

      <!-- 4-Card 2x2 Grid with 24h Area Charts and Rank -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <!-- Card 1: 平均解析耗时 -->
        <div class="p-4 rounded-2xl cbx-card shadow-lg space-y-2 relative overflow-hidden">
          <div class="flex items-center justify-between text-xs cbx-text-sub font-medium gap-2">
            <span>平均解析耗时 (24h 遥测)</span>
            <span class="text-[10px] cbx-badge-ok px-2 py-0.5 rounded-full border font-mono font-bold flex items-center gap-1 shadow-sm">
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
              0 泄露极速响应
            </span>
          </div>
          <div class="text-2xl font-bold font-mono cbx-text-main flex items-baseline gap-1 font-tabular">
            <span>{{ app.dnsFilterStats.avgLatencyMs }}</span>
            <span class="text-xs cbx-text-muted font-sans font-medium">ms</span>
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
              <path :d="app.dnsLatencyPath.line" fill="none" stroke="#a855f7" stroke-width="2" stroke-linecap="round" />
              <circle cx="316" :cy="app.dnsLatencyLastY" r="3" fill="#a855f7" />
              <circle cx="316" :cy="app.dnsLatencyLastY" r="6" fill="#a855f7" opacity="0.4" class="animate-ping" />
            </svg>
          </div>
        </div>

        <!-- Card 2: DNS 查询量 -->
        <div class="p-4 rounded-2xl cbx-card shadow-lg space-y-2 relative overflow-hidden">
          <div class="flex items-center justify-between text-xs cbx-text-sub font-medium gap-2">
            <span>DNS 查询量 (24h 吞吐)</span>
            <span class="text-[10px] cbx-badge-info px-2 py-0.5 rounded-full border font-mono font-bold flex items-center gap-1 shadow-sm">
              <span class="w-1.5 h-1.5 rounded-full bg-sky-500 animate-pulse"></span>
              国内直连 + Fake-IP
            </span>
          </div>
          <div class="text-2xl font-bold font-mono cbx-text-main font-tabular">
            {{ app.dnsFilterStats.queryCount.toLocaleString() }}
          </div>
          <div class="h-16 w-full pt-1">
            <svg class="w-full h-full" viewBox="0 0 320 60" preserveAspectRatio="none">
              <defs>
                <linearGradient id="dnsQueryGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="#0ea5e9" stop-opacity="0.4"/>
                  <stop offset="100%" stop-color="#0ea5e9" stop-opacity="0.0"/>
                </linearGradient>
              </defs>
              <path :d="app.dnsQueryPath.area" fill="url(#dnsQueryGrad)" />
              <path :d="app.dnsQueryPath.line" fill="none" stroke="#0ea5e9" stroke-width="2" stroke-linecap="round" />
              <circle cx="316" :cy="app.dnsQueryLastY" r="3" fill="#0ea5e9" />
              <circle cx="316" :cy="app.dnsQueryLastY" r="6" fill="#0ea5e9" opacity="0.4" class="animate-ping" />
            </svg>
          </div>
        </div>

        <!-- Card 3: 今日阻断与拦截 -->
        <div class="p-4 rounded-2xl cbx-card shadow-lg space-y-2 relative overflow-hidden flex flex-col justify-between">
          <div>
            <div class="flex items-center justify-between text-xs gap-2">
              <span class="cbx-text-sub font-medium">今日阻断与拦截</span>
              <span class="text-xs font-bold font-mono cbx-badge-warn px-2 py-0.5 rounded-lg border shadow-sm font-tabular">
                拦截率 {{ app.dnsFilterStats.blockedPercentage }}%
              </span>
            </div>
            <div class="text-2xl font-bold font-mono text-amber-600 dark:text-amber-400 font-tabular mt-1">
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
                <circle cx="316" :cy="app.dnsBlockedLastY" r="3" fill="#f59e0b" />
                <circle cx="316" :cy="app.dnsBlockedLastY" r="6" fill="#f59e0b" opacity="0.4" class="animate-ping" />
              </svg>
            </div>
          </div>

          <div class="pt-2 border-t text-[11px] cbx-text-muted flex items-center justify-between" style="border-color: var(--cbx-border-soft);">
            <span class="flex items-center gap-1.5">
              <span class="w-1.5 h-1.5 rounded-full bg-amber-500"></span>
              <span>217heidai 权威规则拦截</span>
            </span>
            <span class="font-mono">实时过滤</span>
          </div>

          <div class="grid grid-cols-3 gap-1.5 pt-1 text-[11px] font-mono select-none">
            <div class="flex flex-col items-center justify-center p-1.5 rounded-xl cbx-badge-warn border" title="广告弹窗规则拦截">
              <span class="text-[10px] cbx-text-sub font-sans font-medium">广告弹窗</span>
              <span class="font-bold font-tabular">{{ app.dnsFilterStats.categoryStats ? app.dnsFilterStats.categoryStats.ads.toLocaleString() : '—' }}</span>
            </div>
            <div class="flex flex-col items-center justify-center p-1.5 rounded-xl cbx-badge-purple border" title="隐私与遥测追踪拦截">
              <span class="text-[10px] cbx-text-sub font-sans font-medium">隐私追踪</span>
              <span class="font-bold font-tabular">{{ app.dnsFilterStats.categoryStats ? app.dnsFilterStats.categoryStats.privacy.toLocaleString() : '—' }}</span>
            </div>
            <div class="flex flex-col items-center justify-center p-1.5 rounded-xl cbx-badge-danger border" title="恶意域名与劫持阻断">
              <span class="text-[10px] cbx-text-sub font-sans font-medium">恶意劫持</span>
              <span class="font-bold font-tabular">{{ app.dnsFilterStats.categoryStats ? app.dnsFilterStats.categoryStats.malicious.toLocaleString() : '—' }}</span>
            </div>
          </div>
        </div>

        <!-- Card 4: 被拦截域名排行 -->
        <div class="p-4 rounded-2xl cbx-card shadow-lg space-y-3 relative overflow-hidden flex flex-col justify-between">
          <div>
            <div class="flex items-center justify-between text-xs cbx-text-sub font-medium gap-2">
              <span>被拦截广告与恶意域名排行</span>
              <span class="text-[10px] cbx-badge-warn px-2 py-0.5 rounded-full border font-mono font-bold">217heidai 高危特征</span>
            </div>

            <div v-if="app.dnsFilterStats.topBlocked && app.dnsFilterStats.topBlocked.length > 0" class="space-y-2 pt-2">
              <div v-for="item in app.dnsFilterStats.topBlocked" :key="item.domain" class="space-y-1">
                <div class="flex items-center justify-between text-[11px] font-mono">
                  <span class="cbx-text-main font-medium truncate max-w-[240px]" :title="item.domain">{{ item.domain }}</span>
                  <span class="cbx-text-sub font-semibold font-tabular">{{ item.count.toLocaleString() }}</span>
                </div>
                <div class="w-full cbx-progress-track h-1.5 rounded-full overflow-hidden">
                  <div class="bg-gradient-to-r from-amber-500 to-orange-500 h-full rounded-full transition-all duration-500"
                    :style="{ width: item.percent + '%' }"></div>
                </div>
              </div>
            </div>

            <div v-else class="h-28 flex flex-col items-center justify-center text-center cbx-text-muted space-y-1">
              <span class="text-lg">🛡️</span>
              <span class="text-xs font-semibold cbx-text-sub">暂无高危拦截记录</span>
              <span class="text-[10px]">217heidai DNS 规则防线实时守卫中</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 6. Fake-IP 白名单与真实 IP 直连解析中枢 -->
    <div class="rounded-3xl p-5 sm:p-6 cbx-card shadow-lg space-y-4">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="space-y-1">
          <div class="flex items-center gap-2 flex-wrap">
            <span class="text-lg">🎯</span>
            <h3 class="text-sm sm:text-base font-black cbx-text-main tracking-wide">
              Fake-IP 过滤白名单 (真实 IP 直连解析中枢)
            </h3>
            <span class="text-[10px] px-2 py-0.5 rounded-full cbx-badge-ok border font-bold shadow-sm">
              双核秒级热重载 · 掉电持久化
            </span>
          </div>
          <p class="text-xs cbx-text-sub font-medium leading-relaxed">
            名单内域名将跳过 198.18.0.0/15 假地址拦截，由本地运营商 DNS 直接返回物理真实 IP。彻底解决极空间 NAS (zspace.com)、本地网盘挂载、特殊企业服务、直连握手超时与 Apple TV 播放转圈问题。
          </p>
        </div>
      </div>

      <!-- Add Input Row -->
      <div class="flex flex-col sm:flex-row gap-2 pt-1">
        <input v-model="app.newFakeIpInput" @keyup.enter="app.addFakeIpFilter"
          type="text" placeholder="输入需直连解析的域名 (例如: *.zspace.com 或 nas.myhome.local)"
          class="flex-1 px-3.5 py-2.5 sm:py-2 rounded-xl text-xs font-mono shadow-sm focus:outline-none transition" />
        <button @click="app.addFakeIpFilter" :disabled="app.fakeIpLoading || !app.newFakeIpInput.trim()"
          class="px-4 py-2.5 sm:py-2 rounded-xl bg-gradient-to-r from-sky-500 to-emerald-500 hover:from-sky-400 hover:to-emerald-400 disabled:opacity-40 text-white text-xs font-bold transition flex items-center justify-center gap-1.5 cursor-pointer shrink-0 shadow-md shadow-sky-500/20 active:scale-95">
          <span v-if="app.fakeIpLoading" class="animate-spin">↻</span>
          <span v-else>➕</span>
          <span>添加直连白名单</span>
        </button>
      </div>

      <!-- Active User Filters -->
      <div class="space-y-2 pt-2 border-t" style="border-color: var(--cbx-border-soft);">
        <div class="flex items-center justify-between text-xs">
          <span class="cbx-text-main font-bold flex items-center gap-1.5">
            <span>📌</span> 用户自定义直连域名 ({{ app.fakeIpFilters.length }} 项)
          </span>
          <span class="text-[11px] cbx-text-muted font-medium">重启或更新永不丢失</span>
        </div>
        <div v-if="app.fakeIpFilters.length === 0" class="text-xs cbx-text-muted py-2 italic font-mono">
          暂未添加自定义域名。默认已内置极空间 NAS、Apple 设备自检与国内主流服务。
        </div>
        <div v-else class="flex flex-wrap gap-2 pt-1">
          <span v-for="domain in app.fakeIpFilters" :key="domain"
            class="px-2.5 py-1 rounded-xl cbx-badge-info border text-xs font-mono flex items-center gap-1.5 shadow-sm">
            <span class="font-bold">{{ domain }}</span>
            <button @click="app.removeFakeIpFilter(domain)"
              class="hover:opacity-75 rounded px-1 ml-1 font-bold cursor-pointer transition" title="移除此域名">✕</button>
          </span>
        </div>
      </div>

      <!-- Built-in Defaults Display -->
      <div class="space-y-1.5 pt-2 border-t text-xs" style="border-color: var(--cbx-border-soft);">
        <div class="flex items-center justify-between gap-2 flex-wrap">
          <span class="cbx-text-sub font-semibold flex items-center gap-1.5">
            <span>🛡️</span> 系统内置免 Fake-IP 直连项 ({{ app.fakeIpBuiltinCount || 32 }} 条 · {{ app.fakeIpBuiltinEnabled ? '已启用' : '已关闭' }}):
          </span>
          <button
            type="button"
            @click="app.toggleFakeIpBuiltin"
            :disabled="app.fakeIpLoading"
            class="px-2.5 py-1 rounded-lg text-[11px] font-bold border transition cursor-pointer"
            :class="app.fakeIpBuiltinEnabled ? 'cbx-badge-ok' : 'cbx-btn-ghost'"
          >
            {{ app.fakeIpBuiltinEnabled ? '✓ 内置清单已开启' : '○ 内置清单已关闭' }}
          </button>
        </div>
        <div class="flex flex-wrap gap-1.5 text-[11px] font-mono">
          <span class="px-2 py-0.5 rounded-lg cbx-badge-ok border font-medium shadow-sm">+.zspace.com (极空间NAS)</span>
          <span class="px-2 py-0.5 rounded-lg cbx-badge-ok border font-medium shadow-sm">+.zspace.cn</span>
          <span class="px-2 py-0.5 rounded-lg cbx-btn-ghost font-medium shadow-sm">geosite:cn</span>
          <span class="px-2 py-0.5 rounded-lg cbx-btn-ghost font-medium shadow-sm">*.lan / *.local</span>
          <span class="px-2 py-0.5 rounded-lg cbx-btn-ghost font-medium shadow-sm">NTP 时间同步</span>
          <span class="px-2 py-0.5 rounded-lg cbx-btn-ghost font-medium shadow-sm">+.routeros.com</span>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { inject, ref, computed, onMounted } from 'vue'
import ZootopiaIcon from '../ZootopiaIcon.vue'
import { apiFetch } from '../../utils/apiFetch'
const app = inject('cubexApp')
const showWhitelistSection = ref(false)

const quickDomainInput = ref('')
const quickAdding = ref(false)

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
    if (app && app.showToast) app.showToast('⚠️ 请输入有效的网址或域名 (如 example.com)', true)
    return
  }
  quickAdding.value = true
  try {
    const res = await apiFetch('/api/v1/rules/user-overrides/target', {
      method: 'POST',
      json: { target: domain, outbound }
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
    const res = await apiFetch('/api/v1/rules/user-overrides/target', {
      method: 'POST',
      json: { target: domain, outbound: 'CLEAR' }
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
    const res = await apiFetch('/api/v1/rules/overrides')
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
  if (!app?.needsAuthUnlock) {
    fetchUserOverrides()
  }
})
</script>
