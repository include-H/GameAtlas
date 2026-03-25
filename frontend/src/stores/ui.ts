import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

type ViewMode = 'grid' | 'list'
type CardSize = 'small' | 'medium' | 'large'
type AmbientBackgroundOverride = {
  key: string
  url: string
} | null

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
  const ambientBackgroundOverride = ref<AmbientBackgroundOverride>(null)

  // Pagination
  const itemsPerPage = ref(20)

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
    localStorage.setItem('sidebarCollapsed', String(sidebarCollapsed.value))
  }

  const setSidebarCollapsed = (value: boolean) => {
    sidebarCollapsed.value = value
    localStorage.setItem('sidebarCollapsed', String(value))
  }

  const toggleFilters = () => {
    showFilters.value = !showFilters.value
  }

  const toggleSortOptions = () => {
    showSortOptions.value = !showSortOptions.value
  }

  const setAmbientBackgroundOverride = (value: AmbientBackgroundOverride) => {
    ambientBackgroundOverride.value = value
  }

  const clearAmbientBackgroundOverride = () => {
    ambientBackgroundOverride.value = null
  }

  // Alert methods
  const addAlert = (
    message: string,
    type: 'info' | 'success' | 'warning' | 'error' = 'info',
    dismissible = true
  ) => {
    const id = Date.now().toString() + Math.random().toString(36).substr(2, 9)
    alerts.value.push({ id, type, message, dismissible })

    // Auto-dismiss alerts by default; keep errors a bit longer for readability.
    if (dismissible) {
      const duration = type === 'error' ? 8000 : type === 'warning' ? 6500 : 5000
      setTimeout(() => {
        removeAlert(id)
      }, duration)
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

  const initializeSidebarCollapsed = () => {
    const stored = localStorage.getItem('sidebarCollapsed')
    if (stored === 'true' || stored === 'false') {
      sidebarCollapsed.value = stored === 'true'
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
    ambientBackgroundOverride,
    itemsPerPage,
    alerts,
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
    setAmbientBackgroundOverride,
    clearAmbientBackgroundOverride,
    addAlert,
    removeAlert,
    clearAlerts,
    setItemsPerPage,
    initializeItemsPerPage,
    initializeSidebarCollapsed,
    setSidebarCollapsed,
  }
})
