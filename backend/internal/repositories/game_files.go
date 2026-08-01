package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

type GameFilesRepository struct {
	db *sqlx.DB
}

func NewGameFilesRepository(db *sqlx.DB) *GameFilesRepository {
	return &GameFilesRepository{db: db}
}

func (r *GameFilesRepository) ListByGameID(gameID int64) ([]domain.GameFile, error) {
	var files []domain.GameFile
	err := r.db.Select(&files, `
		SELECT id, game_id, file_path, label, notes, size_bytes, sort_order, created_at, updated_at, source_created_at
		FROM game_files
		WHERE game_id = ?
		ORDER BY sort_order ASC, id ASC
	`, gameID)
	if err != nil {
		return nil, fmt.Errorf("list game files: %w", err)
	}
	return files, nil
}

func (r *GameFilesRepository) GetByID(gameID, fileID int64) (*domain.GameFile, error) {
	var file domain.GameFile
	err := r.db.Get(&file, `
		SELECT id, game_id, file_path, label, notes, size_bytes, sort_order, created_at, updated_at, source_created_at
		FROM game_files
		WHERE game_id = ? AND id = ?
	`, gameID, fileID)
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *GameFilesRepository) ExistsByPath(filePath string) (bool, error) {
	var count int
	err := r.db.Get(&count, `
		SELECT COUNT(*) FROM game_files WHERE file_path = ?
	`, filePath)
	if err != nil {
		return false, fmt.Errorf("check file path exists: %w", err)
	}
	return count > 0, nil
}

func (r *GameFilesRepository) ListAllPaths() (map[string]bool, error) {
	var paths []string
	err := r.db.Select(&paths, `SELECT file_path FROM game_files`)
	if err != nil {
		return nil, fmt.Errorf("list all file paths: %w", err)
	}
	result := make(map[string]bool, len(paths))
	for _, p := range paths {
		result[p] = true
	}
	return result, nil
}
