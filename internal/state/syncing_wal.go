package state

import (
	"context"
	"fmt"
	"time"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/replication"
	"github.com/pg/dts/internal/repository"
)

// SyncingWALState represents the syncing WAL state
type SyncingWALState struct {
	BaseState
}

// NewSyncingWALState creates a new syncing WAL state
func NewSyncingWALState() *SyncingWALState {
	return &SyncingWALState{
		BaseState: BaseState{name: model.StateSyncingWAL.String()},
	}
}

// Execute executes the WAL synchronization logic
func (s *SyncingWALState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// Parse table list
	tables, err := repository.ParseTables(task)
	if err != nil {
		return fmt.Errorf("failed to parse tables: %w", err)
	}

	// Get or create source GORM connection (using connection pool)
	sourceDB, err := repository.GetOrCreateSourceGORMConnection(task)
	if err != nil {
		return fmt.Errorf("failed to get source connection: %w", err)
	}

	// Ensure connection is valid
	sqlDB, err := sourceDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("source connection is not valid: %w", err)
	}

	// Create replication slot manager (using existing connection)
	slotManager, err := replication.NewSlotManagerFromDB(sourceDB)
	if err != nil {
		return fmt.Errorf("failed to create slot manager: %w", err)
	}
	// Note: slotManager uses shared connection, don't close separately

	// Create publication manager (using existing connection)
	pubManager, err := replication.NewPublicationManagerFromDB(sourceDB)
	if err != nil {
		return fmt.Errorf("failed to create publication manager: %w", err)
	}
	// Note: pubManager uses shared connection, don't close separately

	// Generate replication slot and publication names
	slotName := fmt.Sprintf("dts_slot_%s", task.ID)
	pubName := fmt.Sprintf("dts_pub_%s", task.ID)

	// Check and create replication slot
	exists, err := slotManager.SlotExists(slotName)
	if err != nil {
		return fmt.Errorf("failed to check slot existence: %w", err)
	}

	if !exists {
		if err := slotManager.CreateSlot(slotName, "pgoutput"); err != nil {
			return fmt.Errorf("failed to create replication slot: %w", err)
		}
	}

	// Check and create publication
	exists, err = pubManager.PublicationExists(pubName)
	if err != nil {
		return fmt.Errorf("failed to check publication existence: %w", err)
	}

	if !exists {
		// Build table name list (format: schema.table)
		schema := "public"
		tableNames := make([]string, len(tables))
		for i, table := range tables {
			tableNames[i] = fmt.Sprintf("%s.%s", schema, table)
		}

		if err := pubManager.CreatePublication(pubName, tableNames); err != nil {
			return fmt.Errorf("failed to create publication: %w", err)
		}
	}

	// Create subscriber and start synchronization
	// Note: Need to start a background goroutine to handle WAL stream
	// In actual implementation, should use context to control synchronization stop
	// Here is a simplified implementation that returns after syncing for a period
	// Should actually run continuously until StoppingWrites state

	// TODO: Start WAL subscriber
	// subscriber, err := replication.NewSubscriber(sourceDB.DSN(), slotName)
	// if err != nil {
	// 	return fmt.Errorf("failed to create subscriber: %w", err)
	// }
	// defer subscriber.Close()
	//
	// if err := subscriber.StartReplication(ctx, pubName); err != nil {
	// 	return fmt.Errorf("failed to start replication: %w", err)
	// }
	//
	// // Process replication stream in background
	// go func() {
	// 	if err := subscriber.ProcessReplicationStream(ctx); err != nil {
	// 		// Handle error
	// 	}
	// }()

	// Wait for a period to let WAL sync (should actually run continuously)
	// This is a simplified implementation
	time.Sleep(1 * time.Second)

	return nil
}

// Next returns the next state
func (s *SyncingWALState) Next() State {
	return NewStoppingWritesState()
}
