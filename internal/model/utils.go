package model

import (
	"time"

	"github.com/google/uuid"
)

// generateUUID generates a UUID
func generateUUID() string {
	return uuid.New().String()
}

// StateType represents state type
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

// String returns the string representation of the state
func (s StateType) String() string {
	return string(s)
}

// IsTerminal checks if the state is terminal
func (s StateType) IsTerminal() bool {
	return s == StateCompleted || s == StateFailed
}

// CanTransition checks if can transition to target state
func (s StateType) CanTransition(target StateType) bool {
	// Define state transition rules
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

// GetStateDisplayName gets the display name of the state
func GetStateDisplayName(state StateType) string {
	names := map[StateType]string{
		StateInit:           "Initializing",
		StateCreatingTables: "Creating target tables",
		StateMigratingData:  "Migrating data",
		StateSyncingWAL:     "Syncing WAL logs",
		StateStoppingWrites: "Stopping source database writes",
		StateValidating:     "Validating data",
		StateFinalizing:     "Finalizing migration",
		StateCompleted:      "Completed",
		StateFailed:         "Failed",
		StatePaused:         "Paused",
	}

	if name, ok := names[state]; ok {
		return name
	}
	return string(state)
}

// UpdateTaskState updates task state
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
