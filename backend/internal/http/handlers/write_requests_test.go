package handlers

import (
	"encoding/json"
	"testing"
)

func TestMetadataWriteRequestToInputPreservesTransportSemantics(t *testing.T) {
	raw := []byte(`{"name":" Series ","slug":" custom-slug ","sort_order":7}`)

	var request metadataWriteRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()
	if input.Name != " Series " {
		t.Fatalf("name = %q, want original transport value", input.Name)
	}
	if input.Slug == nil || *input.Slug != " custom-slug " {
		t.Fatalf("slug = %v, want original transport value", input.Slug)
	}
	if input.SortOrder == nil || *input.SortOrder != 7 {
		t.Fatalf("sort_order = %v, want 7", input.SortOrder)
	}
}

func TestTagGroupWriteRequestToInputPreservesTransportSemantics(t *testing.T) {
	raw := []byte(`{"key":"genre","name":" Genre ","description":" Desc ","sort_order":3,"allow_multiple":false,"is_filterable":true}`)

	var request tagGroupWriteRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()
	if input.Key != "genre" || input.Name != " Genre " {
		t.Fatalf("input = %+v, want original key/name transport values", input)
	}
	if input.Description == nil || *input.Description != " Desc " {
		t.Fatalf("description = %v, want original transport value", input.Description)
	}
	if input.SortOrder == nil || *input.SortOrder != 3 {
		t.Fatalf("sort_order = %v, want 3", input.SortOrder)
	}
	if input.AllowMultiple == nil || *input.AllowMultiple {
		t.Fatalf("allow_multiple = %v, want false", input.AllowMultiple)
	}
	if input.IsFilterable == nil || !*input.IsFilterable {
		t.Fatalf("is_filterable = %v, want true", input.IsFilterable)
	}
}

func TestTagWriteRequestToInputPreservesTransportSemantics(t *testing.T) {
	raw := []byte(`{"group_id":9,"name":" Tag ","slug":" custom ","parent_id":2,"sort_order":4,"is_active":false}`)

	var request tagWriteRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()
	if input.GroupID != 9 || input.Name != " Tag " {
		t.Fatalf("input = %+v, want original transport values", input)
	}
	if input.ParentID == nil || *input.ParentID != 2 {
		t.Fatalf("parent_id = %v, want 2", input.ParentID)
	}
	if input.SortOrder == nil || *input.SortOrder != 4 {
		t.Fatalf("sort_order = %v, want 4", input.SortOrder)
	}
	if input.IsActive == nil || *input.IsActive {
		t.Fatalf("is_active = %v, want false", input.IsActive)
	}
}

func TestDeleteAssetRequestToInputPreservesTransportSemantics(t *testing.T) {
	raw := []byte(`{"game_id":3,"asset_id":5,"asset_uid":"asset-a","asset_type":"video","path":"/assets/demo.mp4"}`)

	var request deleteAssetRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()
	if input.GameID != 3 || input.AssetType != "video" || input.Path != "/assets/demo.mp4" {
		t.Fatalf("input = %+v, want original transport values", input)
	}
	if input.AssetID == nil || *input.AssetID != 5 {
		t.Fatalf("asset_id = %v, want 5", input.AssetID)
	}
	if input.AssetUID != "asset-a" {
		t.Fatalf("asset_uid = %q, want asset-a", input.AssetUID)
	}
}

func TestAssetOrderUpdateRequestToInputPreservesTransportSemantics(t *testing.T) {
	raw := []byte(`{"game_id":11,"asset_uids":["shot-a","shot-b"]}`)

	var request assetOrderUpdateRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	screenshotInput := request.toScreenshotInput()
	videoInput := request.toVideoInput()
	if screenshotInput.GameID != 11 || videoInput.GameID != 11 {
		t.Fatalf("game_id = %d/%d, want 11", screenshotInput.GameID, videoInput.GameID)
	}
	if len(screenshotInput.AssetUIDs) != 2 || screenshotInput.AssetUIDs[0] != "shot-a" || screenshotInput.AssetUIDs[1] != "shot-b" {
		t.Fatalf("screenshot asset_uids = %#v, want original order", screenshotInput.AssetUIDs)
	}
	if len(videoInput.AssetUIDs) != 2 || videoInput.AssetUIDs[0] != "shot-a" || videoInput.AssetUIDs[1] != "shot-b" {
		t.Fatalf("video asset_uids = %#v, want original order", videoInput.AssetUIDs)
	}
}

func TestSteamApplyAssetsRequestToInputUsesQueryFallback(t *testing.T) {
	raw := []byte(`{"cover_url":"https://example.com/cover.jpg","screenshot_urls":["https://example.com/1.jpg"]}`)

	var request steamApplyAssetsRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput("7")
	if input.GameID != 7 {
		t.Fatalf("game_id = %d, want query fallback 7", input.GameID)
	}
	if input.CoverURL == nil || *input.CoverURL != "https://example.com/cover.jpg" {
		t.Fatalf("cover_url = %v, want original transport value", input.CoverURL)
	}
	if len(input.ScreenshotURLs) != 1 || input.ScreenshotURLs[0] != "https://example.com/1.jpg" {
		t.Fatalf("screenshot_urls = %#v, want original transport order", input.ScreenshotURLs)
	}
}

func TestSteamApplyAssetsRequestToInputPrefersBodyGameID(t *testing.T) {
	raw := []byte(`{"game_id":9}`)

	var request steamApplyAssetsRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput("7")
	if input.GameID != 9 {
		t.Fatalf("game_id = %d, want body value 9", input.GameID)
	}
}

func TestWikiWriteRequestToInputPreservesTransportSemantics(t *testing.T) {
	raw := []byte(`{"content":"# Demo","change_summary":"  summary  "}`)

	var request wikiWriteRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()
	if input.Content != "# Demo" {
		t.Fatalf("content = %q, want original transport value", input.Content)
	}
	if input.ChangeSummary == nil || *input.ChangeSummary != "  summary  " {
		t.Fatalf("change_summary = %v, want original transport value", input.ChangeSummary)
	}
}
