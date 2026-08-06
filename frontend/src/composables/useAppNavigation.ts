import { ref, watch } from 'vue'
import type { ComputedRef, Ref } from 'vue'
import type { Router } from 'vue-router'

interface UseAppNavigationOptions {
  router: Router
  routeOpenKeys: ComputedRef<string[]>
  activeKey: ComputedRef<string>
  closeMobileMenu: () => void
}

interface UseAppNavigationReturn {
  desktopOpenKeys: Ref<string[]>
  mobileOpenKeys: Ref<string[]>
  handleMenuClick: (key: string) => void
  handleMobileMenuClick: (key: string) => void
  handleDesktopOpenKeysChange: (keys: string[]) => void
  handleMobileOpenKeysChange: (keys: string[]) => void
}

/**
 * Desktop/mobile side-menu open-key state for the app shell.
 *
 * Keeps the Arco menu open keys in sync with the active route and routes menu
 * navigation clicks. Desktop clicks close the mobile drawer first.
 */
export const useAppNavigation = ({
  router,
  routeOpenKeys,
  activeKey,
  closeMobileMenu,
}: UseAppNavigationOptions): UseAppNavigationReturn => {
  const desktopOpenKeys = ref<string[]>([])
  const mobileOpenKeys = ref<string[]>([])

  const syncOpenKeysWithRoute = () => {
    desktopOpenKeys.value = [...routeOpenKeys.value]
    mobileOpenKeys.value = [...routeOpenKeys.value]
  }

  const handleMenuClick = (key: string) => {
    router.push({ name: key })
  }

  const handleMobileMenuClick = (key: string) => {
    handleMenuClick(key)
    closeMobileMenu()
  }

  const handleDesktopOpenKeysChange = (keys: string[]) => {
    desktopOpenKeys.value = keys
  }

  const handleMobileOpenKeysChange = (keys: string[]) => {
    mobileOpenKeys.value = keys
  }

  watch(
    [activeKey, routeOpenKeys],
    () => {
      syncOpenKeysWithRoute()
    },
    { immediate: true },
  )

  return {
    desktopOpenKeys,
    mobileOpenKeys,
    handleMenuClick,
    handleMobileMenuClick,
    handleDesktopOpenKeysChange,
    handleMobileOpenKeysChange,
  }
}
