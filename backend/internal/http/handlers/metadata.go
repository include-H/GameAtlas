package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

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
	options, ok := decodeMetadataListOptions(c)
	if !ok {
		return
	}

	items, err := h.service.List(h.resource, isAdminRequest(c), options)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的元数据请求")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    toMetadataResponses(items),
	})
}

func decodeMetadataListOptions(c *gin.Context) (services.MetadataListOptions, bool) {
	limit, ok := parseMetadataListLimit(c)
	if !ok {
		return services.MetadataListOptions{}, false
	}
	sort, ok := parseMetadataListSort(c)
	if !ok {
		return services.MetadataListOptions{}, false
	}

	options := services.MetadataListOptions{
		Search: c.Query("search"),
		Limit:  limit,
		Sort:   sort,
	}
	return options, true
}

func parseMetadataListLimit(c *gin.Context) (int, bool) {
	raw := c.Query("limit")
	if raw == "" {
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		writeMetadataQueryError(c, "limit")
		return 0, false
	}
	return value, true
}

func parseMetadataListSort(c *gin.Context) (string, bool) {
	raw := strings.TrimSpace(c.Query("sort"))
	if raw == "" {
		return "", true
	}
	switch raw {
	case "name", "popular":
		return raw, true
	default:
		writeMetadataQueryError(c, "sort")
		return "", false
	}
}

func writeMetadataQueryError(c *gin.Context, key string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		// 2026-05-09: 统一为中文错误信息
		"error":   "无效的元数据查询参数: " + key,
	})
}

func (h *MetadataHandler) Get(c *gin.Context) {
	// 2026-05-09: only series resources support detail queries today. This guard
	// belongs here rather than in the service because it is a routing-level
	// constraint: the Get endpoint was wired for series specifically, not as a
	// generic metadata detail action.
	if h.resource.Table != "series" {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			// 2026-05-09: 统一为中文错误信息
		"error":   "资源不存在",
		})
		return
	}

	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	detail, err := h.service.GetSeriesDetail(id, isAdminRequest(c))
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的元数据请求")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"series": toMetadataResponse(*detail.Series),
			"games":  toSeriesGameSummaryResponses(detail.Games),
		},
	})
}

func (h *MetadataHandler) Create(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var request metadataWriteRequest
	// 2026-04-06: metadata writes use strict JSON decode so transport contracts
	// do not silently accept unknown fields that service/domain never defined.
	if err := decodeJSONStrict(c, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			// 2026-05-09: 统一为中文错误信息
		"error":   "无效的元数据请求",
		})
		return
	}

	input := request.toInput()
	item, err := h.service.Create(h.resource, input)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "名称为必填项")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    toMetadataResponse(*item),
	})
}
