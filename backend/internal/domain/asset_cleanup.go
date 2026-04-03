package domain

type AssetCleanupTask struct {
	ID           int64   `db:"id"`
	AssetPath    string  `db:"asset_path"`
	Source       string  `db:"source"`
	LastError    *string `db:"last_error"`
	AttemptCount int64   `db:"attempt_count"`
	CreatedAt    string  `db:"created_at"`
	UpdatedAt    string  `db:"updated_at"`
}
