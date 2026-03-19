package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/hao/game/internal/config"
)

const AuthCookieName = "gameatlas_admin"

var ErrUnauthorized = errors.New("unauthorized")
var ErrAuthDisabled = errors.New("authentication is not configured")

type AuthService struct {
	adminPassword string
	sessionSecret string
}

func NewAuthService(cfg config.Config) *AuthService {
	return &AuthService{
		adminPassword: strings.TrimSpace(cfg.AdminPassword),
		sessionSecret: strings.TrimSpace(cfg.SessionSecret),
	}
}

func (s *AuthService) Login(password string) (string, error) {
	if s.adminPassword == "" {
		return "", ErrAuthDisabled
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(password)), []byte(s.adminPassword)) != 1 {
		return "", ErrUnauthorized
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
