<template>
  <div class="inline-flex items-center justify-center shrink-0 w-8 h-8 rounded-xl bg-white/[0.08] border border-white/10 group-hover:scale-105 transition-all shadow-sm">
    <!-- 1. Apple (iPhone / iPad / Mac / Apple TV) -->
    <svg v-if="deviceType === 'apple'" viewBox="0 0 170 170" fill="currentColor" class="w-4 h-4 text-slate-100 drop-shadow-sm">
      <path d="M150.37 130.25c-2.45 5.66-5.35 10.87-8.71 15.66-4.58 6.53-8.33 11.05-11.22 13.56-4.48 4.12-9.28 6.23-14.42 6.35-3.69 0-8.14-1.05-13.32-3.18-5.19-2.12-9.97-3.17-14.34-3.17-4.58 0-9.49 1.05-14.75 3.17-5.26 2.13-9.5 3.24-12.74 3.35-4.35.13-9.16-1.9-14.42-6.08-3.7-3.07-7.64-7.85-11.82-14.34-5.34-8.28-9.49-17.65-12.44-28.12-2.95-10.46-4.43-20.73-4.43-30.81 0-14.82 3.65-27.14 10.95-36.96 7.3-9.82 16.59-14.82 27.87-15 4.35 0 9.4 1.13 15.15 3.39 5.75 2.26 9.68 3.44 11.78 3.55 1.74-.11 5.76-1.34 12.06-3.69 6.3-2.35 11.51-3.39 15.64-3.12 11.96.65 21.62 5.34 28.98 14.07-10.43 6.31-15.54 15.11-15.33 26.4.22 8.7 3.58 16.09 10.08 22.18 6.5 6.09 14.24 9.68 23.22 10.77-2.39 7.4-5.43 15.01-9.12 22.84zM119.22 33.02c0-7.39 2.66-14.24 7.98-20.55 5.32-6.31 11.96-10.33 19.92-12.06.22 1.3.33 2.61.33 3.91 0 7.39-2.83 14.46-8.49 21.2-5.66 6.74-12.29 10.55-19.89 11.42-.11-1.3-.15-2.6-.15-3.92z"/>
    </svg>

    <!-- 2. Windows 11 Official Logo -->
    <svg v-else-if="deviceType === 'windows'" viewBox="0 0 24 24" fill="none" class="w-4 h-4 drop-shadow-[0_0_8px_rgba(0,164,239,0.5)]">
      <rect x="2.5" y="2.5" width="8.8" height="8.8" rx="1.2" fill="#00A4EF"/>
      <rect x="12.7" y="2.5" width="8.8" height="8.8" rx="1.2" fill="#00A4EF"/>
      <rect x="2.5" y="12.7" width="8.8" height="8.8" rx="1.2" fill="#00A4EF"/>
      <rect x="12.7" y="12.7" width="8.8" height="8.8" rx="1.2" fill="#00A4EF"/>
    </svg>

    <!-- 3. Android (Samsung / Xiaomi / Huawei / Pixel) -->
    <svg v-else-if="deviceType === 'android'" viewBox="0 0 24 24" fill="#3DDC84" class="w-4 h-4 drop-shadow-[0_0_8px_rgba(61,220,132,0.4)]">
      <path d="M6 18c0 .55.45 1 1 1h1v3.5c0 .83.67 1.5 1.5 1.5s1.5-.67 1.5-1.5V19h2v3.5c0 .83.67 1.5 1.5 1.5s1.5-.67 1.5-1.5V19h1c.55 0 1-.45 1-1V8H6v10zM3.5 8C2.67 8 2 8.67 2 9.5v6c0 .83.67 1.5 1.5 1.5S5 16.33 5 15.5v-6C5 8.67 4.33 8 3.5 8zm17 0c-.83 0-1.5.67-1.5 1.5v6c0 .83.67 1.5 1.5 1.5s1.5-.67 1.5-1.5v-6c0-.83-.67-1.5-1.5-1.5zm-4.97-4.84l1.3-1.3c.2-.2.2-.51 0-.71-.2-.2-.51-.2-.71 0l-1.48 1.48C13.68 2.23 12.63 2 11.5 2c-1.13 0-2.18.23-3.14.63L6.88 1.15c-.2-.2-.51-.2-.71 0-.2.2-.2.51 0 .71l1.3 1.3C5.78 4.34 4.67 6.01 4.5 8h14c-.17-1.99-1.28-3.66-2.97-4.84zM9 6c-.55 0-1-.45-1-1s.45-1 1-1 1 .45 1 1-.45 1-1 1zm5 0c-.55 0-1-.45-1-1s.45-1 1-1 1 .45 1 1-.45 1-1 1z"/>
    </svg>

    <!-- 4. Smart TV (TCL / Sony / MiTV / Hisense) -->
    <svg v-else-if="deviceType === 'tv'" viewBox="0 0 24 24" fill="none" class="w-4 h-4 drop-shadow-[0_0_8px_rgba(245,158,11,0.4)]">
      <rect x="2" y="3.5" width="20" height="13.5" rx="2.5" stroke="#F59E0B" stroke-width="1.8" fill="#F59E0B" fill-opacity="0.15"/>
      <line x1="8" y1="20.5" x2="16" y2="20.5" stroke="#F59E0B" stroke-width="2" stroke-linecap="round"/>
      <line x1="12" y1="17" x2="12" y2="20.5" stroke="#F59E0B" stroke-width="1.8"/>
      <circle cx="12" cy="10" r="1.5" fill="#FEF08A"/>
    </svg>

    <!-- 5. 蓝光机 / 影音播放器 (Sony BDP / 芝杜 Zidoo / Hi-Fi) -->
    <svg v-else-if="deviceType === 'bdp'" viewBox="0 0 24 24" fill="none" class="w-4 h-4 drop-shadow-[0_0_8px_rgba(167,139,250,0.4)]">
      <rect x="2.5" y="6" width="19" height="12" rx="2.5" stroke="#A78BFA" stroke-width="1.8" fill="#8B5CF6" fill-opacity="0.2"/>
      <circle cx="12" cy="12" r="3.5" stroke="#DDD6FE" stroke-width="1.5"/>
      <circle cx="12" cy="12" r="1.2" fill="#A78BFA"/>
      <circle cx="18" cy="9" r="0.9" fill="#34D399"/>
    </svg>

    <!-- 6. 飞牛 fnOS / NAS 存储 (Synology / QNAP) -->
    <svg v-else-if="deviceType === 'nas'" viewBox="0 0 24 24" fill="none" class="w-4 h-4 drop-shadow-[0_0_8px_rgba(96,165,250,0.4)]">
      <rect x="3" y="4" width="18" height="6.5" rx="2" stroke="#60A5FA" stroke-width="1.6" fill="#1E40AF" fill-opacity="0.25"/>
      <circle cx="6" cy="7.2" r="1" fill="#34D399"/>
      <circle cx="9" cy="7.2" r="1" fill="#60A5FA"/>
      <line x1="13" y1="7.2" x2="18.5" y2="7.2" stroke="#93C5FD" stroke-width="1.5" stroke-linecap="round"/>
      <rect x="3" y="13.5" width="18" height="6.5" rx="2" stroke="#60A5FA" stroke-width="1.6" fill="#1E40AF" fill-opacity="0.25"/>
      <circle cx="6" cy="16.7" r="1" fill="#34D399"/>
      <circle cx="9" cy="16.7" r="1" fill="#60A5FA"/>
      <line x1="13" y1="16.7" x2="18.5" y2="16.7" stroke="#93C5FD" stroke-width="1.5" stroke-linecap="round"/>
    </svg>

    <!-- 7. 爱快 iKuai 路由 -->
    <svg v-else-if="deviceType === 'ikuai'" viewBox="0 0 24 24" fill="none" class="w-4 h-4 drop-shadow-[0_0_8px_rgba(255,158,94,0.4)]">
      <circle cx="12" cy="12" r="9.5" fill="#FF8C00" fill-opacity="0.2" stroke="#FF9E5E" stroke-width="1.5"/>
      <path d="M7.5 12l2.5-4h4.5l-2.5 4 3.5 4h-4.5l-3.5-4z" fill="#FF9E5E"/>
      <circle cx="15.5" cy="8.5" r="1.3" fill="#38BDF8"/>
    </svg>

    <!-- 8. OpenWrt 官方网络流标 -->
    <svg v-else-if="deviceType === 'openwrt'" viewBox="0 0 24 24" fill="none" class="w-4 h-4 drop-shadow-[0_0_8px_rgba(0,210,255,0.4)]">
      <circle cx="12" cy="12" r="9.2" stroke="#00D2FF" stroke-width="1.8" stroke-dasharray="6 3"/>
      <circle cx="12" cy="12" r="4.2" fill="#00D2FF" fill-opacity="0.6"/>
      <circle cx="12" cy="12" r="1.8" fill="#FFFFFF"/>
    </svg>

    <!-- 9. MikroTik RouterOS 官方多边形网关标 -->
    <svg v-else-if="deviceType === 'ros'" viewBox="0 0 24 24" fill="none" class="w-4 h-4 drop-shadow-[0_0_8px_rgba(0,210,255,0.4)]">
      <rect x="3.5" y="3.5" width="4.5" height="4.5" rx="1" fill="#00D2FF"/>
      <rect x="9.75" y="3.5" width="4.5" height="4.5" rx="1" fill="#00D2FF"/>
      <rect x="16" y="3.5" width="4.5" height="4.5" rx="1" fill="#00D2FF"/>
      <rect x="3.5" y="9.75" width="4.5" height="4.5" rx="1" fill="#00D2FF"/>
      <rect x="9.75" y="9.75" width="4.5" height="4.5" rx="1" fill="#FFFFFF"/>
      <rect x="16" y="9.75" width="4.5" height="4.5" rx="1" fill="#00D2FF"/>
      <rect x="3.5" y="16" width="4.5" height="4.5" rx="1" fill="#00D2FF"/>
      <rect x="9.75" y="16" width="4.5" height="4.5" rx="1" fill="#00D2FF"/>
      <rect x="16" y="16" width="4.5" height="4.5" rx="1" fill="#00D2FF"/>
    </svg>

    <!-- 10. CubeX 零感网关 / 旁路本机 -->
    <svg v-else-if="deviceType === 'cubex'" viewBox="0 0 24 24" fill="none" class="w-4 h-4 drop-shadow-[0_0_8px_rgba(0,245,212,0.5)]">
      <path d="M12 2L3 7v10l9 5 9-5V7l-9-5z" fill="#00F5D4" fill-opacity="0.2" stroke="#00F5D4" stroke-width="1.6"/>
      <path d="M12 22V12m0 0L3 7m9 5l9-5" stroke="#00D2FF" stroke-width="1.4"/>
      <circle cx="12" cy="12" r="2" fill="#FEF08A"/>
    </svg>

    <!-- 11. Linux / PVE 节点 -->
    <svg v-else-if="deviceType === 'linux'" viewBox="0 0 24 24" fill="none" class="w-4 h-4 drop-shadow-[0_0_8px_rgba(148,163,184,0.4)]">
      <rect x="3" y="3" width="18" height="18" rx="3.5" stroke="#94A3B8" stroke-width="1.6" fill="#1E293B" fill-opacity="0.4"/>
      <path d="M7 8h10M7 12h10M7 16h6" stroke="#E2E8F0" stroke-width="1.8" stroke-linecap="round"/>
      <circle cx="17" cy="16" r="1.5" fill="#38BDF8"/>
    </svg>

    <!-- 12. 游戏机 (PlayStation / Switch / Xbox) -->
    <svg v-else-if="deviceType === 'gaming'" viewBox="0 0 24 24" fill="none" stroke="#F43F5E" stroke-width="1.8" class="w-4 h-4 drop-shadow-[0_0_8px_rgba(244,63,94,0.4)]">
      <rect x="3" y="6" width="18" height="12" rx="4" fill="#881337" fill-opacity="0.3"/>
      <path d="M7 12h4M9 10v4" stroke-linecap="round"/>
      <circle cx="16" cy="10" r="1" fill="#F43F5E"/>
      <circle cx="18" cy="12" r="1" fill="#F43F5E"/>
    </svg>

    <!-- 13. 通用硬件设备 Fallback -->
    <svg v-else viewBox="0 0 24 24" fill="none" stroke="#38BDF8" stroke-width="1.8" class="w-4 h-4">
      <rect x="3" y="4" width="18" height="12" rx="2" fill="#0369A1" fill-opacity="0.2"/>
      <line x1="8" y1="20" x2="16" y2="20" stroke-linecap="round"/>
      <line x1="12" y1="16" x2="12" y2="20"/>
    </svg>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  device: {
    type: Object,
    default: () => ({})
  }
})

const deviceType = computed(() => {
  const dev = props.device || {}
  const name = (dev.name || '').toLowerCase()
  const hostname = (dev.hostname || '').toLowerCase()
  const ip = dev.ip || ''

  // 1. CubeX 旁路本机
  if (name.includes('旁路') || name.includes('cubex') || ip === '10.0.0.7') {
    return 'cubex'
  }

  // 2. MikroTik RouterOS (仅根据设备名称或主路由 IP，绝不检查 dev.source！)
  if (name.includes('主路由') || name.includes('ros') || name.includes('routeros') || name.includes('mikrotik') || ip === '10.0.0.1') {
    return 'ros'
  }

  // 3. 爱快 iKuai
  if (name.includes('ikuai') || name.includes('爱快') || hostname.includes('ikuai')) {
    return 'ikuai'
  }

  // 4. OpenWrt
  if (name.includes('openwrt') || name.includes('immortalwrt') || hostname.includes('openwrt')) {
    return 'openwrt'
  }

  // 5. 飞牛 fnOS / NAS 存储
  if (name.includes('fnos') || name.includes('飞牛') || name.includes('nas') || name.includes('synology') || name.includes('群晖') || hostname.includes('fnos') || hostname.includes('nas')) {
    return 'nas'
  }

  // 6. 智能电视 TV (TCL, Sony TV, MiTV, Apple TV, 电视)
  if (name.includes('tv') || name.includes('电视') || name.includes('tcl') || name.includes('hisense') || name.includes('appletv') || hostname.includes('tv') || hostname.includes('tcl')) {
    return 'tv'
  }

  // 7. 蓝光机 / 影音播放器 (BDP, Sony BDP, 芝杜 Zidoo)
  if (name.includes('bdp') || name.includes('蓝光') || name.includes('zidoo') || name.includes('芝杜') || name.includes('pioneer') || name.includes('x800')) {
    return 'bdp'
  }

  // 8. Apple (iPhone / iPad / Mac / MacBook)
  if (name.includes('iphone') || name.includes('ipad') || name.includes('mac') || name.includes('apple') || hostname.includes('iphone') || hostname.includes('ipad') || hostname.includes('mac') || hostname.includes('apple')) {
    return 'apple'
  }

  // 9. Windows PC / Laptop
  if (name.includes('win') || name.includes('windows') || name.includes('pc') || name.includes('workstation') || hostname.includes('desktop') || hostname.includes('laptop') || hostname.includes('win')) {
    return 'windows'
  }

  // 10. Android / Samsung / 华为 / 小米
  if (name.includes('samsung') || name.includes('galaxy') || name.includes('s24') || name.includes('s23') || name.includes('s22') ||
      name.includes('android') || name.includes('xiaomi') || name.includes('huawei') || name.includes('honor') || name.includes('oppo') || name.includes('vivo') || name.includes('ultra') ||
      hostname.includes('samsung') || hostname.includes('galaxy') || hostname.includes('android') || hostname.includes('xiaomi') || hostname.includes('huawei')) {
    return 'android'
  }

  // 11. 游戏机 (PlayStation / Switch / Xbox)
  if (name.includes('ps5') || name.includes('ps4') || name.includes('playstation') || name.includes('switch') || name.includes('xbox')) {
    return 'gaming'
  }

  // 12. Linux / PVE / 服务器 (Lucky, 10.0.0.9, 10.0.0.5)
  if (name.includes('linux') || name.includes('pve') || name.includes('proxmox') || name.includes('lucky') || name.includes('server') || ip === '10.0.0.9' || ip === '10.0.0.5') {
    return 'linux'
  }

  return 'generic'
})
</script>
