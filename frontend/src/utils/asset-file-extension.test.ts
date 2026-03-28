import { describe, expect, it } from 'vitest'

import { getAssetFileExtension } from './asset-file-extension'

describe('asset-file-extension', () => {
  it('maps known mime types to clean extensions', () => {
    expect(getAssetFileExtension('image/jpeg', 'cover')).toBe('jpg')
    expect(getAssetFileExtension('image/svg+xml', 'cover')).toBe('svg')
    expect(getAssetFileExtension('video/mp4', 'video')).toBe('mp4')
  })

  it('strips mime parameters before resolving the extension', () => {
    expect(getAssetFileExtension('image/jpeg; charset=binary', 'cover')).toBe('jpg')
  })

  it('falls back by asset type when mime is missing or unsupported', () => {
    expect(getAssetFileExtension('', 'cover')).toBe('jpg')
    expect(getAssetFileExtension('application/octet-stream', 'cover')).toBe('jpg')
    expect(getAssetFileExtension(undefined, 'video')).toBe('mp4')
  })
})
