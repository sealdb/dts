package database

import (
	"fmt"

	"github.com/pg/dts/internal/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// PostgresManager manages PostgreSQL database operations
type PostgresManager struct {
	db *gorm.DB
}

// NewPostgresManager creates a new PostgreSQL manager
func NewPostgresManager(dsn string) (*PostgresManager, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info), // TODO: need to delete
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
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
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return &PostgresManager{db: db}, nil
}

// GetDB returns the underlying GORM DB connection
func (pm *PostgresManager) GetDB() *gorm.DB {
	return pm.db
}

// Close closes the connection
func (pm *PostgresManager) Close() error {
	sqlDB, err := pm.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetAllDatabases retrieves all databases from pg_database
// Returns:
//   - []DatabaseInfo: List of database information
//   - error: Error if query fails
func (pm *PostgresManager) GetAllDatabases() ([]DatabaseInfo, error) {
	// First, get basic database information
	type BasicDatabaseInfo struct {
		Datname string `gorm:"column:datname"`
		Oid     uint32 `gorm:"column:oid"`
	}

	var basicDatabases []BasicDatabaseInfo
	err := pm.db.Raw("SELECT datname, oid FROM pg_database WHERE datistemplate = false AND datname != 'postgres'").Scan(&basicDatabases).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query databases: %w", err)
	}

	// Convert to DatabaseInfo with empty Tables slice
	databases := make([]DatabaseInfo, len(basicDatabases))
	for i, basicDB := range basicDatabases {
		databases[i] = DatabaseInfo{
			Datname: basicDB.Datname,
			Oid:     basicDB.Oid,
			Tables:  []TableInfo{}, // Initialize empty tables slice
		}
	}

	log := logger.GetLogger()
	log.WithField("count", len(databases)).Info("Found databases")
	return databases, nil
}

// GetBusinessTablesInDatabase retrieves all business tables in current connected database
// Business tables are defined as tables with oid > 16383 and relkind in ('r', 'p')
//
// Returns:
//   - []TableInfo: List of table information
//   - error: Error if query fails
//
// NOTE: Only be called by connection to the business database.
func (pm *PostgresManager) GetBusinessTablesInDatabase() ([]TableInfo, error) {
	var tables []TableInfo

	/*
		Example: testdb1
		database_name | schema_name | table_name | table_id
		---------------+-------------+------------+----------
		testdb1       | public      | t1         |    16442
		testdb1       | public      | t2         |    16447
		testdb1       | public      | t3         |    24763
		(3 rows)
	*/
	query := `
		SELECT
			current_database() AS database_name,
			n.nspname AS schema_name,
			c.relname AS table_name,
			c.oid AS table_id
		FROM
			pg_class c
		JOIN
			pg_namespace n ON c.relnamespace = n.oid
		WHERE
			(c.relkind = 'r' OR c.relkind = 'p')
			AND c.oid > 16383
			AND n.nspname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY n.nspname, c.relname
	`

	err := pm.db.Raw(query).Scan(&tables).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query business tables: %w", err)
	}

	// Query all indexes information for each table
	log := logger.GetLogger()
	for i := range tables {
		var indexes []string
		if err := pm.db.Raw(`SELECT indexname FROM pg_catalog.pg_indexes WHERE schemaname = ? AND tablename = ?`,
			tables[i].SchemaName, tables[i].TableName).Scan(&indexes).Error; err != nil {
			log.WithError(err).WithFields(map[string]interface{}{
				"database": tables[i].DatabaseName,
				"schema":   tables[i].SchemaName,
				"table":    tables[i].TableName,
			}).Error("Failed to query indexes")
			return nil, fmt.Errorf("failed to query indexes for table %s.%s: %w", tables[i].SchemaName, tables[i].TableName, err)
		}
		tables[i].Indexes = indexes
	}

	log.WithField("count", len(tables)).Info("Found business tables in current database")
	return tables, nil
}

