package handlers

import (
	"testing"

	"github.com/hao/game/internal/domain"
)

func TestToDirectoryListResponseMapsDirectoryDTO(t *testing.T) {
	size := int64(123)
	response := toDirectoryListResponse(&domain.DirectoryListResponse{
		CurrentPath: "/roms",
		Items: []domain.DirectoryItem{
			{Name: "demo", Path: "/roms/demo", IsDirectory: true},
			{Name: "demo.vhdx", Path: "/roms/demo.vhdx", IsDirectory: false, SizeBytes: &size},
		},
	})

	if response.CurrentPath != "/roms" {
		t.Fatalf("current_path = %q, want /roms", response.CurrentPath)
	}
	if len(response.Items) != 2 || !response.Items[0].IsDirectory {
		t.Fatalf("items = %+v, want mapped directory entries", response.Items)
	}
	if response.Items[1].SizeBytes == nil || *response.Items[1].SizeBytes != size {
		t.Fatalf("size_bytes = %v, want %d", response.Items[1].SizeBytes, size)
	}
}

func TestToSteamResponsesMapSteamTransportDTOs(t *testing.T) {
	releaseDate := "2024-03-05"
	tinyImage := "https://cdn.example.com/tiny.jpg"
	searchResponse := toSteamSearchResultResponses([]domain.SteamSearchResult{{
		AppID:       10,
		Name:        "Portal",
		ReleaseDate: &releaseDate,
		TinyImage:   &tinyImage,
	}})

	if len(searchResponse) != 1 || searchResponse[0].AppID != 10 || searchResponse[0].Name != "Portal" {
		t.Fatalf("search response = %+v, want mapped steam search result", searchResponse)
	}

	preview := toSteamAssetsPreviewResponse(&domain.SteamAssetsPreview{
		AppID:          20,
		Name:           "Half-Life",
		Description:    "Desc",
		ReleaseDate:    "2024-04-06",
		Developers:     []string{"Valve"},
		Publishers:     []string{"Valve"},
		ScreenshotURLs: []string{"https://cdn.example.com/shot.jpg"},
	})
	if preview.AppID != 20 || preview.Name != "Half-Life" || len(preview.ScreenshotURLs) != 1 {
		t.Fatalf("preview = %+v, want mapped steam preview", preview)
	}
}

func TestToPendingIssueCatalogResponseMapsCatalogDTO(t *testing.T) {
	response := toPendingIssueCatalogResponse(domain.PendingIssueCatalog{
		Groups: []domain.PendingIssueDefinition{
			{Key: domain.PendingIssueMissingAssets, Label: "缺少图片", Description: "desc"},
		},
		Details: []domain.PendingIssueDetailDefinition{
			{Key: domain.PendingIssueDetailMissingCover, Label: "缺封面", Group: domain.PendingIssueMissingAssets},
		},
	})

	if len(response.Groups) != 1 || response.Groups[0].Key != "missing-assets" {
		t.Fatalf("groups = %+v, want mapped pending issue groups", response.Groups)
	}
	if len(response.Details) != 1 || response.Details[0].Group != "missing-assets" {
		t.Fatalf("details = %+v, want mapped pending issue details", response.Details)
	}
}

func TestToPendingIssueEvaluationResponseMapsNestedPendingDTO(t *testing.T) {
	reason := "accepted"
	response := toPendingIssueEvaluationResponse(&domain.PendingIssueEvaluation{
		Groups: []domain.PendingIssueKey{
			domain.PendingIssueMissingAssets,
		},
		Details: []domain.PendingIssueDetailState{
			{
				Key:     domain.PendingIssueDetailMissingCover,
				Group:   domain.PendingIssueMissingAssets,
				Ignored: true,
				Reason:  &reason,
			},
		},
		Severe: true,
	})

	if response == nil || len(response.Groups) != 1 || response.Groups[0] != "missing-assets" {
		t.Fatalf("response = %+v, want mapped pending issue groups", response)
	}
	if len(response.Details) != 1 || response.Details[0].Key != "missing-cover" || response.Details[0].Reason == nil {
		t.Fatalf("details = %+v, want mapped pending issue detail state", response.Details)
	}
}

func TestToPendingIssueCountSummaryResponseMapsPaginationDTO(t *testing.T) {
	response := toPendingIssueCountSummaryResponse(&domain.PendingIssueCountSummary{
		Groups: map[domain.PendingIssueKey]int{
			domain.PendingIssueMissingAssets: 2,
			domain.PendingIssueMissingWiki:   1,
		},
		IgnoredTotal: 3,
	})

	if response == nil || response.Groups["missing-assets"] != 2 || response.Groups["missing-wiki"] != 1 {
		t.Fatalf("response = %+v, want mapped count summary", response)
	}
	if response.IgnoredTotal != 3 {
		t.Fatalf("ignored_total = %d, want 3", response.IgnoredTotal)
	}
}
