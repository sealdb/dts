package replication

import (
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PublicationManager manages publications
type PublicationManager struct {
	db *gorm.DB
}

// NewPublicationManager creates a publication manager
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

// NewPublicationManagerFromDB creates a publication manager from existing GORM connection
func NewPublicationManagerFromDB(db *gorm.DB) (*PublicationManager, error) {
	return &PublicationManager{db: db}, nil
}

// Close closes the connection (note: should not call this method if using shared connection)
func (pm *PublicationManager) Close() error {
	sqlDB, err := pm.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CreatePublication creates a publication
func (pm *PublicationManager) CreatePublication(pubName string, tables []string) error {
	if len(tables) == 0 {
		return fmt.Errorf("no tables specified")
	}

	// Build table list
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

// DropPublication drops a publication
func (pm *PublicationManager) DropPublication(pubName string) error {
	query := fmt.Sprintf("DROP PUBLICATION IF EXISTS %s", pubName)
	err := pm.db.Exec(query).Error
	if err != nil {
		return fmt.Errorf("failed to drop publication: %w", err)
	}
	return nil
}

// PublicationExists checks if publication exists
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

// AddTables adds tables to publication
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
