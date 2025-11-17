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
	StateInit         StateType = "init"
	StateConnect      StateType = "connect"
	StateCreateTables StateType = "create_tables"
	StateFullSync     StateType = "full_sync"
	StateIncSync      StateType = "inc_sync"
	StateWaiting      StateType = "waiting"
	StateValidating   StateType = "validating"
	StateCompleted    StateType = "completed"
	StateFailed       StateType = "failed"
	StatePaused       StateType = "paused"
	StateDeleted      StateType = "deleted"
)

// String returns the string representation of the state
func (s StateType) String() string {
	return string(s)
}

// IsTerminal checks if the state is terminal
func (s StateType) IsTerminal() bool {
	return s == StateCompleted || s == StateFailed || s == StateDeleted
}

// CanTransition checks if can transition to target state
func (s StateType) CanTransition(target StateType) bool {
	// Define state transition rules
	transitions := map[StateType][]StateType{
		StateInit:       {StateConnect, StateFailed},
		StateConnect:    {StateCreateTables, StateFailed, StatePaused},
		StateCreateTables: {StateFullSync, StateFailed, StatePaused},
		StateFullSync:   {StateIncSync, StateFailed, StatePaused},
		StateIncSync:    {StateWaiting, StateFailed, StatePaused},
		StateWaiting:    {StateValidating, StateFailed, StatePaused},
		StateValidating: {StateCompleted, StateFailed},
		StatePaused:     {StateConnect, StateCreateTables, StateFullSync, StateIncSync, StateWaiting, StateFailed},
		// Terminal states cannot transition
		StateCompleted: {},
		StateFailed:    {},
		StateDeleted:   {},
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
		StateInit:       "Initializing",
		StateConnect:    "Connecting to databases",
		StateCreateTables: "Creating target tables",
		StateFullSync:   "Full data synchronization",
		StateIncSync:    "Incremental synchronization",
		StateWaiting:    "Waiting for switchover",
		StateValidating: "Validating data",
		StateCompleted:  "Completed",
		StateFailed:     "Failed",
		StatePaused:     "Paused",
		StateDeleted:    "Deleted",
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
	if newState == StateConnect && task.StartedAt == nil {
		task.StartedAt = &now
	}
	if newState.IsTerminal() {
		task.CompletedAt = &now
	}
	task.UpdatedAt = now
}
