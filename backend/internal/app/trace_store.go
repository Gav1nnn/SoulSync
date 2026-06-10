package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type TraceStore struct {
	mu     sync.Mutex
	path   string
	traces []Trace
}

func NewTraceStore() *TraceStore {
	return &TraceStore{
		traces: make([]Trace, 0, 16),
	}
}

func NewTraceStoreWithPath(path string) (*TraceStore, error) {
	store := NewTraceStore()
	store.path = path

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *TraceStore) Append(trace Trace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.traces = append(s.traces, trace)
	return s.persistLocked()
}

func (s *TraceStore) Get(traceID string) (Trace, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for index := len(s.traces) - 1; index >= 0; index-- {
		if s.traces[index].TraceID == traceID {
			return s.traces[index], true
		}
	}

	return Trace{}, false
}

func (s *TraceStore) load() error {
	if s.path == "" {
		return nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read trace store: %w", err)
	}
	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, &s.traces); err != nil {
		return fmt.Errorf("decode trace store: %w", err)
	}
	if s.traces == nil {
		s.traces = make([]Trace, 0, 16)
	}

	return nil
}

func (s *TraceStore) persistLocked() error {
	if s.path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create trace store directory: %w", err)
	}

	data, err := json.MarshalIndent(s.traces, "", "  ")
	if err != nil {
		return fmt.Errorf("encode trace store: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write trace store: %w", err)
	}

	return nil
}
