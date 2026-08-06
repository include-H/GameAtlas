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

import startScreenService from './start-screen.service'

describe('start screen service', () => {
  beforeEach(() => {
    delMock.mockReset()
    getMock.mockReset()
    postMock.mockReset()
    putMock.mockReset()
  })

  it('loads start screen layout with normalized empty collections', async () => {
    getMock.mockResolvedValue({
      data: {
        columns: [{ id: 1, name: 'Main', sort_order: 0 }],
        tiles: [],
      },
    })

    await expect(startScreenService.getTiles()).resolves.toEqual({
      columns: [{ id: 1, name: 'Main', sort_order: 0 }],
      tiles: [],
    })
    expect(getMock).toHaveBeenCalledWith('/start-screen/tiles')
  })

  it('updates start screen layout', async () => {
    putMock.mockResolvedValue({
      data: {
        columns: [],
        tiles: [],
      },
    })
    const input = {
      columns: [{ name: 'Main' }],
      tiles: [],
    }

    await expect(startScreenService.updateTiles(input)).resolves.toEqual({
      columns: [],
      tiles: [],
    })
    expect(putMock).toHaveBeenCalledWith('/start-screen/tiles', input)
  })

  it('adds and removes start screen tiles', async () => {
    postMock.mockResolvedValue({
      data: {
        columns: [],
        tiles: [],
      },
    })
    delMock.mockResolvedValue({
      data: {
        columns: [],
        tiles: [],
      },
    })

    await startScreenService.addTile(7, 'wide')
    await startScreenService.removeTile(7)

    expect(postMock).toHaveBeenCalledWith('/start-screen/tiles', {
      game_id: 7,
      tile_size: 'wide',
    })
    expect(delMock).toHaveBeenCalledWith('/start-screen/tiles/7')
  })

  it('uploads tile image as multipart form data', async () => {
    postMock.mockResolvedValue({
      data: {
        path: '/assets/start-screen/tile.png',
      },
    })
    const file = new File(['tile'], 'tile.png', { type: 'image/png' })

    await expect(startScreenService.uploadTileImage(file, 'large')).resolves.toBe(
      '/assets/start-screen/tile.png',
    )
    expect(postMock).toHaveBeenCalledWith('/start-screen/tiles/image', expect.any(FormData), {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    const form = postMock.mock.calls[0]?.[1] as FormData | undefined
    expect(form?.get('size')).toBe('large')
    expect(form?.get('file')).toBe(file)
  })
})
