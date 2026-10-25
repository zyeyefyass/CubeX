<template>
  <div
    class="cubex-logo-wrapper select-none inline-flex items-center justify-center relative cursor-pointer group"
    :style="{ width: normalizedSize, height: normalizedSize }"
    :title="`CubeX 3D 暮光晶曜魔方核心 [${statusTitle}]`"
  >
    <!-- Background Ambient Glow (Twilight Sunset Warm Glow) -->
    <div
      class="absolute inset-0 rounded-full blur-xl pointer-events-none transition-all duration-700"
      :class="ambientGlowClass"
    ></div>

    <!-- Official 3D X-Cube Vector with Twilight Crystal Theme Palette -->
    <svg
      :width="normalizedSize"
      :height="normalizedSize"
      viewBox="10 8 80 84"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      class="relative z-10 transition-transform duration-500 group-hover:scale-110 active:scale-95"
      :class="animationClass"
    >
      <defs>
        <!-- Ultra High-Contrast Facet Shading (Twilight Deep Violet & Amber Obsidian) -->
        <linearGradient id="vueFacetTop" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#2a1b42" stop-opacity="0.95"/>
          <stop offset="50%" stop-color="#180e28" stop-opacity="0.98"/>
          <stop offset="100%" stop-color="#0d0716" stop-opacity="1"/>
        </linearGradient>

        <linearGradient id="vueFacetLeft" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#4a240c" stop-opacity="0.95"/>
          <stop offset="60%" stop-color="#2a1406" stop-opacity="0.98"/>
          <stop offset="100%" stop-color="#120701" stop-opacity="1"/>
        </linearGradient>

        <linearGradient id="vueFacetRight" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#341638" stop-opacity="0.95"/>
          <stop offset="60%" stop-color="#1d0a20" stop-opacity="0.98"/>
          <stop offset="100%" stop-color="#0c040d" stop-opacity="1"/>
        </linearGradient>

        <!-- X Arm 1: High-Voltage Sunset Warm Gold Laser (#FF9E5E / #F59E0B / #FBBF24) -->
        <linearGradient id="vueLaserSunset" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#FF9E5E"/>
          <stop offset="35%" stop-color="#F59E0B"/>
          <stop offset="50%" stop-color="#FFFFFF"/>
          <stop offset="65%" stop-color="#FF9E5E"/>
          <stop offset="100%" stop-color="#D97706"/>
        </linearGradient>

        <!-- X Arm 2: High-Voltage Twilight Violet & Rose Laser (#C084FC / #E879F9 / #F472B6) -->
        <linearGradient id="vueLaserViolet" x1="100%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stop-color="#C084FC"/>
          <stop offset="35%" stop-color="#E879F9"/>
          <stop offset="50%" stop-color="#FFFFFF"/>
          <stop offset="65%" stop-color="#C084FC"/>
          <stop offset="100%" stop-color="#9333EA"/>
        </linearGradient>

        <!-- Celestial Quantum Orbit Rings Gradients -->
        <linearGradient id="vueOrbitSunsetGrad" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" stop-color="#FF9E5E" stop-opacity="0.95"/>
          <stop offset="50%" stop-color="#FF9E5E" stop-opacity="0.15"/>
          <stop offset="100%" stop-color="#F59E0B" stop-opacity="0.85"/>
        </linearGradient>

        <linearGradient id="vueOrbitVioletGrad" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#C084FC" stop-opacity="0.95"/>
          <stop offset="50%" stop-color="#E879F9" stop-opacity="0.15"/>
          <stop offset="100%" stop-color="#C084FC" stop-opacity="0.85"/>
        </linearGradient>

        <!-- Glow Filters -->
        <filter id="vueHyperBloom" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="3" result="glow1"/>
          <feGaussianBlur stdDeviation="1.5" result="glow2"/>
          <feMerge>
            <feMergeNode in="glow1"/>
            <feMergeNode in="glow2"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>

        <filter id="vueIntenseCore" x="-60%" y="-60%" width="220%" height="220%">
          <feGaussianBlur stdDeviation="4" result="flare"/>
          <feMerge>
            <feMergeNode in="flare"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
      </defs>

      <!-- 1. DUAL-COLOR CELESTIAL ORBITS (SUNSET GOLD VS TWILIGHT VIOLET) -->
      <!-- Sunset Gold Planetary Ring (Clockwise, Tilted Orbit) -->
      <g class="orbit-gold">
        <ellipse cx="50" cy="50" rx="43" ry="24" transform="rotate(-25 50 50)" 
          stroke="url(#vueOrbitSunsetGrad)" stroke-width="1.6" stroke-dasharray="8 12 3 16" fill="none" opacity="0.85"/>
        <!-- Orbiting Sunset Quantum Bead -->
        <circle cx="88" cy="40" r="2" fill="#FF9E5E" filter="url(#vueHyperBloom)">
          <animate attributeName="r" values="1.6;2.5;1.6" dur="2s" repeatCount="indefinite"/>
        </circle>
      </g>

      <!-- Twilight Violet Counter-Rotating Orbit -->
      <g class="orbit-violet">
        <ellipse cx="50" cy="50" rx="44" ry="22" transform="rotate(35 50 50)" 
          stroke="url(#vueOrbitVioletGrad)" stroke-width="1.2" stroke-dasharray="14 8 2 10" fill="none" opacity="0.85"/>
        <!-- Orbiting Violet Quantum Bead -->
        <circle cx="12" cy="58" r="2.2" fill="#C084FC" filter="url(#vueHyperBloom)">
          <animate attributeName="r" values="2.4;1.5;2.4" dur="2s" repeatCount="indefinite"/>
        </circle>
      </g>

      <!-- Equatorial Twilight Radar Ring -->
      <circle cx="50" cy="50" r="46.5" stroke="#FF9E5E" stroke-width="0.6" stroke-opacity="0.35" stroke-dasharray="2 18 1 12" fill="none"/>

      <!-- 2. LEVITATING 3D HIGH-CONTRAST OBSIDIAN HYPERCUBE -->
      <g class="cube-float-group">
        <!-- Top Facet (Twilight Violet Luminescence) -->
        <polygon points="50,14 82,32 50,50 18,32" fill="url(#vueFacetTop)" stroke="#C084FC" stroke-width="1.6" stroke-opacity="0.8" stroke-linejoin="round"/>
        
        <!-- Left Facet (Sunset Warm Luminescence) -->
        <polygon points="18,32 50,50 50,86 18,68" fill="url(#vueFacetLeft)" stroke="#FF9E5E" stroke-width="1.6" stroke-opacity="0.85" stroke-linejoin="round"/>
        
        <!-- Right Facet (Deep Twilight Void) -->
        <polygon points="50,50 82,32 82,68 50,86" fill="url(#vueFacetRight)" stroke="#E879F9" stroke-width="1.6" stroke-opacity="0.75" stroke-linejoin="round"/>

        <!-- Inner Circuit Detail Geometry on Facets -->
        <path d="M50,18 L76,33 M24,33 L50,48" stroke="#C084FC" stroke-width="0.6" stroke-opacity="0.45" stroke-dasharray="3 4"/>
        <path d="M22,37 L46,51 M22,63 L46,77" stroke="#FF9E5E" stroke-width="0.6" stroke-opacity="0.45" stroke-dasharray="3 4"/>

        <!-- Subtle Silhouette Wireframe Outer Ring -->
        <path d="M50,14 L82,32 L82,68 L50,86 L18,68 L18,32 Z" stroke="#FFFFFF" stroke-width="0.6" stroke-opacity="0.35" fill="none"/>

        <!-- 3. THE "X" LIGHT STRIPS -->
        <!-- Arm 1: High-Voltage Sunset Warm Gold Laser (18,32 -> 50,50 -> 82,68) -->
        <path d="M18,32 L50,50 L82,68" stroke="#3d1e0a" stroke-width="4.5" stroke-linecap="round" stroke-linejoin="round"/>
        <path class="ambient-flash" d="M18,32 L50,50 L82,68" stroke="#FF9E5E" stroke-width="6.5" stroke-linecap="round" stroke-linejoin="round" opacity="0.6" filter="url(#vueHyperBloom)"/>
        <path class="flow-sunset-beam" d="M18,32 L50,50 L82,68" stroke="url(#vueLaserSunset)" stroke-width="3.2" stroke-linecap="round" stroke-linejoin="round" filter="url(#vueHyperBloom)"/>
        <path d="M18,32 L50,50 L82,68" stroke="#FFFFFF" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" opacity="0.95"/>

        <!-- Arm 2: High-Voltage Twilight Violet Laser (82,32 -> 50,50 -> 18,68) -->
        <path d="M82,32 L50,50 L18,68" stroke="#2a122e" stroke-width="4.5" stroke-linecap="round" stroke-linejoin="round"/>
        <path class="ambient-flash" d="M82,32 L50,50 L18,68" stroke="#C084FC" stroke-width="6.5" stroke-linecap="round" stroke-linejoin="round" opacity="0.6" filter="url(#vueHyperBloom)"/>
        <path class="flow-violet-beam" d="M82,32 L50,50 L18,68" stroke="url(#vueLaserViolet)" stroke-width="3.2" stroke-linecap="round" stroke-linejoin="round" filter="url(#vueHyperBloom)"/>
        <path d="M82,32 L50,50 L18,68" stroke="#FFFFFF" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" opacity="0.95"/>

        <!-- 4. HIGH-ENERGY CORNER CAPACITORS (4 ANCHOR TERMINALS) -->
        <!-- Top-Left Terminal (Sunset Warm Gold) -->
        <circle cx="18" cy="32" r="3.2" fill="#FF9E5E" filter="url(#vueHyperBloom)"/>
        <circle cx="18" cy="32" r="1.4" fill="#FFFFFF"/>
        <circle cx="18" cy="32" r="5" stroke="#FF9E5E" stroke-width="0.8" fill="none" opacity="0.7">
          <animate attributeName="r" values="3.5;6.5;3.5" dur="1.8s" repeatCount="indefinite"/>
          <animate attributeName="opacity" values="0.8;0;0.8" dur="1.8s" repeatCount="indefinite"/>
        </circle>

        <!-- Bottom-Right Terminal (Sunset Warm Gold) -->
        <circle cx="82" cy="68" r="3.2" fill="#FF9E5E" filter="url(#vueHyperBloom)"/>
        <circle cx="82" cy="68" r="1.4" fill="#FFFFFF"/>
        <circle cx="82" cy="68" r="5" stroke="#FF9E5E" stroke-width="0.8" fill="none" opacity="0.7">
          <animate attributeName="r" values="3.5;6.5;3.5" dur="1.8s" repeatCount="indefinite"/>
          <animate attributeName="opacity" values="0.8;0;0.8" dur="1.8s" repeatCount="indefinite"/>
        </circle>

        <!-- Top-Right Terminal (Twilight Violet) -->
        <circle cx="82" cy="32" r="3.2" fill="#C084FC" filter="url(#vueHyperBloom)"/>
        <circle cx="82" cy="32" r="1.4" fill="#FFFFFF"/>
        <circle cx="82" cy="32" r="5" stroke="#C084FC" stroke-width="0.8" fill="none" opacity="0.7">
          <animate attributeName="r" values="3.5;6.5;3.5" dur="1.8s" repeatCount="indefinite" begin="0.9s"/>
          <animate attributeName="opacity" values="0.8;0;0.8" dur="1.8s" repeatCount="indefinite" begin="0.9s"/>
        </circle>

        <!-- Bottom-Left Terminal (Twilight Violet) -->
        <circle cx="18" cy="68" r="3.2" fill="#C084FC" filter="url(#vueHyperBloom)"/>
        <circle cx="18" cy="68" r="1.4" fill="#FFFFFF"/>
        <circle cx="18" cy="68" r="5" stroke="#C084FC" stroke-width="0.8" fill="none" opacity="0.7">
          <animate attributeName="r" values="3.5;6.5;3.5" dur="1.8s" repeatCount="indefinite" begin="0.9s"/>
          <animate attributeName="opacity" values="0.8;0;0.8" dur="1.8s" repeatCount="indefinite" begin="0.9s"/>
        </circle>

        <!-- 5. NUCLEAR FUSION CENTER SINGULARITY (INTERSECTION AT 50, 50) -->
        <!-- Concentric Shockwave Energy Ripples from Collision -->
        <circle cx="50" cy="50" r="4" stroke="#FFFFFF" stroke-width="1.5" fill="none" opacity="0.9">
          <animate attributeName="r" values="2;18" dur="1.8s" repeatCount="indefinite"/>
          <animate attributeName="opacity" values="1;0" dur="1.8s" repeatCount="indefinite"/>
          <animate attributeName="stroke" values="#FFFFFF;#FF9E5E;#C084FC;#F59E0B" dur="1.8s" repeatCount="indefinite"/>
        </circle>
        <circle cx="50" cy="50" r="4" stroke="#FF9E5E" stroke-width="1.6" fill="none" opacity="0.9">
          <animate attributeName="r" values="2;18" dur="1.8s" repeatCount="indefinite" begin="0.9s"/>
          <animate attributeName="opacity" values="1;0" dur="1.8s" repeatCount="indefinite" begin="0.9s"/>
          <animate attributeName="stroke" values="#FFFFFF;#C084FC;#F59E0B;#FF9E5E" dur="1.8s" repeatCount="indefinite" begin="0.9s"/>
        </circle>

        <!-- Anamorphic Laser Flare Beam across the horizontal axis -->
        <line x1="28" y1="50" x2="72" y2="50" stroke="#FFFFFF" stroke-width="1.4" opacity="0.9" filter="url(#vueIntenseCore)">
          <animate attributeName="stroke-width" values="0.8;2;0.8" dur="2s" repeatCount="indefinite"/>
          <animate attributeName="opacity" values="0.7;1;0.7" dur="2s" repeatCount="indefinite"/>
        </line>

        <!-- 4-Point Rotating Cyber Nova Starburst -->
        <g class="nova-star">
          <!-- Outer Sunset Gold Starburst -->
          <polygon points="50,38 53.5,46.5 62,50 53.5,53.5 50,62 46.5,53.5 38,50 46.5,46.5" 
            fill="url(#vueLaserSunset)" filter="url(#vueHyperBloom)"/>
          <!-- Inner Twilight Violet Starlet -->
          <polygon points="50,42 52.5,47.5 58,50 52.5,52.5 50,58 47.5,52.5 42,50 47.5,47.5" 
            fill="url(#vueLaserViolet)" transform="rotate(45 50 50)"/>
          <!-- Brilliant Pure White Fusion Core -->
          <circle cx="50" cy="50" r="3.2" fill="#FFFFFF" filter="url(#vueHyperBloom)"/>
          <circle cx="50" cy="50" r="1.6" fill="#FF9E5E"/>
        </g>
      </g>
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
      return 'bg-[#FF9E5E]/25 group-hover:bg-[#FF9E5E]/40'
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
@keyframes cubeFloat {
  0%, 100% { transform: translateY(0px); }
  50% { transform: translateY(-3.5px); }
}
@keyframes spinClockwise {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
@keyframes spinCounter {
  0% { transform: rotate(360deg); }
  100% { transform: rotate(0deg); }
}
@keyframes starSpin {
  0% { transform: rotate(0deg) scale(0.9); }
  50% { transform: rotate(180deg) scale(1.25); }
  100% { transform: rotate(360deg) scale(0.9); }
}
@keyframes beamFlow1 {
  0% { stroke-dashoffset: 148; }
  100% { stroke-dashoffset: 0; }
}
@keyframes beamFlow2 {
  0% { stroke-dashoffset: -148; }
  100% { stroke-dashoffset: 0; }
}
@keyframes flashPulse {
  0%, 100% { opacity: 0.55; }
  50% { opacity: 1; }
}

.cube-float-group {
  animation: cubeFloat 4s ease-in-out infinite;
  transform-origin: 50px 50px;
}
.orbit-gold {
  animation: spinClockwise 10s linear infinite;
  transform-origin: 50px 50px;
}
.orbit-violet {
  animation: spinCounter 12s linear infinite;
  transform-origin: 50px 50px;
}
.nova-star {
  animation: starSpin 6s linear infinite;
  transform-origin: 50px 50px;
}
.flow-sunset-beam {
  stroke-dasharray: 24 50;
  animation: beamFlow1 1.4s linear infinite;
}
.flow-violet-beam {
  stroke-dasharray: 24 50;
  animation: beamFlow2 1.4s linear infinite;
}
.ambient-flash {
  animation: flashPulse 2.4s ease-in-out infinite;
}
</style>
