import { beforeEach, describe, expect, it, vi } from 'vitest'

const { delMock, getMock, postMock, putMock } = vi.hoisted(() => ({
  delMock: vi.fn(),
  getMock: vi.fn(),
  postMock: vi.fn(),
  putMock: vi.fn(),
}))

vi.mock('./api', () => ({
  del: delMock,
  get: getMock,
  post: postMock,
  put: putMock,
}))

import { settingsService } from './settings.service'

describe('settings service', () => {
  beforeEach(() => {
    delMock.mockReset()
    getMock.mockReset()
    postMock.mockReset()
    putMock.mockReset()
  })

  it('loads config entries from the settings endpoint', async () => {
    getMock.mockResolvedValue({
      data: [
        {
          key: 'PORT',
          value: '3000',
          label: '监听端口',
          group: 'runtime',
        },
      ],
    })

    await expect(settingsService.getConfig()).resolves.toEqual([
      {
        key: 'PORT',
        value: '3000',
        label: '监听端口',
        group: 'runtime',
      },
    ])
    expect(getMock).toHaveBeenCalledWith('/settings/config')
  })

  it('updates config through the settings endpoint', async () => {
    putMock.mockResolvedValue({
      data: { message: '配置已保存' },
    })

    await expect(settingsService.updateConfig({ PORT: '3001' })).resolves.toEqual({
      message: '配置已保存',
    })
    expect(putMock).toHaveBeenCalledWith('/settings/config', { PORT: '3001' })
  })

  it('uploads background as multipart form data', async () => {
    postMock.mockResolvedValue({
      data: { message: '背景图片已上传' },
    })
    const file = new File(['bg'], 'bg.jpg', { type: 'image/jpeg' })

    await expect(settingsService.uploadBackground(file)).resolves.toEqual({
      message: '背景图片已上传',
    })
    expect(postMock).toHaveBeenCalledWith('/settings/bg', expect.any(FormData), {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    const form = postMock.mock.calls[0]?.[1] as FormData | undefined
    expect(form?.get('bg')).toBe(file)
  })

  it('removes background and restarts the service', async () => {
    delMock.mockResolvedValue({
      data: { message: '背景图片已删除' },
    })
    postMock.mockResolvedValue({
      data: { message: '正在重启服务...' },
    })

    await expect(settingsService.removeBackground()).resolves.toEqual({
      message: '背景图片已删除',
    })
    await expect(settingsService.restart()).resolves.toEqual({
      message: '正在重启服务...',
    })
    expect(delMock).toHaveBeenCalledWith('/settings/bg')
    expect(postMock).toHaveBeenCalledWith('/settings/restart', {})
  })
})
