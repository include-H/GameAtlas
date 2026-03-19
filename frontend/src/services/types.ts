export interface ApiResponse<T> {
  success?: boolean
  data: T
  message?: string
  error?: string
}

export interface PaginatedResponse<T> {
  data: T[]
  pagination: {
    page: number
    limit: number
    total: number
    totalPages: number
  }
}

export interface Series {
  id: number
  name: string
  slug?: string
  sort_order?: number
  created_at: string
  game_count?: number
  cover_image?: string | null
  latest_updated_at?: string | null
}

export interface Platform {
  id: number
  name: string
  slug: string
  sort_order: number
  created_at: string
}

export interface Developer {
  id: number
  name: string
  slug: string
  sort_order: number
  created_at: string
}

export interface Publisher {
  id: number
  name: string
  slug: string
  sort_order: number
  created_at: string
}

export interface Tag {
  id: number
  group_id: number
  group_key: string
  group_name: string
  name: string
  slug: string
  parent_id?: number | null
  sort_order: number
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface TagGroup {
  id: number
  key: string
  name: string
  description?: string | null
  sort_order: number
  allow_multiple: boolean
  is_filterable: boolean
  created_at: string
  updated_at: string
}

export interface GameTagGroup {
  id: number
  key: string
  name: string
  allow_multiple: boolean
  is_filterable: boolean
  tags: Tag[]
}

export interface GameFileEntry {
  id: number
  game_id: number
  file_path: string
  label?: string | null
  notes?: string | null
  size_bytes?: number | null
  sort_order: number
  created_at: string
  updated_at: string
}

export interface ScreenshotItem {
  id?: number
  asset_uid?: string
  path: string
  sort_order?: number
}

export interface VideoAssetItem {
  id?: number
  asset_uid?: string
  path: string
  sort_order?: number
}

export interface Game {
  id: number
  title: string
  title_alt?: string | null
  visibility?: 'public' | 'private'
  summary?: string | null
  developer?: string
  publisher?: string
  release_date?: string | null
  engine?: string | null
  platform?: string
  platforms?: string[]
  series?: Series[]
  developers?: Developer[]
  publishers?: Publisher[]
  tags?: Tag[]
  tag_groups?: GameTagGroup[]
  cover_image?: string | null
  banner_image?: string | null
  preview_video?: VideoAssetItem | null
  preview_videos?: VideoAssetItem[]
  screenshots?: string[]
  screenshot_items?: ScreenshotItem[]
  wiki_content?: string | null
  wiki_content_html?: string | null
  file_path?: string
  file_paths?: Array<string | { id?: number; path: string; label?: string; notes?: string; size?: number | null; sort_order?: number }>
  files?: GameFileEntry[]
  needs_review?: boolean
  primary_screenshot?: string | null
  screenshot_count?: number
  file_count?: number
  developer_count?: number
  publisher_count?: number
  platform_count?: number
  views: number
  downloads: number
  created_at: string
  updated_at: string
  wiki_updated_at?: string
  isFavorite?: boolean
}

export interface GameInput {
  title: string
  title_alt?: string | null
  visibility?: 'public' | 'private'
  summary?: string | null
  release_date?: string | null
  engine?: string | null
  cover_image?: string | null
  banner_image?: string | null
  needs_review?: boolean
  series?: number[]
  developers?: number[]
  publishers?: number[]
  platforms?: Array<number | string>
  preview_video_asset_uid?: string | null
  tag_ids?: number[]
  screenshots?: string[]
  file_paths?: string[]
}

export interface GameVersion {
  id: string
  gameId: string
  version: string
  buildNumber?: string
  releaseDate: string
  size: number
  checksum?: string
  isLatest: boolean
  downloadUrl?: string
  launchScriptUrl?: string
  changelog?: string
}

export interface GameStats {
  total_games: number
  total_downloads: number
  total_views: number
  total_size: number
  recent_games: Game[]
  popular_games: Game[]
  favorite_count: number
  pending_reviews: number
}

export interface GameFilter {
  search?: string
  series?: string
  platform?: string
  tag_ids?: number[]
  favorite?: boolean
  status?: string
}

export interface GameSort {
  field: 'title' | 'created_at' | 'updated_at' | 'views' | 'downloads'
  order: 'asc' | 'desc'
}

export interface FileInfo {
  name: string
  path: string
  isDirectory: boolean
  size?: number | null
  extension?: string
}

export interface BrowseResponse {
  currentPath: string
  items: FileInfo[]
  parentPath: string | null
}

export interface SteamGameSearchResult {
  id: string
  name: string
  releaseDate?: string
  tinyImage?: string
}

export interface SteamGameDetails {
  name: string
  originalName?: string
  description: string
  releaseDate: string
  developers: string[]
  publishers: string[]
  previewVideos: Array<{ url: string; name: string; isDash: boolean }>
  previewVideoUrl?: string
  previewVideoName?: string
  previewVideoDebug?: string[]
  genres: string[]
  tags: string[]
  platforms: string[]
  screenshots: string[]
  headerImage: string
  libraryHero?: string
  background?: string
  website?: string
}

export interface SteamFetchImagesResponse {
  coverImage: string
  bannerImage: string
  previewVideo?: string
  screenshots: string[]
}

export interface ReviewIssueOverride {
  id: number
  game_id: number
  issue_key: string
  status: string
  reason?: string | null
  created_at: string
  updated_at: string
}
