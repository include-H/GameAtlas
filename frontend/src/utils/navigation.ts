import type { RouteLocationNormalizedLoaded, RouteLocationRaw } from 'vue-router'

const RETURN_PATH_QUERY_KEY = 'returnPath'
const RETURN_NAME_QUERY_KEY = 'returnName'
const RETURN_QUERY_PREFIX = 'returnQuery_'

export const createDetailRouteQuery = (route: RouteLocationNormalizedLoaded) => {
  if (!route.path) {
    return {}
  }

  const query: Record<string, string> = {
    [RETURN_PATH_QUERY_KEY]: route.path,
  }
  if (typeof route.name === 'string' && route.name.trim()) {
    query[RETURN_NAME_QUERY_KEY] = route.name
  }

  Object.entries(route.query || {}).forEach(([key, value]) => {
    const first = Array.isArray(value) ? value[0] : value
    if (typeof first === 'string' && first.trim() !== '') {
      query[`${RETURN_QUERY_PREFIX}${key}`] = first
    }
  })

  return query
}

export const resolveReturnRoute = (
  route: RouteLocationNormalizedLoaded,
  fallback: RouteLocationRaw,
): RouteLocationRaw => {
  const returnPathRaw = route.query[RETURN_PATH_QUERY_KEY]
  const returnNameRaw = route.query[RETURN_NAME_QUERY_KEY]
  const returnPath = Array.isArray(returnPathRaw) ? returnPathRaw[0] : returnPathRaw
  const returnName = Array.isArray(returnNameRaw) ? returnNameRaw[0] : returnNameRaw
  const query: Record<string, string> = {}

  Object.entries(route.query || {}).forEach(([key, value]) => {
    if (!key.startsWith(RETURN_QUERY_PREFIX)) {
      return
    }
    const first = Array.isArray(value) ? value[0] : value
    if (typeof first !== 'string') {
      return
    }
    const sourceKey = key.slice(RETURN_QUERY_PREFIX.length)
    if (!sourceKey) {
      return
    }
    query[sourceKey] = first
  })

  if (typeof returnName === 'string' && returnName.trim()) {
    return {
      name: returnName,
      query,
    }
  }
  if (typeof returnPath === 'string' && returnPath.trim()) {
    return {
      path: returnPath,
      query,
    }
  }

  return fallback
}
