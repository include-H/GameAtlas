export const downloadService = {
  getDownloadUrl(gameId: string): string {
    return `/api/games/${gameId}`
  },

  getFileDownloadUrl(gameId: string, fileId: string): string {
    return `/api/games/${gameId}/files/${fileId}/download`
  },

  downloadGameVersion(gameId: string, fileId: string): void {
    const link = document.createElement('a')
    link.href = this.getFileDownloadUrl(gameId, fileId)
    link.download = ''
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  },

  async startDownload(gameId: string, versionId?: string): Promise<{
    id: string
    gameId: string
    versionId: string
    status: 'downloading'
    progress: 0
  }> {
    if (versionId) {
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
