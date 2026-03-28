export interface ApiEnvelope<T> {
  success?: boolean
  data: T
  message?: string
  error?: string
}

export interface ApiPageEnvelope<T> {
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
  cover_candidates?: string[]
  latest_updated_at?: string | null
}

export interface SeriesDetail {
  series: Series
  games: GameListItemView[]
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
  file_name: string
  file_path?: string
  label?: string | null
  notes?: string | null
  size_bytes?: number | null
  sort_order: number
  source_created_at: string | null
  created_at: string
  updated_at: string
}

export interface ScreenshotItem {
  id: number
  asset_uid: string
  path: string
  sort_order: number
}

export interface VideoAssetItem {
  id: number
  asset_uid: string
  path: string
  sort_order: number
}

export interface GameListItemDto {
  id: number
  public_id: string
  title: string
  title_alt: string | null
  visibility: 'public' | 'private'
  summary: string | null
  release_date: string | null
  engine: string | null
  cover_image: string | null
  banner_image: string | null
  wiki_content: string | null
  wiki_content_html: string | null
  needs_review: boolean
  primary_screenshot: string | null
  screenshot_count: number
  file_count: number
  developer_count: number
  publisher_count: number
  platform_count: number
  downloads: number
  created_at: string
  updated_at: string
}

export interface GameDetailDto extends Omit<GameListItemDto, 'primary_screenshot' | 'screenshot_count' | 'file_count' | 'developer_count' | 'publisher_count' | 'platform_count'> {
  preview_video: VideoAssetItem | null
  preview_videos: VideoAssetItem[]
  screenshots: ScreenshotItem[]
  series: Series | null
  platforms: Platform[]
  developers: Developer[]
  publishers: Publisher[]
  tags: Tag[]
  tag_groups: GameTagGroup[]
  files: GameFileEntry[]
}

export interface TimelineGameResponse {
  id: number
  public_id: string
  title: string
  release_date: string | null
  cover_image: string | null
  banner_image: string | null
}

export interface GameWriteRequest {
  title: string
  title_alt?: string | null
  visibility?: 'public' | 'private'
  summary?: string | null
  release_date?: string | null
  engine?: string | null
  cover_image?: string | null
  banner_image?: string | null
  needs_review?: boolean
  series_id?: number | null
  developer_ids?: number[]
  publisher_ids?: number[]
  platform_ids?: number[]
  preview_video_asset_uid?: string | null
  tag_ids?: number[]
}

export interface GameListQuery {
  page?: number
  limit?: number
  search?: string
  series?: string
  platform?: string
  tag?: number[]
  needs_review?: boolean
  pending?: boolean
  favorite?: boolean
}

export interface Favoritable {
  isFavorite?: boolean
}

export type GameListItemView = GameListItemDto & Favoritable
export type GameDetailView = GameDetailDto & Favoritable
export type GameListItem = GameListItemView
export type GameDetail = GameDetailView
export type TimelineGame = TimelineGameResponse & Favoritable

export interface GameVersion {
  id: string
  gameId: string
  version: string
  buildNumber?: string
  releaseDate: string
  size: number
  checksum?: string
  isLatest: boolean
  canLaunch?: boolean
  downloadUrl?: string
  launchScriptUrl?: string
  changelog?: string
}

export interface GameStats {
  total_games: number
  total_downloads: number
  recent_games: GameListItem[]
  popular_games: GameListItem[]
  favorite_count: number
  pending_reviews: number
}

export interface GameSort {
  field: 'title' | 'created_at' | 'updated_at' | 'release_date' | 'downloads' | 'random'
  order: 'asc' | 'desc'
  seed?: number
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
