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

// Initialize persisted UI state
const uiStore = useUiStore()
uiStore.initializeViewMode()
uiStore.initializeSidebarCollapsed()

const bootstrap = async () => {
  await uiStore.initializeSharedBackgroundAvailability()
  app.mount('#app')
}

void bootstrap()
