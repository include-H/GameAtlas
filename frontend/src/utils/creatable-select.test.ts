import { describe, expect, it, vi } from 'vitest'

import {
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

  it('merges selected options into search results', async () => {
    const search = vi.fn().mockResolvedValue([{ id: 2, name: '战棋' }])

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
