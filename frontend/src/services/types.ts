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
    pending_issue_counts?: PendingIssueCounts | null
  }
}

export interface MetadataPagination {
  page: number
  limit: number
  total: number
  totalPages: number
}

export interface PendingIssueCounts {
  groups: Record<string, number>
  ignored_total: number
}

export interface PendingIssueDefinition {
  key: string
  label: string
  description: string
}

export interface PendingIssueDetailDefinition {
  key: string
  label: string
  group: string
}

export interface PendingIssueCatalog {
  groups: PendingIssueDefinition[]
  details: PendingIssueDetailDefinition[]
}

export interface PendingIssueDetailState {
  key: string
  group: string
  ignored: boolean
  reason?: string | null
}

export interface PendingIssueEvaluation {
  groups: string[]
  details: PendingIssueDetailState[]
  severe: boolean
}

export interface Series {
  id: number
  name: string
  slug: string
  sort_order: number
  created_at: string
  game_count?: number
  cover_image?: string | null
  cover_candidates?: string[]
  background_candidates?: string[]
  latest_updated_at?: string | null
}

export interface SeriesDetail {
  series: Series
  games: GameListItemView[]
  pagination: MetadataPagination
}

export interface Developer {
  id: number
  name: string
  slug: string
  sort_order: number
  created_at: string
  game_count?: number
  cover_image?: string | null
  cover_candidates?: string[]
  background_candidates?: string[]
  latest_updated_at?: string | null
}

export interface Publisher {
  id: number
  name: string
  slug: string
  sort_order: number
  created_at: string
  game_count?: number
  cover_image?: string | null
  cover_candidates?: string[]
  background_candidates?: string[]
  latest_updated_at?: string | null
}

export interface PublisherDetail {
  publisher: Publisher
  games: GameListItemView[]
  pagination: MetadataPagination
}


export interface PublicGameFileEntry {
  id: number
  game_id: number
  file_name: string
  label: string | null
  notes: string | null
  size_bytes: number | null
  sort_order: number
  source_created_at: string | null
  created_at: string
  updated_at: string
}

export interface AdminGameFileEntry extends PublicGameFileEntry {
  file_path: string
}

export type GameFileEntry = PublicGameFileEntry | AdminGameFileEntry

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
  poster_path: string | null
  sort_order: number
}

export interface CoverItem {
  id: number
  asset_uid: string
  path: string
  sort_order: number
}

export interface BannerItem {
  id: number
  asset_uid: string
  path: string
  sort_order: number
}

export interface LogoItem {
  id: number
  asset_uid: string
  path: string
  sort_order: number
  position_x: number | null
  position_y: number | null
  width_pct: number | null
}

export interface GameListItemDto {
  id: number
  public_id: string
  title: string
  title_alt: string | null
  visibility: 'public' | 'private'
  summary: string | null
  release_date: string | null
  cover_image: string | null
  banner_image: string | null
  wiki_content: string | null
  primary_screenshot: string | null
  screenshot_count?: number
  logo_visible: boolean
  file_count?: number
  developer_count?: number
  publisher_count?: number
  is_favorite: boolean
  series: GameSeriesSummary | null
  pending_issues?: PendingIssueEvaluation
  downloads: number
  created_at: string
  updated_at: string
}

export interface GameSeriesSummary {
  id: number
  name: string
}

interface GameDetailDtoBase<TFile extends GameFileEntry = GameFileEntry> extends Omit<GameListItemDto, 'primary_screenshot' | 'screenshot_count' | 'file_count' | 'developer_count' | 'publisher_count'> {
  preview_videos: VideoAssetItem[]
  screenshots: ScreenshotItem[]
  covers: CoverItem[]
  banners: BannerItem[]
  logos: LogoItem[]
  series: Series | null
  developers: Developer[]
  publishers: Publisher[]
  files: TFile[]
}

type PublicGameDetailDto = GameDetailDtoBase<PublicGameFileEntry>
export type AdminGameDetailDto = GameDetailDtoBase<AdminGameFileEntry>
export type GameDetailDto = PublicGameDetailDto | AdminGameDetailDto

export interface TimelineGameResponse {
  id: number
  public_id: string
  title: string
  release_date: string | null
  cover_image: string | null
  banner_image: string | null
}

export interface GameCreateRequest {
  title: string
  visibility?: 'public' | 'private'
}

interface GameAggregateCoreRequest {
  title: string
  title_alt: string | null
  visibility: 'public' | 'private'
  summary: string | null
  release_date: string | null
  logo_visible: boolean
}

// Aggregate update rewrites the editable game aggregate in one request.
export interface GameAggregateGameUpdateRequest extends GameAggregateCoreRequest {
  series_id: number | null
  developer_ids: number[]
  publisher_ids: number[]
}

interface GameAggregateFileRequest {
  id?: number
  file_path: string
  label?: string | null
  notes?: string | null
}

export interface GameAggregateNewAsset {
  asset_uid: string
  asset_type: string
  path: string
  poster_path?: string | null
}

export interface GameAggregateUpdateRequest {
  game: GameAggregateGameUpdateRequest
  assets: {
    files: GameAggregateFileRequest[]
    new_assets: GameAggregateNewAsset[]
    screenshot_order_asset_uids: string[]
    video_order_asset_uids: string[]
    cover_order_asset_uids: string[]
    logo_order_asset_uids: string[]
    banner_order_asset_uids: string[]
    logo_positions: LogoPosition[]
  }
}

interface LogoPosition {
  asset_uid: string
  position_x: number | null
  position_y: number | null
  width_pct: number | null
}

export interface GameListQuery {
  page?: number
  limit?: number
  search?: string
  series?: number
  visibility?: 'public' | 'private'
  pending?: boolean
  pending_issue?: string
  pending_include_ignored?: boolean
  pending_severe?: boolean
  pending_recent_days?: number
  favorite?: boolean
  favorite_raw?: string
}

interface Favoritable {
  isFavorite: boolean
}

type GameListItemView = Omit<GameListItemDto, 'is_favorite'> & Favoritable
export type GameDetailView<TFile extends GameFileEntry = GameFileEntry> = Omit<GameDetailDtoBase<TFile>, 'is_favorite'> & Favoritable
export type GameListItem = GameListItemView
export type PublicGameDetail = GameDetailView<PublicGameFileEntry>
export type AdminGameDetail = GameDetailView<AdminGameFileEntry>
export type GameDetail = PublicGameDetail | AdminGameDetail
export type TimelineGame = TimelineGameResponse

// Keep the public/admin detail contract explicit in the frontend because the backend
// serves both shapes from the same route depending on auth state. Without this split,
// edit bootstrap code silently drifts back to optional file_path handling.
const hasResolvedGameFilePath = (file: GameFileEntry): file is AdminGameFileEntry => {
  return 'file_path' in file && typeof file.file_path === 'string' && file.file_path.trim().length > 0
}

export const isAdminGameDetail = (game: GameDetail | null | undefined): game is AdminGameDetail => {
  if (!game) return false
  return game.files.every((file) => hasResolvedGameFilePath(file))
}

export interface GameVersion {
  id: string
  version: string
  releaseDate: string
  size: number
  isLatest: boolean
  canLaunch?: boolean
  downloadUrl?: string
  launchScriptUrl?: string
}

export interface GameStats {
  total_games: number
  total_downloads: number
  recent_games: GameListItem[]
  recently_updated_games: GameListItem[]
  popular_games: GameListItem[]
  favorite_games: GameListItem[]
  favorite_count: number
  pending_reviews: number
  pending_issue_counts: PendingIssueCounts | null
}

export interface GameSort {
  field: 'title' | 'created_at' | 'updated_at' | 'release_date' | 'downloads' | 'random' | 'pending_issue_count'
  order: 'asc' | 'desc'
  seed?: number
}

export interface GameSortQuery {
  field?: string
  order?: string
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
  description: string
  releaseDate: string
  developers: string[]
  publishers: string[]
  screenshots: string[]
  coverImage: string
  bannerImage?: string
}

export interface ReviewIssueOverride {
  id: number
  game_id: number
  issue_key: string
  status: string
  reason: string | null
  created_at: string
  updated_at: string
}

export type StartScreenTileSize = 'small' | 'wide' | 'large'

export interface StartScreenTile {
  game_id: number
  public_id: string
  title: string
  cover_image: string | null
  banner_image: string | null
  tile_size: StartScreenTileSize
  image_small_path: string | null
  image_wide_path: string | null
  image_large_path: string | null
  sort_order: number
  column_index: number
  grid_row: number
  grid_col: number
}

export interface StartScreenTileWrite {
  game_id: number
  tile_size: StartScreenTileSize
  image_small_path: string | null
  image_wide_path: string | null
  image_large_path: string | null
  column_index: number
  grid_row: number
  grid_col: number
}

export interface StartScreenColumn {
  id: number
  name: string
  sort_order: number
}

export interface StartScreenLayout {
  columns: StartScreenColumn[]
  tiles: StartScreenTile[]
}

export interface StartScreenLayoutInput {
  columns: Array<{ name: string }>
  tiles: StartScreenTileWrite[]
}
