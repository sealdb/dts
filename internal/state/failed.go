package state

import (
	"context"

	"github.com/pg/dts/internal/model"
)

// FailedState represents the failed state
type FailedState struct {
	BaseState
}

// NewFailedState creates a new failed state
func NewFailedState() *FailedState {
	return &FailedState{
		BaseState: BaseState{name: model.StateFailed.String()},
	}
}

// Execute executes the failed state logic
func (s *FailedState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// TODO: Implement failure handling logic
	// 1. Clean up resources (replication slots, publications, etc.)
	// 2. Record error information

	return nil
}

// Next returns the next state (failed state is a terminal state)
func (s *FailedState) Next() State {
	return nil
}

// CanTransition returns whether the failed state can transition (it cannot)
func (s *FailedState) CanTransition() bool {
	return false
}
