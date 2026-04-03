import { defineStore } from 'pinia'
import { ref } from 'vue'
import { safeLocalStorageGetItem, safeLocalStorageSetItem } from '@/utils/safe-local-storage'

type ViewMode = 'grid' | 'list'
type AmbientBackgroundSource = {
  owner: string
  key: string
  urls: string[]
} | null

type SharedBackgroundAvailability = 'unknown' | 'available' | 'missing'

const CUSTOM_BACKGROUND_PATH = '/data/bg.jpg'

export const useUiStore = defineStore('ui', () => {
  // View modes
  const gamesViewMode = ref<ViewMode>('grid')

  // UI state
  const sidebarCollapsed = ref(false)
  const ambientBackgroundSource = ref<AmbientBackgroundSource>(null)
  const sharedBackgroundAvailability = ref<SharedBackgroundAvailability>('unknown')

  // Alerts
  const alerts = ref<Array<{
    id: string
    type: 'info' | 'success' | 'warning' | 'error'
    message: string
    dismissible?: boolean
  }>>([])

  // View mode methods
  const setGamesViewMode = (mode: ViewMode) => {
    gamesViewMode.value = mode
    safeLocalStorageSetItem('gamesViewMode', mode)
  }

  const initializeViewMode = () => {
    const stored = safeLocalStorageGetItem('gamesViewMode')
    if (stored && (stored === 'grid' || stored === 'list')) {
      gamesViewMode.value = stored as ViewMode
    }
  }

  // UI state methods
  const toggleSidebar = () => {
    sidebarCollapsed.value = !sidebarCollapsed.value
    safeLocalStorageSetItem('sidebarCollapsed', String(sidebarCollapsed.value))
  }

  const setSidebarCollapsed = (value: boolean) => {
    sidebarCollapsed.value = value
    safeLocalStorageSetItem('sidebarCollapsed', String(value))
  }

  const setAmbientBackgroundSource = (value: AmbientBackgroundSource) => {
    ambientBackgroundSource.value = value
  }

  const clearAmbientBackgroundSource = (owner: string) => {
    if (ambientBackgroundSource.value?.owner !== owner) {
      return
    }

    ambientBackgroundSource.value = null
  }

  const initializeSharedBackgroundAvailability = async () => {
    try {
      const response = await fetch(CUSTOM_BACKGROUND_PATH, {
        method: 'HEAD',
        cache: 'no-store',
      })
      sharedBackgroundAvailability.value = response.ok ? 'available' : 'missing'
    } catch {
      sharedBackgroundAvailability.value = 'missing'
    }
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

  const initializeSidebarCollapsed = () => {
    const stored = safeLocalStorageGetItem('sidebarCollapsed')
    if (stored === 'true' || stored === 'false') {
      sidebarCollapsed.value = stored === 'true'
    }
  }

  return {
    // State
    gamesViewMode,
    sidebarCollapsed,
    ambientBackgroundSource,
    sharedBackgroundAvailability,
    alerts,
    // Actions
    setGamesViewMode,
    initializeViewMode,
    toggleSidebar,
    setAmbientBackgroundSource,
    clearAmbientBackgroundSource,
    initializeSharedBackgroundAvailability,
    addAlert,
    removeAlert,
    clearAlerts,
    initializeSidebarCollapsed,
    setSidebarCollapsed,
  }
})
