package service

import (
	"sync"

	"github.com/pg/dts/internal/model"
)

// TaskManager manages all running migration tasks
type TaskManager struct {
	tasks map[string]*model.MigrationTask // key: task ID, value: MigrationTask
	mu    sync.RWMutex                    // protects concurrent access to tasks
}

// NewTaskManager creates a new task manager
func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*model.MigrationTask),
	}
}

// AddTask adds a task
func (tm *TaskManager) AddTask(task *model.MigrationTask) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tasks[task.ID] = task
}

// GetTask gets a task
func (tm *TaskManager) GetTask(taskID string) (*model.MigrationTask, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	task, ok := tm.tasks[taskID]
	return task, ok
}

// RemoveTask removes a task (and closes all connections)
func (tm *TaskManager) RemoveTask(taskID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[taskID]
	if !ok {
		return nil // Task does not exist, no need to process
	}

	// Close all connections for the task
	if err := task.CloseAllConnections(); err != nil {
		// Remove task even if closing connections fails
		delete(tm.tasks, taskID)
		return err
	}

	delete(tm.tasks, taskID)
	return nil
}

// ListTasks lists all tasks
func (tm *TaskManager) ListTasks() []*model.MigrationTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*model.MigrationTask, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// GetTaskCount returns the number of tasks
func (tm *TaskManager) GetTaskCount() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.tasks)
}

// CleanupCompletedTasks cleans up completed or failed tasks
func (tm *TaskManager) CleanupCompletedTasks() []error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var errors []error
	taskIDsToRemove := make([]string, 0)

	for taskID, task := range tm.tasks {
		state := model.StateType(task.State)
		if state.IsTerminal() {
			// Task is completed or failed, needs cleanup
			if err := task.CloseAllConnections(); err != nil {
				errors = append(errors, err)
			}
			taskIDsToRemove = append(taskIDsToRemove, taskID)
		}
	}

	// Remove completed tasks
	for _, taskID := range taskIDsToRemove {
		delete(tm.tasks, taskID)
	}

	return errors
}
