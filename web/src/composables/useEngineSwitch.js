// web/src/composables/useEngineSwitch.js
// CubeX v9.9.4 双核热重载与 1.5s 宽限期调度器
import { ref } from 'vue'

export function useEngineSwitch(onSwitchComplete) {
  const isSwitching = ref(false)
  const targetEngine = ref('')
  const switchGraceCountdown = ref(0)
  let graceTimer = null

  const triggerEngineSwitch = async (engineName) => {
    if (isSwitching.value) return
    isSwitching.value = true
    targetEngine.value = engineName
    switchGraceCountdown.value = 1500 // 1.5s 物理切核宽限期

    const step = 100
    graceTimer = setInterval(() => {
      switchGraceCountdown.value -= step
      if (switchGraceCountdown.value <= 0) {
        clearInterval(graceTimer)
        graceTimer = null
      }
    }, step)

    try {
      const res = await fetch('/api/v1/engine/switch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ engine: engineName })
      })
      const result = await res.json()
      // 等待 1.5s 宽限期平滑结束，确保内核完成重载且探针无假死
      await new Promise(resolve => setTimeout(resolve, 1500))
      if (onSwitchComplete) {
        onSwitchComplete(result)
      }
      return result
    } catch (err) {
      console.error('[EngineSwitch] Error:', err)
      throw err
    } finally {
      isSwitching.value = false
      targetEngine.value = ''
      if (graceTimer) {
        clearInterval(graceTimer)
        graceTimer = null
      }
    }
  }

  return {
    isSwitching,
    targetEngine,
    switchGraceCountdown,
    triggerEngineSwitch
  }
}
