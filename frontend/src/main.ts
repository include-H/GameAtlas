import { createApp } from 'vue'
import { createPinia } from 'pinia'
import '@arco-design/web-vue/dist/arco.css'
import './assets/style.css' // Import custom premium overrides and utilities

import App from './App.vue'
import router from './router'
import { useUiStore } from './stores/ui'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

document.body.setAttribute('arco-theme', 'dark')

// SPA 同文档历史导航下，浏览器会按历史条目自动恢复 .content 的滚动位置，
// 与游戏库虚拟滚动的自行恢复（scrollTo 离开位置）竞态，导致恢复帧被覆盖成旧位置、
// 渲染与可视区错位而白屏。统一交由应用管理滚动恢复，禁用原生行为。
if ('scrollRestoration' in history) {
  history.scrollRestoration = 'manual'
}

// Initialize persisted UI state
const uiStore = useUiStore()
uiStore.initializeViewMode()
uiStore.initializeSidebarCollapsed()

const bootstrap = async () => {
  await uiStore.initializeSharedBackgroundAvailability()
  app.mount('#app')
}

void bootstrap()
