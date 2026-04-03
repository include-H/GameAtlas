package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/services"
)

type TagsHandler struct {
	service *services.TagsService
}

func NewTagsHandler(service *services.TagsService) *TagsHandler {
	return &TagsHandler{service: service}
}

func (h *TagsHandler) ListGroups(c *gin.Context) {
	items, err := h.service.ListGroups()
	if err != nil {
		writeServiceError(c, err, "invalid tag group payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
	})
}

func (h *TagsHandler) CreateGroup(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	var request tagGroupWriteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid tag group payload",
		})
		return
	}

	input := request.toInput()
	item, err := h.service.CreateGroup(input)
	if err != nil {
		writeServiceError(c, err, "key and name are required")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    item,
	})
}

func (h *TagsHandler) ListTags(c *gin.Context) {
	params := domain.TagsListParams{
		GroupID:  parseQueryInt64(c, "group_id", 0),
		GroupKey: c.Query("group_key"),
	}

	if raw := c.Query("active"); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			params.Active = &value
		}
	}

	items, err := h.service.ListTags(params)
	if err != nil {
		writeServiceError(c, err, "invalid tag payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
	})
}

func (h *TagsHandler) CreateTag(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	var request tagWriteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid tag payload",
		})
		return
	}

	input := request.toInput()
	item, err := h.service.CreateTag(input)
	if err != nil {
		writeServiceError(c, err, "group_id and name are required")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    item,
	})
}
