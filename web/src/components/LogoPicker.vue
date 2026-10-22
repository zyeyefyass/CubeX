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
  { id: 'antigravity', name: 'Antigravity / Google 官方全息矢量' },
  { id: 'gemini', name: 'Gemini / 谷歌人工智能' },
  { id: 'openai', name: 'OpenAI / ChatGPT' },
  { id: 'claude', name: 'Claude / Anthropic' },
  { id: 'youtube', name: 'YouTube / 视频流媒体' },
  { id: 'netflix', name: 'Netflix / 网飞 3D 丝带' },
  { id: 'disney', name: 'Disney+ / 迪士尼流媒体' },
  { id: 'google', name: 'Google / 搜索与 Play' },
  { id: 'github', name: 'GitHub / 开源与 Docker' },
  { id: 'telegram', name: 'Telegram / 消息通信' },
  { id: 'twitter', name: 'Twitter (X) / 社交媒体' },
  { id: 'spotify', name: 'Spotify / 高保真音乐' },
  { id: 'steam', name: 'Steam / 游戏主机加速' },
  { id: 'epicgames', name: 'Epic Games / 游戏平台' },
  { id: 'playstation', name: 'PlayStation (PSN) / 主机' },
  { id: 'nintendo', name: 'Nintendo Switch / 任天堂' },
  { id: 'bahamut', name: '巴哈姆特 / 动画疯' },
  { id: 'emby', name: 'Emby / 私有影视中心' },
  { id: 'tiktok', name: 'TikTok / 国际短视频' },
  { id: 'bilibili', name: 'Bilibili / 港澳台番剧' },
  { id: 'discord', name: 'Discord / 社区语音聊天' },
  { id: 'amazon', name: 'Amazon / Prime Video' },
  { id: 'paypal', name: 'PayPal / 国际安全支付' },
  { id: 'speedtest', name: 'Speedtest / 测速隔离' },
  { id: 'reddit', name: 'Reddit / 海外讨论社区' },
  { id: 'apple', name: 'Apple / 苹果专有服务' },
  { id: 'microsoft', name: 'Microsoft / 微软 365' },
  { id: 'onedrive', name: 'OneDrive / 微软云盘' },
  { id: 'zoom', name: 'Zoom / 远程视频会议' },
  { id: 'notion', name: 'Notion / 知识协同笔记' },
  { id: 'binance', name: 'Binance / 加密货币交易' },
  { id: 'direct', name: 'DIRECT / 国内直连绕过' },
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
