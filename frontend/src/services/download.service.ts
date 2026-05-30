import { post } from './api'
import type { ApiEnvelope } from './types'

const downloadService = {
  async recordDownload(gameId: string, fileId: string): Promise<{ recorded: boolean }> {
    const response = await post<ApiEnvelope<{ recorded: boolean }>>(`/games/${gameId}/files/${fileId}/downloads`)
    return response.data
  },
}

export default downloadService
