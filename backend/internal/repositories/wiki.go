package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

type WikiRepository struct {
	db *sqlx.DB
}

func NewWikiRepository(db *sqlx.DB) *WikiRepository {
	return &WikiRepository{db: db}
}

func (r *WikiRepository) Get(gameID int64) (*domain.Game, error) {
	var game domain.Game
	err := r.db.Get(&game, `
		SELECT id, title, wiki_content, updated_at
		FROM games
		WHERE id = ?
	`, gameID)
	if err != nil {
		return nil, err
	}
	return &game, nil
}

func (r *WikiRepository) Update(gameID int64, content string, changeSummary *string, historyLimit int) (*domain.Game, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("begin wiki update: %w", err)
	}

	if _, err := tx.Exec(`
		UPDATE games
		SET wiki_content = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, content, gameID); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("update wiki content: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO wiki_history (game_id, content, change_summary)
		VALUES (?, ?, ?)
	`, gameID, content, changeSummary); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("insert wiki history: %w", err)
	}

	if historyLimit > 0 {
		if _, err := tx.Exec(`
			DELETE FROM wiki_history
			WHERE game_id = ?
			  AND id NOT IN (
			    SELECT id
			    FROM wiki_history
			    WHERE game_id = ?
			    ORDER BY id DESC
			    LIMIT ?
			  )
		`, gameID, gameID, historyLimit); err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("prune wiki history: %w", err)
		}
	}

	var game domain.Game
	if err := tx.Get(&game, `
		SELECT id, title, wiki_content, updated_at
		FROM games
		WHERE id = ?
	`, gameID); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("reload wiki content: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit wiki update: %w", err)
	}

	return &game, nil
}

func (r *WikiRepository) ListHistory(gameID int64) ([]domain.WikiHistoryEntry, error) {
	var items []domain.WikiHistoryEntry
	err := r.db.Select(&items, `
		SELECT id, game_id, content, change_summary, created_at
		FROM wiki_history
		WHERE game_id = ?
		ORDER BY id DESC
	`, gameID)
	if err != nil {
		return nil, fmt.Errorf("list wiki history: %w", err)
	}
	return items, nil
}
