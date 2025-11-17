package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// CreatingTablesState represents the creating tables state
type CreatingTablesState struct {
	BaseState
}

// NewCreatingTablesState creates a new creating tables state
func NewCreatingTablesState() *CreatingTablesState {
	return &CreatingTablesState{
		BaseState: BaseState{name: model.StateCreatingTables.String()},
	}
}

// Execute executes the table creation logic
func (s *CreatingTablesState) Execute(ctx context.Context, task *model.MigrationTask) error {
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

	// Create target tables for each table
	schema := "public"
	for _, tableName := range tables {
		// Get table structure
		tableInfo, err := sourceRepo.GetTableInfo(schema, tableName)
		if err != nil {
			return fmt.Errorf("failed to get table info for %s: %w", tableName, err)
		}

		// Create target table
		if err := targetRepo.CreateTable(tableInfo, task.TableSuffix); err != nil {
			return fmt.Errorf("failed to create target table for %s: %w", tableName, err)
		}
	}

	return nil
}

// Next returns the next state
func (s *CreatingTablesState) Next() State {
	return NewMigratingDataState()
}
