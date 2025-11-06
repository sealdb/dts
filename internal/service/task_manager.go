package service

import (
	"sync"

	"github.com/pg/dts/internal/model"
)

// TaskManager 任务管理器，管理所有正在运行的迁移任务
type TaskManager struct {
	tasks map[string]*model.MigrationTask // key: task ID, value: MigrationTask
	mu    sync.RWMutex                    // 保护 tasks 的并发访问
}

// NewTaskManager 创建任务管理器
func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*model.MigrationTask),
	}
}

// AddTask 添加任务
func (tm *TaskManager) AddTask(task *model.MigrationTask) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tasks[task.ID] = task
}

// GetTask 获取任务
func (tm *TaskManager) GetTask(taskID string) (*model.MigrationTask, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	task, ok := tm.tasks[taskID]
	return task, ok
}

// RemoveTask 移除任务（并关闭所有连接）
func (tm *TaskManager) RemoveTask(taskID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[taskID]
	if !ok {
		return nil // 任务不存在，无需处理
	}

	// 关闭任务的所有连接
	if err := task.CloseAllConnections(); err != nil {
		// 即使关闭连接失败，也移除任务
		delete(tm.tasks, taskID)
		return err
	}

	delete(tm.tasks, taskID)
	return nil
}

// ListTasks 列出所有任务
func (tm *TaskManager) ListTasks() []*model.MigrationTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*model.MigrationTask, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// GetTaskCount 获取任务数量
func (tm *TaskManager) GetTaskCount() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.tasks)
}

// CleanupCompletedTasks 清理已完成或失败的任务
func (tm *TaskManager) CleanupCompletedTasks() []error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var errors []error
	taskIDsToRemove := make([]string, 0)

	for taskID, task := range tm.tasks {
		state := model.StateType(task.State)
		if state.IsTerminal() {
			// 任务已完成或失败，需要清理
			if err := task.CloseAllConnections(); err != nil {
				errors = append(errors, err)
			}
			taskIDsToRemove = append(taskIDsToRemove, taskID)
		}
	}

	// 移除已完成的任务
	for _, taskID := range taskIDsToRemove {
		delete(tm.tasks, taskID)
	}

	return errors
}
