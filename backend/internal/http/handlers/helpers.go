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
		writeJSONError(c, http.StatusBadRequest, "invalid id parameter")
		return 0, false
	}
	return value, true
}

func parseGamePublicIDParam(c *gin.Context, name string, resolver func(publicID string) (int64, error)) (int64, bool) {
	publicID := strings.TrimSpace(c.Param(name))
	if publicID == "" {
		writeJSONError(c, http.StatusBadRequest, "invalid public_id parameter")
		return 0, false
	}

	id, err := resolver(publicID)
	if err == nil {
		return id, true
	}
	if errors.Is(err, services.ErrNotFound) {
		writeJSONError(c, http.StatusNotFound, "resource not found")
		return 0, false
	}

	writeJSONError(c, http.StatusBadRequest, "invalid public_id parameter")
	return 0, false
}

func writeServiceError(c *gin.Context, err error, validationMessage string) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		writeJSONError(c, http.StatusNotFound, "resource not found")
	case errors.Is(err, services.ErrForbiddenPath):
		writeJSONError(c, http.StatusForbidden, "file path is outside PRIMARY_ROM_ROOT")
	case errors.Is(err, services.ErrMissingFile), errors.Is(err, services.ErrInvalidFile):
		writeJSONError(c, http.StatusBadRequest, "registered file is unavailable")
	case errors.Is(err, services.ErrValidation):
		writeJSONError(c, http.StatusBadRequest, validationMessage)
	case errors.Is(err, services.ErrUpstream):
		writeJSONError(c, http.StatusBadGateway, err.Error())
	case errors.Is(err, services.ErrMissingConfig):
		writeJSONError(c, http.StatusBadRequest, err.Error())
	default:
		writeJSONError(c, http.StatusInternalServerError, "internal server error")
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

func parseQueryInt(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func parseQueryInt64(c *gin.Context, key string, fallback int64) int64 {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}
