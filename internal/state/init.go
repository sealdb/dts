package state

import (
	"context"
	"fmt"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/repository"
)

// InitState 初始化状态
type InitState struct {
	BaseState
}

// NewInitState 创建初始化状态
func NewInitState() *InitState {
	return &InitState{
		BaseState: BaseState{name: model.StateInit.String()},
	}
}

// Execute 执行初始化逻辑
func (s *InitState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// 解析表列表
	tables, err := repository.ParseTables(task)
	if err != nil {
		return fmt.Errorf("failed to parse tables: %w", err)
	}

	// 验证源库连接和 wal_level（使用连接池，不关闭连接）
	sourceRepo, err := repository.NewSourceRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}
	// 注意：不在这里关闭连接，连接由任务管理器统一管理

	walLevel, err := sourceRepo.CheckWALLevel()
	if err != nil {
		return fmt.Errorf("failed to check wal_level: %w", err)
	}

	if walLevel != "logical" {
		return fmt.Errorf("source database wal_level must be 'logical', got '%s'", walLevel)
	}

	// 验证目标库连接（使用连接池，不关闭连接）
	_, err = repository.NewTargetRepositoryFromTask(task)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}
	// 注意：不在这里关闭连接，连接由任务管理器统一管理

	// 验证表是否存在
	schema := "public" // 默认schema，可以从配置中读取
	for _, tableName := range tables {
		_, err := sourceRepo.GetTableInfo(schema, tableName)
		if err != nil {
			return fmt.Errorf("table %s.%s not found or inaccessible: %w", schema, tableName, err)
		}
	}

	return nil
}

// Next 返回下一个状态
func (s *InitState) Next() State {
	return NewCreatingTablesState()
}

// CanTransition 判断是否可以转换
func (s *InitState) CanTransition() bool {
	return true
}
