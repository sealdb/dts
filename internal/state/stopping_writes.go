package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// StoppingWritesState 停止写操作状态
type StoppingWritesState struct {
	BaseState
}

// NewStoppingWritesState 创建停止写操作状态
func NewStoppingWritesState() *StoppingWritesState {
	return &StoppingWritesState{
		BaseState: BaseState{name: model.StateStoppingWrites.String()},
	}
}

// Execute 执行停止写操作逻辑
func (s *StoppingWritesState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// 创建源库仓储（使用连接池）
	sourceRepo, err := repository.NewSourceRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}
	// 连接由任务管理器统一管理，不在这里关闭

	// 设置数据库为只读模式
	if err := sourceRepo.SetReadOnly(); err != nil {
		return fmt.Errorf("failed to set source database read-only: %w", err)
	}

	// TODO: 等待所有写操作完成
	// 可以查询 pg_stat_activity 检查是否有活跃的写事务

	return nil
}

// Next 返回下一个状态
func (s *StoppingWritesState) Next() State {
	return NewValidatingState()
}
