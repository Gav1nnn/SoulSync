package app

import (
	"testing"
	"time"
)

func TestSaveCandidatesDeduplicatesUserProfileNameVariants(t *testing.T) {
	store, err := NewMemoryStore(t.TempDir() + "/memory-store.json")
	if err != nil {
		t.Fatalf("create memory store: %v", err)
	}

	now := time.Date(2026, 6, 5, 15, 0, 0, 0, time.UTC)
	candidates := []MemoryCandidate{
		{
			Type:       "user_profile",
			Content:    "用户名叫 Gavin",
			Reason:     "用户明确介绍姓名",
			Confidence: 0.9,
		},
		{
			Type:       "user_profile",
			Content:    "用户叫Gavin",
			Reason:     "用户明确介绍姓名",
			Confidence: 0.9,
		},
		{
			Type:       "user_profile",
			Content:    "用户名字是Gavin",
			Reason:     "用户明确介绍姓名",
			Confidence: 0.9,
		},
	}

	if _, err := store.SaveCandidates(candidates, "trace-1", "message-1", now); err != nil {
		t.Fatalf("save candidates: %v", err)
	}

	memories := store.ListMemories()
	if len(memories) != 1 {
		t.Fatalf("expected one deduplicated user profile memory, got %#v", memories)
	}
	if memories[0].Type != "user_profile" || memories[0].Content != "用户名字是Gavin" {
		t.Fatalf("unexpected memory after dedupe: %#v", memories[0])
	}
}
