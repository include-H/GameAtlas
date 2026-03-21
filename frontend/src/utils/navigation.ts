import type { RouteLocationNormalizedLoaded, RouteLocationRaw } from 'vue-router'

const RETURN_TO_QUERY_KEY = 'returnTo'

export const createDetailRouteQuery = (route: RouteLocationNormalizedLoaded) => {
  if (!route.fullPath) {
    return {}
  }

  return {
    [RETURN_TO_QUERY_KEY]: route.fullPath,
  }
}

export const resolveReturnRoute = (
  route: RouteLocationNormalizedLoaded,
  fallback: RouteLocationRaw,
): RouteLocationRaw => {
  const returnTo = route.query[RETURN_TO_QUERY_KEY]
  const target = Array.isArray(returnTo) ? returnTo[0] : returnTo

  if (typeof target === 'string' && target && target !== route.fullPath) {
    return target
  }

  return fallback
}
