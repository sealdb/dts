package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pg/dts/internal/logger"
	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
	"github.com/pg/dts/internal/service"
)

// TaskHandler handles tasks (according to the new API specification)
type TaskHandler struct {
	service *service.MigrationService
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(svc *service.MigrationService) *TaskHandler {
	return &TaskHandler{service: svc}
}

// CreateTaskRequest represents a create task request
type CreateTaskRequest struct {
	TaskID      string       `json:"task_id" binding:"required"`
	DatabaseType string      `json:"database_type" binding:"required"` // postgresql, mysql, etc.
	Source      DBConnection `json:"source" binding:"required"`
	Dest        DBConnection `json:"dest" binding:"required"`
	Tables      []string     `json:"tables,omitempty"` // Optional, if not specified, sync all tables
}

// DBConnection represents database connection information
type DBConnection struct {
	Domin    string `json:"domin" binding:"required"` // Note: API specification uses "domin" instead of "domain"
	Port     string `json:"port" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Database string `json:"database,omitempty"` // Optional, defaults to username
}

// CreateTaskResponse represents a create task response
type CreateTaskResponse struct {
	State   string `json:"state"`   // OK, ERROR
	Message string `json:"message"` // Error description
}

// CreateTask starts a data synchronization task
// POST /dts/api/tasks
func (h *TaskHandler) CreateTask(c *gin.Context) {
	log := logger.GetLogger()

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithError(err).Warn("Failed to bind request JSON")
		c.JSON(http.StatusBadRequest, CreateTaskResponse{
			State:   "ERROR",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	log.WithFields(map[string]interface{}{
		"task_id": req.TaskID,
		"source":  fmt.Sprintf("%s:%s", req.Source.Domin, req.Source.Port),
		"dest":    fmt.Sprintf("%s:%s", req.Dest.Domin, req.Dest.Port),
		"tables":  len(req.Tables),
	}).Info("Creating migration task")

	// Convert request format to internal format
	sourceDB := model.DBConfig{
		Host:     req.Source.Domin, // Note: API specification uses "domin" instead of "domain"
		Port:     parseInt(req.Source.Port, 5432),
		User:     req.Source.Username,
		Password: req.Source.Password,
		DBName:   getStringOrDefault(req.Source.Database, "postgres"),
		SSLMode:  "disable",
	}

	targetDB := model.DBConfig{
		Host:     req.Dest.Domin, // Note: API specification uses "domin" instead of "domain"
		Port:     parseInt(req.Dest.Port, 5432),
		User:     req.Dest.Username,
		Password: req.Dest.Password,
		DBName:   getStringOrDefault(req.Dest.Database, "postgres"),
		SSLMode:  "disable",
	}

	// If no tables specified, get all tables from source database
	tables := req.Tables
	if len(tables) == 0 {
		log.Info("No tables specified, fetching all tables from source database")
		// Get all tables from source database
		sourceRepo, err := repository.NewSourceRepository(sourceDB.DSN())
		if err != nil {
			log.WithError(err).Error("Failed to connect to source database")
			c.JSON(http.StatusInternalServerError, CreateTaskResponse{
				State:   "ERROR",
				Message: "Failed to connect to source database: " + err.Error(),
			})
			return
		}
		defer sourceRepo.Close()

		// Get all tables (from public schema)
		allTables, err := sourceRepo.GetAllTables("public")
		if err != nil {
			log.WithError(err).Error("Failed to get tables from source database")
			c.JSON(http.StatusInternalServerError, CreateTaskResponse{
				State:   "ERROR",
				Message: "Failed to get tables from source database: " + err.Error(),
			})
			return
		}

		if len(allTables) == 0 {
			log.Warn("No tables found in source database")
			c.JSON(http.StatusBadRequest, CreateTaskResponse{
				State:   "ERROR",
				Message: "No tables found in source database. Please specify tables manually.",
			})
			return
		}

		log.WithField("table_count", len(allTables)).Info("Found tables in source database")
		tables = allTables
	}

	// Create task
	createReq := &service.CreateTaskRequest{
		DatabaseType: req.DatabaseType,
		SourceDB:     sourceDB,
		TargetDB:     targetDB,
		Tables:       tables,
		TableSuffix:  "", // Default no suffix
	}

	task, err := h.service.CreateTaskWithID(req.TaskID, createReq)
	if err != nil {
		log.WithError(err).Error("Failed to create task")
		c.JSON(http.StatusInternalServerError, CreateTaskResponse{
			State:   "ERROR",
			Message: "Failed to create task: " + err.Error(),
		})
		return
	}

	log.WithField("task_id", task.ID).Info("Task created successfully")
	// Note: Task is created in Init state, need to call /:task_id/start to begin
	c.JSON(http.StatusOK, CreateTaskResponse{
		State:   "OK",
		Message: "Task created successfully. Call /:task_id/start to begin migration.",
	})
}

// GetTaskStatusResponse represents a get task status response
type GetTaskStatusResponse struct {
	State    string `json:"state"`    // OK, ERROR
	Message  string `json:"message"`  // Error description
	Stage    string `json:"stage"`    // none, syncing, waiting, switching, finished
	Duration int64  `json:"duration"` // Time from switchover start to completion, in ms, -1 means meaningless
	Delay    int64  `json:"delay"`    // Synchronization delay, in ms, -1 means meaningless
}

// GetTaskStatus queries synchronization task status
// GET /dts/api/tasks/{task_id}
func (h *TaskHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")

	task, err := h.service.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, GetTaskStatusResponse{
			State:    "ERROR",
			Message:  "Task not found: " + err.Error(),
			Stage:    "none",
			Duration: -1,
			Delay:    -1,
		})
		return
	}

	// Map internal state to API specification state
	stage := mapStateToStage(task.State)

	// Calculate duration (time from switchover start to completion)
	duration := int64(-1)
	if task.StartedAt != nil && task.CompletedAt != nil {
		// If task is completed, calculate total time
		if task.State == string(model.StateCompleted) {
			duration = task.CompletedAt.Sub(*task.StartedAt).Milliseconds()
		}
	}

	// Calculate delay (synchronization delay)
	// TODO: Implement actual delay calculation (needs to get from WAL replication)
	delay := int64(-1)
	if stage == "syncing" || stage == "waiting" || stage == "switching" {
		// Need to get delay from WAL replication status
		// Temporarily return -1
		delay = -1
	}

	c.JSON(http.StatusOK, GetTaskStatusResponse{
		State:    "OK",
		Message:  "",
		Stage:    stage,
		Duration: duration,
		Delay:    delay,
	})
}

// SwitchTaskResponse represents a switch task response
type SwitchTaskResponse struct {
	State   string `json:"state"`   // OK, ERROR
	Message string `json:"message"` // Error description
}

// SwitchTask performs switchover
// POST /dts/api/tasks/{task_id}/switch
func (h *TaskHandler) SwitchTask(c *gin.Context) {
	taskID := c.Param("task_id")

	task, err := h.service.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, SwitchTaskResponse{
			State:   "ERROR",
			Message: "Task not found: " + err.Error(),
		})
		return
	}

	// Switchover operation: only allowed in Waiting state
	// This transitions from Waiting -> Validating -> Completed
	if task.State != string(model.StateWaiting) {
		c.JSON(http.StatusBadRequest, SwitchTaskResponse{
			State:   "ERROR",
			Message: fmt.Sprintf("Task must be in 'waiting' state to perform switchover. Current state: %s", task.State),
		})
		return
	}

	// Trigger switchover flow (Waiting -> Validating)
	if err := h.service.TriggerSwitchover(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusInternalServerError, SwitchTaskResponse{
			State:   "ERROR",
			Message: "Failed to trigger switchover: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SwitchTaskResponse{
		State:   "OK",
		Message: "Switchover triggered successfully",
	})
}

// DeleteTaskResponse represents a delete task response
type DeleteTaskResponse struct {
	State   string `json:"state"`   // OK, ERROR
	Message string `json:"message"` // Error description
}

// StartTask starts a task
// POST /dts/api/tasks/{task_id}/start
func (h *TaskHandler) StartTask(c *gin.Context) {
	taskID := c.Param("task_id")

	if err := h.service.StartTask(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusInternalServerError, SwitchTaskResponse{
			State:   "ERROR",
			Message: "Failed to start task: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SwitchTaskResponse{
		State:   "OK",
		Message: "Task started successfully",
	})
}

// StopTask stops a task (task remains, just stops running)
// POST /dts/api/tasks/{task_id}/stop
func (h *TaskHandler) StopTask(c *gin.Context) {
	taskID := c.Param("task_id")

	if err := h.service.StopTask(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, SwitchTaskResponse{
			State:   "ERROR",
			Message: "Failed to stop task: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SwitchTaskResponse{
		State:   "OK",
		Message: "Task stopped successfully",
	})
}

// PauseTask pauses a task
// POST /dts/api/tasks/{task_id}/pause
func (h *TaskHandler) PauseTask(c *gin.Context) {
	taskID := c.Param("task_id")

	if err := h.service.PauseTask(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, SwitchTaskResponse{
			State:   "ERROR",
			Message: "Failed to pause task: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SwitchTaskResponse{
		State:   "OK",
		Message: "Task paused successfully",
	})
}

// ResumeTask resumes a task
// POST /dts/api/tasks/{task_id}/resume
func (h *TaskHandler) ResumeTask(c *gin.Context) {
	taskID := c.Param("task_id")

	if err := h.service.ResumeTask(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusInternalServerError, SwitchTaskResponse{
			State:   "ERROR",
			Message: "Failed to resume task: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SwitchTaskResponse{
		State:   "OK",
		Message: "Task resumed successfully",
	})
}

// DeleteTask deletes a task
// DELETE /dts/api/tasks/{task_id}
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	taskID := c.Param("task_id")

	// Delete task (this will also cancel it if running)
	if err := h.service.DeleteTask(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, DeleteTaskResponse{
			State:   "ERROR",
			Message: "Failed to delete task: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, DeleteTaskResponse{
		State:   "OK",
		Message: "Task deleted successfully",
	})
}

// mapStateToStage maps internal state to API specification state
func mapStateToStage(state string) string {
	switch state {
	case string(model.StateInit):
		return "none"
	case string(model.StateConnect), string(model.StateCreateTables), string(model.StateFullSync):
		return "syncing"
	case string(model.StateIncSync):
		return "syncing"
	case string(model.StateWaiting):
		return "waiting"
	case string(model.StateValidating):
		return "switching"
	case string(model.StateCompleted):
		return "finished"
	case string(model.StateFailed):
		return "none"
	case string(model.StatePaused):
		return "waiting"
	case string(model.StateDeleted):
		return "none"
	default:
		return "none"
	}
}

// parseInt parses string to integer, returns default value on failure
func parseInt(s string, defaultValue int) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

// getStringOrDefault gets string, returns default value if empty
func getStringOrDefault(s, defaultValue string) string {
	if s == "" {
		return defaultValue
	}
	return s
}
