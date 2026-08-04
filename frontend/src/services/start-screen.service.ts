import { get, post, put } from './api'
import type { ApiEnvelope, StartScreenTile, StartScreenTileSize, StartScreenTileWrite } from './types'

const startScreenService = {
  async getTiles(): Promise<StartScreenTile[]> {
    const response = await get<ApiEnvelope<StartScreenTile[]>>('/start-screen/tiles')
    return response.data ?? []
  },

  async updateTiles(tiles: StartScreenTileWrite[]): Promise<StartScreenTile[]> {
    const response = await put<ApiEnvelope<StartScreenTile[]>>('/start-screen/tiles', { tiles })
    return response.data ?? []
  },

  async uploadTileImage(file: File, size: StartScreenTileSize): Promise<string> {
    const form = new FormData()
    form.append('size', size)
    form.append('file', file)
    const response = await post<ApiEnvelope<{ path: string }>>('/start-screen/tiles/image', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    return response.data.path
  },
}

export default startScreenService
