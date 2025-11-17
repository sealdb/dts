package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pg/dts/internal/database"
	"github.com/pg/dts/internal/model"
	"gorm.io/gorm"
)

// CreateTablesState represents the create tables state
type CreateTablesState struct {
	BaseState
}

// NewCreateTablesState creates a new create tables state
func NewCreateTablesState() *CreateTablesState {
	return &CreateTablesState{
		BaseState: BaseState{name: model.StateCreateTables.String()},
	}
}

// Execute executes the table creation logic
func (s *CreateTablesState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// Parse database type
	dbType := database.DatabaseType(task.DatabaseType)
	if dbType == "" {
		dbType = database.DatabaseTypePostgreSQL // Default to PostgreSQL
	}

	// Parse source database configuration
	var sourceConfig model.DBConfig
	if err := json.Unmarshal([]byte(task.SourceDB), &sourceConfig); err != nil {
		return fmt.Errorf("failed to parse source database config: %w", err)
	}

	// Parse target database configuration
	var targetConfig model.DBConfig
	if err := json.Unmarshal([]byte(task.TargetDB), &targetConfig); err != nil {
		return fmt.Errorf("failed to parse target database config: %w", err)
	}

	// For PostgreSQL, use pg_dump to get table definitions
	if dbType == database.DatabaseTypePostgreSQL {
		return s.createTablesForPostgreSQL(ctx, task, &sourceConfig, &targetConfig)
	}

	// For other database types, use existing repository approach
	return s.createTablesGeneric(ctx, task)
}

// createTablesForPostgreSQL creates tables using pg_dump for PostgreSQL
func (s *CreateTablesState) createTablesForPostgreSQL(ctx context.Context, task *model.MigrationTask, sourceConfig, targetConfig *model.DBConfig) error {
	// Parse table list - we need to get all tables from all databases
	// For now, we'll iterate through connections stored in task
	// The connections were created in ConnectState

	// Get all source connections (format: host:port:database)
	for connKey, conn := range task.Connections {
		if conn == nil {
			continue
		}

		_, ok := conn.(*gorm.DB)
		if !ok {
			continue
		}

		// Parse connection key to get database name
		parts := strings.Split(connKey, ":")
		if len(parts) < 3 {
			continue
		}
		databaseName := parts[2]

		// Check if this is a source connection (by comparing host:port)
		sourceKey := fmt.Sprintf("%s:%d", sourceConfig.Host, sourceConfig.Port)
		if !strings.HasPrefix(connKey, sourceKey) {
			continue
		}

		// Get tables for this database from task metadata or query again
		// For now, we'll need to query tables from the database
		// TODO: Use cached table information from ConnectState

		// Use pg_dump to get schema for all tables in this database
		pgDumpCmd := exec.CommandContext(ctx, "pg_dump",
			"-h", sourceConfig.Host,
			"-p", fmt.Sprintf("%d", sourceConfig.Port),
			"-U", sourceConfig.User,
			"-d", databaseName,
			"--schema-only",
			"--no-owner",
			"--no-privileges",
		)
		pgDumpCmd.Env = append(pgDumpCmd.Env, fmt.Sprintf("PGPASSWORD=%s", sourceConfig.Password))

		schemaSQL, err := pgDumpCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to run pg_dump for database %s: %w", databaseName, err)
		}

		// Modify table names in schema SQL
		// Replace table names with new names (table + suffix)
		modifiedSQL := s.modifyTableNames(string(schemaSQL), task.TableSuffix)

		// Get target connection for this database
		targetConnKey := fmt.Sprintf("%s:%d:%s", targetConfig.Host, targetConfig.Port, databaseName)
		targetConn, ok := task.GetConnection(targetConnKey)
		if !ok {
			return fmt.Errorf("target connection not found for database %s", databaseName)
		}

		targetGormDB, ok := targetConn.(*gorm.DB)
		if !ok {
			return fmt.Errorf("invalid target connection type for database %s", databaseName)
		}

		// Split by semicolon and execute each statement
		statements := strings.Split(modifiedSQL, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || strings.HasPrefix(stmt, "--") {
				continue
			}
			if err := targetGormDB.Exec(stmt).Error; err != nil {
				// Some statements might fail (e.g., if table already exists), log but continue
				// TODO: Better error handling
				continue
			}
		}
	}

	return nil
}

// modifyTableNames modifies table names in SQL schema
// Replaces table names with new names (table + suffix)
// Also modifies index names, constraint names that reference table names
func (s *CreateTablesState) modifyTableNames(schemaSQL, suffix string) string {
	if suffix == "" {
		return schemaSQL
	}

	// Simple approach: replace table names in CREATE TABLE, ALTER TABLE, CREATE INDEX statements
	// This is a simplified implementation - a more robust solution would use SQL parser
	lines := strings.Split(schemaSQL, "\n")
	var modifiedLines []string

	for _, line := range lines {
		modifiedLine := line

		// Match CREATE TABLE statements
		if strings.Contains(line, "CREATE TABLE") {
			// Extract table name and replace
			// Format: CREATE TABLE public.tablename ( or CREATE TABLE tablename (
			modifiedLine = s.replaceTableNameInLine(line, suffix)
		}

		// Match ALTER TABLE statements
		if strings.Contains(line, "ALTER TABLE") {
			modifiedLine = s.replaceTableNameInLine(line, suffix)
		}

		// Match CREATE INDEX statements
		if strings.Contains(line, "CREATE INDEX") {
			// Extract index name and table name
			modifiedLine = s.replaceIndexNameInLine(line, suffix)
		}

		// Match ALTER TABLE ... ADD CONSTRAINT statements
		if strings.Contains(line, "ADD CONSTRAINT") {
			modifiedLine = s.replaceConstraintNameInLine(line, suffix)
		}

		modifiedLines = append(modifiedLines, modifiedLine)
	}

	return strings.Join(modifiedLines, "\n")
}

// replaceTableNameInLine replaces table name in a line
func (s *CreateTablesState) replaceTableNameInLine(line, suffix string) string {
	// Simple regex-like replacement
	// This is a simplified implementation
	// Match patterns like: public.tablename or tablename
	// Replace with: public.tablename_suffix or tablename_suffix

	// For now, we'll do a simple string replacement
	// A more robust solution would parse the SQL properly
	words := strings.Fields(line)
	for i, word := range words {
		// Check if word contains a table name pattern
		if strings.Contains(word, ".") {
			// Format: schema.tablename
			parts := strings.Split(word, ".")
			if len(parts) == 2 {
				// Replace tablename with tablename + suffix
				words[i] = parts[0] + "." + parts[1] + suffix
			}
		} else if i > 0 && (words[i-1] == "TABLE" || words[i-1] == "ON") {
			// Might be a table name
			// Simple heuristic: if previous word is TABLE or ON, this might be table name
			words[i] = word + suffix
		}
	}
	return strings.Join(words, " ")
}

// replaceIndexNameInLine replaces index name in a line
func (s *CreateTablesState) replaceIndexNameInLine(line, suffix string) string {
	// Match: CREATE INDEX indexname ON tablename
	// Replace indexname if it contains table name pattern (e.g., tablename_pkey -> tablename_suffix_pkey)
	words := strings.Fields(line)
	for i, word := range words {
		if i > 0 && words[i-1] == "INDEX" {
			// This is index name
			// If index name contains table name pattern, replace it
			words[i] = word + suffix
		} else if i > 1 && words[i-2] == "INDEX" && words[i-1] == "ON" {
			// This is table name after ON
			words[i] = word + suffix
		}
	}
	return strings.Join(words, " ")
}

// replaceConstraintNameInLine replaces constraint name in a line
func (s *CreateTablesState) replaceConstraintNameInLine(line, suffix string) string {
	// Similar to replaceIndexNameInLine
	return s.replaceIndexNameInLine(line, suffix)
}

// createTablesGeneric creates tables using generic repository approach
func (s *CreateTablesState) createTablesGeneric(ctx context.Context, task *model.MigrationTask) error {
	// Use existing repository approach for non-PostgreSQL databases
	// This is a fallback implementation
	// TODO: Implement for MySQL and other databases
	return fmt.Errorf("generic table creation not implemented yet")
}

// Next returns the next state
func (s *CreateTablesState) Next() State {
	return NewFullSyncState()
}

