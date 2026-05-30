package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type HitokotoHandler struct {
	service *services.HitokotoService
}

func NewHitokotoHandler(service *services.HitokotoService) *HitokotoHandler {
	return &HitokotoHandler{service: service}
}

func (h *HitokotoHandler) Get(c *gin.Context) {
	minLength, ok := parseHitokotoLengthQuery(c, "min_length", 0)
	if !ok {
		return
	}
	maxLength, ok := parseHitokotoLengthQuery(c, "max_length", 34)
	if !ok {
		return
	}
	if maxLength > 1000 {
		maxLength = 1000
	}
	if maxLength < minLength {
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusBadRequest, "`max_length` 不能小于 `min_length`")
		return
	}

	sentence, err := h.service.Random(services.HitokotoQuery{
		Categories: c.QueryArray("c"),
		MinLength:  minLength,
		MaxLength:  maxLength,
	})
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的一言查询")
		return
	}

	switch strings.ToLower(strings.TrimSpace(c.Query("encode"))) {
	case "text":
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(sentence.Hitokoto))
	default:
		writeJSONSuccess(c, http.StatusOK, toHitokotoSentenceResponse(sentence))
	}
}

func parseHitokotoLengthQuery(c *gin.Context, key string, fallback int) (int, bool) {
	raw := c.Query(key)
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusBadRequest, "无效的一言查询参数: "+key)
		return 0, false
	}
	return value, true
}
