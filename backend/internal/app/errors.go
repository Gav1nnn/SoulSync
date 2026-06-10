package app

import "errors"

var (
	ErrInvalidMessage   = errors.New("message must not be empty")
	ErrAIUnavailable    = errors.New("ai engine unavailable")
	ErrInvalidWorkspace = errors.New("invalid workspace")
	ErrWorkspaceMissing = errors.New("workspace not connected")
)
