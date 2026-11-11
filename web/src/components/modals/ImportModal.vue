<template>
    <div v-if="app.showImportModal" class="fixed inset-0 z-50 bg-black/75 backdrop-blur-xl flex items-center justify-center p-4 overflow-y-auto animate-in fade-in">

      <div class="bg-white/[0.03] border border-slate-700/70 rounded-3xl p-6 sm:p-7 max-w-3xl w-full space-y-5 shadow-2xl max-h-[92vh] flex flex-col text-slate-200">

        <!-- Modal Header -->

        <div class="flex items-center justify-between border-b border-slate-700/60 pb-3.5">

          <div class="flex items-center gap-3">

            <span class="text-2xl p-2 rounded-2xl bg-cyan-950/80 border border-cyan-500/30">⚡</span>

            <div>

              <h3 class="text-base font-black text-white flex items-center gap-2">

                <span>订阅源与自建节点批量导入中枢</span>

                <span class="text-[10px] px-2 py-0.5 rounded-full bg-white/10 text-sky-200 border border-sky-400/30 font-bold drop-shadow-sm font-mono font-bold">

                  Batch Multi-Import

                </span>

              </h3>

              <p class="text-[11px] text-slate-400 mt-0.5">支持每行一条、直接指定名称，实时智能解析并一键合并挂载到核心代理集群</p>

            </div>

          </div>

          <button @click="app.showImportModal = false" class="text-slate-400 hover:text-white px-2.5 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white/80 hover:bg-slate-700 text-sm font-bold transition">✕</button>

        </div>

        <!-- 3-Segment Navigation Capsule -->

        <div class="flex p-1 bg-slate-950/90 rounded-2xl border border-slate-800 text-xs font-mono shrink-0">

          <button @click="app.subModalTab = 'sub'" :class="['flex-1 py-2 rounded-xl font-bold transition flex items-center justify-center gap-1.5', app.subModalTab === 'sub' ? 'bg-cyan-500 text-slate-950 shadow-md shadow-sky-500/20' : 'text-slate-400 hover:text-slate-200']">

            <span>🌐 批量导入订阅</span>

            <span v-if="app.parsedBatchSubs.length > 0" class="px-1.5 py-0.2 rounded-full text-[10px] bg-slate-950 text-emerald-300 font-mono">

              {{ app.parsedBatchSubs.length }}

            </span>

          </button>

          <button @click="app.subModalTab = 'node'" :class="['flex-1 py-2 rounded-xl font-bold transition flex items-center justify-center gap-1.5', app.subModalTab === 'node' ? 'bg-cyan-500 text-slate-950 shadow-md shadow-sky-500/20' : 'text-slate-400 hover:text-slate-200']">

            <span>🔗 批量导入自建节点</span>

            <span v-if="app.parsedBatchNodes.length > 0" class="px-1.5 py-0.2 rounded-full text-[10px] bg-slate-950 text-emerald-300 font-mono">

              {{ app.parsedBatchNodes.length }}

            </span>

          </button>

          <button @click="app.subModalTab = 'list'" :class="['flex-1 py-2 rounded-xl font-bold transition flex items-center justify-center gap-1.5', app.subModalTab === 'list' ? 'bg-cyan-500 text-slate-950 shadow-md shadow-sky-500/20' : 'text-slate-400 hover:text-slate-200']">

            <span>📋 订阅与节点清单 ({{ app.ir.subscriptions?.length || 0 }} 订阅源 · {{ app.customNodes.length }} 节点)</span>

          </button>

        </div>

        <!-- Tab 1: 批量导入订阅 -->

        <div v-if="app.subModalTab === 'sub'" class="space-y-3.5 overflow-y-auto pr-1 flex-1">

          <!-- Format Helper Banner -->

          <div class="p-3 bg-cyan-950/40 border border-cyan-800/50 rounded-2xl text-xs space-y-1.5 text-emerald-200">

            <div class="font-bold flex items-center gap-1.5 text-emerald-300 font-mono">

              <span>📋 多条订阅格式说明 (每行一条，可直接在行内指定自定义名称)：</span>

            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-1.5 text-[11px] font-mono text-emerald-200/90 pl-1">

              <div>• <span class="text-white font-bold">名称, 订阅链接</span> (如: <code>香港主力, https://airport.com/sub?token=xxx</code>)</div>

              <div>• <span class="text-white font-bold">名称 | 订阅链接</span> (如: <code>备用专线 | https://airport.com/clash</code>)</div>

              <div>• <span class="text-white font-bold">订阅链接#名称</span> (如: <code>https://sub.link.net/api#日本BGP专线</code>)</div>

              <div>• <span class="text-white font-bold">直接粘贴纯链接</span> (系统将自动按序号命名: <code>订阅源 1, 2...</code>)</div>

            </div>

          </div>

          <!-- Multi-line Textarea -->

          <div>

            <div class="flex items-center justify-between text-xs font-semibold text-slate-300 mb-1.5">

              <span class="flex items-center gap-1.5">

                <span>粘贴订阅源列表 (每行一条)</span>

                <span class="text-[11px] text-slate-400 font-normal">支持 Clash / sing-box / V2Ray 协议</span>

              </span>

              <span v-if="app.parsedBatchSubs.length > 0" class="text-emerald-300 font-mono text-xs font-bold flex items-center gap-1">

                <span>⚡ 实时识别:</span>

                <span class="px-2 py-0.5 rounded bg-cyan-950 border border-cyan-500/40 text-emerald-300">{{ app.parsedBatchSubs.length }} 条有效订阅</span>

              </span>

            </div>

            <textarea v-model="app.batchSubText" rows="6"

              placeholder="每行一条订阅源，支持直接指定自定义名称。例如：&#10;香港主力专线, https://sub.network1.com/api/v1/client/subscribe?token=abc&#10;备用海外节点 | https://sub.network2.com/clash/token123&#10;https://sub.myfast.net/singbox#游戏低延迟加速"

              class="w-full bg-slate-950 border border-slate-700/80 rounded-2xl p-3 text-xs text-slate-100 font-mono outline-none focus:border-cyan-400 focus:ring-1 focus:ring-cyan-400 shadow-inner resize-y leading-relaxed" />

          </div>

          <!-- Real-time Live Preview List -->

          <div v-if="app.parsedBatchSubs.length > 0" class="space-y-1.5 bg-slate-950/80 p-3 rounded-2xl border border-cyan-500/30">

            <div class="text-[11px] font-bold text-emerald-300 flex items-center justify-between font-mono">

              <span>⚡ 实时解析预览 (共 {{ app.parsedBatchSubs.length }} 条订阅将一次性合并导入)：</span>

              <span class="text-slate-400 text-[10px]">单次原子热重载 · 零频闪</span>

            </div>

            <div class="max-h-36 overflow-y-auto space-y-1.5 pr-1 font-mono text-[11px]">

              <div v-for="(item, idx) in app.parsedBatchSubs" :key="idx" class="flex items-center justify-between gap-2 p-2 rounded-xl bg-white/10 text-white border border-slate-800">

                <div class="flex items-center gap-2 truncate">

                  <span class="text-slate-300 text-[10px] w-5 text-right shrink-0">{{ idx + 1 }}.</span>

                  <span class="px-2 py-0.5 rounded bg-white/10 text-sky-200 border border-sky-400/30 font-bold drop-shadow-sm text-xs font-bold shrink-0">

                    ✈️ {{ item.name }}

                  </span>

                  <span class="text-slate-400 truncate text-[11px]">{{ item.url }}</span>

                </div>

              </div>

            </div>

          </div>

          <!-- Actions -->

          <div class="flex gap-2 pt-2 border-t border-slate-800/80">

            <button @click="app.submitBatchSubs" :disabled="app.parsedBatchSubs.length === 0 || app.batchImportingSubs"

              class="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-sky-600 via-sky-500 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 text-white font-extrabold font-bold text-xs shadow-lg shadow-sky-500/20 active:scale-95 transition flex items-center justify-center gap-1.5 disabled:opacity-50 disabled:cursor-not-allowed">

              <span v-if="app.batchImportingSubs" class="inline-block animate-spin">↻</span>

              <span>{{ app.batchImportingSubs ? '正在并发拉取并解析所有订阅...' : `🚀 立即一键导入 (${app.parsedBatchSubs.length} 条订阅源)` }}</span>

            </button>

            <button v-if="app.batchSubText" @click="app.batchSubText = ''" class="px-3.5 py-2.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-slate-300 text-xs font-semibold transition">

              清空

            </button>

            <button @click="app.showImportModal = false" class="px-4 py-2.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-slate-300 text-xs font-semibold transition">

              取消

            </button>

          </div>

        </div>

        <!-- Tab 2: 批量导入自建单节点 -->

        <div v-else-if="app.subModalTab === 'node'" class="space-y-3.5 overflow-y-auto pr-1 flex-1">

          <!-- Format Helper Banner -->

          <div class="p-3 bg-purple-950/40 border border-purple-800/50 rounded-2xl text-xs space-y-1.5 text-purple-200">

            <div class="font-bold flex items-center gap-1.5 text-purple-300 font-mono">

              <span>📋 多条自建节点格式说明 (每行一条，可直接在行内指定自定义名称)：</span>

            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-1.5 text-[11px] font-mono text-purple-200/90 pl-1">

              <div>• <span class="text-white font-bold">名称, 节点链接</span> (如: <code>东京自建, ss://...</code> 或 <code>美西家宽, vless://...</code>)</div>

              <div>• <span class="text-white font-bold">名称 | 节点链接</span> (如: <code>新加坡专线 | hysteria2://...</code>)</div>

              <div>• <span class="text-white font-bold">节点链接#名称</span> (如: <code>vless://...#香港01专线</code>)</div>

              <div>• <span class="text-white font-bold">直接粘贴单节点链接</span> (支持 ss, vmess, vless, trojan, hy2, tuic)</div>

            </div>

          </div>

          <!-- Multi-line Textarea -->

          <div>

            <div class="flex items-center justify-between text-xs font-semibold text-slate-300 mb-1.5">

              <span class="flex items-center gap-1.5">

                <span>粘贴自建节点列表 (每行一条)</span>

                <span class="text-[11px] text-slate-400 font-normal">支持 Shadowsocks, VMess, VLESS, Trojan, Hysteria2, TUIC</span>

              </span>

              <span v-if="app.parsedBatchNodes.length > 0" class="text-purple-400 font-mono text-xs font-bold flex items-center gap-1">

                <span>⚡ 实时识别:</span>

                <span class="px-2 py-0.5 rounded bg-purple-950 border border-purple-500/40 text-purple-300">{{ app.parsedBatchNodes.length }} 个有效节点</span>

              </span>

            </div>

            <textarea v-model="app.batchNodeText" rows="6"

              placeholder="每行一条自建节点，支持直接指定自定义名称。例如：&#10;东京落地 01, ss://YWVzLTI1Ni1nY206cGFzc0AxLjIuMy40OjgzODg=#东京&#10;美西家宽直连 | vless://uuid@us.example.com:443?security=reality&sni=yahoo.com#美西&#10;hy2://password@sg.speed.org:8443#新加坡高速"

              class="w-full bg-slate-950 border border-slate-700/80 rounded-2xl p-3 text-xs text-slate-100 font-mono outline-none focus:border-purple-400 focus:ring-1 focus:ring-purple-400 shadow-inner resize-y leading-relaxed" />

          </div>

          <!-- Real-time Live Preview List -->

          <div v-if="app.parsedBatchNodes.length > 0" class="space-y-1.5 bg-slate-950/80 p-3 rounded-2xl border border-purple-500/30">

            <div class="text-[11px] font-bold text-purple-300 flex items-center justify-between font-mono">

              <span>⚡ 实时解析预览 (共 {{ app.parsedBatchNodes.length }} 个节点将合并导入并归入「🏠 自建」集群)：</span>

              <span class="text-slate-400 text-[10px]">单次原子热重载 · 自动去重</span>

            </div>

            <div class="max-h-36 overflow-y-auto space-y-1.5 pr-1 font-mono text-[11px]">

              <div v-for="(item, idx) in app.parsedBatchNodes" :key="idx" class="flex items-center justify-between gap-2 p-2 rounded-xl bg-white/10 text-white border border-slate-800">

                <div class="flex items-center gap-2 truncate">

                  <span class="text-slate-300 text-[10px] w-5 text-right shrink-0">{{ idx + 1 }}.</span>

                  <span class="px-1.5 py-0.5 rounded bg-purple-950 text-purple-300 border border-purple-500/40 text-[10px] font-bold shrink-0 uppercase">

                    {{ item.protocol }}

                  </span>

                  <span class="px-2 py-0.5 rounded bg-white/10 text-white text-white text-xs font-bold shrink-0 truncate max-w-[140px]">

                    🔗 {{ item.name }}

                  </span>

                  <span class="text-slate-400 truncate text-[11px]">{{ item.link }}</span>

                </div>

              </div>

            </div>

          </div>

          <!-- Actions -->

          <div class="flex gap-2 pt-2 border-t border-slate-800/80">

            <button @click="app.submitBatchNodes" :disabled="app.parsedBatchNodes.length === 0 || app.batchAddingNodes"

              class="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-purple-500 to-indigo-500 hover:from-purple-400 hover:to-indigo-400 text-white font-bold text-xs shadow-lg shadow-purple-500/20 active:scale-95 transition flex items-center justify-center gap-1.5 disabled:opacity-50 disabled:cursor-not-allowed">

              <span v-if="app.batchAddingNodes" class="inline-block animate-spin">↻</span>

              <span>{{ app.batchAddingNodes ? '正在添加自建节点并重载核心...' : `➕ 立即一键添加 (${app.parsedBatchNodes.length} 个节点)` }}</span>

            </button>

            <button v-if="app.batchNodeText" @click="app.batchNodeText = ''" class="px-3.5 py-2.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-slate-300 text-xs font-semibold transition">

              清空

            </button>

            <button @click="app.showImportModal = false" class="px-4 py-2.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-slate-300 text-xs font-semibold transition">

              取消

            </button>

          </div>

        </div>

        <!-- Tab 3: 订阅源与自建节点清单 -->

        <div v-else class="space-y-3.5 pt-1 max-h-[60vh] overflow-y-auto pr-1 flex-1">

          <!-- Subscriptions list -->

          <div class="space-y-2">

            <div class="text-xs font-bold text-slate-300 flex items-center justify-between">

              <span class="flex items-center gap-1.5">

                <span>✈️ 已保存订阅源</span>

                <span class="text-emerald-300 font-mono">({{ app.ir.subscriptions?.length || 0 }} 个)</span>

              </span>

              <button @click="app.updateAllSubscriptions" class="text-emerald-300 hover:text-emerald-200 text-[11px] font-mono flex items-center gap-1 font-bold">

                <span>🔄 并发拉取全部订阅源最新节点</span>

              </button>

            </div>

            <div v-if="app.ir.subscriptions && app.ir.subscriptions.length > 0" class="space-y-3">

              <div v-for="sub in app.ir.subscriptions" :key="sub.id"

                class="rounded-2xl bg-slate-950/80 border border-slate-800 hover:border-cyan-500/40 transition-all p-3.5 space-y-2.5">

                <!-- Top Row: Name, count, status, health & actions -->

                <div class="flex items-start justify-between gap-2">

                  <div class="flex items-center gap-1.5 flex-wrap min-w-0">

                    <span class="text-xs font-black text-white flex items-center gap-1.5 truncate">

                      <span>✈️</span>

                      <span class="truncate">{{ sub.name }}</span>

                    </span>

                    <span class="text-[10px] px-2 py-0.5 rounded-full font-mono bg-white/10 text-sky-200 border border-sky-400/30 font-bold drop-shadow-sm font-bold shrink-0">

                      {{ sub.node_count || 0 }} 节点

                    </span>

                    <span :class="['text-[10px] px-1.5 py-0.2 rounded-full font-mono shrink-0', sub.enabled ? 'text-emerald-400' : 'text-slate-300']">

                      {{ sub.enabled ? '● 启用' : '○ 停用' }}

                    </span>

                    <span :class="['text-[10px] px-1.5 py-0.2 rounded-full font-mono font-bold border shrink-0', app.subHealthMap[sub.id || sub.name]?.class]">

                      {{ app.subHealthMap[sub.id || sub.name]?.icon }} {{ app.subHealthMap[sub.id || sub.name]?.text }}

                    </span>

                  </div>

                  <div class="flex items-center gap-1 shrink-0 font-mono">

                    <button @click="app.runSubSpeedTest(sub.id)" :disabled="app.subSpeedTesting[sub.id]" title="仅对该订阅源测速诊断"

                      class="p-1 rounded bg-cyan-950 hover:bg-cyan-900 text-emerald-300 border border-cyan-500/30 text-xs">

                      ⚡

                    </button>

                    <button @click="app.updateSubscription(sub.id)" title="更新此订阅" class="p-1 rounded bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-slate-200 text-xs">🔄</button>

                    <button @click="app.toggleSubscription(sub.id, !sub.enabled)" :title="sub.enabled ? '停用此订阅' : '启用此订阅'" class="p-1 rounded bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white hover:bg-slate-700 text-xs">

                      {{ sub.enabled ? '👁️' : '🕶️' }}

                    </button>

                    <button @click="app.deleteSubscription(sub.id)" title="删除此订阅" class="p-1 rounded bg-rose-950/50 hover:bg-rose-900 border border-rose-500/40 text-rose-300 text-xs">🗑️</button>

                  </div>

                </div>

                <!-- Telemetry Progress Bar & Data -->

                <div class="bg-white/10 text-white/85 p-2 rounded-xl border border-slate-800 space-y-1">

                  <div class="flex items-center justify-between text-[10px] font-mono">

                    <span class="text-slate-300">

                      剩余流量: <strong class="text-white">{{ app.subQuotaMap[sub.id || sub.name]?.remStr }}</strong>

                      <span class="text-slate-300" v-if="app.subQuotaMap[sub.id || sub.name]?.hasQuota"> / {{ app.subQuotaMap[sub.id || sub.name]?.totalStr }}</span>

                    </span>

                    <span :class="['px-1.5 py-0.2 rounded font-bold border', app.formatExpire(sub.expire_at).badgeClass]">

                      {{ app.subMetaMap[sub.id]?.expireObj?.text }}

                    </span>

                  </div>

                  <div class="w-full bg-slate-950 rounded-full h-1.5 overflow-hidden border border-slate-800">

                    <div :class="['h-full rounded-full transition-all duration-500', app.subQuotaMap[sub.id || sub.name]?.barClass]"

                         :style="{ width: (app.subQuotaMap[sub.id || sub.name]?.percent || 0) + '%' }"></div>

                  </div>

                  <div class="flex items-center justify-between text-[10px] text-slate-400 font-mono">

                    <span>📅 重置: {{ app.subMetaMap[sub.id]?.resetInfo }}</span>

                    <span>⏱️ 更新于: {{ app.subMetaMap[sub.id]?.timeAgo }}</span>

                  </div>

                </div>

                <!-- Regional distribution & Drawer toggle -->

                <div class="flex items-center justify-between text-[10px] font-mono pt-0.5">

                  <div class="flex items-center gap-1 flex-wrap">

                    <span class="text-slate-300">地区:</span>

                    <span v-for="reg in app.getSubRegions(sub).slice(0, 4)" :key="reg.name" class="px-1.5 py-0.2 rounded bg-white/10 text-white text-slate-300 border border-slate-800">

                      {{ reg.flag }} {{ reg.count }}

                    </span>

                  </div>

                  <button @click="app.toggleSubDrawer(sub.id)" class="text-emerald-300 hover:text-emerald-300 font-bold underline">

                    {{ app.isSubDrawerExpanded(sub.id) ? '收起详情' : `展开节点 (${sub.node_count || 0})` }}

                  </button>

                </div>

                <!-- Modal In-place Drawer -->

                <div v-if="app.isSubDrawerExpanded(sub.id)" class="p-2.5 bg-black/50 rounded-xl border border-cyan-900/40 space-y-1.5 max-h-48 overflow-y-auto text-xs font-mono">

                  <div v-for="(node, nIdx) in (app.subNodesMap[sub.id] || [])" :key="node.name"

                    class="p-1.5 rounded-lg bg-white/10 hover:bg-white/20 border border-white/20 hover:border-sky-400/50 text-white/90 border border-slate-800 flex items-center justify-between gap-1.5">

                    <div class="flex items-center gap-1.5 truncate">

                      <span class="text-slate-300 text-[10px]">{{ nIdx + 1 }}.</span>

                      <span class="text-[11px] px-1 rounded uppercase bg-white/10 text-white text-emerald-300 font-bold">{{ node.type }}</span>

                      <span v-if="node.ports || node.mport" class="text-[8px] px-1 py-0.2 rounded font-mono bg-cyan-950 text-cyan-300 border border-cyan-500/40 animate-pulse" :title="'端口跳跃: ' + (node.ports || node.mport)">⚡跳跃</span>

                      <span class="text-white text-[11px] truncate">{{ node.name }}</span>

                    </div>

                    <div class="flex items-center gap-1.5 shrink-0">

                      <span :class="['px-1.5 py-0.2 rounded text-[11px] font-bold border', app.getNodeDelayClass(node.delay, node.delay_status)]">

                        {{ node.delay > 0 ? `${node.delay}ms` : (node.delay_text || '未测速') }}

                      </span>

                      <button @click="app.setNodeAsPrimaryOutbound(node.name)" class="px-1.5 py-0.2 rounded bg-cyan-950 hover:bg-cyan-900 text-emerald-300 border border-cyan-500/30 text-[11px]">

                        设为出口

                      </button>

                    </div>

                  </div>

                </div>

              </div>

            </div>

            <div v-else class="text-xs text-slate-300 p-4 text-center border border-dashed border-slate-800 rounded-xl">

              暂无保存的订阅源，点击「批量导入订阅」即可添加。

            </div>

          </div>

          <!-- Custom nodes list -->

          <div class="space-y-2 pt-2 border-t border-slate-800">

            <div class="text-xs font-bold text-slate-300 flex items-center justify-between">

              <span class="flex items-center gap-1.5">

                <span>🔗 自建单节点清单</span>

                <span class="text-purple-400 font-mono">({{ app.customNodes.length }} 个)</span>

              </span>

            </div>

            <div v-if="app.customNodes.length > 0" class="space-y-2">

              <div v-for="node in app.customNodes" :key="node.name" class="p-2.5 rounded-xl bg-slate-950/60 border border-slate-800 flex items-center justify-between gap-2">

                <div class="truncate">

                  <div class="text-xs font-bold text-white flex items-center gap-2">

                    <span class="text-[10px] px-1.5 py-0.5 rounded uppercase font-mono bg-purple-950 text-purple-300 border border-purple-500/40">

                      {{ node.type }}

                    </span>

                    <span v-if="node.ports || node.mport" class="text-[11px] px-1.5 py-0.5 rounded font-mono bg-cyan-950 text-cyan-300 border border-cyan-500/40 animate-pulse">

                      ⚡ 跳频: {{ node.ports || node.mport }}

                    </span>

                    <span class="truncate">{{ node.name }}</span>

                  </div>

                  <div class="text-[10px] text-slate-300 font-mono truncate mt-0.5">{{ node.server }}:{{ node.port }}</div>

                </div>

                <button @click="app.deleteNode(node.name)" title="删除此节点" class="p-1.5 rounded-lg bg-rose-950/50 hover:bg-rose-900 border border-rose-500/40 text-rose-300 text-xs shrink-0">🗑️</button>

              </div>

            </div>

            <div v-else class="text-xs text-slate-300 p-3 text-center border border-dashed border-slate-800 rounded-xl">

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
