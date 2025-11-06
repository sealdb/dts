package replication

import (
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PublicationManager 发布管理器
type PublicationManager struct {
	db *gorm.DB
}

// NewPublicationManager 创建发布管理器
func NewPublicationManager(dsn string) (*PublicationManager, error) {
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

	return &PublicationManager{db: db}, nil
}

// NewPublicationManagerFromDB 从已有 GORM 连接创建发布管理器
func NewPublicationManagerFromDB(db *gorm.DB) (*PublicationManager, error) {
	return &PublicationManager{db: db}, nil
}

// Close 关闭连接（注意：如果使用共享连接，不应该调用此方法）
func (pm *PublicationManager) Close() error {
	sqlDB, err := pm.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CreatePublication 创建发布
func (pm *PublicationManager) CreatePublication(pubName string, tables []string) error {
	if len(tables) == 0 {
		return fmt.Errorf("no tables specified")
	}

	// 构建表列表
	tableList := make([]string, len(tables))
	for i, table := range tables {
		tableList[i] = fmt.Sprintf("'%s'", table)
	}

	query := fmt.Sprintf(
		"CREATE PUBLICATION %s FOR TABLE %s",
		pubName,
		strings.Join(tableList, ", "),
	)

	err := pm.db.Exec(query).Error
	if err != nil {
		return fmt.Errorf("failed to create publication: %w", err)
	}

	return nil
}

// DropPublication 删除发布
func (pm *PublicationManager) DropPublication(pubName string) error {
	query := fmt.Sprintf("DROP PUBLICATION IF EXISTS %s", pubName)
	err := pm.db.Exec(query).Error
	if err != nil {
		return fmt.Errorf("failed to drop publication: %w", err)
	}
	return nil
}

// PublicationExists 检查发布是否存在
func (pm *PublicationManager) PublicationExists(pubName string) (bool, error) {
	var exists bool
	err := pm.db.Raw(
		"SELECT EXISTS(SELECT 1 FROM pg_publication WHERE pubname = ?)",
		pubName,
	).Scan(&exists).Error

	if err != nil {
		return false, fmt.Errorf("failed to check publication existence: %w", err)
	}

	return exists, nil
}

// AddTables 向发布添加表
func (pm *PublicationManager) AddTables(pubName string, tables []string) error {
	if len(tables) == 0 {
		return fmt.Errorf("no tables specified")
	}

	tableList := make([]string, len(tables))
	for i, table := range tables {
		tableList[i] = fmt.Sprintf("'%s'", table)
	}

	query := fmt.Sprintf(
		"ALTER PUBLICATION %s ADD TABLE %s",
		pubName,
		strings.Join(tableList, ", "),
	)

	err := pm.db.Exec(query).Error
	if err != nil {
		return fmt.Errorf("failed to add tables to publication: %w", err)
	}

	return nil
}
