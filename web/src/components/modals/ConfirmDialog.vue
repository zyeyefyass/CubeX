<template>
  <teleport to="body">
    <div
      v-if="modelValue"
      class="fixed inset-0 z-[10500] flex items-center justify-center p-4 bg-black/60 animate-in fade-in"
      @click.self="handleCancel"
    >
      <div
        class="w-full max-w-md rounded-3xl cbx-modal-shell p-6 space-y-5 relative overflow-hidden"
        :style="{ borderColor: variantBorderColor }"
      >
        <!-- Header -->
        <div class="flex items-start gap-3.5 relative z-10">
          <div
            class="w-11 h-11 rounded-2xl flex items-center justify-center text-xl shrink-0"
            :class="variantBadgeClass"
          >
            {{ variantIcon }}
          </div>
          <div class="min-w-0 flex-1">
            <h3 class="text-base font-black cbx-text-main tracking-wide">{{ title || '操作确认' }}</h3>
            <p class="text-xs cbx-text-sub mt-1.5 leading-relaxed whitespace-pre-line break-words font-medium">{{ message }}</p>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex items-center justify-end gap-2.5 pt-3 border-t relative z-10" style="border-color: var(--cbx-border-sub);">
          <button
            v-if="showCancel"
            type="button"
            @click="handleCancel"
            class="cbx-btn-ghost px-4 py-2 rounded-xl text-xs font-bold transition active:scale-95 cursor-pointer"
          >
            {{ cancelText || '取消' }}
          </button>
          <button
            type="button"
            @click="handleConfirm"
            class="px-5 py-2 rounded-xl text-white text-xs font-extrabold transition active:scale-95 cursor-pointer shadow-sm"
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
  variant: { type: String, default: 'warning' },
  showCancel: { type: Boolean, default: true }
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel'])

const variantIcon = computed(() => {
  if (props.variant === 'danger') return '⚠️'
  if (props.variant === 'info') return '💡'
  return '🛡️'
})

const variantBorderColor = computed(() => {
  if (props.variant === 'danger') return 'var(--cbx-danger-border)'
  if (props.variant === 'info') return 'var(--cbx-accent-border)'
  return 'var(--cbx-warn-border)'
})

const variantBadgeClass = computed(() => {
  if (props.variant === 'danger') return 'cbx-badge-danger'
  if (props.variant === 'info') return 'cbx-badge-info'
  return 'cbx-badge-warn'
})

const variantButtonClass = computed(() => {
  if (props.variant === 'danger') return 'bg-rose-600 hover:bg-rose-700'
  if (props.variant === 'info') return 'bg-sky-600 hover:bg-sky-700'
  return 'bg-amber-600 hover:bg-amber-700'
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
