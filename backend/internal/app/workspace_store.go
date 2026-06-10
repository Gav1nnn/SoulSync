package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type workspaceStoreState struct {
	Current *Workspace `json:"current"`
}

type WorkspaceStore struct {
	mu    sync.Mutex
	path  string
	state workspaceStoreState
}

func NewWorkspaceStore(path string) (*WorkspaceStore, error) {
	store := &WorkspaceStore{
		path: path,
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *WorkspaceStore) Save(workspace Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.Current = &workspace
	return s.persistLocked()
}

func (s *WorkspaceStore) Current() (Workspace, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.Current == nil {
		return Workspace{}, false
	}

	return *s.state.Current, true
}

func (s *WorkspaceStore) load() error {
	if s.path == "" {
		return nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read workspace store: %w", err)
	}
	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, &s.state); err != nil {
		return fmt.Errorf("decode workspace store: %w", err)
	}

	return nil
}

func (s *WorkspaceStore) persistLocked() error {
	if s.path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create workspace store directory: %w", err)
	}

	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode workspace store: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write workspace store: %w", err)
	}

	return nil
}
