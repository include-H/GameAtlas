import { describe, expect, it } from 'vitest'
import {
  createEditableBanner,
  createEditableCover,
  createEditableLogo,
  createEditableScreenshot,
  createEditableVideo,
  createScreenshotKey,
} from './edit-game-form'

describe('edit-game-form DTO factories', () => {
  describe('createScreenshotKey', () => {
    it('prefers asset_uid for backend assets', () => {
      expect(createScreenshotKey({ id: 1, asset_uid: 'uid-1', path: '/a.jpg' })).toBe('uid:uid-1')
    })

    it('falls back to db id when asset_uid is missing', () => {
      expect(createScreenshotKey({ id: 7, path: '/a.jpg' })).toBe('db:7')
    })

    it('uses path+index fallback for session-only assets', () => {
      const key = createScreenshotKey({ path: '/tmp/new.jpg' }, 3)
      expect(key).toMatch(/^path:\/tmp\/new\.jpg:3:\d+$/)
    })
  })

  describe('createEditableScreenshot', () => {
    it('builds a client_key for string paths', () => {
      const result = createEditableScreenshot('/tmp/shot.png', 1)
      expect(result.path).toBe('/tmp/shot.png')
      expect(result.client_key).toMatch(/^path:\/tmp\/shot\.png:1:\d+$/)
    })

    it('keeps backend id and asset_uid for ScreenshotItem', () => {
      const result = createEditableScreenshot(
        { id: 5, asset_uid: 'shot-5', path: '/assets/shot-5.jpg', sort_order: 0 },
        0,
      )
      expect(result).toMatchObject({ id: 5, asset_uid: 'shot-5', path: '/assets/shot-5.jpg' })
      expect(result.client_key).toBe('uid:shot-5')
    })

    it('drops upload-only assets to session keys', () => {
      const result = createEditableScreenshot({ path: '/uploads/new.jpg', asset_uid: 'new' }, 2)
      expect(result.client_key).toBe('uid:new')
      expect(result.id).toBeUndefined()
    })
  })

  describe('createEditableVideo', () => {
    it('maps string paths', () => {
      expect(createEditableVideo('/tmp/v.mp4')).toEqual({ path: '/tmp/v.mp4' })
    })

    it('keeps poster_path for VideoAssetItem', () => {
      const result = createEditableVideo({
        id: 9,
        asset_uid: 'v-9',
        path: '/assets/v-9.mp4',
        poster_path: '/assets/v-9.jpg',
        sort_order: 0,
      })
      expect(result).toEqual({
        id: 9,
        asset_uid: 'v-9',
        path: '/assets/v-9.mp4',
        poster_path: '/assets/v-9.jpg',
      })
    })

    it('defaults poster_path to null for upload results', () => {
      const result = createEditableVideo({ path: '/uploads/v.mp4', asset_uid: 'v' })
      expect(result.poster_path).toBeNull()
    })
  })

  describe('createEditableCover / createEditableBanner', () => {
    it('maps backend items and string paths', () => {
      expect(createEditableCover({
        id: 3,
        asset_uid: 'c-3',
        path: '/assets/c-3.jpg',
        sort_order: 0,
      })).toEqual({ id: 3, asset_uid: 'c-3', path: '/assets/c-3.jpg' })
      expect(createEditableCover('/tmp/c.jpg')).toEqual({ path: '/tmp/c.jpg' })
      expect(createEditableBanner({
        id: 4,
        asset_uid: 'b-4',
        path: '/assets/b-4.jpg',
        sort_order: 0,
      })).toEqual({ id: 4, asset_uid: 'b-4', path: '/assets/b-4.jpg' })
      expect(createEditableBanner('/tmp/b.jpg')).toEqual({ path: '/tmp/b.jpg' })
    })
  })

  describe('createEditableLogo', () => {
    it('maps string paths with null positioning', () => {
      expect(createEditableLogo('/tmp/l.png')).toEqual({
        path: '/tmp/l.png',
        position_x: null,
        position_y: null,
        width_pct: null,
      })
    })

    it('keeps position fields for LogoItem', () => {
      const result = createEditableLogo({
        id: 6,
        asset_uid: 'l-6',
        path: '/assets/l-6.png',
        sort_order: 0,
        position_x: 12,
        position_y: 34,
        width_pct: 56,
      })
      expect(result).toEqual({
        id: 6,
        asset_uid: 'l-6',
        path: '/assets/l-6.png',
        position_x: 12,
        position_y: 34,
        width_pct: 56,
      })
    })

    it('defaults positions to null for upload results', () => {
      const result = createEditableLogo({ path: '/uploads/l.png', asset_uid: 'l' })
      expect(result.position_x).toBeNull()
      expect(result.position_y).toBeNull()
      expect(result.width_pct).toBeNull()
    })
  })
})
