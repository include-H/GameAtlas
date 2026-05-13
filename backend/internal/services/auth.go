package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/repositories"
)

// 2026-05-09: AuthCookieName is defined here rather than in the handler package
// because the session creation and validation logic in AuthService implicitly
// depends on this name being consistent. Both the handler and the router consume
// this constant; keeping it in the service avoids a circular dependency between
// handler and route packages.
const AuthCookieName = "gameatlas_admin"

var ErrAuthDisabled = errors.New("authentication is not configured")

type sessionCacheEntry struct {
	valid         bool
	checkedAtUnix int64
}

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

type AuthService struct {
	adminPassword string
	// Auth lockout policy is owned by the service, but the persistence details
	// stay in repository so future auth features do not mix SQL with rules here.
	attemptsRepo *repositories.AuthAttemptRepository
	sessionsRepo *repositories.AuthSessionRepository
	maxFails     int
	cooldown     time.Duration
	failWindow   time.Duration
	stateTTL     time.Duration
	sessionTTL   time.Duration
	trackBy      string

	// Session validation cache
	sessionCache    sync.Map
	sessionCacheTTL int64 // seconds

	// Cleanup throttle
	lastCleanupUnix atomic.Int64
	cleanupInterval int64 // seconds
}

func NewAuthService(cfg config.Config, attemptsRepo *repositories.AuthAttemptRepository, sessionsRepo *repositories.AuthSessionRepository) *AuthService {
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
	sessionTTL := 30 * 24 * time.Hour

	return &AuthService{
		adminPassword:   strings.TrimSpace(cfg.AdminPassword),
		attemptsRepo:    attemptsRepo,
		sessionsRepo:    sessionsRepo,
		maxFails:        maxFails,
		cooldown:        cooldown,
		failWindow:      failWindow,
		stateTTL:        stateTTL,
		sessionTTL:      sessionTTL,
		trackBy:         trackBy,
		sessionCacheTTL: 30,
		cleanupInterval: 300,
	}
}

func (s *AuthService) SourceKey(clientIP, userAgent string) string {
	// SourceKey is a privacy-preserving, approximate request-source fingerprint
	// used for auth lockout tracking and download dedupe. It is intentionally
	// derived from request metadata available at runtime, so callers must not
	// treat it as a stable device identity across proxy changes, NAT changes,
	// user-agent changes, restarts, or different deployments.
	base := strings.TrimSpace(clientIP)
	if base == "" {
		base = "unknown-ip"
	}
	if s.trackBy == "ip_ua" {
		base = base + "|" + strings.TrimSpace(userAgent)
	}

	secret := s.adminPassword
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

	session, err := s.createSession(nowUnix)
	if err != nil {
		return "", err
	}
	return session, nil
}

func (s *AuthService) IsAdmin(session string) bool {
	if s.adminPassword == "" {
		return true
	}
	session = strings.TrimSpace(session)
	if session == "" {
		return false
	}
	nowUnix := time.Now().UTC().Unix()

	// Check cache first
	if cached, ok := s.sessionCache.Load(session); ok {
		entry := cached.(sessionCacheEntry)
		if nowUnix-entry.checkedAtUnix < s.sessionCacheTTL {
			return entry.valid
		}
	}

	// Cache miss or expired: throttle cleanup, then query DB
	_ = s.cleanupExpiredThrottled(nowUnix)

	item, err := s.loadSession(session)
	if err != nil || item == nil {
		s.sessionCache.Store(session, sessionCacheEntry{valid: false, checkedAtUnix: nowUnix})
		return false
	}
	if item.ExpiresAtUnix <= nowUnix {
		_ = s.deleteSession(session)
		s.sessionCache.Store(session, sessionCacheEntry{valid: false, checkedAtUnix: nowUnix})
		return false
	}
	valid := subtle.ConstantTimeCompare([]byte(session), []byte(item.Token)) == 1
	s.sessionCache.Store(session, sessionCacheEntry{valid: valid, checkedAtUnix: nowUnix})
	return valid
}

func (s *AuthService) Logout(session string) error {
	session = strings.TrimSpace(session)
	if session == "" {
		return nil
	}
	s.sessionCache.Delete(session)
	return s.deleteSession(session)
}

func (s *AuthService) SessionTTLSeconds() int {
	return int(s.sessionTTL.Seconds())
}

func (s *AuthService) cleanupExpired(nowUnix int64) error {
	if s.attemptsRepo != nil {
		if err := s.attemptsRepo.CleanupExpired(nowUnix); err != nil {
			return err
		}
	}
	if s.sessionsRepo != nil {
		if err := s.sessionsRepo.CleanupExpired(nowUnix); err != nil {
			return err
		}
	}
	return nil
}

func (s *AuthService) cleanupExpiredThrottled(nowUnix int64) error {
	lastCleanup := s.lastCleanupUnix.Load()
	if nowUnix-lastCleanup < s.cleanupInterval {
		return nil
	}
	if !s.lastCleanupUnix.CompareAndSwap(lastCleanup, nowUnix) {
		return nil
	}
	return s.cleanupExpired(nowUnix)
}

func (s *AuthService) loadAttempt(sourceKey string) (*repositories.AuthAttemptState, error) {
	if s.attemptsRepo == nil {
		return nil, nil
	}
	return s.attemptsRepo.GetBySourceKey(sourceKey)
}

func (s *AuthService) recordFailure(sourceKey string, nowUnix int64, existing *repositories.AuthAttemptState) (*repositories.AuthAttemptState, error) {
	if s.attemptsRepo == nil {
		return &repositories.AuthAttemptState{
			SourceKey:       sourceKey,
			FailCount:       s.maxFails,
			FirstFailedUnix: nowUnix,
			LastFailedUnix:  nowUnix,
			LockedUntilUnix: nowUnix + int64(s.cooldown.Seconds()),
			ExpiresAtUnix:   nowUnix + int64(s.stateTTLLimit().Seconds()),
		}, nil
	}

	next := repositories.AuthAttemptState{
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

	if err := s.attemptsRepo.Upsert(next); err != nil {
		return nil, err
	}

	return &next, nil
}

func (s *AuthService) deleteAttempt(sourceKey string) error {
	if s.attemptsRepo == nil {
		return nil
	}
	return s.attemptsRepo.Delete(sourceKey)
}

func (s *AuthService) loadSession(token string) (*repositories.AuthSessionState, error) {
	if s.sessionsRepo == nil {
		return nil, nil
	}
	return s.sessionsRepo.Get(token)
}

func (s *AuthService) deleteSession(token string) error {
	if s.sessionsRepo == nil {
		return nil
	}
	return s.sessionsRepo.Delete(token)
}

func (s *AuthService) createSession(nowUnix int64) (string, error) {
	token, err := newAuthSessionToken()
	if err != nil {
		return "", fmt.Errorf("create auth session token: %w", err)
	}
	if s.sessionsRepo == nil {
		return "", fmt.Errorf("create auth session: missing repository")
	}
	if err := s.sessionsRepo.Create(repositories.AuthSessionState{
		Token:         token,
		ExpiresAtUnix: nowUnix + int64(s.sessionTTL.Seconds()),
	}); err != nil {
		return "", fmt.Errorf("create auth session: %w", err)
	}
	return token, nil
}

func newAuthSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *AuthService) stateTTLLimit() time.Duration {
	longest := math.Max(
		s.stateTTL.Seconds(),
		math.Max(s.failWindow.Seconds(), s.cooldown.Seconds())+s.cooldown.Seconds(),
	)
	return time.Duration(longest) * time.Second
}
