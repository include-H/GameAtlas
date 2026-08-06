import type { FilePathItem } from '@/composables/useGameFilePaths'
import type { UploadedAssetResult } from '@/services/assets'
import type {
  BannerItem,
  CoverItem,
  LogoItem,
  ScreenshotItem,
  VideoAssetItem,
} from '@/services/types'

export interface EditGameEditableScreenshot {
  id?: number
  asset_uid?: string
  path: string
  client_key: string
}

export interface EditGameEditableVideo {
  id?: number
  asset_uid?: string
  path: string
  poster_path?: string | null
}

export interface EditGameEditableCover {
  id?: number
  asset_uid?: string
  path: string
}

export interface EditGameEditableBanner {
  id?: number
  asset_uid?: string
  path: string
}

export interface EditGameEditableLogo {
  id?: number
  asset_uid?: string
  path: string
  position_x: number | null
  position_y: number | null
  width_pct: number | null
}

export interface LogoPositionChange {
  key: string
  position_x: number
  position_y: number
  width_pct: number
  logo_visible: boolean
}

export interface EditGameForm {
  title: string
  title_alt: string
  visibility: 'public' | 'private'
  developer_ids: number[]
  publisher_ids: number[]
  release_date: string | undefined
  series_id: number | null
  summary: string
  // The first item is always the primary cover.
  covers: EditGameEditableCover[]
  // The first item is always the primary banner.
  banners: EditGameEditableBanner[]
  // The first item is always the primary logo.
  logos: EditGameEditableLogo[]
  logo_visible: boolean
  // The first item is always the canonical preview video.
  preview_videos: EditGameEditableVideo[]
  screenshots: EditGameEditableScreenshot[]
  file_paths: FilePathItem[]
}

export const createEmptyEditGameForm = (): EditGameForm => ({
  title: '',
  title_alt: '',
  visibility: 'public',
  developer_ids: [],
  publisher_ids: [],
  release_date: undefined,
  series_id: null,
  summary: '',
  covers: [],
  banners: [],
  logos: [],
  logo_visible: true,
  preview_videos: [],
  screenshots: [],
  file_paths: [{ path: '', label: '' }],
})

export const parseEditGameReleaseDate = (value?: string | null): Date | null => {
  const normalized = value?.trim()
  if (!normalized) return null

  const parts = normalized.split('-')
  if (parts.length === 3) {
    return new Date(
      Number.parseInt(parts[0], 10),
      Number.parseInt(parts[1], 10) - 1,
      Number.parseInt(parts[2], 10),
    )
  }

  const parsed = new Date(normalized)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

export const formatEditGameReleaseDate = (
  value: Date | number | string | null,
): string | undefined => {
  if (!value) return undefined

  const dateObj = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(dateObj.getTime())) return undefined

  const year = dateObj.getFullYear()
  const month = String(dateObj.getMonth() + 1).padStart(2, '0')
  const day = String(dateObj.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

// 会话新素材没有后端 id，靠 client_key 支撑拖拽排序与移除；已入库素材用 asset_uid / id 定位。
export const createScreenshotKey = (
  asset: Pick<EditGameEditableScreenshot, 'id' | 'asset_uid' | 'path'>,
  index = 0,
) => {
  if (asset.asset_uid) return `uid:${asset.asset_uid}`
  if (typeof asset.id === 'number') return `db:${asset.id}`
  return `path:${asset.path}:${index}:${Date.now()}`
}

export const createEditableScreenshot = (
  asset: ScreenshotItem | UploadedAssetResult | string,
  index: number,
): EditGameEditableScreenshot => {
  if (typeof asset === 'string') {
    return {
      path: asset,
      client_key: createScreenshotKey({ path: asset }, index),
    }
  }

  const screenshotId = 'id' in asset ? asset.id : undefined

  return {
    id: screenshotId,
    asset_uid: asset.asset_uid,
    path: asset.path,
    client_key: createScreenshotKey({
      id: screenshotId,
      asset_uid: asset.asset_uid,
      path: asset.path,
    }, index),
  }
}

export const createEditableVideo = (asset: VideoAssetItem | UploadedAssetResult | string): EditGameEditableVideo => {
  if (typeof asset === 'string') {
    return { path: asset }
  }
  return {
    id: 'id' in asset ? asset.id : undefined,
    asset_uid: asset.asset_uid,
    path: asset.path,
    poster_path: 'poster_path' in asset ? (asset.poster_path ?? null) : null,
  }
}

export const createEditableCover = (asset: CoverItem | UploadedAssetResult | string): EditGameEditableCover => {
  if (typeof asset === 'string') {
    return { path: asset }
  }
  return {
    id: 'id' in asset ? asset.id : undefined,
    asset_uid: asset.asset_uid,
    path: asset.path,
  }
}

export const createEditableBanner = (asset: BannerItem | UploadedAssetResult | string): EditGameEditableBanner => {
  if (typeof asset === 'string') {
    return { path: asset }
  }
  return {
    id: 'id' in asset ? asset.id : undefined,
    asset_uid: asset.asset_uid,
    path: asset.path,
  }
}

export const createEditableLogo = (asset: LogoItem | UploadedAssetResult | string): EditGameEditableLogo => {
  if (typeof asset === 'string') {
    return { path: asset, position_x: null, position_y: null, width_pct: null }
  }
  const isLogoItem = 'sort_order' in asset
  return {
    id: 'id' in asset ? asset.id : undefined,
    asset_uid: asset.asset_uid,
    path: asset.path,
    position_x: isLogoItem ? (asset as LogoItem).position_x ?? null : null,
    position_y: isLogoItem ? (asset as LogoItem).position_y ?? null : null,
    width_pct: isLogoItem ? (asset as LogoItem).width_pct ?? null : null,
  }
}
