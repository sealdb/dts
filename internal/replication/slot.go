package replication

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SlotManager 复制槽管理器
type SlotManager struct {
	db *gorm.DB
}

// NewSlotManager 创建复制槽管理器
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

// NewSlotManagerFromDB 从已有 GORM 连接创建复制槽管理器
func NewSlotManagerFromDB(db *gorm.DB) (*SlotManager, error) {
	return &SlotManager{db: db}, nil
}

// Close 关闭连接（注意：如果使用共享连接，不应该调用此方法）
func (sm *SlotManager) Close() error {
	sqlDB, err := sm.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CreateSlot 创建逻辑复制槽
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

// DropSlot 删除复制槽
func (sm *SlotManager) DropSlot(slotName string) error {
	query := "SELECT pg_drop_replication_slot(?)"
	err := sm.db.Exec(query, slotName).Error
	if err != nil {
		return fmt.Errorf("failed to drop replication slot: %w", err)
	}
	return nil
}

// SlotExists 检查复制槽是否存在
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
