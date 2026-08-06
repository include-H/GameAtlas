import { beforeEach, describe, expect, it, vi } from 'vitest'

const { postMock } = vi.hoisted(() => ({
  postMock: vi.fn(),
}))

vi.mock('./api', () => ({
  post: postMock,
}))

import downloadService from './download.service'

describe('download service', () => {
  beforeEach(() => {
    postMock.mockReset()
  })

  it('records a game file download through the expected endpoint', async () => {
    postMock.mockResolvedValue({
      data: { recorded: true },
    })

    await expect(downloadService.recordDownload('game-1', '12')).resolves.toEqual({
      recorded: true,
    })
    expect(postMock).toHaveBeenCalledWith('/games/game-1/files/12/downloads')
  })
})
