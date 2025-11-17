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

// MigrationService provides migration service
type MigrationService struct {
	taskRepo    *repository.MigrationRepository
	db          *gorm.DB
	taskManager *TaskManager
}

// NewMigrationService creates a new migration service
func NewMigrationService(db *gorm.DB) *MigrationService {
	return &MigrationService{
		taskRepo:    repository.NewMigrationRepository(db),
		db:          db,
		taskManager: NewTaskManager(),
	}
}

// GetTaskManager returns the task manager
func (s *MigrationService) GetTaskManager() *TaskManager {
	return s.taskManager
}

// CreateTaskWithID creates a migration task with specified ID
func (s *MigrationService) CreateTaskWithID(id string, req *CreateTaskRequest) (*model.MigrationTask, error) {
	// Check if ID already exists
	if _, err := s.taskRepo.GetByID(id); err == nil {
		return nil, fmt.Errorf("task with id %s already exists", id)
	}

	// Serialize database configuration
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

// CreateTask creates a migration task (auto-generates ID)
func (s *MigrationService) CreateTask(req *CreateTaskRequest) (*model.MigrationTask, error) {
	// Serialize database configuration
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

// GetTask gets a task
func (s *MigrationService) GetTask(id string) (*model.MigrationTask, error) {
	return s.taskRepo.GetByID(id)
}

// ListTasks gets the task list
func (s *MigrationService) ListTasks(limit, offset int) ([]*model.MigrationTask, error) {
	return s.taskRepo.List(limit, offset)
}

// StartTask starts a task
func (s *MigrationService) StartTask(ctx context.Context, id string) error {
	log := logger.GetLogger()
	log.WithField("task_id", id).Info("Starting migration task")

	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		log.WithError(err).Error("Failed to get task")
		return err
	}

	// Check task state
	currentState := model.StateType(task.State)
	if currentState.IsTerminal() {
		err := fmt.Errorf("task is in terminal state: %s", currentState)
		log.WithError(err).Warn("Cannot start task")
		return err
	}

	// Check if task is already running
	if _, exists := s.taskManager.GetTask(id); exists {
		err := fmt.Errorf("task %s is already running", id)
		log.WithError(err).Warn("Task already running")
		return err
	}

	// Ensure task connection pool is initialized
	if task.Connections == nil {
		task.Connections = make(map[string]interface{})
	}

	// Add to task manager
	s.taskManager.AddTask(task)
	log.WithField("task_id", id).Info("Task added to task manager")

	// Create state machine
	sm := state.NewStateMachine(task)

	// Execute state machine
	go func() {
		log := logger.GetLogger()
		log.WithField("task_id", id).Info("State machine goroutine started")
		// Ensure connections are cleaned up after task completion
		defer func() {
			log.WithField("task_id", id).Info("Cleaning up task")
			if err := s.taskManager.RemoveTask(id); err != nil {
				// Log cleanup error but don't affect task state
				log.WithError(err).Warn("Failed to cleanup task")
			}
		}()

		// Simple retry mechanism configuration
		maxRetries := 3
		baseDelayMs := 500

		for {
			// Get current state
			currentState := sm.GetCurrentState()
			if currentState != nil {
				log.WithFields(map[string]interface{}{
					"task_id": id,
					"state":   currentState.Name(),
				}).Info("Executing state")
			}

			// Execute current state with retry
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
				// Simple sleep (avoid introducing additional dependencies)
				select {
				case <-ctx.Done():
					log.WithField("task_id", id).Warn("Context cancelled")
					s.taskRepo.UpdateState(task.ID, model.StateFailed, ctx.Err().Error())
					return
				case <-time.After(time.Duration(delay) * time.Millisecond):
				}
			}

			if execErr != nil {
				// Update task to failed state
				log.WithError(execErr).WithField("task_id", id).Error("State execution failed")
				s.taskRepo.UpdateState(task.ID, model.StateFailed, execErr.Error())
				// Clean up connections on failure
				task.CloseAllConnections()
				return
			}

			// Update task state
			currentState = sm.GetCurrentState()
			if currentState != nil {
				newState := model.StateType(currentState.Name())
				log.WithFields(map[string]interface{}{
					"task_id":   id,
					"new_state": newState.String(),
				}).Info("State transition completed")
				s.taskRepo.UpdateState(task.ID, newState, "")
				// Coarse-grained progress: advance by state
				s.taskRepo.UpdateProgress(task.ID, progressForState(newState))
			}

			// Check if reached terminal state
			if task.State == model.StateCompleted.String() || task.State == model.StateFailed.String() {
				log.WithFields(map[string]interface{}{
					"task_id":     id,
					"final_state": task.State,
				}).Info("Task reached terminal state")
				// Task completed, clean up connections (defer will also execute, but explicit call here ensures cleanup)
				task.CloseAllConnections()
				return
			}

			// Reload task to get latest state
			task, _ = s.taskRepo.GetByID(id)
		}
	}()

	return nil
}

// isRetryable simply determines if an error is retryable
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	// Can be extended with more fine-grained judgment, here simply based on error message
	msg := err.Error()
	retryHints := []string{"timeout", "temporarily", "connection refused", "deadlock detected"}
	for _, h := range retryHints {
		if strings.Contains(strings.ToLower(msg), h) {
			return true
		}
	}
	return false
}

// progressForState provides coarse-grained progress for different states
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

// PauseTask pauses a task
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

// ResumeTask resumes a task
func (s *MigrationService) ResumeTask(ctx context.Context, id string) error {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return err
	}

	if task.State != model.StatePaused.String() {
		return fmt.Errorf("task is not paused")
	}

	// Resume task
	return s.StartTask(ctx, id)
}

// DeleteTask deletes a task
func (s *MigrationService) DeleteTask(id string) error {
	// Cancel task first (if running)
	_ = s.CancelTask(id)

	// Remove from task manager
	s.taskManager.RemoveTask(id)

	// Delete from database
	return s.taskRepo.Delete(id)
}

// TriggerSwitchover triggers switchover
func (s *MigrationService) TriggerSwitchover(ctx context.Context, id string) error {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return err
	}

	// If task is in syncing_wal state, switch to stopping_writes
	if task.State == string(model.StateSyncingWAL) {
		// Update state to stopping_writes
		return s.taskRepo.UpdateState(id, model.StateStoppingWrites, "")
	}

	return fmt.Errorf("task is not in a state that allows switchover: %s", task.State)
}

// StopTask stops a task (task remains, just stops running)
func (s *MigrationService) StopTask(id string) error {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return err
	}

	currentState := model.StateType(task.State)
	if currentState.IsTerminal() {
		return fmt.Errorf("cannot stop task in terminal state: %s", currentState)
	}

	// If task is paused, it's already stopped
	if currentState == model.StatePaused {
		return nil
	}

	// Pause the task to stop it
	return s.PauseTask(id)
}

// CancelTask cancels a task
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

// CreateTaskRequest represents a create task request
type CreateTaskRequest struct {
	SourceDB    model.DBConfig `json:"source_db"`
	TargetDB    model.DBConfig `json:"target_db"`
	Tables      []string       `json:"tables"`
	TableSuffix string         `json:"table_suffix"`
}
