import { beforeEach, describe, expect, it, vi } from 'vitest'

const { postMock } = vi.hoisted(() => ({
  postMock: vi.fn(),
}))

vi.mock('./api', () => ({
  default: {
    post: postMock,
  },
}))

import { uploadAsset } from './assets'

describe('assets service', () => {
  beforeEach(() => {
    postMock.mockReset()
  })

  it('uploads assets with multipart payload and returns response data', async () => {
    const progressValues: number[] = []
    let capturedForm: FormData | undefined

    postMock.mockImplementation(async (_url: string, form: FormData, config: { onUploadProgress?: (event: { loaded: number; total?: number }) => void }) => {
      capturedForm = form
      config.onUploadProgress?.({ loaded: 25, total: 100 })
      config.onUploadProgress?.({ loaded: 200, total: 100 })

      return {
        data: {
          data: {
            path: '/assets/cover.png',
            asset_id: 9,
            asset_uid: 'asset-9',
          },
        },
      }
    })

    const file = new File(['hello'], 'cover.png', { type: 'image/png' })
    const result = await uploadAsset('cover', 12, file, 3, (value) => {
      progressValues.push(value)
    })

    expect(postMock).toHaveBeenCalledWith(
      '/assets/cover',
      expect.any(FormData),
      expect.objectContaining({
        headers: { 'Content-Type': 'multipart/form-data' },
      }),
    )
    expect(capturedForm?.get('game_id')).toBe('12')
    expect(capturedForm?.get('sort_order')).toBe('3')
    expect(capturedForm?.get('file')).toBe(file)
    expect(progressValues).toEqual([25, 100])
    expect(result).toEqual({
      path: '/assets/cover.png',
      asset_id: 9,
      asset_uid: 'asset-9',
    })
  })

  it('skips progress updates when total is missing or callback is absent', async () => {
    postMock.mockImplementation(async (_url: string, _form: FormData, config: { onUploadProgress?: (event: { loaded: number; total?: number }) => void }) => {
      config.onUploadProgress?.({ loaded: 25 })
      return {
        data: {
          data: {
            path: '/assets/video.mp4',
          },
        },
      }
    })

    const file = new File(['video'], 'clip.mp4', { type: 'video/mp4' })
    await expect(uploadAsset('video', 3, file)).resolves.toEqual({
      path: '/assets/video.mp4',
    })
  })
})
