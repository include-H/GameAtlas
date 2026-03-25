import { post } from './api'
import type { ApiEnvelope } from './types'

const downloadService = {
  getDownloadUrl(gameId: string): string {
    return `/api/games/${gameId}`
  },

  getFileDownloadUrl(gameId: string, fileId: string): string {
    return `/api/games/${gameId}/files/${fileId}/download`
  },

  getLaunchScriptUrl(gameId: string, fileId: string): string {
    return `/api/games/${gameId}/files/${fileId}/launch-script`
  },

  downloadGameVersion(gameId: string, fileId: string): void {
    const link = document.createElement('a')
    link.href = this.getFileDownloadUrl(gameId, fileId)
    link.download = ''
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  },

  downloadLaunchScript(gameId: string, fileId: string): void {
    const link = document.createElement('a')
    link.href = this.getLaunchScriptUrl(gameId, fileId)
    link.download = ''
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  },

  async recordDownload(gameId: string, fileId: string): Promise<void> {
    await post<ApiEnvelope<{ recorded: boolean }>>(`/games/${gameId}/files/${fileId}/downloads`)
  },

  async startDownload(gameId: string, versionId?: string): Promise<{
    id: string
    gameId: string
    versionId: string
    status: 'downloading'
    progress: 0
  }> {
    if (versionId) {
      await this.recordDownload(gameId, versionId)
      this.downloadGameVersion(gameId, versionId)
    }

    return {
      id: Date.now().toString(),
      gameId,
      versionId: versionId || '',
      status: 'downloading',
      progress: 0,
    }
  },
}

export default downloadService
