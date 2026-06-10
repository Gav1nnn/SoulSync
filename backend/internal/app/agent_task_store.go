package app

import (
	"fmt"
	"sync"
	"time"
)

type AgentTaskStore struct {
	mu    sync.Mutex
	tasks map[string]AgentTask
}

func NewAgentTaskStore() *AgentTaskStore {
	return &AgentTaskStore{
		tasks: make(map[string]AgentTask),
	}
}

func (s *AgentTaskStore) Create(goal string, workspace *Workspace, now time.Time) AgentTask {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := AgentTask{
		ID:                 fmt.Sprintf("task-%d", now.UnixNano()),
		Goal:               goal,
		Status:             AgentTaskQueued,
		Workspace:          workspace,
		Plan:               []string{},
		FilesToRead:        []string{},
		PlannerContextUsed: []string{},
		Steps:              []AgentTaskStep{},
		Logs:               []AgentTaskLog{},
		ChangedFiles:       []string{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	task.Logs = append(task.Logs, AgentTaskLog{
		At:      now,
		Status:  AgentTaskQueued,
		Message: "Task queued.",
	})

	s.tasks[task.ID] = task
	return cloneAgentTask(task)
}

func (s *AgentTaskStore) Get(id string) (AgentTask, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return AgentTask{}, false
	}

	return cloneAgentTask(task), true
}

func (s *AgentTaskStore) Update(id string, update func(*AgentTask)) (AgentTask, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return AgentTask{}, false
	}

	update(&task)
	task.UpdatedAt = time.Now()
	s.tasks[id] = task
	return cloneAgentTask(task), true
}

func cloneAgentTask(task AgentTask) AgentTask {
	task.Plan = append([]string{}, task.Plan...)
	task.FilesToRead = append([]string{}, task.FilesToRead...)
	task.PlannerContextUsed = append([]string{}, task.PlannerContextUsed...)
	task.Steps = cloneAgentTaskSteps(task.Steps)
	task.Logs = append([]AgentTaskLog{}, task.Logs...)
	task.ChangedFiles = append([]string{}, task.ChangedFiles...)
	if task.Workspace != nil {
		workspace := *task.Workspace
		task.Workspace = &workspace
	}
	if task.Verification != nil {
		verification := *task.Verification
		verification.Output = append([]string{}, task.Verification.Output...)
		task.Verification = &verification
	}
	if task.Result != nil {
		result := *task.Result
		result.NextSuggestions = append([]string{}, task.Result.NextSuggestions...)
		task.Result = &result
	}
	if task.InitialAction != nil {
		initialAction := *task.InitialAction
		task.InitialAction = &initialAction
	}
	if task.CompletedAt != nil {
		completedAt := *task.CompletedAt
		task.CompletedAt = &completedAt
	}

	return task
}

func cloneAgentTaskSteps(steps []AgentTaskStep) []AgentTaskStep {
	cloned := make([]AgentTaskStep, len(steps))
	for index, step := range steps {
		cloned[index] = step
		cloned[index].Observation.Items = append([]string{}, step.Observation.Items...)
		cloned[index].Observation.Matches = append([]string{}, step.Observation.Matches...)
		cloned[index].Observation.Output = append([]string{}, step.Observation.Output...)
		cloned[index].ContextUsed = append([]string{}, step.ContextUsed...)
	}

	return cloned
}
