package state

import (
	"context"

	"github.com/pg/dts/internal/model"
)

// DeletedState represents the deleted state
type DeletedState struct {
	BaseState
}

// NewDeletedState creates a new deleted state
func NewDeletedState() *DeletedState {
	return &DeletedState{
		BaseState: BaseState{name: model.StateDeleted.String()},
	}
}

// Execute executes the deleted state logic
func (s *DeletedState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// Deleted state: delete the task record from metadata database
	// This is handled by the service layer, so this state just marks completion
	return nil
}

// Next returns the next state (deleted state is a terminal state)
func (s *DeletedState) Next() State {
	return nil
}

// CanTransition returns whether the deleted state can transition (it cannot)
func (s *DeletedState) CanTransition() bool {
	return false
}


