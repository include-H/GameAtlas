import { describe, expect, it, vi } from 'vitest'
import { useStartScreen } from './useStartScreen'
import type { GameListItem } from '@/services/types'

const makeGame = (publicId: string): GameListItem => ({
  id: 1,
  public_id: publicId,
  title: publicId,
  title_alt: null,
  visibility: 'public',
  summary: null,
  release_date: null,
  cover_image: null,
  banner_image: null,
  wiki_content: null,
  downloads: 0,
  primary_screenshot: null,
  logo_visible: true,
  isFavorite: false,
  series: null,
  created_at: '',
  updated_at: '',
} as unknown as GameListItem)

describe('useStartScreen', () => {
  it('loads favorites when opened for the first time', async () => {
    const fetchFavorites = vi.fn().mockResolvedValue([makeGame('a'), makeGame('b')])
    const screen = useStartScreen({
      fetchFavorites,
      removeFavorite: vi.fn(),
      addAlert: vi.fn(),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

    expect(screen.visible.value).toBe(true)
    expect(fetchFavorites).toHaveBeenCalledTimes(1)
    expect(screen.games.value.map((game) => game.public_id)).toEqual(['a', 'b'])
  })

  it('refetches favorites every time the screen is opened', async () => {
    const fetchFavorites = vi.fn().mockResolvedValue([makeGame('a')])
    const screen = useStartScreen({
      fetchFavorites,
      removeFavorite: vi.fn(),
      addAlert: vi.fn(),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    screen.close()
    screen.open()
    await vi.waitFor(() => expect(screen.visible.value).toBe(true))
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

    expect(fetchFavorites).toHaveBeenCalledTimes(2)
  })

  it('alerts and keeps a retryable failure state when favorites fail to load', async () => {
    const addAlert = vi.fn()
    const screen = useStartScreen({
      fetchFavorites: vi.fn().mockRejectedValue(new Error('boom')),
      removeFavorite: vi.fn(),
      addAlert,
    })

    screen.open()
    await vi.waitFor(() => expect(screen.hasLoadFailure.value).toBe(true))

    expect(screen.isLoading.value).toBe(false)
    expect(addAlert).toHaveBeenCalledWith('开始屏幕加载失败，请稍后重试', 'error')
  })

  it('unpins optimistically and restores the tile when removal fails', async () => {
    const removeFavorite = vi.fn().mockRejectedValue(new Error('boom'))
    const addAlert = vi.fn()
    const screen = useStartScreen({
      fetchFavorites: vi.fn().mockResolvedValue([makeGame('a'), makeGame('b')]),
      removeFavorite,
      addAlert,
    })

    screen.open()
    await vi.waitFor(() => expect(screen.games.value.length).toBe(2))
    await screen.unpin('a')

    expect(removeFavorite).toHaveBeenCalledWith('a')
    expect(screen.games.value.map((game) => game.public_id)).toEqual(['a', 'b'])
    expect(addAlert).toHaveBeenCalledWith('取消收藏失败，请稍后重试', 'error')
  })

  it('removes the tile immediately when unpin succeeds', async () => {
    const screen = useStartScreen({
      fetchFavorites: vi.fn().mockResolvedValue([makeGame('a'), makeGame('b')]),
      removeFavorite: vi.fn().mockResolvedValue(undefined),
      addAlert: vi.fn(),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.games.value.length).toBe(2))
    await screen.unpin('a')

    expect(screen.games.value.map((game) => game.public_id)).toEqual(['b'])
  })
})
