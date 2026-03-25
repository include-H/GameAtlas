package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type ReviewIssueOverrideHandler struct {
	service *services.ReviewIssueOverrideService
}

func NewReviewIssueOverrideHandler(service *services.ReviewIssueOverrideService) *ReviewIssueOverrideHandler {
	return &ReviewIssueOverrideHandler{service: service}
}

func (h *ReviewIssueOverrideHandler) List(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	gameIDs, err := parseGamePublicIDsQuery(c.Query("game_ids"), h.service.ResolveGameID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid game_ids query",
		})
		return
	}

	items, err := h.service.List(gameIDs)
	if err != nil {
		writeServiceError(c, err, "invalid review override payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
	})
}

func (h *ReviewIssueOverrideHandler) Ignore(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	gameID, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}

	issueKey := c.Param("issueKey")
	var payload struct {
		Reason *string `json:"reason"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid review override payload",
			})
			return
		}
	}

	item, err := h.service.Ignore(gameID, issueKey, payload.Reason)
	if err != nil {
		writeServiceError(c, err, "invalid review override payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    item,
	})
}

func (h *ReviewIssueOverrideHandler) Delete(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	gameID, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}

	if err := h.service.Delete(gameID, c.Param("issueKey")); err != nil {
		writeServiceError(c, err, "invalid review override payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"deleted": true,
		},
	})
}

func parseGamePublicIDsQuery(raw string, resolver func(publicID string) (int64, error)) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	items := make([]int64, 0, len(parts))
	for _, part := range parts {
		publicID := strings.TrimSpace(part)
		if publicID == "" {
			continue
		}
		id, err := resolver(publicID)
		if err != nil || id <= 0 {
			return nil, err
		}
		items = append(items, id)
	}
	return items, nil
}
