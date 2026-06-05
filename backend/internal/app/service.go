package app

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	aiClient    *AIClient
	traceStore  *TraceStore
	memoryStore *MemoryStore
}

func NewService(aiClient *AIClient, traceStore *TraceStore, memoryStore *MemoryStore) *Service {
	return &Service{
		aiClient:    aiClient,
		traceStore:  traceStore,
		memoryStore: memoryStore,
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
		UserMessage:   trimmedMessage,
		CharacterID:   defaultCharacterID,
		CharacterName: defaultCharacterName,
		Persona:       DefaultBerryPersona(),
		Memories:      memoriesToContext(memories),
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

	s.traceStore.Append(Trace{
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
	})

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
