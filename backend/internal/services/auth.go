package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
)

const AuthCookieName = "gameatlas_admin"

var ErrAuthDisabled = errors.New("authentication is not configured")

type LoginDeniedReason string

const (
	LoginDeniedInvalidPassword LoginDeniedReason = "invalid_password"
	LoginDeniedLocked          LoginDeniedReason = "locked"
)

type LoginDeniedError struct {
	Reason             LoginDeniedReason
	RemainingAttempts  int
	RetryAfterSeconds  int64
	LockedUntilUnixUTC int64
}

func (e *LoginDeniedError) Error() string {
	switch e.Reason {
	case LoginDeniedLocked:
		return "login locked"
	default:
		return "invalid password"
	}
}

type authAttemptState struct {
	SourceKey       string `db:"source_key"`
	FailCount       int    `db:"fail_count"`
	FirstFailedUnix int64  `db:"first_failed_unix"`
	LastFailedUnix  int64  `db:"last_failed_unix"`
	LockedUntilUnix int64  `db:"locked_until_unix"`
	ExpiresAtUnix   int64  `db:"expires_at_unix"`
}

type AuthService struct {
	adminPassword string
	sessionSecret string
	db            *sqlx.DB
	maxFails      int
	cooldown      time.Duration
	failWindow    time.Duration
	stateTTL      time.Duration
	trackBy       string
}

func NewAuthService(cfg config.Config, db *sqlx.DB) *AuthService {
	maxFails := cfg.AuthMaxFails
	if maxFails <= 0 {
		maxFails = 5
	}
	cooldown := cfg.AuthCooldown
	if cooldown <= 0 {
		cooldown = 10 * time.Minute
	}
	failWindow := cfg.AuthFailWindow
	if failWindow <= 0 {
		failWindow = 30 * time.Minute
	}
	stateTTL := cfg.AuthStateTTL
	if stateTTL <= 0 {
		stateTTL = 24 * time.Hour
	}
	trackBy := strings.ToLower(strings.TrimSpace(cfg.AuthTrackBy))
	if trackBy != "ip_ua" {
		trackBy = "ip"
	}

	return &AuthService{
		adminPassword: strings.TrimSpace(cfg.AdminPassword),
		sessionSecret: strings.TrimSpace(cfg.SessionSecret),
		db:            db,
		maxFails:      maxFails,
		cooldown:      cooldown,
		failWindow:    failWindow,
		stateTTL:      stateTTL,
		trackBy:       trackBy,
	}
}

func (s *AuthService) SourceKey(clientIP, userAgent string) string {
	base := strings.TrimSpace(clientIP)
	if base == "" {
		base = "unknown-ip"
	}
	if s.trackBy == "ip_ua" {
		base = base + "|" + strings.TrimSpace(userAgent)
	}

	secret := s.sessionSecret
	if secret == "" {
		secret = "auth-source-default-secret"
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(base))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *AuthService) Login(password, sourceKey string) (string, error) {
	if s.adminPassword == "" {
		return "", ErrAuthDisabled
	}
	if sourceKey == "" {
		sourceKey = s.SourceKey("unknown-ip", "")
	}

	now := time.Now().UTC()
	nowUnix := now.Unix()
	_ = s.cleanupExpired(nowUnix)

	attempt, err := s.loadAttempt(sourceKey)
	if err != nil {
		return "", fmt.Errorf("load auth attempt: %w", err)
	}
	if attempt != nil {
		if attempt.ExpiresAtUnix <= nowUnix || (attempt.FirstFailedUnix > 0 && now.Sub(time.Unix(attempt.FirstFailedUnix, 0)) > s.failWindow) {
			if err := s.deleteAttempt(sourceKey); err != nil {
				return "", fmt.Errorf("reset expired auth attempt: %w", err)
			}
			attempt = nil
		}
	}

	if attempt != nil && attempt.LockedUntilUnix > nowUnix {
		return "", &LoginDeniedError{
			Reason:             LoginDeniedLocked,
			RetryAfterSeconds:  attempt.LockedUntilUnix - nowUnix,
			LockedUntilUnixUTC: attempt.LockedUntilUnix,
		}
	}

	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(password)), []byte(s.adminPassword)) != 1 {
		updated, updateErr := s.recordFailure(sourceKey, nowUnix, attempt)
		if updateErr != nil {
			return "", fmt.Errorf("record auth failure: %w", updateErr)
		}

		if updated.LockedUntilUnix > nowUnix {
			return "", &LoginDeniedError{
				Reason:             LoginDeniedLocked,
				RetryAfterSeconds:  updated.LockedUntilUnix - nowUnix,
				LockedUntilUnixUTC: updated.LockedUntilUnix,
			}
		}

		remaining := s.maxFails - updated.FailCount
		if remaining < 0 {
			remaining = 0
		}
		return "", &LoginDeniedError{
			Reason:            LoginDeniedInvalidPassword,
			RemainingAttempts: remaining,
		}
	}

	if err := s.deleteAttempt(sourceKey); err != nil {
		return "", fmt.Errorf("clear auth attempts after success: %w", err)
	}
	return s.sessionValue(), nil
}

func (s *AuthService) IsAdmin(session string) bool {
	if s.adminPassword == "" {
		return true
	}
	expected := s.sessionValue()
	if expected == "" || session == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(session), []byte(expected)) == 1
}

func (s *AuthService) sessionValue() string {
	if s.sessionSecret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(s.sessionSecret))
	_, _ = mac.Write([]byte("admin-session"))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *AuthService) cleanupExpired(nowUnix int64) error {
	if s.db == nil {
		return nil
	}

	_, err := s.db.Exec("DELETE FROM auth_login_attempts WHERE expires_at_unix <= ?", nowUnix)
	return err
}

func (s *AuthService) loadAttempt(sourceKey string) (*authAttemptState, error) {
	if s.db == nil {
		return nil, nil
	}

	var item authAttemptState
	err := s.db.Get(&item, `
		SELECT source_key, fail_count, first_failed_unix, last_failed_unix, locked_until_unix, expires_at_unix
		FROM auth_login_attempts
		WHERE source_key = ?
	`, sourceKey)
	if err == nil {
		return &item, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return nil, err
}

func (s *AuthService) recordFailure(sourceKey string, nowUnix int64, existing *authAttemptState) (*authAttemptState, error) {
	if s.db == nil {
		return &authAttemptState{
			SourceKey:       sourceKey,
			FailCount:       s.maxFails,
			FirstFailedUnix: nowUnix,
			LastFailedUnix:  nowUnix,
			LockedUntilUnix: nowUnix + int64(s.cooldown.Seconds()),
			ExpiresAtUnix:   nowUnix + int64(s.stateTTLLimit().Seconds()),
		}, nil
	}

	next := authAttemptState{
		SourceKey:       sourceKey,
		FailCount:       1,
		FirstFailedUnix: nowUnix,
		LastFailedUnix:  nowUnix,
		LockedUntilUnix: 0,
		ExpiresAtUnix:   nowUnix + int64(s.stateTTLLimit().Seconds()),
	}

	if existing != nil {
		next = *existing
		if next.FirstFailedUnix <= 0 || nowUnix-next.FirstFailedUnix > int64(s.failWindow.Seconds()) {
			next.FailCount = 0
			next.FirstFailedUnix = nowUnix
			next.LockedUntilUnix = 0
		}
		next.FailCount++
		next.LastFailedUnix = nowUnix
		next.ExpiresAtUnix = nowUnix + int64(s.stateTTLLimit().Seconds())
	}

	if next.FailCount >= s.maxFails {
		next.LockedUntilUnix = nowUnix + int64(s.cooldown.Seconds())
	}

	_, err := s.db.Exec(`
		INSERT INTO auth_login_attempts (
			source_key, fail_count, first_failed_unix, last_failed_unix, locked_until_unix, expires_at_unix
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_key) DO UPDATE SET
			fail_count = excluded.fail_count,
			first_failed_unix = excluded.first_failed_unix,
			last_failed_unix = excluded.last_failed_unix,
			locked_until_unix = excluded.locked_until_unix,
			expires_at_unix = excluded.expires_at_unix
	`, next.SourceKey, next.FailCount, next.FirstFailedUnix, next.LastFailedUnix, next.LockedUntilUnix, next.ExpiresAtUnix)
	if err != nil {
		return nil, err
	}

	return &next, nil
}

func (s *AuthService) deleteAttempt(sourceKey string) error {
	if s.db == nil {
		return nil
	}

	_, err := s.db.Exec("DELETE FROM auth_login_attempts WHERE source_key = ?", sourceKey)
	return err
}

func (s *AuthService) stateTTLLimit() time.Duration {
	longest := math.Max(
		s.stateTTL.Seconds(),
		math.Max(s.failWindow.Seconds(), s.cooldown.Seconds())+s.cooldown.Seconds(),
	)
	return time.Duration(longest) * time.Second
}
