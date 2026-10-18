<template>
  <div :class="['inline-flex items-center justify-center shrink-0', size || 'w-5 h-5']">
    <!-- OpenAI -->
    <svg v-if="logoKey === 'openai'" class="w-full h-full text-teal-400 filter drop-shadow-[0_0_6px_rgba(45,212,191,0.4)]" viewBox="0 0 24 24" fill="currentColor">
      <path d="M22.28 9.53a5.98 5.98 0 0 0-.52-4.91 6.05 6.05 0 0 0-6.51-2.9 6.06 6.06 0 0 0-4.88-2.43 6.05 6.05 0 0 0-5.76 4.19 6.05 6.05 0 0 0-4.08 2.96 6.05 6.05 0 0 0 .76 7.07 5.98 5.98 0 0 0 .52 4.91 6.05 6.05 0 0 0 6.51 2.9 6.06 6.06 0 0 0 4.88 2.43 6.05 6.05 0 0 0 5.76-4.19 6.05 6.05 0 0 0 4.08-2.96 6.05 6.05 0 0 0-.76-7.07zm-7.6 11.23a4.54 4.54 0 0 1-3.08-1.22l.14-.08 5.09-2.94a.78.78 0 0 0 .39-.68v-7.18l2.15 1.24a.07.07 0 0 1 .04.05v6.23a4.56 4.56 0 0 1-4.73 4.58zm-9.35-4.04a4.54 4.54 0 0 1-.58-3.26l.14.08 5.09 2.94a.78.78 0 0 0 .78 0l6.22-3.59v2.48a.07.07 0 0 1-.03.06l-5.4 3.12a4.56 4.56 0 0 1-6.22-1.83zm-1.74-9.52a4.54 4.54 0 0 1 2.5-2.04v6.13a.78.78 0 0 0 .39.68l6.22 3.59-2.15 1.24a.07.07 0 0 1-.07 0l-5.4-3.12a4.56 4.56 0 0 1-1.49-6.48zm12.33 2.65l-6.22-3.59 2.15-1.24a.07.07 0 0 1 .07 0l5.4 3.12a4.56 4.56 0 0 1 1.49 6.48 4.54 4.54 0 0 1-2.5 2.04v-6.13a.78.78 0 0 0-.39-.68zm3.03-1.63l-.14-.08-5.09-2.94a.78.78 0 0 0-.78 0l-6.22 3.59v-2.48a.07.07 0 0 1 .03-.06l5.4-3.12a4.56 4.56 0 0 1 6.8 5.09zm-8.95 5.17l-2.76-1.59 2.76-1.59 2.76 1.59-2.76 1.59z"/>
    </svg>

    <!-- Gemini -->
    <svg v-else-if="logoKey === 'gemini'" class="w-full h-full filter drop-shadow-[0_0_6px_rgba(59,130,246,0.5)]" viewBox="0 0 24 24" fill="none">
      <path d="M12 0C12 6.627 6.627 12 0 12C6.627 12 12 17.373 12 24C12 17.373 17.373 12 24 12C17.373 12 12 6.627 12 0Z" fill="url(#geminiGrad)" />
      <defs>
        <linearGradient id="geminiGrad" x1="0" y1="0" x2="24" y2="24" gradientUnits="userSpaceOnUse">
          <stop stop-color="#1BA1E2"/>
          <stop offset="0.5" stop-color="#7B2CBF"/>
          <stop offset="1" stop-color="#00F5D4"/>
        </linearGradient>
      </defs>
    </svg>

    <!-- Claude (Anthropic) -->
    <svg v-else-if="logoKey === 'claude'" class="w-full h-full filter drop-shadow-[0_0_6px_rgba(217,119,6,0.5)]" viewBox="0 0 24 24" fill="#D97706">
      <path d="M12 0l2.2 7.8 7.8-2.2-4.8 6.4 6.8 4.4-7.8 2.2 2.2 7.8-6.4-4.8-4.4 6.8-2.2-7.8-7.8 2.2 4.8-6.4-6.8-4.4 7.8-2.2-2.2-7.8 6.4 4.8z"/>
    </svg>

    <!-- YouTube -->
    <svg v-else-if="logoKey === 'youtube'" class="w-full h-full filter drop-shadow-[0_0_6px_rgba(255,0,0,0.35)]" viewBox="0 0 24 24" fill="#FF0000">
      <path d="M23.498 6.186a3.016 3.016 0 0 0-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 0 0 .502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 0 0 2.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 0 0 2.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814z"/>
      <path d="M9.545 15.568V8.432L15.818 12l-6.273 3.568z" fill="#FFFFFF"/>
    </svg>

    <!-- Google -->
    <svg v-else-if="logoKey === 'google'" class="w-full h-full" viewBox="0 0 24 24">
      <path fill="#4285F4" d="M23.745 12.27c0-.7-.06-1.4-.19-2.07H12v4.51h6.6c-.29 1.52-1.14 2.82-2.4 3.68v3.05h3.88c2.27-2.09 3.66-5.17 3.66-9.17z"/>
      <path fill="#34A853" d="M12 24c3.24 0 5.95-1.08 7.93-2.91l-3.88-3.05c-1.08.72-2.45 1.16-4.05 1.16-3.12 0-5.77-2.1-6.72-4.93H1.25v3.15C3.26 21.36 7.33 24 12 24z"/>
      <path fill="#FBBC05" d="M5.28 14.27c-.25-.72-.38-1.49-.38-2.27s.13-1.55.38-2.27V6.58H1.25C.45 8.18 0 9.99 0 12s.45 3.82 1.25 5.42l4.03-3.15z"/>
      <path fill="#EA4335" d="M12 4.75c1.77 0 3.35.61 4.6 1.8l3.42-3.42C17.95 1.19 15.24 0 12 0 7.33 0 3.26 2.64 1.25 6.58l4.03 3.15c.95-2.83 3.6-4.98 6.72-4.98z"/>
    </svg>

    <!-- Cloudflare -->
    <svg v-else-if="logoKey === 'cloudflare'" class="w-full h-full filter drop-shadow-[0_0_6px_rgba(243,128,32,0.4)]" viewBox="0 0 24 24">
      <path fill="#F38020" d="M16.5 9.4c-.4-2.8-2.8-4.9-5.7-4.9-2.5 0-4.6 1.6-5.4 3.9-2 .2-3.6 1.9-3.6 4 0 2.2 1.8 4 4 4h10.5c2 0 3.7-1.7 3.7-3.7 0-1.9-1.5-3.5-3.5-3.7v.4z"/>
      <path fill="#FAAD3F" d="M19.8 11.2c-.2 0-.4 0-.6.1.5.7.8 1.5.8 2.4 0 2-1.7 3.7-3.7 3.7H6c.6 1.4 2 2.3 3.6 2.3h10.2c2.2 0 4-1.8 4-4 0-2.1-1.7-3.9-3.8-4.1-.1-.3-.1-.4-.2-.4z"/>
    </svg>

    <!-- GitHub / Docker -->
    <svg v-else-if="logoKey === 'github'" class="w-full h-full text-white" viewBox="0 0 24 24" fill="currentColor">
      <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"/>
    </svg>

    <!-- Telegram -->
    <svg v-else-if="logoKey === 'telegram'" class="w-full h-full filter drop-shadow-[0_0_6px_rgba(42,171,238,0.4)]" viewBox="0 0 24 24" fill="none">
      <circle cx="12" cy="12" r="12" fill="#2AABEE"/>
      <path d="M17.5 7.5L5.5 12.1L9.5 13.6L14.5 10.2L10.7 14.7L15.8 17.8L17.5 7.5Z" fill="#FFFFFF"/>
    </svg>

    <!-- Twitter (X) -->
    <svg v-else-if="logoKey === 'twitter'" class="w-full h-full text-white" viewBox="0 0 24 24" fill="currentColor">
      <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/>
    </svg>

    <!-- Netflix (Official Authentic 3D Ribbon "N") -->
    <svg v-else-if="logoKey === 'netflix'" class="w-full h-full filter drop-shadow-[0_0_8px_rgba(229,9,20,0.5)]" viewBox="0 0 24 24" fill="none">
      <path d="M4 0h4.8v24H4z" fill="#B1060E"/>
      <path d="M15.2 0H20v24h-4.8z" fill="#B1060E"/>
      <path d="M4 0h4.9l11.1 23.6c-1.8.3-3.6.4-5.2.4L4 0z" fill="#E50914"/>
    </svg>

    <!-- Disney+ -->
    <svg v-else-if="logoKey === 'disney'" class="w-full h-full filter drop-shadow-[0_0_6px_rgba(17,60,207,0.5)]" viewBox="0 0 24 24" fill="none">
      <path d="M2.5 17.5C7 7.5 16.5 5 21.5 8" stroke="#113CCF" stroke-width="2.5" stroke-linecap="round"/>
      <path d="M18 4l1 3 3 1-3 1-1 3-1-3-3-1 3-1z" fill="#38BDF8"/>
    </svg>

    <!-- Bahamut (巴哈姆特) -->
    <svg v-else-if="logoKey === 'bahamut'" class="w-full h-full text-sky-400" viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 2L3 8v8l9 6 9-6V8l-9-6zm0 3.5l6 4v5l-6 4-6-4v-5l6-4z"/>
      <circle cx="12" cy="12.5" r="2.5" fill="#38BDF8"/>
    </svg>

    <!-- Emby -->
    <svg v-else-if="logoKey === 'emby'" class="w-full h-full text-emerald-400 filter drop-shadow-[0_0_6px_rgba(16,185,129,0.4)]" viewBox="0 0 24 24" fill="currentColor">
      <path d="M11 2L2 7v10l9 5 9-5V7l-9-5zm1 14.5l-5-3V9.5l5 3.2 5-3.2v4l-5 3z"/>
    </svg>

    <!-- Epic Games -->
    <svg v-else-if="logoKey === 'epicgames'" class="w-full h-full text-slate-100" viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 2C6.5 2 2 4.2 2 7v10c0 4 10 5 10 5s10-1 10-5V7c0-2.8-4.5-5-10-5zm0 3c3.5 0 6 1.2 6 2.5S15.5 10 12 10 6 8.8 6 7.5 8.5 5 12 5z"/>
    </svg>

    <!-- PlayStation (PSN) -->
    <svg v-else-if="logoKey === 'playstation'" class="w-full h-full text-sky-400" viewBox="0 0 24 24" fill="currentColor">
      <path d="M8.5 2.5C7.2 3.1 6.5 4.5 6.5 6.2v10.3l3.5 1.2V6.5c0-.8.4-1.2 1-1.4.6-.2 1.2 0 1.5.5.3.4.4 1 .4 1.8v10.3l3.5 1.2V7.1c0-1.8-.7-3.2-2-3.8-1.7-.8-4.3-.8-5.4-.8zm-4.7 15.3l5.8 2v1.5L2 18.7v-1.2l1.8.3zm12.4 0l5.8-2v1.2L14.4 20v-1.5l1.8-.7z"/>
    </svg>

    <!-- Nintendo Switch -->
    <svg v-else-if="logoKey === 'nintendo'" class="w-full h-full" viewBox="0 0 24 24" fill="#E60012">
      <path d="M10 2H6C3.8 2 2 3.8 2 6v12c0 2.2 1.8 4 4 4h4V2zm-4 7a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3zm8-7h4c2.2 0 4 1.8 4 4v12c0 2.2-1.8 4-4 4h-4V2zm4 13a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3z"/>
    </svg>

    <!-- Amazon -->
    <svg v-else-if="logoKey === 'amazon'" class="w-full h-full" viewBox="0 0 24 24" fill="#FF9900">
      <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4.5 14.5c-3.5 2-7.5 1.5-10 0-.3-.2-.1-.6.2-.5 2.2 1.2 5.8 1.4 8.8-.2.4-.2.7.4.3.7zm1.5-2c-.2.3-.6.4-.9.2-.8-.5-1.8-.8-2.8-.8s-2 .3-2.8.8c-.3.2-.7.1-.9-.2-.2-.3-.1-.7.2-.9 1-.6 2.2-1 3.5-1s2.5.4 3.5 1c.3.2.4.6.2.9z"/>
    </svg>

    <!-- PayPal -->
    <svg v-else-if="logoKey === 'paypal'" class="w-full h-full" viewBox="0 0 24 24">
      <path fill="#003087" d="M7 2h7c3.3 0 5.5 1.6 5 4.5-.4 2.5-2.2 4-4.8 4H11l-1.5 8H5.5L7 2z"/>
      <path fill="#0079C1" d="M10 7h6c2.5 0 4.2 1.2 3.8 3.5-.4 2.2-2 3.5-4.3 3.5h-2.5L11.5 22H8L10 7z"/>
    </svg>

    <!-- Speedtest -->
    <svg v-else-if="logoKey === 'speedtest'" class="w-full h-full text-cyan-400 filter drop-shadow-[0_0_6px_rgba(6,182,212,0.4)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="9"/>
      <path d="M12 7v5l3 3" stroke="#00D2FF" stroke-width="2.5" stroke-linecap="round"/>
    </svg>

    <!-- Reddit -->
    <svg v-else-if="logoKey === 'reddit'" class="w-full h-full" viewBox="0 0 24 24" fill="#FF4500">
      <circle cx="12" cy="12" r="10"/>
      <circle cx="8.5" cy="11.5" r="1.5" fill="#FFFFFF"/>
      <circle cx="15.5" cy="11.5" r="1.5" fill="#FFFFFF"/>
      <path d="M9 15.5c1.5 1.2 4.5 1.2 6 0" stroke="#FFFFFF" stroke-width="1.5" stroke-linecap="round" fill="none"/>
    </svg>

    <!-- Apple -->
    <svg v-else-if="logoKey === 'apple'" class="w-full h-full text-slate-100" viewBox="0 0 24 24" fill="currentColor">
      <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M15.97 4.88c.63-.77 1.06-1.84.94-2.88-.91.04-2.02.61-2.67 1.38-.58.67-1.09 1.76-.95 2.8 1.02.08 2.05-.53 2.68-1.3"/>
    </svg>

    <!-- Microsoft -->
    <svg v-else-if="logoKey === 'microsoft'" class="w-full h-full" viewBox="0 0 24 24">
      <path fill="#F25022" d="M1 1h10v10H1z"/>
      <path fill="#7FBA00" d="M13 1h10v10H13z"/>
      <path fill="#00A4EF" d="M1 13h10v10H1z"/>
      <path fill="#FFB900" d="M13 13h10v10H13z"/>
    </svg>

    <!-- OneDrive -->
    <svg v-else-if="logoKey === 'onedrive'" class="w-full h-full" viewBox="0 0 24 24" fill="#0078D4">
      <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z"/>
    </svg>

    <!-- Spotify -->
    <svg v-else-if="logoKey === 'spotify'" class="w-full h-full filter drop-shadow-[0_0_6px_rgba(29,185,84,0.4)]" viewBox="0 0 24 24" fill="#1DB954">
      <path d="M12 0C5.4 0 0 5.4 0 12s5.4 12 12 12 12-5.4 12-12S18.66 0 12 0zm5.521 17.34c-.24.359-.66.48-1.021.24-2.82-1.74-6.36-2.101-10.561-1.141-.418.122-.779-.179-.899-.539-.12-.421.18-.78.54-.9 4.56-1.021 8.52-.6 11.64 1.32.42.18.479.659.301 1.02zm1.44-3.3c-.301.42-.841.6-1.262.3-3.239-1.98-8.159-2.58-11.939-1.38-.479.12-1.02-.12-1.14-.6-.12-.48.12-1.021.6-1.141C9.6 9.9 15 10.561 18.72 12.84c.361.181.54.78.241 1.2zm.12-3.36C15.24 8.4 8.82 8.16 5.16 9.301c-.6.179-1.2-.181-1.38-.721-.18-.601.18-1.2.72-1.381 4.26-1.26 11.28-1.02 15.721 1.621.539.3.719 1.02.419 1.56-.299.421-1.02.599-1.559.3z"/>
    </svg>

    <!-- Steam -->
    <svg v-else-if="logoKey === 'steam'" class="w-full h-full text-sky-400" viewBox="0 0 24 24" fill="currentColor">
      <path d="M11.979 0C5.678 0 .511 4.86.022 11.037l6.432 2.658c.545-.371 1.203-.59 1.912-.59.063 0 .125.004.188.006l2.861-4.142V8.91c0-2.495 2.028-4.524 4.524-4.524 2.494 0 4.524 2.031 4.524 4.527s-2.03 4.525-4.524 4.525h-.105l-4.076 2.911c0 .052.005.105.005.159 0 1.875-1.515 3.396-3.39 3.396-1.635 0-3.016-1.173-3.331-2.727L.436 14.819C1.905 20.074 6.702 24 12 24c6.627 0 12-5.373 12-12S18.627 0 11.979 0"/>
    </svg>

    <!-- TikTok -->
    <svg v-else-if="logoKey === 'tiktok'" class="w-full h-full" viewBox="0 0 24 24" fill="currentColor">
      <path fill="#25F4EE" d="M12.5 0v16.2a3.8 3.8 0 1 1-3.8-3.8c.4 0 .8.1 1.2.2V8.2a8.2 8.2 0 0 0-1.2-.1 8.3 8.3 0 1 0 8.3 8.3V7.1a11.8 11.8 0 0 0 7-2.3V.5a7.3 7.3 0 0 1-7.1 2.3V0h-4.4z"/>
      <path fill="#FE2C55" d="M11.5 1v16.2a3.8 3.8 0 1 1-3.8-3.8c.4 0 .8.1 1.2.2V9.2a8.2 8.2 0 0 0-1.2-.1 8.3 8.3 0 1 0 8.3 8.3V8.1a11.8 11.8 0 0 0 7-2.3V1.5a7.3 7.3 0 0 1-7.1 2.3V1h-4.4z" opacity="0.8"/>
    </svg>

    <!-- Bilibili -->
    <svg v-else-if="logoKey === 'bilibili'" class="w-full h-full text-sky-400" viewBox="0 0 24 24" fill="currentColor">
      <path d="M17.813 4.653h.854c1.51 0 2.733 1.224 2.733 2.734v10.146c0 1.51-1.223 2.734-2.733 2.734H5.333c-1.51 0-2.733-1.224-2.733-2.734V7.387c0-1.51 1.223-2.734 2.733-2.734h.854L4.35 2.817a.82.82 0 0 1 .184-1.144.82.82 0 0 1 1.144.184l2.84 3.796h6.964l2.84-3.796a.82.82 0 0 1 1.144-.184.82.82 0 0 1 .184 1.144l-1.837 1.836zM7.5 10.5a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zm9 0a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3z"/>
    </svg>

    <!-- Discord -->
    <svg v-else-if="logoKey === 'discord'" class="w-full h-full text-indigo-400" viewBox="0 0 24 24" fill="currentColor">
      <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.874-1.295 1.226-1.994.021-.041.001-.09-.041-.106a13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.929 1.793 8.18 1.793 12.061 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.894.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.028zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z"/>
    </svg>

    <!-- Zoom -->
    <svg v-else-if="logoKey === 'zoom'" class="w-full h-full text-sky-400" viewBox="0 0 24 24" fill="currentColor">
      <path d="M4 6a3 3 0 0 0-3 3v6a3 3 0 0 0 3 3h10a3 3 0 0 0 3-3V9a3 3 0 0 0-3-3H4zm14.5 3.3L22 7v10l-3.5-2.3V9.3z"/>
    </svg>

    <!-- Notion -->
    <svg v-else-if="logoKey === 'notion'" class="w-full h-full text-white" viewBox="0 0 24 24" fill="currentColor">
      <path d="M4.459 4.208c.746.606 1.026.56 2.428.466l13.215-.793c.28 0 .047-.28-.046-.326L17.86 1.782c-.466-.373-.84-.373-1.867-.326L3.992 2.389c-.373 0-.42.233-.28.466l.747 1.353zm.607 3.546v13.43c0 .84.42 1.12 1.353 1.073l13.916-.84c.933-.047 1.073-.607 1.073-1.307V6.868c0-.653-.28-.98-.933-.933l-14.476.84c-.653.047-.933.373-.933.979zm12.378 1.446c.14 0 .233.093.233.28v10.593c0 .28-.14.373-.326.373-.14 0-.327-.093-.42-.28l-5.647-8.82v8.587c0 .28-.14.373-.373.373h-1.633c-.187 0-.28-.093-.28-.28V9.387c0-.28.14-.373.373-.373.187 0 .373.093.467.28l5.693 8.773V9.48c0-.28.14-.28.28-.28h1.633z"/>
    </svg>

    <!-- Binance / Crypto -->
    <svg v-else-if="logoKey === 'binance'" class="w-full h-full" viewBox="0 0 24 24" fill="#F3BA2F">
      <path d="M16.624 13.92l2.715 2.715-7.34 7.34-7.339-7.34 2.715-2.715 4.624 4.624 4.626-4.624zm-4.625-7.344l3.82 3.82-3.82 3.82-3.82-3.82 3.82-3.82zm7.34 2.72l2.715 2.716-2.715 2.715-2.715-2.715 2.715-2.716zm-14.679 0l2.715 2.716-2.715 2.715-2.715-2.715 2.715-2.716zm7.339-9.3l7.34 7.34-2.715 2.715-4.625-4.624-4.625 4.624-2.715-2.715 7.34-7.34z"/>
    </svg>

    <!-- Baidu (百度官方熊掌) -->
    <svg v-else-if="logoKey === 'baidu'" class="w-full h-full filter drop-shadow-[0_0_6px_rgba(41,50,225,0.4)]" viewBox="0 0 24 24" fill="#2932E1">
      <path d="M12.002 12.87c-1.89 0-3.834.787-4.113 2.871-.237 1.77 1.096 3.86 2.724 4.545 1.156.486 2.83.473 3.842-.26 1.134-.823 1.66-2.502 1.393-3.903-.338-1.777-1.956-3.253-3.846-3.253zm-5.467-3.41c-1.127.135-2.073 1.34-2.112 2.69-.04 1.397.87 2.656 2.036 2.81 1.165.155 2.274-.897 2.476-2.35.202-1.455-.718-3.08-2.4-3.15zm10.934 0c-1.682.07-2.602 1.695-2.4 3.15.202 1.453 1.31 2.505 2.476 2.35 1.166-.154 2.076-1.413 2.036-2.81-.04-1.35-.985-2.555-2.112-2.69zm-7.69-5.45c-1.22.464-1.96 1.947-1.65 3.313.31 1.365 1.54 2.222 2.748 1.916 1.21-.305 1.944-1.745 1.637-3.158-.306-1.414-1.515-2.535-2.735-2.071zm4.444 0c-1.22-.464-2.43 0.657-2.736 2.07-.306 1.414.428 2.854 1.637 3.16 1.208.305 2.438-.552 2.748-1.917.31-1.366-.43-2.85-1.65-3.313z"/>
    </svg>

    <!-- Direct (直连) -->
    <svg v-else-if="logoKey === 'direct'" class="w-full h-full text-emerald-400 filter drop-shadow-[0_0_6px_rgba(16,185,129,0.4)]" viewBox="0 0 24 24" fill="currentColor">
      <path d="M13 2L3 14h7v8l10-12h-7z"/>
    </svg>

    <!-- 漏网之鱼 (兜底分流星芒) -->
    <svg v-else-if="logoKey === 'fish'" class="w-full h-full text-amber-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="9"/>
      <polygon points="12 3 14 10 21 12 14 14 12 21 10 14 3 12 10 10 12 3" fill="#FBBF24"/>
    </svg>

    <!-- Cyber Gateway Ring Fallback (No Crude Emoji) -->
    <svg v-else class="w-full h-full text-cyan-400 filter drop-shadow-[0_0_6px_rgba(6,182,212,0.4)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="9"/>
      <path d="M12 3a15 15 0 0 1 0 18M12 3a15 15 0 0 0 0 18M3 12h18"/>
    </svg>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: {
    type: [String, Object],
    default: ''
  },
  icon: {
    type: String,
    default: ''
  },
  size: {
    type: String,
    default: 'w-5 h-5'
  }
})

const logoKey = computed(() => {
  let str = ''
  if (props.icon) {
    str += props.icon.toLowerCase() + ' '
  }
  if (typeof props.name === 'string') {
    str += props.name.toLowerCase()
  } else if (props.name && typeof props.name === 'object') {
    str += (props.name.name || '').toLowerCase() + ' ' + (props.name.icon || '').toLowerCase()
  }

  if (str.includes('openai') || str.includes('chatgpt')) return 'openai'
  if (str.includes('gemini')) return 'gemini'
  if (str.includes('claude') || str.includes('anthropic')) return 'claude'
  if (str.includes('youtube')) return 'youtube'
  if (str.includes('google')) return 'google'
  if (str.includes('cloudflare')) return 'cloudflare'
  if (str.includes('github') || str.includes('docker')) return 'github'
  if (str.includes('telegram')) return 'telegram'
  if (str.includes('twitter') || str.includes(' x ') || str.endsWith(' x') || str.includes('(x)')) return 'twitter'
  if (str.includes('netflix') || str.includes('奈飞') || str.includes('网飞')) return 'netflix'
  if (str.includes('disney') || str.includes('迪士尼')) return 'disney'
  if (str.includes('bahamut') || str.includes('巴哈姆特')) return 'bahamut'
  if (str.includes('emby')) return 'emby'
  if (str.includes('epic') || str.includes('epicgames')) return 'epicgames'
  if (str.includes('playstation') || str.includes('psn')) return 'playstation'
  if (str.includes('nintendo') || str.includes('switch') || str.includes('任天堂')) return 'nintendo'
  if (str.includes('amazon') || str.includes('亚马逊') || str.includes('prime')) return 'amazon'
  if (str.includes('paypal')) return 'paypal'
  if (str.includes('speedtest') || str.includes('测速')) return 'speedtest'
  if (str.includes('reddit')) return 'reddit'
  if (str.includes('apple') || str.includes('苹果')) return 'apple'
  if (str.includes('onedrive')) return 'onedrive'
  if (str.includes('microsoft') || str.includes('微软')) return 'microsoft'
  if (str.includes('spotify')) return 'spotify'
  if (str.includes('steam')) return 'steam'
  if (str.includes('tiktok')) return 'tiktok'
  if (str.includes('bilibili') || str.includes('哔哩哔哩')) return 'bilibili'
  if (str.includes('discord')) return 'discord'
  if (str.includes('zoom')) return 'zoom'
  if (str.includes('notion')) return 'notion'
  if (str.includes('binance') || str.includes('okx') || str.includes('币安') || str.includes('加密货币')) return 'binance'
  if (str.includes('baidu') || str.includes('百度')) return 'baidu'
  if (str.includes('direct') || str.includes('直连') || str.includes('国内')) return 'direct'
  if (str.includes('fish') || str.includes('漏网之鱼') || str.includes('兜底')) return 'fish'
  if (str.includes('globe') || str.includes('国外') || str.includes('海外')) return 'default'

  return 'default'
})
</script>
