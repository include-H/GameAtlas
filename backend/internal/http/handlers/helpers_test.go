package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

func TestParseIDParam(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "id", Value: "42"}}

	got, ok := parseIDParam(context, "id")
	if !ok || got != 42 {
		t.Fatalf("parseIDParam() = (%d, %v), want (42, true)", got, ok)
	}
}

func TestParseIDParamRejectsInvalidValue(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "id", Value: "abc"}}

	_, ok := parseIDParam(context, "id")
	if ok {
		t.Fatalf("expected parseIDParam to fail")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestParseGamePublicIDParam(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "publicId", Value: "game-1"}}

	got, ok := parseGamePublicIDParam(context, "publicId", func(publicID string) (int64, error) {
		if publicID != "game-1" {
			t.Fatalf("resolver called with %q, want game-1", publicID)
		}
		return 7, nil
	})
	if !ok || got != 7 {
		t.Fatalf("parseGamePublicIDParam() = (%d, %v), want (7, true)", got, ok)
	}
}

func TestParseGamePublicIDParamHandlesNotFound(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "publicId", Value: "missing"}}

	_, ok := parseGamePublicIDParam(context, "publicId", func(publicID string) (int64, error) {
		return 0, services.ErrNotFound
	})
	if ok {
		t.Fatalf("expected parseGamePublicIDParam to fail")
	}
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestWriteServiceError(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	tests := []struct {
		name   string
		err    error
		status int
		msg    string
	}{
		{name: "not found", err: services.ErrNotFound, status: http.StatusNotFound, msg: "resource not found"},
		{name: "forbidden path", err: services.ErrForbiddenPath, status: http.StatusForbidden, msg: "file path is outside PRIMARY_ROM_ROOT"},
		{name: "missing file", err: services.ErrMissingFile, status: http.StatusBadRequest, msg: "registered file is unavailable"},
		{name: "validation", err: services.ErrValidation, status: http.StatusBadRequest, msg: "bad payload"},
		{name: "upstream", err: services.ErrUpstream, status: http.StatusBadGateway, msg: services.ErrUpstream.Error()},
		{name: "missing config", err: services.ErrMissingConfig, status: http.StatusBadRequest, msg: services.ErrMissingConfig.Error()},
		{name: "internal", err: errors.New("boom"), status: http.StatusInternalServerError, msg: "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)

			writeServiceError(context, tt.err, "bad payload")

			if recorder.Code != tt.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.status)
			}
			var response map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response["error"] != tt.msg {
				t.Fatalf("error = %#v, want %q", response["error"], tt.msg)
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	if requireAdmin(context) {
		t.Fatalf("expected requireAdmin to reject anonymous request")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}

	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	context.Set("is_admin", true)
	if !requireAdmin(context) {
		t.Fatalf("expected requireAdmin to allow admin request")
	}
}
