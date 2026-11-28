<template>
  <div v-if="app.showImportModal" @click.self="app.showImportModal = false" class="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-3 sm:p-4 overflow-y-auto animate-in fade-in">
    <div class="cbx-modal-shell rounded-3xl p-5 sm:p-7 max-w-3xl w-full space-y-5 shadow-2xl max-h-[92vh] flex flex-col">
      <!-- Modal Header -->
      <div class="flex items-center justify-between gap-2 pb-3.5 border-b" style="border-color: var(--cbx-border-soft);">
        <div class="flex items-center gap-3 min-w-0">
          <span class="text-xl p-2 rounded-2xl cbx-badge-info border shrink-0">⚡</span>
          <div class="min-w-0">
            <h3 class="text-base font-black cbx-text-main flex items-center gap-2 flex-wrap">
              <span>订阅源与自建节点批量导入中枢</span>
              <span class="text-[10px] px-2 py-0.5 rounded-full cbx-badge-info border font-mono font-bold">
                Batch Multi-Import
              </span>
            </h3>
            <p class="text-[11px] cbx-text-sub mt-0.5">支持每行一条、直接指定名称，实时智能解析并一键合并挂载到核心代理集群</p>
          </div>
        </div>
        <button @click="app.showImportModal = false" class="cbx-btn-ghost px-3 py-1.5 rounded-xl text-sm font-bold shrink-0 cursor-pointer" aria-label="关闭导入弹窗">✕</button>
      </div>

      <!-- 3-Segment Navigation Capsule -->
      <div class="flex flex-wrap sm:flex-nowrap gap-1 p-1 cbx-sub-card rounded-2xl text-xs font-mono shrink-0">
        <button @click="app.subModalTab = 'sub'" :class="['flex-1 min-w-[120px] py-2 px-2 rounded-xl font-bold transition flex items-center justify-center gap-1.5 cursor-pointer', app.subModalTab === 'sub' ? 'bg-gradient-to-r from-sky-600 to-cyan-500 text-white shadow-md shadow-sky-500/20' : 'cbx-text-sub hover:cbx-text-main']">
          <span>🌐 批量导入订阅</span>
          <span v-if="app.parsedBatchSubs.length > 0" class="px-1.5 py-0.5 rounded-full text-[10px] cbx-badge-ok font-mono">
            {{ app.parsedBatchSubs.length }}
          </span>
        </button>

        <button @click="app.subModalTab = 'node'" :class="['flex-1 min-w-[120px] py-2 px-2 rounded-xl font-bold transition flex items-center justify-center gap-1.5 cursor-pointer', app.subModalTab === 'node' ? 'bg-gradient-to-r from-sky-600 to-cyan-500 text-white shadow-md shadow-sky-500/20' : 'cbx-text-sub hover:cbx-text-main']">
          <span>🔗 批量导入自建节点</span>
          <span v-if="app.parsedBatchNodes.length > 0" class="px-1.5 py-0.5 rounded-full text-[10px] cbx-badge-purple font-mono">
            {{ app.parsedBatchNodes.length }}
          </span>
        </button>

        <button @click="app.subModalTab = 'list'" :class="['flex-1 min-w-[160px] py-2 px-2 rounded-xl font-bold transition flex items-center justify-center gap-1.5 cursor-pointer', app.subModalTab === 'list' ? 'bg-gradient-to-r from-sky-600 to-cyan-500 text-white shadow-md shadow-sky-500/20' : 'cbx-text-sub hover:cbx-text-main']">
          <span class="truncate">📋 清单 ({{ app.ir.subscriptions?.length || 0 }} 订阅 · {{ app.customNodes.length }} 节点)</span>
        </button>
      </div>

      <!-- Tab 1: 批量导入订阅 -->
      <div v-if="app.subModalTab === 'sub'" class="space-y-3.5 overflow-y-auto pr-1 flex-1">
        <!-- Format Helper Banner -->
        <div class="p-3 cbx-sub-card rounded-2xl text-xs space-y-1.5">
          <div class="font-bold flex items-center gap-1.5 cbx-text-main font-mono">
            <span>📋 多条订阅格式说明 (每行一条，可直接在行内指定自定义名称)：</span>
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-1.5 text-[11px] font-mono cbx-text-sub pl-1">
            <div>• <span class="cbx-text-main font-bold">名称, 订阅链接</span> (如: <code>香港主力, https://example.com/sub</code>)</div>
            <div>• <span class="cbx-text-main font-bold">名称 | 订阅链接</span> (如: <code>备用专线 | https://example.com/clash</code>)</div>
            <div>• <span class="cbx-text-main font-bold">订阅链接#名称</span> (如: <code>https://example.com/api#日本专线</code>)</div>
            <div>• <span class="cbx-text-main font-bold">直接粘贴纯链接</span> (系统自动按序号命名: <code>订阅源 1, 2...</code>)</div>
          </div>
        </div>

        <!-- Multi-line Textarea -->
        <div>
          <div class="flex flex-wrap items-center justify-between gap-2 text-xs font-semibold cbx-text-main mb-1.5">
            <span class="flex items-center gap-1.5 flex-wrap">
              <span>粘贴订阅源列表 (每行一条)</span>
              <span class="text-[11px] cbx-text-muted font-normal">支持 Clash / sing-box / V2Ray 协议</span>
            </span>
            <span v-if="app.parsedBatchSubs.length > 0" class="font-mono text-xs font-bold flex items-center gap-1">
              <span class="cbx-text-sub">⚡ 实时识别:</span>
              <span class="px-2 py-0.5 rounded border cbx-badge-ok">{{ app.parsedBatchSubs.length }} 条有效订阅</span>
            </span>
          </div>

          <textarea v-model="app.batchSubText" rows="6"
            placeholder="每行一条配置订阅链接。支持直接在行内指定自定义标签，例如：&#10;业务分流-A, https://example.com/sub/profile1&#10;业务分流-B | https://example.com/sub/profile2&#10;https://example.com/sub/profile3#专属通道"
            class="w-full rounded-2xl p-3 text-xs font-mono outline-none shadow-inner resize-y leading-relaxed" />
        </div>

        <!-- Real-time Live Preview List -->
        <div v-if="app.parsedBatchSubs.length > 0" class="space-y-1.5 cbx-sub-card p-3 rounded-2xl">
          <div class="text-[11px] font-bold cbx-text-main flex items-center justify-between font-mono">
            <span>⚡ 实时解析预览 (共 {{ app.parsedBatchSubs.length }} 条订阅将一次性合并导入)：</span>
            <span class="cbx-text-muted text-[10px]">单次原子热重载 · 零频闪</span>
          </div>
          <div class="max-h-36 overflow-y-auto space-y-1.5 pr-1 font-mono text-[11px]">
            <div v-for="(item, idx) in app.parsedBatchSubs" :key="idx" class="flex items-center justify-between gap-2 p-2 rounded-xl cbx-sub-card">
              <div class="flex items-center gap-2 truncate">
                <span class="cbx-text-muted text-[10px] w-5 text-right shrink-0">{{ idx + 1 }}.</span>
                <span class="px-2 py-0.5 rounded border cbx-badge-info text-xs font-bold shrink-0">
                  ✈️ {{ item.name }}
                </span>
                <span class="cbx-text-sub truncate text-[11px]">{{ item.url }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex flex-wrap gap-2 pt-2 border-t" style="border-color: var(--cbx-border-soft);">
          <button @click="app.submitBatchSubs" :disabled="app.parsedBatchSubs.length === 0 || app.batchImportingSubs"
            class="flex-1 min-w-[180px] py-2.5 px-4 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white font-bold text-xs shadow-lg shadow-sky-500/20 active:scale-95 transition flex items-center justify-center gap-1.5 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer">
            <span v-if="app.batchImportingSubs" class="inline-block animate-spin">↻</span>
            <span>{{ app.batchImportingSubs ? '正在并发拉取并解析所有订阅...' : `🚀 立即一键导入 (${app.parsedBatchSubs.length} 条订阅源)` }}</span>
          </button>
          <button v-if="app.batchSubText" @click="app.batchSubText = ''" class="px-3.5 py-2.5 rounded-xl cbx-btn-ghost text-xs font-semibold transition cursor-pointer">
            清空
          </button>
          <button @click="app.showImportModal = false" class="px-4 py-2.5 rounded-xl cbx-btn-ghost text-xs font-semibold transition cursor-pointer">
            取消
          </button>
        </div>
      </div>

      <!-- Tab 2: 批量导入自建单节点 -->
      <div v-else-if="app.subModalTab === 'node'" class="space-y-3.5 overflow-y-auto pr-1 flex-1">
        <!-- Format Helper Banner -->
        <div class="p-3 cbx-sub-card rounded-2xl text-xs space-y-1.5">
          <div class="font-bold flex items-center gap-1.5 cbx-text-main font-mono">
            <span>📋 多条自建节点格式说明 (每行一条，可直接在行内指定自定义名称)：</span>
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-1.5 text-[11px] font-mono cbx-text-sub pl-1">
            <div>• <span class="cbx-text-main font-bold">自定义标签, 节点链接</span> (如: <code>节点A, 协议://...</code>)</div>
            <div>• <span class="cbx-text-main font-bold">自定义标签 | 节点链接</span> (如: <code>节点C | 协议://...</code>)</div>
            <div>• <span class="cbx-text-main font-bold">节点链接#自定义标签</span> (如: <code>协议://...#节点D</code>)</div>
            <div>• <span class="cbx-text-main font-bold">直接粘贴单节点链接</span> (支持常见通用代理协议链接)</div>
          </div>
        </div>

        <!-- Multi-line Textarea -->
        <div>
          <div class="flex flex-wrap items-center justify-between gap-2 text-xs font-semibold cbx-text-main mb-1.5">
            <span class="flex items-center gap-1.5 flex-wrap">
              <span>粘贴自建节点列表 (每行一条)</span>
              <span class="text-[11px] cbx-text-muted font-normal">支持 Shadowsocks, VMess, VLESS, Trojan, Hysteria2, TUIC</span>
            </span>
            <span v-if="app.parsedBatchNodes.length > 0" class="font-mono text-xs font-bold flex items-center gap-1">
              <span class="cbx-text-sub">⚡ 实时识别:</span>
              <span class="px-2 py-0.5 rounded border cbx-badge-purple">{{ app.parsedBatchNodes.length }} 个有效节点</span>
            </span>
          </div>

          <textarea v-model="app.batchNodeText" rows="6"
            placeholder="每行一条自建节点链接，支持指定自定义名称。例如：&#10;节点A, 协议://凭据@域名:端口#别名&#10;节点B | 协议://凭据@域名:端口&#10;协议://凭据@域名:端口#节点C"
            class="w-full rounded-2xl p-3 text-xs font-mono outline-none shadow-inner resize-y leading-relaxed" />
        </div>

        <!-- Real-time Live Preview List -->
        <div v-if="app.parsedBatchNodes.length > 0" class="space-y-1.5 cbx-sub-card p-3 rounded-2xl">
          <div class="text-[11px] font-bold cbx-text-main flex items-center justify-between font-mono">
            <span>⚡ 实时解析预览 (共 {{ app.parsedBatchNodes.length }} 个节点将合并导入并归入「🏠 自建」集群)：</span>
            <span class="cbx-text-muted text-[10px]">单次原子热重载 · 自动去重</span>
          </div>
          <div class="max-h-36 overflow-y-auto space-y-1.5 pr-1 font-mono text-[11px]">
            <div v-for="(item, idx) in app.parsedBatchNodes" :key="idx" class="flex items-center justify-between gap-2 p-2 rounded-xl cbx-sub-card">
              <div class="flex items-center gap-2 truncate">
                <span class="cbx-text-muted text-[10px] w-5 text-right shrink-0">{{ idx + 1 }}.</span>
                <span class="px-1.5 py-0.5 rounded border cbx-badge-purple text-[10px] font-bold shrink-0 uppercase">
                  {{ item.protocol }}
                </span>
                <span class="px-2 py-0.5 rounded cbx-badge-info border text-xs font-bold shrink-0 truncate max-w-[140px]">
                  🔗 {{ item.name }}
                </span>
                <span class="cbx-text-sub truncate text-[11px]">{{ item.link }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex flex-wrap gap-2 pt-2 border-t" style="border-color: var(--cbx-border-soft);">
          <button @click="app.submitBatchNodes" :disabled="app.parsedBatchNodes.length === 0 || app.batchAddingNodes"
            class="flex-1 min-w-[180px] py-2.5 px-4 rounded-xl bg-gradient-to-r from-purple-600 to-indigo-600 hover:from-purple-500 hover:to-indigo-500 text-white font-bold text-xs shadow-lg shadow-purple-500/20 active:scale-95 transition flex items-center justify-center gap-1.5 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer">
            <span v-if="app.batchAddingNodes" class="inline-block animate-spin">↻</span>
            <span>{{ app.batchAddingNodes ? '正在添加自建节点并重载核心...' : `➕ 立即一键添加 (${app.parsedBatchNodes.length} 个节点)` }}</span>
          </button>
          <button v-if="app.batchNodeText" @click="app.batchNodeText = ''" class="px-3.5 py-2.5 rounded-xl cbx-btn-ghost text-xs font-semibold transition cursor-pointer">
            清空
          </button>
          <button @click="app.showImportModal = false" class="px-4 py-2.5 rounded-xl cbx-btn-ghost text-xs font-semibold transition cursor-pointer">
            取消
          </button>
        </div>
      </div>

      <!-- Tab 3: 订阅源与自建节点清单 -->
      <div v-else class="space-y-3.5 pt-1 max-h-[60vh] overflow-y-auto pr-1 flex-1">
        <!-- Subscriptions list -->
        <div class="space-y-2">
          <div class="text-xs font-bold cbx-text-main flex items-center justify-between gap-2 flex-wrap">
            <span class="flex items-center gap-1.5">
              <span>✈️ 已保存订阅源</span>
              <span class="cbx-badge-ok border px-1.5 py-0.2 rounded font-mono text-[11px]">{{ app.ir.subscriptions?.length || 0 }} 个</span>
            </span>
            <button @click="app.updateAllSubscriptions" class="cbx-btn-ghost px-2.5 py-1 rounded-lg text-[11px] font-mono flex items-center gap-1 font-bold cursor-pointer">
              <span>🔄 并发拉取全部订阅源最新节点</span>
            </button>
          </div>

          <div v-if="app.ir.subscriptions && app.ir.subscriptions.length > 0" class="space-y-3">
            <div v-for="sub in app.ir.subscriptions" :key="sub.id"
              class="rounded-2xl cbx-sub-card transition-all p-3.5 space-y-2.5">
              <!-- Top Row: Name, count, status, health & actions -->
              <div class="flex items-start justify-between gap-2 flex-wrap">
                <div class="flex items-center gap-1.5 flex-wrap min-w-0">
                  <span class="text-xs font-black cbx-text-main flex items-center gap-1.5 truncate">
                    <span>✈️</span>
                    <span class="truncate">{{ sub.name }}</span>
                  </span>
                  <span class="text-[10px] px-2 py-0.5 rounded-full font-mono cbx-badge-info border font-bold shrink-0">
                    {{ sub.node_count || 0 }} 节点
                  </span>
                  <span :class="['text-[10px] px-1.5 py-0.5 rounded-full font-mono border shrink-0', sub.enabled ? 'cbx-badge-ok' : 'cbx-btn-ghost']">
                    {{ sub.enabled ? '● 启用' : '○ 停用' }}
                  </span>
                  <span v-if="app.subHealthMap[sub.id || sub.name]?.text" :class="['text-[10px] px-1.5 py-0.5 rounded-full font-mono font-bold border shrink-0', app.subHealthMap[sub.id || sub.name]?.class]">
                    {{ app.subHealthMap[sub.id || sub.name]?.icon }} {{ app.subHealthMap[sub.id || sub.name]?.text }}
                  </span>
                </div>

                <div class="flex items-center gap-1.5 shrink-0 font-mono">
                  <button @click="app.runSubSpeedTest(sub.id)" :disabled="app.subSpeedTesting[sub.id]" title="仅对该订阅源测速诊断"
                    class="px-2 py-1 rounded-lg cbx-badge-ok border text-xs cursor-pointer">
                    ⚡
                  </button>
                  <button @click="app.updateSubscription(sub.id)" title="更新此订阅" class="px-2 py-1 rounded-lg cbx-btn-ghost text-xs cursor-pointer">🔄</button>
                  <button @click="app.toggleSubscription(sub.id, !sub.enabled)" :title="sub.enabled ? '停用此订阅' : '启用此订阅'" class="px-2 py-1 rounded-lg cbx-btn-ghost text-xs cursor-pointer">
                    {{ sub.enabled ? '👁️' : '🕶️' }}
                  </button>
                  <button @click="app.deleteSubscription(sub.id)" title="删除此订阅" class="px-2 py-1 rounded-lg cbx-badge-danger border text-xs cursor-pointer">🗑️</button>
                </div>
              </div>

              <!-- Telemetry Progress Bar & Data -->
              <div class="cbx-sub-card p-2.5 rounded-xl space-y-1.5">
                <div class="flex items-center justify-between text-[10px] font-mono flex-wrap gap-1">
                  <span class="cbx-text-sub">
                    剩余流量: <strong class="cbx-text-main font-tabular">{{ app.subQuotaMap[sub.id || sub.name]?.remStr }}</strong>
                    <span class="cbx-text-muted font-tabular" v-if="app.subQuotaMap[sub.id || sub.name]?.hasQuota"> / {{ app.subQuotaMap[sub.id || sub.name]?.totalStr }}</span>
                  </span>
                  <span :class="['px-1.5 py-0.5 rounded font-bold border', app.formatExpire(sub.expire_at).badgeClass]">
                    {{ app.subMetaMap[sub.id]?.expireObj?.text }}
                  </span>
                </div>

                <div class="w-full cbx-progress-track rounded-full h-1.5 overflow-hidden">
                  <div :class="['h-full rounded-full transition-all duration-500', app.subQuotaMap[sub.id || sub.name]?.barClass]"
                       :style="{ width: (app.subQuotaMap[sub.id || sub.name]?.percent || 0) + '%' }"></div>
                </div>

                <div class="flex items-center justify-between text-[10px] cbx-text-muted font-mono flex-wrap gap-1">
                  <span>📅 重置: {{ app.subMetaMap[sub.id]?.resetInfo }}</span>
                  <span>⏱️ 更新于: {{ app.subMetaMap[sub.id]?.timeAgo }}</span>
                </div>
              </div>

              <!-- Regional distribution & Drawer toggle -->
              <div class="flex items-center justify-between text-[10px] font-mono pt-0.5 flex-wrap gap-2">
                <div class="flex items-center gap-1 flex-wrap">
                  <span class="cbx-text-muted">地区:</span>
                  <span v-for="reg in app.getSubRegions(sub).slice(0, 4)" :key="reg.name" class="px-1.5 py-0.5 rounded cbx-btn-ghost">
                    {{ reg.flag }} {{ reg.count }}
                  </span>
                </div>
                <button @click="app.toggleSubDrawer(sub.id)" class="text-sky-600 dark:text-cyan-300 font-bold underline cursor-pointer">
                  {{ app.isSubDrawerExpanded(sub.id) ? '收起详情' : `展开节点 (${sub.node_count || 0})` }}
                </button>
              </div>

              <!-- Modal In-place Drawer -->
              <div v-if="app.isSubDrawerExpanded(sub.id)" class="p-2.5 cbx-sub-card rounded-xl space-y-1.5 max-h-48 overflow-y-auto text-xs font-mono">
                <div v-for="(node, nIdx) in (app.subNodesMap[sub.id] || [])" :key="node.name"
                  class="p-2 rounded-lg cbx-sub-card flex items-center justify-between gap-1.5 flex-wrap sm:flex-nowrap">
                  <div class="flex items-center gap-1.5 min-w-0 truncate">
                    <span class="cbx-text-muted text-[10px] shrink-0">{{ nIdx + 1 }}.</span>
                    <span class="text-[10px] px-1.5 py-0.2 rounded uppercase cbx-badge-info border font-bold shrink-0">{{ node.type }}</span>
                    <span v-if="node.ports || node.mport" class="text-[9px] px-1 py-0.2 rounded font-mono cbx-badge-warn border shrink-0" :title="'端口跳跃: ' + (node.ports || node.mport)">⚡跳跃</span>
                    <span class="cbx-text-main text-[11px] truncate">{{ node.name }}</span>
                  </div>
                  <div class="flex items-center gap-1.5 shrink-0">
                    <span :class="['px-1.5 py-0.5 rounded text-[11px] font-bold border font-tabular', app.getNodeDelayClass(node.delay, node.delay_status)]">
                      {{ node.delay > 0 ? `${node.delay}ms` : (node.delay_text || '未测速') }}
                    </span>
                    <button @click="app.setNodeAsPrimaryOutbound(node.name)" class="px-2 py-0.5 rounded cbx-badge-ok border text-[11px] font-bold cursor-pointer">
                      设为出口
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-else class="text-xs cbx-text-muted p-4 text-center border border-dashed rounded-xl" style="border-color: var(--cbx-border-soft);">
            暂无保存的订阅源，点击「批量导入订阅」即可添加。
          </div>
        </div>

        <!-- Custom nodes list -->
        <div class="space-y-2 pt-2 border-t" style="border-color: var(--cbx-border-soft);">
          <div class="text-xs font-bold cbx-text-main flex items-center justify-between">
            <span class="flex items-center gap-1.5">
              <span>🔗 自建单节点清单</span>
              <span class="cbx-badge-purple border px-1.5 py-0.2 rounded font-mono text-[11px]">{{ app.customNodes.length }} 个</span>
            </span>
          </div>

          <div v-if="app.customNodes.length > 0" class="space-y-2">
            <div v-for="node in app.customNodes" :key="node.name" class="p-2.5 rounded-xl cbx-sub-card flex items-center justify-between gap-2">
              <div class="min-w-0 truncate">
                <div class="text-xs font-bold cbx-text-main flex items-center gap-2">
                  <span class="text-[10px] px-1.5 py-0.5 rounded uppercase font-mono cbx-badge-purple border">
                    {{ node.type }}
                  </span>
                  <span v-if="node.ports || node.mport" class="text-[11px] px-1.5 py-0.5 rounded font-mono cbx-badge-info border">
                    ⚡ 跳频: {{ node.ports || node.mport }}
                  </span>
                  <span class="truncate">{{ node.name }}</span>
                </div>
                <div class="text-[10px] cbx-text-muted font-mono truncate mt-0.5">{{ node.server }}:{{ node.port }}</div>
              </div>
              <button @click="app.deleteNode(node.name)" title="删除此节点" class="p-1.5 rounded-lg cbx-badge-danger border text-xs shrink-0 cursor-pointer">🗑️</button>
            </div>
          </div>

          <div v-else class="text-xs cbx-text-muted p-3 text-center border border-dashed rounded-xl" style="border-color: var(--cbx-border-soft);">
            暂无自建单节点，点击「批量导入自建节点」即可导入落地专线。
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject } from 'vue'
const app = inject('cubexApp')
</script>
