package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
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

	return NewService(aiClient, traceStore, memoryStore)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
