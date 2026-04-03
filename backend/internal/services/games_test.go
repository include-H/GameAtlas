package services

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestGamesServiceGetDetailUsesFirstSortedVideoAndGroupsTags(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "detail-game", "Detail Game", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/detail-game/video-a.mp4", 0)
	insertServicesGameAsset(t, db, gameID, "video-b", "video", "/assets/detail-game/video-b.mp4", 1)
	insertServicesGameAsset(t, db, gameID, "screen-a", "screenshot", "/assets/detail-game/screen-a.png", 0)
	groupID := insertServicesTagGroup(t, db, "detail-custom", "Detail Custom")
	actionID := insertServicesTag(t, db, groupID, "Action", "action")
	puzzleID := insertServicesTag(t, db, groupID, "Puzzle", "puzzle")
	linkServicesGameTag(t, db, gameID, actionID, 0)
	linkServicesGameTag(t, db, gameID, puzzleID, 1)
	insertServicesGameFile(t, db, gameID, "/roms/detail-game.rom", 0)

	service := newServicesDetailService(db)
	detail, err := service.Get(gameID, true)
	if err != nil {
		t.Fatalf("GetDetail returned error: %v", err)
	}

	if len(detail.PreviewVideos) != 2 {
		t.Fatalf("len(PreviewVideos) = %d, want 2", len(detail.PreviewVideos))
	}
	if detail.PreviewVideos[0].AssetUID != "video-a" {
		t.Fatalf("PreviewVideos[0] = %#v, want first sorted video-a", detail.PreviewVideos[0])
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
	if !detail.TagGroups[0].AllowMultiple {
		t.Fatalf("TagGroups[0].AllowMultiple = %v, want true", detail.TagGroups[0].AllowMultiple)
	}
	if !detail.TagGroups[0].IsFilterable {
		t.Fatalf("TagGroups[0].IsFilterable = %v, want true", detail.TagGroups[0].IsFilterable)
	}
	if detail.Platforms == nil || len(detail.Platforms) != 0 {
		t.Fatalf("Platforms = %#v, want empty non-nil slice", detail.Platforms)
	}
}

func TestGamesServiceGetDetailPreservesTagGroupCapabilities(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "detail-tag-capabilities", "Detail Tag Capabilities", domain.GameVisibilityPublic)
	singleSelectGroupID := insertServicesTagGroupWithOptions(t, db, "mode", "Mode", false, true)
	internalGroupID := insertServicesTagGroupWithOptions(t, db, "internal", "Internal", true, false)
	storyID := insertServicesTag(t, db, singleSelectGroupID, "Story", "story")
	curatedID := insertServicesTag(t, db, internalGroupID, "Curated", "curated")
	linkServicesGameTag(t, db, gameID, storyID, 0)
	linkServicesGameTag(t, db, gameID, curatedID, 1)

	service := newServicesDetailService(db)
	detail, err := service.Get(gameID, true)
	if err != nil {
		t.Fatalf("GetDetail returned error: %v", err)
	}

	if len(detail.TagGroups) != 2 {
		t.Fatalf("len(TagGroups) = %d, want 2", len(detail.TagGroups))
	}

	groupByID := make(map[int64]domain.GameTagGroup, len(detail.TagGroups))
	for _, group := range detail.TagGroups {
		groupByID[group.ID] = group
	}

	singleSelectGroup, ok := groupByID[singleSelectGroupID]
	if !ok {
		t.Fatalf("single-select group %d missing from %#v", singleSelectGroupID, detail.TagGroups)
	}
	if singleSelectGroup.AllowMultiple {
		t.Fatalf("singleSelectGroup.AllowMultiple = %v, want false", singleSelectGroup.AllowMultiple)
	}
	if !singleSelectGroup.IsFilterable {
		t.Fatalf("singleSelectGroup.IsFilterable = %v, want true", singleSelectGroup.IsFilterable)
	}

	internalGroup, ok := groupByID[internalGroupID]
	if !ok {
		t.Fatalf("internal group %d missing from %#v", internalGroupID, detail.TagGroups)
	}
	if !internalGroup.AllowMultiple {
		t.Fatalf("internalGroup.AllowMultiple = %v, want true", internalGroup.AllowMultiple)
	}
	if internalGroup.IsFilterable {
		t.Fatalf("internalGroup.IsFilterable = %v, want false", internalGroup.IsFilterable)
	}
}

func TestGamesServiceGetDetailUsesFirstVideoAndRejectsPrivateGame(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	publicGameID := insertServicesTestGame(t, db, "fallback-game", "Fallback Game", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, publicGameID, "video-a", "video", "/assets/fallback-game/video-a.mp4", 0)
	insertServicesGameAsset(t, db, publicGameID, "video-b", "video", "/assets/fallback-game/video-b.mp4", 1)
	privateGameID := insertServicesTestGame(t, db, "private-detail", "Private Detail", domain.GameVisibilityPrivate)

	service := newServicesDetailService(db)

	detail, err := service.Get(publicGameID, true)
	if err != nil {
		t.Fatalf("GetDetail returned error: %v", err)
	}
	if len(detail.PreviewVideos) == 0 || detail.PreviewVideos[0].AssetUID != "video-a" {
		t.Fatalf("PreviewVideos = %#v, want first sorted video at index 0", detail.PreviewVideos)
	}

	_, err = service.Get(privateGameID, false)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetDetail private error = %v, want ErrNotFound", err)
	}
}

func TestGamesServiceListReturnsOverrideLookupError(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	insertServicesTestGame(t, db, "pending-list-error", "Pending List Error", domain.GameVisibilityPublic)

	if _, err := db.Exec(`DROP TABLE game_review_issue_overrides`); err != nil {
		t.Fatalf("drop override table: %v", err)
	}

	service := newServicesCatalogService(db)

	_, err := service.List(domain.GamesListParams{Page: 1, Limit: 20})
	if err == nil {
		t.Fatal("List returned nil error, want override lookup failure")
	}
	if !strings.Contains(err.Error(), "list review overrides") {
		t.Fatalf("List error = %v, want review override lookup context", err)
	}
}

func TestGamesServiceDeleteRemovesTrackedAssetFiles(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "delete-game-assets", "Delete Game Assets", domain.GameVisibilityPublic)
	if _, err := db.Exec(`
		UPDATE games
		SET cover_image = ?, banner_image = ?
		WHERE id = ?
	`, "/assets/delete-game-assets/cover.png", "/assets/delete-game-assets/banner.png", gameID); err != nil {
		t.Fatalf("set game images: %v", err)
	}
	insertServicesGameAsset(t, db, gameID, "shot-a", "screenshot", "/assets/delete-game-assets/shot-a.png", 0)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/delete-game-assets/video-a.mp4", 1)
	writeServicesAssetFile(t, assetsDir, "delete-game-assets", "cover.png", []byte("cover"))
	writeServicesAssetFile(t, assetsDir, "delete-game-assets", "banner.png", []byte("banner"))
	writeServicesAssetFile(t, assetsDir, "delete-game-assets", "shot-a.png", []byte("shot"))
	writeServicesAssetFile(t, assetsDir, "delete-game-assets", "video-a.mp4", []byte("video"))

	service := newServicesAggregateService(db, config.Config{AssetsDir: assetsDir})

	result, err := service.Delete(gameID)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if result == nil || len(result.Warnings) != 0 {
		t.Fatalf("result = %#v, want no warnings", result)
	}
	if _, err := repositories.NewGamesRepository(db).GetByID(gameID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected deleted game row, got err=%v", err)
	}
	for _, name := range []string{"cover.png", "banner.png", "shot-a.png", "video-a.mp4"} {
		if _, err := os.Stat(filepath.Join(assetsDir, "delete-game-assets", name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %s to be deleted, got err=%v", name, err)
		}
	}
	assertNoAssetCleanupTasks(t, db)
}

func TestGamesServiceDeleteQueuesCleanupTaskWhenFileRemovalFails(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "delete-game-warning", "Delete Game Warning", domain.GameVisibilityPublic)
	if _, err := db.Exec(`
		UPDATE games
		SET cover_image = ?
		WHERE id = ?
	`, "/assets/../bad-delete-cover.png", gameID); err != nil {
		t.Fatalf("set game cover image: %v", err)
	}

	service := newServicesAggregateService(db, config.Config{AssetsDir: filepath.Join(t.TempDir(), "assets")})

	result, err := service.Delete(gameID)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if result == nil || len(result.Warnings) != 1 || result.Warnings[0] != "/assets/../bad-delete-cover.png" {
		t.Fatalf("result = %#v, want bad cover warning", result)
	}
	if _, err := repositories.NewGamesRepository(db).GetByID(gameID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected deleted game row, got err=%v", err)
	}

	task := mustLoadAssetCleanupTask(t, db, "/assets/../bad-delete-cover.png")
	if task.Source != "games.delete" {
		t.Fatalf("task.Source = %q, want games.delete", task.Source)
	}
	if task.AttemptCount != 1 {
		t.Fatalf("task.AttemptCount = %d, want 1", task.AttemptCount)
	}
}

func TestGamesServiceProcessPendingAssetCleanupDeletesRecoveredFile(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	path := "/assets/retry-game/retry-cover.png"
	writeServicesAssetFile(t, assetsDir, "retry-game", "retry-cover.png", []byte("cover"))

	tasksRepo := repositories.NewAssetCleanupTasksRepository(db)
	if err := tasksRepo.Enqueue(path, "games.delete", "temporary failure"); err != nil {
		t.Fatalf("enqueue cleanup task: %v", err)
	}

	service := newServicesAggregateService(db, config.Config{AssetsDir: assetsDir})

	processed, err := service.ProcessPendingAssetCleanup(100)
	if err != nil {
		t.Fatalf("ProcessPendingAssetCleanup returned error: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "retry-game", "retry-cover.png")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected retry asset file to be deleted, got err=%v", err)
	}
	assertAssetCleanupTaskMissing(t, db, path)
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

func TestValidateAndTrimGameInputNormalizesSharedCoreFields(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	tagsRepo := repositories.NewTagsRepository(db)
	titleAlt := " Alt "
	summary := "   "
	engine := " Unreal Engine 5 "

	trimmed, err := validateAndTrimGameInput(domain.GameWriteInput{
		GameCoreInput: domain.GameCoreInput{
			Title:      "  Shared Core Game  ",
			TitleAlt:   &titleAlt,
			Visibility: "  ",
			Summary:    &summary,
			Engine:     &engine,
		},
		PlatformIDs:  []int64{3, 3, 1},
		DeveloperIDs: []int64{4, 4},
		PublisherIDs: []int64{7, 2, 7},
	}, tagsRepo)
	if err != nil {
		t.Fatalf("validateAndTrimGameInput returned error: %v", err)
	}

	if trimmed.Title != "Shared Core Game" {
		t.Fatalf("Title = %q, want trimmed title", trimmed.Title)
	}
	if trimmed.TitleAlt == nil || *trimmed.TitleAlt != "Alt" {
		t.Fatalf("TitleAlt = %v, want Alt", trimmed.TitleAlt)
	}
	if trimmed.Visibility != domain.GameVisibilityPublic {
		t.Fatalf("Visibility = %q, want public default", trimmed.Visibility)
	}
	if trimmed.Summary != nil {
		t.Fatalf("Summary = %v, want nil after blank trim", trimmed.Summary)
	}
	if trimmed.Engine == nil || *trimmed.Engine != "Unreal Engine 5" {
		t.Fatalf("Engine = %v, want trimmed engine", trimmed.Engine)
	}
	if len(trimmed.PlatformIDs) != 2 || trimmed.PlatformIDs[0] != 3 || trimmed.PlatformIDs[1] != 1 {
		t.Fatalf("PlatformIDs = %#v, want deduped [3 1]", trimmed.PlatformIDs)
	}
	if len(trimmed.DeveloperIDs) != 1 || trimmed.DeveloperIDs[0] != 4 {
		t.Fatalf("DeveloperIDs = %#v, want deduped [4]", trimmed.DeveloperIDs)
	}
	if len(trimmed.PublisherIDs) != 2 || trimmed.PublisherIDs[0] != 7 || trimmed.PublisherIDs[1] != 2 {
		t.Fatalf("PublisherIDs = %#v, want deduped [7 2]", trimmed.PublisherIDs)
	}
	if len(trimmed.TagIDs) != 0 {
		t.Fatalf("TagIDs = %#v, want empty slice", trimmed.TagIDs)
	}
}

func TestValidateAndTrimGameAggregatePatchInputNormalizesSharedCoreFields(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	tagsRepo := repositories.NewTagsRepository(db)
	titleAlt := " Alt Patch "
	summary := "   "
	engine := " RE Engine "

	trimmed, err := validateAndTrimGameAggregatePatchInput(domain.GameAggregatePatchInput{
		GameCoreInput: domain.GameCoreInput{
			Title:      "  Aggregate Shared Core  ",
			TitleAlt:   &titleAlt,
			Visibility: " ",
			Summary:    &summary,
			Engine:     &engine,
		},
		PlatformIDs:  domain.Int64SlicePatch{Present: true, Values: []int64{8, 8, 5}},
		DeveloperIDs: domain.Int64SlicePatch{Present: true, Values: []int64{6, 6}},
		PublisherIDs: domain.Int64SlicePatch{Present: true, Values: []int64{9, 4, 9}},
		TagIDs:       domain.Int64SlicePatch{Present: true, Values: nil},
	}, tagsRepo)
	if err != nil {
		t.Fatalf("validateAndTrimGameAggregatePatchInput returned error: %v", err)
	}

	if trimmed.Title != "Aggregate Shared Core" {
		t.Fatalf("Title = %q, want trimmed title", trimmed.Title)
	}
	if trimmed.TitleAlt == nil || *trimmed.TitleAlt != "Alt Patch" {
		t.Fatalf("TitleAlt = %v, want Alt Patch", trimmed.TitleAlt)
	}
	if trimmed.Visibility != domain.GameVisibilityPublic {
		t.Fatalf("Visibility = %q, want public default", trimmed.Visibility)
	}
	if trimmed.Summary != nil {
		t.Fatalf("Summary = %v, want nil after blank trim", trimmed.Summary)
	}
	if trimmed.Engine == nil || *trimmed.Engine != "RE Engine" {
		t.Fatalf("Engine = %v, want trimmed engine", trimmed.Engine)
	}
	if !trimmed.PlatformIDs.Present || len(trimmed.PlatformIDs.Values) != 2 || trimmed.PlatformIDs.Values[0] != 8 || trimmed.PlatformIDs.Values[1] != 5 {
		t.Fatalf("PlatformIDs = %#v, want deduped [8 5]", trimmed.PlatformIDs)
	}
	if !trimmed.DeveloperIDs.Present || len(trimmed.DeveloperIDs.Values) != 1 || trimmed.DeveloperIDs.Values[0] != 6 {
		t.Fatalf("DeveloperIDs = %#v, want deduped [6]", trimmed.DeveloperIDs)
	}
	if !trimmed.PublisherIDs.Present || len(trimmed.PublisherIDs.Values) != 2 || trimmed.PublisherIDs.Values[0] != 9 || trimmed.PublisherIDs.Values[1] != 4 {
		t.Fatalf("PublisherIDs = %#v, want deduped [9 4]", trimmed.PublisherIDs)
	}
	if !trimmed.TagIDs.Present || len(trimmed.TagIDs.Values) != 0 {
		t.Fatalf("TagIDs = %#v, want empty present slice", trimmed.TagIDs)
	}
}

func TestGamesServiceUpdateAggregateRejectsUnsupportedDeleteAssetType(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-invalid-asset", "Aggregate Invalid Asset", domain.GameVisibilityPublic)
	service := newServicesAggregateService(db, config.Config{})

	_, _, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Invalid Asset"},
		},
		Assets: domain.GameAggregateAssetsInput{
			DeleteAssets: []domain.GameAssetDeleteInput{
				{AssetType: "manual", Path: "/assets/manual.pdf"},
			},
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
	service := newServicesAggregateService(db, config.Config{})

	_, _, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Files"},
		},
		Assets: domain.GameAggregateAssetsInput{
			Files: []domain.GameFileUpsertInput{
				{FilePath: "/tmp/demo.rom"},
			},
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
	service := newServicesAggregateService(db, config.Config{AssetsDir: t.TempDir()})

	game, warnings, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Warning"},
		},
		Assets: domain.GameAggregateAssetsInput{
			DeleteAssets: []domain.GameAssetDeleteInput{
				{AssetType: "cover", Path: "/assets/../bad-cover.png"},
			},
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
	task := mustLoadAssetCleanupTask(t, db, "/assets/../bad-cover.png")
	if task.Source != "games.update_aggregate" {
		t.Fatalf("task.Source = %q, want games.update_aggregate", task.Source)
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
	service := newServicesAggregateService(db, config.Config{PrimaryROMRoot: root})

	label := "  Updated Label  "
	notes := "  Fresh Notes  "
	game, warnings, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Files Success"},
		},
		Assets: domain.GameAggregateAssetsInput{
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
	service := newServicesAggregateService(db, config.Config{PrimaryROMRoot: root})

	_, _, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Outside Root"},
		},
		Assets: domain.GameAggregateAssetsInput{
			Files: []domain.GameFileUpsertInput{
				{FilePath: outsidePath},
			},
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
	service := newServicesAggregateService(db, config.Config{PrimaryROMRoot: root})

	_, warnings, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Delete Files"},
		},
		Assets: domain.GameAggregateAssetsInput{
			Files: []domain.GameFileUpsertInput{},
		},
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
	service := newServicesAggregateService(db, config.Config{PrimaryROMRoot: root})

	missingID := int64(9999)
	_, _, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Missing File ID"},
		},
		Assets: domain.GameAggregateAssetsInput{
			Files: []domain.GameFileUpsertInput{
				{
					ID:       &missingID,
					FilePath: filePath,
				},
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
	service := newServicesAggregateService(db, config.Config{})

	_, _, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Missing Shot"},
		},
		Assets: domain.GameAggregateAssetsInput{
			ScreenshotOrderAssetUIDs: []string{"missing-shot"},
		},
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
	service := newServicesAggregateService(db, config.Config{})

	_, _, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Missing Video"},
		},
		Assets: domain.GameAggregateAssetsInput{
			VideoOrderAssetUIDs: []string{"missing-video"},
		},
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateAggregate error = %v, want ErrNotFound", err)
	}
}

func TestGamesServiceUpdateAggregatePreservesOmittedRelationsAndSeries(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-preserve-relations", "Aggregate Preserve Relations", domain.GameVisibilityPublic)

	seriesResult, err := db.Exec(`INSERT INTO series (name, slug) VALUES (?, ?)`, "Existing Series", "existing-series")
	if err != nil {
		t.Fatalf("insert series: %v", err)
	}
	seriesID, err := seriesResult.LastInsertId()
	if err != nil {
		t.Fatalf("series LastInsertId returned error: %v", err)
	}
	if _, err := db.Exec(`UPDATE games SET series_id = ? WHERE id = ?`, seriesID, gameID); err != nil {
		t.Fatalf("update game series: %v", err)
	}

	platformResult, err := db.Exec(`INSERT INTO platforms (name, slug) VALUES (?, ?)`, "Windows", "windows")
	if err != nil {
		t.Fatalf("insert platform: %v", err)
	}
	platformID, err := platformResult.LastInsertId()
	if err != nil {
		t.Fatalf("platform LastInsertId returned error: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO game_platforms (game_id, platform_id, sort_order) VALUES (?, ?, 0)`, gameID, platformID); err != nil {
		t.Fatalf("link game platform: %v", err)
	}

	developerResult, err := db.Exec(`INSERT INTO developers (name, slug) VALUES (?, ?)`, "Valve", "valve")
	if err != nil {
		t.Fatalf("insert developer: %v", err)
	}
	developerID, err := developerResult.LastInsertId()
	if err != nil {
		t.Fatalf("developer LastInsertId returned error: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO game_developers (game_id, developer_id, sort_order) VALUES (?, ?, 0)`, gameID, developerID); err != nil {
		t.Fatalf("link game developer: %v", err)
	}

	publisherResult, err := db.Exec(`INSERT INTO publishers (name, slug) VALUES (?, ?)`, "Sega", "sega")
	if err != nil {
		t.Fatalf("insert publisher: %v", err)
	}
	publisherID, err := publisherResult.LastInsertId()
	if err != nil {
		t.Fatalf("publisher LastInsertId returned error: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO game_publishers (game_id, publisher_id, sort_order) VALUES (?, ?, 0)`, gameID, publisherID); err != nil {
		t.Fatalf("link game publisher: %v", err)
	}

	tagGroupID := insertServicesTagGroup(t, db, "aggregate-preserve", "Aggregate Preserve")
	tagID := insertServicesTag(t, db, tagGroupID, "RPG", "rpg")
	linkServicesGameTag(t, db, gameID, tagID, 0)

	service := newServicesAggregateService(db, config.Config{})

	_, warnings, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Preserve Relations Updated"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
	repo := repositories.NewGamesRepository(db)
	series, err := repo.GetSeriesMetadata(gameID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata returned error: %v", err)
	}
	if series == nil || series.ID != seriesID {
		t.Fatalf("series = %#v, want existing series preserved", series)
	}

	platforms, err := repo.ListMetadata("platforms", "game_platforms", "platform_id", gameID)
	if err != nil {
		t.Fatalf("ListMetadata(platforms) returned error: %v", err)
	}
	if len(platforms) != 1 || platforms[0].ID != platformID {
		t.Fatalf("platforms = %#v, want existing platform preserved", platforms)
	}

	developers, err := repo.ListMetadata("developers", "game_developers", "developer_id", gameID)
	if err != nil {
		t.Fatalf("ListMetadata(developers) returned error: %v", err)
	}
	if len(developers) != 1 || developers[0].ID != developerID {
		t.Fatalf("developers = %#v, want existing developer preserved", developers)
	}

	publishers, err := repo.ListMetadata("publishers", "game_publishers", "publisher_id", gameID)
	if err != nil {
		t.Fatalf("ListMetadata(publishers) returned error: %v", err)
	}
	if len(publishers) != 1 || publishers[0].ID != publisherID {
		t.Fatalf("publishers = %#v, want existing publisher preserved", publishers)
	}

	tags, err := repositories.NewTagsRepository(db).ListByGameID(gameID)
	if err != nil {
		t.Fatalf("ListByGameID(tags) returned error: %v", err)
	}
	if len(tags) != 1 || tags[0].ID != tagID {
		t.Fatalf("tags = %#v, want existing tag preserved", tags)
	}
}

func TestGamesServiceUpdateAggregateClearsRelationsAndSeriesWhenPresent(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-clear-relations", "Aggregate Clear Relations", domain.GameVisibilityPublic)

	seriesResult, err := db.Exec(`INSERT INTO series (name, slug) VALUES (?, ?)`, "Clear Series", "clear-series")
	if err != nil {
		t.Fatalf("insert series: %v", err)
	}
	seriesID, err := seriesResult.LastInsertId()
	if err != nil {
		t.Fatalf("series LastInsertId returned error: %v", err)
	}
	if _, err := db.Exec(`UPDATE games SET series_id = ? WHERE id = ?`, seriesID, gameID); err != nil {
		t.Fatalf("update game series: %v", err)
	}

	developerResult, err := db.Exec(`INSERT INTO developers (name, slug) VALUES (?, ?)`, "Capcom", "capcom")
	if err != nil {
		t.Fatalf("insert developer: %v", err)
	}
	developerID, err := developerResult.LastInsertId()
	if err != nil {
		t.Fatalf("developer LastInsertId returned error: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO game_developers (game_id, developer_id, sort_order) VALUES (?, ?, 0)`, gameID, developerID); err != nil {
		t.Fatalf("link game developer: %v", err)
	}

	service := newServicesAggregateService(db, config.Config{})

	_, warnings, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Clear Relations Updated"},
			SeriesID:      domain.OptionalInt64Patch{Present: true, Value: nil},
			DeveloperIDs:  domain.Int64SlicePatch{Present: true, Values: []int64{}},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
	repo := repositories.NewGamesRepository(db)
	series, err := repo.GetSeriesMetadata(gameID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata returned error: %v", err)
	}
	if series != nil {
		t.Fatalf("series = %#v, want nil", series)
	}

	developers, err := repo.ListMetadata("developers", "game_developers", "developer_id", gameID)
	if err != nil {
		t.Fatalf("ListMetadata(developers) returned error: %v", err)
	}
	if len(developers) != 0 {
		t.Fatalf("developers = %#v, want cleared developers", developers)
	}
}

func TestGamesServiceUpdateAggregateReordersVideos(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-reorder-video", "Aggregate Reorder Video", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/aggregate-reorder-video/video-a.mp4", 5)
	insertServicesGameAsset(t, db, gameID, "video-b", "video", "/assets/aggregate-reorder-video/video-b.mp4", 6)
	service := newServicesAggregateService(db, config.Config{})

	game, warnings, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Reorder Video"},
		},
		Assets: domain.GameAggregateAssetsInput{
			VideoOrderAssetUIDs: []string{" video-b ", " video-a "},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
	if game == nil {
		t.Fatalf("game = %#v, want updated game", game)
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

func TestGamesServiceUpdateAggregateDeletesFirstVideoAndKeepsNextVideo(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "aggregate-delete-primary-video", "Aggregate Delete Primary Video", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/aggregate-delete-primary-video/video-a.mp4", 0)
	insertServicesGameAsset(t, db, gameID, "video-b", "video", "/assets/aggregate-delete-primary-video/video-b.mp4", 1)
	writeServicesAssetFile(t, assetsDir, "aggregate-delete-primary-video", "video-a.mp4", []byte("a"))
	writeServicesAssetFile(t, assetsDir, "aggregate-delete-primary-video", "video-b.mp4", []byte("b"))
	service := newServicesAggregateService(db, config.Config{AssetsDir: assetsDir})

	game, warnings, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Delete Primary Video"},
		},
		Assets: domain.GameAggregateAssetsInput{
			DeleteAssets: []domain.GameAssetDeleteInput{
				{AssetType: "video", AssetUID: "video-a"},
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
	if game == nil {
		t.Fatalf("game = %#v, want updated game", game)
	}

	videos, err := repositories.NewGamesRepository(db).ListVideos(gameID)
	if err != nil {
		t.Fatalf("ListVideos returned error: %v", err)
	}
	if len(videos) != 1 || videos[0].AssetUID != "video-b" {
		t.Fatalf("videos = %+v, want only fallback video-b", videos)
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "aggregate-delete-primary-video", "video-a.mp4")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected deleted first video file, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "aggregate-delete-primary-video", "video-b.mp4")); err != nil {
		t.Fatalf("expected fallback video file to remain, got err=%v", err)
	}
}
