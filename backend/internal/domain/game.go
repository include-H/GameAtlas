package domain

type Game struct {
	ID                int64   `db:"id"`
	PublicID          string  `db:"public_id"`
	Title             string  `db:"title"`
	TitleAlt          *string `db:"title_alt"`
	Visibility        string  `db:"visibility"`
	Summary           *string `db:"summary"`
	ReleaseDate       *string `db:"release_date"`
	Engine            *string `db:"engine"`
	CoverImage        *string `db:"cover_image"`
	BannerImage       *string `db:"banner_image"`
	WikiContent       *string `db:"wiki_content"`
	NeedsReview       bool    `db:"needs_review"`
	Downloads         int64   `db:"downloads"`
	PrimaryScreenshot *string `db:"primary_screenshot"`
	ScreenshotCount   int64   `db:"screenshot_count"`
	FileCount         int64   `db:"file_count"`
	DeveloperCount    int64   `db:"developer_count"`
	PublisherCount    int64   `db:"publisher_count"`
	PlatformCount     int64   `db:"platform_count"`
	PendingIssues     *PendingIssueEvaluation
	CreatedAt         string `db:"created_at"`
	UpdatedAt         string `db:"updated_at"`
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
	SourceCreatedAt *string `db:"source_created_at" json:"source_created_at"`
}

type MetadataItem struct {
	ID              int64    `db:"id" json:"id"`
	Name            string   `db:"name" json:"name"`
	Slug            string   `db:"slug" json:"slug"`
	SortOrder       int      `db:"sort_order" json:"sort_order"`
	CreatedAt       string   `db:"created_at" json:"created_at"`
	GameCount       int      `json:"game_count,omitempty"`
	CoverImage      *string  `json:"cover_image,omitempty"`
	CoverCandidates []string `json:"cover_candidates,omitempty"`
	LatestUpdatedAt *string  `json:"latest_updated_at,omitempty"`
}

type MetadataWriteInput struct {
	Name      string  `json:"name"`
	Slug      *string `json:"slug"`
	SortOrder *int    `json:"sort_order"`
}

type TagGroup struct {
	ID            int64   `db:"id" json:"id"`
	Key           string  `db:"key" json:"key"`
	Name          string  `db:"name" json:"name"`
	Description   *string `db:"description" json:"description"`
	SortOrder     int     `db:"sort_order" json:"sort_order"`
	AllowMultiple bool    `db:"allow_multiple" json:"allow_multiple"`
	IsFilterable  bool    `db:"is_filterable" json:"is_filterable"`
	CreatedAt     string  `db:"created_at" json:"created_at"`
	UpdatedAt     string  `db:"updated_at" json:"updated_at"`
}

type Tag struct {
	ID                 int64  `db:"id" json:"id"`
	GroupID            int64  `db:"group_id" json:"group_id"`
	GroupKey           string `db:"group_key" json:"group_key"`
	GroupName          string `db:"group_name" json:"group_name"`
	GroupAllowMultiple bool   `db:"group_allow_multiple" json:"-"`
	GroupIsFilterable  bool   `db:"group_is_filterable" json:"-"`
	Name               string `db:"name" json:"name"`
	Slug               string `db:"slug" json:"slug"`
	ParentID           *int64 `db:"parent_id" json:"parent_id"`
	SortOrder          int    `db:"sort_order" json:"sort_order"`
	IsActive           bool   `db:"is_active" json:"is_active"`
	CreatedAt          string `db:"created_at" json:"created_at"`
	UpdatedAt          string `db:"updated_at" json:"updated_at"`
}

type GameTagGroup struct {
	ID            int64  `json:"id"`
	Key           string `json:"key"`
	Name          string `json:"name"`
	AllowMultiple bool   `json:"allow_multiple"`
	IsFilterable  bool   `json:"is_filterable"`
	Tags          []Tag  `json:"tags"`
}

type TagGroupWriteInput struct {
	Key           string  `json:"key"`
	Name          string  `json:"name"`
	Description   *string `json:"description"`
	SortOrder     *int    `json:"sort_order"`
	AllowMultiple *bool   `json:"allow_multiple"`
	IsFilterable  *bool   `json:"is_filterable"`
}

type TagWriteInput struct {
	GroupID   int64   `json:"group_id"`
	Name      string  `json:"name"`
	Slug      *string `json:"slug"`
	ParentID  *int64  `json:"parent_id"`
	SortOrder *int    `json:"sort_order"`
	IsActive  *bool   `json:"is_active"`
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
	PlatformID            int64
	TagIDs                []int64
	NeedsReview           *bool
	PendingOnly           bool
	PendingIncludeIgnored bool
	PendingIssue          string
	PendingSevereOnly     bool
	PendingRecentDays     int
	Visibility            string
	IncludeAll            bool
	Sort                  string
	Order                 string
	SortSeed              int64
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
	MissingAssets   int `db:"missing_assets" json:"missing_assets"`
	MissingWiki     int `db:"missing_wiki" json:"missing_wiki"`
	MissingFiles    int `db:"missing_files" json:"missing_files"`
	MissingMetadata int `db:"missing_metadata" json:"missing_metadata"`
	IgnoredTotal    int `db:"ignored_total" json:"ignored_total"`
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
	Title       string  `json:"title"`
	TitleAlt    *string `json:"title_alt"`
	Visibility  string  `json:"visibility"`
	Summary     *string `json:"summary"`
	ReleaseDate *string `json:"release_date"`
	Engine      *string `json:"engine"`
	CoverImage  *string `json:"cover_image"`
	BannerImage *string `json:"banner_image"`
	NeedsReview bool    `json:"needs_review"`
}

type GameWriteInput struct {
	GameCoreInput
	SeriesID     *int64  `json:"series_id"`
	PlatformIDs  []int64 `json:"platform_ids"`
	DeveloperIDs []int64 `json:"developer_ids"`
	PublisherIDs []int64 `json:"publisher_ids"`
	TagIDs       []int64 `json:"tag_ids"`
}

type OptionalInt64Patch struct {
	Present bool
	Value   *int64
}

type Int64SlicePatch struct {
	Present bool
	Values  []int64
}

type GameAggregatePatchInput struct {
	GameCoreInput
	SeriesID     OptionalInt64Patch
	PlatformIDs  Int64SlicePatch
	DeveloperIDs Int64SlicePatch
	PublisherIDs Int64SlicePatch
	TagIDs       Int64SlicePatch
}

type GameFileWriteInput struct {
	FilePath  string  `json:"file_path"`
	Label     *string `json:"label"`
	Notes     *string `json:"notes"`
	SortOrder int     `json:"sort_order"`
}

type GameFileUpsertInput struct {
	ID        *int64  `json:"id"`
	FilePath  string  `json:"file_path"`
	Label     *string `json:"label"`
	Notes     *string `json:"notes"`
	SortOrder int     `json:"sort_order"`
}

type GameAssetDeleteInput struct {
	AssetType string `json:"asset_type"`
	Path      string `json:"path"`
	AssetID   *int64 `json:"asset_id"`
	AssetUID  string `json:"asset_uid"`
}

type GameAggregateUpdateInput struct {
	Game   GameAggregatePatchInput  `json:"game"`
	Assets GameAggregateAssetsInput `json:"assets"`
}

type GameAggregateAssetsInput struct {
	Files                    []GameFileUpsertInput  `json:"files"`
	DeleteAssets             []GameAssetDeleteInput `json:"delete_assets"`
	ScreenshotOrderAssetUIDs []string               `json:"screenshot_order_asset_uids"`
	VideoOrderAssetUIDs      []string               `json:"video_order_asset_uids"`
}

type WikiWriteInput struct {
	Content       string  `json:"content"`
	ChangeSummary *string `json:"change_summary"`
}

type WikiHistoryEntry struct {
	ID            int64   `db:"id"`
	GameID        int64   `db:"game_id"`
	Content       string  `db:"content"`
	ChangeSummary *string `db:"change_summary"`
	CreatedAt     string  `db:"created_at"`
}

type ReviewIssueOverride struct {
	ID        int64   `db:"id" json:"id"`
	GameID    int64   `db:"game_id" json:"game_id"`
	IssueKey  string  `db:"issue_key" json:"issue_key"`
	Status    string  `db:"status" json:"status"`
	Reason    *string `db:"reason" json:"reason"`
	CreatedAt string  `db:"created_at" json:"created_at"`
	UpdatedAt string  `db:"updated_at" json:"updated_at"`
}

type GameStats struct {
	TotalGames     int
	TotalDownloads int64
	RecentGames    []Game
	PopularGames   []Game
	PendingReviews int
}

const (
	GameVisibilityPublic  = "public"
	GameVisibilityPrivate = "private"
)
