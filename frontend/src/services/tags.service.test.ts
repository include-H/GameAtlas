import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}))

vi.mock('./api', () => ({
  get: getMock,
  post: postMock,
}))

import tagsService from './tags.service'

describe('tags service', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
  })

  it('loads tag groups from the api envelope', async () => {
    getMock.mockResolvedValue({
      data: [{ id: 1, key: 'genre', name: '题材' }],
    })

    await expect(tagsService.getTagGroups()).resolves.toEqual([
      { id: 1, key: 'genre', name: '题材' },
    ])
    expect(getMock).toHaveBeenCalledWith('/tag-groups')
  })

  it('builds tag query params from provided filters', async () => {
    getMock.mockResolvedValue({
      data: [{ id: 2, name: '动作' }],
    })

    const result = await tagsService.getTags({
      group_id: 3,
      group_key: 'genre',
      active: false,
    })

    expect(result).toEqual([{ id: 2, name: '动作' }])
    expect(getMock).toHaveBeenCalledTimes(1)
    expect(getMock.mock.calls[0]?.[0]).toBe('/tags')

    const params = getMock.mock.calls[0]?.[1]?.params as URLSearchParams
    expect(params.get('group_id')).toBe('3')
    expect(params.get('group_key')).toBe('genre')
    expect(params.get('active')).toBe('false')
  })

  it('creates tag groups and tags via post', async () => {
    postMock
      .mockResolvedValueOnce({
        data: { id: 4, key: 'theme', name: '内容属性' },
      })
      .mockResolvedValueOnce({
        data: { id: 5, group_id: 4, name: '黑暗奇幻' },
      })

    await expect(
      tagsService.createTagGroup({
        key: 'theme',
        name: '内容属性',
      }),
    ).resolves.toEqual({ id: 4, key: 'theme', name: '内容属性' })

    await expect(
      tagsService.createTag({
        group_id: 4,
        name: '黑暗奇幻',
      }),
    ).resolves.toEqual({ id: 5, group_id: 4, name: '黑暗奇幻' })

    expect(postMock).toHaveBeenNthCalledWith(1, '/tag-groups', {
      key: 'theme',
      name: '内容属性',
    })
    expect(postMock).toHaveBeenNthCalledWith(2, '/tags', {
      group_id: 4,
      name: '黑暗奇幻',
    })
  })
})
