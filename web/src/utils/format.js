// 共享字节格式化工具（避免 App.vue / LogsModal.vue 等多组件重复定义）
export function formatBytes(bytes, fixedDecimals = null) {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const digits = fixedDecimals !== null ? fixedDecimals : (i >= 3 ? 2 : 1)
  const val = (bytes / Math.pow(1024, i)).toFixed(digits)
  return `${fixedDecimals !== null ? parseFloat(val) : val} ${units[i]}`
}
