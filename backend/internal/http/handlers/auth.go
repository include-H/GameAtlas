package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type AuthHandler struct {
	service *services.AuthService
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var payload struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid auth payload"})
		return
	}

	session, err := h.service.Login(payload.Password)
	if err != nil {
		switch err {
		case services.ErrAuthDisabled:
			c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "auth is not configured"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid password"})
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "login failed"})
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
			"is_admin": isAdminRequest(c),
			"role":     strings.TrimSpace("admin"),
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
