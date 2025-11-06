package state

import (
	"context"

	"github.com/pg/dts/internal/model"
)

// FailedState 失败状态
type FailedState struct {
	BaseState
}

// NewFailedState 创建失败状态
func NewFailedState() *FailedState {
	return &FailedState{
		BaseState: BaseState{name: model.StateFailed.String()},
	}
}

// Execute 执行失败状态逻辑
func (s *FailedState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// TODO: 实现失败处理逻辑
	// 1. 清理资源（复制槽、Publication等）
	// 2. 记录错误信息

	return nil
}

// Next 返回下一个状态（失败状态是终止状态）
func (s *FailedState) Next() State {
	return nil
}

// CanTransition 失败状态不能转换
func (s *FailedState) CanTransition() bool {
	return false
}
