import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ArcoVue from '@arco-design/web-vue'
import ArcoVueIcon from '@arco-design/web-vue/es/icon'
import '@arco-design/web-vue/dist/arco.css'
import './assets/style.css' // Import custom premium overrides and utilities

import App from './App.vue'
import router from './router'
import { useUiStore } from './stores/ui'
import registerDirectives from './directives'
import { setRootPixel } from './utils/flexible'

// Initialize mobile adaptation (rem-based responsive)
// This sets root font-size for mobile devices (< 540px)
setRootPixel({
  baseFontSize: 50,
  sketchWidth: 375,
  maxFontSize: 64,
  minWidth: 320,
  maxWidth: 768, // Apply to tablet and mobile
})

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.use(ArcoVue)
app.use(ArcoVueIcon)

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
