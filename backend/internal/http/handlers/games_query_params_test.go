package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDecodeGamesListParamsDefaultsLimitTo24(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	for _, requestURL := range []string{
		"/api/games",
		"/api/games?limit=0",
	} {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(http.MethodGet, requestURL, nil)

		params, ok := decodeGamesListParams(context)
		if !ok {
			t.Fatalf("decodeGamesListParams(%q) returned ok=false", requestURL)
		}
		if params.Limit != 24 {
			t.Fatalf("decodeGamesListParams(%q) limit = %d, want 24", requestURL, params.Limit)
		}
	}
}
