package services

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hao/game/internal/config"
	dbpkg "github.com/hao/game/internal/db"
	"github.com/hao/game/internal/repositories"
	"github.com/jmoiron/sqlx"
)

func TestAuthServiceSourceKeyDependsOnTrackingMode(t *testing.T) {
	ipOnly := NewAuthService(config.Config{
		AdminPassword: "secret",
		AuthTrackBy:   "ip",
	}, nil, nil)
	ipUA := NewAuthService(config.Config{
		AdminPassword: "secret",
		AuthTrackBy:   "ip_ua",
	}, nil, nil)

	if ipOnly.SourceKey("127.0.0.1", "ua1") != ipOnly.SourceKey("127.0.0.1", "ua2") {
		t.Fatalf("expected ip tracking to ignore user agent")
	}
	if ipUA.SourceKey("127.0.0.1", "ua1") == ipUA.SourceKey("127.0.0.1", "ua2") {
		t.Fatalf("expected ip_ua tracking to include user agent")
	}
}

func TestAuthServiceLoginTracksFailuresAndLocks(t *testing.T) {
	db := openAuthTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewAuthService(config.Config{
		AdminPassword:  "secret",
		AuthMaxFails:   3,
		AuthCooldown:   time.Minute,
		AuthFailWindow: time.Hour,
		AuthStateTTL:   time.Hour,
	}, repositories.NewAuthAttemptRepository(db), repositories.NewAuthSessionRepository(db))

	sourceKey := service.SourceKey("127.0.0.1", "")
	for attempt := 1; attempt <= 2; attempt++ {
		_, err := service.Login("wrong", sourceKey)
		denied, ok := err.(*LoginDeniedError)
		if !ok || denied.Reason != LoginDeniedInvalidPassword {
			t.Fatalf("attempt %d error = %v, want invalid password denial", attempt, err)
		}
		if denied.RemainingAttempts != 3-attempt {
			t.Fatalf("attempt %d remaining attempts = %d, want %d", attempt, denied.RemainingAttempts, 3-attempt)
		}
	}

	_, err := service.Login("wrong", sourceKey)
	denied, ok := err.(*LoginDeniedError)
	if !ok || denied.Reason != LoginDeniedLocked {
		t.Fatalf("third attempt error = %v, want locked denial", err)
	}
	if denied.RetryAfterSeconds <= 0 || denied.LockedUntilUnixUTC <= 0 {
		t.Fatalf("expected lock metadata, got %+v", denied)
	}
}

func TestAuthServiceLoginClearsAttemptsAfterSuccess(t *testing.T) {
	db := openAuthTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewAuthService(config.Config{
		AdminPassword:  "secret",
		AuthMaxFails:   3,
		AuthCooldown:   time.Minute,
		AuthFailWindow: time.Hour,
		AuthStateTTL:   time.Hour,
	}, repositories.NewAuthAttemptRepository(db), repositories.NewAuthSessionRepository(db))

	sourceKey := service.SourceKey("127.0.0.1", "")
	_, _ = service.Login("wrong", sourceKey)

	session, err := service.Login("secret", sourceKey)
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if session == "" || !service.IsAdmin(session) {
		t.Fatalf("expected successful admin session, got %q", session)
	}

	attempt, err := service.loadAttempt(sourceKey)
	if err != nil {
		t.Fatalf("loadAttempt returned error: %v", err)
	}
	if attempt != nil {
		t.Fatalf("expected attempts to be cleared after successful login")
	}

	if err := service.Logout(session); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}
	if service.IsAdmin(session) {
		t.Fatalf("expected logged out session to be rejected")
	}
}

func TestAuthServiceStateTTLLimitUsesLongestWindow(t *testing.T) {
	service := NewAuthService(config.Config{
		AdminPassword:  "secret",
		AuthCooldown:   10 * time.Minute,
		AuthFailWindow: 30 * time.Minute,
		AuthStateTTL:   5 * time.Minute,
	}, nil, nil)

	if got := service.stateTTLLimit(); got != 40*time.Minute {
		t.Fatalf("stateTTLLimit() = %s, want 40m0s", got)
	}
}

func openAuthTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := dbpkg.OpenSQLite(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	if err := dbpkg.RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	return db
}
