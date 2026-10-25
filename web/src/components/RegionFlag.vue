<template>
  <div :class="asBackground ? 'w-full h-full block' : 'inline-flex items-center justify-center shrink-0'">
    <!-- 0. 多国组合通用全息同盟旗 (Universal Twilight Crystal Multi-Country Alliance Ensign) - 优先判定！ -->
    <!-- 彻底告别多旗挤压变形，全域多国组合 (2国、3国、多国) 统一采用尊贵自研同盟极光旗 -->
    <svg v-if="multiCodes.length >= 2 && asBackground" viewBox="0 0 36 24" preserveAspectRatio="none" class="w-full h-full block">
      <defs>
        <!-- Deep Navy Celestial Space Background -->
        <linearGradient id="alliance-bg" x1="0" y1="0" x2="36" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#021124"/>
          <stop offset="35%" stop-color="#042244"/>
          <stop offset="70%" stop-color="#073a6a"/>
          <stop offset="100%" stop-color="#094d8b"/>
        </linearGradient>
        <!-- Twilight Crystal Speed Band Gradient -->
        <linearGradient id="aurora-band" x1="0" y1="0" x2="36" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#7b2cbf"/>
          <stop offset="50%" stop-color="#00d2ff"/>
          <stop offset="100%" stop-color="#00f5d4"/>
        </linearGradient>
        <!-- Mini 5-Point Alliance Star -->
        <g id="alliance-star-canton">
          <polygon points="0,-0.9 0.27,-0.27 0.9,-0.27 0.36,0.09 0.54,0.72 0,0.36 -0.54,0.72 -0.36,0.09 -0.9,-0.27 -0.27,-0.27" fill="#00f5d4"/>
        </g>
      </defs>

      <!-- Background: Deep Navy Void -->
      <rect width="36" height="24" fill="url(#alliance-bg)"/>

      <!-- Diagonal Twilight Crystal Speed Bands (Connecting the Multi-Nation Network) -->
      <path d="M-2 0 L14 0 L38 24 L22 24 Z" fill="url(#aurora-band)" opacity="0.22"/>
      <line x1="14" y1="0" x2="38" y2="24" stroke="#00f5d4" stroke-width="0.6" opacity="0.8"/>
      <line x1="-2" y1="0" x2="22" y2="24" stroke="#38bdf8" stroke-width="0.4" opacity="0.55"/>

      <!-- Top-Left Canton: Triad Alliance Stars (Symbolizing Sovereign Coalition) -->
      <use href="#alliance-star-canton" x="4.2" y="3.8"/>
      <use href="#alliance-star-canton" x="7.4" y="3.8"/>
      <use href="#alliance-star-canton" x="10.6" y="3.8"/>

      <!-- Dual Intertwined Orbital Rings (East-West Planetary Synthesis) -->
      <ellipse cx="18" cy="12" rx="10.5" ry="4.2" stroke="#00f5d4" stroke-width="0.55" fill="none" opacity="0.75" transform="rotate(-22 18 12)" stroke-dasharray="2 0.8"/>
      <ellipse cx="18" cy="12" rx="10.5" ry="4.2" stroke="#38bdf8" stroke-width="0.55" fill="none" opacity="0.75" transform="rotate(22 18 12)" stroke-dasharray="2 0.8"/>

      <!-- Multi-Nation Node Coordinates on the Rings -->
      <circle cx="8.5" cy="8.2" r="0.8" fill="#00f5d4" stroke="#ffffff" stroke-width="0.3"/>
      <circle cx="27.5" cy="15.8" r="0.8" fill="#00f5d4" stroke="#ffffff" stroke-width="0.3"/>
      <circle cx="8.5" cy="15.8" r="0.8" fill="#38bdf8" stroke="#ffffff" stroke-width="0.3"/>
      <circle cx="27.5" cy="8.2" r="0.8" fill="#38bdf8" stroke="#ffffff" stroke-width="0.3"/>

      <!-- Center Radiant Core -->
      <circle cx="18" cy="12" r="4.5" fill="#00f5d4" opacity="0.15"/>
      <circle cx="18" cy="12" r="3.2" fill="#062244" stroke="#00f5d4" stroke-width="0.6"/>

      <!-- 8-Point Sovereign Alliance Compass Star (Majestic & High-Tech) -->
      <polygon points="18,5.2 19.1,10.8 16.9,10.8" fill="#ffffff"/>
      <polygon points="18,18.8 19.1,13.2 16.9,13.2" fill="#ffffff"/>
      <polygon points="24.8,12 19.2,10.9 19.2,13.1" fill="#ffffff"/>
      <polygon points="11.2,12 16.8,10.9 16.8,13.1" fill="#ffffff"/>

      <polygon points="22.8,7.2 19,11 18,10" fill="#00f5d4"/>
      <polygon points="13.2,7.2 18,10 17,11" fill="#00f5d4"/>
      <polygon points="22.8,16.8 18,14 19,13" fill="#00f5d4"/>
      <polygon points="13.2,16.8 17,13 18,14" fill="#00f5d4"/>

      <!-- Center Star Nexus Gem -->
      <circle cx="18" cy="12" r="1.4" fill="#ffffff" stroke="#00d2ff" stroke-width="0.5"/>
      <polygon points="18,10.9 19.1,12 18,13.1 16.9,12" fill="#00f5d4"/>
    </svg>

    <!-- 非背景小图标模式：全能同盟徽章 (Universal Alliance Mini Badge) -->
    <div v-else-if="multiCodes.length >= 2" class="text-sm shrink-0 select-none">
      🌐⚡
    </div>

    <!-- 1. 香港 (香港特别行政区区旗: 法定标准洋紫荆五瓣带五星矢量) -->
    <svg v-else-if="code === 'HK'" viewBox="0 0 900 600" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="900" height="600" fill="#DE2910"/>
      <g id="hk-p">
        <path d="M492.936022 125.19583a27.917245 27.917245 0 0 0-14.901582 41.791893 45.171052 45.171052 0 0 1-20.289932 66.204239 38.650583 38.650583 0 0 0-10.816469 64.313021 68.374981 68.374981 0 0 1-17.067835-93.913589 15.809868 15.809868 0 0 1-1.109109-1.047777 69.880683 69.880683 0 0 0 16.754434 95.793146 90.342104 90.342104 0 0 1 47.430493-173.140933" fill="#ffffff"/>
        <path d="M451.979333 181.099409l-27.564739 12.021638 29.366483 6.476437-19.951229-22.500734 2.915284 29.930515" fill="#DE2910"/>
      </g>
      <use href="#hk-p" transform="rotate(72 450 300)"/>
      <use href="#hk-p" transform="rotate(144 450 300)"/>
      <use href="#hk-p" transform="rotate(216 450 300)"/>
      <use href="#hk-p" transform="rotate(288 450 300)"/>
    </svg>

    <!-- 2. 台湾 -->
    <svg v-else-if="code === 'TW'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#FE0000"/>
      <rect width="18" height="12" fill="#000095"/>
      <g transform="translate(9,6)" fill="#FFFFFF">
        <circle cx="0" cy="0" r="3.2"/>
        <polygon points="0,-5 0.7,-2 -0.7,-2" />
        <polygon points="0,-5 0.7,-2 -0.7,-2" transform="rotate(30)" />
        <polygon points="0,-5 0.7,-2 -0.7,-2" transform="rotate(60)" />
        <polygon points="0,-5 0.7,-2 -0.7,-2" transform="rotate(90)" />
        <polygon points="0,-5 0.7,-2 -0.7,-2" transform="rotate(120)" />
        <polygon points="0,-5 0.7,-2 -0.7,-2" transform="rotate(150)" />
        <polygon points="0,-5 0.7,-2 -0.7,-2" transform="rotate(180)" />
        <polygon points="0,-5 0.7,-2 -0.7,-2" transform="rotate(210)" />
        <polygon points="0,-5 0.7,-2 -0.7,-2" transform="rotate(240)" />
        <polygon points="0,-5 0.7,-2 -0.7,-2" transform="rotate(270)" />
        <polygon points="0,-5 0.7,-2 -0.7,-2" transform="rotate(300)" />
        <polygon points="0,-5 0.7,-2 -0.7,-2" transform="rotate(330)" />
        <circle cx="0" cy="0" r="2.2" fill="#000095"/>
        <circle cx="0" cy="0" r="1.8" fill="#FFFFFF"/>
      </g>
    </svg>

    <!-- 3. 日本 -->
    <svg v-else-if="code === 'JP'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm border border-slate-700/60 shrink-0']">
      <rect width="36" height="24" fill="#FFFFFF"/>
      <circle cx="18" cy="12" r="7" fill="#BC002D"/>
    </svg>

    <!-- 4. 韩国 (大韩民国国旗: 太极旗法定标准矢量) -->
    <svg v-else-if="code === 'KR'" viewBox="-72 -48 144 96" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm border border-slate-700/60 shrink-0']">
      <path fill="#ffffff" d="M-72-48v96H72v-96z"/>
      <g stroke="#000000" stroke-width="4">
        <path transform="rotate(33.69006752598)" d="M-50-12v24m6 0v-24m6 0v24m76 0V1m0-2v-11m6 0v11m0 2v11m6 0V1m0-2v-11"/>
        <path transform="rotate(-33.69006752598)" d="M-50-12v24m6 0V1m0-2v-11m6 0v24m76 0V1m0-2v-11m6 0v24m6 0V1m0-2v-11"/>
      </g>
      <g transform="rotate(33.69006752598)">
        <path fill="#cd2e3a" d="M12 0a18 18 0 11-36 0 24 24 0 1148 0"/>
        <path fill="#0047a0" d="M-24 0a24 24 0 1048 0A12 12 0 100 0a12 12 0 11-24 0"/>
      </g>
    </svg>

    <!-- 5. 新加坡 -->
    <svg v-else-if="code === 'SG'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="12" fill="#ED2939"/>
      <rect y="12" width="36" height="12" fill="#FFFFFF"/>
      <g fill="#FFFFFF" transform="translate(7, 6)">
        <path d="M-2,-4 A 4.2 4.2 0 1 0 -2 4 A 3.3 3.3 0 1 1 -2 -4 Z"/>
        <polygon points="1.5,-2 1.8,-1.2 2.6,-1.2 2,-0.7 2.2,0 1.5,-0.4 0.8,0 1,-0.7 0.4,-1.2 1.2,-1.2" transform="scale(0.8) translate(0.5, -1.8)"/>
        <polygon points="1.5,-2 1.8,-1.2 2.6,-1.2 2,-0.7 2.2,0 1.5,-0.4 0.8,0 1,-0.7 0.4,-1.2 1.2,-1.2" transform="scale(0.8) translate(2.5, 0)"/>
        <polygon points="1.5,-2 1.8,-1.2 2.6,-1.2 2,-0.7 2.2,0 1.5,-0.4 0.8,0 1,-0.7 0.4,-1.2 1.2,-1.2" transform="scale(0.8) translate(1.8, 2.2)"/>
        <polygon points="1.5,-2 1.8,-1.2 2.6,-1.2 2,-0.7 2.2,0 1.5,-0.4 0.8,0 1,-0.7 0.4,-1.2 1.2,-1.2" transform="scale(0.8) translate(-1, 2.2)"/>
        <polygon points="1.5,-2 1.8,-1.2 2.6,-1.2 2,-0.7 2.2,0 1.5,-0.4 0.8,0 1,-0.7 0.4,-1.2 1.2,-1.2" transform="scale(0.8) translate(-1.5, 0)"/>
      </g>
    </svg>

    <!-- 6. 美国 (US - 50 Stars and 13 Stripes 50星法定矢量国旗) -->
    <svg v-else-if="code === 'US'" viewBox="0 0 190 100" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <defs>
        <g id="us-star">
          <polygon points="0,-2.2 0.65,-0.68 2.09,-0.68 0.92,0.18 1.36,1.7 0,0.79 -1.36,1.7 -0.92,0.18 -2.09,-0.68 -0.65,-0.68" fill="#FFFFFF"/>
        </g>
        <g id="us-r6">
          <use href="#us-star" x="6.33"/>
          <use href="#us-star" x="19.0"/>
          <use href="#us-star" x="31.67"/>
          <use href="#us-star" x="44.33"/>
          <use href="#us-star" x="57.0"/>
          <use href="#us-star" x="69.67"/>
        </g>
        <g id="us-r5">
          <use href="#us-star" x="12.67"/>
          <use href="#us-star" x="25.33"/>
          <use href="#us-star" x="38.0"/>
          <use href="#us-star" x="50.67"/>
          <use href="#us-star" x="63.33"/>
        </g>
      </defs>
      <!-- 13 Stripes (7 Red, 6 White) -->
      <rect width="190" height="100" fill="#B22234"/>
      <rect y="7.692" width="190" height="7.692" fill="#FFFFFF"/>
      <rect y="23.077" width="190" height="7.692" fill="#FFFFFF"/>
      <rect y="38.462" width="190" height="7.692" fill="#FFFFFF"/>
      <rect y="53.846" width="190" height="7.692" fill="#FFFFFF"/>
      <rect y="69.231" width="190" height="7.692" fill="#FFFFFF"/>
      <rect y="84.615" width="190" height="7.692" fill="#FFFFFF"/>
      <!-- Blue Canton (Union) -->
      <rect width="76" height="53.846" fill="#3C3B6E"/>
      <!-- 50 Stars in 9 Alternating Rows -->
      <use href="#us-r6" y="5.38"/>
      <use href="#us-r5" y="10.77"/>
      <use href="#us-r6" y="16.15"/>
      <use href="#us-r5" y="21.54"/>
      <use href="#us-r6" y="26.92"/>
      <use href="#us-r5" y="32.31"/>
      <use href="#us-r6" y="37.69"/>
      <use href="#us-r5" y="43.08"/>
      <use href="#us-r6" y="48.46"/>
    </svg>

    <!-- 7. 英国 (GB / UK) -->
    <svg v-else-if="code === 'GB' || code === 'UK'" viewBox="0 0 60 40" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <clipPath id="gb-tp"><path d="M30,20 L60,0 V40 Z M30,20 L0,0 V40 Z M30,20 L0,40 H60 Z M30,20 L0,0 H60 Z"/></clipPath>
      <path d="M0,0 v40 h60 v-40 z" fill="#012169"/>
      <path d="M0,0 L60,40 M60,0 L0,40" stroke="#ffffff" stroke-width="6"/>
      <path d="M0,0 L60,40 M60,0 L0,40" clip-path="url(#gb-tp)" stroke="#C8102E" stroke-width="4"/>
      <path d="M30,0 v40 M0,20 h60" stroke="#ffffff" stroke-width="10"/>
      <path d="M30,0 v40 M0,20 h60" stroke="#C8102E" stroke-width="6"/>
    </svg>

    <!-- 8. 德国 (DE) -->
    <svg v-else-if="code === 'DE'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="8" fill="#000000"/>
      <rect y="8" width="36" height="8" fill="#DD0000"/>
      <rect y="16" width="36" height="8" fill="#FFCE00"/>
    </svg>

    <!-- 9. 法国 (FR) -->
    <svg v-else-if="code === 'FR'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="12" height="24" fill="#002654"/>
      <rect x="12" width="12" height="24" fill="#FFFFFF"/>
      <rect x="24" width="12" height="24" fill="#ED2939"/>
    </svg>

    <!-- 10. 加拿大 (CA) -->
    <svg v-else-if="code === 'CA'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="9" height="24" fill="#FF0000"/>
      <rect x="9" width="18" height="24" fill="#FFFFFF"/>
      <rect x="27" width="9" height="24" fill="#FF0000"/>
      <path d="M18,5 L19.2,8.5 L22.5,8 L20.5,10.5 L23,12.5 L20.5,13.5 L21.5,16.5 L18.8,15 L18.5,19 H17.5 L17.2,15 L14.5,16.5 L15.5,13.5 L13,12.5 L15.5,10.5 L13.5,8 L16.8,8.5 Z" fill="#FF0000"/>
    </svg>

    <!-- 11. 澳大利亚 (AU) -->
    <svg v-else-if="code === 'AU'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#00008B"/>
      <!-- Canton: Union Flag -->
      <rect width="18" height="12" fill="#012169"/>
      <path d="M0,0 L18,12 M18,0 L0,12" stroke="#fff" stroke-width="2"/>
      <path d="M0,0 L18,12 M18,0 L0,12" stroke="#C8102E" stroke-width="1.2"/>
      <path d="M9,0 v12 M0,6 h18" stroke="#fff" stroke-width="3"/>
      <path d="M9,0 v12 M0,6 h18" stroke="#C8102E" stroke-width="1.8"/>
      <!-- Commonwealth 7-pointed Star -->
      <polygon points="0,-2.3 0.42,-0.9 1.8,-1.45 0.94,-0.2 2.25,0.52 0.75,0.6 1.0,2.08 0,0.95 -1.0,2.08 -0.75,0.6 -2.25,0.52 -0.94,-0.2 -1.8,-1.45 -0.42,-0.9" fill="#ffffff" transform="translate(9, 18)"/>
      <!-- Southern Cross: Alpha, Beta, Gamma, Delta (7-pointed) and Epsilon (5-pointed) -->
      <polygon points="0,-1.4 0.25,-0.55 1.1,-0.88 0.57,-0.12 1.36,0.32 0.45,0.36 0.6,1.26 0,0.58 -0.6,1.26 -0.45,0.36 -1.36,0.32 -0.57,-0.12 -1.1,-0.88 -0.25,-0.55" fill="#ffffff" transform="translate(27, 4.8)"/>
      <polygon points="0,-1.4 0.25,-0.55 1.1,-0.88 0.57,-0.12 1.36,0.32 0.45,0.36 0.6,1.26 0,0.58 -0.6,1.26 -0.45,0.36 -1.36,0.32 -0.57,-0.12 -1.1,-0.88 -0.25,-0.55" fill="#ffffff" transform="translate(22.5, 10.5)"/>
      <polygon points="0,-1.4 0.25,-0.55 1.1,-0.88 0.57,-0.12 1.36,0.32 0.45,0.36 0.6,1.26 0,0.58 -0.6,1.26 -0.45,0.36 -1.36,0.32 -0.57,-0.12 -1.1,-0.88 -0.25,-0.55" fill="#ffffff" transform="translate(31.5, 9.2)"/>
      <polygon points="0,-1.4 0.25,-0.55 1.1,-0.88 0.57,-0.12 1.36,0.32 0.45,0.36 0.6,1.26 0,0.58 -0.6,1.26 -0.45,0.36 -1.36,0.32 -0.57,-0.12 -1.1,-0.88 -0.25,-0.55" fill="#ffffff" transform="translate(27, 19.2)"/>
      <polygon points="0,-0.9 0.26,-0.28 0.86,-0.28 0.37,0.08 0.55,0.73 0,0.33 -0.55,0.73 -0.37,0.08 -0.86,-0.28 -0.26,-0.28" fill="#ffffff" transform="translate(28.8, 13.5)"/>
    </svg>

    <!-- 12. 荷兰 (NL) -->
    <svg v-else-if="code === 'NL'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="8" fill="#AE1C28"/>
      <rect y="8" width="36" height="8" fill="#FFFFFF"/>
      <rect y="16" width="36" height="8" fill="#21468B"/>
    </svg>

    <!-- 13. 俄罗斯 (RU) -->
    <svg v-else-if="code === 'RU'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="8" fill="#FFFFFF"/>
      <rect y="8" width="36" height="8" fill="#0039A6"/>
      <rect y="16" width="36" height="8" fill="#D52B1E"/>
    </svg>

    <!-- 14. 印度 (IN) -->
    <svg v-else-if="code === 'IN'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="8" fill="#FF9933"/>
      <rect y="8" width="36" height="8" fill="#FFFFFF"/>
      <rect y="16" width="36" height="8" fill="#138808"/>
      <!-- Ashoka Chakra 24 spokes -->
      <circle cx="18" cy="12" r="3.2" stroke="#000080" stroke-width="0.5" fill="none"/>
      <circle cx="18" cy="12" r="0.65" fill="#000080"/>
      <g stroke="#000080" stroke-width="0.22">
        <line x1="18" y1="8.9" x2="18" y2="15.1"/>
        <line x1="18" y1="8.9" x2="18" y2="15.1" transform="rotate(15 18 12)"/>
        <line x1="18" y1="8.9" x2="18" y2="15.1" transform="rotate(30 18 12)"/>
        <line x1="18" y1="8.9" x2="18" y2="15.1" transform="rotate(45 18 12)"/>
        <line x1="18" y1="8.9" x2="18" y2="15.1" transform="rotate(60 18 12)"/>
        <line x1="18" y1="8.9" x2="18" y2="15.1" transform="rotate(75 18 12)"/>
        <line x1="18" y1="8.9" x2="18" y2="15.1" transform="rotate(90 18 12)"/>
        <line x1="18" y1="8.9" x2="18" y2="15.1" transform="rotate(105 18 12)"/>
        <line x1="18" y1="8.9" x2="18" y2="15.1" transform="rotate(120 18 12)"/>
        <line x1="18" y1="8.9" x2="18" y2="15.1" transform="rotate(135 18 12)"/>
        <line x1="18" y1="8.9" x2="18" y2="15.1" transform="rotate(150 18 12)"/>
        <line x1="18" y1="8.9" x2="18" y2="15.1" transform="rotate(165 18 12)"/>
      </g>
    </svg>

    <!-- 15. 马来西亚 (MY) -->
    <svg v-else-if="code === 'MY'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#CC0000"/>
      <rect y="1.71" width="36" height="1.71" fill="#fff"/><rect y="5.14" width="36" height="1.71" fill="#fff"/>
      <rect y="8.57" width="36" height="1.71" fill="#fff"/><rect y="12.0" width="36" height="1.71" fill="#fff"/>
      <rect y="15.43" width="36" height="1.71" fill="#fff"/><rect y="18.86" width="36" height="1.71" fill="#fff"/>
      <rect width="18" height="13.71" fill="#010066"/>
      <circle cx="7.8" cy="6.85" r="4.2" fill="#FFCC00"/>
      <circle cx="9.1" cy="6.85" r="3.6" fill="#010066"/>
      <!-- 14-pointed Federal Star -->
      <polygon points="0,-2.8 0.3,-1.32 1.21,-2.52 0.84,-1.06 2.19,-1.74 1.2,-0.62 2.73,-0.62 1.33,-0.08 2.73,0.62 1.2,0.62 2.19,1.74 0.84,1.06 1.21,2.52 0.3,1.32 0,2.8 -0.3,1.32 -1.21,2.52 -0.84,1.06 -2.19,1.74 -1.2,0.62 -2.73,0.62 -1.33,-0.08 -2.73,-0.62 -1.2,-0.62 -2.19,-1.74 -0.84,-1.06 -1.21,-2.52 -0.3,-1.32" fill="#FFCC00" transform="translate(12.8, 6.85)"/>
    </svg>

    <!-- 16. 泰国 (TH) -->
    <svg v-else-if="code === 'TH'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="4" fill="#A51931"/>
      <rect y="4" width="36" height="4" fill="#F4F5F8"/>
      <rect y="8" width="36" height="8" fill="#2D2A4A"/>
      <rect y="16" width="36" height="4" fill="#F4F5F8"/>
      <rect y="20" width="36" height="4" fill="#A51931"/>
    </svg>

    <!-- 17. 越南 (VN) -->
    <svg v-else-if="code === 'VN'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#DA251D"/>
      <polygon points="18,5 19.8,10.6 25.6,10.6 20.9,14 22.7,19.6 18,16.2 13.3,19.6 15.1,14 10.4,10.6 16.2,10.6" fill="#FFFF00"/>
    </svg>

    <!-- 18. 菲律宾 (PH) -->
    <svg v-else-if="code === 'PH'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="12" fill="#0038A8"/>
      <rect y="12" width="36" height="12" fill="#CE1126"/>
      <polygon points="0,0 20.8,12 0,24" fill="#FFFFFF"/>
      <!-- Sun with 8 rays -->
      <circle cx="7.2" cy="12" r="1.7" fill="#FCD116"/>
      <g fill="#FCD116">
        <polygon points="7.2,8.4 6.7,10.5 7.7,10.5"/>
        <polygon points="7.2,15.6 6.7,13.5 7.7,13.5"/>
        <polygon points="3.6,12 5.7,11.5 5.7,12.5"/>
        <polygon points="10.8,12 8.7,11.5 8.7,12.5"/>
        <polygon points="4.6,9.4 6.3,10.6 5.6,11.3"/>
        <polygon points="9.8,14.6 8.1,13.4 8.8,12.7"/>
        <polygon points="9.8,9.4 8.1,10.6 8.8,11.3"/>
        <polygon points="4.6,14.6 6.3,13.4 5.6,12.7"/>
      </g>
      <!-- 3 Corner Stars -->
      <polygon points="0,-0.85 0.25,-0.27 0.82,-0.27 0.35,0.08 0.52,0.7 0,0.31 -0.52,0.7 -0.35,0.08 -0.82,-0.27 -0.25,-0.27" fill="#FCD116" transform="translate(2.6, 3.8)"/>
      <polygon points="0,-0.85 0.25,-0.27 0.82,-0.27 0.35,0.08 0.52,0.7 0,0.31 -0.52,0.7 -0.35,0.08 -0.82,-0.27 -0.25,-0.27" fill="#FCD116" transform="translate(2.6, 20.2)"/>
      <polygon points="0,-0.85 0.25,-0.27 0.82,-0.27 0.35,0.08 0.52,0.7 0,0.31 -0.52,0.7 -0.35,0.08 -0.82,-0.27 -0.25,-0.27" fill="#FCD116" transform="translate(16.5, 12)"/>
    </svg>

    <!-- 19. 印尼 (ID) -->
    <svg v-else-if="code === 'ID'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="12" fill="#FF0000"/>
      <rect y="12" width="36" height="12" fill="#FFFFFF"/>
    </svg>

    <!-- 20. 土耳其 (TR) -->
    <svg v-else-if="code === 'TR'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#E30A17"/>
      <circle cx="15" cy="12" r="6" fill="#FFFFFF"/>
      <circle cx="16.5" cy="12" r="4.8" fill="#E30A17"/>
      <polygon points="21,12 24.5,13.2 23.2,9.8 24.5,6.4 21,7.6 17.5,6.4 18.8,9.8 17.5,13.2" fill="#FFFFFF" transform="scale(0.5) translate(22, 12)"/>
    </svg>

    <!-- 21. 阿根廷 (AR) -->
    <svg v-else-if="code === 'AR'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="8" fill="#74ACDF"/>
      <rect y="8" width="36" height="8" fill="#FFFFFF"/>
      <rect y="16" width="36" height="8" fill="#74ACDF"/>
      <!-- Sol de Mayo -->
      <polygon points="18,8.8 18.5,11.2 21.2,12 18.5,12.8 18,15.2 17.5,12.8 14.8,12 17.5,11.2" fill="#F6B40E"/>
      <polygon points="18,8.8 18.5,11.2 21.2,12 18.5,12.8 18,15.2 17.5,12.8 14.8,12 17.5,11.2" fill="#F6B40E" transform="rotate(45 18 12)"/>
      <polygon points="18,9.3 18.4,11.3 20.7,12 18.4,12.7 18,14.7 17.6,12.7 15.3,12 17.6,11.3" fill="#D99700" transform="rotate(22.5 18 12)"/>
      <polygon points="18,9.3 18.4,11.3 20.7,12 18.4,12.7 18,14.7 17.6,12.7 15.3,12 17.6,11.3" fill="#D99700" transform="rotate(67.5 18 12)"/>
      <circle cx="18" cy="12" r="1.6" fill="#F6B40E" stroke="#855B14" stroke-width="0.25"/>
    </svg>

    <!-- 22. 巴西 (BR) -->
    <svg v-else-if="code === 'BR'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#009C3B"/>
      <polygon points="18,2.2 33.8,12 18,21.8 2.2,12" fill="#FFDF00"/>
      <circle cx="18" cy="12" r="4.6" fill="#002776"/>
      <!-- White equatorial arch -->
      <path d="M13.6,12.8 A 5.5,4.5 0 0,1 22.4,10.6 A 6.2,5.2 0 0,0 13.6,12.8 Z" fill="#FFFFFF"/>
      <!-- Constellation Stars -->
      <circle cx="18" cy="13.6" r="0.26" fill="#ffffff"/>
      <circle cx="18" cy="15.2" r="0.26" fill="#ffffff"/>
      <circle cx="17.2" cy="14.4" r="0.26" fill="#ffffff"/>
      <circle cx="18.8" cy="14.2" r="0.26" fill="#ffffff"/>
      <circle cx="18.3" cy="14.7" r="0.18" fill="#ffffff"/>
      <circle cx="15.8" cy="13.2" r="0.22" fill="#ffffff"/>
      <circle cx="20.4" cy="13.8" r="0.22" fill="#ffffff"/>
    </svg>

    <!-- 23. 瑞士 (CH) -->
    <svg v-else-if="code === 'CH'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#D52B1E"/>
      <rect x="15" y="6" width="6" height="12" fill="#FFFFFF"/>
      <rect x="10" y="9" width="16" height="6" fill="#FFFFFF"/>
    </svg>

    <!-- 24. 瑞典 (SE) -->
    <svg v-else-if="code === 'SE'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#006AA7"/>
      <rect x="11" width="4.5" height="24" fill="#FECC00"/>
      <rect y="9.75" width="36" height="4.5" fill="#FECC00"/>
    </svg>

    <!-- 25. 意大利 (IT) -->
    <svg v-else-if="code === 'IT'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="12" height="24" fill="#009246"/>
      <rect x="12" width="12" height="24" fill="#FFFFFF"/>
      <rect x="24" width="12" height="24" fill="#CE2B37"/>
    </svg>

    <!-- 26. 西班牙 (ES) -->
    <svg v-else-if="code === 'ES'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="6" fill="#AA151B"/>
      <rect y="6" width="36" height="12" fill="#F1BF00"/>
      <rect y="18" width="36" height="6" fill="#AA151B"/>
    </svg>

    <!-- 27. 澳门 (MO) -->
    <svg v-else-if="code === 'MO'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#00785E"/>
      <!-- 5 Golden Stars in Arc -->
      <polygon points="18,4.2 18.3,5.1 19.2,5.1 18.5,5.6 18.7,6.5 18,6.0 17.3,6.5 17.5,5.6 16.8,5.1 17.7,5.1" fill="#FFDE00"/>
      <polygon points="0,-0.6 0.18,-0.18 0.6,-0.18 0.25,0.06 0.38,0.48 0,0.22 -0.38,0.48 -0.25,0.06 -0.6,-0.18 -0.18,-0.18" fill="#FFDE00" transform="translate(15.2, 5.8)"/>
      <polygon points="0,-0.6 0.18,-0.18 0.6,-0.18 0.25,0.06 0.38,0.48 0,0.22 -0.38,0.48 -0.25,0.06 -0.6,-0.18 -0.18,-0.18" fill="#FFDE00" transform="translate(20.8, 5.8)"/>
      <polygon points="0,-0.5 0.15,-0.15 0.5,-0.15 0.21,0.05 0.32,0.4 0,0.18 -0.32,0.4 -0.21,0.05 -0.5,-0.15 -0.15,-0.15" fill="#FFDE00" transform="translate(12.8, 7.2)"/>
      <polygon points="0,-0.5 0.15,-0.15 0.5,-0.15 0.21,0.05 0.32,0.4 0,0.18 -0.32,0.4 -0.21,0.05 -0.5,-0.15 -0.15,-0.15" fill="#FFDE00" transform="translate(23.2, 7.2)"/>
      <!-- Lotus Petals -->
      <path d="M18,9.2 C19.2,11.2 19.4,13.2 18,14.6 C16.6,13.2 16.8,11.2 18,9.2 Z" fill="#FFFFFF"/>
      <path d="M15.2,11.2 C16.8,11.8 17.2,13.5 16.2,14.6 C14.6,14.2 14.2,12.5 15.2,11.2 Z" fill="#FFFFFF"/>
      <path d="M20.8,11.2 C19.2,11.8 18.8,13.5 19.8,14.6 C21.4,14.2 21.8,12.5 20.8,11.2 Z" fill="#FFFFFF"/>
      <!-- Macau-Taipa Bridge & Waves -->
      <path d="M12.5,15.6 Q18,14.2 23.5,15.6" stroke="#FFFFFF" stroke-width="0.6" fill="none"/>
      <path d="M12,17.2 Q15,16.5 18,17.2 Q21,17.9 24,17.2" stroke="#FFFFFF" stroke-width="0.5" fill="none"/>
      <path d="M13.5,18.8 Q16,18.2 18,18.8 Q20,19.4 22.5,18.8" stroke="#FFFFFF" stroke-width="0.4" fill="none"/>
    </svg>

    <!-- 28. 中国 (CN) -->
    <svg v-else-if="code === 'CN'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#DE2910"/>
      <!-- Big Star centered at (6, 6) -->
      <polygon points="6,2.4 6.84,4.98 9.55,4.98 7.36,6.58 8.19,9.12 6,7.54 3.81,9.12 4.64,6.58 2.45,4.98 5.16,4.98" fill="#FFDE00"/>
      <!-- 4 Small Stars pointing at (6, 6) -->
      <polygon points="0,-1.2 0.28,-0.37 1.14,-0.37 0.44,0.14 0.71,0.97 0,0.46 -0.71,0.97 -0.44,0.14 -1.14,-0.37 -0.28,-0.37" fill="#FFDE00" transform="translate(12, 2.4) rotate(-31)"/>
      <polygon points="0,-1.2 0.28,-0.37 1.14,-0.37 0.44,0.14 0.71,0.97 0,0.46 -0.71,0.97 -0.44,0.14 -1.14,-0.37 -0.28,-0.37" fill="#FFDE00" transform="translate(14.4, 4.8) rotate(-8.1)"/>
      <polygon points="0,-1.2 0.28,-0.37 1.14,-0.37 0.44,0.14 0.71,0.97 0,0.46 -0.71,0.97 -0.44,0.14 -1.14,-0.37 -0.28,-0.37" fill="#FFDE00" transform="translate(14.4, 8.4) rotate(15.9)"/>
      <polygon points="0,-1.2 0.28,-0.37 1.14,-0.37 0.44,0.14 0.71,0.97 0,0.46 -0.71,0.97 -0.44,0.14 -1.14,-0.37 -0.28,-0.37" fill="#FFDE00" transform="translate(12, 10.8) rotate(38.7)"/>
    </svg>

    <!-- 29. 阿联酋 (AE) -->
    <svg v-else-if="code === 'AE'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="8" fill="#00732F"/>
      <rect y="8" width="36" height="8" fill="#FFFFFF"/>
      <rect y="16" width="36" height="8" fill="#000000"/>
      <rect width="9" height="24" fill="#FF0000"/>
    </svg>

    <!-- 30. 爱尔兰 (IE) -->
    <svg v-else-if="code === 'IE'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="12" height="24" fill="#169B62"/>
      <rect x="12" width="12" height="24" fill="#FFFFFF"/>
      <rect x="24" width="12" height="24" fill="#FF883E"/>
    </svg>

    <!-- 31. 乌克兰 (UA) -->
    <svg v-else-if="code === 'UA'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="12" fill="#0057B7"/>
      <rect y="12" width="36" height="12" fill="#FFDD00"/>
    </svg>

    <!-- 32. 自建节点主权旗 (Sovereign Cyber HomeLab Ensign) -->
    <svg v-else-if="code === 'CUSTOM' && asBackground" viewBox="0 0 36 24" preserveAspectRatio="none" class="w-full h-full block">
      <defs>
        <linearGradient id="custom-bg" x1="0" y1="0" x2="36" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#180b2c"/>
          <stop offset="60%" stop-color="#3b0764"/>
          <stop offset="100%" stop-color="#581c87"/>
        </linearGradient>
      </defs>
      <rect width="36" height="24" fill="url(#custom-bg)"/>
      <rect x="0" y="9.5" width="36" height="5" fill="#0d041a"/>
      <rect x="9.5" y="0" width="5" height="24" fill="#0d041a"/>
      <rect x="0" y="10.8" width="36" height="2.4" fill="#a855f7" opacity="0.55"/>
      <rect x="10.8" y="0" width="2.4" height="24" fill="#a855f7" opacity="0.55"/>
      <g transform="translate(5, 5)">
        <circle cx="0" cy="0" r="1.3" fill="#00f5d4"/>
        <circle cx="3.5" cy="0" r="1.3" fill="#38bdf8"/>
      </g>
      <g transform="translate(22.5, 12)">
        <polygon points="0,-5.8 5.2,-2.9 0,0 -5.2,-2.9" fill="#00f5d4" opacity="0.95"/>
        <polygon points="-5.2,-2.9 0,0 0,5.8 -5.2,2.9" fill="#7b2cbf"/>
        <polygon points="0,0 5.2,-2.9 5.2,2.9 0,5.8" fill="#38bdf8"/>
        <circle cx="0" cy="0" r="1.2" fill="#ffffff"/>
      </g>
    </svg>
    <div v-else-if="code === 'CUSTOM'" class="text-sm shrink-0">
      🏠
    </div>

    <!-- 33. 全球公海 / 其它 (United Earth & Maritime Ensign) -->
    <svg v-else-if="code === 'OTHERS' && asBackground" viewBox="0 0 36 24" preserveAspectRatio="none" class="w-full h-full block">
      <defs>
        <linearGradient id="others-bg" x1="0" y1="0" x2="36" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#021c38"/>
          <stop offset="50%" stop-color="#0284c7"/>
          <stop offset="100%" stop-color="#082f49"/>
        </linearGradient>
      </defs>
      <rect width="36" height="24" fill="url(#others-bg)"/>
      <rect x="0" y="2.5" width="36" height="1.8" fill="#fbbf24" opacity="0.8"/>
      <rect x="0" y="19.7" width="36" height="1.8" fill="#fbbf24" opacity="0.8"/>
      <g transform="translate(6, 12)" fill="#ffffff" opacity="0.8">
        <polygon points="0,-2 0.5,-0.5 2,0 0.5,0.5 0,2 -0.5,0.5 -2,0 -0.5,-0.5"/>
      </g>
      <g transform="translate(30, 12)" fill="#ffffff" opacity="0.8">
        <polygon points="0,-2 0.5,-0.5 2,0 0.5,0.5 0,2 -0.5,0.5 -2,0 -0.5,-0.5"/>
      </g>
      <g transform="translate(18, 12)">
        <circle cx="0" cy="0" r="6.8" fill="#0369a1" stroke="#ffffff" stroke-width="0.8"/>
        <ellipse cx="0" cy="0" rx="3.4" ry="6.8" fill="none" stroke="#ffffff" stroke-width="0.6" opacity="0.85"/>
        <line x1="-6.8" y1="0" x2="6.8" y2="0" stroke="#ffffff" stroke-width="0.6" opacity="0.85"/>
        <ellipse cx="0" cy="0" rx="6.8" ry="3.4" fill="none" stroke="#38bdf8" stroke-width="0.5" opacity="0.65"/>
        <ellipse cx="0" cy="0" rx="9.8" ry="3.6" fill="none" stroke="#fbbf24" stroke-width="0.75" stroke-dasharray="1.5 1" transform="rotate(-25)"/>
      </g>
    </svg>
    <div v-else-if="code === 'OTHERS'" class="text-sm shrink-0">
      🌐
    </div>

    <!-- 34. 机场订阅专属旗 (Supersonic Cyber Airway Ensign) -->
    <svg v-else-if="(code === 'SUB' || type === 'sub') && asBackground" viewBox="0 0 36 24" preserveAspectRatio="none" class="w-full h-full block">
      <defs>
        <linearGradient id="sub-bg" x1="0" y1="0" x2="36" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#0b1b36"/>
          <stop offset="50%" stop-color="#0284c7"/>
          <stop offset="100%" stop-color="#075985"/>
        </linearGradient>
        <linearGradient id="sub-gold" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#fef08a"/>
          <stop offset="100%" stop-color="#ca8a04"/>
        </linearGradient>
      </defs>
      <rect width="36" height="24" fill="url(#sub-bg)"/>
      <path d="M0 24 L13 0 L17 0 L4 24 Z" fill="#ffffff" opacity="0.3"/>
      <path d="M5 24 L18 0 L22 0 L9 24 Z" fill="#38bdf8" opacity="0.45"/>
      <g transform="translate(6.5, 6)" fill="#ffffff" opacity="0.9">
        <polygon points="0,-3.5 0.7,-0.7 3.5,0 0.7,0.7 0,3.5 -0.7,0.7 -3.5,0 -0.7,-0.7"/>
        <circle cx="0" cy="0" r="0.7" fill="#facc15"/>
      </g>
      <g transform="translate(22, 12)">
        <ellipse cx="-4.5" cy="0" rx="3.5" ry="1.4" fill="#00f5d4" opacity="0.7"/>
        <path d="M6.5 0 L-6 -5 L-3 -1.5 L-6.5 0 L-3 1.5 L-6 5 Z" fill="#ffffff"/>
        <path d="M6.5 0 L-2 -1.5 L-6.5 0 L-2 1.5 Z" fill="url(#sub-gold)"/>
      </g>
    </svg>
    <div v-else-if="code === 'SUB' || type === 'sub'" class="text-sm shrink-0">
      ✈️
    </div>

    <!-- 35. 拒绝出站 / 规则拦截旗 (Sentinel Citadel & Aegis Defense Flag) -->
    <svg v-else-if="code === 'REJECT' && asBackground" viewBox="0 0 36 24" preserveAspectRatio="none" class="w-full h-full block">
      <defs>
        <linearGradient id="reject-bg" x1="0" y1="0" x2="36" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#4c0519"/>
          <stop offset="50%" stop-color="#9f1239"/>
          <stop offset="100%" stop-color="#2a030d"/>
        </linearGradient>
        <linearGradient id="reject-shield" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stop-color="#fb7185"/>
          <stop offset="100%" stop-color="#e11d48"/>
        </linearGradient>
      </defs>
      <rect width="36" height="24" fill="url(#reject-bg)"/>
      <path d="M0 0 L15 12 L0 24 Z" fill="#180208" opacity="0.55"/>
      <line x1="0" y1="0" x2="15" y2="12" stroke="#f43f5e" stroke-width="0.9" opacity="0.7"/>
      <line x1="0" y1="24" x2="15" y2="12" stroke="#f43f5e" stroke-width="0.9" opacity="0.7"/>
      <g transform="translate(22, 12)">
        <path d="M0 -6.5 L6.2 -3.3 V2 C6.2 5.8 0 8.2 0 8.2 C0 8.2 -6.2 5.8 -6.2 2 V-3.3 Z" fill="url(#reject-shield)" stroke="#ffffff" stroke-width="0.8"/>
        <line x1="-3.2" y1="-3.2" x2="3.2" y2="3.2" stroke="#ffffff" stroke-width="1.3" stroke-linecap="round"/>
        <line x1="3.2" y1="-3.2" x2="-3.2" y2="3.2" stroke="#ffffff" stroke-width="1.3" stroke-linecap="round"/>
      </g>
    </svg>
    <div v-else-if="code === 'REJECT'" class="text-sm shrink-0">
      🛑
    </div>

    <!-- 35. 其它通用国旗/区旗后备 (带大号 Emoji 全息居中展示) -->
    <div v-else-if="asBackground" class="w-full h-full bg-[#0a1626] flex items-center justify-center relative overflow-hidden">
      <div class="absolute inset-0 bg-gradient-to-br from-cyan-950/40 via-slate-900/60 to-slate-950/90"></div>
      <span class="text-4xl opacity-35 filter drop-shadow-[0_0_12px_rgba(0,210,255,0.4)] select-none relative z-10">{{ flag || '🌍' }}</span>
    </div>
    <div v-else class="text-sm shrink-0">
      {{ flag || '🏷️' }}
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  id: { type: String, default: '' },
  code: { type: String, default: '' },
  countryCode: { type: String, default: '' },
  name: { type: String, default: '' },
  type: { type: String, default: 'region' },
  flag: { type: String, default: '' },
  sizeClass: { type: String, default: 'w-7 h-4.5' },
  asBackground: { type: Boolean, default: false }
})

const CHINESE_NAME_MAP = {
  '俄罗斯': 'RU', '俄国': 'RU', '英国': 'GB', '德国': 'DE', '法国': 'FR', '加拿大': 'CA',
  '澳大利亚': 'AU', '澳洲': 'AU', '荷兰': 'NL', '印度': 'IN', '马来西亚': 'MY',
  '大马': 'MY', '泰国': 'TH', '越南': 'VN', '菲律宾': 'PH', '印尼': 'ID',
  '印度尼西亚': 'ID', '土耳其': 'TR', '阿根廷': 'AR', '巴西': 'BR', '瑞士': 'CH',
  '瑞典': 'SE', '意大利': 'IT', '西班牙': 'ES', '澳门': 'MO', '中国': 'CN',
  '阿联酋': 'AE', '迪拜': 'AE', '爱尔兰': 'IE', '乌克兰': 'UA', '香港': 'HK',
  '台湾': 'TW', '日本': 'JP', '韩国': 'KR', '新加坡': 'SG', '美国': 'US'
}

function parseCountryCode(val) {
  if (!val) return ''
  val = String(val).trim()
  if (/^[A-Za-z]{2}$/.test(val)) return val.toUpperCase()
  const codePoints = [...val].map(c => c.codePointAt(0))
  if (codePoints.length === 2 && codePoints[0] >= 0x1F1E6 && codePoints[0] <= 0x1F1FF && codePoints[1] >= 0x1F1E6 && codePoints[1] <= 0x1F1FF) {
    return String.fromCharCode(codePoints[0] - 0x1F1E6 + 65, codePoints[1] - 0x1F1E6 + 65)
  }
  for (const [k, code] of Object.entries(CHINESE_NAME_MAP)) {
    if (val.includes(k)) return code
  }
  return ''
}

const code = computed(() => {
  if (props.code) {
    const uc = props.code.toUpperCase().trim()
    if (uc === 'REJECT' || uc === 'BLOCK') return 'REJECT'
    if (uc === 'CUSTOM') return 'CUSTOM'
    if (uc === 'OTHERS' || uc === 'OTHER') return 'OTHERS'
    if (uc === 'SUB') return 'SUB'
    let c = parseCountryCode(uc)
    if (c) return c
  }
  if (props.id === 'REJECT' || props.name === '拒绝' || props.name === 'REJECT') return 'REJECT'
  if (props.id === 'CUSTOM' || props.countryCode === 'CUSTOM') return 'CUSTOM'
  if (props.id === 'OTHERS' || props.countryCode === 'OTHERS') return 'OTHERS'
  if (props.type === 'sub' || props.flag === '✈️') return 'SUB'

  let c = parseCountryCode(props.code)
  if (c) return c
  c = parseCountryCode(props.countryCode)
  if (c) return c
  c = parseCountryCode(props.flag)
  if (c) return c
  c = parseCountryCode(props.name)
  if (c) return c
  c = parseCountryCode(props.id)
  if (c) return c

  // Custom airport subscription groups or custom node groups
  if (props.id && props.id.toUpperCase().startsWith('CUSTOM')) {
    if (props.type === 'sub' || (props.name && (props.name.includes('专属') || props.name.includes('机场') || props.name.includes('Sub') || props.name.includes('HHost') || props.name.includes('Mtice')))) {
      return 'SUB'
    }
    return 'CUSTOM'
  }
  if (props.id && props.id.toUpperCase().startsWith('OTHER')) return 'OTHERS'
  if (props.name && (props.name.includes('专属') || props.name.includes('机场') || props.name.includes('HHost') || props.name.includes('Mtice'))) {
    return 'SUB'
  }
  return (props.countryCode || props.id || props.code || '').toUpperCase().trim()
})

const multiCodes = computed(() => {
  // If props.code is a single 2-letter country code, never treat as multi (terminates recursion)
  if (props.code && /^[A-Za-z]{2}$/.test(props.code.trim())) {
    return []
  }

  // 1. Check comma / dash / slash delimited 2-letter ISO codes in countryCode or code
  const parseCodeList = (raw) => {
    if (!raw) return []
    const parts = raw.split(/[/,_ |、·•\-]+/).map(c => c.trim().toUpperCase()).filter(c => /^[A-Z]{2}$/.test(c))
    return parts.length > 1 ? parts : []
  }
  const fromCountryCode = parseCodeList(props.countryCode)
  if (fromCountryCode.length > 1) return fromCountryCode
  const fromCode = parseCodeList(props.code)
  if (fromCode.length > 1) return fromCode

  // 2. Parse from group name
  const name = props.name || ''
  if (!name || name.length < 2) return []

  const positions = [] // { idx: number, len: number, code: string }

  // 2.1 Pass 1: Match 2-letter ISO country codes in name (e.g. HK-TW-JP, [US/GB])
  const isoRegex = /(?:^|[^A-Za-z])(HK|TW|JP|KR|SG|US|GB|UK|DE|FR|CA|AU|NL|RU|IN|MY|TH|VN|PH|ID|TR|AR|BR|CH|SE|IT|ES|MO|CN|AE|IE|UA)(?=$|[^A-Za-z])/gi
  let m
  while ((m = isoRegex.exec(name)) !== null) {
    let c = m[1].toUpperCase()
    if (c === 'UK') c = 'GB'
    const matchIdx = m.index + (m[0].length - m[1].length)
    positions.push({ idx: matchIdx, len: c.length, code: c })
  }

  // 2.2 Pass 2: Full Chinese names from CHINESE_NAME_MAP (sorted by length descending)
  const sortedNames = Object.entries(CHINESE_NAME_MAP).sort((a, b) => b[0].length - a[0].length)
  for (const [k, code] of sortedNames) {
    let startIdx = 0
    while ((startIdx = name.indexOf(k, startIdx)) !== -1) {
      const endIdx = startIdx + k.length
      const overlaps = positions.some(p => Math.max(p.idx, startIdx) < Math.min(p.idx + p.len, endIdx))
      if (!overlaps) {
        positions.push({ idx: startIdx, len: k.length, code })
      }
      startIdx = endIdx
    }
  }

  // 2.3 Pass 3: Chinese single-character abbreviations
  // Smart disambiguation for '澳':
  // '港澳', '港澳台', '澳台' -> MO (Macau)
  // '澳新', '澳洲', '澳大利亚' -> AU (Australia)
  // Default to MO if group also mentions 港 or 台, otherwise AU
  let aoCode = 'AU'
  if (name.includes('港澳') || name.includes('澳台') || (name.includes('港') && !name.includes('新') && !name.includes('澳洲'))) {
    aoCode = 'MO'
  } else if (name.includes('港') || name.includes('台')) {
    aoCode = 'MO'
  }

  const singleCharAbbrs = [
    { k: '港', code: 'HK' }, { k: '台', code: 'TW' }, { k: '日', code: 'JP' },
    { k: '韩', code: 'KR' }, { k: '新', code: 'SG' }, { k: '马', code: 'MY' },
    { k: '泰', code: 'TH' }, { k: '美', code: 'US' }, { k: '加', code: 'CA' },
    { k: '英', code: 'GB' }, { k: '法', code: 'FR' }, { k: '德', code: 'DE' },
    { k: '俄', code: 'RU' }, { k: '越', code: 'VN' }, { k: '菲', code: 'PH' },
    { k: '土', code: 'TR' }, { k: '巴', code: 'BR' }, { k: '意', code: 'IT' },
    { k: '西', code: 'ES' }, { k: '荷', code: 'NL' }, { k: '中', code: 'CN' },
    { k: '华', code: 'CN' }, { k: '澳', code: aoCode }
  ]

  for (const item of singleCharAbbrs) {
    let startIdx = 0
    while ((startIdx = name.indexOf(item.k, startIdx)) !== -1) {
      const endIdx = startIdx + 1
      // CRITICAL: Strictly avoid matching if this character index is ALREADY inside a longer matched country name!
      // (e.g. '加' inside '新加坡' or '大' inside '加拿大' must NEVER create a spurious CA / AU match!)
      const overlaps = positions.some(p => Math.max(p.idx, startIdx) < Math.min(p.idx + p.len, endIdx))
      if (!overlaps) {
        positions.push({ idx: startIdx, len: 1, code: item.code })
      }
      startIdx = endIdx
    }
  }

  // 2.4 Sort strictly by physical appearance order in name
  positions.sort((a, b) => a.idx - b.idx)
  const found = []
  const seen = new Set()
  for (const p of positions) {
    if (!seen.has(p.code)) {
      seen.add(p.code)
      found.push(p.code)
    }
  }

  return found.length > 1 ? found : []
})
</script>
