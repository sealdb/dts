package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// InitState represents the initialization state
type InitState struct {
	BaseState
}

// NewInitState creates a new initialization state
func NewInitState() *InitState {
	return &InitState{
		BaseState: BaseState{name: model.StateInit.String()},
	}
}

// Execute executes the initialization logic
func (s *InitState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// Parse table list
	tables, err := repository.ParseTables(task)
	if err != nil {
		return fmt.Errorf("failed to parse tables: %w", err)
	}

	// Verify source database connection and wal_level (using connection pool, don't close connection)
	sourceRepo, err := repository.NewSourceRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}
	// Note: Do not close connection here, connections are managed by task manager

	walLevel, err := sourceRepo.CheckWALLevel()
	if err != nil {
		return fmt.Errorf("failed to check wal_level: %w", err)
	}

	if walLevel != "logical" {
		return fmt.Errorf("source database wal_level must be 'logical', got '%s'", walLevel)
	}

	// Verify target database connection (using connection pool, don't close connection)
	_, err = repository.NewTargetRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}
	// Note: Do not close connection here, connections are managed by task manager

	// Verify tables exist
	schema := "public" // Default schema, can be read from configuration
	for _, tableName := range tables {
		_, err := sourceRepo.GetTableInfo(schema, tableName)
		if err != nil {
			return fmt.Errorf("table %s.%s not found or inaccessible: %w", schema, tableName, err)
		}
	}

	return nil
}

// Next returns the next state
func (s *InitState) Next() State {
	return NewCreatingTablesState()
}

// CanTransition returns whether the state can transition
func (s *InitState) CanTransition() bool {
	return true
}
