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

// TaskHandler 任务处理器（按照新的 API 规范）
type TaskHandler struct {
	service *service.MigrationService
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(svc *service.MigrationService) *TaskHandler {
	return &TaskHandler{service: svc}
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	TaskID string        `json:"task_id" binding:"required"`
	Source DBConnection  `json:"source" binding:"required"`
	Dest   DBConnection  `json:"dest" binding:"required"`
	Tables []string      `json:"tables,omitempty"` // 可选，如果不指定则同步所有表
}

// DBConnection 数据库连接信息
type DBConnection struct {
	Domin    string `json:"domin" binding:"required"` // 注意：API 规范中使用 "domin" 而不是 "domain"
	Port     string `json:"port" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Database string `json:"database,omitempty"` // 可选，默认使用 username
}

// CreateTaskResponse 创建任务响应
type CreateTaskResponse struct {
	State   string `json:"state"`   // OK, ERROR
	Message string `json:"message"` // 错误描述
}

// CreateTask 启动数据同步任务
// POST /rdscheduler/api/tasks
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

	// 转换请求格式为内部格式
    sourceDB := model.DBConfig{
		Host:     req.Source.Domin, // 注意：API 规范中使用 "domin" 而不是 "domain"
		Port:     parseInt(req.Source.Port, 5432),
		User:     req.Source.Username,
		Password: req.Source.Password,
        DBName:   getStringOrDefault(req.Source.Database, "postgres"),
		SSLMode:  "disable",
	}

    targetDB := model.DBConfig{
		Host:     req.Dest.Domin, // 注意：API 规范中使用 "domin" 而不是 "domain"
		Port:     parseInt(req.Dest.Port, 5432),
		User:     req.Dest.Username,
		Password: req.Dest.Password,
        DBName:   getStringOrDefault(req.Dest.Database, "postgres"),
		SSLMode:  "disable",
	}

	// 如果没有指定表，则从源库获取所有表
	tables := req.Tables
	if len(tables) == 0 {
		log.Info("No tables specified, fetching all tables from source database")
		// 从源库获取所有表
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

		// 获取所有表（从 public schema）
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

	// 创建任务
	createReq := &service.CreateTaskRequest{
		SourceDB:    sourceDB,
		TargetDB:    targetDB,
		Tables:      tables,
		TableSuffix: "", // 默认无后缀
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

	log.WithField("task_id", task.ID).Info("Task created successfully, starting task")

	// 自动启动任务
	if err := h.service.StartTask(c.Request.Context(), task.ID); err != nil {
		log.WithError(err).Error("Failed to start task")
		c.JSON(http.StatusInternalServerError, CreateTaskResponse{
			State:   "ERROR",
			Message: "Failed to start task: " + err.Error(),
		})
		return
	}

	log.WithField("task_id", task.ID).Info("Task started successfully")
	c.JSON(http.StatusOK, CreateTaskResponse{
		State:   "OK",
		Message: "Task created and started successfully",
	})
}

// GetTaskStatusResponse 查询任务状态响应
type GetTaskStatusResponse struct {
	State    string `json:"state"`    // OK, ERROR
	Message  string `json:"message"`  // 错误描述
	Stage    string `json:"stage"`    // none, syncing, waiting, switching, finished
	Duration int64  `json:"duration"` // 从切流开始到完成的时间，单位 ms，-1 表示无意义
	Delay    int64  `json:"delay"`    // 同步延迟，单位 ms，-1 表示无意义
}

// GetTaskStatus 查询同步任务状态
// GET /rdscheduler/api/tasks/{task_id}
func (h *TaskHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")

	task, err := h.service.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, GetTaskStatusResponse{
			State:   "ERROR",
			Message: "Task not found: " + err.Error(),
			Stage:   "none",
			Duration: -1,
			Delay:    -1,
		})
		return
	}

	// 映射内部状态到 API 规范的状态
	stage := mapStateToStage(task.State)

	// 计算 duration（从切流开始到完成的时间）
	duration := int64(-1)
	if task.StartedAt != nil && task.CompletedAt != nil {
		// 如果任务已完成，计算总耗时
		if task.State == string(model.StateCompleted) {
			duration = task.CompletedAt.Sub(*task.StartedAt).Milliseconds()
		}
	}

	// 计算 delay（同步延迟）
	// TODO: 实现实际的延迟计算（需要从 WAL 复制中获取）
	delay := int64(-1)
	if stage == "syncing" || stage == "waiting" || stage == "switching" {
		// 这里需要从 WAL 复制状态中获取延迟
		// 暂时返回 -1
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

// SwitchTaskResponse 切流响应
type SwitchTaskResponse struct {
	State   string `json:"state"`   // OK, ERROR
	Message string `json:"message"` // 错误描述
}

// SwitchTask 切流
// POST /rdscheduler/api/tasks/{task_id}/switch
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

	// 切流操作：停止源库写入，验证数据，恢复写入
	// 这对应状态机中的 StoppingWrites -> Validating -> Finalizing
	// 如果任务在 syncing 状态，需要先切换到 stopping_writes
	if task.State == string(model.StateSyncingWAL) {
		// 触发切流流程
		if err := h.service.TriggerSwitchover(c.Request.Context(), taskID); err != nil {
			c.JSON(http.StatusInternalServerError, SwitchTaskResponse{
				State:   "ERROR",
				Message: "Failed to trigger switchover: " + err.Error(),
			})
			return
		}
	} else if task.State == string(model.StateStoppingWrites) ||
		task.State == string(model.StateValidating) ||
		task.State == string(model.StateFinalizing) {
		// 已经在切流流程中
		c.JSON(http.StatusOK, SwitchTaskResponse{
			State:   "OK",
			Message: "Switchover is already in progress",
		})
		return
	} else {
		c.JSON(http.StatusBadRequest, SwitchTaskResponse{
			State:   "ERROR",
			Message: "Task is not in a state that allows switchover. Current state: " + task.State,
		})
		return
	}

	c.JSON(http.StatusOK, SwitchTaskResponse{
		State:   "OK",
		Message: "Switchover triggered successfully",
	})
}

// DeleteTaskResponse 删除任务响应
type DeleteTaskResponse struct {
	State   string `json:"state"`   // OK, ERROR
	Message string `json:"message"` // 错误描述
}

// DeleteTask 结束任务
// DELETE /rdscheduler/api/tasks/{task_id}
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	taskID := c.Param("task_id")

	// 取消任务
	if err := h.service.CancelTask(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, DeleteTaskResponse{
			State:   "ERROR",
			Message: "Failed to cancel task: " + err.Error(),
		})
		return
	}

	// 删除任务
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

// mapStateToStage 映射内部状态到 API 规范的状态
func mapStateToStage(state string) string {
	switch state {
	case string(model.StateInit), string(model.StateCreatingTables), string(model.StateMigratingData):
		return "syncing"
	case string(model.StateSyncingWAL):
		return "syncing"
	case string(model.StateStoppingWrites):
		return "switching"
	case string(model.StateValidating), string(model.StateFinalizing):
		return "switching"
	case string(model.StateCompleted):
		return "finished"
	case string(model.StateFailed):
		return "none"
	case string(model.StatePaused):
		return "waiting"
	default:
		return "none"
	}
}

// parseInt 解析字符串为整数，失败返回默认值
func parseInt(s string, defaultValue int) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

// getStringOrDefault 获取字符串，如果为空则返回默认值
func getStringOrDefault(s, defaultValue string) string {
	if s == "" {
		return defaultValue
	}
	return s
}

