package services

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestGamesServiceGetDetailUsesConfiguredPreviewVideoAndGroupsTags(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "detail-game", "Detail Game", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/detail-game/video-a.mp4", 0)
	insertServicesGameAsset(t, db, gameID, "video-b", "video", "/assets/detail-game/video-b.mp4", 1)
	insertServicesGameAsset(t, db, gameID, "screen-a", "screenshot", "/assets/detail-game/screen-a.png", 0)
	previewUID := "video-b"
	updateServicesGamePreviewVideo(t, db, gameID, &previewUID)
	groupID := insertServicesTagGroup(t, db, "detail-custom", "Detail Custom")
	actionID := insertServicesTag(t, db, groupID, "Action", "action")
	puzzleID := insertServicesTag(t, db, groupID, "Puzzle", "puzzle")
	linkServicesGameTag(t, db, gameID, actionID, 0)
	linkServicesGameTag(t, db, gameID, puzzleID, 1)
	insertServicesGameFile(t, db, gameID, "/roms/detail-game.rom", 0)

	service := newServicesGamesService(db)
	detail, err := service.GetDetail(gameID, true)
	if err != nil {
		t.Fatalf("GetDetail returned error: %v", err)
	}

	if detail.PreviewVideo == nil || detail.PreviewVideo.AssetUID != "video-b" {
		t.Fatalf("PreviewVideo = %#v, want video-b", detail.PreviewVideo)
	}
	if len(detail.PreviewVideos) != 2 {
		t.Fatalf("len(PreviewVideos) = %d, want 2", len(detail.PreviewVideos))
	}
	if len(detail.Screenshots) != 1 {
		t.Fatalf("len(Screenshots) = %d, want 1", len(detail.Screenshots))
	}
	if len(detail.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(detail.Files))
	}
	if len(detail.TagGroups) != 1 {
		t.Fatalf("len(TagGroups) = %d, want 1", len(detail.TagGroups))
	}
	if detail.TagGroups[0].ID != groupID || len(detail.TagGroups[0].Tags) != 2 {
		t.Fatalf("TagGroups[0] = %#v, want grouped tags", detail.TagGroups[0])
	}
	if detail.Platforms == nil || len(detail.Platforms) != 0 {
		t.Fatalf("Platforms = %#v, want empty non-nil slice", detail.Platforms)
	}
}

func TestGamesServiceGetDetailFallsBackToFirstVideoAndRejectsPrivateGame(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	publicGameID := insertServicesTestGame(t, db, "fallback-game", "Fallback Game", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, publicGameID, "video-a", "video", "/assets/fallback-game/video-a.mp4", 0)
	insertServicesGameAsset(t, db, publicGameID, "video-b", "video", "/assets/fallback-game/video-b.mp4", 1)
	missingPreviewUID := "missing-video"
	updateServicesGamePreviewVideo(t, db, publicGameID, &missingPreviewUID)
	privateGameID := insertServicesTestGame(t, db, "private-detail", "Private Detail", domain.GameVisibilityPrivate)

	service := newServicesGamesService(db)

	detail, err := service.GetDetail(publicGameID, true)
	if err != nil {
		t.Fatalf("GetDetail returned error: %v", err)
	}
	if detail.PreviewVideo == nil || detail.PreviewVideo.AssetUID != "video-a" {
		t.Fatalf("PreviewVideo = %#v, want fallback to first video", detail.PreviewVideo)
	}

	_, err = service.GetDetail(privateGameID, false)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetDetail private error = %v, want ErrNotFound", err)
	}
}

func TestNormalizeTimelineParamsNormalizesDatesAndValidatesCursorRange(t *testing.T) {
	params := domain.GamesTimelineParams{
		Limit:             120,
		FromDate:          "2024",
		ToDate:            "2024-2",
		CursorReleaseDate: "2024-02",
		CursorID:          7,
	}

	if err := normalizeTimelineParams(&params); err != nil {
		t.Fatalf("normalizeTimelineParams returned error: %v", err)
	}
	if params.Limit != 100 {
		t.Fatalf("Limit = %d, want 100", params.Limit)
	}
	if params.FromDate != "2024-01-01" {
		t.Fatalf("FromDate = %q, want 2024-01-01", params.FromDate)
	}
	if params.ToDate != "2024-02-01" {
		t.Fatalf("ToDate = %q, want 2024-02-01", params.ToDate)
	}
	if params.CursorReleaseDate != "2024-02-01" {
		t.Fatalf("CursorReleaseDate = %q, want 2024-02-01", params.CursorReleaseDate)
	}
	if params.Visibility != domain.GameVisibilityPublic {
		t.Fatalf("Visibility = %q, want public", params.Visibility)
	}

	err := normalizeTimelineParams(&domain.GamesTimelineParams{
		FromDate:          "2024-01-01",
		ToDate:            "2024-02-01",
		CursorReleaseDate: "2024-03-01",
		CursorID:          1,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("out-of-range cursor error = %v, want ErrValidation", err)
	}
}

func TestGamesServiceUpdateRejectsUnknownPreviewVideoAssetUID(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "update-preview", "Update Preview", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/update-preview/video-a.mp4", 0)
	service := newServicesGamesService(db)

	targetUID := "missing-video"
	_, err := service.Update(gameID, domain.GameWriteInput{
		Title:                "Update Preview",
		PreviewVideoAssetUID: &targetUID,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update error = %v, want ErrValidation", err)
	}
}

func TestGamesServiceUpdateAcceptsTrimmedExistingPreviewVideoAssetUID(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "update-trimmed-preview", "Update Trimmed Preview", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/update-trimmed-preview/video-a.mp4", 0)
	service := newServicesGamesService(db)

	targetUID := " video-a "
	game, err := service.Update(gameID, domain.GameWriteInput{
		Title:                "Update Trimmed Preview",
		PreviewVideoAssetUID: &targetUID,
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if game.PreviewVideoAssetUID == nil || *game.PreviewVideoAssetUID != "video-a" {
		t.Fatalf("PreviewVideoAssetUID = %v, want video-a", game.PreviewVideoAssetUID)
	}
}

func TestGamesServiceUpdateAggregateRejectsUnsupportedDeleteAssetType(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-invalid-asset", "Aggregate Invalid Asset", domain.GameVisibilityPublic)
	service := newServicesGamesService(db)

	_, _, err := service.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title: "Aggregate Invalid Asset",
		},
		DeleteAssets: []domain.GameAssetDeleteInput{
			{AssetType: "manual", Path: "/assets/manual.pdf"},
		},
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("UpdateAggregate error = %v, want ErrValidation", err)
	}
}

func TestGamesServiceUpdateAggregateRejectsUnknownPreviewVideoAssetUID(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-preview", "Aggregate Preview", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/aggregate-preview/video-a.mp4", 0)
	service := newServicesGamesService(db)

	targetUID := "missing-video"
	_, _, err := service.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title:                "Aggregate Preview",
			PreviewVideoAssetUID: &targetUID,
		},
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("UpdateAggregate error = %v, want ErrValidation", err)
	}
}

func TestGamesServiceUpdateAggregateReturnsMissingConfigForFileValidation(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-files", "Aggregate Files", domain.GameVisibilityPublic)
	service := newServicesGamesService(db)

	_, _, err := service.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title: "Aggregate Files",
		},
		Files: []domain.GameFileUpsertInput{
			{FilePath: "/tmp/demo.rom"},
		},
	})
	if !errors.Is(err, ErrMissingConfig) {
		t.Fatalf("UpdateAggregate error = %v, want ErrMissingConfig", err)
	}
}

func TestGamesServiceUpdateAggregateReturnsDeleteWarningsWhenAssetRemovalFails(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-warning", "Aggregate Warning", domain.GameVisibilityPublic)
	service := NewGamesService(
		config.Config{AssetsDir: t.TempDir()},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)

	game, warnings, err := service.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title: "Aggregate Warning",
		},
		DeleteAssets: []domain.GameAssetDeleteInput{
			{AssetType: "cover", Path: "/assets/../bad-cover.png"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}
	if game == nil || game.ID != gameID {
		t.Fatalf("game = %#v, want updated game %d", game, gameID)
	}
	if len(warnings) != 1 || warnings[0] != "/assets/../bad-cover.png" {
		t.Fatalf("warnings = %#v, want delete warning path", warnings)
	}
}

func TestGamesServiceUpdateAggregateNormalizesAndReplacesFiles(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	root := t.TempDir()
	firstPath := filepath.Join(root, "first.rom")
	secondPath := filepath.Join(root, "second.rom")
	if err := os.WriteFile(firstPath, []byte("first"), 0o644); err != nil {
		t.Fatalf("WriteFile(first) returned error: %v", err)
	}
	if err := os.WriteFile(secondPath, []byte("second"), 0o644); err != nil {
		t.Fatalf("WriteFile(second) returned error: %v", err)
	}

	gameID := insertServicesTestGame(t, db, "aggregate-files-success", "Aggregate Files Success", domain.GameVisibilityPublic)
	existingFileID := insertServicesGameFile(t, db, gameID, firstPath, 9)
	service := NewGamesService(
		config.Config{PrimaryROMRoot: root},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)

	label := "  Updated Label  "
	notes := "  Fresh Notes  "
	game, warnings, err := service.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title: "Aggregate Files Success",
		},
		Files: []domain.GameFileUpsertInput{
			{
				ID:       &existingFileID,
				FilePath: firstPath,
				Label:    &label,
				Notes:    &notes,
			},
			{
				FilePath: secondPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
	if game == nil || game.ID != gameID {
		t.Fatalf("game = %#v, want updated game %d", game, gameID)
	}

	files, err := repositories.NewGameFilesRepository(db).ListByGameID(gameID)
	if err != nil {
		t.Fatalf("ListByGameID returned error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0].ID != existingFileID || files[0].SortOrder != 0 {
		t.Fatalf("files[0] = %+v, want existing file updated to sort 0", files[0])
	}
	if files[0].Label == nil || *files[0].Label != "Updated Label" {
		t.Fatalf("files[0].Label = %v, want trimmed Updated Label", files[0].Label)
	}
	if files[0].Notes == nil || *files[0].Notes != "Fresh Notes" {
		t.Fatalf("files[0].Notes = %v, want trimmed Fresh Notes", files[0].Notes)
	}
	if files[1].ID == existingFileID || files[1].SortOrder != 1 {
		t.Fatalf("files[1] = %+v, want new file at sort 1", files[1])
	}
	if files[1].FilePath != secondPath {
		t.Fatalf("files[1].FilePath = %q, want %q", files[1].FilePath, secondPath)
	}
}

func TestGamesServiceUpdateAggregateReturnsForbiddenPathForFileOutsideRoot(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	root := t.TempDir()
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "outside.rom")
	if err := os.WriteFile(outsidePath, []byte("outside"), 0o644); err != nil {
		t.Fatalf("WriteFile(outside) returned error: %v", err)
	}

	gameID := insertServicesTestGame(t, db, "aggregate-outside-root", "Aggregate Outside Root", domain.GameVisibilityPublic)
	service := NewGamesService(
		config.Config{PrimaryROMRoot: root},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)

	_, _, err := service.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title: "Aggregate Outside Root",
		},
		Files: []domain.GameFileUpsertInput{
			{FilePath: outsidePath},
		},
	})
	if !errors.Is(err, ErrForbiddenPath) {
		t.Fatalf("UpdateAggregate error = %v, want ErrForbiddenPath", err)
	}
}

func TestGamesServiceUpdateAggregateDeletesOmittedExistingFiles(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	root := t.TempDir()
	existingPath := filepath.Join(root, "existing.rom")
	if err := os.WriteFile(existingPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing) returned error: %v", err)
	}

	gameID := insertServicesTestGame(t, db, "aggregate-delete-files", "Aggregate Delete Files", domain.GameVisibilityPublic)
	insertServicesGameFile(t, db, gameID, existingPath, 0)
	service := NewGamesService(
		config.Config{PrimaryROMRoot: root},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)

	_, warnings, err := service.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title: "Aggregate Delete Files",
		},
		Files: []domain.GameFileUpsertInput{},
	})
	if err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}

	files, err := repositories.NewGameFilesRepository(db).ListByGameID(gameID)
	if err != nil {
		t.Fatalf("ListByGameID returned error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("files = %#v, want all files deleted", files)
	}
}

func TestGamesServiceUpdateAggregateReturnsNotFoundForMissingExistingFileID(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	root := t.TempDir()
	filePath := filepath.Join(root, "existing.rom")
	if err := os.WriteFile(filePath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing) returned error: %v", err)
	}

	gameID := insertServicesTestGame(t, db, "aggregate-missing-file-id", "Aggregate Missing File ID", domain.GameVisibilityPublic)
	service := NewGamesService(
		config.Config{PrimaryROMRoot: root},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)

	missingID := int64(9999)
	_, _, err := service.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title: "Aggregate Missing File ID",
		},
		Files: []domain.GameFileUpsertInput{
			{
				ID:       &missingID,
				FilePath: filePath,
			},
		},
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateAggregate error = %v, want ErrNotFound", err)
	}
}

func TestGamesServiceUpdateAggregateReturnsNotFoundForMissingScreenshotReorderUID(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-missing-shot", "Aggregate Missing Shot", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "shot-a", "screenshot", "/assets/aggregate-missing-shot/shot-a.png", 0)
	service := newServicesGamesService(db)

	_, _, err := service.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title: "Aggregate Missing Shot",
		},
		ScreenshotOrderAssetUIDs: []string{"missing-shot"},
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateAggregate error = %v, want ErrNotFound", err)
	}
}

func TestGamesServiceUpdateAggregateReturnsNotFoundForMissingVideoReorderUID(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-missing-video", "Aggregate Missing Video", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/aggregate-missing-video/video-a.mp4", 0)
	service := newServicesGamesService(db)

	_, _, err := service.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title: "Aggregate Missing Video",
		},
		VideoOrderAssetUIDs: []string{"missing-video"},
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateAggregate error = %v, want ErrNotFound", err)
	}
}

func TestGamesServiceUpdateAggregateReordersVideos(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-reorder-video", "Aggregate Reorder Video", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/aggregate-reorder-video/video-a.mp4", 5)
	insertServicesGameAsset(t, db, gameID, "video-b", "video", "/assets/aggregate-reorder-video/video-b.mp4", 6)
	previewUID := "video-a"
	service := newServicesGamesService(db)

	game, warnings, err := service.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title:                "Aggregate Reorder Video",
			PreviewVideoAssetUID: &previewUID,
		},
		VideoOrderAssetUIDs: []string{" video-b ", " video-a "},
	})
	if err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
	if game == nil || game.PreviewVideoAssetUID == nil || *game.PreviewVideoAssetUID != "video-a" {
		t.Fatalf("PreviewVideoAssetUID = %v, want video-a", game.PreviewVideoAssetUID)
	}

	videos, err := repositories.NewGamesRepository(db).ListVideos(gameID)
	if err != nil {
		t.Fatalf("ListVideos returned error: %v", err)
	}
	if len(videos) != 2 {
		t.Fatalf("len(videos) = %d, want 2", len(videos))
	}
	if videos[0].AssetUID != "video-b" || videos[0].SortOrder != 0 {
		t.Fatalf("videos[0] = %+v, want video-b at sort 0", videos[0])
	}
	if videos[1].AssetUID != "video-a" || videos[1].SortOrder != 1 {
		t.Fatalf("videos[1] = %+v, want video-a at sort 1", videos[1])
	}
}

func TestGamesServiceUpdateAggregateDeletesPrimaryVideoAndFallsBack(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "aggregate-delete-primary-video", "Aggregate Delete Primary Video", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/aggregate-delete-primary-video/video-a.mp4", 0)
	insertServicesGameAsset(t, db, gameID, "video-b", "video", "/assets/aggregate-delete-primary-video/video-b.mp4", 1)
	writeServicesAssetFile(t, assetsDir, "aggregate-delete-primary-video", "video-a.mp4", []byte("a"))
	writeServicesAssetFile(t, assetsDir, "aggregate-delete-primary-video", "video-b.mp4", []byte("b"))
	previewUID := "video-a"
	service := NewGamesService(
		config.Config{AssetsDir: assetsDir},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)

	game, warnings, err := service.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title:                "Aggregate Delete Primary Video",
			PreviewVideoAssetUID: &previewUID,
		},
		DeleteAssets: []domain.GameAssetDeleteInput{
			{AssetType: "video", AssetUID: "video-a"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
	if game == nil || game.PreviewVideoAssetUID == nil || *game.PreviewVideoAssetUID != "video-b" {
		t.Fatalf("PreviewVideoAssetUID = %v, want video-b", game.PreviewVideoAssetUID)
	}

	videos, err := repositories.NewGamesRepository(db).ListVideos(gameID)
	if err != nil {
		t.Fatalf("ListVideos returned error: %v", err)
	}
	if len(videos) != 1 || videos[0].AssetUID != "video-b" {
		t.Fatalf("videos = %+v, want only fallback video-b", videos)
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "aggregate-delete-primary-video", "video-a.mp4")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected deleted primary video file, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "aggregate-delete-primary-video", "video-b.mp4")); err != nil {
		t.Fatalf("expected fallback video file to remain, got err=%v", err)
	}
}
