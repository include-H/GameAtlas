package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
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
		return 0, domain.ErrNotFound
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
		{name: "not found", err: domain.ErrNotFound, status: http.StatusNotFound, msg: "资源不存在"},
		{name: "forbidden path", err: domain.ErrForbiddenPath, status: http.StatusForbidden, msg: "文件路径超出允许范围"},
		{name: "missing file", err: domain.ErrMissingFile, status: http.StatusBadRequest, msg: "注册文件不可用"},
		{name: "validation", err: domain.ErrValidation, status: http.StatusBadRequest, msg: "bad payload"},
		{name: "upstream", err: domain.ErrUpstream, status: http.StatusBadGateway, msg: "上游服务请求失败"},
		{name: "missing config", err: domain.ErrMissingConfig, status: http.StatusBadRequest, msg: "服务配置不完整"},
		{name: "internal", err: errors.New("boom"), status: http.StatusInternalServerError, msg: "服务器内部错误"},
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
