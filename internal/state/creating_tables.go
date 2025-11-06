package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// CreatingTablesState 创建表状态
type CreatingTablesState struct {
	BaseState
}

// NewCreatingTablesState 创建创建表状态
func NewCreatingTablesState() *CreatingTablesState {
	return &CreatingTablesState{
		BaseState: BaseState{name: model.StateCreatingTables.String()},
	}
}

// Execute 执行创建表逻辑
func (s *CreatingTablesState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// 解析表列表
	tables, err := repository.ParseTables(task)
	if err != nil {
		return fmt.Errorf("failed to parse tables: %w", err)
	}

	// 创建仓储（使用连接池）
	sourceRepo, err := repository.NewSourceRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}
	// 连接由任务管理器统一管理，不在这里关闭

	targetRepo, err := repository.NewTargetRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}
	// 连接由任务管理器统一管理，不在这里关闭

	// 为每个表创建目标表
	schema := "public"
	for _, tableName := range tables {
		// 获取表结构
		tableInfo, err := sourceRepo.GetTableInfo(schema, tableName)
		if err != nil {
			return fmt.Errorf("failed to get table info for %s: %w", tableName, err)
		}

		// 创建目标表
		if err := targetRepo.CreateTable(tableInfo, task.TableSuffix); err != nil {
			return fmt.Errorf("failed to create target table for %s: %w", tableName, err)
		}
	}

	return nil
}

// Next 返回下一个状态
func (s *CreatingTablesState) Next() State {
	return NewMigratingDataState()
}
