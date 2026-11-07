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

const app = createApp(App)

app.config.errorHandler = (err, vm, info) => {
  console.error('[CubeX Vue Runtime Error]:', err, info)
}

app.mount('#app')
