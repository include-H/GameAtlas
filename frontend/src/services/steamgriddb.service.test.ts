import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}))

vi.mock('./api', () => ({
  get: getMock,
}))

import steamGridDBService from './steamgriddb.service'

describe('steamgriddb service', () => {
  beforeEach(() => {
    getMock.mockReset()
  })

  it('reports availability from the backend and treats failures as unavailable', async () => {
    getMock
      .mockResolvedValueOnce({ data: true })
      .mockResolvedValueOnce({ data: false })
      .mockRejectedValueOnce(new Error('network'))

    await expect(steamGridDBService.isAvailable()).resolves.toBe(true)
    await expect(steamGridDBService.isAvailable()).resolves.toBe(false)
    await expect(steamGridDBService.isAvailable()).resolves.toBe(false)
    expect(getMock).toHaveBeenNthCalledWith(1, '/steamgriddb/available')
    expect(getMock).toHaveBeenNthCalledWith(2, '/steamgriddb/available')
    expect(getMock).toHaveBeenNthCalledWith(3, '/steamgriddb/available')
  })

  it('trims search queries and skips blank searches', async () => {
    await expect(steamGridDBService.search('   ')).resolves.toEqual([])
    expect(getMock).not.toHaveBeenCalled()

    const games = [
      {
        id: 7,
        name: 'Portal',
        release_date: 1234567890,
        types: ['game'],
        verified: true,
      },
    ]
    getMock.mockResolvedValue({ data: games })

    await expect(steamGridDBService.search(' Portal ')).resolves.toEqual(games)
    expect(getMock).toHaveBeenCalledWith('/steamgriddb/search', {
      params: { q: 'Portal' },
    })
  })

  it('loads grids and heroes by game id', async () => {
    const image = {
      id: 9,
      score: 5,
      style: 'alternate',
      notes: 'alt',
      language: 'en',
      url: 'https://cdn.example.com/grid.png',
      thumb: 'https://cdn.example.com/thumb.png',
    }
    getMock
      .mockResolvedValueOnce({ data: [image] })
      .mockResolvedValueOnce({ data: null })

    await expect(steamGridDBService.getGridsByGameId(12)).resolves.toEqual([image])
    await expect(steamGridDBService.getHeroesByGameId(12)).resolves.toEqual([])
    expect(getMock).toHaveBeenNthCalledWith(1, '/steamgriddb/game/12/grids')
    expect(getMock).toHaveBeenNthCalledWith(2, '/steamgriddb/game/12/heroes')
  })

  it('loads logos by game id', async () => {
    getMock.mockResolvedValue({
      data: [],
    })

    await steamGridDBService.getLogosByGameId(12)

    expect(getMock).toHaveBeenCalledWith('/steamgriddb/game/12/logos')
  })
})
