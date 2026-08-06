import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useInfiniteScroll } from './useInfiniteScroll'

describe('useInfiniteScroll', () => {
  it('loads the first page and appends the next page', async () => {
    const loadPage = vi.fn()
      .mockResolvedValueOnce({
        data: [{ id: 1 }],
        pagination: { page: 1, limit: 24, total: 2, totalPages: 2 },
      })
      .mockResolvedValueOnce({
        data: [{ id: 2 }],
        pagination: { page: 2, limit: 24, total: 2, totalPages: 2 },
      })
    const sentinel = ref<HTMLElement | null>(null)
    const searchQuery = ref('')
    const onError = vi.fn()

    const list = useInfiniteScroll<{ id: number }>({
      pageSize: 24,
      sentinel,
      searchQuery,
      loadPage,
      onError,
    })

    await list.loadFirstPage()
    await list.loadMore()

    expect(list.items.value).toEqual([{ id: 1 }, { id: 2 }])
    expect(list.total.value).toBe(2)
    expect(list.hasMore.value).toBe(false)
    expect(onError).not.toHaveBeenCalled()
  })
})
