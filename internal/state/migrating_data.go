package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// MigratingDataState represents the migrating data state
type MigratingDataState struct {
	BaseState
}

// NewMigratingDataState creates a new migrating data state
func NewMigratingDataState() *MigratingDataState {
	return &MigratingDataState{
		BaseState: BaseState{name: model.StateMigratingData.String()},
	}
}

// Execute executes the data migration logic
func (s *MigratingDataState) Execute(ctx context.Context, task *model.MigrationTask) error {
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

	// Migrate data for each table
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
func (s *MigratingDataState) Next() State {
	return NewSyncingWALState()
}
