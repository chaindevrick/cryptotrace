package http

import (
	"backend/internal/domain"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type ForensicsHandler struct {
	analyzer domain.AnalyzerUsecase
	tracer   domain.TracerUsecase
	graph    domain.GraphUsecase
}

func NewForensicsHandler(a domain.AnalyzerUsecase, t domain.TracerUsecase, g domain.GraphUsecase) *ForensicsHandler {
	return &ForensicsHandler{analyzer: a, tracer: t, graph: g}
}

func (h *ForensicsHandler) Analyze(c *gin.Context) {
	var req domain.RequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format"})
		return
	}

	address := strings.ToLower(strings.TrimSpace(req.Address))

	if len(address) != 42 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BROAD 模式只支援錢包地址 (42字元)。欲追蹤交易哈希，請點擊 FLOW 模式。"})
		return
	}

	count, err := h.analyzer.Analyze(c.Request.Context(), address)
	if err != nil {
		if count == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Analysis failed"})
		return
	}

	if count == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "No data"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "count": count})
}

func (h *ForensicsHandler) Trace(c *gin.Context) {
	var req domain.RequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format"})
		return
	}

	address := strings.ToLower(strings.TrimSpace(req.Address))

	err := h.tracer.Trace(c.Request.Context(), address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *ForensicsHandler) GetGraph(c *gin.Context) {
	address := strings.ToLower(strings.TrimSpace(c.Param("address")))

	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address is required"})
		return
	}

	elements, err := h.graph.GetGraph(c.Request.Context(), address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Graph failed"})
		return
	}
	c.JSON(http.StatusOK, elements)
}
