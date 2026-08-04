import { get, put } from './api'
import type { ApiEnvelope, StartScreenTile, StartScreenTileWrite } from './types'

const startScreenService = {
  async getTiles(): Promise<StartScreenTile[]> {
    const response = await get<ApiEnvelope<StartScreenTile[]>>('/start-screen/tiles')
    return response.data ?? []
  },

  async updateTiles(tiles: StartScreenTileWrite[]): Promise<StartScreenTile[]> {
    const response = await put<ApiEnvelope<StartScreenTile[]>>('/start-screen/tiles', { tiles })
    return response.data ?? []
  },
}

export default startScreenService
