package repositories

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// AuthAttemptState is a persistence-facing record for auth_login_attempts.
// We keep it in repository to make the service own lockout rules while the
// repository stays responsible for SQL and row mapping.
type AuthAttemptState struct {
	SourceKey       string `db:"source_key"`
	FailCount       int    `db:"fail_count"`
	FirstFailedUnix int64  `db:"first_failed_unix"`
	LastFailedUnix  int64  `db:"last_failed_unix"`
	LockedUntilUnix int64  `db:"locked_until_unix"`
	ExpiresAtUnix   int64  `db:"expires_at_unix"`
}

type AuthAttemptRepository struct {
	db *sqlx.DB
}

func NewAuthAttemptRepository(db *sqlx.DB) *AuthAttemptRepository {
	return &AuthAttemptRepository{db: db}
}

func (r *AuthAttemptRepository) CleanupExpired(nowUnix int64) error {
	if r == nil || r.db == nil {
		return nil
	}

	_, err := r.db.Exec("DELETE FROM auth_login_attempts WHERE expires_at_unix <= ?", nowUnix)
	if err != nil {
		return fmt.Errorf("cleanup expired auth attempts: %w", err)
	}
	return nil
}

func (r *AuthAttemptRepository) GetBySourceKey(sourceKey string) (*AuthAttemptState, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}

	var item AuthAttemptState
	err := r.db.Get(&item, `
		SELECT source_key, fail_count, first_failed_unix, last_failed_unix, locked_until_unix, expires_at_unix
		FROM auth_login_attempts
		WHERE source_key = ?
	`, sourceKey)
	if err == nil {
		return &item, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return nil, fmt.Errorf("get auth attempt by source key: %w", err)
}

func (r *AuthAttemptRepository) Upsert(item AuthAttemptState) error {
	if r == nil || r.db == nil {
		return nil
	}

	_, err := r.db.Exec(`
		INSERT INTO auth_login_attempts (
			source_key, fail_count, first_failed_unix, last_failed_unix, locked_until_unix, expires_at_unix
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_key) DO UPDATE SET
			fail_count = excluded.fail_count,
			first_failed_unix = excluded.first_failed_unix,
			last_failed_unix = excluded.last_failed_unix,
			locked_until_unix = excluded.locked_until_unix,
			expires_at_unix = excluded.expires_at_unix
	`, item.SourceKey, item.FailCount, item.FirstFailedUnix, item.LastFailedUnix, item.LockedUntilUnix, item.ExpiresAtUnix)
	if err != nil {
		return fmt.Errorf("upsert auth attempt: %w", err)
	}
	return nil
}

func (r *AuthAttemptRepository) Delete(sourceKey string) error {
	if r == nil || r.db == nil {
		return nil
	}

	_, err := r.db.Exec("DELETE FROM auth_login_attempts WHERE source_key = ?", sourceKey)
	if err != nil {
		return fmt.Errorf("delete auth attempt: %w", err)
	}
	return nil
}
