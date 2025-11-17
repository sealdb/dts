package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// StoppingWritesState represents the stopping writes state
type StoppingWritesState struct {
	BaseState
}

// NewStoppingWritesState creates a new stopping writes state
func NewStoppingWritesState() *StoppingWritesState {
	return &StoppingWritesState{
		BaseState: BaseState{name: model.StateStoppingWrites.String()},
	}
}

// Execute executes the stopping writes logic
func (s *StoppingWritesState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// Create source repository (using connection pool)
	sourceRepo, err := repository.NewSourceRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}
	// Connections are managed by task manager, don't close here

	// Set database to read-only mode
	if err := sourceRepo.SetReadOnly(); err != nil {
		return fmt.Errorf("failed to set source database read-only: %w", err)
	}

	// TODO: Wait for all write operations to complete
	// Can query pg_stat_activity to check for active write transactions

	return nil
}

// Next returns the next state
func (s *StoppingWritesState) Next() State {
	return NewValidatingState()
}
