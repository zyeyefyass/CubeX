<template>
  <div
    class="cubex-logo-wrapper select-none inline-flex items-center justify-center relative group"
    :style="{ width: normalizedSize, height: normalizedSize }"
    :title="`CubeX · 双子星曜折叠超立方 [${statusTitle}]`"
  >
    <!-- Background Ambient Glow -->
    <div
      class="absolute inset-0 rounded-full blur-xl pointer-events-none transition-all duration-700"
      :class="ambientGlowClass"
    ></div>

    <!-- Official Dual-Nova Geometric Prism Cube Vector -->
    <svg
      :width="normalizedSize"
      :height="normalizedSize"
      viewBox="0 0 200 200"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      class="relative z-10 transition-transform duration-500 group-hover:scale-105 active:scale-95"
      :class="animationClass"
    >
      <defs>
        <!-- Background Vignette -->
        <radialGradient id="prism-bg" cx="50%" cy="50%" r="65%">
          <stop offset="0%" stop-color="#002b4d" stop-opacity="0.6"/>
          <stop offset="60%" stop-color="#001426" stop-opacity="0.3"/>
          <stop offset="100%" stop-color="#000b14" stop-opacity="0"/>
        </radialGradient>

        <!-- Star 1 (Sing-Box Nova Facets): Violet to Indigo Gradients -->
        <linearGradient id="singbox-facet-1" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#c084fc"/>
          <stop offset="100%" stop-color="#7e22ce"/>
        </linearGradient>
        <linearGradient id="singbox-facet-2" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#9333ea"/>
          <stop offset="100%" stop-color="#581c87"/>
        </linearGradient>
        <linearGradient id="singbox-facet-3" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#a855f7"/>
          <stop offset="100%" stop-color="#3b0764"/>
        </linearGradient>

        <!-- Star 2 (Mihomo Nova Facets): Cyan to Sky Blue Gradients -->
        <linearGradient id="mihomo-facet-1" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#67e8f9"/>
          <stop offset="100%" stop-color="#0284c7"/>
        </linearGradient>
        <linearGradient id="mihomo-facet-2" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#00d2ff"/>
          <stop offset="100%" stop-color="#0369a1"/>
        </linearGradient>
        <linearGradient id="mihomo-facet-3" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#38bdf8"/>
          <stop offset="100%" stop-color="#075985"/>
        </linearGradient>

        <!-- Central Meridian Laser Fusion Line -->
        <linearGradient id="fusion-beam" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#c084fc"/>
          <stop offset="50%" stop-color="#ffffff"/>
          <stop offset="100%" stop-color="#00d2ff"/>
        </linearGradient>

        <!-- Bevel & Shadow Filter -->
        <filter id="nova-glow" x="-20%" y="-20%" width="140%" height="140%">
          <feGaussianBlur stdDeviation="3" result="blur"/>
          <feMerge>
            <feMergeNode in="blur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
      </defs>

      <!-- Ambient Space Core -->
      <rect x="0" y="0" width="200" height="200" rx="36" fill="url(#prism-bg)"/>

      <!-- Geometric Crystal Prismatic Hypercube (Interlocking Dual Star Tetrahedrons) -->
      <!-- Top-Left Nova Star Body (sing-box Quantum Core) -->
      <polygon points="100,26 150,56 100,88" fill="url(#singbox-facet-1)" opacity="0.95"/>
      <polygon points="100,26 100,88 46,58" fill="url(#singbox-facet-2)" opacity="0.95"/>
      <polygon points="46,58 100,88 74,124 46,120" fill="url(#singbox-facet-3)" opacity="0.9"/>

      <!-- Bottom-Right Nova Star Body (mihomo Kinetic Laser Core) -->
      <polygon points="100,166 154,120 100,104" fill="url(#mihomo-facet-2)" opacity="0.95"/>
      <polygon points="100,166 100,104 46,120" fill="url(#mihomo-facet-3)" opacity="0.95"/>
      <polygon points="154,58 154,120 100,104 100,88" fill="url(#mihomo-facet-1)" opacity="0.9"/>

      <!-- Central Interlocking Dual Star Crystal Facets -->
      <polygon points="100,88 126,96 100,104 74,96" fill="#ffffff" opacity="0.85" filter="url(#nova-glow)"/>

      <!-- Outer Cube Silhouette Wireframe -->
      <polygon points="100,26 154,58 154,120 100,166 46,120 46,58" 
               fill="none" stroke="#00d2ff" stroke-width="1.8" stroke-linejoin="round" opacity="0.8"/>

      <!-- Internal Diagonal Geometric Creases -->
      <line x1="100" y1="26" x2="100" y2="88" stroke="#e0e7ff" stroke-width="2" stroke-linecap="round"/>
      <line x1="100" y1="104" x2="100" y2="166" stroke="#e0e7ff" stroke-width="2" stroke-linecap="round"/>
      <line x1="46" y1="58" x2="100" y2="88" stroke="#c084fc" stroke-width="1.6"/>
      <line x1="154" y1="58" x2="100" y2="88" stroke="#38bdf8" stroke-width="1.6"/>
      <line x1="46" y1="120" x2="100" y2="104" stroke="#a855f7" stroke-width="1.6"/>
      <line x1="154" y1="120" x2="100" y2="104" stroke="#00d2ff" stroke-width="1.6"/>

      <!-- Binary Focal Star Points (Twin Stars Sparkles) -->
      <g filter="url(#nova-glow)" class="star-sparkle-1">
        <circle cx="82" cy="62" r="5" fill="#ffffff"/>
        <circle cx="82" cy="62" r="2.5" fill="#c084fc"/>
        <line x1="74" y1="62" x2="90" y2="62" stroke="#ffffff" stroke-width="1.4" stroke-linecap="round"/>
        <line x1="82" y1="54" x2="82" y2="70" stroke="#ffffff" stroke-width="1.4" stroke-linecap="round"/>
      </g>

      <g filter="url(#nova-glow)" class="star-sparkle-2">
        <circle cx="118" cy="130" r="5" fill="#ffffff"/>
        <circle cx="118" cy="130" r="2.5" fill="#00d2ff"/>
        <line x1="110" y1="130" x2="126" y2="130" stroke="#ffffff" stroke-width="1.4" stroke-linecap="round"/>
        <line x1="118" y1="122" x2="118" y2="138" stroke="#ffffff" stroke-width="1.4" stroke-linecap="round"/>
      </g>

      <!-- Central Laser Interconnection (Millisecond Dual-Core Bus) -->
      <line x1="82" y1="62" x2="118" y2="130" stroke="url(#fusion-beam)" stroke-width="2" stroke-dasharray="3 2" filter="url(#nova-glow)" class="laser-bus"/>
    </svg>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  size: {
    type: [Number, String],
    default: 48
  },
  status: {
    type: String,
    default: 'running',
    validator: (v) => ['running', 'healing', 'idle'].includes(v)
  }
})

const normalizedSize = computed(() => {
  return typeof props.size === 'number' ? `${props.size}px` : props.size
})

const statusTitle = computed(() => {
  switch (props.status) {
    case 'healing':
      return '自愈重载中'
    case 'idle':
      return '待命'
    case 'running':
    default:
      return '满血运行'
  }
})

const ambientGlowClass = computed(() => {
  switch (props.status) {
    case 'healing':
      return 'bg-amber-400/40 scale-110 animate-pulse'
    case 'idle':
      return 'bg-slate-500/15'
    case 'running':
    default:
      return 'bg-cyan-500/20 group-hover:bg-cyan-500/35'
  }
})

const animationClass = computed(() => {
  if (props.status === 'healing') {
    return 'animate-bounce'
  }
  return ''
})
</script>

<style scoped>
@keyframes pulseSparkle {
  0%, 100% { opacity: 0.75; transform: scale(0.95); }
  50% { opacity: 1; transform: scale(1.15); }
}

@keyframes laserFlow {
  0% { stroke-dashoffset: 20; }
  100% { stroke-dashoffset: 0; }
}

.star-sparkle-1 {
  animation: pulseSparkle 3s ease-in-out infinite;
  transform-origin: 82px 62px;
}

.star-sparkle-2 {
  animation: pulseSparkle 3s ease-in-out infinite 1.5s;
  transform-origin: 118px 130px;
}

.laser-bus {
  animation: laserFlow 1.2s linear infinite;
}
</style>
