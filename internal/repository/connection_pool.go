package repository

import (
	"fmt"

	"github.com/pg/dts/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// GetOrCreateGORMConnection gets or creates GORM database connection (with connection pool management)
func GetOrCreateGORMConnection(task *model.MigrationTask, dbConfig *model.DBConfig) (*gorm.DB, error) {
	connectionKey := dbConfig.ConnectionKey()

	// Try to get existing connection from task
	if conn, ok := task.GetConnection(connectionKey); ok {
		if gormDB, ok := conn.(*gorm.DB); ok {
			// Verify connection is still valid
			sqlDB, err := gormDB.DB()
			if err == nil {
				if err := sqlDB.Ping(); err == nil {
					return gormDB, nil
				}
			}
			// Connection invalid, need to recreate
		}
	}

	// Create new connection
	db, err := gorm.Open(postgres.Open(dbConfig.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
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
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Add connection to task connection pool
	task.AddConnection(connectionKey, db)

	return db, nil
}

// GetOrCreateSourceGORMConnection gets or creates source database GORM connection
func GetOrCreateSourceGORMConnection(task *model.MigrationTask) (*gorm.DB, error) {
	sourceDB, err := ParseSourceDB(task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source db config: %w", err)
	}

	return GetOrCreateGORMConnection(task, sourceDB)
}

// GetOrCreateTargetGORMConnection gets or creates target database GORM connection
func GetOrCreateTargetGORMConnection(task *model.MigrationTask) (*gorm.DB, error) {
	targetDB, err := ParseTargetDB(task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target db config: %w", err)
	}

	return GetOrCreateGORMConnection(task, targetDB)
}

// GetOrCreateSourceConnection gets or creates source database connection (compatible with old interface, returns sql.DB)
// Note: This method is mainly used for scenarios requiring pgx native API (such as COPY FROM STDIN)
func GetOrCreateSourceConnection(task *model.MigrationTask) (*gorm.DB, error) {
	return GetOrCreateSourceGORMConnection(task)
}

// GetOrCreateTargetConnection gets or creates target database connection (compatible with old interface, returns sql.DB)
// Note: This method is mainly used for scenarios requiring pgx native API (such as COPY FROM STDIN)
func GetOrCreateTargetConnection(task *model.MigrationTask) (*gorm.DB, error) {
	return GetOrCreateTargetGORMConnection(task)
}
