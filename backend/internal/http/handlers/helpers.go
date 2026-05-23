package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

// Transport helpers only write the shared HTTP envelope.
// They must not assemble business payload fields. Handler-specific response
// shapes belong to explicit DTO structs so transport contracts stay reviewable.
func writeJSONSuccess[T any](c *gin.Context, status int, data T) {
	c.JSON(status, successEnvelope[T]{
		Success: true,
		Data:    data,
	})
}

func writeJSONError(c *gin.Context, status int, message string) {
	c.JSON(status, errorEnvelope{
		Success: false,
		Error:   message,
	})
}

func writeJSONErrorWithData[T any](c *gin.Context, status int, message string, data T) {
	c.JSON(status, errorEnvelopeWithData[T]{
		Success: false,
		Error:   message,
		Data:    data,
	})
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || value <= 0 {
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusBadRequest, "无效的 ID 参数")
		return 0, false
	}
	return value, true
}

func parseGamePublicIDParam(c *gin.Context, name string, resolver func(publicID string) (int64, error)) (int64, bool) {
	publicID := strings.TrimSpace(c.Param(name))
	if publicID == "" {
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusBadRequest, "无效的公开 ID 参数")
		return 0, false
	}

	id, err := resolver(publicID)
	if err == nil {
		return id, true
	}
	if errors.Is(err, services.ErrNotFound) {
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusNotFound, "资源不存在")
		return 0, false
	}

	// 2026-05-09: 统一为中文错误信息
	writeJSONError(c, http.StatusBadRequest, "无效的公开 ID 参数")
	return 0, false
}

func writeServiceError(c *gin.Context, err error, validationMessage string) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusNotFound, "资源不存在")
	case errors.Is(err, services.ErrForbiddenPath):
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusForbidden, "文件路径超出允许范围")
	case errors.Is(err, services.ErrMissingFile), errors.Is(err, services.ErrInvalidFile):
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusBadRequest, "注册文件不可用")
	case errors.Is(err, services.ErrValidation):
		writeJSONError(c, http.StatusBadRequest, validationMessage)
	case errors.Is(err, services.ErrUpstream):
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusBadGateway, err.Error())
	case errors.Is(err, services.ErrMissingConfig):
		writeJSONError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, services.ErrInvalidLaunchFile), errors.Is(err, services.ErrMissingSMBConfig):
		writeJSONError(c, http.StatusBadRequest, err.Error())
	default:
		// 2026-05-09: 统一为中文错误信息 (internal server error)
		writeJSONError(c, http.StatusInternalServerError, "服务器内部错误")
	}
}

func decodeJSONStrict(c *gin.Context, target any) error {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("unexpected trailing json")
	}

	return nil
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}
