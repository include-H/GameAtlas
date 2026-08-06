import { ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { useEditGameUploadUrls } from './useEditGameUploadUrls'

describe('useEditGameUploadUrls', () => {
  it('builds per-asset-type upload actions and data', () => {
    const urls = useEditGameUploadUrls({ gameId: ref(42) })
    expect(urls.uploadAction.value).toMatch(/\/assets\/cover$/)
    expect(urls.bannerUploadAction.value).toMatch(/\/assets\/banner$/)
    expect(urls.screenshotUploadAction.value).toMatch(/\/assets\/screenshot$/)
    expect(urls.logoUploadAction.value).toMatch(/\/assets\/logo$/)
    expect(urls.uploadData.value).toEqual({ game_id: '42' })
    expect(urls.bannerUploadData.value).toEqual({ game_id: '42' })
    expect(urls.screenshotUploadData.value).toEqual({ game_id: '42' })
    expect(urls.logoUploadData.value).toEqual({ game_id: '42' })
    expect(urls.uploadHeaders.value).toEqual({})
  })

  it('emits empty game_id when game is missing', () => {
    const urls = useEditGameUploadUrls({ gameId: ref(undefined) })
    expect(urls.uploadData.value).toEqual({ game_id: '' })
    expect(urls.logoUploadData.value).toEqual({ game_id: '' })
  })
})
