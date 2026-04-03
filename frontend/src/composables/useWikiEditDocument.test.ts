import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useWikiEditDocument } from './useWikiEditDocument'

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

describe('useWikiEditDocument', () => {
  beforeEach(() => {
    getWikiPageMock.mockReset()
    updateWikiPageMock.mockReset()
  })

  it('loads existing wiki content into editor state', async () => {
    const currentGame = ref(null)
    const fetchGame = vi.fn().mockImplementation(async (gameId: string) => {
      currentGame.value = {
        id: 1,
        public_id: gameId,
        title: 'Game One',
      }
    })
    getWikiPageMock.mockResolvedValue({
      content: '# Existing Wiki',
      updated_at: '2026-04-03T00:00:00Z',
    })

    const document = useWikiEditDocument({
      gamesStore: {
        get currentGame() {
          return currentGame.value
        },
        fetchGame,
      } as never,
      uiStore: { addAlert: vi.fn() } as never,
      onLoadGameFailed: vi.fn(),
    })

    document.wikiData.value = {
      content: 'stale',
      change_summary: 'stale summary',
    }

    await document.loadWikiEditorData('game-1')

    expect(fetchGame).toHaveBeenCalledWith('game-1')
    expect(getWikiPageMock).toHaveBeenCalledWith('game-1')
    expect(document.wiki.value?.content).toBe('# Existing Wiki')
    expect(document.wikiData.value).toEqual({
      content: '# Existing Wiki',
      change_summary: '',
    })
    expect(document.isExisting.value).toBe(true)
  })

  it('saves wiki content and trims empty summaries', async () => {
    const addAlert = vi.fn()
    const onSaveSuccess = vi.fn()
    const currentGame = ref({
      id: 1,
      public_id: 'game-1',
      title: 'Game One',
    })

    updateWikiPageMock.mockResolvedValue({
      content: 'new content',
      updated_at: '2026-04-03T00:00:00Z',
    })

    const document = useWikiEditDocument({
      gamesStore: {
        get currentGame() {
          return currentGame.value
        },
        fetchGame: vi.fn(),
      } as never,
      uiStore: { addAlert } as never,
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
})
