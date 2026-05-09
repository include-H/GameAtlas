package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
)

func TestReviewIssueOverrideHandlerIgnoreRejectsInvalidJSON(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "review-ignore", "Review Ignore", "public", "")
	handler := NewReviewIssueOverrideHandler(newReviewOverrideHandlerService(db))
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/games/review-ignore/review-overrides/missing-cover", strings.NewReader("{"))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{
		{Key: "publicId", Value: "review-ignore"},
		{Key: "issueKey", Value: "missing-cover"},
	}
	context.Set("is_admin", true)

	handler.Ignore(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的审查覆盖请求"`) {
		t.Fatalf("body = %s, want 无效的审查覆盖请求", recorder.Body.String())
	}
}

func TestReviewIssueOverrideHandlerIgnoreRejectsUnknownJSONFields(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "review-ignore-unknown", "Review Ignore Unknown", "public", "")
	handler := NewReviewIssueOverrideHandler(newReviewOverrideHandlerService(db))
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/games/review-ignore-unknown/review-overrides/missing-cover", strings.NewReader(`{"reason":"ok","legacy":true}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{
		{Key: "publicId", Value: "review-ignore-unknown"},
		{Key: "issueKey", Value: "missing-cover"},
	}
	context.Set("is_admin", true)

	handler.Ignore(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的审查覆盖请求"`) {
		t.Fatalf("body = %s, want 无效的审查覆盖请求", recorder.Body.String())
	}
}

func TestReviewIssueOverrideHandlerIgnoreAndDeleteFlow(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "review-flow", "Review Flow", "public", "")
	handler := NewReviewIssueOverrideHandler(newReviewOverrideHandlerService(db))

	ignoreRecorder := httptest.NewRecorder()
	ignoreContext, _ := gin.CreateTestContext(ignoreRecorder)
	ignoreContext.Request = httptest.NewRequest(http.MethodPost, "/api/games/review-flow/review-overrides/missing-cover", strings.NewReader(`{"reason":"  accepted  "}`))
	ignoreContext.Request.Header.Set("Content-Type", "application/json")
	ignoreContext.Params = gin.Params{
		{Key: "publicId", Value: "review-flow"},
		{Key: "issueKey", Value: "missing-cover"},
	}
	ignoreContext.Set("is_admin", true)

	handler.Ignore(ignoreContext)

	if ignoreRecorder.Code != http.StatusOK {
		t.Fatalf("ignore status = %d, want %d", ignoreRecorder.Code, http.StatusOK)
	}

	deleteRecorder := httptest.NewRecorder()
	deleteContext, _ := gin.CreateTestContext(deleteRecorder)
	deleteContext.Request = httptest.NewRequest(http.MethodDelete, "/api/games/review-flow/review-overrides/missing-cover", nil)
	deleteContext.Params = gin.Params{
		{Key: "publicId", Value: "review-flow"},
		{Key: "issueKey", Value: "missing-cover"},
	}
	deleteContext.Set("is_admin", true)

	handler.Delete(deleteContext)

	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d", deleteRecorder.Code, http.StatusOK)
	}
}

func newReviewOverrideHandlerService(db *sqlx.DB) *services.ReviewIssueOverrideService {
	return services.NewReviewIssueOverrideService(
		repositories.NewGamesRepository(db),
		repositories.NewReviewIssueOverrideRepository(db),
	)
}
