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

	branchName := agentTaskBranchName(time.Now())
	if err := createAgentTaskBranch(task.Workspace.Path, branchName); err != nil {
		s.failAgentTask(taskID, fmt.Sprintf("create task branch failed: %v", err))
		return
	}
	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		task.BranchName = branchName
		task.Logs = append(task.Logs, AgentTaskLog{
			At:      time.Now(),
			Status:  AgentTaskPlanning,
			Message: "Created isolated branch: " + branchName,
		})
	})

	s.transitionAgentTask(taskID, AgentTaskRunning, "Mock runner is writing a bounded test file inside the workspace.")
	changedFile, err := writeAgentTaskTestFile(task.Workspace.Path, taskID, task.Goal)
	if err != nil {
		s.failAgentTask(taskID, fmt.Sprintf("write mock task file failed: %v", err))
		return
	}
	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		task.ChangedFiles = []string{changedFile}
		task.Logs = append(task.Logs, AgentTaskLog{
			At:      time.Now(),
			Status:  AgentTaskRunning,
			Message: "Changed file: " + changedFile,
		})
	})

	s.transitionAgentTask(taskID, AgentTaskVerifying, "Running whitelisted verification command.")
	verification := runWhitelistedValidationCommand(task.Workspace.Path, summary.ValidationCommands)
	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		task.Verification = &verification
	})
	if verification.Status != "passed" {
		s.failAgentTask(taskID, "verification failed or unavailable")
		return
	}

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
	steps = append(steps, "创建独立分支，写入 mock 测试文件，并运行白名单验证命令。")

	return steps
}

func agentTaskBranchName(now time.Time) string {
	return "agent/frontend-from-api-" + now.Format("20060102-150405")
}

func createAgentTaskBranch(workspaceRoot string, branchName string) error {
	if strings.TrimSpace(branchName) == "" || strings.Contains(branchName, "..") {
		return fmt.Errorf("invalid branch name")
	}
	if _, err := runGit(workspaceRoot, "checkout", "-b", branchName); err != nil {
		return err
	}

	return nil
}

func writeAgentTaskTestFile(workspaceRoot string, taskID string, goal string) (string, error) {
	relPath := filepath.ToSlash(filepath.Join(".soulsync", taskID+".md"))
	absPath, err := safeWorkspacePath(workspaceRoot, relPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", fmt.Errorf("create task directory: %w", err)
	}

	content := strings.Join([]string{
		"# SoulSync Mock Agent Task",
		"",
		"Task ID: " + taskID,
		"Goal: " + goal,
		"",
		"This file is written by the safe execution mock runtime.",
		"It proves workspace-bounded writes before real Agent actions are enabled.",
		"",
	}, "\n")
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write task file: %w", err)
	}

	return relPath, nil
}

func safeWorkspacePath(workspaceRoot string, relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("path must be relative")
	}

	root := filepath.Clean(workspaceRoot)
	absPath := filepath.Clean(filepath.Join(root, filepath.FromSlash(relPath)))
	relative, err := filepath.Rel(root, absPath)
	if err != nil {
		return "", fmt.Errorf("check workspace path: %w", err)
	}
	if relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return "", fmt.Errorf("path escapes workspace")
	}

	return absPath, nil
}

func runWhitelistedValidationCommand(workspaceRoot string, commands []string) AgentVerification {
	if len(commands) == 0 {
		return AgentVerification{
			Status:  "failed",
			Command: "not detected",
			Output:  []string{"No validation command was discovered for this workspace."},
		}
	}

	for _, command := range commands {
		result := runValidationCommand(workspaceRoot, command)
		if result.Status != "unsupported" {
			return result
		}
	}

	return AgentVerification{
		Status:  "failed",
		Command: commands[0],
		Output:  []string{"No discovered validation command is supported by the safe executor."},
	}
}

func runValidationCommand(workspaceRoot string, command string) AgentVerification {
	workdir := workspaceRoot
	commandText := strings.TrimSpace(command)
	if commandText == "" {
		return AgentVerification{Status: "unsupported", Command: command, Output: []string{"Empty command."}}
	}

	if strings.Contains(commandText, "&&") {
		parts := strings.Split(commandText, "&&")
		if len(parts) != 2 {
			return AgentVerification{Status: "unsupported", Command: command, Output: []string{"Only one cd prefix is supported."}}
		}

		cdPart := strings.TrimSpace(parts[0])
		if !strings.HasPrefix(cdPart, "cd ") {
			return AgentVerification{Status: "unsupported", Command: command, Output: []string{"Only cd prefixes are supported."}}
		}

		nextWorkdir, err := safeWorkspacePath(workspaceRoot, strings.TrimSpace(strings.TrimPrefix(cdPart, "cd ")))
		if err != nil {
			return AgentVerification{Status: "failed", Command: command, Output: []string{err.Error()}}
		}
		workdir = nextWorkdir
		commandText = strings.TrimSpace(parts[1])
	}

	name, args, ok := parseValidationCommand(commandText)
	if !ok {
		return AgentVerification{Status: "unsupported", Command: command, Output: []string{"Command is not in the safe executor allowlist."}}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	execCommand := exec.CommandContext(ctx, name, args...)
	execCommand.Dir = workdir
	output, err := execCommand.CombinedOutput()
	outputLines := commandOutputLines(output)
	if ctx.Err() == context.DeadlineExceeded {
		outputLines = append(outputLines, "Command timed out after 60 seconds.")
		return AgentVerification{Status: "failed", Command: command, Output: outputLines}
	}
	if err != nil {
		outputLines = append(outputLines, err.Error())
		return AgentVerification{Status: "failed", Command: command, Output: outputLines}
	}

	if len(outputLines) == 0 {
		outputLines = []string{"Command completed successfully."}
	}
	return AgentVerification{Status: "passed", Command: command, Output: outputLines}
}

func parseValidationCommand(command string) (string, []string, bool) {
	fields := splitCommandFields(command)
	if len(fields) == 0 {
		return "", nil, false
	}

	switch {
	case len(fields) == 3 && fields[0] == "npm" && fields[1] == "run":
		return "npm", fields[1:], true
	case len(fields) == 3 && fields[0] == "go" && fields[1] == "test" && fields[2] == "./...":
		return "go", fields[1:], true
	case len(fields) >= 3 && fields[0] == "uv" && fields[1] == "run" && fields[2] == "python":
		return "uv", fields[1:], true
	case len(fields) >= 3 && fields[0] == "python" && fields[1] == "-m" && fields[2] == "unittest":
		return "python", fields[1:], true
	default:
		return "", nil, false
	}
}

func splitCommandFields(command string) []string {
	rawFields := strings.Fields(command)
	fields := make([]string, 0, len(rawFields))
	for _, field := range rawFields {
		fields = append(fields, strings.Trim(field, `"'`))
	}
	return fields
}

func commandOutputLines(output []byte) []string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return []string{}
	}

	lines := strings.Split(text, "\n")
	if len(lines) > 40 {
		lines = append(lines[:40], "... output truncated ...")
	}

	return lines
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
