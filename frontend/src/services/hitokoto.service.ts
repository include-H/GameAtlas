import { get } from './api'

interface HitokotoSentence {
  id: number
  uuid: string
  hitokoto: string
  type: string
  from: string
  from_who: string | null
  creator: string
  creator_uid: number
  reviewer: number
  commit_from: string
  created_at: string
  length: number
}

const hitokotoService = {
  async getGameSentence(params?: {
    min_length?: number
    max_length?: number
  }): Promise<HitokotoSentence> {
    return get<HitokotoSentence>('/hitokoto', {
      params: {
        c: 'c',
        min_length: params?.min_length ?? 10,
        max_length: params?.max_length ?? 34,
      },
    })
  },
}

export default hitokotoService
