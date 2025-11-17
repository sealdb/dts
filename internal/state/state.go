package state

import (
	"context"

	"github.com/pg/dts/internal/model"
)

// State represents the state interface
type State interface {
	// Name returns the state name
	Name() string

	// Execute executes the state logic
	Execute(ctx context.Context, task *model.MigrationTask) error

	// Next returns the next state
	Next() State

	// CanTransition returns whether the state can transition
	CanTransition() bool
}

// BaseState provides base state implementation
type BaseState struct {
	name string
}

// Name returns the state name
func (b *BaseState) Name() string {
	return b.name
}

// CanTransition provides default implementation
func (b *BaseState) CanTransition() bool {
	return true
}

// StateMachine represents the state machine
type StateMachine struct {
	currentState State
	task         *model.MigrationTask
}

// NewStateMachine creates a new state machine
func NewStateMachine(task *model.MigrationTask) *StateMachine {
	state := getInitialState(task.State)
	return &StateMachine{
		currentState: state,
		task:         task,
	}
}

// Execute executes the current state
func (sm *StateMachine) Execute(ctx context.Context) error {
	if sm.currentState == nil {
		return ErrInvalidState
	}

	err := sm.currentState.Execute(ctx, sm.task)
	if err != nil {
		return err
	}

	// Check if can transition to next state
	if sm.currentState.CanTransition() {
		nextState := sm.currentState.Next()
		if nextState != nil {
			sm.currentState = nextState
		}
	}

	return nil
}

// GetCurrentState returns the current state
func (sm *StateMachine) GetCurrentState() State {
	return sm.currentState
}

// SetState sets the state (for recovery)
func (sm *StateMachine) SetState(stateName string) {
	sm.currentState = getStateByName(stateName)
}

// getInitialState gets the initial state by state name
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

// getStateByName gets the state by name
func getStateByName(name string) State {
	return getInitialState(name)
}
