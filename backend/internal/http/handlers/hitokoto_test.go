package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

func TestHitokotoHandlerGetReturnsJSONSentence(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/hitokoto?c=c&min_length=8&max_length=20", nil)

	handler := NewHitokotoHandler(services.NewHitokotoService())
	handler.Get(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Hitokoto string `json:"hitokoto"`
		Type     string `json:"type"`
		From     string `json:"from"`
		Length   int    `json:"length"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Hitokoto == "" || response.From == "" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.Type != "c" {
		t.Fatalf("type = %q, want c", response.Type)
	}
	if response.Length < 8 || response.Length > 20 {
		t.Fatalf("length = %d, want within [8,20]", response.Length)
	}
}

func TestHitokotoHandlerGetReturnsTextWhenEncodeText(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/hitokoto?c=c&encode=text&min_length=8&max_length=20", nil)

	handler := NewHitokotoHandler(services.NewHitokotoService())
	handler.Get(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("expected text body")
	}
}

func TestHitokotoHandlerGetRejectsInvalidLengthRange(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/hitokoto?min_length=20&max_length=10", nil)

	handler := NewHitokotoHandler(services.NewHitokotoService())
	handler.Get(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
