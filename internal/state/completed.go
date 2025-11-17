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
	// Completed state does not need to perform any operations
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
