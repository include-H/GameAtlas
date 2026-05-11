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
