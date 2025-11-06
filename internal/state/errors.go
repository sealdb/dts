package state

import "errors"

var (
	// ErrInvalidState 无效状态错误
	ErrInvalidState = errors.New("invalid state")

	// ErrStateTransitionFailed 状态转换失败
	ErrStateTransitionFailed = errors.New("state transition failed")

	// ErrTaskNotFound 任务不存在
	ErrTaskNotFound = errors.New("task not found")

	// ErrTaskAlreadyCompleted 任务已完成
	ErrTaskAlreadyCompleted = errors.New("task already completed")

	// ErrTaskFailed 任务失败
	ErrTaskFailed = errors.New("task failed")
)
