package handlers

import (
	"encoding/json"
	"testing"
)

func TestGameAggregateUpdateRequestToInputKeepsTransportRelationShapeUntouched(t *testing.T) {
	raw := []byte(`{
		"game": {
			"title": "Aggregate Test",
			"series_id": null,
			"developer_ids": [],
			"tag_ids": [3, 3, 9]
		},
		"assets": {}
	}`)

	var request gameAggregateUpdateRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()

	if input.Game.Title != "Aggregate Test" {
		t.Fatalf("title = %q, want Aggregate Test", input.Game.Title)
	}
	if input.Game.SeriesID != nil {
		t.Fatalf("series_id = %v, want nil for explicit clear", input.Game.SeriesID)
	}
	if input.Game.PlatformIDs != nil {
		t.Fatalf("platform_ids = %#v, want nil when field is omitted", input.Game.PlatformIDs)
	}
	if got := len(input.Game.DeveloperIDs); got != 0 {
		t.Fatalf("developer_ids len = %d, want 0 for explicit clear", got)
	}
	if got := len(input.Game.TagIDs); got != 3 {
		t.Fatalf("tag_ids len = %d, want decode to stay transport-only before service normalization", got)
	}
}

func TestGameCreateRequestToInputKeepsOnlyQuickCreateFields(t *testing.T) {
	raw := []byte(`{
		"title": "Create Test",
		"visibility": "private"
	}`)

	var request gameCreateRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()

	if input.Title != "Create Test" {
		t.Fatalf("title = %q, want Create Test", input.Title)
	}
	if input.Visibility != "private" {
		t.Fatalf("visibility = %q, want private", input.Visibility)
	}
}

func TestGameAggregateUpdateRequestToInputPreservesAssetPayloadWithoutLegacyFileSortOrder(t *testing.T) {
	raw := []byte(`{
		"game": {
			"title": "Aggregate Test"
		},
		"assets": {
			"files": [
				{
					"id": 7,
					"file_path": "/roms/demo.vhdx",
					"label": "Demo",
					"notes": "note"
				}
			],
			"delete_assets": [
				{
					"asset_type": "video",
					"path": "/assets/demo/video.mp4",
					"asset_id": 11,
					"asset_uid": "video-a"
				}
			],
			"screenshot_order_asset_uids": ["shot-a"],
			"video_order_asset_uids": ["video-a"]
		}
	}`)

	var request gameAggregateUpdateRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()

	if len(input.Assets.Files) != 1 {
		t.Fatalf("files len = %d, want 1", len(input.Assets.Files))
	}
	file := input.Assets.Files[0]
	if file.ID == nil || *file.ID != 7 {
		t.Fatalf("file id = %v, want 7", file.ID)
	}
	if file.FilePath != "/roms/demo.vhdx" {
		t.Fatalf("file_path = %q, want /roms/demo.vhdx", file.FilePath)
	}
	if len(input.Assets.DeleteAssets) != 1 {
		t.Fatalf("delete_assets len = %d, want 1", len(input.Assets.DeleteAssets))
	}
	if input.Assets.DeleteAssets[0].AssetUID != "video-a" {
		t.Fatalf("delete asset uid = %q, want video-a", input.Assets.DeleteAssets[0].AssetUID)
	}
	if len(input.Assets.ScreenshotOrderAssetUIDs) != 1 || input.Assets.ScreenshotOrderAssetUIDs[0] != "shot-a" {
		t.Fatalf("screenshot order = %#v, want [shot-a]", input.Assets.ScreenshotOrderAssetUIDs)
	}
	if len(input.Assets.VideoOrderAssetUIDs) != 1 || input.Assets.VideoOrderAssetUIDs[0] != "video-a" {
		t.Fatalf("video order = %#v, want [video-a]", input.Assets.VideoOrderAssetUIDs)
	}
}

func TestGameAggregateUpdateRequestToInputKeepsOmittedAssetOrderSlicesNil(t *testing.T) {
	raw := []byte(`{
		"game": {
			"title": "Aggregate Test"
		},
		"assets": {}
	}`)

	var request gameAggregateUpdateRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()

	if input.Assets.ScreenshotOrderAssetUIDs != nil {
		t.Fatalf("screenshot_order_asset_uids = %#v, want nil when field is omitted", input.Assets.ScreenshotOrderAssetUIDs)
	}
	if input.Assets.VideoOrderAssetUIDs != nil {
		t.Fatalf("video_order_asset_uids = %#v, want nil when field is omitted", input.Assets.VideoOrderAssetUIDs)
	}
}
