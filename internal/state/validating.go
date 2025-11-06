package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// ValidatingState 验证状态
type ValidatingState struct {
	BaseState
}

// NewValidatingState 创建验证状态
func NewValidatingState() *ValidatingState {
	return &ValidatingState{
		BaseState: BaseState{name: model.StateValidating.String()},
	}
}

// Execute 执行验证逻辑
func (s *ValidatingState) Execute(ctx context.Context, task *model.MigrationTask) error {
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

	// 验证每个表的行数
	schema := "public"
	for _, tableName := range tables {
		sourceTable := tableName
		targetTable := tableName + task.TableSuffix

		// 获取源表行数
		sourceCount, err := sourceRepo.GetTableCount(schema, sourceTable)
		if err != nil {
			return fmt.Errorf("failed to get source table count for %s: %w", tableName, err)
		}

		// 获取目标表行数
		targetCount, err := targetRepo.GetTableCount(schema, targetTable)
		if err != nil {
			return fmt.Errorf("failed to get target table count for %s: %w", tableName, err)
		}

		// 对比行数
		if sourceCount != targetCount {
			return fmt.Errorf("row count mismatch for table %s: source=%d, target=%d",
				tableName, sourceCount, targetCount)
		}
	}

	return nil
}

// Next 返回下一个状态
func (s *ValidatingState) Next() State {
	return NewFinalizingState()
}
