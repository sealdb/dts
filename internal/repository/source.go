package repository

import (
	"fmt"
	"strings"

	"github.com/pg/dts/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SourceRepository handles source database operations
type SourceRepository struct {
	db *gorm.DB
}

// NewSourceRepository creates a source repository
func NewSourceRepository(dsn string) (*SourceRepository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to source database: %w", err)
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
		return nil, fmt.Errorf("failed to ping source database: %w", err)
	}

	return &SourceRepository{db: db}, nil
}

// NewSourceRepositoryFromTask creates a source repository from task (using connection pool)
func NewSourceRepositoryFromTask(task *model.MigrationTask) (*SourceRepository, error) {
	db, err := GetOrCreateSourceGORMConnection(task)
	if err != nil {
		return nil, err
	}

	return &SourceRepository{db: db}, nil
}

// Close closes the connection
func (r *SourceRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB gets the underlying GORM DB (for special operations)
func (r *SourceRepository) GetDB() *gorm.DB {
	return r.db
}

// CheckWALLevel checks WAL level
func (r *SourceRepository) CheckWALLevel() (string, error) {
	var walLevel string
	err := r.db.Raw("SHOW wal_level").Scan(&walLevel).Error
	if err != nil {
		return "", fmt.Errorf("failed to check wal_level: %w", err)
	}
	return walLevel, nil
}

// GetTableInfo gets table structure information
func (r *SourceRepository) GetTableInfo(schema, tableName string) (*model.TableInfo, error) {
	tableInfo := &model.TableInfo{
		Schema:      schema,
		Name:        tableName,
		Columns:     []model.ColumnInfo{},
		Indexes:     []model.IndexInfo{},
		Constraints: []model.ConstraintInfo{},
	}

	// Get column information
	columns, err := r.getColumns(schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	tableInfo.Columns = columns

	// Get index information
	indexes, err := r.getIndexes(schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	tableInfo.Indexes = indexes

	// Get constraint information
	constraints, err := r.getConstraints(schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get constraints: %w", err)
	}
	tableInfo.Constraints = constraints

	// Generate DDL
	ddl, err := r.generateDDL(tableInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to generate DDL: %w", err)
	}
	tableInfo.DDL = ddl

	return tableInfo, nil
}

// getColumns gets column information
func (r *SourceRepository) getColumns(schema, tableName string) ([]model.ColumnInfo, error) {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_primary_key
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT ku.table_schema, ku.table_name, ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku
				ON tc.constraint_name = ku.constraint_name
				AND tc.table_schema = ku.table_schema
			WHERE tc.constraint_type = 'PRIMARY KEY'
		) pk ON c.table_schema = pk.table_schema
			AND c.table_name = pk.table_name
			AND c.column_name = pk.column_name
		WHERE c.table_schema = ? AND c.table_name = ?
		ORDER BY c.ordinal_position
	`

	type ColumnRow struct {
		Name         string
		DataType     string
		IsNullable   string
		DefaultValue *string
		IsPrimaryKey bool
	}

	var rows []ColumnRow
	if err := r.db.Raw(query, schema, tableName).Scan(&rows).Error; err != nil {
		return nil, err
	}

	columns := make([]model.ColumnInfo, len(rows))
	for i, row := range rows {
		columns[i] = model.ColumnInfo{
			Name:         row.Name,
			DataType:     row.DataType,
			IsNullable:   row.IsNullable == "YES",
			DefaultValue: "",
			IsPrimaryKey: row.IsPrimaryKey,
		}
		if row.DefaultValue != nil {
			columns[i].DefaultValue = *row.DefaultValue
		}
	}

	return columns, nil
}

// getIndexes gets index information
func (r *SourceRepository) getIndexes(schema, tableName string) ([]model.IndexInfo, error) {
	query := `
		SELECT
			i.indexname,
			i.indexdef,
			i.indexdef LIKE '%UNIQUE%' as is_unique
		FROM pg_indexes i
		WHERE i.schemaname = ? AND i.tablename = ?
		AND i.indexname NOT LIKE '%_pkey'
		ORDER BY i.indexname
	`

	type IndexRow struct {
		Name     string
		IndexDef string
		IsUnique bool
	}

	var rows []IndexRow
	if err := r.db.Raw(query, schema, tableName).Scan(&rows).Error; err != nil {
		return nil, err
	}

	indexes := make([]model.IndexInfo, len(rows))
	for i, row := range rows {
		indexes[i] = model.IndexInfo{
			Name:    row.Name,
			DDL:     row.IndexDef,
			Unique:  row.IsUnique,
			Columns: extractColumnsFromIndexDef(row.IndexDef),
		}
	}

	return indexes, nil
}

// getConstraints gets constraint information
func (r *SourceRepository) getConstraints(schema, tableName string) ([]model.ConstraintInfo, error) {
	query := `
		SELECT
			tc.constraint_name,
			tc.constraint_type,
			STRING_AGG(kcu.column_name, ', ' ORDER BY kcu.ordinal_position) as columns,
			cc.check_clause
		FROM information_schema.table_constraints tc
		LEFT JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		LEFT JOIN information_schema.check_constraints cc
			ON tc.constraint_name = cc.constraint_name
		WHERE tc.table_schema = ?
			AND tc.table_name = ?
			AND tc.constraint_type != 'PRIMARY KEY'
		GROUP BY tc.constraint_name, tc.constraint_type, cc.check_clause
		ORDER BY tc.constraint_name
	`

	type ConstraintRow struct {
		Name        string
		Type        string
		Columns     *string
		CheckClause *string
	}

	var rows []ConstraintRow
	if err := r.db.Raw(query, schema, tableName).Scan(&rows).Error; err != nil {
		return nil, err
	}

	constraints := make([]model.ConstraintInfo, len(rows))
	for i, row := range rows {
		constraints[i] = model.ConstraintInfo{
			Name:       row.Name,
			Type:       row.Type,
			Columns:    []string{},
			Definition: "",
		}

		if row.Columns != nil {
			constraints[i].Columns = parseStringArray(*row.Columns)
		}

		if row.CheckClause != nil {
			constraints[i].Definition = *row.CheckClause
		}
	}

	return constraints, nil
}

// generateDDL generates CREATE TABLE DDL
func (r *SourceRepository) generateDDL(tableInfo *model.TableInfo) (string, error) {
	var ddl strings.Builder

	ddl.WriteString(fmt.Sprintf("CREATE TABLE %s.%s (\n", tableInfo.Schema, tableInfo.Name))

	// Column definitions
	var columnDefs []string
	for _, col := range tableInfo.Columns {
		def := fmt.Sprintf("  %s %s", col.Name, col.DataType)

		// Add NOT NULL
		if !col.IsNullable {
			def += " NOT NULL"
		}

		// Add default value
		if col.DefaultValue != "" {
			def += " DEFAULT " + col.DefaultValue
		}

		columnDefs = append(columnDefs, def)
	}

	// Add primary key constraint
	var pkColumns []string
	for _, col := range tableInfo.Columns {
		if col.IsPrimaryKey {
			pkColumns = append(pkColumns, col.Name)
		}
	}
	if len(pkColumns) > 0 {
		columnDefs = append(columnDefs, fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(pkColumns, ", ")))
	}

	ddl.WriteString(strings.Join(columnDefs, ",\n"))
	ddl.WriteString("\n)")

	return ddl.String(), nil
}

// extractColumnsFromIndexDef extracts column names from index definition
func extractColumnsFromIndexDef(indexDef string) []string {
	// Simple implementation: extract column names from CREATE INDEX ... ON table (col1, col2)
	// More complex implementation can use SQL parser
	start := strings.Index(indexDef, "(")
	end := strings.Index(indexDef, ")")
	if start == -1 || end == -1 {
		return []string{}
	}

	colsStr := indexDef[start+1 : end]
	cols := strings.Split(colsStr, ",")

	var result []string
	for _, col := range cols {
		col = strings.TrimSpace(col)
		// Remove possible sort direction (ASC/DESC)
		col = strings.TrimSuffix(strings.TrimSuffix(col, " ASC"), " DESC")
		result = append(result, strings.TrimSpace(col))
	}

	return result
}

// parseStringArray parses comma-separated string array
func parseStringArray(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ", ")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// GetTableCount gets table row count
func (r *SourceRepository) GetTableCount(schema, tableName string) (int64, error) {
	var count int64
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s.%s`, schema, tableName)
	err := r.db.Raw(query).Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get table count: %w", err)
	}
	return count, nil
}

// SetReadOnly sets database to read-only
func (r *SourceRepository) SetReadOnly() error {
	err := r.db.Exec("ALTER DATABASE current_database() SET default_transaction_read_only = true").Error
	if err != nil {
		return fmt.Errorf("failed to set database read-only: %w", err)
	}
	return nil
}

// RevokeWritePermissions revokes write permissions
func (r *SourceRepository) RevokeWritePermissions(schema string, tables []string) error {
	// TODO: Implement revoke write permissions
	return fmt.Errorf("not implemented")
}

// RestoreWritePermissions restores write permissions
func (r *SourceRepository) RestoreWritePermissions() error {
	err := r.db.Exec("ALTER DATABASE current_database() RESET default_transaction_read_only").Error
	if err != nil {
		return fmt.Errorf("failed to restore database write permissions: %w", err)
	}
	return nil
}

// GetAllTables gets all tables under specified schema
func (r *SourceRepository) GetAllTables(schema string) ([]string, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ?
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	var tables []string
	if err := r.db.Raw(query, schema).Scan(&tables).Error; err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	return tables, nil
}
