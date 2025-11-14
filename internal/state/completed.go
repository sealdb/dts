package state

import (
	"context"

	"github.com/pg/dts/internal/model"
)

// CompletedState 已完成状态
type CompletedState struct {
	BaseState
}

// NewCompletedState 创建已完成状态
func NewCompletedState() *CompletedState {
	return &CompletedState{
		BaseState: BaseState{name: model.StateCompleted.String()},
	}
}

// Execute 执行完成状态逻辑
func (s *CompletedState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// 已完成状态不需要执行任何操作
	return nil
}

// Next 返回下一个状态（已完成状态是终止状态）
func (s *CompletedState) Next() State {
	return nil
}

// CanTransition 已完成状态不能转换
func (s *CompletedState) CanTransition() bool {
	return false
}




