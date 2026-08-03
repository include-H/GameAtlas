package repositories

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

type AppSettingsRepository struct {
	db *sqlx.DB
}

func NewAppSettingsRepository(db *sqlx.DB) *AppSettingsRepository {
	return &AppSettingsRepository{db: db}
}

func (r *AppSettingsRepository) List() (map[string]string, error) {
	rows := make([]struct {
		Key   string `db:"key"`
		Value string `db:"value"`
	}, 0)
	if err := r.db.Select(&rows, `SELECT key, value FROM app_settings ORDER BY key ASC`); err != nil {
		return nil, fmt.Errorf("list app settings: %w", err)
	}

	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Key] = row.Value
	}
	return values, nil
}

func (r *AppSettingsRepository) EnsureDefaults(values map[string]string) error {
	for _, key := range sortedMapKeys(values) {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if _, err := r.db.Exec(`
			INSERT INTO app_settings (key, value)
			VALUES (?, ?)
			ON CONFLICT(key) DO NOTHING
		`, key, values[key]); err != nil {
			return fmt.Errorf("ensure app setting %s: %w", key, err)
		}
	}
	return nil
}

func (r *AppSettingsRepository) UpsertMany(values map[string]string) error {
	for _, key := range sortedMapKeys(values) {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if _, err := r.db.Exec(`
			INSERT INTO app_settings (key, value)
			VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET
				value = excluded.value,
				updated_at = CURRENT_TIMESTAMP
		`, key, values[key]); err != nil {
			return fmt.Errorf("upsert app setting %s: %w", key, err)
		}
	}
	return nil
}

func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
