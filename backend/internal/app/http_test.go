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

func newTestService(t *testing.T, aiClient *AIClient) *Service {
	t.Helper()

	memoryStore, err := NewMemoryStore(t.TempDir() + "/memory-store.json")
	if err != nil {
		t.Fatalf("create memory store: %v", err)
	}

	return NewService(aiClient, NewTraceStore(), memoryStore)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
