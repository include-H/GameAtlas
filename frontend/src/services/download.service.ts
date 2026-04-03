import { post } from './api'
import type { ApiEnvelope } from './types'

const downloadService = {
  async recordDownload(gameId: string, fileId: string): Promise<void> {
    await post<ApiEnvelope<{ recorded: boolean }>>(`/games/${gameId}/files/${fileId}/downloads`)
  },
}

export default downloadService
