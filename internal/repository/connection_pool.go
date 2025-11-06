package repository

import (
	"fmt"

	"github.com/pg/dts/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// GetOrCreateGORMConnection 获取或创建 GORM 数据库连接（带连接池管理）
func GetOrCreateGORMConnection(task *model.MigrationTask, dbConfig *model.DBConfig) (*gorm.DB, error) {
	connectionKey := dbConfig.ConnectionKey()

	// 尝试从任务中获取已有连接
	if conn, ok := task.GetConnection(connectionKey); ok {
		if gormDB, ok := conn.(*gorm.DB); ok {
			// 验证连接是否仍然有效
			sqlDB, err := gormDB.DB()
			if err == nil {
				if err := sqlDB.Ping(); err == nil {
					return gormDB, nil
				}
			}
			// 连接无效，需要重新创建
		}
	}

	// 创建新连接
	db, err := gorm.Open(postgres.Open(dbConfig.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// 设置连接池参数
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)

	// 验证连接
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 将连接添加到任务连接池
	task.AddConnection(connectionKey, db)

	return db, nil
}

// GetOrCreateSourceGORMConnection 获取或创建源库 GORM 连接
func GetOrCreateSourceGORMConnection(task *model.MigrationTask) (*gorm.DB, error) {
	sourceDB, err := ParseSourceDB(task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source db config: %w", err)
	}

	return GetOrCreateGORMConnection(task, sourceDB)
}

// GetOrCreateTargetGORMConnection 获取或创建目标库 GORM 连接
func GetOrCreateTargetGORMConnection(task *model.MigrationTask) (*gorm.DB, error) {
	targetDB, err := ParseTargetDB(task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target db config: %w", err)
	}

	return GetOrCreateGORMConnection(task, targetDB)
}

// GetOrCreateSourceConnection 获取或创建源库连接（兼容旧接口，返回 sql.DB）
// 注意：此方法主要用于需要 pgx 原生 API 的场景（如 COPY FROM STDIN）
func GetOrCreateSourceConnection(task *model.MigrationTask) (*gorm.DB, error) {
	return GetOrCreateSourceGORMConnection(task)
}

// GetOrCreateTargetConnection 获取或创建目标库连接（兼容旧接口，返回 sql.DB）
// 注意：此方法主要用于需要 pgx 原生 API 的场景（如 COPY FROM STDIN）
func GetOrCreateTargetConnection(task *model.MigrationTask) (*gorm.DB, error) {
	return GetOrCreateTargetGORMConnection(task)
}
