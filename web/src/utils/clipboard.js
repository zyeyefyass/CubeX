// 统一剪贴板复制工具（优先使用安全上下文 Clipboard API，HTTP 局域网管理台自动回退 textarea）
export async function copyTextToClipboard(text) {
  if (!text) return false

  if (typeof navigator !== 'undefined' && navigator?.clipboard && typeof window !== 'undefined' && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // fallback below
    }
  }

  try {
    const el = document.createElement('textarea')
    el.value = text
    el.setAttribute('readonly', '')
    el.style.position = 'fixed'
    el.style.left = '-9999px'
    el.style.top = '-9999px'
    el.style.opacity = '0'
    document.body.appendChild(el)
    el.focus()
    el.select()
    el.setSelectionRange(0, el.value.length)
    const copied = document.execCommand('copy')
    document.body.removeChild(el)
    return !!copied
  } catch {
    return false
  }
}
