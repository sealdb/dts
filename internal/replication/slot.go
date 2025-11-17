package replication

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SlotManager manages replication slots
type SlotManager struct {
	db *gorm.DB
}

// NewSlotManager creates a replication slot manager
func NewSlotManager(dsn string) (*SlotManager, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	return &SlotManager{db: db}, nil
}

// NewSlotManagerFromDB creates a replication slot manager from existing GORM connection
func NewSlotManagerFromDB(db *gorm.DB) (*SlotManager, error) {
	return &SlotManager{db: db}, nil
}

// Close closes the connection (note: should not call this method if using shared connection)
func (sm *SlotManager) Close() error {
	sqlDB, err := sm.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CreateSlot creates a logical replication slot
func (sm *SlotManager) CreateSlot(slotName, plugin string) error {
	if plugin == "" {
		plugin = "pgoutput"
	}

	query := "SELECT pg_create_logical_replication_slot(?, ?)"
	err := sm.db.Exec(query, slotName, plugin).Error
	if err != nil {
		return fmt.Errorf("failed to create replication slot: %w", err)
	}

	return nil
}

// DropSlot drops a replication slot
func (sm *SlotManager) DropSlot(slotName string) error {
	query := "SELECT pg_drop_replication_slot(?)"
	err := sm.db.Exec(query, slotName).Error
	if err != nil {
		return fmt.Errorf("failed to drop replication slot: %w", err)
	}
	return nil
}

// SlotExists checks if replication slot exists
func (sm *SlotManager) SlotExists(slotName string) (bool, error) {
	var exists bool
	err := sm.db.Raw(
		"SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name = ?)",
		slotName,
	).Scan(&exists).Error

	if err != nil {
		return false, fmt.Errorf("failed to check slot existence: %w", err)
	}

	return exists, nil
}
