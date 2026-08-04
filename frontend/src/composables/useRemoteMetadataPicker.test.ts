import { describe, expect, it, vi } from 'vitest'

import {
  normalizeMetadataPickerID,
  normalizeMetadataPickerIDs,
  useRemoteMetadataPicker,
} from './useRemoteMetadataPicker'

describe('useRemoteMetadataPicker', () => {
  it('normalizes select values to unique positive integer ids', () => {
    expect(normalizeMetadataPickerID(4)).toBe(4)
    expect(normalizeMetadataPickerID('4')).toBeNull()
    expect(normalizeMetadataPickerID(0)).toBeNull()
    expect(normalizeMetadataPickerIDs([4, 2, 4, '3', 0, -1, 1.5])).toEqual([4, 2])
  })

  it('does not enable creation until the current search finishes without an exact match', async () => {
    let resolveSearch: (items: Array<{ id: number; name: string }>) => void = () => {}
    const picker = useRemoteMetadataPicker({
      selectedIds: () => [],
      search: () => new Promise<Array<{ id: number; name: string }>>((resolve) => {
        resolveSearch = resolve
      }),
      create: vi.fn(),
    })

    const searchPromise = picker.search('黑手党')
    expect(picker.canCreate.value).toBe(false)

    resolveSearch([{ id: 1, name: '黑手党' }])
    await searchPromise

    expect(picker.canCreate.value).toBe(false)
  })

  it('ignores stale search responses', async () => {
    let resolveFirst: (items: Array<{ id: number; name: string }>) => void = () => {}
    let resolveSecond: (items: Array<{ id: number; name: string }>) => void = () => {}
    const search = vi.fn()
      .mockImplementationOnce(() => new Promise<Array<{ id: number; name: string }>>((resolve) => {
        resolveFirst = resolve
      }))
      .mockImplementationOnce(() => new Promise<Array<{ id: number; name: string }>>((resolve) => {
        resolveSecond = resolve
      }))
    const picker = useRemoteMetadataPicker({
      selectedIds: () => [],
      search,
      create: vi.fn(),
    })

    const first = picker.search('黑')
    const second = picker.search('黑手党')
    resolveSecond([{ id: 2, name: '黑手党' }])
    await second
    resolveFirst([{ id: 1, name: '黑' }])
    await first

    expect(picker.options.value).toEqual([{ id: 2, name: '黑手党' }])
    expect(picker.isSearching.value).toBe(false)
  })

  it('cancels a pending search when the query is cleared', async () => {
    let resolveSearch: (items: Array<{ id: number; name: string }>) => void = () => {}
    const picker = useRemoteMetadataPicker({
      selectedIds: () => [],
      search: () => new Promise<Array<{ id: number; name: string }>>((resolve) => {
        resolveSearch = resolve
      }),
      create: vi.fn(),
    })

    const searchPromise = picker.search('待取消')
    picker.clearSearch()
    resolveSearch([{ id: 1, name: '待取消' }])
    await searchPromise

    expect(picker.query.value).toBe('')
    expect(picker.options.value).toEqual([])
    expect(picker.canCreate.value).toBe(false)
  })

  it('creates a selected metadata item from the explicit create action', async () => {
    const create = vi.fn().mockResolvedValue({ id: 3, name: '新系列' })
    const picker = useRemoteMetadataPicker({
      selectedIds: () => [],
      search: vi.fn().mockResolvedValue([]),
      create,
    })

    await picker.search('新系列')
    const created = await picker.createFromQuery()

    expect(create).toHaveBeenCalledWith('新系列')
    expect(created).toEqual({ id: 3, name: '新系列' })
    expect(picker.options.value).toEqual([{ id: 3, name: '新系列' }])
  })

  it('resolves an exact query match instead of selecting the first search result', async () => {
    const create = vi.fn()
    const picker = useRemoteMetadataPicker({
      selectedIds: () => [],
      search: vi.fn().mockResolvedValue([
        { id: 1, name: '使命召唤2' },
        { id: 2, name: '使命召唤' },
      ]),
      create,
    })

    await picker.search('使命召唤')

    await expect(picker.resolveQuery()).resolves.toEqual({ id: 2, name: '使命召唤' })
    expect(create).not.toHaveBeenCalled()
  })

  it('searches and creates an unresolved query when resolving directly', async () => {
    const create = vi.fn().mockResolvedValue({ id: 3, name: '新系列' })
    const search = vi.fn().mockResolvedValue([{ id: 1, name: '已有系列' }])
    const picker = useRemoteMetadataPicker({
      selectedIds: () => [],
      search,
      create,
    })

    picker.query.value = '新系列'

    await expect(picker.resolveQuery()).resolves.toEqual({ id: 3, name: '新系列' })
    expect(search).toHaveBeenCalledWith('新系列')
    expect(create).toHaveBeenCalledWith('新系列')
  })

  it('resolves imported names to ids without duplicating normalized options', async () => {
    const create = vi.fn().mockResolvedValue({ id: 3, name: '新开发商' })
    const picker = useRemoteMetadataPicker({
      selectedIds: () => [],
      search: vi.fn(),
      create,
    })
    picker.options.value = [{ id: 1, name: '已有开发商' }]

    await expect(picker.ensureNames([' 已有开发商 ', '新开发商', '新开发商'])).resolves.toEqual([1, 3])
    expect(create).toHaveBeenCalledTimes(1)
  })
})
