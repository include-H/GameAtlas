package domain

type SteamSearchResult struct {
	AppID       int64
	Name        string
	ReleaseDate *string
	TinyImage   *string
}

type SteamAssetsPreview struct {
	AppID          int64
	Name           string
	Description    string
	ReleaseDate    string
	Developers     []string
	Publishers     []string
	CoverURL       *string
	BannerURL      *string
	ScreenshotURLs []string
}

type SteamApplyAssetsInput struct {
	GameID         int64
	CoverURL       *string
	BannerURL      *string
	ScreenshotURLs []string
}
