package state

import (
	"context"

	"github.com/pg/dts/internal/model"
)

// PausedState represents the paused state
type PausedState struct {
	BaseState
}

// NewPausedState creates a new paused state
func NewPausedState() *PausedState {
	return &PausedState{
		BaseState: BaseState{name: model.StatePaused.String()},
	}
}

// Execute executes the paused state logic
func (s *PausedState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// Paused state does not need to perform any operations
	// Wait for resume command
	return nil
}

// Next returns the next state (needs to be determined based on current task state)
func (s *PausedState) Next() State {
	// When resuming after pause, need to determine where to resume based on task state
	// Return nil here, let external caller decide
	return nil
}

// CanTransition returns whether the paused state can transition
func (s *PausedState) CanTransition() bool {
	return false // Paused state requires external command to transition
}
