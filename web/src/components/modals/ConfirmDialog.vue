<template>
  <teleport to="body">
    <div
      v-if="modelValue"
      class="fixed inset-0 z-[10500] flex items-center justify-center p-4 bg-black/75 backdrop-blur-xl animate-in fade-in"
      @click.self="handleCancel"
    >
      <div
        class="w-full max-w-md rounded-3xl macos-glass-card shadow-2xl p-6 space-y-5 text-slate-200 border relative overflow-hidden"
        :class="variantBorderClass"
      >
        <div
          class="absolute -right-20 -top-20 w-56 h-56 rounded-full blur-3xl pointer-events-none"
          :class="variantGlowClass"
        ></div>

        <!-- Header -->
        <div class="flex items-start gap-3.5 relative z-10">
          <div
            class="w-11 h-11 rounded-2xl flex items-center justify-center text-xl shrink-0 border shadow-inner"
            :class="variantBadgeClass"
          >
            {{ variantIcon }}
          </div>
          <div class="min-w-0 flex-1">
            <h3 class="text-base font-bold text-white tracking-wide">{{ title || '操作确认' }}</h3>
            <p class="text-xs text-slate-300 mt-1.5 leading-relaxed whitespace-pre-line break-words">{{ message }}</p>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex items-center justify-end gap-2.5 pt-2 border-t border-white/10 relative z-10">
          <button
            v-if="showCancel"
            type="button"
            @click="handleCancel"
            class="px-4 py-2 rounded-xl bg-white/10 hover:bg-white/20 border border-white/20 text-slate-200 hover:text-white text-xs font-bold transition active:scale-95 cursor-pointer"
          >
            {{ cancelText || '取消' }}
          </button>
          <button
            type="button"
            @click="handleConfirm"
            class="px-5 py-2 rounded-xl text-white text-xs font-extrabold transition active:scale-95 cursor-pointer shadow-lg"
            :class="variantButtonClass"
          >
            {{ confirmText || '确认执行' }}
          </button>
        </div>
      </div>
    </div>
  </teleport>
</template>

<script setup>
import { computed, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '操作确认' },
  message: { type: String, default: '' },
  confirmText: { type: String, default: '确认执行' },
  cancelText: { type: String, default: '取消' },
  variant: { type: String, default: 'warning' }, // 'danger' | 'warning' | 'info'
  showCancel: { type: Boolean, default: true }
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel'])

const variantIcon = computed(() => {
  if (props.variant === 'danger') return '⚠️'
  if (props.variant === 'info') return '💡'
  return '🛡️'
})

const variantBorderClass = computed(() => {
  if (props.variant === 'danger') return 'border-rose-500/40'
  if (props.variant === 'info') return 'border-cyan-500/40'
  return 'border-amber-500/40'
})

const variantGlowClass = computed(() => {
  if (props.variant === 'danger') return 'bg-rose-500/15'
  if (props.variant === 'info') return 'bg-cyan-500/15'
  return 'bg-amber-500/15'
})

const variantBadgeClass = computed(() => {
  if (props.variant === 'danger') return 'bg-rose-500/20 border-rose-400/40 text-rose-300'
  if (props.variant === 'info') return 'bg-cyan-500/20 border-cyan-400/40 text-cyan-300'
  return 'bg-amber-500/20 border-amber-400/40 text-amber-300'
})

const variantButtonClass = computed(() => {
  if (props.variant === 'danger') return 'bg-gradient-to-r from-rose-600 to-red-500 hover:from-rose-500 hover:to-red-400 shadow-rose-500/25'
  if (props.variant === 'info') return 'bg-gradient-to-r from-sky-600 to-cyan-500 hover:from-sky-500 hover:to-cyan-400 shadow-cyan-500/25'
  return 'bg-gradient-to-r from-amber-600 to-orange-500 hover:from-amber-500 hover:to-orange-400 shadow-amber-500/25'
})

function handleConfirm() {
  emit('confirm')
  emit('update:modelValue', false)
}

function handleCancel() {
  emit('cancel')
  emit('update:modelValue', false)
}

function onKeydown(e) {
  if (!props.modelValue) return
  if (e.key === 'Escape') {
    handleCancel()
  }
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>
