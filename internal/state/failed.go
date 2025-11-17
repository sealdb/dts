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
	// Failed state: clean up resources and close connections
	// 1. Clean up replication resources (slots, publications)
	// 2. Remove read-only mode from source database (if set)
	// 3. Close all database connections

	// Close all connections
	if err := task.CloseAllConnections(); err != nil {
		// Log error but don't fail the state
		// TODO: Use proper logger
		_ = err
	}

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
