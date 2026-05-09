package domain

import "strings"

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
	ScreenshotCount   int64   `db:"screenshot_count"`
	FileCount         int64   `db:"file_count"`
	DeveloperCount    int64   `db:"developer_count"`
	PublisherCount    int64   `db:"publisher_count"`
	IsFavorite        bool    `db:"is_favorite"`
	CreatedAt         string `db:"created_at"`
	UpdatedAt         string `db:"updated_at"`
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
	FileCount         int64   `db:"file_count"`
	DeveloperCount    int64   `db:"developer_count"`
	PublisherCount    int64   `db:"publisher_count"`
	IsFavorite        bool    `db:"is_favorite"`
	PendingIssues     *PendingIssueEvaluation
	CreatedAt         string `db:"created_at"`
	UpdatedAt         string `db:"updated_at"`
}

// 2026-05-09: SeriesGameSummary 是系列摘要视图，保留 WikiContent 用于摘要展示，省略聚合计数字段（ScreenshotCount 等）因为系列视图不需要这些数据。
type SeriesGameSummary struct {
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
	CreatedAt         string  `db:"created_at"`
	UpdatedAt         string  `db:"updated_at"`
}

type GameAsset struct {
	ID        int64  `db:"id"`
	GameID    int64  `db:"game_id"`
	AssetUID  string `db:"asset_uid"`
	AssetType string `db:"asset_type"`
	Path      string `db:"path"`
	SortOrder int    `db:"sort_order"`
	CreatedAt string `db:"created_at"`
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
	ID              int64    `db:"id"`
	Name            string   `db:"name"`
	Slug            string   `db:"slug"`
	SortOrder       int      `db:"sort_order"`
	CreatedAt       string   `db:"created_at"`
	GameCount       int
	CoverImage      *string
	CoverCandidates []string
	LatestUpdatedAt *string
}

type MetadataWriteInput struct {
	Name      string
	Slug      *string
	SortOrder *int
}

type TagGroup struct {
	ID            int64   `db:"id"`
	Key           string  `db:"key"`
	Name          string  `db:"name"`
	Description   *string `db:"description"`
	SortOrder     int     `db:"sort_order"`
	AllowMultiple bool    `db:"allow_multiple"`
	IsFilterable  bool    `db:"is_filterable"`
	CreatedAt     string  `db:"created_at"`
	UpdatedAt     string  `db:"updated_at"`
}

type Tag struct {
	ID                 int64  `db:"id"`
	GroupID            int64  `db:"group_id"`
	GroupKey           string `db:"group_key"`
	GroupName          string `db:"group_name"`
	GroupAllowMultiple bool   `db:"group_allow_multiple"`
	GroupIsFilterable  bool   `db:"group_is_filterable"`
	Name               string `db:"name"`
	Slug               string `db:"slug"`
	ParentID           *int64 `db:"parent_id"`
	SortOrder          int    `db:"sort_order"`
	IsActive           bool   `db:"is_active"`
	CreatedAt          string `db:"created_at"`
	UpdatedAt          string `db:"updated_at"`
}

type GameTagGroup struct {
	ID            int64
	Key           string
	Name          string
	AllowMultiple bool
	IsFilterable  bool
	Tags          []Tag
}

type TagGroupWriteInput struct {
	Key           string
	Name          string
	Description   *string
	SortOrder     *int
	AllowMultiple *bool
	IsFilterable  *bool
}

type TagWriteInput struct {
	GroupID   int64
	Name      string
	Slug      *string
	ParentID  *int64
	SortOrder *int
	IsActive  *bool
}

type TagsListParams struct {
	GroupID  int64
	GroupKey string
	Active   *bool
}

type GamesListParams struct {
	Page                  int
	Limit                 int
	Search                string
	SeriesID              int64
	TagIDs                []int64
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
	Limit             int
	FromDate          string
	ToDate            string
	CursorReleaseDate string
	CursorID          int64
	Visibility        string
	IncludeAll        bool
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
	CoverImage  *string
	BannerImage *string
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
	TagIDs       []int64
}

type GameFileWriteInput struct {
	FilePath  string
	Label     *string
	Notes     *string
	SortOrder int
}

type GameFileUpsertInput struct {
	ID        *int64
	FilePath  string
	Label     *string
	Notes     *string
	SortOrder int
}

type GameAssetDeleteInput struct {
	AssetType string
	Path      string
	AssetID   *int64
	AssetUID  string
}

type GameAggregateUpdateInput struct {
	Game   GameAggregateCoreUpdateInput
	Assets GameAggregateAssetsInput
}

type GameAggregateAssetsInput struct {
	Files                    []GameFileUpsertInput
	DeleteAssets             []GameAssetDeleteInput
	ScreenshotOrderAssetUIDs []string
	VideoOrderAssetUIDs      []string
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
