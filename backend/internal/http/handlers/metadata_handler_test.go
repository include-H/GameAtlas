package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
)

func TestMetadataHandlerGetPublisherReturnsPublisherAndGames(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	publisherID := insertGamesHandlerMetadataItem(t, db, "publishers", "Atlus", "atlus")
	gameID := insertGamesHandlerTestGame(t, db, "publisher-game", "Publisher Game", domain.GameVisibilityPublic, "2024-01-01")
	linkGamesHandlerGameRelation(t, db, "game_publishers", "publisher_id", gameID, publisherID)

	handler := NewMetadataHandler(
		services.NewMetadataService(repositories.NewMetadataRepository(db)),
		services.MetadataResource{Type: domain.MetadataPublishers},
	)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	publisherIDValue := strconv.FormatInt(publisherID, 10)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/publishers/"+publisherIDValue, nil)
	context.Params = gin.Params{{Key: "id", Value: publisherIDValue}}
	handler.Get(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Publisher struct {
				ID   int64  `json:"id"`
				Name string `json:"name"`
			} `json:"publisher"`
			Games []struct {
				PublicID string `json:"public_id"`
			} `json:"games"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("success = false, body=%s", recorder.Body.String())
	}
	if response.Data.Publisher.ID != publisherID || response.Data.Publisher.Name != "Atlus" {
		t.Fatalf("publisher = %+v, want Atlus", response.Data.Publisher)
	}
	if len(response.Data.Games) != 1 || response.Data.Games[0].PublicID != "publisher-game" {
		t.Fatalf("games = %+v, want publisher-game", response.Data.Games)
	}
}
