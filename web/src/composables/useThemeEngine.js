import { ref, computed, watch, onMounted } from 'vue'

const themes = ref([])
const activeThemeID = ref('twilight-default')
const playbackMode = ref('auto')
const isImporting = ref(false)
const importError = ref('')
const isLoading = ref(false)

const tokenKeyRegex = /^--cbx-[a-z0-9_-]+$/
const tokenColorRegex = /^(#[0-9a-fA-F]{3,8}|rgba?\([0-9.,\s%]+\)|hsla?\([0-9.,\s%deg]+\))$/

function applyTokens(tokens) {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  if (!tokens || typeof tokens !== 'object') {
    // Reset to defaults
    root.style.removeProperty('--cbx-theme-accent')
    root.style.removeProperty('--cbx-theme-accent-glow')
    root.style.removeProperty('--cbx-theme-card-bg')
    root.style.removeProperty('--cbx-theme-card-border')
    return
  }

  // Double-ended whitelist validation against injection
  for (const [key, val] of Object.entries(tokens)) {
    if (!tokenKeyRegex.test(key)) continue
    const cleanVal = String(val).trim()
    const lower = cleanVal.toLowerCase()
    if (lower.includes('url(') || lower.includes(';') || lower.includes('expression') || lower.includes('<') || lower.includes('>')) {
      console.warn(`[ThemeEngine] Blocked unsafe CSS token: ${key} = ${val}`)
      continue
    }
    if (!tokenColorRegex.test(cleanVal)) {
      console.warn(`[ThemeEngine] Blocked invalid color CSS token: ${key} = ${val}`)
      continue
    }
    root.style.setProperty(key, cleanVal)
  }
}

export function useThemeEngine() {
  const activeTheme = computed(() => {
    return themes.value.find(t => t.id === activeThemeID.value) || {
      id: 'twilight-default',
      name: '暮光冰晶 (默认)',
      type: 'vector',
      tier: 'builtin',
      tokens: {}
    }
  })

  async function fetchThemes() {
    isLoading.value = true
    try {
      const res = await fetch('/api/themes')
      if (res.ok) {
        const data = await res.json()
        if (data && data.themes) {
          themes.value = data.themes
          if (data.active_theme_id) activeThemeID.value = data.active_theme_id
          if (data.playback_mode) playbackMode.value = data.playback_mode
          applyTokens(activeTheme.value.tokens)
        }
      }
    } catch (e) {
      console.warn('[ThemeEngine] Failed to load themes:', e)
    } finally {
      isLoading.value = false
    }
  }

  async function activateTheme(id, mode = null) {
    try {
      const payload = { id }
      if (mode) payload.playback_mode = mode
      const res = await fetch('/api/themes/activate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      })
      if (res.ok) {
        activeThemeID.value = id
        if (mode) playbackMode.value = mode
        applyTokens(activeTheme.value.tokens)
        // Refresh theme list
        await fetchThemes()
        return true
      }
    } catch (e) {
      console.error('[ThemeEngine] Failed to activate theme:', e)
    }
    return false
  }

  async function setPlaybackMode(mode) {
    if (!['auto', 'always', 'poster'].includes(mode)) return
    playbackMode.value = mode
    await activateTheme(activeThemeID.value, mode)
  }

  async function importThemeFile(file) {
    if (!file) return false
    isImporting.value = true
    importError.value = ''
    try {
      const formData = new FormData()
      formData.append('theme', file)

      const res = await fetch('/api/themes/import', {
        method: 'POST',
        body: formData
      })
      const result = await res.json()
      if (res.ok && result.ok) {
        await fetchThemes()
        if (result.theme?.id) {
          activeThemeID.value = result.theme.id
          applyTokens(result.theme.tokens)
        }
        return true
      } else {
        importError.value = result.error || '主题导入失败，请检查文件格式'
      }
    } catch (e) {
      importError.value = e.message || '网络异常，导入失败'
    } finally {
      isImporting.value = false
    }
    return false
  }

  async function deleteTheme(id) {
    if (!id || id === 'twilight-default' || id === 'legacy-wallpapers') return false
    try {
      const res = await fetch(`/api/themes/${id}`, { method: 'DELETE' })
      if (res.ok) {
        await fetchThemes()
        return true
      }
    } catch (e) {
      console.error('[ThemeEngine] Failed to delete theme:', e)
    }
    return false
  }

  watch(activeTheme, (newTheme) => {
    if (newTheme) {
      applyTokens(newTheme.tokens)
    }
  })

  return {
    themes,
    activeThemeID,
    playbackMode,
    activeTheme,
    isImporting,
    importError,
    isLoading,
    fetchThemes,
    activateTheme,
    setPlaybackMode,
    importThemeFile,
    deleteTheme
  }
}
