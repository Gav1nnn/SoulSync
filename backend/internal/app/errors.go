package app

import "errors"

var (
	ErrInvalidMessage   = errors.New("message must not be empty")
	ErrAIUnavailable    = errors.New("ai engine unavailable")
	ErrInvalidAgentTask = errors.New("invalid agent task")
	ErrAgentTaskMissing = errors.New("agent task not found")
	ErrInvalidMemory    = errors.New("invalid memory")
	ErrMemoryMissing    = errors.New("memory not found")
	ErrInvalidWorkspace = errors.New("invalid workspace")
	ErrWorkspaceMissing = errors.New("workspace not connected")
)
