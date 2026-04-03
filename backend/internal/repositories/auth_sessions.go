package repositories

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type AuthSessionState struct {
	Token         string `db:"token"`
	ExpiresAtUnix int64  `db:"expires_at_unix"`
}

type AuthSessionRepository struct {
	db *sqlx.DB
}

func NewAuthSessionRepository(db *sqlx.DB) *AuthSessionRepository {
	return &AuthSessionRepository{db: db}
}

func (r *AuthSessionRepository) CleanupExpired(nowUnix int64) error {
	if r == nil || r.db == nil {
		return nil
	}

	if _, err := r.db.Exec("DELETE FROM auth_sessions WHERE expires_at_unix <= ?", nowUnix); err != nil {
		return fmt.Errorf("cleanup expired auth sessions: %w", err)
	}
	return nil
}

func (r *AuthSessionRepository) Create(item AuthSessionState) error {
	if r == nil || r.db == nil {
		return nil
	}

	if _, err := r.db.Exec(`
		INSERT INTO auth_sessions (token, expires_at_unix)
		VALUES (?, ?)
	`, item.Token, item.ExpiresAtUnix); err != nil {
		return fmt.Errorf("create auth session: %w", err)
	}
	return nil
}

func (r *AuthSessionRepository) Get(token string) (*AuthSessionState, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}

	var item AuthSessionState
	err := r.db.Get(&item, `
		SELECT token, expires_at_unix
		FROM auth_sessions
		WHERE token = ?
	`, token)
	if err == nil {
		return &item, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return nil, fmt.Errorf("get auth session: %w", err)
}

func (r *AuthSessionRepository) Delete(token string) error {
	if r == nil || r.db == nil {
		return nil
	}

	if _, err := r.db.Exec("DELETE FROM auth_sessions WHERE token = ?", token); err != nil {
		return fmt.Errorf("delete auth session: %w", err)
	}
	return nil
}
