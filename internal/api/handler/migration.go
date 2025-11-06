package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pg/dts/internal/service"
)

// MigrationHandler 迁移任务处理器
type MigrationHandler struct {
	service *service.MigrationService
}

// NewMigrationHandler 创建迁移任务处理器
func NewMigrationHandler(svc *service.MigrationService) *MigrationHandler {
	return &MigrationHandler{service: svc}
}

// CreateTask 创建迁移任务
// @Summary 创建迁移任务
// @Description 创建新的数据库迁移任务
// @Tags migrations
// @Accept json
// @Produce json
// @Param task body service.CreateTaskRequest true "迁移任务信息"
// @Success 201 {object} model.MigrationTask
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/migrations [post]
func (h *MigrationHandler) CreateTask(c *gin.Context) {
	var req service.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
		return
	}

	task, err := h.service.CreateTask(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed to create task",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// GetTask 获取任务详情
// @Summary 获取任务详情
// @Description 根据ID获取迁移任务详情
// @Tags migrations
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} model.MigrationTask
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/migrations/{id} [get]
func (h *MigrationHandler) GetTask(c *gin.Context) {
	id := c.Param("id")

	task, err := h.service.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "task not found",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, task)
}

// ListTasks 获取任务列表
// @Summary 获取任务列表
// @Description 获取迁移任务列表
// @Tags migrations
// @Accept json
// @Produce json
// @Param limit query int false "限制数量" default(10)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {array} model.MigrationTask
// @Router /api/v1/migrations [get]
func (h *MigrationHandler) ListTasks(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	tasks, err := h.service.ListTasks(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed to list tasks",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// StartTask 启动任务
// @Summary 启动任务
// @Description 启动迁移任务
// @Tags migrations
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/migrations/{id}/start [post]
func (h *MigrationHandler) StartTask(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.StartTask(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed to start task",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "task started successfully",
	})
}

// PauseTask 暂停任务
// @Summary 暂停任务
// @Description 暂停迁移任务
// @Tags migrations
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} SuccessResponse
// @Router /api/v1/migrations/{id}/pause [post]
func (h *MigrationHandler) PauseTask(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.PauseTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed to pause task",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "task paused successfully",
	})
}

// ResumeTask 恢复任务
// @Summary 恢复任务
// @Description 恢复暂停的迁移任务
// @Tags migrations
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} SuccessResponse
// @Router /api/v1/migrations/{id}/resume [post]
func (h *MigrationHandler) ResumeTask(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.ResumeTask(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed to resume task",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "task resumed successfully",
	})
}

// CancelTask 取消任务
// @Summary 取消任务
// @Description 取消迁移任务
// @Tags migrations
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} SuccessResponse
// @Router /api/v1/migrations/{id}/cancel [post]
func (h *MigrationHandler) CancelTask(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.CancelTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed to cancel task",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "task cancelled successfully",
	})
}

// GetTaskStatus 获取任务状态
// @Summary 获取任务状态
// @Description 获取迁移任务当前状态
// @Tags migrations
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} StatusResponse
// @Router /api/v1/migrations/{id}/status [get]
func (h *MigrationHandler) GetTaskStatus(c *gin.Context) {
	id := c.Param("id")

	task, err := h.service.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "task not found",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, StatusResponse{
		ID:       task.ID,
		State:    task.State,
		Progress: task.Progress,
		Error:    task.ErrorMessage,
	})
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Message string `json:"message"`
}

// StatusResponse 状态响应
type StatusResponse struct {
	ID       string `json:"id"`
	State    string `json:"state"`
	Progress int    `json:"progress"`
	Error    string `json:"error,omitempty"`
}
