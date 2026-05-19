package app

import "time"

const personaName = "Berry"

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply   string `json:"reply"`
	TraceID string `json:"trace_id"`
	Persona string `json:"persona"`
}

type AIReplyRequest struct {
	Message string `json:"message"`
}

type AIReplyResponse struct {
	Reply       string   `json:"reply"`
	Persona     string   `json:"persona"`
	ContextUsed []string `json:"context_used"`
}

type Trace struct {
	TraceID     string    `json:"trace_id"`
	UserMessage string    `json:"user_message"`
	Reply       string    `json:"reply"`
	ContextUsed []string  `json:"context_used"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
	DurationMS  int64     `json:"duration_ms"`
}
