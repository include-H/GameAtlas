package handlers

// 2026-04-03: response DTOs were split from the old games.go transport file.
// Keep JSON-facing shapes here so handlers and mappers do not need to redefine them.
type successEnvelope[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

type errorEnvelope struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type errorEnvelopeWithData[T any] struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Data    T      `json:"data"`
}

type authSessionResponse struct {
	IsAdmin bool `json:"is_admin"`
}

type authStateResponse struct {
	IsAdmin          bool   `json:"is_admin"`
	Role             string `json:"role"`
	AdminDisplayName string `json:"admin_display_name"`
}

type authLogoutResponse struct {
	LoggedOut bool `json:"logged_out"`
}

type authLockedResponse struct {
	RetryAfterSeconds int64 `json:"retry_after_seconds"`
	LockedUntilUnix   int64 `json:"locked_until_unix"`
}

type authDeniedResponse struct {
	RemainingAttempts int `json:"remaining_attempts"`
}

type assetUploadResponse struct {
	Path     string `json:"path"`
	AssetUID string `json:"asset_uid,omitempty"`
}

type operationStatusResponse struct {
	Deleted  bool `json:"deleted"`
	Updated  bool `json:"updated"`
	Recorded bool `json:"recorded"`
}

type wikiDocumentResponse struct {
	GameID       int64   `json:"game_id"`
	Title        string  `json:"title"`
	Content      *string `json:"content"`
	UpdatedAt    string  `json:"updated_at"`
	HistoryCount int     `json:"history_count,omitempty"`
}

type wikiHistoryItemResponse struct {
	ID            int64   `json:"id"`
	GameID        int64   `json:"game_id"`
	Content       string  `json:"content"`
	ChangeSummary *string `json:"change_summary"`
	CreatedAt     string  `json:"created_at"`
}

type gameListItemResponse struct {
	ID                int64                           `json:"id"`
	PublicID          string                          `json:"public_id"`
	Title             string                          `json:"title"`
	TitleAlt          *string                         `json:"title_alt"`
	Visibility        string                          `json:"visibility"`
	Summary           *string                         `json:"summary"`
	ReleaseDate       *string                         `json:"release_date"`
	CoverImage        *string                         `json:"cover_image"`
	BannerImage       *string                         `json:"banner_image"`
	WikiContent       *string                         `json:"wiki_content"`
	Downloads         int64                           `json:"downloads"`
	PrimaryScreenshot *string                         `json:"primary_screenshot"`
	ScreenshotCount   int64                           `json:"screenshot_count,omitempty"`
	LogoVisible       bool                            `json:"logo_visible"`
	FileCount         int64                           `json:"file_count,omitempty"`
	DeveloperCount    int64                           `json:"developer_count,omitempty"`
	PublisherCount    int64                           `json:"publisher_count,omitempty"`
	IsFavorite        bool                            `json:"is_favorite"`
	Series            *gameSeriesResponse             `json:"series"`
	PendingIssues     *pendingIssueEvaluationResponse `json:"pending_issues,omitempty"`
	CreatedAt         string                          `json:"created_at"`
	UpdatedAt         string                          `json:"updated_at"`
}

type gameSeriesResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type timelineGameItemResponse struct {
	ID          int64   `json:"id"`
	PublicID    string  `json:"public_id"`
	Title       string  `json:"title"`
	ReleaseDate *string `json:"release_date"`
	CoverImage  *string `json:"cover_image"`
	BannerImage *string `json:"banner_image"`
}

type gamePreviewVideosResponse struct {
	PublicID      string              `json:"public_id"`
	PreviewVideos []gameAssetResponse `json:"preview_videos"`
}

type gameAssetResponse struct {
	ID        int64    `json:"id"`
	AssetUID  string   `json:"asset_uid"`
	Path      string   `json:"path"`
	SortOrder int      `json:"sort_order"`
	PositionX *float64 `json:"position_x"`
	PositionY *float64 `json:"position_y"`
	WidthPct  *float64 `json:"width_pct"`
}

type metadataItemResponse struct {
	ID                   int64    `json:"id"`
	Name                 string   `json:"name"`
	Slug                 string   `json:"slug"`
	SortOrder            int      `json:"sort_order"`
	CreatedAt            string   `json:"created_at"`
	GameCount            int      `json:"game_count,omitempty"`
	CoverImage           *string  `json:"cover_image,omitempty"`
	CoverCandidates      []string `json:"cover_candidates,omitempty"`
	BackgroundCandidates []string `json:"background_candidates,omitempty"`
	LatestUpdatedAt      *string  `json:"latest_updated_at,omitempty"`
}

type gameFileResponse struct {
	ID              int64   `json:"id"`
	GameID          int64   `json:"game_id"`
	FileName        string  `json:"file_name"`
	FilePath        string  `json:"file_path,omitempty"`
	Label           *string `json:"label"`
	Notes           *string `json:"notes"`
	SizeBytes       *int64  `json:"size_bytes"`
	SortOrder       int     `json:"sort_order"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	SourceCreatedAt *string `json:"source_created_at"`
}

type gameDetailResponse struct {
	ID            int64                           `json:"id"`
	PublicID      string                          `json:"public_id"`
	Title         string                          `json:"title"`
	TitleAlt      *string                         `json:"title_alt"`
	Visibility    string                          `json:"visibility"`
	Summary       *string                         `json:"summary"`
	ReleaseDate   *string                         `json:"release_date"`
	CoverImage    *string                         `json:"cover_image"`
	BannerImage   *string                         `json:"banner_image"`
	WikiContent   *string                         `json:"wiki_content"`
	Downloads     int64                           `json:"downloads"`
	PreviewVideos []gameAssetResponse             `json:"preview_videos"`
	Screenshots   []gameAssetResponse             `json:"screenshots"`
	Covers        []gameAssetResponse             `json:"covers"`
	Banners       []gameAssetResponse             `json:"banners"`
	Logos         []gameAssetResponse             `json:"logos"`
	Series        *metadataItemResponse           `json:"series"`
	Developers    []metadataItemResponse          `json:"developers"`
	Publishers    []metadataItemResponse          `json:"publishers"`
	Files         []gameFileResponse              `json:"files"`
	LogoVisible   bool                            `json:"logo_visible"`
	IsFavorite    bool                            `json:"is_favorite"`
	PendingIssues *pendingIssueEvaluationResponse `json:"pending_issues,omitempty"`
	CreatedAt     string                          `json:"created_at"`
	UpdatedAt     string                          `json:"updated_at"`
}

type reviewIssueOverrideResponse struct {
	ID        int64   `json:"id"`
	GameID    int64   `json:"game_id"`
	IssueKey  string  `json:"issue_key"`
	Status    string  `json:"status"`
	Reason    *string `json:"reason"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type directoryItemResponse struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDirectory bool   `json:"is_directory"`
	SizeBytes   *int64 `json:"size_bytes"`
}

type directoryListResponse struct {
	CurrentPath  string                  `json:"current_path"`
	ParentPath   *string                 `json:"parent_path"`
	Items        []directoryItemResponse `json:"items"`
	Incomplete   bool                    `json:"incomplete"`
	SkippedCount int                     `json:"skipped_count"`
}

type steamSearchResultResponse struct {
	AppID       int64   `json:"app_id"`
	Name        string  `json:"name"`
	ReleaseDate *string `json:"release_date"`
	TinyImage   *string `json:"tiny_image"`
}

type steamAssetsPreviewResponse struct {
	AppID          int64    `json:"app_id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	ReleaseDate    string   `json:"release_date"`
	Developers     []string `json:"developers"`
	Publishers     []string `json:"publishers"`
	CoverURL       *string  `json:"cover_url"`
	BannerURL      *string  `json:"banner_url"`
	ScreenshotURLs []string `json:"screenshot_urls"`
}

type pendingIssueDefinitionResponse struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type pendingIssueDetailDefinitionResponse struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Group string `json:"group"`
}

type pendingIssueDetailStateResponse struct {
	Key     string  `json:"key"`
	Group   string  `json:"group"`
	Ignored bool    `json:"ignored"`
	Reason  *string `json:"reason,omitempty"`
}

type pendingIssueEvaluationResponse struct {
	Groups  []string                          `json:"groups"`
	Details []pendingIssueDetailStateResponse `json:"details"`
	Severe  bool                              `json:"severe"`
}

type pendingIssueCountSummaryResponse struct {
	Groups       map[string]int `json:"groups"`
	IgnoredTotal int            `json:"ignored_total"`
}

type pendingIssueCatalogResponse struct {
	Groups  []pendingIssueDefinitionResponse       `json:"groups"`
	Details []pendingIssueDetailDefinitionResponse `json:"details"`
}
