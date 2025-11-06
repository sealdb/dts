package state

import (
	"context"

	"github.com/pg/dts/internal/model"
)

// PausedState 暂停状态
type PausedState struct {
	BaseState
}

// NewPausedState 创建暂停状态
func NewPausedState() *PausedState {
	return &PausedState{
		BaseState: BaseState{name: model.StatePaused.String()},
	}
}

// Execute 执行暂停状态逻辑
func (s *PausedState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// 暂停状态不需要执行任何操作
	// 等待恢复命令
	return nil
}

// Next 返回下一个状态（需要根据当前任务状态决定）
func (s *PausedState) Next() State {
	// 暂停后恢复时，需要根据任务状态决定恢复到哪里
	// 这里返回 nil，由外部调用者决定
	return nil
}

// CanTransition 暂停状态可以转换
func (s *PausedState) CanTransition() bool {
	return false // 暂停状态需要外部命令才能转换
}
