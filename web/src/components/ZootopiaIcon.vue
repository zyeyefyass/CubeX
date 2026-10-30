<template>
  <span
    class="zootopia-icon-container inline-flex items-center justify-center select-none shrink-0 transition-transform duration-300"
    :class="[
      active ? 'scale-105 filter drop-shadow-[0_2px_8px_rgba(255,158,94,0.45)]' : 'opacity-90 hover:opacity-100 hover:scale-110',
      customClass
    ]"
    :style="{ width: computedSize, height: computedSize }"
    :title="title || defaultTitle"
  >
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- 1. OVERVIEW (全息中枢 / 概览): 动物城中央生态大都会全景天际线与极光轨交 -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-if="normalizedName === 'overview' || normalizedName === 'metropolis'"
      :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="zt-metro-bg" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#221838" />
          <stop offset="50%" stop-color="#181126" />
          <stop offset="100%" stop-color="#0F0A1A" />
        </linearGradient>
        <linearGradient id="zt-metro-skyline" x1="16" y1="10" x2="16" y2="28" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#FF9E5E" />
          <stop offset="60%" stop-color="#D97706" />
          <stop offset="100%" stop-color="#312E81" />
        </linearGradient>
        <linearGradient id="zt-metro-rail" x1="2" y1="26" x2="30" y2="16" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#FF9E5E" stop-opacity="0.2" />
          <stop offset="50%" stop-color="#FDE68A" />
          <stop offset="100%" stop-color="#FF9E5E" />
        </linearGradient>
      </defs>

      <!-- Glass Rounded Badge Removed for Pure Float -->

      <!-- Twilight Stars -->
      <circle cx="6" cy="6.5" r="0.8" fill="#FDE68A" opacity="0.8" />
      <circle cx="26" cy="7.5" r="0.7" fill="#FFFFFF" opacity="0.9" />
      <circle cx="20" cy="5" r="0.6" fill="#FDE68A" opacity="0.7" />
      <circle cx="10" cy="11" r="0.5" fill="#FFFFFF" opacity="0.6" />

      <!-- Zootopia Skyline Silhouette -->
      <!-- Sahara Palm Dome (Left) -->
      <path d="M4 27 L4 21 C4 18 7 15 10 15 C13 15 15 17 15 21 L15 27 Z" fill="url(#zt-metro-skyline)" opacity="0.85" />
      <path d="M10 15 L10 27" stroke="rgba(254, 240, 138, 0.4)" stroke-width="0.8" stroke-dasharray="1 1.5" />

      <!-- Center Metropolis Spire (Judy & Nick's City Tower) -->
      <path d="M14 27 L15 10 L16 6 L17 10 L18 27 Z" fill="url(#zt-metro-skyline)" />
      <!-- Glowing Spire Needle -->
      <line x1="16" y1="4" x2="16" y2="7" stroke="#FDE68A" stroke-width="1.2" stroke-linecap="round" />
      <circle cx="16" cy="4" r="1.2" fill="#FEF08A" />

      <!-- Tundratown Crystal Highrises (Right) -->
      <path d="M19 27 L19 18 L22 14 L25 18 L25 27 Z" fill="url(#zt-metro-skyline)" opacity="0.9" />
      <path d="M25 27 L25 20 L28 17 L29 27 Z" fill="url(#zt-metro-skyline)" opacity="0.75" />

      <!-- Micro Illuminated Windows -->
      <circle cx="16" cy="13" r="0.6" fill="#FEF08A" />
      <circle cx="16" cy="17" r="0.6" fill="#FEF08A" />
      <circle cx="16" cy="21" r="0.6" fill="#FEF08A" />
      <circle cx="9" cy="19" r="0.6" fill="#FEF08A" opacity="0.9" />
      <circle cx="22" cy="18" r="0.6" fill="#38BDF8" opacity="0.9" />

      <!-- Dynamic High-Speed Maglev Rail Arc (跨生态光轨) -->
      <path d="M2 25.5 C10 25.5 14 18 22 17.5 C26 17.2 28.5 19 30 20.5" stroke="url(#zt-metro-rail)" stroke-width="1.8" stroke-linecap="round" fill="none" />
      <!-- Speeding Train Light Pulse -->
      <circle cx="22" cy="17.5" r="1.6" fill="#FFFFFF" />
      <circle cx="22" cy="17.5" r="2.8" stroke="#FDE68A" stroke-width="0.8" fill="none" opacity="0.8" />
    </svg>

    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- 2. ROUTING (应用分流 / 节点): 动物城全生态领航罗盘 · 狐兔极光指针   -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-else-if="normalizedName === 'routing' || normalizedName === 'compass'"
      :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="zt-compass-rim" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#FDE68A" />
          <stop offset="40%" stop-color="#FF9E5E" />
          <stop offset="100%" stop-color="#B45309" />
        </linearGradient>
        <linearGradient id="zt-compass-glass" x1="16" y1="4" x2="16" y2="28" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#241B3B" />
          <stop offset="100%" stop-color="#120D20" />
        </linearGradient>
        <linearGradient id="zt-needle-nick" x1="16" y1="16" x2="24" y2="8" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#EA580C" />
          <stop offset="100%" stop-color="#FF9E5E" />
        </linearGradient>
        <linearGradient id="zt-needle-judy" x1="16" y1="16" x2="8" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#0284C7" />
          <stop offset="100%" stop-color="#38BDF8" />
        </linearGradient>
      </defs>

      <!-- Top Watch Crown / Suspension Ring -->
      <path d="M16 1.5 C14.5 1.5 13.5 2.5 13.5 4 C13.5 4.8 14 5.5 14.8 5.8 L17.2 5.8 C18 5.5 18.5 4.8 18.5 4 C18.5 2.5 17.5 1.5 16 1.5 Z" stroke="url(#zt-compass-rim)" stroke-width="1.2" fill="none" />

      <!-- Outer Chronometer Bezel -->
      <circle cx="16" cy="17" r="13" fill="url(#zt-compass-glass)" stroke="url(#zt-compass-rim)" stroke-width="1.8" />
      <circle cx="16" cy="17" r="11" stroke="rgba(255, 240, 230, 0.18)" stroke-width="0.8" fill="none" />

      <!-- Cardinal Dial Ticks (N, S, E, W) -->
      <line x1="16" y1="7" x2="16" y2="9" stroke="#FDE68A" stroke-width="1.5" stroke-linecap="round" />
      <line x1="16" y1="25" x2="16" y2="27" stroke="rgba(255,240,230,0.5)" stroke-width="1.2" stroke-linecap="round" />
      <line x1="6" y1="17" x2="8" y2="17" stroke="rgba(255,240,230,0.5)" stroke-width="1.2" stroke-linecap="round" />
      <line x1="24" y1="17" x2="26" y2="17" stroke="rgba(255,240,230,0.5)" stroke-width="1.2" stroke-linecap="round" />

      <!-- Sub-degree Dots -->
      <circle cx="10" cy="11" r="0.6" fill="rgba(255,240,230,0.4)" />
      <circle cx="22" cy="11" r="0.6" fill="rgba(255,240,230,0.4)" />
      <circle cx="10" cy="23" r="0.6" fill="rgba(255,240,230,0.4)" />
      <circle cx="22" cy="23" r="0.6" fill="rgba(255,240,230,0.4)" />

      <!-- North Indicator: Nick Wilde Sunset Amber Point (Pointing NE) -->
      <polygon points="16,17 14.5,14 23.5,9.5 19,18.5" fill="url(#zt-needle-nick)" />
      <polygon points="16,17 19,18.5 23.5,9.5" fill="#FFA970" opacity="0.6" />

      <!-- South Indicator: Judy Hopps ZPD Cyan Point (Pointing SW) -->
      <polygon points="16,17 17.5,20 8.5,24.5 13,15.5" fill="url(#zt-needle-judy)" />
      <polygon points="16,17 13,15.5 8.5,24.5" fill="#7DD3FC" opacity="0.6" />

      <!-- Center Brass Axis Pin -->
      <circle cx="16" cy="17" r="2.6" fill="#F59E0B" stroke="#FEF08A" stroke-width="0.8" />
      <circle cx="16" cy="17" r="1.1" fill="#FFFFFF" />
    </svg>

    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- 3. RULES (安全防护 / 规则): ZPD 朱迪警官荣誉金星八角警徽 · 极光防线 -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-else-if="normalizedName === 'rules' || normalizedName === 'shield'"
      :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="zt-badge-gold" x1="4" y1="2" x2="28" y2="30" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#FEF08A" />
          <stop offset="35%" stop-color="#F59E0B" />
          <stop offset="70%" stop-color="#FF9E5E" />
          <stop offset="100%" stop-color="#B45309" />
        </linearGradient>
        <linearGradient id="zt-badge-glass" x1="16" y1="3" x2="16" y2="29" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#1E1B4B" />
          <stop offset="50%" stop-color="#141128" />
          <stop offset="100%" stop-color="#0B0914" />
        </linearGradient>
      </defs>

      <!-- ZPD Police Hex-Shield Contour with High-Relief Bevel -->
      <path d="M16 2.5 L27 6.5 C27 18 16 29.5 16 29.5 C16 29.5 5 18 5 6.5 Z" fill="url(#zt-badge-gold)" />
      <path d="M16 4.5 L25 7.8 C25 17.2 16 26.8 16 26.8 C16 26.8 7 17.2 7 7.8 Z" fill="url(#zt-badge-glass)" />

      <!-- Inner Specular Light Line (macOS 1px rim highlight) -->
      <path d="M16 5.5 L24 8.5 C24 16.5 16 25 16 25" stroke="rgba(255, 255, 255, 0.35)" stroke-width="0.8" fill="none" />

      <!-- Center: 8-Point ZPD Officer Gold Star -->
      <g transform="translate(16, 15.5) scale(0.9)">
        <!-- Vertical & Horizontal Primary Rays -->
        <polygon points="0,-9.5 2,-2.5 9,-2.5 3.5,1.5 6,8.5 0,4 -6,8.5 -3.5,1.5 -9,-2.5 -2,-2.5" fill="url(#zt-badge-gold)" />
        <!-- Diagonal Accent Rays -->
        <polygon points="0,-7 1.5,-2 7,-2 2.5,1.5 4.5,6.5 0,3 -4.5,6.5 -2.5,1.5 -7,-2 -1.5,-2" fill="#FEF08A" opacity="0.6" />
      </g>

      <!-- Carved Carrot Emblem Badge at Star Center (Officer Judy Hopps Motif) -->
      <!-- Carrot Body -->
      <path d="M16 13 L17.5 15.5 C17.5 17.5 16 20.5 16 20.5 C16 20.5 14.5 17.5 14.5 15.5 Z" fill="#FF9E5E" stroke="#FEF08A" stroke-width="0.5" />
      <!-- Carrot Leaves -->
      <path d="M16 13 L15 11.5 M16 13 L16 11 M16 13 L17 11.5" stroke="#34D399" stroke-width="0.9" stroke-linecap="round" />
      <!-- Star Center Sparkle -->
      <circle cx="16" cy="16" r="0.8" fill="#FFFFFF" />
    </svg>

    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- 4. KSPEEDER (容器加速 / 极速): 动物城穿梭特快磁悬浮列车 · 极速光轨 -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-else-if="normalizedName === 'kspeeder' || normalizedName === 'train' || normalizedName === 'speed'"
      :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="zt-train-body" x1="6" y1="26" x2="26" y2="6" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#94A3B8" />
          <stop offset="50%" stop-color="#F1F5F9" />
          <stop offset="100%" stop-color="#FFFFFF" />
        </linearGradient>
        <linearGradient id="zt-train-stripe" x1="8" y1="24" x2="24" y2="8" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#EA580C" />
          <stop offset="50%" stop-color="#FF9E5E" />
          <stop offset="100%" stop-color="#FDE68A" />
        </linearGradient>
        <linearGradient id="zt-cockpit-glass" x1="18" y1="8" x2="26" y2="14" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#38BDF8" />
          <stop offset="100%" stop-color="#0369A1" />
        </linearGradient>
        <linearGradient id="zt-thrust-glow" x1="4" y1="28" x2="14" y2="18" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#FF9E5E" stop-opacity="0" />
          <stop offset="100%" stop-color="#FF9E5E" stop-opacity="0.9" />
        </linearGradient>
      </defs>

      <!-- Supersonic Thrust Tracks / Kinetic Slipstream Lines -->
      <line x1="2" y1="28" x2="11" y2="20" stroke="url(#zt-train-stripe)" stroke-width="2" stroke-linecap="round" />
      <line x1="5" y1="31" x2="12" y2="25" stroke="#FDE68A" stroke-width="1.2" stroke-linecap="round" opacity="0.8" />
      <line x1="1" y1="23" x2="8" y2="17" stroke="#38BDF8" stroke-width="1.2" stroke-linecap="round" opacity="0.8" />

      <!-- High-Speed Zootopia Express Bullet Train Nose (Aerodynamic Curvature) -->
      <path d="M7 25 L16 14 C18 11.5 22 7.5 25 6 C27.5 5 28.5 6 28 8.5 C26.5 11.5 22.5 15.5 20 17.5 L9 27 Z"
        fill="url(#zt-train-body)" stroke="#CBD5E1" stroke-width="1" />

      <!-- Aerodynamic Amber Racing Stripe (Nick & Judy Fast Line) -->
      <path d="M8 25 L18 14.5 C20 12.5 24 9 27 7.5 L26 9.5 C23 11 19 14.5 17 16.5 L8.5 25.5 Z"
        fill="url(#zt-train-stripe)" />

      <!-- Panoramic Cockpit Glass (VisionOS Tint) -->
      <path d="M22 8.5 C23.5 7.5 26 6.8 27.5 6.8 C27.8 7.5 27.2 9 26 10 C24.5 11.2 23 12 21.5 11 Z"
        fill="url(#zt-cockpit-glass)" />
      <!-- Cockpit Specular Reflection -->
      <path d="M23 8 C24.5 7.4 26 7.2 26.8 7.2" stroke="#FFFFFF" stroke-width="0.8" stroke-linecap="round" />

      <!-- Passenger Windows Strip -->
      <circle cx="15.5" cy="17" r="0.9" fill="#0284C7" />
      <circle cx="13" cy="19.5" r="0.9" fill="#0284C7" />
      <circle cx="10.5" cy="22" r="0.9" fill="#0284C7" />

      <!-- Kinetic Particle Sparkles -->
      <circle cx="28" cy="4" r="0.8" fill="#FDE68A" />
      <circle cx="30" cy="9" r="0.6" fill="#FFFFFF" />
    </svg>

    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- 5. SYSTEM (宿主共生 / 运维): 生态调控动力晶核 · 狐兔共生双环齿轮   -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-else-if="normalizedName === 'system' || normalizedName === 'core' || normalizedName === 'gears'"
      :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="zt-gear-gold" x1="4" y1="6" x2="24" y2="26" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#FEF08A" />
          <stop offset="50%" stop-color="#FF9E5E" />
          <stop offset="100%" stop-color="#B45309" />
        </linearGradient>
        <linearGradient id="zt-gear-slate" x1="16" y1="4" x2="28" y2="16" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#CBD5E1" />
          <stop offset="100%" stop-color="#475569" />
        </linearGradient>
        <radialGradient id="zt-core-pulse" cx="13" cy="18" r="6" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#FEF08A" />
          <stop offset="40%" stop-color="#FF9E5E" />
          <stop offset="100%" stop-color="#2DD4BF" stop-opacity="0" />
        </radialGradient>
      </defs>

      <!-- Secondary Interlocking Satellite Gear (Titanium Alloy - Top Right) -->
      <g transform="translate(22, 10)">
        <circle cx="0" cy="0" r="6" fill="#1E192E" stroke="url(#zt-gear-slate)" stroke-width="1.2" />
        <!-- 6 Satellite Teeth -->
        <rect x="-1" y="-7.5" width="2" height="2" rx="0.5" fill="url(#zt-gear-slate)" />
        <rect x="-1" y="5.5" width="2" height="2" rx="0.5" fill="url(#zt-gear-slate)" />
        <rect x="-7.5" y="-1" width="2" height="2" rx="0.5" fill="url(#zt-gear-slate)" />
        <rect x="5.5" y="-1" width="2" height="2" rx="0.5" fill="url(#zt-gear-slate)" />
        <circle cx="0" cy="0" r="2.2" fill="#0F0C1B" stroke="#94A3B8" stroke-width="0.8" />
      </g>

      <!-- Primary Climate Symbiosis Master Gear (Sunset Gold - Center/Bottom Left) -->
      <g transform="translate(13, 19)">
        <!-- 8 Precision Master Gear Teeth -->
        <rect x="-2" y="-12" width="4" height="3" rx="0.8" fill="url(#zt-gear-gold)" />
        <rect x="-2" y="9" width="4" height="3" rx="0.8" fill="url(#zt-gear-gold)" />
        <rect x="-12" y="-2" width="3" height="4" rx="0.8" fill="url(#zt-gear-gold)" />
        <rect x="9" y="-2" width="3" height="4" rx="0.8" fill="url(#zt-gear-gold)" />
        <!-- Diagonal Teeth -->
        <rect x="-9" y="-9" width="3.5" height="3.5" rx="0.8" fill="url(#zt-gear-gold)" transform="rotate(45)" />
        <rect x="5.5" y="-9" width="3.5" height="3.5" rx="0.8" fill="url(#zt-gear-gold)" transform="rotate(45)" />
        <rect x="-9" y="5.5" width="3.5" height="3.5" rx="0.8" fill="url(#zt-gear-gold)" transform="rotate(45)" />
        <rect x="5.5" y="5.5" width="3.5" height="3.5" rx="0.8" fill="url(#zt-gear-gold)" transform="rotate(45)" />

        <!-- Gear Rim Body -->
        <circle cx="0" cy="0" r="9.5" fill="#1C142D" stroke="url(#zt-gear-gold)" stroke-width="1.8" />
        <circle cx="0" cy="0" r="7.5" stroke="rgba(255, 240, 230, 0.2)" stroke-width="0.8" fill="none" />

        <!-- Central Quantum Symbiotic Power Core -->
        <circle cx="0" cy="0" r="5" fill="url(#zt-core-pulse)" />
        <circle cx="0" cy="0" r="3" fill="#F59E0B" stroke="#FFFFFF" stroke-width="1" />
        <circle cx="0" cy="0" r="1.2" fill="#FFFFFF" />
      </g>

      <!-- Symbiotic Bridge Conduits (RouterOS x CubeX Interconnect) -->
      <path d="M17 14 C19 13 20 12 21 11" stroke="#2DD4BF" stroke-width="1.2" stroke-linecap="round" stroke-dasharray="1 1.5" />
    </svg>

    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- 6. AI COPILOT (AI管家): 动物城全息智慧晶核 · AI 伴生管家星芒         -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-else-if="normalizedName === 'ai' || normalizedName === 'copilot' || normalizedName === 'bot'"
      :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="zt-ai-bg" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#2D124D" />
          <stop offset="50%" stop-color="#1B0F2E" />
          <stop offset="100%" stop-color="#0E081A" />
        </linearGradient>
        <linearGradient id="zt-ai-star" x1="16" y1="6" x2="16" y2="26" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#FFFFFF" />
          <stop offset="50%" stop-color="#FDE68A" />
          <stop offset="100%" stop-color="#FF9E5E" />
        </linearGradient>
      </defs>

      <!-- Glass Rounded Badge Removed for Pure Float -->

      <!-- Quantum Orbital Ring -->
      <ellipse cx="16" cy="16" rx="11" ry="5.5" stroke="rgba(56, 189, 248, 0.5)" stroke-width="1" stroke-dasharray="2 2" transform="rotate(-25 16 16)" fill="none" />
      <circle cx="24" cy="12" r="1.2" fill="#38BDF8" />

      <!-- Central 4-Point AI Sparkle Star (AI 核心星芒) -->
      <path d="M16 6 C16 12 12 16 6 16 C12 16 16 20 16 26 C16 20 20 16 26 16 C20 16 16 12 16 6 Z" fill="url(#zt-ai-star)" filter="drop-shadow(0 0 4px rgba(253, 230, 138, 0.8))" />

      <!-- Inner Micro Core -->
      <circle cx="16" cy="16" r="2.2" fill="#FFFFFF" />

      <!-- Small Companion Stars -->
      <circle cx="9" cy="9" r="0.8" fill="#FDE68A" opacity="0.9" />
      <circle cx="23" cy="23" r="0.8" fill="#A855F7" opacity="0.9" />
    </svg>

    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- 6. DIAGNOSTICS (日志诊断): 朱迪数字录音胡萝卜笔与全息终端档案     -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-else-if="normalizedName === 'diagnostics' || normalizedName === 'recorder' || normalizedName === 'logs'"
      :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="zt-carrot-body" x1="8" y1="26" x2="22" y2="8" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#EA580C" />
          <stop offset="60%" stop-color="#FF9E5E" />
          <stop offset="100%" stop-color="#FBBF24" />
        </linearGradient>
        <linearGradient id="zt-leaf-green" x1="18" y1="8" x2="28" y2="2" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#10B981" />
          <stop offset="100%" stop-color="#34D399" />
        </linearGradient>
        <linearGradient id="zt-hud-glass" x1="2" y1="6" x2="18" y2="20" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="rgba(30, 24, 52, 0.75)" />
          <stop offset="100%" stop-color="rgba(15, 11, 26, 0.85)" />
        </linearGradient>
      </defs>

      <!-- Holographic Audio Waveform / Log Telemetry Screen (Left/Background) -->
      <rect x="2.5" y="4.5" width="16" height="15" rx="3.5" fill="url(#zt-hud-glass)" stroke="rgba(255, 240, 230, 0.22)" stroke-width="1" />
      <!-- Waveform Bars (Telemetry Soundwave captured by Carrot Pen) -->
      <line x1="5.5" y1="12" x2="5.5" y2="14" stroke="#FDE68A" stroke-width="1.2" stroke-linecap="round" />
      <line x1="8" y1="10" x2="8" y2="15" stroke="#FF9E5E" stroke-width="1.2" stroke-linecap="round" />
      <line x1="10.5" y1="8" x2="10.5" y2="16" stroke="#FEF08A" stroke-width="1.2" stroke-linecap="round" />
      <line x1="13" y1="11" x2="13" y2="14" stroke="#38BDF8" stroke-width="1.2" stroke-linecap="round" />
      <line x1="15.5" y1="12.5" x2="15.5" y2="13.5" stroke="#FDE68A" stroke-width="1.2" stroke-linecap="round" />
      <!-- Recording Status Dot -->
      <circle cx="5" cy="7" r="1" fill="#EF4444" />

      <!-- Judy's Carrot Voice Recorder Pen Body (Angled at 45°) -->
      <!-- Carrot Tip & Barrel -->
      <path d="M7 27 L6.5 24.5 C7.5 21 11 16 16 11 L18.5 13.5 C15.5 18 11.5 24 8.5 26.5 Z"
        fill="url(#zt-carrot-body)" stroke="#F59E0B" stroke-width="0.8" />

      <!-- Carrot Horizontal Ring Grooves -->
      <path d="M10 21.5 C11 20.8 12.5 20.8 13.5 21.5" stroke="rgba(180, 83, 9, 0.6)" stroke-width="0.7" fill="none" stroke-linecap="round" />
      <path d="M12.5 17.5 C13.5 16.8 15 16.8 16 17.5" stroke="rgba(180, 83, 9, 0.6)" stroke-width="0.7" fill="none" stroke-linecap="round" />

      <!-- Silver Clip & Microphone Grille -->
      <rect x="15" y="10" width="4.5" height="2" rx="0.5" fill="#E2E8F0" stroke="#94A3B8" stroke-width="0.5" transform="rotate(45 15 10)" />
      <!-- Mic Hole -->
      <circle cx="16" cy="13.5" r="0.6" fill="#1E293B" />

      <!-- Judy's Carrot Leaves (Stylus Top / Antenna) -->
      <!-- Center Leaf -->
      <path d="M18.5 11 C20 9 22 7 24 5 C24 8 22 10 20 12 Z" fill="url(#zt-leaf-green)" />
      <!-- Left Leaf -->
      <path d="M17.5 9.5 C18 7.5 19 5.5 20.5 4 C20.5 6 19.5 8 18.5 9.5 Z" fill="url(#zt-leaf-green)" opacity="0.85" />
      <!-- Right Leaf -->
      <path d="M20 12 C22 12.5 24 13 26 12 C24.5 11 22.5 11 20 12 Z" fill="url(#zt-leaf-green)" opacity="0.9" />

      <!-- Voice Record Button (Silver Clicker with Red Glow) -->
      <circle cx="21.5" cy="8.5" r="1.5" fill="#E2E8F0" stroke="#CBD5E1" stroke-width="0.5" />
      <circle cx="21.5" cy="8.5" r="0.7" fill="#EF4444" />
    </svg>

    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- 7. HOST-BASE (宿主共生底座): Mac Studio / 钛合金共生节点             -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-else-if="normalizedName === 'host-base' || normalizedName === 'server'"
      :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="zt-host-metal" x1="4" y1="6" x2="28" y2="26" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#E2E8F0" />
          <stop offset="45%" stop-color="#94A3B8" />
          <stop offset="100%" stop-color="#475569" />
        </linearGradient>
        <linearGradient id="zt-host-face" x1="16" y1="10" x2="16" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#1E1635" />
          <stop offset="100%" stop-color="#0F0B1A" />
        </linearGradient>
      </defs>

      <!-- Sleek Metal Server Blade / Mac Studio Chassis -->
      <rect x="3.5" y="8.5" width="25" height="17" rx="4.5" fill="url(#zt-host-face)" stroke="url(#zt-host-metal)" stroke-width="1.8" />
      <!-- Top Specular Reflection Line -->
      <line x1="7" y1="10.5" x2="25" y2="10.5" stroke="rgba(255,255,255,0.4)" stroke-width="0.8" stroke-linecap="round" />

      <!-- Host Dual Status Ports (RouterOS Green & CubeX Gold) -->
      <!-- RouterOS Status LED -->
      <circle cx="8" cy="17" r="1.8" fill="#34D399" />
      <circle cx="8" cy="17" r="3" stroke="#34D399" stroke-width="0.6" opacity="0.6" />

      <!-- CubeX Status LED -->
      <circle cx="14" cy="17" r="1.8" fill="#FF9E5E" />
      <circle cx="14" cy="17" r="3" stroke="#FF9E5E" stroke-width="0.6" opacity="0.6" />

      <!-- High-Speed USB-C / SFP+ Optical Ports -->
      <rect x="19.5" y="15" width="6" height="4" rx="1.5" fill="#334155" stroke="rgba(255,240,230,0.3)" stroke-width="0.6" />
      <line x1="21.5" y1="17" x2="23.5" y2="17" stroke="#38BDF8" stroke-width="1" stroke-linecap="round" />

      <!-- Symbiotic Bridge Antenna Wave (Top Micro Arc) -->
      <path d="M11 5 C14 3.5 18 3.5 21 5" stroke="#FDE68A" stroke-width="1.2" stroke-linecap="round" fill="none" opacity="0.85" />
      <circle cx="16" cy="3.5" r="0.8" fill="#FFFFFF" />
    </svg>

    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- 8. AUTH-KEY (宿主凭据与鉴权): ZPD 金色密匙凭据与鉴权                 -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-else-if="normalizedName === 'auth-key' || normalizedName === 'key'"
      :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="zt-key-gold" x1="4" y1="4" x2="28" y2="28" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#FEF08A" />
          <stop offset="40%" stop-color="#FF9E5E" />
          <stop offset="100%" stop-color="#B45309" />
        </linearGradient>
      </defs>

      <!-- Key Bow / Octagonal Head (ZPD Crest Shape) -->
      <circle cx="11" cy="11" r="7.5" fill="#1C142E" stroke="url(#zt-key-gold)" stroke-width="2" />
      <circle cx="11" cy="11" r="4" stroke="rgba(255,240,230,0.3)" stroke-width="1" fill="none" />
      <!-- Center Keyhole / Star Cutout -->
      <polygon points="11,8.5 12,10.5 14,11 12,11.5 11,13.5 10,11.5 8,11 10,10.5" fill="url(#zt-key-gold)" />

      <!-- Key Shaft (Angled down to bottom-right) -->
      <path d="M16 16 L26 26" stroke="url(#zt-key-gold)" stroke-width="2.6" stroke-linecap="round" />
      <!-- Key Bit / Teeth (Security Notches) -->
      <line x1="22" y1="22" x2="24.5" y2="19.5" stroke="url(#zt-key-gold)" stroke-width="2" stroke-linecap="round" />
      <line x1="25" y1="25" x2="27.5" y2="22.5" stroke="url(#zt-key-gold)" stroke-width="2" stroke-linecap="round" />

      <!-- Sparkle on Key Bow -->
      <circle cx="7" cy="7" r="1" fill="#FFFFFF" />
    </svg>

    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- 9. LIGHTBULB (提示与说明): 暮光落日温润爱迪生灯丝泡                   -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-else-if="normalizedName === 'lightbulb' || normalizedName === 'tip'"
      :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <radialGradient id="zt-bulb-glow" cx="16" cy="12" r="10" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#FEF08A" stop-opacity="0.8" />
          <stop offset="50%" stop-color="#FF9E5E" stop-opacity="0.3" />
          <stop offset="100%" stop-color="#FF9E5E" stop-opacity="0" />
        </radialGradient>
      </defs>

      <!-- Ambient Glow Behind Bulb -->
      <circle cx="16" cy="12" r="10" fill="url(#zt-bulb-glow)" />

      <!-- Glass Bulb Envelope -->
      <path d="M10 12 C10 8.7 12.7 6 16 6 C19.3 6 22 8.7 22 12 C22 14.5 20.5 16.5 19.5 18 L12.5 18 C11.5 16.5 10 14.5 10 12 Z"
        fill="rgba(35, 25, 55, 0.45)" stroke="rgba(255, 240, 230, 0.4)" stroke-width="1.2" />

      <!-- Spiral Incandescent Glowing Filament (Sunset Gold) -->
      <path d="M14 18 L14 13 C14 11.5 15 10.5 16 10.5 C17 10.5 18 11.5 18 13 L18 18"
        stroke="#FEF08A" stroke-width="1.2" stroke-linecap="round" fill="none" />
      <circle cx="16" cy="10.5" r="1" fill="#FFFFFF" />

      <!-- Metallic Base Screw Cap -->
      <rect x="13" y="19" width="6" height="2" rx="0.5" fill="#F59E0B" stroke="#B45309" stroke-width="0.5" />
      <rect x="13.5" y="21.5" width="5" height="1.5" rx="0.5" fill="#D97706" />
      <path d="M14.5 23.5 C14.5 24.5 15.2 25 16 25 C16.8 25 17.5 24.5 17.5 23.5 Z" fill="#78350F" />

      <!-- Radiant Idea Sparks -->
      <line x1="16" y1="2" x2="16" y2="4" stroke="#FDE68A" stroke-width="1.2" stroke-linecap="round" />
      <line x1="7" y1="6" x2="8.5" y2="7.5" stroke="#FDE68A" stroke-width="1" stroke-linecap="round" />
      <line x1="25" y1="6" x2="23.5" y2="7.5" stroke="#FDE68A" stroke-width="1" stroke-linecap="round" />
    </svg>

    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- 10. TELEMETRY (实时网络遥测): 全息脉冲雷达与均衡频谱                 -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-else-if="normalizedName === 'telemetry' || normalizedName === 'chart'"
      :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="zt-bar-gold" x1="0" y1="26" x2="0" y2="6" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#EA580C" />
          <stop offset="50%" stop-color="#FF9E5E" />
          <stop offset="100%" stop-color="#FEF08A" />
        </linearGradient>
      </defs>

      <!-- Glass Frame -->
      <!-- Background box removed -->

      <!-- Equalizer Bars -->
      <rect x="6.5" y="16" width="3.5" height="9" rx="1.5" fill="url(#zt-bar-gold)" opacity="0.7" />
      <rect x="12" y="10" width="3.5" height="15" rx="1.5" fill="url(#zt-bar-gold)" />
      <rect x="17.5" y="6" width="3.5" height="19" rx="1.5" fill="url(#zt-bar-gold)" />
      <rect x="23" y="13" width="3.5" height="12" rx="1.5" fill="url(#zt-bar-gold)" opacity="0.85" />

      <!-- High-Frequency Trend Line Overlay -->
      <path d="M7 14 L13.5 8 L19 5 L25 11" stroke="#38BDF8" stroke-width="1.4" stroke-linecap="round" fill="none" />
      <circle cx="19" cy="5" r="1.5" fill="#FFFFFF" stroke="#38BDF8" stroke-width="1" />
    </svg>

    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- 11. TERMINAL (日志流终端): 全息玻璃终端面板与提示符                   -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-else-if="normalizedName === 'terminal' || normalizedName === 'console'"
      :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <!-- Glass Console Window -->
      <!-- Background box removed -->

      <!-- Window Title Bar Traffic Dots (macOS Style) -->
      <circle cx="6.5" cy="8.5" r="1.3" fill="#EF4444" />
      <circle cx="10.5" cy="8.5" r="1.3" fill="#F59E0B" />
      <circle cx="14.5" cy="8.5" r="1.3" fill="#10B981" />
      <line x1="3" y1="12" x2="29" y2="12" stroke="rgba(255, 240, 230, 0.12)" stroke-width="0.8" />

      <!-- Terminal Command Prompt (>_ ) in Warm Sunset Gold -->
      <path d="M7 16 L11 19 L7 22" stroke="#FF9E5E" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" fill="none" />
      <line x1="13.5" y1="22" x2="18.5" y2="22" stroke="#FDE68A" stroke-width="1.5" stroke-linecap="round" />

      <!-- Telemetry Status Bit -->
      <circle cx="24" cy="19" r="1.2" fill="#34D399" />
    </svg>

    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <!-- Fallback / Default Generic Crystal Icon                             -->
    <!-- ══════════════════════════════════════════════════════════════════════ -->
    <svg v-else :width="computedSize" :height="computedSize" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx="16" cy="16" r="12" fill="rgba(255, 158, 94, 0.15)" stroke="#FF9E5E" stroke-width="1.5" />
      <circle cx="16" cy="16" r="4" fill="#FDE68A" />
    </svg>
  </span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: {
    type: String,
    required: true
  },
  size: {
    type: [Number, String],
    default: 24
  },
  active: {
    type: Boolean,
    default: false
  },
  customClass: {
    type: String,
    default: ''
  },
  title: {
    type: String,
    default: ''
  }
})

const normalizedName = computed(() => (props.name || '').toLowerCase().trim())

const computedSize = computed(() => {
  const s = props.size
  return typeof s === 'number' ? `${s}px` : (s.endsWith('px') || s.endsWith('rem') || s.endsWith('%') ? s : `${s}px`)
})

const defaultTitle = computed(() => {
  const map = {
    overview: '网络中枢 · 动物城中央生态大都会全景',
    routing: '应用分流 · 动物城全生态领航罗盘',
    rules: '安全防护 · ZPD 朱迪警官荣誉金星八角警徽',
    kspeeder: '容器加速 · 动物城穿梭特快磁悬浮列车',
    system: '宿主共生 · 生态调控动力晶核双环齿轮',
    ai: 'AI管家 · 动物城全息智慧晶核与伴生网络领航员',
    copilot: 'AI管家 · 动物城全息智慧晶核与伴生网络领航员',
    diagnostics: '日志诊断 · 朱迪数字录音胡萝卜笔',
    'host-base': '宿主共生底座 · 钛合金共生节点',
    'auth-key': '宿主凭据与鉴权 · ZPD 金色密匙',
    lightbulb: '提示说明 · 暮光温润爱迪生灯丝泡',
    telemetry: '实时网络遥测 · 全息均衡频谱',
    terminal: '实时日志流终端面板'
  }
  return map[normalizedName.value] || 'CubeX 动物城主题图标'
})
</script>

<style scoped>
.zootopia-icon-container {
  display: inline-flex;
  vertical-align: middle;
}
</style>
