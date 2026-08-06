import axios, { type AxiosInstance, type AxiosRequestConfig, type AxiosError } from 'axios'

// Axios requests read the API base from this file only.
// Non-axios URLs such as download href/action must use buildApiUrl().
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

// Create axios instance
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Response interceptor
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    if (error.response) {
      switch (error.response.status) {
        case 404:
          // 2026-05-13: 页面级 API 的 GET 404 才跳转 not-found。
          // 目录浏览、素材/搜索弹窗、装饰组件等非页面 GET 直接 reject，避免整页误跳转。
          if (isPageLevelGet(error.config)) {
            import('@/router').then(({ default: router }) => {
              if (router.currentRoute.value.name !== 'not-found') {
                router.replace({ name: 'not-found' })
              }
            })
          }
          break
        case 500:
          import('@/stores/ui').then(({ useUiStore }) => {
            useUiStore().addAlert('服务器内部错误', 'error')
          })
          break
      }
    }
    return Promise.reject(error)
  }
)

// /games/ 下的静态路由（列表/统计/预览），404 不代表资源不存在，不跳转。
const GAMES_STATIC_SEGMENTS = ['timeline', 'stats', 'preview-videos']

/**
 * 仅「页面级 GET」的 404 才跳转 not-found：带资源标识的详情端点
 * （游戏详情、Wiki 编辑、系列/发行商详情）。列表、统计、目录浏览、
 * 素材/Steam 搜索弹窗、装饰组件等其余 GET 一律不跳转。
 */
function isPageLevelGet(config?: AxiosRequestConfig): boolean {
  if (!config) return false
  if ((config.method ?? 'get').toLowerCase() !== 'get') return false

  const path = (config.url ?? '').split(/[?#]/)[0]

  // /games/:publicId — 游戏详情（也是媒体/Wiki 编辑的数据源）；排除 /games/timeline|stats|preview-videos
  const gamesDetail = /^\/games\/([^/]+)$/.exec(path)
  if (gamesDetail) {
    return !GAMES_STATIC_SEGMENTS.includes(gamesDetail[1] ?? '')
  }

  // /games/:publicId/wiki — Wiki 编辑页（/wiki/history 历史弹窗多一段，不匹配）
  if (/^\/games\/[^/]+\/wiki$/.test(path)) return true

  // /series/:publicId、/publishers/:publicId — 系列/发行商详情页（列表 /series、/publishers 不匹配）
  if (/^\/series\/[^/]+$/.test(path)) return true
  if (/^\/publishers\/[^/]+$/.test(path)) return true

  return false
}

// Generic request wrapper
const request = async <T = unknown>(config: AxiosRequestConfig): Promise<T> => {
  const response = await apiClient.request<T>(config)
  return response.data
}

// HTTP method helpers
export const get = <T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> => {
  return request<T>({ ...config, method: 'GET', url })
}

export const post = <T = unknown, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig<D>): Promise<T> => {
  return request<T>({ ...config, method: 'POST', url, data })
}

export const put = <T = unknown, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig<D>): Promise<T> => {
  return request<T>({ ...config, method: 'PUT', url, data })
}

export const del = <T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> => {
  return request<T>({ ...config, method: 'DELETE', url })
}

export default apiClient
