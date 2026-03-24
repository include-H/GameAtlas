package db

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"

	embeddedmigrations "github.com/hao/game/migrations"
)

type migration struct {
	Name        string
	LegacyNames []string
	Skip        func(*sqlx.DB) (bool, error)
}

var registeredMigrations = []migration{
	{Name: "000001_initial_schema.sql"},
	{
		Name:        "000002_game_review_issue_overrides.sql",
		LegacyNames: []string{"000002_review_issue_overrides.sql"},
		Skip:        skipIfReviewIssueOverridesTableExists,
	},
	{Name: "000003_game_assets_unique_paths.sql", Skip: skipIfGameAssetPathIndexExists},
	{Name: "000004_game_assets_uid.sql", Skip: skipIfGameAssetsUIDColumnExists},
	{Name: "000005_games_preview_video_asset_uid.sql", Skip: skipIfGamesPreviewVideoAssetUIDColumnExists},
	{
		Name:        "000006_legacy_series_table_rebuild.sql",
		LegacyNames: []string{"000006_drop_unused_series_columns.sql"},
		Skip:        skipIfSeriesSchemaIsCurrent,
	},
	{
		Name:        "000007_legacy_games_visibility.sql",
		LegacyNames: []string{"000007_games_visibility.sql"},
		Skip:        skipIfGamesVisibilityColumnExists,
	},
	{Name: "000008_tag_system.sql", Skip: skipIfTagSystemSchemaIsCurrent},
	{Name: "000009_games_release_date_index.sql", Skip: skipIfGamesReleaseDateIndexExists},
	{Name: "000010_auth_login_attempts.sql", Skip: skipIfAuthLoginAttemptsTableExists},
	{Name: "000011_game_files_source_created_at.sql", Skip: skipIfGameFilesSourceCreatedAtColumnExists},
	{Name: "000012_drop_games_views.sql", Skip: skipIfGamesViewsColumnMissing},
}

func RunMigrations(db *sqlx.DB) error {
	if err := ensureMigrationTable(db); err != nil {
		return err
	}

	contentsByName, err := loadMigrationContents()
	if err != nil {
		return err
	}

	for _, item := range registeredMigrations {
		names := append([]string{item.Name}, item.LegacyNames...)
		applied, err := hasMigration(db, names...)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if item.Skip != nil {
			skip, err := item.Skip(db)
			if err != nil {
				return fmt.Errorf("evaluate migration %s: %w", item.Name, err)
			}
			if skip {
				continue
			}
		}

		content, ok := contentsByName[item.Name]
		if !ok {
			return fmt.Errorf("migration %s is registered but not embedded", item.Name)
		}

		if err := applyMigration(db, item.Name, content); err != nil {
			return err
		}
	}

	return nil
}

func loadMigrationContents() (map[string]string, error) {
	files, err := embeddedmigrations.Files.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	registered := make(map[string]struct{}, len(registeredMigrations))
	for _, item := range registeredMigrations {
		registered[item.Name] = struct{}{}
	}

	names := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		names = append(names, file.Name())
	}
	sort.Strings(names)

	contents := make(map[string]string, len(names))
	for _, name := range names {
		if _, ok := registered[name]; !ok {
			return nil, fmt.Errorf("embedded migration %s is not registered", name)
		}

		content, err := embeddedmigrations.Files.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}
		contents[name] = string(content)
	}

	return contents, nil
}

func applyMigration(db *sqlx.DB, name string, content string) error {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}

	for _, stmt := range splitMigrationStatements(content) {
		if _, err := tx.Exec(stmt); err != nil {
			if isIgnorableMigrationError(err) {
				continue
			}
			_ = tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", name, err)
		}
	}

	if _, err := tx.Exec("INSERT INTO schema_migrations (name) VALUES (?)", name); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration %s: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}

	return nil
}

func skipIfSeriesSchemaIsCurrent(db *sqlx.DB) (bool, error) {
	columns, err := tableColumnNames(db, "series")
	if err != nil {
		return false, err
	}
	expected := []string{"id", "name", "slug", "sort_order", "created_at"}
	if len(columns) != len(expected) {
		return false, nil
	}
	for idx, column := range expected {
		if columns[idx] != column {
			return false, nil
		}
	}
	return true, nil
}

func skipIfReviewIssueOverridesTableExists(db *sqlx.DB) (bool, error) {
	return hasTable(db, "game_review_issue_overrides")
}

func skipIfGameAssetPathIndexExists(db *sqlx.DB) (bool, error) {
	return hasIndex(db, "idx_game_assets_game_type_path_unique")
}

func skipIfGameAssetsUIDColumnExists(db *sqlx.DB) (bool, error) {
	return hasTableColumn(db, "game_assets", "asset_uid")
}

func skipIfGamesPreviewVideoAssetUIDColumnExists(db *sqlx.DB) (bool, error) {
	return hasTableColumn(db, "games", "preview_video_asset_uid")
}

func skipIfGamesVisibilityColumnExists(db *sqlx.DB) (bool, error) {
	return hasTableColumn(db, "games", "visibility")
}

func skipIfTagSystemSchemaIsCurrent(db *sqlx.DB) (bool, error) {
	requiredTables := []string{"tag_groups", "tags", "game_tags"}
	for _, table := range requiredTables {
		exists, err := hasTable(db, table)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}

	requiredGroups := []string{"genre", "subgenre", "perspective", "theme"}
	for _, key := range requiredGroups {
		exists, err := hasTagGroupKey(db, key)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}

	return true, nil
}

func skipIfGamesReleaseDateIndexExists(db *sqlx.DB) (bool, error) {
	return hasIndex(db, "idx_games_release_date_id")
}

func skipIfAuthLoginAttemptsTableExists(db *sqlx.DB) (bool, error) {
	return hasTable(db, "auth_login_attempts")
}

func skipIfGameFilesSourceCreatedAtColumnExists(db *sqlx.DB) (bool, error) {
	return hasTableColumn(db, "game_files", "source_created_at")
}

func skipIfGamesViewsColumnMissing(db *sqlx.DB) (bool, error) {
	exists, err := hasTableColumn(db, "games", "views")
	if err != nil {
		return false, err
	}
	return !exists, nil
}

func hasTable(db *sqlx.DB, table string) (bool, error) {
	var exists int
	if err := db.Get(&exists, "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1", table); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check table %s: %w", table, err)
	}
	return true, nil
}

func hasIndex(db *sqlx.DB, index string) (bool, error) {
	var exists int
	if err := db.Get(&exists, "SELECT 1 FROM sqlite_master WHERE type = 'index' AND name = ? LIMIT 1", index); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check index %s: %w", index, err)
	}
	return true, nil
}

func hasTagGroupKey(db *sqlx.DB, key string) (bool, error) {
	var exists int
	if err := db.Get(&exists, "SELECT 1 FROM tag_groups WHERE key = ? LIMIT 1", key); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check tag group %s: %w", key, err)
	}
	return true, nil
}

func tableColumnNames(db *sqlx.DB, table string) ([]string, error) {
	query := fmt.Sprintf("SELECT name FROM pragma_table_info('%s') ORDER BY cid", table)

	var columns []string
	if err := db.Select(&columns, query); err != nil {
		return nil, fmt.Errorf("query table info for %s: %w", table, err)
	}

	return columns, nil
}

func hasTableColumn(db *sqlx.DB, table string, column string) (bool, error) {
	columns, err := tableColumnNames(db, table)
	if err != nil {
		return false, err
	}

	for _, current := range columns {
		if current == column {
			return true, nil
		}
	}

	return false, nil
}

func splitMigrationStatements(content string) []string {
	parts := strings.Split(content, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		statements = append(statements, stmt)
	}
	return statements
}

func isIgnorableMigrationError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate column name:") ||
		(strings.Contains(message, "already exists") && strings.Contains(message, "index"))
}

func ensureMigrationTable(db *sqlx.DB) error {
	const query = `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	return nil
}

func hasMigration(db *sqlx.DB, names ...string) (bool, error) {
	const query = `SELECT 1 FROM schema_migrations WHERE name = ? LIMIT 1`

	for _, name := range names {
		var exists int
		err := db.Get(&exists, query, name)
		if err == nil {
			return true, nil
		}
		if err == sql.ErrNoRows {
			continue
		}
		return false, fmt.Errorf("check migration %s: %w", name, err)
	}

	return false, nil
}
