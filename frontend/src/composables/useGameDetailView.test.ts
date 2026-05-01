import { describe, expect, it, vi } from 'vitest'

import { startVersionDownload } from './useGameDetailView'

describe('useGameDetailView download chain', () => {
  it('starts the file download even when download recording fails', async () => {
    const recordDownload = vi.fn().mockRejectedValue(new Error('record failed'))
    const navigateToUrl = vi.fn()
    const addAlert = vi.fn()

    await startVersionDownload({
      gameId: 'game-1',
      versionId: '11',
      versionLabel: 'v1.0',
      downloadUrl: '/api/games/game-1/files/11/download',
      recordDownload,
      navigateToUrl,
      addAlert,
    })

    expect(recordDownload).toHaveBeenCalledWith('game-1', '11')
    expect(navigateToUrl).toHaveBeenCalledWith('/api/games/game-1/files/11/download')
    expect(addAlert).toHaveBeenCalledWith('已开始下载 v1.0，但下载记录失败', 'warning')
  })

  it('reports an error when the download itself cannot start', async () => {
    const recordDownload = vi.fn().mockResolvedValue(undefined)
    const navigateToUrl = vi.fn().mockImplementation(() => {
      throw new Error('navigation failed')
    })
    const addAlert = vi.fn()

    await startVersionDownload({
      gameId: 'game-1',
      versionId: '11',
      versionLabel: 'v1.0',
      downloadUrl: '/api/games/game-1/files/11/download',
      recordDownload,
      navigateToUrl,
      addAlert,
    })

    expect(addAlert).toHaveBeenCalledWith('下载启动失败', 'error')
  })
})
