package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
)

func TestTagsHandlerListRejectsInvalidActiveQuery(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := NewTagsHandler(services.NewTagsService(repositories.NewTagsRepository(db)))
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/tags?active=maybe", nil)

	handler.ListTags(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid tag query parameter: active"`) {
		t.Fatalf("body = %s, want invalid active query error", recorder.Body.String())
	}
}

func TestTagsHandlerListRejectsInvalidGroupIDQuery(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := NewTagsHandler(services.NewTagsService(repositories.NewTagsRepository(db)))
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/tags?group_id=oops", nil)

	handler.ListTags(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid tag query parameter: group_id"`) {
		t.Fatalf("body = %s, want invalid group_id query error", recorder.Body.String())
	}
}

func TestTagsHandlerCreateGroupRejectsUnknownJSONFields(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := NewTagsHandler(services.NewTagsService(repositories.NewTagsRepository(db)))
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/tag-groups", strings.NewReader(`{"key":"genre","name":"Genre","legacy":true}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("is_admin", true)

	handler.CreateGroup(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid tag group payload"`) {
		t.Fatalf("body = %s, want invalid tag group payload", recorder.Body.String())
	}
}

func TestMetadataHandlerCreateRejectsUnknownJSONFields(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := NewMetadataHandler(
		services.NewMetadataService(repositories.NewMetadataRepository(db)),
		services.MetadataResource{Table: "developers", ResourceName: "developers"},
	)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/developers", strings.NewReader(`{"name":"Valve","legacy":true}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("is_admin", true)

	handler.Create(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid metadata payload"`) {
		t.Fatalf("body = %s, want invalid metadata payload", recorder.Body.String())
	}
}

func TestMetadataHandlerListRejectsInvalidLimitQuery(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := NewMetadataHandler(
		services.NewMetadataService(repositories.NewMetadataRepository(db)),
		services.MetadataResource{Table: "developers", ResourceName: "developers"},
	)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/developers?limit=oops", nil)

	handler.List(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid metadata query parameter: limit"`) {
		t.Fatalf("body = %s, want invalid metadata limit query error", recorder.Body.String())
	}
}

func TestMetadataHandlerListRejectsInvalidSortQuery(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := NewMetadataHandler(
		services.NewMetadataService(repositories.NewMetadataRepository(db)),
		services.MetadataResource{Table: "series", ResourceName: "series"},
	)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/series?sort=random", nil)

	handler.List(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid metadata query parameter: sort"`) {
		t.Fatalf("body = %s, want invalid metadata sort query error", recorder.Body.String())
	}
}
