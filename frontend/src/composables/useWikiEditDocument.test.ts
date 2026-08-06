import { ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useWikiEditDocument } from './useWikiEditDocument'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'
import type { GameDetail } from '@/services/types'

type WikiEditTestGame = {
  id: number
  public_id: string
  title: string
}

const {
  getWikiPageMock,
  updateWikiPageMock,
} = vi.hoisted(() => ({
  getWikiPageMock: vi.fn(),
  updateWikiPageMock: vi.fn(),
}))

vi.mock('@/services/wiki.service', () => ({
  default: {
    getWikiPage: getWikiPageMock,
    updateWikiPage: updateWikiPageMock,
  },
}))

interface CreateWikiHarnessOptions {
  currentGame?: WikiEditTestGame | null
  fetchGame?: (
    gameId: string,
    gamesStore: ReturnType<typeof useGamesStore>,
  ) => Promise<GameDetail>
  addAlert?: ReturnType<typeof vi.fn>
}

const createWikiHarness = (options: CreateWikiHarnessOptions = {}) => {
  const gamesStore = useGamesStore()
  const uiStore = useUiStore()
  const addAlert = options.addAlert ?? vi.fn()
  gamesStore.currentGame = options.currentGame as GameDetail | null
  vi.spyOn(uiStore, 'addAlert').mockImplementation(addAlert)
  if (options.fetchGame) {
    const fetchGame = options.fetchGame
    vi.spyOn(gamesStore, 'fetchGame').mockImplementation((gameId) => fetchGame(gameId, gamesStore))
  }
  return { gamesStore, uiStore, addAlert }
}

describe('useWikiEditDocument', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getWikiPageMock.mockReset()
    updateWikiPageMock.mockReset()
  })

  it('loads existing wiki content into editor state', async () => {
    const { gamesStore, uiStore } = createWikiHarness({
      fetchGame: async (gameId, store) => {
        const game = {
          id: 1,
          public_id: gameId,
          title: 'Game One',
        } as GameDetail
        store.currentGame = game
        return game
      },
    })
    getWikiPageMock.mockResolvedValue({
      content: '# Existing Wiki',
      updated_at: '2026-04-03T00:00:00Z',
    })

    const document = useWikiEditDocument({
      gamesStore,
      uiStore,
      requestedGameId: ref('game-1'),
      onLoadGameFailed: vi.fn(),
    })

    document.wikiData.value = {
      content: 'stale',
      change_summary: 'stale summary',
    }

    const loaded = await document.loadWikiEditorData('game-1')

    expect(gamesStore.fetchGame).toHaveBeenCalledWith('game-1')
    expect(getWikiPageMock).toHaveBeenCalledWith('game-1')
    expect(loaded).toBe(true)
    expect(document.wiki.value?.content).toBe('# Existing Wiki')
    expect(document.wikiData.value).toEqual({
      content: '# Existing Wiki',
      change_summary: '',
    })
    expect(document.isExisting.value).toBe(true)
  })

  it('treats empty wiki content as an existing document instead of missing state', async () => {
    const { gamesStore, uiStore } = createWikiHarness({
      fetchGame: async (gameId, store) => {
        const game = {
          id: 1,
          public_id: gameId,
          title: 'Game One',
        } as GameDetail
        store.currentGame = game
        return game
      },
    })
    getWikiPageMock.mockResolvedValue({
      content: '',
      updated_at: '2026-04-03T00:00:00Z',
    })

    const document = useWikiEditDocument({
      gamesStore,
      uiStore,
      requestedGameId: ref('game-1'),
      onLoadGameFailed: vi.fn(),
    })

    const loaded = await document.loadWikiEditorData('game-1')

    expect(loaded).toBe(true)
    expect(document.wiki.value).toEqual({
      content: '',
      updated_at: '2026-04-03T00:00:00Z',
    })
    expect(document.wikiData.value).toEqual({
      content: '',
      change_summary: '',
    })
    expect(document.isExisting.value).toBe(true)
  })

  it('ignores stale currentGame data that does not match the requested route game', async () => {
    const { gamesStore, uiStore } = createWikiHarness({
      currentGame: {
        id: 1,
        public_id: 'old-game',
        title: 'Old Game',
      },
    })

    const document = useWikiEditDocument({
      gamesStore,
      uiStore,
      requestedGameId: ref('new-game'),
      onLoadGameFailed: vi.fn(),
    })

    expect(document.game.value).toBeNull()
  })

  it('saves wiki content and trims empty summaries', async () => {
    const addAlert = vi.fn()
    const { gamesStore, uiStore } = createWikiHarness({
      currentGame: {
        id: 1,
        public_id: 'game-1',
        title: 'Game One',
      },
      fetchGame: vi.fn(),
      addAlert,
    })
    const onSaveSuccess = vi.fn()

    updateWikiPageMock.mockResolvedValue({
      content: 'new content',
      updated_at: '2026-04-03T00:00:00Z',
    })

    const document = useWikiEditDocument({
      gamesStore,
      uiStore,
      requestedGameId: ref('game-1'),
      onLoadGameFailed: vi.fn(),
      onSaveSuccess,
    })

    document.wikiData.value = {
      content: 'new content',
      change_summary: '   ',
    }

    await document.handleSave()

    expect(updateWikiPageMock).toHaveBeenCalledWith('game-1', {
      content: 'new content',
      change_summary: undefined,
    })
    expect(addAlert).toHaveBeenCalledWith('Wiki 已创建', 'success')
    expect(onSaveSuccess).toHaveBeenCalledWith('game-1')
    expect(document.wikiData.value.change_summary).toBe('')
    expect(document.isSaving.value).toBe(false)
  })

  it('keeps the update semantics when the loaded wiki exists but its content is empty', async () => {
    const addAlert = vi.fn()
    const { gamesStore, uiStore } = createWikiHarness({
      currentGame: {
        id: 1,
        public_id: 'game-1',
        title: 'Game One',
      },
      fetchGame: vi.fn(),
      addAlert,
    })

    getWikiPageMock.mockResolvedValue({
      content: '',
      updated_at: '2026-04-03T00:00:00Z',
    })
    updateWikiPageMock.mockResolvedValue({
      content: 'filled later',
      updated_at: '2026-04-04T00:00:00Z',
    })

    const document = useWikiEditDocument({
      gamesStore,
      uiStore,
      requestedGameId: ref('game-1'),
      onLoadGameFailed: vi.fn(),
      onSaveSuccess: vi.fn(),
    })

    const loaded = await document.loadWikiEditorData('game-1')
    document.wikiData.value = {
      content: 'filled later',
      change_summary: '',
    }

    expect(loaded).toBe(true)
    await document.handleSave()

    expect(addAlert).toHaveBeenCalledWith('Wiki 已更新', 'success')
    expect(document.wiki.value?.content).toBe('filled later')
    expect(document.isExisting.value).toBe(true)
  })

  it('surfaces wiki load failures instead of pretending the document is missing', async () => {
    const addAlert = vi.fn()
    const onLoadGameFailed = vi.fn()
    const { gamesStore, uiStore } = createWikiHarness({
      fetchGame: async (gameId, store) => {
        const game = {
          id: 1,
          public_id: gameId,
          title: 'Game One',
        } as GameDetail
        store.currentGame = game
        return game
      },
      addAlert,
    })

    const notFoundError = {
      isAxiosError: true,
      response: {
        status: 404,
        data: {
          error: 'resource not found',
        },
      },
      message: 'Not Found',
    }
    getWikiPageMock.mockRejectedValueOnce(notFoundError)

    const document = useWikiEditDocument({
      gamesStore,
      uiStore,
      requestedGameId: ref('game-1'),
      onLoadGameFailed,
    })

    let loaded = await document.loadWikiEditorData('game-1')

    expect(addAlert).toHaveBeenCalledWith('resource not found', 'error')
    expect(onLoadGameFailed).toHaveBeenCalledTimes(1)
    expect(document.wiki.value).toBeNull()
    expect(loaded).toBe(false)

    addAlert.mockClear()
    onLoadGameFailed.mockClear()

    const serverError = {
      isAxiosError: true,
      response: {
        status: 500,
        data: {
          error: 'internal server error',
        },
      },
      message: 'Internal Server Error',
    }
    getWikiPageMock.mockRejectedValueOnce(serverError)

    loaded = await document.loadWikiEditorData('game-1')

    expect(addAlert).toHaveBeenCalledWith('internal server error', 'error')
    expect(onLoadGameFailed).toHaveBeenCalledTimes(1)
    expect(loaded).toBe(false)
  })
})
