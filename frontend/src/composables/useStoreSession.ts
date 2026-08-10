import { ref } from 'vue'
import gamesService from '@/services/games.service'
import { useUiStore } from '@/stores/ui'
import {
  getAmbientBackgroundPoolFromGames,
  mergeAmbientBackgroundPools,
  type AmbientBackgroundPool,
} from '@/utils/ambient-background'
import { createRequestGeneration, type RequestGenerationGuard } from '@/utils/request-generation'

export interface StoreShelfGame {
  publicId: string
  title: string
  titleAlt: string
  year: string
  coverUrl: string
}

/**
 * 游戏店会话：负责货架游戏、海报、CRT 轮播片源三类数据的加载。
 * 海报与货架会话并行拉取；CRT 片源依赖会话结果（getPreviewVideos）。
 * start() 创建 AbortController 并并行加载，dispose() 标记已卸载并中止请求。
 */
export const useStoreSession = () => {
  const uiStore = useUiStore()

  const gameStoreSessionGames = ref<StoreShelfGame[]>([])
  const storePosters = ref<string[]>([])
  const crtPlaylist = ref<string[]>([])

  let isStoreDisposed = false
  let storeAbortController: AbortController | null = null
  const sessionRequests = createRequestGeneration()

  const pickPosterImages = (pool: AmbientBackgroundPool): string[] => {
    const picks: string[] = []
    const add = (url: string | undefined | null) => {
      if (url && !picks.includes(url)) picks.push(url)
    }
    add(pool.screenshots[Math.floor(Math.random() * pool.screenshots.length)])
    add(pool.banners[Math.floor(Math.random() * pool.banners.length)])
    if (picks.length < 2) {
      const rest = [...pool.screenshots, ...pool.banners].filter((url) => !picks.includes(url))
      if (rest.length > 0) {
        add(rest[Math.floor(Math.random() * rest.length)])
      }
    }
    return picks
  }

  const loadStorePosters = async (signal: AbortSignal, request: RequestGenerationGuard) => {
    try {
      // 与全局抽卡池同一数据源：遍历游戏列表收集 banner 与截图
      const pools: AmbientBackgroundPool[] = []
      let page = 1
      while (true) {
        const result = await gamesService.getGames({
          query: { page, limit: 100 },
          sort: { field: 'created_at', order: 'desc' },
          signal,
        })
        if (isStoreDisposed || !request.isCurrent()) return
        pools.push(getAmbientBackgroundPoolFromGames(result.data))
        const totalPages = Math.max(1, result.pagination.totalPages || 1)
        if (page >= totalPages) break
        page += 1
      }
      if (isStoreDisposed || !request.isCurrent()) return
      storePosters.value = pickPosterImages(mergeAmbientBackgroundPools(pools))
    } catch {
      if (isStoreDisposed || !request.isCurrent()) return
      // 拉取失败时保留纯色海报兜底
      uiStore.addAlert('游戏店海报加载失败，已显示兜底样式', 'warning')
    }
  }

  const loadStoreSession = async (signal: AbortSignal, request: RequestGenerationGuard) => {
    try {
      // 每次进入生成一个随机种子，后端直接按 random 排序返回 20 个游戏
      const seed = Math.floor(Math.random() * 2_147_483_647) + 1
      const result = await gamesService.getGames({
        query: { page: 1, limit: 20 },
        sort: { field: 'random', order: 'desc', seed },
        signal,
      })
      if (isStoreDisposed || !request.isCurrent()) return
      // 货架固定 4 行 × 5 盒
      const picked = result.data.filter((game) => game.cover_image)
      gameStoreSessionGames.value = picked.map((game) => ({
        publicId: game.public_id,
        title: game.title,
        titleAlt: game.title_alt ?? '',
        year: game.release_date ? game.release_date.slice(0, 4) : '',
        coverUrl: game.cover_image || '',
      }))

      // CRT 播放真实预告：把本次 Session 所有预告片拍平成轮播列表，播完自动换下一个
      const videoBundles = await gamesService.getPreviewVideos(
        picked.map((game) => game.public_id),
        signal,
      )
      if (isStoreDisposed || !request.isCurrent()) return
      crtPlaylist.value = videoBundles.flatMap((bundle) => bundle.preview_videos.map((video) => video.path))
    } catch {
      if (isStoreDisposed || !request.isCurrent()) return
      // 拉取失败时保留空货架，避免展示 mock 数据
      uiStore.addAlert('游戏店数据加载失败，货架暂时为空', 'warning')
    }
  }

  const start = () => {
    sessionRequests.invalidate()
    storeAbortController?.abort()
    isStoreDisposed = false
    const request = sessionRequests.begin()
    storeAbortController = new AbortController()
    const storeSignal = storeAbortController.signal
    void loadStorePosters(storeSignal, request)
    void loadStoreSession(storeSignal, request)
  }

  const dispose = () => {
    isStoreDisposed = true
    sessionRequests.invalidate()
    storeAbortController?.abort()
    storeAbortController = null
  }

  return {
    gameStoreSessionGames,
    storePosters,
    crtPlaylist,
    start,
    dispose,
  }
}
