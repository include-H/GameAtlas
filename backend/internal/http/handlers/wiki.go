package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type WikiHandler struct {
	service *services.WikiService
}

func NewWikiHandler(service *services.WikiService) *WikiHandler {
	return &WikiHandler{service: service}
}

func (h *WikiHandler) Get(c *gin.Context) {
	gameID, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}

	document, err := h.service.Get(gameID, isAdminRequest(c))
	if err != nil {
		writeServiceError(c, err, "invalid wiki payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    document,
	})
}

func (h *WikiHandler) Update(c *gin.Context) {
	gameID, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}
	if !requireAdmin(c) {
		return
	}

	var request wikiWriteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid wiki payload",
		})
		return
	}

	input := request.toInput()
	document, err := h.service.Update(gameID, input)
	if err != nil {
		writeServiceError(c, err, "invalid wiki payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    document,
	})
}

func (h *WikiHandler) History(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	gameID, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}

	items, err := h.service.History(gameID, isAdminRequest(c))
	if err != nil {
		writeServiceError(c, err, "invalid wiki payload")
		return
	}

	response := make([]gin.H, 0, len(items))
	for _, item := range items {
		response = append(response, gin.H{
			"id":             item.ID,
			"game_id":        item.GameID,
			"content":        item.Content,
			"change_summary": item.ChangeSummary,
			"created_at":     item.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}
