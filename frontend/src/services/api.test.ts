import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  AxiosError,
  AxiosHeaders,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios'
import apiClient from './api'

const routerMock = vi.hoisted(() => ({
  replace: vi.fn(),
  currentRoute: { value: { name: 'home' } },
}))

const uiStoreMock = vi.hoisted(() => ({
  addAlert: vi.fn(),
}))

vi.mock('@/router', () => ({ default: routerMock }))
vi.mock('@/stores/ui', () => ({ useUiStore: () => uiStoreMock }))

function buildError(status: number, url: string, method: string): AxiosError {
  const config: InternalAxiosRequestConfig = {
    url,
    method,
    headers: new AxiosHeaders(),
  }
  const response: AxiosResponse = {
    status,
    statusText: status === 404 ? 'Not Found' : 'Internal Server Error',
    data: {},
    headers: {},
    config,
  }
  return new AxiosError(
    `Request failed with status code ${status}`,
    'ERR_BAD_REQUEST',
    config,
    undefined,
    response
  )
}

function requestFail(url: string, status = 404, method = 'get'): Promise<void> {
  return apiClient
    .request({ url, method, adapter: () => Promise.reject(buildError(status, url, method)) })
    .then(
      () => undefined,
      () => undefined
    )
}

const flush = () => new Promise<void>((resolve) => setTimeout(resolve, 0))

describe('response interceptor 404 handling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    routerMock.currentRoute.value.name = 'home'
  })

  it('redirects to not-found when game detail GET 404s', async () => {
    await requestFail('/games/game-1')
    await vi.waitFor(() => expect(routerMock.replace).toHaveBeenCalledWith({ name: 'not-found' }))
  })

  it('redirects to not-found when wiki page GET 404s', async () => {
    await requestFail('/games/game-1/wiki')
    await vi.waitFor(() => expect(routerMock.replace).toHaveBeenCalledWith({ name: 'not-found' }))
  })

  it('redirects to not-found when series detail GET 404s', async () => {
    await requestFail('/series/series-1')
    await vi.waitFor(() => expect(routerMock.replace).toHaveBeenCalledWith({ name: 'not-found' }))
  })

  it('redirects to not-found when publisher detail GET 404s', async () => {
    await requestFail('/publishers/pub-1')
    await vi.waitFor(() => expect(routerMock.replace).toHaveBeenCalledWith({ name: 'not-found' }))
  })

  it('does NOT redirect for directory browsing GET 404', async () => {
    await requestFail('/directory/list')
    await flush()
    expect(routerMock.replace).not.toHaveBeenCalled()
  })

  it('does NOT redirect for default directory GET 404', async () => {
    await requestFail('/directory/default')
    await flush()
    expect(routerMock.replace).not.toHaveBeenCalled()
  })

  it('does NOT redirect for directory search GET 404', async () => {
    await requestFail('/directory/search')
    await flush()
    expect(routerMock.replace).not.toHaveBeenCalled()
  })

  it('does NOT redirect for games list GET 404', async () => {
    await requestFail('/games')
    await flush()
    expect(routerMock.replace).not.toHaveBeenCalled()
  })

  it.each(['/games/timeline', '/games/stats', '/games/preview-videos'])(
    'does NOT redirect for static games route GET 404 (%s)',
    async (url) => {
      await requestFail(url)
      await flush()
      expect(routerMock.replace).not.toHaveBeenCalled()
    }
  )

  it('does NOT redirect for wiki history dialog GET 404', async () => {
    await requestFail('/games/game-1/wiki/history')
    await flush()
    expect(routerMock.replace).not.toHaveBeenCalled()
  })

  it('does NOT redirect for hitokoto decorative GET 404', async () => {
    await requestFail('/hitokoto')
    await flush()
    expect(routerMock.replace).not.toHaveBeenCalled()
  })

  it('does NOT redirect for non-GET 404 on a page-level URL', async () => {
    await requestFail('/games/game-1', 404, 'post')
    await flush()
    expect(routerMock.replace).not.toHaveBeenCalled()
  })

  it('does NOT redirect when already on the not-found page', async () => {
    routerMock.currentRoute.value.name = 'not-found'
    await requestFail('/games/game-1')
    await flush()
    expect(routerMock.replace).not.toHaveBeenCalled()
  })
})

describe('response interceptor 500 handling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows a server error alert on 500', async () => {
    await requestFail('/games', 500)
    await vi.waitFor(() =>
      expect(uiStoreMock.addAlert).toHaveBeenCalledWith('服务器内部错误', 'error')
    )
  })
})
