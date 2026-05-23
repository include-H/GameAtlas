package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type PendingIssuesHandler struct {
	service *services.PendingIssuesService
}

func NewPendingIssuesHandler(service *services.PendingIssuesService) *PendingIssuesHandler {
	return &PendingIssuesHandler{service: service}
}

func (h *PendingIssuesHandler) List(c *gin.Context) {
	writeJSONSuccess(c, http.StatusOK, toPendingIssueCatalogResponse(h.service.Catalog()))
}
