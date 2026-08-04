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

	items, err := h.service.List(h.resource, isAdminRequest(c), options)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的元数据请求")
		return
	}

	writeJSONSuccess(c, http.StatusOK, toMetadataResponses(items))
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
		writeJSONError(c, http.StatusBadRequest, "无效的元数据查询参数: limit")
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

	includeAll := isAdminRequest(c)
	response := gin.H{"games": []gameListItemResponse{}}
	switch h.resource.Type {
	case domain.MetadataSeries:
		detail, err := h.service.GetSeriesDetail(id, includeAll)
		if err != nil {
			writeServiceError(c, err, "无效的元数据请求")
			return
		}
		response["series"] = toMetadataResponse(*detail.Series)
		response["games"] = toMetadataGameSummaryResponses(detail.Games)
	case domain.MetadataPublishers:
		detail, err := h.service.GetPublisherDetail(id, includeAll)
		if err != nil {
			writeServiceError(c, err, "无效的元数据请求")
			return
		}
		response["publisher"] = toMetadataResponse(*detail.Publisher)
		response["games"] = toMetadataGameSummaryResponses(detail.Games)
	}

	writeJSONSuccess(c, http.StatusOK, response)
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
