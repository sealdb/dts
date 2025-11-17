package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// ValidatingState represents the validating state
type ValidatingState struct {
	BaseState
}

// NewValidatingState creates a new validating state
func NewValidatingState() *ValidatingState {
	return &ValidatingState{
		BaseState: BaseState{name: model.StateValidating.String()},
	}
}

// Execute executes the validation logic
func (s *ValidatingState) Execute(ctx context.Context, task *model.MigrationTask) error {
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
	// Connections are managed by task manager, don't close here

	targetRepo, err := repository.NewTargetRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}
	// Connections are managed by task manager, don't close here

	// Validate row count for each table
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

		// Compare row counts
		if sourceCount != targetCount {
			return fmt.Errorf("row count mismatch for table %s: source=%d, target=%d",
				tableName, sourceCount, targetCount)
		}
	}

	return nil
}

// Next returns the next state
func (s *ValidatingState) Next() State {
	return NewFinalizingState()
}
