import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { WikiHistoryEntry } from '@/services/wiki.service'
import { useWikiEditHistory } from './useWikiEditHistory'

const { getWikiHistoryMock } = vi.hoisted(() => ({
  getWikiHistoryMock: vi.fn(),
}))

vi.mock('@/services/wiki.service', () => ({
  default: {
    getWikiHistory: getWikiHistoryMock,
  },
}))

describe('useWikiEditHistory', () => {
  beforeEach(() => {
    getWikiHistoryMock.mockReset()
  })

  it('loads history entries and selects the newest item by default', async () => {
    const entries: WikiHistoryEntry[] = [
      {
        id: 2,
        content: 'latest',
        change_summary: 'latest summary',
        created_at: '2026-04-03T10:00:00Z',
      },
      {
        id: 1,
        content: 'older',
        change_summary: 'older summary',
        created_at: '2026-04-02T10:00:00Z',
      },
    ]
    getWikiHistoryMock.mockResolvedValue(entries)

    const history = useWikiEditHistory({
      wikiData: ref({
        content: '',
        change_summary: '',
      }),
      addAlert: vi.fn(),
      formatDateTime: (value) => value || '',
    })

    await history.loadHistory('game-1')

    expect(getWikiHistoryMock).toHaveBeenCalledWith('game-1')
    expect(history.historyEntries.value).toEqual(entries)
    expect(history.selectedHistory.value).toEqual(entries[0])
    expect(history.isHistoryLoading.value).toBe(false)
  })

  it('restores selected history content back to editor state', () => {
    const addAlert = vi.fn()
    const wikiData = ref({
      content: '',
      change_summary: '',
    })

    const history = useWikiEditHistory({
      wikiData,
      addAlert,
      formatDateTime: (value) => `formatted:${value || ''}`,
    })

    history.selectedHistory.value = {
      id: 1,
      content: 'restored content',
      created_at: '2026-04-01T10:00:00Z',
    }
    history.historyPreviewVisible.value = true

    history.restoreHistory()

    expect(wikiData.value).toEqual({
      content: 'restored content',
      change_summary: '恢复历史版本：formatted:2026-04-01T10:00:00Z',
    })
    expect(history.historyPreviewVisible.value).toBe(false)
    expect(addAlert).toHaveBeenCalledWith('已将历史版本内容恢复到编辑器', 'success')
  })
})
