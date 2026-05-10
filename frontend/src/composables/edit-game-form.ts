import type { FilePathItem } from '@/composables/useGameFilePaths'

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
}

export interface EditGameEditableCover {
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

export interface EditGameForm {
  title: string
  title_alt: string
  visibility: 'public' | 'private'
  // 2026-04-04: these fields intentionally accept existing ids or new names.
  // Impact: the edit modal remains the canonical place where metadata can be created during game editing.
  developer_ids: Array<string | number>
  publisher_ids: Array<string | number>
  release_date: string | undefined
  series_id: string | number | null
  summary: string
  banner_image: string
  // The first item is always the primary cover.
  covers: EditGameEditableCover[]
  logo: EditGameEditableLogo | null
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
  banner_image: '',
  covers: [],
  logo: null,
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
