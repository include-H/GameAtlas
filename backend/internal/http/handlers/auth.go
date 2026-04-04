package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/services"
)

type AuthHandler struct {
	service          *services.AuthService
	adminDisplayName string
}

func NewAuthHandler(service *services.AuthService, cfg config.Config) *AuthHandler {
	return &AuthHandler{
		service:          service,
		adminDisplayName: strings.TrimSpace(cfg.AdminDisplayName),
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var payload struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		writeJSONError(c, http.StatusBadRequest, "invalid auth payload")
		return
	}

	sourceKey := h.service.SourceKey(c.ClientIP(), c.Request.UserAgent())
	session, err := h.service.Login(payload.Password, sourceKey)
	if err != nil {
		var denied *services.LoginDeniedError
		if errors.As(err, &denied) {
			switch denied.Reason {
			case services.LoginDeniedLocked:
				writeJSONErrorWithData(c, http.StatusTooManyRequests, "错误次数过多，请稍后再试", authLockedResponse{
					RetryAfterSeconds: denied.RetryAfterSeconds,
					LockedUntilUnix:   denied.LockedUntilUnixUTC,
				})
			default:
				writeJSONErrorWithData(c, http.StatusUnauthorized, "密码错误", authDeniedResponse{
					RemainingAttempts: denied.RemainingAttempts,
				})
			}
			return
		}

		switch err {
		case services.ErrAuthDisabled:
			writeJSONError(c, http.StatusServiceUnavailable, "登录功能未配置")
		default:
			writeJSONError(c, http.StatusUnauthorized, "登录失败")
		}
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(services.AuthCookieName, session, h.service.SessionMaxAgeSeconds(), "/", "", requestUsesHTTPS(c.Request), true)
	writeJSONSuccess(c, http.StatusOK, authSessionResponse{IsAdmin: true})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	session, _ := c.Cookie(services.AuthCookieName)
	if h.service != nil {
		_ = h.service.Logout(session)
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(services.AuthCookieName, "", -1, "/", "", requestUsesHTTPS(c.Request), true)
	writeJSONSuccess(c, http.StatusOK, authLogoutResponse{LoggedOut: true})
}

func (h *AuthHandler) Me(c *gin.Context) {
	isAdmin := isAdminRequest(c)
	adminDisplayName := ""
	if isAdmin {
		adminDisplayName = h.adminDisplayName
	}
	writeJSONSuccess(c, http.StatusOK, authStateResponse{
		IsAdmin:          isAdmin,
		Role:             currentAuthRole(isAdmin),
		AdminDisplayName: adminDisplayName,
	})
}

func currentAuthRole(isAdmin bool) string {
	if isAdmin {
		return "admin"
	}
	return "guest"
}

func isAdminRequest(c *gin.Context) bool {
	value, exists := c.Get("is_admin")
	if !exists {
		return false
	}
	flag, ok := value.(bool)
	return ok && flag
}

func requireAdmin(c *gin.Context) bool {
	if isAdminRequest(c) {
		return true
	}
	writeJSONError(c, http.StatusUnauthorized, "admin login required")
	return false
}

func requestUsesHTTPS(req *http.Request) bool {
	if req == nil {
		return false
	}
	if req.TLS != nil {
		return true
	}

	forwardedProto := strings.TrimSpace(strings.Split(req.Header.Get("X-Forwarded-Proto"), ",")[0])
	return strings.EqualFold(forwardedProto, "https")
}
