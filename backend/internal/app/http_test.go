package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChatReturnsReplyAndTraceID(t *testing.T) {
	aiClient := NewAIClientWithHTTPClient("http://ai-engine.local", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost || r.URL.Path != "/generate" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}

			var request AIGenerateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode ai request: %v", err)
			}
			if request.UserMessage != "帮我写一个列表页" {
				t.Fatalf("unexpected user message: %q", request.UserMessage)
			}
			if request.CharacterID != defaultCharacterID || request.CharacterName != defaultCharacterName {
				t.Fatalf("unexpected character payload: %#v", request)
			}
			if request.Persona.Background == "" || len(request.Persona.SampleLines) == 0 {
				t.Fatalf("expected default persona to be populated: %#v", request.Persona)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(bytes.NewBufferString(
					`{"reply":"Berry reply","persona":"Berry","context_used":["persona","persona.examples","mock_fallback"],"used_persona":true,"used_memory_ids":[],"used_knowledge_chunk_ids":[],"memory_written":false,"memory_candidates":[{"type":"project_fact","content":"项目使用 Vue 3。","reason":"用户明确说明项目技术栈。","confidence":0.9}]}`,
				)),
			}, nil
		}),
	})

	server := NewHTTPServer(newTestService(t, aiClient)).Router()
	body := bytes.NewBufferString(`{"message":"帮我写一个列表页"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/chat", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response ChatResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Reply != "Berry reply" {
		t.Fatalf("unexpected reply: %q", response.Reply)
	}

	if response.Persona != defaultCharacterName {
		t.Fatalf("unexpected persona: %q", response.Persona)
	}

	if response.TraceID == "" {
		t.Fatal("expected trace id to be populated")
	}

	if len(response.ContextUsed) != 3 || response.ContextUsed[0] != "persona" {
		t.Fatalf("unexpected context used: %#v", response.ContextUsed)
	}

	if !response.MemoryWritten || response.MemoryCandidateCount != 1 {
		t.Fatalf("expected memory to be written: %#v", response)
	}
}

func TestChatRejectsEmptyMessage(t *testing.T) {
	server := NewHTTPServer(newTestService(t, NewAIClient("http://127.0.0.1:1"))).Router()
	body := bytes.NewBufferString(`{"message":"   "}`)
	request := httptest.NewRequest(http.MethodPost, "/api/chat", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestChatReturnsBadGatewayWhenAIUnavailable(t *testing.T) {
	aiClient := NewAIClientWithHTTPClient("http://ai-engine.local", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Body:       io.NopCloser(bytes.NewBufferString("upstream error")),
			}, nil
		}),
	})

	server := NewHTTPServer(newTestService(t, aiClient)).Router()
	body := bytes.NewBufferString(`{"message":"帮我看下组件拆分"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/chat", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", recorder.Code)
	}
}

func TestMemoriesEndpointReturnsSavedMemories(t *testing.T) {
	aiClient := NewAIClientWithHTTPClient("http://ai-engine.local", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(bytes.NewBufferString(
					`{"reply":"Berry reply","persona":"Berry","context_used":["persona","persona.examples","ollama"],"used_persona":true,"used_memory_ids":[],"used_knowledge_chunk_ids":[],"memory_written":false,"memory_candidates":[{"type":"project_fact","content":"项目使用 Vue 3。","reason":"用户明确说明项目技术栈。","confidence":0.9}]}`,
				)),
			}, nil
		}),
	})
	server := NewHTTPServer(newTestService(t, aiClient)).Router()

	chatBody := bytes.NewBufferString(`{"message":"我们项目使用 Vue 3"}`)
	chatRequest := httptest.NewRequest(http.MethodPost, "/api/chat", chatBody)
	chatRequest.Header.Set("Content-Type", "application/json")
	chatRecorder := httptest.NewRecorder()
	server.ServeHTTP(chatRecorder, chatRequest)
	if chatRecorder.Code != http.StatusOK {
		t.Fatalf("expected chat status 200, got %d", chatRecorder.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/memories", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("项目使用 Vue 3")) {
		t.Fatalf("expected memories response to include saved memory: %s", recorder.Body.String())
	}
}

func TestTraceEndpointReturnsCompleteTrace(t *testing.T) {
	aiClient := NewAIClientWithHTTPClient("http://ai-engine.local", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(bytes.NewBufferString(
					`{"reply":"Berry reply","persona":"Berry","context_used":["persona","knowledge","memory","ollama"],"used_persona":true,"used_memory_ids":["mem-1"],"used_knowledge_chunk_ids":["chunk-1"],"memory_written":false,"memory_candidates":[]}`,
				)),
			}, nil
		}),
	})
	server := NewHTTPServer(newTestService(t, aiClient)).Router()

	chatBody := bytes.NewBufferString(`{"message":"帮我解释这个接口"}`)
	chatRequest := httptest.NewRequest(http.MethodPost, "/api/chat", chatBody)
	chatRequest.Header.Set("Content-Type", "application/json")
	chatRecorder := httptest.NewRecorder()
	server.ServeHTTP(chatRecorder, chatRequest)
	if chatRecorder.Code != http.StatusOK {
		t.Fatalf("expected chat status 200, got %d", chatRecorder.Code)
	}

	var chatResponse ChatResponse
	if err := json.Unmarshal(chatRecorder.Body.Bytes(), &chatResponse); err != nil {
		t.Fatalf("decode chat response: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/traces/"+chatResponse.TraceID, nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response struct {
		Trace Trace `json:"trace"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode trace response: %v", err)
	}

	if response.Trace.TraceID != chatResponse.TraceID {
		t.Fatalf("unexpected trace id: %#v", response.Trace)
	}
	if len(response.Trace.UsedMemoryIDs) != 1 || response.Trace.UsedMemoryIDs[0] != "mem-1" {
		t.Fatalf("expected memory ids in trace: %#v", response.Trace)
	}
	if len(response.Trace.UsedKnowledgeChunkIDs) != 1 || response.Trace.UsedKnowledgeChunkIDs[0] != "chunk-1" {
		t.Fatalf("expected knowledge chunk ids in trace: %#v", response.Trace)
	}
	if response.Trace.DurationMS < 0 {
		t.Fatalf("expected non-negative duration: %#v", response.Trace)
	}
}

func TestMessagesEndpointReturnsRecentMessages(t *testing.T) {
	aiClient := NewAIClientWithHTTPClient("http://ai-engine.local", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(bytes.NewBufferString(
					`{"reply":"Berry reply","persona":"Berry","context_used":["persona","persona.examples","mock_fallback"],"used_persona":true,"used_memory_ids":[],"used_knowledge_chunk_ids":[],"memory_written":false,"memory_candidates":[]}`,
				)),
			}, nil
		}),
	})
	server := NewHTTPServer(newTestService(t, aiClient)).Router()

	for _, message := range []string{"第一条", "第二条"} {
		chatBody := bytes.NewBufferString(`{"message":"` + message + `"}`)
		chatRequest := httptest.NewRequest(http.MethodPost, "/api/chat", chatBody)
		chatRequest.Header.Set("Content-Type", "application/json")
		chatRecorder := httptest.NewRecorder()
		server.ServeHTTP(chatRecorder, chatRequest)
		if chatRecorder.Code != http.StatusOK {
			t.Fatalf("expected chat status 200, got %d", chatRecorder.Code)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/api/messages?limit=2", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response struct {
		Messages []Message `json:"messages"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode messages response: %v", err)
	}

	if len(response.Messages) != 2 {
		t.Fatalf("expected 2 recent messages, got %#v", response.Messages)
	}
	if response.Messages[0].Role != "user" || response.Messages[0].Content != "第二条" {
		t.Fatalf("unexpected recent messages: %#v", response.Messages)
	}
	if response.Messages[1].Role != "assistant" || response.Messages[1].Content != "Berry reply" {
		t.Fatalf("unexpected recent messages: %#v", response.Messages)
	}
}

func TestConnectWorkspaceSavesGitRepositoryStatus(t *testing.T) {
	server := NewHTTPServer(newTestService(t, NewAIClient("http://127.0.0.1:1"))).Router()
	workspacePath := newGitFixture(t)

	body := bytes.NewBufferString(`{"path":"` + workspacePath + `"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/workspaces", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Workspace Workspace `json:"workspace"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode workspace response: %v", err)
	}

	if response.Workspace.Path != workspacePath {
		t.Fatalf("unexpected workspace path: %#v", response.Workspace)
	}
	if response.Workspace.Branch == "" {
		t.Fatalf("expected branch to be populated: %#v", response.Workspace)
	}
	if response.Workspace.Dirty {
		t.Fatalf("expected clean fixture: %#v", response.Workspace)
	}
}

func TestCurrentWorkspaceRefreshesDirtyStatus(t *testing.T) {
	server := NewHTTPServer(newTestService(t, NewAIClient("http://127.0.0.1:1"))).Router()
	workspacePath := newGitFixture(t)

	connectBody := bytes.NewBufferString(`{"path":"` + workspacePath + `"}`)
	connectRequest := httptest.NewRequest(http.MethodPost, "/api/workspaces", connectBody)
	connectRequest.Header.Set("Content-Type", "application/json")
	connectRecorder := httptest.NewRecorder()
	server.ServeHTTP(connectRecorder, connectRequest)
	if connectRecorder.Code != http.StatusOK {
		t.Fatalf("expected connect status 200, got %d: %s", connectRecorder.Code, connectRecorder.Body.String())
	}

	if err := os.WriteFile(filepath.Join(workspacePath, "dirty.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/workspaces/current", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Workspace Workspace `json:"workspace"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode workspace response: %v", err)
	}

	if !response.Workspace.Dirty {
		t.Fatalf("expected dirty workspace: %#v", response.Workspace)
	}
}

func TestConnectWorkspaceRejectsRelativePath(t *testing.T) {
	server := NewHTTPServer(newTestService(t, NewAIClient("http://127.0.0.1:1"))).Router()
	body := bytes.NewBufferString(`{"path":"relative-project"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/workspaces", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("path must be absolute")) {
		t.Fatalf("expected absolute path error, got %s", recorder.Body.String())
	}
}

func TestCurrentWorkspaceSummaryScansProjectCandidates(t *testing.T) {
	server := NewHTTPServer(newTestService(t, NewAIClient("http://127.0.0.1:1"))).Router()
	workspacePath := newFrontendBackendFixture(t)

	connectBody := bytes.NewBufferString(`{"path":"` + workspacePath + `"}`)
	connectRequest := httptest.NewRequest(http.MethodPost, "/api/workspaces", connectBody)
	connectRequest.Header.Set("Content-Type", "application/json")
	connectRecorder := httptest.NewRecorder()
	server.ServeHTTP(connectRecorder, connectRequest)
	if connectRecorder.Code != http.StatusOK {
		t.Fatalf("expected connect status 200, got %d: %s", connectRecorder.Code, connectRecorder.Body.String())
	}

	request := httptest.NewRequest(http.MethodGet, "/api/workspaces/current/summary", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Summary WorkspaceSummary `json:"summary"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode summary response: %v", err)
	}

	if response.Summary.WorkspacePath != workspacePath {
		t.Fatalf("unexpected workspace path: %#v", response.Summary)
	}
	if !containsString(response.Summary.PackageManagers, "npm") {
		t.Fatalf("expected npm package manager: %#v", response.Summary.PackageManagers)
	}
	if !containsString(response.Summary.FrontendFrameworks, "Vue") || !containsString(response.Summary.FrontendFrameworks, "Vite") {
		t.Fatalf("expected Vue/Vite frontend frameworks: %#v", response.Summary.FrontendFrameworks)
	}
	if !containsString(response.Summary.BackendFrameworks, "Gin") {
		t.Fatalf("expected Gin backend framework: %#v", response.Summary.BackendFrameworks)
	}
	if !containsCandidate(response.Summary.BackendRouteCandidates, "backend/main.go") {
		t.Fatalf("expected backend route candidate: %#v", response.Summary.BackendRouteCandidates)
	}
	if !containsCandidate(response.Summary.TypeFileCandidates, "frontend/src/types/user.ts") {
		t.Fatalf("expected type file candidate: %#v", response.Summary.TypeFileCandidates)
	}
	if !containsCandidate(response.Summary.FrontendEntryCandidates, "frontend/src/views/UserListView.vue") {
		t.Fatalf("expected frontend page candidate: %#v", response.Summary.FrontendEntryCandidates)
	}
	if !containsCandidate(response.Summary.APIClientCandidates, "frontend/src/api/users.ts") {
		t.Fatalf("expected api client candidate: %#v", response.Summary.APIClientCandidates)
	}
	if !containsString(response.Summary.ValidationCommands, "cd frontend && npm run build") {
		t.Fatalf("expected frontend build command: %#v", response.Summary.ValidationCommands)
	}
	if len(response.Summary.Tree) == 0 {
		t.Fatalf("expected tree summary to be populated: %#v", response.Summary)
	}
}

func TestCurrentWorkspaceSummaryRequiresConnectedWorkspace(t *testing.T) {
	server := NewHTTPServer(newTestService(t, NewAIClient("http://127.0.0.1:1"))).Router()
	request := httptest.NewRequest(http.MethodGet, "/api/workspaces/current/summary", nil)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(ErrWorkspaceMissing.Error())) {
		t.Fatalf("expected missing workspace error, got %s", recorder.Body.String())
	}
}

func TestCreateAgentTaskRequiresConnectedWorkspace(t *testing.T) {
	server := NewHTTPServer(newTestService(t, NewAIClient("http://127.0.0.1:1"))).Router()
	body := bytes.NewBufferString(`{"goal":"根据用户接口生成页面"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/agent/tasks", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(ErrWorkspaceMissing.Error())) {
		t.Fatalf("expected workspace missing error, got %s", recorder.Body.String())
	}
}

func TestCreateAgentTaskRunsMockLifecycle(t *testing.T) {
	server := NewHTTPServer(newTestService(t, NewAIClient("http://127.0.0.1:1"))).Router()
	workspacePath := newFrontendBackendFixture(t)

	connectBody := bytes.NewBufferString(`{"path":"` + workspacePath + `"}`)
	connectRequest := httptest.NewRequest(http.MethodPost, "/api/workspaces", connectBody)
	connectRequest.Header.Set("Content-Type", "application/json")
	connectRecorder := httptest.NewRecorder()
	server.ServeHTTP(connectRecorder, connectRequest)
	if connectRecorder.Code != http.StatusOK {
		t.Fatalf("expected connect status 200, got %d: %s", connectRecorder.Code, connectRecorder.Body.String())
	}

	body := bytes.NewBufferString(`{"goal":"根据用户列表接口生成 Vue 页面"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/agent/tasks", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Task AgentTask `json:"task"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode task response: %v", err)
	}
	if response.Task.ID == "" || response.Task.Status != AgentTaskQueued {
		t.Fatalf("unexpected created task: %#v", response.Task)
	}

	task := waitForAgentTaskStatus(t, server, response.Task.ID, AgentTaskCompleted)
	if task.BranchName == "" {
		t.Fatalf("expected branch name to be populated: %#v", task)
	}
	if len(task.Plan) == 0 {
		t.Fatalf("expected plan to be populated: %#v", task)
	}
	if len(task.Logs) < 4 {
		t.Fatalf("expected lifecycle logs: %#v", task.Logs)
	}
	if task.Verification == nil || task.Verification.Status != "passed" {
		t.Fatalf("expected skipped verification: %#v", task.Verification)
	}
	if len(task.ChangedFiles) != 1 {
		t.Fatalf("expected one changed file in safe execution stage: %#v", task.ChangedFiles)
	}
	if task.ChangedFiles[0] != ".soulsync/"+task.ID+".md" {
		t.Fatalf("unexpected changed file: %#v", task.ChangedFiles)
	}
	if _, err := os.Stat(filepath.Join(workspacePath, filepath.FromSlash(task.ChangedFiles[0]))); err != nil {
		t.Fatalf("expected changed file to exist: %v", err)
	}
	currentBranch := strings.TrimSpace(runTestGitOutput(t, workspacePath, "branch", "--show-current"))
	if currentBranch != task.BranchName {
		t.Fatalf("expected task branch %q, got %q", task.BranchName, currentBranch)
	}
}

func TestSafeWorkspacePathRejectsEscapes(t *testing.T) {
	root := t.TempDir()

	if _, err := safeWorkspacePath(root, "../outside.txt"); err == nil {
		t.Fatal("expected escaping path to be rejected")
	}

	inside, err := safeWorkspacePath(root, ".soulsync/task.md")
	if err != nil {
		t.Fatalf("expected inside path to be allowed: %v", err)
	}
	if !strings.HasPrefix(inside, root) {
		t.Fatalf("expected path inside root, got %q", inside)
	}
}

func TestChatSendsRecentMessagesToAIEngine(t *testing.T) {
	callCount := 0
	aiClient := NewAIClientWithHTTPClient("http://ai-engine.local", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			callCount++
			var request AIGenerateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode ai request: %v", err)
			}

			if callCount == 2 {
				if len(request.RecentMessages) == 0 {
					t.Fatal("expected recent messages to be sent on second request")
				}
				if request.RecentMessages[0].Role != "user" || request.RecentMessages[0].Content != "你好，我叫 Gavin" {
					t.Fatalf("unexpected recent messages: %#v", request.RecentMessages)
				}
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(bytes.NewBufferString(
					`{"reply":"Berry reply","persona":"Berry","context_used":["persona","persona.examples","deepseek"],"used_persona":true,"used_memory_ids":[],"used_knowledge_chunk_ids":[],"memory_written":false,"memory_candidates":[]}`,
				)),
			}, nil
		}),
	})
	server := NewHTTPServer(newTestService(t, aiClient)).Router()

	firstBody := bytes.NewBufferString(`{"message":"你好，我叫 Gavin"}`)
	firstRequest := httptest.NewRequest(http.MethodPost, "/api/chat", firstBody)
	firstRequest.Header.Set("Content-Type", "application/json")
	firstRecorder := httptest.NewRecorder()
	server.ServeHTTP(firstRecorder, firstRequest)
	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("expected first status 200, got %d", firstRecorder.Code)
	}

	secondBody := bytes.NewBufferString(`{"message":"我的名字是什么"}`)
	secondRequest := httptest.NewRequest(http.MethodPost, "/api/chat", secondBody)
	secondRequest.Header.Set("Content-Type", "application/json")
	secondRecorder := httptest.NewRecorder()
	server.ServeHTTP(secondRecorder, secondRequest)
	if secondRecorder.Code != http.StatusOK {
		t.Fatalf("expected second status 200, got %d", secondRecorder.Code)
	}
}

func newTestService(t *testing.T, aiClient *AIClient) *Service {
	t.Helper()

	memoryStore, err := NewMemoryStore(t.TempDir() + "/memory-store.json")
	if err != nil {
		t.Fatalf("create memory store: %v", err)
	}

	traceStore, err := NewTraceStoreWithPath(t.TempDir() + "/trace-store.json")
	if err != nil {
		t.Fatalf("create trace store: %v", err)
	}

	workspaceStore, err := NewWorkspaceStore(t.TempDir() + "/workspaces.json")
	if err != nil {
		t.Fatalf("create workspace store: %v", err)
	}

	return NewService(aiClient, traceStore, memoryStore, workspaceStore, NewAgentTaskStore())
}

func waitForAgentTaskStatus(t *testing.T, server http.Handler, taskID string, expected AgentTaskStatus) AgentTask {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	var lastTask AgentTask
	for time.Now().Before(deadline) {
		request := httptest.NewRequest(http.MethodGet, "/api/agent/tasks/"+taskID, nil)
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
		}

		var response struct {
			Task AgentTask `json:"task"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode task response: %v", err)
		}
		lastTask = response.Task
		if response.Task.Status == expected {
			return response.Task
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("task did not reach %s, last task: %#v", expected, lastTask)
	return AgentTask{}
}

func newGitFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	runTestGit(t, dir, "init")
	runTestGit(t, dir, "config", "user.email", "berry@example.test")
	runTestGit(t, dir, "config", "user.name", "Berry")

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Fixture\n"), 0o644); err != nil {
		t.Fatalf("write fixture readme: %v", err)
	}

	runTestGit(t, dir, "add", "README.md")
	runTestGit(t, dir, "commit", "-m", "init fixture")

	return dir
}

func newFrontendBackendFixture(t *testing.T) string {
	t.Helper()

	dir := newGitFixture(t)
	writeFixtureFile(t, dir, "frontend/package.json", `{"scripts":{"build":"node -e \"console.log('fixture build ok')\"","test":"node -e \"console.log('fixture test ok')\""},"dependencies":{"vue":"^3.5.0"},"devDependencies":{"vite":"^6.0.0"}}`)
	writeFixtureFile(t, dir, "frontend/package-lock.json", `{"name":"fixture"}`)
	writeFixtureFile(t, dir, "frontend/src/views/UserListView.vue", `<template><main>User list</main></template>`)
	writeFixtureFile(t, dir, "frontend/src/api/users.ts", `export async function listUsers(){ return fetch("/api/users") }`)
	writeFixtureFile(t, dir, "frontend/src/types/user.ts", `export type User = { id: string; name: string }`)
	writeFixtureFile(t, dir, "backend/main.go", `package main

import "github.com/gin-gonic/gin"

func main() {
	router := gin.Default()
	router.GET("/api/users", func(c *gin.Context) {})
}`)

	runTestGit(t, dir, "add", ".")
	runTestGit(t, dir, "commit", "-m", "add app fixture")

	return dir
}

func writeFixtureFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	absPath := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("create fixture dir: %v", err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file %s: %v", relPath, err)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func containsCandidate(candidates []WorkspaceCandidate, expectedPath string) bool {
	for _, candidate := range candidates {
		if candidate.Path == expectedPath {
			return true
		}
	}
	return false
}

func runTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}

func runTestGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}

	return string(output)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
