import { del, get, post, put } from './api'
import type {
  ApiEnvelope,
  StartScreenLayout,
  StartScreenLayoutInput,
  StartScreenTileSize,
} from './types'

const startScreenService = {
  async getTiles(): Promise<StartScreenLayout> {
    const response = await get<ApiEnvelope<StartScreenLayout>>('/start-screen/tiles')
    return {
      columns: response.data?.columns ?? [],
      tiles: response.data?.tiles ?? [],
    }
  },

  async updateTiles(input: StartScreenLayoutInput): Promise<StartScreenLayout> {
    const response = await put<ApiEnvelope<StartScreenLayout>>('/start-screen/tiles', input)
    return {
      columns: response.data?.columns ?? [],
      tiles: response.data?.tiles ?? [],
    }
  },

  async addTile(gameId: number, tileSize: StartScreenTileSize = 'small'): Promise<StartScreenLayout> {
    const response = await post<ApiEnvelope<StartScreenLayout>>('/start-screen/tiles', {
      game_id: gameId,
      tile_size: tileSize,
    })
    return {
      columns: response.data?.columns ?? [],
      tiles: response.data?.tiles ?? [],
    }
  },

  async removeTile(gameId: number): Promise<StartScreenLayout> {
    const response = await del<ApiEnvelope<StartScreenLayout>>(`/start-screen/tiles/${gameId}`)
    return {
      columns: response.data?.columns ?? [],
      tiles: response.data?.tiles ?? [],
    }
  },
}

export default startScreenService
