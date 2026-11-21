<template>
  <div v-if="isOpen" class="fixed inset-0 z-50 flex justify-end">
    <!-- 背景遮罩 (点击空白处关闭) -->
    <div
      class="fixed inset-0 bg-black/60 backdrop-blur-sm transition-opacity duration-300 animate-in fade-in"
      @click="$emit('close')"
    ></div>

    <!-- 右侧滑出抽屉面板 (微晶玻璃质感 · contain: content 隔绝重绘) -->
    <aside
      class="theme-drawer relative w-full max-w-md sm:max-w-lg h-full bg-[#001529]/95 backdrop-blur-2xl border-l border-white/10 shadow-[-10px_0_40px_rgba(0,0,0,0.6)] flex flex-col z-10 animate-in slide-in-from-right duration-300"
      style="contain: content;"
    >
      <!-- 抽屉头部 (标题与关闭按钮) -->
      <div class="h-16 px-6 border-b border-white/10 flex items-center justify-between shrink-0 select-none bg-white/[0.02]">
        <div class="flex items-center space-x-2.5">
          <span class="text-xl">🎨</span>
          <div>
            <h2 class="text-base font-bold text-white tracking-wide">主题视效中心</h2>
            <p class="text-[11px] text-slate-400">零显存浪费 · 60FPS 顺滑流光 · 本地离线直出</p>
          </div>
        </div>
        <button
          @click="$emit('close')"
          class="w-8 h-8 rounded-full bg-white/5 hover:bg-white/10 border border-white/10 flex items-center justify-center text-slate-300 hover:text-white transition cursor-pointer"
          title="关闭 (或按 ESC)"
        >
          ✕
        </button>
      </div>

      <!-- 抽屉滚动主内容视口 -->
      <div class="flex-1 overflow-y-auto p-6 space-y-6 custom-scrollbar text-slate-200">
        <!-- ════════════════════════════════════════════════════════════ -->
        <!-- 区域 1: 播放策略选择 (智能自动 / 始终动态 / 静态省电)             -->
        <!-- ════════════════════════════════════════════════════════════ -->
        <section class="space-y-2.5">
          <div class="flex items-center justify-between">
            <h3 class="text-xs font-bold text-slate-300 uppercase tracking-wider">视效播放策略</h3>
            <span class="text-[11px] text-cyan-300 font-mono">切后台自动休眠</span>
          </div>
          <div class="grid grid-cols-3 gap-2 p-1 rounded-2xl bg-black/40 border border-white/10 text-xs font-medium">
            <button
              @click="setPlaybackMode('auto')"
              :class="[
                'py-2 px-2 rounded-xl transition text-center cursor-pointer select-none',
                playbackMode === 'auto'
                  ? 'bg-gradient-to-r from-sky-500 to-cyan-500 text-white font-bold shadow-md shadow-cyan-500/25'
                  : 'text-slate-400 hover:text-white'
              ]"
            >
              智能自动
            </button>
            <button
              @click="setPlaybackMode('always')"
              :class="[
                'py-2 px-2 rounded-xl transition text-center cursor-pointer select-none',
                playbackMode === 'always'
                  ? 'bg-gradient-to-r from-sky-500 to-cyan-500 text-white font-bold shadow-md shadow-cyan-500/25'
                  : 'text-slate-400 hover:text-white'
              ]"
            >
              始终动态
            </button>
            <button
              @click="setPlaybackMode('poster')"
              :class="[
                'py-2 px-2 rounded-xl transition text-center cursor-pointer select-none',
                playbackMode === 'poster'
                  ? 'bg-gradient-to-r from-sky-500 to-cyan-500 text-white font-bold shadow-md shadow-cyan-500/25'
                  : 'text-slate-400 hover:text-white'
              ]"
            >
              静态省电
            </button>
          </div>
        </section>

        <!-- ════════════════════════════════════════════════════════════ -->
        <!-- 区域 2: 已安装主题列表                                       -->
        <!-- ════════════════════════════════════════════════════════════ -->
        <section class="space-y-3">
          <div class="flex items-center justify-between">
            <h3 class="text-xs font-bold text-slate-300 uppercase tracking-wider">已安装视效主题</h3>
            <span class="text-[11px] text-slate-400 font-mono">{{ themes.length }} 个主题就绪</span>
          </div>

          <div class="grid grid-cols-1 gap-2.5">
            <div
              v-for="th in themes"
              :key="th.id"
              :class="[
                'p-3.5 rounded-2xl border transition-all flex items-center justify-between gap-3',
                th.id === activeThemeID
                  ? 'bg-gradient-to-r from-cyan-950/40 to-slate-900/60 border-cyan-400/50 shadow-[0_0_20px_rgba(0,210,255,0.15)] ring-1 ring-cyan-400/30'
                  : 'bg-white/[0.03] hover:bg-white/[0.06] border-white/10'
              ]"
            >
              <!-- 左侧图标与信息 -->
              <div class="flex items-center space-x-3 min-w-0">
                <div
                  class="w-10 h-10 rounded-xl flex items-center justify-center shrink-0 border"
                  :style="{
                    backgroundColor: th.tokens?.['--cbx-theme-card-bg'] || 'rgba(0, 210, 255, 0.1)',
                    borderColor: th.tokens?.['--cbx-theme-accent'] || 'rgba(0, 210, 255, 0.3)'
                  }"
                >
                  <span v-if="th.type === 'video'" class="text-lg">🎞️</span>
                  <span v-else-if="th.type === 'multi-image'" class="text-lg">🖼️</span>
                  <span v-else class="text-lg">🧊</span>
                </div>

                <div class="min-w-0">
                  <div class="flex items-center space-x-2">
                    <span class="text-sm font-bold text-white truncate">{{ th.name }}</span>
                    <span
                      v-if="th.is_builtin"
                      class="px-1.5 py-0.2 rounded text-[10px] font-mono bg-white/10 text-slate-300 border border-white/10"
                    >
                      系统内置
                    </span>
                  </div>
                  <p class="text-[11px] text-slate-400 truncate mt-0.5">{{ th.description }}</p>
                </div>
              </div>

              <!-- 右侧操作区 -->
              <div class="flex items-center space-x-2 shrink-0">
                <span
                  v-if="th.id === activeThemeID"
                  class="px-2.5 py-1 rounded-full text-xs font-bold font-mono bg-cyan-500/20 text-cyan-300 border border-cyan-400/40 flex items-center space-x-1"
                >
                  <span class="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-pulse"></span>
                  <span>使用中</span>
                </span>
                <button
                  v-else
                  @click="activateTheme(th.id)"
                  class="px-3 py-1 rounded-xl text-xs font-bold bg-white/10 hover:bg-white/20 text-slate-200 hover:text-white border border-white/15 transition cursor-pointer"
                >
                  启用
                </button>

                <!-- 删除自定义主题按钮 -->
                <button
                  v-if="!th.is_builtin && th.id !== 'twilight-default' && th.id !== 'legacy-wallpapers'"
                  @click="deleteTheme(th.id)"
                  class="p-1 rounded-lg text-slate-500 hover:text-rose-400 hover:bg-rose-500/10 transition cursor-pointer"
                  title="删除该主题"
                >
                  🗑️
                </button>
              </div>
            </div>
          </div>
        </section>

        <!-- ════════════════════════════════════════════════════════════ -->
        <!-- 区域 3: 拖拽极速导入 .cbxtheme 主题包                           -->
        <!-- ════════════════════════════════════════════════════════════ -->
        <section class="space-y-2.5">
          <h3 class="text-xs font-bold text-slate-300 uppercase tracking-wider">拖拽安装专属主题</h3>

          <div
            @dragover.prevent="isDragging = true"
            @dragleave.prevent="isDragging = false"
            @drop.prevent="onDropFile"
            @click="triggerFileBrowse"
            :class="[
              'p-5 rounded-2xl border-2 border-dashed transition-all flex flex-col items-center justify-center text-center cursor-pointer select-none',
              isDragging
                ? 'border-cyan-400 bg-cyan-950/40 shadow-[0_0_25px_rgba(0,210,255,0.25)] scale-[1.01]'
                : 'border-white/15 hover:border-cyan-400/50 bg-black/30 hover:bg-black/40'
            ]"
          >
            <!-- 兼容各类系统的后缀选择器 -->
            <input
              ref="fileInput"
              type="file"
              accept=".cbxtheme,.zip,application/zip"
              @change="onFileChange"
              class="hidden"
            />

            <div v-if="isImporting" class="flex flex-col items-center space-y-2">
              <span class="text-2xl animate-spin">🔄</span>
              <span class="text-xs text-cyan-300 font-bold">正在解密校验与解压主题包...</span>
            </div>
            <div v-else class="flex flex-col items-center space-y-1.5">
              <span class="text-2xl">📥</span>
              <p class="text-xs font-bold text-slate-200">拖入 <span class="text-cyan-300 font-mono">.cbxtheme</span> 主题包，3 秒解锁专属视效</p>
              <p class="text-[11px] text-slate-400 font-mono">支持点击选择文件 · 单包上限 35 MB</p>
            </div>
          </div>

          <p v-if="importError" class="text-xs font-medium text-rose-400 px-1">
            ⚠️ {{ importError }}
          </p>
        </section>

        <!-- ════════════════════════════════════════════════════════════ -->
        <!-- 区域 4: 精选扩展主题 (粉丝专享 · 纯本地零联网)                   -->
        <!-- ════════════════════════════════════════════════════════════ -->
        <section class="space-y-3 pt-2">
          <div class="flex items-center justify-between">
            <h3 class="text-xs font-bold text-amber-300 uppercase tracking-wider flex items-center space-x-1.5">
              <span>👑</span>
              <span>精选扩展视效 (粉丝专享)</span>
            </h3>
            <span class="text-[10px] px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/30 font-bold">
              免费领取引流版
            </span>
          </div>

          <!-- 黑金奢华骷髅推荐卡片 (基于 <30KB preview.webp 纯离线静态展示) -->
          <div class="rounded-2xl border border-amber-500/30 bg-gradient-to-br from-[#161208]/90 via-[#0d131f]/90 to-[#0a0c12]/95 overflow-hidden shadow-xl">
            <!-- 缩略图容器 -->
            <div class="relative h-36 w-full overflow-hidden bg-black/60">
              <img
                src="/assets/themes/preview-dark-skull-luxury.webp"
                alt="暗夜曜金 · 奢华骷髅"
                class="w-full h-full object-cover object-center hover:scale-105 transition-transform duration-500"
              />
              <div class="absolute inset-0 bg-gradient-to-t from-[#0a0c12] via-transparent to-transparent"></div>
              <span class="absolute top-2.5 right-2.5 px-2 py-0.5 rounded-full text-[10px] font-bold font-mono bg-black/70 border border-amber-400/40 text-amber-300">
                1080p 30fps
              </span>
            </div>

            <!-- 卡片文本与领取说明 -->
            <div class="p-4 space-y-3">
              <div>
                <h4 class="text-sm font-black text-white flex items-center space-x-1.5">
                  <span>暗夜曜金 · 奢华骷髅</span>
                  <span class="text-[10px] px-1.5 py-0.2 rounded bg-amber-500/20 text-amber-300 font-mono">1080p 引流版</span>
                </h4>
                <p class="text-xs text-slate-300 mt-1">黑金曜石质感全景动态壁纸 · 60FPS 顺滑循环 · 纯离线无感渲染</p>
              </div>

              <!-- 极简领取指南 -->
              <div class="p-2.5 rounded-xl bg-black/50 border border-white/10 text-[11px] text-slate-300 space-y-1">
                <div class="flex items-start space-x-1.5">
                  <span class="text-amber-400 shrink-0">1.</span>
                  <span>前往 <strong>GitHub</strong> 点亮 Star，或关注 <strong>B站</strong> 官方账号；</span>
                </div>
                <div class="flex items-start space-x-1.5">
                  <span class="text-amber-400 shrink-0">2.</span>
                  <span>加入交流粉丝群或回复关键词【<strong class="text-amber-300">骷髅</strong>】，即可免费领取 <code class="text-cyan-300">dark-skull-luxury-1080p.cbxtheme</code>；</span>
                </div>
                <div class="flex items-start space-x-1.5">
                  <span class="text-amber-400 shrink-0">3.</span>
                  <span>下载后拖入上方虚线框，3 秒即刻生效！</span>
                </div>
              </div>

              <!-- 外链按钮组 -->
              <div class="flex items-center space-x-2 pt-1">
                <a
                  href="https://github.com/zyeyefyass/CubeX"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="flex-1 py-1.5 rounded-xl bg-white/10 hover:bg-white/20 border border-white/15 text-xs font-bold text-white text-center transition cursor-pointer"
                >
                  🌟 前往 GitHub 点亮 Star
                </a>
                <a
                  href="https://space.bilibili.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="flex-1 py-1.5 rounded-xl bg-gradient-to-r from-amber-500/30 to-amber-600/30 hover:from-amber-500/40 hover:to-amber-600/40 border border-amber-400/40 text-xs font-bold text-amber-200 text-center transition cursor-pointer"
                >
                  📺 B站关注领主题
                </a>
              </div>

              <!-- 长线埋点小字 -->
              <p class="text-[10px] text-slate-500 text-center font-sans">
                更多 2K 臻享动态主题与季度主题库持续更新中，欢迎通过充电/赞助支持项目长久维护
              </p>
            </div>
          </div>
        </section>
      </div>
    </aside>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useThemeEngine } from '../composables/useThemeEngine.js'

defineProps({
  isOpen: {
    type: Boolean,
    default: false
  }
})

defineEmits(['close'])

const {
  themes,
  activeThemeID,
  playbackMode,
  isImporting,
  importError,
  activateTheme,
  setPlaybackMode,
  importThemeFile,
  deleteTheme
} = useThemeEngine()

const fileInput = ref(null)
const isDragging = ref(false)

function triggerFileBrowse() {
  if (fileInput.value) {
    fileInput.value.click()
  }
}

async function onFileChange(e) {
  const file = e.target.files?.[0]
  if (file) {
    await importThemeFile(file)
    e.target.value = ''
  }
}

async function onDropFile(e) {
  isDragging.value = false
  const file = e.dataTransfer?.files?.[0]
  if (file) {
    await importThemeFile(file)
  }
}
</script>
