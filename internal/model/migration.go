package model

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// MigrationTask represents a migration task
type MigrationTask struct {
	ID           string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	SourceDB     string     `gorm:"type:text;not null" json:"source_db"`   // Source database configuration in JSON format
	TargetDB     string     `gorm:"type:text;not null" json:"target_db"`   // Target database configuration in JSON format
	Tables       string     `gorm:"type:text;not null" json:"tables"`      // Table list in JSON format
	TableSuffix  string     `gorm:"type:varchar(100)" json:"table_suffix"` // Target table suffix
	State        string     `gorm:"type:varchar(50);not null;default:'init'" json:"state"`
	Progress     int        `gorm:"default:0" json:"progress"` // Progress 0-100
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`

	// Runtime fields (not persisted)
	Connections map[string]interface{} `gorm:"-" json:"-"` // Database connection pool key: connectionKey (host:port:dbname), value: *sql.DB or *gorm.DB
	mu          sync.RWMutex           `gorm:"-" json:"-"` // Protects concurrent access to connections
}

// TableName specifies the table name
func (*MigrationTask) TableName() string {
	return "migration_tasks"
}

// BeforeCreate is a hook before creation
func (m *MigrationTask) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = generateUUID()
	}
	// Initialize connection pool
	if m.Connections == nil {
		m.Connections = make(map[string]interface{})
	}
	return nil
}

// AfterFind is a hook after query, initializes connection pool
func (m *MigrationTask) AfterFind(tx *gorm.DB) error {
	if m.Connections == nil {
		m.Connections = make(map[string]interface{})
	}
	return nil
}

// ConnectionKey generates a connection key
func ConnectionKey(host string, port int, dbname string) string {
	return fmt.Sprintf("%s:%d:%s", host, port, dbname)
}

// AddConnection adds a database connection
func (m *MigrationTask) AddConnection(key string, conn interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Connections == nil {
		m.Connections = make(map[string]interface{})
	}
	m.Connections[key] = conn
}

// GetConnection gets a database connection
func (m *MigrationTask) GetConnection(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Connections == nil {
		return nil, false
	}
	conn, ok := m.Connections[key]
	return conn, ok
}

// CloseAllConnections closes all connections
func (m *MigrationTask) CloseAllConnections() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errors []error
	for key, conn := range m.Connections {
		if conn == nil {
			continue
		}

		switch c := conn.(type) {
		case *gorm.DB:
			sqlDB, err := c.DB()
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to get sql.DB from gorm.DB for %s: %w", key, err))
				continue
			}
			if err := sqlDB.Close(); err != nil {
				errors = append(errors, fmt.Errorf("failed to close gorm.DB connection %s: %w", key, err))
			}
		default:
			// Unknown connection type, log warning but don't error
			continue
		}
	}

	// Clear connection pool
	m.Connections = make(map[string]interface{})

	if len(errors) > 0 {
		return fmt.Errorf("errors closing connections: %v", errors)
	}

	return nil
}

// GetConnectionCount returns the number of connections (for debugging)
func (m *MigrationTask) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Connections == nil {
		return 0
	}
	return len(m.Connections)
}

// DBConfig represents database configuration
type DBConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

// DSN returns database connection string
func (d *DBConfig) DSN() string {
	if d.SSLMode == "" {
		d.SSLMode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

// ConnectionKey returns connection key
func (d *DBConfig) ConnectionKey() string {
	return ConnectionKey(d.Host, d.Port, d.DBName)
}

// TableInfo represents table structure information
type TableInfo struct {
	Schema      string           `json:"schema"`
	Name        string           `json:"name"`
	Columns     []ColumnInfo     `json:"columns"`
	Indexes     []IndexInfo      `json:"indexes"`
	Constraints []ConstraintInfo `json:"constraints"`
	DDL         string           `json:"ddl"`
}

// ColumnInfo represents column information
type ColumnInfo struct {
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	IsNullable   bool   `json:"is_nullable"`
	DefaultValue string `json:"default_value"`
	IsPrimaryKey bool   `json:"is_primary_key"`
}

// IndexInfo represents index information
type IndexInfo struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	DDL     string   `json:"ddl"`
}

// ConstraintInfo represents constraint information
type ConstraintInfo struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"` // PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK
	Columns    []string `json:"columns"`
	Definition string   `json:"definition"`
}
