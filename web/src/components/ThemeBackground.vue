<template>
  <div class="fixed inset-0 pointer-events-none z-0 overflow-hidden select-none" style="contain: layout style paint;">
    <!-- ════════════════════════════════════════════════════════════════ -->
    <!-- Mode 1: 动态视频背景 (黑金曜石奢华动态骷髅 / 60FPS 循环播放)          -->
    <!-- ════════════════════════════════════════════════════════════════ -->
    <template v-if="activeTheme.type === 'video'">
      <!-- 静默降级海报 (当静态省电模式启用、或 autoplay 异常拦截时) -->
      <img
        v-if="playbackMode === 'poster' || isPosterFallback"
        :src="posterUrl"
        alt="CubeX Theme Poster"
        class="w-full h-full object-cover object-center transition-opacity duration-700 opacity-90 scale-105"
        style="contain: content;"
      />

      <!-- 硬件加速 60FPS 视频渲染层 (锁定独立合成层 · 防 IDM 浮条嗅探 · 零黑屏兜底) -->
      <video
        v-else
        ref="videoEl"
        class="w-full h-full object-cover object-center transition-opacity duration-700 opacity-90 scale-105 pointer-events-none select-none"
        autoplay
        loop
        muted
        playsinline
        preload="metadata"
        disablePictureInPicture
        disableRemotePlayback
        controlsList="nodownload nofullscreen noremoteplayback"
        data-idm-ignore="true"
        data-download-ignore="true"
        :poster="posterUrl"
        @error="onVideoError"
        style="contain: content; pointer-events: none; transform: translateZ(0);"
      >
        <source :src="blobVideoUrl || videoUrl" type="video/mp4" />
      </video>
    </template>

    <!-- ════════════════════════════════════════════════════════════════ -->
    <!-- Mode 2: 多标签全景相册海报 (v9.9.4 疯狂动物城等存量本地图片)            -->
    <!-- ════════════════════════════════════════════════════════════════ -->
    <template v-else-if="activeTheme.type === 'multi-image'">
      <img
        :src="multiImageUrl"
        alt="CubeX Panoramic Wallpaper"
        class="w-full h-full object-cover object-center transition-opacity duration-700 opacity-90 scale-105"
        style="contain: content;"
        @error="onMultiImageError"
      />
    </template>

    <!-- ════════════════════════════════════════════════════════════════ -->
    <!-- Mode 3: 原创暮光冰晶几何矢量流光画布 (纯 CSS · 0 显存开销)           -->
    <!-- ════════════════════════════════════════════════════════════════ -->
    <div v-else class="w-full h-full relative" style="background: var(--cbx-bg-canvas);">
      <!-- 极客网格晶格线 -->
      <div
        class="absolute inset-0 opacity-[0.035]"
        style="background-image: radial-gradient(#00d2ff 1px, transparent 1px); background-size: 32px 32px;"
      ></div>
    </div>

    <!-- ════════════════════════════════════════════════════════════════ -->
    <!-- 纯 CSS 矢量径向渐变网格 (零 GPU 离屏纹理显存消耗，替代高耗能高斯模糊)     -->
    <!-- ════════════════════════════════════════════════════════════════ -->
    <div
      class="absolute inset-0 transition-opacity duration-500 pointer-events-none"
      :style="{
        background: activeTheme.type === 'video'
          ? 'radial-gradient(circle at 10% 10%, rgba(229,185,92,0.12) 0%, transparent 50%), radial-gradient(circle at 90% 90%, rgba(123,44,191,0.15) 0%, transparent 60%)'
          : 'radial-gradient(circle at 5% 5%, rgba(0,210,255,0.12) 0%, transparent 45%), radial-gradient(circle at 95% 95%, rgba(123,44,191,0.18) 0%, transparent 55%)'
      }"
    ></div>

    <!-- 柔和压暗/柔光垫层 (双主题自适应，赏景模式淡出至 0) -->
    <div
      class="absolute inset-0 transition-all duration-500 pointer-events-none"
      style="background-color: var(--cbx-bg-scrim);"
      :class="isPeekMode ? 'opacity-0' : 'opacity-100'"
    ></div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useThemeEngine } from '../composables/useThemeEngine.js'
import { apiFetch } from '../utils/apiFetch.js'

const props = defineProps({
  activeMasterTab: {
    type: String,
    default: 'overview'
  },
  isPeekMode: {
    type: Boolean,
    default: false
  }
})

const { activeTheme, playbackMode } = useThemeEngine()
const videoEl = ref(null)
const isPosterFallback = ref(false)
const hasMultiImageError = ref(false)
const blobVideoUrl = ref('')

// Video & Poster URLs
const videoUrl = computed(() => {
  if (activeTheme.value.type !== 'video') return ''
  const assetName = activeTheme.value.assets?.video || 'bg.mp4'
  return `/api/v1/themes/${activeTheme.value.id}/assets/${assetName}`
})

// Anti-IDM Blob URL generator: Fetch video into in-memory Blob URL to prevent IDM floating download bar
async function resolveAntiIDMVideoBlob(url) {
  if (!url || typeof window === 'undefined') return
  try {
    const res = await apiFetch(url, { timeoutMs: 30000 })
    if (res.ok) {
      const blob = await res.blob()
      if (blobVideoUrl.value && blobVideoUrl.value.startsWith('blob:')) {
        URL.revokeObjectURL(blobVideoUrl.value)
      }
      blobVideoUrl.value = URL.createObjectURL(blob)
    }
  } catch (err) {
    // Graceful fallback to direct URL
    blobVideoUrl.value = ''
  }
}

watch(videoUrl, (newUrl) => {
  if (newUrl) resolveAntiIDMVideoBlob(newUrl)
}, { immediate: true })

const posterUrl = computed(() => {
  if (activeTheme.value.type !== 'video') return ''
  const assetName = activeTheme.value.assets?.poster || 'poster.webp'
  return `/api/themes/${activeTheme.value.id}/assets/${assetName}`
})

// Multi-Image Wallpaper URL (v9.9.4 legacy backwards compatibility)
const multiImageUrl = computed(() => {
  if (hasMultiImageError.value) return ''
  const tab = props.activeMasterTab || 'overview'
  const assetName = activeTheme.value.assets?.[tab] || `bg-tab-${tab}.webp`
  return `/api/themes/${activeTheme.value.id}/assets/${assetName}`
})

function onVideoError(e) {
  console.warn('[ThemeBackground] Video stream playback error, falling back to poster:', e)
  isPosterFallback.value = true
}

function onMultiImageError() {
  hasMultiImageError.value = true
}

// Playback Lifecycle & GPU Resource Management
function setupVideoPlayback() {
  if (activeTheme.value.type !== 'video' || playbackMode.value === 'poster') return
  isPosterFallback.value = false

  nextTick(() => {
    if (videoEl.value) {
      videoEl.value.muted = true
      const playPromise = videoEl.value.play()
      if (playPromise !== undefined) {
        playPromise.catch((err) => {
          console.warn('[ThemeBackground] Autoplay prevented, smoothly displaying poster:', err)
          isPosterFallback.value = true
        })
      }
    }
  })
}

// Background Visibility Handler: pause video immediately when user switches tabs/minimizes
function onVisibilityChange() {
  if (!videoEl.value || activeTheme.value.type !== 'video') return
  if (document.hidden) {
    videoEl.value.pause()
  } else {
    if (playbackMode.value !== 'poster' && !isPosterFallback.value) {
      videoEl.value.play().catch(() => {
        isPosterFallback.value = true
      })
    }
  }
}

// Video Mode Class toggle on body
function updateVideoModeClass() {
  if (typeof document === 'undefined') return
  if (activeTheme.value.type === 'video') {
    document.body.classList.add('cbx-video-mode')
  } else {
    document.body.classList.remove('cbx-video-mode')
  }
}

watch(activeTheme, () => {
  hasMultiImageError.value = false
  updateVideoModeClass()
  setupVideoPlayback()
}, { immediate: true })

watch(playbackMode, () => {
  setupVideoPlayback()
})

onMounted(() => {
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', onVisibilityChange)
  }
  updateVideoModeClass()
  setupVideoPlayback()
})

onUnmounted(() => {
  if (typeof document !== 'undefined') {
    document.removeEventListener('visibilitychange', onVisibilityChange)
    document.body.classList.remove('cbx-video-mode')
  }
  if (blobVideoUrl.value && blobVideoUrl.value.startsWith('blob:')) {
    URL.revokeObjectURL(blobVideoUrl.value)
    blobVideoUrl.value = ''
  }
})
</script>
