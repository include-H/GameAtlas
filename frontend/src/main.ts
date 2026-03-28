import { createApp } from 'vue'
import { createPinia } from 'pinia'
import '@arco-design/web-vue/dist/arco.css'
import './assets/style.css' // Import custom premium overrides and utilities

import App from './App.vue'
import router from './router'
import { useUiStore } from './stores/ui'
import registerDirectives from './directives'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// Register directives
registerDirectives(app)

// Initialize UI store (theme, view mode, etc.)
const uiStore = useUiStore()
uiStore.initializeTheme()
uiStore.initializeViewMode()
uiStore.initializeCardSize()
uiStore.initializeItemsPerPage()
uiStore.initializeSidebarCollapsed()

app.mount('#app')
