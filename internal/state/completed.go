package state

import (
	"context"

	"github.com/pg/dts/internal/model"
)

// CompletedState represents the completed state
type CompletedState struct {
	BaseState
}

// NewCompletedState creates a new completed state
func NewCompletedState() *CompletedState {
	return &CompletedState{
		BaseState: BaseState{name: model.StateCompleted.String()},
	}
}

// Execute executes the completed state logic
func (s *CompletedState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// Completed state: clean up resources and close connections
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

// Next returns the next state (completed state is a terminal state)
func (s *CompletedState) Next() State {
	return nil
}

// CanTransition returns whether the completed state can transition (it cannot)
func (s *CompletedState) CanTransition() bool {
	return false
}
