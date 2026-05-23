package repositories

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type AssetCleanupTask struct {
	ID           int64   `db:"id"`
	AssetPath    string  `db:"asset_path"`
	Source       string  `db:"source"`
	LastError    *string `db:"last_error"`
	AttemptCount int64   `db:"attempt_count"`
	CreatedAt    string  `db:"created_at"`
	UpdatedAt    string  `db:"updated_at"`
}

type AssetCleanupTasksRepository struct {
	db *sqlx.DB
}

func NewAssetCleanupTasksRepository(db *sqlx.DB) *AssetCleanupTasksRepository {
	return &AssetCleanupTasksRepository{db: db}
}

func (r *AssetCleanupTasksRepository) Enqueue(assetPath string, source string, lastError string) error {
	trimmedPath := strings.TrimSpace(assetPath)
	if trimmedPath == "" {
		return nil
	}

	if _, err := r.db.Exec(`
		INSERT INTO asset_cleanup_tasks (asset_path, source, last_error, attempt_count)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(asset_path) DO UPDATE SET
			source = excluded.source,
			last_error = excluded.last_error,
			attempt_count = asset_cleanup_tasks.attempt_count + 1,
			updated_at = CURRENT_TIMESTAMP
	`, trimmedPath, strings.TrimSpace(source), strings.TrimSpace(lastError)); err != nil {
		return fmt.Errorf("enqueue asset cleanup task: %w", err)
	}

	return nil
}

func (r *AssetCleanupTasksRepository) DeleteByPath(assetPath string) error {
	trimmedPath := strings.TrimSpace(assetPath)
	if trimmedPath == "" {
		return nil
	}

	if _, err := r.db.Exec(`DELETE FROM asset_cleanup_tasks WHERE asset_path = ?`, trimmedPath); err != nil {
		return fmt.Errorf("delete asset cleanup task: %w", err)
	}

	return nil
}

func (r *AssetCleanupTasksRepository) ListPending(limit int) ([]AssetCleanupTask, error) {
	query := `
		SELECT id, asset_path, source, last_error, attempt_count, created_at, updated_at
		FROM asset_cleanup_tasks
		ORDER BY updated_at ASC, id ASC
	`
	args := make([]any, 0, 1)
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	tasks := make([]AssetCleanupTask, 0)
	if err := r.db.Select(&tasks, query, args...); err != nil {
		return nil, fmt.Errorf("list asset cleanup tasks: %w", err)
	}

	return tasks, nil
}
