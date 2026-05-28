package app

import "time"

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply       string   `json:"reply"`
	TraceID     string   `json:"trace_id"`
	Persona     string   `json:"persona"`
	ContextUsed []string `json:"context_used"`
}

type PersonaProfile struct {
	Background    string   `json:"background"`
	Traits        []string `json:"traits"`
	SpeakingStyle string   `json:"speaking_style"`
	Taboos        []string `json:"taboos"`
	Expertise     []string `json:"expertise"`
	SampleLines   []string `json:"sample_lines"`
}

type AIGenerateRequest struct {
	UserMessage   string         `json:"user_message"`
	CharacterID   string         `json:"character_id"`
	CharacterName string         `json:"character_name"`
	Persona       PersonaProfile `json:"persona"`
}

type AIGenerateResponse struct {
	Reply                 string   `json:"reply"`
	Persona               string   `json:"persona"`
	ContextUsed           []string `json:"context_used"`
	UsedPersona           bool     `json:"used_persona"`
	UsedMemoryIDs         []string `json:"used_memory_ids"`
	UsedKnowledgeChunkIDs []string `json:"used_knowledge_chunk_ids"`
	MemoryWritten         bool     `json:"memory_written"`
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
