package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// FinalizingState represents the finalizing state
type FinalizingState struct {
	BaseState
}

// NewFinalizingState creates a new finalizing state
func NewFinalizingState() *FinalizingState {
	return &FinalizingState{
		BaseState: BaseState{name: model.StateFinalizing.String()},
	}
}

// Execute executes the finalizing logic
func (s *FinalizingState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// Restore source database write permissions (using connection pool)
	sourceRepo, err := repository.NewSourceRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}
	// Connections are managed by task manager, don't close here

	if err := sourceRepo.RestoreWritePermissions(); err != nil {
		return fmt.Errorf("failed to restore write permissions: %w", err)
	}

	// TODO: Clean up logical replication resources
	// 1. Delete replication slots
	// 2. Delete publications
	// These should be managed in the SyncingWAL state

	return nil
}

// Next returns the next state
func (s *FinalizingState) Next() State {
	return NewCompletedState()
}
