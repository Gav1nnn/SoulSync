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
	agentTaskStore *AgentTaskStore
}

func NewService(
	aiClient *AIClient,
	traceStore *TraceStore,
	memoryStore *MemoryStore,
	workspaceStore *WorkspaceStore,
	agentTaskStore *AgentTaskStore,
) *Service {
	return &Service{
		aiClient:       aiClient,
		traceStore:     traceStore,
		memoryStore:    memoryStore,
		workspaceStore: workspaceStore,
		agentTaskStore: agentTaskStore,
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

func (s *Service) CreateAgentTask(goal string) (AgentTask, error) {
	trimmedGoal := strings.TrimSpace(goal)
	if trimmedGoal == "" {
		return AgentTask{}, fmt.Errorf("%w: goal is required", ErrInvalidAgentTask)
	}

	workspace, ok, err := s.CurrentWorkspace()
	if err != nil {
		return AgentTask{}, err
	}
	if !ok {
		return AgentTask{}, ErrWorkspaceMissing
	}

	taskWorkspace := workspace
	task := s.agentTaskStore.Create(trimmedGoal, &taskWorkspace, time.Now())
	go s.runMockAgentTask(task.ID)

	return task, nil
}

func (s *Service) AgentTask(id string) (AgentTask, error) {
	task, ok := s.agentTaskStore.Get(strings.TrimSpace(id))
	if !ok {
		return AgentTask{}, ErrAgentTaskMissing
	}

	return task, nil
}

func (s *Service) runMockAgentTask(taskID string) {
	s.transitionAgentTask(taskID, AgentTaskPlanning, "Mock planner is reading workspace summary.")

	task, ok := s.agentTaskStore.Get(taskID)
	if !ok || task.Workspace == nil {
		s.failAgentTask(taskID, "workspace missing while planning")
		return
	}

	summary, err := buildWorkspaceSummary(task.Workspace.Path)
	if err != nil {
		s.failAgentTask(taskID, fmt.Sprintf("workspace summary failed: %v", err))
		return
	}

	plan := mockAgentPlan(task.Goal, summary)
	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		task.Plan = plan
		task.Logs = append(task.Logs, AgentTaskLog{
			At:      time.Now(),
			Status:  AgentTaskPlanning,
			Message: fmt.Sprintf("Mock planner produced %d plan steps.", len(plan)),
		})
	})

	s.transitionAgentTask(taskID, AgentTaskRunning, "Mock runner recorded candidate files. No files were changed in this stage.")
	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		task.ChangedFiles = []string{}
		task.Logs = append(task.Logs, AgentTaskLog{
			At:      time.Now(),
			Status:  AgentTaskRunning,
			Message: "Write execution is disabled until safe execution is implemented.",
		})
	})

	s.transitionAgentTask(taskID, AgentTaskVerifying, "Mock verification selected the first discovered validation command.")
	command := "not detected"
	if len(summary.ValidationCommands) > 0 {
		command = summary.ValidationCommands[0]
	}
	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		task.Verification = &AgentVerification{
			Status:  "skipped",
			Command: command,
			Output:  []string{"Mock verification only. Command execution arrives in the safe execution stage."},
		}
	})

	completedAt := time.Now()
	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		task.Status = AgentTaskCompleted
		task.CompletedAt = &completedAt
		task.Logs = append(task.Logs, AgentTaskLog{
			At:      completedAt,
			Status:  AgentTaskCompleted,
			Message: "Task completed by mock runtime.",
		})
	})
}

func (s *Service) transitionAgentTask(taskID string, status AgentTaskStatus, message string) {
	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		task.Status = status
		task.Logs = append(task.Logs, AgentTaskLog{
			At:      time.Now(),
			Status:  status,
			Message: message,
		})
	})
}

func (s *Service) failAgentTask(taskID string, message string) {
	completedAt := time.Now()
	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		task.Status = AgentTaskFailed
		task.Error = message
		task.CompletedAt = &completedAt
		task.Logs = append(task.Logs, AgentTaskLog{
			At:      completedAt,
			Status:  AgentTaskFailed,
			Message: message,
		})
	})
}

func mockAgentPlan(goal string, summary WorkspaceSummary) []string {
	steps := []string{
		"确认用户目标：" + goal,
		"读取 workspace 摘要，定位前端入口、API client 和后端路由候选。",
	}
	if len(summary.BackendRouteCandidates) > 0 {
		steps = append(steps, "优先查看后端路由候选："+summary.BackendRouteCandidates[0].Path)
	}
	if len(summary.FrontendEntryCandidates) > 0 {
		steps = append(steps, "参考前端入口或页面："+summary.FrontendEntryCandidates[0].Path)
	}
	if len(summary.APIClientCandidates) > 0 {
		steps = append(steps, "复用 API client 候选："+summary.APIClientCandidates[0].Path)
	}
	steps = append(steps, "本阶段不写文件，等待安全执行层接入。")

	return steps
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
