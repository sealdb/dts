package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// FullSyncState represents the full sync state
type FullSyncState struct {
	BaseState
}

// NewFullSyncState creates a new full sync state
func NewFullSyncState() *FullSyncState {
	return &FullSyncState{
		BaseState: BaseState{name: model.StateFullSync.String()},
	}
}

// Execute executes the full data synchronization logic
func (s *FullSyncState) Execute(ctx context.Context, task *model.MigrationTask) error {
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

	// Migrate data for each table using replication technology
	schema := "public"
	for i, tableName := range tables {
		sourceTable := tableName
		targetTable := tableName + task.TableSuffix

		if err := targetRepo.CopyData(sourceRepo, schema, sourceTable, schema, targetTable); err != nil {
			return fmt.Errorf("failed to copy data for table %s: %w", tableName, err)
		}

		// Update progress (simple implementation, can be more precise)
		progress := (i + 1) * 100 / len(tables)
		// TODO: Update task progress to database
		_ = progress
	}

	return nil
}

// Next returns the next state
func (s *FullSyncState) Next() State {
	return NewIncSyncState()
}

