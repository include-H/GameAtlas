package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || value <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid id parameter",
		})
		return 0, false
	}
	return value, true
}

func writeServiceError(c *gin.Context, err error, validationMessage string) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "resource not found",
		})
	case errors.Is(err, services.ErrForbiddenPath):
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "file path is outside PRIMARY_ROM_ROOT",
		})
	case errors.Is(err, services.ErrMissingFile), errors.Is(err, services.ErrInvalidFile):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "registered file is unavailable",
		})
	case errors.Is(err, services.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   validationMessage,
		})
	case errors.Is(err, services.ErrMissingConfig):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "internal server error",
		})
	}
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}
