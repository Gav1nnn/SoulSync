package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatReturnsReplyAndTraceID(t *testing.T) {
	aiEngine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/generate" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"reply":"Berry reply","persona":"Berry","context_used":["persona"],"used_persona":true,"used_memory_ids":[],"used_knowledge_chunk_ids":[],"memory_written":false}`))
	}))
	defer aiEngine.Close()

	server := NewHTTPServer(NewService(NewAIClient(aiEngine.URL), NewTraceStore())).Router()
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
}

func TestChatRejectsEmptyMessage(t *testing.T) {
	server := NewHTTPServer(NewService(NewAIClient("http://127.0.0.1:1"), NewTraceStore())).Router()
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
	aiEngine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream error", http.StatusInternalServerError)
	}))
	defer aiEngine.Close()

	server := NewHTTPServer(NewService(NewAIClient(aiEngine.URL), NewTraceStore())).Router()
	body := bytes.NewBufferString(`{"message":"帮我看下组件拆分"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/chat", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", recorder.Code)
	}
}
