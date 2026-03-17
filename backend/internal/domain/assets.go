package domain

type DeleteAssetInput struct {
	GameID    int64  `json:"game_id"`
	AssetType string `json:"asset_type"`
	Path      string `json:"path"`
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
