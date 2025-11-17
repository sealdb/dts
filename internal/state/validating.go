package state

import (
	"context"
	"fmt"
	"time"

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
	// Step 1: Set source database to read-only
	// TODO: Implement setting source database to read-only mode
	// This might require superuser privileges

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

	// Step 2: Loop to check source and target table data until they match
	// Check if PostgreSQL checksum is enabled, if so use checksum, otherwise use count(*)
	schema := "public"
	maxRetries := 10
	retryInterval := 5 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		allMatch := true

		for _, tableName := range tables {
			sourceTable := tableName
			targetTable := tableName + task.TableSuffix

			// Check if checksum is enabled (simplified: always use count for now)
			// TODO: Check PostgreSQL checksum configuration
			useChecksum := false

			var sourceValue, targetValue int64
			var err error

			if useChecksum {
				// Use checksum comparison
				// TODO: Implement checksum comparison
				// For now, checksum is not implemented, so we'll use count
				// sourceValue, err = sourceRepo.GetTableChecksum(schema, sourceTable)
				// if err != nil {
				// 	return fmt.Errorf("failed to get source table checksum for %s: %w", tableName, err)
				// }
				//
				// targetValue, err = targetRepo.GetTableChecksum(schema, targetTable)
				// if err != nil {
				// 	return fmt.Errorf("failed to get target table checksum for %s: %w", tableName, err)
				// }
				// Fall through to count(*) method
			}
			// Use count(*) comparison
			sourceValue, err = sourceRepo.GetTableCount(schema, sourceTable)
			if err != nil {
				return fmt.Errorf("failed to get source table count for %s: %w", tableName, err)
			}

			targetValue, err = targetRepo.GetTableCount(schema, targetTable)
			if err != nil {
				return fmt.Errorf("failed to get target table count for %s: %w", tableName, err)
			}

			// Compare values
			if sourceValue != targetValue {
				allMatch = false
				// TODO: Log mismatch
				break
			}
		}

		if allMatch {
			// All tables match, validation successful
			return nil
		}

		// Wait before next retry
		if attempt < maxRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryInterval):
			}
		}
	}

	// If we reach here, validation failed after max retries
	return fmt.Errorf("validation failed: source and target data do not match after %d attempts", maxRetries)
}

// Next returns the next state
func (s *ValidatingState) Next() State {
	return NewCompletedState()
}
