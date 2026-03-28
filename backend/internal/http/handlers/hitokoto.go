package handlers

import (
	"net/http"
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
	minLength := parseQueryInt(c, "min_length", 0)
	maxLength := parseQueryInt(c, "max_length", 30)
	if maxLength > 1000 {
		maxLength = 1000
	}
	if maxLength < minLength {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "`max_length` cannot be less than `min_length`",
		})
		return
	}

	sentence, err := h.service.Random(services.HitokotoQuery{
		Categories: c.QueryArray("c"),
		MinLength:  minLength,
		MaxLength:  maxLength,
	})
	if err != nil {
		switch err {
		case services.ErrValidation:
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid hitokoto query",
			})
		case services.ErrNotFound:
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "no sentence matched the query",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "internal server error",
			})
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(c.Query("encode"))) {
	case "text":
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(sentence.Hitokoto))
	default:
		c.JSON(http.StatusOK, sentence)
	}
}
