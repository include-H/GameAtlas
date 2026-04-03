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

export interface EditGameForm {
  title: string
  title_alt: string
  visibility: 'public' | 'private'
  developer_ids: Array<string | number>
  publisher_ids: Array<string | number>
  release_date: string | undefined
  engine: string
  platform_ids: Array<string | number>
  series_id: string | number | null
  tag_ids: Array<string | number>
  summary: string
  cover_image: string
  banner_image: string
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
  engine: '',
  platform_ids: [],
  series_id: null,
  tag_ids: [],
  summary: '',
  cover_image: '',
  banner_image: '',
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
