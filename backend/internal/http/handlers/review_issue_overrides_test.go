package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
)

func TestParseGamePublicIDsQuery(t *testing.T) {
	resolved := []string{}
	resolver := func(publicID string) (int64, error) {
		resolved = append(resolved, publicID)
		switch publicID {
		case "game-1":
			return 1, nil
		case "game-2":
			return 2, nil
		default:
			return 0, errors.New("not found")
		}
	}

	got, err := parseGamePublicIDsQuery(" game-1, ,game-2 ", resolver)
	if err != nil {
		t.Fatalf("parseGamePublicIDsQuery returned error: %v", err)
	}
	if !reflect.DeepEqual(got, []int64{1, 2}) {
		t.Fatalf("parseGamePublicIDsQuery() = %#v, want []int64{1, 2}", got)
	}
	if !reflect.DeepEqual(resolved, []string{"game-1", "game-2"}) {
		t.Fatalf("resolver calls = %#v", resolved)
	}
}

func TestParseGamePublicIDsQueryReturnsErrorOnInvalidID(t *testing.T) {
	resolver := func(publicID string) (int64, error) {
		return 0, errors.New("bad id: " + publicID)
	}
	if _, err := parseGamePublicIDsQuery("missing", resolver); err == nil {
		t.Fatalf("expected parseGamePublicIDsQuery to return error")
	}
}

func TestReviewIssueOverrideHandlerListReturnsBadRequestForInvalidGameIDsQuery(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := NewReviewIssueOverrideHandler(newReviewOverrideHandlerService(db))
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/review-overrides?game_ids=missing", nil)
	context.Set("is_admin", true)

	handler.List(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid game_ids query"`) {
		t.Fatalf("body = %s, want invalid game_ids query error", recorder.Body.String())
	}
}

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
	if !strings.Contains(recorder.Body.String(), `"error":"invalid review override payload"`) {
		t.Fatalf("body = %s, want invalid review override payload error", recorder.Body.String())
	}
}

func TestReviewIssueOverrideHandlerIgnoreDeleteAndListFlow(t *testing.T) {
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

	listRecorder := httptest.NewRecorder()
	listContext, _ := gin.CreateTestContext(listRecorder)
	listContext.Request = httptest.NewRequest(http.MethodGet, "/api/review-overrides?game_ids=review-flow", nil)
	listContext.Set("is_admin", true)

	handler.List(listContext)

	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRecorder.Code, http.StatusOK)
	}
	var listResponse struct {
		Data []struct {
			IssueKey string  `json:"issue_key"`
			Reason   *string `json:"reason"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResponse.Data) != 1 || listResponse.Data[0].IssueKey != "missing-cover" {
		t.Fatalf("list data = %+v, want missing-cover override", listResponse.Data)
	}
	if listResponse.Data[0].Reason == nil || *listResponse.Data[0].Reason != "accepted" {
		t.Fatalf("list reason = %v, want trimmed accepted", listResponse.Data[0].Reason)
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
