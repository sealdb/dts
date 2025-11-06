package state

import (
	"context"

	"github.com/pg/dts/internal/model"
)

// State 状态接口
type State interface {
	// Name 返回状态名称
	Name() string

	// Execute 执行状态逻辑
	Execute(ctx context.Context, task *model.MigrationTask) error

	// Next 返回下一个状态
	Next() State

	// CanTransition 判断是否可以转换
	CanTransition() bool
}

// BaseState 基础状态实现
type BaseState struct {
	name string
}

// Name 返回状态名称
func (b *BaseState) Name() string {
	return b.name
}

// CanTransition 默认实现
func (b *BaseState) CanTransition() bool {
	return true
}

// StateMachine 状态机
type StateMachine struct {
	currentState State
	task         *model.MigrationTask
}

// NewStateMachine 创建状态机
func NewStateMachine(task *model.MigrationTask) *StateMachine {
	state := getInitialState(task.State)
	return &StateMachine{
		currentState: state,
		task:         task,
	}
}

// Execute 执行当前状态
func (sm *StateMachine) Execute(ctx context.Context) error {
	if sm.currentState == nil {
		return ErrInvalidState
	}

	err := sm.currentState.Execute(ctx, sm.task)
	if err != nil {
		return err
	}

	// 检查是否可以转换到下一个状态
	if sm.currentState.CanTransition() {
		nextState := sm.currentState.Next()
		if nextState != nil {
			sm.currentState = nextState
		}
	}

	return nil
}

// GetCurrentState 获取当前状态
func (sm *StateMachine) GetCurrentState() State {
	return sm.currentState
}

// SetState 设置状态（用于恢复）
func (sm *StateMachine) SetState(stateName string) {
	sm.currentState = getStateByName(stateName)
}

// getInitialState 根据状态名获取初始状态
func getInitialState(stateName string) State {
	stateType := model.StateType(stateName)
	switch stateType {
	case model.StateInit:
		return NewInitState()
	case model.StateCreatingTables:
		return NewCreatingTablesState()
	case model.StateMigratingData:
		return NewMigratingDataState()
	case model.StateSyncingWAL:
		return NewSyncingWALState()
	case model.StateStoppingWrites:
		return NewStoppingWritesState()
	case model.StateValidating:
		return NewValidatingState()
	case model.StateFinalizing:
		return NewFinalizingState()
	case model.StateCompleted:
		return NewCompletedState()
	case model.StateFailed:
		return NewFailedState()
	case model.StatePaused:
		return NewPausedState()
	default:
		return NewInitState()
	}
}

// getStateByName 根据名称获取状态
func getStateByName(name string) State {
	return getInitialState(name)
}
