import { describe, expect, it } from 'vitest'
import { shouldResetVideoPlaybackState } from './video-playback'

describe('shouldResetVideoPlaybackState', () => {
  it('resets playback state when the video source changes', () => {
    expect(shouldResetVideoPlaybackState('/assets/a.mp4', '/assets/b.mp4')).toBe(true)
    expect(shouldResetVideoPlaybackState(undefined, '/assets/a.mp4')).toBe(true)
  })

  it('keeps playback state when the video source is unchanged', () => {
    expect(shouldResetVideoPlaybackState('/assets/a.mp4', '/assets/a.mp4')).toBe(false)
  })
})
