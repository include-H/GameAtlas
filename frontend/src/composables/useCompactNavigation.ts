import { onMounted, onUnmounted, ref } from 'vue'

/**
 * Viewport width below which the sidebar collapses into a mobile drawer.
 */
export const COMPACT_NAVIGATION_BREAKPOINT = 992

interface UseCompactNavigationOptions {
  breakpoint?: number
}

/**
 * App-shell compact navigation state.
 *
 * Owns whether the sidebar should render in compact (mobile) mode, the mobile
 * drawer visibility, and the window resize listener that keeps both in sync.
 */
export const useCompactNavigation = ({
  breakpoint = COMPACT_NAVIGATION_BREAKPOINT,
}: UseCompactNavigationOptions = {}) => {
  const isCompactNavigation = ref(false)
  const showMobileMenu = ref(false)

  const handleResize = () => {
    const compact = window.innerWidth < breakpoint
    isCompactNavigation.value = compact

    if (compact) {
      showMobileMenu.value = false
    }
  }

  onMounted(() => {
    handleResize()
    window.addEventListener('resize', handleResize)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
  })

  return {
    isCompactNavigation,
    showMobileMenu,
    handleResize,
    cleanup: () => {
      window.removeEventListener('resize', handleResize)
    },
  }
}
