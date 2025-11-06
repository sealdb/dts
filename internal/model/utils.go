package model

import (
	"time"

	"github.com/google/uuid"
)

// generateUUID 生成UUID
func generateUUID() string {
	return uuid.New().String()
}

// StateType 状态类型
type StateType string

const (
	StateInit           StateType = "init"
	StateCreatingTables StateType = "creating_tables"
	StateMigratingData  StateType = "migrating_data"
	StateSyncingWAL     StateType = "syncing_wal"
	StateStoppingWrites StateType = "stopping_writes"
	StateValidating     StateType = "validating"
	StateFinalizing     StateType = "finalizing"
	StateCompleted      StateType = "completed"
	StateFailed         StateType = "failed"
	StatePaused         StateType = "paused"
)

// String 返回状态的字符串表示
func (s StateType) String() string {
	return string(s)
}

// IsTerminal 判断是否为终止状态
func (s StateType) IsTerminal() bool {
	return s == StateCompleted || s == StateFailed
}

// CanTransition 判断是否可以转换到目标状态
func (s StateType) CanTransition(target StateType) bool {
	// 定义状态转换规则
	transitions := map[StateType][]StateType{
		StateInit:           {StateCreatingTables, StateFailed},
		StateCreatingTables: {StateMigratingData, StateFailed, StatePaused},
		StateMigratingData:  {StateSyncingWAL, StateFailed, StatePaused},
		StateSyncingWAL:     {StateStoppingWrites, StateFailed, StatePaused},
		StateStoppingWrites: {StateValidating, StateFailed},
		StateValidating:     {StateFinalizing, StateFailed},
		StateFinalizing:     {StateCompleted, StateFailed},
		StatePaused:         {StateMigratingData, StateSyncingWAL, StateFailed},
	}

	allowed, exists := transitions[s]
	if !exists {
		return false
	}

	for _, state := range allowed {
		if state == target {
			return true
		}
	}

	return false
}

// GetStateDisplayName 获取状态的显示名称
func GetStateDisplayName(state StateType) string {
	names := map[StateType]string{
		StateInit:           "初始化",
		StateCreatingTables: "创建目标表",
		StateMigratingData:  "迁移数据",
		StateSyncingWAL:     "同步WAL日志",
		StateStoppingWrites: "停止源库写操作",
		StateValidating:     "数据校验",
		StateFinalizing:     "完成迁移",
		StateCompleted:      "已完成",
		StateFailed:         "失败",
		StatePaused:         "已暂停",
	}

	if name, ok := names[state]; ok {
		return name
	}
	return string(state)
}

// UpdateTaskState 更新任务状态
func UpdateTaskState(task *MigrationTask, newState StateType, errorMsg string) {
	task.State = newState.String()
	if errorMsg != "" {
		task.ErrorMessage = errorMsg
	}

	now := time.Now()
	if newState == StateMigratingData && task.StartedAt == nil {
		task.StartedAt = &now
	}
	if newState.IsTerminal() {
		task.CompletedAt = &now
	}
	task.UpdatedAt = now
}
