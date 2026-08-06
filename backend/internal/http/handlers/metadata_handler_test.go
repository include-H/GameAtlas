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

func TestMetadataHandlerListReturnsPageEnvelope(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	atlusID := insertGamesHandlerMetadataItem(t, db, "publishers", "Atlus", "atlus")
	segaID := insertGamesHandlerMetadataItem(t, db, "publishers", "SEGA", "sega")
	insertGamesHandlerMetadataItem(t, db, "publishers", "Unused", "unused")
	atlusGame := insertGamesHandlerTestGame(t, db, "publisher-atlus", "Atlus Game", domain.GameVisibilityPublic, "2024-01-01")
	segaGame := insertGamesHandlerTestGame(t, db, "publisher-sega", "SEGA Game", domain.GameVisibilityPublic, "2024-01-02")
	linkGamesHandlerGameRelation(t, db, "game_publishers", "publisher_id", atlusGame, atlusID)
	linkGamesHandlerGameRelation(t, db, "game_publishers", "publisher_id", segaGame, segaID)

	handler := NewMetadataHandler(
		services.NewMetadataService(repositories.NewMetadataRepository(db)),
		services.MetadataResource{Type: domain.MetadataPublishers},
	)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/publishers?page=1&limit=2&sort=name", nil)
	handler.List(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Success    bool `json:"success"`
		Data       []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
		Pagination struct {
			Page       int `json:"page"`
			Limit      int `json:"limit"`
			Total      int `json:"total"`
			TotalPages int `json:"totalPages"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || len(response.Data) != 2 {
		t.Fatalf("response = %+v, want two publishers", response)
	}
	if response.Pagination.Total != 2 || response.Pagination.TotalPages != 1 || response.Pagination.Limit != 2 {
		t.Fatalf("pagination = %+v, want total 2, pages 1, limit 2", response.Pagination)
	}
}
