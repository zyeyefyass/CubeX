// web/src/utils/command-palette.js

// 根据实时字节速率映射为 3 个离散动画档位，避免每帧修改 DOM 触发样式重算
export function computeFlowDurationTier(bytesPerSec) {
  if (bytesPerSec < 64 * 1024) return '4.5s'       // 轻载呼吸态
  if (bytesPerSec < 4 * 1024 * 1024) return '1.8s' // 中速巡航态
  return '0.75s'                                   // 千兆满血狂飙态
}

export function updateTopologySpeedTiers(railContainerEl, proxyBps, directBps) {
  if (!railContainerEl) return
  const nextJudy = computeFlowDurationTier(proxyBps)
  const nextNick = computeFlowDurationTier(directBps)

  if (railContainerEl.dataset.judyTier !== nextJudy) {
    railContainerEl.dataset.judyTier = nextJudy
    railContainerEl.style.setProperty('--judy-flow-duration', nextJudy)
  }
  if (railContainerEl.dataset.nickTier !== nextNick) {
    railContainerEl.dataset.nickTier = nextNick
    railContainerEl.style.setProperty('--nick-flow-duration', nextNick)
  }
}

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
