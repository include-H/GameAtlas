import { computed, watch, type WatchOptions, type WatchSource } from 'vue'
import type { RouteLocationNormalizedLoaded } from 'vue-router'

type MaybePromise<T> = T | Promise<T>

/**
 * Helpers for keep-alive pages that read from route params.
 *
 * Why:
 * Cached views can stay mounted after route switches. If multiple pages watch
 * the same param key such as `route.params.id`, an inactive page may still fire
 * requests with another page's id.
 *
 * Typical usage in a detail page:
 * ```ts
 * const route = useRoute()
 * const { runWhenActive } = useNamedRouteGuard(route, 'game-detail')
 *
 * const loadGameDetail = async (gameId: string) => {
 *   await runWhenActive(async () => {
 *     await gamesStore.fetchGame(gameId)
 *   })
 * }
 *
 * watchRouteParamWhenActive(route, 'game-detail', 'id', async (gameId) => {
 *   await loadGameDetail(gameId)
 * })
 * ```
 *
 * Use `runWhenActive` for any async task that must only run on one named route.
 * Use `watchRouteParamWhenActive` when the page reacts to a specific route param.
 */
export const getRouteParamString = (route: RouteLocationNormalizedLoaded, paramName: string) => {
  const rawValue = route.params[paramName]
  const value = Array.isArray(rawValue) ? rawValue[0] : rawValue
  if (typeof value !== 'string') {
    return ''
  }

  return value.trim()
}

export const useNamedRouteGuard = (route: RouteLocationNormalizedLoaded, routeName: string) => {
  const isActiveRoute = computed(() => route.name === routeName)

  const runWhenActive = <T>(task: () => MaybePromise<T>) => {
    if (!isActiveRoute.value) {
      return undefined
    }

    return task()
  }

  const watchWhenActive = <T>(
    source: WatchSource<T>,
    callback: (value: T, oldValue: T | undefined) => MaybePromise<void>,
    options?: WatchOptions,
  ) => {
    return watch(
      source as WatchSource<T>,
      async (value: T, oldValue: T | undefined) => {
        if (!isActiveRoute.value) {
          return
        }

        await callback(value, oldValue)
      },
      options,
    )
  }

  return {
    isActiveRoute,
    runWhenActive,
    watchWhenActive,
  }
}

export const watchRouteParamWhenActive = (
  route: RouteLocationNormalizedLoaded,
  routeName: string,
  paramName: string,
  callback: (value: string, oldValue?: string) => MaybePromise<void>,
  options?: WatchOptions,
) => {
  const { watchWhenActive } = useNamedRouteGuard(route, routeName)

  return watchWhenActive(
    () => getRouteParamString(route, paramName),
    async (value, oldValue) => {
      if (!value) {
        return
      }

      await callback(value, oldValue)
    },
    {
      immediate: true,
      ...options,
    },
  )
}
