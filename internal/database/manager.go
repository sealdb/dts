package database

import (
	"gorm.io/gorm"
)

// DatabaseType represents database type
type DatabaseType string

const (
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
	DatabaseTypeMySQL      DatabaseType = "mysql"
)

// DatabaseInfo represents database information
type DatabaseInfo struct {
	Datname string      `json:"datname"` // Database name
	Oid     uint32      `json:"oid"`     // Database OID (PostgreSQL specific)
	Tables  []TableInfo `json:"tables"`  // Tables in this database
}

// TableInfo represents table information
type TableInfo struct {
	DatabaseName string   `json:"database_name"` // Database name
	SchemaName   string   `json:"schema_name"`   // Schema name
	TableName    string   `json:"table_name"`     // Table name
	TableID      uint32   `json:"table_id"`       // Table OID (PostgreSQL specific)
	Indexes      []string `json:"indexes"`        // Index names
}

// Manager is the interface for database operations
type Manager interface {
	// GetAllDatabases retrieves all databases
	GetAllDatabases() ([]DatabaseInfo, error)

	// GetBusinessTablesInDatabase retrieves all business tables in current connected database
	// NOTE: Only be called by connection to the business database
	GetBusinessTablesInDatabase() ([]TableInfo, error)

	// GetDB returns the underlying GORM DB connection
	GetDB() *gorm.DB

	// Close closes the connection
	Close() error
}

// NewManager creates a database manager based on database type
func NewManager(dbType DatabaseType, dsn string) (Manager, error) {
	switch dbType {
	case DatabaseTypePostgreSQL:
		return NewPostgresManager(dsn)
	case DatabaseTypeMySQL:
		return NewMySQLManager(dsn)
	default:
		return nil, ErrUnsupportedDatabaseType
	}
}

