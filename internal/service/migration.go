package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pg/dts/internal/logger"
	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
	"github.com/pg/dts/internal/state"
	"gorm.io/gorm"
)

// MigrationService 迁移服务
type MigrationService struct {
	taskRepo    *repository.MigrationRepository
	db          *gorm.DB
	taskManager *TaskManager
}

// NewMigrationService 创建迁移服务
func NewMigrationService(db *gorm.DB) *MigrationService {
	return &MigrationService{
		taskRepo:    repository.NewMigrationRepository(db),
		db:          db,
		taskManager: NewTaskManager(),
	}
}

// GetTaskManager 获取任务管理器
func (s *MigrationService) GetTaskManager() *TaskManager {
	return s.taskManager
}

// CreateTaskWithID 使用指定ID创建迁移任务
func (s *MigrationService) CreateTaskWithID(id string, req *CreateTaskRequest) (*model.MigrationTask, error) {
	// 检查ID是否已存在
	if _, err := s.taskRepo.GetByID(id); err == nil {
		return nil, fmt.Errorf("task with id %s already exists", id)
	}

	// 序列化数据库配置
	sourceDBJSON, err := json.Marshal(req.SourceDB)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal source db config: %w", err)
	}

	targetDBJSON, err := json.Marshal(req.TargetDB)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal target db config: %w", err)
	}

	tablesJSON, err := json.Marshal(req.Tables)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tables: %w", err)
	}

	task := &model.MigrationTask{
		ID:           id,
		SourceDB:     string(sourceDBJSON),
		TargetDB:     string(targetDBJSON),
		Tables:       string(tablesJSON),
		TableSuffix:  req.TableSuffix,
		State:        model.StateInit.String(),
		Progress:     0,
		ErrorMessage: "",
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

// CreateTask 创建迁移任务（自动生成ID）
func (s *MigrationService) CreateTask(req *CreateTaskRequest) (*model.MigrationTask, error) {
	// 序列化数据库配置
	sourceDBJSON, err := json.Marshal(req.SourceDB)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal source db config: %w", err)
	}

	targetDBJSON, err := json.Marshal(req.TargetDB)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal target db config: %w", err)
	}

	tablesJSON, err := json.Marshal(req.Tables)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tables: %w", err)
	}

	task := &model.MigrationTask{
		SourceDB:     string(sourceDBJSON),
		TargetDB:     string(targetDBJSON),
		Tables:       string(tablesJSON),
		TableSuffix:  req.TableSuffix,
		State:        model.StateInit.String(),
		Progress:     0,
		ErrorMessage: "",
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

// GetTask 获取任务
func (s *MigrationService) GetTask(id string) (*model.MigrationTask, error) {
	return s.taskRepo.GetByID(id)
}

// ListTasks 获取任务列表
func (s *MigrationService) ListTasks(limit, offset int) ([]*model.MigrationTask, error) {
	return s.taskRepo.List(limit, offset)
}

// StartTask 启动任务
func (s *MigrationService) StartTask(ctx context.Context, id string) error {
	log := logger.GetLogger()
	log.WithField("task_id", id).Info("Starting migration task")

	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		log.WithError(err).Error("Failed to get task")
		return err
	}

	// 检查任务状态
	currentState := model.StateType(task.State)
	if currentState.IsTerminal() {
		err := fmt.Errorf("task is in terminal state: %s", currentState)
		log.WithError(err).Warn("Cannot start task")
		return err
	}

	// 检查任务是否已在运行
	if _, exists := s.taskManager.GetTask(id); exists {
		err := fmt.Errorf("task %s is already running", id)
		log.WithError(err).Warn("Task already running")
		return err
	}

	// 确保任务连接池已初始化
	if task.Connections == nil {
		task.Connections = make(map[string]interface{})
	}

	// 添加到任务管理器
	s.taskManager.AddTask(task)
	log.WithField("task_id", id).Info("Task added to task manager")

	// 创建状态机
	sm := state.NewStateMachine(task)

	// 执行状态机
	go func() {
		log := logger.GetLogger()
		log.WithField("task_id", id).Info("State machine goroutine started")
		// 确保任务完成后清理连接
		defer func() {
			log.WithField("task_id", id).Info("Cleaning up task")
			if err := s.taskManager.RemoveTask(id); err != nil {
				// 记录清理错误，但不影响任务状态
				log.WithError(err).Warn("Failed to cleanup task")
			}
		}()

		// 简单的重试机制配置
		maxRetries := 3
		baseDelayMs := 500

		for {
			// 获取当前状态
			currentState := sm.GetCurrentState()
			if currentState != nil {
				log.WithFields(map[string]interface{}{
					"task_id": id,
					"state":   currentState.Name(),
				}).Info("Executing state")
			}

			// 执行当前状态，带重试
			var execErr error
			for attempt := 0; attempt <= maxRetries; attempt++ {
				if attempt > 0 {
					log.WithFields(map[string]interface{}{
						"task_id": id,
						"attempt": attempt,
					}).Warn("Retrying state execution")
				}
				execErr = sm.Execute(ctx)
				if execErr == nil {
					break
				}
				if !isRetryable(execErr) || attempt == maxRetries {
					break
				}
				delay := baseDelayMs * (1 << attempt)
				// 简单 sleep（避免引入额外依赖）
				select {
				case <-ctx.Done():
					log.WithField("task_id", id).Warn("Context cancelled")
					s.taskRepo.UpdateState(task.ID, model.StateFailed, ctx.Err().Error())
					return
				case <-time.After(time.Duration(delay) * time.Millisecond):
				}
			}

			if execErr != nil {
				// 更新任务为失败状态
				log.WithError(execErr).WithField("task_id", id).Error("State execution failed")
				s.taskRepo.UpdateState(task.ID, model.StateFailed, execErr.Error())
				// 失败时清理连接
				task.CloseAllConnections()
				return
			}

			// 更新任务状态
			currentState = sm.GetCurrentState()
			if currentState != nil {
				newState := model.StateType(currentState.Name())
				log.WithFields(map[string]interface{}{
					"task_id": id,
					"new_state": newState.String(),
				}).Info("State transition completed")
				s.taskRepo.UpdateState(task.ID, newState, "")
				// 粗粒度进度：按状态推进
				s.taskRepo.UpdateProgress(task.ID, progressForState(newState))
			}

			// 检查是否到达终止状态
			if task.State == model.StateCompleted.String() || task.State == model.StateFailed.String() {
				log.WithFields(map[string]interface{}{
					"task_id": id,
					"final_state": task.State,
				}).Info("Task reached terminal state")
				// 任务完成，清理连接（defer 也会执行，但这里显式调用确保清理）
				task.CloseAllConnections()
				return
			}

			// 重新加载任务以获取最新状态
			task, _ = s.taskRepo.GetByID(id)
		}
	}()

	return nil
}

// isRetryable 简单判断是否可重试
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	// 可扩展更细粒度判断，此处简单基于错误信息
	msg := err.Error()
	retryHints := []string{"timeout", "temporarily", "connection refused", "deadlock detected"}
	for _, h := range retryHints {
		if strings.Contains(strings.ToLower(msg), h) {
			return true
		}
	}
	return false
}

// progressForState 为不同状态提供粗粒度进度
func progressForState(s model.StateType) int {
	switch s {
	case model.StateInit:
		return 5
	case model.StateCreatingTables:
		return 20
	case model.StateMigratingData:
		return 60
	case model.StateSyncingWAL:
		return 75
	case model.StateStoppingWrites:
		return 85
	case model.StateValidating:
		return 95
	case model.StateFinalizing:
		return 99
	case model.StateCompleted:
		return 100
	default:
		return 0
	}
}

// PauseTask 暂停任务
func (s *MigrationService) PauseTask(id string) error {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return err
	}

	currentState := model.StateType(task.State)
	if currentState.IsTerminal() {
		return fmt.Errorf("cannot pause task in terminal state: %s", currentState)
	}

	return s.taskRepo.UpdateState(id, model.StatePaused, "")
}

// ResumeTask 恢复任务
func (s *MigrationService) ResumeTask(ctx context.Context, id string) error {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return err
	}

	if task.State != model.StatePaused.String() {
		return fmt.Errorf("task is not paused")
	}

	// 恢复任务
	return s.StartTask(ctx, id)
}

// DeleteTask 删除任务
func (s *MigrationService) DeleteTask(id string) error {
	// 先取消任务（如果正在运行）
	_ = s.CancelTask(id)

	// 从任务管理器移除
	s.taskManager.RemoveTask(id)

	// 从数据库删除
	return s.taskRepo.Delete(id)
}

// TriggerSwitchover 触发切流
func (s *MigrationService) TriggerSwitchover(ctx context.Context, id string) error {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return err
	}

	// 如果任务在 syncing_wal 状态，切换到 stopping_writes
	if task.State == string(model.StateSyncingWAL) {
		// 更新状态为 stopping_writes
		return s.taskRepo.UpdateState(id, model.StateStoppingWrites, "")
	}

	return fmt.Errorf("task is not in a state that allows switchover: %s", task.State)
}

// CancelTask 取消任务
func (s *MigrationService) CancelTask(id string) error {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return err
	}

	currentState := model.StateType(task.State)
	if currentState.IsTerminal() {
		return fmt.Errorf("cannot cancel task in terminal state: %s", currentState)
	}

	return s.taskRepo.UpdateState(id, model.StateFailed, "task cancelled by user")
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	SourceDB    model.DBConfig `json:"source_db"`
	TargetDB    model.DBConfig `json:"target_db"`
	Tables      []string       `json:"tables"`
	TableSuffix string         `json:"table_suffix"`
}
