import { createApp } from 'vue'
import './style.css'
import App from './App.vue'

// 全局异常捕获与白屏防护
window.addEventListener('error', (event) => {
  if (event.message && (event.message.includes('Failed to fetch') || event.message.includes('dynamically imported'))) {
    console.warn('[CubeX] 资源版本更新检测，正在自动刷新拉取最新组件...', event)
    window.location.reload()
  }
})

async function bootstrap() {
  const isDemoMode =
    window.location.hostname.includes('github.io') ||
    window.location.search.includes('demo=1') ||
    window.location.protocol === 'file:' ||
    import.meta.env.VITE_DEMO_MODE === 'true'

  // 仅在 Demo 模式下按需动态加载仿真拦截器，避免污染生产主包体积
  if (isDemoMode) {
    const { initDemoMock } = await import('./mock.js')
    initDemoMock()
  }

  const app = createApp(App)
  app.config.errorHandler = (err, vm, info) => {
    console.error('[CubeX Vue Runtime Error]:', err, info)
  }
  app.mount('#app')
}

bootstrap()
