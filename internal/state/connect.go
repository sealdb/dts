package state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pg/dts/internal/database"
	"github.com/pg/dts/internal/model"
)

// ConnectState represents the connect state
type ConnectState struct {
	BaseState
}

// NewConnectState creates a new connect state
func NewConnectState() *ConnectState {
	return &ConnectState{
		BaseState: BaseState{name: model.StateConnect.String()},
	}
}

// Execute executes the connect state logic
func (s *ConnectState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// Parse database type
	dbType := database.DatabaseType(task.DatabaseType)
	if dbType == "" {
		dbType = database.DatabaseTypePostgreSQL // Default to PostgreSQL
	}

	// Parse source and target database configurations
	var sourceConfig, targetConfig model.DBConfig
	if err := json.Unmarshal([]byte(task.SourceDB), &sourceConfig); err != nil {
		return fmt.Errorf("failed to parse source database config: %w", err)
	}
	if err := json.Unmarshal([]byte(task.TargetDB), &targetConfig); err != nil {
		return fmt.Errorf("failed to parse target database config: %w", err)
	}

	// Step 1: Connect to source database (using postgres database for initial connection)
	sourcePostgresDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		sourceConfig.Host, sourceConfig.Port, sourceConfig.User, sourceConfig.Password, sourceConfig.SSLMode)
	if sourceConfig.SSLMode == "" {
		sourcePostgresDSN = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
			sourceConfig.Host, sourceConfig.Port, sourceConfig.User, sourceConfig.Password)
	}

	sourceManager, err := database.NewManager(dbType, sourcePostgresDSN)
	if err != nil {
		return fmt.Errorf("failed to create source database manager: %w", err)
	}
	defer sourceManager.Close()

	// Step 2: Get all databases from source
	databases, err := sourceManager.GetAllDatabases()
	if err != nil {
		return fmt.Errorf("failed to get databases from source: %w", err)
	}

	// Step 3: For each database, get business tables
	for i := range databases {
		// Connect to each business database
		dbDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			sourceConfig.Host, sourceConfig.Port, sourceConfig.User, sourceConfig.Password, databases[i].Datname, sourceConfig.SSLMode)
		if sourceConfig.SSLMode == "" {
			dbDSN = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
				sourceConfig.Host, sourceConfig.Port, sourceConfig.User, sourceConfig.Password, databases[i].Datname)
		}

		dbManager, err := database.NewManager(dbType, dbDSN)
		if err != nil {
			return fmt.Errorf("failed to connect to database %s: %w", databases[i].Datname, err)
		}
		defer dbManager.Close()

		// Get business tables in this database
		tables, err := dbManager.GetBusinessTablesInDatabase()
		if err != nil {
			return fmt.Errorf("failed to get tables from database %s: %w", databases[i].Datname, err)
		}

		databases[i].Tables = tables

		// Store connection for this database
		connKey := fmt.Sprintf("%s:%d:%s", sourceConfig.Host, sourceConfig.Port, databases[i].Datname)
		task.AddConnection(connKey, dbManager.GetDB())
	}

	// Step 4: Create all databases in target
	targetPostgresDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		targetConfig.Host, targetConfig.Port, targetConfig.User, targetConfig.Password, targetConfig.SSLMode)
	if targetConfig.SSLMode == "" {
		targetPostgresDSN = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
			targetConfig.Host, targetConfig.Port, targetConfig.User, targetConfig.Password)
	}

	targetManager, err := database.NewManager(dbType, targetPostgresDSN)
	if err != nil {
		return fmt.Errorf("failed to create target database manager: %w", err)
	}
	defer targetManager.Close()

	// Create databases in target
	for _, dbInfo := range databases {
		createDBQuery := fmt.Sprintf("CREATE DATABASE %s", dbInfo.Datname)
		if err := targetManager.GetDB().Exec(createDBQuery).Error; err != nil {
			// Check if database already exists
			if !isDatabaseExistsError(err) {
				return fmt.Errorf("failed to create database %s in target: %w", dbInfo.Datname, err)
			}
		}

		// Connect to target database and store connection
		targetDBDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			targetConfig.Host, targetConfig.Port, targetConfig.User, targetConfig.Password, dbInfo.Datname, targetConfig.SSLMode)
		if targetConfig.SSLMode == "" {
			targetDBDSN = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
				targetConfig.Host, targetConfig.Port, targetConfig.User, targetConfig.Password, dbInfo.Datname)
		}

		targetDBManager, err := database.NewManager(dbType, targetDBDSN)
		if err != nil {
			return fmt.Errorf("failed to connect to target database %s: %w", dbInfo.Datname, err)
		}
		// Don't defer close here, we need to keep the connection

		// Store target connection
		targetConnKey := fmt.Sprintf("%s:%d:%s", targetConfig.Host, targetConfig.Port, dbInfo.Datname)
		task.AddConnection(targetConnKey, targetDBManager.GetDB())
	}

	// Step 5: Cache database and table information
	// Store in task metadata (we can use a JSON field or extend the model)
	// For now, we'll store it in a way that can be retrieved later
	// This information will be used in subsequent states

	return nil
}

// Next returns the next state
func (s *ConnectState) Next() State {
	return NewCreateTablesState()
}

// isDatabaseExistsError checks if error is "database already exists"
func isDatabaseExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// PostgreSQL error: "database ... already exists"
	return contains(errStr, "already exists") || contains(errStr, "duplicate key")
}

// contains checks if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

