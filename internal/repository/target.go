package repository

import (
	"fmt"
	"strings"

	"github.com/pg/dts/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TargetRepository handles target database operations
type TargetRepository struct {
	db *gorm.DB
}

// NewTargetRepository creates a target repository
func NewTargetRepository(dsn string) (*TargetRepository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target database: %w", err)
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
		return nil, fmt.Errorf("failed to ping target database: %w", err)
	}

	return &TargetRepository{db: db}, nil
}

// NewTargetRepositoryFromTask creates a target repository from task (using connection pool)
func NewTargetRepositoryFromTask(task *model.MigrationTask) (*TargetRepository, error) {
	db, err := GetOrCreateTargetGORMConnection(task)
	if err != nil {
		return nil, err
	}

	return &TargetRepository{db: db}, nil
}

// Close closes the connection
func (r *TargetRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB gets the underlying GORM DB (for special operations)
func (r *TargetRepository) GetDB() *gorm.DB {
	return r.db
}

// CreateTable creates a table
func (r *TargetRepository) CreateTable(tableInfo *model.TableInfo, suffix string) error {
	// Modify table name to tableName + suffix
	targetTableName := tableInfo.Name + suffix

	// Modify table name in DDL
	ddl := strings.Replace(tableInfo.DDL,
		fmt.Sprintf("%s.%s", tableInfo.Schema, tableInfo.Name),
		fmt.Sprintf("%s.%s", tableInfo.Schema, targetTableName),
		1)

	// Execute DDL
	if err := r.db.Exec(ddl).Error; err != nil {
		return fmt.Errorf("failed to create table %s: %w", targetTableName, err)
	}

	// Create indexes
	for _, idx := range tableInfo.Indexes {
		if err := r.createIndex(tableInfo.Schema, targetTableName, idx, suffix); err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.Name, err)
		}
	}

	// Create constraints (except primary key, already in DDL)
	for _, constraint := range tableInfo.Constraints {
		if constraint.Type != "PRIMARY KEY" {
			if err := r.createConstraint(tableInfo.Schema, targetTableName, constraint); err != nil {
				return fmt.Errorf("failed to create constraint %s: %w", constraint.Name, err)
			}
		}
	}

	return nil
}

// createIndex creates an index
func (r *TargetRepository) createIndex(schema, tableName string, index model.IndexInfo, suffix string) error {
	// Modify index name and table name
	indexName := index.Name + suffix
	indexDDL := strings.Replace(index.DDL, index.Name, indexName, 1)
	indexDDL = strings.Replace(indexDDL,
		fmt.Sprintf("ON %s.%s", schema, tableName),
		fmt.Sprintf("ON %s.%s", schema, tableName),
		1)

	return r.db.Exec(indexDDL).Error
}

// createConstraint creates a constraint
func (r *TargetRepository) createConstraint(schema, tableName string, constraint model.ConstraintInfo) error {
	var constraintDDL string

	switch constraint.Type {
	case "UNIQUE":
		constraintDDL = fmt.Sprintf("ALTER TABLE %s.%s ADD CONSTRAINT %s UNIQUE (%s)",
			schema, tableName, constraint.Name, strings.Join(constraint.Columns, ", "))
	case "CHECK":
		constraintDDL = fmt.Sprintf("ALTER TABLE %s.%s ADD CONSTRAINT %s CHECK (%s)",
			schema, tableName, constraint.Name, constraint.Definition)
	case "FOREIGN KEY":
		// Foreign keys require more complex handling, simplified here
		constraintDDL = fmt.Sprintf("ALTER TABLE %s.%s ADD CONSTRAINT %s %s",
			schema, tableName, constraint.Name, constraint.Definition)
	default:
		return fmt.Errorf("unsupported constraint type: %s", constraint.Type)
	}

	return r.db.Exec(constraintDDL).Error
}

// GetTableCount gets table row count
func (r *TargetRepository) GetTableCount(schema, tableName string) (int64, error) {
	var count int64
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s.%s`, schema, tableName)
	err := r.db.Raw(query).Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get table count: %w", err)
	}
	return count, nil
}

// CopyData copies data
func (r *TargetRepository) CopyData(sourceRepo *SourceRepository, sourceSchema, sourceTable, targetSchema, targetTable string) error {
	// Get source table column information
	tableInfo, err := sourceRepo.GetTableInfo(sourceSchema, sourceTable)
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}

	// Build column name list
	var columns []string
	for _, col := range tableInfo.Columns {
		columns = append(columns, col.Name)
	}
	_ = strings.Join

	// Use COPY command to copy data (reserved for future optimization)
	// Simplified here to batch read + insert
	// Need to get source database pgx.Conn connection
	// Simplified implementation: use batch query and insert
	return r.copyDataBatched(sourceRepo.db, sourceSchema, sourceTable, targetSchema, targetTable, columns)
}

// copyDataBatched copies data in batches
func (r *TargetRepository) copyDataBatched(sourceDB *gorm.DB, sourceSchema, sourceTable, targetSchema, targetTable string, columns []string) error {
	batchSize := 1000
	offset := 0

	for {
		// Query a batch of data from source database
		query := fmt.Sprintf("SELECT %s FROM %s.%s ORDER BY 1 LIMIT ? OFFSET ?",
			strings.Join(columns, ", "), sourceSchema, sourceTable)

		type Row map[string]interface{}
		var rows []Row

		// Use GORM to query
		if err := sourceDB.Raw(query, batchSize, offset).Scan(&rows).Error; err != nil {
			return fmt.Errorf("failed to query source: %w", err)
		}

		if len(rows) == 0 {
			break
		}

		// Convert to batch insert format
		batch := make([][]interface{}, len(rows))
		for i, row := range rows {
			values := make([]interface{}, len(columns))
			for j, col := range columns {
				values[j] = row[col]
			}
			batch[i] = values
		}

		// Batch insert into target database
		if err := r.batchInsert(targetSchema, targetTable, columns, batch); err != nil {
			return fmt.Errorf("failed to insert batch: %w", err)
		}

		// If batch size is less than batchSize, all data has been read
		if len(batch) < batchSize {
			break
		}

		offset += len(batch)
	}

	return nil
}

// batchInsert performs batch insert
func (r *TargetRepository) batchInsert(schema, table string, columns []string, batch [][]interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	// Build INSERT statement
	placeholders := make([]string, len(batch))
	args := make([]interface{}, 0, len(batch)*len(columns))

	for i, row := range batch {
		rowPlaceholders := make([]string, len(columns))
		for j := range rowPlaceholders {
			rowPlaceholders[j] = fmt.Sprintf("$%d", i*len(columns)+j+1)
			args = append(args, row[j])
		}
		placeholders[i] = "(" + strings.Join(rowPlaceholders, ", ") + ")"
	}

	query := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES %s",
		schema, table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	return r.db.Exec(query, args...).Error
}

// ApplyInsert applies insert operation
func (r *TargetRepository) ApplyInsert(schema, tableName string, values map[string]interface{}) error {
	if len(values) == 0 {
		return nil
	}
	cols := make([]string, 0, len(values))
	args := make([]interface{}, 0, len(values))
	placeholders := make([]string, 0, len(values))
	i := 1
	for k, v := range values {
		cols = append(cols, k)
		args = append(args, v)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		i++
	}
	query := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)",
		schema, tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	return r.db.Exec(query, args...).Error
}

// ApplyUpdate applies update operation
func (r *TargetRepository) ApplyUpdate(schema, tableName string, oldValues, newValues map[string]interface{}) error {
	if len(newValues) == 0 || len(oldValues) == 0 {
		return nil
	}
	setClauses := make([]string, 0, len(newValues))
	whereClauses := make([]string, 0, len(oldValues))
	args := make([]interface{}, 0, len(newValues)+len(oldValues))
	i := 1
	for k, v := range newValues {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", k, i))
		args = append(args, v)
		i++
	}
	for k, v := range oldValues {
		whereClauses = append(whereClauses, fmt.Sprintf("%s IS NOT DISTINCT FROM $%d", k, i))
		args = append(args, v)
		i++
	}
	query := fmt.Sprintf("UPDATE %s.%s SET %s WHERE %s",
		schema, tableName, strings.Join(setClauses, ", "), strings.Join(whereClauses, " AND "))
	return r.db.Exec(query, args...).Error
}

// ApplyDelete applies delete operation
func (r *TargetRepository) ApplyDelete(schema, tableName string, values map[string]interface{}) error {
	if len(values) == 0 {
		return nil
	}
	whereClauses := make([]string, 0, len(values))
	args := make([]interface{}, 0, len(values))
	i := 1
	for k, v := range values {
		whereClauses = append(whereClauses, fmt.Sprintf("%s IS NOT DISTINCT FROM $%d", k, i))
		args = append(args, v)
		i++
	}
	query := fmt.Sprintf("DELETE FROM %s.%s WHERE %s", schema, tableName, strings.Join(whereClauses, " AND "))
	return r.db.Exec(query, args...).Error
}
