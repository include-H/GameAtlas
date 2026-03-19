package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/services"
)

type MetadataHandler struct {
	service  *services.MetadataService
	resource services.MetadataResource
}

func NewMetadataHandler(service *services.MetadataService, resource services.MetadataResource) *MetadataHandler {
	return &MetadataHandler{
		service:  service,
		resource: resource,
	}
}

func (h *MetadataHandler) List(c *gin.Context) {
	items, err := h.service.List(h.resource)
	if err != nil {
		writeServiceError(c, err, "invalid metadata payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
	})
}

func (h *MetadataHandler) Create(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var input domain.MetadataWriteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid metadata payload",
		})
		return
	}

	item, err := h.service.Create(h.resource, input)
	if err != nil {
		writeServiceError(c, err, "name is required")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    item,
	})
}
