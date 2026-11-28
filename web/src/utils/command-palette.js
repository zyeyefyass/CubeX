// web/src/utils/command-palette.js

// Ctrl/Cmd + K 极客指令舱键盘状态机（支持上下键循环、关键词过滤、Enter 执行）
export class GeekCommandController {
  constructor(commands = []) {
    this.allCommands = commands // [{ id, title, shortcut, keywords, action, icon }]
    this.filtered = [...commands]
    this.selectedIndex = 0
    this.isOpen = false
  }

  setCommands(commands) {
    this.allCommands = commands
    this.filtered = [...commands]
    this.selectedIndex = 0
  }

  filter(query) {
    const q = (query || '').trim().toLowerCase()
    if (!q) {
      this.filtered = [...this.allCommands]
    } else {
      this.filtered = this.allCommands.filter(
        (cmd) =>
          cmd.title.toLowerCase().includes(q) ||
          (cmd.keywords && cmd.keywords.toLowerCase().includes(q))
      )
    }
    this.selectedIndex = 0
    return this.filtered
  }

  handleKeyDown(e, closeFn) {
    if (!this.isOpen) return
    if (e.key === 'Escape') {
      e.preventDefault()
      closeFn()
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      if (this.filtered.length > 0) {
        this.selectedIndex = (this.selectedIndex + 1) % this.filtered.length
      }
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      if (this.filtered.length > 0) {
        this.selectedIndex = (this.selectedIndex - 1 + this.filtered.length) % this.filtered.length
      }
    } else if (e.key === 'Enter') {
      e.preventDefault()
      const target = this.filtered[this.selectedIndex]
      if (target && typeof target.action === 'function') {
        closeFn()
        target.action()
      }
    }
  }
}

// 注册 Ctrl+K / Cmd+K 极客指令舱快捷键
export function bindGeekCommandPalette(toggleModalFn) {
  if (typeof window === 'undefined') return () => {}
  const handler = (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault()
      toggleModalFn()
    }
  }
  window.addEventListener('keydown', handler)
  return () => window.removeEventListener('keydown', handler)
}
