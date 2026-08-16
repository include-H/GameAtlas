import { createApp } from 'vue'
import '@arco-design/web-vue/dist/arco.css'
import './streaming.css'

import App from './App.vue'

const app = createApp(App)

// Arco 暗色主题
document.body.setAttribute('arco-theme', 'dark')

app.mount('#app')
