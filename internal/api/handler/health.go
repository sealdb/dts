package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health checks
type HealthHandler struct{}

// NewHealthHandler creates a new health check handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check performs health check
// @Summary Health check
// @Description Check service health status
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /api/v1/health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "ok",
	})
}

// HealthResponse represents a health response
type HealthResponse struct {
	Status string `json:"status"`
}
