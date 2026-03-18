package domain

type DeleteAssetInput struct {
	GameID    int64  `json:"game_id"`
	AssetID   *int64 `json:"asset_id"`
	AssetUID  string `json:"asset_uid"`
	AssetType string `json:"asset_type"`
	Path      string `json:"path"`
}

type ScreenshotOrderUpdateInput struct {
	GameID    int64    `json:"game_id"`
	AssetUIDs []string `json:"asset_uids"`
}

type SteamSearchResult struct {
	AppID       int64   `json:"app_id"`
	Name        string  `json:"name"`
	ReleaseDate *string `json:"release_date"`
	TinyImage   *string `json:"tiny_image"`
}

type SteamAssetsPreview struct {
	AppID          int64    `json:"app_id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	CoverURL       *string  `json:"cover_url"`
	BannerURL      *string  `json:"banner_url"`
	ScreenshotURLs []string `json:"screenshot_urls"`
}

type SteamApplyAssetsInput struct {
	GameID         int64    `json:"game_id"`
	CoverURL       *string  `json:"cover_url"`
	BannerURL      *string  `json:"banner_url"`
	ScreenshotURLs []string `json:"screenshot_urls"`
}
