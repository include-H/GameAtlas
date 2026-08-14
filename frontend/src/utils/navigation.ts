import type { LocationQueryValue, RouteLocationRaw, Router } from 'vue-router'

export const GAME_DETAIL_RETURN_QUERY = 'returnTo'

export const hasHistoryBack = (historyLength: number) => historyLength > 1

export const readInternalReturnPath = (
  value: LocationQueryValue | LocationQueryValue[] | undefined,
): string | undefined => {
  if (Array.isArray(value) || typeof value !== 'string') {
    return undefined
  }

  const path = value.trim()
  return path.startsWith('/') && !path.startsWith('//') ? path : undefined
}

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

export const navigateToExplicitReturnOrFallback = (
  router: Router,
  returnTo: LocationQueryValue | LocationQueryValue[] | undefined,
  fallback: RouteLocationRaw,
) => {
  const returnPath = readInternalReturnPath(returnTo)
  if (returnPath) {
    void router.replace(returnPath)
    return
  }

  navigateBackOrFallback(router, fallback)
}
