package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pg/dts/internal/service"
)

// MigrationHandler handles migration tasks
type MigrationHandler struct {
	service *service.MigrationService
}

// NewMigrationHandler creates a new migration task handler
func NewMigrationHandler(svc *service.MigrationService) *MigrationHandler {
	return &MigrationHandler{service: svc}
}

// CreateTask creates a migration task
// @Summary Create migration task
// @Description Create a new database migration task
// @Tags migrations
// @Accept json
// @Produce json
// @Param task body service.CreateTaskRequest true "Migration task information"
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

// GetTask gets task details
// @Summary Get task details
// @Description Get migration task details by ID
// @Tags migrations
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
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

// ListTasks gets task list
// @Summary Get task list
// @Description Get migration task list
// @Tags migrations
// @Accept json
// @Produce json
// @Param limit query int false "Limit count" default(10)
// @Param offset query int false "Offset" default(0)
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

// StartTask starts a task
// @Summary Start task
// @Description Start migration task
// @Tags migrations
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
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

// PauseTask pauses a task
// @Summary Pause task
// @Description Pause migration task
// @Tags migrations
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
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

// ResumeTask resumes a task
// @Summary Resume task
// @Description Resume paused migration task
// @Tags migrations
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
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

// CancelTask cancels a task
// @Summary Cancel task
// @Description Cancel migration task
// @Tags migrations
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
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

// GetTaskStatus gets task status
// @Summary Get task status
// @Description Get current status of migration task
// @Tags migrations
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
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

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// StatusResponse represents a status response
type StatusResponse struct {
	ID       string `json:"id"`
	State    string `json:"state"`
	Progress int    `json:"progress"`
	Error    string `json:"error,omitempty"`
}
