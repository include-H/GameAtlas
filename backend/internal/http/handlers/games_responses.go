package handlers

import "github.com/hao/game/internal/domain"

// 2026-04-03: response DTOs were split from the old games.go transport file.
// Keep JSON-facing shapes here so handlers and mappers do not need to redefine them.
type gameListItemResponse struct {
	ID                int64                          `json:"id"`
	PublicID          string                         `json:"public_id"`
	Title             string                         `json:"title"`
	TitleAlt          *string                        `json:"title_alt"`
	Visibility        string                         `json:"visibility"`
	Summary           *string                        `json:"summary"`
	ReleaseDate       *string                        `json:"release_date"`
	Engine            *string                        `json:"engine"`
	CoverImage        *string                        `json:"cover_image"`
	BannerImage       *string                        `json:"banner_image"`
	WikiContent       *string                        `json:"wiki_content"`
	Downloads         int64                          `json:"downloads"`
	PrimaryScreenshot *string                        `json:"primary_screenshot"`
	ScreenshotCount   int64                          `json:"screenshot_count"`
	FileCount         int64                          `json:"file_count"`
	DeveloperCount    int64                          `json:"developer_count"`
	PublisherCount    int64                          `json:"publisher_count"`
	PlatformCount     int64                          `json:"platform_count"`
	PendingIssues     *domain.PendingIssueEvaluation `json:"pending_issues,omitempty"`
	CreatedAt         string                         `json:"created_at"`
	UpdatedAt         string                         `json:"updated_at"`
}

type timelineGameItemResponse struct {
	ID          int64   `json:"id"`
	PublicID    string  `json:"public_id"`
	Title       string  `json:"title"`
	ReleaseDate *string `json:"release_date"`
	CoverImage  *string `json:"cover_image"`
	BannerImage *string `json:"banner_image"`
}

type gameAssetResponse struct {
	ID        int64  `json:"id"`
	AssetUID  string `json:"asset_uid"`
	Path      string `json:"path"`
	SortOrder int    `json:"sort_order"`
}

type metadataItemResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
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
	ID            int64                          `json:"id"`
	PublicID      string                         `json:"public_id"`
	Title         string                         `json:"title"`
	TitleAlt      *string                        `json:"title_alt"`
	Visibility    string                         `json:"visibility"`
	Summary       *string                        `json:"summary"`
	ReleaseDate   *string                        `json:"release_date"`
	Engine        *string                        `json:"engine"`
	CoverImage    *string                        `json:"cover_image"`
	BannerImage   *string                        `json:"banner_image"`
	WikiContent   *string                        `json:"wiki_content"`
	Downloads     int64                          `json:"downloads"`
	PreviewVideo  *gameAssetResponse             `json:"preview_video"`
	PreviewVideos []gameAssetResponse            `json:"preview_videos"`
	Screenshots   []gameAssetResponse            `json:"screenshots"`
	Series        *metadataItemResponse          `json:"series"`
	Platforms     []metadataItemResponse         `json:"platforms"`
	Developers    []metadataItemResponse         `json:"developers"`
	Publishers    []metadataItemResponse         `json:"publishers"`
	Tags          []tagResponse                  `json:"tags"`
	TagGroups     []gameTagGroupResponse         `json:"tag_groups"`
	Files         []gameFileResponse             `json:"files"`
	PendingIssues *domain.PendingIssueEvaluation `json:"pending_issues,omitempty"`
	CreatedAt     string                         `json:"created_at"`
	UpdatedAt     string                         `json:"updated_at"`
}

type tagResponse struct {
	ID        int64  `json:"id"`
	GroupID   int64  `json:"group_id"`
	GroupKey  string `json:"group_key"`
	GroupName string `json:"group_name"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	ParentID  *int64 `json:"parent_id"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type gameTagGroupResponse struct {
	ID            int64         `json:"id"`
	Key           string        `json:"key"`
	Name          string        `json:"name"`
	AllowMultiple bool          `json:"allow_multiple"`
	IsFilterable  bool          `json:"is_filterable"`
	Tags          []tagResponse `json:"tags"`
}
