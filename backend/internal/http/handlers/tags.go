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
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的标签组请求")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    toTagGroupResponses(items),
	})
}

func (h *TagsHandler) CreateGroup(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	var request tagGroupWriteRequest
	// 2026-04-06: tag writes reject unknown fields and trailing JSON so
	// transport decode stays explicit instead of relying on service trimming.
	if err := decodeJSONStrict(c, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			// 2026-05-09: 统一为中文错误信息
		"error":   "无效的标签组请求",
		})
		return
	}

	input := request.toInput()
	item, err := h.service.CreateGroup(input)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "标识和名称为必填项")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    toTagGroupResponse(*item),
	})
}

func (h *TagsHandler) ListTags(c *gin.Context) {
	groupID, ok := parseTagsListInt64Query(c, "group_id", 0)
	if !ok {
		return
	}
	params := domain.TagsListParams{
		GroupID:  groupID,
		GroupKey: c.Query("group_key"),
	}

	if raw := c.Query("active"); raw != "" {
		// 2026-04-06: invalid boolean filters are request errors, not "unset".
		// Impact: tag list query semantics now match the stricter games filters.
		value, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				// 2026-05-09: 统一为中文错误信息
		"error":   "无效的标签查询参数: active",
			})
			return
		}
		params.Active = &value
	}

	items, err := h.service.ListTags(params)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的标签请求")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    toTagResponses(items),
	})
}

func parseTagsListInt64Query(c *gin.Context, key string, fallback int64) (int64, bool) {
	raw := c.Query(key)
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			// 2026-05-09: 统一为中文错误信息
		"error":   "无效的标签查询参数: " + key,
		})
		return 0, false
	}
	return value, true
}

func (h *TagsHandler) CreateTag(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	var request tagWriteRequest
	if err := decodeJSONStrict(c, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			// 2026-05-09: 统一为中文错误信息
		"error":   "无效的标签请求",
		})
		return
	}

	input := request.toInput()
	item, err := h.service.CreateTag(input)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "所属标签组 ID 和名称为必填项")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    toTagResponse(*item),
	})
}
