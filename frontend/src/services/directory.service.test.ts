import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}))

vi.mock('./api', () => ({
  get: getMock,
}))

import { directoryService } from './directory.service'

describe('directory service', () => {
  beforeEach(() => {
    getMock.mockReset()
  })

  it('returns the default directory path', async () => {
    getMock.mockResolvedValue({
      data: {
        path: '/roms',
      },
    })

    await expect(directoryService.getDefaultDirectory()).resolves.toBe('/roms')
    expect(getMock).toHaveBeenCalledWith('/directory/default')
  })

  it('maps directory list items to frontend shape', async () => {
    getMock.mockResolvedValue({
      data: {
        current_path: '/roms',
        parent_path: '/',
        items: [
          {
            name: 'PS2',
            path: '/roms/PS2',
            is_directory: true,
            size_bytes: null,
          },
          {
            name: 'game.iso',
            path: '/roms/game.iso',
            is_directory: false,
            size_bytes: 1234,
          },
        ],
      },
    })

    await expect(directoryService.listDirectory('/roms')).resolves.toEqual({
      currentPath: '/roms',
      parentPath: '/',
      items: [
        {
          name: 'PS2',
          path: '/roms/PS2',
          type: 'directory',
          size: null,
        },
        {
          name: 'game.iso',
          path: '/roms/game.iso',
          type: 'file',
          size: 1234,
        },
      ],
    })
    expect(getMock).toHaveBeenCalledWith('/directory/list', { params: { path: '/roms' } })
  })
})
