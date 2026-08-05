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
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetDetail private error = %v, want domain.ErrNotFound", err)
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

	_, err := service.List(domain.GamesListParams{
		Page:  1,
		Limit: 20,
		Sort:  "updated_at",
		Order: "desc",
	})
	if err == nil {
		t.Fatal("List returned nil error, want override lookup failure")
	}
	if !strings.Contains(err.Error(), "list review overrides") {
		t.Fatalf("List error = %v, want review override lookup context", err)
	}
}

func TestNormalizeListParamsKeepsDomainOnlyNormalization(t *testing.T) {
	params := domain.GamesListParams{
		Visibility:        "  ",
		PendingIssue:      "missing-assets",
		PendingRecentDays: 999,
	}

	if err := normalizeListParams(&params); err != nil {
		t.Fatalf("normalizeListParams returned error: %v", err)
	}
	if params.Visibility != domain.GameVisibilityPublic {
		t.Fatalf("Visibility = %q, want public default", params.Visibility)
	}
	if params.PendingIssue != "missing-assets" {
		t.Fatalf("PendingIssue = %q, want service to leave transport-decoded value untouched", params.PendingIssue)
	}
	if params.PendingRecentDays != 365 {
		t.Fatalf("PendingRecentDays = %d, want domain clamp 365", params.PendingRecentDays)
	}
}

func TestGamesServiceListRejectsUndecodedSortContract(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	service := newServicesCatalogService(db)

	testCases := []struct {
		name   string
		params domain.GamesListParams
	}{
		{
			name: "missing sort and order",
			params: domain.GamesListParams{
				Page:  1,
				Limit: 20,
			},
		},
		{
			name: "invalid sort",
			params: domain.GamesListParams{
				Page:  1,
				Limit: 20,
				Sort:  "legacy_default",
				Order: "desc",
			},
		},
		{
			name: "invalid order",
			params: domain.GamesListParams{
				Page:  1,
				Limit: 20,
				Sort:  "updated_at",
				Order: "sideways",
			},
		},
		{
			name: "random sort without seed",
			params: domain.GamesListParams{
				Page:  1,
				Limit: 20,
				Sort:  "random",
				Order: "desc",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.List(tc.params)
			if !errors.Is(err, domain.ErrValidation) {
				t.Fatalf("List error = %v, want domain.ErrValidation", err)
			}
		})
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

func TestGamesServiceDeletePrunesSeriesAndInvalidatesMetadataListCache(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "delete-series-cache", "Delete Series Cache", domain.GameVisibilityPublic)
	if _, err := db.Exec(`
		INSERT INTO series (name, slug, sort_order)
		VALUES ('Linked Series', 'linked-series', 0), ('Loose Series', 'loose-series', 1)
	`); err != nil {
		t.Fatalf("insert series: %v", err)
	}
	var linkedSeriesID int64
	if err := db.Get(&linkedSeriesID, `SELECT id FROM series WHERE slug = 'linked-series'`); err != nil {
		t.Fatalf("load linked series id: %v", err)
	}
	if _, err := db.Exec(`UPDATE games SET series_id = ? WHERE id = ?`, linkedSeriesID, gameID); err != nil {
		t.Fatalf("attach linked series: %v", err)
	}

	gamesRepo := repositories.NewGamesRepository(db)
	metadataService := NewMetadataService(repositories.NewMetadataRepository(db))
	catalogService := NewGameCatalogService(
		repositories.NewGameCatalogRepository(gamesRepo, repositories.NewFavoriteGamesRepository(db)),
		repositories.NewReviewIssueOverrideRepository(db),
	)
	service := NewGameAggregateService(
		config.Config{},
		gamesRepo,
		metadataService,
		catalogService,
		repositories.NewAssetCleanupTasksRepository(db),
	)

	cached, err := metadataService.List(MetadataResource{Type: domain.MetadataSeries}, true, MetadataListOptions{Sort: "name"})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(cached) != 2 {
		t.Fatalf("cached series = %+v, want linked and loose rows before delete", cached)
	}

	if _, err := service.Delete(gameID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	refreshed, err := metadataService.List(MetadataResource{Type: domain.MetadataSeries}, true, MetadataListOptions{Sort: "name"})
	if err != nil {
		t.Fatalf("List after delete returned error: %v", err)
	}
	if len(refreshed) != 0 {
		t.Fatalf("refreshed series = %+v, want no orphan series after game delete", refreshed)
	}
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

func TestNormalizeTimelineParamsClampsLimit(t *testing.T) {
	params := domain.GamesTimelineParams{
		Limit: 120,
	}

	if err := normalizeTimelineParams(&params); err != nil {
		t.Fatalf("normalizeTimelineParams returned error: %v", err)
	}
	if params.Limit != 100 {
		t.Fatalf("Limit = %d, want 100", params.Limit)
	}
	if params.Visibility != domain.GameVisibilityPublic {
		t.Fatalf("Visibility = %q, want public", params.Visibility)
	}
}

func TestValidateAndTrimGameCreateInputNormalizesQuickCreateFields(t *testing.T) {
	trimmed, err := validateAndTrimGameCreateInput(domain.GameCreateInput{
		Title:      "  Shared Core Game  ",
		Visibility: "  ",
	})
	if err != nil {
		t.Fatalf("validateAndTrimGameCreateInput returned error: %v", err)
	}

	if trimmed.Title != "Shared Core Game" {
		t.Fatalf("Title = %q, want trimmed title", trimmed.Title)
	}
	if trimmed.Visibility != domain.GameVisibilityPublic {
		t.Fatalf("Visibility = %q, want public default", trimmed.Visibility)
	}
}

func TestValidateAndTrimGameAggregateCoreUpdateInputNormalizesSharedCoreFields(t *testing.T) {
	titleAlt := " Alt Patch "
	summary := "   "

	trimmed, err := validateAndTrimGameAggregateCoreUpdateInput(domain.GameAggregateCoreUpdateInput{
		GameCoreInput: domain.GameCoreInput{
			Title:      "  Aggregate Shared Core  ",
			TitleAlt:   &titleAlt,
			Visibility: " public ",
			Summary:    &summary,
		},
		DeveloperIDs: []int64{6, 6},
		PublisherIDs: []int64{9, 4, 9},
	})
	if err != nil {
		t.Fatalf("validateAndTrimGameAggregateCoreUpdateInput returned error: %v", err)
	}

	if trimmed.Title != "Aggregate Shared Core" {
		t.Fatalf("Title = %q, want trimmed title", trimmed.Title)
	}
	if trimmed.TitleAlt == nil || *trimmed.TitleAlt != "Alt Patch" {
		t.Fatalf("TitleAlt = %v, want Alt Patch", trimmed.TitleAlt)
	}
	if trimmed.Visibility != domain.GameVisibilityPublic {
		t.Fatalf("Visibility = %q, want public", trimmed.Visibility)
	}
	if trimmed.Summary != nil {
		t.Fatalf("Summary = %v, want nil after blank trim", trimmed.Summary)
	}
	if len(trimmed.DeveloperIDs) != 1 || trimmed.DeveloperIDs[0] != 6 {
		t.Fatalf("DeveloperIDs = %#v, want deduped [6]", trimmed.DeveloperIDs)
	}
	if len(trimmed.PublisherIDs) != 2 || trimmed.PublisherIDs[0] != 9 || trimmed.PublisherIDs[1] != 4 {
		t.Fatalf("PublisherIDs = %#v, want deduped [9 4]", trimmed.PublisherIDs)
	}
}

func TestGamesServiceUpdateAggregateReturnsMissingConfigForFileValidation(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-files", "Aggregate Files", domain.GameVisibilityPublic)
	service := newServicesAggregateService(db, config.Config{})

	_, _, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Files", Visibility: domain.GameVisibilityPublic},
		},
		Assets: domain.GameAggregateAssetsInput{
			Files: []domain.GameFileUpsertInput{
				{FilePath: "/tmp/demo.rom"},
			},
		},
	})
	if !errors.Is(err, domain.ErrMissingConfig) {
		t.Fatalf("UpdateAggregate error = %v, want domain.ErrMissingConfig", err)
	}
}

func TestGamesServiceUpdateAggregateReturnsDeleteWarningsWhenAssetRemovalFails(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-warning", "Aggregate Warning", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "bad-cover", "cover", "/assets/../bad-cover.png", 0)
	service := newServicesAggregateService(db, config.Config{AssetsDir: t.TempDir()})

	game, warnings, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Warning", Visibility: domain.GameVisibilityPublic},
		},
		Assets: domain.GameAggregateAssetsInput{
			// Auto-diff: bad-cover is not in CoverOrderAssetUIDs, so it gets auto-deleted.
			CoverOrderAssetUIDs: []string{},
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
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Files Success", Visibility: domain.GameVisibilityPublic},
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
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Outside Root", Visibility: domain.GameVisibilityPublic},
		},
		Assets: domain.GameAggregateAssetsInput{
			Files: []domain.GameFileUpsertInput{
				{FilePath: outsidePath},
			},
		},
	})
	if !errors.Is(err, domain.ErrForbiddenPath) {
		t.Fatalf("UpdateAggregate error = %v, want domain.ErrForbiddenPath", err)
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
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Delete Files", Visibility: domain.GameVisibilityPublic},
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
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Missing File ID", Visibility: domain.GameVisibilityPublic},
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
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("UpdateAggregate error = %v, want domain.ErrNotFound", err)
	}
}

func TestGamesServiceUpdateAggregateReturnsNotFoundForMissingScreenshotReorderUID(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-missing-shot", "Aggregate Missing Shot", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "shot-a", "screenshot", "/assets/aggregate-missing-shot/shot-a.png", 0)
	service := newServicesAggregateService(db, config.Config{})

	_, _, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Missing Shot", Visibility: domain.GameVisibilityPublic},
		},
		Assets: domain.GameAggregateAssetsInput{
			ScreenshotOrderAssetUIDs: []string{"missing-shot"},
		},
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("UpdateAggregate error = %v, want domain.ErrNotFound", err)
	}
}

func TestGamesServiceUpdateAggregateReturnsNotFoundForMissingVideoReorderUID(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "aggregate-missing-video", "Aggregate Missing Video", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/aggregate-missing-video/video-a.mp4", 0)
	service := newServicesAggregateService(db, config.Config{})

	_, _, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Missing Video", Visibility: domain.GameVisibilityPublic},
		},
		Assets: domain.GameAggregateAssetsInput{
			VideoOrderAssetUIDs: []string{"missing-video"},
		},
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("UpdateAggregate error = %v, want domain.ErrNotFound", err)
	}
}

func TestGamesServiceUpdateAggregateReplacesRelationsAndSeries(t *testing.T) {
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

	_ = seriesID
	_ = developerID
	_ = publisherID

	service := newServicesAggregateService(db, config.Config{})

	_, warnings, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Preserve Relations Updated", Visibility: domain.GameVisibilityPublic},
			SeriesID:      nil,
			DeveloperIDs:  []int64{},
			PublisherIDs:  []int64{},
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
		t.Fatalf("series = %#v, want cleared series", series)
	}

	developers, err := repo.ListMetadata(domain.MetadataDevelopers, gameID)
	if err != nil {
		t.Fatalf("ListMetadata(developers) returned error: %v", err)
	}
	if len(developers) != 0 {
		t.Fatalf("developers = %#v, want cleared developers", developers)
	}

	publishers, err := repo.ListMetadata(domain.MetadataPublishers, gameID)
	if err != nil {
		t.Fatalf("ListMetadata(publishers) returned error: %v", err)
	}
	if len(publishers) != 0 {
		t.Fatalf("publishers = %#v, want cleared publishers", publishers)
	}
}

func TestGamesServiceUpdateAggregateClearsRelationsAndSeries(t *testing.T) {
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
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Clear Relations Updated", Visibility: domain.GameVisibilityPublic},
			SeriesID:      nil,
			DeveloperIDs:  []int64{},
			PublisherIDs:  []int64{},
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

	developers, err := repo.ListMetadata(domain.MetadataDevelopers, gameID)
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
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Reorder Video", Visibility: domain.GameVisibilityPublic},
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
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Aggregate Delete Primary Video", Visibility: domain.GameVisibilityPublic},
		},
		Assets: domain.GameAggregateAssetsInput{
			VideoOrderAssetUIDs: []string{"video-b"},
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

func TestGamesServiceUpdateAggregateKeepsSharedAssetFileWhenOtherGameReferencesIt(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	sharedPath := "/assets/shared/cover.png"
	gameA := insertServicesTestGame(t, db, "aggregate-shared-a", "Shared A", domain.GameVisibilityPublic)
	gameB := insertServicesTestGame(t, db, "aggregate-shared-b", "Shared B", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameA, "shared-a", "cover", sharedPath, 0)
	insertServicesGameAsset(t, db, gameB, "shared-b", "cover", sharedPath, 0)
	writeServicesAssetFile(t, assetsDir, "shared", "cover.png", []byte("cover"))

	service := newServicesAggregateService(db, config.Config{AssetsDir: assetsDir})
	_, _, err := service.Update(gameA, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Shared A Updated", Visibility: domain.GameVisibilityPublic},
		},
		Assets: domain.GameAggregateAssetsInput{
			CoverOrderAssetUIDs: []string{},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(assetsDir, "shared", "cover.png")); err != nil {
		t.Fatalf("shared asset file should remain while game B references it, got err=%v", err)
	}

	referenced, err := repositories.NewGamesRepository(db).IsAssetPathReferenced(sharedPath)
	if err != nil {
		t.Fatalf("IsAssetPathReferenced returned error: %v", err)
	}
	if !referenced {
		t.Fatal("shared path should still be referenced by game B")
	}
}

func TestGamesServiceUpdateAggregateKeepsAssetFileReferencedByStartScreenTile(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	assetPath := "/assets/tile-game/cover.png"
	gameID := insertServicesTestGame(t, db, "aggregate-tile-game", "Tile Game", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "tile-cover", "cover", assetPath, 0)
	writeServicesAssetFile(t, assetsDir, "tile-game", "cover.png", []byte("cover"))
	if _, err := db.Exec(`
		INSERT INTO start_screen_tiles (game_id, tile_size, image_small_path)
		VALUES (?, 'small', ?)
	`, gameID, assetPath); err != nil {
		t.Fatalf("insert start screen tile: %v", err)
	}

	service := newServicesAggregateService(db, config.Config{AssetsDir: assetsDir})
	_, _, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Tile Game Updated", Visibility: domain.GameVisibilityPublic},
		},
		Assets: domain.GameAggregateAssetsInput{
			CoverOrderAssetUIDs: []string{},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(assetsDir, "tile-game", "cover.png")); err != nil {
		t.Fatalf("asset file should remain while start-screen tile references it, got err=%v", err)
	}
}

func TestGamesServiceUpdateAggregatePersistsVideoPosterPath(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "aggregate-video-poster", "Video Poster", domain.GameVisibilityPublic)
	writeServicesAssetFile(t, assetsDir, "aggregate-video-poster", "video.mp4", []byte("video"))
	posterPath := "/assets/aggregate-video-poster/poster.jpg"
	writeServicesAssetFile(t, assetsDir, "aggregate-video-poster", "poster.jpg", []byte("poster"))

	service := newServicesAggregateService(db, config.Config{AssetsDir: assetsDir})
	_, _, err := service.Update(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Video Poster", Visibility: domain.GameVisibilityPublic},
		},
		Assets: domain.GameAggregateAssetsInput{
			VideoOrderAssetUIDs: []string{"video-1"},
			NewAssets: []domain.NewAssetEntry{
				{
					AssetUID:   "video-1",
					AssetType:  "video",
					Path:       "/assets/aggregate-video-poster/video.mp4",
					PosterPath: &posterPath,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}

	videos, err := repositories.NewGamesRepository(db).ListVideos(gameID)
	if err != nil {
		t.Fatalf("ListVideos returned error: %v", err)
	}
	if len(videos) != 1 {
		t.Fatalf("len(videos) = %d, want 1", len(videos))
	}
	if videos[0].PosterPath == nil || *videos[0].PosterPath != posterPath {
		t.Fatalf("PosterPath = %v, want %q", videos[0].PosterPath, posterPath)
	}
}
