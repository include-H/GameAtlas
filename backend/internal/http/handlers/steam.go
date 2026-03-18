package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/services"
)

type SteamHandler struct {
	service *services.SteamService
}

func NewSteamHandler(service *services.SteamService) *SteamHandler {
	return &SteamHandler{service: service}
}

func (h *SteamHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	proxy := strings.TrimSpace(c.Query("proxy"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "search query is required"})
		return
	}
	results, err := h.service.Search(query, proxy)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": results})
}

func (h *SteamHandler) Preview(c *gin.Context) {
	appID, ok := parseIDParam(c, "appId")
	if !ok {
		return
	}
	proxy := strings.TrimSpace(c.Query("proxy"))
	preview, err := h.service.PreviewAssets(appID, proxy)
	if err != nil {
		writeServiceError(c, err, "invalid steam request")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": preview})
}

func (h *SteamHandler) Apply(c *gin.Context) {
	appID, ok := parseIDParam(c, "appId")
	if !ok {
		return
	}

	var input domain.SteamApplyAssetsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid steam asset payload"})
		return
	}
	if input.GameID <= 0 {
		if raw := c.Query("game_id"); raw != "" {
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
				input.GameID = parsed
			}
		}
	}

	preview, err := h.service.ApplyAssets(appID, input)
	if err != nil {
		writeServiceError(c, err, "invalid steam asset payload")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": preview})
}

func (h *SteamHandler) Proxy(c *gin.Context) {
	rawURL := strings.TrimSpace(c.Query("url"))
	proxy := strings.TrimSpace(c.Query("proxy"))
	if rawURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "url is required"})
		return
	}

	contentType, payload, err := h.service.ProxyAsset(rawURL, proxy)
	if err != nil {
		writeServiceError(c, err, "invalid steam proxy request")
		return
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Access-Control-Expose-Headers", "Content-Type, Content-Length")
	c.Data(http.StatusOK, contentType, payload)
}
