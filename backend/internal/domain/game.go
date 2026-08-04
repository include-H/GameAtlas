package domain

import "strings"

// 2026-05-09: 架构说明 — domain 层结构体带 db:"..." 标签，是项目初始化时的有意设计。
// SQLX 直接扫描到 domain 类型，省去 repository 层的私有 scan struct + mapper。
// 严格来说 domain 不应感知持久化细节，但当前写法在 Go + SQLX 项目中属常见简化，
// 且项目已进入稳定期，大规模重构收益不高。如需改为严格分层，需为每个查询创建
// repository 私有 scan struct 再 mapper 到 domain 类型。

// 2026-05-09: A-5 审查结论 — Game 与 GameListItem 字段重复但不合并。
// 两者各有清晰边界：Game 用于单条读写路径（detail/create/update），GameListItem
// 用于列表聚合查询（含 screenshot_stats/file_stats 等 CTE）。合并会模糊 SQL
// 语义差异，收益低。Game.PendingIssues 已删除（详情路径从未写入，结果在
// GameDetail.PendingIssues 上）。
type Game struct {
	ID                int64   `db:"id"`
	PublicID          string  `db:"public_id"`
	Title             string  `db:"title"`
	TitleAlt          *string `db:"title_alt"`
	Visibility        string  `db:"visibility"`
	Summary           *string `db:"summary"`
	ReleaseDate       *string `db:"release_date"`
	CoverImage        *string `db:"cover_image"`
	BannerImage       *string `db:"banner_image"`
	WikiContent       *string `db:"wiki_content"`
	Downloads         int64   `db:"downloads"`
	PrimaryScreenshot *string `db:"primary_screenshot"`
	LogoVisible       bool    `db:"logo_visible"`
	IsFavorite        bool    `db:"is_favorite"`
	CreatedAt         string  `db:"created_at"`
	UpdatedAt         string  `db:"updated_at"`
}

type GameListItem struct {
	ID                int64   `db:"id"`
	PublicID          string  `db:"public_id"`
	Title             string  `db:"title"`
	TitleAlt          *string `db:"title_alt"`
	Visibility        string  `db:"visibility"`
	Summary           *string `db:"summary"`
	ReleaseDate       *string `db:"release_date"`
	CoverImage        *string `db:"cover_image"`
	BannerImage       *string `db:"banner_image"`
	WikiContent       *string `db:"wiki_content"`
	Downloads         int64   `db:"downloads"`
	PrimaryScreenshot *string `db:"primary_screenshot"`
	ScreenshotCount   int64   `db:"screenshot_count"`
	LogoCount         int64   `db:"logo_count"`
	LogoVisible       bool    `db:"logo_visible"`
	FileCount         int64   `db:"file_count"`
	DeveloperCount    int64   `db:"developer_count"`
	PublisherCount    int64   `db:"publisher_count"`
	IsFavorite        bool    `db:"is_favorite"`
	SeriesID          *int64  `db:"series_id"`
	SeriesName        *string `db:"series_name"`
	PendingIssues     *PendingIssueEvaluation
	CreatedAt         string `db:"created_at"`
	UpdatedAt         string `db:"updated_at"`
}

// MetadataGameSummary 是系列、发行商等分组详情视图，保留 WikiContent 用于摘要展示，省略聚合计数字段（ScreenshotCount 等）因为分组视图不需要这些数据。
type MetadataGameSummary struct {
	ID                int64   `db:"id"`
	PublicID          string  `db:"public_id"`
	Title             string  `db:"title"`
	TitleAlt          *string `db:"title_alt"`
	Visibility        string  `db:"visibility"`
	Summary           *string `db:"summary"`
	ReleaseDate       *string `db:"release_date"`
	CoverImage        *string `db:"cover_image"`
	BannerImage       *string `db:"banner_image"`
	WikiContent       *string `db:"wiki_content"`
	Downloads         int64   `db:"downloads"`
	PrimaryScreenshot *string `db:"primary_screenshot"`
	IsFavorite        bool    `db:"is_favorite"`
	SeriesID          *int64  `db:"series_id"`
	SeriesName        *string `db:"series_name"`
	CreatedAt         string  `db:"created_at"`
	UpdatedAt         string  `db:"updated_at"`
}

type GameAsset struct {
	ID        int64    `db:"id"`
	GameID    int64    `db:"game_id"`
	AssetUID  string   `db:"asset_uid"`
	AssetType string   `db:"asset_type"`
	Path      string   `db:"path"`
	SortOrder int      `db:"sort_order"`
	PositionX *float64 `db:"position_x"`
	PositionY *float64 `db:"position_y"`
	WidthPct  *float64 `db:"width_pct"`
	CreatedAt string   `db:"created_at"`
}

type GameFile struct {
	ID              int64   `db:"id"`
	GameID          int64   `db:"game_id"`
	FilePath        string  `db:"file_path"`
	Label           *string `db:"label"`
	Notes           *string `db:"notes"`
	SizeBytes       *int64  `db:"size_bytes"`
	SortOrder       int     `db:"sort_order"`
	CreatedAt       string  `db:"created_at"`
	UpdatedAt       string  `db:"updated_at"`
	SourceCreatedAt *string `db:"source_created_at"`
}

type MetadataItem struct {
	ID                   int64  `db:"id"`
	Name                 string `db:"name"`
	Slug                 string `db:"slug"`
	SortOrder            int    `db:"sort_order"`
	CreatedAt            string `db:"created_at"`
	GameCount            int
	CoverImage           *string
	CoverCandidates      []string
	BackgroundCandidates []string
	LatestUpdatedAt      *string
}

type MetadataWriteInput struct {
	Name      string
	Slug      *string
	SortOrder *int
}

type GamesListParams struct {
	Page                  int
	Limit                 int
	Search                string
	SeriesID              int64
	PendingOnly           bool
	PendingIncludeIgnored bool
	PendingIssue          string
	PendingSevereOnly     bool
	PendingRecentDays     int
	FavoriteOnly          bool
	Visibility            string
	IncludeAll            bool
	Sort                  string
	Order                 string
	SortSeed              int64
}

// 2026-05-09: allowedGamesListSorts 定义允许的排序 key 白名单，负责校验合法性。SQL 列映射在 repository 层的 allowedGameSortFields 中，两层职责有意分离。
var allowedGamesListSorts = map[string]struct{}{
	"title":               {},
	"created_at":          {},
	"updated_at":          {},
	"release_date":        {},
	"downloads":           {},
	"random":              {},
	"pending_issue_count": {},
}

func IsAllowedGamesListSort(value string) bool {
	_, ok := allowedGamesListSorts[value]
	return ok
}

func IsAllowedGamesListOrder(value string) bool {
	return value == "asc" || value == "desc"
}

type GamesTimelineParams struct {
	Limit      int
	AfterDate  string // cursor: release_date of last item from previous page
	AfterID    int64  // cursor: id of last item from previous page
	Visibility string
	IncludeAll bool
}

type PendingGroupCounts struct {
	MissingAssets   int `db:"missing_assets"`
	MissingWiki     int `db:"missing_wiki"`
	MissingFiles    int `db:"missing_files"`
	MissingMetadata int `db:"missing_metadata"`
	IgnoredTotal    int `db:"ignored_total"`
}

type TimelineGame struct {
	ID          int64   `db:"id"`
	PublicID    string  `db:"public_id"`
	Title       string  `db:"title"`
	ReleaseDate *string `db:"release_date"`
	CoverImage  *string `db:"cover_image"`
	BannerImage *string `db:"banner_image"`
}

type GameCoreInput struct {
	Title       string
	TitleAlt    *string
	Visibility  string
	Summary     *string
	ReleaseDate *string
	LogoVisible *bool
}

// Create keeps the add-game flow intentionally narrow: it only creates the base game row.
// Full aggregate edits happen through GameAggregateUpdateInput.
type GameCreateInput struct {
	Title      string
	Visibility string
}

// 2026-05-01: aggregate update is a full-replacement write for the editable game aggregate.
// Impact: omitted relation/order collections are interpreted as empty at the write boundary;
// this endpoint does not support sparse patch semantics.
type GameAggregateCoreUpdateInput struct {
	GameCoreInput
	SeriesID     *int64
	DeveloperIDs []int64
	PublisherIDs []int64
}

type GameFileWriteInput struct {
	FilePath  string
	Label     *string
	Notes     *string
	SortOrder int
}

type GameFileUpsertInput struct {
	ID              *int64
	FilePath        string
	Label           *string
	Notes           *string
	SortOrder       int
	SizeBytes       *int64
	SourceCreatedAt *string
}

type GameAggregateUpdateInput struct {
	Game   GameAggregateCoreUpdateInput
	Assets GameAggregateAssetsInput
}

type LogoPositionInput struct {
	AssetUID  string
	PositionX *float64
	PositionY *float64
	WidthPct  *float64
}

type GameAggregateAssetsInput struct {
	Files                    []GameFileUpsertInput
	ScreenshotOrderAssetUIDs []string
	VideoOrderAssetUIDs      []string
	CoverOrderAssetUIDs      []string
	LogoOrderAssetUIDs       []string
	BannerOrderAssetUIDs     []string
	LogoPositions            []LogoPositionInput
	NewAssets                []NewAssetEntry
}

type NewAssetEntry struct {
	AssetUID  string
	AssetType string
	Path      string
}

type WikiWriteInput struct {
	Content       string
	ChangeSummary *string
}

type WikiHistoryEntry struct {
	ID            int64   `db:"id"`
	GameID        int64   `db:"game_id"`
	Content       string  `db:"content"`
	ChangeSummary *string `db:"change_summary"`
	CreatedAt     string  `db:"created_at"`
}

type ReviewIssueOverride struct {
	ID        int64   `db:"id"`
	GameID    int64   `db:"game_id"`
	IssueKey  string  `db:"issue_key"`
	Status    string  `db:"status"`
	Reason    *string `db:"reason"`
	CreatedAt string  `db:"created_at"`
	UpdatedAt string  `db:"updated_at"`
}

type GameStats struct {
	TotalGames     int
	TotalDownloads int64
	RecentGames    []GameListItem
	PopularGames   []GameListItem
	FavoriteCount  int
	PendingReviews int
}

const (
	GameVisibilityPublic  = "public"
	GameVisibilityPrivate = "private"
)

// 2026-05-09: DefaultVisibility returns the effective visibility for list queries.
// When the caller does not specify a value and is not requesting all games,
// public visibility is the default.
func DefaultVisibility(explicit string, includeAll bool) string {
	if includeAll {
		return ""
	}
	trimmed := strings.TrimSpace(explicit)
	if trimmed == "" {
		return GameVisibilityPublic
	}
	return trimmed
}
