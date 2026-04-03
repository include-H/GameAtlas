import type { RouteLocationRaw, Router } from 'vue-router'

export const hasHistoryBack = (historyLength: number) => historyLength > 1

export const navigateBackOrFallback = (
  router: Router,
  fallback: RouteLocationRaw,
) => {
  if (typeof window !== 'undefined' && hasHistoryBack(window.history.length)) {
    router.back()
    return
  }

  router.push(fallback)
}
