package database

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// MySQLManager manages MySQL database operations
type MySQLManager struct {
	db *gorm.DB
}

// NewMySQLManager creates a new MySQL manager
func NewMySQLManager(dsn string) (*MySQLManager, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info), // TODO: need to delete
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Set connection pool parameters
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	return &MySQLManager{db: db}, nil
}

// GetDB returns the underlying GORM DB connection
func (mm *MySQLManager) GetDB() *gorm.DB {
	return mm.db
}

// Close closes the connection
func (mm *MySQLManager) Close() error {
	sqlDB, err := mm.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetAllDatabases retrieves all databases
// TODO: Implement MySQL database listing
func (mm *MySQLManager) GetAllDatabases() ([]DatabaseInfo, error) {
	// TODO: Implement MySQL database listing
	return nil, fmt.Errorf("MySQL database listing not implemented yet")
}

// GetBusinessTablesInDatabase retrieves all business tables in current connected database
// TODO: Implement MySQL table listing
func (mm *MySQLManager) GetBusinessTablesInDatabase() ([]TableInfo, error) {
	// TODO: Implement MySQL table listing
	return nil, fmt.Errorf("MySQL table listing not implemented yet")
}

