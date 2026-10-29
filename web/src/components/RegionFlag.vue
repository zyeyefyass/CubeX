<template>
  <div :class="asBackground ? 'w-full h-full block' : 'inline-flex items-center justify-center shrink-0'">
    <!-- 0. 多国组合全息同盟卡 (Zootopia Metropolis & Climate Wall Nexus · 疯狂动物城极光生态之冠) -->
    <svg v-if="multiCodes.length >= 2 && asBackground" viewBox="0 0 36 24" preserveAspectRatio="none" class="w-full h-full block">
      <defs>
        <!-- Cinematic Twilight Amethyst to Deep Navy Sky -->
        <linearGradient id="ztp-alliance-bg" x1="0" y1="0" x2="36" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#060b18"/>
          <stop offset="35%" stop-color="#141a38"/>
          <stop offset="70%" stop-color="#2d124d"/>
          <stop offset="100%" stop-color="#0a2a44"/>
        </linearGradient>
        <!-- Zootopia Aurora Canopy (Violet to Cyan to Emerald) -->
        <linearGradient id="ztp-aurora" x1="0" y1="0" x2="36" y2="16" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#7b2cbf" stop-opacity="0.85"/>
          <stop offset="45%" stop-color="#00d2ff" stop-opacity="0.75"/>
          <stop offset="100%" stop-color="#00f5d4" stop-opacity="0.6"/>
        </linearGradient>
        <!-- Golden Sunlight Rim -->
        <linearGradient id="ztp-gold" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#fef08a"/>
          <stop offset="100%" stop-color="#f59e0b"/>
        </linearGradient>
      </defs>

      <!-- Background: Deep Twilight Void -->
      <rect width="36" height="24" fill="url(#ztp-alliance-bg)"/>

      <!-- Aurora Curtain Waves in Sky -->
      <path d="M-2 2 C8 8, 18 0, 38 7 L38 0 L-2 0 Z" fill="url(#ztp-aurora)" opacity="0.35"/>
      <path d="M-2 5 C10 1, 22 9, 38 3 L38 0 L-2 0 Z" fill="#00d2ff" opacity="0.2"/>

      <!-- Constellation Coalition Stars in Deep Sky -->
      <circle cx="4.5" cy="3.5" r="0.45" fill="#ffffff" opacity="0.9"/>
      <circle cx="8" cy="2.2" r="0.35" fill="#fef08a" opacity="0.85"/>
      <circle cx="12" cy="4" r="0.4" fill="#ffffff" opacity="0.9"/>
      <circle cx="28" cy="2.8" r="0.35" fill="#00f5d4" opacity="0.85"/>
      <circle cx="32" cy="4.2" r="0.45" fill="#ffffff" opacity="0.9"/>

      <!-- Background Silhouette: Giant Climate Wall Pillars & Mountain Range -->
      <path d="M0 24 L0 16 L3 16 L3 14 L5 14 L5 17 L9 17 L9 13 L11 13 L11 18 L15 18 L15 12 L17 12 L17 18 L21 18 L21 11 L23 11 L23 18 L27 18 L27 14 L29 14 L29 18 L36 18 L36 24 Z" fill="#091224" opacity="0.85"/>

      <!-- Iconic Palm-Spire City Nexus Skyline Silhouette -->
      <!-- Palm Spire Tower in Center -->
      <polygon points="18,3 19.4,17 16.6,17" fill="#034870" opacity="0.95"/>
      <!-- Glowing Spire Spikes / Crystal Palm Crown -->
      <path d="M18 3 Q13 7 11 10 Q14 8 18 6 Q22 8 25 10 Q23 7 18 3 Z" fill="url(#ztp-gold)" opacity="0.9"/>
      <!-- Spire Beacon -->
      <circle cx="18" cy="3" r="0.75" fill="#ffffff"/>
      <circle cx="18" cy="3" r="1.6" fill="#fef08a" opacity="0.35"/>

      <!-- Twin Hyperloop Transit Arcs (Connecting the Biomes) -->
      <ellipse cx="18" cy="14" rx="14" ry="5.5" stroke="#00f5d4" stroke-width="0.55" fill="none" opacity="0.7" stroke-dasharray="2 0.8" transform="rotate(-12 18 14)"/>
      <ellipse cx="18" cy="14" rx="14" ry="5.5" stroke="#38bdf8" stroke-width="0.55" fill="none" opacity="0.7" stroke-dasharray="2 0.8" transform="rotate(12 18 14)"/>

      <!-- Transit Pods on the Arcs -->
      <circle cx="7" cy="11.5" r="0.75" fill="#00f5d4" stroke="#ffffff" stroke-width="0.25"/>
      <circle cx="29" cy="16.5" r="0.75" fill="#00f5d4" stroke="#ffffff" stroke-width="0.25"/>
      <circle cx="7" cy="16.5" r="0.75" fill="#fef08a" stroke="#ffffff" stroke-width="0.25"/>
      <circle cx="29" cy="11.5" r="0.75" fill="#fef08a" stroke="#ffffff" stroke-width="0.25"/>

      <!-- Center Nexus Radiant Energy Core -->
      <circle cx="18" cy="14" r="3.2" fill="#02213a" stroke="#00f5d4" stroke-width="0.6"/>
      <circle cx="18" cy="14" r="2.2" fill="url(#ztp-gold)" opacity="0.9"/>
      <polygon points="18,10.8 19,13.2 21.2,14 19,14.8 18,17.2 17,14.8 14.8,14 17,13.2" fill="#ffffff"/>
    </svg>

    <!-- 非背景小图标模式：全能同盟徽章 (Universal Alliance Mini Badge) -->
    <div v-else-if="multiCodes.length >= 2" class="text-sm shrink-0 select-none">
      🌐⚡
    </div>

    <!-- 1. 香港 (香港特别行政区区旗: 法定标准洋紫荆五瓣带五星矢量) -->
    <svg v-else-if="code === 'HK'" viewBox="0 0 900 600" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="900" height="600" fill="#DE2910"/>
      <g>
        <g id="hk-p">
          <path d="M492.936022 125.19583a27.917245 27.917245 0 0 0-14.901582 41.791893 45.171052 45.171052 0 0 1-20.289932 66.204239 38.650583 38.650583 0 0 0-10.816469 64.313021 68.374981 68.374981 0 0 1-17.067835-93.913589 15.809868 15.809868 0 0 1-1.109109-1.047777 69.880683 69.880683 0 0 0 16.754434 95.793146 90.342104 90.342104 0 0 1 47.430493-173.140933" fill="#ffffff"/>
          <path d="M451.979333 181.099409l-27.564739 12.021638 29.366483 6.476437-19.951229-22.500734 2.915284 29.930515" fill="#DE2910"/>
        </g>
        <use href="#hk-p" transform="rotate(72 450 300)"/>
        <use href="#hk-p" transform="rotate(144 450 300)"/>
        <use href="#hk-p" transform="rotate(216 450 300)"/>
        <use href="#hk-p" transform="rotate(288 450 300)"/>
      </g>
    </svg>

    <!-- 2. 台湾 (青天白日满地红法定比例矢量: 左上四分之一青天白日，右下满地红) -->
    <svg v-else-if="code === 'TW'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#FE0000"/>
      <g>
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
      </g>
    </svg>

    <!-- 3. 日本 (日章旗法定标准矢量: 纯白底色，正红日轮居中，直径为旗高 3/5) -->
    <svg v-else-if="code === 'JP'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm border border-slate-700/60 shrink-0']">
      <rect width="36" height="24" fill="#FFFFFF"/>
      <circle cx="18" cy="12" r="7.2" fill="#BC002D"/>
    </svg>

    <!-- 4. 韩国 (大韩民国国旗: 太极旗法定标准矢量，中央红蓝太极，四角乾坤坎离四卦) -->
    <svg v-else-if="code === 'KR'" viewBox="-72 -48 144 96" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm border border-slate-700/60 shrink-0']">
      <path fill="#ffffff" d="M-72-48v96H72v-96z"/>
      <g>
        <g stroke="#000000" stroke-width="4">
          <path transform="rotate(33.69006752598)" d="M-50-12v24m6 0v-24m6 0v24m76 0V1m0-2v-11m6 0v11m0 2v11m6 0V1m0-2v-11"/>
          <path transform="rotate(-33.69006752598)" d="M-50-12v24m6 0V1m0-2v-11m6 0v24m76 0V1m0-2v-11m6 0v24m6 0V1m0-2v-11"/>
        </g>
        <g transform="rotate(33.69006752598)">
          <path fill="#cd2e3a" d="M12 0a18 18 0 11-36 0 24 24 0 1148 0"/>
          <path fill="#0047a0" d="M-24 0a24 24 0 1048 0A12 12 0 100 0a12 12 0 11-24 0"/>
        </g>
      </g>
    </svg>

    <!-- 5. 新加坡 (红白双色旗: 左上红底嵌白色新月与五颗五角星) -->
    <svg v-else-if="code === 'SG'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="12" fill="#ED2939"/>
      <rect y="12" width="36" height="12" fill="#FFFFFF"/>
      <g fill="#FFFFFF" transform="translate(7.5, 6)">
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
      <!-- Blue Canton (Union - 覆盖上 7 道条纹与 40% 旗宽) -->
      <g>
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
      </g>
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

    <!-- 10. 加拿大 (CA: 枫叶旗法定比例矢量) -->
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

    <!-- 17. 越南 (VN: 金星红旗法定标准矢量) -->
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

    <!-- 23. 瑞士 (CH: 瑞士红白十字国旗法定矢量) -->
    <svg v-else-if="code === 'CH'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#D52B1E"/>
      <g>
        <rect x="15" y="6" width="6" height="12" fill="#FFFFFF"/>
        <rect x="10" y="9" width="16" height="6" fill="#FFFFFF"/>
      </g>
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

    <!-- 28. 中国 (CN: 中华人民共和国国旗五星红旗法定标准矢量) -->
    <svg v-else-if="code === 'CN'" viewBox="0 0 36 24" preserveAspectRatio="none" :class="asBackground ? 'w-full h-full block' : [sizeClass, 'rounded-xs shadow-sm shrink-0']">
      <rect width="36" height="24" fill="#DE2910"/>
      <!-- 5 Stars Group (法定左上象限大五角星与四颗环拱小星) -->
      <g>
        <!-- Big Star centered at (6, 6) -->
        <polygon points="6,2.4 6.84,4.98 9.55,4.98 7.36,6.58 8.19,9.12 6,7.54 3.81,9.12 4.64,6.58 2.45,4.98 5.16,4.98" fill="#FFDE00"/>
        <!-- 4 Small Stars pointing at (6, 6) -->
        <polygon points="0,-1.2 0.28,-0.37 1.14,-0.37 0.44,0.14 0.71,0.97 0,0.46 -0.71,0.97 -0.44,0.14 -1.14,-0.37 -0.28,-0.37" fill="#FFDE00" transform="translate(12, 2.4) rotate(-31)"/>
        <polygon points="0,-1.2 0.28,-0.37 1.14,-0.37 0.44,0.14 0.71,0.97 0,0.46 -0.71,0.97 -0.44,0.14 -1.14,-0.37 -0.28,-0.37" fill="#FFDE00" transform="translate(14.4, 4.8) rotate(-8.1)"/>
        <polygon points="0,-1.2 0.28,-0.37 1.14,-0.37 0.44,0.14 0.71,0.97 0,0.46 -0.71,0.97 -0.44,0.14 -1.14,-0.37 -0.28,-0.37" fill="#FFDE00" transform="translate(14.4, 8.4) rotate(15.9)"/>
        <polygon points="0,-1.2 0.28,-0.37 1.14,-0.37 0.44,0.14 0.71,0.97 0,0.46 -0.71,0.97 -0.44,0.14 -1.14,-0.37 -0.28,-0.37" fill="#FFDE00" transform="translate(12, 10.8) rotate(38.7)"/>
      </g>
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

    <!-- 32. 兔窝镇私人电台基站 / 自建节点 (Bunnyburrow Carrot Farm & HomeLab Transmitter) -->
    <svg v-else-if="code === 'CUSTOM' && asBackground" viewBox="0 0 36 24" preserveAspectRatio="none" class="w-full h-full block">
      <defs>
        <!-- Twilight Pastoral Sunset Gradient -->
        <linearGradient id="ztp-custom-sky" x1="0" y1="0" x2="36" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#190e2e"/>
          <stop offset="40%" stop-color="#3b1054"/>
          <stop offset="75%" stop-color="#831843"/>
          <stop offset="100%" stop-color="#9a3412"/>
        </linearGradient>
        <!-- Glowing Carrot Gradient -->
        <linearGradient id="ztp-carrot-body" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stop-color="#fb923c"/>
          <stop offset="100%" stop-color="#ea580c"/>
        </linearGradient>
        <!-- Carrot Leaves Gradient -->
        <linearGradient id="ztp-carrot-leaf" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#4ade80"/>
          <stop offset="100%" stop-color="#15803d"/>
        </linearGradient>
      </defs>
      <!-- Twilight Sky -->
      <rect width="36" height="24" fill="url(#ztp-custom-sky)"/>
      <!-- Twinkling Countryside Stars & Golden Crescent Moon -->
      <circle cx="5" cy="4" r="0.4" fill="#ffffff" opacity="0.9"/>
      <circle cx="10" cy="2.5" r="0.3" fill="#fef08a" opacity="0.8"/>
      <circle cx="15" cy="5" r="0.35" fill="#ffffff" opacity="0.85"/>
      <circle cx="28" cy="3.5" r="0.4" fill="#fde047" opacity="0.85"/>
      <!-- Crescent Moon -->
      <path d="M7 2 A 2 2 0 1 0 9 5 A 1.6 1.6 0 1 1 7 2 Z" fill="#fef08a" opacity="0.9"/>
      <!-- Soft Rolling Countryside Hills (Bunnyburrow Farmland) -->
      <path d="M-2 24 Q 8 16, 18 19 Q 28 22, 38 17 L38 24 Z" fill="#14532d" opacity="0.9"/>
      <path d="M-2 24 Q 10 20, 22 22 Q 32 18, 38 21 L38 24 Z" fill="#052e16"/>
      <!-- Wi-Fi / Mesh Radio Transmission Waves from the Carrot Mast -->
      <circle cx="24" cy="9" r="4" fill="none" stroke="#ff9e5e" stroke-width="0.5" opacity="0.6" stroke-dasharray="1.5 1"/>
      <circle cx="24" cy="9" r="6.5" fill="none" stroke="#00f5d4" stroke-width="0.45" opacity="0.45" stroke-dasharray="2 1"/>
      <!-- Carrot Cottage & Antenna Mast -->
      <line x1="24" y1="4" x2="24" y2="18" stroke="#fef08a" stroke-width="0.6"/>
      <line x1="22" y1="7" x2="26" y2="7" stroke="#fef08a" stroke-width="0.5"/>
      <line x1="21" y1="11" x2="27" y2="11" stroke="#fef08a" stroke-width="0.5"/>
      <!-- Carrot House Body -->
      <path d="M21 13 Q 24 10, 27 13 L 25.5 20 Q 24 21, 22.5 20 Z" fill="url(#ztp-carrot-body)"/>
      <!-- Carrot House Leaves on Top -->
      <path d="M24 10 Q 22 6, 20 7 Q 22 9, 24 10 Q 26 6, 28 7 Q 26 9, 24 10 Z" fill="url(#ztp-carrot-leaf)"/>
      <circle cx="24" cy="4" r="0.75" fill="#ffffff" stroke="#ff9e5e" stroke-width="0.3"/>
      <!-- House Door and Cozy Window Glow -->
      <circle cx="24" cy="14" r="0.8" fill="#fef08a"/>
      <rect x="23.2" y="16.5" width="1.6" height="3" rx="0.5" fill="#78350f"/>
    </svg>
    <div v-else-if="code === 'CUSTOM'" class="text-sm shrink-0">
      🏠
    </div>

    <!-- 33. 环球生态全景仪 / 其它与公海 (Zootopia World Biosphere Transit) -->
    <svg v-else-if="code === 'OTHERS' && asBackground" viewBox="0 0 36 24" preserveAspectRatio="none" class="w-full h-full block">
      <defs>
        <linearGradient id="ztp-others-bg" x1="0" y1="0" x2="36" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#021c38"/>
          <stop offset="45%" stop-color="#0369a1"/>
          <stop offset="80%" stop-color="#075985"/>
          <stop offset="100%" stop-color="#064e3b"/>
        </linearGradient>
        <linearGradient id="ztp-brass" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#fef08a"/>
          <stop offset="100%" stop-color="#ca8a04"/>
        </linearGradient>
      </defs>
      <!-- Background: Deep Oceanic Void -->
      <rect width="36" height="24" fill="url(#ztp-others-bg)"/>
      <!-- Aurora Shimmer Bands Across Oceans -->
      <path d="M-2 4 Q 18 12, 38 6 L38 0 L-2 0 Z" fill="#00f5d4" opacity="0.15"/>
      <path d="M-2 20 Q 18 14, 38 22 L38 24 L-2 24 Z" fill="#38bdf8" opacity="0.18"/>
      <!-- Distant Golden Horizon Navigational Rules -->
      <line x1="0" y1="3" x2="36" y2="3" stroke="#fbbf24" stroke-width="0.5" opacity="0.55" stroke-dasharray="2 1"/>
      <line x1="0" y1="21" x2="36" y2="21" stroke="#fbbf24" stroke-width="0.5" opacity="0.55" stroke-dasharray="2 1"/>
      <!-- Celestial Armillary Sphere -->
      <g :transform="asBackground ? 'translate(18, 11) scale(0.72)' : 'translate(18, 12)'">
        <!-- Outer Golden Armillary Ring -->
        <circle cx="0" cy="0" r="9.2" fill="none" stroke="url(#ztp-brass)" stroke-width="0.8"/>
        <!-- Inner Holographic Globe -->
        <circle cx="0" cy="0" r="7.5" fill="#0c4a6e" stroke="#38bdf8" stroke-width="0.55" opacity="0.9"/>
        <!-- Globe Latitude & Longitude Grids -->
        <ellipse cx="0" cy="0" rx="3.6" ry="7.5" fill="none" stroke="#7dd3fc" stroke-width="0.45" opacity="0.75"/>
        <line x1="-7.5" y1="0" x2="7.5" y2="0" stroke="#7dd3fc" stroke-width="0.45" opacity="0.75"/>
        <ellipse cx="0" cy="0" rx="7.5" ry="3.5" fill="none" stroke="#00f5d4" stroke-width="0.4" opacity="0.6"/>
        <!-- 3D Tilted Brass Orbital Ring -->
        <ellipse cx="0" cy="0" rx="11.5" ry="4.2" fill="none" stroke="url(#ztp-brass)" stroke-width="0.75" transform="rotate(-26)" stroke-dasharray="2 0.8"/>
        <!-- 4 Biome Nexus Coordinates on the Ring (Glacier, Desert, Rainforest, City) -->
        <circle cx="-9" cy="4" r="0.9" fill="#38bdf8" stroke="#ffffff" stroke-width="0.3"/>
        <circle cx="9" cy="-4" r="0.9" fill="#facc15" stroke="#ffffff" stroke-width="0.3"/>
        <circle cx="0" cy="-7.5" r="0.9" fill="#22c55e" stroke="#ffffff" stroke-width="0.3"/>
        <circle cx="0" cy="7.5" r="0.9" fill="#c084fc" stroke="#ffffff" stroke-width="0.3"/>
        <!-- Center Navigator Star -->
        <polygon points="0,-2.2 0.7,-0.7 2.2,0 0.7,0.7 0,2.2 -0.7,0.7 -2.2,0 -0.7,-0.7" fill="#ffffff"/>
      </g>
    </svg>
    <div v-else-if="code === 'OTHERS'" class="text-sm shrink-0">
      🌐
    </div>

    <!-- 34. 动物城穿梭动车 / 网络订阅专属 (Zootopia Express Monorail Streamliner) -->
    <svg v-else-if="(code === 'SUB' || type === 'sub') && asBackground" viewBox="0 0 36 24" preserveAspectRatio="none" class="w-full h-full block">
      <defs>
        <!-- Sunset Dramatic Sky: Deep Navy to Blazing Sunset Orange -->
        <linearGradient id="ztp-sub-sky" x1="0" y1="0" x2="36" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#0b162c"/>
          <stop offset="35%" stop-color="#1e3a8a"/>
          <stop offset="70%" stop-color="#ea580c"/>
          <stop offset="100%" stop-color="#f59e0b"/>
        </linearGradient>
        <!-- Sleek High-Speed Train Body Gradient -->
        <linearGradient id="ztp-train-silver" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#ffffff"/>
          <stop offset="50%" stop-color="#e2e8f0"/>
          <stop offset="100%" stop-color="#94a3b8"/>
        </linearGradient>
        <!-- Headlight Beam Flare -->
        <linearGradient id="ztp-headlight" x1="0" y1="0" x2="1" y2="0">
          <stop offset="0%" stop-color="#fef08a" stop-opacity="0.9"/>
          <stop offset="100%" stop-color="#ffffff" stop-opacity="0"/>
        </linearGradient>
      </defs>
      <!-- Sunset Sky Background -->
      <rect width="36" height="24" fill="url(#ztp-sub-sky)"/>
      <!-- Distant City Skyline Silhouettes in Sunset Glow -->
      <polygon points="2,14 4,14 4,11 6,11 6,14 10,14 10,9 12,9 12,14 16,14 16,12 18,12 18,14 26,14 26,10 28,10 28,14 36,14 36 24 0 24 0 14" fill="#0f172a" opacity="0.6"/>
      <!-- Giant Golden Setting Sun at Horizon -->
      <circle cx="22" cy="14" r="5" fill="#fef08a" opacity="0.35"/>
      <circle cx="22" cy="14" r="3.2" fill="#fffbeb" opacity="0.5"/>
      <!-- Elevated Monorail Track: Sweeping Perspective Curve from Left to Right -->
      <path d="M-2 19 Q 14 16, 38 12" stroke="#64748b" stroke-width="1.8" fill="none"/>
      <path d="M-2 19 Q 14 16, 38 12" stroke="#38bdf8" stroke-width="0.5" fill="none" opacity="0.8"/>
      <!-- Track Pillar Columns -->
      <line x1="6" y1="18" x2="6" y2="24" stroke="#1e293b" stroke-width="0.9"/>
      <line x1="16" y1="16.5" x2="16" y2="24" stroke="#1e293b" stroke-width="0.9"/>
      <line x1="28" y1="14" x2="28" y2="24" stroke="#1e293b" stroke-width="0.9"/>
      <!-- Supersonic Speed Streaks (Try Everything Jet Trails) -->
      <line x1="0" y1="14.5" x2="18" y2="12.5" stroke="#00f5d4" stroke-width="0.6" opacity="0.75"/>
      <line x1="-2" y1="16.5" x2="14" y2="14.5" stroke="#ffffff" stroke-width="0.7" opacity="0.9"/>
      <line x1="4" y1="13.2" x2="20" y2="11.5" stroke="#38bdf8" stroke-width="0.4" opacity="0.6"/>
      <!-- Streamlined Zootopia Express Bullet Train -->
      <path d="M12 14.5 Q 22 13, 27 12 Q 29.5 11.5, 30.5 13 Q 29 14.2, 26 14.5 L 12 15.8 Z" fill="url(#ztp-train-silver)"/>
      <path d="M26 12.3 Q 28.5 12, 29.5 12.8 Q 28 13.5, 26 13.5 Z" fill="#0284c7"/>
      <path d="M13 14.8 L 27 13.6" stroke="#00f5d4" stroke-width="0.4"/>
      <!-- Glowing Passenger Windows -->
      <rect x="14" y="13.8" width="1.5" height="0.8" rx="0.2" fill="#fef08a"/>
      <rect x="17" y="13.5" width="1.5" height="0.8" rx="0.2" fill="#fef08a"/>
      <rect x="20" y="13.2" width="1.5" height="0.8" rx="0.2" fill="#fef08a"/>
      <rect x="23" y="12.9" width="1.5" height="0.8" rx="0.2" fill="#fef08a"/>
      <!-- Powerful Headlight Beam Shooting Forward -->
      <polygon points="30.2,13 38,11 38,15.5 30.2,13.5" fill="url(#ztp-headlight)"/>
      <circle cx="30.2" cy="13.2" r="0.5" fill="#ffffff"/>
    </svg>
    <div v-else-if="code === 'SUB' || type === 'sub'" class="text-sm shrink-0">
      ✈️
    </div>

    <!-- 35. 动物城警察局警徽盾牌 / 规则拦截阻断 (ZPD Officer Star Badge & Security Shield) -->
    <svg v-else-if="code === 'REJECT' && asBackground" viewBox="0 0 36 24" preserveAspectRatio="none" class="w-full h-full block">
      <defs>
        <!-- Midnight Police Navy Void with Tactical Carbon Texture -->
        <linearGradient id="ztp-zpd-bg" x1="0" y1="0" x2="36" y2="24" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stop-color="#040914"/>
          <stop offset="50%" stop-color="#081830"/>
          <stop offset="100%" stop-color="#0f274a"/>
        </linearGradient>
        <!-- Officer Golden Star / Shield Rim -->
        <linearGradient id="ztp-zpd-gold" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stop-color="#fef08a"/>
          <stop offset="40%" stop-color="#facc15"/>
          <stop offset="80%" stop-color="#ca8a04"/>
          <stop offset="100%" stop-color="#78350f"/>
        </linearGradient>
        <!-- Police Warning Beacons: Red & Blue Flasher Flare -->
        <radialGradient id="ztp-flasher-red" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="#ef4444" stop-opacity="0.8"/>
          <stop offset="100%" stop-color="#ef4444" stop-opacity="0"/>
        </radialGradient>
        <radialGradient id="ztp-flasher-blue" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="#3b82f6" stop-opacity="0.8"/>
          <stop offset="100%" stop-color="#3b82f6" stop-opacity="0"/>
        </radialGradient>
      </defs>
      <!-- Background: Police Midnight Blue -->
      <rect width="36" height="24" fill="url(#ztp-zpd-bg)"/>
      <!-- Twin Police Emergency Flasher Warning Beacons (Left Red / Right Blue) -->
      <circle cx="2" cy="12" r="10" fill="url(#ztp-flasher-red)"/>
      <circle cx="34" cy="12" r="10" fill="url(#ztp-flasher-blue)"/>
      <line x1="0" y1="12" x2="12" y2="12" stroke="#ef4444" stroke-width="0.4" opacity="0.6" stroke-dasharray="1 1"/>
      <line x1="24" y1="12" x2="36" y2="12" stroke="#3b82f6" stroke-width="0.4" opacity="0.6" stroke-dasharray="1 1"/>
      <!-- Grand Officer ZPD Badge Shield (Centered with Judy's Golden Star) -->
      <g :transform="asBackground ? 'translate(18, 11) scale(0.85)' : 'translate(18, 12)'">
        <!-- Outer Heavy Gold Shield Rim -->
        <path d="M0 -11 L8.5 -6 V2 C8.5 7.8 0 11.2 0 11.2 C0 11.2 -8.5 7.8 -8.5 2 V-6 Z" fill="url(#ztp-zpd-gold)" stroke="#ffffff" stroke-width="0.5"/>
        <!-- Inner Police Navy Shield Plate -->
        <path d="M0 -9.8 L7.2 -5.2 V1.6 C7.2 6.5 0 9.8 0 9.8 C0 9.8 -7.2 6.5 -7.2 1.6 V-5.2 Z" fill="#0a1a36"/>
        <!-- Top ZPD Police Ribbon Banner -->
        <path d="M-6 -4.8 H 6 V-2.2 H -6 Z" fill="url(#ztp-zpd-gold)" rx="0.5"/>
        <text x="0" y="-3" font-size="2" font-weight="900" font-family="sans-serif" fill="#1e1106" text-anchor="middle" letter-spacing="0.4">ZPD</text>
        <!-- Radiant 8-Point Officer Star in Center of Badge -->
        <polygon points="0,-1.8 1,1.5 -1,1.5" fill="url(#ztp-zpd-gold)"/>
        <polygon points="0,5.8 1,2.5 -1,2.5" fill="url(#ztp-zpd-gold)"/>
        <polygon points="-3.8,2 0,1 0,3" fill="url(#ztp-zpd-gold)"/>
        <polygon points="3.8,2 0,1 0,3" fill="url(#ztp-zpd-gold)"/>
        <polygon points="-2.6,-0.6 0,1.2 -0.8,2" fill="#fef08a"/>
        <polygon points="2.6,-0.6 0,1.2 0.8,2" fill="#fef08a"/>
        <polygon points="-2.6,4.6 0,2.8 -0.8,2" fill="#fef08a"/>
        <polygon points="2.6,4.6 0,2.8 0.8,2" fill="#fef08a"/>
        <!-- Center Police Star Diamond Core -->
        <circle cx="0" cy="2" r="1.3" fill="#ffffff" stroke="#facc15" stroke-width="0.3"/>
        <polygon points="0,1 0.7,2 0,3 -0.7,2" fill="#ef4444"/>
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
