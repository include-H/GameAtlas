package repositories

import (
	"database/sql"
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
		SELECT id, game_id, file_path, label, notes, size_bytes, sort_order, created_at, updated_at
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
		SELECT id, game_id, file_path, label, notes, size_bytes, sort_order, created_at, updated_at
		FROM game_files
		WHERE game_id = ? AND id = ?
	`, gameID, fileID)
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *GameFilesRepository) Create(gameID int64, input domain.GameFileWriteInput) (*domain.GameFile, error) {
	var file domain.GameFile
	err := r.db.Get(&file, `
		INSERT INTO game_files (game_id, file_path, label, notes, sort_order)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, game_id, file_path, label, notes, size_bytes, sort_order, created_at, updated_at
	`, gameID, input.FilePath, input.Label, input.Notes, input.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("create game file: %w", err)
	}
	return &file, nil
}

func (r *GameFilesRepository) Update(gameID, fileID int64, input domain.GameFileWriteInput) (*domain.GameFile, error) {
	result, err := r.db.Exec(`
		UPDATE game_files
		SET file_path = ?, label = ?, notes = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = ? AND id = ?
	`, input.FilePath, input.Label, input.Notes, input.SortOrder, gameID, fileID)
	if err != nil {
		return nil, fmt.Errorf("update game file: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read updated game file rows: %w", err)
	}
	if rows == 0 {
		return nil, sql.ErrNoRows
	}

	return r.GetByID(gameID, fileID)
}

func (r *GameFilesRepository) Delete(gameID, fileID int64) (bool, error) {
	result, err := r.db.Exec("DELETE FROM game_files WHERE game_id = ? AND id = ?", gameID, fileID)
	if err != nil {
		return false, fmt.Errorf("delete game file: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read deleted game file rows: %w", err)
	}
	return rows > 0, nil
}

func (r *GameFilesRepository) UpdateSizeBytes(gameID, fileID, sizeBytes int64) error {
	_, err := r.db.Exec(`
		UPDATE game_files
		SET size_bytes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = ? AND id = ?
	`, sizeBytes, gameID, fileID)
	if err != nil {
		return fmt.Errorf("update game file size: %w", err)
	}
	return nil
}
