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
  description?: string
  sort_order?: number
  created_at: string
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
  cover_image?: string | null
  banner_image?: string | null
  preview_video?: VideoAssetItem | null
  screenshots?: string[]
  screenshot_items?: ScreenshotItem[]
  wiki_content?: string | null
  wiki_content_html?: string | null
  file_path?: string
  file_paths?: Array<string | { id?: number; path: string; label?: string; notes?: string; size?: number | null; sort_order?: number }>
  files?: GameFileEntry[]
  needs_review?: boolean
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
  changelog?: string
}

export interface GameStats {
  total_games: number
  total_downloads: number
  total_views: number
  total_size: number
  recent_games: Game[]
  popular_games: Game[]
  favorite_games: Game[]
  pending_reviews: number
}

export interface GameFilter {
  search?: string
  series?: string
  platform?: string
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
