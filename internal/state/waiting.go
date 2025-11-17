package state

import (
	"context"
	"fmt"
	"time"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// WaitingState represents the waiting state
type WaitingState struct {
	BaseState
}

// NewWaitingState creates a new waiting state
func NewWaitingState() *WaitingState {
	return &WaitingState{
		BaseState: BaseState{name: model.StateWaiting.String()},
	}
}

// Execute executes the waiting state logic
func (s *WaitingState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// This state mainly waits for switch API
	// Periodically check synchronization status

	// Parse table list
	tables, err := repository.ParseTables(task)
	if err != nil {
		return fmt.Errorf("failed to parse tables: %w", err)
	}

	// Create repositories (using connection pool)
	sourceRepo, err := repository.NewSourceRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}

	targetRepo, err := repository.NewTargetRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}

	// Periodically check synchronization status
	// Compare row counts between source and target tables
	schema := "public"
	for _, tableName := range tables {
		sourceTable := tableName
		targetTable := tableName + task.TableSuffix

		// Get source table row count
		sourceCount, err := sourceRepo.GetTableCount(schema, sourceTable)
		if err != nil {
			return fmt.Errorf("failed to get source table count for %s: %w", tableName, err)
		}

		// Get target table row count
		targetCount, err := targetRepo.GetTableCount(schema, targetTable)
		if err != nil {
			return fmt.Errorf("failed to get target table count for %s: %w", tableName, err)
		}

		// Log synchronization status
		// TODO: Use proper logger
		_ = fmt.Sprintf("Table %s: source=%d, target=%d, diff=%d",
			tableName, sourceCount, targetCount, sourceCount-targetCount)
	}

	// Wait a bit before next check (this is a simplified implementation)
	// In actual implementation, this should be controlled by external switch API
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		// Continue waiting
	}

	return nil
}

// Next returns the next state
func (s *WaitingState) Next() State {
	return NewValidatingState()
}

// CanTransition returns whether the waiting state can transition
func (s *WaitingState) CanTransition() bool {
	// Waiting state can only transition when switch API is called
	// This is controlled externally, so return false here
	return false
}

