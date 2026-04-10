package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type ReviewIssueOverrideHandler struct {
	service *services.ReviewIssueOverrideService
}

func NewReviewIssueOverrideHandler(service *services.ReviewIssueOverrideService) *ReviewIssueOverrideHandler {
	return &ReviewIssueOverrideHandler{service: service}
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
		// 2026-04-06: review override writes are strict so temporary client-side
		// fields do not silently become part of this narrow transport contract.
		if err := decodeJSONStrict(c, &payload); err != nil {
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
		"data":    toReviewIssueOverrideResponse(*item),
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
