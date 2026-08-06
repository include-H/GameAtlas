import { getCurrentInstance, onBeforeUnmount, ref, watch, type Ref } from 'vue'
import type { ApiPageEnvelope } from '@/services/types'

interface InfiniteScrollOptions<T> {
  pageSize: number
  sentinel: Ref<HTMLElement | null>
  searchQuery: Ref<string>
  loadPage: (params: { page: number; limit: number; search?: string }) => Promise<ApiPageEnvelope<T>>
  normalizeItems?: (items: T[]) => T[]
  onError: (message: string) => void
}

export const useInfiniteScroll = <T>(options: InfiniteScrollOptions<T>) => {
  const items = ref<T[]>([])
  const isLoading = ref(false)
  const isLoadingMore = ref(false)
  const hasLoadFailure = ref(false)
  const currentPage = ref(1)
  const total = ref(0)
  const hasMore = ref(false)

  let requestId = 0
  let observer: IntersectionObserver | null = null
  let searchTimer: ReturnType<typeof setTimeout> | null = null

  const normalizeItems = (incoming: T[]) => {
    return options.normalizeItems ? options.normalizeItems(incoming) : incoming
  }

  const applyPagination = (pagination: ApiPageEnvelope<T>['pagination']) => {
    currentPage.value = pagination.page
    total.value = pagination.total
    hasMore.value = pagination.page < pagination.totalPages
  }

  const loadFirstPage = async () => {
    const snapshot = ++requestId
    isLoading.value = true
    hasLoadFailure.value = false
    try {
      const response = await options.loadPage({
        page: 1,
        limit: options.pageSize,
        search: options.searchQuery.value.trim() || undefined,
      })
      if (snapshot !== requestId) {
        return
      }
      items.value = normalizeItems(response.data)
      applyPagination(response.pagination)
    } catch {
      if (snapshot !== requestId) {
        return
      }
      hasLoadFailure.value = true
      items.value = []
      total.value = 0
      hasMore.value = false
      options.onError('加载失败')
    } finally {
      if (snapshot === requestId) {
        isLoading.value = false
      }
    }
  }

  const loadMore = async () => {
    if (isLoading.value || isLoadingMore.value || !hasMore.value) {
      return
    }

    const snapshot = requestId
    isLoadingMore.value = true
    try {
      const response = await options.loadPage({
        page: currentPage.value + 1,
        limit: options.pageSize,
        search: options.searchQuery.value.trim() || undefined,
      })
      if (snapshot !== requestId) {
        return
      }
      const incoming = normalizeItems(response.data)
      if (incoming.length === 0) {
        hasMore.value = false
        return
      }
      const currentItems = items.value as T[]
      currentItems.push(...incoming)
      applyPagination(response.pagination)
    } catch {
      if (snapshot === requestId) {
        options.onError('加载更多失败')
      }
    } finally {
      isLoadingMore.value = false
    }
  }

  const setupObserver = () => {
    if (observer) {
      observer.disconnect()
      observer = null
    }
    if (!options.sentinel.value || typeof IntersectionObserver === 'undefined') {
      return
    }

    observer = new IntersectionObserver((entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        void loadMore()
      }
    }, {
      rootMargin: '320px 0px',
    })
    observer.observe(options.sentinel.value)
  }

  watch(options.sentinel, () => {
    setupObserver()
  })

  watch(options.searchQuery, () => {
    if (searchTimer) {
      clearTimeout(searchTimer)
    }
    searchTimer = setTimeout(() => {
      void loadFirstPage()
    }, 250)
  })

  const cleanup = () => {
    if (observer) {
      observer.disconnect()
      observer = null
    }
    if (searchTimer) {
      clearTimeout(searchTimer)
    }
  }

  if (getCurrentInstance()) {
    onBeforeUnmount(cleanup)
  }

  return {
    items,
    isLoading,
    isLoadingMore,
    hasLoadFailure,
    currentPage,
    total,
    hasMore,
    loadFirstPage,
    loadMore,
  }
}
