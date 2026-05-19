package app

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	aiClient   *AIClient
	traceStore *TraceStore
}

func NewService(aiClient *AIClient, traceStore *TraceStore) *Service {
	return &Service{
		aiClient:   aiClient,
		traceStore: traceStore,
	}
}

func (s *Service) Chat(ctx context.Context, message string) (ChatResponse, error) {
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return ChatResponse{}, ErrInvalidMessage
	}

	startedAt := time.Now()
	traceID := fmt.Sprintf("trace-%d", startedAt.UnixNano())

	reply, err := s.aiClient.Reply(ctx, AIReplyRequest{Message: trimmedMessage})
	if err != nil {
		return ChatResponse{}, fmt.Errorf("%w: %v", ErrAIUnavailable, err)
	}

	finishedAt := time.Now()
	s.traceStore.Append(Trace{
		TraceID:     traceID,
		UserMessage: trimmedMessage,
		Reply:       reply.Reply,
		ContextUsed: reply.ContextUsed,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		DurationMS:  finishedAt.Sub(startedAt).Milliseconds(),
	})

	return ChatResponse{
		Reply:   reply.Reply,
		TraceID: traceID,
		Persona: reply.Persona,
	}, nil
}
