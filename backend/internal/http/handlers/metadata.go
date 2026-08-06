package handlers

import (
	"net/http"
	"strconv"
	"strings"

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
	options, ok := decodeMetadataListOptions(c)
	if !ok {
		return
	}

	result, err := h.service.ListPage(h.resource, isAdminRequest(c), options)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的元数据请求")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    toMetadataResponses(result.Items),
		"pagination": gin.H{
			"page":       result.Page,
			"limit":      result.Limit,
			"total":      result.Total,
			"totalPages": result.TotalPages,
		},
	})
}

func decodeMetadataListOptions(c *gin.Context) (services.MetadataListOptions, bool) {
	page, ok := parseMetadataListPage(c)
	if !ok {
		return services.MetadataListOptions{}, false
	}
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
		Page:   page,
	}
	return options, true
}

func parseMetadataListPage(c *gin.Context) (int, bool) {
	raw := c.Query("page")
	if raw == "" {
		return 1, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		writeJSONError(c, http.StatusBadRequest, "无效的元数据查询参数: page")
		return 0, false
	}
	if value <= 0 {
		value = 1
	}
	return value, true
}

func parseMetadataListLimit(c *gin.Context) (int, bool) {
	raw := c.Query("limit")
	if raw == "" {
		return 24, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		writeJSONError(c, http.StatusBadRequest, "无效的元数据查询参数: limit")
		return 0, false
	}
	if value <= 0 {
		value = 24
	}
	if value > 100 {
		value = 100
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
		writeJSONError(c, http.StatusBadRequest, "无效的元数据查询参数: sort")
		return "", false
	}
}

func (h *MetadataHandler) Get(c *gin.Context) {
	if h.resource.Type != domain.MetadataSeries && h.resource.Type != domain.MetadataPublishers {
		writeJSONError(c, http.StatusNotFound, "资源不存在")
		return
	}

	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	page, limit, ok := parseMetadataDetailPagination(c)
	if !ok {
		return
	}

	includeAll := isAdminRequest(c)
	response := gin.H{"games": []gameListItemResponse{}}
	switch h.resource.Type {
	case domain.MetadataSeries:
		detail, err := h.service.GetSeriesDetail(id, includeAll, services.MetadataDetailOptions{
			Page:  page,
			Limit: limit,
		})
		if err != nil {
			writeServiceError(c, err, "无效的元数据请求")
			return
		}
		response["series"] = toMetadataResponse(*detail.Series)
		response["games"] = toMetadataGameSummaryResponses(detail.Games)
		response["pagination"] = metadataPaginationResponse(detail.Page, detail.Limit, detail.Total, detail.TotalPages)
	case domain.MetadataPublishers:
		detail, err := h.service.GetPublisherDetail(id, includeAll, services.MetadataDetailOptions{
			Page:  page,
			Limit: limit,
		})
		if err != nil {
			writeServiceError(c, err, "无效的元数据请求")
			return
		}
		response["publisher"] = toMetadataResponse(*detail.Publisher)
		response["games"] = toMetadataGameSummaryResponses(detail.Games)
		response["pagination"] = metadataPaginationResponse(detail.Page, detail.Limit, detail.Total, detail.TotalPages)
	}

	writeJSONSuccess(c, http.StatusOK, response)
}

func parseMetadataDetailPagination(c *gin.Context) (int, int, bool) {
	page, ok := parseMetadataListPage(c)
	if !ok {
		return 0, 0, false
	}
	limit, ok := parseMetadataListLimit(c)
	if !ok {
		return 0, 0, false
	}
	return page, limit, true
}

func metadataPaginationResponse(page int, limit int, total int, totalPages int) gin.H {
	return gin.H{
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
	}
}

func (h *MetadataHandler) Create(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var request metadataWriteRequest
	// 2026-04-06: metadata writes use strict JSON decode so transport contracts
	// do not silently accept unknown fields that service/domain never defined.
	if err := decodeJSONStrict(c, &request); err != nil {
		writeJSONError(c, http.StatusBadRequest, "无效的元数据请求")
		return
	}

	input := request.toInput()
	item, err := h.service.Create(h.resource, input)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "名称为必填项")
		return
	}

	writeJSONSuccess(c, http.StatusCreated, toMetadataResponse(*item))
}
