import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export type ViewMode = 'grid' | 'list'
export type CardSize = 'small' | 'medium' | 'large'

export interface ProxyConfig {
  enabled: boolean
  host: string
  port: string
  protocol: 'http' | 'https' | 'socks4' | 'socks5'
}

export const useUiStore = defineStore('ui', () => {
  // Theme
  const isDark = ref(true)
  const theme = computed(() => (isDark.value ? 'dark' : 'light'))

  // View modes
  const gamesViewMode = ref<ViewMode>('grid')
  const cardSize = ref<CardSize>('medium')

  // UI state
  const sidebarCollapsed = ref(false)
  const showFilters = ref(false)
  const showSortOptions = ref(false)

  // Pagination
  const itemsPerPage = ref(20)

  // Proxy configuration for Steam API
  const proxyConfig = ref<ProxyConfig>({
    enabled: false,
    host: '',
    port: '',
    protocol: 'http'
  })

  // Alerts
  const alerts = ref<Array<{
    id: string
    type: 'info' | 'success' | 'warning' | 'error'
    message: string
    dismissible?: boolean
  }>>([])

  // Apply theme to DOM (Arco Design Vue specific)
  const applyTheme = (dark: boolean) => {
    if (dark) {
      document.body.setAttribute('arco-theme', 'dark')
    } else {
      document.body.removeAttribute('arco-theme')
    }
  }

  // Theme methods - Fixed to dark mode
  const initializeTheme = () => {
    // Always use dark mode
    isDark.value = true
    applyTheme(true)
  }

  // Toggle theme - disabled, always dark
  const toggleTheme = () => {
    // Dark mode is fixed, do nothing
  }

  // View mode methods
  const setGamesViewMode = (mode: ViewMode) => {
    gamesViewMode.value = mode
    localStorage.setItem('gamesViewMode', mode)
  }

  const toggleGamesViewMode = () => {
    setGamesViewMode(gamesViewMode.value === 'grid' ? 'list' : 'grid')
  }

  const initializeViewMode = () => {
    const stored = localStorage.getItem('gamesViewMode')
    if (stored && (stored === 'grid' || stored === 'list')) {
      gamesViewMode.value = stored as ViewMode
    }
  }

  // Card size methods
  const setCardSize = (size: CardSize) => {
    cardSize.value = size
    localStorage.setItem('cardSize', size)
  }

  const initializeCardSize = () => {
    const stored = localStorage.getItem('cardSize')
    if (stored && (stored === 'small' || stored === 'medium' || stored === 'large')) {
      cardSize.value = stored as CardSize
    }
  }

  // UI state methods
  const toggleSidebar = () => {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  const toggleFilters = () => {
    showFilters.value = !showFilters.value
  }

  const toggleSortOptions = () => {
    showSortOptions.value = !showSortOptions.value
  }

  // Alert methods
  const addAlert = (
    message: string,
    type: 'info' | 'success' | 'warning' | 'error' = 'info',
    dismissible = true
  ) => {
    const id = Date.now().toString() + Math.random().toString(36).substr(2, 9)
    alerts.value.push({ id, type, message, dismissible })

    // Auto-dismiss after 5 seconds for info and success
    if (type === 'info' || type === 'success') {
      setTimeout(() => {
        removeAlert(id)
      }, 5000)
    }

    return id
  }

  const removeAlert = (id: string) => {
    const index = alerts.value.findIndex(a => a.id === id)
    if (index > -1) {
      alerts.value.splice(index, 1)
    }
  }

  const clearAlerts = () => {
    alerts.value = []
  }

  // Items per page
  const setItemsPerPage = (count: number) => {
    itemsPerPage.value = count
    localStorage.setItem('itemsPerPage', count.toString())
  }

  const initializeItemsPerPage = () => {
    const stored = localStorage.getItem('itemsPerPage')
    if (stored) {
      const count = parseInt(stored, 10)
      if (count > 0 && count <= 100) {
        itemsPerPage.value = count
      }
    }
  }

  // Proxy configuration methods
  const setProxyConfig = (config: ProxyConfig) => {
    proxyConfig.value = config
    localStorage.setItem('proxyConfig', JSON.stringify(config))
  }

  const getProxyUrl = () => {
    if (!proxyConfig.value.enabled || !proxyConfig.value.host || !proxyConfig.value.port) {
      return null
    }
    return `${proxyConfig.value.protocol}://${proxyConfig.value.host}:${proxyConfig.value.port}`
  }

  const initializeProxyConfig = () => {
    const stored = localStorage.getItem('proxyConfig')
    if (stored) {
      try {
        proxyConfig.value = JSON.parse(stored)
      } catch (e) {
        console.error('Failed to parse proxy config:', e)
      }
    }
  }

  return {
    // State
    isDark,
    theme,
    gamesViewMode,
    cardSize,
    sidebarCollapsed,
    showFilters,
    showSortOptions,
    itemsPerPage,
    alerts,
    proxyConfig,
    // Actions
    initializeTheme,
    toggleTheme,
    setGamesViewMode,
    toggleGamesViewMode,
    initializeViewMode,
    setCardSize,
    initializeCardSize,
    toggleSidebar,
    toggleFilters,
    toggleSortOptions,
    addAlert,
    removeAlert,
    clearAlerts,
    setItemsPerPage,
    initializeItemsPerPage,
    setProxyConfig,
    getProxyUrl,
    initializeProxyConfig,
  }
})
