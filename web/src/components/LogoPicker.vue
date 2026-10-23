<template>
  <div class="relative" ref="pickerRef">
    <!-- Trigger Button -->
    <button
      type="button"
      @click="isOpen = !isOpen"
      class="w-full px-3 py-2.5 rounded-xl bg-slate-950 border border-slate-800 hover:border-cyan-500/50 focus:border-cyan-400 outline-none text-white font-mono flex items-center justify-between gap-2 transition select-none"
    >
      <div class="flex items-center gap-2.5 truncate">
        <BrandLogo :icon="modelValue" size="w-5 h-5" />
        <span class="text-xs font-semibold text-slate-200 truncate">{{ currentItem.name }}</span>
      </div>
      <div class="flex items-center gap-1.5 text-slate-400 shrink-0">
        <span class="text-[10px] text-cyan-400 uppercase font-mono">官方LOGO</span>
        <svg class="w-3.5 h-3.5 transition-transform duration-200" :class="{ 'rotate-180': isOpen }" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd"/>
        </svg>
      </div>
    </button>

    <!-- Dropdown / Popover -->
    <div
      v-if="isOpen"
      class="absolute left-0 right-0 mt-1.5 z-50 p-2.5 rounded-2xl bg-[#00172e] border border-cyan-500/50 shadow-2xl backdrop-blur-xl animate-in fade-in zoom-in-95 space-y-2"
    >
      <!-- Search Box -->
      <div class="relative">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="搜索官方应用标识 (例: Disney, Netflix, OpenAI)..."
          class="w-full px-3 py-1.5 pl-8 rounded-lg bg-slate-900 border border-slate-800 focus:border-cyan-400 outline-none text-xs text-white placeholder-slate-500 font-mono"
          @click.stop
        />
        <svg class="w-3.5 h-3.5 text-slate-400 absolute left-2.5 top-2.5" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd"/>
        </svg>
      </div>

      <!-- Icon Options List (Grid with Official Vector SVGs) -->
      <div class="max-h-56 overflow-y-auto space-y-1 pr-1 custom-scroll">
        <button
          v-for="item in filteredOptions"
          :key="item.id"
          type="button"
          @click="selectItem(item)"
          :class="[
            'w-full flex items-center justify-between p-2 rounded-xl transition text-left select-none text-xs',
            modelValue === item.id
              ? 'bg-cyan-950/80 text-cyan-300 border border-cyan-500/50 font-bold'
              : 'hover:bg-slate-900/90 text-slate-300 hover:text-white border border-transparent'
          ]"
        >
          <div class="flex items-center gap-2.5 truncate">
            <BrandLogo :icon="item.id" size="w-5 h-5" />
            <span class="truncate">{{ item.name }}</span>
          </div>
          <span v-if="modelValue === item.id" class="text-cyan-400 font-bold text-xs">✓</span>
        </button>
        <div v-if="filteredOptions.length === 0" class="text-center py-4 text-xs text-slate-500">
          未找到匹配的官方应用
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import BrandLogo from './BrandLogo.vue'

const props = defineProps({
  modelValue: {
    type: String,
    default: 'default'
  }
})

const emit = defineEmits(['update:modelValue', 'change'])

const isOpen = ref(false)
const searchQuery = ref('')
const pickerRef = ref(null)

const options = [
  { id: 'antigravity', name: 'Google Antigravity / 官方全息极光' },
  { id: 'gemini', name: 'Google Gemini / 谷歌人工智能' },
  { id: 'openai', name: 'OpenAI / ChatGPT' },
  { id: 'claude', name: 'Claude / Anthropic' },
  { id: 'youtube', name: 'YouTube / 4K/8K 视频流媒体' },
  { id: 'netflix', name: 'Netflix / 网飞 3D 丝带' },
  { id: 'disney', name: 'Disney+ / 迪士尼流媒体' },
  { id: 'amazon', name: 'Amazon Prime Video / 亚马逊影视' },
  { id: 'hbo', name: 'HBO Max / 华纳超清剧集' },
  { id: 'twitch', name: 'Twitch / 游戏电竞直播' },
  { id: 'bahamut', name: '巴哈姆特 / 动画疯番剧' },
  { id: 'emby', name: 'Emby / 私有家庭影音中心' },
  { id: 'spotify', name: 'Spotify / 高保真无损音乐' },
  { id: 'bilibili', name: 'Bilibili / 港澳台番剧' },
  { id: 'netease', name: '网易云音乐 / 黑胶无损音频' },
  { id: 'google', name: 'Google / 搜索与 Play 商店' },
  { id: 'apple', name: 'Apple / 苹果专有服务' },
  { id: 'microsoft', name: 'Microsoft / 微软 Office 365' },
  { id: 'onedrive', name: 'OneDrive / 微软云盘' },
  { id: 'github', name: 'GitHub / 开源协同与 Docker' },
  { id: 'gitlab', name: 'GitLab / 源码代码仓库' },
  { id: 'telegram', name: 'Telegram / 极速消息通信' },
  { id: 'twitter', name: 'Twitter (X) / 全球社交媒体' },
  { id: 'discord', name: 'Discord / 极客社区与语音' },
  { id: 'whatsapp', name: 'WhatsApp / 国际即时通讯' },
  { id: 'line', name: 'LINE / 亚洲即时通讯' },
  { id: 'signal', name: 'Signal / 端到端隐私加密通讯' },
  { id: 'tiktok', name: 'TikTok / 国际短视频专线' },
  { id: 'reddit', name: 'Reddit / 海外极客讨论社区' },
  { id: 'steam', name: 'Steam / 游戏主机与创意工坊' },
  { id: 'epicgames', name: 'Epic Games / 虚幻引擎商城' },
  { id: 'playstation', name: 'PlayStation (PSN) / 索尼主机' },
  { id: 'nintendo', name: 'Nintendo Switch / 任天堂' },
  { id: 'blizzard', name: 'Blizzard / 暴雪战网' },
  { id: 'ea', name: 'EA / 艺电 App 联机加速' },
  { id: 'tencent', name: '腾讯生态 / 微信 / QQ 直连' },
  { id: 'alibaba', name: '阿里生态 / 淘宝 / 支付宝' },
  { id: 'baidu', name: '百度服务 / 百度网盘加速' },
  { id: 'binance', name: 'Binance / 币安加密货币' },
  { id: 'okx', name: 'OKX / 欧易 Web3 钱包' },
  { id: 'paypal', name: 'PayPal / 全球安全支付' },
  { id: 'zoom', name: 'Zoom / 远程高清视频会议' },
  { id: 'notion', name: 'Notion / 知识协同笔记' },
  { id: 'speedtest', name: 'Speedtest / 测速隔离' },
  { id: 'cn', name: '国内直连 / 真实公网物理直出' },
  { id: 'cn_not', name: '国外加速 / 境外极速穿透' },
  { id: 'gfw', name: 'GFW 拦截 / 敏感受限专线' },
  { id: 'privacy', name: '局域网私网 / 反追踪保护' },
  { id: 'direct', name: 'DIRECT / 默认物理直连' },
  { id: 'fish', name: '漏网之鱼 / 兜底出站' },
  { id: 'default', name: '默认应用罗盘' }
]

const currentItem = computed(() => {
  return options.find(o => o.id === props.modelValue) || { id: props.modelValue || 'default', name: props.modelValue || '默认应用罗盘' }
})

const filteredOptions = computed(() => {
  if (!searchQuery.value.trim()) return options
  const q = searchQuery.value.toLowerCase().trim()
  return options.filter(o => o.name.toLowerCase().includes(q) || o.id.includes(q))
})

function selectItem(item) {
  emit('update:modelValue', item.id)
  emit('change', item)
  isOpen.value = false
  searchQuery.value = ''
}

function handleClickOutside(e) {
  if (pickerRef.value && !pickerRef.value.contains(e.target)) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>
