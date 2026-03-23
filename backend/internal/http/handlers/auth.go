package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

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
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid auth payload"})
		return
	}

	sourceKey := h.service.SourceKey(c.ClientIP(), c.Request.UserAgent())
	session, err := h.service.Login(payload.Password, sourceKey)
	if err != nil {
		var denied *services.LoginDeniedError
		if errors.As(err, &denied) {
			switch denied.Reason {
			case services.LoginDeniedLocked:
				c.JSON(http.StatusTooManyRequests, gin.H{
					"success": false,
					"error":   "错误次数过多，请稍后再试",
					"data": gin.H{
						"retry_after_seconds": denied.RetryAfterSeconds,
						"locked_until_unix":   denied.LockedUntilUnixUTC,
					},
				})
			default:
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error":   "密码错误",
					"data": gin.H{
						"remaining_attempts": denied.RemainingAttempts,
					},
				})
			}
			return
		}

		switch err {
		case services.ErrAuthDisabled:
			c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "登录功能未配置"})
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "登录失败"})
		}
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(services.AuthCookieName, session, int((30 * 24 * time.Hour).Seconds()), "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"is_admin": true,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(services.AuthCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"logged_out": true},
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"is_admin":           isAdminRequest(c),
			"role":               strings.TrimSpace("admin"),
			"admin_display_name": h.adminDisplayName,
		},
	})
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
	c.JSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error":   "admin login required",
	})
	return false
}
