package services

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hao/game/internal/domain"
)

func TestNormalizeSteamReleaseDateSupportsCommonFormats(t *testing.T) {
	tests := []struct {
		name string
		date *struct {
			ComingSoon bool   `json:"coming_soon"`
			Date       string `json:"date"`
		}
		want string
	}{
		{
			name: "chinese date",
			date: &struct {
				ComingSoon bool   `json:"coming_soon"`
				Date       string `json:"date"`
			}{Date: "2024 年 2 月 3 日"},
			want: "2024-02-03",
		},
		{
			name: "english date",
			date: &struct {
				ComingSoon bool   `json:"coming_soon"`
				Date       string `json:"date"`
			}{Date: "Feb 4, 2024"},
			want: "2024-02-04",
		},
		{
			name: "coming soon",
			date: &struct {
				ComingSoon bool   `json:"coming_soon"`
				Date       string `json:"date"`
			}{ComingSoon: true, Date: "Coming soon"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeSteamReleaseDate(tt.date); got != tt.want {
				t.Fatalf("normalizeSteamReleaseDate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSteamServiceSearchMergesLocalesAndDedupesResults(t *testing.T) {
	service := &SteamService{
		client: &http.Client{Transport: steamRoundTripper(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case "https://store.steampowered.com/api/storesearch/?term=portal&l=schinese&cc=CN":
				return steamJSONResponse(`{"items":[{"id":10,"name":"传送门","tiny_image":"https://cdn/10.png","release_date":{"date":"2024-02-03"}},{"id":11,"name":"半条命"}]}`), nil
			case "https://store.steampowered.com/api/storesearch/?term=portal&l=english&cc=US":
				return steamJSONResponse(`{"items":[{"id":10,"name":"Portal","tiny_image":"https://cdn/10-en.png"},{"id":12,"name":"Portal 2","release_date":{"date":"2024-03-04"}}]}`), nil
			default:
				return steamTextResponse(http.StatusNotFound, ""), nil
			}
		})},
	}

	results, err := service.Search("portal", "")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	if results[0].AppID != 10 || results[0].Name != "传送门" {
		t.Fatalf("results[0] = %+v, want first CN item", results[0])
	}
	if results[0].TinyImage == nil || *results[0].TinyImage != "https://cdn/10.png" {
		t.Fatalf("results[0].TinyImage = %v, want CN image", results[0].TinyImage)
	}
	if results[1].AppID != 11 || results[1].ReleaseDate != nil {
		t.Fatalf("results[1] = %+v, want item without release date", results[1])
	}
	if results[2].AppID != 12 || results[2].ReleaseDate == nil || *results[2].ReleaseDate != "2024-03-04" {
		t.Fatalf("results[2] = %+v, want english-only item", results[2])
	}
}

func TestSteamServicePreviewAssetsMergesFallbackDataAndStorePageContent(t *testing.T) {
	service := &SteamService{
		client: &http.Client{Transport: steamRoundTripper(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && req.URL.String() == "https://store.steampowered.com/api/appdetails?appids=123&l=schinese":
				return steamJSONResponse(`{"123":{"success":true,"data":{"name":"中文名","release_date":{"coming_soon":false,"date":"2024 年 2 月 3 日"},"developers":[" 开发组 ","开发组"],"publishers":[],"screenshots":[]}}}`), nil
			case req.Method == http.MethodGet && req.URL.String() == "https://store.steampowered.com/api/appdetails?appids=123&l=english":
				return steamJSONResponse(`{"123":{"success":true,"data":{"name":"English Name","short_description":"English short","developers":["Dev One","Dev Two"],"publishers":["Pub One"],"screenshots":[]}}}`), nil
			case req.Method == http.MethodGet && req.URL.String() == "https://store.steampowered.com/app/123/?l=schinese":
				return steamTextResponse(http.StatusOK, `<div class="game_description_snippet"> 中文 <b>简介</b> </div>`), nil
			case req.Method == http.MethodGet && req.URL.String() == "https://store.steampowered.com/app/123":
				return steamTextResponse(http.StatusOK, `"path_full":"https:\/\/cdn.example.com\/screen-1.jpg""path_full":"https:\/\/cdn.example.com\/screen-2.jpg"`), nil
			case req.Method == http.MethodHead && req.URL.String() == "https://steamcdn-a.akamaihd.net/steam/apps/123/library_600x900_2x.jpg":
				return steamTextResponse(http.StatusNotFound, ""), nil
			case req.Method == http.MethodHead && req.URL.String() == "https://steamcdn-a.akamaihd.net/steam/apps/123/library_600x900.jpg":
				return steamTextResponse(http.StatusOK, ""), nil
			case req.Method == http.MethodHead && req.URL.String() == "https://steamcdn-a.akamaihd.net/steam/apps/123/library_hero_2x.jpg":
				return steamTextResponse(http.StatusOK, ""), nil
			default:
				return steamTextResponse(http.StatusNotFound, ""), nil
			}
		})},
	}

	preview, err := service.PreviewAssets(123, "")
	if err != nil {
		t.Fatalf("PreviewAssets returned error: %v", err)
	}

	if preview.AppID != 123 || preview.Name != "中文名" {
		t.Fatalf("preview = %+v, want app 123 with CN name", preview)
	}
	if preview.Description != "中文 简介" {
		t.Fatalf("Description = %q, want store-page description", preview.Description)
	}
	if preview.ReleaseDate != "2024-02-03" {
		t.Fatalf("ReleaseDate = %q, want 2024-02-03", preview.ReleaseDate)
	}
	if len(preview.Developers) != 1 || preview.Developers[0] != "开发组" {
		t.Fatalf("Developers = %#v, want deduped CN developers", preview.Developers)
	}
	if len(preview.Publishers) != 1 || preview.Publishers[0] != "Pub One" {
		t.Fatalf("Publishers = %#v, want fallback english publisher", preview.Publishers)
	}
	if preview.CoverURL == nil || *preview.CoverURL != "https://steamcdn-a.akamaihd.net/steam/apps/123/library_600x900.jpg" {
		t.Fatalf("CoverURL = %v, want fallback cover asset", preview.CoverURL)
	}
	if preview.BannerURL == nil || *preview.BannerURL != "https://steamcdn-a.akamaihd.net/steam/apps/123/library_hero_2x.jpg" {
		t.Fatalf("BannerURL = %v, want primary banner asset", preview.BannerURL)
	}
	if len(preview.ScreenshotURLs) != 2 || preview.ScreenshotURLs[0] != "https://cdn.example.com/screen-1.jpg" {
		t.Fatalf("ScreenshotURLs = %#v, want store page screenshots", preview.ScreenshotURLs)
	}
}

func TestSteamServicePreviewAssetsUsesEnglishFallbackWhenPrimaryMissing(t *testing.T) {
	service := &SteamService{
		client: &http.Client{Transport: steamRoundTripper(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && req.URL.String() == "https://store.steampowered.com/api/appdetails?appids=555&l=schinese":
				return steamJSONResponse(`{"555":{"success":false}}`), nil
			case req.Method == http.MethodGet && req.URL.String() == "https://store.steampowered.com/api/appdetails?appids=555&l=english":
				return steamJSONResponse(`{"555":{"success":true,"data":{"name":"Fallback Name","short_description":"Fallback Description","release_date":{"coming_soon":false,"date":"Mar 5, 2024"},"developers":[" Dev A "],"publishers":[" Pub A "],"screenshots":[{"path_full":"https://cdn.example.com/fallback-shot.jpg"}]}}}`), nil
			case req.Method == http.MethodGet && strings.HasPrefix(req.URL.String(), "https://store.steampowered.com/app/555"):
				return steamTextResponse(http.StatusNotFound, ""), nil
			case req.Method == http.MethodHead && strings.Contains(req.URL.String(), "/steam/apps/555/"):
				return steamTextResponse(http.StatusNotFound, ""), nil
			default:
				return steamTextResponse(http.StatusNotFound, ""), nil
			}
		})},
	}

	preview, err := service.PreviewAssets(555, "")
	if err != nil {
		t.Fatalf("PreviewAssets returned error: %v", err)
	}
	if preview.Name != "Fallback Name" || preview.Description != "Fallback Description" {
		t.Fatalf("preview = %+v, want english fallback metadata", preview)
	}
	if preview.ReleaseDate != "2024-03-05" {
		t.Fatalf("ReleaseDate = %q, want 2024-03-05", preview.ReleaseDate)
	}
	if len(preview.Developers) != 1 || preview.Developers[0] != "Dev A" {
		t.Fatalf("Developers = %#v, want trimmed english fallback", preview.Developers)
	}
	if len(preview.Publishers) != 1 || preview.Publishers[0] != "Pub A" {
		t.Fatalf("Publishers = %#v, want trimmed english fallback", preview.Publishers)
	}
	if len(preview.ScreenshotURLs) != 1 || preview.ScreenshotURLs[0] != "https://cdn.example.com/fallback-shot.jpg" {
		t.Fatalf("ScreenshotURLs = %#v, want english fallback screenshot", preview.ScreenshotURLs)
	}
}

func TestSteamServiceApplyAssetsRejectsMissingGameID(t *testing.T) {
	service := &SteamService{}

	_, err := service.ApplyAssets(42, domain.SteamApplyAssetsInput{
		GameID: 0,
	})
	if err != ErrValidation {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestSteamServiceSearchReturnsErrorWhenAllLocalesFail(t *testing.T) {
	service := &SteamService{
		client: &http.Client{Transport: steamRoundTripper(func(req *http.Request) (*http.Response, error) {
			return steamTextResponse(http.StatusBadGateway, ""), nil
		})},
	}

	results, err := service.Search("portal", "")
	if err == nil {
		t.Fatalf("Search error = nil, want failure")
	}
	if results != nil {
		t.Fatalf("results = %#v, want nil on total failure", results)
	}
	if err.Error() != "steam search failed" {
		t.Fatalf("error = %q, want steam search failed", err.Error())
	}
}

func TestSteamServiceSearchReturnsEmptySliceForBlankQuery(t *testing.T) {
	service := &SteamService{}

	results, err := service.Search("", "")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("results = %#v, want empty slice", results)
	}
}

func TestSteamServiceProxyAssetReturnsPayloadForPartialContent(t *testing.T) {
	service := &SteamService{
		client: &http.Client{Transport: steamRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Fatalf("request method = %s, want GET", req.Method)
			}
			if req.URL.String() != "https://cdn.example.com/demo.jpg" {
				t.Fatalf("request url = %s, want asset url", req.URL.String())
			}
			resp := steamTextResponse(http.StatusPartialContent, "asset-bytes")
			resp.Header.Set("Content-Type", "image/jpeg")
			return resp, nil
		})},
	}

	contentType, payload, err := service.ProxyAsset("https://cdn.example.com/demo.jpg", "")
	if err != nil {
		t.Fatalf("ProxyAsset returned error: %v", err)
	}
	if contentType != "image/jpeg" {
		t.Fatalf("contentType = %q, want image/jpeg", contentType)
	}
	if string(payload) != "asset-bytes" {
		t.Fatalf("payload = %q, want asset-bytes", string(payload))
	}
}

func TestSteamServiceProxyAssetReturnsErrorForUnexpectedStatus(t *testing.T) {
	service := &SteamService{
		client: &http.Client{Transport: steamRoundTripper(func(req *http.Request) (*http.Response, error) {
			return steamTextResponse(http.StatusBadGateway, ""), nil
		})},
	}

	_, _, err := service.ProxyAsset("https://cdn.example.com/demo.jpg", "")
	if err == nil {
		t.Fatalf("ProxyAsset error = nil, want failure")
	}
	if err.Error() != "steam request failed with status 502" {
		t.Fatalf("error = %q, want status failure", err.Error())
	}
}

func TestSteamServiceProxyAssetRejectsMissingHost(t *testing.T) {
	service := &SteamService{}

	_, _, err := service.ProxyAsset("https:///missing-host.jpg", "")
	if err != ErrValidation {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestSteamServiceFetchDescriptionFallsBackToEnglishStorePage(t *testing.T) {
	service := &SteamService{
		client: &http.Client{Transport: steamRoundTripper(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case "https://store.steampowered.com/app/456/?l=schinese":
				return steamTextResponse(http.StatusOK, `<html><body>no snippet here</body></html>`), nil
			case "https://store.steampowered.com/app/456/?l=english":
				return steamTextResponse(http.StatusOK, `<div class="game_description_snippet"> English&nbsp;<b>summary</b> </div>`), nil
			default:
				return steamTextResponse(http.StatusNotFound, ""), nil
			}
		})},
	}

	description := service.fetchDescriptionFromStorePage(456, "")
	if description != "English summary" {
		t.Fatalf("description = %q, want English summary", description)
	}
}

func TestSteamServiceResolveSteamAssetURLFallsBackToLastCandidate(t *testing.T) {
	service := &SteamService{
		client: &http.Client{Transport: steamRoundTripper(func(req *http.Request) (*http.Response, error) {
			return steamTextResponse(http.StatusNotFound, ""), nil
		})},
	}

	assetURL := service.resolveSteamAssetURL(789, "",
		"https://cdn.example.com/%d/high.jpg",
		"https://cdn.example.com/%d/fallback.jpg",
	)
	if assetURL == nil || *assetURL != "https://cdn.example.com/789/fallback.jpg" {
		t.Fatalf("assetURL = %v, want fallback asset url", assetURL)
	}
}

func TestSteamServiceFetchScreenshotURLsParsesEscapedPatternAndDedupes(t *testing.T) {
	service := &SteamService{
		client: &http.Client{Transport: steamRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://store.steampowered.com/app/999" {
				t.Fatalf("request url = %s, want store page", req.URL.String())
			}
			body := `"path_full":"https:\/\/cdn.example.com\/screen-1.jpg?x=1\u0026y=2"` +
				`"path_full":"https:\/\/cdn.example.com\/screen-1.jpg?x=1\u0026y=2"`
			return steamTextResponse(http.StatusOK, body), nil
		})},
	}

	urls := service.fetchScreenshotURLsFromStorePage(999, "")
	if len(urls) != 1 {
		t.Fatalf("urls = %#v, want one deduped screenshot", urls)
	}
	if urls[0] != `https://cdn.example.com/screen-1.jpg?x=1&y=2` {
		t.Fatalf("urls[0] = %q, want decoded screenshot url", urls[0])
	}
}

func TestSteamServicePreviewAssetsReturnsUpstreamErrorWhenRequestsFail(t *testing.T) {
	service := &SteamService{
		client: &http.Client{Transport: steamRoundTripper(func(req *http.Request) (*http.Response, error) {
			return steamTextResponse(http.StatusBadGateway, ""), nil
		})},
	}

	preview, err := service.PreviewAssets(321, "")
	if preview != nil {
		t.Fatalf("preview = %+v, want nil on upstream failure", preview)
	}
	if !errors.Is(err, ErrUpstream) {
		t.Fatalf("error = %v, want ErrUpstream", err)
	}
	if !strings.Contains(err.Error(), "schinese appdetails") || !strings.Contains(err.Error(), "english appdetails") {
		t.Fatalf("error = %q, want both locale failures in message", err.Error())
	}
}

func TestSteamServicePreviewAssetsReturnsNotFoundWhenAppMissingInAllLocales(t *testing.T) {
	service := &SteamService{
		client: &http.Client{Transport: steamRoundTripper(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case "https://store.steampowered.com/api/appdetails?appids=321&l=schinese":
				return steamJSONResponse(`{"321":{"success":false}}`), nil
			case "https://store.steampowered.com/api/appdetails?appids=321&l=english":
				return steamJSONResponse(`{"321":{"success":false}}`), nil
			default:
				return steamTextResponse(http.StatusNotFound, ""), nil
			}
		})},
	}

	preview, err := service.PreviewAssets(321, "")
	if preview != nil {
		t.Fatalf("preview = %+v, want nil when app is missing", preview)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestSteamServicePreviewAssetsUsesPrimaryAppDetailsWhenStorePageUnavailable(t *testing.T) {
	service := &SteamService{
		client: &http.Client{Transport: steamRoundTripper(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && req.URL.String() == "https://store.steampowered.com/api/appdetails?appids=777&l=schinese":
				return steamJSONResponse(`{"777":{"success":true,"data":{"name":"主条目","short_description":"主描述","release_date":{"coming_soon":false,"date":"Apr 6, 2024"},"developers":[" 主开发 "],"publishers":[" 主发行 "],"screenshots":[{"path_full":"https://cdn.example.com/primary-shot.jpg"}]}}}`), nil
			case req.Method == http.MethodGet && req.URL.String() == "https://store.steampowered.com/api/appdetails?appids=777&l=english":
				return steamJSONResponse(`{"777":{"success":true,"data":{"name":"Fallback Name","short_description":"Fallback Description","developers":["Fallback Dev"],"publishers":["Fallback Pub"],"screenshots":[{"path_full":"https://cdn.example.com/fallback-shot.jpg"}]}}}`), nil
			case req.Method == http.MethodGet && strings.HasPrefix(req.URL.String(), "https://store.steampowered.com/app/777"):
				return steamTextResponse(http.StatusNotFound, ""), nil
			case req.Method == http.MethodHead && strings.Contains(req.URL.String(), "/steam/apps/777/"):
				return steamTextResponse(http.StatusNotFound, ""), nil
			default:
				return steamTextResponse(http.StatusNotFound, ""), nil
			}
		})},
	}

	preview, err := service.PreviewAssets(777, "")
	if err != nil {
		t.Fatalf("PreviewAssets returned error: %v", err)
	}
	if preview.Name != "主条目" {
		t.Fatalf("Name = %q, want primary appdetails name", preview.Name)
	}
	if preview.Description != "主描述" {
		t.Fatalf("Description = %q, want primary short description", preview.Description)
	}
	if preview.ReleaseDate != "2024-04-06" {
		t.Fatalf("ReleaseDate = %q, want 2024-04-06", preview.ReleaseDate)
	}
	if len(preview.Developers) != 1 || preview.Developers[0] != "主开发" {
		t.Fatalf("Developers = %#v, want primary developers", preview.Developers)
	}
	if len(preview.Publishers) != 1 || preview.Publishers[0] != "主发行" {
		t.Fatalf("Publishers = %#v, want primary publishers", preview.Publishers)
	}
	if len(preview.ScreenshotURLs) != 1 || preview.ScreenshotURLs[0] != "https://cdn.example.com/primary-shot.jpg" {
		t.Fatalf("ScreenshotURLs = %#v, want primary appdetails screenshot", preview.ScreenshotURLs)
	}
}

type steamRoundTripper func(req *http.Request) (*http.Response, error)

func (fn steamRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func steamJSONResponse(body string) *http.Response {
	resp := steamTextResponse(http.StatusOK, body)
	resp.Header.Set("Content-Type", "application/json")
	return resp
}

func steamTextResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
