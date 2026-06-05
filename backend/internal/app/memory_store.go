package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	activeMemoryStatus       = "active"
	defaultMemoryType        = "project_fact"
	memoryWriteMinConfidence = 0.75
)

type memoryStoreState struct {
	Messages []Message `json:"messages"`
	Memories []Memory  `json:"memories"`
}

type MemoryStore struct {
	mu    sync.Mutex
	path  string
	state memoryStoreState
}

func NewMemoryStore(path string) (*MemoryStore, error) {
	store := &MemoryStore{
		path: path,
		state: memoryStoreState{
			Messages: make([]Message, 0, 32),
			Memories: make([]Memory, 0, 16),
		},
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *MemoryStore) AppendMessage(message Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.Messages = append(s.state.Messages, message)
	return s.persistLocked()
}

func (s *MemoryStore) RecentMessages(limit int) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || len(s.state.Messages) == 0 {
		return []Message{}
	}

	start := len(s.state.Messages) - limit
	if start < 0 {
		start = 0
	}

	messages := make([]Message, 0, len(s.state.Messages[start:]))
	messages = append(messages, s.state.Messages[start:]...)
	return messages
}

func (s *MemoryStore) ListMemories() []Memory {
	s.mu.Lock()
	defer s.mu.Unlock()

	memories := make([]Memory, 0, len(s.state.Memories))
	for _, memory := range s.state.Memories {
		if memory.Status == activeMemoryStatus {
			memories = append(memories, memory)
		}
	}

	sort.SliceStable(memories, func(i, j int) bool {
		return memories[i].UpdatedAt.After(memories[j].UpdatedAt)
	})

	return memories
}

func (s *MemoryStore) FindRelevantMemories(query string, limit int) ([]Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 {
		return []Memory{}, nil
	}

	type scoredMemory struct {
		memory Memory
		score  int
	}

	scored := make([]scoredMemory, 0, len(s.state.Memories))
	for _, memory := range s.state.Memories {
		if memory.Status != activeMemoryStatus {
			continue
		}
		scored = append(scored, scoredMemory{
			memory: memory,
			score:  scoreMemory(query, memory),
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].memory.UpdatedAt.After(scored[j].memory.UpdatedAt)
	})

	result := make([]Memory, 0, min(limit, len(scored)))
	now := time.Now()
	changed := false
	for _, item := range scored {
		if len(result) >= limit {
			break
		}
		if item.score <= 0 && len(result) > 0 {
			continue
		}
		item.memory.LastUsedAt = now
		result = append(result, item.memory)
		for index := range s.state.Memories {
			if s.state.Memories[index].ID == item.memory.ID {
				s.state.Memories[index].LastUsedAt = now
				changed = true
				break
			}
		}
	}

	if changed {
		if err := s.persistLocked(); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *MemoryStore) SaveCandidates(
	candidates []MemoryCandidate,
	traceID string,
	sourceMessageID string,
	now time.Time,
) ([]Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	written := make([]Memory, 0, len(candidates))
	for index, candidate := range candidates {
		content := strings.TrimSpace(candidate.Content)
		if content == "" || candidate.Confidence < memoryWriteMinConfidence {
			continue
		}

		memoryType := strings.TrimSpace(candidate.Type)
		if memoryType == "" {
			memoryType = defaultMemoryType
		}

		existingIndex := findExistingMemoryIndex(s.state.Memories, memoryType, content)

		if existingIndex >= 0 {
			memory := s.state.Memories[existingIndex]
			memory.Type = memoryType
			memory.Content = content
			memory.Reason = strings.TrimSpace(candidate.Reason)
			memory.Confidence = candidate.Confidence
			memory.Status = activeMemoryStatus
			memory.SourceTraceID = traceID
			memory.SourceMessageID = sourceMessageID
			memory.UpdatedAt = now
			s.state.Memories[existingIndex] = memory
			written = append(written, memory)
			continue
		}

		memory := Memory{
			ID:              fmt.Sprintf("mem-%d-%d", now.UnixNano(), index),
			Type:            memoryType,
			Content:         content,
			Reason:          strings.TrimSpace(candidate.Reason),
			Confidence:      candidate.Confidence,
			Status:          activeMemoryStatus,
			SourceTraceID:   traceID,
			SourceMessageID: sourceMessageID,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		s.state.Memories = append(s.state.Memories, memory)
		written = append(written, memory)
	}

	if len(written) == 0 {
		return written, nil
	}

	if err := s.persistLocked(); err != nil {
		return nil, err
	}

	return written, nil
}

func (s *MemoryStore) load() error {
	if s.path == "" {
		return nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read memory store: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, &s.state); err != nil {
		return fmt.Errorf("decode memory store: %w", err)
	}
	if s.state.Messages == nil {
		s.state.Messages = make([]Message, 0, 32)
	}
	if s.state.Memories == nil {
		s.state.Memories = make([]Memory, 0, 16)
	}

	return nil
}

func (s *MemoryStore) persistLocked() error {
	if s.path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create memory store directory: %w", err)
	}

	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode memory store: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write memory store: %w", err)
	}

	return nil
}

func scoreMemory(query string, memory Memory) int {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return 0
	}

	haystack := strings.ToLower(memory.Type + " " + memory.Content)
	score := 0
	if memory.Type == "user_profile" && asksAboutUserProfile(query) {
		score += 10
	}
	for _, token := range memoryTokens(query) {
		if strings.Contains(haystack, token) {
			score++
		}
	}

	if score == 0 && strings.Contains(haystack, query) {
		score = 1
	}

	return score
}

func memoryTokens(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
	})
}

func normalizeMemoryContent(content string) string {
	return strings.ToLower(strings.Join(strings.Fields(content), " "))
}

func findExistingMemoryIndex(memories []Memory, memoryType string, content string) int {
	normalized := normalizeMemoryContent(content)
	profileKey := ""
	if memoryType == "user_profile" {
		profileKey = userProfileMemoryKey(content)
	}

	for index, memory := range memories {
		if memoryType == "user_profile" &&
			memory.Type == "user_profile" &&
			profileKey != "" &&
			userProfileMemoryKey(memory.Content) == profileKey {
			return index
		}
		if normalizeMemoryContent(memory.Content) == normalized {
			return index
		}
	}

	return -1
}

func userProfileMemoryKey(content string) string {
	text := strings.ToLower(content)
	text = strings.NewReplacer(
		"：", " ",
		"。", " ",
		"，", " ",
		"、", " ",
		",", " ",
		".", " ",
		":", " ",
		";", " ",
		"；", " ",
		"「", " ",
		"」", " ",
		"“", " ",
		"”", " ",
		"\"", " ",
		"'", " ",
		"`", " ",
		"　", " ",
	).Replace(text)
	text = strings.Join(strings.Fields(text), "")

	for _, phrase := range []string{
		"用户的名字是",
		"用户名字是",
		"用户名叫",
		"用户名是",
		"用户姓名是",
		"用户姓名叫",
		"用户叫做",
		"用户叫",
		"用户是",
		"我的名字是",
		"我叫",
		"名字是",
		"姓名是",
		"称呼我为",
		"称呼为",
		"叫做",
		"用户",
		"名字",
		"姓名",
		"叫",
		"是",
		"mynameis",
		"callme",
		"iam",
		"i'm",
	} {
		text = strings.ReplaceAll(text, phrase, "")
	}

	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, text)
}

func asksAboutUserProfile(query string) bool {
	markers := []string{
		"名字",
		"我叫什么",
		"我叫啥",
		"我是谁",
		"怎么称呼",
		"name",
	}
	for _, marker := range markers {
		if strings.Contains(query, marker) {
			return true
		}
	}
	return false
}

func memoriesToContext(memories []Memory) []MemoryContext {
	context := make([]MemoryContext, 0, len(memories))
	for _, memory := range memories {
		context = append(context, MemoryContext{
			ID:      memory.ID,
			Content: memory.Content,
			Type:    memory.Type,
		})
	}
	return context
}

func messagesToConversationContext(messages []Message) []ConversationMessage {
	context := make([]ConversationMessage, 0, len(messages))
	for _, message := range messages {
		if message.Role != "user" && message.Role != "assistant" {
			continue
		}
		context = append(context, ConversationMessage{
			Role:    message.Role,
			Content: message.Content,
		})
	}
	return context
}
