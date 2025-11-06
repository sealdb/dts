package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// FinalizingState 完成状态
type FinalizingState struct {
	BaseState
}

// NewFinalizingState 创建完成状态
func NewFinalizingState() *FinalizingState {
	return &FinalizingState{
		BaseState: BaseState{name: model.StateFinalizing.String()},
	}
}

// Execute 执行完成逻辑
func (s *FinalizingState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// 恢复源库写权限（使用连接池）
	sourceRepo, err := repository.NewSourceRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}
	// 连接由任务管理器统一管理，不在这里关闭

	if err := sourceRepo.RestoreWritePermissions(); err != nil {
		return fmt.Errorf("failed to restore write permissions: %w", err)
	}

	// TODO: 清理逻辑复制资源
	// 1. 删除复制槽
	// 2. 删除 Publication
	// 这些应该在 SyncingWAL 状态中管理

	return nil
}

// Next 返回下一个状态
func (s *FinalizingState) Next() State {
	return NewCompletedState()
}
