package state

import "errors"

var (
	// ErrInvalidState indicates an invalid state error
	ErrInvalidState = errors.New("invalid state")

	// ErrStateTransitionFailed indicates a state transition failure
	ErrStateTransitionFailed = errors.New("state transition failed")

	// ErrTaskNotFound indicates the task was not found
	ErrTaskNotFound = errors.New("task not found")

	// ErrTaskAlreadyCompleted indicates the task is already completed
	ErrTaskAlreadyCompleted = errors.New("task already completed")

	// ErrTaskFailed indicates the task has failed
	ErrTaskFailed = errors.New("task failed")
)
