package model

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// MigrationTask 迁移任务
type MigrationTask struct {
	ID           string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	SourceDB     string     `gorm:"type:text;not null" json:"source_db"`   // JSON格式的源数据库配置
	TargetDB     string     `gorm:"type:text;not null" json:"target_db"`   // JSON格式的目标数据库配置
	Tables       string     `gorm:"type:text;not null" json:"tables"`      // JSON格式的表列表
	TableSuffix  string     `gorm:"type:varchar(100)" json:"table_suffix"` // 目标表后缀
	State        string     `gorm:"type:varchar(50);not null;default:'init'" json:"state"`
	Progress     int        `gorm:"default:0" json:"progress"` // 进度 0-100
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`

	// 运行时字段（不持久化）
	Connections map[string]interface{} `gorm:"-" json:"-"` // 数据库连接池 key: connectionKey (host:port:dbname), value: *sql.DB 或 *gorm.DB
	mu          sync.RWMutex           `gorm:"-" json:"-"` // 保护 connections 的并发访问
}

// TableName 指定表名
func (*MigrationTask) TableName() string {
	return "migration_tasks"
}

// BeforeCreate 创建前钩子
func (m *MigrationTask) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = generateUUID()
	}
	// 初始化连接池
	if m.Connections == nil {
		m.Connections = make(map[string]interface{})
	}
	return nil
}

// AfterFind 查询后钩子，初始化连接池
func (m *MigrationTask) AfterFind(tx *gorm.DB) error {
	if m.Connections == nil {
		m.Connections = make(map[string]interface{})
	}
	return nil
}

// ConnectionKey 生成连接键
func ConnectionKey(host string, port int, dbname string) string {
	return fmt.Sprintf("%s:%d:%s", host, port, dbname)
}

// AddConnection 添加数据库连接
func (m *MigrationTask) AddConnection(key string, conn interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Connections == nil {
		m.Connections = make(map[string]interface{})
	}
	m.Connections[key] = conn
}

// GetConnection 获取数据库连接
func (m *MigrationTask) GetConnection(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Connections == nil {
		return nil, false
	}
	conn, ok := m.Connections[key]
	return conn, ok
}

// CloseAllConnections 关闭所有连接
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
			// 未知类型的连接，记录警告但不报错
			continue
		}
	}

	// 清空连接池
	m.Connections = make(map[string]interface{})

	if len(errors) > 0 {
		return fmt.Errorf("errors closing connections: %v", errors)
	}

	return nil
}

// GetConnectionCount 获取连接数量（用于调试）
func (m *MigrationTask) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Connections == nil {
		return 0
	}
	return len(m.Connections)
}

// DBConfig 数据库配置
type DBConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

// DSN 返回数据库连接字符串
func (d *DBConfig) DSN() string {
	if d.SSLMode == "" {
		d.SSLMode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

// ConnectionKey 返回连接键
func (d *DBConfig) ConnectionKey() string {
	return ConnectionKey(d.Host, d.Port, d.DBName)
}

// TableInfo 表结构信息
type TableInfo struct {
	Schema      string           `json:"schema"`
	Name        string           `json:"name"`
	Columns     []ColumnInfo     `json:"columns"`
	Indexes     []IndexInfo      `json:"indexes"`
	Constraints []ConstraintInfo `json:"constraints"`
	DDL         string           `json:"ddl"`
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	IsNullable   bool   `json:"is_nullable"`
	DefaultValue string `json:"default_value"`
	IsPrimaryKey bool   `json:"is_primary_key"`
}

// IndexInfo 索引信息
type IndexInfo struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	DDL     string   `json:"ddl"`
}

// ConstraintInfo 约束信息
type ConstraintInfo struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"` // PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK
	Columns    []string `json:"columns"`
	Definition string   `json:"definition"`
}
