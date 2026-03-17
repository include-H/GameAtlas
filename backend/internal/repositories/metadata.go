package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

type MetadataRepository struct {
	db *sqlx.DB
}

func NewMetadataRepository(db *sqlx.DB) *MetadataRepository {
	return &MetadataRepository{db: db}
}

func (r *MetadataRepository) List(table string) ([]domain.MetadataItem, error) {
	query := fmt.Sprintf(`
		SELECT id, name, slug, sort_order, created_at
		FROM %s
		ORDER BY sort_order ASC, id ASC
	`, table)

	var items []domain.MetadataItem
	if err := r.db.Select(&items, query); err != nil {
		return nil, fmt.Errorf("list metadata from %s: %w", table, err)
	}

	return items, nil
}

func (r *MetadataRepository) CreateSeries(input domain.MetadataWriteInput, slug string, sortOrder int) (*domain.MetadataItem, error) {
	var item domain.MetadataItem
	err := r.db.Get(&item, `
		INSERT INTO series (name, slug, description, parent_series_id, sort_order)
		VALUES (?, ?, ?, NULL, ?)
		RETURNING id, name, slug, sort_order, created_at
	`, input.Name, slug, input.Description, sortOrder)
	if err != nil {
		return nil, fmt.Errorf("create series: %w", err)
	}
	return &item, nil
}

func (r *MetadataRepository) CreateSimple(table string, input domain.MetadataWriteInput, slug string, sortOrder int) (*domain.MetadataItem, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s (name, slug, sort_order)
		VALUES (?, ?, ?)
		RETURNING id, name, slug, sort_order, created_at
	`, table)

	var item domain.MetadataItem
	if err := r.db.Get(&item, query, input.Name, slug, sortOrder); err != nil {
		return nil, fmt.Errorf("create metadata in %s: %w", table, err)
	}
	return &item, nil
}
