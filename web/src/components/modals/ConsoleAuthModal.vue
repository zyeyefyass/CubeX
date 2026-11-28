<template>
  <teleport to="body">
    <!-- Mode 1: Full-screen Auth Gate -->
    <div
      v-if="mode === 'login' && visible"
      class="fixed inset-0 z-[11000] flex items-center justify-center p-4 bg-black/80 animate-in fade-in"
    >
      <div class="w-full max-w-md rounded-3xl cbx-modal-shell p-6 sm:p-7 space-y-6 relative overflow-hidden">
        <div class="flex flex-col items-center text-center space-y-3">
          <img src="/logo-cubex.svg" class="w-14 h-14 object-contain" alt="CubeX" />
          <div>
            <h2 class="text-lg font-black cbx-text-main tracking-wide">CubeX 控制台访问锁</h2>
            <p class="text-xs cbx-text-muted mt-1">当前控制台已启用口令保护，请输入访问密码解锁</p>
          </div>
        </div>

        <form @submit.prevent="submitLogin" class="space-y-4">
          <div>
            <label class="block text-xs font-semibold cbx-text-sub mb-1.5">访问密码</label>
            <div class="relative">
              <input
                v-model="loginPassword"
                :type="showPass ? 'text' : 'password'"
                autocomplete="current-password"
                autofocus
                placeholder="请输入控制台访问密码"
                class="w-full px-3.5 py-2.5 pr-14 rounded-xl text-sm font-mono"
              />
              <button
                type="button"
                @click="showPass = !showPass"
                class="absolute right-2.5 top-1/2 -translate-y-1/2 text-xs cbx-text-muted hover:cbx-text-main px-2 py-1 cursor-pointer font-semibold"
              >
                {{ showPass ? '隐藏' : '显示' }}
              </button>
            </div>
          </div>

          <div v-if="errorMsg" class="p-2.5 rounded-xl cbx-badge-danger text-xs font-medium">
            {{ errorMsg }}
          </div>

          <button
            type="submit"
            :disabled="submitting || !loginPassword"
            class="w-full py-2.5 rounded-xl bg-sky-600 hover:bg-sky-700 text-white font-extrabold text-sm transition active:scale-95 cursor-pointer shadow-md disabled:opacity-50"
          >
            {{ submitting ? '正在核验身份...' : '🔓 解锁进入控制台' }}
          </button>
        </form>
      </div>
    </div>

    <!-- Mode 2: Console Lock Management Modal -->
    <div
      v-else-if="mode === 'manage' && visible"
      class="fixed inset-0 z-[10400] flex items-center justify-center p-4 bg-black/70 animate-in fade-in"
      @click.self="$emit('close')"
    >
      <div class="w-full max-w-md rounded-3xl cbx-modal-shell p-6 space-y-5 relative overflow-hidden">
        <div class="flex items-center justify-between border-b pb-3.5" style="border-color: var(--cbx-border-sub);">
          <div class="flex items-center gap-2.5">
            <span class="text-xl">🔐</span>
            <div>
              <h3 class="text-base font-bold cbx-text-main">控制台访问安全锁</h3>
              <p class="text-xs cbx-text-muted mt-0.5">
                {{ authEnabled ? '已启用口令锁定（支持随时修改或关闭）' : '默认免密直连 · 可按需设定局域网访问密码' }}
              </p>
            </div>
          </div>
          <button
            type="button"
            @click="$emit('close')"
            aria-label="关闭弹窗"
            class="cbx-btn-ghost min-w-[36px] min-h-[36px] p-1.5 rounded-xl transition cursor-pointer flex items-center justify-center cbx-touch-extend"
          >✕</button>
        </div>

        <form @submit.prevent="submitPasswordUpdate" class="space-y-3.5">
          <div v-if="authEnabled">
            <label class="block text-xs font-semibold cbx-text-sub mb-1">当前密码</label>
            <input
              v-model="currentPassword"
              type="password"
              autocomplete="current-password"
              placeholder="输入当前访问密码"
              class="w-full px-3.5 py-2 rounded-xl text-xs font-mono"
            />
          </div>

          <div>
            <label class="block text-xs font-semibold cbx-text-sub mb-1">
              {{ authEnabled ? '新密码（留空则关闭访问锁）' : '设置控制台访问密码（至少 4 位）' }}
            </label>
            <input
              v-model="newPassword"
              type="password"
              autocomplete="new-password"
              :placeholder="authEnabled ? '输入新密码，或留空关闭密码保护' : '请输入至少 4 位访问密码'"
              class="w-full px-3.5 py-2 rounded-xl text-xs font-mono"
            />
          </div>

          <div v-if="errorMsg" class="p-2.5 rounded-xl cbx-badge-danger text-xs font-medium">
            {{ errorMsg }}
          </div>

          <div class="flex flex-wrap items-center justify-between gap-2 pt-2 border-t" style="border-color: var(--cbx-border-sub);">
            <button
              v-if="authEnabled"
              type="button"
              @click="lockNow"
              :disabled="submitting"
              class="px-3.5 py-2 rounded-xl cbx-badge-warn text-xs font-bold transition cursor-pointer"
            >
              🔒 立即锁定
            </button>
            <div class="flex items-center gap-2 ml-auto">
              <button
                type="button"
                @click="$emit('close')"
                class="cbx-btn-ghost px-4 py-2 rounded-xl text-xs font-bold transition cursor-pointer"
              >取消</button>
              <button
                type="submit"
                :disabled="submitting"
                class="px-4 py-2 rounded-xl bg-sky-600 hover:bg-sky-700 text-white text-xs font-extrabold transition cursor-pointer shadow-sm disabled:opacity-50"
              >
                {{ submitting ? '保存中...' : (authEnabled && !newPassword ? '关闭密码锁' : '保存并生效') }}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  </teleport>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  mode: { type: String, default: 'login' },
  visible: { type: Boolean, default: false },
  authEnabled: { type: Boolean, default: false }
})

const emit = defineEmits(['unlocked', 'updated', 'locked', 'close'])

const loginPassword = ref('')
const currentPassword = ref('')
const newPassword = ref('')
const showPass = ref(false)
const submitting = ref(false)
const errorMsg = ref('')

watch(() => props.visible, (val) => {
  if (val) {
    errorMsg.value = ''
    loginPassword.value = ''
    currentPassword.value = ''
    newPassword.value = ''
  }
})

async function submitLogin() {
  if (!loginPassword.value || submitting.value) return
  submitting.value = true
  errorMsg.value = ''
  try {
    const fn = window.__cubexNativeFetch || fetch
    const res = await fn('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: loginPassword.value })
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok || data.ok === false) {
      errorMsg.value = data.error || '访问密码验证失败'
      return
    }
    loginPassword.value = ''
    emit('unlocked')
  } catch (e) {
    errorMsg.value = '网络请求失败: ' + (e.message || '未知错误')
  } finally {
    submitting.value = false
  }
}

async function submitPasswordUpdate() {
  if (submitting.value) return
  errorMsg.value = ''
  if (!props.authEnabled && newPassword.value.trim().length < 4) {
    errorMsg.value = '首次开启访问锁密码长度不能少于 4 位'
    return
  }
  if (props.authEnabled && newPassword.value.trim() && newPassword.value.trim().length < 4) {
    errorMsg.value = '新密码长度不能少于 4 位（如需关闭请完全留空）'
    return
  }
  submitting.value = true
  try {
    const fn = window.__cubexNativeFetch || fetch
    const res = await fn('/api/v1/auth/password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        current_password: currentPassword.value,
        new_password: newPassword.value
      })
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok || data.ok === false) {
      errorMsg.value = data.error || '密码更新失败'
      return
    }
    emit('updated', Boolean(data.enabled))
    emit('close')
  } catch (e) {
    errorMsg.value = '请求异常: ' + (e.message || '未知错误')
  } finally {
    submitting.value = false
  }
}

async function lockNow() {
  if (submitting.value) return
  submitting.value = true
  try {
    const fn = window.__cubexNativeFetch || fetch
    await fn('/api/v1/auth/logout', { method: 'POST' })
    emit('locked')
    emit('close')
  } catch (e) {
    errorMsg.value = '锁定失败: ' + (e.message || '未知错误')
  } finally {
    submitting.value = false
  }
}
</script>
