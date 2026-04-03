package services

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/hao/game/internal/config"
	dbpkg "github.com/hao/game/internal/db"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
	"github.com/jmoiron/sqlx"
)

func openServicesTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := dbpkg.OpenSQLite(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	if err := dbpkg.RunMigrations(db); err != nil {
		_ = db.Close()
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	return db
}

func insertServicesTestGame(t *testing.T, db *sqlx.DB, publicID string, title string, visibility string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO games (public_id, title, visibility)
		VALUES (?, ?, ?)
	`, publicID, title, visibility)
	if err != nil {
		t.Fatalf("insert test game: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func insertServicesGameAsset(t *testing.T, db *sqlx.DB, gameID int64, assetUID string, assetType string, path string, sortOrder int) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
		VALUES (?, ?, ?, ?, ?)
	`, gameID, assetUID, assetType, path, sortOrder)
	if err != nil {
		t.Fatalf("insert test asset: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func insertServicesTagGroup(t *testing.T, db *sqlx.DB, key string, name string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO tag_groups (key, name)
		VALUES (?, ?)
	`, key, name)
	if err != nil {
		t.Fatalf("insert test tag group: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func insertServicesTagGroupWithOptions(t *testing.T, db *sqlx.DB, key string, name string, allowMultiple bool, isFilterable bool) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO tag_groups (key, name, allow_multiple, is_filterable)
		VALUES (?, ?, ?, ?)
	`, key, name, allowMultiple, isFilterable)
	if err != nil {
		t.Fatalf("insert test tag group with options: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func insertServicesTag(t *testing.T, db *sqlx.DB, groupID int64, name string, slug string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO tags (group_id, name, slug)
		VALUES (?, ?, ?)
	`, groupID, name, slug)
	if err != nil {
		t.Fatalf("insert test tag: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func linkServicesGameTag(t *testing.T, db *sqlx.DB, gameID int64, tagID int64, sortOrder int) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO game_tags (game_id, tag_id, sort_order)
		VALUES (?, ?, ?)
	`, gameID, tagID, sortOrder); err != nil {
		t.Fatalf("link test game tag: %v", err)
	}
}

func insertServicesGameFile(t *testing.T, db *sqlx.DB, gameID int64, path string, sortOrder int) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO game_files (game_id, file_path, sort_order)
		VALUES (?, ?, ?)
	`, gameID, path, sortOrder)
	if err != nil {
		t.Fatalf("insert test game file: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func newServicesAssetsService(db *sqlx.DB, assetsDir string) *AssetsService {
	return NewAssetsService(
		config.Config{AssetsDir: assetsDir},
		repositories.NewGamesRepository(db),
		repositories.NewAssetsRepository(db),
	)
}

func newServicesCatalogService(db *sqlx.DB) *GameCatalogService {
	gamesRepo := repositories.NewGamesRepository(db)
	return NewGameCatalogService(
		repositories.NewGameCatalogRepository(gamesRepo),
		repositories.NewReviewIssueOverrideRepository(db),
	)
}

func newServicesDetailService(db *sqlx.DB) *GameDetailService {
	gamesRepo := repositories.NewGamesRepository(db)
	return NewGameDetailService(
		repositories.NewGameDetailRepository(gamesRepo),
		repositories.NewGameFilesRepository(db),
		repositories.NewTagsRepository(db),
		repositories.NewReviewIssueOverrideRepository(db),
	)
}

func newServicesAggregateService(db *sqlx.DB, cfg config.Config) *GameAggregateService {
	gamesRepo := repositories.NewGamesRepository(db)
	return NewGameAggregateService(
		cfg,
		repositories.NewGameAggregateRepository(gamesRepo),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
}

func writeServicesAssetFile(t *testing.T, assetsDir string, gamePublicID string, filename string, content []byte) string {
	t.Helper()

	targetDir := filepath.Join(assetsDir, gamePublicID)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	targetPath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(targetPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	return targetPath
}

func mustLoadServicesGame(t *testing.T, db *sqlx.DB, gameID int64) domain.Game {
	t.Helper()

	var game domain.Game
	if err := db.Get(&game, `
		SELECT
			id,
			public_id,
			title,
			title_alt,
			visibility,
			summary,
			release_date,
			engine,
			cover_image,
			banner_image,
			wiki_content,
			downloads,
			NULL AS primary_screenshot,
			0 AS screenshot_count,
			0 AS file_count,
			0 AS developer_count,
			0 AS publisher_count,
			0 AS platform_count,
			0 AS is_favorite,
			created_at,
			updated_at
		FROM games
		WHERE id = ?
	`, gameID); err != nil {
		t.Fatalf("load test game: %v", err)
	}

	return game
}

func mustLoadAssetCleanupTask(t *testing.T, db *sqlx.DB, assetPath string) domain.AssetCleanupTask {
	t.Helper()

	var task domain.AssetCleanupTask
	if err := db.Get(&task, `
		SELECT id, asset_path, source, last_error, attempt_count, created_at, updated_at
		FROM asset_cleanup_tasks
		WHERE asset_path = ?
	`, assetPath); err != nil {
		t.Fatalf("load asset cleanup task: %v", err)
	}

	return task
}

func assertAssetCleanupTaskMissing(t *testing.T, db *sqlx.DB, assetPath string) {
	t.Helper()

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM asset_cleanup_tasks WHERE asset_path = ?`, assetPath); err != nil {
		t.Fatalf("count asset cleanup tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("asset cleanup task count for %q = %d, want 0", assetPath, count)
	}
}

func assertNoAssetCleanupTasks(t *testing.T, db *sqlx.DB) {
	t.Helper()

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM asset_cleanup_tasks`); err != nil && err != sql.ErrNoRows {
		t.Fatalf("count asset cleanup tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("asset cleanup task total = %d, want 0", count)
	}
}
