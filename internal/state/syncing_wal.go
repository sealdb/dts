package state

import (
	"context"
	"fmt"
	"time"

	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/replication"
	"github.com/pg/dts/internal/repository"
)

// SyncingWALState 同步WAL状态
type SyncingWALState struct {
	BaseState
}

// NewSyncingWALState 创建同步WAL状态
func NewSyncingWALState() *SyncingWALState {
	return &SyncingWALState{
		BaseState: BaseState{name: model.StateSyncingWAL.String()},
	}
}

// Execute 执行WAL同步逻辑
func (s *SyncingWALState) Execute(ctx context.Context, task *model.MigrationTask) error {
	// 解析表列表
	tables, err := repository.ParseTables(task)
	if err != nil {
		return fmt.Errorf("failed to parse tables: %w", err)
	}

	// 获取或创建源库 GORM 连接（使用连接池）
	sourceDB, err := repository.GetOrCreateSourceGORMConnection(task)
	if err != nil {
		return fmt.Errorf("failed to get source connection: %w", err)
	}

	// 确保连接有效
	sqlDB, err := sourceDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("source connection is not valid: %w", err)
	}

	// 创建复制槽管理器（使用已有连接）
	slotManager, err := replication.NewSlotManagerFromDB(sourceDB)
	if err != nil {
		return fmt.Errorf("failed to create slot manager: %w", err)
	}
	// 注意：slotManager 使用共享连接，不单独关闭

	// 创建发布管理器（使用已有连接）
	pubManager, err := replication.NewPublicationManagerFromDB(sourceDB)
	if err != nil {
		return fmt.Errorf("failed to create publication manager: %w", err)
	}
	// 注意：pubManager 使用共享连接，不单独关闭

	// 生成复制槽和发布名称
	slotName := fmt.Sprintf("dts_slot_%s", task.ID)
	pubName := fmt.Sprintf("dts_pub_%s", task.ID)

	// 检查并创建复制槽
	exists, err := slotManager.SlotExists(slotName)
	if err != nil {
		return fmt.Errorf("failed to check slot existence: %w", err)
	}

	if !exists {
		if err := slotManager.CreateSlot(slotName, "pgoutput"); err != nil {
			return fmt.Errorf("failed to create replication slot: %w", err)
		}
	}

	// 检查并创建发布
	exists, err = pubManager.PublicationExists(pubName)
	if err != nil {
		return fmt.Errorf("failed to check publication existence: %w", err)
	}

	if !exists {
		// 构建表名列表（格式：schema.table）
		schema := "public"
		tableNames := make([]string, len(tables))
		for i, table := range tables {
			tableNames[i] = fmt.Sprintf("%s.%s", schema, table)
		}

		if err := pubManager.CreatePublication(pubName, tableNames); err != nil {
			return fmt.Errorf("failed to create publication: %w", err)
		}
	}

	// 创建订阅者并开始同步
	// 注意：这里需要启动一个后台 goroutine 来处理 WAL 流
	// 实际实现中，应该使用 context 来控制同步的停止
	// 这里简化实现，同步一段时间后返回
	// 实际应该持续运行直到 StoppingWrites 状态

	// TODO: 启动 WAL 订阅者
	// subscriber, err := replication.NewSubscriber(sourceDB.DSN(), slotName)
	// if err != nil {
	// 	return fmt.Errorf("failed to create subscriber: %w", err)
	// }
	// defer subscriber.Close()
	//
	// if err := subscriber.StartReplication(ctx, pubName); err != nil {
	// 	return fmt.Errorf("failed to start replication: %w", err)
	// }
	//
	// // 在后台处理复制流
	// go func() {
	// 	if err := subscriber.ProcessReplicationStream(ctx); err != nil {
	// 		// 处理错误
	// 	}
	// }()

	// 等待一段时间让 WAL 同步（实际应该持续运行）
	// 这里简化实现
	time.Sleep(1 * time.Second)

	return nil
}

// Next 返回下一个状态
func (s *SyncingWALState) Next() State {
	return NewStoppingWritesState()
}
