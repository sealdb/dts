package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct{}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check 健康检查
// @Summary 健康检查
// @Description 检查服务健康状态
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

// HealthResponse 健康响应
type HealthResponse struct {
	Status string `json:"status"`
}
