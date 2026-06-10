package app

import "time"

type ChatRequest struct {
	Message string `json:"message"`
}

type WorkspaceRequest struct {
	Path string `json:"path"`
}

type AgentTaskRequest struct {
	Goal string `json:"goal"`
}

type ChatResponse struct {
	Reply                string   `json:"reply"`
	TraceID              string   `json:"trace_id"`
	Persona              string   `json:"persona"`
	ContextUsed          []string `json:"context_used"`
	UsedMemoryIDs        []string `json:"used_memory_ids"`
	MemoryWritten        bool     `json:"memory_written"`
	MemoryCandidateCount int      `json:"memory_candidate_count"`
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
	UserMessage    string                `json:"user_message"`
	CharacterID    string                `json:"character_id"`
	CharacterName  string                `json:"character_name"`
	Persona        PersonaProfile        `json:"persona"`
	Memories       []MemoryContext       `json:"memories"`
	RecentMessages []ConversationMessage `json:"recent_messages"`
}

type AIGenerateResponse struct {
	Reply                 string            `json:"reply"`
	Persona               string            `json:"persona"`
	ContextUsed           []string          `json:"context_used"`
	UsedPersona           bool              `json:"used_persona"`
	UsedMemoryIDs         []string          `json:"used_memory_ids"`
	UsedKnowledgeChunkIDs []string          `json:"used_knowledge_chunk_ids"`
	MemoryWritten         bool              `json:"memory_written"`
	MemoryCandidates      []MemoryCandidate `json:"memory_candidates"`
}

type Trace struct {
	TraceID               string    `json:"trace_id"`
	UserMessage           string    `json:"user_message"`
	Reply                 string    `json:"reply"`
	ContextUsed           []string  `json:"context_used"`
	UsedMemoryIDs         []string  `json:"used_memory_ids"`
	UsedKnowledgeChunkIDs []string  `json:"used_knowledge_chunk_ids"`
	MemoryWritten         bool      `json:"memory_written"`
	MemoryCandidateCount  int       `json:"memory_candidate_count"`
	StartedAt             time.Time `json:"started_at"`
	FinishedAt            time.Time `json:"finished_at"`
	DurationMS            int64     `json:"duration_ms"`
}

type MemoryContext struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

type ConversationMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MemoryCandidate struct {
	Type       string  `json:"type"`
	Content    string  `json:"content"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
}

type Message struct {
	ID        string    `json:"id"`
	TraceID   string    `json:"trace_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Workspace struct {
	Path      string    `json:"path"`
	Branch    string    `json:"branch"`
	Dirty     bool      `json:"dirty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WorkspaceSummary struct {
	WorkspacePath           string               `json:"workspace_path"`
	RootName                string               `json:"root_name"`
	Tree                    []WorkspaceTreeItem  `json:"tree"`
	PackageManagers         []string             `json:"package_managers"`
	FrontendFrameworks      []string             `json:"frontend_frameworks"`
	BackendFrameworks       []string             `json:"backend_frameworks"`
	BackendRouteCandidates  []WorkspaceCandidate `json:"backend_route_candidates"`
	TypeFileCandidates      []WorkspaceCandidate `json:"type_file_candidates"`
	FrontendEntryCandidates []WorkspaceCandidate `json:"frontend_entry_candidates"`
	APIClientCandidates     []WorkspaceCandidate `json:"api_client_candidates"`
	ValidationCommands      []string             `json:"validation_commands"`
	GeneratedAt             time.Time            `json:"generated_at"`
}

type WorkspaceTreeItem struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type WorkspaceCandidate struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

type AgentTaskStatus string

const (
	AgentTaskQueued    AgentTaskStatus = "queued"
	AgentTaskPlanning  AgentTaskStatus = "planning"
	AgentTaskRunning   AgentTaskStatus = "running"
	AgentTaskVerifying AgentTaskStatus = "verifying"
	AgentTaskCompleted AgentTaskStatus = "completed"
	AgentTaskFailed    AgentTaskStatus = "failed"
)

type AgentTask struct {
	ID           string             `json:"id"`
	Goal         string             `json:"goal"`
	Status       AgentTaskStatus    `json:"status"`
	Workspace    *Workspace         `json:"workspace,omitempty"`
	Plan         []string           `json:"plan"`
	Logs         []AgentTaskLog     `json:"logs"`
	ChangedFiles []string           `json:"changed_files"`
	Verification *AgentVerification `json:"verification,omitempty"`
	Error        string             `json:"error,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	CompletedAt  *time.Time         `json:"completed_at,omitempty"`
}

type AgentTaskLog struct {
	At      time.Time       `json:"at"`
	Status  AgentTaskStatus `json:"status"`
	Message string          `json:"message"`
}

type AgentVerification struct {
	Status  string   `json:"status"`
	Command string   `json:"command"`
	Output  []string `json:"output"`
}

type Memory struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	Content         string    `json:"content"`
	Reason          string    `json:"reason"`
	Confidence      float64   `json:"confidence"`
	Status          string    `json:"status"`
	SourceTraceID   string    `json:"source_trace_id"`
	SourceMessageID string    `json:"source_message_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LastUsedAt      time.Time `json:"last_used_at"`
}
