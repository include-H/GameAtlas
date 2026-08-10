package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

type StartScreenColumnsRepository struct {
	db *sqlx.DB
}

func NewStartScreenColumnsRepository(db *sqlx.DB) *StartScreenColumnsRepository {
	return &StartScreenColumnsRepository{db: db}
}

func (r *StartScreenColumnsRepository) List() ([]domain.StartScreenColumn, error) {
	var columns []domain.StartScreenColumn
	err := r.db.Select(&columns, `
		SELECT id, name, sort_order, created_at, updated_at
		FROM start_screen_columns
		ORDER BY sort_order ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list start screen columns: %w", err)
	}
	if columns == nil {
		return []domain.StartScreenColumn{}, nil
	}
	return columns, nil
}

// Replace 是全量替换：sort_order 按传入顺序写入。
func (r *StartScreenColumnsRepository) Replace(columns []domain.StartScreenColumnWrite) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("begin start screen columns replace: %w", err)
	}
	defer tx.Rollback()

	if err := replaceStartScreenColumns(tx, columns); err != nil {
		return fmt.Errorf("replace start screen columns: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit start screen columns replace: %w", err)
	}
	return nil
}
