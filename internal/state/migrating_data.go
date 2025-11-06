package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// MigratingDataState 迁移数据状态
type MigratingDataState struct {
	BaseState
}

// NewMigratingDataState 创建迁移数据状态
func NewMigratingDataState() *MigratingDataState {
	return &MigratingDataState{
		BaseState: BaseState{name: model.StateMigratingData.String()},
	}
}

// Execute 执行数据迁移逻辑
func (s *MigratingDataState) Execute(ctx context.Context, task *model.MigrationTask) error {
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

	// 迁移每个表的数据
	schema := "public"
	for i, tableName := range tables {
		sourceTable := tableName
		targetTable := tableName + task.TableSuffix

		if err := targetRepo.CopyData(sourceRepo, schema, sourceTable, schema, targetTable); err != nil {
			return fmt.Errorf("failed to copy data for table %s: %w", tableName, err)
		}

		// 更新进度（简单实现，实际可以更精确）
		progress := (i + 1) * 100 / len(tables)
		// TODO: 更新任务进度到数据库
		_ = progress
	}

	return nil
}

// Next 返回下一个状态
func (s *MigratingDataState) Next() State {
	return NewSyncingWALState()
}
