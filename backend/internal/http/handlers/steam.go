package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type SteamHandler struct {
	service *services.SteamService
}

func NewSteamHandler(service *services.SteamService) *SteamHandler {
	return &SteamHandler{service: service}
}

func (h *SteamHandler) Search(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	query := strings.TrimSpace(c.Query("q"))
	proxy, ok := parseSteamProxyQuery(c)
	if !ok {
		return
	}
	if query == "" {
		// 2026-05-09: 统一为中文错误信息
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "搜索关键词为必填项"})
		return
	}
	results, err := h.service.Search(query, proxy)
	if err != nil {
		// 2026-05-09: upstream Steam errors surface as 502 Bad Gateway rather than
		// 500 because the failure originates from an external dependency, not from
		// this server. Use writeJSONError to preserve the status code.
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusBadGateway, "Steam 搜索失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": toSteamSearchResultResponses(results)})
}

func (h *SteamHandler) Preview(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	appID, ok := parseIDParam(c, "appId")
	if !ok {
		return
	}
	proxy, ok := parseSteamProxyQuery(c)
	if !ok {
		return
	}
	preview, err := h.service.PreviewAssets(appID, proxy)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的 Steam 请求")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": toSteamAssetsPreviewResponse(preview)})
}

func (h *SteamHandler) Proxy(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	rawURL := strings.TrimSpace(c.Query("url"))
	proxy, ok := parseSteamProxyQuery(c)
	if !ok {
		return
	}
	if rawURL == "" {
		// 2026-05-09: 统一为中文错误信息
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "URL 为必填项"})
		return
	}

	contentType, payload, err := h.service.ProxyAsset(rawURL, proxy)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的 Steam 代理请求")
		return
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Access-Control-Expose-Headers", "Content-Type, Content-Length")
	c.Data(http.StatusOK, contentType, payload)
}

func parseSteamProxyQuery(c *gin.Context) (string, bool) {
	proxy := strings.TrimSpace(c.Query("proxy"))
	if proxy == "" {
		return "", true
	}

	// 2026-04-09: `proxy` is a transport override, not a best-effort hint.
	// Invalid query values must fail here instead of silently falling back to
	// the service default/environment proxy path.
	parsed, err := url.Parse(proxy)
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		// 2026-05-09: 统一为中文错误信息
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "需要有效的 Steam 代理地址"})
		return "", false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		// 2026-05-09: 统一为中文错误信息
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "需要有效的 Steam 代理地址"})
		return "", false
	}

	return parsed.String(), true
}
