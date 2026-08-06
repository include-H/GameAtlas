import { describe, expect, it, vi } from 'vitest'

import { useSteamPicker } from './useSteamPicker'
import type { SteamGameSearchResult } from '@/services/types'

const game: SteamGameSearchResult = {
  id: '1',
  name: 'Splinter Cell',
}

describe('useSteamPicker external selection', () => {
  it('keeps the selected game and clears the searching state after the loader succeeds', async () => {
    const picker = useSteamPicker<string[]>({ onSelect: vi.fn() })
    const load = vi.fn().mockResolvedValue(undefined)

    await picker.selectExternal(game, load)

    expect(picker.selectedGame.value).toEqual(game)
    expect(picker.isSearching.value).toBe(false)
    expect(load).toHaveBeenCalledTimes(1)
  })

  it('clears the selection when the external loader fails', async () => {
    const picker = useSteamPicker<string[]>({ onSelect: vi.fn() })
    const load = vi.fn().mockRejectedValue(new Error('load failed'))

    await expect(picker.selectExternal(game, load)).rejects.toThrow('load failed')

    expect(picker.selectedGame.value).toBeNull()
    expect(picker.isSearching.value).toBe(false)
  })
})
