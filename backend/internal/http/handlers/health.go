package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// 2026-05-09: HealthHandler 直接依赖 *sqlx.DB 而非 repository 层。Health check 是基础设施端点，仅需 Ping 数据库连接，穿透分层是合理的设计选择。
type HealthHandler struct {
	db *sqlx.DB
}

func NewHealthHandler(db *sqlx.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Get(c *gin.Context) {
	if err := h.db.PingContext(c.Request.Context()); err != nil {
		writeJSONError(c, http.StatusServiceUnavailable, "数据库不可用")
		return
	}

	writeJSONSuccess(c, http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
