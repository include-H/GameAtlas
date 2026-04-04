package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type WikiHandler struct {
	service *services.WikiService
}

func NewWikiHandler(service *services.WikiService) *WikiHandler {
	return &WikiHandler{service: service}
}

func (h *WikiHandler) Get(c *gin.Context) {
	gameID, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}

	document, err := h.service.Get(gameID, isAdminRequest(c))
	if err != nil {
		writeServiceError(c, err, "invalid wiki payload")
		return
	}

	writeJSONSuccess(c, http.StatusOK, wikiDocumentResponse{
		GameID:       document.GameID,
		Title:        document.Title,
		Content:      document.Content,
		UpdatedAt:    document.UpdatedAt,
		HistoryCount: document.HistoryCount,
	})
}

func (h *WikiHandler) Update(c *gin.Context) {
	gameID, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}
	if !requireAdmin(c) {
		return
	}

	var request wikiWriteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeJSONError(c, http.StatusBadRequest, "invalid wiki payload")
		return
	}

	input := request.toInput()
	document, err := h.service.Update(gameID, input)
	if err != nil {
		writeServiceError(c, err, "invalid wiki payload")
		return
	}

	writeJSONSuccess(c, http.StatusOK, wikiDocumentResponse{
		GameID:       document.GameID,
		Title:        document.Title,
		Content:      document.Content,
		UpdatedAt:    document.UpdatedAt,
		HistoryCount: document.HistoryCount,
	})
}

func (h *WikiHandler) History(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	gameID, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}

	items, err := h.service.History(gameID, isAdminRequest(c))
	if err != nil {
		writeServiceError(c, err, "invalid wiki payload")
		return
	}

	response := make([]wikiHistoryItemResponse, 0, len(items))
	for _, item := range items {
		response = append(response, wikiHistoryItemResponse{
			ID:            item.ID,
			GameID:        item.GameID,
			Content:       item.Content,
			ChangeSummary: item.ChangeSummary,
			CreatedAt:     item.CreatedAt,
		})
	}

	writeJSONSuccess(c, http.StatusOK, response)
}
