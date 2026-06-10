package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	aiClient       *AIClient
	traceStore     *TraceStore
	memoryStore    *MemoryStore
	workspaceStore *WorkspaceStore
}

func NewService(
	aiClient *AIClient,
	traceStore *TraceStore,
	memoryStore *MemoryStore,
	workspaceStore *WorkspaceStore,
) *Service {
	return &Service{
		aiClient:       aiClient,
		traceStore:     traceStore,
		memoryStore:    memoryStore,
		workspaceStore: workspaceStore,
	}
}

func (s *Service) Chat(ctx context.Context, message string) (ChatResponse, error) {
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return ChatResponse{}, ErrInvalidMessage
	}

	startedAt := time.Now()
	traceID := fmt.Sprintf("trace-%d", startedAt.UnixNano())
	userMessageID := fmt.Sprintf("msg-%d-user", startedAt.UnixNano())
	recentMessages := s.memoryStore.RecentMessages(8)

	if err := s.memoryStore.AppendMessage(Message{
		ID:        userMessageID,
		TraceID:   traceID,
		Role:      "user",
		Content:   trimmedMessage,
		CreatedAt: startedAt,
	}); err != nil {
		return ChatResponse{}, fmt.Errorf("save user message: %w", err)
	}

	memories, err := s.memoryStore.FindRelevantMemories(trimmedMessage, 20)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("find memories: %w", err)
	}

	reply, err := s.aiClient.Generate(ctx, AIGenerateRequest{
		UserMessage:    trimmedMessage,
		CharacterID:    defaultCharacterID,
		CharacterName:  defaultCharacterName,
		Persona:        DefaultBerryPersona(),
		Memories:       memoriesToContext(memories),
		RecentMessages: messagesToConversationContext(recentMessages),
	})
	if err != nil {
		return ChatResponse{}, fmt.Errorf("%w: %v", ErrAIUnavailable, err)
	}

	finishedAt := time.Now()
	assistantMessageID := fmt.Sprintf("msg-%d-assistant", finishedAt.UnixNano())
	if err := s.memoryStore.AppendMessage(Message{
		ID:        assistantMessageID,
		TraceID:   traceID,
		Role:      "assistant",
		Content:   reply.Reply,
		CreatedAt: finishedAt,
	}); err != nil {
		return ChatResponse{}, fmt.Errorf("save assistant message: %w", err)
	}

	writtenMemories, err := s.memoryStore.SaveCandidates(
		reply.MemoryCandidates,
		traceID,
		userMessageID,
		finishedAt,
	)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("save memory candidates: %w", err)
	}
	memoryWritten := len(writtenMemories) > 0

	if err := s.traceStore.Append(Trace{
		TraceID:               traceID,
		UserMessage:           trimmedMessage,
		Reply:                 reply.Reply,
		ContextUsed:           reply.ContextUsed,
		UsedMemoryIDs:         reply.UsedMemoryIDs,
		UsedKnowledgeChunkIDs: reply.UsedKnowledgeChunkIDs,
		MemoryWritten:         memoryWritten,
		MemoryCandidateCount:  len(reply.MemoryCandidates),
		StartedAt:             startedAt,
		FinishedAt:            finishedAt,
		DurationMS:            finishedAt.Sub(startedAt).Milliseconds(),
	}); err != nil {
		return ChatResponse{}, fmt.Errorf("save trace: %w", err)
	}

	return ChatResponse{
		Reply:                reply.Reply,
		TraceID:              traceID,
		Persona:              reply.Persona,
		ContextUsed:          reply.ContextUsed,
		UsedMemoryIDs:        reply.UsedMemoryIDs,
		MemoryWritten:        memoryWritten,
		MemoryCandidateCount: len(reply.MemoryCandidates),
	}, nil
}

func (s *Service) Memories() []Memory {
	return s.memoryStore.ListMemories()
}

func (s *Service) RecentMessages(limit int) []Message {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	return s.memoryStore.RecentMessages(limit)
}

func (s *Service) Trace(traceID string) (Trace, bool) {
	return s.traceStore.Get(traceID)
}

func (s *Service) ConnectWorkspace(path string) (Workspace, error) {
	workspace, err := readWorkspace(path)
	if err != nil {
		return Workspace{}, err
	}

	if err := s.workspaceStore.Save(workspace); err != nil {
		return Workspace{}, fmt.Errorf("save workspace: %w", err)
	}

	return workspace, nil
}

func (s *Service) CurrentWorkspace() (Workspace, bool, error) {
	workspace, ok := s.workspaceStore.Current()
	if !ok {
		return Workspace{}, false, nil
	}

	refreshed, err := readWorkspace(workspace.Path)
	if err != nil {
		return Workspace{}, false, err
	}

	if err := s.workspaceStore.Save(refreshed); err != nil {
		return Workspace{}, false, fmt.Errorf("save workspace: %w", err)
	}

	return refreshed, true, nil
}

func (s *Service) CurrentWorkspaceSummary() (WorkspaceSummary, bool, error) {
	workspace, ok, err := s.CurrentWorkspace()
	if err != nil || !ok {
		return WorkspaceSummary{}, ok, err
	}

	summary, err := buildWorkspaceSummary(workspace.Path)
	if err != nil {
		return WorkspaceSummary{}, false, err
	}

	return summary, true, nil
}

func readWorkspace(rawPath string) (Workspace, error) {
	trimmedPath := strings.TrimSpace(rawPath)
	if trimmedPath == "" {
		return Workspace{}, fmt.Errorf("%w: path is required", ErrInvalidWorkspace)
	}
	if !filepath.IsAbs(trimmedPath) {
		return Workspace{}, fmt.Errorf("%w: path must be absolute", ErrInvalidWorkspace)
	}

	cleanPath := filepath.Clean(trimmedPath)
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Workspace{}, fmt.Errorf("%w: path does not exist", ErrInvalidWorkspace)
		}
		return Workspace{}, fmt.Errorf("%w: stat path: %v", ErrInvalidWorkspace, err)
	}
	if !info.IsDir() {
		return Workspace{}, fmt.Errorf("%w: path must be a directory", ErrInvalidWorkspace)
	}

	if _, err := runGit(cleanPath, "rev-parse", "--show-toplevel"); err != nil {
		return Workspace{}, fmt.Errorf("%w: path must be a git repository", ErrInvalidWorkspace)
	}

	branch, err := runGit(cleanPath, "branch", "--show-current")
	if err != nil {
		return Workspace{}, fmt.Errorf("%w: read current branch: %v", ErrInvalidWorkspace, err)
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		branch, err = runGit(cleanPath, "rev-parse", "--short", "HEAD")
		if err != nil {
			branch = "detached"
		} else {
			branch = "detached@" + strings.TrimSpace(branch)
		}
	}

	status, err := runGit(cleanPath, "status", "--porcelain")
	if err != nil {
		return Workspace{}, fmt.Errorf("%w: read git status: %v", ErrInvalidWorkspace, err)
	}

	return Workspace{
		Path:      cleanPath,
		Branch:    branch,
		Dirty:     strings.TrimSpace(status) != "",
		UpdatedAt: time.Now(),
	}, nil
}

func runGit(workdir string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = workdir

	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(string(output)))
	}

	return string(output), nil
}
