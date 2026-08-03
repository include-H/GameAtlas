import { describe, expect, it, vi } from 'vitest'

import {
  canCreateRemoteSearchedOption,
  dedupeCreatableOptionsByName,
  hasCreatableOptionName,
  normalizeOptionId,
  resolveCreatableSelections,
  searchCreatableOptions,
  sortCreatableOptionsByName,
} from './creatable-select'

describe('creatable-select helpers', () => {
  it('normalizes numeric option ids only', () => {
    expect(normalizeOptionId(12)).toBe(12)
    expect(normalizeOptionId(NaN)).toBeNull()
    expect(normalizeOptionId('12')).toBeNull()
    expect(normalizeOptionId(null)).toBeNull()
  })

  it('sorts options by name without mutating the source array', () => {
    const source = [
      { id: 2, name: '战棋' },
      { id: 1, name: '动作' },
    ]

    expect(sortCreatableOptionsByName(source)).toEqual([
      { id: 1, name: '动作' },
      { id: 2, name: '战棋' },
    ])
    expect(source).toEqual([
      { id: 2, name: '战棋' },
      { id: 1, name: '动作' },
    ])
  })

  it('deduplicates options by normalized display name', () => {
    expect(
      dedupeCreatableOptionsByName([
        { id: 1, name: '孤岛惊魂' },
        { id: 2, name: ' 孤岛惊魂 ' },
        { id: 3, name: 'Far  Cry' },
        { id: 4, name: 'far cry' },
      ]),
    ).toEqual([
      { id: 1, name: '孤岛惊魂' },
      { id: 3, name: 'Far  Cry' },
    ])
  })

  it('matches creatable option names after trimming and whitespace normalization', () => {
    expect(hasCreatableOptionName(' far cry ', [{ id: 1, name: 'Far  Cry' }])).toBe(true)
    expect(hasCreatableOptionName('孤岛惊魂', [{ id: 2, name: '孤岛惊魂' }])).toBe(true)
    expect(hasCreatableOptionName('孤岛惊魂4', [{ id: 2, name: '孤岛惊魂' }])).toBe(false)
  })

  it('only enables remote creation after the current query has resolved without an exact match', () => {
    const options = [{ id: 1, name: '黑手党' }]

    expect(canCreateRemoteSearchedOption('黑手党', '', options)).toBe(false)
    expect(canCreateRemoteSearchedOption('黑手党', '黑手党', options)).toBe(false)
    expect(canCreateRemoteSearchedOption('黑手党：新篇章', '黑手党', options)).toBe(false)
    expect(canCreateRemoteSearchedOption('黑手党：新篇章', '黑手党：新篇章', options)).toBe(true)
  })

  it('merges selected options into search results and removes duplicate names', async () => {
    const search = vi.fn().mockResolvedValue([
      { id: 2, name: '战棋' },
      { id: 3, name: ' 战棋 ' },
    ])

    const result = await searchCreatableOptions({
      query: '战',
      selectedValues: [1],
      currentOptions: [
        { id: 1, name: '动作' },
        { id: 2, name: '战棋' },
      ],
      search,
    })

    expect(search).toHaveBeenCalledWith('战')
    expect(result).toEqual([
      { id: 2, name: '战棋' },
      { id: 1, name: '动作' },
    ])
  })

  it('reuses existing options and creates missing ones when resolving selections', async () => {
    const createItem = vi.fn().mockImplementation(async (name: string) => ({
      id: 10,
      name,
    }))

    const result = await resolveCreatableSelections({
      values: [1, '动作', ' 新标签 ', 1, ''],
      options: [{ id: 1, name: '动作' }],
      createItem,
    })

    expect(createItem).toHaveBeenCalledTimes(1)
    expect(createItem).toHaveBeenCalledWith('新标签')
    expect(result).toEqual({
      ids: [1, 10],
      options: [
        { id: 1, name: '动作' },
        { id: 10, name: '新标签' },
      ],
    })
  })
})
