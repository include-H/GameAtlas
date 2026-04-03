package handlers

import (
	"encoding/json"
	"testing"
)

func TestGameAggregateUpdateRequestToInputPreservesRelationPatchPresence(t *testing.T) {
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

	input, err := request.toInput()
	if err != nil {
		t.Fatalf("toInput returned error: %v", err)
	}

	if input.Game.Title != "Aggregate Test" {
		t.Fatalf("title = %q, want Aggregate Test", input.Game.Title)
	}
	if !input.Game.SeriesID.Present {
		t.Fatalf("series_id should be marked present when request sends null")
	}
	if input.Game.SeriesID.Value != nil {
		t.Fatalf("series_id value = %v, want nil for explicit clear", *input.Game.SeriesID.Value)
	}
	if input.Game.PlatformIDs.Present {
		t.Fatalf("platform_ids should remain omitted when field is absent")
	}
	if !input.Game.DeveloperIDs.Present {
		t.Fatalf("developer_ids should be marked present when request sends []")
	}
	if len(input.Game.DeveloperIDs.Values) != 0 {
		t.Fatalf("developer_ids len = %d, want 0 for explicit clear", len(input.Game.DeveloperIDs.Values))
	}
	if !input.Game.TagIDs.Present {
		t.Fatalf("tag_ids should be marked present when request sends values")
	}
	if got := len(input.Game.TagIDs.Values); got != 3 {
		t.Fatalf("tag_ids len = %d, want decode to stay transport-only before service normalization", got)
	}
}

func TestGameCreateRequestToInputPreservesTransportSemantics(t *testing.T) {
	raw := []byte(`{
		"title": "Create Test",
		"series_id": null,
		"developer_ids": [],
		"tag_ids": [3, 3, 9]
	}`)

	var request gameCreateRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()

	if input.Title != "Create Test" {
		t.Fatalf("title = %q, want Create Test", input.Title)
	}
	if input.SeriesID != nil {
		t.Fatalf("series_id = %v, want nil for explicit clear/default create semantics", input.SeriesID)
	}
	if input.PlatformIDs != nil {
		t.Fatalf("platform_ids = %#v, want nil when field is omitted", input.PlatformIDs)
	}
	if got := len(input.DeveloperIDs); got != 0 {
		t.Fatalf("developer_ids len = %d, want 0 for explicit empty array", got)
	}
	if got := len(input.TagIDs); got != 3 {
		t.Fatalf("tag_ids len = %d, want decode to stay transport-only before service normalization", got)
	}
}

func TestGameFileWriteRequestToInputPreservesTransportSemantics(t *testing.T) {
	raw := []byte(`{
		"file_path": "/roms/demo.vhdx",
		"label": "Demo",
		"notes": null,
		"sort_order": 3
	}`)

	var request gameFileWriteRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()
	if input.FilePath != "/roms/demo.vhdx" {
		t.Fatalf("file_path = %q, want /roms/demo.vhdx", input.FilePath)
	}
	if input.Label == nil || *input.Label != "Demo" {
		t.Fatalf("label = %v, want Demo", input.Label)
	}
	if input.Notes != nil {
		t.Fatalf("notes = %v, want nil", input.Notes)
	}
	if input.SortOrder != 3 {
		t.Fatalf("sort_order = %d, want 3", input.SortOrder)
	}
}

func TestGameAggregateUpdateRequestToInputIgnoresRemovedFileSortOrderTransportField(t *testing.T) {
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
					"notes": "note",
					"sort_order": 99
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

	input, err := request.toInput()
	if err != nil {
		t.Fatalf("toInput returned error: %v", err)
	}

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
	if file.SortOrder != 0 {
		t.Fatalf("sort_order = %d, want aggregate decode to ignore removed transport field", file.SortOrder)
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
