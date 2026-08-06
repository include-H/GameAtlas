package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

func TestSteamGridDBHandlerAvailableWithoutAPIKey(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/steamgriddb/available", nil)

	handler := NewSteamGridDBHandler(services.NewSteamGridDBService("", ""))
	handler.Available(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response struct {
		Success bool `json:"success"`
		Data    bool `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data {
		t.Fatalf("response = %+v, want success=true data=false", response)
	}
}

func TestSteamGridDBHandlerRequiresAdmin(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamGridDBHandler(services.NewSteamGridDBService("", ""))
	tests := []struct {
		name string
		call func(c *gin.Context)
	}{
		{name: "search", call: handler.Search},
		{name: "grids by game id", call: handler.GetGridsByGameID},
		{name: "heroes by game id", call: handler.GetHeroesByGameID},
		{name: "logos by game id", call: handler.GetLogosByGameID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodGet, "/api/steamgriddb/search", nil)

			tt.call(context)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), `"error":"需要管理员登录"`) {
				t.Fatalf("body = %s, want admin login error", recorder.Body.String())
			}
		})
	}
}

func TestSteamGridDBHandlerReturnsServiceUnavailableWithoutAPIKey(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamGridDBHandler(services.NewSteamGridDBService("", ""))
	tests := []struct {
		name string
		call func(c *gin.Context)
	}{
		{name: "search", call: handler.Search},
		{name: "grids by game id", call: handler.GetGridsByGameID},
		{name: "heroes by game id", call: handler.GetHeroesByGameID},
		{name: "logos by game id", call: handler.GetLogosByGameID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodGet, "/api/steamgriddb/search?q=portal", nil)
			context.Set("is_admin", true)

			tt.call(context)

			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
			}
		})
	}
}

func TestSteamGridDBResponseMappers(t *testing.T) {
	game := services.SteamGridDBGame{
		ID:          7,
		Name:        "Portal",
		ReleaseDate: 1234567890,
		Types:       []string{"game"},
		Verified:    true,
	}
	image := services.SteamGridDBImage{
		ID:       9,
		Score:    5,
		Style:    "alternate",
		Notes:    "alt",
		Language: "en",
		URL:      "https://cdn.example.com/grid.png",
		Thumb:    "https://cdn.example.com/thumb.png",
	}

	games := toSteamGridDBGameResponses([]services.SteamGridDBGame{game})
	if len(games) != 1 || games[0].ID != 7 || games[0].Name != "Portal" || !games[0].Verified {
		t.Fatalf("games = %+v, want mapped Portal response", games)
	}
	if games[0].Types[0] != "game" || games[0].ReleaseDate != 1234567890 {
		t.Fatalf("games[0] = %+v, want full game mapping", games[0])
	}

	images := toSteamGridDBImageResponses([]services.SteamGridDBImage{image})
	if len(images) != 1 || images[0].ID != 9 || images[0].URL != "https://cdn.example.com/grid.png" || images[0].Thumb == "" {
		t.Fatalf("images = %+v, want mapped image response", images)
	}
}
