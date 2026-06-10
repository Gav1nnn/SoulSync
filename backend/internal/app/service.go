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
	go s.runAgentTask(task.ID)

	return task, nil
}

func (s *Service) AgentTask(id string) (AgentTask, error) {
	task, ok := s.agentTaskStore.Get(strings.TrimSpace(id))
	if !ok {
		return AgentTask{}, ErrAgentTaskMissing
	}

	return task, nil
}

func (s *Service) runAgentTask(taskID string) {
	s.transitionAgentTask(taskID, AgentTaskPlanning, "Python planner is reading workspace summary.")

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

	agentPlan, err := s.aiClient.Plan(context.Background(), AIAgentPlanRequest{
		Goal:             task.Goal,
		WorkspaceSummary: summary,
		CharacterName:    defaultCharacterName,
		Persona:          DefaultBerryPersona(),
		Memories:         memoriesToContext(s.memoryStore.ListMemories()),
		RecentMessages:   messagesToConversationContext(s.memoryStore.RecentMessages(8)),
		ProjectContext:   workspaceSummaryContext(summary),
	})
	if err != nil {
		s.failAgentTask(taskID, fmt.Sprintf("agent planner failed: %v", err))
		return
	}
	if len(agentPlan.Plan) == 0 {
		s.failAgentTask(taskID, "agent planner returned an empty plan")
		return
	}

	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		initialAction := agentPlan.InitialAction
		task.Plan = agentPlan.Plan
		task.FilesToRead = agentPlan.FilesToRead
		task.InitialAction = &initialAction
		task.Planner = agentPlan.Planner
		task.PlannerContextUsed = agentPlan.ContextUsed
		task.Logs = append(task.Logs, AgentTaskLog{
			At:      time.Now(),
			Status:  AgentTaskPlanning,
			Message: fmt.Sprintf("Planner %s produced %d plan steps.", agentPlan.Planner, len(agentPlan.Plan)),
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

	s.transitionAgentTask(taskID, AgentTaskRunning, "ReAct stepper is executing bounded workspace actions.")
	if err := s.runAgentSteps(taskID, summary, agentPlan.InitialAction); err != nil {
		s.failAgentTask(taskID, err.Error())
		return
	}

	taskAfterSteps, ok := s.agentTaskStore.Get(taskID)
	if !ok {
		return
	}
	s.transitionAgentTask(taskID, AgentTaskVerifying, "Checking whether verification is required.")
	verification := AgentVerification{
		Status:  "skipped",
		Command: "not required",
		Output:  []string{"No files were changed by this task."},
	}
	if len(taskAfterSteps.ChangedFiles) > 0 {
		verification = runWhitelistedValidationCommand(task.Workspace.Path, summary.ValidationCommands)
	}
	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		task.Verification = &verification
	})
	if verification.Status != "passed" && verification.Status != "skipped" {
		s.setAgentTaskResult(taskID, AgentTaskFailed, "verification failed or unavailable")
		s.failAgentTask(taskID, "verification failed or unavailable")
		return
	}

	completedAt := time.Now()
	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		task.Status = AgentTaskCompleted
		task.CompletedAt = &completedAt
		task.Result = buildAgentTaskResult(*task, "")
		task.Logs = append(task.Logs, AgentTaskLog{
			At:      completedAt,
			Status:  AgentTaskCompleted,
			Message: "Task completed by ReAct stepper.",
		})
	})
}

func (s *Service) runAgentSteps(taskID string, summary WorkspaceSummary, initialAction AgentAction) error {
	const maxAgentSteps = 6

	task, ok := s.agentTaskStore.Get(taskID)
	if !ok || task.Workspace == nil {
		return fmt.Errorf("task workspace missing")
	}

	readFiles := []AgentReadFile{}
	var previousObservation *AgentObservation
	action := initialAction
	for stepIndex := 0; stepIndex < maxAgentSteps; stepIndex++ {
		if strings.TrimSpace(action.Type) == "" {
			return fmt.Errorf("agent step %d has empty action type", stepIndex+1)
		}

		startedAt := time.Now()
		normalizedAction := normalizeAgentAction(action)
		observation := executeAgentAction(task.Workspace.Path, summary.ValidationCommands, normalizedAction)
		finishedAt := time.Now()
		summaryText := summarizeAgentObservation(normalizedAction, observation)
		step := AgentTaskStep{
			Index:       stepIndex + 1,
			Action:      normalizedAction,
			Observation: observation,
			Summary:     summaryText,
			ContextUsed: []string{"go.safe_executor"},
			Stepper:     "go",
			StartedAt:   startedAt,
			FinishedAt:  finishedAt,
			DurationMS:  finishedAt.Sub(startedAt).Milliseconds(),
		}

		if normalizedAction.Type == "read_file" && observation.Status == "ok" && observation.Path != "" {
			readFiles = upsertAgentReadFile(readFiles, AgentReadFile{Path: observation.Path, Content: observation.Content})
		}
		if normalizedAction.Type == "write_file" && observation.Status == "ok" && observation.Path != "" {
			s.agentTaskStore.Update(taskID, func(task *AgentTask) {
				task.ChangedFiles = appendUniqueString(task.ChangedFiles, observation.Path)
			})
		}
		s.agentTaskStore.Update(taskID, func(task *AgentTask) {
			task.Steps = append(task.Steps, step)
			task.Logs = append(task.Logs, AgentTaskLog{
				At:      finishedAt,
				Status:  AgentTaskRunning,
				Message: fmt.Sprintf("Step %d %s: %s", step.Index, normalizedAction.Type, observation.Message),
			})
		})

		if observation.Status == "failed" || observation.Status == "unsupported" {
			return fmt.Errorf("agent step %d failed: %s", step.Index, observation.Message)
		}
		if normalizedAction.Type == "finish" {
			return nil
		}

		previousObservation = &observation
		currentTask, ok := s.agentTaskStore.Get(taskID)
		if !ok {
			return fmt.Errorf("task missing while stepping")
		}
		stepResponse, err := s.aiClient.Step(context.Background(), AIAgentStepRequest{
			Goal:                currentTask.Goal,
			Plan:                currentTask.Plan,
			WorkspaceSummary:    summary,
			StepIndex:           step.Index + 1,
			PreviousObservation: previousObservation,
			ReadFiles:           readFiles,
			ChangedFiles:        currentTask.ChangedFiles,
			RecentSteps:         lastAgentTaskSteps(currentTask.Steps, 4),
			CharacterName:       defaultCharacterName,
			Persona:             DefaultBerryPersona(),
			ProjectContext:      workspaceSummaryContext(summary),
		})
		if err != nil {
			return fmt.Errorf("agent stepper failed: %v", err)
		}

		action = stepResponse.Action
		if strings.TrimSpace(stepResponse.Summary) != "" || stepResponse.Stepper != "" || len(stepResponse.ContextUsed) > 0 {
			s.agentTaskStore.Update(taskID, func(task *AgentTask) {
				if len(task.Steps) == 0 {
					return
				}
				lastIndex := len(task.Steps) - 1
				task.Steps[lastIndex].Summary = strings.TrimSpace(stepResponse.Summary)
				task.Steps[lastIndex].ContextUsed = append([]string{}, stepResponse.ContextUsed...)
				task.Steps[lastIndex].Stepper = stepResponse.Stepper
			})
		}
	}

	return fmt.Errorf("agent stepper reached max step limit")
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
		if task.Result == nil {
			task.Result = buildAgentTaskResult(*task, message)
		}
		task.Logs = append(task.Logs, AgentTaskLog{
			At:      completedAt,
			Status:  AgentTaskFailed,
			Message: message,
		})
	})
}

func (s *Service) setAgentTaskResult(taskID string, status AgentTaskStatus, message string) {
	s.agentTaskStore.Update(taskID, func(task *AgentTask) {
		task.Status = status
		task.Result = buildAgentTaskResult(*task, message)
	})
}

func workspaceSummaryContext(summary WorkspaceSummary) []string {
	context := []string{
		"workspace=" + summary.RootName,
		"package_managers=" + strings.Join(summary.PackageManagers, ","),
		"frontend_frameworks=" + strings.Join(summary.FrontendFrameworks, ","),
		"backend_frameworks=" + strings.Join(summary.BackendFrameworks, ","),
	}
	appendCandidate := func(label string, candidates []WorkspaceCandidate) {
		for index, candidate := range candidates {
			if index >= 4 {
				break
			}
			context = append(context, fmt.Sprintf("%s=%s (%s)", label, candidate.Path, candidate.Kind))
		}
	}
	appendCandidate("route_candidate", summary.BackendRouteCandidates)
	for index, candidate := range summary.APICandidates {
		if index >= 4 {
			break
		}
		context = append(context, fmt.Sprintf("api_candidate=%s %s handler=%s file=%s", candidate.Method, candidate.Path, candidate.Handler, candidate.HandlerFile))
	}
	appendCandidate("api_client_candidate", summary.APIClientCandidates)
	appendCandidate("frontend_entry_candidate", summary.FrontendEntryCandidates)
	appendCandidate("type_candidate", summary.TypeFileCandidates)
	for index, command := range summary.ValidationCommands {
		if index >= 4 {
			break
		}
		context = append(context, "validation_command="+command)
	}

	return context
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
